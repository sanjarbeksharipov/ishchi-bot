package handlers

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/pkg/logger"

	tele "gopkg.in/telebot.v4"
)

// HandlePaymentSubmission handles when user sends payment receipt photo
func (h *Handler) HandlePaymentSubmission(c tele.Context) error {
	// Get photo
	photo := c.Message().Photo
	if photo == nil {
		return c.Send("❌ Iltimos, to'lov chekini rasm sifatida yuboring.")
	}

	// Check if user has a pending booking (SLOT_RESERVED status)
	// We need to find their most recent reserved booking
	// For now, we'll need user to specify which booking or store it in session
	// Let's check if there's a caption with booking ID or find latest reserved booking

	// TODO: Implement logic to find user's pending booking
	// For now, let's assume we can find it by user_id and status

	return c.Send("📸 To'lov cheki qabul qilindi. Iltimos, qaysi ish uchun to'lov qilganingizni belgilang.")
}

// ForwardPaymentToAdminGroup forwards payment receipt to admin group with approval buttons
func (h *Handler) ForwardPaymentToAdminGroup(ctx context.Context, booking *models.JobBooking, receiptFileID string) error {
	// Get job details
	job, err := h.storage.Job().GetByID(ctx, booking.JobID)
	if err != nil {
		h.log.Error("Failed to get job for admin notification", logger.Error(err))
		return err
	}

	// Get user details from registered_users
	registeredUser, err := h.storage.Registration().GetRegisteredUserByUserID(ctx, booking.UserID)
	if err != nil {
		h.log.Error("Failed to get registered user", logger.Error(err))
		return err
	}

	// Get telegram user info
	telegramUser, err := h.storage.User().GetByID(ctx, booking.UserID)
	if err != nil {
		h.log.Error("Failed to get telegram user", logger.Error(err))
		return err
	}

	// Format message for admin group
	message := fmt.Sprintf(`🆕 <b>YANGI TO'LOV CHEKI</b>

👤 <b>Foydalanuvchi:</b>
• Ism: %s
• Telefon: %s
• Telegram: @%s (ID: <code>%d</code>)

💼 <b>Ish ma'lumotlari:</b>
• Nomi: %s
• Tartib raqami: #%d
• Xizmat haqqi: %d so'm
• Ish haqqi: %s

📋 <b>Booking ID:</b> #%d
⏰ <b>Yuborilgan vaqt:</b> %s

👇 <b>To'lov cheki:</b>`,
		registeredUser.FullName,
		registeredUser.Phone,
		telegramUser.Username,
		booking.UserID,
		job.Salary,
		job.ID,
		job.ServiceFee,
		job.Salary,
		booking.ID,
		time.Now().Format("02.01.2006 15:04"),
	)

	// Create photo message
	photo := &tele.Photo{
		File: tele.File{
			FileID: receiptFileID,
		},
		Caption: message,
	}

	// Create inline keyboard with approval buttons
	keyboard := &tele.ReplyMarkup{}
	keyboard.Inline(
		keyboard.Row(
			keyboard.Data("✅ Tasdiqlash", fmt.Sprintf("approve_payment_%d", booking.ID)),
			keyboard.Data("❌ Rad etish", fmt.Sprintf("reject_payment_%d", booking.ID)),
		),
		keyboard.Row(
			keyboard.Data("🚫 Foydalanuvchini bloklash", fmt.Sprintf("block_user_%d_%d", booking.UserID, booking.ID)),
		),
	)

	// Send to admin group
	_, err = h.bot.Send(
		&tele.Chat{ID: h.cfg.Bot.AdminGroupID},
		photo,
		keyboard,
		tele.ModeHTML,
	)

	if err != nil {
		h.log.Error("Failed to send payment to admin group", logger.Error(err))
		return fmt.Errorf("failed to send to admin group: %w", err)
	}

	h.log.Info("Payment receipt forwarded to admin group",
		logger.Any("booking_id", booking.ID),
		logger.Any("user_id", booking.UserID),
	)

	return nil
}

