// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package inspector

import (
	"testing"
)

func Test_gojqQueryCache_JSONMatch(t *testing.T) {
	tests := []struct {
		name          string
		query         string
		data          any
		want          bool
		wantErr       bool
		wantResultErr bool
	}{
		{
			name:  "api-key-error",
			query: ".message == \"User is not authorized to access this resource\"",
			data: map[string]any{
				"message": "User is not authorized to access this resource",
			},
			want: true,
		},
		{
			name:  "not-api-key-error",
			query: ".message == \"User is not authorized to access this resource\"",
			data: map[string]any{
				"message": "I'm a made-up message telling you that you're not authorized to access this resource",
			},
			want: false,
		},
		{
			name:    "bad-query",
			query:   "$#^KI@a,.i4a,",
			wantErr: true,
			want:    false,
		},
		{
			name:    "non-boolean-result",
			query:   ".foo",
			data:    map[string]any{"foo": "bar"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := gojqCache.JSONMatch(tt.query, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("gojqQueryCache.JSONQuery() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if res != tt.want {
				t.Errorf("gojqQueryCache.JSONMatch() result = %v, want %v", res, tt.want)
			}
		})
	}
}
