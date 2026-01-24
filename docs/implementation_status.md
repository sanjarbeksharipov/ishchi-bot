# Job Booking System Implementation Status

## ✅ Completed

### 1. Database Migrations
- [x] `005_create_jobs_table.up.sql` - Jobs table with slot counters
- [x] `006_create_job_bookings_table.up.sql` - Bookings with idempotency

### 2. Models
- [x] `bot/models/job.go` - Updated Job model with new slot management
- [x] `bot/models/booking.go` - New JobBooking model with all states

### 3. Storage Interfaces
- [x] `storage/storage.go` - Updated interfaces for JobRepoI and new BookingRepoI
- [x] `storage/postgres/booking.go` - Complete PostgreSQL implementation for bookings

## 🚧 TODO

### 4. Storage Implementation
- [ ] Update `storage/postgres/job.go` to implement new race-safe methods:
  - `GetByIDForUpdate()`
  - `IncrementReservedSlots()`
  - `DecrementReservedSlots()`
  - `MoveReservedToConfirmed()`
  - `GetAvailableSlots()`

- [ ] Create `storage/postgres/transaction.go` for transaction management

- [ ] Update `storage/postgres/postgres.go` to include:
  - BookingRepo initialization
  - Transaction manager

### 5. Service Layer
- [ ] Create `service/job.go` with:
  - `ReserveSlot()` - Race-safe slot reservation
  - `SubmitPayment()` - Payment receipt upload
  - `ApprovePayment()` - Admin approval
  - `RejectPayment()` - Admin rejection
  - `CancelBooking()` - User cancellation

- [ ] Create `service/expiry_worker.go` for background expiry processing

- [ ] Update `service/service.go` to include job service

### 6. Handlers
- [ ] Create `bot/handlers/job.go` with:
  - Job creation handler (admin)
  - Job listing handler
  - Deep link handler (`/start job_<id>`)
  - Payment upload handler
  - Booking status check handler

- [ ] Create `bot/handlers/admin_job.go` with:
  - Pending approvals list
  - Approve/reject callbacks
  - Job management UI

### 7. Integration
- [ ] Update `cmd/main.go` to:
  - Start expiry worker goroutine
  - Register new handlers
  - Pass job service to handlers

- [ ] Create channel posting functionality
- [ ] Add inline keyboard to channel messages with booking link

### 8. Testing
- [ ] Unit tests for race condition scenarios
- [ ] Integration tests for booking flow
- [ ] Load testing with concurrent users

## 🎯 Next Steps

1. **Run migrations**:
   ```bash
   make migrate-up
   ```

2. **Update job.go repository** to add atomic slot operations

3. **Implement JobService** with SELECT FOR UPDATE logic from design doc

4. **Create expiry worker** as background goroutine

5. **Add handlers** for user booking flow

6. **Test** with concurrent users (simulate race conditions)

## 📝 Design Reference

See `/docs/job_booking_design.md` for detailed:
- State machine diagrams
- SQL queries with row locking
- Race condition analysis
- Testing scenarios

## ⚠️ Critical Implementation Notes

1. **Always use transactions** for slot operations
2. **Use SELECT FOR UPDATE** to prevent race conditions
3. **Use idempotency keys** for Telegram retries
4. **Test expiry worker** survives bot restarts
5. **Monitor** `reserved_slots + confirmed_slots <= required_workers` constraint
