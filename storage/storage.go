package storage

import (
	"context"
	"errors"

	"telegram-bot-starter/bot/models"
)

// Common errors
var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrInvalidInput  = errors.New("invalid input")
)

// StorageI defines the main storage interface
type StorageI interface {
	// CloseDB closes the database connection
	CloseDB()

	// User returns the user repository
	User() UserRepoI
}

// UserRepoI defines the interface for user data persistence
type UserRepoI interface {
	// Create creates a new user
	Create(ctx context.Context, user *models.User) error

	// GetByID retrieves a user by their ID
	GetByID(ctx context.Context, id int64) (*models.User, error)

	// Update updates an existing user
	Update(ctx context.Context, user *models.User) error

	// Delete deletes a user by their ID
	Delete(ctx context.Context, id int64) error

	// UpdateState updates the user's state
	UpdateState(ctx context.Context, id int64, state models.UserState) error

	// GetOrCreateUser gets a user by ID or creates a new one if not found
	GetOrCreateUser(ctx context.Context, id int64, username, firstName, lastName string) (*models.User, error)
}
