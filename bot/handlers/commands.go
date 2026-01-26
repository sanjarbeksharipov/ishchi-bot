package handlers

import (
	"context"
	"strconv"
	"strings"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/pkg/keyboards"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/pkg/messages"

	tele "gopkg.in/telebot.v4"
)

// HandleStart handles the /start command
func (h *Handler) HandleStart(c tele.Context) error {
	ctx := context.Background()
	user := c.Sender()

	// Get or create user in storage
	dbUser, err := h.storage.User().GetOrCreateUser(ctx, user.ID, user.Username, user.FirstName, user.LastName)
	if err != nil {
		h.log.Error("Failed to get/create user", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Check for deep link parameter (e.g., /start job_123)
	payload := c.Message().Payload
	if payload != "" && strings.HasPrefix(payload, "job_") {
		jobIDStr := strings.TrimPrefix(payload, "job_")
		jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
		if err == nil {
			// Check if user is registered by looking in registered_users table
			registeredUser, err := h.storage.Registration().GetRegisteredUserByUserID(ctx, user.ID)
			if err == nil && registeredUser != nil {
				// User is registered, start booking flow
				return h.HandleJobBookingStart(c, dbUser, jobID)
			}
			// User not registered yet, save job ID and start registration
			return h.HandleRegistrationStartWithJob(c, jobID)
		}
	}

	// Check if this is an admin
	if h.IsAdmin(user.ID) {
		return c.Send(messages.MsgWelcome, keyboards.MainMenuKeyboard())
	}

	// For regular users, start/continue registration flow
	return h.HandleRegistrationStart(c)
}

// HandleHelp handles the /help command
func (h *Handler) HandleHelp(c tele.Context) error {
	return c.Send(messages.MsgHelp, keyboards.BackKeyboard())
}

// HandleAbout handles the /about command
func (h *Handler) HandleAbout(c tele.Context) error {
	return c.Send(messages.MsgAbout, keyboards.BackKeyboard())
}

// HandleSettings handles the /settings command
func (h *Handler) HandleSettings(c tele.Context) error {
	return c.Send(messages.MsgSettings, keyboards.BackKeyboard())
}

// HandleText handles regular text messages
func (h *Handler) HandleText(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()
	text := strings.TrimSpace(c.Text())

	// Get or create user
	user, err := h.storage.User().GetOrCreateUser(ctx, sender.ID, sender.Username, sender.FirstName, sender.LastName)
	if err != nil {
		h.log.Error("Failed to get/create user", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Handle cancel button from reply keyboard
	if text == "❌ Bekor qilish" {
		return h.HandleCancelRegistration(c)
	}

	// Check if user is in registration flow
	if h.IsInRegistrationFlow(user.State) {
		regState := h.GetRegistrationState(user.State)
		return h.HandleRegistrationTextInput(c, regState)
	}

	// Check if user is in job creation/editing flow (admin only)
	if h.IsAdmin(sender.ID) && (strings.HasPrefix(string(user.State), "creating_job_") || strings.HasPrefix(string(user.State), "editing_job_")) {
		return h.HandleAdminTextInput(c, user)
	}

	// Default: check user state
	switch user.State {
	case models.StateIdle:
		// Echo the message back with a prefix
		response := "You said: " + c.Text()
		return c.Send(response)
	default:
		response := "You said: " + c.Text()
		return c.Send(response)
	}
}

// HandleContact handles contact sharing messages
func (h *Handler) HandleContact(c tele.Context) error {
	ctx := context.Background()
	sender := c.Sender()

	// Get user
	user, err := h.storage.User().GetByID(ctx, sender.ID)
	if err != nil {
		h.log.Error("Failed to get user", logger.Error(err))
		return c.Send(messages.MsgError)
	}

	// Check if user is in registration phone state
	if user.State == models.UserState(models.RegStatePhone) {
		return h.HandleRegistrationContact(c)
	}

	return nil
}

// HandlePhoto handles photo messages
func (h *Handler) HandlePhoto(c tele.Context) error {
	photo := c.Message().Photo
	if photo == nil {
		return nil
	}

	return h.HandlePaymentReceiptSubmission(c, photo.FileID)
}

// HandlePaymentReceiptSubmission handles payment receipt photo submission
func (h *Handler) HandlePaymentReceiptSubmission(c tele.Context, photoFileID string) error {
	ctx := context.Background()
	user := c.Sender()

	// Check if user has registered
	_, err := h.storage.Registration().GetRegisteredUserByUserID(ctx, user.ID)
	if err != nil {
		return c.Send("❌ Iltimos, avval ro'yxatdan o'ting: /start")
	}

	// Submit payment through service
	booking, err := h.services.Payment().SubmitPayment(ctx, user.ID, photoFileID, int64(c.Message().ID))
	if err != nil {
		h.log.Error("Failed to submit payment", logger.Error(err))

		if err.Error() == "no pending booking found" {
			return c.Send(`❌ Sizda to'lov kutilayotgan booking topilmadi.

Iltimos, avval ish uchun joy band qiling, keyin to'lov chekini yuboring.`)
		}
		if err.Error() == "booking has expired" {
			return c.Send(`⏰ Vaqt tugadi!

Afsuski, sizning booking vaqti tugagan. Iltimos, qaytadan joy band qiling.`)
		}

		return c.Send("❌ Xatolik yuz berdi. Iltimos, qaytadan urinib ko'ring.")
	}

	// Send confirmation to user
	msg := `✅ <b>TO'LOV CHEKI QABUL QILINDI!</b>

📸 Sizning to'lov chekingiz muvaffaqiyatli qabul qilindi.

⏰ Admin 10-15 daqiqa ichida tekshiradi va javob beradi.

💡 Agar to'lov tasdiqlansa, sizga xabar yuboriladi va booking tasdiqlangan hisoblanadi.

Sabr qilganingiz uchun rahmat! 🙏`

	if err := c.Send(msg, tele.ModeHTML); err != nil {
		h.log.Error("Failed to send confirmation", logger.Error(err))
	}

	// Forward to admin group
	go h.ForwardPaymentToAdminGroup(ctx, booking, photoFileID)

	return nil
}
