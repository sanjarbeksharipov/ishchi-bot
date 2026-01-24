# Job Booking System - Technical Design

## Overview
This document describes the race-condition-safe design for a job booking system where users compete for limited worker slots with payment confirmation within 3 minutes.

---

## 1. State Machine

### Job States
```
DRAFT          -> Admin creating job (not published yet)
ACTIVE         -> Published to channel, accepting bookings
FULL           -> All slots taken (reserved + confirmed = required_workers)
COMPLETED      -> Job finished
CANCELLED      -> Job cancelled by admin
```

### User Booking States
```
INITIAL                    -> User clicked job link
SLOT_RESERVED              -> Slot temporarily held (3 min timer starts)
PAYMENT_SUBMITTED          -> Receipt uploaded, waiting admin approval
CONFIRMED                  -> Admin approved, slot locked
REJECTED                   -> Admin rejected payment
EXPIRED                    -> 3-minute timer ran out
CANCELLED_BY_USER          -> User cancelled before payment
```

### State Transitions
```
INITIAL 
  → SLOT_RESERVED (if slot available via atomic increment)
  → REJECTED (if no slots available)

SLOT_RESERVED
  → PAYMENT_SUBMITTED (receipt uploaded within 3 min)
  → EXPIRED (3 min timeout, auto-release)
  → CANCELLED_BY_USER (user cancels)

PAYMENT_SUBMITTED
  → CONFIRMED (admin approves)
  → REJECTED (admin rejects)
  → EXPIRED (admin doesn't respond + cleanup job runs)

EXPIRED/REJECTED/CANCELLED_BY_USER
  → Released (slot counter decremented atomically)
```

---

## 2. Database Schema

### Table: `jobs`
```sql
CREATE TABLE jobs (
    id BIGSERIAL PRIMARY KEY,
    order_number INT NOT NULL UNIQUE,
    
    -- Job details
    salary VARCHAR(255) NOT NULL,
    food VARCHAR(255),
    work_time VARCHAR(255) NOT NULL,
    address VARCHAR(500) NOT NULL,
    service_fee INT NOT NULL,
    buses VARCHAR(255),
    additional_info TEXT,
    work_date VARCHAR(100) NOT NULL,
    
    -- Capacity management (CRITICAL for race conditions)
    required_workers INT NOT NULL,              -- Total needed (e.g., 3)
    reserved_slots INT NOT NULL DEFAULT 0,      -- Currently reserved (temp holds)
    confirmed_slots INT NOT NULL DEFAULT 0,     -- Admin-confirmed bookings
    
    -- Status
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT',
    
    -- Telegram
    channel_message_id BIGINT,                  -- Message ID in channel
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT check_slots_valid CHECK (
        reserved_slots >= 0 AND 
        confirmed_slots >= 0 AND 
        reserved_slots + confirmed_slots <= required_workers
    )
);

CREATE INDEX idx_jobs_status ON jobs(status);
CREATE INDEX idx_jobs_order_number ON jobs(order_number);
```

### Table: `job_bookings`
```sql
CREATE TABLE job_bookings (
    id BIGSERIAL PRIMARY KEY,
    job_id BIGINT NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    user_id BIGINT NOT NULL REFERENCES registered_users(user_id) ON DELETE CASCADE,
    
    -- State tracking
    status VARCHAR(50) NOT NULL DEFAULT 'SLOT_RESERVED',
    
    -- Payment
    payment_receipt_file_id VARCHAR(255),       -- Telegram file ID
    payment_receipt_message_id BIGINT,          -- For editing later
    
    -- Timing (CRITICAL for expiry)
    reserved_at TIMESTAMP NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMP NOT NULL,              -- reserved_at + 3 minutes
    payment_submitted_at TIMESTAMP,
    confirmed_at TIMESTAMP,
    
    -- Admin action
    reviewed_by_admin_id BIGINT REFERENCES users(id),
    reviewed_at TIMESTAMP,
    rejection_reason TEXT,
    
    -- Idempotency (CRITICAL for Telegram retries)
    idempotency_key VARCHAR(255) UNIQUE,        -- Format: "user_{user_id}_job_{job_id}"
    
    -- Timestamps
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    
    -- Only one active booking per user per job
    CONSTRAINT unique_user_job_booking UNIQUE(job_id, user_id)
);

CREATE INDEX idx_job_bookings_status ON job_bookings(status);
CREATE INDEX idx_job_bookings_expires_at ON job_bookings(expires_at) WHERE status = 'SLOT_RESERVED';
CREATE INDEX idx_job_bookings_job_id ON job_bookings(job_id);
CREATE INDEX idx_job_bookings_user_id ON job_bookings(user_id);
CREATE INDEX idx_job_bookings_idempotency ON job_bookings(idempotency_key);
```

