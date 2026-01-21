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