// HandleApprovePayment handles admin approval of payment
func (h *Handler) HandleApprovePayment(c tele.Context) error {
	ctx := context.Background()

	// Check if user is admin
	if !h.isAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ Sizda bu amalga ruxsat yo'q.",
			ShowAlert: true,
		})
	}

	// Get booking ID from callback data
	callbackData := strings.TrimSpace(c.Callback().Data)
	callbackDataSl := strings.Split(callbackData, "_")
	if len(callbackDataSl) != 3 {
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ Noto'g'ri booking ID.",
			ShowAlert: true,
		})
	}

	bookingIDStr := callbackDataSl[2]
	bookingID, err := strconv.ParseInt(bookingIDStr, 10, 64)
	if err != nil {
		h.log.Error("Failed to parse booking ID", logger.Error(err), logger.Any("callback_data", c.Callback().Data))
		return c.Respond(&tele.CallbackResponse{Text: "❌ Noto'g'ri booking ID.", ShowAlert: true})
	}

	// Approve payment through service
	booking, err := h.services.Payment().ApprovePayment(ctx, bookingID, c.Sender().ID)
	if err != nil {
		h.log.Error("Failed to approve payment", logger.Error(err))

		if err.Error() == "booking not found" {
			return c.Respond(&tele.CallbackResponse{
				Text:      "❌ Booking topilmadi.",
				ShowAlert: true,
			})
		}
		if strings.HasPrefix(err.Error(), "payment already processed") {
			return c.Respond(&tele.CallbackResponse{
				Text:      fmt.Sprintf("⚠️ Bu to'lov allaqachon qayta ishlangan: %s", booking.Status),
				ShowAlert: true,
			})
		}

		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ Xatolik yuz berdi.",
			ShowAlert: true,
		})
	}

	// Notify user
	go h.notifyUserPaymentApproved(booking)

	// Update admin group message
	adminUsername := c.Sender().Username
	if adminUsername == "" {
		adminUsername = c.Sender().FirstName
	}

	updatedCaption := c.Message().Caption + fmt.Sprintf("\n\n✅ <b>TASDIQLANDI</b>\n👤 Admin: @%s\n⏰ Vaqt: %s",
		adminUsername,
		time.Now().Format("02.01.2006 15:04"),
	)

	// Edit photo caption and remove keyboard
	_, err = h.bot.EditCaption(c.Message(), updatedCaption, &tele.ReplyMarkup{}, tele.ModeHTML)
	if err != nil {
		h.log.Error("Failed to edit admin message caption", logger.Error(err))
	}

	return c.Respond(&tele.CallbackResponse{
		Text: "✅ To'lov tasdiqlandi!",
	})
}

