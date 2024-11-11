package parallel_test

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/cloudzero/cirrus-remote-write/app/parallel"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name        string
		workerCount int
		expected    int
	}{
		{"NegativeWorkerCount", -1, runtime.NumCPU()},
		{"ZeroWorkerCount", 0, 2},
		{"PositiveWorkerCount", 5, 5},
		{"LessThanMinWorkers", 1, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := parallel.New(tt.workerCount)
			assert.NotNil(t, manager)
		})
	}
}
