# Ishchi Bot — Complete Functionality Documentation

> **Purpose**: Comprehensive reference for systematic code review.
> **Generated**: 2026-03-03  
> **Scope**: Every user-facing flow, admin workflow, background process, and infrastructure component.

---

## Table of Contents

1. [System Overview](#1-system-overview)
2. [Startup & Infrastructure](#2-startup--infrastructure)
3. [Routing & Middleware](#3-routing--middleware)
4. [Registration Flow](#4-registration-flow)
5. [Booking Flow](#5-booking-flow)
6. [Payment Flow](#6-payment-flow)
7. [Expiry Worker](#7-expiry-worker)
8. [Profile Management](#8-profile-management)
9. [User Commands & Text Router](#9-user-commands--text-router)
10. [Admin: Job Creation](#10-admin-job-creation)
11. [Admin: Job Management](#11-admin-job-management)
12. [Admin: Payment Approval](#12-admin-payment-approval)
13. [Admin: Statistics & User List](#13-admin-statistics--user-list)
14. [Violation & Blocking System](#14-violation--blocking-system)
15. [Sender Service](#15-sender-service)
16. [Storage Layer](#16-storage-layer)
17. [Data Models](#17-data-models)
18. [Validation](#18-validation)
19. [Configuration](#19-configuration)
20. [Known Issues & Review Targets](#20-known-issues--review-targets)

---

## 1. System Overview

### Architecture

```
cmd/main.go
  ├── config.Load()
  ├── postgres.NewPostgres()      → storage layer (pgxpool)
  ├── tele.NewBot()               → telebot v4 (polling or webhook)
  ├── service.NewServiceManager() → Registration, Booking, Payment, Sender
  ├── handlers.NewHandler()       → all Telegram handlers
  ├── bot.RegisterRoutes()        → middleware + route registration
  └── service.NewExpiryWorker()   → background goroutine (10s ticker)
```

### Key Flows (User Journey)

```
Channel post (deep link) → /start job_123 → Registration (if needed) → Booking → Payment → Admin Approval → Confirmed
```

### Technology Stack

| Component | Technology |
|---|---|
| Language | Go 1.24+ |
| Telegram | gopkg.in/telebot.v4 (v4.0.0-beta.7) |
| Database | PostgreSQL via github.com/jackc/pgx/v5/pgxpool |
| Migrations | golang-migrate/migrate/v4 |
| Config | github.com/joho/godotenv |
| Logging | Custom zap-based logger (`pkg/logger/`) |

---

## 2. Startup & Infrastructure

### File: `cmd/main.go` (127 lines)

**Boot sequence:**
1. `config.Load()` — reads `.env`, parses all env vars
2. `logger.NewLogger()` — initializes zap logger
3. `postgres.NewPostgres()` — creates pgxpool, runs migrations, sets `statement_timeout=30s`, `lock_timeout=10s` on every connection via `AfterConnect`
4. Creates `tele.Bot` — either `LongPoller` (dev) or `Webhook` (prod)
5. `service.NewServiceManager()` — wires Registration, Booking, Payment, Sender services
6. `handlers.NewHandler()` — receives logger, storage, bot, config, services
7. `bot.RegisterRoutes()` — registers middleware (recovery → rate limiter) and all handlers
8. `service.NewExpiryWorker()` — starts in separate goroutine
9. `telegramBot.Start()` in goroutine; main waits for SIGINT/SIGTERM
10. Graceful shutdown: stops expiry worker, rate limiter, bot; 5s timeout

### File: `config/config.go` (173 lines)

**Config structure:**
- `BotConfig`: Token, ChannelID, AdminIDs, AdminGroupID, Username, Mode (webhook/polling), WebhookURL/Port, RateLimitMaxRequests/Window
- `DatabaseConfig`: Host, Port, User, Password, DBName, MaxConnections
- `AppConfig`: Environment, LogLevel
- `PaymentConfig`: CardNumber, CardHolderName

**Env vars**: `BOT_TOKEN`, `BOT_CHANNEL_ID`, `BOT_ADMIN_IDS` (comma-separated), `BOT_ADMIN_GROUP_ID`, `BOT_USERNAME`, `BOT_MODE`, `DB_HOST/PORT/USER/PASSWORD/NAME`, `CARD_NUMBER`, `CARD_HOLDER_NAME`, etc.

### File: `storage/postgres/postgres.go`

**Connection pool:**
- Min: 20, Max: 200 connections (hardcoded)
- `AfterConnect`: Sets `statement_timeout = '30s'` and `lock_timeout = '10s'` on every new connection
- Auto-runs migrations from `migrations/` on startup

---

## 3. Routing & Middleware

### File: `bot/bot.go` (52 lines)

**Route registration order:**
1. Middleware: `RecoveryMiddleware` → `RateLimiter.Middleware()`
2. Commands: `/start`, `/help`, `/about`, `/settings`, `/admin`
3. Generic handlers: `OnCallback` → `HandleCallback`, `OnText` → `HandleText`, `OnContact` → `HandleContact`, `OnPhoto` → `HandlePhoto`, `OnLocation` → `HandleLocation`

### File: `bot/middleware/recovery.go` (62 lines)

- Wraps every handler with `defer recover()`
- On panic: logs `user_id`, `username`, `callback_data`, `message_text`, full stack trace
- Returns `fmt.Errorf("panic recovered: %v", r)` so telebot's OnError handler also fires
- **Critical**: Without this, a panic kills the polling goroutine silently

### File: `bot/middleware/rate_limiter.go` (187 lines)

- Per-user sliding window rate limiter
- Default: 30 requests per 60 seconds (configurable via `BOT_RATE_LIMIT_MAX`, `BOT_RATE_LIMIT_WINDOW`)
- **Admins are exempt** from rate limiting
- Uses `sync.RWMutex` + per-user `userBucket` with timestamp slice
- Background `cleanupLoop` goroutine evicts stale buckets (runs every 5 minutes, removes buckets inactive for 10 minutes)
- When rate limit exceeded: silently drops the update (returns nil, no error message to user)

### File: `bot/handlers/callback_router.go` (120 lines)

**Two-tier routing:**
1. **Static callbacks** (exact match map): `help`, `about`, `settings`, `back`, `confirm_yes/no`, `admin_menu`, `admin_create_job`, `admin_job_list`, `cancel_job_creation`, `skip_field`, `reg_accept_offer`, `reg_decline_offer`, `reg_continue`, `reg_restart`, `reg_confirm`, `reg_edit`, `reg_cancel`, `reg_back_to_confirm`, `reg_edit_{field}`, `book_cancel`, `user_my_jobs`, `user_profile`, `edit_profile_{field}`
2. **Dynamic callbacks** (ordered prefix match, slice not map): `job_detail_`, `edit_job_`, `job_status_`, `publish_job_`, `delete_channel_msg_`, `delete_job_`, `view_job_bookings_`, `book_confirm_`, `start_reg_job_`, `approve_payment_`, `reject_payment_`, `block_user_`, `users_page_`

**Order matters**: More specific prefixes must come before shorter overlapping ones.

---

## 4. Registration Flow

### Files: `bot/handlers/registration.go` (523 lines), `service/registration.go` (532 lines)

### State Machine

```
/start (unregistered)
  → RegStatePublicOffer (show public_offer.txt)
  → accept → RegStateFullName
  → decline → deleted, idle

RegStateFullName → validate (2+ words, no digits/emoji) → RegStatePhone
RegStatePhone → validate (+998 format, contact or text) → RegStateAge
RegStateAge → validate (16-65) → RegStateBodyParams
RegStateBodyParams → validate (weight 30-200, height 100-250) → RegStateConfirm

RegStateConfirm:
  → "✅ Tasdiqlash" → CompleteRegistration (moves draft → registered_users) → idle
  → "✏️ Tahrirlash" → choose field → edit → back to confirm
  → "❌ Bekor qilish" → delete draft → idle
```

### Deep Link Registration (with pending job)

1. User clicks channel link → `/start job_123`
2. `HandleStart` detects `job_` payload, checks if registered
3. If NOT registered: calls `HandleRegistrationStartWithJob(c, jobID)`
4. Shows registration info with job preview → user clicks "✅ Ro'yxatdan o'tish"
5. `HandleStartRegistrationForJob` saves `draft.PendingJobID = &jobID`
6. Normal registration proceeds
7. On `HandleConfirmRegistration`: checks `draft.PendingJobID`, if set → redirects to `HandleJobBookingStart`

### Key Service Methods

| Method | Purpose |
|---|---|
| `CheckUserRegistrationStatus()` | Returns isRegistered, hasDraft, draft |
| `StartRegistration()` | Deletes old draft, creates new with `RegStatePublicOffer` |
| `ProcessPublicOfferResponse()` | Accept → `RegStateFullName`; Decline → delete |
| `ProcessFullName/Phone/Age/BodyParams()` | Validate, save to draft, return next state |
| `ConfirmRegistration()` | Calls `storage.CompleteRegistration()` (moves draft → registered_users) |
| `GoToEditState()` | Saves `PreviousState=Confirm`, sets state to field; on save, returns to confirm |
| `FormatRegistrationSummary()` | Returns Markdown summary of draft |

### Storage Operations

- `CreateDraft`, `GetDraftByUserID`, `UpdateDraft`, `DeleteDraft`
- `CompleteRegistration` — atomically moves draft to `registered_users` table
- `IsUserRegistered` — checks `registered_users` table
- `GetRegisteredUserByUserID`, `UpdateRegisteredUser`

---

## 5. Booking Flow

### Files: `bot/handlers/booking.go` (260 lines), `service/booking.go` (250 lines)

### Flow

```
1. User clicks deep link → /start job_123 → HandleStart
2. Registered? → HandleJobBookingStart → show job details + ✅/❌ buttons
3. User clicks "✅ Ha, yozilaman" → book_confirm_{jobID} → HandleBookingConfirm
4. Service: ConfirmBooking (in transaction):
   a. Check block status (permanent/temporary/expired-auto-unblock)
   b. Check idempotency (existing booking for same user+job)
   c. Check no other active bookings (user can have only ONE pending at a time)
   d. BEGIN TX → FOR UPDATE lock job → validate active + available slots
   e. IncrementReservedSlots → Create booking (SLOT_RESERVED, 3min expiry) → COMMIT
5. Show payment instructions (card number, amount, 3-min countdown)
6. Background goroutine: stores PaymentInstructionMsgID in booking
```

### Idempotency Checks (HandleBookingConfirm)

Before calling service, handler checks via `CheckIdempotency`:
- `SLOT_RESERVED` + not expired → "⚠️ already booked, X minutes Y seconds remaining"
- `PAYMENT_SUBMITTED` → "⚠️ payment under review"
- `CONFIRMED` → "✅ already confirmed"

### Service: ConfirmBooking Business Logic

1. **Block check**: `GetBlockStatus` → permanent block (nil BlockedUntil) → error; temporary block (now < BlockedUntil) → error with remaining time; expired block → auto-unblock
2. **Idempotency**: Generate key `user_{id}_job_{id}`, check existing booking
3. **Cross-job constraint**: Only ONE active booking per user (checks both RESERVED and SUBMITTED across all jobs)
4. **Transaction**: `BEGIN` → `GetByIDForUpdate(job)` → validate status=ACTIVE, available slots > 0 → `IncrementReservedSlots` → `Create(booking)` → `COMMIT`
5. Booking created with 3-minute `ExpiresAt`

### Slot Accounting Model

```
RequiredWorkers = total capacity
ReservedSlots = temporarily held (3-min timer, pending payment)
ConfirmedSlots = admin-approved (permanent)
AvailableSlots = RequiredWorkers - ReservedSlots - ConfirmedSlots
IsFull = AvailableSlots <= 0
IsCompletelyFull = ConfirmedSlots >= RequiredWorkers
```

---

## 6. Payment Flow

### Files: `bot/handlers/commands.go` (HandlePhoto, HandlePaymentReceiptSubmission), `bot/handlers/payment.go` (587 lines), `service/payment.go` (350 lines)

### User Side

```
1. User sends photo → OnPhoto → HandlePhoto → HandlePaymentReceiptSubmission
2. Service: SubmitPayment:
   a. Find most recent SLOT_RESERVED booking for user
   b. Check not expired
   c. TX: Update status → PAYMENT_SUBMITTED, store receipt FileID + MsgID → COMMIT
3. Send "✅ TO'LOV CHEKI QABUL QILINDI" to user
4. go ForwardPaymentToAdminGroup (sends photo + info to admin group)
```

### Admin Side (see Section 12)

```
1. Admin group receives photo with ✅ Tasdiqlash | ❌ Rad etish | 🚫 Bloklash buttons
2. Approve → ApprovePayment service → CONFIRMED
3. Reject → RejectPayment service → REJECTED + slot released
4. Block → BlockUserAndRejectPayment → violation recorded + progressive blocking
```

### Service: SubmitPayment

1. Query `GetUserBookingsByStatus(SLOT_RESERVED)` → take first (most recent)
2. Check `time.Now().After(booking.ExpiresAt)` → "booking has expired"
3. TX: Update booking status to `PAYMENT_SUBMITTED`, save `PaymentReceiptFileID`, `PaymentReceiptMsgID`, `PaymentSubmittedAt`
4. Return booking for admin forwarding

### Service: ApprovePayment

1. TX: `GetByIDForUpdate(booking)` → verify status == `PAYMENT_SUBMITTED`
2. Set status = `CONFIRMED`, `ConfirmedAt`, `ReviewedByAdminID`, `ReviewedAt`
3. `MoveReservedToConfirmed(jobID)` — atomic: `reserved_slots -= 1`, `confirmed_slots += 1`
4. `GetByIDForUpdate(job)` → if `IsCompletelyFull()` → `UpdateStatusInTx(FULL)`
5. COMMIT
6. Post-commit goroutines: `UpdateChannelJobPost`, `UpdateAdminJobPost`

### Service: RejectPayment

1. TX: `GetByIDForUpdate(booking)` → verify status == `PAYMENT_SUBMITTED`
2. Set status = `REJECTED`, `RejectionReason`, `ReviewedByAdminID`, `ReviewedAt`
3. `DecrementReservedSlots(jobID)` — releases the slot
4. COMMIT

---

## 7. Expiry Worker

### File: `service/expiry_worker.go` (267 lines)

### Architecture

- Runs as background goroutine via `go expiryWorker.Start()`
- 10-second ticker checks for expired bookings
- Stopped via `expiryWorker.Stop()` (closes channel)

### Processing Pipeline

```
Start() → ticker every 10s → safeProcessExpiredBookings()
  └── defer recover() (panic recovery wrapper)
  └── processExpiredBookings()
      └── context.WithTimeout(10s) → GetExpiredBookings(limit=100)
      └── for each booking:
          └── processExpiredBooking(booking)
              └── context.WithTimeout(10s)
              └── TX: MarkAsExpired + DecrementReservedSlots → COMMIT
              └── notifyUserExpiredSafe(booking)
                  └── goroutine with 15s timeout
                  └── defer recover()
                  └── notifyUserExpired: edit/delete payment instruction msg → send expiry msg
```

### Timeouts

| Operation | Timeout |
|---|---|
| DB query (GetExpiredBookings) | 10s |
| Per-booking transaction | 10s |
| Telegram notification | 15s |

### Notification Logic

- If `PaymentInstructionMsgID != 0`: try to edit the payment instruction message with expiry text; if edit fails, try delete then send new
- If no message ID: send new notification directly

---

## 8. Profile Management

### File: `bot/handlers/commands.go` (lines ~360-650)

### View Profile

`HandleUserProfile`: Fetches `RegisteredUser`, displays full name, phone, age, weight, height with inline edit buttons.

### Edit Profile

**Via reply keyboard buttons** (not inline):
1. User is on profile screen, sees reply keyboard: "👤 Ism familiya", "📞 Telefon raqami", "🎂 Yosh", "📏 Vazn va Bo'y", "🏠 Asosiy menyu"
2. User taps button → `HandleEditProfileField(field)` → sets `UserState` to `editing_profile_{field}`, sends prompt with current value
3. User types new value → `HandleText` → detects `editing_profile_` prefix → `HandleProfileEditInput`
4. Validates with same validators as registration → updates `RegisteredUser` → resets state to idle → shows updated profile

**Phone editing**: Supports both manual text input and contact sharing (`HandleContact` detects `StateEditingProfilePhone`)

**Cancel**: "❌ Bekor qilish" button → `HandleCancelProfileEdit` → resets state, shows current profile

**Back to main menu**: "🏠 Asosiy menyu" → `HandleBackToMainMenu` → resets state to idle, shows main menu reply keyboard

---

## 9. User Commands & Text Router

### File: `bot/handlers/commands.go` (653 lines)

### `/start` Command — `HandleStart`

Decision tree:
1. Get or create user in DB
2. Check deep link payload `job_` → if registered, start booking; if not, start registration with pending job
3. If admin → show admin panel
4. If regular user → start/continue registration

### `/help`, `/about`, `/settings` Commands

Simple static messages.

### `/admin` Command — `HandleAdminPanel`

Checks `IsAdmin`, shows admin menu reply keyboard.

### `HandleText` — Text Message Router

Priority order:
1. **"❌ Bekor qilish"** → if editing profile → cancel edit; else → cancel registration
2. **Registration flow** (`IsInRegistrationFlow`) → `HandleRegistrationTextInput`
3. **Job creation/editing** (admin, `creating_job_` or `editing_job_` prefix) → `HandleAdminTextInput`
4. **Profile editing** (`editing_profile_` prefix) → `HandleProfileEditInput`
5. **Admin menu buttons** (admin): "➕ Ish yaratish", "📋 Ishlar ro'yxati", "👥 Foydalanuvchilar", "📊 Statistika"
6. **User menu buttons**: "👤 Profil", "📋 Mening ishlarim", "❓ Yordam"
7. **Profile edit buttons**: "👤 Ism familiya", "📞 Telefon raqami", "🎂 Yosh", "📏 Vazn va Bo'y", "🏠 Asosiy menyu"
8. **Default**: if idle → ignore silently

### `HandleContact` — Contact Sharing

1. If user state == `RegStatePhone` → `HandleRegistrationContact`
2. If user state == `StateEditingProfilePhone` → validate + update phone
3. Otherwise → ignore

### `HandlePhoto` — Photo Messages

Routes to `HandlePaymentReceiptSubmission` with `photo.FileID`.

### `HandleLocation` — Location Messages

1. Only handled if user state == `StateCreatingJobLocation` or `StateEditingJobLocation`
2. Formats as `"lat,lng"` string → routes to creation or editing handler

### `HandleUserMyJobs`

Fetches all active bookings (SLOT_RESERVED, PAYMENT_SUBMITTED, CONFIRMED), displays formatted list with job details and status.

---

## 10. Admin: Job Creation

### File: `bot/handlers/admin.go` (lines 38-700)

### State Machine

```
"➕ Ish yaratish" → HandleCreateJob:
  state = creating_job_ish_haqqi, init temp job in session

State progression (each via HandleAdminTextInput → handleJobCreationInput):
  creating_job_ish_haqqi     → Salary (text)
  creating_job_ovqat         → Food (text)
  creating_job_vaqt          → WorkTime (text)
  creating_job_manzil        → Address (text)
  creating_job_location      → Location (Telegram location OR text, skippable)
  creating_job_xizmat_haqqi  → ServiceFee (integer only)
  creating_job_avtobuslar    → Buses (text, skippable)
  creating_job_ish_tavsifi   → AdditionalInfo (text)
  creating_job_ish_kuni      → WorkDate (text)
  creating_job_kerakli       → RequiredWorkers (integer, ≥1)
  creating_job_employer_phone → EmployerPhone (text) → SAVE TO DB
```

### On Final Step (EmployerPhone)

1. `storage.Job().Create()` — saves job with `status=ACTIVE`
2. Reset admin state to idle, clear temp job
3. Show job preview with `JobDetailKeyboard` (publish, edit, status, delete buttons)
4. Save admin job message ID to `admin_job_messages` table
5. `go notifyOtherAdminsNewJob()` — sends new job detail to all other admins

### Session Storage (session.go)

In-memory maps with `sync.RWMutex`:
- `tempJobs map[int64]*models.Job` — temp job during creation
- `editingJobIDs map[int64]int64` — which job admin is editing

### Cancellation

`HandleCancelJobCreation`: Resets state, clears temp job, shows admin panel.

### Skip Field

`HandleSkipField`: For optional fields (location, buses), sets empty value and advances to next step.

---

## 11. Admin: Job Management

### File: `bot/handlers/admin.go` (lines 150-1100)

### Job List

`HandleJobList`: Fetches all jobs (`GetAll`), shows inline keyboard with job entries.

### Job Detail

`HandleJobDetail(jobIDStr)`:
1. **Single-message enforcement per admin**: Deletes admin's previous message for this job
2. Sends new message with full job info + management keyboard
3. Saves message ID to `admin_job_messages` table (upsert by job_id + admin_id)

### Job Detail Keyboard

Shows contextual buttons based on job state:
- Edit fields (salary, food, time, address, location, service fee, buses, description, work date, workers, confirmed, employer phone)
- Status change: Open / Toldi / Closed
- Publish to channel (if not yet published)
- Delete channel message (if published)
- Delete job
- View bookings

### Edit Job Field

`HandleEditJobField(params)`:
1. Parse `{jobID}_{fieldName}` from callback data
2. Set admin state to `editing_job_{field}`, save editing job ID
3. Show prompt with current value
4. Admin types new value → `handleJobEditingInput` → validate → update DB → update channel message → update other admin messages → reset state → show updated job detail

### Change Job Status

`HandleChangeJobStatus(params)`:
- Parse `{jobID}_{statusStr}` (open/toldi/closed)
- Map: open→ACTIVE, toldi→FULL, closed→COMPLETED
- Update DB → update channel message → respond → update all admin messages → edit current admin's message

### Special: Edit Confirmed Slots

Admin can manually adjust `ConfirmedSlots`:
- Validates new value ≤ RequiredWorkers
- Auto-adjusts job status: if confirmed ≥ required → FULL; if was FULL and now < required → ACTIVE

### Publish to Channel

`HandlePublishJob(jobIDStr)`:
1. Format job for channel → send to `ChannelID`
2. Save `ChannelMessageID`
3. If job has location → send location as reply to channel message
4. Update all admin messages (shows "✅ Kanalga yuborilgan")

### Delete Channel Message

`HandleDeleteChannelMessage`: Deletes Telegram channel message, clears `ChannelMessageID`, updates admin views.

### Delete Job

`HandleDeleteJob`:
1. Delete channel message if exists
2. Delete ALL admin messages from Telegram
3. Delete job from DB (cascades to `admin_job_messages`)

### View Job Bookings

`HandleViewJobBookings(jobIDStr)`: Shows all users with PAYMENT_SUBMITTED or CONFIRMED status for the job, including full profile details.

### Admin Message Broadcasting

Helpers maintain consistency across multiple admins viewing the same job:
- `updateAllAdminMessages(job)` — edits all admins' messages for this job
- `updateOtherAdminMessages(jobID, excludeAdminID)` — same but excludes one admin
- `deleteAdminMessageForAdmin(jobID, adminID)` — deletes specific admin's message
- `deleteAllAdminMessages(jobID)` — deletes all on job deletion
- `notifyOtherAdminsNewJob(job, creatorID)` — sends new job to other admins

---

## 12. Admin: Payment Approval

### File: `bot/handlers/payment.go` (587 lines)

### Forward to Admin Group

`ForwardPaymentToAdminGroup(ctx, booking, receiptFileID)`:
1. Fetch job, registered user, telegram user details
2. Compose photo caption with full user info + job info + booking ID
3. Create inline keyboard: ✅ Tasdiqlash | ❌ Rad etish | 🚫 Bloklash
4. Send to `AdminGroupID` (separate group chat, not individual admin)
5. **Note**: Uses `h.bot.Send()` directly (not SenderService) — this is in the handler layer

### Approve Payment

`HandleApprovePayment(c, bookingIDStr)`:
1. Verify admin → parse booking ID → call `PaymentService.ApprovePayment()`
2. `go notifyUserPaymentApproved(booking)` — full job details + employer phone + location
3. Edit admin group message: append "✅ TASDIQLANDI" + admin name + timestamp, remove buttons

### Reject Payment

`HandleRejectPayment(c, bookingIDStr)`:
1. Verify admin → parse booking ID → call `PaymentService.RejectPayment(reason)` (hardcoded reason: "To'lov cheki noto'g'ri yoki aniq emas")
2. `go notifyUserPaymentRejected(booking)` — instructions to retry
3. Edit admin group message: append "❌ RAD ETILDI" + admin + time + reason, remove buttons

### Block User

`HandleBlockUser(c, params)`:
1. Parse `{userID}_{bookingID}` from callback data
2. Call `PaymentService.BlockUserAndRejectPayment()`
3. Get violation count → `go notifyUserViolation(userID, jobOrderNumber, violationCount)`
4. Edit admin group message: append "🚫 FOYDALANUVCHI BLOKLANDI", remove buttons

### User Notifications

**Approved**: Full job details including employer phone, location (sent as separate Telegram location message), next steps instructions.

**Rejected**: Job number, reason, retry instructions.

**Violation**: Progressive messages (see Section 14).

---

## 13. Admin: Statistics & User List

### Statistics

`HandleAdminStatistics`: Gathers counts from storage:
- Users: total bot users, registered, blocked
- Jobs: total, active, full, completed
- Bookings: total, confirmed, pending, rejected

### Registered Users List

`HandleRegisteredUsersList` → `showUsersListPage(page=1)`:
- Paginated (15 per page)
- Shows: full name, phone, age, weight, height, Telegram user ID, registration date
- Active/inactive status indicator
- Keyboard: ◀️ Previous | Page X/Y | ▶️ Next

---

## 14. Violation & Blocking System

### Service: `BlockUserAndRejectPayment` (payment.go)

**Progressive blocking:**

| Violations | Action |
|---|---|
| 1 | Warning only (no block) |
| 2 | 24-hour temporary block |
| 3+ | Permanent block (BlockedUntil = nil) |

### Database Records

**UserViolation**: `user_id`, `violation_type="fake_payment"`, `booking_id`, `admin_id`, `created_at`

**BlockedUser**: `user_id`, `blocked_until` (nil=permanent), `total_violations`, `blocked_by_admin_id`, `reason`, `created_at`, `updated_at`

### Block Check (in ConfirmBooking)

1. `GetBlockStatus` → if permanent → reject with message
2. If temporary and still active → reject with remaining time
3. If temporary and expired → `UnblockUser()` auto-unblock, continue with booking

### User Notifications (notifyUserViolation)

- **1st strike**: Warning message, explains consequences
- **2nd strike**: "24 SOAT BLOKLANGANSIZ", lists restrictions
- **3rd strike**: "DOIMIY BLOKLANGANSIZ", appeal instructions

---

## 15. Sender Service

### File: `service/sender.go` (270 lines)

### Purpose

Centralizes all Telegram message sending. Planned: queue-based sending with rate limiting (currently direct).

### Context-Based Methods (immediate response in handlers)

| Method | Usage |
|---|---|
| `Reply(c, msg, opts...)` | `c.Send()` wrapper |
| `ReplyWithPhoto(c, photo, opts...)` | Photo send |
| `EditMessage(c, msg, opts...)` | `c.Edit()` wrapper |
| `Respond(c, response)` | Callback query response |
| `RemoveKeyboard(c)` | Sends zero-width space with RemoveKeyboard |
| `DeleteMessage(c)` | `c.Delete()` wrapper |

### Direct Methods (background/async use)

| Method | Usage |
|---|---|
| `Send(ctx, chatID, msg, opts...)` | Direct send to chat |
| `SendPhoto(ctx, chatID, photo, opts...)` | Direct photo send |
| `Edit(ctx, chatID, msgID, msg, opts...)` | Direct edit |

### Broadcast Methods

| Method | Usage |
|---|---|
| `UpdateChannelJobPost(ctx, job)` | Updates channel message with latest job info |
| `UpdateAdminJobPost(ctx, job)` | Updates all admin messages for a job |

### Notes

- Mutex was removed (Telegram API is thread-safe)
- Queue implementation is stubbed out (commented code, `useQueue=false`)
- `UpdateAdminJobPost` auto-cleans stale messages (deletes from DB on "message not found" error)

---

## 16. Storage Layer

### File: `storage/storage.go` (220 lines) — Interfaces

**Repositories**: `UserRepoI`, `JobRepoI`, `BookingRepoI`, `RegistrationRepoI`, `AdminMessageRepoI`, `TransactionI`

### Transaction Pattern

```go
tx, err := s.storage.Transaction().Begin(ctx)
defer s.storage.Transaction().Rollback(ctx, tx) // harmless no-op after commit
// ... operations with tx ...
s.storage.Transaction().Commit(ctx, tx)
```

Isolation level: READ COMMITTED.  
`TransactionI` uses `any` for tx parameter (wraps `pgx.Tx`).

### Key Storage Operations

**JobRepoI critical methods:**
- `GetByIDForUpdate(ctx, tx, id)` — `SELECT ... FOR UPDATE` (row locking)
- `IncrementReservedSlots(ctx, tx, jobID)` — atomic `UPDATE SET reserved_slots = reserved_slots + 1 WHERE reserved_slots + confirmed_slots < required_workers`
- `DecrementReservedSlots(ctx, tx, jobID)` — atomic `UPDATE SET reserved_slots = reserved_slots - 1 WHERE reserved_slots > 0`
- `MoveReservedToConfirmed(ctx, tx, jobID)` — atomic `UPDATE SET reserved_slots = reserved_slots - 1, confirmed_slots = confirmed_slots + 1`

**BookingRepoI critical methods:**
- `GetByIDForUpdate(ctx, tx, id)` — row lock for payment approval
- `GetByIdempotencyKey(ctx, tx, key)` — idempotency check
- `GetExpiredBookings(ctx, limit)` — `WHERE status = 'SLOT_RESERVED' AND expires_at < NOW()`
- `MarkAsExpired(ctx, tx, id)` — `UPDATE SET status = 'EXPIRED'`

### Implementations: `storage/postgres/`

Each file implements the corresponding interface using `pgxpool.Pool` and raw SQL.

---

## 17. Data Models

### File: `bot/models/user.go`

**User**: `ID` (Telegram user ID), `Username`, `FirstName`, `LastName`, `State` (UserState), `CreatedAt`, `UpdatedAt`

**UserState constants**: `idle`, `creating_job_*` (11 states), `editing_job_*` (12 states), `editing_profile_*` (4 states)

**UserViolation**: violation record  
**BlockedUser**: block record with optional `BlockedUntil`

### File: `bot/models/job.go`

**Job**:
- Details: `Salary`, `Food`, `WorkTime`, `Address`, `Location`, `ServiceFee`, `Buses`, `AdditionalInfo`, `WorkDate`, `EmployerPhone`
- Slots: `RequiredWorkers`, `ReservedSlots`, `ConfirmedSlots`
- Metadata: `Status`, `ChannelMessageID`, `AdminMessageID`, `OrderNumber`, `CreatedByAdminID`

**JobStatus**: `DRAFT`, `ACTIVE`, `FULL`, `COMPLETED`, `CANCELLED`

**Helper methods**: `AvailableSlots()`, `IsFull()`, `IsCompletelyFull()`, `IsActive()`

### File: `bot/models/booking.go`

**JobBooking**:
- Core: `ID`, `JobID`, `UserID`, `Status`
- Payment: `PaymentReceiptFileID`, `PaymentReceiptMsgID`, `PaymentInstructionMsgID`
- Timing: `ReservedAt`, `ExpiresAt` (3 min), `PaymentSubmittedAt`, `ConfirmedAt`
- Admin: `ReviewedByAdminID`, `ReviewedAt`, `RejectionReason`
- Idempotency: `IdempotencyKey` = `"user_{id}_job_{id}"`

**BookingStatus**: `SLOT_RESERVED`, `PAYMENT_SUBMITTED`, `CONFIRMED`, `REJECTED`, `EXPIRED`, `CANCELLED_BY_USER`

**Helper methods**: `IsExpired()`, `CanSubmitPayment()`, `CanBeApproved()`, `TimeRemaining()`

### File: `bot/models/registration.go`

**RegistrationDraft**: Temp registration data with state machine
- Fields: `FullName`, `Phone`, `Age`, `Weight`, `Height`, `PassportPhotoID`, `PendingJobID`
- States: `reg_public_offer`, `reg_full_name`, `reg_phone`, `reg_age`, `reg_body_params`, `reg_confirm`, `reg_declined`, `reg_completed`
- `PreviousState` (in-memory only): tracks edit mode (not persisted to DB)

**RegisteredUser**: Final registered user data (separate table from draft)

### File: `bot/models/admin_message.go`

**AdminJobMessage**: `JobID`, `AdminID`, `MessageID` — maps each admin to their Telegram message for a specific job.

---

## 18. Validation

### File: `pkg/validation/validation.go` (365 lines)

| Function | Rules |
|---|---|
| `ValidateFullName(text)` | Non-empty, 3-100 chars, no digits, no emojis, only letters/spaces/hyphens, ≥2 words |
| `ValidateAge(text)` | Integer, 16-65 |
| `ValidatePhone(phone)` | Must start with `+998`, exactly 13 chars, all digits after `+` |
| `ValidateWeight(text)` | Integer, 30-200 kg |
| `ValidateHeight(text)` | Integer, 100-250 cm |
| `ParseBodyParams(text)` | Parses "weight height" format, validates both |
| `NormalizeFullName(text)` | Trims, collapses whitespace, title-cases each word |
| `NormalizePhone(phone)` | Adds `+` prefix if missing |

**Returns**: `*ValidationError` with `Field` and user-friendly `Message` (in Uzbek).

---

## 19. Configuration

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `BOT_TOKEN` | (required) | Telegram bot token |
| `BOT_CHANNEL_ID` | 0 | Channel ID for job posts |
| `BOT_ADMIN_IDS` | (required) | Comma-separated admin Telegram IDs |
| `BOT_ADMIN_GROUP_ID` | 0 | Group chat for payment approvals |
| `BOT_USERNAME` | "" | Bot username (for deep links) |
| `BOT_MODE` | "polling" | "polling" or "webhook" |
| `BOT_WEBHOOK_URL` | "" | Public webhook URL |
| `BOT_WEBHOOK_PORT` | 8443 | Webhook listener port |
| `BOT_RATE_LIMIT_MAX` | 30 | Max requests per window |
| `BOT_RATE_LIMIT_WINDOW` | 60s | Rate limit window |
| `DB_HOST/PORT/USER/PASSWORD/NAME` | localhost:5432/postgres | PostgreSQL connection |
| `DB_MAX_CONNECTIONS` | 25 | Pool max connections |
| `CARD_NUMBER` | "8600..." | Payment card number |
| `CARD_HOLDER_NAME` | "ADMIN NAME" | Card holder name |
| `APP_ENV` | "development" | Environment |
| `LOG_LEVEL` | "info" | Log level |

---

## 20. Known Issues & Review Targets

### High Priority

1. **ForwardPaymentToAdminGroup uses `h.bot.Send()` directly** (payment.go line ~130) — violates the "all sends via SenderService" rule. Also `notifyUserPaymentApproved/Rejected/Blocked` and `notifyUserViolation` all use `h.bot.Send()` directly.

2. **`RegistrationDraft.PreviousState` not persisted** — stored only in-memory. If bot restarts during registration edit, user loses edit-mode context and can't return to confirmation. Field is tagged `db:"-"`.

3. **Session storage is in-memory** (`session.go`) — `tempJobs` and `editingJobIDs` maps are lost on restart. Admin mid-creation job data is gone.

4. **Duplicate admin check functions** — `h.IsAdmin()` (exported, used in most places) and `h.isAdmin()` (unexported, used in payment.go). Same logic: `slices.Contains(h.cfg.Bot.AdminIDs, userID)`.

5. **Handler directly accesses storage** — Many handlers call `h.storage.*` directly instead of going through services (e.g., `HandleUserProfile`, `HandleEditProfileField`, `HandleProfileEditInput`, `HandleText`, `HandleContact`, `HandleAdminStatistics`, etc.). This breaks the clean architecture layering.

6. **No validation on admin job creation fields** — Salary, food, work time, address, etc. are accepted as-is with no validation. Only `ServiceFee` (must be integer) and `RequiredWorkers` (≥1) are validated.

7. **Hardcoded rejection reason** — `HandleRejectPayment` always uses fixed reason "To'lov cheki noto'g'ri yoki aniq emas". Admin can't specify custom reason.

### Medium Priority

8. **Race condition in HandleBookingConfirm idempotency check** — The pre-service idempotency check in the handler (lines ~140-160) is done without a transaction/lock. Between that check and `ConfirmBooking()`, another booking could be created. The service has its own idempotency check inside the transaction, but the handler's check is informational only.

9. **`time.Now().Add(time.Hour*5)` hardcoded UTC+5** — Used in payment.go for display timestamps. Should use proper timezone handling.

10. **GetExpiredBookings has no FOR UPDATE** — By design (was removed in the hang fix), but means two concurrent expiry workers could process the same booking. Currently only one worker exists, but fragile.

11. **Profile editing state not checked on /start** — If user is in `editing_profile_*` state and sends `/start`, the state isn't reset. Could cause confusion.

12. **Pool min/max connections hardcoded** — `postgres.go` hardcodes `MinConns: 20, MaxConns: 200` regardless of `DB_MAX_CONNECTIONS` config.

### Low Priority

13. **MsgError is in English** — `"⚠️ Something went wrong. Please try again later."` — should be Uzbek per project convention.

14. **`HandlePaymentSubmission` is dead code** — payment.go lines 18-37, never called (photos go through `HandlePhoto` → `HandlePaymentReceiptSubmission` in commands.go).

15. **Mixed HTML/Markdown** — Registration summary uses Markdown (`*bold*`), everything else uses HTML (`<b>bold</b>`). Inconsistent.

16. **No graceful shutdown for background goroutines in payment.go** — `go h.notifyUserPaymentApproved(booking)` and similar fire-and-forget goroutines. On shutdown, they may not complete.

---

## File Reference Map

| Area | Handler | Service | Storage |
|---|---|---|---|
| Registration | registration.go (523 lines) | registration.go (532 lines) | registration.go |
| Booking | booking.go (260 lines) | booking.go (250 lines) | booking.go |
| Payment | payment.go (587 lines) | payment.go (350 lines) | booking.go, job.go, user.go |
| Expiry | — | expiry_worker.go (267 lines) | booking.go, job.go |
| Profile | commands.go (360-650) | — (direct storage) | registration.go, user.go |
| Job Creation | admin.go (38-700) | — (direct storage) | job.go |
| Job Management | admin.go (150-1100) | — (direct storage) | job.go, admin_message.go |
| Admin Panel | admin.go (28-180) | — (direct storage) | user.go, job.go, booking.go, registration.go |
| User Commands | commands.go (1-170) | — | user.go |
| Callbacks | callback_router.go, callbacks.go | — | user.go |
| Sending | — | sender.go (270 lines) | admin_message.go |
| Middleware | recovery.go, rate_limiter.go | — | — |
| Models | models/*.go | — | — |
| Config | — | — | config/config.go |
| Validation | — | — | pkg/validation/ |
| Messages | — | — | pkg/messages/ |
