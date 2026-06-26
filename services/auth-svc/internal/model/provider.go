package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AuthProvider struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key"`
	OrgID     uuid.UUID `gorm:"type:uuid;not null;index"`
	Provider  string    `gorm:"size:20;not null"`
	Name      string    `gorm:"size:100"`
	Config    string    `gorm:"type:text"`
	IsEnabled bool      `gorm:"default:true"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (p *AuthProvider) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}

func (AuthProvider) TableName() string { return "auth_providers" }

type RefreshToken struct {
	ID            uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	UserID        uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	SessionID     uuid.UUID  `gorm:"type:uuid;index" json:"session_id"`
	JTI           string     `gorm:"size:100;index" json:"jti"`
	TokenHash     string     `gorm:"size:255;not null;uniqueIndex" json:"-"`
	ExpiresAt     time.Time  `gorm:"not null" json:"expires_at"`
	Revoked       bool       `gorm:"default:false" json:"revoked"`
	RevokedAt     *time.Time `json:"revoked_at"`
	RevokedReason string     `gorm:"size:200" json:"revoked_reason"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (t *RefreshToken) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

func (RefreshToken) TableName() string { return "refresh_tokens" }

type Session struct {
	ID            uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	UserID        uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	OrgID         uuid.UUID  `gorm:"type:uuid;not null;index" json:"org_id"`
	Status        string     `gorm:"size:20;default:active;index" json:"status"`
	IP            string     `gorm:"size:64" json:"ip"`
	UserAgent     string     `gorm:"type:text" json:"user_agent"`
	DeviceName    string     `gorm:"size:200" json:"device_name"`
	LastSeenAt    time.Time  `json:"last_seen_at"`
	ExpiresAt     time.Time  `gorm:"not null;index" json:"expires_at"`
	RevokedAt     *time.Time `json:"revoked_at"`
	RevokedReason string     `gorm:"size:200" json:"revoked_reason"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

func (s *Session) BeforeCreate(tx *gorm.DB) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	if s.Status == "" {
		s.Status = "active"
	}
	if s.LastSeenAt.IsZero() {
		s.LastSeenAt = time.Now()
	}
	return nil
}

func (Session) TableName() string { return "sessions" }
