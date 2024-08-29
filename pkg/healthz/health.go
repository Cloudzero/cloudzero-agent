package healthz

import (
	"net/http"
	"sync"
)

type HealthCheck func() error

type HealthChecker interface {
	EndpointHandler() http.HandlerFunc
}

// Register a health check function
// can be used to add specific health checks
func Register(name string, fn HealthCheck) {
	// get the interface and cast to internal type
	NewHealthz().(*checker).add(name, fn)
}

var (
	// global protected access to health checker
	// once to ensure singleton
	h    *checker
	once sync.Once
)

type checker struct {
	mu     sync.Mutex
	checks map[string]HealthCheck
}

func NewHealthz() HealthChecker {
	once.Do(func() {
		h = &checker{}
	})
	return h
}

func (x *checker) add(name string, fn HealthCheck) {
	// lock and unlock on return
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.checks == nil {
		x.checks = make(map[string]HealthCheck)
	}
	x.checks[name] = fn
}

func (x *checker) EndpointHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for name, check := range x.checks {
			if err := check(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(name + " failed: " + err.Error()))
				return
			}
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok")) // ignore return values
	}
}
