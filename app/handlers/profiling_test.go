package handlers_test

import (
	"testing"

	"github.com/cloudzero/cloudzero-agent/app/handlers"
	"github.com/go-obvious/server/test"
	"github.com/stretchr/testify/assert"
)

func TestUnit_Handlers_Profile(t *testing.T) {
	profiling := handlers.NewProfilingAPI("/")

	tests := []struct {
		name               string
		path               string
		expectedStatusCode int
	}{
		{
			name:               "QueryIndex",
			path:               "/",
			expectedStatusCode: 200,
		},
		{
			name:               "QueryCMDLine",
			path:               "/cmdline",
			expectedStatusCode: 200,
		},
		{
			name:               "QueryProfile",
			path:               "/profile?seconds=1",
			expectedStatusCode: 200,
		},
		{
			name:               "QuerySymbol",
			path:               "/symbol",
			expectedStatusCode: 200,
		},
		{
			name:               "QueryTrace",
			path:               "/trace",
			expectedStatusCode: 200,
		},
		{
			name:               "QueryErr",
			path:               "/does/not/exist",
			expectedStatusCode: 404,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := createRequest("GET", tc.path, nil)
			resp, err := test.InvokeService(profiling.Service, tc.path, *req)
			assert.NoError(t, err)
			defer resp.Body.Close()
			assert.Equal(t, tc.expectedStatusCode, resp.StatusCode)
		})
	}
}
