package healthz

import (
	"net/http"
	"sync"
)

type HealthCheck func() error

var (
	h    *healthz // global variable to allow registration of new health checks
	once sync.Once
)

type healthz struct{}

func NewHealthz() *healthz {
	once.Do(func() {
		h = &healthz{}
	})
	return h
}

func (x *healthz) Register(fn healthz) {
	// TODO: implement
}

func (x *healthz) Handler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Default for now, later we can add more health checks
		// as we built this out
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}
}
