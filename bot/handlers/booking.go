package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/pkg/messages"

	tele "gopkg.in/telebot.v4"
)

// HandleJobBookingStart starts the job booking flow for a registered user
func (h *Handler) HandleJobBookingStart(c tele.Context, user *models.User, jobID int64) error {
	ctx := context.Background()

	// Get job details
	job, err := h.storage.Job().GetByID(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get job", logger.Error(err))
		return c.Send("❌ Ish topilmadi.")
	}

	// Check if job is still accepting bookings
	if job.Status != models.JobStatusActive {
		return c.Send("❌ Bu ish endi faol emas.")
	}

	// Check if job is full
	if job.IsFull() {
		// Check if there are reserved slots that might expire
		if job.ReservedSlots > 0 {
			msg := messages.FormatNoAvailableSlots(job)
			return c.Send(msg, tele.ModeHTML)
		}
		return c.Send("❌ Bu ishga barcha joylar band.")
	}

	// Show job details with booking confirmation
	msg := messages.FormatJobDetailUser(job)

	// Create confirmation keyboard
	menu := &tele.ReplyMarkup{}
	btnConfirm := menu.Data("✅ Ha, yozilaman", fmt.Sprintf("book_confirm_%d", jobID))
	btnCancel := menu.Data("❌ Yo'q, bekor qilish", "book_cancel")
	menu.Inline(
		menu.Row(btnConfirm),
		menu.Row(btnCancel),
	)

	return c.Send(msg, menu, tele.ModeHTML)
}

// HandleRegistrationStartWithJob starts registration flow while saving the target job ID
func (h *Handler) HandleRegistrationStartWithJob(c tele.Context, jobID int64) error {
	ctx := context.Background()

	// Get job to show what they're signing up for
	job, err := h.storage.Job().GetByID(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get job", logger.Error(err))
		return c.Send("❌ Ish topilmadi.")
	}

	msg := fmt.Sprintf(`
👋 Salom!

Siz <b>№%d</b> raqamli ishga yozilmoqchisiz.

Avval ro'yxatdan o'tishingiz kerak. Ro'yxatdan o'tish bir necha daqiqani oladi.

Ro'yxatdan o'tgandan so'ng, ishga yozilish jarayonini davom ettirishingiz mumkin bo'ladi.

<b>Ish haqida qisqacha:</b>
💰 %s
📅 %s
📍 %s

Davom etamizmi?
`,
		job.OrderNumber,
		job.Salary,
		job.WorkDate,
		job.Address,
	)

	menu := &tele.ReplyMarkup{}
	btnStart := menu.Data("✅ Ro'yxatdan o'tish", fmt.Sprintf("start_reg_job_%d", jobID))
	btnCancel := menu.Data("❌ Bekor qilish", "book_cancel")
	menu.Inline(
		menu.Row(btnStart),
		menu.Row(btnCancel),
	)

	return c.Send(msg, menu, tele.ModeHTML)
}

// HandleStartRegistrationForJob starts the registration process and saves the job ID
func (h *Handler) HandleStartRegistrationForJob(c tele.Context, jobIDStr string) error {
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		h.log.Error("Invalid job ID in callback", logger.Error(err), logger.Any("job_id_str", jobIDStr))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Noto'g'ri ish ID"})
	}

	ctx := context.Background()
	userID := c.Sender().ID

	if err := c.Respond(); err != nil {
		if strings.Contains(err.Error(), "query is too old") {
			h.log.Warn("Stale callback query (user clicked during downtime)", logger.Any("user_id", userID))
		} else {
			h.log.Error("Failed to respond to callback", logger.Error(err))
		}
	}

	// Get or create draft
	draft, err := h.services.Registration().GetOrCreateDraft(ctx, userID)
	if err != nil {
		h.log.Error("Failed to get draft", logger.Error(err))
		return c.Send("❌ Xatolik yuz berdi.")
	}

	// Save the job ID to redirect after registration
	draft.PendingJobID = &jobID
	if err := h.storage.Registration().UpdateDraft(ctx, draft); err != nil {
		h.log.Error("Failed to save pending job ID", logger.Error(err))
		// Continue anyway - not critical
	}

	h.log.Info("Saved pending job ID for post-registration redirect",
		logger.Any("user_id", userID),
		logger.Any("job_id", jobID),
	)

	return h.HandleRegistrationStart(c)
}