---

## 3. Race-Condition-Safe Slot Management

### Problem: Simultaneous Bookings
5 users click at the exact same time for a job with `required_workers = 3`.

### Solution: PostgreSQL Atomic Operations + Row-Level Locking

#### Strategy 1: SELECT FOR UPDATE + Atomic Increment (RECOMMENDED)

```go
func (s *JobService) ReserveSlot(ctx context.Context, userID, jobID int64) (*Booking, error) {
    // Generate idempotency key FIRST
    idempotencyKey := fmt.Sprintf("user_%d_job_%d", userID, jobID)
    
    // Start transaction with SERIALIZABLE isolation
    tx, err := s.db.BeginTx(ctx, &sql.TxOptions{
        Isolation: sql.LevelSerializable, // Strongest isolation
    })
    if err != nil {
        return nil, err
    }
    defer tx.Rollback()
    
    // CRITICAL: Lock the job row to prevent concurrent modifications
    var job Job
    err = tx.QueryRow(`
        SELECT id, required_workers, reserved_slots, confirmed_slots, status
        FROM jobs
        WHERE id = $1 AND status = 'ACTIVE'
        FOR UPDATE  -- EXCLUSIVE LOCK - blocks other transactions
    `, jobID).Scan(&job.ID, &job.RequiredWorkers, &job.ReservedSlots, &job.ConfirmedSlots, &job.Status)
    
    if err == sql.ErrNoRows {
        return nil, ErrJobNotFound
    }
    if err != nil {
        return nil, err
    }
    
    // Check if slots available
    totalOccupied := job.ReservedSlots + job.ConfirmedSlots
    if totalOccupied >= job.RequiredWorkers {
        return nil, ErrJobFull
    }
    
    // Check for existing booking (idempotency)
    var existingBooking Booking
    err = tx.QueryRow(`
        SELECT id, status, reserved_at, expires_at
        FROM job_bookings
        WHERE idempotency_key = $1
    `, idempotencyKey).Scan(&existingBooking.ID, &existingBooking.Status, 
        &existingBooking.ReservedAt, &existingBooking.ExpiresAt)
    
    if err == nil {
        // Booking already exists (Telegram retry or duplicate click)
        if existingBooking.Status == "SLOT_RESERVED" && time.Now().Before(existingBooking.ExpiresAt) {
            // Still valid, return existing booking
            tx.Commit()
            return &existingBooking, nil
        } else if existingBooking.Status == "EXPIRED" {
            // Previous booking expired, allow re-booking
            // (Will be handled below with INSERT ON CONFLICT)
        } else {
            // Already confirmed/rejected/submitted
            return nil, ErrAlreadyBooked
        }
    }
    
    // Atomically increment reserved_slots
    expiresAt := time.Now().Add(3 * time.Minute)
    
    _, err = tx.Exec(`
        UPDATE jobs
        SET reserved_slots = reserved_slots + 1,
            updated_at = NOW()
        WHERE id = $1
    `, jobID)
    if err != nil {
        return nil, err
    }
    
    // Create booking record with idempotency
    var booking Booking
    err = tx.QueryRow(`
        INSERT INTO job_bookings (
            job_id, user_id, status, reserved_at, expires_at, idempotency_key
        ) VALUES ($1, $2, 'SLOT_RESERVED', NOW(), $3, $4)
        ON CONFLICT (idempotency_key) 
        DO UPDATE SET 
            status = 'SLOT_RESERVED',
            reserved_at = NOW(),
            expires_at = $3,
            updated_at = NOW()
        RETURNING id, job_id, user_id, status, reserved_at, expires_at
    `, jobID, userID, expiresAt, idempotencyKey).Scan(
        &booking.ID, &booking.JobID, &booking.UserID, 
        &booking.Status, &booking.ReservedAt, &booking.ExpiresAt,
    )
    if err != nil {
        return nil, err
    }
    
    // Commit transaction
    if err := tx.Commit(); err != nil {
        return nil, err
    }
    
    // Start background goroutine for expiry (non-blocking)
    go s.scheduleExpiry(booking.ID, booking.ExpiresAt)
    
    return &booking, nil
}
```

