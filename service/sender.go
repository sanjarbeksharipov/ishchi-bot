package service

import (
	"context"
	"fmt"
	"sync"

	"telegram-bot-starter/bot/models"
	"telegram-bot-starter/config"
	"telegram-bot-starter/pkg/keyboards"
	"telegram-bot-starter/pkg/logger"
	"telegram-bot-starter/pkg/messages"
	"telegram-bot-starter/storage"

	tele "gopkg.in/telebot.v4"
)

// MessageRequest represents a message to be sent
type MessageRequest struct {
	ChatID    int64
	Message   string
	Options   []any // ReplyMarkup, ParseMode, etc.
	Photo     *tele.Photo
	IsEdit    bool
	MessageID int // For editing existing messages
}

// MessageResponse represents the result of sending a message
type MessageResponse struct {
	Success   bool
	MessageID int
	Error     error
}

// SenderService handles all message sending operations
// This centralizes message sending for future queue implementation
type SenderService struct {
	cfg     config.Config
	log     logger.LoggerI
	bot     *tele.Bot
	service ServiceManagerI
	storage storage.StorageI
	mu      sync.Mutex

	// Queue settings (for future implementation)
	useQueue bool
	// queue    chan *MessageRequest
}

// NewSenderService creates a new sender service
func NewSenderService(cfg config.Config, log logger.LoggerI, bot *tele.Bot, storage storage.StorageI, service ServiceManagerI) *SenderService {
	return &SenderService{
		cfg:      cfg,
		log:      log,
		bot:      bot,
		storage:  storage,
		service:  service,
		useQueue: false, // Will be enabled when queue is implemented
	}
}

// Send sends a message to a user
func (s *SenderService) Send(ctx context.Context, chatID int64, message string, opts ...any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// TODO: When queue is implemented, add to queue instead
	// if s.useQueue {
	//     return s.enqueue(&MessageRequest{ChatID: chatID, Message: message, Options: opts})
	// }

	chat := &tele.Chat{ID: chatID}
	_, err := s.bot.Send(chat, message, opts...)
	if err != nil {
		s.log.Error("Failed to send message", logger.Error(err), logger.Any("chat_id", chatID))
		return err
	}

	return nil
}

// SendPhoto sends a photo to a user
func (s *SenderService) SendPhoto(ctx context.Context, chatID int64, photo *tele.Photo, opts ...any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	chat := &tele.Chat{ID: chatID}
	_, err := s.bot.Send(chat, photo, opts...)
	if err != nil {
		s.log.Error("Failed to send photo", logger.Error(err), logger.Any("chat_id", chatID))
		return err
	}

	return nil
}

// Edit edits an existing message
func (s *SenderService) Edit(ctx context.Context, chatID int64, messageID int, message string, opts ...any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg := &tele.Message{
		ID:   messageID,
		Chat: &tele.Chat{ID: chatID},
	}

	_, err := s.bot.Edit(msg, message, opts...)
	if err != nil {
		s.log.Error("Failed to edit message", logger.Error(err), logger.Any("chat_id", chatID))
		return err
	}

	return nil
}

// Reply sends a reply using telebot context (for immediate responses)
func (s *SenderService) Reply(c tele.Context, message string, opts ...any) error {
	// For immediate context-based replies, we don't need queue
	// This is used for user-initiated actions where immediate response is expected
	return c.Send(message, opts...)
}

// ReplyWithPhoto sends a photo reply using telebot context
func (s *SenderService) ReplyWithPhoto(c tele.Context, photo *tele.Photo, opts ...any) error {
	return c.Send(photo, opts...)
}

// EditMessage edits the message in callback context
func (s *SenderService) EditMessage(c tele.Context, message string, opts ...any) error {
	return c.Edit(message, opts...)
}

// Respond responds to a callback query
func (s *SenderService) Respond(c tele.Context, response *tele.CallbackResponse) error {
	return c.Respond(response)
}

// RemoveKeyboard removes reply keyboard by sending a message with RemoveKeyboard option
func (s *SenderService) RemoveKeyboard(c tele.Context) error {
	return c.Send("\u200B", &tele.ReplyMarkup{RemoveKeyboard: true})
}

// DeleteMessage deletes the message in callback context
func (s *SenderService) DeleteMessage(c tele.Context) error {
	return c.Delete()
}

