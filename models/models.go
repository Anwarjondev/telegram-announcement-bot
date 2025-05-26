package models

import (
	"time"

	"gorm.io/gorm"
)

type Channel struct {
	gorm.Model
	ChannelID   int64 `gorm:"uniqueIndex"`
	ChannelName string
	AddedBy     string
	IsActive    bool `gorm:"default:true"`
}

type Announcement struct {
	gorm.Model
	MessageID   int64
	ChannelID   int64
	Text        string
	PostedBy    string
	PostedAt    time.Time
	IsPublished bool `gorm:"default:false"`
}
