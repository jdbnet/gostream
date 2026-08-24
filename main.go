package main

import (
	"embed"
	"log"
	"os"
	"os/signal"
	"syscall"

	"gostream/internal/api"
	"gostream/internal/config"
	"gostream/internal/db"
	"gostream/internal/storage"
	"gostream/internal/stream"
)

//go:embed all:frontend/dist
var frontendFS embed.FS

func main() {
	log.Println("Starting GoStream...")

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := config.ApplyTimezone(cfg); err != nil {
		log.Printf("Warning: Failed to load timezone: %v", err)
	}

	database, err := db.Connect(cfg)
	if err != nil {
		log.Printf("Warning: Failed to connect to database: %v", err)
	} else {
		defer database.Close()
	}

	store, err := storage.New(cfg)
	if err != nil {
		log.Printf("Warning: Failed to initialize storage: %v", err)
	} else if local, ok := store.(*storage.LocalBackend); ok && !local.Writable() {
		log.Printf("Warning: local storage at %s is not writable", local.BasePath())
	}

	engine := stream.NewEngine(cfg, database, store)
	if database != nil && store != nil {
		engine.Start()
		defer engine.Stop()
	} else {
		log.Printf("Warning: Stream engine disabled until database and storage are configured.")
	}

	if database != nil && store != nil && cfg.Storage.Type == "local" && cfg.Storage.AutoScanOnStartup {
		go func() {
			result, err := storage.ScanLibrary(database, store)
			if err != nil {
				log.Printf("Startup library scan failed: %v", err)
				return
			}
			if result.Imported > 0 || len(result.Errors) > 0 {
				log.Printf("Startup library scan: imported %d, skipped %d, errors %d", result.Imported, result.Skipped, len(result.Errors))
				for _, scanErr := range result.Errors {
					log.Printf("Scan error: %s", scanErr)
				}
			}
		}()
	}

	server := api.NewServer(cfg, database, store, engine, frontendFS)

	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down GoStream...")
}
