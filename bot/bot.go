package bot

import (
	"fmt"
	"log"
	"strings"

	"github.com/Anwarjondev/telegram-announcement-bot/config"
	"github.com/Anwarjondev/telegram-announcement-bot/models"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

type Bot struct {
	api    *tgbotapi.BotAPI
	config *config.Config
	db     *gorm.DB
}

func NewBot(cfg *config.Config, db *gorm.DB) (*Bot, error) {
	if cfg.TelegramToken == "" {
		return nil, fmt.Errorf("telegram token is empty")
	}

	api, err := tgbotapi.NewBotAPI(cfg.TelegramToken)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot API: %w", err)
	}

	// Test the bot token by getting bot info
	_, err = api.GetMe()
	if err != nil {
		return nil, fmt.Errorf("failed to get bot info: %w", err)
	}

	return &Bot{
		api:    api,
		config: cfg,
		db:     db,
	}, nil
}

// GetAPI returns the Telegram Bot API instance
func (b *Bot) GetAPI() *tgbotapi.BotAPI {
	return b.api
}

func (b *Bot) Start() {
	log.Printf("Starting bot...")
	log.Printf("Bot username: %s", b.api.Self.UserName)
	log.Printf("Admin username: %s", b.config.AdminUsername)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := b.api.GetUpdatesChan(u)
	log.Printf("Waiting for updates...")

	for update := range updates {
		if update.ChannelPost != nil {
			log.Printf("Received channel post")
			b.handleChannelMessage(update.ChannelPost)
		} else if update.Message != nil {
			log.Printf("Received message update")
			if update.Message.IsCommand() {
				b.handleCommand(update.Message)
			} else if update.Message.Chat.Type == "channel" {
				log.Printf("Received channel message")
				b.handleChannelMessage(update.Message)
			} else {
				// Handle user messages
				b.handleUserMessage(update.Message)
			}
		}
	}
}

func (b *Bot) handleCommand(message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		msg := tgbotapi.NewMessage(message.Chat.ID, "Welcome! I'm an announcement bot. Send me any message and I'll forward it to all connected channels.")
		b.api.Send(msg)
	case "help":
		msg := tgbotapi.NewMessage(message.Chat.ID, "Available commands:\n/start - Start the bot\n/help - Show this help message\n\nTo send an announcement, simply send me any message and I'll forward it to all connected channels.")
		b.api.Send(msg)
	}
}