// HandleRejectPayment handles admin rejection of payment
func (h *Handler) HandleRejectPayment(c tele.Context) error {
	ctx := context.Background()

	// Check if user is admin
	if !h.isAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ Sizda bu amalga ruxsat yo'q.",
			ShowAlert: true,
		})
	}

	// Get booking ID from callback data
	callbackData := strings.TrimSpace(c.Callback().Data)
	callbackDataSl := strings.Split(callbackData, "_")
	if len(callbackDataSl) != 3 {
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ Noto'g'ri booking ID.",
			ShowAlert: true,
		})
	}
	bookingIDStr := callbackDataSl[2]
	bookingID, err := strconv.ParseInt(bookingIDStr, 10, 64)
	if err != nil {
		h.log.Error("Failed to parse booking ID", logger.Error(err), logger.Any("callback_data", c.Callback().Data))
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ Noto'g'ri booking ID.",
			ShowAlert: true,
		})
	}

	// Reject payment through service
	reason := "To'lov cheki noto'g'ri yoki aniq emas"
	booking, err := h.services.Payment().RejectPayment(ctx, bookingID, c.Sender().ID, reason)
	if err != nil {
		h.log.Error("Failed to reject payment", logger.Error(err))

		if err.Error() == "booking not found" {
			return c.Respond(&tele.CallbackResponse{
				Text:      "❌ Booking topilmadi.",
				ShowAlert: true,
			})
		}
		if strings.HasPrefix(err.Error(), "payment already processed") {
			return c.Respond(&tele.CallbackResponse{
				Text:      fmt.Sprintf("⚠️ Bu to'lov allaqachon qayta ishlangan: %s", booking.Status),
				ShowAlert: true,
			})
		}

		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ Xatolik yuz berdi.",
			ShowAlert: true,
		})
	}

	// Notify user
	go h.notifyUserPaymentRejected(booking)

	// Update admin group message
	adminUsername := c.Sender().Username
	if adminUsername == "" {
		adminUsername = c.Sender().FirstName
	}

	updatedCaption := c.Message().Caption + fmt.Sprintf("\n\n❌ <b>RAD ETILDI</b>\n👤 Admin: @%s\n⏰ Vaqt: %s\n💬 Sabab: %s",
		adminUsername,
		time.Now().Format("02.01.2006 15:04"),
		booking.RejectionReason,
	)

	// Edit photo caption and remove keyboard
	_, err = h.bot.EditCaption(c.Message(), updatedCaption, &tele.ReplyMarkup{}, tele.ModeHTML)
	if err != nil {
		h.log.Error("Failed to edit admin message caption", logger.Error(err), logger.Any("message", updatedCaption))
	}

	return c.Respond(&tele.CallbackResponse{
		Text: "❌ To'lov rad etildi.",
	})
}

// HandleBlockUser handles blocking a user
func (h *Handler) HandleBlockUser(c tele.Context) error {
	ctx := context.Background()

	// Check if user is admin
	if !h.isAdmin(c.Sender().ID) {
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ Sizda bu amalga ruxsat yo'q.",
			ShowAlert: true,
		})
	}

	// Parse callback data: block_user_userID_bookingID
	data := strings.TrimPrefix(c.Callback().Data, "block_user_")
	var userID, bookingID int64
	_, err := fmt.Sscanf(data, "%d_%d", &userID, &bookingID)
	if err != nil {
		h.log.Error("Failed to parse user and booking IDs", logger.Error(err), logger.Any("callback_data", c.Callback().Data))
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ Noto'g'ri ma'lumot.",
			ShowAlert: true,
		})
	}

	// Block user and reject payment through service
	_, err = h.services.Payment().BlockUserAndRejectPayment(ctx, bookingID, userID, c.Sender().ID)
	if err != nil {
		h.log.Error("Failed to block user", logger.Error(err))
		return c.Respond(&tele.CallbackResponse{
			Text:      "❌ Xatolik yuz berdi.",
			ShowAlert: true,
		})
	}

	// Notify user
	go h.notifyUserBlocked(userID)

	// Update admin group message
	adminUsername := c.Sender().Username
	if adminUsername == "" {
		adminUsername = c.Sender().FirstName
	}

	updatedCaption := c.Message().Caption + fmt.Sprintf("\n\n🚫 <b>FOYDALANUVCHI BLOKLANDI</b>\n👤 Admin: @%s\n⏰ Vaqt: %s",
		adminUsername,
		time.Now().Format("02.01.2006 15:04"),
	)

	// Edit photo caption and remove keyboard
	_, err = h.bot.EditCaption(c.Message(), updatedCaption, &tele.ReplyMarkup{}, tele.ModeHTML)
	if err != nil {
		h.log.Error("Failed to edit admin message caption", logger.Error(err))
	}

	h.log.Warn("User blocked by admin",
		logger.Any("user_id", userID),
		logger.Any("admin_id", c.Sender().ID),
		logger.Any("booking_id", bookingID),
	)

	return c.Respond(&tele.CallbackResponse{
		Text: "🚫 Foydalanuvchi bloklandi.",
	})
}