// HandleBookingConfirm handles the booking confirmation with atomic slot reservation
func (h *Handler) HandleBookingConfirm(c tele.Context, jobIDStr string) error {
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		h.log.Error("Invalid job ID in callback", logger.Error(err), logger.Any("job_id_str", jobIDStr))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Noto'g'ri ish ID"})
	}

	ctx := context.Background()
	userID := c.Sender().ID

	if err := c.Respond(); err != nil {
		if strings.Contains(err.Error(), "query is too old") {
			h.log.Warn("Stale callback query (user clicked during downtime)", logger.Any("user_id", userID))
		} else {
			h.log.Error("Failed to respond to callback", logger.Error(err))
		}
	}

	// Check idempotency through service
	existingBooking, _ := h.services.Booking().CheckIdempotency(ctx, userID, jobID)
	if existingBooking != nil {
		if existingBooking.Status == models.BookingStatusSlotReserved && !existingBooking.IsExpired() {
			// User already has a reservation, show remaining time
			remaining := existingBooking.TimeRemaining()
			minutes := int(remaining.Minutes())
			seconds := int(remaining.Seconds()) % 60
			return c.Edit(fmt.Sprintf("⚠️ Siz allaqachon bu ishga yozilgansiz!\n\nQolgan vaqt: %d daqiqa %d soniya", minutes, seconds))
		}
		if existingBooking.Status == models.BookingStatusPaymentSubmitted {
			return c.Edit("⚠️ Sizning to'lovingiz ko'rib chiqilmoqda. Iltimos, admin javobini kuting.")
		}
		if existingBooking.Status == models.BookingStatusConfirmed {
			return c.Edit("✅ Siz allaqachon tasdiqlangansiz!")
		}
	}

	// Get job details for payment info
	job, err := h.storage.Job().GetByID(ctx, jobID)
	if err != nil {
		h.log.Error("Failed to get job", logger.Error(err))
		return c.Edit("❌ Xatolik yuz berdi.")
	}

	// Confirm booking through service (handles all business logic)
	booking, err := h.services.Booking().ConfirmBooking(ctx, userID, jobID)
	if err != nil {
		h.log.Error("Failed to confirm booking", logger.Error(err), logger.Any("error_msg", err.Error()))

		// Handle known errors
		errStr := err.Error()

		// 1. Blocked user errors
		if strings.Contains(errStr, "Siz doimiy bloklangansiz") || strings.Contains(errStr, "Siz vaqtincha bloklangansiz") {
			return c.Edit(errStr, tele.ModeHTML)
		}

		// 2. Job status errors
		if errStr == "job is not active" {
			return c.Edit("❌ Bu ish endi faol emas.")
		}
		if errStr == "all slots are full" {
			return c.Edit("❌ Kechirasiz, barcha joylar band bo'lib qoldi! 😔")
		}
		if errStr == "all slots reserved, try again in a few minutes" {
			msg := messages.FormatNoAvailableSlots(job)
			return c.Edit(msg, tele.ModeHTML)
		}

		// 3. User constraint errors
		if strings.Contains(errStr, "you have another active booking") {
			return c.Edit("⚠️ Sizda allaqachon boshqa faol bandlovingiz bor. Iltimos, avval uni yakunlang yoki bekor qiling.")
		}
		if strings.Contains(errStr, "payment is being reviewed") || strings.Contains(errStr, "you have a payment under review") {
			return c.Edit("⚠️ Sizning boshqa ish uchun to'lovingiz ko'rib chiqilmoqda. Iltimos, admin javobini kuting.")
		}
		if errStr == "booking already confirmed" {
			return c.Edit("✅ Siz allaqachon tasdiqlangansiz!")
		}

		return c.Edit("❌ Xatolik yuz berdi. Iltimos, qaytadan urinib ko'ring.")
	}

	// Success! Send payment instructions
	msg := messages.FormatPaymentInstructions(job, h.cfg.Payment.CardNumber, h.cfg.Payment.CardHolderName)

	// Edit the message
	if err := c.Edit(msg, tele.ModeHTML); err != nil {
		h.log.Error("Failed to edit message", logger.Error(err))
		return c.Send(msg, tele.ModeHTML)
	}

	// Store the callback message ID in the booking for later deletion/editing
	if c.Callback() != nil && c.Callback().Message != nil {
		messageID := int64(c.Callback().Message.ID)
		// Update booking with message ID in a separate transaction (non-critical)
		go func() {
			updateCtx := context.Background()
			tx, err := h.storage.Transaction().Begin(updateCtx)
			if err != nil {
				return
			}
			// Always rollback on exit — Rollback after Commit is a harmless no-op in pgx.
			defer h.storage.Transaction().Rollback(updateCtx, tx)

			booking.PaymentInstructionMsgID = messageID
			if err := h.storage.Booking().Update(updateCtx, tx, booking); err != nil {
				return
			}
			h.storage.Transaction().Commit(updateCtx, tx)
		}()
	}

	return nil
}

