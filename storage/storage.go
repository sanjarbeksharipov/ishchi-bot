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

	// Job returns the job repository
	Job() JobRepoI

	// Registration returns the registration repository
	Registration() RegistrationRepoI
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

// JobRepoI defines the interface for job data persistence
type JobRepoI interface {
	// Create creates a new job
	Create(ctx context.Context, job *models.Job) error

	// GetByID retrieves a job by ID
	GetByID(ctx context.Context, id int64) (*models.Job, error)

	// GetAll retrieves all jobs with optional status filter
	GetAll(ctx context.Context, status *models.JobStatus) ([]*models.Job, error)

	// Update updates a job
	Update(ctx context.Context, job *models.Job) error

	// UpdateStatus updates only the job status
	UpdateStatus(ctx context.Context, id int64, status models.JobStatus) error

	// UpdateChannelMessageID updates the channel message ID for a job
	UpdateChannelMessageID(ctx context.Context, id int64, messageID int64) error

	// IncrementBookedWorkers increments the booked workers count
	IncrementBookedWorkers(ctx context.Context, id int64) error

	// Delete deletes a job by ID
	Delete(ctx context.Context, id int64) error
}

// RegistrationRepoI defines the interface for registration data persistence
type RegistrationRepoI interface {
	// Draft operations
	// CreateDraft creates a new registration draft
	CreateDraft(ctx context.Context, draft *models.RegistrationDraft) error

	// GetDraftByUserID retrieves a draft by user ID
	GetDraftByUserID(ctx context.Context, userID int64) (*models.RegistrationDraft, error)

	// UpdateDraft updates an existing draft
	UpdateDraft(ctx context.Context, draft *models.RegistrationDraft) error

	// DeleteDraft deletes a draft by user ID
	DeleteDraft(ctx context.Context, userID int64) error

	// Registered user operations
	// CreateRegisteredUser creates a new fully registered user
	CreateRegisteredUser(ctx context.Context, user *models.RegisteredUser) error

	// GetRegisteredUserByUserID retrieves a registered user by Telegram user ID
	GetRegisteredUserByUserID(ctx context.Context, userID int64) (*models.RegisteredUser, error)

	// UpdateRegisteredUser updates a registered user
	UpdateRegisteredUser(ctx context.Context, user *models.RegisteredUser) error

	// IsUserRegistered checks if a user is fully registered
	IsUserRegistered(ctx context.Context, userID int64) (bool, error)

	// DeleteRegisteredUser deletes a registered user (for account deletion)
	DeleteRegisteredUser(ctx context.Context, userID int64) error

	// CompleteRegistration moves a draft to registered_users table
	CompleteRegistration(ctx context.Context, userID int64) error
}
