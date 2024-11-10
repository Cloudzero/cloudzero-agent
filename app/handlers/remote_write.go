package handlers

import (
	"io"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-obvious/server"
	"github.com/go-obvious/server/api"
	baseconf "github.com/go-obvious/server/config"
	"github.com/go-obvious/server/request"

	"github.com/cloudzero/cirrus-remote-write/app/config"
	"github.com/cloudzero/cirrus-remote-write/app/domain"
	"github.com/cloudzero/cirrus-remote-write/app/validation"
)

const MaxPayloadSize = 16 * 1024 * 1024

type RemoteWrite struct {
	api.Service
	metrics *domain.Metrics
	cfg     config.MetricServiceConfig
}

func NewRemoteWrite(base string, d *domain.Metrics) *RemoteWrite {
	a := &RemoteWrite{
		metrics: d,
		Service: api.Service{
			APIName: "remotewrite",
			Mounts:  map[string]*chi.Mux{},
		},
	}
	a.Service.Mounts[base] = a.Routes()
	baseconf.Register(&a.cfg)
	return a
}

func (a *RemoteWrite) Register(app server.Server) error {
	if err := a.Service.Register(app); err != nil {
		return err
	}
	return nil
}

func (a *RemoteWrite) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Get("/", a.GetAllMetrics)
	r.Post("/", a.PostMetrics)
	r.Post("/reset", a.PostResetMetrics)
	r.Get("/names", a.GetMetricNames)
	r.Get("/count", a.GetMetricCount)
	r.Get("/names/{name}", a.GetMetricName)
	return r
}

func (a *RemoteWrite) PostResetMetrics(w http.ResponseWriter, r *http.Request) {
	a.metrics.Flush(r.Context())
	request.Reply(r, w, nil, http.StatusNoContent)
}

func (a *RemoteWrite) GetAllMetrics(w http.ResponseWriter, r *http.Request) {
	metrics, err := a.metrics.AllMetrics(r.Context(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	request.Reply(r, w, metrics, http.StatusOK)
}

func (a *RemoteWrite) GetMetricNames(w http.ResponseWriter, r *http.Request) {
	names, err := a.metrics.GetMetricNames(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	request.Reply(r, w, names, http.StatusOK)
}

func (a *RemoteWrite) GetMetricName(w http.ResponseWriter, r *http.Request) {
	name := chi.URLParam(r, "name")
	if name == "" {
		request.Reply(r, w, "name is required", http.StatusBadRequest)
		return
	}
	names, err := a.metrics.GetMetricNames(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// create a unique set of metric names
	metricNames := make(map[string]struct{})
	for _, metric := range names {
		metricNames[metric] = struct{}{}
	}
	if _, ok := metricNames[name]; !ok {
		request.Reply(r, w, nil, http.StatusNotFound)
		return
	}
	request.Reply(r, w, name, http.StatusOK)
}

func (a *RemoteWrite) GetMetricCount(w http.ResponseWriter, r *http.Request) {
	metrics, err := a.metrics.AllMetrics(r.Context(), nil)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	request.Reply(r, w, len(metrics.Metrics), http.StatusOK)
}

func (a *RemoteWrite) PostMetrics(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	contentLen := r.ContentLength

	if contentLen <= 0 {
		request.Reply(r, w, "empty body", http.StatusOK)
		return
	}

	if contentLen > MaxPayloadSize {
		request.Reply(r, w, "too big", http.StatusOK)
		return
	}

	organizationID := r.Header.Get("organization_id")
	if organizationID == "" {
		request.Reply(r, w, "organization_id is required", http.StatusBadRequest)
		return
	}

	clusterName := request.QS(r, "cluster_name")
	if err := validation.ValidateClusterName(clusterName); err != nil {
		request.Reply(r, w, "cluster_name is required", http.StatusBadRequest)
		return
	}

	cleanAccountID, err := validation.ValidateCloudAccountID(request.QS(r, "cloud_account_id"))
	if err != nil || cleanAccountID == "" {
		request.Reply(r, w, "cloud_account_id is required", http.StatusBadRequest)
		return
	}

	contentType := r.Header.Get("Content-Type")
	encodingType := r.Header.Get("Content-Encoding")
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	stats, err := a.metrics.PutMetrics(r.Context(), contentType, encodingType, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if stats != nil {
		stats.SetHeaders(w)
	}

	request.Reply(r, w, nil, http.StatusNoContent)
}
