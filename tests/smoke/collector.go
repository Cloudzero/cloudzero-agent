// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package smoke

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func (t *testContext) StartCollector() *testcontainers.Container {
	t.CreateNetwork()

	if t.collector == nil {
		fmt.Println("Building collector ...")

		collectorReq := testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:    "../..",
				Dockerfile: "tests/docker/Dockerfile.collector",
				KeepImage:  true,
			},
			Name:     t.collectorName,
			Networks: []string{t.network.Name},
			HostConfigModifier: func(hc *container.HostConfig) {
				hc.Binds = append(hc.Binds, fmt.Sprintf("%s:%s", t.tmpDir, t.tmpDir)) // bind the tmp dir to the container
			},
			Entrypoint: []string{"/app/collector", "-config", t.configFile},
			Env:        map[string]string{},
			LogConsumerCfg: &testcontainers.LogConsumerConfig{
				Consumers: []testcontainers.LogConsumer{&stdoutLogConsumer{}},
			},
			WaitingFor: wait.ForLog("Starting service"),
		}

		collector, err := testcontainers.GenericContainer(t.ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: collectorReq,
			Started:          true,
		})
		require.NoError(t, err, "failed to create the collector")

		fmt.Println("Collector built successfully")
		t.collector = &collector
	}

	return t.collector
}
