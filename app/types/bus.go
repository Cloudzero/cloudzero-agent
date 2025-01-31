package types

import "github.com/wagoodman/go-partybus"

type Event = partybus.Event
type Subscription = partybus.Subscription

type Bus interface {
	Subscribe() *Subscription
	Unsubscribe(*Subscription) error
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
