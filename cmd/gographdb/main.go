package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"gographdb/internal/api"
	"gographdb/internal/config"
	"gographdb/internal/graph"
	"gographdb/internal/index"
	"gographdb/internal/storage"
)

func main() {
	cfg := config.Parse()
	store := storage.NewMemStore()
	database := graph.New(store)
	indexes := index.NewManager()
	database.AddListener(indexes)
	persistence := storage.NewPersistence(cfg.DataDir)
	if err := persistence.LoadInto(database); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("load snapshot: %v", err)
	}
	indexes.Rebuild(database)
	wal, err := storage.OpenWAL(filepath.Join(cfg.DataDir, "wal.log"))
	if err != nil {
		log.Fatalf("open WAL: %v", err)
	}
	defer wal.Close()
	if err := wal.ReplayInto(database); err != nil {
		log.Fatalf("replay WAL: %v", err)
	}
	indexes.Rebuild(database)
	database.SetWAL(wal)
	server := &http.Server{Addr: cfg.Addr, Handler: api.NewServer(database, indexes, persistence, api.WebFS("web")).Handler(), ReadHeaderTimeout: 5 * time.Second}
	stopSnapshots := make(chan struct{})
	if cfg.SnapshotInterval > 0 {
		go func() {
			ticker := time.NewTicker(cfg.SnapshotInterval)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					if err := persistence.Save(database); err != nil {
						log.Printf("snapshot: %v", err)
					} else if err := wal.Truncate(); err != nil {
						log.Printf("truncate WAL: %v", err)
					}
				case <-stopSnapshots:
					return
				}
			}
		}()
	}
	go func() {
		log.Printf("GoGraphDB listening on http://%s", cfg.Addr)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve: %v", err)
		}
	}()
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
	<-signals
	close(stopSnapshots)
	database.SetWAL(nil)
	if err := persistence.Save(database); err != nil {
		log.Printf("final snapshot: %v", err)
	} else if err := wal.Truncate(); err != nil {
		log.Printf("truncate WAL: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
