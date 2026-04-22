// SPDX-FileCopyrightText: Copyright (c) 2016-2026, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"net/http"
	"strconv"
)

// WriteResponseStats reports the counts returned by the Prometheus Remote-Write
// 2.0 response (https://prometheus.io/docs/specs/prw/remote_write_spec_2_0/).
//
// This is a local replacement for prometheus/prometheus/storage/remote's
// WriteResponseStats, kept in-repo to avoid the transitive dependency on
// docker/docker that prometheus/prometheus's storage/remote package pulls in
// via its config package's discovery/moby import.
type WriteResponseStats struct {
	Samples    int
	Histograms int
	Exemplars  int
	Confirmed  bool
}

// SetHeaders writes the Prometheus Remote-Write 2.0 response headers that
// report the counts of successfully written samples, histograms, and exemplars.
// Call this before ResponseWriter.WriteHeader or Write.
func (s WriteResponseStats) SetHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("X-Prometheus-Remote-Write-Written-Samples", strconv.Itoa(s.Samples))
	h.Set("X-Prometheus-Remote-Write-Written-Histograms", strconv.Itoa(s.Histograms))
	h.Set("X-Prometheus-Remote-Write-Written-Exemplars", strconv.Itoa(s.Exemplars))
}
