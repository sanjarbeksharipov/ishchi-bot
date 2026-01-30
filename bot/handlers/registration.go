package handlers

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/pkg/keyboards"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/pkg/messages"

	tele "gopkg.in/telebot.v4"
)

// HandleRegistrationStart handles the start of registration flow
func (h *Handler) HandleRegistrationStart(c tele.Context) error {
	ctx := context.Background()
	userID := c.Sender().ID

	// Get services from service manager
	regService := h.services.Registration()
	senderService := h.services.Sender()

	// Check if user is already registered
	isRegistered, hasDraft, draft, err := regService.CheckUserRegistrationStatus(ctx, userID)
	if err != nil {
		h.log.Error("Failed to check registration status", logger.Error(err))
		return senderService.Reply(c, messages.MsgError)
	}

	// If registered, show main menu
	if isRegistered {
		registeredUser, err := regService.GetRegisteredUser(ctx, userID)
		if err != nil {
			h.log.Error("Failed to get registered user", logger.Error(err))
			return senderService.Reply(c, messages.MsgError)
		}
		return senderService.Reply(c, messages.FormatWelcomeRegistered(registeredUser.FullName), keyboards.UserMainMenuKeyboard())
	}

	// If has draft, ask to continue or restart
	if hasDraft && draft != nil {
		if draft.State == models.RegStateDeclined {
			// Previous registration was declined, start fresh
			_, err = regService.RestartRegistration(ctx, userID)
			if err != nil {
				h.log.Error("Failed to restart registration", logger.Error(err))
				return senderService.Reply(c, messages.MsgError)
			}
			// Show public offer
			return h.showPublicOffer(c)
		}
		return senderService.Reply(c, messages.MsgRegistrationContinue, keyboards.ContinueRegistrationKeyboard())
	}

	// No draft, start new registration
	_, err = regService.StartRegistration(ctx, userID)
	if err != nil {
		h.log.Error("Failed to start registration", logger.Error(err))
		return senderService.Reply(c, messages.MsgError)
	}

	// Show public offer
	return h.showPublicOffer(c)
}

// showPublicOffer displays the public offer and accept/decline buttons
func (h *Handler) showPublicOffer(c tele.Context) error {
	// Load public offer text
	offerPath := filepath.Join(".", "public_offer.txt")
	_, err := h.services.Registration().LoadPublicOffer(offerPath)
	if err != nil {
		h.log.Error("Failed to load public offer", logger.Error(err))
		// Use fallback summary
	}

	// Generate summary
	summary := h.services.Registration().GeneratePublicOfferSummary("")

	return h.services.Sender().Reply(c, summary, keyboards.PublicOfferKeyboard(), tele.ModeMarkdown)
}

// HandleAcceptOffer handles the accept offer callback
func (h *Handler) HandleAcceptOffer(c tele.Context) error {
	ctx := context.Background()
	userID := c.Sender().ID

	result, err := h.services.Registration().ProcessPublicOfferResponse(ctx, userID, true)
	if err != nil {
		h.log.Error("Failed to process offer acceptance", logger.Error(err))
		return h.services.Sender().Respond(c, &tele.CallbackResponse{Text: "Xatolik yuz berdi"})
	}

	// Answer callback
	h.services.Sender().Respond(c, &tele.CallbackResponse{Text: "Oferta qabul qilindi ✅"})

	// Update state in users table
	h.storage.User().UpdateState(ctx, userID, models.UserState(models.RegStateFullName))

	// Edit the message to remove buttons (so they can't be pressed again)
	h.services.Sender().EditMessage(c, "✅ Oferta qabul qilindi")

	// Send next step as new message
	return h.services.Sender().Reply(c, result.Message, keyboards.ReplyCancelKeyboard())
}

