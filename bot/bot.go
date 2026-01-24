package bot

import (
	"fmt"
	"strconv"
	"strings"
	"telegram-bot-starter/bot/handlers"
	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/pkg/logger"

	tele "gopkg.in/telebot.v4"
)

func SetUpRoutes(bot *tele.Bot, handler *handlers.Handler, log logger.LoggerI) {
	// Initialize handlers

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
		data := c.Callback().Data
		data = strings.TrimSpace(data)

		// Handle admin callbacks with job IDs
		if strings.HasPrefix(data, "job_detail_") {
			jobID, _ := strconv.ParseInt(strings.TrimPrefix(data, "job_detail_"), 10, 64)
			return handler.HandleJobDetail(c, jobID)
		}

		if strings.HasPrefix(data, "edit_job_") {
			// Format: edit_job_{id}_{field}
			parts := strings.Split(strings.TrimPrefix(data, "edit_job_"), "_")
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
					status = models.JobStatusOpen
				case "toldi":
					status = models.JobStatusToldi
				case "closed":
					status = models.JobStatusClosed
				}
				return handler.HandleChangeJobStatus(c, jobID, status)
			}
		}

		if strings.HasPrefix(data, "publish_job_") {
			jobID, _ := strconv.ParseInt(strings.TrimPrefix(data, "publish_job_"), 10, 64)
			return handler.HandlePublishJob(c, jobID)
		}

		if strings.HasPrefix(data, "delete_job_") {
			jobID, _ := strconv.ParseInt(strings.TrimPrefix(data, "delete_job_"), 10, 64)
			return handler.HandleDeleteJob(c, jobID)
		}

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

		default:
			fmt.Println(data)
			return c.Respond(&tele.CallbackResponse{Text: "Unknown action"})
		}
	})

	// Register text message handler
	bot.Handle(tele.OnText, handler.HandleText)

	// Register contact handler (for phone sharing)
	bot.Handle(tele.OnContact, handler.HandleContact)

	// Register photo handler (for passport photo)
	bot.Handle(tele.OnPhoto, handler.HandlePhoto)
}
