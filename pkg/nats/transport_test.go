package nats

import (
	"context"
	"testing"
	"time"
)

func TestTransport_PublishSubscribe(t *testing.T) {
	tr := NewTransport("")
	if err := tr.Connect(); err != nil {
		t.Fatal(err)
	}

	var received Message
	done := make(chan bool, 1)
	_ = tr.Subscribe("test.subject", func(msg Message) {
		received = msg
		done <- true
	})

	msg := Message{From: "te", To: "tsg", Type: "message", Payload: map[string]interface{}{"data": 42}}
	if err := tr.Publish("test.subject", msg); err != nil {
		t.Fatal(err)
	}

	select {
	case <-done:
		if received.From != "te" {
			t.Error("from mismatch")
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for message")
	}
}

func TestTransport_Request(t *testing.T) {
	tr := NewTransport("")
	_ = tr.Connect()
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	resp, err := tr.Request(ctx, "test.req", Message{From: "te", To: "tsg", Type: "command"})
	if err != nil {
		t.Fatal(err)
	}
	if resp.Type != "ack" {
		t.Error("expected ack")
	}
}

func TestTransport_StateSync(t *testing.T) {
	tr := NewTransport("")
	_ = tr.Connect()
	err := tr.StateSync("push", map[string]interface{}{"status": "ok"})
	if err != nil {
		t.Fatal(err)
	}
}