func (b *Bot) handleChannelMessage(message *tgbotapi.Message) {
	// Debug logging
	log.Printf("=== New Message Received ===")
	log.Printf("Chat Type: %s", message.Chat.Type)
	log.Printf("Chat ID: %d", message.Chat.ID)
	log.Printf("Chat Title: %s", message.Chat.Title)
	log.Printf("Message Text: %s", message.Text)

	// For channel posts, we need to check if the bot is an admin
	// and if the channel is in our database
	var channel models.Channel
	if err := b.db.Where("channel_id = ?", message.Chat.ID).First(&channel).Error; err != nil {
		log.Printf("Channel %d not found in database", message.Chat.ID)
		return
	}

	log.Printf("Channel found in database: %s", channel.ChannelName)

	// Store the announcement
	announcement := models.Announcement{
		MessageID:   int64(message.MessageID),
		ChannelID:   message.Chat.ID,
		Text:        message.Text,
		PostedBy:    "channel", // Channel posts don't have a sender
		PostedAt:    message.Time(),
		IsPublished: false,
	}

	if err := b.db.Create(&announcement).Error; err != nil {
		log.Printf("Error storing announcement: %v", err)
		return
	}

	log.Printf("Announcement stored in database")

	// Forward to all active channels
	var channels []models.Channel
	if err := b.db.Where("is_active = ?", true).Find(&channels).Error; err != nil {
		log.Printf("Error fetching channels: %v", err)
		return
	}

	log.Printf("Found %d active channels to forward to", len(channels))

	successCount := 0
	failedChannels := []string{}

	for _, targetChannel := range channels {
		if targetChannel.ChannelID != message.Chat.ID { // Don't forward to the source channel
			log.Printf("Attempting to forward to channel %d (%s)", targetChannel.ChannelID, targetChannel.ChannelName)

			// Verify channel access before sending
			_, err := b.api.GetChat(tgbotapi.ChatInfoConfig{
				ChatConfig: tgbotapi.ChatConfig{
					ChatID: targetChannel.ChannelID,
				},
			})

			if err != nil {
				log.Printf("Error verifying channel %s (%d): %v", targetChannel.ChannelName, targetChannel.ChannelID, err)
				failedChannels = append(failedChannels, targetChannel.ChannelName)
				continue
			}

			// Check if bot is an admin in the channel
			member, err := b.api.GetChatMember(tgbotapi.GetChatMemberConfig{
				ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
					ChatID: targetChannel.ChannelID,
					UserID: b.api.Self.ID,
				},
			})

			if err != nil || !member.IsAdministrator() && !member.IsCreator() {
				log.Printf("Bot is not an admin in channel %s (%d)", targetChannel.ChannelName, targetChannel.ChannelID)
				failedChannels = append(failedChannels, targetChannel.ChannelName)
				continue
			}

			msg := tgbotapi.NewMessage(targetChannel.ChannelID, message.Text)
			if _, err := b.api.Send(msg); err != nil {
				log.Printf("Error forwarding message to channel %d: %v", targetChannel.ChannelID, err)
				failedChannels = append(failedChannels, targetChannel.ChannelName)
			} else {
				successCount++
				log.Printf("Successfully forwarded message to channel %d", targetChannel.ChannelID)
			}
			if len(channels) == 0 {
				msg := tgbotapi.NewMessage(message.Chat.ID, "No active channels found. Please add some channels first.")
				b.api.Send(msg)
				return
			}

			successCount := 0
			failedChannels := []string{}

			for _, channel := range channels {
				// Verify channel access before sending
				// Log the channel ID being used for verification
				log.Printf("Attempting to verify channel with ID: %d and Name: %s", channel.ChannelID, channel.ChannelName)

				_, err := b.api.GetChat(tgbotapi.ChatInfoConfig{
					ChatConfig: tgbotapi.ChatConfig{
						ChatID: channel.ChannelID,
					},
				})

				if err != nil {
					log.Printf("Error verifying channel %s (%d): %v", channel.ChannelName, channel.ChannelID, err)
					failedChannels = append(failedChannels, channel.ChannelName)
					continue
				}

				// Check if bot is an admin in the channel
				member, err := b.api.GetChatMember(tgbotapi.GetChatMemberConfig{
					ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
						ChatID: channel.ChannelID,
						UserID: b.api.Self.ID,
					},
				})

				if err != nil || !member.IsAdministrator() && !member.IsCreator() {
					log.Printf("Bot is not an admin in channel %s (%d)", channel.ChannelName, channel.ChannelID)
					failedChannels = append(failedChannels, channel.ChannelName)
					continue
				}

				msg := tgbotapi.NewMessage(channel.ChannelID, message.Text)
				// Log the channel ID and message text before sending
				log.Printf("Attempting to send message to channel with ID: %d. Message text snippet: \"%s...\"", channel.ChannelID, message.Text[:min(len(message.Text), 50)])

				if _, err := b.api.Send(msg); err != nil {
					log.Printf("Error forwarding message to channel %s (%d): %v", channel.ChannelName, channel.ChannelID, err)
					failedChannels = append(failedChannels, channel.ChannelName)
				} else {
					successCount++
				}
			}
		}
	}

	// Mark announcement as published
	if err := b.db.Model(&announcement).Update("is_published", true).Error; err != nil {
		log.Printf("Error marking announcement as published: %v", err)
	}

	// Log summary
	if len(failedChannels) > 0 {
		log.Printf("Failed to forward to channels: %v", failedChannels)
	}
	log.Printf("Successfully forwarded to %d out of %d channels", successCount, len(channels)-1)
}

