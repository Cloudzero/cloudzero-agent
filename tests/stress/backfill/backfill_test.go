// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Out-of-process stress tests for the backfill job. Starts a fake K8s API
// server + collector sink, spawns `cloudzero-webhook -backfill` as a
// subprocess, and measures its RSS externally. Test infrastructure memory
// doesn't contaminate the measurements.
package backfill_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var webhookBinary string

func TestMain(m *testing.M) {
	bin, err := buildBinary()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to build webhook binary: %v\n", err)
		os.Exit(1)
	}
	webhookBinary = bin
	os.Exit(m.Run())
}

func buildBinary() (string, error) {
	dir, err := os.MkdirTemp("", "cz-stress-bin-*")
	if err != nil {
		return "", err
	}
	bin := filepath.Join(dir, "cloudzero-webhook")
	cmd := exec.Command("go", "build", "-o", bin, "github.com/cloudzero/cloudzero-agent/app/functions/webhook")
	cmd.Dir = filepath.Join(moduleRoot(), "..")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}
	return bin, nil
}

func moduleRoot() string {
	// tests/stress/backfill/ → tests/
	dir, _ := os.Getwd()
	return filepath.Dir(filepath.Dir(dir))
}

var scalingCases = []struct {
	namespaces int
	podsPerNS  int
}{
	{1000, 10},   // 11k
	{2000, 10},   // 22k
	{5000, 20},   // 105k
	{5000, 50},   // 255k
	{10000, 50},  // 510k
	{10000, 100}, // 1.01M
}

func TestBackfillMemoryScaling(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in -short mode")
	}

	t.Logf("%-12s %10s %12s %8s %10s", "resources", "duration", "peak_rss", "gc", "reqs")
	t.Logf("%-12s %10s %12s %8s %10s", "---------", "--------", "--------", "--", "----")

	for _, tc := range scalingCases {
		total := tc.namespaces * (1 + tc.podsPerNS)
		name := fmt.Sprintf("%dk", total/1000)
		t.Run(name, func(t *testing.T) {
			r := runOutOfProcess(t, tc.namespaces, tc.podsPerNS, "")
			t.Logf(
				"%-12d %10s %12s %8s %10d",
				total,
				r.duration.Round(time.Millisecond),
				humanBytes(r.peakRSS),
				"-",
				r.sinkReqs,
			)
		})
	}
}

func TestBackfillMemoryScalingWithLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in -short mode")
	}

	t.Logf("GOMEMLIMIT = 256MiB")
	t.Logf("%-12s %10s %12s %10s", "resources", "duration", "peak_rss", "reqs")
	t.Logf("%-12s %10s %12s %10s", "---------", "--------", "--------", "----")

	for _, tc := range scalingCases {
		total := tc.namespaces * (1 + tc.podsPerNS)
		name := fmt.Sprintf("%dk", total/1000)
		t.Run(name, func(t *testing.T) {
			r := runOutOfProcess(t, tc.namespaces, tc.podsPerNS, "256MiB")
			t.Logf(
				"%-12d %10s %12s %10d",
				total,
				r.duration.Round(time.Millisecond),
				humanBytes(r.peakRSS),
				r.sinkReqs,
			)
		})
	}
}

func TestBackfillHeapProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in -short mode")
	}

	r := runOutOfProcess(t, 5000, 20, "")
	t.Logf("peak RSS: %s, duration: %s, sink reqs: %d",
		humanBytes(r.peakRSS), r.duration.Round(time.Millisecond), r.sinkReqs)
	if r.heapProfile != "" {
		t.Logf("heap profile: %s", r.heapProfile)
		t.Logf("analyze with: go tool pprof -inuse_space %s", r.heapProfile)
	}
}

type result struct {
	duration    time.Duration
	peakRSS     uint64
	sinkReqs    int64
	heapProfile string
}

func runOutOfProcess(t *testing.T, namespaces, podsPerNS int, goMemLimit string) result {
	t.Helper()

	// Start fake K8s API + collector sink.
	var sinkReqs int64
	srv := startFakeServer(t, namespaces, podsPerNS, &sinkReqs)
	defer srv.Close()

	// Write configs.
	tmpDir, err := os.MkdirTemp("", "cz-stress-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	kubeconfigPath := filepath.Join(tmpDir, "kubeconfig")
	writeKubeconfig(t, kubeconfigPath, srv.URL)

	cfgPath := filepath.Join(tmpDir, "webhook-config.yaml")
	writeWebhookConfig(t, cfgPath, kubeconfigPath, srv.URL)

	// Spawn the backfill binary.
	cmd := exec.Command(webhookBinary, "-config", cfgPath, "-backfill", "-backfill-no-wait")
	cmd.Stdout = io.Discard
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "KUBECONFIG="+kubeconfigPath)
	if goMemLimit != "" {
		cmd.Env = append(cmd.Env, "GOMEMLIMIT="+goMemLimit)
	}

	require.NoError(t, cmd.Start())
	pid := cmd.Process.Pid

	// Sample RSS until the process exits.
	var peakRSS uint64
	doneCh := make(chan error, 1)
	go func() { doneCh <- cmd.Wait() }()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	start := time.Now()
	var heapProfile string

	for {
		select {
		case err := <-doneCh:
			if err != nil {
				t.Logf("backfill exited with error: %v", err)
			}
			return result{
				duration:    time.Since(start),
				peakRSS:     peakRSS,
				sinkReqs:    atomic.LoadInt64(&sinkReqs),
				heapProfile: heapProfile,
			}
		case <-ticker.C:
			rss := readRSS(pid)
			if rss > peakRSS {
				peakRSS = rss
			}
			// Grab a heap profile once we've seen significant growth.
			if heapProfile == "" && rss > 100*1024*1024 {
				path := filepath.Join(tmpDir, "heap.pprof")
				if captureHeapProfile(srv.URL, path) == nil {
					heapProfile = path
				}
			}
		}
	}
}