#### Why This Works:
1. **SELECT FOR UPDATE**: Creates exclusive row lock on the job. Other transactions must wait.
2. **SERIALIZABLE isolation**: Prevents phantom reads and ensures transaction sees consistent snapshot.
3. **Atomic increment**: `reserved_slots = reserved_slots + 1` is safe within transaction.
4. **CHECK constraint**: Database enforces `reserved_slots + confirmed_slots <= required_workers`.
5. **Idempotency key**: Prevents duplicate bookings from Telegram retries.
6. **ON CONFLICT**: Handles re-booking after expiry gracefully.

#### Execution Flow for 5 Concurrent Users:
```
T0: User1, User2, User3, User4, User5 click simultaneously
T1: User1 acquires lock → reads (reserved=0, confirmed=0) → increments → reserved=1 → commits
T2: User2 waits for lock → acquires → reads (reserved=1, confirmed=0) → increments → reserved=2 → commits  
T3: User3 waits → acquires → reads (reserved=2, confirmed=0) → increments → reserved=3 → commits
T4: User4 waits → acquires → reads (reserved=3, confirmed=0) → FULL → rollback → returns ErrJobFull
T5: User5 waits → acquires → reads (reserved=3, confirmed=0) → FULL → rollback → returns ErrJobFull
```

**Result**: Exactly 3 bookings created. Users 4 and 5 get immediate rejection.

---

## 4. Expiry Management

### Problem: Timer expires, need to release slot

### Solution: Background Worker + Database-Driven Expiry

#### Approach: Scheduled Task (Cron-like)

```go
// Background worker runs every 10 seconds
func (s *JobService) StartExpiryWorker(ctx context.Context) {
    ticker := time.NewTicker(10 * time.Second)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            s.processExpiredBookings(ctx)
        }
    }
}

func (s *JobService) processExpiredBookings(ctx context.Context) {
    // Find all expired reservations
    rows, err := s.db.Query(`
        SELECT id, job_id, user_id, payment_receipt_message_id
        FROM job_bookings
        WHERE status = 'SLOT_RESERVED'
          AND expires_at < NOW()
        FOR UPDATE SKIP LOCKED  -- Skip rows locked by other workers (for multi-instance)
    `)
    if err != nil {
        log.Error("Failed to query expired bookings", err)
        return
    }
    defer rows.Close()
    
    for rows.Next() {
        var bookingID, jobID, userID, messageID int64
        if err := rows.Scan(&bookingID, &jobID, &userID, &messageID); err != nil {
            continue
        }
        
        // Process each expired booking in separate transaction
        s.expireBooking(ctx, bookingID, jobID, userID, messageID)
    }
}

func (s *JobService) expireBooking(ctx context.Context, bookingID, jobID, userID, messageID int64) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Update booking status
    _, err = tx.Exec(`
        UPDATE job_bookings
        SET status = 'EXPIRED', updated_at = NOW()
        WHERE id = $1 AND status = 'SLOT_RESERVED'
    `, bookingID)
    if err != nil {
        return err
    }
    
    // Atomically decrement reserved_slots
    _, err = tx.Exec(`
        UPDATE jobs
        SET reserved_slots = GREATEST(reserved_slots - 1, 0),
            updated_at = NOW()
        WHERE id = $1
    `, jobID)
    if err != nil {
        return err
    }
    
    if err := tx.Commit(); err != nil {
        return err
    }
    
    // Send Telegram notification (non-blocking, outside transaction)
    go s.notifyUserExpired(userID, messageID)
    
    return nil
}
```

