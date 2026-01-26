package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// bookingRepo implements storage.BookingRepoI interface using PostgreSQL
type bookingRepo struct {
	db  *pgxpool.Pool
	log logger.LoggerI
}

// NewBookingRepo creates a new PostgreSQL booking repository
func NewBookingRepo(db *pgxpool.Pool, log logger.LoggerI) storage.BookingRepoI {
	return &bookingRepo{
		db:  db,
		log: log,
	}
}

// Create creates a new booking (must be called within transaction)
func (r *bookingRepo) Create(ctx context.Context, tx any, booking *models.JobBooking) error {
	query := `
		INSERT INTO job_bookings (
			job_id, user_id, status, reserved_at, expires_at, idempotency_key
		) VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (idempotency_key) 
		DO UPDATE SET 
			status = EXCLUDED.status,
			reserved_at = EXCLUDED.reserved_at,
			expires_at = EXCLUDED.expires_at,
			updated_at = NOW()
		RETURNING id, created_at, updated_at
	`

	var err error
	if tx != nil {
		pgxTx := tx.(pgx.Tx)
		err = pgxTx.QueryRow(ctx, query,
			booking.JobID,
			booking.UserID,
			booking.Status,
			booking.ReservedAt,
			booking.ExpiresAt,
			booking.IdempotencyKey,
		).Scan(&booking.ID, &booking.CreatedAt, &booking.UpdatedAt)
	} else {
		err = r.db.QueryRow(ctx, query,
			booking.JobID,
			booking.UserID,
			booking.Status,
			booking.ReservedAt,
			booking.ExpiresAt,
			booking.IdempotencyKey,
		).Scan(&booking.ID, &booking.CreatedAt, &booking.UpdatedAt)
	}

	if err != nil {
		r.log.Error("Failed to create booking", logger.Error(err))
		return fmt.Errorf("failed to create booking: %w", err)
	}

	return nil
}

// GetByID retrieves a booking by ID
func (r *bookingRepo) GetByID(ctx context.Context, id int64) (*models.JobBooking, error) {
	query := `
		SELECT id, job_id, user_id, status, payment_receipt_file_id, payment_receipt_message_id,
			   payment_instruction_message_id, reserved_at, expires_at, payment_submitted_at, confirmed_at,
			   reviewed_by_admin_id, reviewed_at, rejection_reason, idempotency_key,
			   created_at, updated_at
		FROM job_bookings
		WHERE id = $1
	`

	booking := &models.JobBooking{}
	var paymentReceiptFileID, rejectionReason sql.NullString
	var paymentReceiptMsgID, paymentInstructionMsgID, reviewedByAdminID sql.NullInt64
	var paymentSubmittedAt, confirmedAt, reviewedAt sql.NullTime

	err := r.db.QueryRow(ctx, query, id).Scan(
		&booking.ID,
		&booking.JobID,
		&booking.UserID,
		&booking.Status,
		&paymentReceiptFileID,
		&paymentReceiptMsgID,
		&paymentInstructionMsgID,
		&booking.ReservedAt,
		&booking.ExpiresAt,
		&paymentSubmittedAt,
		&confirmedAt,
		&reviewedByAdminID,
		&reviewedAt,
		&rejectionReason,
		&booking.IdempotencyKey,
		&booking.CreatedAt,
		&booking.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		r.log.Error("Failed to get booking", logger.Error(err))
		return nil, fmt.Errorf("failed to get booking: %w", err)
	}

	// Handle nullable fields
	if paymentReceiptFileID.Valid {
		booking.PaymentReceiptFileID = paymentReceiptFileID.String
	}
	if paymentReceiptMsgID.Valid {
		booking.PaymentReceiptMsgID = paymentReceiptMsgID.Int64
	}
	if paymentInstructionMsgID.Valid {
		booking.PaymentInstructionMsgID = paymentInstructionMsgID.Int64
	}
	if paymentSubmittedAt.Valid {
		booking.PaymentSubmittedAt = &paymentSubmittedAt.Time
	}
	if confirmedAt.Valid {
		booking.ConfirmedAt = &confirmedAt.Time
	}
	if reviewedByAdminID.Valid {
		booking.ReviewedByAdminID = &reviewedByAdminID.Int64
	}
	if reviewedAt.Valid {
		booking.ReviewedAt = &reviewedAt.Time
	}
	if rejectionReason.Valid {
		booking.RejectionReason = rejectionReason.String
	}

	return booking, nil
}

