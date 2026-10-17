package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Transport NATS JetStream 传输层抽象（v0.1.0 使用内存模拟，实际部署接入 NATS）
type Transport struct {
	url       string
	streams   map[string][]Message
	subscribers map[string][]func(Message)
}

// Message 互通消息格式
type Message struct {
	ID        string                 `json:"id"`
	From      string                 `json:"from"`      // "tsg" or "te"
	To        string                 `json:"to"`
	Type      string                 `json:"type"`      // "message", "command", "state_sync"
	RoleToken string                 `json:"role_token"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp int64                  `json:"timestamp"`
	AuditHash string                 `json:"audit_hash"`
}

func NewTransport(url string) *Transport {
	return &Transport{
		url:         url,
		streams:     make(map[string][]Message),
		subscribers: make(map[string][]func(Message)),
	}
}

func (t *Transport) Connect() error {
	// v0.1.0: memory-backed transport
	if t.url == "" {
		t.url = "memory://"
	}
	return nil
}

func (t *Transport) Publish(subject string, msg Message) error {
	msg.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	msg.Timestamp = time.Now().Unix()
	msg.AuditHash = simpleHash(msg)
	t.streams[subject] = append(t.streams[subject], msg)
	for _, fn := range t.subscribers[subject] {
		go fn(msg)
	}
	return nil
}

func (t *Transport) Subscribe(subject string, handler func(Message)) error {
	t.subscribers[subject] = append(t.subscribers[subject], handler)
	return nil
}

func (t *Transport) Request(ctx context.Context, subject string, req Message) (*Message, error) {
	req.ID = fmt.Sprintf("req-%d", time.Now().UnixNano())
	req.Timestamp = time.Now().Unix()
	if err := t.Publish(subject, req); err != nil {
		return nil, err
	}
	// Simple sync response simulation
	resp := &Message{
		ID:        req.ID + "-resp",
		From:      req.To,
		To:        req.From,
		Type:      "ack",
		Timestamp: time.Now().Unix(),
		Payload:   map[string]interface{}{"status": "ok"},
	}
	return resp, nil
}

func (t *Transport) StateSync(direction string, state map[string]interface{}) error {
	return t.Publish("te.state."+direction, Message{
		From:    "te",
		To:      "tsg",
		Type:    "state_sync",
		Payload: state,
	})
}

func simpleHash(msg Message) string {
	// Simplified audit hash for v0.1.0
	data, _ := json.Marshal(msg)
	sum := 0
	for _, b := range data {
		sum += int(b)
	}
	return fmt.Sprintf("audit:%x", sum)
}
