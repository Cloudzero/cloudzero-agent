// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package main provides the regurgitator CLI tool for transcoding metrics between formats.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/cloudzero/cloudzero-agent/app/domain/metricio"
	"github.com/cloudzero/cloudzero-agent/app/types"
	"github.com/gogo/protobuf/proto"
	"github.com/golang/snappy"
	"github.com/jpillora/backoff"
	"github.com/prometheus/prometheus/prompb"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

const (
	channelBufferSize   = 8096
	defaultBatchSize    = 10000
	maxRetries          = 10
	maxBackoffDuration  = 30 * time.Second
	readHeaderTimeout   = 10 * time.Second
	extensionBrotli     = ".br"
	extensionCSV        = ".csv"
	extensionJSON       = ".json"
	extensionJSONBrotli = ".json.br"
	extensionParquet    = ".parquet"
)

var (
	outputs   []string
	listen    string
	recursive bool
	readers   int
	writers   int
	dryRun    bool
	logLevel  string
)

// Writer is the interface for output writers.
type Writer interface {
	WriteOne(metric types.Metric) error
	Close() error
}

// remoteWriterAdapter adapts metricio.RemoteWriter to the Writer interface.
type remoteWriterAdapter struct {
	ctx       context.Context
	writer    *metricio.RemoteWriter
	batchSize int
	batch     []prompb.TimeSeries
	backoff   *backoff.Backoff
}

func newRemoteWriterAdapter(ctx context.Context, url string) *remoteWriterAdapter {
	return &remoteWriterAdapter{
		ctx:       ctx,
		writer:    metricio.NewRemoteWriter(url),
		batchSize: defaultBatchSize,
		batch:     make([]prompb.TimeSeries, 0, defaultBatchSize),
		backoff: &backoff.Backoff{
			Min:    100 * time.Millisecond,
			Max:    maxBackoffDuration,
			Factor: 2,
			Jitter: true,
		},
	}
}

func (w *remoteWriterAdapter) WriteOne(metric types.Metric) error {
	ts := metricio.MetricToTimeSeries(metric)
	w.batch = append(w.batch, ts)

	if len(w.batch) >= w.batchSize {
		return w.flush()
	}
	return nil
}

func (w *remoteWriterAdapter) flush() error {
	if len(w.batch) == 0 {
		return nil
	}

	var lastErr error

	for attempt := range maxRetries {
		// Check for cancellation before each attempt
		select {
		case <-w.ctx.Done():
			return w.ctx.Err()
		default:
		}

		err := w.writer.Write(w.ctx, w.batch)
		if err == nil {
			w.batch = make([]prompb.TimeSeries, 0, w.batchSize)
			w.backoff.Reset()
			return nil
		}

		lastErr = err
		dur := w.backoff.Duration()
		log.Warn().
			Err(err).
			Int("attempt", attempt+1).
			Dur("backoff", dur).
			Msg("remote write failed, retrying")

		// Use select for interruptible sleep
		select {
		case <-w.ctx.Done():
			return w.ctx.Err()
		case <-time.After(dur):
		}
	}

	return fmt.Errorf("failed after %d retries: %w", maxRetries, lastErr)
}

func (w *remoteWriterAdapter) Close() error {
	return w.flush()
}

// jsonWriterAdapter adapts metricio.JSONWriter to the Writer interface.
type jsonWriterAdapter struct {
	writer *metricio.JSONWriter
}

func (w *jsonWriterAdapter) WriteOne(metric types.Metric) error {
	return w.writer.WriteOne(metric)
}

func (w *jsonWriterAdapter) Close() error {
	return w.writer.Close()
}

// parquetWriterAdapter adapts metricio.ParquetWriter to the Writer interface.
type parquetWriterAdapter struct {
	writer *metricio.ParquetWriter
}

func (w *parquetWriterAdapter) WriteOne(metric types.Metric) error {
	return w.writer.WriteOne(metric)
}

func (w *parquetWriterAdapter) Close() error {
	return w.writer.Close()
}

// csvWriterAdapter adapts metricio.CSVWriter to the Writer interface.
type csvWriterAdapter struct {
	writer *metricio.CSVWriter
}

func (w *csvWriterAdapter) WriteOne(metric types.Metric) error {
	return w.writer.WriteOne(metric)
}

func (w *csvWriterAdapter) Close() error {
	return w.writer.Close()
}

// dryRunWriter is a no-op writer for dry-run mode.
type dryRunWriter struct{}

func (w *dryRunWriter) WriteOne(metric types.Metric) error {
	log.Trace().
		Str("id", metric.ID.String()).
		Str("metric_name", metric.MetricName).
		Str("node_name", metric.NodeName).
		Str("cluster_name", metric.ClusterName).
		Str("cloud_account_id", metric.CloudAccountID).
		Time("timestamp", metric.TimeStamp).
		Time("created_at", metric.CreatedAt).
		Str("value", metric.Value).
		Interface("labels", metric.Labels).
		Msg("dry-run: would write metric")
	return nil
}

