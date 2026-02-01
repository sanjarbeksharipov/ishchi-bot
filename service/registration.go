package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/config"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/pkg/messages"
	"telegram-bot-starter/pkg/validation"
	"telegram-bot-starter/storage"
)

// RegistrationService handles registration business logic
type RegistrationService struct {
	cfg     config.Config
	log     logger.LoggerI
	storage storage.StorageI
	service ServiceManagerI
}

// NewRegistrationService creates a new registration service
func NewRegistrationService(cfg config.Config, log logger.LoggerI, storage storage.StorageI, service ServiceManagerI) RegistrationService {
	return RegistrationService{
		cfg:     cfg,
		log:     log,
		storage: storage,
		service: service,
	}
}

// RegistrationResult represents the result of a registration action
type RegistrationResult struct {
	Success      bool
	NextState    models.RegistrationState
	Message      string
	ErrorMessage string
	Draft        *models.RegistrationDraft
}

// CheckUserRegistrationStatus checks if user is registered and returns appropriate action
func (s RegistrationService) CheckUserRegistrationStatus(ctx context.Context, userID int64) (isRegistered bool, hasDraft bool, draft *models.RegistrationDraft, err error) {
	s.log.Info("!!!Check User Registration Status", logger.Any("user_id", userID))

	// Check if user is fully registered
	isRegistered, err = s.storage.Registration().IsUserRegistered(ctx, userID)
	if err != nil {
		s.log.Error("???Failed to check registration status", logger.Error(err), logger.Any("user_id", userID))
		return false, false, nil, err
	}

	if isRegistered {
		s.log.Info("User is already registered", logger.Any("user_id", userID))
		return true, false, nil, nil
	}

	// Check for existing draft
	draft, err = s.storage.Registration().GetDraftByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			s.log.Info("No registration draft found", logger.Any("user_id", userID))
			return false, false, nil, nil
		}
		s.log.Error("???Failed to get registration draft", logger.Error(err), logger.Any("user_id", userID))
		return false, false, nil, err
	}

	s.log.Info("Found existing registration draft", logger.Any("user_id", userID), logger.Any("draft_state", draft.State))
	return false, true, draft, nil
}

// StartRegistration creates a new registration draft for the user
func (s RegistrationService) StartRegistration(ctx context.Context, userID int64) (*models.RegistrationDraft, error) {
	s.log.Info("!!!Start Registration", logger.Any("user_id", userID))

	// First, ensure user exists in users table
	_, err := s.storage.User().GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			s.log.Error("???User not found for registration", logger.Any("user_id", userID))
			return nil, fmt.Errorf("user not found, please use /start first")
		}
		s.log.Error("???Failed to get user", logger.Error(err), logger.Any("user_id", userID))
		return nil, err
	}

	// Delete any existing draft
	_ = s.storage.Registration().DeleteDraft(ctx, userID)

	// Create new draft
	draft := models.NewRegistrationDraft(userID)
	err = s.storage.Registration().CreateDraft(ctx, draft)
	if err != nil {
		s.log.Error("???Failed to create registration draft", logger.Error(err), logger.Any("user_id", userID))
		return nil, err
	}

	s.log.Info("Registration draft created successfully", logger.Any("user_id", userID))
	return draft, nil
}

// GetOrCreateDraft gets existing draft or creates a new one
func (s RegistrationService) GetOrCreateDraft(ctx context.Context, userID int64) (*models.RegistrationDraft, error) {
	draft, err := s.storage.Registration().GetDraftByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return s.StartRegistration(ctx, userID)
		}
		return nil, err
	}
	return draft, nil
}

// LoadPublicOffer loads the public offer text from file
func (s RegistrationService) LoadPublicOffer(filePath string) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		s.log.Error("Failed to read public offer file", logger.Error(err))
		return "", fmt.Errorf("failed to read public offer: %w", err)
	}
	return string(content), nil
}

// // GeneratePublicOfferSummary generates a summary of the public offer
// func (s RegistrationService) GeneratePublicOfferSummary(fullText string) string {
// 	summary := ``

// 	return summary
// }

