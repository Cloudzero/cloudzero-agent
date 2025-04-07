package main

import (
	"log"
	"log/slog"
	"os"
	"strconv"

	mock_controller "github.com/cloudzero/cloudzero-agent/mock/controller/controller"
)

func main() {
	slog.Default().Info("Starting the mock insights controller")

	required := []string{
		"COLLECTOR_ENDPOINT",
		"API_KEY",
		"TOTAL_HOURS",
		"NUM_NODES",
		"PODS_PER_NODE",
		"CPU_PER_NODE",
		"MEM_PER_NODE",
	}

	for _, item := range required {
		if _, exists := os.LookupEnv(item); !exists {
			log.Fatalf("The env variable `%s` is required", item)
		}
	}

	collectorEndpoint := os.Getenv("COLLECTOR_ENDPOINT")
	apiKey := os.Getenv("API_KEY")

	// read number values
	totalhours, err := strconv.ParseInt(os.Getenv("TOTAL_HOURS"), 10, 32)
	if err != nil {
		log.Fatalf("failed to parse `TOTAL_HOURS`")
	}

	numNodes, err := strconv.ParseInt(os.Getenv("NUM_NODES"), 10, 32)
	if err != nil {
		log.Fatalf("failed to parse `NUM_NODES`")
	}

	podsPerNode, err := strconv.ParseInt(os.Getenv("PODS_PER_NODE"), 10, 32)
	if err != nil {
		log.Fatalf("failed to parse `PODS_PER_NODE`")
	}

	cpuPerNode, err := strconv.ParseInt(os.Getenv("CPU_PER_NODE"), 10, 32)
	if err != nil {
		log.Fatalf("failed to parse `CPU_PER_NODE`")
	}

	memPerNode, err := strconv.ParseInt(os.Getenv("MEM_PER_NODE"), 10, 64)
	if err != nil {
		log.Fatalf("failed to parse `MEM_PER_NODE`='%s': %s", os.Getenv("MEM_PER_NODE"), err.Error())
	}

	numBatches, err := strconv.ParseInt(os.Getenv("NUM_BATCHES"), 10, 32)
	if err != nil {
		numBatches = 1
	}

	chunkSize, err := strconv.ParseInt(os.Getenv("CHUNK_SIZE"), 10, 32)
	if err != nil {
		chunkSize = 20_000
	}

	slog.Default().
		With("collectorEndpoint", collectorEndpoint, "apiKeyLength", len(apiKey)).
		With("totalHours", totalhours, "numNodes", numNodes, "podsPerNode", podsPerNode).
		With("cpuPerNode", cpuPerNode, "memPerNode", memPerNode).
		Info("Running controller with the provided config")

	// create a mock insights controller
	controller := mock_controller.MockInsightsController{
		CollectorEndpoint: collectorEndpoint,
		APIKey:            apiKey,
		TotalHours:        int(totalhours),
		NumNodes:          int(numNodes),
		PodsPerNode:       int(podsPerNode),
		CPUPerNode:        cpuPerNode,
		MemPerNode:        memPerNode,
		NumBatches:        int(numBatches),
		ChunkSize:         int(chunkSize),
	}

	// run the mock controller
	if err := controller.Run(); err != nil {
		log.Fatalf("there was an error running the mock controller: %s", err.Error())
	}

	slog.Default().Info("Successfully ran the mock controller")
}
