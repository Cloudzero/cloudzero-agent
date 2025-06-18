// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package config contains code for all configs to share
package config

// Serializable is a small interface that ensures certain objects can be freely represented in
// various encoded forms, usually for the purpose of transmitting in the network
type Serializable interface {
	// ToBytes returns a serialized representation of the data in the class
	ToBytes() ([]byte, error)
}
