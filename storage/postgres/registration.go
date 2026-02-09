package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// registrationRepo implements storage.RegistrationRepoI interface using PostgreSQL
type registrationRepo struct {
	db  *pgxpool.Pool
	log logger.LoggerI
}

// NewRegistrationRepo creates a new PostgreSQL registration repository
func NewRegistrationRepo(db *pgxpool.Pool, log logger.LoggerI) storage.RegistrationRepoI {
	return &registrationRepo{
		db:  db,
		log: log,
	}
}

// CreateDraft creates a new registration draft
func (r *registrationRepo) CreateDraft(ctx context.Context, draft *models.RegistrationDraft) error {
	query := `
		INSERT INTO registration_drafts (user_id, state, previous_state, full_name, phone, age, weight, height, passport_photo_id, created_at, updated_at, pending_job_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id
	`

	err := r.db.QueryRow(ctx, query,
		draft.UserID,
		draft.State,
		draft.PreviousState,
		draft.FullName,
		draft.Phone,
		draft.Age,
		draft.Weight,
		draft.Height,
		draft.PassportPhotoID,
		draft.CreatedAt,
		draft.UpdatedAt,
		draft.PendingJobID,
	).Scan(&draft.ID)

	if err != nil {
		r.log.Error("Failed to create registration draft: " + err.Error())
		return fmt.Errorf("failed to create registration draft: %w", err)
	}

	return nil
}

// GetDraftByUserID retrieves a draft by user ID
func (r *registrationRepo) GetDraftByUserID(ctx context.Context, userID int64) (*models.RegistrationDraft, error) {
	query := `
		SELECT id, user_id, state, previous_state, full_name, phone, age, weight, height, passport_photo_id, created_at, updated_at, pending_job_id
		FROM registration_drafts
		WHERE user_id = $1
	`

	var draft models.RegistrationDraft
	var fullName, phone, passportPhotoID *string
	var age, weight, height *int

	err := r.db.QueryRow(ctx, query, userID).Scan(
		&draft.ID,
		&draft.UserID,
		&draft.State,
		&draft.PreviousState,
		&fullName,
		&phone,
		&age,
		&weight,
		&height,
		&passportPhotoID,
		&draft.CreatedAt,
		&draft.UpdatedAt,
		&draft.PendingJobID,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		r.log.Error("Failed to get registration draft: " + err.Error())
		return nil, fmt.Errorf("failed to get registration draft: %w", err)
	}

	// Handle nullable fields
	if fullName != nil {
		draft.FullName = *fullName
	}
	if phone != nil {
		draft.Phone = *phone
	}
	if age != nil {
		draft.Age = *age
	}
	if weight != nil {
		draft.Weight = *weight
	}
	if height != nil {
		draft.Height = *height
	}
	if passportPhotoID != nil {
		draft.PassportPhotoID = *passportPhotoID
	}

	return &draft, nil
}

