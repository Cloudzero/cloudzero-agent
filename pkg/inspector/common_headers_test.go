package inspector

import (
	"bytes"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/rs/zerolog"
)

func Test_addCommonHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers http.Header
		want    map[string]any
	}{
		{
			name: "exact-match",
			headers: http.Header{
				"Content-Type": []string{"application/json"},
			},
			want: map[string]any{
				"Content-Type": "application/json",
			},
		},
		{
			name: "exact-no-match",
			headers: http.Header{
				"X-Content-Type": []string{"application/json"},
			},
			want: map[string]any{},
		},
		{
			name: "contains-match",
			headers: http.Header{
				"X-Foo-Request-Id-Value": []string{"123"},
			},
			want: map[string]any{
				"X-Foo-Request-Id-Value": "123",
			},
		},
		{
			name: "prefix-match",
			headers: http.Header{
				"X-Amz-Apigw-Id": []string{"Cb9XdFv0oAMEdjQ="},
			},
			want: map[string]any{
				"X-Amz-Apigw-Id": "Cb9XdFv0oAMEdjQ=",
			},
		},
		{
			name: "prefix-no-match",
			headers: http.Header{
				"X-X-Amz-Apigw-Id": []string{"Cb9XdFv0oAMEdjQ="},
			},
			want: map[string]any{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loggerOutput := bytes.Buffer{}
			logger := zerolog.New(&loggerOutput)

			logger = addCommonHeaders(logger, tt.headers)
			logger.Info().Msg("Hello")

			logData := map[string]any{}
			err := json.Unmarshal(loggerOutput.Bytes(), &logData)
			if err != nil {
				t.Errorf("failed to unmarshal log output: %v", err)
			}

			delete(logData, "message")
			delete(logData, "level")

			if diff := cmp.Diff(logData, tt.want); diff != "" {
				t.Errorf("Inspector.Inspect() log output mismatch (-got +want):\n%s", diff)
			}
		})
	}
}
