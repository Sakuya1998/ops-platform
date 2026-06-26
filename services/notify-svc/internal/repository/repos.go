package repository

import (
	"time"

	"github.com/google/uuid"
	"github.com/ops-platform/notify-svc/internal/model"
	"gorm.io/gorm"
)

type ChannelRepository struct{ db *gorm.DB }

func NewChannelRepository(db *gorm.DB) *ChannelRepository { return &ChannelRepository{db: db} }

func (r *ChannelRepository) Create(c *model.NotificationChannel) error { return r.db.Create(c).Error }
func (r *ChannelRepository) ListByOrg(orgID uuid.UUID) ([]model.NotificationChannel, error) {
	var channels []model.NotificationChannel
	query := r.db.Order("created_at DESC")
	if orgID != uuid.Nil {
		query = query.Where("org_id = ?", orgID)
	}
	err := query.Find(&channels).Error
	return channels, err
}
func (r *ChannelRepository) GetByID(id uuid.UUID) (*model.NotificationChannel, error) {
	var channel model.NotificationChannel
	if err := r.db.First(&channel, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &channel, nil
}
func (r *ChannelRepository) Update(c *model.NotificationChannel) error { return r.db.Save(c).Error }
func (r *ChannelRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&model.NotificationChannel{}, "id = ?", id).Error
}
func (r *ChannelRepository) GetEnabled() ([]model.NotificationChannel, error) {
	var channels []model.NotificationChannel
	err := r.db.Where("is_enabled = true").Find(&channels).Error
	return channels, err
}

type TemplateRepository struct{ db *gorm.DB }

func NewTemplateRepository(db *gorm.DB) *TemplateRepository { return &TemplateRepository{db: db} }

func (r *TemplateRepository) Create(t *model.NotificationTemplate) error { return r.db.Create(t).Error }
func (r *TemplateRepository) List() ([]model.NotificationTemplate, error) {
	var templates []model.NotificationTemplate
	err := r.db.Order("created_at DESC").Find(&templates).Error
	return templates, err
}
func (r *TemplateRepository) GetByID(id uuid.UUID) (*model.NotificationTemplate, error) {
	var tmpl model.NotificationTemplate
	if err := r.db.First(&tmpl, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &tmpl, nil
}
func (r *TemplateRepository) Update(t *model.NotificationTemplate) error {
	return r.db.Save(t).Error
}
func (r *TemplateRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&model.NotificationTemplate{}, "id = ?", id).Error
}

type LogRepository struct{ db *gorm.DB }

func NewLogRepository(db *gorm.DB) *LogRepository { return &LogRepository{db: db} }

func (r *LogRepository) Create(l *model.NotificationLog) error { return r.db.Create(l).Error }
func (r *LogRepository) List(eventType, status, startTime, endTime string, offset, limit int) ([]model.NotificationLog, int64, error) {
	var logs []model.NotificationLog
	var total int64
	query := r.db.Model(&model.NotificationLog{})
	if eventType != "" {
		query = query.Where("event_type = ?", eventType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			query = query.Where("created_at <= ?", t)
		}
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}
