package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Organization struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key"`
	Name        string    `gorm:"size:200;not null"`
	Code        string    `gorm:"size:100;uniqueIndex;not null"`
	Description string    `gorm:"type:text"`
	Logo        string    `gorm:"size:500"`
	Status      string    `gorm:"size:20;default:active"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (o *Organization) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

func (Organization) TableName() string { return "organizations" }

type User struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primary_key"`
	OrgID               uuid.UUID  `gorm:"type:uuid;not null;index"`
	Username            string     `gorm:"size:100;not null;uniqueIndex:idx_org_username"`
	Email               string     `gorm:"size:200"`
	Phone               string     `gorm:"size:30"`
	PasswordHash        string     `gorm:"size:255" json:"-"`
	DisplayName         string     `gorm:"size:200"`
	Avatar              string     `gorm:"size:500"`
	Status              string     `gorm:"size:20;default:active"`
	Source              string     `gorm:"size:20;default:local"`
	FailedLoginAttempts int        `gorm:"default:0" json:"failed_login_attempts"`
	LockedUntil         *time.Time `json:"locked_until"`
	PasswordChangedAt   *time.Time `json:"password_changed_at"`
	MustChangePassword  bool       `gorm:"default:false" json:"must_change_password"`
	MFAEnabled          bool       `gorm:"default:false" json:"mfa_enabled"`
	MFASecret           string     `gorm:"size:500" json:"-"`
	MFAConfirmedAt      *time.Time `json:"mfa_confirmed_at"`
	LastLoginAt         *time.Time
	DeletedAt           *time.Time `json:"deleted_at"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

func (User) TableName() string { return "users" }

type UserCredential struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	UserID         uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	OrgID          uuid.UUID `gorm:"type:uuid;not null;index" json:"org_id"`
	Provider       string    `gorm:"size:30;not null;uniqueIndex:idx_org_provider_subject" json:"provider"`
	ProviderUserID string    `gorm:"size:255;not null;uniqueIndex:idx_org_provider_subject" json:"provider_user_id"`
	Username       string    `gorm:"size:255" json:"username"`
	Email          string    `gorm:"size:255" json:"email"`
	RawProfile     string    `gorm:"type:text" json:"raw_profile"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (c *UserCredential) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

func (UserCredential) TableName() string { return "user_credentials" }

type PasswordHistory struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key" json:"id"`
	UserID       uuid.UUID `gorm:"type:uuid;not null;index" json:"user_id"`
	PasswordHash string    `gorm:"size:255;not null" json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

func (h *PasswordHistory) BeforeCreate(tx *gorm.DB) error {
	if h.ID == uuid.Nil {
		h.ID = uuid.New()
	}
	return nil
}

func (PasswordHistory) TableName() string { return "password_histories" }

type MFARecoveryCode struct {
	ID        uuid.UUID  `gorm:"type:uuid;primary_key" json:"id"`
	UserID    uuid.UUID  `gorm:"type:uuid;not null;index" json:"user_id"`
	CodeHash  string     `gorm:"size:64;not null;index" json:"-"`
	UsedAt    *time.Time `json:"used_at"`
	CreatedAt time.Time  `json:"created_at"`
}

func (c *MFARecoveryCode) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}

func (MFARecoveryCode) TableName() string { return "mfa_recovery_codes" }
