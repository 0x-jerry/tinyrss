package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"tinyrss/internal/config"
	"tinyrss/internal/feeds"
	"tinyrss/internal/repository"
	"tinyrss/internal/server"
	"tinyrss/internal/store"
)

//go:embed all:web/dist
var distFS embed.FS

func main() {
	cfg := config.Parse()

	if err := os.MkdirAll(filepath.Dir(cfg.DB), 0o755); err != nil {
		log.Fatalf("create db dir: %v", err)
	}
	st, err := store.Open(cfg.DB)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer st.Close()

	repo := repository.NewRepo(st.DB)
	fetcher := feeds.NewFetcher(repo)
	fetcher.Start()

	// go:embed includes the "web/dist" prefix; root the FS at the dist contents.
	dist, err := fs.Sub(distFS, "web/dist")
	if err != nil {
		log.Fatalf("embedded web/dist: %v", err)
	}
	srv := server.New(repo, fetcher, cfg.Token, dist)
	httpSrv := &http.Server{Addr: cfg.Addr, Handler: srv.Handler()}

	errCh := make(chan error, 1)
	go func() { errCh <- httpSrv.ListenAndServe() }()
	log.Printf("tinyrss listening on %s", cfg.Addr)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil && err != http.ErrServerClosed {
			log.Fatalf("http server: %v", err)
		}
	case <-sig:
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpSrv.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	fetcher.Stop()
	log.Print("shutdown complete")
}