// GetByIDForUpdate retrieves a booking with row lock (FOR UPDATE)
func (r *bookingRepo) GetByIDForUpdate(ctx context.Context, tx any, id int64) (*models.JobBooking, error) {
	query := `
		SELECT id, job_id, user_id, status, payment_receipt_file_id, payment_receipt_message_id,
			   payment_instruction_message_id, reserved_at, expires_at, payment_submitted_at, confirmed_at,
			   reviewed_by_admin_id, reviewed_at, rejection_reason, idempotency_key,
			   created_at, updated_at
		FROM job_bookings
		WHERE id = $1
		FOR UPDATE
	`

	booking := &models.JobBooking{}
	var paymentReceiptFileID, rejectionReason sql.NullString
	var paymentReceiptMsgID, paymentInstructionMsgID, reviewedByAdminID sql.NullInt64
	var paymentSubmittedAt, confirmedAt, reviewedAt sql.NullTime

	var err error
	if tx != nil {
		pgxTx := tx.(pgx.Tx)
		err = pgxTx.QueryRow(ctx, query, id).Scan(
			&booking.ID, &booking.JobID, &booking.UserID, &booking.Status,
			&paymentReceiptFileID, &paymentReceiptMsgID, &paymentInstructionMsgID,
			&booking.ReservedAt, &booking.ExpiresAt, &paymentSubmittedAt, &confirmedAt,
			&reviewedByAdminID, &reviewedAt, &rejectionReason, &booking.IdempotencyKey,
			&booking.CreatedAt, &booking.UpdatedAt,
		)
	} else {
		err = r.db.QueryRow(ctx, query, id).Scan(
			&booking.ID, &booking.JobID, &booking.UserID, &booking.Status,
			&paymentReceiptFileID, &paymentReceiptMsgID, &paymentInstructionMsgID,
			&booking.ReservedAt, &booking.ExpiresAt, &paymentSubmittedAt, &confirmedAt,
			&reviewedByAdminID, &reviewedAt, &rejectionReason, &booking.IdempotencyKey,
			&booking.CreatedAt, &booking.UpdatedAt,
		)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get booking for update: %w", err)
	}

	// Handle nullable fields (same as GetByID)
	if paymentReceiptFileID.Valid {
		booking.PaymentReceiptFileID = paymentReceiptFileID.String
	}
	if paymentReceiptMsgID.Valid {
		booking.PaymentReceiptMsgID = paymentReceiptMsgID.Int64
	}
	if paymentInstructionMsgID.Valid {
		booking.PaymentInstructionMsgID = paymentInstructionMsgID.Int64
	}
	if paymentSubmittedAt.Valid {
		booking.PaymentSubmittedAt = &paymentSubmittedAt.Time
	}
	if confirmedAt.Valid {
		booking.ConfirmedAt = &confirmedAt.Time
	}
	if reviewedByAdminID.Valid {
		booking.ReviewedByAdminID = &reviewedByAdminID.Int64
	}
	if reviewedAt.Valid {
		booking.ReviewedAt = &reviewedAt.Time
	}
	if rejectionReason.Valid {
		booking.RejectionReason = rejectionReason.String
	}

	return booking, nil
}

// GetByUserAndJob retrieves a booking by user and job
func (r *bookingRepo) GetByUserAndJob(ctx context.Context, userID, jobID int64) (*models.JobBooking, error) {
	query := `
		SELECT id, job_id, user_id, status, payment_receipt_file_id, payment_receipt_message_id,
			   payment_instruction_message_id, reserved_at, expires_at, payment_submitted_at, confirmed_at,
			   reviewed_by_admin_id, reviewed_at, rejection_reason, idempotency_key,
			   created_at, updated_at
		FROM job_bookings
		WHERE user_id = $1 AND job_id = $2
		ORDER BY created_at DESC
		LIMIT 1
	`

	booking := &models.JobBooking{}
	var paymentReceiptFileID, rejectionReason sql.NullString
	var paymentReceiptMsgID, paymentInstructionMsgID, reviewedByAdminID sql.NullInt64
	var paymentSubmittedAt, confirmedAt, reviewedAt sql.NullTime

	err := r.db.QueryRow(ctx, query, userID, jobID).Scan(
		&booking.ID, &booking.JobID, &booking.UserID, &booking.Status,
		&paymentReceiptFileID, &paymentReceiptMsgID, &paymentInstructionMsgID,
		&booking.ReservedAt, &booking.ExpiresAt, &paymentSubmittedAt, &confirmedAt,
		&reviewedByAdminID, &reviewedAt, &rejectionReason, &booking.IdempotencyKey,
		&booking.CreatedAt, &booking.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get booking by user and job: %w", err)
	}

	// Handle nullable fields
	if paymentReceiptFileID.Valid {
		booking.PaymentReceiptFileID = paymentReceiptFileID.String
	}
	if paymentReceiptMsgID.Valid {
		booking.PaymentReceiptMsgID = paymentReceiptMsgID.Int64
	}
	if paymentInstructionMsgID.Valid {
		booking.PaymentInstructionMsgID = paymentInstructionMsgID.Int64
	}
	if paymentSubmittedAt.Valid {
		booking.PaymentSubmittedAt = &paymentSubmittedAt.Time
	}
	if confirmedAt.Valid {
		booking.ConfirmedAt = &confirmedAt.Time
	}
	if reviewedByAdminID.Valid {
		booking.ReviewedByAdminID = &reviewedByAdminID.Int64
	}
	if reviewedAt.Valid {
		booking.ReviewedAt = &reviewedAt.Time
	}
	if rejectionReason.Valid {
		booking.RejectionReason = rejectionReason.String
	}

	return booking, nil
}

