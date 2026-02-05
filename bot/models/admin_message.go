package models

import "time"

// AdminJobMessage represents a job detail message sent to an admin
// Allows multiple admins to have independent messages for the same job
type AdminJobMessage struct {
	ID        int64     `json:"id"`
	JobID     int64     `json:"job_id"`
	AdminID   int64     `json:"admin_id"`
	MessageID int64     `json:"message_id"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
