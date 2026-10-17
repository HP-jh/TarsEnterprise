package nats

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

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

// Transport NATS 传输层抽象（v0.3.0 支持真实 NATS + 内存 fallback）
type Transport struct {
	url         string
	mode        string // "nats" | "memory"
	conn        net.Conn
	reader      *bufio.Reader
	subscribers map[string][]func(Message)
	streams     map[string][]Message
	mu          sync.RWMutex
	connected   bool
	lastErr     error
}

func NewTransport(url string) *Transport {
	return &Transport{
		url:         url,
		subscribers: make(map[string][]func(Message)),
		streams:     make(map[string][]Message),
	}
}

// Connect 建立连接：如果 url 是真实 NATS 地址则走 NATS 协议，否则回退内存模式
func (t *Transport) Connect() error {
	if t.url == "" || t.url == "memory://" {
		t.mode = "memory"
		t.url = "memory://"
		t.connected = true
		return nil
	}

	// 尝试真实 NATS 连接
	conn, err := net.DialTimeout("tcp", stripScheme(t.url), 5*time.Second)
	if err != nil {
		// 优雅降级到内存模式
		t.mode = "memory"
		t.url = "memory://"
		t.lastErr = fmt.Errorf("nats connect failed (fallback to memory): %w", err)
		t.connected = true
		return nil
	}

	t.mode = "nats"
	t.conn = conn
	t.reader = bufio.NewReader(conn)
	t.connected = true

	// 发送 CONNECT
	connectJSON := `{"verbose":false,"pedantic":false,"tls_required":false,"name":"TarsEnterprise","lang":"go","version":"0.3.0"}`
	if err := t.natsWrite(fmt.Sprintf("CONNECT %s\r\n", connectJSON)); err != nil {
		_ = conn.Close()
		t.mode = "memory"
		t.conn = nil
		t.lastErr = fmt.Errorf("nats handshake failed (fallback to memory): %w", err)
		return nil
	}

	// 启动读循环
	go t.readLoop()
	return nil
}

func (t *Transport) IsConnected() bool {
	return t.connected
}

func (t *Transport) Mode() string {
	return t.mode
}

func (t *Transport) LastError() error {
	return t.lastErr
}

func (t *Transport) Publish(subject string, msg Message) error {
	msg.ID = fmt.Sprintf("%d", time.Now().UnixNano())
	msg.Timestamp = time.Now().Unix()
	msg.AuditHash = simpleHash(msg)

	if t.mode == "memory" {
		t.mu.Lock()
		t.streams[subject] = append(t.streams[subject], msg)
		handlers := make([]func(Message), len(t.subscribers[subject]))
		copy(handlers, t.subscribers[subject])
		t.mu.Unlock()
		for _, fn := range handlers {
			go fn(msg)
		}
		return nil
	}

	// NATS 模式
	payload, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	return t.natsWrite(fmt.Sprintf("PUB %s %d\r\n%s\r\n", subject, len(payload), payload))
}

func (t *Transport) Subscribe(subject string, handler func(Message)) error {
	if t.mode == "memory" {
		t.mu.Lock()
		t.subscribers[subject] = append(t.subscribers[subject], handler)
		t.mu.Unlock()
		return nil
	}

	// NATS 模式：分配 sid
	sid := fmt.Sprintf("sub_%d", time.Now().UnixNano())
	t.mu.Lock()
	t.subscribers[sid] = append(t.subscribers[sid], handler)
	// 也按 subject 索引，方便路由
	t.subscribers[subject] = append(t.subscribers[subject], handler)
	t.mu.Unlock()

	return t.natsWrite(fmt.Sprintf("SUB %s %s\r\n", subject, sid))
}

func (t *Transport) Request(ctx context.Context, subject string, req Message) (*Message, error) {
	req.ID = fmt.Sprintf("req-%d", time.Now().UnixNano())
	req.Timestamp = time.Now().Unix()

	if t.mode == "memory" {
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

	// NATS 模式：使用 reply-to inbox
	inbox := fmt.Sprintf("_INBOX.%d", time.Now().UnixNano())
	replyCh := make(chan Message, 1)

	_ = t.Subscribe(inbox, func(msg Message) {
		replyCh <- msg
	})

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}
	if err := t.natsWrite(fmt.Sprintf("PUB %s %s %d\r\n%s\r\n", subject, inbox, len(payload), payload)); err != nil {
		return nil, err
	}

	select {
	case resp := <-replyCh:
		return &resp, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (t *Transport) StateSync(direction string, state map[string]interface{}) error {
	return t.Publish("te.state."+direction, Message{
		From:    "te",
		To:      "tsg",
		Type:    "state_sync",
		Payload: state,
	})
}

func (t *Transport) Close() error {
	t.connected = false
	if t.conn != nil {
		_ = t.natsWrite("QUIT\r\n")
		return t.conn.Close()
	}
	return nil
}

// readLoop NATS 读循环
func (t *Transport) readLoop() {
	for {
		line, err := t.reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				t.lastErr = err
			}
			return
		}
		line = strings.TrimSpace(line)
		if line == "" || line == "+OK" || strings.HasPrefix(line, "INFO ") {
			continue
		}
		if strings.HasPrefix(line, "MSG ") {
			t.handleMSG(line)
		}
	}
}

func (t *Transport) handleMSG(line string) {
	parts := strings.SplitN(line, " ", 5)
	if len(parts) < 4 {
		return
	}
	subject := parts[1]
	sid := parts[2]
	var sizeStr string
	if len(parts) == 4 {
		sizeStr = parts[3]
	} else {
		// MSG subject sid reply-to size
		sizeStr = parts[4]
		sid = parts[3]
	}
	size, _ := strconv.Atoi(sizeStr)
	if size <= 0 {
		return
	}
	payload := make([]byte, size)
	_, _ = io.ReadFull(t.reader, payload)
	// consume trailing \r\n
	_, _ = t.reader.ReadString('\n')

	var msg Message
	if err := json.Unmarshal(payload, &msg); err != nil {
		return
	}

	t.mu.RLock()
	var handlers []func(Message)
	if h, ok := t.subscribers[sid]; ok {
		handlers = append([]func(Message){}, h...)
	}
	if h, ok := t.subscribers[subject]; ok {
		handlers = append(handlers, h...)
	}
	t.mu.RUnlock()

	for _, fn := range handlers {
		go fn(msg)
	}
}

func (t *Transport) natsWrite(s string) error {
	if t.conn == nil {
		return fmt.Errorf("nats connection not established")
	}
	_, err := t.conn.Write([]byte(s))
	return err
}

func stripScheme(url string) string {
	url = strings.TrimPrefix(url, "nats://")
	url = strings.TrimPrefix(url, "tls://")
	if !strings.Contains(url, ":") {
		url += ":4222"
	}
	return url
}

func simpleHash(msg Message) string {
	data, _ := json.Marshal(msg)
	sum := 0
	for _, b := range data {
		sum += int(b)
	}
	return fmt.Sprintf("audit:%x", sum)
}
