//go:build !nats

package nats

import (
	"context"
	"time"
)

// mockTransport 内存模拟实现（默认构建，无需外部依赖）
type mockTransport struct {
	url           string
	streams       map[string][]Message
	subscribers   map[string][]func(Message)
	connected     bool
}

func newTransportImpl(url string) Transporter {
	return &mockTransport{
		url:         url,
		streams:     make(map[string][]Message),
		subscribers: make(map[string][]func(Message)),
	}
}

func (t *mockTransport) Connect() error {
	if t.url == "" {
		t.url = "memory://"
	}
	t.connected = true
	return nil
}

func (t *mockTransport) Close() {
	t.connected = false
}

func (t *mockTransport) Connected() bool { return t.connected }
func (t *mockTransport) URL() string     { return t.url }

func (t *mockTransport) Publish(subject string, msg Message) error {
	msg.ID = nowNanoID()
	msg.Timestamp = time.Now().Unix()
	msg.AuditHash = simpleHash(msg)
	t.streams[subject] = append(t.streams[subject], msg)
	for _, fn := range t.subscribers[subject] {
		go fn(msg)
	}
	return nil
}

func (t *mockTransport) Subscribe(subject string, handler func(Message)) error {
	t.subscribers[subject] = append(t.subscribers[subject], handler)
	return nil
}

func (t *mockTransport) Request(ctx context.Context, subject string, req Message) (*Message, error) {
	req.ID = "req-" + nowNanoID()
	req.Timestamp = time.Now().Unix()
	if err := t.Publish(subject, req); err != nil {
		return nil, err
	}
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

func (t *mockTransport) StateSync(direction string, state map[string]interface{}) error {
	return t.Publish("te.state."+direction, Message{
		From:    "te",
		To:      "tsg",
		Type:    "state_sync",
		Payload: state,
	})
}
