package storage

import (
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
)

type Labels = map[string]string
type Annotations = map[string]string

type ResourceTags struct {
	ID           string                  `gorm:"unique;autoIncrement"`
	Type         config.ResourceType     `gorm:"primaryKey"`      // Type of k8s resource; deployment, statefulset, pod, node, namespace
	Name         string                  `gorm:"primaryKey"`      // Name of the resource
	Namespace    *string                 `gorm:"primaryKey"`      // Namspace of the resource, if applicable
	MetricLabels *config.MetricLabels    `gorm:"serializer:json"` // Metric labels of the resource; nullable
	Labels       *config.MetricLabelTags `gorm:"serializer:json"` // Labels of the resource; nullable
	Annotations  *config.MetricLabelTags `gorm:"serializer:json"` // Annotations of the resource; nullable
	CreatedAt    time.Time               // Creation time of the record
	UpdatedAt    time.Time               // Time that the record was updated, if the k8s object was updated with different labels
	SentAt       *time.Time              // Time that the record was sent to the cloudzero API, or null if not sent yet
	Size         int                     `gorm:"->;type:GENERATED ALWAYS AS (IFNULL(octet_length(labels), 0) + IFNULL(octet_length(annotations), 0)) VIRTUAL;"` // Size of the record in bytes
}

type RemoteWriteHistory struct {
	LastRemoteWriteTime time.Time
}
