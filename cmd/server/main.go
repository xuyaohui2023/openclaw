package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/flashclaw/flashclaw-im-channel/internal/config"
	"github.com/flashclaw/flashclaw-im-channel/internal/handler"
	"github.com/flashclaw/flashclaw-im-channel/internal/middleware"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "config error: %v\n", err)
		os.Exit(1)
	}

	mux := http.NewServeMux()

	// Public route
	mux.HandleFunc("/health", handler.Health)

	// Protected IM route — all CRUD operations on /api/v1/im
	//   GET    /api/v1/im              list all channels with current config
	//   POST   /api/v1/im              create/replace a channel config
	//   PATCH  /api/v1/im              partial update a channel config
	//   DELETE /api/v1/im?channel=X    delete a channel config
	auth := middleware.APIKey(cfg.APIKey)
	mux.Handle("/api/v1/im", auth(handler.IMHandler(cfg)))
	mux.Handle("/api/v1/port-check", auth(handler.PortCheckHandler()))

	addr := net.JoinHostPort(cfg.Bind, fmt.Sprintf("%d", cfg.Port))
	srv := &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown on SIGINT / SIGTERM
	idleConnsClosed := make(chan struct{})
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh
		log.Println("shutting down...")
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
		close(idleConnsClosed)
	}()

	log.Printf("flashclaw-im-channel env=%s listening on %s", cfg.Env, addr)
	log.Printf("openclaw config: %s", cfg.OpenclawConfigPath)

	if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("listen: %v", err)
	}
	<-idleConnsClosed
}