func captureHeapProfile(baseURL, path string) error {
	// The backfill binary serves pprof when profiling is enabled.
	// Our config sets server.port to 18099 and server.profiling to true.
	resp, err := http.Get("http://localhost:18099/debug/pprof/heap")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("status %d", resp.StatusCode)
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// startFakeServer returns an httptest server that serves both the K8s API
// endpoints the backfiller hits and the collector sink endpoint.
func startFakeServer(t *testing.T, namespaces, podsPerNS int, sinkReqs *int64) *httptest.Server {
	t.Helper()

	// Pre-generate all namespace and pod list responses.
	nsPages := buildNamespacePages(namespaces, 500)
	podPages := buildPodPages(namespaces, podsPerNS, 500)

	mux := http.NewServeMux()

	// K8s API discovery (minimal — the client needs these to not error).
	mux.HandleFunc("/api", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"kind": "APIVersions", "versions": []string{"v1"}})
	})
	mux.HandleFunc("/api/v1", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{
			"kind":         "APIResourceList",
			"groupVersion": "v1",
			"resources":    []map[string]any{},
		})
	})
	mux.HandleFunc("/apis", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, map[string]any{"kind": "APIGroupList", "groups": []any{}})
	})

	// Namespace list (paginated).
	mux.HandleFunc("/api/v1/namespaces", func(w http.ResponseWriter, r *http.Request) {
		servePaginatedList(w, r, nsPages)
	})

	// Catch-all for /api/v1/namespaces/{ns}/{resource} — serve pods, empty
	// list for everything else. The backfiller hits this for every enabled
	// resource type × every namespace, so it needs to be fast.
	mux.HandleFunc("/api/v1/namespaces/", func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/v1/namespaces/"), "/")
		if len(parts) == 2 && parts[1] == "pods" {
			if pages, ok := podPages[parts[0]]; ok {
				servePaginatedList(w, r, pages)
				return
			}
		}
		writeJSON(w, emptyList())
	})

	// /api/v1/nodes, /api/v1/{other} — empty list.
	mux.HandleFunc("/api/v1/nodes", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, emptyList())
	})

	// /apis/{group}/{version}/namespaces/{ns}/{resource} and
	// /apis/{group}/{version}/{resource} — empty list.
	mux.HandleFunc("/apis/", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, emptyList())
	})

	// Collector sink.
	mux.HandleFunc("/v1/container-metrics", func(w http.ResponseWriter, r *http.Request) {
		io.Copy(io.Discard, r.Body)
		r.Body.Close()
		atomic.AddInt64(sinkReqs, 1)
		w.WriteHeader(http.StatusOK)
	})

	// Health.
	mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	return httptest.NewServer(mux)
}

type page struct {
	body      []byte
	continue_ string
}

func servePaginatedList(w http.ResponseWriter, r *http.Request, pages []page) {
	token := r.URL.Query().Get("continue")
	idx := 0
	if token != "" {
		var err error
		idx, err = strconv.Atoi(token)
		if err != nil || idx >= len(pages) {
			idx = 0
		}
	}
	if idx < len(pages) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(pages[idx].body)
	}
}

func buildNamespacePages(total, pageSize int) []page {
	var pages []page
	for i := 0; i < total; i += pageSize {
		end := i + pageSize
		if end > total {
			end = total
		}
		items := make([]corev1.Namespace, 0, end-i)
		for n := i + 1; n <= end; n++ {
			items = append(items, corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: fmt.Sprintf("stress-ns-%d", n),
					Labels: map[string]string{
						"app":         "cz-stress",
						"environment": pickEnv(n),
						"team":        fmt.Sprintf("team-%d", n%20),
						"cost-center": fmt.Sprintf("cc-%d", n%50),
					},
				},
			})
		}
		cont := ""
		if end < total {
			cont = strconv.Itoa(len(pages) + 1)
		}
		list := corev1.NamespaceList{
			TypeMeta: metav1.TypeMeta{Kind: "NamespaceList", APIVersion: "v1"},
			ListMeta: metav1.ListMeta{Continue: cont},
			Items:    items,
		}
		body, _ := json.Marshal(list)
		pages = append(pages, page{body: body, continue_: cont})
	}
	return pages
}

