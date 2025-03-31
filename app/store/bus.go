// SPDX-FileCopyrightText: Copyright (c) 2016-2024, CloudZero, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"github.com/wagoodman/go-partybus"

	"github.com/cloudzero/cloudzero-agent-validator/app/types"
)

// This structure is allows registering for events on the bus
// and allows publisher to publish events to all listeners
type bus struct {
	bus *partybus.Bus
}

// NewBus creates a new Bus instance
func NewBus() types.Bus {
	return &bus{
		bus: partybus.NewBus(),
	}
}

// Subscribe registers a listener for a specific event
func (b *bus) Subscribe() *types.Subscription {
	return b.bus.Subscribe()
}

// Unsubscribe removes a listener from the bus
func (b *bus) Unsubscribe(sub *types.Subscription) error {
	return b.bus.Unsubscribe(sub)
}

// Publish sends an event to all listeners
func (b *bus) Publish(event types.Event) {
	b.bus.Publish(event)
}