// HandleCancelBooking handles admin cancellation of a confirmed booking (payment return)
func (h *Handler) HandleCancelBooking(c tele.Context, params string) error {
	ctx := context.Background()

	// Check if user is admin
	if !h.IsAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ Sizda bu amalga ruxsat yo'q.",
			ShowAlert: true,
		})
	}

	// Callback data format: cancel_booking_<bookingID>
	bookingID, err := strconv.ParseInt(params, 10, 64)
	if err != nil {
		h.log.Error("Failed to parse booking ID", logger.Error(err), logger.Any("params", params))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Noto'g'ri booking ID.", ShowAlert: true})
	}

	// Cancel booking through service
	booking, job, err := h.services.Booking().CancelBookingByAdmin(ctx, bookingID, c.Sender().ID)
	if err != nil {
		h.log.Error("Failed to cancel booking", logger.Error(err))

		errMsg := "❌ Xatolik yuz berdi."
		if strings.HasPrefix(err.Error(), "only confirmed bookings") {
			errMsg = "⚠️ Faqat tasdiqlangan bronlar bekor qilinishi mumkin."
		} else if strings.HasPrefix(err.Error(), "booking not found") {
			errMsg = "❌ Bron topilmadi."
		}

		return c.Respond(&tele.CallbackResponse{
			Text:      errMsg,
			ShowAlert: true,
		})
	}

	// Notify user about cancellation with payment return instructions
	go h.notifyUserBookingCancelledByAdmin(booking, job)

	if booking.AdminGroupMessageID != 0 {

		adminUsername := c.Sender().Username
		if adminUsername == "" {
			adminUsername = c.Sender().FirstName
		}
		// Get user info to rebuild full message if needed
		telegramUser, err := h.storage.User().GetByID(ctx, booking.UserID)
		if err != nil {
			h.log.Error("Failed to get telegram user", logger.Error(err))
			return c.Respond(&tele.CallbackResponse{
				Text:      "❌ Xatolik yuz berdi.",
				ShowAlert: true,
			})
		}

		registeredUser, err := h.storage.Registration().GetRegisteredUserByUserID(ctx, booking.UserID)
		if err != nil {
			h.log.Error("Failed to get registered user", logger.Error(err))
			return c.Respond(&tele.CallbackResponse{
				Text:      "❌ Xatolik yuz berdi.",
				ShowAlert: true,
			})
		}

		updatedCaption := messages.FormatBookingCancelledAdminMessage(registeredUser, telegramUser, job, booking, adminUsername)

		// Admin cancelled from admin job detail panel - update the saved admin group message
		adminGroupMsg := &tele.Message{
			ID:   int(booking.AdminGroupMessageID),
			Chat: &tele.Chat{ID: h.cfg.Bot.AdminGroupID},
		}

		err = h.services.Sender().EditCaption(adminGroupMsg, updatedCaption, &tele.ReplyMarkup{}, tele.ModeHTML)
		if err != nil {
			// Silently handle "message not found" - user may have deleted it
			if !strings.Contains(err.Error(), "message not found") {
				h.log.Error("Failed to update admin group message", logger.Error(err), logger.Any("message_id", booking.AdminGroupMessageID))
			}
		}
	}

	// Update ALL admin job-detail messages to reflect new slot count
	if job != nil {
		if job.ConfirmedSlots == 0 {
			h.updateAllAdminMessages(job)
		} else {
			h.updateOtherAdminMessages(job.ID, c.Sender().ID)
		}

		h.updateChannelMessage(job)
	}

	h.log.Info("Booking cancelled by admin via handler",
		logger.Any("booking_id", bookingID),
		logger.Any("admin_id", c.Sender().ID),
	)

	err = c.Respond(&tele.CallbackResponse{
		Text: "↩️ Bron bekor qilindi, foydalanuvchiga xabar yuborildi.",
	})

	// If the admin cancelled from "Yozilganlarni ko'rish" text menu, refresh it
	if c.Message() != nil && c.Message().Photo == nil && job != nil {
		return h.HandleViewJobBookings(c, strconv.FormatInt(job.ID, 10))
	}

	return err
}

// notifyUserBookingCancelledByAdmin sends a notification to the user that their confirmed booking
// has been cancelled by the admin and their payment will be returned.
func (h *Handler) notifyUserBookingCancelledByAdmin(booking *models.JobBooking, job *models.Job) {
	ctx := context.Background()

	var jobInfo string
	if job != nil {
		jobInfo = fmt.Sprintf("\n\n💼 <b>Ish ma'lumotlari:</b>\n📋 Tartib raqami: #%d\n📅 Ish kuni: %s\n💰 Ish haqqi: %s\n📍 Manzil: %s",
			job.OrderNumber,
			job.WorkDate,
			job.Salary,
			job.Address,
		)
	}

	message := fmt.Sprintf(`↩️ <b>BRONINGIZ BEKOR QILINDI</b>

Hurmatli foydalanuvchi, sizning broningiz admin tomonidan bekor qilindi.%s

💰 <b>TO'LOV HAQIDA:</b>
Siz to'lagan xizmat haqqi qaytariladi. Iltimos, to'lovni qaytarish uchun admin bilan bog'laning.

❓ Savollaringiz bo'lsa, admin bilan bog'laning.`, jobInfo)

	if err := h.services.Sender().Send(ctx, booking.UserID, message, tele.ModeHTML); err != nil {
		h.log.Error("Failed to notify user about booking cancellation", logger.Error(err))
	}
}
