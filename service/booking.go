package service

import (
	"context"
	"fmt"
	"time"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/config"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/storage"
)

// BookingService handles booking-related business logic
type BookingService interface {
	ConfirmBooking(ctx context.Context, userID, jobID int64) (*models.JobBooking, error)
	GetBookingWithStatus(ctx context.Context, userID int64, status models.BookingStatus) (*models.JobBooking, error)
	CheckIdempotency(ctx context.Context, userID, jobID int64) (*models.JobBooking, error)
	ExpireBooking(ctx context.Context, booking *models.JobBooking) error
	// CancelBookingByAdmin cancels a confirmed booking, releases the slot, and reopens the job if it was FULL.
	CancelBookingByAdmin(ctx context.Context, bookingID, adminID int64) (*models.JobBooking, *models.Job, error)
}

type bookingService struct {
	cfg     config.Config
	log     logger.LoggerI
	storage storage.StorageI
	manager ServiceManagerI
}

// NewBookingService creates a new booking service
func NewBookingService(cfg config.Config, log logger.LoggerI, storage storage.StorageI, manager ServiceManagerI) BookingService {
	return &bookingService{
		cfg:     cfg,
		log:     log,
		storage: storage,
		manager: manager,
	}
}

// ConfirmBooking atomically reserves a slot and creates booking with idempotency
func (s *bookingService) ConfirmBooking(ctx context.Context, userID, jobID int64) (*models.JobBooking, error) {
	// Check if user is blocked
	block, err := s.storage.User().GetBlockStatus(ctx, userID)
	if err != nil {
		s.log.Error("Failed to check block status", logger.Error(err))
		return nil, fmt.Errorf("failed to check block status: %w", err)
	}

	if block != nil {
		s.log.Info("Block check for user",
			logger.Any("user_id", userID),
			logger.Any("blocked_until", block.BlockedUntil),
			logger.Any("total_violations", block.TotalViolations),
			logger.Any("current_time", time.Now()),
		)

		if block.BlockedUntil == nil {
			// Permanent block (BlockedUntil is NULL)
			s.log.Warn("User is permanently blocked", logger.Any("user_id", userID))
			return nil, fmt.Errorf("❌ Siz doimiy bloklangansiz.\n\nSabab: %s\n\nQo'shimcha ma'lumot uchun admin bilan bog'laning.", block.Reason)
		}

		now := time.Now()
		if now.Before(*block.BlockedUntil) {
			// Temporary block still active
			remaining := time.Until(*block.BlockedUntil)
			hours := int(remaining.Hours())
			minutes := int(remaining.Minutes()) % 60
			s.log.Warn("User is temporarily blocked",
				logger.Any("user_id", userID),
				logger.Any("blocked_until", block.BlockedUntil),
				logger.Any("remaining_hours", hours),
				logger.Any("remaining_minutes", minutes),
			)
			return nil, fmt.Errorf("⚠️ Siz vaqtincha bloklangansiz.\n\nSabab: %s\n\nQolgan vaqt: %d soat %d daqiqa", block.Reason, hours, minutes)
		}

		// Block expired, auto-unblock
		s.log.Info("Block expired, auto-unblocking user", logger.Any("user_id", userID))
		if err := s.storage.User().UnblockUser(ctx, userID); err != nil {
			s.log.Error("Failed to auto-unblock user", logger.Error(err))
			// Don't return error, continue with booking
		} else {
			s.log.Info("User auto-unblocked after 24h ban", logger.Any("user_id", userID))
		}
	}

	// Check idempotency
	idempotencyKey := models.GenerateIdempotencyKey(userID, jobID)
	existingBooking, _ := s.storage.Booking().GetByIdempotencyKey(ctx, nil, idempotencyKey)
	if existingBooking != nil {
		if existingBooking.Status == models.BookingStatusSlotReserved && !existingBooking.IsExpired() {
			return existingBooking, fmt.Errorf("booking already exists with %d seconds remaining", int(existingBooking.TimeRemaining().Seconds()))
		}
		if existingBooking.Status == models.BookingStatusPaymentSubmitted {
			return existingBooking, fmt.Errorf("payment is being reviewed")
		}
		if existingBooking.Status == models.BookingStatusConfirmed {
			return existingBooking, fmt.Errorf("booking already confirmed")
		}
	}

	// Check if user has ANY other active booking (Reserved or PaymentSubmitted)
	// User can only have one pending booking at a time
	reservedBookings, err := s.storage.Booking().GetUserBookingsByStatus(ctx, userID, models.BookingStatusSlotReserved)
	if err == nil {
		for _, b := range reservedBookings {
			if !b.IsExpired() && b.JobID != jobID {
				return nil, fmt.Errorf("you have another active booking (Job #%d)", b.JobID)
			}
		}
	}

	submittedBookings, err := s.storage.Booking().GetUserBookingsByStatus(ctx, userID, models.BookingStatusPaymentSubmitted)
	if err == nil && len(submittedBookings) > 0 {
		return nil, fmt.Errorf("you have a payment under review for another job (Job #%d)", submittedBookings[0].JobID)
	}

	// Start transaction
	tx, err := s.storage.Transaction().Begin(ctx)
	if err != nil {
		s.log.Error("Failed to begin transaction", logger.Error(err))
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Always rollback on exit — Rollback after Commit is a harmless no-op in pgx.
	defer s.storage.Transaction().Rollback(ctx, tx)

	// Lock job row and get current state
	job, err := s.storage.Job().GetByIDForUpdate(ctx, tx, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to lock job: %w", err)
	}

	// Validate job status
	if job.Status != models.JobStatusActive {
		return nil, fmt.Errorf("job is not active")
	}

	// Check if slots are available
	if job.IsFull() {
		if job.ReservedSlots > 0 {
			return nil, fmt.Errorf("all slots reserved, try again in a few minutes")
		}
		return nil, fmt.Errorf("all slots are full")
	}

	// Atomically increment reserved_slots
	if err := s.storage.Job().IncrementReservedSlots(ctx, tx, jobID); err != nil {
		return nil, fmt.Errorf("failed to reserve slot: %w", err)
	}

	// Create booking
	now := time.Now()
	expiresAt := now.Add(3 * time.Minute)

	booking := &models.JobBooking{
		UserID:         userID,
		JobID:          jobID,
		Status:         models.BookingStatusSlotReserved,
		IdempotencyKey: idempotencyKey,
		CreatedAt:      now,
		ReservedAt:     now,
		ExpiresAt:      expiresAt,
	}

	if err := s.storage.Booking().Create(ctx, tx, booking); err != nil {
		return nil, fmt.Errorf("failed to create booking: %w", err)
	}

	// Commit transaction
	if err := s.storage.Transaction().Commit(ctx, tx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.log.Info("Booking confirmed",
		logger.Any("booking_id", booking.ID),
		logger.Any("user_id", userID),
		logger.Any("job_id", jobID),
	)

	return booking, nil
}

// GetBookingWithStatus finds user's most recent booking with specified status
func (s *bookingService) GetBookingWithStatus(ctx context.Context, userID int64, status models.BookingStatus) (*models.JobBooking, error) {
	bookings, err := s.storage.Booking().GetUserBookingsByStatus(ctx, userID, status)
	if err != nil {
		s.log.Error("Failed to get user bookings", logger.Error(err))
		return nil, fmt.Errorf("failed to get bookings: %w", err)
	}

	if len(bookings) == 0 {
		return nil, fmt.Errorf("no booking found with status %s", status)
	}

	// Return most recent
	return bookings[0], nil
}

// CheckIdempotency checks if user already has a booking for this job
func (s *bookingService) CheckIdempotency(ctx context.Context, userID, jobID int64) (*models.JobBooking, error) {
	idempotencyKey := models.GenerateIdempotencyKey(userID, jobID)
	return s.storage.Booking().GetByIdempotencyKey(ctx, nil, idempotencyKey)
}

// ExpireBooking expires a booking and releases its slot
func (s *bookingService) ExpireBooking(ctx context.Context, booking *models.JobBooking) error {
	tx, err := s.storage.Transaction().Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Always rollback on exit — Rollback after Commit is a harmless no-op in pgx.
	defer s.storage.Transaction().Rollback(ctx, tx)

	booking.Status = models.BookingStatusExpired
	if err := s.storage.Booking().Update(ctx, tx, booking); err != nil {
		return fmt.Errorf("failed to update booking: %w", err)
	}

	if err := s.storage.Transaction().Commit(ctx, tx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// CancelBookingByAdmin cancels a confirmed booking, releases the slot, and reopens the job if it was FULL.
// Returns the updated booking and job so callers can update channel/admin messages.
func (s *bookingService) CancelBookingByAdmin(ctx context.Context, bookingID, adminID int64) (*models.JobBooking, *models.Job, error) {
	tx, err := s.storage.Transaction().Begin(ctx)
	if err != nil {
		s.log.Error("Failed to begin transaction", logger.Error(err))
		return nil, nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Always rollback on exit — Rollback after Commit is a harmless no-op in pgx.
	defer s.storage.Transaction().Rollback(ctx, tx)

	// Get booking with row lock
	booking, err := s.storage.Booking().GetByIDForUpdate(ctx, tx, bookingID)
	if err != nil {
		s.log.Error("Failed to get booking", logger.Error(err))
		return nil, nil, fmt.Errorf("booking not found: %w", err)
	}

	// Only confirmed bookings (payment received) can be cancelled by admin with refund
	if booking.Status != models.BookingStatusConfirmed {
		return nil, nil, fmt.Errorf("only confirmed bookings can be cancelled by admin (current status: %s)", booking.Status)
	}

	// Mark booking as cancelled by admin
	reason := "Admin tomonidan bekor qilindi. To'lov qaytariladi."
	if err := s.storage.Booking().MarkAsCancelledByAdmin(ctx, tx, bookingID, adminID, reason); err != nil {
		s.log.Error("Failed to mark booking as cancelled by admin", logger.Error(err))
		return nil, nil, fmt.Errorf("failed to cancel booking: %w", err)
	}

	// Decrement confirmed_slots to free the slot
	if err := s.storage.Job().DecrementConfirmedSlots(ctx, tx, booking.JobID); err != nil {
		s.log.Error("Failed to decrement confirmed slots", logger.Error(err))
		return nil, nil, fmt.Errorf("failed to release slot: %w", err)
	}

	// Get the updated job within the transaction
	job, err := s.storage.Job().GetByIDForUpdate(ctx, tx, booking.JobID)
	if err != nil {
		s.log.Error("Failed to get job", logger.Error(err))
		return nil, nil, fmt.Errorf("failed to get job: %w", err)
	}

	// If job was FULL and now has an available slot, reopen it to ACTIVE
	if job.Status == models.JobStatusFull && !job.IsCompletelyFull() {
		if err := s.storage.Job().UpdateStatusInTx(ctx, tx, job.ID, models.JobStatusActive); err != nil {
			s.log.Error("Failed to reopen job to ACTIVE", logger.Error(err))
			// Don't fail the whole operation, just log it
		} else {
			job.Status = models.JobStatusActive
			s.log.Info("Job reopened to ACTIVE after booking cancellation", logger.Any("job_id", job.ID))
		}
	}

	// Commit transaction
	if err := s.storage.Transaction().Commit(ctx, tx); err != nil {
		s.log.Error("Failed to commit transaction", logger.Error(err))
		return nil, nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update booking status in memory
	booking.Status = models.BookingStatusCancelledByAdmin
	booking.RejectionReason = reason

	s.log.Info("Booking cancelled by admin",
		logger.Any("booking_id", bookingID),
		logger.Any("admin_id", adminID),
		logger.Any("user_id", booking.UserID),
		logger.Any("job_id", booking.JobID),
	)

	return booking, job, nil
}
