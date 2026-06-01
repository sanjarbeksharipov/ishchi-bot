package keyboards

import (
	"fmt"

	"telegram-bot-starter/bot/models"

	tele "gopkg.in/telebot.v4"
)

// MainMenuKeyboard returns the main menu inline keyboard
func MainMenuKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnHelp := menu.Data("📖 Help", "help")
	btnAbout := menu.Data("ℹ️ About", "about")
	btnSettings := menu.Data("⚙️ Settings", "settings")

	menu.Inline(
		menu.Row(btnHelp, btnAbout),
		menu.Row(btnSettings),
	)

	return menu
}

// ConfirmationKeyboard returns a yes/no confirmation keyboard
func ConfirmationKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnYes := menu.Data("✅ Yes", "confirm_yes")
	btnNo := menu.Data("❌ No", "confirm_no")

	menu.Inline(
		menu.Row(btnYes, btnNo),
	)

	return menu
}

// UsersPaginationKeyboard returns pagination keyboard for users list
func UsersPaginationKeyboard(currentPage, totalPages int) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	var buttons []tele.Btn

	// Previous button
	if currentPage > 1 {
		btnPrev := menu.Data("⬅️ Oldingi", fmt.Sprintf("users_page_%d", currentPage-1))
		buttons = append(buttons, btnPrev)
	}

	// Page indicator (non-clickable)
	btnPage := menu.Data(fmt.Sprintf("%d/%d", currentPage, totalPages), "users_page_current")
	buttons = append(buttons, btnPage)

	// Next button
	if currentPage < totalPages {
		btnNext := menu.Data("Keyingi ➡️", fmt.Sprintf("users_page_%d", currentPage+1))
		buttons = append(buttons, btnNext)
	}

	menu.Inline(
		menu.Row(buttons...),
		menu.Row(menu.Data("⬅️ Admin panel", "admin_menu")),
	)

	return menu
}

// BackKeyboard returns a simple back button keyboard
func BackKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	btnBack := menu.Data("⬅️ Back", "back")
	menu.Inline(menu.Row(btnBack))
	return menu
}

// AdminMenuKeyboard returns the admin panel main menu
func AdminMenuKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnCreateJob := menu.Data("➕ Ish yaratish", "admin_create_job")
	btnJobList := menu.Data("📋 Ishlar ro'yxati", "admin_job_list")

	menu.Inline(
		menu.Row(btnCreateJob),
		menu.Row(btnJobList),
	)

	return menu
}
func AdminMenuReplyKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnCreateJob := menu.Text("➕ Ish yaratish")
	btnJobList := menu.Text("📋 Ishlar ro'yxati")
	btnUsersList := menu.Text("👥 Foydalanuvchilar")
	btnStats := menu.Text("📊 Statistika")

	menu.Reply(
		menu.Row(btnCreateJob),
		menu.Row(btnJobList),
		menu.Row(btnUsersList, btnStats),
	)

	return menu
}

// JobListKeyboard returns keyboard with list of jobs
func JobListKeyboard(jobs []*models.Job) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	var rows []tele.Row
	for _, job := range jobs {
		statusIcon := "🟢"
		switch job.Status {
		case models.JobStatusFull:
			statusIcon = "🔴"
		case models.JobStatusCompleted:
			statusIcon = "⚫"
		}

		btnText := fmt.Sprintf("%s № %d - %s", statusIcon, job.OrderNumber, job.WorkDate)
		btn := menu.Data(btnText, fmt.Sprintf("job_detail_%d", job.ID))
		rows = append(rows, menu.Row(btn))
	}

	// Add back button
	rows = append(rows, menu.Row(menu.Data("⬅️ Orqaga", "admin_menu")))

	menu.Inline(rows...)
	return menu
}

