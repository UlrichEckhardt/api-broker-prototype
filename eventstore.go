package main

// This package contains interface definitions for an event store.
// Generally, this is defined to be agnostic of the storage infrastructure
// behind it. Its main parts are functions to insert events and to follow
// the stream of events.
// TODO:
//  - The payload is still represented as bson.M. This is a serialization format
//    that is mainly used in the context of MongoDB. Firstly, this restricts all
//    implementations to those supporting that format. Secondly, convenient use
//    probably requires a kind of serialization service in between.
//  - The ID is also specific to the default integer representation in MongoDB.
//    For proper function, only a strict ordering is required, it doesn't have
//    to take the form of an integer sequence.

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
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
	// Payload returns the payload contained in the envelope.
	Payload() bson.M
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
	Insert(ctx context.Context, payload bson.M) Envelope
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
