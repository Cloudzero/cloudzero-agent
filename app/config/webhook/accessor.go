// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package config

// ConfigAccessor defines the interface for accessing settings related to labels and annotations.
// It provides methods to check if labels and annotations are enabled globally or for specific resource types,
// retrieve the associated resource type, and access the configuration settings.
type ConfigAccessor interface {
	// LabelsEnabled checks if labels are enabled globally.
	LabelsEnabled() bool

	// AnnotationsEnabled checks if annotations are enabled globally.
	AnnotationsEnabled() bool

	// LabelsEnabledForType checks if labels are enabled for a specific resource type.
	LabelsEnabledForType() bool

	// AnnotationsEnabledForType checks if annotations are enabled for a specific resource type.
	AnnotationsEnabledForType() bool

	// ResourceType returns the type of resource associated with the configuration.
	ResourceType() ResourceType

	// Settings retrieves the configuration settings.
	Settings() *Settings
}
