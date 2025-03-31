package filter_test

import (
	"testing"

	util "github.com/cloudzero/cloudzero-insights-controller/app/domain/filter"
)

func TestFilterChecker_Test(t *testing.T) {
	type args struct {
		value string
	}
	tests := []struct {
		name    string
		filters []util.FilterEntry
		value   string
		want    bool
		wantErr bool
	}{
		{
			name: "unknown match type",
			filters: []util.FilterEntry{
				{
					Pattern: "test",
					Match:   util.FilterMatchType("gibberish"),
				},
			},
			value:   "test",
			want:    false,
			wantErr: true,
		},
		{
			name: "exact match",
			filters: []util.FilterEntry{
				{
					Pattern: "test",
					Match:   util.FilterMatchTypeExact,
				},
			},
			value: "test",
			want:  true,
		},
		{
			name: "no exact match",
			filters: []util.FilterEntry{
				{
					Pattern: "test",
					Match:   util.FilterMatchTypeExact,
				},
			},
			value: "testing",
			want:  false,
		},
		{
			name: "prefix match",
			filters: []util.FilterEntry{
				{
					Pattern: "test",
					Match:   util.FilterMatchTypePrefix,
				},
			},
			value: "testing",
			want:  true,
		},
		{
			name: "no prefix match",
			filters: []util.FilterEntry{
				{
					Pattern: "ing",
					Match:   util.FilterMatchTypePrefix,
				},
			},
			value: "testing",
			want:  false,
		},
		{
			name: "suffix match",
			filters: []util.FilterEntry{
				{
					Pattern: "ing",
					Match:   util.FilterMatchTypeSuffix,
				},
			},
			value: "testing",
			want:  true,
		},
		{
			name: "no suffix match",
			filters: []util.FilterEntry{
				{
					Pattern: "test",
					Match:   util.FilterMatchTypeSuffix,
				},
			},
			value: "testing",
			want:  false,
		},
		{
			name: "contains match",
			filters: []util.FilterEntry{
				{
					Pattern: "est",
					Match:   util.FilterMatchTypeContains,
				},
			},
			value: "testing",
			want:  true,
		},
		{
			name: "no contains match",
			filters: []util.FilterEntry{
				{
					Pattern: "aoeu",
					Match:   util.FilterMatchTypeContains,
				},
			},
			value: "testing",
			want:  false,
		},
		{
			name: "contains match",
			filters: []util.FilterEntry{
				{
					Pattern: "est",
					Match:   util.FilterMatchTypeContains,
				},
			},
			value: "testing",
			want:  true,
		},
		{
			name: "no contains match",
			filters: []util.FilterEntry{
				{
					Pattern: "aoeu",
					Match:   util.FilterMatchTypeContains,
				},
			},
			value: "testing",
			want:  false,
		},
		{
			name: "regex match",
			filters: []util.FilterEntry{
				{
					Pattern: "^test.ng$",
					Match:   util.FilterMatchTypeRegex,
				},
			},
			value: "testing",
			want:  true,
		},
		{
			name: "no regex match",
			filters: []util.FilterEntry{
				{
					Pattern: "^test.ng$",
					Match:   util.FilterMatchTypeRegex,
				},
			},
			value: "test",
			want:  false,
		},
		{
			name: "bad regex",
			filters: []util.FilterEntry{
				{
					Pattern: "he[llo",
					Match:   util.FilterMatchTypeRegex,
				},
			},
			value:   "testing",
			want:    false,
			wantErr: true,
		},
		{
			name:    "empty filters",
			filters: []util.FilterEntry{},
			value:   "testing",
			want:    true,
		},
		{
			name:    "nil filters",
			filters: nil,
			value:   "testing",
			want:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker, err := util.NewFilterChecker(tt.filters)

			if tt.wantErr != (err != nil) {
				t.Errorf("filterChecker.Test() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				return
			}

			if got := checker.Test(tt.value); got != tt.want {
				t.Errorf("filterChecker.Test() = %v, want %v", got, tt.want)
			}
		})
	}
}
