package main

import (
	"context"
	"crypto/tls"
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
	"github.com/HP-jh/TarsEnterprise/pkg/mtls"
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

	// Start 30s bidirectional reconcile loop
	gw.StartReconcileLoop()

	addr := fmt.Sprintf(":%d", *port)
	srv := &http.Server{Addr: addr, Handler: mux}

	// mTLS 零信任网络接入
	var rotMgr *mtls.RotationManager
	if cfg.MTLS.Enabled {
		log.Println("[mtls] 零信任网络接入已启用，初始化双向认证...")
		interval, _ := time.ParseDuration(cfg.MTLS.RotateInterval)
		window, _ := time.ParseDuration(cfg.MTLS.RotateWindow)
		rotMgr = mtls.NewRotationManager(cfg.MTLS.CertPath, cfg.MTLS.KeyPath, cfg.MTLS.CAPath, interval, window)
		if err := rotMgr.Initialize(); err != nil {
			log.Fatalf("[mtls] 证书初始化失败: %v", err)
		}
		if cfg.MTLS.AutoRotate {
			rotMgr.Start(1 * time.Hour)
			log.Println("[mtls] 证书自动轮换已启动（每小时检查）")
		}
		// 构建零信任 TLS 配置
		tlsCfg := rotMgr.GetTLSConfig()
		if tlsCfg == nil {
			log.Fatalf("[mtls] 无法获取 TLS 配置")
		}
		// 注入零信任身份验证
		zt := &mtls.ZeroTrustConfig{
			AllowRoles: cfg.MTLS.AllowRoles,
			Enforce:    cfg.MTLS.Enforce,
		}
		tlsCfg.VerifyPeerCertificate = zt.VerifyPeerCertificate(cfg.MTLS.AllowRoles)
		// 强制拒绝无有效客户端证书的连接
		tlsCfg.ClientAuth = tls.RequireAndVerifyClientCert
		tlsCfg.MinVersion = tls.VersionTLS13

		listener, err := tls.Listen("tcp", addr, tlsCfg)
		if err != nil {
			log.Fatalf("[mtls] TLS listener 启动失败: %v", err)
		}
		go func() {
			log.Printf("Gateway listening on %s (mTLS enforced, zero-trust)", addr)
			if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server error: %v", err)
			}
		}()
	} else {
		go func() {
			log.Printf("Gateway listening on %s", addr)
			if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server error: %v", err)
			}
		}()
	}

	// Wait for interrupt
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
	if rotMgr != nil {
		rotMgr.Stop()
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("Shutdown error: %v", err)
	}
	log.Println("Goodbye.")
}
