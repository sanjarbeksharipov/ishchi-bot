package bot

import (
	"fmt"
	"strconv"
	"strings"
	"telegram-bot-starter/bot/handlers"
	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/pkg/keyboards"
	"telegram-bot-starter/pkg/logger"

	tele "gopkg.in/telebot.v4"
)

func RegisterRoutes(bot *tele.Bot, handler *handlers.Handler, log logger.LoggerI) {
	// Apply middleware
	// bot.Use(middleware.LoggingMiddleware(log))

	// Register command handlers
	bot.Handle("/start", handler.HandleStart)
	bot.Handle("/help", handler.HandleHelp)
	bot.Handle("/about", handler.HandleAbout)
	bot.Handle("/settings", handler.HandleSettings)
	bot.Handle("/admin", handler.HandleAdminPanel)

	// Register callback handlers
	bot.Handle(tele.OnCallback, func(c tele.Context) error {
		return handleCallBacks(c, handler)
	})

	// Register text message handler
	bot.Handle(tele.OnText, handler.HandleText)

	// Register contact handler (for phone sharing)
	bot.Handle(tele.OnContact, handler.HandleContact)

	// Register photo handler (for payment proofs)
	bot.Handle(tele.OnPhoto, handler.HandlePhoto)

	// Register location handler (for job locations)
	bot.Handle(tele.OnLocation, handler.HandleLocation)
}

func handleCallBacks(c tele.Context, handler *handlers.Handler) error {
	data := c.Callback().Data
	data = strings.TrimSpace(data)

	switch data {
	case "help":
		return handler.HandleHelpCallback(c)
	case "about":
		return handler.HandleAboutCallback(c)
	case "settings":
		return handler.HandleSettingsCallback(c)
	case "back":
		return handler.HandleBackCallback(c)
	case "confirm_yes":
		return handler.HandleConfirmYesCallback(c)
	case "confirm_no":
		return handler.HandleConfirmNoCallback(c)
	// Admin callbacks
	case "admin_menu":
		return handler.HandleAdminPanel(c)
	case "admin_create_job":
		return handler.HandleCreateJob(c)
	case "admin_job_list":
		return handler.HandleJobList(c)
	case "cancel_job_creation":
		return handler.HandleCancelJobCreation(c)
	case "skip_field":
		return handler.HandleSkipField(c)
	// Registration callbacks
	case "reg_accept_offer":
		return handler.HandleAcceptOffer(c)
	case "reg_decline_offer":
		return handler.HandleDeclineOffer(c)
	case "reg_continue":
		return handler.HandleContinueRegistration(c)
	case "reg_restart":
		return handler.HandleRestartRegistration(c)
	case "reg_confirm":
		return handler.HandleConfirmRegistration(c)
	case "reg_edit":
		return handler.HandleEditRegistration(c)
	case "reg_cancel":
		return handler.HandleCancelRegistration(c)
	case "reg_back_to_confirm":
		return handler.HandleBackToConfirm(c)
	case "reg_edit_full_name":
		return handler.HandleEditField(c, models.EditFieldFullName)
	case "reg_edit_phone":
		return handler.HandleEditField(c, models.EditFieldPhone)
	case "reg_edit_age":
		return handler.HandleEditField(c, models.EditFieldAge)
	case "reg_edit_body_params":
		return handler.HandleEditField(c, models.EditFieldBodyParams)
	// Booking callbacks
	case "book_cancel":
		return c.Edit("❌ Bekor qilindi.", keyboards.BackKeyboard())
	// User callbacks
	case "user_my_jobs":
		return handler.HandleUserMyJobs(c)
	case "user_profile":
		return handler.HandleUserProfile(c)
	// Profile editing callbacks
	case "edit_profile_full_name":
		return handler.HandleEditProfileField(c, "full_name")
	case "edit_profile_phone":
		return handler.HandleEditProfileField(c, "phone")
	case "edit_profile_age":
		return handler.HandleEditProfileField(c, "age")
	case "edit_profile_body_params":
		return handler.HandleEditProfileField(c, "body_params")
	default:
		// Handle admin callbacks with job IDs
		if after, ok := strings.CutPrefix(data, "job_detail_"); ok {
			jobID, _ := strconv.ParseInt(after, 10, 64)
			return handler.HandleJobDetail(c, jobID)
		}

		if after, ok := strings.CutPrefix(data, "edit_job_"); ok {
			// Format: edit_job_{id}_{field}
			parts := strings.Split(after, "_")
			if len(parts) >= 2 {
				jobID, _ := strconv.ParseInt(parts[0], 10, 64)
				field := strings.Join(parts[1:], "_")
				return handler.HandleEditJobField(c, jobID, field)
			}
		}

		if strings.HasPrefix(data, "job_status_") {
			// Format: job_status_{id}_{status}
			parts := strings.Split(strings.TrimPrefix(data, "job_status_"), "_")
			if len(parts) >= 2 {
				jobID, _ := strconv.ParseInt(parts[0], 10, 64)
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
				return handler.HandleChangeJobStatus(c, jobID, status)
			}
		}

		if strings.HasPrefix(data, "publish_job_") {
			jobID, _ := strconv.ParseInt(strings.TrimPrefix(data, "publish_job_"), 10, 64)
			return handler.HandlePublishJob(c, jobID)
		}

		if strings.HasPrefix(data, "delete_channel_msg_") {
			jobID, _ := strconv.ParseInt(strings.TrimPrefix(data, "delete_channel_msg_"), 10, 64)
			return handler.HandleDeleteChannelMessage(c, jobID)
		}

		if strings.HasPrefix(data, "delete_job_") {
			jobID, _ := strconv.ParseInt(strings.TrimPrefix(data, "delete_job_"), 10, 64)
			return handler.HandleDeleteJob(c, jobID)
		}

		if strings.HasPrefix(data, "view_job_bookings_") {
			jobID, _ := strconv.ParseInt(strings.TrimPrefix(data, "view_job_bookings_"), 10, 64)
			return handler.HandleViewJobBookings(c, jobID)
		}

		// Handle booking confirmation with job ID
		if strings.HasPrefix(data, "book_confirm_") {
			jobID, _ := strconv.ParseInt(strings.TrimPrefix(data, "book_confirm_"), 10, 64)
			return handler.HandleBookingConfirm(c, jobID)
		}

		// Handle registration start with job ID
		if strings.HasPrefix(data, "start_reg_job_") {
			jobID, _ := strconv.ParseInt(strings.TrimPrefix(data, "start_reg_job_"), 10, 64)
			return handler.HandleStartRegistrationForJob(c, jobID)
		}

		// Handle admin payment approval callbacks
		if strings.HasPrefix(data, "approve_payment") {
			return handler.HandleApprovePayment(c)
		}

		if strings.HasPrefix(data, "reject_payment") {
			return handler.HandleRejectPayment(c)
		}

		if strings.HasPrefix(data, "block_user") {
			return handler.HandleBlockUser(c)
		}

		fmt.Println(data)
		return c.Respond(&tele.CallbackResponse{Text: "Unknown action"})
	}
}
