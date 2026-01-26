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

// PaymentService handles payment-related business logic
type PaymentService interface {
	SubmitPayment(ctx context.Context, userID int64, photoFileID string, msgID int64) (*models.JobBooking, error)
	ApprovePayment(ctx context.Context, bookingID, adminID int64) (*models.JobBooking, error)
	RejectPayment(ctx context.Context, bookingID, adminID int64, reason string) (*models.JobBooking, error)
	BlockUserAndRejectPayment(ctx context.Context, bookingID, userID, adminID int64) (*models.JobBooking, error)
}

type paymentService struct {
	cfg     config.Config
	log     logger.LoggerI
	storage storage.StorageI
	manager ServiceManagerI
}

// NewPaymentService creates a new payment service
func NewPaymentService(cfg config.Config, log logger.LoggerI, storage storage.StorageI, manager ServiceManagerI) PaymentService {
	return &paymentService{
		cfg:     cfg,
		log:     log,
		storage: storage,
		manager: manager,
	}
}

// SubmitPayment handles payment receipt submission
func (s *paymentService) SubmitPayment(ctx context.Context, userID int64, photoFileID string, msgID int64) (*models.JobBooking, error) {
	// Find user's most recent SLOT_RESERVED booking
	bookings, err := s.storage.Booking().GetUserBookingsByStatus(ctx, userID, models.BookingStatusSlotReserved)
	if err != nil {
		s.log.Error("Failed to get user bookings", logger.Error(err))
		return nil, fmt.Errorf("failed to get bookings: %w", err)
	}

	if len(bookings) == 0 {
		return nil, fmt.Errorf("no pending booking found")
	}

	booking := bookings[0]

	// Check if booking has expired
	if time.Now().After(booking.ExpiresAt) {
		return nil, fmt.Errorf("booking has expired")
	}

	// Start transaction
	tx, err := s.storage.Transaction().Begin(ctx)
	if err != nil {
		s.log.Error("Failed to start transaction", logger.Error(err))
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			s.storage.Transaction().Rollback(ctx, tx)
		}
	}()

	// Update booking with payment info
	now := time.Now()
	booking.Status = models.BookingStatusPaymentSubmitted
	booking.PaymentReceiptFileID = photoFileID
	booking.PaymentReceiptMsgID = msgID
	booking.PaymentSubmittedAt = &now

	if err := s.storage.Booking().Update(ctx, tx, booking); err != nil {
		s.log.Error("Failed to update booking", logger.Error(err))
		return nil, fmt.Errorf("failed to update booking: %w", err)
	}

	// Commit transaction
	if err := s.storage.Transaction().Commit(ctx, tx); err != nil {
		s.log.Error("Failed to commit transaction", logger.Error(err))
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.log.Info("Payment submitted",
		logger.Any("booking_id", booking.ID),
		logger.Any("user_id", userID),
	)

	return booking, nil
}

// ApprovePayment approves a payment and confirms the booking
func (s *paymentService) ApprovePayment(ctx context.Context, bookingID, adminID int64) (*models.JobBooking, error) {
	// Start transaction
	tx, err := s.storage.Transaction().Begin(ctx)
	if err != nil {
		s.log.Error("Failed to start transaction", logger.Error(err))
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			s.storage.Transaction().Rollback(ctx, tx)
		}
	}()

	// Get booking with lock
	booking, err := s.storage.Booking().GetByIDForUpdate(ctx, tx, bookingID)
	if err != nil {
		s.log.Error("Failed to get booking", logger.Error(err))
		return nil, fmt.Errorf("booking not found: %w", err)
	}

	// Check if already processed
	if booking.Status != models.BookingStatusPaymentSubmitted {
		return nil, fmt.Errorf("payment already processed: %s", booking.Status)
	}

	// Update booking status to CONFIRMED
	now := time.Now()
	booking.Status = models.BookingStatusConfirmed
	booking.ConfirmedAt = &now
	booking.ReviewedByAdminID = &adminID
	booking.ReviewedAt = &now

	if err := s.storage.Booking().Update(ctx, tx, booking); err != nil {
		s.log.Error("Failed to update booking", logger.Error(err))
		return nil, fmt.Errorf("failed to update booking: %w", err)
	}

	// Move slot from reserved to confirmed
	if err := s.storage.Job().MoveReservedToConfirmed(ctx, tx, booking.JobID); err != nil {
		s.log.Error("Failed to move slot", logger.Error(err))
		return nil, fmt.Errorf("failed to move slot: %w", err)
	}

	// Commit transaction
	if err := s.storage.Transaction().Commit(ctx, tx); err != nil {
		s.log.Error("Failed to commit transaction", logger.Error(err))
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.log.Info("Payment approved",
		logger.Any("booking_id", bookingID),
		logger.Any("admin_id", adminID),
	)

	return booking, nil
}