// ProcessPublicOfferResponse handles accept/decline response
func (s RegistrationService) ProcessPublicOfferResponse(ctx context.Context, userID int64, accepted bool) (*RegistrationResult, error) {
	draft, err := s.storage.Registration().GetDraftByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !accepted {
		// User declined, mark and delete draft
		draft.State = models.RegStateDeclined
		_ = s.storage.Registration().DeleteDraft(ctx, userID)

		return &RegistrationResult{
			Success:      false,
			NextState:    models.RegStateDeclined,
			Message:      "❌ Siz ofertani qabul qilmadingiz. Ro'yxatdan o'tish bekor qilindi.\n\nQayta ro'yxatdan o'tish uchun /start buyrug'ini yuboring.",
			ErrorMessage: "",
			Draft:        nil,
		}, nil
	}

	// User accepted, move to next state
	draft.State = models.RegStateFullName
	draft.UpdatedAt = time.Now()
	err = s.storage.Registration().UpdateDraft(ctx, draft)
	if err != nil {
		return nil, err
	}

	return &RegistrationResult{
		Success:   true,
		NextState: models.RegStateFullName,
		Message:   "👤 Iltimos, to'liq ism-familiyangizni kiriting (pasportdagidek):\n\nMasalan: Abdullayev Abdulloh",
		Draft:     draft,
	}, nil
}

// ProcessFullName validates and saves the full name
func (s RegistrationService) ProcessFullName(ctx context.Context, userID int64, name string) (*RegistrationResult, error) {
	draft, err := s.storage.Registration().GetDraftByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Validate name
	validationErr := validation.ValidateFullName(name)
	if validationErr != nil {
		return &RegistrationResult{
			Success:      false,
			NextState:    models.RegStateFullName,
			ErrorMessage: validationErr.Message,
			Draft:        draft,
		}, nil
	}

	// Normalize and save
	normalizedName := validation.NormalizeFullName(name)
	draft.FullName = normalizedName

	// If we were editing from confirmation, go back to confirmation
	if draft.PreviousState == models.RegStateConfirm {
		draft.State = models.RegStateConfirm
		draft.PreviousState = models.RegStateIdle
	} else {
		draft.State = models.RegStatePhone
	}

	draft.UpdatedAt = time.Now()

	err = s.storage.Registration().UpdateDraft(ctx, draft)
	if err != nil {
		return nil, err
	}

	// Return appropriate next state and message
	if draft.State == models.RegStateConfirm {
		return &RegistrationResult{
			Success:   true,
			NextState: models.RegStateConfirm,
			Message:   "✅ O'zgartirildi",
			Draft:     draft,
		}, nil
	}

	return &RegistrationResult{
		Success:   true,
		NextState: models.RegStatePhone,
		Message:   "📱 Telefon raqamingizni yuboring:",
		Draft:     draft,
	}, nil
}

// ProcessPhone validates and saves the phone number from Telegram contact
func (s RegistrationService) ProcessPhone(ctx context.Context, userID int64, phone string) (*RegistrationResult, error) {
	draft, err := s.storage.Registration().GetDraftByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Validate phone
	validationErr := validation.ValidatePhone(phone)
	if validationErr != nil {
		return &RegistrationResult{
			Success:      false,
			NextState:    models.RegStatePhone,
			ErrorMessage: validationErr.Message,
			Draft:        draft,
		}, nil
	}

	// Normalize and save
	normalizedPhone := validation.NormalizePhone(phone)
	draft.Phone = normalizedPhone

	// If we were editing from confirmation, go back to confirmation
	if draft.PreviousState == models.RegStateConfirm {
		draft.State = models.RegStateConfirm
		draft.PreviousState = models.RegStateIdle
	} else {
		draft.State = models.RegStateAge
	}

	draft.UpdatedAt = time.Now()

	err = s.storage.Registration().UpdateDraft(ctx, draft)
	if err != nil {
		return nil, err
	}
	// Return appropriate next state and message
	if draft.State == models.RegStateConfirm {
		return &RegistrationResult{
			Success:   true,
			NextState: models.RegStateConfirm,
			Message:   "✅ O'zgartirildi",
			Draft:     draft,
		}, nil
	}

	return &RegistrationResult{
		Success:   true,
		NextState: models.RegStateAge,
		Message:   "🎂 Yoshingizni kiriting (faqat raqam):\n\nMasalan: 25",
		Draft:     draft,
	}, nil
}

