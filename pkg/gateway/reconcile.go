package gateway

import (
	"context"
	"fmt"
	"time"
)

// StartReconcileLoop 启动 30s 双向 reconcile loop
func (g *Gateway) StartReconcileLoop() {
	if g.tsgInterop == nil || g.tsgInterop.tsClient == nil {
		fmt.Println("[reconcile] TSG client not configured, skipping reconcile loop")
		return
	}

	ticker := time.NewTicker(30 * time.Second)
	go func() {
		defer ticker.Stop()
		for range ticker.C {
			g.reconcileOnce()
		}
	}()
	fmt.Println("[reconcile] 30s bidirectional reconcile loop started")
}

func (g *Gateway) reconcileOnce() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. 推送本地 state 到 TSG
	g.tsgInterop.mu.RLock()
	localState := make(map[string]interface{})
	for k, v := range g.tsgInterop.state {
		localState[k] = v
	}
	g.tsgInterop.mu.RUnlock()

	if len(localState) > 0 {
		_, _ = g.tsgInterop.tsClient.StateSync(ctx, localState)
	}

	// 2. 从 TSG 拉取 remote state
	remoteState, err := g.tsgInterop.tsClient.FetchCommands(ctx)
	_ = remoteState // commands fetched, state reconciliation via dedicated endpoint in future
	if err != nil {
		fmt.Printf("[reconcile] fetch from TSG error: %v\n", err)
	}

	// 3. 推送待处理指令到 TSG
	g.tsgInterop.mu.Lock()
	pending := make([]TSGCommand, 0)
	for _, cmd := range g.tsgInterop.commands {
		if cmd.Status == "pending" {
			pending = append(pending, cmd)
		}
	}
	g.tsgInterop.mu.Unlock()

	for _, cmd := range pending {
		resp, err := g.tsgInterop.tsClient.SendCommand(ctx, cmd.ToolID, cmd.Params)
		if err == nil && resp.StatusCode == 200 {
			g.tsgInterop.mu.Lock()
			for i := range g.tsgInterop.commands {
				if g.tsgInterop.commands[i].ID == cmd.ID {
					g.tsgInterop.commands[i].Status = "delivered"
					break
				}
			}
			g.tsgInterop.mu.Unlock()
		}
	}

	// 4. 检查 TSG 健康
	hr, err := g.tsgInterop.tsClient.HealthCheck(ctx)
	if err != nil {
		fmt.Printf("[reconcile] TSG health check failed: %v\n", err)
	} else {
		fmt.Printf("[reconcile] TSG health: %s (version %s)\n", hr.Status, hr.Version)
	}

	// 5. 更新 last_sync 标记
	g.tsgInterop.mu.Lock()
	g.tsgInterop.lastSync = time.Now()
	g.tsgInterop.mu.Unlock()
}
