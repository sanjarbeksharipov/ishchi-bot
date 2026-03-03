# Fix: Bot Hang After Processing Expired Bookings

**Date:** 2026-03-03  
**Symptom:** Bot stopped responding to all messages after logging `"Processing expired bookings" count=1` at 09:00:19 UTC. Container stayed alive but the bot was completely unresponsive for ~30 minutes until manually restarted.  
**PostgreSQL logs confirmed:** Two connections had open transactions when the bot was killed (`unexpected EOF on client connection with an open transaction`).



## Root Cause

A combination of 6 issues created a deadlock/hang scenario during expired booking processing:

1. The expiry worker's DB operations used `context.Background()` with **no timeout** — a stuck query or lock wait would block the goroutine forever.
2. All transactions used **SERIALIZABLE isolation**, which adds SSI predicate locks on top of `FOR UPDATE` row locks. Under concurrent load (multiple bookings on job #11), this caused serialization conflicts and retryable errors that were not retried.
3. The `GetExpiredBookings` query had `FOR UPDATE SKIP LOCKED` **outside any transaction**, making the locks meaningless and causing unpredictable interactions with concurrent transactions.
4. The expiry worker had **no panic recovery** — a panic would silently kill the goroutine (container stays up, bot stops responding).
5. The `SenderService` used a **single global `sync.Mutex`** for all message sending. One slow/hung Telegram API call would block every other Send, Edit, and notification in the entire bot.
6. Transaction rollback patterns were **broken across all services** — the `defer func() { if err != nil { Rollback() } }()` pattern fails silently when inner `if err :=` creates a new scope, leaving `err` as `nil` in the defer. Failed commits leaked connections.

---

## Fixes Applied

### Fix 1: Transaction Isolation Level
**File:** `storage/postgres/transaction.go`  
**Change:** `pgx.Serializable` → `pgx.ReadCommitted`  
**Why:** All critical sections already use `FOR UPDATE` row locks for correctness. SERIALIZABLE added SSI predicate locks that caused unnecessary serialization failures under concurrent load, with no retry logic to handle them. READ COMMITTED + FOR UPDATE provides the same correctness guarantees without the conflict risk.

### Fix 2: Remove Invalid FOR UPDATE SKIP LOCKED  
**File:** `storage/postgres/booking.go` — `GetExpiredBookings()`  
**Change:** Removed `FOR UPDATE SKIP LOCKED` clause  
**Why:** This query runs outside a transaction (bare `r.db.Query`), so `FOR UPDATE` locks are released immediately after the query completes — they protect nothing. The single-threaded expiry worker processes each booking in its own transaction with proper locking via `MarkAsExpired`. The dangling locks could interact unpredictably with concurrent SERIALIZABLE transactions.

### Fix 3: Expiry Worker Hardening  
**File:** `service/expiry_worker.go` — Full rewrite  
**Changes:**
- **Panic recovery:** `safeProcessExpiredBookings()` wraps each tick with `defer recover()` so a panic logs a stack trace instead of killing the goroutine.
- **DB timeouts:** Every DB operation uses `context.WithTimeout(ctx, 10s)` instead of `context.Background()`. A stuck query now fails after 10 seconds instead of blocking forever.
- **Notification timeout:** `notifyUserExpiredSafe()` runs the Telegram API call in a goroutine with a 15-second deadline. A hung API call no longer blocks the worker loop.
- **Unconditional defer rollback:** Replaced the broken manual `rollback()` helper with `defer w.storage.Transaction().Rollback(ctx, tx)`. This guarantees cleanup even if Commit fails (pgx's Rollback is a no-op after successful Commit).

### Fix 4: Remove Global Mutex from SenderService  
**File:** `service/sender.go`  
**Change:** Removed `sync.Mutex` field and all `s.mu.Lock()`/`s.mu.Unlock()` calls from `Send()`, `SendPhoto()`, `Edit()`, `UpdateChannelJobPost()`, `UpdateAdminJobPost()`  
**Why:** Telegram's HTTP API is inherently thread-safe — concurrent calls to `bot.Send()` or `bot.Edit()` from different goroutines are safe. The mutex serialized ALL outgoing messages behind a single lock, meaning one slow Telegram API response (e.g., network timeout) would block every other message in the bot, including handler responses and expiry notifications.

### Fix 5: PostgreSQL Connection-Level Timeouts  
**File:** `storage/postgres/postgres.go`  
**Change:** Added `AfterConnect` hook that runs `SET statement_timeout = '30s'; SET lock_timeout = '10s'` on every new connection  
**Why:** This is a safety net. Even if application code forgets to set a context timeout, no single SQL statement can run longer than 30 seconds and no lock wait can exceed 10 seconds. Without this, a single stuck query could hold a connection from the pool indefinitely, eventually exhausting all connections and hanging the entire bot.

### Fix 6: Broken Transaction Rollback Pattern  
**Files:** `service/booking.go`, `service/payment.go`, `bot/handlers/booking.go` (8 transaction sites total)  
**Change:** Replaced all `defer func() { if err != nil { Rollback() } }()` with unconditional `defer Rollback()`  
**Why:** The conditional pattern was silently broken. In Go, `if err := someFunc(); err != nil { return err }` creates a **new scoped `err` variable** — it does NOT assign to the outer `err` that the defer closure captures. So the defer always saw `err == nil` and never rolled back. With the unconditional pattern, `defer Rollback()` always fires — and pgx's `Rollback()` after a successful `Commit()` is a harmless no-op that returns `ErrTxClosed`.

---

## Timeline Reconstruction

```
08:52:42  Booking #48 confirmed (user 7255739605, job 11)
08:53:14  Payment #48 submitted
08:53:21  Payment #48 approved → channel + admin messages updated (via mutex-locked sender)
08:53:39  3 new users start registration for job 11 (concurrent load spike)
08:53:49  Booking #49 confirmed (user 8446127076, job 11)
08:56:59  Booking #49 expired → successfully released (expiry worker OK)
08:57:12  Booking #50 confirmed (user 8589040907, job 11)  
08:57:15  Another user gets "all slots reserved" error
09:00:19  Expiry worker picks up 1 expired booking → HANGS HERE
          ├─ Likely cause: SERIALIZABLE conflict with concurrent transaction on job 11
          ├─ No context timeout → blocked indefinitely on DB
          ├─ OR: Telegram notification call hung → blocked worker (no timeout)
          └─ Mutex in SenderService may have amplified the deadlock
09:30:00  Manual restart initiated
09:30:10  PostgreSQL sees 2 connections killed with open transactions (leaked)
09:30:05  Bot stopped
```

---

## Files Modified

| File | Lines Changed |
|------|--------------|
| `storage/postgres/transaction.go` | ~3 |
| `storage/postgres/booking.go` | ~5 |
| `storage/postgres/postgres.go` | ~6 |
| `service/expiry_worker.go` | ~80 (rewrite) |
| `service/sender.go` | ~30 |
| `service/booking.go` | ~12 |
| `service/payment.go` | ~20 |
| `bot/handlers/booking.go` | ~3 |
