package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
)

func main() {
	// check for an api key
	apiKey, exists := os.LookupEnv("API_KEY")
	if !exists {
		apiKey = "ak-test"
	}

	// check for a port
	port, exists := os.LookupEnv("PORT")
	if !exists {
		port = "8081"
	}

	// get s3 credentials
	s3Endpoint := os.Getenv("S3_ENDPOINT")
	if s3Endpoint == "" {
		log.Fatalf("Please pass `S3_ENDPOINT` env variable")
	}
	s3AccessKey := os.Getenv("S3_ACCESS_KEY")
	if s3AccessKey == "" {
		log.Fatalf("Please pass `S3_ACCESS_KEY` env variable")
	}
	s3PrivateKey := os.Getenv("S3_PRIVATE_KEY")
	if s3PrivateKey == "" {
		log.Fatalf("Please pass `S3_PRIVATE_KEY` env variable")
	}

	// create the client
	rw, err := newRemoteWrite(context.Background(), s3Endpoint, s3AccessKey, s3PrivateKey)
	if err != nil {
		log.Fatalf("failed to create the remotewrite client: %s", err.Error())
	}

	// create the http mux
	r := chi.NewRouter()

	// add middleware
	r.Use(func(h http.Handler) http.Handler {
		return authMiddleware(h, apiKey)
	})

	// add routes
	r.Route("/v1/container-metrics", rw.Handler)

	fmt.Printf("Server is running on :%s\n", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), r); err != nil {
		fmt.Println("Error starting server:", err)
	}
}
