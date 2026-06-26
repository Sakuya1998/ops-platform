package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/smtp"
	"strings"
	"text/template"

	"github.com/Sakuya1998/ops-platform/services/notify-svc/internal/model"
	"github.com/Sakuya1998/ops-platform/services/notify-svc/internal/repository"
	"github.com/Sakuya1998/ops-platform/pkg/kafka"
)

type NotifyService struct {
	channelRepo *repository.ChannelRepository
	tmplRepo    *repository.TemplateRepository
	logRepo     *repository.LogRepository
}

func NewNotifyService(cr *repository.ChannelRepository, tr *repository.TemplateRepository, lr *repository.LogRepository) *NotifyService {
	return &NotifyService{channelRepo: cr, tmplRepo: tr, logRepo: lr}
}

func (s *NotifyService) GetEnabledChannels() ([]model.NotificationChannel, error) {
	return s.channelRepo.GetEnabled()
}

func (s *NotifyService) SendFromEvent(ctx context.Context, ch *model.NotificationChannel, event *kafka.Event) error {
	_ = ctx
	title := fmt.Sprintf("[%s] %s", strings.ToUpper(event.EventType), event.Action)
	body := fmt.Sprintf("事件类型: %s\n用户: %s\n详情: %s\n时间: %s",
		event.EventType, event.Username, event.Detail, event.Timestamp)

	var cfgMap map[string]interface{}
	if err := json.Unmarshal([]byte(ch.Config), &cfgMap); err != nil {
		cfgMap = map[string]interface{}{}
	}

	var sendErr error
	switch ch.ChannelType {
	case "email":
		sendErr = s.SendEmail(cfgMap, stringConfig(cfgMap, "to_address"), title, body)
	case "webhook":
		sendErr = s.SendWebhook(stringConfig(cfgMap, "url"), map[string]interface{}{
			"event_type": event.EventType, "title": title, "body": body,
		})
	case "dingtalk":
		sendErr = s.SendDingTalk(stringConfig(cfgMap, "webhook_url"), fmt.Sprintf("%s\n%s", title, body))
	case "wechat":
		sendErr = s.SendWeChat(stringConfig(cfgMap, "webhook_url"), fmt.Sprintf("%s\n%s", title, body))
	case "feishu":
		sendErr = s.SendFeishu(stringConfig(cfgMap, "webhook_url"), fmt.Sprintf("%s\n%s", title, body))
	default:
		sendErr = fmt.Errorf("unsupported channel type: %s", ch.ChannelType)
	}

	status := "success"
	errMsg := ""
	if sendErr != nil {
		status = "failed"
		errMsg = sendErr.Error()
	}

	logEntry := &model.NotificationLog{
		ChannelID: &ch.ID, EventType: event.EventType,
		Recipient: ch.Name, Title: title,
		Status: status, ErrorMsg: errMsg,
	}
	if dbErr := s.logRepo.Create(logEntry); dbErr != nil {
		log.Printf("[Notify] Failed to log notification: %v", dbErr)
	}

	return sendErr
}

func (s *NotifyService) Send(cfgMap map[string]interface{}, to, subject, body string) error {
	chType := stringConfig(cfgMap, "channel_type")
	switch chType {
	case "email":
		return s.SendEmail(cfgMap, to, subject, body)
	case "webhook":
		return s.SendWebhook(stringConfig(cfgMap, "url"), map[string]interface{}{"subject": subject, "body": body})
	case "dingtalk":
		return s.SendDingTalk(stringConfig(cfgMap, "webhook_url"), fmt.Sprintf("%s\n%s", subject, body))
	case "wechat":
		return s.SendWeChat(stringConfig(cfgMap, "webhook_url"), fmt.Sprintf("%s\n%s", subject, body))
	case "feishu":
		return s.SendFeishu(stringConfig(cfgMap, "webhook_url"), fmt.Sprintf("%s\n%s", subject, body))
	}
	return fmt.Errorf("unsupported channel type: %s", chType)
}