func (w *dryRunWriter) Close() error {
	return nil
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "regurgitator [options] -o <output> [input-files...]",
		Short: "Transcode metrics between formats",
		Long: `Regurgitator reads metrics from various formats (CSV, Parquet, JSON, JSON.br)
and writes them to one or more outputs (remote_write URL or file).

Output destinations are specified with -o/--output flags. URLs starting with
http:// or https:// are treated as Prometheus remote_write endpoints. Other
values are treated as file paths with format determined by extension.`,
		RunE: run,
	}

	// Output destinations (repeatable)
	rootCmd.Flags().StringArrayVarP(&outputs, "output", "o", nil, "Output destination (file or URL, can be repeated)")
	_ = rootCmd.MarkFlagRequired("output")

	// Input mode
	rootCmd.Flags().StringVar(&listen, "listen", "", "HTTP address to listen for remote_write")

	// Processing flags
	rootCmd.Flags().BoolVarP(&recursive, "recursive", "r", false, "Recursively discover files in directories")
	rootCmd.Flags().IntVar(&readers, "readers", 1, "Number of parallel file readers")
	rootCmd.Flags().IntVar(&writers, "writers", 1, "Number of parallel writers per output")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Parse input without writing output")

	// Logging
	rootCmd.Flags().StringVar(&logLevel, "log-level", "warn", "Log level (trace, debug, info, warn, error)")

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	// Setup logging
	level, err := zerolog.ParseLevel(logLevel)
	if err != nil {
		level = zerolog.WarnLevel
	}
	zerolog.SetGlobalLevel(level)
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	inputs := args // All positional args are input files

	// Validate
	if listen == "" && len(inputs) == 0 {
		return errors.New("must specify input files or --listen")
	}
	if listen != "" && len(inputs) > 0 {
		return errors.New("cannot use both --listen and input files")
	}

	// Create a cancellable context (kept as fallback, but closing channel should suffice)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Create main input channel
	inputCh := make(chan types.Metric, channelBufferSize)

	// Set up signal handling - on SIGINT we close the shutdown channel to trigger
	// graceful shutdown, allowing all in-flight data to drain through the pipeline.
	// After a grace period, we cancel the context to force exit.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	shutdown := make(chan struct{})
	go func() {
		sig := <-sigCh
		log.Info().Str("signal", sig.String()).Msg("received signal, initiating graceful shutdown")
		close(shutdown)

		// Give a grace period for graceful drain, then force cancel
		select {
		case <-time.After(5 * time.Second):
			log.Warn().Msg("grace period expired, forcing shutdown")
			cancel()
		case <-sigCh:
			log.Warn().Msg("received second signal, forcing shutdown")
			cancel()
		}
	}()

	// Create per-output channels and writers
	outputChs := make([]chan types.Metric, len(outputs))
	fanOutChs := make([]chan<- types.Metric, len(outputs))
	var writerWg sync.WaitGroup

	for i, output := range outputs {
		outputChs[i] = make(chan types.Metric, channelBufferSize)
		fanOutChs[i] = outputChs[i]

		// Multiple writers only make sense for remote write (HTTP).
		// File writers write to a single file and can't be parallelized.
		numWriters := 1
		if isRemoteWriteURL(output) {
			numWriters = writers
		}

		for w := range numWriters {
			writer, err := createWriter(ctx, output)
			if err != nil {
				return fmt.Errorf("failed to create writer for %s: %w", output, err)
			}

			writerWg.Add(1)
			go func(ch <-chan types.Metric, w Writer, outputName string, writerID int) {
				defer writerWg.Done()
				defer w.Close()

				count := 0
				for metric := range ch {
					if err := w.WriteOne(metric); err != nil {
						log.Error().Err(err).Str("output", outputName).Int("writer", writerID).Msg("failed to write metric")
					}
					count++

					log.Trace().
						Str("id", metric.ID.String()).
						Str("metric_name", metric.MetricName).
						Str("node_name", metric.NodeName).
						Str("cluster_name", metric.ClusterName).
						Str("cloud_account_id", metric.CloudAccountID).
						Time("timestamp", metric.TimeStamp).
						Time("created_at", metric.CreatedAt).
						Str("value", metric.Value).
						Interface("labels", metric.Labels).
						Str("output", outputName).
						Int("writer", writerID).
						Msg("wrote metric")
				}

				log.Info().
					Str("output", outputName).
					Int("writer", writerID).
					Int("metrics", count).
					Msg("writer completed")
			}(outputChs[i], writer, output, w)
		}
	}

	// Start fan-out goroutine - exits when input channel closes
	var fanOutWg sync.WaitGroup
	fanOutWg.Add(1)
	go func() {
		defer fanOutWg.Done()
		FanOut(ctx, inputCh, fanOutChs)
	}()

	// Start readers - they close inputCh when done or when shutdown is signaled
	if listen != "" {
		if err := startHTTPServer(shutdown, listen, inputCh); err != nil {
			return err
		}
	} else {
		if err := startFileReaders(shutdown, inputs, inputCh); err != nil {
			return err
		}
	}

	// Wait for fan-out to complete
	fanOutWg.Wait()

	// Wait for all writers to complete
	writerWg.Wait()

	return nil
}

