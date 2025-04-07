package smoke

import (
	"fmt"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

type controllerArgs struct {
	hours, nodes, pods, cpu, mem, batches, chunks int64
}

func (t *testContext) StartController(args controllerArgs) *testcontainers.Container {
	t.CreateNetwork()

	if t.controller == nil {
		fmt.Println("Building mock controller ...")

		// create an endpoint for the controller to talk to
		endpoint := fmt.Sprintf("http://%s:8080/collector", t.collectorName)
		fmt.Printf("Running the mock controller with the collector endpoint: %s\n", endpoint)

		controllerReq := testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:    "../..",
				Dockerfile: "tests/docker/Dockerfile.controller",
				KeepImage:  true,
			},
			Name:       t.controllerName,
			Networks:   []string{t.network.Name},
			Entrypoint: []string{"/app/controller"},
			Env: map[string]string{
				"COLLECTOR_ENDPOINT": endpoint,
				"API_KEY":            t.apiKey,
				"TOTAL_HOURS":        fmt.Sprintf("%d", args.hours),   // number of hours calculated as time.Now() - (time.Hour * `env.TOTAL_HOURS`)
				"NUM_NODES":          fmt.Sprintf("%d", args.nodes),   // total number of nodes
				"PODS_PER_NODE":      fmt.Sprintf("%d", args.pods),    // number of pods to generate data per node
				"CPU_PER_NODE":       fmt.Sprintf("%d", args.cpu),     // number of CPUs to use for each node
				"MEM_PER_NODE":       fmt.Sprintf("%d", args.mem),     // number of memory in bytes for each node
				"NUM_BATCHES":        fmt.Sprintf("%d", args.batches), // number of times to send the data
				"CHUNK_SIZE":         fmt.Sprintf("%d", args.chunks),  // size to break the metrics into when sending to the collector
			},
			LogConsumerCfg: &testcontainers.LogConsumerConfig{
				Consumers: []testcontainers.LogConsumer{&stdoutLogConsumer{}},
			},
			WaitingFor: wait.ForLog("Running controller with the provided config"),
		}

		controller, err := testcontainers.GenericContainer(t.ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: controllerReq,
			Started:          true,
		})
		require.NoError(t, err, "failed to create the controller")

		fmt.Println("Controller built successfully")
		t.controller = &controller
	}

	return t.controller
}