// ProcessAge validates and saves the age
func (s RegistrationService) ProcessAge(ctx context.Context, userID int64, ageStr string) (*RegistrationResult, error) {
	draft, err := s.storage.Registration().GetDraftByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Validate age
	age, validationErr := validation.ValidateAge(ageStr)
	if validationErr != nil {
		return &RegistrationResult{
			Success:      false,
			NextState:    models.RegStateAge,
			ErrorMessage: validationErr.Message,
			Draft:        draft,
		}, nil
	}

	// Save
	draft.Age = age

	// If we were editing from confirmation, go back to confirmation
	if draft.PreviousState == models.RegStateConfirm {
		draft.State = models.RegStateConfirm
		draft.PreviousState = models.RegStateIdle
	} else {
		draft.State = models.RegStateBodyParams
	}

	draft.UpdatedAt = time.Now()

	err = s.storage.Registration().UpdateDraft(ctx, draft)
	if err != nil {
		return nil, err
	}

	// Return appropriate next state and message
	if draft.State == models.RegStateConfirm {
		return &RegistrationResult{
			Success:   true,
			NextState: models.RegStateConfirm,
			Message:   "✅ O'zgartirildi",
			Draft:     draft,
		}, nil
	}

	return &RegistrationResult{
		Success:   true,
		NextState: models.RegStateBodyParams,
		Message:   "📏 Vazningiz (kg) va bo'yingizni (sm) kiriting:\n\nMasalan: 70 175",
		Draft:     draft,
	}, nil
}

// ProcessBodyParams validates and saves weight and height
func (s RegistrationService) ProcessBodyParams(ctx context.Context, userID int64, input string) (*RegistrationResult, error) {
	draft, err := s.storage.Registration().GetDraftByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Validate and parse body params
	weight, height, validationErr := validation.ParseBodyParams(input)
	if validationErr != nil {
		return &RegistrationResult{
			Success:      false,
			NextState:    models.RegStateBodyParams,
			ErrorMessage: validationErr.Message,
			Draft:        draft,
		}, nil
	}

	// Save
	draft.Weight = weight
	draft.Height = height

	// Always go to confirmation after body params (skip passport photo)
	draft.State = models.RegStateConfirm
	if draft.PreviousState == models.RegStateConfirm {
		draft.PreviousState = models.RegStateIdle
	}

	draft.UpdatedAt = time.Now()

	err = s.storage.Registration().UpdateDraft(ctx, draft)
	if err != nil {
		return nil, err
	}

	// Always return confirmation state
	return &RegistrationResult{
		Success:   true,
		NextState: models.RegStateConfirm,
		Message:   "✅ Ma'lumotlar saqlandi",
		Draft:     draft,
	}, nil
}

// ProcessPassportPhoto saves the passport photo file ID
func (s RegistrationService) ProcessPassportPhoto(ctx context.Context, userID int64, fileID string) (*RegistrationResult, error) {
	draft, err := s.storage.Registration().GetDraftByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if fileID == "" {
		return &RegistrationResult{
			Success:      false,
			NextState:    models.RegStatePassportPhoto,
			ErrorMessage: "❌ Iltimos, pasport rasmini yuboring (faqat rasm formatida)",
			Draft:        draft,
		}, nil
	}

	// Save
	draft.PassportPhotoID = fileID

	// Always go to confirmation after photo (whether editing or first time)
	draft.State = models.RegStateConfirm
	draft.PreviousState = models.RegStateIdle
	draft.UpdatedAt = time.Now()

	err = s.storage.Registration().UpdateDraft(ctx, draft)
	if err != nil {
		return nil, err
	}

	return &RegistrationResult{
		Success:   true,
		NextState: models.RegStateConfirm,
		Message:   "✅ O'zgartirildi",
		Draft:     draft,
	}, nil
}

