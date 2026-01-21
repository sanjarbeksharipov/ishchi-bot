package models

import "time"

// User represents a Telegram user in the system
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	State     UserState `json:"state"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// UserState represents the current state of a user in the conversation flow
type UserState string

const (
	StateIdle          UserState = "idle"
	StateAwaitingInput UserState = "awaiting_input"
	StateProcessing    UserState = "processing"
)

// NewUser creates a new User instance
func NewUser(id int64, username, firstName, lastName string) *User {
	now := time.Now()
	return &User{
		ID:        id,
		Username:  username,
		FirstName: firstName,
		LastName:  lastName,
		State:     StateIdle,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// FullName returns the user's full name
func (u *User) FullName() string {
	if u.LastName != "" {
		return u.FirstName + " " + u.LastName
	}
	return u.FirstName
}
