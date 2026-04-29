// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package profiling provides a self-contained pprof profiling server and
// periodic profile dump facility for CloudZero Agent components.
//
// Usage:
//
//	stop := profiling.Start(profiling.Options{
//	    Enabled:      true,
//	    Port:         6060,
//	    Dir:          "/var/lib/agent/profiles",
//	    HeapInterval: 30 * time.Second,
//	    CPUInterval:  5 * time.Minute,
//	    CPUDuration:  30 * time.Second,
//	})
//	defer stop()
//
// When Enabled is false, Start returns immediately with a no-op stop function
// and starts no goroutines or listeners.
package profiling

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof" //nolint:gosec // intentional: we proxy DefaultServeMux through a private mux, not exposed unless profiling is enabled
	"os"
	"path/filepath"
	"runtime/pprof"
	"time"

	"github.com/rs/zerolog/log"
)

// Options configures the profiling server and periodic profile dumps.
type Options struct {
	// Enabled is the master switch. Nothing runs unless this is true.
	Enabled bool

	// Port is the TCP port for the pprof HTTP server. Defaults to 6060 when zero.
	Port int

	// Dir is the output directory for periodic profile dumps. Heap and CPU dumps
	// are disabled when Dir is empty.
	Dir string

	// HeapInterval is the interval between heap profile snapshots. Defaults to
	// 30 seconds when zero. Heap dumps are only written when Dir is set.
	HeapInterval time.Duration

	// CPUInterval is the interval between CPU capture sessions. CPU profiling is
	// disabled when CPUInterval is zero or Dir is empty.
	CPUInterval time.Duration

	// CPUDuration is the length of each CPU capture window. Defaults to 30 seconds
	// when zero (only meaningful when CPUInterval > 0).
	CPUDuration time.Duration
}

const (
	defaultProfilingPort = 6060
	defaultHeapInterval  = 30 * time.Second
	defaultCPUDuration   = 30 * time.Second
)

// applyDefaults fills in zero-value fields with their documented defaults.
func applyDefaults(opts *Options) {
	if opts.Port == 0 {
		opts.Port = defaultProfilingPort
	}
	if opts.HeapInterval == 0 {
		opts.HeapInterval = defaultHeapInterval
	}
	if opts.CPUDuration == 0 {
		opts.CPUDuration = defaultCPUDuration
	}
}

// Start launches the profiling subsystem according to opts and returns a stop
// function that shuts everything down cleanly.
//
// If opts.Enabled is false, Start is a no-op and the returned function does
// nothing. No goroutines are started and no ports are opened.
func Start(opts Options) (stop func()) {
	if !opts.Enabled {
		return func() {}
	}

	applyDefaults(&opts)

	done := make(chan struct{})

	// --- pprof HTTP server ---
	addr := fmt.Sprintf(":%d", opts.Port)
	mux := http.NewServeMux()
	// net/http/pprof registers its handlers on http.DefaultServeMux, so we
	// proxy through to it so the caller's server mux is unaffected.
	mux.Handle("/debug/pprof/", http.DefaultServeMux)

	srv := &http.Server{ //nolint:gosec // G112: profiling server is internal-only, not exposed to untrusted clients
		Addr:    addr,
		Handler: mux,
	}

	go func() {
		log.Info().Str("addr", addr).Msg("profiling: pprof HTTP server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error().Err(err).Msg("profiling: pprof HTTP server error")
		}
	}()

	// --- heap dump goroutine ---
	if opts.Dir != "" {
		go func() {
			ticker := time.NewTicker(opts.HeapInterval)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case t := <-ticker.C:
					writeHeapProfile(opts.Dir, t)
				}
			}
		}()
		log.Info().
			Str("dir", opts.Dir).
			Dur("interval", opts.HeapInterval).
			Msg("profiling: heap dump goroutine started")
	}

	// --- CPU profile goroutine ---
	if opts.CPUInterval > 0 && opts.Dir != "" {
		go func() {
			ticker := time.NewTicker(opts.CPUInterval)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case t := <-ticker.C:
					writeCPUProfile(opts.Dir, t, opts.CPUDuration, done)
				}
			}
		}()
		log.Info().
			Str("dir", opts.Dir).
			Dur("interval", opts.CPUInterval).
			Dur("duration", opts.CPUDuration).
			Msg("profiling: CPU profile goroutine started")
	}

	return func() {
		log.Info().Msg("profiling: shutting down")
		close(done)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(ctx); err != nil {
			log.Error().Err(err).Msg("profiling: error during HTTP server shutdown")
		}
	}
}

// writeHeapProfile writes a heap profile to opts.Dir with a timestamped filename.
func writeHeapProfile(dir string, t time.Time) {
	name := filepath.Join(dir, "heap-"+t.UTC().Format("20060102-150405")+".pb.gz")
	f, err := os.Create(name)
	if err != nil {
		log.Error().Err(err).Str("path", name).Msg("profiling: failed to create heap profile file")
		return
	}
	defer f.Close()

	if err := pprof.WriteHeapProfile(f); err != nil {
		log.Error().Err(err).Str("path", name).Msg("profiling: failed to write heap profile")
		return
	}
	log.Debug().Str("path", name).Msg("profiling: heap profile written")
}

// writeCPUProfile captures a CPU profile of length duration and writes it to dir.
// done is checked before starting so a shutdown during the inter-tick gap is clean.
func writeCPUProfile(dir string, t time.Time, duration time.Duration, done <-chan struct{}) {
	// Don't start a new CPU capture if we've been asked to stop.
	select {
	case <-done:
		return
	default:
	}

	name := filepath.Join(dir, "cpu-"+t.UTC().Format("20060102-150405")+".pb.gz")
	f, err := os.Create(name)
	if err != nil {
		log.Error().Err(err).Str("path", name).Msg("profiling: failed to create CPU profile file")
		return
	}
	defer f.Close()

	if err := pprof.StartCPUProfile(f); err != nil {
		log.Error().Err(err).Str("path", name).Msg("profiling: failed to start CPU profile")
		return
	}

	// Wait for the capture window, but honour an early stop request.
	select {
	case <-done:
	case <-time.After(duration):
	}

	pprof.StopCPUProfile()
	log.Debug().Str("path", name).Msg("profiling: CPU profile written")
}
