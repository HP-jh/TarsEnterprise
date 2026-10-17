package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HP-jh/TarsEnterprise/pkg/config"
	"github.com/HP-jh/TarsEnterprise/pkg/gateway"
	"github.com/HP-jh/TarsEnterprise/pkg/lang"
	"github.com/HP-jh/TarsEnterprise/pkg/metrics"
	"github.com/HP-jh/TarsEnterprise/pkg/wizard"
)

func main() {
	var (
		port       = flag.Int("port", 8888, "Gateway listen port")
		cfgPath    = flag.String("config", "", "Config file path (auto-detect if empty)")
		wizardMode = flag.Bool("wizard", false, "Force wizard mode")
	)
	flag.Parse()

	log.Printf("TarsEnterprise %s starting...", config.Version)

	// Detect local environment languages/tools
	env := lang.Detect()
	log.Printf("Detected environments: %v", env.Summary())

	// Load or create config
	cfg, err := config.LoadOrInit(*cfgPath)
	if err != nil || *wizardMode || cfg.FirstRun {
		log.Println("Launching first-run wizard...")
		cfg = wizard.Run()
		if err := cfg.Save(); err != nil {
			log.Fatalf("Failed to save config: %v", err)
		}
	}

	// Start metrics
	metrics.Start()

	// Build and start gateway
	gw := gateway.New(cfg)
	mux := http.NewServeMux()
	gw.RegisterRoutes(mux)

	addr := fmt.Sprintf(":%d", *port)
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		log.Printf("Gateway listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// v0.3.0: start reconcile loop if TSG bridge enabled
	var reconcileCancel context.CancelFunc
	if cfg.TSGBridge.Enabled && cfg.TSGBridge.Address != "" {
		var ctx context.Context
		ctx, reconcileCancel = context.WithCancel(context.Background())
		go reconcileLoop(ctx, gw)
		log.Println("TSG reconcile loop started (30s interval)")
	}

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	if reconcileCancel != nil {
		reconcileCancel()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
	log.Println("Goodbye.")
}

// reconcileLoop 双向状态同步与指令拉取（每 30 秒）
func reconcileLoop(ctx context.Context, gw *gateway.Gateway) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	// Run immediately on start
	gw.ReconcileOnce()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			gw.ReconcileOnce()
		}
	}
}