// HandleDeclineOffer handles the decline offer callback
func (h *Handler) HandleDeclineOffer(c tele.Context) error {
	ctx := context.Background()
	userID := c.Sender().ID

	result, err := h.services.Registration().ProcessPublicOfferResponse(ctx, userID, false)
	if err != nil {
		h.log.Error("Failed to process offer decline", logger.Error(err))
		return h.services.Sender().Respond(c, &tele.CallbackResponse{Text: "Xatolik yuz berdi"})
	}

	// Answer callback
	h.services.Sender().Respond(c, &tele.CallbackResponse{Text: "Oferta rad etildi ❌"})

	// Reset user state
	h.storage.User().UpdateState(ctx, userID, models.StateIdle)

	// Send message
	return h.services.Sender().EditMessage(c, result.Message)
}

// HandleContinueRegistration continues the registration from where user left off
func (h *Handler) HandleContinueRegistration(c tele.Context) error {
	ctx := context.Background()
	userID := c.Sender().ID

	draft, err := h.services.Registration().GetOrCreateDraft(ctx, userID)
	if err != nil {
		h.log.Error("Failed to get draft", logger.Error(err))
		return h.services.Sender().Respond(c, &tele.CallbackResponse{Text: "Xatolik yuz berdi"})
	}

	h.services.Sender().Respond(c, &tele.CallbackResponse{Text: "Davom etamiz"})

	// Continue from current state
	return h.sendStatePrompt(c, draft.State)
}

// HandleRestartRegistration restarts the registration from beginning
func (h *Handler) HandleRestartRegistration(c tele.Context) error {
	ctx := context.Background()
	userID := c.Sender().ID

	_, err := h.services.Registration().RestartRegistration(ctx, userID)
	if err != nil {
		h.log.Error("Failed to restart registration", logger.Error(err))
		return h.services.Sender().Respond(c, &tele.CallbackResponse{Text: "Xatolik yuz berdi"})
	}

	h.services.Sender().Respond(c, &tele.CallbackResponse{Text: "Qayta boshlaymiz"})

	// Show public offer
	return h.showPublicOffer(c)
}

// HandleRegistrationTextInput handles text input during registration
func (h *Handler) HandleRegistrationTextInput(c tele.Context, state models.RegistrationState) error {
	ctx := context.Background()
	userID := c.Sender().ID
	text := strings.TrimSpace(c.Text())

	// Handle cancel button
	if text == "❌ Bekor qilish" {
		return h.HandleCancelRegistration(c)
	}

	switch state {
	case models.RegStateFullName:
		return h.processFullName(ctx, c, userID, text)

	case models.RegStatePhone:
		// Accept phone as text input and validate it
		return h.processPhone(ctx, c, userID, text)
	case models.RegStateAge:
		return h.processAge(ctx, c, userID, text)

	case models.RegStateBodyParams:
		return h.processBodyParams(ctx, c, userID, text)

	default:
		return nil
	}
}

// HandleCancelText handles the "❌ Bekor qilish" text command
func (h *Handler) HandleCancelText(c tele.Context) error {
	return h.HandleCancelRegistration(c)
}

// HandleRegistrationContact handles contact sharing during registration
func (h *Handler) HandleRegistrationContact(c tele.Context) error {
	ctx := context.Background()
	userID := c.Sender().ID
	contact := c.Message().Contact

	if contact == nil {
		return h.services.Sender().Reply(c, messages.MsgPhoneRequestManualInput, keyboards.PhoneRequestKeyboard())
	}

	// Verify that the contact is from the same user
	if contact.UserID != userID {
		return h.services.Sender().Reply(c, "❌ Iltimos, o'z telefon raqamingizni yuboring!", keyboards.PhoneRequestKeyboard())
	}

	result, err := h.services.Registration().ProcessPhone(ctx, userID, contact.PhoneNumber)
	if err != nil {
		h.log.Error("Failed to process phone", logger.Error(err))
		return h.services.Sender().Reply(c, messages.MsgError)
	}

	if !result.Success {
		return h.services.Sender().Reply(c, result.ErrorMessage, keyboards.PhoneRequestKeyboard())
	}

	// Update state
	h.storage.User().UpdateState(ctx, userID, models.UserState(result.NextState))

	// If we're returning to confirmation, show confirmation screen
	if result.NextState == models.RegStateConfirm {
		return h.showRegistrationConfirmation(ctx, c, userID)
	}

	// Send next step with ReplyCancelKeyboard (replaces the phone button)
	return h.services.Sender().Reply(c, result.Message, keyboards.ReplyCancelKeyboard())
}

