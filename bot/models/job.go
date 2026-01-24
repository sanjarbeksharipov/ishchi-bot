package models

import "time"

// JobStatus represents the status of a job posting
type JobStatus string

const (
	JobStatusOpen   JobStatus = "open"
	JobStatusToldi  JobStatus = "toldi" // Full
	JobStatusClosed JobStatus = "closed"
)

// Job represents a job posting
type Job struct {
	ID               int64     `json:"id"`
	OrderNumber      int       `json:"order_number"`
	IshHaqqi         string    `json:"ish_haqqi"`
	Ovqat            string    `json:"ovqat"`
	Vaqt             string    `json:"vaqt"`
	Manzil           string    `json:"manzil"`
	XizmatHaqqi      int       `json:"xizmat_haqqi"`
	Avtobuslar       string    `json:"avtobuslar"`
	Qoshimcha        string    `json:"qoshimcha"`
	IshKuni          string    `json:"ish_kuni"`
	Status           JobStatus `json:"status"`
	KerakliIshchilar int       `json:"kerakli_ishchilar"`
	BandIshchilar    int       `json:"band_ishchilar"`
	ChannelMessageID int64     `json:"channel_message_id"`
	CreatedBy        int64     `json:"created_by"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// JobStatusDisplay returns the display text for job status
func (s JobStatus) Display() string {
	switch s {
	case JobStatusOpen:
		return "🟢 Ochiq"
	case JobStatusToldi:
		return "🔴 To'ldi"
	case JobStatusClosed:
		return "⚫ Yopilgan"
	default:
		return string(s)
	}
}

// IsValid checks if the status is valid
func (s JobStatus) IsValid() bool {
	switch s {
	case JobStatusOpen, JobStatusToldi, JobStatusClosed:
		return true
	}
	return false
}
