package main

import (
	"embed"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"gostream/internal/api"
	"gostream/internal/config"
	"gostream/internal/db"
	"gostream/internal/s3"
	"gostream/internal/stream"
)

//go:embed all:frontend/dist
var frontendFS embed.FS

func main() {
	log.Println("Starting GoStream...")
	
	// Force application timezone to Europe/London to align with the database
	if loc, err := time.LoadLocation("Europe/London"); err == nil {
		time.Local = loc
	} else {
		log.Printf("Warning: Failed to load timezone: %v", err)
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	database, err := db.Connect(cfg)
	if err != nil {
		log.Printf("Warning: Failed to connect to database: %v", err)
	} else {
		defer database.Close()
	}

	s3Client, err := s3.NewClient(cfg)
	if err != nil {
		log.Printf("Warning: Failed to initialize S3 client: %v", err)
	}

	engine := stream.NewEngine(cfg, database, s3Client)
	if database != nil && s3Client != nil {
		engine.Start()
		defer engine.Stop()
	} else {
		log.Printf("Warning: Stream engine disabled until Database and S3 are configured.")
	}

	server := api.NewServer(cfg, database, s3Client, engine, frontendFS)
	
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Server failed: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("Shutting down GoStream...")
}