// showRegistrationConfirmation shows the registration summary for confirmation
func (h *Handler) showRegistrationConfirmation(ctx context.Context, c tele.Context, userID int64) error {
	draft, err := h.services.Registration().GetOrCreateDraft(ctx, userID)
	if err != nil {
		h.log.Error("Failed to get draft for confirmation", logger.Error(err))
		return h.services.Sender().Reply(c, messages.MsgError)
	}

	summary := h.services.Registration().FormatRegistrationSummary(draft)

	// Send summary with buttons (no photo)
	return h.services.Sender().Reply(c, summary, keyboards.RegistrationConfirmKeyboard(), tele.ModeMarkdown)
}

// HandleConfirmRegistration handles the confirmation callback
func (h *Handler) HandleConfirmRegistration(c tele.Context) error {
	ctx := context.Background()
	userID := c.Sender().ID

	// Check if there's a pending job ID before completing registration
	draft, err := h.services.Registration().GetOrCreateDraft(ctx, userID)
	if err != nil {
		h.log.Error("Failed to get draft", logger.Error(err))
	}

	var pendingJobID *int64
	if draft != nil && draft.PendingJobID != nil {
		pendingJobID = draft.PendingJobID
		h.log.Info("Found pending job ID for post-registration redirect",
			logger.Any("user_id", userID),
			logger.Any("job_id", *pendingJobID),
		)
	}

	result, err := h.services.Registration().ConfirmRegistration(ctx, userID)
	if err != nil {
		h.log.Error("Failed to confirm registration", logger.Error(err))
		return h.services.Sender().Respond(c, &tele.CallbackResponse{Text: "Xatolik yuz berdi"})
	}

	if !result.Success {
		return h.services.Sender().Respond(c, &tele.CallbackResponse{Text: result.ErrorMessage, ShowAlert: true})
	}

	h.services.Sender().Respond(c, &tele.CallbackResponse{Text: "Ro'yxatdan o'tdingiz! 🎉"})

	// Update user state
	h.storage.User().UpdateState(ctx, userID, models.StateIdle)

	// If there was a pending job, redirect to booking flow
	if pendingJobID != nil {
		user, err := h.storage.User().GetByID(ctx, userID)
		if err != nil {
			h.log.Error("Failed to get user for booking redirect", logger.Error(err))
			return h.services.Sender().EditMessage(c, result.Message, keyboards.UserMainMenuKeyboard())
		}

		// Send success message first
		if err := h.services.Sender().EditMessage(c, result.Message); err != nil {
			h.log.Error("Failed to send success message", logger.Error(err))
		}

		// Small delay to let user see the success message
		time.Sleep(1 * time.Second)

		// Redirect to job booking
		return h.HandleJobBookingStart(c, user, *pendingJobID)
	}
	h.services.Sender().DeleteMessage(c)
	// We need to send a new message to ensure the ReplyCancelKeyboard is removed/replaced
	return h.services.Sender().Reply(c, result.Message, keyboards.UserMainMenuReplyKeyboard())
}

// HandleEditRegistration shows edit field selection
func (h *Handler) HandleEditRegistration(c tele.Context) error {
	h.services.Sender().Respond(c, &tele.CallbackResponse{Text: "Tahrirlash"})
	return h.services.Sender().EditMessage(c, messages.MsgSelectEditField, keyboards.RegistrationEditFieldKeyboard())
}

