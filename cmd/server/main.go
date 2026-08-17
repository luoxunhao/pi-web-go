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

	"github.com/luoxunhao/pi-web-go/internal/config"
	"github.com/luoxunhao/pi-web-go/internal/events"
	"github.com/luoxunhao/pi-web-go/internal/files"
	"github.com/luoxunhao/pi-web-go/internal/pigo"
	"github.com/luoxunhao/pi-web-go/internal/server"
	"github.com/luoxunhao/pi-web-go/internal/session"
	"github.com/luoxunhao/pi-web-go/internal/webui"
)

func main() {
	configPath := flag.String("config", "", "path to config.toml")
	flag.Parse()
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	supervisor := pigo.NewSupervisor(cfg.Pigo)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := supervisor.Start(ctx); err != nil {
		log.Fatalf("start pigo: %v", err)
	}
	defer supervisor.Stop()

	client := pigo.NewClient(supervisor.BaseURL(), supervisor.Password())
	access := files.NewAccess(cfg.Filesystem.AllowedRoots)
	deps := server.Dependencies{
		PigoClient:   client,
		Converter:    events.NewConverter(),
		Cursor:       events.NewCursorStore(),
		SessionMgr:   session.NewManager(10 * time.Minute),
		FileAccess:   access,
		Static:       webui.StaticFS(cfg.Web.FrontendDir),
		WebPassword:  cfg.Web.Password,
		AllowedHosts: cfg.Web.AllowedHosts,
	}
	httpServer := &http.Server{
		Addr:              fmt.Sprintf("%s:%d", cfg.Server.Hostname, cfg.Server.Port),
		Handler:           server.NewRouter(deps),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("pi-web-go listening on http://%s:%d", cfg.Server.Hostname, cfg.Server.Port)
		errCh <- httpServer.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("server: %v", err)
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
	}
}
