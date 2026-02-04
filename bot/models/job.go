package models

import "time"

// JobStatus represents the status of a job posting
type JobStatus string

const (
	JobStatusDraft     JobStatus = "DRAFT"     // Admin creating job (not published)
	JobStatusActive    JobStatus = "ACTIVE"    // Published to channel, accepting bookings
	JobStatusFull      JobStatus = "FULL"      // All slots taken
	JobStatusCompleted JobStatus = "COMPLETED" // Job finished
	JobStatusCancelled JobStatus = "CANCELLED" // Job cancelled by admin
)

// Job represents a job posting with race-safe slot management
type Job struct {
	ID          int64 `json:"id"`
	OrderNumber int   `json:"order_number"`

	// Job details
	Salary         string `json:"salary"`          // Ish haqqi
	Food           string `json:"food"`            // Ovqat
	WorkTime       string `json:"work_time"`       // Vaqt
	Address        string `json:"address"`         // Manzil
	ServiceFee     int    `json:"service_fee"`     // Xizmat haqqi
	Buses          string `json:"buses"`           // Avtobuslar
	AdditionalInfo string `json:"additional_info"` // Qo'shimcha
	WorkDate       string `json:"work_date"`       // Ish kuni
	EmployerPhone  string `json:"employer_phone"`  // Ish beruvchining telefon raqami (faqat tasdiqlangan foydalanuvchilar uchun)

	// Slot management (CRITICAL for race conditions)
	RequiredWorkers int `json:"required_workers"` // Total slots needed
	ReservedSlots   int `json:"reserved_slots"`   // Temporarily held (3-min timer)
	ConfirmedSlots  int `json:"confirmed_slots"`  // Admin-approved bookings

	// Status and metadata
	Status           JobStatus `json:"status"`
	ChannelMessageID int64     `json:"channel_message_id"`
	AdminMessageID   int64     `json:"admin_message_id"` // Admin job detail message ID for single-message enforcement
	CreatedByAdminID int64     `json:"created_by_admin_id"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// Backwards compatibility aliases
type (
	IshHaqqi         = string
	Ovqat            = string
	Vaqt             = string
	Manzil           = string
	XizmatHaqqi      = int
	Avtobuslar       = string
	Qoshimcha        = string
	IshKuni          = string
	KerakliIshchilar = int
	BandIshchilar    = int
)

// JobStatusDisplay returns the display text for job status
func (s JobStatus) Display() string {
	switch s {
	case JobStatusDraft:
		return "📝 Qoralama"
	case JobStatusActive:
		return "🟢 Faol"
	case JobStatusFull:
		return "🔴 To'ldi"
	case JobStatusCompleted:
		return "⚫ Yakunlandi"
	case JobStatusCancelled:
		return "⚫ Bekor qilindi"
	default:
		return string(s)
	}
}

// IsValid checks if the status is valid
func (s JobStatus) IsValid() bool {
	switch s {
	case JobStatusDraft, JobStatusActive, JobStatusFull, JobStatusCompleted, JobStatusCancelled:
		return true
	default:
		return false
	}
}

// AvailableSlots returns how many slots are still available for reservation
func (j *Job) AvailableSlots() int {
	occupied := j.ReservedSlots + j.ConfirmedSlots
	available := j.RequiredWorkers - occupied
	if available < 0 {
		return 0
	}
	return available
}

// IsFull checks if the job has no available slots
func (j *Job) IsFull() bool {
	return j.AvailableSlots() <= 0
}

// IsActive checks if the job is accepting bookings
func (j *Job) IsActive() bool {
	return j.Status == JobStatusActive && !j.IsFull()
}
