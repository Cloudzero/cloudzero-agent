// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package smoke

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/cloudzero/cloudzero-agent-validator/app/config/gator"
	remotewrite "github.com/cloudzero/cloudzero-agent-validator/mock/remotewrite/pkg"
	"github.com/cloudzero/cloudzero-agent-validator/tests/utils"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

func (t *testContext) StartMockRemoteWrite() *testcontainers.Container {
	t.CreateNetwork()

	var wg sync.WaitGroup

	if t.s3instance == nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("Creating the mock s3 instance ...")

			s3instanceRequest := testcontainers.ContainerRequest{
				Image:    "minio/minio:latest",
				Networks: []string{t.network.Name},
				Name:     t.s3instanceName,
				Env: map[string]string{
					"MINIO_ROOT_USER":     "minio-admin",
					"MINIO_ROOT_PASSWORD": "minio-admin",
				},
				Cmd:             []string{"server", "/data"},
				WaitingFor:      wait.ForLog("API: http://").WithStartupTimeout(2 * time.Minute),
				AutoRemove:      true,
				AlwaysPullImage: true,
				LogConsumerCfg: &testcontainers.LogConsumerConfig{
					Consumers: []testcontainers.LogConsumer{&stdoutLogConsumer{}},
				},
			}

			s3instance, err := testcontainers.GenericContainer(t.ctx, testcontainers.GenericContainerRequest{
				ContainerRequest: s3instanceRequest,
				Started:          true,
			})
			require.NoError(t, err, "failed to create the s3 instance")
			t.s3instance = &s3instance

			fmt.Println("Successfully created the s3 instance")
		}()
	}

	if t.remotewrite == nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			fmt.Println("Building the mock remotewrite ...")

			s3Endpoint := fmt.Sprintf("%s:9000", t.s3instanceName)
			fmt.Printf("With mock s3 endpoint: '%s'\n", s3Endpoint)

			remotewriteReq := testcontainers.ContainerRequest{
				FromDockerfile: testcontainers.FromDockerfile{
					Context:    "../..",
					Dockerfile: "tests/docker/Dockerfile.remotewrite",
					KeepImage:  true,
				},
				Name:         t.remotewriteName,
				Networks:     []string{t.network.Name},
				ExposedPorts: []string{t.remoteWritePort},
				Entrypoint:   []string{"/app/remotewrite"},
				Env: map[string]string{
					"API_KEY": t.apiKey,
					"PORT":    t.remoteWritePort,

					// minio creds
					"S3_ENDPOINT":    s3Endpoint,
					"S3_ACCESS_KEY":  "minio-admin",
					"S3_PRIVATE_KEY": "minio-admin",

					// mock delays
					"UPLOAD_DELAY_MS": t.uploadDelayMs,
				},
				LogConsumerCfg: &testcontainers.LogConsumerConfig{
					Consumers: []testcontainers.LogConsumer{&stdoutLogConsumer{}},
				},
				WaitingFor: wait.ForLog("Mock remotewrite is listening on: 'localhost:"),
			}

			remotewrite, err := testcontainers.GenericContainer(t.ctx, testcontainers.GenericContainerRequest{
				ContainerRequest: remotewriteReq,
				Started:          true,
			})
			require.NoError(t, err, "failed to create the mock remotewrite")

			fmt.Println("Mock remotewrite built successfully")
			t.remotewrite = &remotewrite

			// set the host as the setting
			t.SetSettings(func(settings *config.Settings) error {
				settings.Cloudzero.Host = fmt.Sprintf("%s:%s", t.remotewriteName, t.remoteWritePort)
				return nil
			})
		}()
	}

	wg.Wait()

	return t.remotewrite
}

func (t *testContext) QueryMinio() *remotewrite.QueryMinioResponse {
	if t.s3instance == nil || t.remotewrite == nil {
		t.Fatalf("the remotewrite is null")
	}

	host, err := utils.ContainerExternalHost(t.ctx, (*t.remotewrite), t.remoteWritePort)
	require.NoError(t, err, "failed to create the external host")

	// create the request
	endpoint := fmt.Sprintf("%s/v1/container-metrics/mock/queryMinio", host.String())
	fmt.Printf("Querying with: %s\n", endpoint)
	req, err := http.NewRequestWithContext(t.ctx, "GET", endpoint, nil)
	require.NoError(t, err, "failed to create the request for the minio query")

	// add the headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", t.apiKey)

	// send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	require.NoError(t, err, "failed to send the minio request")
	defer resp.Body.Close()
	require.Equal(t, 200, resp.StatusCode, "invalid minio request response")

	// read the body
	var response remotewrite.QueryMinioResponse
	err = json.NewDecoder(resp.Body).Decode(&response)
	require.NoError(t, err, "failed to read minio response body")

	return &response
}