// UpdateDraft updates an existing draft
func (r *registrationRepo) UpdateDraft(ctx context.Context, draft *models.RegistrationDraft) error {
	query := `
		UPDATE registration_drafts
		SET state = $2, previous_state = $3, full_name = $4, phone = $5, age = $6, weight = $7, height = $8, passport_photo_id = $9, updated_at = $10, pending_job_id = $11
		WHERE user_id = $1
	`

	draft.UpdatedAt = time.Now()

	commandTag, err := r.db.Exec(ctx, query,
		draft.UserID,
		draft.State,
		draft.PreviousState,
		draft.FullName,
		draft.Phone,
		draft.Age,
		draft.Weight,
		draft.Height,
		draft.PassportPhotoID,
		draft.UpdatedAt,
		draft.PendingJobID,
	)

	if err != nil {
		r.log.Error("Failed to update registration draft: " + err.Error())
		return fmt.Errorf("failed to update registration draft: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// DeleteDraft deletes a draft by user ID
func (r *registrationRepo) DeleteDraft(ctx context.Context, userID int64) error {
	query := `DELETE FROM registration_drafts WHERE user_id = $1`

	commandTag, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		r.log.Error("Failed to delete registration draft: " + err.Error())
		return fmt.Errorf("failed to delete registration draft: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// CreateRegisteredUser creates a new fully registered user
func (r *registrationRepo) CreateRegisteredUser(ctx context.Context, user *models.RegisteredUser) error {
	query := `
		INSERT INTO registered_users (user_id, full_name, phone, age, weight, height, passport_photo_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id
	`

	err := r.db.QueryRow(ctx, query,
		user.UserID,
		user.FullName,
		user.Phone,
		user.Age,
		user.Weight,
		user.Height,
		user.PassportPhotoID,
		user.IsActive,
		user.CreatedAt,
		user.UpdatedAt,
	).Scan(&user.ID)

	if err != nil {
		r.log.Error("Failed to create registered user: " + err.Error())
		return fmt.Errorf("failed to create registered user: %w", err)
	}

	return nil
}

// GetRegisteredUserByUserID retrieves a registered user by Telegram user ID
func (r *registrationRepo) GetRegisteredUserByUserID(ctx context.Context, userID int64) (*models.RegisteredUser, error) {
	query := `
		SELECT id, user_id, full_name, phone, age, weight, height, passport_photo_id, is_active, created_at, updated_at
		FROM registered_users
		WHERE user_id = $1
	`

	var user models.RegisteredUser
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&user.ID,
		&user.UserID,
		&user.FullName,
		&user.Phone,
		&user.Age,
		&user.Weight,
		&user.Height,
		&user.PassportPhotoID,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		r.log.Error("Failed to get registered user: " + err.Error())
		return nil, fmt.Errorf("failed to get registered user: %w", err)
	}

	return &user, nil
}

// UpdateRegisteredUser updates a registered user
func (r *registrationRepo) UpdateRegisteredUser(ctx context.Context, user *models.RegisteredUser) error {
	query := `
		UPDATE registered_users
		SET full_name = $2, phone = $3, age = $4, weight = $5, height = $6, passport_photo_id = $7, is_active = $8, updated_at = $9
		WHERE user_id = $1
	`

	user.UpdatedAt = time.Now()

	commandTag, err := r.db.Exec(ctx, query,
		user.UserID,
		user.FullName,
		user.Phone,
		user.Age,
		user.Weight,
		user.Height,
		user.PassportPhotoID,
		user.IsActive,
		user.UpdatedAt,
	)

	if err != nil {
		r.log.Error("Failed to update registered user: " + err.Error())
		return fmt.Errorf("failed to update registered user: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// IsUserRegistered checks if a user is fully registered
func (r *registrationRepo) IsUserRegistered(ctx context.Context, userID int64) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM registered_users WHERE user_id = $1)`

	var exists bool
	err := r.db.QueryRow(ctx, query, userID).Scan(&exists)
	if err != nil {
		r.log.Error("Failed to check if user is registered: " + err.Error())
		return false, fmt.Errorf("failed to check if user is registered: %w", err)
	}

	return exists, nil
}

// DeleteRegisteredUser deletes a registered user
func (r *registrationRepo) DeleteRegisteredUser(ctx context.Context, userID int64) error {
	query := `DELETE FROM registered_users WHERE user_id = $1`

	commandTag, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		r.log.Error("Failed to delete registered user: " + err.Error())
		return fmt.Errorf("failed to delete registered user: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// CompleteRegistration moves a draft to registered_users table
func (r *registrationRepo) CompleteRegistration(ctx context.Context, userID int64) error {
	// Start transaction
	tx, err := r.db.Begin(ctx)
	if err != nil {
		r.log.Error("Failed to begin transaction: " + err.Error())
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Get draft
	draftQuery := `
		SELECT full_name, phone, age, weight, height, passport_photo_id
		FROM registration_drafts
		WHERE user_id = $1
	`

	var fullName, phone, passportPhotoID string
	var age, weight, height int

	err = tx.QueryRow(ctx, draftQuery, userID).Scan(
		&fullName,
		&phone,
		&age,
		&weight,
		&height,
		&passportPhotoID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return storage.ErrNotFound
		}
		r.log.Error("Failed to get draft for completion: " + err.Error())
		return fmt.Errorf("failed to get draft: %w", err)
	}

	// Insert into registered_users
	insertQuery := `
		INSERT INTO registered_users (user_id, full_name, phone, age, weight, height, passport_photo_id, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, true, NOW(), NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			full_name = EXCLUDED.full_name,
			phone = EXCLUDED.phone,
			age = EXCLUDED.age,
			weight = EXCLUDED.weight,
			height = EXCLUDED.height,
			passport_photo_id = EXCLUDED.passport_photo_id,
			is_active = true,
			updated_at = NOW()
	`

	_, err = tx.Exec(ctx, insertQuery,
		userID,
		fullName,
		phone,
		age,
		weight,
		height,
		passportPhotoID,
	)
	if err != nil {
		r.log.Error("Failed to insert registered user: " + err.Error())
		return fmt.Errorf("failed to create registered user: %w", err)
	}

	// Delete draft
	deleteQuery := `DELETE FROM registration_drafts WHERE user_id = $1`
	_, err = tx.Exec(ctx, deleteQuery, userID)
	if err != nil {
		r.log.Error("Failed to delete draft after completion: " + err.Error())
		return fmt.Errorf("failed to delete draft: %w", err)
	}

	// Commit transaction
	if err = tx.Commit(ctx); err != nil {
		r.log.Error("Failed to commit transaction: " + err.Error())
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetAllRegistered retrieves all registered users ordered by creation date (newest first)
func (r *registrationRepo) GetAllRegistered(ctx context.Context) ([]*models.RegisteredUser, error) {
	query := `
		SELECT id, user_id, full_name, phone, age, weight, height, passport_photo_id, is_active, created_at, updated_at
		FROM registered_users
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		r.log.Error("Failed to get all registered users: " + err.Error())
		return nil, fmt.Errorf("failed to get all registered users: %w", err)
	}
	defer rows.Close()

	var users []*models.RegisteredUser

	for rows.Next() {
		var user models.RegisteredUser
		var passportPhotoID *string

		err := rows.Scan(
			&user.ID,
			&user.UserID,
			&user.FullName,
			&user.Phone,
			&user.Age,
			&user.Weight,
			&user.Height,
			&passportPhotoID,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			r.log.Error("Failed to scan registered user: " + err.Error())
			return nil, fmt.Errorf("failed to scan registered user: %w", err)
		}

		if passportPhotoID != nil {
			user.PassportPhotoID = *passportPhotoID
		}

		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		r.log.Error("Error iterating registered users: " + err.Error())
		return nil, fmt.Errorf("error iterating registered users: %w", err)
	}

	return users, nil
}

// GetRegisteredUsersPaginated retrieves registered users with pagination
func (r *registrationRepo) GetRegisteredUsersPaginated(ctx context.Context, limit, offset int) ([]*models.RegisteredUser, error) {
	query := `
		SELECT id, user_id, full_name, phone, age, weight, height, passport_photo_id, is_active, created_at, updated_at
		FROM registered_users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(ctx, query, limit, offset)
	if err != nil {
		r.log.Error("Failed to get paginated registered users: " + err.Error())
		return nil, fmt.Errorf("failed to get paginated registered users: %w", err)
	}
	defer rows.Close()

	var users []*models.RegisteredUser

	for rows.Next() {
		var user models.RegisteredUser
		var passportPhotoID *string

		err := rows.Scan(
			&user.ID,
			&user.UserID,
			&user.FullName,
			&user.Phone,
			&user.Age,
			&user.Weight,
			&user.Height,
			&passportPhotoID,
			&user.IsActive,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			r.log.Error("Failed to scan registered user: " + err.Error())
			return nil, fmt.Errorf("failed to scan registered user: %w", err)
		}

		if passportPhotoID != nil {
			user.PassportPhotoID = *passportPhotoID
		}

		users = append(users, &user)
	}

	if err := rows.Err(); err != nil {
		r.log.Error("Error iterating registered users: " + err.Error())
		return nil, fmt.Errorf("error iterating registered users: %w", err)
	}

	return users, nil
}

// GetTotalRegisteredCount returns the total count of registered users
func (r *registrationRepo) GetTotalRegisteredCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM registered_users`

	var count int
	err := r.db.QueryRow(ctx, query).Scan(&count)
	if err != nil {
		r.log.Error("Failed to get total registered count: " + err.Error())
		return 0, fmt.Errorf("failed to get total registered count: %w", err)
	}

	return count, nil
}
