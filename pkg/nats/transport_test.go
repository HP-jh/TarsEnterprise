package nats

import (
	"context"
	"testing"
	"time"
)

func TestTransport_Memory_PublishSubscribe(t *testing.T) {
	tr := NewTransport("")
	if err := tr.Connect(); err != nil {
		t.Fatal(err)
	}
	if tr.Mode() != "memory" {
		t.Fatalf("expected memory mode, got %s", tr.Mode())
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
		if received.AuditHash == "" {
			t.Error("audit hash missing")
		}
	case <-time.After(time.Second):
		t.Error("timeout waiting for message")
	}
}

func TestTransport_Memory_Request(t *testing.T) {
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

func TestTransport_Memory_StateSync(t *testing.T) {
	tr := NewTransport("")
	_ = tr.Connect()
	err := tr.StateSync("push", map[string]interface{}{"status": "ok"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestTransport_NATS_Fallback(t *testing.T) {
	// 连接一个不存在的 NATS 服务器，应优雅降级到内存模式
	tr := NewTransport("nats://127.0.0.1:19999")
	if err := tr.Connect(); err != nil {
		t.Fatal(err)
	}
	if tr.Mode() != "memory" {
		t.Fatalf("expected fallback to memory, got %s", tr.Mode())
	}
	if tr.LastError() == nil {
		t.Error("expected LastError to be set after fallback")
	}
	if !tr.IsConnected() {
		t.Error("expected connected even in fallback mode")
	}
}

func TestTransport_Close(t *testing.T) {
	tr := NewTransport("")
	_ = tr.Connect()
	if err := tr.Close(); err != nil {
		t.Fatal(err)
	}
	if tr.IsConnected() {
		t.Error("expected disconnected after close")
	}
}
