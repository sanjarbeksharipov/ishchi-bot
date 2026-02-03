package handlers

import (
	"context"
	"fmt"
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
		return c.Send(messages.MsgAdminPanel, keyboards.AdminMenuReplyKeyboard())
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
	isCreatingJob := strings.HasPrefix(string(user.State), "creating_job_")
	isEditingJob := strings.HasPrefix(string(user.State), "editing_job_")

	if h.IsAdmin(sender.ID) && (isCreatingJob || isEditingJob) {
		return h.HandleAdminTextInput(c, user)
	}

	// Handle admin menu reply buttons
	if h.IsAdmin(sender.ID) {
		switch text {
		case "➕ Ish yaratish":
			return h.HandleCreateJob(c)
		case "📋 Ishlar ro'yxati":
			return h.HandleJobList(c)
		}
	}

	// Handle user menu reply buttons
	switch text {
	case "👤 Profil":
		return h.HandleUserProfile(c)
	case "📋 Mening ishlarim":
		return h.HandleUserMyJobs(c)
	case "❓ Yordam":
		// Check if we have a specific help message for users, otherwise generic
		return h.HandleHelp(c)
	}

	// Default: check user state
	switch user.State {
	case models.StateIdle:
		// Echo the message back with a prefix
		// Only echo if not an admin command or handled above
		if !h.IsAdmin(sender.ID) {
			// Don't echo, just ignore or send help hint?
			// For now, let's just ignore to avoid spamming
			return nil
		}
		return nil
	default:
		// If state is not handled, do nothing
		return nil
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

💡 Agar to'lov tasdiqlansa, sizga xabar yuboriladi.

Sabr qilganingiz uchun rahmat! 🙏`

	if err := c.Send(msg, tele.ModeHTML); err != nil {
		h.log.Error("Failed to send confirmation", logger.Error(err))
	}

	// Forward to admin group
	go h.ForwardPaymentToAdminGroup(ctx, booking, photoFileID)

	return nil
}

// HandleUserProfile displays the user's profile
func (h *Handler) HandleUserProfile(c tele.Context) error {
	ctx := context.Background()
	userID := c.Sender().ID

	// Get registered user details
	regUser, err := h.storage.Registration().GetRegisteredUserByUserID(ctx, userID)
	if err != nil {
		return c.Send("❌ Siz hali ro'yxatdan o'tmagansiz. /start buyrug'ini bosing.")
	}

	msg := fmt.Sprintf(`👤 <b>SIZNING PROFILINGIZ</b>

📝 <b>F.I.SH:</b> %s
📱 <b>Telefon:</b> %s
🎂 <b>Yosh:</b> %d
⚖️ <b>Vazn:</b> %d kg
📏 <b>Bo'y:</b> %d sm

✅ <b>Holat:</b> Faol
📅 <b>Ro'yxatdan o'tgan sana:</b> %s
`,
		regUser.FullName,
		regUser.Phone,
		regUser.Age,
		regUser.Weight,
		regUser.Height,
		regUser.CreatedAt.Format("02.01.2006"),
	)

	return c.Send(msg, tele.ModeHTML)
}

// HandleUserMyJobs displays the user's bookings
func (h *Handler) HandleUserMyJobs(c tele.Context) error {
	ctx := context.Background()
	userID := c.Sender().ID

	// Get user's bookings
	// We want active bookings: Reserved, PaymentSubmitted, Confirmed
	statuses := []models.BookingStatus{
		models.BookingStatusSlotReserved,
		models.BookingStatusPaymentSubmitted,
		models.BookingStatusConfirmed,
	}

	var activeBookings []*models.JobBooking
	for _, status := range statuses {
		bookings, err := h.storage.Booking().GetUserBookingsByStatus(ctx, userID, status)
		if err == nil {
			activeBookings = append(activeBookings, bookings...)
		}
	}

	if len(activeBookings) == 0 {
		return c.Send("📭 Sizda hozircha faol ishlar yo'q.")
	}

	var sb strings.Builder
	sb.WriteString("📋 <b>SIZNING ISHLARINGIZ</b>\n\n")

	for _, booking := range activeBookings {
		job, err := h.storage.Job().GetByID(ctx, booking.JobID)
		if err != nil {
			continue
		}

		statusIcon := "❓"
		statusText := string(booking.Status)

		switch booking.Status {
		case models.BookingStatusSlotReserved:
			statusIcon = "⏳"
			statusText = "To'lov kutilmoqda"
		case models.BookingStatusPaymentSubmitted:
			statusIcon = "📩"
			statusText = "Tekshirilmoqda"
		case models.BookingStatusConfirmed:
			statusIcon = "✅"
			statusText = "Tasdiqlangan"
		}

		fmt.Fprintf(&sb, "<b>━━━━━ ISH №%d ━━━━━</b>\n", job.OrderNumber)
		fmt.Fprintf(&sb, "📊 Holat: %s %s\n\n", statusIcon, statusText)
		fmt.Fprintf(&sb, "📅 Ish kuni: %s\n", job.WorkDate)
		fmt.Fprintf(&sb, "💰 Ish haqqi: %s\n", job.Salary)
		fmt.Fprintf(&sb, "⏰ Ish vaqti: %s\n", job.WorkTime)
		fmt.Fprintf(&sb, "📍 Manzil: %s\n", job.Address)

		if job.Food != "" {
			fmt.Fprintf(&sb, "🍛 Ovqat: %s\n", job.Food)
		} else {
			sb.WriteString("🍛 Ovqat: Berilmaydi\n")
		}

		if job.Buses != "" {
			fmt.Fprintf(&sb, "🚌 Avtobuslar: %s\n", job.Buses)
		}

		fmt.Fprintf(&sb, "💳 Xizmat haqi: %d so'm\n", job.ServiceFee)

		if job.AdditionalInfo != "" {
			fmt.Fprintf(&sb, "📝 Qo'shimcha: %s\n", job.AdditionalInfo)
		}

		sb.WriteString("\n")
	}

	return c.Send(sb.String(), tele.ModeHTML)
}