func buildPodPages(namespaces, podsPerNS, pageSize int) map[string][]page {
	result := make(map[string][]page, namespaces)
	for n := 1; n <= namespaces; n++ {
		nsName := fmt.Sprintf("stress-ns-%d", n)
		var pages []page
		for i := 0; i < podsPerNS; i += pageSize {
			end := i + pageSize
			if end > podsPerNS {
				end = podsPerNS
			}
			items := make([]corev1.Pod, 0, end-i)
			for p := i + 1; p <= end; p++ {
				items = append(items, corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("pod-%d", p),
						Namespace: nsName,
						Labels: map[string]string{
							"app":         "cz-stress",
							"team":        fmt.Sprintf("team-%d", n%20),
							"cost-center": fmt.Sprintf("cc-%d", n%50),
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{{Name: "c", Image: "pause"}},
					},
				})
			}
			cont := ""
			if end < podsPerNS {
				cont = strconv.Itoa(len(pages) + 1)
			}
			list := corev1.PodList{
				TypeMeta: metav1.TypeMeta{Kind: "PodList", APIVersion: "v1"},
				ListMeta: metav1.ListMeta{Continue: cont},
				Items:    items,
			}
			body, _ := json.Marshal(list)
			pages = append(pages, page{body: body, continue_: cont})
		}
		result[nsName] = pages
	}
	return result
}

func emptyList() map[string]any {
	return map[string]any{
		"kind":       "List",
		"apiVersion": "v1",
		"metadata":   map[string]any{},
		"items":      []any{},
	}
}

func pickEnv(i int) string {
	switch i % 3 {
	case 0:
		return "production"
	case 1:
		return "staging"
	default:
		return "development"
	}
}

func writeKubeconfig(t *testing.T, path, serverURL string) {
	t.Helper()
	content := fmt.Sprintf(`apiVersion: v1
kind: Config
clusters:
- cluster:
    server: %s
    insecure-skip-tls-verify: true
  name: stress
contexts:
- context:
    cluster: stress
    user: stress
  name: stress
current-context: stress
users:
- name: stress
  user: {}
`, serverURL)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func writeWebhookConfig(t *testing.T, cfgPath, kubeconfigPath, serverURL string) {
	t.Helper()
	apiKeyPath := filepath.Join(filepath.Dir(cfgPath), "api-key")
	require.NoError(t, os.WriteFile(apiKeyPath, []byte("test-key"), 0o600))
	sinkURL := fmt.Sprintf("%s/v1/container-metrics?cloud_account_id=test&cluster_name=stress&region=us-west-2", serverURL)
	content := fmt.Sprintf(`cloud_account_id: test
region: us-west-2
cluster_name: stress
api_key_path: %s
host: %s
destination: %s
logging:
  level: warn
remote_write:
  host: %s
  send_interval: 500ms
  max_bytes_per_send: 500000
  send_timeout: 30s
  max_retries: 3
k8s_client:
  kube_config: %s
  pagination_limit: 500
database:
  retention_time: 24h
  cleanup_interval: 3h
  batch_update_size: 500
server:
  port: 0
  read_timeout: 10s
  write_timeout: 10s
  idle_timeout: 120s
  profiling: false
filters:
  labels:
    enabled: true
    patterns:
      - "^environment$"
      - "^team$"
      - "^cost-center$"
      - "^app$"
    resources:
      namespaces: true
      pods: true
      deployments: false
      jobs: false
      cronjobs: false
      statefulsets: false
      daemonsets: false
      nodes: false
  annotations:
    enabled: false
    patterns: []
    resources:
      namespaces: false
      pods: false
`, apiKeyPath, serverURL, sinkURL, sinkURL, kubeconfigPath)
	require.NoError(t, os.WriteFile(cfgPath, []byte(content), 0o644))
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

func readRSS(pid int) uint64 {
	switch runtime.GOOS {
	case "linux":
		data, err := os.ReadFile(fmt.Sprintf("/proc/%d/statm", pid))
		if err != nil {
			return 0
		}
		fields := strings.Fields(string(data))
		if len(fields) < 2 {
			return 0
		}
		pages, _ := strconv.ParseUint(fields[1], 10, 64)
		return pages * uint64(os.Getpagesize())
	case "darwin":
		out, err := exec.Command("ps", "-o", "rss=", "-p", strconv.Itoa(pid)).Output()
		if err != nil {
			return 0
		}
		kb, _ := strconv.ParseUint(strings.TrimSpace(string(out)), 10, 64)
		return kb * 1024
	default:
		return 0
	}
}

func humanBytes(b uint64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%dB", b)
	}
	div, exp := uint64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f%ciB", float64(b)/float64(div), "KMGTPE"[exp])
}

// sync pool to avoid lint complaints about the mutex in page
var _ = sync.Mutex{}
