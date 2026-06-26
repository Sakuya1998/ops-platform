package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Role struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key"`
	OrgID       uuid.UUID `gorm:"type:uuid;not null;index"`
	Name        string    `gorm:"size:200;not null"`
	Code        string    `gorm:"size:100;not null;uniqueIndex:idx_org_role"`
	Description string    `gorm:"type:text"`
	IsSystem    bool      `gorm:"default:false"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (r *Role) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
func (Role) TableName() string { return "roles" }

type Permission struct {
	ID        uuid.UUID     `gorm:"type:uuid;primary_key"`
	ParentID  *uuid.UUID    `gorm:"type:uuid;index"`
	Name      string        `gorm:"size:200;not null"`
	Code      string        `gorm:"size:100;uniqueIndex;not null"`
	Resource  string        `gorm:"size:100;not null"`
	Action    string        `gorm:"size:50;not null"`
	Type      string        `gorm:"size:20;default:api"`
	Sort      int           `gorm:"default:0"`
	Children  []*Permission `gorm:"-"`
	CreatedAt time.Time
}

func (p *Permission) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
func (Permission) TableName() string { return "permissions" }

type UserRole struct {
	UserID    uuid.UUID `gorm:"type:uuid;primary_key;column:user_id"`
	RoleID    uuid.UUID `gorm:"type:uuid;primary_key"`
	CreatedAt time.Time
}

func (UserRole) TableName() string { return "user_roles" }

type RolePermission struct {
	RoleID       uuid.UUID `gorm:"type:uuid;primary_key"`
	PermissionID uuid.UUID `gorm:"type:uuid;primary_key"`
	CreatedAt    time.Time
}

func (RolePermission) TableName() string { return "role_permissions" }

type Resource struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key"`
	OrgID        uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_org_resource"`
	ResourceType string    `gorm:"size:100;not null;uniqueIndex:idx_org_resource"`
	ResourceKey  string    `gorm:"size:255;not null;uniqueIndex:idx_org_resource"`
	Name         string    `gorm:"size:200;not null"`
	Attributes   string    `gorm:"type:jsonb;default:'{}'"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (r *Resource) BeforeCreate(tx *gorm.DB) error {
	if r.ID == uuid.Nil {
		r.ID = uuid.New()
	}
	return nil
}
func (Resource) TableName() string { return "resources" }

type Policy struct {
	ID           uuid.UUID `gorm:"type:uuid;primary_key"`
	OrgID        uuid.UUID `gorm:"type:uuid;not null;index"`
	Name         string    `gorm:"size:200;not null"`
	Effect       string    `gorm:"size:20;not null;default:allow"`
	ResourceType string    `gorm:"size:100;not null"`
	Action       string    `gorm:"size:50;not null"`
	Condition    string    `gorm:"type:jsonb;default:'{}'"`
	IsEnabled    bool      `gorm:"default:true"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (p *Policy) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
func (Policy) TableName() string { return "policies" }

type PolicyBinding struct {
	PolicyID    uuid.UUID `gorm:"type:uuid;primary_key"`
	SubjectType string    `gorm:"size:50;primary_key"`
	SubjectID   string    `gorm:"size:255;primary_key"`
	CreatedAt   time.Time
}

func (PolicyBinding) TableName() string { return "policy_bindings" }

type APIPermission struct {
	ID             uuid.UUID `gorm:"type:uuid;primary_key"`
	Method         string    `gorm:"size:16;not null;uniqueIndex:idx_api_permission_route"`
	PathPattern    string    `gorm:"size:255;not null;uniqueIndex:idx_api_permission_route"`
	PermissionCode string    `gorm:"size:100;not null;index"`
	Description    string    `gorm:"size:500"`
	Enabled        bool      `gorm:"default:true;index"`
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (p *APIPermission) BeforeCreate(tx *gorm.DB) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
func (APIPermission) TableName() string { return "api_permissions" }