// RejectPayment rejects a payment and releases the slot
func (s *paymentService) RejectPayment(ctx context.Context, bookingID, adminID int64, reason string) (*models.JobBooking, error) {
	// Start transaction
	tx, err := s.storage.Transaction().Begin(ctx)
	if err != nil {
		s.log.Error("Failed to start transaction", logger.Error(err))
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			s.storage.Transaction().Rollback(ctx, tx)
		}
	}()

	// Get booking with lock
	booking, err := s.storage.Booking().GetByIDForUpdate(ctx, tx, bookingID)
	if err != nil {
		s.log.Error("Failed to get booking", logger.Error(err))
		return nil, fmt.Errorf("booking not found: %w", err)
	}

	// Check if already processed
	if booking.Status != models.BookingStatusPaymentSubmitted {
		return nil, fmt.Errorf("payment already processed: %s", booking.Status)
	}

	// Update booking status to REJECTED
	now := time.Now()
	booking.Status = models.BookingStatusRejected
	booking.ReviewedByAdminID = &adminID
	booking.ReviewedAt = &now
	booking.RejectionReason = reason

	if err := s.storage.Booking().Update(ctx, tx, booking); err != nil {
		s.log.Error("Failed to update booking", logger.Error(err))
		return nil, fmt.Errorf("failed to update booking: %w", err)
	}

	// Decrement reserved slots (release the slot)
	if err := s.storage.Job().DecrementReservedSlots(ctx, tx, booking.JobID); err != nil {
		s.log.Error("Failed to decrement slots", logger.Error(err))
		return nil, fmt.Errorf("failed to release slot: %w", err)
	}

	// Commit transaction
	if err := s.storage.Transaction().Commit(ctx, tx); err != nil {
		s.log.Error("Failed to commit transaction", logger.Error(err))
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.log.Info("Payment rejected",
		logger.Any("booking_id", bookingID),
		logger.Any("admin_id", adminID),
		logger.Any("reason", reason),
	)

	return booking, nil
}

// BlockUserAndRejectPayment blocks a user and rejects their payment
func (s *paymentService) BlockUserAndRejectPayment(ctx context.Context, bookingID, userID, adminID int64) (*models.JobBooking, error) {
	// Start transaction
	tx, err := s.storage.Transaction().Begin(ctx)
	if err != nil {
		s.log.Error("Failed to start transaction", logger.Error(err))
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if err != nil {
			s.storage.Transaction().Rollback(ctx, tx)
		}
	}()

	// Get booking
	booking, err := s.storage.Booking().GetByIDForUpdate(ctx, tx, bookingID)
	if err != nil {
		s.log.Error("Failed to get booking", logger.Error(err))
		return nil, fmt.Errorf("booking not found: %w", err)
	}

	// Reject booking if not already processed
	if booking.Status == models.BookingStatusPaymentSubmitted {
		now := time.Now()
		booking.Status = models.BookingStatusRejected
		booking.ReviewedByAdminID = &adminID
		booking.ReviewedAt = &now
		booking.RejectionReason = "Foydalanuvchi bloklandi"

		if err := s.storage.Booking().Update(ctx, tx, booking); err != nil {
			s.log.Error("Failed to update booking", logger.Error(err))
			return nil, fmt.Errorf("failed to update booking: %w", err)
		}

		// Release slot
		if err := s.storage.Job().DecrementReservedSlots(ctx, tx, booking.JobID); err != nil {
			s.log.Error("Failed to decrement slots", logger.Error(err))
			return nil, fmt.Errorf("failed to release slot: %w", err)
		}
	}

	// TODO: Add to blocked_users table when implemented

	// Commit transaction
	if err := s.storage.Transaction().Commit(ctx, tx); err != nil {
		s.log.Error("Failed to commit transaction", logger.Error(err))
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	s.log.Info("User blocked and payment rejected",
		logger.Any("user_id", userID),
		logger.Any("booking_id", bookingID),
		logger.Any("admin_id", adminID),
	)

	return booking, nil
}
