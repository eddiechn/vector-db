package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"vectordb/api"
	"vectordb/internal"
)

// Config holds command-line configuration
type Config struct {
	Port         int
	Dimensions   int
	DataPath     string
	AutoSave     bool
	SaveInterval time.Duration
	LoadData     bool
	LogLevel     string
}

func main() {
	// Parse command-line flags
	config := parseFlags()

	// Setup logging
	logger := log.New(os.Stdout, "[VectorDB] ", log.LstdFlags|log.Lshortfile)

	logger.Printf("Starting VectorDB with configuration:")
	logger.Printf("  Port: %d", config.Port)
	logger.Printf("  Dimensions: %d", config.Dimensions)
	logger.Printf("  Data Path: %s", config.DataPath)
	logger.Printf("  Auto Save: %v", config.AutoSave)
	logger.Printf("  Save Interval: %v", config.SaveInterval)

	// Create database configuration
	dbConfig := internal.DatabaseConfig{
		Dimensions:     config.Dimensions,
		DistanceMetric: internal.CosineSimilarity,
		IndexConfig: internal.IndexConfig{
			Type:       "hnsw",
			Parameters: make(map[string]interface{}),
		},
		HNSWConfig:   internal.DefaultHNSWConfig(),
		PersistPath:  config.DataPath,
		AutoSave:     config.AutoSave,
		SaveInterval: config.SaveInterval,
	}

	// Create the database
	db, err := internal.NewVectorDatabase(dbConfig)
	if err != nil {
		logger.Fatalf("Failed to create database: %v", err)
	}

	// Load existing data if requested
	if config.LoadData {
		logger.Printf("Loading existing data from %s", config.DataPath)
		if err := db.Load(); err != nil {
			logger.Printf("Warning: Failed to load existing data: %v", err)
		} else {
			stats := db.GetStats()
			logger.Printf("Loaded %d vectors", stats.VectorCount)
		}
	}

	// Create HTTP API server
	server := api.NewServer(db, config.Port, logger)

	// Setup graceful shutdown
	setupGracefulShutdown(db, logger)

	// Start the server
	logger.Printf("VectorDB is ready!")
	if err := server.Start(); err != nil {
		logger.Fatalf("Server failed: %v", err)
	}
}

// parseFlags parses command-line flags
func parseFlags() Config {
	config := Config{}

	flag.IntVar(&config.Port, "port", 8080, "HTTP server port")
	flag.IntVar(&config.Dimensions, "dimensions", 128, "Vector dimensions")
	flag.StringVar(&config.DataPath, "data", "./vectordb_data", "Data persistence path")
	flag.BoolVar(&config.AutoSave, "autosave", true, "Enable automatic saving")
	flag.DurationVar(&config.SaveInterval, "save-interval", 5*time.Minute, "Automatic save interval")
	flag.BoolVar(&config.LoadData, "load", true, "Load existing data on startup")
	flag.StringVar(&config.LogLevel, "log-level", "info", "Log level (debug, info, warn, error)")

	// Custom usage message
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "VectorDB - High-Performance Vector Database\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -port 8080 -dimensions 256\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -data /tmp/vectordb -autosave=false\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nAPI Endpoints:\n")
		fmt.Fprintf(os.Stderr, "  POST /vectors          - Insert vector\n")
		fmt.Fprintf(os.Stderr, "  GET  /vectors          - List vectors\n")
		fmt.Fprintf(os.Stderr, "  POST /search           - Search vectors\n")
		fmt.Fprintf(os.Stderr, "  GET  /stats            - Database stats\n")
		fmt.Fprintf(os.Stderr, "  GET  /health           - Health check\n")
		fmt.Fprintf(os.Stderr, "  GET  /                 - Web dashboard\n")
	}

	flag.Parse()

	// Validate configuration
	if config.Port < 1 || config.Port > 65535 {
		fmt.Fprintf(os.Stderr, "Error: Port must be between 1 and 65535\n")
		os.Exit(1)
	}

	if config.Dimensions < 1 || config.Dimensions > 10000 {
		fmt.Fprintf(os.Stderr, "Error: Dimensions must be between 1 and 10000\n")
		os.Exit(1)
	}

	if config.SaveInterval < time.Second {
		fmt.Fprintf(os.Stderr, "Error: Save interval must be at least 1 second\n")
		os.Exit(1)
	}

	return config
}

// setupGracefulShutdown handles shutdown signals
func setupGracefulShutdown(db *internal.VectorDatabase, logger *log.Logger) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		logger.Printf("Received shutdown signal, gracefully shutting down...")

		// Save database
		if err := db.Save(); err != nil {
			logger.Printf("Error saving database during shutdown: %v", err)
		} else {
			logger.Printf("Database saved successfully")
		}

		// Close database
		if err := db.Close(); err != nil {
			logger.Printf("Error closing database: %v", err)
		}

		logger.Printf("Shutdown complete")
		os.Exit(0)
	}()
}
