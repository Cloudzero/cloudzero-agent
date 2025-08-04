// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-obvious/server"

	"github.com/cloudzero/cloudzero-agent/app/domain/upload"
	"github.com/cloudzero/cloudzero-agent/app/handlers"
	"github.com/cloudzero/cloudzero-agent/app/storage/minio"
	"github.com/cloudzero/cloudzero-agent/app/storage/tracker"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Configuration
	cfg := minio.Config{
		Endpoint:        getEnv("MINIO_ENDPOINT", "localhost:9000"),
		AccessKeyID:     getEnv("MINIO_ACCESS_KEY", "minioadmin"),
		SecretAccessKey: getEnv("MINIO_SECRET_KEY", "minioadmin"),
		BucketName:      getEnv("MINIO_BUCKET", "uploads"),
		UseSSL:          getEnv("MINIO_USE_SSL", "false") == "true",
	}

	// Initialize MinIO client
	minioClient, err := minio.NewClient(cfg)
	if err != nil {
		log.Fatalf("Failed to create MinIO client: %v", err)
	}

	// Ensure bucket exists
	if err := minioClient.EnsureBucket(ctx); err != nil {
		log.Fatalf("Failed to ensure bucket exists: %v", err)
	}

	// Initialize file tracker
	fileTracker := tracker.NewMemoryTracker()

	// Initialize upload service with dependency injection
	uploadService := upload.NewService(minioClient, fileTracker)

	// Initialize HTTP server
	srv := server.New()

	// Register upload API
	uploadAPI := handlers.NewUploadAPI("/upload", uploadService)
	if err := uploadAPI.Register(srv); err != nil {
		log.Fatalf("Failed to register upload API: %v", err)
	}

	// Start server
	port := getEnv("PORT", "8080")
	addr := fmt.Sprintf(":%s", port)
	
	httpServer := &http.Server{
		Addr:    addr,
		Handler: srv,
	}

	// Graceful shutdown
	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan
		
		log.Println("Shutting down server...")
		
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer shutdownCancel()
		
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		cancel()
	}()

	log.Printf("Starting upload API server on %s", addr)
	log.Printf("MinIO endpoint: %s", cfg.Endpoint)
	log.Printf("MinIO bucket: %s", cfg.BucketName)
	
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server failed to start: %v", err)
	}
	
	log.Println("Server stopped")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}