// HandleEditField handles edit field selection
func (h *Handler) HandleEditField(c tele.Context, field models.EditField) error {
	ctx := context.Background()
	userID := c.Sender().ID

	result, err := h.services.Registration().GoToEditState(ctx, userID, field)
	if err != nil {
		h.log.Error("Failed to set edit state", logger.Error(err))
		return h.services.Sender().Respond(c, &tele.CallbackResponse{Text: "Xatolik yuz berdi"})
	}

	h.services.Sender().Respond(c, &tele.CallbackResponse{Text: "Tahrirlash"})

	// Update user state
	h.storage.User().UpdateState(ctx, userID, models.UserState(result.NextState))

	// Send appropriate prompt
	return h.sendStatePrompt(c, result.NextState)
}

// HandleBackToConfirm returns to confirmation screen
func (h *Handler) HandleBackToConfirm(c tele.Context) error {
	ctx := context.Background()
	userID := c.Sender().ID

	// Get draft
	draft, err := h.services.Registration().GetOrCreateDraft(ctx, userID)
	if err != nil {
		h.log.Error("Failed to get draft", logger.Error(err))
		return h.services.Sender().Respond(c, &tele.CallbackResponse{Text: "Xatolik yuz berdi"})
	}

	// Update state to confirm
	draft.State = models.RegStateConfirm
	h.storage.User().UpdateState(ctx, userID, models.UserState(models.RegStateConfirm))

	h.services.Sender().Respond(c, &tele.CallbackResponse{Text: "Orqaga"})

	return h.showRegistrationConfirmation(ctx, c, userID)
}

// HandleCancelRegistration cancels the registration
func (h *Handler) HandleCancelRegistration(c tele.Context) error {
	ctx := context.Background()
	userID := c.Sender().ID

	err := h.services.Registration().CancelRegistration(ctx, userID)
	if err != nil {
		h.log.Error("Failed to cancel registration", logger.Error(err))
	}

	// Reset user state
	h.storage.User().UpdateState(ctx, userID, models.StateIdle)

	// Check if this is a callback or text message
	if c.Callback() != nil {
		h.services.Sender().Respond(c, &tele.CallbackResponse{Text: "Bekor qilindi"})
		// Remove any reply keyboard just in case
		h.services.Sender().RemoveKeyboard(c)
		return h.services.Sender().EditMessage(c, messages.MsgRegistrationCancelled)
	}

	// If text message (from Reply button)
	return h.services.Sender().Reply(c, messages.MsgRegistrationCancelled, keyboards.RemoveReplyKeyboard())
}

// processPhone handles phone input (text or contact)
func (h *Handler) processPhone(ctx context.Context, c tele.Context, userID int64, phone string) error {
	result, err := h.services.Registration().ProcessPhone(ctx, userID, phone)
	if err != nil {
		h.log.Error("Failed to process phone", logger.Error(err))
		return h.services.Sender().Reply(c, messages.MsgError)
	}

	if !result.Success {
		return h.services.Sender().Reply(c, result.ErrorMessage+"\n\n"+messages.MsgEnterPhone, keyboards.PhoneRequestKeyboard())
	}

	// Update state
	h.storage.User().UpdateState(ctx, userID, models.UserState(result.NextState))

	// If we're returning to confirmation (edit mode), show confirmation screen directly
	if result.NextState == models.RegStateConfirm {
		return h.showRegistrationConfirmation(ctx, c, userID)
	}

	// Send next step with ReplyCancelKeyboard (replaces phone button)
	return h.services.Sender().Reply(c, result.Message, keyboards.ReplyCancelKeyboard())
}

