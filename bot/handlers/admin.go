package handlers

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/config"
	"telegram-bot-starter/pkg/keyboards"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/pkg/messages"

	tele "gopkg.in/telebot.v4"
)

// IsAdmin checks if a user is an admin
func (h *Handler) IsAdmin(userID int64) bool {
	return slices.Contains(h.cfg.Bot.AdminIDs, userID)
}

// HandleAdminPanel shows the admin panel
func (h *Handler) HandleAdminPanel(c tele.Context) error {
	if !h.IsAdmin(c.Sender().ID) {
		return c.Send("❌ Sizda admin huquqi yo'q.")
	}

	return c.Send(messages.MsgAdminPanel, keyboards.AdminMenuReplyKeyboard())
}

// HandleCreateJob starts the job creation flow
func (h *Handler) HandleCreateJob(c tele.Context) error {
	if !h.IsAdmin(c.Sender().ID) {
		return c.Send("❌ Sizda admin huquqi yo'q.")
	}

	ctx := context.Background()

	// Update user state to start job creation
	if err := h.storage.User().UpdateState(ctx, c.Sender().ID, models.StateCreatingJobIshHaqqi); err != nil {
		h.log.Error("Failed to update user state", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Store empty job in session (we'll use user state + temp storage)
	h.setTempJob(c.Sender().ID, &models.Job{
		Status:          models.JobStatusActive,
		RequiredWorkers: 1,
	})

	return c.Send(messages.MsgEnterIshHaqqi, keyboards.CancelKeyboard())
}

// HandleJobList shows the list of jobs
func (h *Handler) HandleJobList(c tele.Context) error {
	if !h.IsAdmin(c.Sender().ID) {
		return c.Send("❌ Sizda admin huquqi yo'q.")
	}

	ctx := context.Background()
	jobs, err := h.storage.Job().GetAll(ctx, nil)
	if err != nil {
		h.log.Error("Failed to get jobs", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	if len(jobs) == 0 {
		// if err := c.Respond(); err != nil {
		// 	h.log.Error("Failed to respond to callback", logger.Error(err))
		// }
		return c.Send("📋 Hozircha ishlar yo'q.", keyboards.AdminMenuReplyKeyboard())
	}

	// if err := c.Respond(); err != nil {
	// 	h.log.Error("Failed to respond to callback", logger.Error(err))
	// }

	return c.Send("📋 Ishlar ro'yxati:", keyboards.JobListKeyboard(jobs))
}

// HandleJobDetail shows job detail with edit options
// Implements single-message per admin: each admin has their own independent message
func (h *Handler) HandleJobDetail(c tele.Context, jobIDStr string) error {
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		h.log.Error("Invalid job ID in callback", logger.Error(err), logger.Any("job_id_str", jobIDStr))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Noto'g'ri ish ID"})
	}

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

	// Single-message enforcement per admin: Delete this admin's previous message if exists
	h.deleteAdminMessageForAdmin(job.ID, c.Sender().ID)

	// Send new admin message
	msg := messages.FormatJobDetailAdmin(job)
	sentMsg, err := c.Bot().Send(c.Sender(), msg, keyboards.JobDetailKeyboard(job), tele.ModeHTML)
	if err != nil {
		h.log.Error("Failed to send job detail", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Save new admin message ID to database
	adminMsg := &models.AdminJobMessage{
		JobID:     jobID,
		AdminID:   c.Sender().ID,
		MessageID: int64(sentMsg.ID),
	}
	if err := h.storage.AdminMessage().Upsert(ctx, adminMsg); err != nil {
		h.log.Error("Failed to save admin message ID", logger.Error(err))
	}

	return nil
}

// HandleEditJobField starts editing a specific job field
func (h *Handler) HandleEditJobField(c tele.Context, params string) error {
	parts := strings.Split(params, "_")
	if len(parts) < 2 {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Noto'g'ri parametrlar"})
	}

	jobID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		h.log.Error("Invalid job ID in callback", logger.Error(err), logger.Any("job_id_str", parts[0]))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Noto'g'ri ish ID"})
	}

	field := strings.Join(parts[1:], "_")

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
	case "location":
		state = models.StateEditingJobLocation
		prompt = messages.MsgEnterLocation
	case "xizmat_haqqi":
		state = models.StateEditingJobXizmatHaqqi
		prompt = messages.MsgEnterXizmatHaqqi
	case "avtobuslar":
		state = models.StateEditingJobAvtobuslar
		prompt = messages.MsgEnterAvtobuslar
	case "ish_tavsifi":
		state = models.StateEditingJobIshTavsifi
		prompt = messages.MsgEnterIshTavsifi
	case "ish_kuni":
		state = models.StateEditingJobIshKuni
		prompt = messages.MsgEnterIshKuni
	case "kerakli":
		state = models.StateEditingJobKerakli
		prompt = messages.MsgEnterKerakliIshchilar
	case "confirmed":
		state = models.StateEditingJobConfirmed
		prompt = messages.MsgEnterConfirmedSlots
	case "employer_phone":
		state = models.StateEditingJobEmployerPhone
		prompt = messages.MsgEnterEmployerPhone
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

	// Special handling for location field - send as Telegram location
	if state == models.StateEditingJobLocation && job.Location != "" {
		// Parse and send current location
		parts := strings.Split(job.Location, ",")
		if len(parts) == 2 {
			lat, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
			lng, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)

			if err1 == nil && err2 == nil {
				// Send prompt first
				c.Send(prompt, keyboards.CancelEditKeyboard(job.ID))

				// Send current location
				location := &tele.Location{
					Lat: float32(lat),
					Lng: float32(lng),
				}

				_, err := h.bot.Send(c.Sender(), location)
				if err != nil {
					h.log.Error("Failed to send current location", logger.Error(err))
				} else {
					return c.Send("📌 <b>Joriy qiymat yuqorida ko'rsatilgan</b>", tele.ModeHTML)
				}
			}
		}
		// Fallback if parsing fails
		return c.Send(prompt+"\n\nJoriy qiymat: "+job.Location, keyboards.CancelEditKeyboard(job.ID))
	}

	// Use special keyboard with skip button for buses field
	if state == models.StateEditingJobAvtobuslar {
		return c.Send(prompt+"\n\nJoriy qiymat: "+getJobFieldValue(job, field), keyboards.CancelOrSkipKeyboard())
	}

	return c.Send(prompt+"\n\nJoriy qiymat: "+getJobFieldValue(job, field), keyboards.CancelEditKeyboard(job.ID))
}

// HandleChangeJobStatus changes the job status
// Implements single-message enforcement
func (h *Handler) HandleChangeJobStatus(c tele.Context, params string) error {

	if !h.IsAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Sizda admin huquqi yo'q."})
	}
	// Format: job_status_{id}_{status}
	parts := strings.Split(params, "_")
	if len(parts) != 2 {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Noto'g'ri parametrlar"})
	}

	jobID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		h.log.Error("Invalid job ID in callback", logger.Error(err), logger.Any("job_id_str", parts[0]))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Noto'g'ri ish ID"})
	}

	statusStr := parts[1]
	var status models.JobStatus
	switch statusStr {
	case "open":
		status = models.JobStatusActive
	case "toldi":
		status = models.JobStatusFull
	case "closed":
		status = models.JobStatusCompleted
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

	// Update ALL admin messages (broadcasts to all admins)
	h.updateAllAdminMessages(job)

	// Show updated job detail to current admin
	msg := messages.FormatJobDetailAdmin(job)
	return c.Edit(msg, keyboards.JobDetailKeyboard(job), tele.ModeHTML)
}

// HandlePublishJob publishes the job to the channel (only if not yet published)
func (h *Handler) HandlePublishJob(c tele.Context, jobIDStr string) error {
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		h.log.Error("Invalid job ID in callback", logger.Error(err), logger.Any("job_id_str", jobIDStr))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Noto'g'ri ish ID"})
	}

	if !h.IsAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Sizda admin huquqi yo'q."})
	}

	ctx := context.Background()
	job, err := h.storage.Job().GetByID(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get job", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Check if already published - should not happen with proper UI
	if job.ChannelMessageID != 0 {
		return c.Respond(&tele.CallbackResponse{Text: "⚠️ Bu ish allaqachon kanalda"})
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

	// Send location as a reply to the channel message if it exists
	if job.Location != "" {
		parts := strings.SplitN(job.Location, ",", 2)
		if len(parts) == 2 {
			lat, errLat := strconv.ParseFloat(strings.TrimSpace(parts[0]), 32)
			lng, errLng := strconv.ParseFloat(strings.TrimSpace(parts[1]), 32)
			if errLat == nil && errLng == nil {
				location := &tele.Location{
					Lat: float32(lat),
					Lng: float32(lng),
				}
				_, err := h.bot.Send(channelID, location, &tele.SendOptions{
					ReplyTo: sentMsg,
				})
				if err != nil {
					h.log.Error("Failed to send location to channel",
						logger.Error(err),
						logger.Any("job_id", job.ID),
					)
				}
			}
		}
	}

	if err := c.Respond(&tele.CallbackResponse{Text: "✅ Kanalga yuborildi!"}); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}

	// Update ALL admin messages (broadcast to all admins)
	h.updateAllAdminMessages(job)

	// Update current admin's message view
	detailMsg := messages.FormatJobDetailAdmin(job)
	return c.Edit(detailMsg, keyboards.JobDetailKeyboard(job), tele.ModeHTML)
}

// HandleDeleteChannelMessage deletes the channel message only (keeps job in DB)
func (h *Handler) HandleDeleteChannelMessage(c tele.Context, jobIDStr string) error {
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		h.log.Error("Invalid job ID in callback", logger.Error(err), logger.Any("job_id_str", jobIDStr))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Noto'g'ri ish ID"})
	}

	if !h.IsAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Sizda admin huquqi yo'q."})
	}

	ctx := context.Background()
	job, err := h.storage.Job().GetByID(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get job", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Check if channel message exists
	if job.ChannelMessageID == 0 {
		return c.Respond(&tele.CallbackResponse{Text: "⚠️ Kanal xabari mavjud emas"})
	}

	// Delete channel message
	msgToDelete := &tele.Message{ID: int(job.ChannelMessageID), Chat: &tele.Chat{ID: h.cfg.Bot.ChannelID}}
	if err := h.bot.Delete(msgToDelete); err != nil {
		h.log.Error("Failed to delete channel message", logger.Error(err))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Xabarni o'chirishda xatolik"})
	}

	// Clear channel message ID from job
	if err := h.storage.Job().UpdateChannelMessageID(ctx, job.ID, 0); err != nil {
		h.log.Error("Failed to clear channel message ID", logger.Error(err))
	}

	job.ChannelMessageID = 0

	if err := c.Respond(&tele.CallbackResponse{Text: "✅ Kanal xabari o'chirildi"}); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}

	// Update ALL admin messages (broadcast channel message deletion to all admins)
	h.updateAllAdminMessages(job)

	// Show updated job detail to current admin
	msg := messages.FormatJobDetailAdmin(job)
	return c.Edit(msg, keyboards.JobDetailKeyboard(job), tele.ModeHTML)
}

