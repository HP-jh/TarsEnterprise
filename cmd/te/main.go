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
		port    = flag.Int("port", 8888, "Gateway listen port")
		cfgPath = flag.String("config", "", "Config file path (auto-detect if empty)")
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

	// Start 30s bidirectional reconcile loop
	gw.StartReconcileLoop()

	addr := fmt.Sprintf(":%d", *port)
	srv := &http.Server{Addr: addr, Handler: mux}

	go func() {
		log.Printf("Gateway listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
	log.Println("Goodbye.")
}
