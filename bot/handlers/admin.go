package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/config"
	"telegram-bot-starter/pkg/keyboards"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/pkg/messages"

	tele "gopkg.in/telebot.v4"
)

// IsAdmin checks if a user is an admin
func (h *Handler) IsAdmin(userID int64) bool {
	for _, adminID := range h.cfg.Bot.AdminIDs {
		if adminID == userID {
			return true
		}
	}
	return false
}

// HandleAdminPanel shows the admin panel
func (h *Handler) HandleAdminPanel(c tele.Context) error {
	if !h.IsAdmin(c.Sender().ID) {
		return c.Send("❌ Sizda admin huquqi yo'q.")
	}

	return c.Send(messages.MsgAdminPanel, keyboards.AdminMenuKeyboard())
}

// HandleCreateJob starts the job creation flow
func (h *Handler) HandleCreateJob(c tele.Context) error {
	if !h.IsAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Sizda admin huquqi yo'q."})
	}

	ctx := context.Background()

	// Update user state to start job creation
	if err := h.storage.User().UpdateState(ctx, c.Sender().ID, models.StateCreatingJobIshHaqqi); err != nil {
		h.log.Error("Failed to update user state", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Store empty job in session (we'll use user state + temp storage)
	h.setTempJob(c.Sender().ID, &models.Job{
		Status:           models.JobStatusOpen,
		KerakliIshchilar: 1,
	})

	if err := c.Respond(); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}

	return c.Send(messages.MsgEnterIshHaqqi, keyboards.CancelKeyboard())
}

// HandleJobList shows the list of jobs
func (h *Handler) HandleJobList(c tele.Context) error {
	if !h.IsAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Sizda admin huquqi yo'q."})
	}

	ctx := context.Background()
	jobs, err := h.storage.Job().GetAll(ctx, nil)
	if err != nil {
		h.log.Error("Failed to get jobs", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	if len(jobs) == 0 {
		if err := c.Respond(); err != nil {
			h.log.Error("Failed to respond to callback", logger.Error(err))
		}
		return c.Send("📋 Hozircha ishlar yo'q.", keyboards.AdminMenuKeyboard())
	}

	if err := c.Respond(); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}

	return c.Send("📋 Ishlar ro'yxati:", keyboards.JobListKeyboard(jobs))
}

// HandleJobDetail shows job detail with edit options
func (h *Handler) HandleJobDetail(c tele.Context, jobID int64) error {
	if !h.IsAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Sizda admin huquqi yo'q."})
	}

	ctx := context.Background()
	job, err := h.storage.Job().GetByID(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get job", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	if err := c.Respond(); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}

	msg := messages.FormatJobDetailAdmin(job)
	return c.Edit(msg, keyboards.JobDetailKeyboard(job), tele.ModeHTML)
}

// HandleEditJobField starts editing a specific job field
func (h *Handler) HandleEditJobField(c tele.Context, jobID int64, field string) error {
	if !h.IsAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Sizda admin huquqi yo'q."})
	}

	ctx := context.Background()
	job, err := h.storage.Job().GetByID(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get job", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Set the editing state
	var state models.UserState
	var prompt string

	switch field {
	case "ish_haqqi":
		state = models.StateEditingJobIshHaqqi
		prompt = messages.MsgEnterIshHaqqi
	case "ovqat":
		state = models.StateEditingJobOvqat
		prompt = messages.MsgEnterOvqat
	case "vaqt":
		state = models.StateEditingJobVaqt
		prompt = messages.MsgEnterVaqt
	case "manzil":
		state = models.StateEditingJobManzil
		prompt = messages.MsgEnterManzil
	case "xizmat_haqqi":
		state = models.StateEditingJobXizmatHaqqi
		prompt = messages.MsgEnterXizmatHaqqi
	case "avtobuslar":
		state = models.StateEditingJobAvtobuslar
		prompt = messages.MsgEnterAvtobuslar
	case "qoshimcha":
		state = models.StateEditingJobQoshimcha
		prompt = messages.MsgEnterQoshimcha
	case "ish_kuni":
		state = models.StateEditingJobIshKuni
		prompt = messages.MsgEnterIshKuni
	case "kerakli":
		state = models.StateEditingJobKerakli
		prompt = messages.MsgEnterKerakliIshchilar
	default:
		return c.Respond(&tele.CallbackResponse{Text: "❌ Noto'g'ri maydon"})
	}

	if err := h.storage.User().UpdateState(ctx, c.Sender().ID, state); err != nil {
		h.log.Error("Failed to update user state", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Store job being edited
	h.setEditingJobID(c.Sender().ID, job.ID)

	if err := c.Respond(); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}

	return c.Send(prompt+"\n\nJoriy qiymat: "+getJobFieldValue(job, field), keyboards.CancelEditKeyboard(job.ID))
}

// HandleChangeJobStatus changes the job status
func (h *Handler) HandleChangeJobStatus(c tele.Context, jobID int64, status models.JobStatus) error {
	if !h.IsAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Sizda admin huquqi yo'q."})
	}

	ctx := context.Background()

	// Update status in database
	if err := h.storage.Job().UpdateStatus(ctx, jobID, status); err != nil {
		h.log.Error("Failed to update job status", logger.Error(err))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Xatolik yuz berdi"})
	}

	// Get updated job
	job, err := h.storage.Job().GetByID(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get job", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Update channel message if exists
	if job.ChannelMessageID != 0 {
		h.updateChannelMessage(job)
	}

	if err := c.Respond(&tele.CallbackResponse{Text: "✅ Status yangilandi"}); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}

	// Show updated job detail
	msg := messages.FormatJobDetailAdmin(job)
	return c.Edit(msg, keyboards.JobDetailKeyboard(job), tele.ModeHTML)
}

// HandlePublishJob publishes the job to the channel
func (h *Handler) HandlePublishJob(c tele.Context, jobID int64) error {
	if !h.IsAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Sizda admin huquqi yo'q."})
	}

	ctx := context.Background()
	job, err := h.storage.Job().GetByID(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get job", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Format job message for channel
	msg := messages.FormatJobForChannel(job)

	// Create inline keyboard with signup button
	signupBtn := keyboards.JobSignupKeyboard(job.ID, h.cfg.Bot.Username)

	// Send to channel
	channelID := tele.ChatID(h.cfg.Bot.ChannelID)
	sentMsg, err := h.bot.Send(channelID, msg, signupBtn, tele.ModeHTML)
	if err != nil {
		h.log.Error("Failed to send job to channel", logger.Error(err))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Kanalga yuborishda xatolik"})
	}

	// Save channel message ID
	if err := h.storage.Job().UpdateChannelMessageID(ctx, job.ID, int64(sentMsg.ID)); err != nil {
		h.log.Error("Failed to save channel message ID", logger.Error(err))
	}

	job.ChannelMessageID = int64(sentMsg.ID)

	if err := c.Respond(&tele.CallbackResponse{Text: "✅ Kanalga yuborildi!"}); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}

	// Update job detail view
	detailMsg := messages.FormatJobDetailAdmin(job)
	return c.Edit(detailMsg, keyboards.JobDetailKeyboard(job), tele.ModeHTML)
}

