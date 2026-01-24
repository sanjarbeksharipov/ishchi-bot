package models

import (
	"fmt"
	"time"
)

// BookingStatus represents the status of a job booking
type BookingStatus string

const (
	BookingStatusSlotReserved     BookingStatus = "SLOT_RESERVED"     // Slot temporarily held (3-min timer)
	BookingStatusPaymentSubmitted BookingStatus = "PAYMENT_SUBMITTED" // Receipt uploaded, waiting admin
	BookingStatusConfirmed        BookingStatus = "CONFIRMED"         // Admin approved, slot locked
	BookingStatusRejected         BookingStatus = "REJECTED"          // Admin rejected payment
	BookingStatusExpired          BookingStatus = "EXPIRED"           // 3-minute timer ran out
	BookingStatusCancelledByUser  BookingStatus = "CANCELLED_BY_USER" // User cancelled before payment
)

// JobBooking represents a user's booking for a job
type JobBooking struct {
	ID     int64 `json:"id"`
	JobID  int64 `json:"job_id"`
	UserID int64 `json:"user_id"`

	// State tracking
	Status BookingStatus `json:"status"`

	// Payment tracking
	PaymentReceiptFileID    string `json:"payment_receipt_file_id"`        // User's payment receipt file ID
	PaymentReceiptMsgID     int64  `json:"payment_receipt_message_id"`     // User's payment receipt message ID
	PaymentInstructionMsgID int64  `json:"payment_instruction_message_id"` // Bot's payment instruction message ID

	// Timing (CRITICAL for expiry)
	ReservedAt         time.Time  `json:"reserved_at"`
	ExpiresAt          time.Time  `json:"expires_at"`
	PaymentSubmittedAt *time.Time `json:"payment_submitted_at,omitempty"`
	ConfirmedAt        *time.Time `json:"confirmed_at,omitempty"`

	// Admin review
	ReviewedByAdminID *int64     `json:"reviewed_by_admin_id,omitempty"`
	ReviewedAt        *time.Time `json:"reviewed_at,omitempty"`
	RejectionReason   string     `json:"rejection_reason,omitempty"`

	// Idempotency (CRITICAL for Telegram retries)
	IdempotencyKey string `json:"idempotency_key"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// BookingStatusDisplay returns the display text for booking status
func (s BookingStatus) Display() string {
	switch s {
	case BookingStatusSlotReserved:
		return "⏳ Band qilindi"
	case BookingStatusPaymentSubmitted:
		return "💳 To'lov yuborildi"
	case BookingStatusConfirmed:
		return "✅ Tasdiqlandi"
	case BookingStatusRejected:
		return "❌ Rad etildi"
	case BookingStatusExpired:
		return "⏰ Vaqt tugadi"
	case BookingStatusCancelledByUser:
		return "🚫 Bekor qilindi"
	default:
		return string(s)
	}
}

// IsValid checks if the status is valid
func (s BookingStatus) IsValid() bool {
	switch s {
	case BookingStatusSlotReserved, BookingStatusPaymentSubmitted,
		BookingStatusConfirmed, BookingStatusRejected,
		BookingStatusExpired, BookingStatusCancelledByUser:
		return true
	default:
		return false
	}
}

// IsExpired checks if the booking has expired based on current time
func (b *JobBooking) IsExpired() bool {
	return b.Status == BookingStatusSlotReserved && time.Now().After(b.ExpiresAt)
}

// CanSubmitPayment checks if payment can be submitted for this booking
func (b *JobBooking) CanSubmitPayment() bool {
	return b.Status == BookingStatusSlotReserved && !b.IsExpired()
}

// CanBeApproved checks if booking is waiting for admin approval
func (b *JobBooking) CanBeApproved() bool {
	return b.Status == BookingStatusPaymentSubmitted
}

// TimeRemaining returns duration until expiry (0 if expired)
func (b *JobBooking) TimeRemaining() time.Duration {
	if b.Status != BookingStatusSlotReserved {
		return 0
	}
	remaining := time.Until(b.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

// GenerateIdempotencyKey creates an idempotency key for a user-job pair
func GenerateIdempotencyKey(userID, jobID int64) string {
	return fmt.Sprintf("user_%d_job_%d", userID, jobID)
}