// UpdateChannelJobPost updates a job post in the channel with latest info
func (s *SenderService) UpdateChannelJobPost(ctx context.Context, job *models.Job) error {
	if job.ChannelMessageID == 0 {
		s.log.Warn("Cannot update channel message: no channel message ID", logger.Any("job_id", job.ID))
		return fmt.Errorf("no channel message ID for job %d", job.ID)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	msg := &tele.Message{
		ID:   int(job.ChannelMessageID),
		Chat: &tele.Chat{ID: s.cfg.Bot.ChannelID},
	}

	channelMsg := messages.FormatJobForChannel(job)

	var err error
	// Only show signup button if job is ACTIVE
	if job.Status == models.JobStatusActive {
		signupBtn := keyboards.JobSignupKeyboard(job.ID, s.cfg.Bot.Username)
		_, err = s.bot.Edit(msg, channelMsg, signupBtn, tele.ModeHTML)
	} else {
		// Remove buttons for non-active jobs (FULL, COMPLETED, CANCELLED, DRAFT)
		_, err = s.bot.Edit(msg, channelMsg, nil, tele.ModeHTML)
	}

	if err != nil {
		s.log.Error("Failed to update channel message",
			logger.Error(err),
			logger.Any("job_id", job.ID),
			logger.Any("channel_message_id", job.ChannelMessageID),
		)
		return fmt.Errorf("failed to update channel message: %w", err)
	}

	s.log.Info("Channel message updated successfully",
		logger.Any("job_id", job.ID),
		logger.Any("confirmed_slots", job.ConfirmedSlots),
		logger.Any("required_workers", job.RequiredWorkers),
		logger.Any("status", job.Status),
	)

	return nil
}

// UpdateAdminJobPost updates all admin job detail messages (broadcasts to all admins)
func (s *SenderService) UpdateAdminJobPost(ctx context.Context, job *models.Job) error {
	// Get all admin messages for this job
	adminMessages, err := s.storage.AdminMessage().GetAllByJobID(ctx, job.ID)
	if err != nil {
		s.log.Error("Failed to get admin messages",
			logger.Error(err),
			logger.Any("job_id", job.ID))
		return fmt.Errorf("failed to get admin messages: %w", err)
	}

	if len(adminMessages) == 0 {
		s.log.Debug("No admin messages to update", logger.Any("job_id", job.ID))
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	adminMsg := messages.FormatJobDetailAdmin(job)
	adminKeyboard := keyboards.JobDetailKeyboard(job)

	// Update each admin's message
	for _, adminMessage := range adminMessages {
		msg := &tele.Message{
			ID:   int(adminMessage.MessageID),
			Chat: &tele.Chat{ID: adminMessage.AdminID},
		}

		_, err := s.bot.Edit(msg, adminMsg, adminKeyboard, tele.ModeHTML)
		if err != nil {
			s.log.Error("Failed to update admin message",
				logger.Error(err),
				logger.Any("job_id", job.ID),
				logger.Any("admin_id", adminMessage.AdminID),
				logger.Any("message_id", adminMessage.MessageID),
			)
			// If message not found, remove from database
			if err.Error() == "telegram: message not found (400)" ||
				err.Error() == "telegram: message to edit not found (400)" {
				s.storage.AdminMessage().Delete(ctx, job.ID, adminMessage.AdminID)
			}
			continue
		}

		s.log.Debug("Admin message updated successfully",
			logger.Any("job_id", job.ID),
			logger.Any("admin_id", adminMessage.AdminID),
		)
	}

	s.log.Info("All admin messages updated successfully",
		logger.Any("job_id", job.ID),
		logger.Any("confirmed_slots", job.ConfirmedSlots),
		logger.Any("required_workers", job.RequiredWorkers),
		logger.Any("status", job.Status),
		logger.Any("admins_notified", len(adminMessages)),
	)

	return nil
}

// ============ Queue Implementation (Future) ============

// EnableQueue enables queue-based message sending
// func (s *SenderService) EnableQueue(bufferSize int) {
//     s.mu.Lock()
//     defer s.mu.Unlock()
//
//     s.useQueue = true
//     s.queue = make(chan *MessageRequest, bufferSize)
//
//     // Start queue processor
//     go s.processQueue()
// }

// processQueue processes messages from the queue with rate limiting
// func (s *SenderService) processQueue() {
//     // Telegram limit: 30 messages per second to different chats
//     // 1 message per second to same chat
//     ticker := time.NewTicker(35 * time.Millisecond) // ~28 messages per second
//     defer ticker.Stop()
//
//     for req := range s.queue {
//         <-ticker.C
//         s.sendDirect(req)
//     }
// }

// enqueue adds a message request to the queue
// func (s *SenderService) enqueue(req *MessageRequest) error {
//     select {
//     case s.queue <- req:
//         return nil
//     default:
//         return errors.New("message queue is full")
//     }
// }
