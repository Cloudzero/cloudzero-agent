// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package loader

// These tests pin the implicit config contract between this loader (the
// producer) and the consumer service that renders the CloudZero cluster
// details page. run() in command.go ships two independent blobs that the page
// reaches into with hardcoded paths:
//
//   - config_values_base64          raw helm/values.yaml, base64'd off disk
//                                    (command.go, "read the values file")
//   - config_webhook_server_base64  a cfg_webhook.Settings serialized via
//                                    ToBytes() (command.go, "parse the ... config")
//
// The consumer has no compile-time link to these paths, so a rename or
// relocation on this side silently returns a default and the page renders wrong
// data with no error (see CP-43434, which removed the ERROR log that used to
// surface such drift). These tests fail in the PR that moves a key, before it
// ships.
//
// The authoritative list of consumed paths lives in the consumer service's
// config-path lookups; keep the tables below in sync with it (tracked in
// CP-43520). A new consumed path added on the consumer side is NOT protected
// until it is added here — this producer repo cannot see the consumer.
//
// NOTE: we assert key-path PRESENCE, not a non-null value. Several consumed
// paths are legitimately null/"auto" in the shipped defaults (e.g.
// insightsController.enabled: null, components.webhookServer.enabled: auto), and
// the consumer tolerates null. A missing key is the break we care about.

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	cfg_webhook "github.com/cloudzero/cloudzero-agent/app/config/webhook"
)

// valuesYAMLPath locates helm/values.yaml relative to this package directory
// (go test sets the working directory to the package under test).
const valuesYAMLPath = "../../../../helm/values.yaml"

// consumedValuesPaths are the paths the page reads out of config_values_base64,
// governed by the Helm values structure (helm/values.yaml).
var consumedValuesPaths = [][]string{
	{"prometheusConfig", "globalScrapeInterval"},
	{"prometheusConfig", "scrapeJobs", "kubeStateMetrics", "scrapeInterval"},
	{"prometheusConfig", "scrapeJobs", "cadvisor", "scrapeInterval"},
	{"prometheusConfig", "scrapeJobs", "prometheus", "scrapeInterval"},
	{"prometheusConfig", "scrapeJobs", "aggregator", "scrapeInterval"},
	{"insightsController", "enabled"},
	{"components", "webhookServer", "enabled"},
}

// consumedWebhookPaths are the paths the page reads out of
// config_webhook_server_base64, governed by the cfg_webhook.Settings struct.
var consumedWebhookPaths = [][]string{
	{"filters", "labels", "enabled"},
	{"filters", "labels", "patterns"},
	{"filters", "labels", "resources"},
	{"filters", "annotations", "enabled"},
	{"filters", "annotations", "patterns"},
	{"filters", "annotations", "resources"},
}

// pathExists reports whether every key in path is present when walking root. The
// leaf value may be nil: a present-but-null key still counts, because the
// contract we protect is key-path presence, not the value at the leaf.
func pathExists(root map[string]any, path []string) bool {
	var cur any = root
	for _, key := range path {
		m, ok := cur.(map[string]any)
		if !ok {
			return false
		}
		next, present := m[key]
		if !present {
			return false
		}
		cur = next
	}
	return true
}

func TestConfigValuesContract(t *testing.T) {
	raw, err := os.ReadFile(valuesYAMLPath)
	require.NoError(t, err, "read helm/values.yaml (path is relative to this package)")

	var values map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &values), "parse helm/values.yaml")

	for _, path := range consumedValuesPaths {
		t.Run(strings.Join(path, "."), func(t *testing.T) {
			require.Truef(t, pathExists(values, path),
				"path %q is read from config_values_base64 by the cluster details page "+
					"consumer but no longer resolves in helm/values.yaml. A rename/relocation here "+
					"silently breaks the page. Restore the path, or update the consumer and this table together.",
				strings.Join(path, "."))
		})
	}
}

func TestConfigWebhookContract(t *testing.T) {
	// Serialize a zero-value (default) Settings through the exact ToBytes() the
	// loader uses in run(). We construct the struct directly rather than via
	// NewSettings() on purpose: NewSettings() performs cloud auto-detection over
	// the network, and we are asserting the serialized key structure, not values.
	raw, err := (&cfg_webhook.Settings{}).ToBytes()
	require.NoError(t, err, "serialize default webhook Settings via ToBytes()")

	var cfg map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &cfg), "parse serialized webhook Settings")

	for _, path := range consumedWebhookPaths {
		t.Run(strings.Join(path, "."), func(t *testing.T) {
			require.Truef(t, pathExists(cfg, path),
				"path %q is read from config_webhook_server_base64 by the cluster details page "+
					"consumer but no longer resolves in the serialized cfg_webhook.Settings. A "+
					"rename/relocation of the struct's yaml tags silently breaks the page. Restore the "+
					"path, or update the consumer and this table together.",
				strings.Join(path, "."))
		})
	}
}