// GetByIdempotencyKey retrieves a booking by idempotency key (within transaction)
func (r *bookingRepo) GetByIdempotencyKey(ctx context.Context, tx any, key string) (*models.JobBooking, error) {
	query := `
		SELECT id, job_id, user_id, status, reserved_at, expires_at, created_at, updated_at
		FROM job_bookings
		WHERE idempotency_key = $1
	`

	booking := &models.JobBooking{}
	var err error

	if tx != nil {
		pgxTx := tx.(pgx.Tx)
		err = pgxTx.QueryRow(ctx, query, key).Scan(
			&booking.ID, &booking.JobID, &booking.UserID, &booking.Status,
			&booking.ReservedAt, &booking.ExpiresAt, &booking.CreatedAt, &booking.UpdatedAt,
		)
	} else {
		err = r.db.QueryRow(ctx, query, key).Scan(
			&booking.ID, &booking.JobID, &booking.UserID, &booking.Status,
			&booking.ReservedAt, &booking.ExpiresAt, &booking.CreatedAt, &booking.UpdatedAt,
		)
	}

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		return nil, fmt.Errorf("failed to get booking by idempotency key: %w", err)
	}

	booking.IdempotencyKey = key
	return booking, nil
}

// Update updates a booking
func (r *bookingRepo) Update(ctx context.Context, tx any, booking *models.JobBooking) error {
	query := `
		UPDATE job_bookings
		SET status = $2, payment_receipt_file_id = $3, payment_receipt_message_id = $4,
			payment_instruction_message_id = $5, payment_submitted_at = $6, confirmed_at = $7,
			reviewed_by_admin_id = $8, reviewed_at = $9, rejection_reason = $10,
			updated_at = NOW()
		WHERE id = $1
	`

	var err error
	if tx != nil {
		pgxTx := tx.(pgx.Tx)
		_, err = pgxTx.Exec(ctx, query,
			booking.ID,
			booking.Status,
			toNullString(booking.PaymentReceiptFileID),
			toNullInt64(booking.PaymentReceiptMsgID),
			toNullInt64(booking.PaymentInstructionMsgID),
			toNullTime(booking.PaymentSubmittedAt),
			toNullTime(booking.ConfirmedAt),
			toNullInt64Ptr(booking.ReviewedByAdminID),
			toNullTime(booking.ReviewedAt),
			toNullString(booking.RejectionReason),
		)
	} else {
		_, err = r.db.Exec(ctx, query,
			booking.ID,
			booking.Status,
			toNullString(booking.PaymentReceiptFileID),
			toNullInt64(booking.PaymentReceiptMsgID),
			toNullInt64(booking.PaymentInstructionMsgID),
			toNullTime(booking.PaymentSubmittedAt),
			toNullTime(booking.ConfirmedAt),
			toNullInt64Ptr(booking.ReviewedByAdminID),
			toNullTime(booking.ReviewedAt),
			toNullString(booking.RejectionReason),
		)
	}

	if err != nil {
		r.log.Error("Failed to update booking", logger.Error(err))
		return fmt.Errorf("failed to update booking: %w", err)
	}

	return nil
}

// Delete deletes a booking
func (r *bookingRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM job_bookings WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}