// HandleDeleteJob deletes the entire job from database (and channel message if exists)
func (h *Handler) HandleDeleteJob(c tele.Context, jobIDStr string) error {
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		h.log.Error("Invalid job ID in callback", logger.Error(err), logger.Any("job_id_str", jobIDStr))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Noto'g'ri ish ID"})
	}

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

	// Delete ALL admin messages from Telegram chats
	h.deleteAllAdminMessages(jobID)

	// Delete from database (will cascade delete admin_job_messages)
	if err := h.storage.Job().Delete(ctx, jobID); err != nil {
		h.log.Error("Failed to delete job", logger.Error(err))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Xatolik yuz berdi"})
	}

	if err := c.Respond(&tele.CallbackResponse{Text: "✅ Ish o'chirildi"}); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}

	c.Delete()
	return c.Send("✅ Ish muvaffaqiyatli o'chirildi.", keyboards.AdminMenuReplyKeyboard())
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
		job = &models.Job{Status: models.JobStatusDraft, RequiredWorkers: 1}
	}

	var nextState models.UserState
	var nextPrompt string

	switch user.State {
	case models.StateCreatingJobIshHaqqi:
		job.Salary = text
		nextState = models.StateCreatingJobOvqat
		nextPrompt = messages.MsgEnterOvqat

	case models.StateCreatingJobOvqat:
		job.Food = text
		nextState = models.StateCreatingJobVaqt
		nextPrompt = messages.MsgEnterVaqt

	case models.StateCreatingJobVaqt:
		job.WorkTime = text
		nextState = models.StateCreatingJobManzil
		nextPrompt = messages.MsgEnterManzil

	case models.StateCreatingJobManzil:
		job.Address = text
		nextState = models.StateCreatingJobLocation
		nextPrompt = messages.MsgEnterLocation
		// Location will be handled by HandleLocation, not text input

	case models.StateCreatingJobLocation:
		// This state is handled by HandleLocation, not text
		// But if user sends text, we'll accept it as fallback
		// Allow skipping location field
		if text == "Skip" || text == "skip" || text == "-" {
			job.Location = ""
		} else {
			job.Location = text
		}
		nextState = models.StateCreatingJobXizmatHaqqi
		nextPrompt = messages.MsgEnterXizmatHaqqi

	case models.StateCreatingJobXizmatHaqqi:
		xizmatHaqqi, err := strconv.Atoi(text)
		if err != nil {
			return c.Send("❌ Iltimos, raqam kiriting. Masalan: 9990")
		}
		job.ServiceFee = xizmatHaqqi
		nextState = models.StateCreatingJobAvtobuslar
		nextPrompt = messages.MsgEnterAvtobuslar

	case models.StateCreatingJobAvtobuslar:
		// Allow skipping buses field
		if text == "Skip" || text == "skip" || text == "-" {
			job.Buses = ""
		} else {
			job.Buses = text
		}
		nextState = models.StateCreatingJobIshTavsifi
		nextPrompt = messages.MsgEnterIshTavsifi

	case models.StateCreatingJobIshTavsifi:
		job.AdditionalInfo = text
		nextState = models.StateCreatingJobIshKuni
		nextPrompt = messages.MsgEnterIshKuni

	case models.StateCreatingJobIshKuni:
		job.WorkDate = text
		nextState = models.StateCreatingJobKerakli
		nextPrompt = messages.MsgEnterKerakliIshchilar

	case models.StateCreatingJobKerakli:
		kerakli, err := strconv.Atoi(text)
		if err != nil || kerakli < 1 {
			return c.Send("❌ Iltimos, 1 dan katta raqam kiriting.")
		}
		job.RequiredWorkers = kerakli
		nextState = models.StateCreatingJobEmployerPhone
		nextPrompt = messages.MsgEnterEmployerPhone

	case models.StateCreatingJobEmployerPhone:
		job.EmployerPhone = text

		// Save job to database
		job.CreatedByAdminID = c.Sender().ID
		newJob, err := h.storage.Job().Create(ctx, job)
		if err != nil {
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
		adminMsg, err := c.Bot().Send(c.Sender(), msg, keyboards.JobDetailKeyboard(job), tele.ModeHTML)
		if err != nil {
			h.log.Error("Failed to send updated job detail", logger.Error(err))
			return c.Send(messages.MsgError)
		}

		// Save new admin message ID using new system
		adminMessage := &models.AdminJobMessage{
			JobID:     newJob.ID,
			AdminID:   c.Sender().ID,
			MessageID: int64(adminMsg.ID),
		}
		if err := h.storage.AdminMessage().Upsert(ctx, adminMessage); err != nil {
			h.log.Error("Failed to save admin message ID", logger.Error(err))
		}

		// Notify all other admins about the new job
		go h.notifyOtherAdminsNewJob(newJob, c.Sender().ID)

		return nil

	}

	// Update temp job and state
	h.setTempJob(c.Sender().ID, job)
	if err := h.storage.User().UpdateState(ctx, c.Sender().ID, nextState); err != nil {
		h.log.Error("Failed to update user state", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Use skip button for optional fields (location, buses)
	if nextState == models.StateCreatingJobLocation || nextState == models.StateCreatingJobAvtobuslar {
		return c.Send(nextPrompt, keyboards.CancelOrSkipKeyboard())
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
		job.Salary = text
	case models.StateEditingJobOvqat:
		job.Food = text
	case models.StateEditingJobVaqt:
		job.WorkTime = text
	case models.StateEditingJobManzil:
		job.Address = text
	case models.StateEditingJobLocation:
		job.Location = text
	case models.StateEditingJobXizmatHaqqi:
		xizmatHaqqi, err := strconv.Atoi(text)
		if err != nil {
			return c.Send("❌ Iltimos, raqam kiriting. Masalan: 9990")
		}
		job.ServiceFee = xizmatHaqqi
	case models.StateEditingJobAvtobuslar:
		// Allow skipping buses field
		if text == "Skip" || text == "skip" || text == "-" {
			job.Buses = ""
		} else {
			job.Buses = text
		}
	case models.StateEditingJobIshTavsifi:
		job.AdditionalInfo = text
	case models.StateEditingJobIshKuni:
		job.WorkDate = text
	case models.StateEditingJobKerakli:
		kerakli, err := strconv.Atoi(text)
		if err != nil || kerakli < 1 {
			return c.Send("❌ Iltimos, 1 dan katta raqam kiriting.")
		}
		job.RequiredWorkers = kerakli
	case models.StateEditingJobConfirmed:
		confirmed, err := strconv.Atoi(text)
		if err != nil || confirmed < 0 {
			return c.Send("❌ Iltimos, 0 yoki undan katta raqam kiriting.")
		}
		if confirmed > job.RequiredWorkers {
			return c.Send(fmt.Sprintf("❌ Qabul qilingan soni kerakli sondan (%d) oshmasligi kerak.", job.RequiredWorkers))
		}
		job.ConfirmedSlots = confirmed

		// Automatically update job status based on confirmed slots
		if job.ConfirmedSlots >= job.RequiredWorkers {
			job.Status = models.JobStatusFull
		} else if job.Status == models.JobStatusFull && job.ConfirmedSlots < job.RequiredWorkers {
			// If job was full but now has available slots, reopen it
			job.Status = models.JobStatusActive
		}
	case models.StateEditingJobEmployerPhone:
		job.EmployerPhone = text
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

	// Update ALL other admin messages (excluding current admin)
	go h.updateOtherAdminMessages(job.ID, c.Sender().ID)

	// Reset user state
	if err := h.storage.User().UpdateState(ctx, c.Sender().ID, models.StateIdle); err != nil {
		h.log.Error("Failed to update user state", logger.Error(err))
	}

	// Clear editing job ID
	h.clearEditingJobID(c.Sender().ID)

	// Delete the edit prompt message and user's text message to keep chat clean
	if c.Message() != nil {
		// Delete user's text input
		if err := c.Delete(); err != nil {
			h.log.Error("Failed to delete user message", logger.Error(err))
		}
	}

	// Single-message enforcement per admin: Delete this admin's previous message
	h.deleteAdminMessageForAdmin(job.ID, c.Sender().ID)

	// Send new admin message with updated info and success notification
	msg := fmt.Sprintf("✅ Yangilandi!\n\n%s", messages.FormatJobDetailAdmin(job))
	adminMsg, err := c.Bot().Send(c.Sender(), msg, keyboards.JobDetailKeyboard(job), tele.ModeHTML)
	if err != nil {
		h.log.Error("Failed to send updated job detail", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Save new admin message ID using new system
	adminMessage := &models.AdminJobMessage{
		JobID:     jobID,
		AdminID:   c.Sender().ID,
		MessageID: int64(adminMsg.ID),
	}
	if err := h.storage.AdminMessage().Upsert(ctx, adminMessage); err != nil {
		h.log.Error("Failed to save admin message ID", logger.Error(err))
	}

	return nil
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

	// If called from callback, delete the message or edit it to simple text
	if c.Callback() != nil {
		c.Delete()
	}
	return c.Send(messages.MsgAdminPanel, keyboards.AdminMenuReplyKeyboard())
}

// HandleSkipField handles skipping optional fields during job creation
func (h *Handler) HandleSkipField(c tele.Context) error {
	ctx := context.Background()
	user, err := h.storage.User().GetOrCreateUser(ctx, c.Sender().ID, c.Sender().Username, c.Sender().FirstName, c.Sender().LastName)
	if err != nil {
		h.log.Error("Failed to get user", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Handle skip for location field during job creation
	if user.State == models.StateCreatingJobLocation {
		return h.handleJobCreationLocationInput(c, user, "")
	}

	// Handle skip for location field during editing
	if user.State == models.StateEditingJobLocation {
		return h.handleJobEditingLocationInput(c, user, "")
	}

	// Handle skip for buses field during job creation
	if user.State == models.StateCreatingJobAvtobuslar {
		return h.handleJobCreationInput(c, user, "Skip")
	}

	// For editing, handle skip similarly
	if user.State == models.StateEditingJobAvtobuslar {
		return h.handleJobEditingInput(c, user, "Skip")
	}

	if err := c.Respond(&tele.CallbackResponse{Text: "❌ Bu maydon o'tkazib yuborilmaydi"}); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}

	return nil
}

// Helper to update channel message
func (h *Handler) updateChannelMessage(job *models.Job) {
	msg := &tele.Message{
		ID:   int(job.ChannelMessageID),
		Chat: &tele.Chat{ID: h.cfg.Bot.ChannelID},
	}

	channelMsg := messages.FormatJobForChannel(job)

	// Only show signup button if job is ACTIVE
	var keyboard *tele.ReplyMarkup
	if job.Status == models.JobStatusActive {
		keyboard = keyboards.JobSignupKeyboard(job.ID, h.cfg.Bot.Username)
	} else {
		// Remove buttons for non-active jobs (FULL, COMPLETED, CANCELLED, DRAFT)
		keyboard = &tele.ReplyMarkup{}
	}

	if _, err := h.bot.Edit(msg, channelMsg, keyboard, tele.ModeHTML); err != nil {
		h.log.Error("Failed to update channel message", logger.Error(err))
	}
}

// Helper to get job field value for display
func getJobFieldValue(job *models.Job, field string) string {
	switch field {
	case "ish_haqqi":
		return job.Salary
	case "ovqat":
		return job.Food
	case "vaqt":
		return job.WorkTime
	case "manzil":
		return job.Address
	case "location":
		return job.Location
	case "xizmat_haqqi":
		return fmt.Sprintf("%d", job.ServiceFee)
	case "avtobuslar":
		return job.Buses
	case "ish_tavsifi":
		return job.AdditionalInfo
	case "ish_kuni":
		return job.WorkDate
	case "kerakli":
		return fmt.Sprintf("%d", job.RequiredWorkers)
	case "confirmed":
		return fmt.Sprintf("%d", job.ConfirmedSlots)
	case "employer_phone":
		return job.EmployerPhone
	default:
		return ""
	}
}

// HandleViewJobBookings shows all users who booked a specific job
func (h *Handler) HandleViewJobBookings(c tele.Context, jobIDStr string) error {
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		h.log.Error("Invalid job ID in callback", logger.Error(err), logger.Any("job_id_str", jobIDStr))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Noto'g'ri ish ID"})
	}

	if !h.IsAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{Text: "❌ Sizda admin huquqi yo'q."})
	}

	ctx := context.Background()

	// Get job details
	job, err := h.storage.Job().GetByID(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get job", logger.Error(err))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Ish topilmadi."})
	}

	// Get all bookings for this job (confirmed and payment submitted)
	allBookings, err := h.storage.Booking().GetJobBookings(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get job bookings", logger.Error(err))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Xatolik yuz berdi."})
	}

	// Filter for active bookings (PaymentSubmitted and Confirmed)
	var activeBookings []*models.JobBooking
	for _, booking := range allBookings {
		if booking.Status == models.BookingStatusPaymentSubmitted || booking.Status == models.BookingStatusConfirmed {
			activeBookings = append(activeBookings, booking)
		}
	}

	if len(activeBookings) == 0 {
		return c.Respond(&tele.CallbackResponse{
			Text:      "📭 Bu ishga hech kim yozilmagan.",
			ShowAlert: true,
		})
	}

	// Build message with user details
	var sb strings.Builder
	fmt.Fprintf(&sb, "👥 <b>ISH №%d - YOZILGANLAR</b>\n\n", job.OrderNumber)
	fmt.Fprintf(&sb, "📅 Ish kuni: %s\n", job.WorkDate)
	fmt.Fprintf(&sb, "📊 Jami: %d ta ishchi\n\n", len(activeBookings))
	sb.WriteString("━━━━━━━━━━━━━━━━━━━\n\n")

	for i, booking := range activeBookings {
		// Get user's Telegram info
		user, err := h.storage.User().GetByID(ctx, booking.UserID)
		if err != nil {
			h.log.Error("Failed to get user", logger.Error(err))
			continue
		}

		// Get registered user info (full name and phone)
		registeredUser, err := h.storage.Registration().GetRegisteredUserByUserID(ctx, booking.UserID)
		if err != nil {
			h.log.Error("Failed to get registered user", logger.Error(err))
			continue
		}

		// Status icon
		statusIcon := "📩"
		statusText := "To'lov tekshirilmoqda"
		if booking.Status == models.BookingStatusConfirmed {
			statusIcon = "✅"
			statusText = "Tasdiqlangan"
		}

		fmt.Fprintf(&sb, "<b>%d. %s</b>\n", i+1, registeredUser.FullName)

		// Telegram username with link
		if user.Username != "" {
			fmt.Fprintf(&sb, "📱 Telegram: @%s\n", user.Username)
		} else {
			fmt.Fprintf(&sb, "📱 Telegram: <a href=\"tg://user?id=%d\">%s</a>\n", user.ID, user.FirstName)
		}

		fmt.Fprintf(&sb, "📞 Telefon: %s\n", registeredUser.Phone)
		fmt.Fprintf(&sb, "🎂 Yosh: %d\n", registeredUser.Age)
		fmt.Fprintf(&sb, "⚖️ Vazn/Bo'y: %d kg / %d cm\n", registeredUser.Weight, registeredUser.Height)
		fmt.Fprintf(&sb, "📊 Holat: %s %s\n", statusIcon, statusText)
		sb.WriteString("\n")
	}

	// Add back button
	menu := &tele.ReplyMarkup{}
	btnBack := menu.Data("⬅️ Orqaga", fmt.Sprintf("job_detail_%d", jobID))
	menu.Inline(menu.Row(btnBack))

	if err := c.Respond(); err != nil {
		h.log.Error("Failed to respond to callback", logger.Error(err))
	}

	return c.Edit(sb.String(), menu, tele.ModeHTML)
}

