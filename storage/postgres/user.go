package postgres

import (
	"context"
	"errors"
	"fmt"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/storage"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// userRepo implements storage.UserRepoI interface using PostgreSQL
type userRepo struct {
	db  *pgxpool.Pool
	log logger.LoggerI
}

// NewUserRepo creates a new PostgreSQL user repository
func NewUserRepo(db *pgxpool.Pool, log logger.LoggerI) storage.UserRepoI {
	return &userRepo{
		db:  db,
		log: log,
	}
}

// Create creates a new user in the database
func (r *userRepo) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, username, first_name, last_name, state, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.Exec(ctx, query,
		user.ID,
		user.Username,
		user.FirstName,
		user.LastName,
		user.State,
		user.CreatedAt,
		user.UpdatedAt,
	)

	if err != nil {
		// Check for unique constraint violation (user already exists)
		if err.Error() == "duplicate key value violates unique constraint \"users_pkey\"" {
			return storage.ErrAlreadyExists
		}
		r.log.Error("Failed to create user: " + err.Error())
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by their ID
func (r *userRepo) GetByID(ctx context.Context, id int64) (*models.User, error) {
	query := `
		SELECT id, username, first_name, last_name, state, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	var user models.User
	err := r.db.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Username,
		&user.FirstName,
		&user.LastName,
		&user.State,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, storage.ErrNotFound
		}
		r.log.Error("Failed to get user: " + err.Error())
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

// Update updates an existing user
func (r *userRepo) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users
		SET username = $2, first_name = $3, last_name = $4, state = $5, updated_at = $6
		WHERE id = $1
	`

	commandTag, err := r.db.Exec(ctx, query,
		user.ID,
		user.Username,
		user.FirstName,
		user.LastName,
		user.State,
		user.UpdatedAt,
	)

	if err != nil {
		r.log.Error("Failed to update user: " + err.Error())
		return fmt.Errorf("failed to update user: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// Delete deletes a user by their ID
func (r *userRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM users WHERE id = $1`

	commandTag, err := r.db.Exec(ctx, query, id)
	if err != nil {
		r.log.Error("Failed to delete user: " + err.Error())
		return fmt.Errorf("failed to delete user: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// UpdateState updates the user's state
func (r *userRepo) UpdateState(ctx context.Context, id int64, state models.UserState) error {
	query := `
		UPDATE users
		SET state = $2
		WHERE id = $1
	`

	commandTag, err := r.db.Exec(ctx, query, id, state)
	if err != nil {
		r.log.Error("Failed to update user state: " + err.Error())
		return fmt.Errorf("failed to update user state: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// GetOrCreateUser gets a user by ID or creates a new one if not found
func (r *userRepo) GetOrCreateUser(ctx context.Context, id int64, username, firstName, lastName string) (*models.User, error) {
	// First, try to get existing user
	user, err := r.GetByID(ctx, id)
	if err == nil {
		return user, nil
	}

	// If not found, create new user
	if errors.Is(err, storage.ErrNotFound) {
		newUser := models.NewUser(id, username, firstName, lastName)
		if err := r.Create(ctx, newUser); err != nil {
			if errors.Is(err, storage.ErrAlreadyExists) {
				// Race condition: user was created by another request
				// Try to get it again
				return r.GetByID(ctx, id)
			}
			return nil, err
		}
		return newUser, nil
	}

	// Return the error if it's not ErrNotFound
	return nil, err
}

// AddViolation adds a violation record for a user
func (r *userRepo) AddViolation(ctx context.Context, tx any, violation *models.UserViolation) error {
	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return fmt.Errorf("invalid transaction type")
	}

	query := `
		INSERT INTO user_violations (user_id, violation_type, booking_id, admin_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at
	`

	err := pgxTx.QueryRow(ctx, query,
		violation.UserID,
		violation.ViolationType,
		violation.BookingID,
		violation.AdminID,
	).Scan(&violation.ID, &violation.CreatedAt)

	if err != nil {
		r.log.Error("Failed to add violation: " + err.Error())
		return fmt.Errorf("failed to add violation: %w", err)
	}

	return nil
}

// GetViolationCount returns the total number of violations for a user
func (r *userRepo) GetViolationCount(ctx context.Context, tx any, userID int64) (int, error) {
	query := `SELECT COUNT(*) FROM user_violations WHERE user_id = $1`

	var count int
	var err error

	if tx != nil {
		pgxTx, ok := tx.(pgx.Tx)
		if !ok {
			return 0, fmt.Errorf("invalid transaction type")
		}
		err = pgxTx.QueryRow(ctx, query, userID).Scan(&count)
	} else {
		err = r.db.QueryRow(ctx, query, userID).Scan(&count)
	}

	if err != nil {
		r.log.Error("Failed to get violation count: " + err.Error())
		return 0, fmt.Errorf("failed to get violation count: %w", err)
	}

	return count, nil
}

// BlockUser blocks a user
func (r *userRepo) BlockUser(ctx context.Context, tx any, block *models.BlockedUser) error {
	pgxTx, ok := tx.(pgx.Tx)
	if !ok {
		return fmt.Errorf("invalid transaction type")
	}

	query := `
		INSERT INTO blocked_users (user_id, blocked_until, total_violations, blocked_by_admin_id, reason)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) 
		DO UPDATE SET 
			blocked_until = EXCLUDED.blocked_until,
			total_violations = EXCLUDED.total_violations,
			blocked_by_admin_id = EXCLUDED.blocked_by_admin_id,
			reason = EXCLUDED.reason,
			updated_at = NOW()
		RETURNING created_at, updated_at
	`

	err := pgxTx.QueryRow(ctx, query,
		block.UserID,
		block.BlockedUntil,
		block.TotalViolations,
		block.BlockedByAdminID,
		block.Reason,
	).Scan(&block.CreatedAt, &block.UpdatedAt)

	if err != nil {
		r.log.Error("Failed to block user: " + err.Error())
		return fmt.Errorf("failed to block user: %w", err)
	}

	return nil
}

// GetBlockStatus checks if a user is blocked
func (r *userRepo) GetBlockStatus(ctx context.Context, userID int64) (*models.BlockedUser, error) {
	query := `
		SELECT user_id, blocked_until, total_violations, blocked_by_admin_id, reason, created_at, updated_at
		FROM blocked_users
		WHERE user_id = $1
	`

	var block models.BlockedUser
	err := r.db.QueryRow(ctx, query, userID).Scan(
		&block.UserID,
		&block.BlockedUntil,
		&block.TotalViolations,
		&block.BlockedByAdminID,
		&block.Reason,
		&block.CreatedAt,
		&block.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // Not blocked
		}
		r.log.Error("Failed to get block status: " + err.Error())
		return nil, fmt.Errorf("failed to get block status: %w", err)
	}

	return &block, nil
}

// UnblockUser removes a block from a user
func (r *userRepo) UnblockUser(ctx context.Context, userID int64) error {
	query := `DELETE FROM blocked_users WHERE user_id = $1`

	_, err := r.db.Exec(ctx, query, userID)
	if err != nil {
		r.log.Error("Failed to unblock user: " + err.Error())
		return fmt.Errorf("failed to unblock user: %w", err)
	}

	return nil
}
