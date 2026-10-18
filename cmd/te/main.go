package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/HP-jh/TarsEnterprise/pkg/config"
	"github.com/HP-jh/TarsEnterprise/pkg/gateway"
	"github.com/HP-jh/TarsEnterprise/pkg/lang"
	"github.com/HP-jh/TarsEnterprise/pkg/metrics"
	"github.com/HP-jh/TarsEnterprise/pkg/mtls"
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

	// 零信任网络接入：mTLS 双向认证启动
	var listener net.Listener
	var listenErr error
	if cfg.MTLS.Enabled {
		log.Println("[mTLS] 零信任网络接入已启用，正在加载证书...")
		cm, err := setupMTLS(cfg)
		if err != nil {
			log.Fatalf("[mTLS] 证书配置失败: %v", err)
		}
		tlsConfig := cm.TLSConfig()
		listener, listenErr = tls.Listen("tcp", addr, tlsConfig)
		if listenErr != nil {
			log.Fatalf("[mTLS] TLS 监听失败: %v", listenErr)
		}
		log.Printf("[mTLS] 双向认证监听于 %s (模式: %s)", addr, cfg.MTLS.Mode)
	} else {
		listener, listenErr = net.Listen("tcp", addr)
		if listenErr != nil {
			log.Fatalf("监听失败: %v", listenErr)
		}
		log.Printf("Gateway listening on %s (plaintext)", addr)
	}

	go func() {
		if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
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

// setupMTLS 根据配置初始化 mTLS 证书管理器
// 支持自签名 CA（自动生成）和自定义 CA（从文件加载）
func setupMTLS(cfg *config.Config) (*mtls.CertificateManager, error) {
	if cfg.MTLS.CertFile != "" && cfg.MTLS.KeyFile != "" && cfg.MTLS.CAFile != "" {
		// 自定义 CA 模式：从指定路径加载
		return mtls.NewManager(cfg.MTLS.CertFile, cfg.MTLS.KeyFile, cfg.MTLS.CAFile)
	}

	// 自签名 CA 模式：自动生成服务端证书与 CA
	log.Println("[mTLS] 自签名 CA 模式：正在生成临时证书...")
	caPair, err := mtls.GenerateSelfSigned("TarsEnterprise-CA")
	if err != nil {
		return nil, fmt.Errorf("生成 CA 证书失败: %w", err)
	}
	serverPair, err := mtls.GenerateSelfSigned("TarsEnterprise-Server")
	if err != nil {
		return nil, fmt.Errorf("生成服务端证书失败: %w", err)
	}
	// 自签名模式下，服务端证书自身就是 CA
	return mtls.NewManagerFromPEM(serverPair.CertPEM, serverPair.KeyPEM, caPair.CertPEM)
}
