package repository

import (
	"time"

	"github.com/ops-platform/audit-svc/internal/model"
	"gorm.io/gorm"
)

type AuditRepository struct{ db *gorm.DB }

func NewAuditRepository(db *gorm.DB) *AuditRepository { return &AuditRepository{db: db} }

func (r *AuditRepository) Create(log *model.AuditLog) error { return r.db.Create(log).Error }

func (r *AuditRepository) BulkCreate(logs []model.AuditLog) error {
	return r.db.CreateInBatches(logs, 100).Error
}

func (r *AuditRepository) List(orgID string, eventType string, startTime, endTime string, offset, limit int) ([]model.AuditLog, int64, error) {
	var logs []model.AuditLog
	var total int64
	q := r.db.Model(&model.AuditLog{})
	if orgID != "" {
		q = q.Where("org_id = ?", orgID)
	}
	if eventType != "" {
		q = q.Where("event_type = ?", eventType)
	}
	if startTime != "" {
		if t, err := time.Parse(time.RFC3339, startTime); err == nil {
			q = q.Where("created_at >= ?", t)
		}
	}
	if endTime != "" {
		if t, err := time.Parse(time.RFC3339, endTime); err == nil {
			q = q.Where("created_at <= ?", t)
		}
	}
	q.Count(&total)
	err := q.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}
