// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package inspector

import (
	"bytes"
	"io"
	"net/http"
	"reflect"
	"testing"
)

func Test_responseData_IsJSON(t *testing.T) {
	tests := []struct {
		name string
		resp *http.Response
		want bool
	}{
		{
			name: "json",
			resp: &http.Response{
				Header: http.Header{"Content-Type": []string{"application/json"}},
			},
			want: true,
		},
		{
			name: "not-json",
			resp: &http.Response{},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &responseData{
				resp: tt.resp,
			}
			if got := resp.IsJSON(); got != tt.want {
				t.Errorf("responseData.IsJSON() = %v, want %v", got, tt.want)
			}
			// Hit the cache.
			if got := resp.IsJSON(); got != tt.want {
				t.Errorf("responseData.IsJSON() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_responseData_jsonBody(t *testing.T) {
	tests := []struct {
		name    string
		resp    *http.Response
		want    any
		wantErr bool
	}{
		{
			name: "json",
			resp: &http.Response{
				Header: http.Header{"Content-Type": []string{"application/json"}},
				Body:   io.NopCloser(bytes.NewBufferString(`{"foo": "bar"}`)),
			},
			want: map[string]any{"foo": "bar"},
		},
		{
			name:    "not-json",
			resp:    &http.Response{},
			wantErr: true,
		},
		{
			name: "invalid-json",
			resp: &http.Response{
				Header: http.Header{"Content-Type": []string{"application/json"}},
				Body:   io.NopCloser(bytes.NewBufferString(`{"foo": "bar"`)),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &responseData{
				resp: tt.resp,
			}
			got, err := resp.jsonBody()
			if (err != nil) != tt.wantErr {
				t.Errorf("responseData.jsonBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("responseData.jsonBody() = %v, want %v", got, tt.want)
			}
			// Test the cache.
			_, err = resp.jsonBody()
			if (err != nil) != tt.wantErr {
				t.Errorf("responseData.jsonBody() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}
