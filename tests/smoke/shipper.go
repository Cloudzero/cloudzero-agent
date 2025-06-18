// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package smoke

import (
	"fmt"

	"github.com/docker/docker/api/types/container"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func (t *testContext) StartShipper() *testcontainers.Container {
	t.CreateNetwork()

	if t.shipper == nil {
		fmt.Println("Building shipper ...")

		shipperReq := testcontainers.ContainerRequest{
			FromDockerfile: testcontainers.FromDockerfile{
				Context:    "../..",
				Dockerfile: "tests/docker/Dockerfile.shipper",
				KeepImage:  true,
			},
			Name:     t.shipperName,
			Networks: []string{t.network.Name},
			HostConfigModifier: func(hc *container.HostConfig) {
				hc.Binds = append(hc.Binds, fmt.Sprintf("%s:%s", t.tmpDir, t.tmpDir)) // bind the tmp dir to the container
			},
			Entrypoint: []string{"/app/cloudzero-shipper", "-config", t.configFile},
			Env:        map[string]string{},
			LogConsumerCfg: &testcontainers.LogConsumerConfig{
				Consumers: []testcontainers.LogConsumer{&stdoutLogConsumer{}},
			},
			WaitingFor: wait.ForLog("Shipper service starting"),
		}

		shipper, err := testcontainers.GenericContainer(t.ctx, testcontainers.GenericContainerRequest{
			ContainerRequest: shipperReq,
			Started:          true,
		})
		require.NoError(t, err, "failed to create the shipper")

		fmt.Println("Shipper built successfully")
		t.shipper = &shipper
	}

	return t.shipper
}