// createWriter creates appropriate writer based on output destination.
func createWriter(ctx context.Context, output string) (Writer, error) {
	if dryRun {
		return &dryRunWriter{}, nil
	}

	if isRemoteWriteURL(output) {
		return newRemoteWriterAdapter(ctx, output), nil
	}

	ext := strings.ToLower(filepath.Ext(output))
	if ext == extensionBrotli && strings.HasSuffix(output, extensionJSONBrotli) {
		ext = extensionJSONBrotli
	}

	switch ext {
	case extensionParquet:
		pw, err := metricio.NewParquetWriter(output)
		if err != nil {
			return nil, err
		}
		return &parquetWriterAdapter{writer: pw}, nil
	case extensionJSON:
		jw, err := metricio.NewJSONWriter(output)
		if err != nil {
			return nil, err
		}
		return &jsonWriterAdapter{writer: jw}, nil
	case extensionJSONBrotli:
		jw, err := metricio.NewJSONWriter(output)
		if err != nil {
			return nil, err
		}
		return &jsonWriterAdapter{writer: jw}, nil
	case extensionCSV:
		cw, err := metricio.NewCSVWriter(output)
		if err != nil {
			return nil, err
		}
		return &csvWriterAdapter{writer: cw}, nil
	default:
		return nil, fmt.Errorf("unsupported output format: %s", ext)
	}
}

// isRemoteWriteURL returns true if the output is a remote write URL.
func isRemoteWriteURL(output string) bool {
	return strings.HasPrefix(output, "http://") || strings.HasPrefix(output, "https://")
}

// FanOut distributes items from an input channel to all output channels.
// It runs until the input channel is closed or context is canceled.
func FanOut[T any](ctx context.Context, input <-chan T, outputs []chan<- T) {
	defer func() {
		for _, out := range outputs {
			close(out)
		}
	}()

	for {
		select {
		case <-ctx.Done():
			return
		case item, ok := <-input:
			if !ok {
				return
			}
			for _, out := range outputs {
				select {
				case <-ctx.Done():
					return
				case out <- item:
				}
			}
		}
	}
}

// startHTTPServer starts an HTTP server that receives remote_write requests.
func startHTTPServer(shutdown <-chan struct{}, addr string, ch chan<- types.Metric) error {
	defer close(ch)

	var metricCount int64
	var mu sync.Mutex

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		r.Body.Close()
		if err != nil {
			log.Error().Err(err).Msg("failed to read request body")
			http.Error(w, "failed to read body", http.StatusBadRequest)
			return
		}

		// Decompress snappy
		decompressed, err := snappy.Decode(nil, body)
		if err != nil {
			log.Error().Err(err).Msg("failed to decompress snappy")
			http.Error(w, "failed to decompress", http.StatusBadRequest)
			return
		}

		// Parse protobuf
		var writeReq prompb.WriteRequest
		if err := proto.Unmarshal(decompressed, &writeReq); err != nil {
			log.Error().Err(err).Msg("failed to parse protobuf")
			http.Error(w, "failed to parse protobuf", http.StatusBadRequest)
			return
		}

		// Convert and send to channel
		count := 0
		for _, ts := range writeReq.Timeseries {
			metrics := metricio.TimeSeriesToMetrics(ts)
			for _, m := range metrics {
				ch <- m
				count++
			}
		}

		mu.Lock()
		metricCount += int64(count)
		mu.Unlock()

		log.Debug().Int("metrics", count).Int64("total", metricCount).Msg("received remote_write batch")

		w.WriteHeader(http.StatusNoContent)
	})

	server := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		log.Info().Str("addr", addr).Msg("starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	// Wait for shutdown signal or server error
	select {
	case <-shutdown:
		log.Info().Msg("shutting down HTTP server")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("failed to shutdown server gracefully")
		}
	case err := <-errCh:
		if err != nil {
			return fmt.Errorf("HTTP server error: %w", err)
		}
	}

	log.Info().Int64("total_metrics", metricCount).Msg("HTTP server stopped")
	return nil
}

