package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alirezataban/monitor-engine-go/internal/browser"
	"github.com/alirezataban/monitor-engine-go/internal/collector"
	"github.com/alirezataban/monitor-engine-go/internal/config"
	"github.com/alirezataban/monitor-engine-go/internal/handler"
)

func main() {
	cfg := config.Load()

	mgr := browser.NewManager(cfg)
	if err := mgr.Start(); err != nil {
		log.Fatalf("browser start failed: %v", err)
	}
	defer mgr.Stop()

	registry := collector.DefaultRegistry()
	mux := http.NewServeMux()
	mux.Handle("/monitor", handler.NewMonitorHandlerWithRegistry(mgr, cfg, registry))
	mux.Handle("/health", handler.NewHealthHandler(mgr))
	mux.HandleFunc("/collectors", func(w http.ResponseWriter, r *http.Request) {
		handler.CollectorsHandlerWithRegistry(w, r, registry)
	})
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"service":"monitor-engine-go","endpoints":["/monitor","/health","/collectors"]}`))
	})

	srv := &http.Server{
		Addr:         cfg.ListenAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 130 * time.Second, // allow long checks
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
