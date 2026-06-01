package models

import "time"

// JobChannelMessage tracks which message was sent to which channel for a given job.
// This enables scalable multi-channel support: adding a new channel only requires
// a config change (BOT_CHANNEL_IDS), no DB schema or code changes needed.
type JobChannelMessage struct {
	ID        int64     `json:"id"`
	JobID     int64     `json:"job_id"`
	ChannelID int64     `json:"channel_id"`
	MessageID int64     `json:"message_id"`
	CreatedAt time.Time `json:"created_at"`
}