// Helper to delete admin message for a specific admin (single-message per admin enforcement)
func (h *Handler) deleteAdminMessageForAdmin(jobID, adminID int64) {
	ctx := context.Background()

	// Get the admin's message for this job
	adminMsg, err := h.storage.AdminMessage().Get(ctx, jobID, adminID)
	if err != nil {
		// No message exists, nothing to delete
		return
	}

	// Try to delete the message
	msgToDelete := &tele.Message{
		ID:   int(adminMsg.MessageID),
		Chat: &tele.Chat{ID: adminID},
	}

	if err := h.bot.Delete(msgToDelete); err != nil {
		// Message might already be deleted by user or not found
		// This is not critical - log and continue
		h.log.Error("Failed to delete admin message (might be already deleted)", logger.Error(err))
	}

	// Clear admin message from database
	if err := h.storage.AdminMessage().Delete(ctx, jobID, adminID); err != nil {
		h.log.Error("Failed to clear admin message", logger.Error(err))
	}
}

// Helper to update all admin messages for a job (broadcasts job updates)
func (h *Handler) updateAllAdminMessages(job *models.Job) {
	ctx := context.Background()

	// Get all admin messages for this job
	adminMessages, err := h.storage.AdminMessage().GetAllByJobID(ctx, job.ID)
	if err != nil {
		h.log.Error("Failed to get admin messages", logger.Error(err))
		return
	}

	// Update each admin's message
	for _, adminMsg := range adminMessages {
		msgToEdit := &tele.Message{
			ID:   int(adminMsg.MessageID),
			Chat: &tele.Chat{ID: adminMsg.AdminID},
		}

		msg := messages.FormatJobDetailAdmin(job)
		_, err := h.bot.Edit(msgToEdit, msg, keyboards.JobDetailKeyboard(job), tele.ModeHTML)
		if err != nil {
			h.log.Error("Failed to update admin message",
				logger.Error(err),
				logger.Any("admin_id", adminMsg.AdminID),
				logger.Any("job_id", job.ID))
			// If message not found, remove from database
			if err.Error() == "telegram: message not found (400)" ||
				err.Error() == "telegram: message to edit not found (400)" {
				h.storage.AdminMessage().Delete(ctx, job.ID, adminMsg.AdminID)
			}
		}
	}
}