// JobDetailKeyboard returns keyboard for job detail view with edit options
func JobDetailKeyboard(job *models.Job) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	// Edit field buttons
	btnEditIshHaqqi := menu.Data("💰 Ish haqqi", fmt.Sprintf("edit_job_%d_ish_haqqi", job.ID))
	btnEditOvqat := menu.Data("🍛 Ovqat", fmt.Sprintf("edit_job_%d_ovqat", job.ID))
	btnEditVaqt := menu.Data("⏰ Vaqt", fmt.Sprintf("edit_job_%d_vaqt", job.ID))
	btnEditManzil := menu.Data("📍 Manzil", fmt.Sprintf("edit_job_%d_manzil", job.ID))
	btnEditLocation := menu.Data("📌 Joylashuv", fmt.Sprintf("edit_job_%d_location", job.ID))
	btnEditXizmatHaqqi := menu.Data("🌟 Xizmat haqqi", fmt.Sprintf("edit_job_%d_xizmat_haqqi", job.ID))
	btnEditAvtobuslar := menu.Data("🚌 Avtobuslar", fmt.Sprintf("edit_job_%d_avtobuslar", job.ID))
	btnEditIshTavsifi := menu.Data("📝 Ish tavsifi", fmt.Sprintf("edit_job_%d_ish_tavsifi", job.ID))
	btnEditIshKuni := menu.Data("📅 Ish kuni", fmt.Sprintf("edit_job_%d_ish_kuni", job.ID))
	btnEditKerakli := menu.Data("👥 Kerakli ishchilar", fmt.Sprintf("edit_job_%d_kerakli", job.ID))
	btnEditConfirmed := menu.Data("✅ Qabul qilingan", fmt.Sprintf("edit_job_%d_confirmed", job.ID))
	btnEditEmployerPhone := menu.Data("📞 Ish beruvchi tel", fmt.Sprintf("edit_job_%d_employer_phone", job.ID))

	// Status buttons
	btnStatusOpen := menu.Data("🟢 Ochiq", fmt.Sprintf("job_status_%d_open", job.ID))
	btnStatusToldi := menu.Data("🔴 To'ldi", fmt.Sprintf("job_status_%d_toldi", job.ID))
	btnStatusClosed := menu.Data("⚫ Yopilgan", fmt.Sprintf("job_status_%d_closed", job.ID))

	// Action buttons
	var rows []tele.Row
	rows = append(rows, menu.Row(btnEditIshHaqqi, btnEditOvqat))
	rows = append(rows, menu.Row(btnEditVaqt, btnEditManzil))
	rows = append(rows, menu.Row(btnEditLocation, btnEditXizmatHaqqi))
	rows = append(rows, menu.Row(btnEditAvtobuslar, btnEditIshTavsifi))
	rows = append(rows, menu.Row(btnEditIshKuni, btnEditKerakli))
	rows = append(rows, menu.Row(btnEditConfirmed, btnEditEmployerPhone))
	rows = append(rows, menu.Row(btnStatusOpen, btnStatusToldi, btnStatusClosed))

	// Publish or delete message buttons
	if !job.IsPublished {
		btnPublish := menu.Data("📢 Kanalga yuborish", fmt.Sprintf("publish_job_%d", job.ID))
		rows = append(rows, menu.Row(btnPublish))
	} else {
		btnDeleteMsg := menu.Data("🗑 Kanaldagi xabarni o'chirish", fmt.Sprintf("delete_channel_msg_%d", job.ID))
		rows = append(rows, menu.Row(btnDeleteMsg))
	}

	// View bookings button
	btnViewBookings := menu.Data("👥 Yozilganlarni ko'rish", fmt.Sprintf("view_job_bookings_%d", job.ID))
	rows = append(rows, menu.Row(btnViewBookings))

	btnDelete := menu.Data("❌ Ishni butunlay o'chirish", fmt.Sprintf("delete_job_%d", job.ID))
	btnBack := menu.Data("⬅️ Orqaga", "admin_job_list")

	rows = append(rows, menu.Row(btnDelete))
	rows = append(rows, menu.Row(btnBack))

	menu.Inline(rows...)

	return menu
}

// CancelKeyboard returns a cancel button keyboard
func CancelKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	btnCancel := menu.Data("❌ Bekor qilish", "cancel_job_creation")
	menu.Inline(menu.Row(btnCancel))
	return menu
}

// CancelOrSkipKeyboard returns cancel and skip buttons for optional fields
func CancelOrSkipKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	btnSkip := menu.Data("⏭ O'tkazib yuborish", "skip_field")
	btnCancel := menu.Data("❌ Bekor qilish", "cancel_job_creation")
	menu.Inline(
		menu.Row(btnSkip),
		menu.Row(btnCancel),
	)
	return menu
}

// CancelEditKeyboard returns cancel button for editing with return to job detail
func CancelEditKeyboard(jobID int64) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	btnCancel := menu.Data("❌ Bekor qilish", fmt.Sprintf("job_detail_%d", jobID))
	menu.Inline(menu.Row(btnCancel))
	return menu
}

// JobSignupKeyboard returns keyboard with signup button for channel posts
func JobSignupKeyboard(jobID int64, botUsername string) *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	signupURL := fmt.Sprintf("https://t.me/%s?start=job_%d", botUsername, jobID)
	btnSignup := menu.URL("✍️ Ishga yozilish", signupURL)
	menu.Inline(menu.Row(btnSignup))
	return menu
}

// ========== Registration Keyboards ==========

// PublicOfferKeyboard returns accept/decline buttons for public offer
func PublicOfferKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnAccept := menu.Data("✅ Qabul qilaman", "reg_accept_offer")
	btnDecline := menu.Data("❌ Rad etaman", "reg_decline_offer")

	menu.Inline(
		menu.Row(btnAccept, btnDecline),
	)

	return menu
}