// FormatRegistrationSummary creates a summary of all registration data
func (s RegistrationService) FormatRegistrationSummary(draft *models.RegistrationDraft) string {
	var sb strings.Builder

	fmt.Fprintf(&sb, "📋 *Ro'yxatdan o'tish ma'lumotlari*\n\n")
	fmt.Fprintf(&sb, "👤 Ism-familiya: %s\n", draft.FullName)
	fmt.Fprintf(&sb, "📱 Telefon: %s\n", draft.Phone)
	fmt.Fprintf(&sb, "🎂 Yosh: %d\n", draft.Age)
	fmt.Fprintf(&sb, "⚖️ Vazn: %d kg\n", draft.Weight)
	fmt.Fprintf(&sb, "📏 Bo'y: %d sm\n", draft.Height)
	fmt.Fprintf(&sb, "Ma'lumotlar to'g'ri bo'lsa \"✅ Tasdiqlash\" tugmasini bosing.")

	return sb.String()
}

// ConfirmRegistration completes the registration
func (s RegistrationService) ConfirmRegistration(ctx context.Context, userID int64) (*RegistrationResult, error) {
	s.log.Info("!!!Confirm Registration", logger.Any("user_id", userID))

	draft, err := s.storage.Registration().GetDraftByUserID(ctx, userID)
	if err != nil {
		s.log.Error("???Failed to get draft for confirmation", logger.Error(err), logger.Any("user_id", userID))
		return nil, err
	}

	// Verify draft is complete
	if !draft.IsComplete() {
		s.log.Warn("Registration draft is incomplete", logger.Any("user_id", userID))
		return &RegistrationResult{
			Success:      false,
			NextState:    models.RegStateConfirm,
			ErrorMessage: "❌ Ma'lumotlar to'liq emas. Iltimos, barcha bosqichlarni to'ldiring.",
			Draft:        draft,
		}, nil
	}

	// Complete registration (moves from draft to registered_users)
	err = s.storage.Registration().CompleteRegistration(ctx, userID)
	if err != nil {
		s.log.Error("???Failed to complete registration", logger.Error(err), logger.Any("user_id", userID))
		return nil, err
	}

	s.log.Info("Registration completed successfully", logger.Any("user_id", userID))
	return &RegistrationResult{
		Success:   true,
		NextState: models.RegStateCompleted,
		Message:   messages.MsgRegistrationComplete,
		Draft:     nil,
	}, nil
}

// CancelRegistration cancels the registration and deletes the draft
func (s RegistrationService) CancelRegistration(ctx context.Context, userID int64) error {
	return s.storage.Registration().DeleteDraft(ctx, userID)
}

// GoToEditState sets the draft state to a specific field for editing
func (s RegistrationService) GoToEditState(ctx context.Context, userID int64, field models.EditField) (*RegistrationResult, error) {
	draft, err := s.storage.Registration().GetDraftByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	var nextState models.RegistrationState
	var message string

	switch field {
	case models.EditFieldFullName:
		nextState = models.RegStateFullName
		message = "✏️ Ism-familiyangizni qayta kiriting:"
	case models.EditFieldPhone:
		nextState = models.RegStatePhone
		message = "✏️ Telefon raqamingizni qayta yuboring:"
	case models.EditFieldAge:
		nextState = models.RegStateAge
		message = "✏️ Yoshingizni qayta kiriting:"
	case models.EditFieldBodyParams:
		nextState = models.RegStateBodyParams
		message = "✏️ Vazn va bo'yingizni qayta kiriting (masalan: 70 175):"
	default:
		return nil, fmt.Errorf("unknown edit field: %s", field)
	}

	// Mark that we're in edit mode by storing previous state
	draft.PreviousState = draft.State
	draft.State = nextState
	draft.UpdatedAt = time.Now()

	err = s.storage.Registration().UpdateDraft(ctx, draft)
	if err != nil {
		return nil, err
	}

	return &RegistrationResult{
		Success:   true,
		NextState: nextState,
		Message:   message,
		Draft:     draft,
	}, nil
}

// RestartRegistration deletes the current draft and starts fresh
func (s RegistrationService) RestartRegistration(ctx context.Context, userID int64) (*models.RegistrationDraft, error) {
	// Delete existing draft
	_ = s.storage.Registration().DeleteDraft(ctx, userID)

	// Start fresh
	return s.StartRegistration(ctx, userID)
}

// GetRegisteredUser returns the full registered user data
func (s RegistrationService) GetRegisteredUser(ctx context.Context, userID int64) (*models.RegisteredUser, error) {
	return s.storage.Registration().GetRegisteredUserByUserID(ctx, userID)
}
