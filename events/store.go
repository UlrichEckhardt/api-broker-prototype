package events

// This package contains interface definitions for an event store.
//
// Generally, this is defined to be agnostic of the storage infrastructure
// behind it. Its main parts are functions to insert events and to follow
// the stream of events.
//
// TODO:
//  - The ID is also specific to the default integer representation in MongoDB.
//    For proper function, only a strict ordering is required, it doesn't have
//    to take the form of an integer sequence.

import (
	"context"
	"time"

	"github.com/gofrs/uuid"
)

// The Envelope is a container for the actual event, which it carries as
// payload. In addition, it contains the time when the envelope was persisted
// and an ID which is a sequence counter.
type Envelope interface {
	// ID returns the identifier for the event.
	ID() int32
	// Created returns the time the event was persisted.
	Created() time.Time
	// ExternalUUID returns the (optional) UUID assigned to the event
	ExternalUUID() uuid.UUID
	// CausationID returns the ID of the event that caused this event.
	// This can be zero if this event was not caused by another event.
	CausationID() int32
	// Event returns the event contained in the envelope.
	Event() Event
}

// Notification just carries the ID of a persisted event.
//
// The idea of a notification is mostly the info that something happened,
// without actually specifying what happened. That actual info is carried
// by the `Event` type instead.
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
	// The external UUID can be used to attach a client-supplied identifier
	// to the event. If the UUID is not the nil UUID, it must be unique.
	// If the UUID is already used, DuplicateEventUUID is returned as error.
	// The causation ID is that of the preceding event that caused this new
	// event. It can be zero when its cause is not a preceding event.
	Insert(ctx context.Context, externalUUID uuid.UUID, event Event, causationID int32) (Envelope, error)

	// Resolve an external UUID to the according internal ID
	ResolveUUID(ctx context.Context, externalUUID uuid.UUID) (int32, error)

	// Retrieve just the event with the given ID.
	RetrieveOne(ctx context.Context, id int32) (Envelope, error)

	// Retrieve existing events.
	//
	// The events are provided via the returned channel. `startAfter` parameter
	// specifies the ID preceding the first event to retrieve. If this is zero,
	// the first event from the store is loaded first. The channel is closed
	// when all events have been retrieved.
	LoadEvents(ctx context.Context, startAfter int32) (<-chan Envelope, error)

	// Follow the stream of notifications.
	//
	// This function emits any newly created notification via the returned
	// channel.
	FollowNotifications(ctx context.Context) (<-chan Notification, error)

	// Follow the stream of events.
	//
	// The events are provided via the returned channel. `startAfter` parameter
	// specifies the ID preceding the first event to retrieve. If this is zero,
	// the first event from the store is loaded first. When all events from the
	// store are emitted, the channel blocks until new events are stored.
	FollowEvents(ctx context.Context, startAfter int32) (<-chan Envelope, error)
}
