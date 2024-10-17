package storage_test

import (
	"testing"
	"time"

	"github.com/cloudzero/cirrus-remote-write/app/remotewrite/internal/storage"
)

func TestGetDateParts(t *testing.T) {
	type args struct {
		t time.Time
	}
	dt := time.Date(2024, 4, 20, 3, 15, 0, 0, time.UTC)
	tests := []struct {
		name   string
		want   string
		method func(t time.Time) string
	}{
		{
			name:   "Get Year",
			want:   "2024",
			method: storage.GetYear,
		},
		{
			name:   "Get Month",
			want:   "04",
			method: storage.GetMonth,
		},
		{
			name:   "Get Day",
			want:   "20",
			method: storage.GetDay,
		},
		{
			name:   "Get Hour",
			want:   "03",
			method: storage.GetHour,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.method(dt); got != tt.want {
				t.Errorf("GetYear() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildPath(t *testing.T) {
	type args struct {
		prefix         string
		t              time.Time
		cloudAccountID string
		clusterName    string
		extension      string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "Valid Path",
			args: args{
				prefix:         "org_metrics",
				t:              time.Date(2024, 4, 20, 3, 15, 0, 0, time.UTC),
				cloudAccountID: "123456789012",
				clusterName:    "test-cluster",
				extension:      "snappy",
			},
			want: "org_metrics/year=2024/month=04/day=20/hour=03/cloud_account_id=123456789012/cluster_name=test-cluster/1713582900000.snappy",
		},
		{
			name: "Different Time",
			args: args{
				prefix:         "raw_compressed",
				t:              time.Date(2023, 12, 31, 23, 59, 0, 0, time.UTC),
				cloudAccountID: "987654321098",
				clusterName:    "prod-cluster",
				extension:      "json",
			},
			want: "raw_compressed/year=2023/month=12/day=31/hour=23/cloud_account_id=987654321098/cluster_name=prod-cluster/1704067140000.json",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := storage.BuildPath(tt.args.prefix, tt.args.t, tt.args.cloudAccountID, tt.args.clusterName, tt.args.extension); got != tt.want {
				t.Errorf("BuildPath() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBucketName(t *testing.T) {
	type args struct {
		organizationID string
	}
	tests := []struct {
		name string
		args args
		want string
		env  map[string]string
	}{
		{
			name: "Valid Namespace and OrganizationID",
			args: args{
				organizationID: "org123",
			},
			want: "cz-prod-container-analysis-org123",
			env:  map[string]string{"NAMESPACE": "prod"},
		},
		{
			name: "Different Namespace and OrganizationID",
			args: args{
				organizationID: "org456",
			},
			want: "cz-dev-container-analysis-org456",
			env:  map[string]string{"NAMESPACE": "dev"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for k, v := range tt.env {
				t.Setenv(k, v)
			}
			if got := storage.BucketName(tt.args.organizationID); got != tt.want {
				t.Errorf("BucketName() = %v, want %v", got, tt.want)
			}
		})
	}
}
