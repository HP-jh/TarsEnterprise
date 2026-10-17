//go:build nats

package nats

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// realTransport 基于 github.com/nats-io/nats.go 的真实实现
type realTransport struct {
	url       string
	nc        *nats.Conn
	js        nats.JetStreamContext
	subs      []*nats.Subscription
	connected bool
}

func newTransportImpl(url string) Transporter {
	if url == "" {
		url = nats.DefaultURL
	}
	return &realTransport{url: url}
}

func (t *realTransport) Connect() error {
	nc, err := nats.Connect(t.url, nats.Timeout(5*time.Second))
	if err != nil {
		return fmt.Errorf("nats connect: %w", err)
	}
	js, err := nc.JetStream()
	if err != nil {
		nc.Close()
		return fmt.Errorf("nats jetstream: %w", err)
	}
	// Ensure stream exists for te.* subjects
	_, _ = js.AddStream(&nats.StreamConfig{
		Name:     "TE_BRIDGE",
		Subjects: []string{"te.>", "tsg.>"},
		Storage:  nats.MemoryStorage,
	})
	t.nc = nc
	t.js = js
	t.connected = true
	return nil
}

func (t *realTransport) Close() {
	for _, s := range t.subs {
		_ = s.Unsubscribe()
	}
	if t.nc != nil {
		t.nc.Close()
	}
	t.connected = false
}

func (t *realTransport) Connected() bool { return t.connected && t.nc != nil && t.nc.IsConnected() }
func (t *realTransport) URL() string     { return t.url }

func (t *realTransport) Publish(subject string, msg Message) error {
	msg.ID = nowNanoID()
	msg.Timestamp = time.Now().Unix()
	msg.AuditHash = simpleHash(msg)
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if t.js != nil {
		_, err = t.js.Publish(subject, data)
	} else {
		err = t.nc.Publish(subject, data)
	}
	return err
}

func (t *realTransport) Subscribe(subject string, handler func(Message)) error {
	h := func(m *nats.Msg) {
		var msg Message
		if err := json.Unmarshal(m.Data, &msg); err == nil {
			handler(msg)
		}
	}
	var sub *nats.Subscription
	var err error
	if t.js != nil {
		sub, err = t.js.Subscribe(subject, h)
	} else {
		sub, err = t.nc.Subscribe(subject, h)
	}
	if err != nil {
		return err
	}
	t.subs = append(t.subs, sub)
	return nil
}

func (t *realTransport) Request(ctx context.Context, subject string, req Message) (*Message, error) {
	req.ID = "req-" + nowNanoID()
	req.Timestamp = time.Now().Unix()
	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	msg, err := t.nc.RequestWithContext(ctx, subject, data)
	if err != nil {
		return nil, err
	}
	var resp Message
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

func (t *realTransport) StateSync(direction string, state map[string]interface{}) error {
	return t.Publish("te.state."+direction, Message{
		From:    "te",
		To:      "tsg",
		Type:    "state_sync",
		Payload: state,
	})
}
