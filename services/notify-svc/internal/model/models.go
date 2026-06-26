package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type NotificationChannel struct {
	ID          uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	OrgID       uuid.UUID `json:"org_id" gorm:"type:uuid;not null;index"`
	Name        string    `json:"name" gorm:"size:200"`
	ChannelType string    `json:"channel_type" gorm:"size:20;not null"`
	Config      string    `json:"config" gorm:"type:text"`
	IsEnabled   bool      `json:"is_enabled" gorm:"default:true"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (c *NotificationChannel) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
func (NotificationChannel) TableName() string { return "notification_channels" }

type NotificationTemplate struct {
	ID            uuid.UUID `json:"id" gorm:"type:uuid;primary_key"`
	ChannelType   string    `json:"channel_type" gorm:"size:20;not null"`
	Name          string    `json:"name" gorm:"size:200"`
	TitleTemplate string    `json:"title_template" gorm:"type:text"`
	BodyTemplate  string    `json:"body_template" gorm:"type:text"`
	CreatedAt     time.Time `json:"created_at"`
}

func (t *NotificationTemplate) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}
func (NotificationTemplate) TableName() string { return "notification_templates" }

type NotificationLog struct {
	ID        uuid.UUID  `json:"id" gorm:"type:uuid;primary_key"`
	ChannelID *uuid.UUID `json:"channel_id"`
	EventType string     `json:"event_type" gorm:"size:100"`
	Recipient string     `json:"recipient" gorm:"size:500"`
	Title     string     `json:"title" gorm:"size:500"`
	Status    string     `json:"status" gorm:"size:20"`
	ErrorMsg  string     `json:"error_msg" gorm:"type:text"`
	CreatedAt time.Time  `json:"created_at"`
}

func (l *NotificationLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}
func (NotificationLog) TableName() string { return "notification_logs" }
