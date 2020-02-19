package main

// This package contains interface definitions for an event store.
// Generally, this is defined to be agnostic of the storage infrastructure
// behind it. Its main parts are functions to insert events and to follow
// the stream of events.
// TODO:
//  - The ID is also specific to the default integer representation in MongoDB.
//    For proper function, only a strict ordering is required, it doesn't have
//    to take the form of an integer sequence.

import (
	"context"
	"time"
)

// The Event interface defines methods common to events.
type Event interface {
	// Class returns a string that identifies the event type.
	// This is distinct from Go's type name because it should be
	// agnostic of the language.
	Class() string
}

// The Envelope is a container for the actual event, which it carries as
// payload. In addition, it contains the time when the envelope was persisted
// and an ID which is a sequence counter.
type Envelope interface {
	// ID returns the identifier for the event.
	ID() int32
	// Created returns the time the event was persisted.
	Created() time.Time
	// Event returns the event contained in the envelope.
	Event() Event
}

// Notification just carries the ID of a persisted event.
type Notification interface {
	// ID returns the identifier for the event associated with the notification.
	ID() int32
}

// The EventStore interface defines a few basic functions that an event store has to provide.
type EventStore interface {
	// ParseEventID parses a string that represents an event identifier.
	ParseEventID(str string) (int32, error)

	// Retrieve error state of the event store.
	// All functions below set this error state in order to signal failure.
	Error() error
	// Insert an event as payload into the store.
	// The event is wrapped in an envelope and returned.
	Insert(ctx context.Context, event Event) Envelope
	// Retrieve just the event with the given ID.
	RetrieveOne(ctx context.Context, id int32) Envelope
	// Retrieve all currently existing events, which are provided via the returned channel.
	LoadEvents(ctx context.Context, start int32) <-chan Envelope
	// Follow the stream of notifications. This function emits any newly
	// created notification via the returned channel.
	FollowNotifications(ctx context.Context) <-chan Notification
	// Follow the stream of events. This function emits any newly created
	// persisted event via the returned channel.
	FollowEvents(ctx context.Context, start int32) <-chan Envelope
}