// startFileReaders reads metrics from input files and sends them to the channel.
func startFileReaders(shutdown <-chan struct{}, inputs []string, ch chan<- types.Metric) error {
	defer close(ch)

	// Expand directories if recursive flag is set
	files, err := expandInputs(inputs)
	if err != nil {
		return err
	}

	log.Debug().Int("files", len(files)).Msg("starting file readers")

	// Create a semaphore for parallel readers
	sem := make(chan struct{}, readers)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var firstErr error

	// Note: We don't actively check shutdown here. The shutdown channel
	// is used by the HTTP server mode. For file readers, we let in-flight
	// reads complete naturally - they'll finish quickly and then the
	// deferred close(ch) will trigger the pipeline to drain.
	_ = shutdown

	for _, file := range files {
		wg.Add(1)
		go func(path string) {
			defer wg.Done()

			sem <- struct{}{}
			defer func() { <-sem }()

			log.Info().Str("file", path).Msg("processing file")

			if err := readFile(path, ch); err != nil {
				log.Error().Err(err).Str("file", path).Msg("failed to read file")
				mu.Lock()
				if firstErr == nil {
					firstErr = err
				}
				mu.Unlock()
			}
		}(file)
	}

	wg.Wait()
	return firstErr
}

// expandInputs expands directories into file lists.
func expandInputs(inputs []string) ([]string, error) {
	var files []string

	for _, input := range inputs {
		info, err := os.Stat(input)
		if err != nil {
			return nil, fmt.Errorf("failed to stat %s: %w", input, err)
		}

		if info.IsDir() {
			if !recursive {
				return nil, fmt.Errorf("%s is a directory (use -r to process directories)", input)
			}

			err := filepath.Walk(input, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && isSupportedInputFile(path) {
					files = append(files, path)
				}
				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("failed to walk directory %s: %w", input, err)
			}
		} else {
			files = append(files, input)
		}
	}

	return files, nil
}

// isSupportedInputFile checks if a file has a supported extension.
func isSupportedInputFile(path string) bool {
	return metricio.IsSupportedFile(path)
}

// readFile reads metrics from a single file and sends them to the channel.
func readFile(path string, ch chan<- types.Metric) error {
	ext := strings.ToLower(filepath.Ext(path))
	if strings.HasSuffix(path, extensionJSONBrotli) {
		ext = extensionJSONBrotli
	}

	switch ext {
	case extensionCSV:
		return readCSVFile(path, ch)
	case extensionParquet:
		return readParquetFile(path, ch)
	case extensionJSON, extensionJSONBrotli:
		return readJSONFile(path, ch)
	default:
		return fmt.Errorf("unsupported input format: %s", ext)
	}
}

// readCSVFile reads metrics from a CSV file.
func readCSVFile(path string, ch chan<- types.Metric) error {
	reader := metricio.NewCSVReader(defaultBatchSize)

	count := 0
	err := reader.ReadCSVFile(path, func(timeSeries []prompb.TimeSeries) error {
		for _, ts := range timeSeries {
			metrics := metricio.TimeSeriesToMetrics(ts)
			for _, m := range metrics {
				ch <- m
				count++
			}
		}

		log.Debug().
			Str("file", path).
			Int("batch_size", len(timeSeries)).
			Msg("processed CSV batch")

		return nil
	})

	if err == nil {
		log.Info().Str("file", path).Int("metrics", count).Msg("completed CSV file")
	}
	return err
}

// readParquetFile reads metrics from a Parquet file.
func readParquetFile(path string, ch chan<- types.Metric) error {
	reader := metricio.NewParquetReader(defaultBatchSize)

	count := 0
	err := reader.ReadFile(path, func(parquetMetrics []types.ParquetMetric) error {
		for _, pm := range parquetMetrics {
			ch <- pm.Metric()
			count++
		}

		log.Debug().
			Str("file", path).
			Int("batch_size", len(parquetMetrics)).
			Msg("processed Parquet batch")

		return nil
	})

	if err == nil {
		log.Info().Str("file", path).Int("metrics", count).Msg("completed Parquet file")
	}
	return err
}

// readJSONFile reads metrics from a JSON or JSON.br file.
func readJSONFile(path string, ch chan<- types.Metric) error {
	reader := metricio.NewJSONReader(defaultBatchSize)

	count := 0
	err := reader.ReadJSONFile(path, func(metrics []types.Metric) error {
		for _, m := range metrics {
			ch <- m
			count++
		}

		log.Debug().
			Str("file", path).
			Int("batch_size", len(metrics)).
			Msg("processed JSON batch")

		return nil
	})

	if err == nil {
		log.Info().Str("file", path).Int("metrics", count).Msg("completed JSON file")
	}
	return err
}