// Helper to update other admin messages (excluding current admin)
func (h *Handler) updateOtherAdminMessages(jobID, currentAdminID int64) {
	ctx := context.Background()

	// Get the updated job
	job, err := h.storage.Job().GetByID(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get job for update", logger.Error(err))
		return
	}

	// Get all admin messages for this job
	adminMessages, err := h.storage.AdminMessage().GetAllByJobID(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get admin messages", logger.Error(err))
		return
	}

	// Update each admin's message (except current admin)
	for _, adminMsg := range adminMessages {
		if adminMsg.AdminID == currentAdminID {
			continue // Skip current admin, they already got their updated message
		}

		msgToEdit := &tele.Message{
			ID:   int(adminMsg.MessageID),
			Chat: &tele.Chat{ID: adminMsg.AdminID},
		}

		msg := messages.FormatJobDetailAdmin(job)
		_, err := h.bot.Edit(msgToEdit, msg, keyboards.JobDetailKeyboard(job), tele.ModeHTML)
		if err != nil {
			h.log.Error("Failed to update other admin message",
				logger.Error(err),
				logger.Any("admin_id", adminMsg.AdminID),
				logger.Any("job_id", jobID))
			// If message not found, remove from database
			if err.Error() == "telegram: message not found (400)" ||
				err.Error() == "telegram: message to edit not found (400)" {
				h.storage.AdminMessage().Delete(ctx, jobID, adminMsg.AdminID)
			}
		}
	}
}