// HandleDeleteJob deletes a job
func (h *Handler) HandleDeleteJob(c tele.Context, jobID int64) error {
	if !h.IsAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Sizda admin huquqi yo'q."})
	}

	ctx := context.Background()

	// Get job first to delete channel message
	job, err := h.storage.Job().GetByID(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get job", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Delete channel message if exists
	if job.ChannelMessageID != 0 {
		msgToDelete := &tele.Message{ID: int(job.ChannelMessageID), Chat: &tele.Chat{ID: h.cfg.Bot.ChannelID}}
		if err := h.bot.Delete(msgToDelete); err != nil {
			h.log.Error("Failed to delete channel message", logger.Error(err))
		}
	}

	// Delete from database
	if err := h.storage.Job().Delete(ctx, jobID); err != nil {
		h.log.Error("Failed to delete job", logger.Error(err))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Xatolik yuz berdi"})
	}

	if err := c.Respond(&tele.CallbackResponse{Text: "✅ Ish o'chirildi"}); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}

	return c.Edit("✅ Ish muvaffaqiyatli o'chirildi.", keyboards.AdminMenuKeyboard())
}

// HandleAdminTextInput handles text input during job creation/editing
func (h *Handler) HandleAdminTextInput(c tele.Context, user *models.User) error {
	text := strings.TrimSpace(c.Text())

	// Handle job creation flow
	if strings.HasPrefix(string(user.State), "creating_job_") {
		return h.handleJobCreationInput(c, user, text)
	}

	// Handle job editing flow
	if strings.HasPrefix(string(user.State), "editing_job_") {
		return h.handleJobEditingInput(c, user, text)
	}

	return nil
}

