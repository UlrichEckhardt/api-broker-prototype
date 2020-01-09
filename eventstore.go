package main

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"time"
)

// The Envelope is a container for the actual event, which it carries as
// payload. In addition, it contains the time when the envelope was persisted
// and an ID which is a sequence counter.
type Envelope struct {
	IDVal      int32              `bson:"_id"`
	CreatedVal primitive.DateTime `bson:"created"`
	PayloadVal bson.M             `bson:"payload"`
}

// ID returns the identifier for the event.
func (env *Envelope) ID() int32 {
	return env.IDVal
}

// Created returns the time the event was persisted.
func (env *Envelope) Created() time.Time {
	return env.CreatedVal.Time()
}

// Payload returns the payload contained in the envelope.
func (env *Envelope) Payload() bson.M {
	return env.PayloadVal
}

// Notification just carries the ID of a persisted event.
type Notification struct {
	IDVal int32 `bson:"_id"`
}

// ID returns the identifier for the event stored in the notification.
func (note *Notification) ID() int32 {
	return note.IDVal
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
	Insert(ctx context.Context, payload bson.M) *Envelope
	// Retrieve just the event with the given ID.
	RetrieveOne(ctx context.Context, id int32) *Envelope
	// Retrieve all currently existing events, which are provided via the returned channel.
	LoadEvents(ctx context.Context, start int32) <-chan Envelope
	// Follow the stream of notifications. This function emits any newly
	// created notification via the returned channel.
	FollowNotifications(ctx context.Context) <-chan Notification
	// Follow the stream of events. This function emits any newly created
	// persisted event via the returned channel.
	FollowEvents(ctx context.Context, start int32) <-chan Envelope
}
