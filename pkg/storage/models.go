package storage

import (
	"time"

	"github.com/cloudzero/cloudzero-insights-controller/pkg/config"
)

type Labels = map[string]string
type Annotations = map[string]string

type ResourceTags struct {
	ID            string              // Standard field for the primary key
	Type          config.ResourceType // Type of k8s resource; deployment, statefulset, pod, node, namespace
	Namespace     *string             // Namspace of the resource, if applicable
	Name          string              // Name of the resource
	CreatedAt     time.Time           // Creation time of the record
	CreatedAtSent *time.Time          // Time that the record was sent to the cloudzero API, or null if not sent yet
	UpdatedAt     time.Time           // Time that the record was updated, if the k8s object was updated with different labels
	UpdatedAtSent *time.Time          // Time that the record was sent to the cloudzero API, or null if not sent yet
	Labels        Labels              `gorm:"serializer:json"` // Labels of the resource
	Annotations   *Annotations        `gorm:"serializer:json"` // Annotations of the resource; can be null
}

type RemoteWriteHistory struct {
	LastRemoteWriteTime time.Time
}
