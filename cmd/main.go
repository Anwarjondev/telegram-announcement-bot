package main

import (
	"fmt"
	"log"

	"github.com/Anwarjondev/telegram-announcement-bot/bot"
	"github.com/Anwarjondev/telegram-announcement-bot/config"
	"github.com/Anwarjondev/telegram-announcement-bot/models"
	"github.com/Anwarjondev/telegram-announcement-bot/web"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Validate configuration
	if cfg.TelegramToken == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN environment variable is not set")
	}
	if cfg.AdminUsername == "" {
		log.Fatal("ADMIN_USERNAME environment variable is not set")
	}

	fmt.Printf("Starting bot with token: %s...\n", cfg.TelegramToken[:10]+"...")

	// Initialize database
	db, err := gorm.Open(postgres.Open(cfg.GetDSN()), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto-migrate the schema
	db.AutoMigrate(&models.Channel{}, &models.Announcement{})

	// Initialize bot
	bot, err := bot.NewBot(cfg, db)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	// Initialize web server
	server := web.NewServer(db, cfg)

	// Start bot in a goroutine
	go func() {
		log.Println("Starting bot...")
		bot.Start()
	}()

	// Start web server
	log.Printf("Starting web server on port %s...\n", cfg.WebPort)
	if err := server.Start(); err != nil {
		log.Fatal("Failed to start web server:", err)
	}
}