// GetExpiredBookings retrieves bookings that have expired (FOR UPDATE SKIP LOCKED)
func (r *bookingRepo) GetExpiredBookings(ctx context.Context, limit int) ([]*models.JobBooking, error) {
	query := `
		SELECT id, job_id, user_id, payment_instruction_message_id
		FROM job_bookings
		WHERE status = 'SLOT_RESERVED'
		  AND expires_at < $1
		LIMIT $2
		FOR UPDATE SKIP LOCKED
	`

	rows, err := r.db.Query(ctx, query, time.Now(), limit)
	if err != nil {
		r.log.Error("Failed to get expired bookings", logger.Error(err))
		return nil, fmt.Errorf("failed to get expired bookings: %w", err)
	}
	defer rows.Close()

	var bookings []*models.JobBooking
	for rows.Next() {
		booking := &models.JobBooking{}
		var msgID sql.NullInt64

		if err := rows.Scan(&booking.ID, &booking.JobID, &booking.UserID, &msgID); err != nil {
			r.log.Error("Failed to scan expired booking", logger.Error(err))
			continue
		}

		if msgID.Valid {
			booking.PaymentInstructionMsgID = msgID.Int64
		}

		bookings = append(bookings, booking)
	}

	return bookings, nil
}

// GetPendingApprovals retrieves bookings waiting for admin approval
func (r *bookingRepo) GetPendingApprovals(ctx context.Context) ([]*models.JobBooking, error) {
	query := `
		SELECT id, job_id, user_id, payment_receipt_file_id, payment_receipt_message_id,
			payment_instruction_message_id, payment_submitted_at, idempotency_key, created_at
		FROM job_bookings
		WHERE status = 'PAYMENT_SUBMITTED'
		ORDER BY payment_submitted_at ASC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get pending approvals: %w", err)
	}
	defer rows.Close()

	var bookings []*models.JobBooking
	for rows.Next() {
		booking := &models.JobBooking{Status: models.BookingStatusPaymentSubmitted}
		var paymentSubmittedAt sql.NullTime
		var paymentInstructionMsgID sql.NullInt64

		if err := rows.Scan(
			&booking.ID, &booking.JobID, &booking.UserID,
			&booking.PaymentReceiptFileID, &booking.PaymentReceiptMsgID,
			&paymentInstructionMsgID, &paymentSubmittedAt,
			&booking.IdempotencyKey, &booking.CreatedAt,
		); err != nil {
			continue
		}

		if paymentInstructionMsgID.Valid {
			booking.PaymentInstructionMsgID = paymentInstructionMsgID.Int64
		}
		if paymentSubmittedAt.Valid {
			booking.PaymentSubmittedAt = &paymentSubmittedAt.Time
		}

		bookings = append(bookings, booking)
	}

	return bookings, nil
}

// GetUserBookings retrieves all bookings for a user
func (r *bookingRepo) GetUserBookings(ctx context.Context, userID int64) ([]*models.JobBooking, error) {
	query := `
		SELECT id, job_id, status, reserved_at, expires_at, created_at
		FROM job_bookings
		WHERE user_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user bookings: %w", err)
	}
	defer rows.Close()

	var bookings []*models.JobBooking
	for rows.Next() {
		booking := &models.JobBooking{UserID: userID}
		if err := rows.Scan(&booking.ID, &booking.JobID, &booking.Status,
			&booking.ReservedAt, &booking.ExpiresAt, &booking.CreatedAt); err != nil {
			continue
		}
		bookings = append(bookings, booking)
	}

	return bookings, nil
}