#### Why This Works:
- **Database-driven**: Uses `expires_at` column, survives bot restarts
- **FOR UPDATE SKIP LOCKED**: Safe for multi-instance deployments
- **Atomic decrement**: Thread-safe slot release
- **Separate transactions**: One failed expiry doesn't block others

---

## 5. Payment Submission

```go
func (s *JobService) SubmitPayment(ctx context.Context, bookingID int64, receiptFileID string, messageID int64) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Verify booking is still valid
    var status string
    var expiresAt time.Time
    err = tx.QueryRow(`
        SELECT status, expires_at
        FROM job_bookings
        WHERE id = $1
        FOR UPDATE
    `, bookingID).Scan(&status, &expiresAt)
    
    if err != nil {
        return err
    }
    
    // Check if still in valid state
    if status != "SLOT_RESERVED" {
        return ErrInvalidBookingState
    }
    
    if time.Now().After(expiresAt) {
        // Too late, already expired (race between user and cron)
        return ErrBookingExpired
    }
    
    // Update to payment submitted (stops expiry timer)
    _, err = tx.Exec(`
        UPDATE job_bookings
        SET status = 'PAYMENT_SUBMITTED',
            payment_receipt_file_id = $2,
            payment_receipt_message_id = $3,
            payment_submitted_at = NOW(),
            updated_at = NOW()
        WHERE id = $1
    `, bookingID, receiptFileID, messageID)
    
    if err != nil {
        return err
    }
    
    return tx.Commit()
}
```

---

## 6. Admin Approval/Rejection

```go
func (s *JobService) ApprovePayment(ctx context.Context, bookingID, adminID int64) error {
    tx, err := s.db.BeginTx(ctx, &sql.TxOptions{
        Isolation: sql.LevelSerializable,
    })
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    // Lock booking
    var jobID int64
    var status string
    err = tx.QueryRow(`
        SELECT job_id, status
        FROM job_bookings
        WHERE id = $1
        FOR UPDATE
    `, bookingID).Scan(&jobID, &status)
    
    if err != nil {
        return err
    }
    
    if status != "PAYMENT_SUBMITTED" {
        return ErrInvalidBookingState
    }
    
    // Update booking to confirmed
    _, err = tx.Exec(`
        UPDATE job_bookings
        SET status = 'CONFIRMED',
            confirmed_at = NOW(),
            reviewed_by_admin_id = $2,
            reviewed_at = NOW(),
            updated_at = NOW()
        WHERE id = $1
    `, bookingID, adminID)
    
    if err != nil {
        return err
    }
    
    // Move slot from reserved to confirmed (atomic swap)
    _, err = tx.Exec(`
        UPDATE jobs
        SET reserved_slots = GREATEST(reserved_slots - 1, 0),
            confirmed_slots = confirmed_slots + 1,
            updated_at = NOW()
        WHERE id = $1
    `, jobID)
    
    if err != nil {
        return err
    }
    
    // Check if job is now full
    var requiredWorkers, confirmedSlots int
    err = tx.QueryRow(`
        SELECT required_workers, confirmed_slots
        FROM jobs
        WHERE id = $1
    `, jobID).Scan(&requiredWorkers, &confirmedSlots)
    
    if confirmedSlots >= requiredWorkers {
        // Mark job as FULL
        _, err = tx.Exec(`
            UPDATE jobs
            SET status = 'FULL',
                updated_at = NOW()
            WHERE id = $1
        `, jobID)
        
        if err != nil {
            return err
        }
    }
    
    return tx.Commit()
}

func (s *JobService) RejectPayment(ctx context.Context, bookingID, adminID int64, reason string) error {
    tx, err := s.db.BeginTx(ctx, nil)
    if err != nil {
        return err
    }
    defer tx.Rollback()
    
    var jobID int64
    var status string
    err = tx.QueryRow(`
        SELECT job_id, status
        FROM job_bookings
        WHERE id = $1
        FOR UPDATE
    `, bookingID).Scan(&jobID, &status)
    
    if err != nil {
        return err
    }
    
    if status != "PAYMENT_SUBMITTED" {
        return ErrInvalidBookingState
    }
    
    // Mark as rejected
    _, err = tx.Exec(`
        UPDATE job_bookings
        SET status = 'REJECTED',
            rejection_reason = $3,
            reviewed_by_admin_id = $2,
            reviewed_at = NOW(),
            updated_at = NOW()
        WHERE id = $1
    `, bookingID, adminID, reason)
    
    if err != nil {
        return err
    }
    
    // Release the reserved slot
    _, err = tx.Exec(`
        UPDATE jobs
        SET reserved_slots = GREATEST(reserved_slots - 1, 0),
            updated_at = NOW()
        WHERE id = $1
    `, jobID)
    
    return tx.Commit()
}
```

