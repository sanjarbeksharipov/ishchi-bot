package handlers

import (
	"sync"

	"telegram-bot-starter/bot/models"
)

// In-memory session storage for job creation
// In production, consider using Redis or database
var (
	tempJobs      = make(map[int64]*models.Job)
	tempJobsMu    sync.RWMutex
	editingJobIDs = make(map[int64]int64)
	editingMu     sync.RWMutex
)

func (h *Handler) setTempJob(userID int64, job *models.Job) {
	tempJobsMu.Lock()
	defer tempJobsMu.Unlock()
	tempJobs[userID] = job
}

func (h *Handler) getTempJob(userID int64) *models.Job {
	tempJobsMu.RLock()
	defer tempJobsMu.RUnlock()
	return tempJobs[userID]
}

func (h *Handler) clearTempJob(userID int64) {
	tempJobsMu.Lock()
	defer tempJobsMu.Unlock()
	delete(tempJobs, userID)
}

func (h *Handler) setEditingJobID(userID int64, jobID int64) {
	editingMu.Lock()
	defer editingMu.Unlock()
	editingJobIDs[userID] = jobID
}

func (h *Handler) getEditingJobID(userID int64) int64 {
	editingMu.RLock()
	defer editingMu.RUnlock()
	return editingJobIDs[userID]
}

func (h *Handler) clearEditingJobID(userID int64) {
	editingMu.Lock()
	defer editingMu.Unlock()
	delete(editingJobIDs, userID)
}