func (s *NotifyService) SendEmail(cfg map[string]interface{}, to, subject, body string) error {
	host := stringConfig(cfg, "smtp_host")
	port := stringConfig(cfg, "smtp_port")
	user := stringConfig(cfg, "smtp_user")
	pass := stringConfig(cfg, "smtp_password")
	from := stringConfig(cfg, "from_address")
	if from == "" {
		from = user
	}
	if host == "" || port == "" || to == "" {
		return fmt.Errorf("smtp config incomplete")
	}

	auth := smtp.PlainAuth("", user, pass, host)
	msg := []byte(fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\n\r\n%s\r\n", from, to, subject, body))
	return smtp.SendMail(host+":"+port, auth, user, []string{to}, msg)
}

func (s *NotifyService) SendWebhook(url string, payload interface{}) error {
	if strings.TrimSpace(url) == "" {
		return fmt.Errorf("webhook url is required")
	}
	data, _ := json.Marshal(payload)
	resp, err := http.Post(url, "application/json", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("webhook returned %d", resp.StatusCode)
	}
	return nil
}

func (s *NotifyService) SendDingTalk(webhookURL, msg string) error {
	payload := map[string]interface{}{
		"msgtype": "text",
		"text":    map[string]string{"content": msg},
	}
	return s.SendWebhook(webhookURL, payload)
}

func (s *NotifyService) SendWeChat(webhookURL, msg string) error {
	payload := map[string]interface{}{
		"msgtype": "text",
		"text":    map[string]string{"content": msg},
	}
	return s.SendWebhook(webhookURL, payload)
}

func (s *NotifyService) SendFeishu(webhookURL, msg string) error {
	payload := map[string]interface{}{
		"msg_type": "text",
		"content":  map[string]string{"text": msg},
	}
	return s.SendWebhook(webhookURL, payload)
}

func (s *NotifyService) renderTemplate(tmpl string, data map[string]interface{}) (string, error) {
	t, err := template.New("notify").Parse(tmpl)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, data); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func stringConfig(cfg map[string]interface{}, key string) string {
	value, _ := cfg[key].(string)
	return strings.TrimSpace(value)
}

type NotifyConsumer struct {
	svc *NotifyService
}

func NewNotifyConsumer(svc *NotifyService) *NotifyConsumer {
	return &NotifyConsumer{svc: svc}
}

func (c *NotifyConsumer) HandleEvent(ctx context.Context, key string, value []byte) error {
	var event kafka.Event
	if err := json.Unmarshal(value, &event); err != nil {
		log.Printf("[Notify] Failed to unmarshal event: %v", err)
		return nil
	}
	log.Printf("[Notify] Processing event: %s from user %s", event.EventType, event.Username)

	channels, err := c.svc.GetEnabledChannels()
	if err != nil {
		log.Printf("[Notify] Failed to get channels: %v", err)
		return nil
	}

	if len(channels) == 0 {
		log.Printf("[Notify] No enabled channels configured, skipping event %s", event.EventType)
		return nil
	}

	for i := range channels {
		ch := channels[i]
		if err := c.svc.SendFromEvent(ctx, &ch, &event); err != nil {
			log.Printf("[Notify] Failed to send via channel %s (%s): %v", ch.Name, ch.ChannelType, err)
		} else {
			log.Printf("[Notify] Sent event %s via channel %s (%s)", event.EventType, ch.Name, ch.ChannelType)
		}
	}
	return nil
}

func (c *NotifyConsumer) Start(ctx context.Context, brokers []string, topic, groupID string) {
	consumer := kafka.NewConsumer(brokers, topic, groupID)
	go func() {
		if err := consumer.Consume(ctx, c.HandleEvent); err != nil {
			log.Printf("[Notify] Consumer error: %v", err)
		}
	}()
	log.Printf("[Notify] Consumer started for topic %s, group %s", topic, groupID)
}
