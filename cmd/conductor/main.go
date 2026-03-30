package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"conductor/internal/api"
	"conductor/internal/orchestrator"
	"conductor/internal/sftp"
	"conductor/internal/store"
	"conductor/internal/ws"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Config from env with sane defaults.
	httpAddr  := envOr("CONDUCTOR_HTTP_ADDR", ":3100")
	sftpAddr  := envOr("CONDUCTOR_SFTP_ADDR", ":2222")
	jwtSecret := envOr("CONDUCTOR_JWT_SECRET", "change-me-in-production")
	fsRoot    := envOr("CONDUCTOR_FS_ROOT", "/tmp/conductor")
	staticDir := envOr("CONDUCTOR_STATIC_DIR", "./web/dist")

	// Postgres
	db, err := store.Connect(ctx)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer db.Close()
	log.Println("conductor: database connected")

	// Run migrations
	if err := runMigrations(ctx, db); err != nil {
		log.Fatalf("migrations: %v", err)
	}

	// WebSocket hub
	hub := ws.NewHub()

	// Orchestrator
	orch := orchestrator.New(db, hub, fsRoot)
	if err := orch.Start(ctx); err != nil {
		log.Fatalf("orchestrator start: %v", err)
	}
	defer orch.Stop()

	// SFTP server
	sftpSrv, err := sftp.New(db, fsRoot)
	if err != nil {
		log.Fatalf("sftp init: %v", err)
	}
	if err := sftpSrv.Listen(ctx, sftpAddr); err != nil {
		log.Printf("sftp listen: %v (non-fatal, FS access unavailable)", err)
	} else {
		log.Printf("conductor: SFTP listening on %s", sftpAddr)
	}

	// HTTP server
	srv := api.NewServer(db, hub, orch, jwtSecret, staticDir)
	httpSrv := &http.Server{
		Addr:         httpAddr,
		Handler:      srv.Router(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		log.Printf("conductor: HTTP listening on %s", httpAddr)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("http serve: %v", err)
		}
	}()

	<-ctx.Done()
	log.Println("conductor: shutting down…")

	shutCtx, shutCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer shutCancel()
	httpSrv.Shutdown(shutCtx) //nolint
	log.Println("conductor: stopped")
}

func runMigrations(ctx context.Context, db *store.DB) error {
	entries, err := os.ReadDir("migrations")
	if err != nil {
		return err
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".sql") {
			continue
		}
		data, err := os.ReadFile("migrations/" + e.Name())
		if err != nil {
			return err
		}
		if _, err := db.Pool.Exec(ctx, string(data)); err != nil {
			if strings.Contains(err.Error(), "already exists") {
				continue
			}
			return fmt.Errorf("migration %s: %w", e.Name(), err)
		}
		log.Printf("conductor: migration %s applied", e.Name())
	}
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
