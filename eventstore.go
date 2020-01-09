package main

import (
	"context"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// The Envelope is a container for the actual event, which it carries as
// payload. In addition, it contains the time when the envelope was persisted
// and an ID which is a sequence counter.
type Envelope struct {
	ID      int32              `bson:"_id"`
	Created primitive.DateTime `bson:"created"`
	Payload bson.M             `bson:"payload"`
}

// Notification just carries the ID of a persisted event.
type Notification struct {
	ID int32 `bson:"_id"`
}

// The EventStore interface defines a few basic functions that an event store has to provide.
type EventStore interface {
	// ParseEventID parses a string that represents an event identifier.
	ParseEventID(str string) (int32, error)

	// Retrieve error state of the event store.
	// All functions below set this error state in order to signal failure.
	Error() error
	// Insert an envelope into the store.
	// The ID of the envelope is ignored. Instead, an ID is generated and returned.
	Insert(ctx context.Context, env Envelope) int32
	// Retrieve just the event with the given ID.
	RetrieveOne(ctx context.Context, id int32) *Envelope
	// Retrieve all currently existing events, which are provided via the returned channel.
	LoadEvents(ctx context.Context, start int32) <-chan Envelope
	// Follow the stream of notifications. This function emits any newly
	// created notification via the returned channel.
	FollowNotifications(ctx context.Context) <-chan int32
	// Follow the stream of events. This function emits any newly created
	// persisted event via the returned channel.
	FollowEvents(ctx context.Context, start int32) <-chan Envelope
}
