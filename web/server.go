package web

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/Anwarjondev/telegram-announcement-bot/config"
	"github.com/Anwarjondev/telegram-announcement-bot/models"
	"github.com/gin-gonic/gin"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"gorm.io/gorm"
)

type Server struct {
	router *gin.Engine
	db     *gorm.DB
	config *config.Config
	api    *tgbotapi.BotAPI
}

func NewServer(db *gorm.DB, cfg *config.Config, api *tgbotapi.BotAPI) *Server {
	router := gin.Default()
	server := &Server{
		router: router,
		db:     db,
		config: cfg,
		api:    api,
	}

	// Setup routes
	router.LoadHTMLGlob("web/templates/*")
	router.Static("/static", "web/static")

	router.GET("/", server.handleHome)
	router.GET("/channels", server.handleChannels)
	router.GET("/announcements", server.handleAnnouncements)
	router.POST("/channels/add", server.handleAddChannel)
	router.POST("/channels/remove/:id", server.handleRemoveChannel)

	return server
}

func (s *Server) Start() error {
	return s.router.Run(":" + s.config.WebPort)
}

func (s *Server) handleHome(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{
		"title": "Telegram Announcement Bot Admin",
	})
}

func (s *Server) handleChannels(c *gin.Context) {
	var channels []models.Channel
	s.db.Find(&channels)

	c.HTML(http.StatusOK, "channels.html", gin.H{
		"channels": channels,
	})
}

func (s *Server) handleAnnouncements(c *gin.Context) {
	var announcements []models.Announcement
	s.db.Order("created_at desc").Find(&announcements)

	c.HTML(http.StatusOK, "announcements.html", gin.H{
		"announcements": announcements,
	})
}

func (s *Server) handleAddChannel(c *gin.Context) {
	channelIdentifier := c.PostForm("channel_identifier")
	channelName := c.PostForm("channel_name")

	// Prepend -100 to the entered ID as required by Telegram API for channels/supergroups
	fullChannelIDStr := "-100" + channelIdentifier

	// Parse the full channel ID string into an int64
	chatID, err := strconv.ParseInt(fullChannelIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Invalid channel ID format: %v", err)})
		return
	}

	// Get chat info using the full numerical ID to validate the ID and bot's access
	// We don't need the 'chat' object itself for this simplified flow, just the validation from the API call.
	_, err = s.api.GetChat(tgbotapi.ChatInfoConfig{ChatConfig: tgbotapi.ChatConfig{ChatID: chatID}})

	// Handle API errors. GetChat will return an error if the ID is invalid or the bot is not in the chat.
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Failed to get chat information for ID \"%s\": %v. Ensure the bot is added to the channel/supergroup and the ID is correct.", channelIdentifier, err)})
		return
	}

	channel := models.Channel{
		ChannelID:   chatID,
		ChannelName: channelName,
		IsActive:    true,
	}

	// Check if the channel already exists in the database
	var existingChannel models.Channel
	if s.db.Where("channel_id = ?", chatID).First(&existingChannel).Error == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Channel with ID %d already exists", chatID)})
		return
	}

	if err := s.db.Create(&channel).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": fmt.Sprintf("Error saving channel to database: %v", err)})
		return
	}

	c.Redirect(http.StatusFound, "/channels")
}

func (s *Server) handleRemoveChannel(c *gin.Context) {
	id := c.Param("id")
	// Use Unscoped() to perform a hard delete instead of a soft delete
	s.db.Unscoped().Delete(&models.Channel{}, id)
	c.Redirect(http.StatusFound, "/channels")
}
