package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuditLog struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key"`
	OrgID        uuid.UUID `gorm:"type:uuid;index"`
	UserID       uuid.UUID `gorm:"type:uuid;index"`
	Username     string    `gorm:"size:100"`
	EventType    string    `gorm:"size:100;not null;index"`
	Action       string    `gorm:"size:50"`
	ResourceType string    `gorm:"size:50"`
	ResourceID   string    `gorm:"size:100"`
	Detail       string    `gorm:"type:text"`
	IP           string    `gorm:"size:50"`
	UserAgent    string    `gorm:"size:500"`
	SessionID    string    `gorm:"size:100;index"`
	ReasonCode   string    `gorm:"size:100;index"`
	RequestID    string    `gorm:"size:100;index"`
	CreatedAt    time.Time `gorm:"index"`
}

func (l *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}
func (AuditLog) TableName() string { return "audit_logs" }