// GetUserBookingsByStatus retrieves user bookings filtered by status
func (r *bookingRepo) GetUserBookingsByStatus(ctx context.Context, userID int64, status models.BookingStatus) ([]*models.JobBooking, error) {
	query := `
		SELECT id, job_id, user_id, status, payment_receipt_file_id, payment_receipt_message_id,
			   payment_instruction_message_id, reserved_at, expires_at, payment_submitted_at, confirmed_at,
			   reviewed_by_admin_id, reviewed_at, rejection_reason, idempotency_key,
			   created_at, updated_at
		FROM job_bookings
		WHERE user_id = $1 AND status = $2
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, userID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get user bookings by status: %w", err)
	}
	defer rows.Close()

	var bookings []*models.JobBooking
	for rows.Next() {
		booking := &models.JobBooking{}
		var paymentReceiptFileID, rejectionReason sql.NullString
		var paymentReceiptMsgID, paymentInstructionMsgID, reviewedByAdminID sql.NullInt64
		var paymentSubmittedAt, confirmedAt, reviewedAt sql.NullTime

		if err := rows.Scan(
			&booking.ID, &booking.JobID, &booking.UserID, &booking.Status,
			&paymentReceiptFileID, &paymentReceiptMsgID, &paymentInstructionMsgID,
			&booking.ReservedAt, &booking.ExpiresAt, &paymentSubmittedAt, &confirmedAt,
			&reviewedByAdminID, &reviewedAt, &rejectionReason, &booking.IdempotencyKey,
			&booking.CreatedAt, &booking.UpdatedAt,
		); err != nil {
			r.log.Error("Failed to scan booking", logger.Error(err))
			continue
		}

		// Handle nullable fields
		if paymentReceiptFileID.Valid {
			booking.PaymentReceiptFileID = paymentReceiptFileID.String
		}
		if paymentReceiptMsgID.Valid {
			booking.PaymentReceiptMsgID = paymentReceiptMsgID.Int64
		}
		if paymentInstructionMsgID.Valid {
			booking.PaymentInstructionMsgID = paymentInstructionMsgID.Int64
		}
		if paymentSubmittedAt.Valid {
			booking.PaymentSubmittedAt = &paymentSubmittedAt.Time
		}
		if confirmedAt.Valid {
			booking.ConfirmedAt = &confirmedAt.Time
		}
		if reviewedByAdminID.Valid {
			booking.ReviewedByAdminID = &reviewedByAdminID.Int64
		}
		if reviewedAt.Valid {
			booking.ReviewedAt = &reviewedAt.Time
		}
		if rejectionReason.Valid {
			booking.RejectionReason = rejectionReason.String
		}

		bookings = append(bookings, booking)
	}

	return bookings, nil
}

// GetJobBookings retrieves all bookings for a job
func (r *bookingRepo) GetJobBookings(ctx context.Context, jobID int64) ([]*models.JobBooking, error) {
	query := `
		SELECT id, user_id, status, reserved_at, expires_at, created_at
		FROM job_bookings
		WHERE job_id = $1
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query, jobID)
	if err != nil {
		return nil, fmt.Errorf("failed to get job bookings: %w", err)
	}
	defer rows.Close()

	var bookings []*models.JobBooking
	for rows.Next() {
		booking := &models.JobBooking{JobID: jobID}
		if err := rows.Scan(&booking.ID, &booking.UserID, &booking.Status,
			&booking.ReservedAt, &booking.ExpiresAt, &booking.CreatedAt); err != nil {
			continue
		}
		bookings = append(bookings, booking)
	}

	return bookings, nil
}

// UpdateStatus updates booking status
func (r *bookingRepo) UpdateStatus(ctx context.Context, tx any, bookingID int64, status models.BookingStatus) error {
	query := `
		UPDATE job_bookings
		SET status = $2, updated_at = NOW()
		WHERE id = $1
	`

	var err error
	if tx != nil {
		pgxTx := tx.(pgx.Tx)
		_, err = pgxTx.Exec(ctx, query, bookingID, status)
	} else {
		_, err = r.db.Exec(ctx, query, bookingID, status)
	}

	return err
}

// MarkAsExpired marks a booking as expired
func (r *bookingRepo) MarkAsExpired(ctx context.Context, tx any, bookingID int64) error {
	return r.UpdateStatus(ctx, tx, bookingID, models.BookingStatusExpired)
}

// MarkAsConfirmed marks a booking as confirmed by admin
func (r *bookingRepo) MarkAsConfirmed(ctx context.Context, tx any, bookingID int64, adminID int64) error {
	query := `
		UPDATE job_bookings
		SET status = 'CONFIRMED',
			confirmed_at = NOW(),
			reviewed_by_admin_id = $2,
			reviewed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`

	var err error
	if tx != nil {
		pgxTx := tx.(pgx.Tx)
		_, err = pgxTx.Exec(ctx, query, bookingID, adminID)
	} else {
		_, err = r.db.Exec(ctx, query, bookingID, adminID)
	}

	return err
}

// MarkAsRejected marks a booking as rejected by admin
func (r *bookingRepo) MarkAsRejected(ctx context.Context, tx any, bookingID int64, adminID int64, reason string) error {
	query := `
		UPDATE job_bookings
		SET status = 'REJECTED',
			rejection_reason = $2,
			reviewed_by_admin_id = $3,
			reviewed_at = NOW(),
			updated_at = NOW()
		WHERE id = $1
	`

	var err error
	if tx != nil {
		pgxTx := tx.(pgx.Tx)
		_, err = pgxTx.Exec(ctx, query, bookingID, reason, adminID)
	} else {
		_, err = r.db.Exec(ctx, query, bookingID, reason, adminID)
	}

	return err
}

// Helper functions for null handling
func toNullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}

func toNullInt64(i int64) sql.NullInt64 {
	return sql.NullInt64{Int64: i, Valid: i != 0}
}

func toNullInt64Ptr(p *int64) sql.NullInt64 {
	if p == nil {
		return sql.NullInt64{Valid: false}
	}
	return sql.NullInt64{Int64: *p, Valid: true}
}

func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{Valid: false}
	}
	return sql.NullTime{Time: *t, Valid: true}
}
