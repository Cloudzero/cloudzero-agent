package handlers

import (
	"io"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-obvious/server"
	"github.com/go-obvious/server/api"
	"github.com/go-obvious/server/request"

	"github.com/cloudzero/cloudzero-insights-controller/app/domain"
	"github.com/cloudzero/cloudzero-insights-controller/app/validation"
)

const MaxPayloadSize = 16 * 1024 * 1024

type RemoteWriteAPI struct {
	api.Service
	metrics *domain.MetricCollector
}

func NewRemoteWriteAPI(base string, d *domain.MetricCollector) *RemoteWriteAPI {
	a := &RemoteWriteAPI{
		metrics: d,
		Service: api.Service{
			APIName: "remotewrite",
			Mounts:  map[string]*chi.Mux{},
		},
	}
	a.Service.Mounts[base] = a.Routes()
	return a
}

func (a *RemoteWriteAPI) Register(app server.Server) error {
	if err := a.Service.Register(app); err != nil {
		return err
	}
	return nil
}

func (a *RemoteWriteAPI) Routes() *chi.Mux {
	r := chi.NewRouter()
	r.Post("/", a.PostMetrics)
	return r
}

func (a *RemoteWriteAPI) PostMetrics(w http.ResponseWriter, r *http.Request) {
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