// Helper to notify other admins about a new job
func (h *Handler) notifyOtherAdminsNewJob(job *models.Job, creatorAdminID int64) {
	ctx := context.Background()

	// Notify all other admins
	for _, adminID := range h.cfg.Bot.AdminIDs {
		if adminID == creatorAdminID {
			continue // Skip the admin who created the job
		}

		// Send job detail to other admin
		msg := fmt.Sprintf("🆕 Yangi ish yaratildi!\n\n%s", messages.FormatJobDetailAdmin(job))
		chat := &tele.Chat{ID: adminID}
		sentMsg, err := h.bot.Send(chat, msg, keyboards.JobDetailKeyboard(job), tele.ModeHTML)
		if err != nil {
			h.log.Error("Failed to notify other admin",
				logger.Error(err),
				logger.Any("admin_id", adminID),
				logger.Any("job_id", job.ID))
			continue
		}

		// Save admin message
		adminMessage := &models.AdminJobMessage{
			JobID:     job.ID,
			AdminID:   adminID,
			MessageID: int64(sentMsg.ID),
		}
		if err := h.storage.AdminMessage().Upsert(ctx, adminMessage); err != nil {
			h.log.Error("Failed to save admin message for other admin", logger.Error(err))
		}
	}
}

// Helper to delete all admin messages for a job (used when deleting job)
func (h *Handler) deleteAllAdminMessages(jobID int64) {
	ctx := context.Background()

	// Get all admin messages for this job
	adminMessages, err := h.storage.AdminMessage().GetAllByJobID(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get admin messages for deletion", logger.Error(err))
		return
	}

	// Delete each admin's Telegram message
	for _, adminMsg := range adminMessages {
		msgToDelete := &tele.Message{
			ID:   int(adminMsg.MessageID),
			Chat: &tele.Chat{ID: adminMsg.AdminID},
		}

		if err := h.bot.Delete(msgToDelete); err != nil {
			// Message might already be deleted by user or not found - not critical
			h.log.Error("Failed to delete admin message during job deletion",
				logger.Error(err),
				logger.Any("admin_id", adminMsg.AdminID),
				logger.Any("job_id", jobID))
		}
	}
}