func (b *Bot) handleUserMessage(message *tgbotapi.Message) {
	// Check if the user is authorized
	if message.From.UserName != b.config.AdminUsername {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Sorry, you are not authorized to send announcements.")
		b.api.Send(msg)
		return
	}

	// Validate message
	if strings.TrimSpace(message.Text) == "" {
		msg := tgbotapi.NewMessage(message.Chat.ID, "Please provide a non-empty message for the announcement.")
		b.api.Send(msg)
		return
	}

	// Store the announcement
	announcement := models.Announcement{
		MessageID:   int64(message.MessageID),
		ChannelID:   message.Chat.ID,
		Text:        message.Text,
		PostedBy:    message.From.UserName,
		PostedAt:    message.Time(),
		IsPublished: false,
	}

	if err := b.db.Create(&announcement).Error; err != nil {
		log.Printf("Error storing announcement: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Error storing your announcement. Please try again.")
		b.api.Send(msg)
		return
	}

	// Forward to all active channels
	var channels []models.Channel
	if err := b.db.Where("is_active = ?", true).Find(&channels).Error; err != nil {
		log.Printf("Error fetching channels: %v", err)
		msg := tgbotapi.NewMessage(message.Chat.ID, "Error fetching channels. Please try again.")
		b.api.Send(msg)
		return
	}

	if len(channels) == 0 {
		msg := tgbotapi.NewMessage(message.Chat.ID, "No active channels found. Please add some channels first.")
		b.api.Send(msg)
		return
	}

	successCount := 0
	failedChannels := []string{}

	for _, channel := range channels {
		// Verify channel access before sending
		_, err := b.api.GetChat(tgbotapi.ChatInfoConfig{
			ChatConfig: tgbotapi.ChatConfig{
				ChatID: channel.ChannelID,
			},
		})

		if err != nil {
			log.Printf("Error verifying channel %s (%d): %v", channel.ChannelName, channel.ChannelID, err)
			failedChannels = append(failedChannels, channel.ChannelName)
			continue
		}

		// Check if bot is an admin in the channel
		member, err := b.api.GetChatMember(tgbotapi.GetChatMemberConfig{
			ChatConfigWithUser: tgbotapi.ChatConfigWithUser{
				ChatID: channel.ChannelID,
				UserID: b.api.Self.ID,
			},
		})

		if err != nil || !member.IsAdministrator() && !member.IsCreator() {
			log.Printf("Bot is not an admin in channel %s (%d)", channel.ChannelName, channel.ChannelID)
			failedChannels = append(failedChannels, channel.ChannelName)
			continue
		}

		msg := tgbotapi.NewMessage(channel.ChannelID, message.Text)
		if _, err := b.api.Send(msg); err != nil {
			log.Printf("Error forwarding message to channel %s (%d): %v", channel.ChannelName, channel.ChannelID, err)
			failedChannels = append(failedChannels, channel.ChannelName)
		} else {
			successCount++
		}
	}

	// Mark announcement as published
	if err := b.db.Model(&announcement).Update("is_published", true).Error; err != nil {
		log.Printf("Error marking announcement as published: %v", err)
	}

	// Prepare response message
	var response strings.Builder
	response.WriteString(fmt.Sprintf("Your announcement has been sent to %d out of %d channels.\n", successCount, len(channels)))

	if len(failedChannels) > 0 {
		response.WriteString("\nFailed to send to the following channels:\n")
		for _, channel := range failedChannels {
			response.WriteString(fmt.Sprintf("- %s\n", channel))
		}
		response.WriteString("\nPlease make sure the bot is an admin in these channels and has permission to post messages.")
	}

	msg := tgbotapi.NewMessage(message.Chat.ID, response.String())
	b.api.Send(msg)
}