// notifyUserPaymentApproved sends notification to user about approved payment
func (h *Handler) notifyUserPaymentApproved(booking *models.JobBooking) {
	ctx := context.Background()

	// Get job details
	job, err := h.storage.Job().GetByID(ctx, booking.JobID)
	if err != nil {
		h.log.Error("Failed to get job for notification", logger.Error(err))
		return
	}

	message := fmt.Sprintf(`✅ <b>TO'LOVINGIZ TASDIQLANDI!</b>

🎉 Tabriklaymiz! Sizning to'lovingiz admin tomonidan tasdiqlandi.

💼 <b>Ish ma'lumotlari:</b>
• Nomi: %s
• Tartib raqami: #%d
• Ish haqqi: %s

📋 <b>Keyingi qadamlar:</b>
1️⃣ Ishga tayyor bo'ling
2️⃣ Belgilangan vaqtda kelib turing
3️⃣ Ish haqqi ish tugagandan keyin to'lanadi

📞 <b>Savol bo'lsa:</b>
Agar savollaringiz bo'lsa, ish beruvchi bilan bog'laning.

✨ Omad tilaymiz!`,
		job.Salary,
		job.ID,
		job.Salary,
	)

	_, err = h.bot.Send(&tele.User{ID: booking.UserID}, message, tele.ModeHTML)
	if err != nil {
		h.log.Error("Failed to notify user", logger.Error(err))
	}
}

// notifyUserPaymentRejected sends notification to user about rejected payment
func (h *Handler) notifyUserPaymentRejected(booking *models.JobBooking) {
	ctx := context.Background()

	// Get job details
	job, err := h.storage.Job().GetByID(ctx, booking.JobID)
	if err != nil {
		h.log.Error("Failed to get job for notification", logger.Error(err))
		return
	}

	message := fmt.Sprintf(`❌ <b>TO'LOV RAD ETILDI</b>

Afsuski, sizning to'lov chekingiz admin tomonidan rad etildi.

💼 <b>Ish:</b> %s (Tartib #%d)
💬 <b>Sabab:</b> %s

📝 <b>Nima qilish kerak:</b>
1️⃣ To'lov chekini tekshiring
2️⃣ Agar to'lov qilgan bo'lsangiz, aniq va to'liq rasm yuboring
3️⃣ Agar to'lov qilmagan bo'lsangiz, qaytadan to'lov qiling va chekni yuboring

💡 <b>Maslahat:</b>
• Chek aniq va o'qilishi kerak
• Summa to'g'ri ko'rsatilgan bo'lishi kerak
• Sana bugungi kunni ko'rsatishi kerak

Agar joylar to'lgan bo'lsa, keyingi ishlar e'lon qilinishini kuting.`,
		job.Salary,
		job.ID,
		booking.RejectionReason,
	)

	_, err = h.bot.Send(&tele.User{ID: booking.UserID}, message, tele.ModeHTML)
	if err != nil {
		h.log.Error("Failed to notify user", logger.Error(err))
	}
}

// notifyUserBlocked sends notification to blocked user
func (h *Handler) notifyUserBlocked(userID int64) {
	message := `🚫 <b>SIZNING HISOBINGIZ BLOKLANDI</b>

Afsuski, qoidabuzarlik sababli sizning hisobingiz bloklandi.

❌ Siz endi ish bandlash imkoniyatiga ega emassiz.

📞 Agar bu xato deb hisoblasangiz, admin bilan bog'laning.`

	_, err := h.bot.Send(&tele.User{ID: userID}, message, tele.ModeHTML)
	if err != nil {
		h.log.Error("Failed to notify blocked user", logger.Error(err))
	}
}

// isAdmin checks if user is admin
func (h *Handler) isAdmin(userID int64) bool {
	return slices.Contains(h.cfg.Bot.AdminIDs, userID)
}