func (h *Handler) handleJobCreationInput(c tele.Context, user *models.User, text string) error {
	ctx := context.Background()
	job := h.getTempJob(c.Sender().ID)
	if job == nil {
		job = &models.Job{Status: models.JobStatusOpen, KerakliIshchilar: 1}
	}

	var nextState models.UserState
	var nextPrompt string

	switch user.State {
	case models.StateCreatingJobIshHaqqi:
		job.IshHaqqi = text
		nextState = models.StateCreatingJobOvqat
		nextPrompt = messages.MsgEnterOvqat

	case models.StateCreatingJobOvqat:
		job.Ovqat = text
		nextState = models.StateCreatingJobVaqt
		nextPrompt = messages.MsgEnterVaqt

	case models.StateCreatingJobVaqt:
		job.Vaqt = text
		nextState = models.StateCreatingJobManzil
		nextPrompt = messages.MsgEnterManzil

	case models.StateCreatingJobManzil:
		job.Manzil = text
		nextState = models.StateCreatingJobXizmatHaqqi
		nextPrompt = messages.MsgEnterXizmatHaqqi

	case models.StateCreatingJobXizmatHaqqi:
		xizmatHaqqi, err := strconv.Atoi(text)
		if err != nil {
			return c.Send("❌ Iltimos, raqam kiriting. Masalan: 9990")
		}
		job.XizmatHaqqi = xizmatHaqqi
		nextState = models.StateCreatingJobAvtobuslar
		nextPrompt = messages.MsgEnterAvtobuslar

	case models.StateCreatingJobAvtobuslar:
		job.Avtobuslar = text
		nextState = models.StateCreatingJobQoshimcha
		nextPrompt = messages.MsgEnterQoshimcha

	case models.StateCreatingJobQoshimcha:
		job.Qoshimcha = text
		nextState = models.StateCreatingJobIshKuni
		nextPrompt = messages.MsgEnterIshKuni

	case models.StateCreatingJobIshKuni:
		job.IshKuni = text
		nextState = models.StateCreatingJobKerakli
		nextPrompt = messages.MsgEnterKerakliIshchilar

	case models.StateCreatingJobKerakli:
		kerakli, err := strconv.Atoi(text)
		if err != nil || kerakli < 1 {
			return c.Send("❌ Iltimos, 1 dan katta raqam kiriting.")
		}
		job.KerakliIshchilar = kerakli

		// Save job to database
		job.CreatedBy = c.Sender().ID
		if err := h.storage.Job().Create(ctx, job); err != nil {
			h.log.Error("Failed to create job", logger.Error(err))
			return c.Send(messages.MsgError)
		}

		// Reset user state
		if err := h.storage.User().UpdateState(ctx, c.Sender().ID, models.StateIdle); err != nil {
			h.log.Error("Failed to update user state", logger.Error(err))
		}

		// Clear temp job
		h.clearTempJob(c.Sender().ID)

		// Show job preview with publish option
		msg := fmt.Sprintf("✅ Ish yaratildi!\n\n%s", messages.FormatJobDetailAdmin(job))
		return c.Send(msg, keyboards.JobDetailKeyboard(job), tele.ModeHTML)
	}

	// Update temp job and state
	h.setTempJob(c.Sender().ID, job)
	if err := h.storage.User().UpdateState(ctx, c.Sender().ID, nextState); err != nil {
		h.log.Error("Failed to update user state", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	return c.Send(nextPrompt, keyboards.CancelKeyboard())
}

func (h *Handler) handleJobEditingInput(c tele.Context, user *models.User, text string) error {
	ctx := context.Background()
	jobID := h.getEditingJobID(c.Sender().ID)
	if jobID == 0 {
		return c.Send(messages.MsgError)
	}

	job, err := h.storage.Job().GetByID(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get job", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	switch user.State {
	case models.StateEditingJobIshHaqqi:
		job.IshHaqqi = text
	case models.StateEditingJobOvqat:
		job.Ovqat = text
	case models.StateEditingJobVaqt:
		job.Vaqt = text
	case models.StateEditingJobManzil:
		job.Manzil = text
	case models.StateEditingJobXizmatHaqqi:
		xizmatHaqqi, err := strconv.Atoi(text)
		if err != nil {
			return c.Send("❌ Iltimos, raqam kiriting. Masalan: 9990")
		}
		job.XizmatHaqqi = xizmatHaqqi
	case models.StateEditingJobAvtobuslar:
		job.Avtobuslar = text
	case models.StateEditingJobQoshimcha:
		job.Qoshimcha = text
	case models.StateEditingJobIshKuni:
		job.IshKuni = text
	case models.StateEditingJobKerakli:
		kerakli, err := strconv.Atoi(text)
		if err != nil || kerakli < 1 {
			return c.Send("❌ Iltimos, 1 dan katta raqam kiriting.")
		}
		job.KerakliIshchilar = kerakli
	}

	// Update job in database
	if err := h.storage.Job().Update(ctx, job); err != nil {
		h.log.Error("Failed to update job", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Update channel message if exists
	if job.ChannelMessageID != 0 {
		h.updateChannelMessage(job)
	}

	// Reset user state
	if err := h.storage.User().UpdateState(ctx, c.Sender().ID, models.StateIdle); err != nil {
		h.log.Error("Failed to update user state", logger.Error(err))
	}

	// Clear editing job ID
	h.clearEditingJobID(c.Sender().ID)

	// Show updated job detail
	msg := fmt.Sprintf("✅ Yangilandi!\n\n%s", messages.FormatJobDetailAdmin(job))
	return c.Send(msg, keyboards.JobDetailKeyboard(job), tele.ModeHTML)
}

// HandleCancelJobCreation cancels the job creation flow
func (h *Handler) HandleCancelJobCreation(c tele.Context) error {
	ctx := context.Background()

	if err := h.storage.User().UpdateState(ctx, c.Sender().ID, models.StateIdle); err != nil {
		h.log.Error("Failed to update user state", logger.Error(err))
	}

	h.clearTempJob(c.Sender().ID)
	h.clearEditingJobID(c.Sender().ID)

	if err := c.Respond(&tele.CallbackResponse{Text: "❌ Bekor qilindi"}); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}

	return c.Edit(messages.MsgAdminPanel, keyboards.AdminMenuKeyboard())
}

// Helper to update channel message
func (h *Handler) updateChannelMessage(job *models.Job) {
	msg := &tele.Message{
		ID:   int(job.ChannelMessageID),
		Chat: &tele.Chat{ID: h.cfg.Bot.ChannelID},
	}

	channelMsg := messages.FormatJobForChannel(job)
	signupBtn := keyboards.JobSignupKeyboard(job.ID, h.cfg.Bot.Username)

	if _, err := h.bot.Edit(msg, channelMsg, signupBtn, tele.ModeHTML); err != nil {
		h.log.Error("Failed to update channel message", logger.Error(err))
	}
}

// Helper to get job field value for display
func getJobFieldValue(job *models.Job, field string) string {
	switch field {
	case "ish_haqqi":
		return job.IshHaqqi
	case "ovqat":
		return job.Ovqat
	case "vaqt":
		return job.Vaqt
	case "manzil":
		return job.Manzil
	case "xizmat_haqqi":
		return fmt.Sprintf("%d", job.XizmatHaqqi)
	case "avtobuslar":
		return job.Avtobuslar
	case "qoshimcha":
		return job.Qoshimcha
	case "ish_kuni":
		return job.IshKuni
	case "kerakli":
		return fmt.Sprintf("%d", job.KerakliIshchilar)
	default:
		return ""
	}
}

// SetConfig sets the config for admin handlers
func (h *Handler) SetConfig(cfg *config.Config) {
	h.cfg = cfg
}