---

## 7. Edge Cases & Solutions

### 7.1 Duplicate Telegram Updates (Retries)
**Problem**: Telegram may send same update twice if bot doesn't respond quickly.

**Solution**: 
- Use `idempotency_key` in `job_bookings` table
- Format: `user_{userID}_job_{jobID}`
- `UNIQUE` constraint prevents duplicates
- `ON CONFLICT DO UPDATE` for safe re-processing

### 7.2 Bot Restart During Processing
**Problem**: Bot crashes mid-transaction, what happens to reserved slots?

**Solution**:
- All state is in database (not in-memory)
- Expiry worker runs on startup and checks for expired bookings
- Use `FOR UPDATE SKIP LOCKED` for multi-instance safety
- Transactions ensure atomic state changes

### 7.3 User Submits Payment After Expiry
**Problem**: Network delay causes payment to arrive after 3-minute window.

**Solution**:
```go
// In SubmitPayment function
if time.Now().After(expiresAt) {
    // Check if expiry job already processed it
    if status == "EXPIRED" {
        return ErrBookingExpired
    }
    // Race condition: user wins, accept payment
    // Continue with payment submission
}
```

Decision: **Allow submission** if status is still `SLOT_RESERVED`, even if slightly past `expires_at`. This gives user benefit of doubt for network delays. Expiry worker will skip it once status changes to `PAYMENT_SUBMITTED`.

### 7.4 Admin Approves Same Payment Twice
**Problem**: Admin clicks approve button twice (UI lag).

**Solution**:
- Check `status = 'PAYMENT_SUBMITTED'` before approval
- If already `CONFIRMED`, return error
- Idempotent: second approval has no effect

### 7.5 Job Becomes Full While User is Paying
**Problem**: User reserves slot #3, but while uploading receipt, admin approves other users' payments first, filling the job.

**Solution**:
- **Reserved slots are sacred**: Once reserved, user has guaranteed 3 minutes
- Constraint: `reserved_slots + confirmed_slots <= required_workers`
- This prevents overbooking
- If job fills up with confirmed users, remaining reserved slots still valid until expiry

**Alternative (Overbooking Prevention)**:
If you want to avoid having reserved slots when job is full:
```sql
-- Prevent new reservations when job is full
WHERE (reserved_slots + confirmed_slots) < required_workers
```

### 7.6 Multiple Bot Instances (Horizontal Scaling)
**Problem**: Running multiple bot instances, need coordination.

**Solution**:
- Database is single source of truth
- `FOR UPDATE SKIP LOCKED` in expiry worker
- Row-level locks prevent conflicts
- Idempotency keys prevent duplicates

---

## 8. Alternative Approaches (With Trade-offs)

### Option 1: Queue System (More Complex)
```
User clicks → Added to queue
Queue processor picks top N users
Others get waitlist notification
```

**Pros**: 
- Fair ordering (FIFO)
- Can handle large influx

**Cons**:
- More complex infrastructure (Redis, RabbitMQ)
- Delayed feedback to users
- Still need slot management

**Verdict**: Overkill for this use case.

### Option 2: Overbooking Buffer (Airline Model)
```
required_workers = 3
allowed_reservations = 5 (buffer of 2)
```

**Pros**:
- Handles no-shows
- Maximizes confirmed bookings

**Cons**:
- May need refunds
- Against requirements ("avoid refunds by design")

**Verdict**: NOT RECOMMENDED per requirements.

### Option 3: Optimistic Locking (Version Field)
```sql
UPDATE jobs 
SET reserved_slots = reserved_slots + 1, version = version + 1
WHERE id = $1 AND version = $2
```

