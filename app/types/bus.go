// SPDX-FileCopyrightText: Copyright (c) 2016-2025, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package types

import "github.com/wagoodman/go-partybus"

type (
	Event        = partybus.Event
	Subscription = partybus.Subscription
)

type Bus interface {
	// Subscribe returns a new subscription to the bus.
	Subscribe() *Subscription
	// Unsubscribe removes a subscription from the bus.
	Unsubscribe(*Subscription) error
	// Publish sends an event to the bus.
	Publish(event Event)
}

type FileCreated struct {
	Name string
}

type FileChanged struct {
	Name string
}

type FileDeleted struct {
	Name string
}

type FileRenamed struct {
	Name string
}
