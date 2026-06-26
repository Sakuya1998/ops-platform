package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string, topic string) *Producer {
	w := &kafka.Writer{
		Addr:         kafka.TCP(brokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		BatchTimeout: 10 * time.Millisecond,
	}
	log.Printf("[Kafka] Producer initialized for topic: %s, brokers: %v", topic, brokers)
	return &Producer{writer: w}
}

func (p *Producer) Publish(ctx context.Context, key string, msg interface{}) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshal message: %w", err)
	}
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(key),
		Value: data,
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

type Consumer struct {
	reader *kafka.Reader
}

func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:     brokers,
		Topic:       topic,
		GroupID:     groupID,
		MinBytes:    10,
		MaxBytes:    10e6,
		MaxWait:     1 * time.Second,
		StartOffset: kafka.LastOffset,
	})
	log.Printf("[Kafka] Consumer initialized for topic: %s, group: %s", topic, groupID)
	return &Consumer{reader: r}
}

type ConsumeFn func(ctx context.Context, key string, value []byte) error

func (c *Consumer) Consume(ctx context.Context, fn ConsumeFn) error {
	for {
		msg, err := c.reader.ReadMessage(ctx)
		if err != nil {
			return fmt.Errorf("read message: %w", err)
		}
		if err := fn(ctx, string(msg.Key), msg.Value); err != nil {
			log.Printf("[Kafka] Consume error: %v", err)
			continue
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

type Event struct {
	EventType    string `json:"event_type"`
	UserID       string `json:"user_id"`
	OrgID        string `json:"org_id"`
	Username     string `json:"username"`
	Action       string `json:"action"`
	ResourceType string `json:"resource_type"`
	ResourceID   string `json:"resource_id"`
	Detail       string `json:"detail"`
	IP           string `json:"ip"`
	UserAgent    string `json:"user_agent"`
	SessionID    string `json:"session_id"`
	ReasonCode   string `json:"reason_code"`
	RequestID    string `json:"request_id"`
	Timestamp    string `json:"timestamp"`
}

func (p *Producer) PublishEvent(ctx context.Context, event Event) error {
	return p.Publish(ctx, event.EventType, event)
}
