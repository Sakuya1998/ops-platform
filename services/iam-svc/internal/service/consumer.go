package service

import (
	"context"
	"encoding/json"
	"log"

	"github.com/google/uuid"
	"github.com/ops-platform/iam-svc/internal/repository"
	"github.com/ops-platform/pkg/kafka"
)

type IAMConsumer struct {
	roleRepo *repository.RoleRepository
	iamSvc   *IAMService
}

func NewIAMConsumer(roleRepo *repository.RoleRepository) *IAMConsumer {
	return &IAMConsumer{roleRepo: roleRepo}
}

func (c *IAMConsumer) WithIAMService(iamSvc *IAMService) *IAMConsumer {
	c.iamSvc = iamSvc
	return c
}

func (c *IAMConsumer) HandleEvent(ctx context.Context, key string, value []byte) error {
	var event kafka.Event
	if err := json.Unmarshal(value, &event); err != nil {
		log.Printf("[IAM] Failed to unmarshal event: %v", err)
		return nil
	}
	if event.EventType != "user.deleted" {
		if isUserPermissionChangedEvent(event.EventType) {
			c.invalidateUserPermissionCache(ctx, event.UserID)
		}
		return nil
	}
	userID, err := uuid.Parse(event.UserID)
	if err != nil {
		log.Printf("[IAM] Ignore user.deleted with invalid user_id %q", event.UserID)
		return nil
	}
	if err := c.roleRepo.DeleteUserRoles(userID); err != nil {
		return err
	}
	c.invalidateUserPermissionCache(ctx, event.UserID)
	log.Printf("[IAM] Cleared role bindings for deleted user %s", userID)
	return nil
}

func (c *IAMConsumer) invalidateUserPermissionCache(ctx context.Context, userID string) {
	if c.iamSvc == nil || userID == "" {
		return
	}
	if err := c.iamSvc.InvalidateUserPermissionCache(ctx, userID); err != nil {
		log.Printf("[IAM] Failed to invalidate permission cache for user %q: %v", userID, err)
	}
}

func isUserPermissionChangedEvent(eventType string) bool {
	switch eventType {
	case "user.role_changed":
		return true
	default:
		return false
	}
}

func (c *IAMConsumer) Start(ctx context.Context, brokers []string, topic, groupID string) {
	consumer := kafka.NewConsumer(brokers, topic, groupID)
	go func() {
		if err := consumer.Consume(ctx, c.HandleEvent); err != nil {
			log.Printf("[IAM] Consumer error: %v", err)
		}
	}()
	log.Printf("[IAM] Consumer started for topic %s, group %s", topic, groupID)
}