// processFullName handles full name input
func (h *Handler) processFullName(ctx context.Context, c tele.Context, userID int64, text string) error {
	result, err := h.services.Registration().ProcessFullName(ctx, userID, text)
	if err != nil {
		h.log.Error("Failed to process full name", logger.Error(err))
		return h.services.Sender().Reply(c, messages.MsgError)
	}

	if !result.Success {
		return h.services.Sender().Reply(c, result.ErrorMessage+"\n\n"+messages.MsgEnterFullName, keyboards.RegistrationCancelKeyboard())
	}

	// Update state
	h.storage.User().UpdateState(ctx, userID, models.UserState(result.NextState))

	// If we're returning to confirmation, show confirmation screen directly
	if result.NextState == models.RegStateConfirm {
		return h.showRegistrationConfirmation(ctx, c, userID)
	}

	// Send info message about phone input options
	infoMsg := "ℹ️ Telefon raqamingizni yuborishingiz mumkin — uni qo'lda yozish yoki tugma orqali yuborish mumkin.\n\n"
	// Send next step with phone keyboard
	return h.services.Sender().Reply(c, infoMsg+result.Message, keyboards.PhoneRequestKeyboard())
}

// processAge handles age input
func (h *Handler) processAge(ctx context.Context, c tele.Context, userID int64, text string) error {
	result, err := h.services.Registration().ProcessAge(ctx, userID, text)
	if err != nil {
		h.log.Error("Failed to process age", logger.Error(err))
		return h.services.Sender().Reply(c, messages.MsgError)
	}

	if !result.Success {
		return h.services.Sender().Reply(c, result.ErrorMessage+"\n\n"+messages.MsgEnterAge, keyboards.RegistrationCancelKeyboard())
	}

	// Update state
	h.storage.User().UpdateState(ctx, userID, models.UserState(result.NextState))

	// If we're returning to confirmation (edit mode), show confirmation screen directly
	if result.NextState == models.RegStateConfirm {
		return h.showRegistrationConfirmation(ctx, c, userID)
	}

	return h.services.Sender().Reply(c, result.Message)
}

// processBodyParams handles body params input
func (h *Handler) processBodyParams(ctx context.Context, c tele.Context, userID int64, text string) error {
	result, err := h.services.Registration().ProcessBodyParams(ctx, userID, text)
	if err != nil {
		h.log.Error("Failed to process body params", logger.Error(err))
		return h.services.Sender().Reply(c, messages.MsgError)
	}

	if !result.Success {
		return h.services.Sender().Reply(c, result.ErrorMessage+"\n\n"+messages.MsgEnterBodyParams)
	}

	// Update state
	h.storage.User().UpdateState(ctx, userID, models.UserState(result.NextState))

	// Always show confirmation after body params (no passport photo step)
	// Remove any keyboard first
	h.services.Sender().RemoveKeyboard(c)
	return h.showRegistrationConfirmation(ctx, c, userID)
}

// sendStatePrompt sends the appropriate prompt for the given state
func (h *Handler) sendStatePrompt(c tele.Context, state models.RegistrationState) error {
	switch state {
	case models.RegStatePublicOffer:
		return h.showPublicOffer(c)

	case models.RegStateFullName:
		return h.services.Sender().Reply(c, messages.MsgEnterFullName, keyboards.RegistrationCancelKeyboard())

	case models.RegStatePhone:
		return h.services.Sender().Reply(c, messages.MsgEnterPhone, keyboards.PhoneRequestKeyboard())

	case models.RegStateAge:
		return h.services.Sender().Reply(c, messages.MsgEnterAge, keyboards.RegistrationCancelKeyboard())

	case models.RegStateBodyParams:
		return h.services.Sender().Reply(c, messages.MsgEnterBodyParams, keyboards.RegistrationCancelKeyboard())

	case models.RegStateConfirm:
		ctx := context.Background()
		return h.showRegistrationConfirmation(ctx, c, c.Sender().ID)

	default:
		return nil
	}
}

// IsInRegistrationFlow checks if user is in registration flow based on their state
func (h *Handler) IsInRegistrationFlow(userState models.UserState) bool {
	return models.IsRegistrationState(userState)
}

// GetRegistrationState converts UserState to RegistrationState
func (h *Handler) GetRegistrationState(userState models.UserState) models.RegistrationState {
	return models.RegistrationState(userState)
}
