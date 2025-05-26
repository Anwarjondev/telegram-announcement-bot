package web

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/Anwarjondev/telegram-announcement-bot/config"
	"github.com/Anwarjondev/telegram-announcement-bot/models"
	"gorm.io/gorm"
)

type Server struct {
	router *gin.Engine
	db     *gorm.DB
	config *config.Config
}

func NewServer(db *gorm.DB, cfg *config.Config) *Server {
	router := gin.Default()
	server := &Server{
		router: router,
		db:     db,
		config: cfg,
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
	channelIDStr := c.PostForm("channel_id")
	channelName := c.PostForm("channel_name")

	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid channel ID"})
		return
	}

	channel := models.Channel{
		ChannelID:   channelID,
		ChannelName: channelName,
		IsActive:    true,
	}

	if err := s.db.Create(&channel).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.Redirect(http.StatusFound, "/channels")
}

func (s *Server) handleRemoveChannel(c *gin.Context) {
	id := c.Param("id")
	s.db.Delete(&models.Channel{}, id)
	c.Redirect(http.StatusFound, "/channels")
}
