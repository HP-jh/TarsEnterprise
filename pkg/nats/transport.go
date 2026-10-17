package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Message 互通消息格式
type Message struct {
	ID        string                 `json:"id"`
	From      string                 `json:"from"` // "tsg" or "te"
	To        string                 `json:"to"`
	Type      string                 `json:"type"` // "message", "command", "state_sync"
	RoleToken string                 `json:"role_token"`
	Payload   map[string]interface{} `json:"payload"`
	Timestamp int64                  `json:"timestamp"`
	AuditHash string                 `json:"audit_hash"`
}

// Transporter 定义 NATS 传输接口（mock / real 两种实现）
type Transporter interface {
	Connect() error
	Close()
	Publish(subject string, msg Message) error
	Subscribe(subject string, handler func(Message)) error
	Request(ctx context.Context, subject string, req Message) (*Message, error)
	StateSync(direction string, state map[string]interface{}) error
	Connected() bool
	URL() string
}

// Transport 是 Transporter 的别名，保持向后兼容
type Transport struct {
	impl Transporter
}

func NewTransport(url string) *Transport {
	return &Transport{impl: newTransportImpl(url)}
}

func (t *Transport) Connect() error          { return t.impl.Connect() }
func (t *Transport) Close()                  { t.impl.Close() }
func (t *Transport) Publish(subject string, msg Message) error {
	return t.impl.Publish(subject, msg)
}
func (t *Transport) Subscribe(subject string, handler func(Message)) error {
	return t.impl.Subscribe(subject, handler)
}
func (t *Transport) Request(ctx context.Context, subject string, req Message) (*Message, error) {
	return t.impl.Request(ctx, subject, req)
}
func (t *Transport) StateSync(direction string, state map[string]interface{}) error {
	return t.impl.StateSync(direction, state)
}
func (t *Transport) Connected() bool         { return t.impl.Connected() }
func (t *Transport) URL() string             { return t.impl.URL() }

func simpleHash(msg Message) string {
	data, _ := json.Marshal(msg)
	sum := 0
	for _, b := range data {
		sum += int(b)
	}
	return fmt.Sprintf("audit:%x", sum)
}

func nowNanoID() string { return fmt.Sprintf("%d", time.Now().UnixNano()) }
