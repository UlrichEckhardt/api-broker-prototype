package events

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

// The Envelope is a container for the actual event, which it carries as
// payload. In addition, it contains the time when the envelope was persisted
// and an ID which is a sequence counter.
type Envelope interface {
	// ID returns the identifier for the event.
	ID() int32
	// Created returns the time the event was persisted.
	Created() time.Time
	// CausationID returns the ID of the event that caused this event.
	// This can be zero if this event was not caused by another event.
	CausationID() int32
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

	// Close the connection to the underlying storage.
	// This implements the io.Closer interface.
	Close() error

	// Insert an event as payload into the store.
	// The event is wrapped in an envelope and returned.
	// The causation ID is that of the preceding event that caused this new
	// event. It can be zero when its cause is not a preceding event.
	Insert(ctx context.Context, event Event, causationID int32) (Envelope, error)
	// Retrieve just the event with the given ID.
	RetrieveOne(ctx context.Context, id int32) (Envelope, error)
	// Retrieve all currently existing events, which are provided via the returned channel.
	LoadEvents(ctx context.Context, start int32) (<-chan Envelope, error)
	// Follow the stream of notifications. This function emits any newly
	// created notification via the returned channel.
	FollowNotifications(ctx context.Context) <-chan Notification
	// Follow the stream of events. This function emits any newly created
	// persisted event via the returned channel.
	FollowEvents(ctx context.Context, start int32) <-chan Envelope
}