**Pros**:
- No locks, higher throughput

**Cons**:
- Retry logic needed
- More complex error handling
- Can still have race conditions

**Verdict**: Pessimistic locking (SELECT FOR UPDATE) is simpler and safer.

---

## 9. Monitoring & Metrics

### Key Metrics to Track:
1. **Booking Attempt Rate**: How many users try to book per job
2. **Rejection Rate**: % of users who get "job full" error
3. **Expiry Rate**: % of bookings that expire (indicates payment friction)
4. **Time to Payment**: Average time from reservation to receipt upload
5. **Admin Approval Time**: How long admins take to approve/reject
6. **Slot Utilization**: `confirmed_slots / required_workers` ratio

### Alerting:
- Alert if expiry rate > 50% (UX issue)
- Alert if admin approval time > 30 minutes (staffing issue)
- Alert if job remains ACTIVE for > 24 hours (possible bug)

---

## 10. Implementation Checklist

- [ ] Create migration for `jobs` table with slot counters
- [ ] Create migration for `job_bookings` table with idempotency
- [ ] Implement `ReserveSlot` with SELECT FOR UPDATE
- [ ] Implement expiry background worker
- [ ] Implement `SubmitPayment` with expiry check
- [ ] Implement admin `ApprovePayment` / `RejectPayment`
- [ ] Add Telegram handlers for job link deep linking
- [ ] Add payment receipt upload handler
- [ ] Add admin approval inline keyboard
- [ ] Add monitoring/logging for all state transitions
- [ ] Test concurrent booking with 10+ users
- [ ] Test bot restart during active reservations
- [ ] Test Telegram retry scenarios
- [ ] Load test with 100 concurrent users

---

## 11. Testing Scenarios

### Concurrency Test:
```go
func TestConcurrentBooking(t *testing.T) {
    // Create job with required_workers = 3
    job := createTestJob(3)
    
    // Launch 10 goroutines trying to book simultaneously
    var wg sync.WaitGroup
    results := make(chan error, 10)
    
    for i := 0; i < 10; i++ {
        wg.Add(1)
        go func(userID int64) {
            defer wg.Done()
            _, err := service.ReserveSlot(ctx, userID, job.ID)
            results <- err
        }(int64(i + 1))
    }
    
    wg.Wait()
    close(results)
    
    // Count successes and failures
    successes := 0
    fullErrors := 0
    
    for err := range results {
        if err == nil {
            successes++
        } else if errors.Is(err, ErrJobFull) {
            fullErrors++
        }
    }
    
    // Verify exactly 3 succeeded, 7 got ErrJobFull
    assert.Equal(t, 3, successes)
    assert.Equal(t, 7, fullErrors)
    
    // Verify database state
    var reservedSlots int
    db.QueryRow("SELECT reserved_slots FROM jobs WHERE id = $1", job.ID).Scan(&reservedSlots)
    assert.Equal(t, 3, reservedSlots)
}
```

---

## 12. Summary

### Why This Design is Race-Condition Safe:

1. **Database-Level Atomicity**: All critical operations use transactions with proper isolation levels.

2. **Exclusive Locks**: `SELECT FOR UPDATE` ensures only one transaction modifies slot counters at a time.

3. **CHECK Constraints**: Database enforces business rules at schema level.

4. **Idempotency**: Duplicate requests (Telegram retries) are handled safely with unique keys.

5. **Atomic Operations**: Slot increments/decrements use database-level atomic operations.

6. **No In-Memory State**: Everything is persisted, survives restarts.

7. **Multi-Instance Safe**: `FOR UPDATE SKIP LOCKED` allows horizontal scaling.

### Performance:
- Lock contention only during reservation (< 100ms per user)
- Read-heavy operations (viewing jobs) have no locks
- Background worker is lightweight (runs every 10s)
- Can handle 100+ concurrent bookings per job

### Maintenance:
- No external dependencies (Redis, message queues)
- Standard PostgreSQL features
- Simple state machine
- Easy to debug (all state in one table)

**Conclusion**: This design guarantees that no more than `required_workers` users can proceed past reservation, eliminating refunds by design.
