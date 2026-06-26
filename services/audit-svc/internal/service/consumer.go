package service

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/Sakuya1998/ops-platform/services/audit-svc/internal/model"
	"github.com/Sakuya1998/ops-platform/services/audit-svc/internal/repository"
	"github.com/Sakuya1998/ops-platform/pkg/kafka"
)

type AuditConsumer struct {
	repo   *repository.AuditRepository
	buffer []model.AuditLog
	mu     sync.Mutex
	stopCh chan struct{}
}

func NewAuditConsumer(repo *repository.AuditRepository) *AuditConsumer {
	return &AuditConsumer{
		repo:   repo,
		buffer: make([]model.AuditLog, 0, 100),
		stopCh: make(chan struct{}),
	}
}

func (c *AuditConsumer) HandleEvent(ctx context.Context, key string, value []byte) error {
	var event kafka.Event
	if err := json.Unmarshal(value, &event); err != nil {
		log.Printf("[Audit] Failed to unmarshal event: %v", err)
		return nil
	}

	logEntry := auditLogFromEvent(event)

	c.mu.Lock()
	c.buffer = append(c.buffer, logEntry)
	bufSize := len(c.buffer)
	c.mu.Unlock()

	if bufSize >= 50 {
		c.flush()
	}

	return nil
}

func auditLogFromEvent(event kafka.Event) model.AuditLog {
	orgID, _ := uuid.Parse(event.OrgID)
	userID, _ := uuid.Parse(event.UserID)

	ts := time.Now()
	if event.Timestamp != "" {
		if parsed, err := time.Parse(time.RFC3339, event.Timestamp); err == nil {
			ts = parsed
		}
	}

	return model.AuditLog{
		OrgID:        orgID,
		UserID:       userID,
		Username:     event.Username,
		EventType:    event.EventType,
		Action:       event.Action,
		ResourceType: event.ResourceType,
		ResourceID:   event.ResourceID,
		Detail:       event.Detail,
		IP:           event.IP,
		UserAgent:    event.UserAgent,
		SessionID:    event.SessionID,
		ReasonCode:   event.ReasonCode,
		RequestID:    event.RequestID,
		CreatedAt:    ts,
	}
}

func (c *AuditConsumer) flush() {
	c.mu.Lock()
	if len(c.buffer) == 0 {
		c.mu.Unlock()
		return
	}
	batch := c.buffer
	c.buffer = make([]model.AuditLog, 0, 100)
	c.mu.Unlock()

	if err := c.repo.BulkCreate(batch); err != nil {
		log.Printf("[Audit] Failed to flush %d logs: %v", len(batch), err)
		c.mu.Lock()
		c.buffer = append(c.buffer, batch...)
		c.mu.Unlock()
		return
	}
	log.Printf("[Audit] Flushed %d audit log entries", len(batch))
}

func (c *AuditConsumer) Start(ctx context.Context, brokers []string, topic, groupID string) {
	consumer := kafka.NewConsumer(brokers, topic, groupID)

	flushTicker := time.NewTicker(10 * time.Second)
	go func() {
		defer flushTicker.Stop()
		for {
			select {
			case <-flushTicker.C:
				c.flush()
			case <-ctx.Done():
				c.flush()
				close(c.stopCh)
				return
			}
		}
	}()

	go func() {
		if err := consumer.Consume(ctx, c.HandleEvent); err != nil {
			log.Printf("[Audit] Consumer error: %v", err)
		}
	}()
	log.Printf("[Audit] Consumer started for topic %s, group %s", topic, groupID)
}
