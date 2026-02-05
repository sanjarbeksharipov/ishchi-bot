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

// UserViolation represents a user violation record
type UserViolation struct {
	ID            int64     `json:"id"`
	UserID        int64     `json:"user_id"`
	ViolationType string    `json:"violation_type"`
	BookingID     *int64    `json:"booking_id,omitempty"`
	AdminID       *int64    `json:"admin_id,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

// BlockedUser represents a blocked user
type BlockedUser struct {
	UserID           int64      `json:"user_id"`
	BlockedUntil     *time.Time `json:"blocked_until,omitempty"` // nil = permanent
	TotalViolations  int        `json:"total_violations"`
	BlockedByAdminID int64      `json:"blocked_by_admin_id"`
	Reason           string     `json:"reason"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

// UserState represents the current state of a user in the conversation flow
type UserState string

const (
	StateIdle          UserState = "idle"
	StateAwaitingInput UserState = "awaiting_input"
	StateProcessing    UserState = "processing"

	// Job creation states
	StateCreatingJobIshHaqqi      UserState = "creating_job_ish_haqqi"
	StateCreatingJobOvqat         UserState = "creating_job_ovqat"
	StateCreatingJobVaqt          UserState = "creating_job_vaqt"
	StateCreatingJobManzil        UserState = "creating_job_manzil"
	StateCreatingJobLocation      UserState = "creating_job_location"
	StateCreatingJobXizmatHaqqi   UserState = "creating_job_xizmat_haqqi"
	StateCreatingJobAvtobuslar    UserState = "creating_job_avtobuslar"
	StateCreatingJobIshTavsifi    UserState = "creating_job_ish_tavsifi"
	StateCreatingJobIshKuni       UserState = "creating_job_ish_kuni"
	StateCreatingJobKerakli       UserState = "creating_job_kerakli"
	StateCreatingJobEmployerPhone UserState = "creating_job_employer_phone"

	// Job editing states
	StateEditingJobIshHaqqi      UserState = "editing_job_ish_haqqi"
	StateEditingJobOvqat         UserState = "editing_job_ovqat"
	StateEditingJobVaqt          UserState = "editing_job_vaqt"
	StateEditingJobManzil        UserState = "editing_job_manzil"
	StateEditingJobLocation      UserState = "editing_job_location"
	StateEditingJobXizmatHaqqi   UserState = "editing_job_xizmat_haqqi"
	StateEditingJobAvtobuslar    UserState = "editing_job_avtobuslar"
	StateEditingJobIshTavsifi    UserState = "editing_job_ish_tavsifi"
	StateEditingJobIshKuni       UserState = "editing_job_ish_kuni"
	StateEditingJobKerakli       UserState = "editing_job_kerakli"
	StateEditingJobConfirmed     UserState = "editing_job_confirmed"
	StateEditingJobEmployerPhone UserState = "editing_job_employer_phone"

	// Profile editing states
	StateEditingProfileFullName   UserState = "editing_profile_full_name"
	StateEditingProfilePhone      UserState = "editing_profile_phone"
	StateEditingProfileAge        UserState = "editing_profile_age"
	StateEditingProfileBodyParams UserState = "editing_profile_body_params"
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
