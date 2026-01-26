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

	// Booking returns the booking repository
	Booking() BookingRepoI

	// Registration returns the registration repository
	Registration() RegistrationRepoI

	// Transaction support
	Transaction() TransactionI
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
	// Job CRUD operations
	Create(ctx context.Context, job *models.Job) error
	GetByID(ctx context.Context, id int64) (*models.Job, error)
	GetByIDForUpdate(ctx context.Context, tx any, id int64) (*models.Job, error) // For row locking
	GetAll(ctx context.Context, status *models.JobStatus) ([]*models.Job, error)
	Update(ctx context.Context, job *models.Job) error
	UpdateStatus(ctx context.Context, id int64, status models.JobStatus) error
	Delete(ctx context.Context, id int64) error

	// Channel message tracking
	UpdateChannelMessageID(ctx context.Context, id int64, messageID int64) error

	// CRITICAL: Race-safe slot management
	// IncrementReservedSlots atomically increments reserved_slots with validation
	IncrementReservedSlots(ctx context.Context, tx any, jobID int64) error

	// DecrementReservedSlots atomically decrements reserved_slots
	DecrementReservedSlots(ctx context.Context, tx any, jobID int64) error

	// MoveReservedToConfirmed atomically moves slot from reserved to confirmed
	MoveReservedToConfirmed(ctx context.Context, tx any, jobID int64) error

	// GetAvailableSlots returns how many slots are available
	GetAvailableSlots(ctx context.Context, jobID int64) (int, error)
}

// BookingRepoI defines the interface for job booking persistence
type BookingRepoI interface {
	// Booking CRUD operations
	Create(ctx context.Context, tx any, booking *models.JobBooking) error
	GetByID(ctx context.Context, id int64) (*models.JobBooking, error)
	GetByIDForUpdate(ctx context.Context, tx any, id int64) (*models.JobBooking, error)
	GetByUserAndJob(ctx context.Context, userID, jobID int64) (*models.JobBooking, error)
	GetByIdempotencyKey(ctx context.Context, tx any, key string) (*models.JobBooking, error)
	Update(ctx context.Context, tx any, booking *models.JobBooking) error
	Delete(ctx context.Context, id int64) error

	// Query operations
	GetExpiredBookings(ctx context.Context, limit int) ([]*models.JobBooking, error)
	GetPendingApprovals(ctx context.Context) ([]*models.JobBooking, error)
	GetUserBookings(ctx context.Context, userID int64) ([]*models.JobBooking, error)
	GetUserBookingsByStatus(ctx context.Context, userID int64, status models.BookingStatus) ([]*models.JobBooking, error)
	GetJobBookings(ctx context.Context, jobID int64) ([]*models.JobBooking, error)

	// State transitions
	UpdateStatus(ctx context.Context, tx any, bookingID int64, status models.BookingStatus) error
	MarkAsExpired(ctx context.Context, tx any, bookingID int64) error
	MarkAsConfirmed(ctx context.Context, tx any, bookingID int64, adminID int64) error
	MarkAsRejected(ctx context.Context, tx any, bookingID int64, adminID int64, reason string) error
}

// TransactionI defines transaction interface
type TransactionI interface {
	Begin(ctx context.Context) (any, error)
	Commit(ctx context.Context, tx any) error
	Rollback(ctx context.Context, tx any) error
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
