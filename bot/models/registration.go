package models

import "time"

// RegistrationState represents the current state of user registration
type RegistrationState string

const (
	// Registration states
	RegStateNone          RegistrationState = ""
	RegStateIdle          RegistrationState = "idle" // Not in registration
	RegStatePublicOffer   RegistrationState = "reg_public_offer"
	RegStateFullName      RegistrationState = "reg_full_name"
	RegStatePhone         RegistrationState = "reg_phone"
	RegStateAge           RegistrationState = "reg_age"
	RegStateBodyParams    RegistrationState = "reg_body_params"
	RegStatePassportPhoto RegistrationState = "reg_passport_photo"
	RegStateConfirm       RegistrationState = "reg_confirm"
	RegStateDeclined      RegistrationState = "reg_declined"
	RegStateCompleted     RegistrationState = "reg_completed"
)

// RegistrationDraft holds the temporary registration data during the registration process
type RegistrationDraft struct {
	ID              int64             `json:"id" db:"id"`
	UserID          int64             `json:"user_id" db:"user_id"`
	State           RegistrationState `json:"state" db:"state"`
	FullName        string            `json:"full_name" db:"full_name"`
	Phone           string            `json:"phone" db:"phone"`
	Age             int               `json:"age" db:"age"`
	Weight          int               `json:"weight" db:"weight"`
	Height          int               `json:"height" db:"height"`
	PassportPhotoID string            `json:"passport_photo_id" db:"passport_photo_id"`
	CreatedAt       time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time         `json:"updated_at" db:"updated_at"`
	PreviousState   RegistrationState `json:"-" db:"-"` // Used to track edit mode (not stored in DB)
}

// NewRegistrationDraft creates a new registration draft for a user
func NewRegistrationDraft(userID int64) *RegistrationDraft {
	now := time.Now()
	return &RegistrationDraft{
		UserID:    userID,
		State:     RegStatePublicOffer,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// IsComplete checks if all required fields are filled
func (d *RegistrationDraft) IsComplete() bool {
	return d.FullName != "" &&
		d.Phone != "" &&
		d.Age > 0 &&
		d.Weight > 0 &&
		d.Height > 0
}

// RegisteredUser represents a fully registered user with all required data
type RegisteredUser struct {
	ID              int64     `json:"id" db:"id"`
	UserID          int64     `json:"user_id" db:"user_id"`
	FullName        string    `json:"full_name" db:"full_name"`
	Phone           string    `json:"phone" db:"phone"`
	Age             int       `json:"age" db:"age"`
	Weight          int       `json:"weight" db:"weight"`
	Height          int       `json:"height" db:"height"`
	PassportPhotoID string    `json:"passport_photo_id" db:"passport_photo_id"`
	IsActive        bool      `json:"is_active" db:"is_active"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}

// EditField represents which field the user wants to edit during confirmation
type EditField string

const (
	EditFieldFullName   EditField = "full_name"
	EditFieldPhone      EditField = "phone"
	EditFieldAge        EditField = "age"
	EditFieldBodyParams EditField = "body_params"
)

// RegistrationStateFromString converts a string to RegistrationState
func RegistrationStateFromString(s string) RegistrationState {
	switch s {
	case "reg_public_offer":
		return RegStatePublicOffer
	case "reg_full_name":
		return RegStateFullName
	case "reg_phone":
		return RegStatePhone
	case "reg_age":
		return RegStateAge
	case "reg_body_params":
		return RegStateBodyParams
	case "reg_passport_photo":
		return RegStatePassportPhoto
	case "reg_confirm":
		return RegStateConfirm
	case "reg_declined":
		return RegStateDeclined
	case "reg_completed":
		return RegStateCompleted
	default:
		return RegStateNone
	}
}

// IsRegistrationState checks if the given user state is a registration state
func IsRegistrationState(state UserState) bool {
	regState := RegistrationState(state)
	return regState == RegStatePublicOffer ||
		regState == RegStateFullName ||
		regState == RegStatePhone ||
		regState == RegStateAge ||
		regState == RegStateBodyParams ||
		regState == RegStatePassportPhoto ||
		regState == RegStateConfirm
}