// handleJobCreationLocationInput handles location input during job creation
func (h *Handler) handleJobCreationLocationInput(c tele.Context, user *models.User, locationStr string) error {
	ctx := context.Background()
	job := h.getTempJob(c.Sender().ID)
	if job == nil {
		job = &models.Job{Status: models.JobStatusDraft, RequiredWorkers: 1}
	}

	// Store location
	job.Location = locationStr

	// Update state to next step
	if err := h.storage.User().UpdateState(ctx, c.Sender().ID, models.StateCreatingJobXizmatHaqqi); err != nil {
		h.log.Error("Failed to update user state", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Update temp job
	h.setTempJob(c.Sender().ID, job)

	return c.Send(messages.MsgEnterXizmatHaqqi, keyboards.CancelKeyboard())
}

// handleJobEditingLocationInput handles location input during job editing
func (h *Handler) handleJobEditingLocationInput(c tele.Context, user *models.User, locationStr string) error {
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

	// Update location
	job.Location = locationStr

	// Update job in database
	if err := h.storage.Job().Update(ctx, job); err != nil {
		h.log.Error("Failed to update job", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Update channel message if exists
	if job.ChannelMessageID != 0 {
		h.updateChannelMessage(job)
	}

	// Update ALL other admin messages (excluding current admin)
	go h.updateOtherAdminMessages(job.ID, c.Sender().ID)

	// Reset user state
	if err := h.storage.User().UpdateState(ctx, c.Sender().ID, models.StateIdle); err != nil {
		h.log.Error("Failed to update user state", logger.Error(err))
	}

	// Clear temp job
	h.clearEditingJobID(c.Sender().ID)

	// Show updated job detail
	msg := fmt.Sprintf("✅ Yangilandi!\n\n%s", messages.FormatJobDetailAdmin(job))

	// Try to edit current admin's message
	_, err = h.bot.Edit(c.Message(), msg, keyboards.JobDetailKeyboard(job), tele.ModeHTML)
	if err != nil {
		// If edit fails, send new message
		adminMsg, err := c.Bot().Send(c.Sender(), msg, keyboards.JobDetailKeyboard(job), tele.ModeHTML)
		if err != nil {
			h.log.Error("Failed to send updated job detail", logger.Error(err))
			return c.Send(messages.MsgError)
		}

		// Save new admin message ID
		adminMessage := &models.AdminJobMessage{
			JobID:     job.ID,
			AdminID:   c.Sender().ID,
			MessageID: int64(adminMsg.ID),
		}
		if err := h.storage.AdminMessage().Upsert(ctx, adminMessage); err != nil {
			h.log.Error("Failed to save admin message ID", logger.Error(err))
		}
	}

	return nil
}

// HandleRegisteredUsersList shows list of all registered users with pagination (admin only)
func (h *Handler) HandleRegisteredUsersList(c tele.Context) error {
	return h.showUsersListPage(c, 1, false)
}

// HandleUsersListPage shows a specific page of registered users
func (h *Handler) HandleUsersListPage(c tele.Context, pageStr string) error {
	if pageStr == "current" {
		return c.Respond(&tele.CallbackResponse{})
	}

	page, err := strconv.Atoi(pageStr)
	if err != nil {
		h.log.Error("Invalid page in callback", logger.Error(err), logger.Any("page_str", pageStr))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Noto'g'ri sahifa"})
	}

	return h.showUsersListPage(c, page, true)
}

// showUsersListPage displays users list with pagination
func (h *Handler) showUsersListPage(c tele.Context, page int, isCallback bool) error {
	if !h.IsAdmin(c.Sender().ID) {
		if isCallback {
			return c.Respond(&tele.CallbackResponse{Text: "❌ Sizda admin huquqi yo'q."})
		}
		return c.Send("❌ Sizda admin huquqi yo'q.")
	}

	ctx := context.Background()

	// Get total count
	totalCount, err := h.storage.Registration().GetTotalRegisteredCount(ctx)
	if err != nil {
		h.log.Error("Failed to get total registered count", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	if totalCount == 0 {
		if isCallback {
			if err := c.Respond(); err != nil {
				h.log.Error("Failed to respond to callback", logger.Error(err))
			}
			return c.Edit("👥 Hozircha ro'yxatdan o'tgan foydalanuvchilar yo'q.")
		}
		return c.Send("👥 Hozircha ro'yxatdan o'tgan foydalanuvchilar yo'q.", keyboards.AdminMenuReplyKeyboard())
	}

	// Pagination settings
	const usersPerPage = 15
	totalPages := (totalCount + usersPerPage - 1) / usersPerPage

	// Validate page number
	if page < 1 {
		page = 1
	}
	if page > totalPages {
		page = totalPages
	}

	offset := (page - 1) * usersPerPage

	// Get paginated users
	users, err := h.storage.Registration().GetRegisteredUsersPaginated(ctx, usersPerPage, offset)
	if err != nil {
		h.log.Error("Failed to get registered users", logger.Error(err))
		if isCallback {
			if err := c.Respond(); err != nil {
				h.log.Error("Failed to respond to callback", logger.Error(err))
			}
		}
		return c.Send(messages.MsgError)
	}

	// Format user list
	var msg strings.Builder
	msg.WriteString("👥 <b>RO'YXATDAN O'TGANLAR</b>\n\n")
	msg.WriteString(fmt.Sprintf("📊 <b>Jami:</b> %d ta foydalanuvchi\n", totalCount))
	msg.WriteString(fmt.Sprintf("📄 <b>Sahifa:</b> %d/%d\n\n", page, totalPages))

	for i, user := range users {
		status := "🟢"
		if !user.IsActive {
			status = "🔴"
		}

		userIndex := offset + i + 1
		msg.WriteString(fmt.Sprintf("<b>%d. %s %s</b>\n", userIndex, status, user.FullName))
		msg.WriteString(fmt.Sprintf("   📞 %s\n", user.Phone))
		msg.WriteString(fmt.Sprintf("   👤 Yosh: %d | Vazn: %d kg | Bo'y: %d sm\n", user.Age, user.Weight, user.Height))
		msg.WriteString(fmt.Sprintf("   🆔 User ID: <code>%d</code>\n", user.UserID))
		msg.WriteString(fmt.Sprintf("   📅 %s\n\n", user.CreatedAt.Add(5*time.Hour).Format("02.01.2006 15:04")))
	}

	// Create pagination keyboard
	keyboard := keyboards.UsersPaginationKeyboard(page, totalPages)

	if isCallback {
		if err := c.Respond(); err != nil {
			h.log.Error("Failed to respond to callback", logger.Error(err))
		}
		return c.Edit(msg.String(), keyboard, tele.ModeHTML)
	}

	return c.Send(msg.String(), keyboard, tele.ModeHTML)
}

// SetConfig sets the config for admin handlers
func (h *Handler) SetConfig(cfg *config.Config) {
	h.cfg = cfg
}