// PhoneRequestKeyboard returns reply keyboard with contact sharing button
func PhoneRequestKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{
		ResizeKeyboard:  true,
		OneTimeKeyboard: true,
	}

	btnContact := menu.Contact("📞 Telefon raqamni yuborish")
	btnCancel := menu.Text("❌ Bekor qilish")

	menu.Reply(
		menu.Row(btnContact),
		menu.Row(btnCancel),
	)

	return menu
}

// RegistrationConfirmKeyboard returns confirm/edit/cancel buttons
func RegistrationConfirmKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnConfirm := menu.Data("✅ Tasdiqlash", "reg_confirm")
	btnEdit := menu.Data("✏️ Tahrirlash", "reg_edit")
	btnCancel := menu.Data("❌ Bekor qilish", "reg_cancel")

	menu.Inline(
		menu.Row(btnConfirm),
		menu.Row(btnEdit, btnCancel),
	)

	return menu
}

// RegistrationEditFieldKeyboard returns buttons to select which field to edit
func RegistrationEditFieldKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnFullName := menu.Data("👤 Ism-familiya", "reg_edit_full_name")
	btnPhone := menu.Data("📱 Telefon", "reg_edit_phone")
	btnAge := menu.Data("🎂 Yosh", "reg_edit_age")
	btnBody := menu.Data("📏 Vazn/Bo'y", "reg_edit_body_params")
	btnBack := menu.Data("⬅️ Orqaga", "reg_back_to_confirm")

	menu.Inline(
		menu.Row(btnFullName, btnPhone),
		menu.Row(btnAge, btnBody),
		menu.Row(btnBack),
	)

	return menu
}

// RegistrationCancelKeyboard returns cancel button for registration flow
func RegistrationCancelKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	btnCancel := menu.Data("❌ Bekor qilish", "reg_cancel")
	menu.Inline(menu.Row(btnCancel))
	return menu
}

// RemoveReplyKeyboard returns an empty reply markup to remove any existing reply keyboard
func RemoveReplyKeyboard() *tele.ReplyMarkup {
	return &tele.ReplyMarkup{
		RemoveKeyboard: true,
	}
}

// UserMainMenuKeyboard returns the main menu for registered users
func UserMainMenuKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnMyJobs := menu.Data("📋 Mening ishlarim", "user_my_jobs")
	btnProfile := menu.Data("👤 Profil", "user_profile")
	btnHelp := menu.Data("❓ Yordam", "help")

	menu.Inline(
		menu.Row(btnMyJobs, btnProfile),
		menu.Row(btnHelp),
	)

	return menu
}
func UserMainMenuReplyKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}
	btnMyJobs := menu.Text("📋 Mening ishlarim")
	btnProfile := menu.Text("👤 Profil")
	btnHelp := menu.Text("❓ Yordam")

	menu.Reply(
		menu.Row(btnMyJobs, btnProfile),
		menu.Row(btnHelp),
	)

	return menu
}

// ContinueRegistrationKeyboard returns keyboard to continue or restart registration
func ContinueRegistrationKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{}

	btnContinue := menu.Data("▶️ Davom ettirish", "reg_continue")
	btnRestart := menu.Data("🔄 Qaytadan boshlash", "reg_restart")

	menu.Inline(
		menu.Row(btnContinue),
		menu.Row(btnRestart),
	)

	return menu
}

// ReplyCancelKeyboard returns a reply keyboard with only cancel button
func ReplyCancelKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{
		ResizeKeyboard: true,
	}

	btnCancel := menu.Text("❌ Bekor qilish")
	menu.Reply(menu.Row(btnCancel))

	return menu
}

// ProfileEditKeyboard returns keyboard for profile editing
func ProfileEditKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{
		ResizeKeyboard: true,
	}

	btnEditFullName := menu.Text("👤 Ism familiya")
	btnEditPhone := menu.Text("📞 Telefon raqami")
	btnEditAge := menu.Text("🎂 Yosh")
	btnEditBodyParams := menu.Text("📏 Vazn va Bo'y")
	btnMainMenu := menu.Text("🏠 Asosiy menyu")

	menu.Reply(
		menu.Row(btnEditFullName, btnEditPhone),
		menu.Row(btnEditAge, btnEditBodyParams),
		menu.Row(btnMainMenu),
	)

	return menu
}

// RequestPhoneKeyboard returns keyboard to request phone number
func RequestPhoneKeyboard() *tele.ReplyMarkup {
	menu := &tele.ReplyMarkup{
		ResizeKeyboard:  true,
		OneTimeKeyboard: true,
	}

	btnPhone := menu.Contact("📱 Telefon raqamni yuborish")
	btnCancel := menu.Text("❌ Bekor qilish")

	menu.Reply(
		menu.Row(btnPhone),
		menu.Row(btnCancel),
	)

	return menu
}
