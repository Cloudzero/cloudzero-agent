package metrics

import "time"

// SummaryRecord is what a transformed output from the summary query.
//
// A subset of SummaryRecord will be generated from a list of MetricRecord
type SummaryRecord struct {
	MetricsHour                   time.Time              `json:"metrics_hour"`
	CloudAccountID                string                 `json:"cloud_account_id"`
	ClusterName                   string                 `json:"cluster_name"`
	NodeName                      string                 `json:"node_name"`
	InstanceType                  string                 `json:"instance_type"`
	CloudLocalID                  string                 `json:"cloud_local_id"`
	CloudRegion                   string                 `json:"cloud_region"`
	CloudProvider                 string                 `json:"cloud_provider"`
	KubernetesNodeID              string                 `json:"kubernetes_node_id"`
	NodeCores                     int                    `json:"node_cores"`
	NodeMemory                    int64                  `json:"node_memory"`
	NodeUsageMinutes              int                    `json:"node_usage_minutes"`
	NodeMinUsageDate              time.Time              `json:"node_min_usage_date"`
	NodeMaxUsageDate              time.Time              `json:"node_max_usage_date"`
	NodeTotalRuntime              int                    `json:"node_total_runtime"`
	WorkloadType                  string                 `json:"workload_type"`
	Tags                          map[string]interface{} `json:"tags"`
	Namespace                     string                 `json:"namespace"`
	PodName                       string                 `json:"pod_name"`
	KubernetesPodName             string                 `json:"kubernetes_pod_name"`
	KubernetesPodID               string                 `json:"kubernetes_pod_id"`
	PodMinUsageDate               time.Time              `json:"pod_min_usage_date"`
	PodMaxUsageDate               time.Time              `json:"pod_max_usage_date"`
	PodTotalRuntime               int                    `json:"pod_total_runtime"`
	PodUsageMinutes               int                    `json:"pod_usage_minutes"`
	SumPodCPUUtilization          float64                `json:"sum_pod_cpu_utilization"`
	SumPodMemoryUtilization       float64                `json:"sum_pod_memory_utilization"`
	PodCPULimit                   float64                `json:"pod_cpu_limit"`
	PodCPURequest                 float64                `json:"pod_cpu_request"`
	PodMemoryLimit                float64                `json:"pod_memory_limit"`
	PodMemoryRequest              float64                `json:"pod_memory_request"`
	SumPodCPUReservedCapacity     int                    `json:"sum_pod_cpu_reserved_capacity"`
	SumPodMemoryReservedCapacity  int                    `json:"sum_pod_memory_reserved_capacity"`
	CPURequestContainerMap        map[string]float64     `json:"cpu_request_container_map"`
	CPULimitContainerMap          map[string]float64     `json:"cpu_limit_container_map"`
	MemoryRequestContainerMap     map[string]float64     `json:"memory_request_container_map"`
	MemoryLimitContainerMap       map[string]float64     `json:"memory_limit_container_map"`
	ContainerValidUtilizationMap  map[string]float64     `json:"container_valid_utilization_map"`
	ContainerRawSumCPUUtilMap     map[string]float64     `json:"container_raw_sum_cpu_utilization_map"`
	ContainerMaxCPUUtilizationMap map[string]float64     `json:"container_max_cpu_utilization_map"`
}
