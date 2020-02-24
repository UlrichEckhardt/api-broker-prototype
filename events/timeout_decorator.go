package events

import (
	"context"
	"github.com/inconshreveable/log15"
)

// EventStoreTimeoutDecorator wraps an EventStore.
// This is an application of the "Decorator Pattern". It is used to insert an
// event into the stream when the timeout for an API call elapses. The approach
// is to hook into the `APIRequestEvent` handling and start a goroutine that
// inserts an according `APITimeoutEvent` on timeout. Note that this event does
// specifically not depend on the result of the request, it is always emitted.
// It is the underlying event handling structure that needs to take into account
// that a response is regularly followed by a timeout. Additionally, you could
// have a timeout followed by a response as well.
type EventStoreTimeoutDecorator struct {
	store  EventStore // decorated event store
	logger log15.Logger
}

// NewTimeoutEventStoreDecorator creates a decorator for an event store.
func NewTimeoutEventStoreDecorator(store EventStore, logger log15.Logger) (*EventStoreTimeoutDecorator, error) {
	res := &EventStoreTimeoutDecorator{
		store:  store,
		logger: logger,
	}
	return res, nil
}

// ParseEventID implements the EventStore interface by simply forwarding.
func (d *EventStoreTimeoutDecorator) ParseEventID(str string) (int32, error) {
	return d.store.ParseEventID(str)
}

// Error implements the EventStore interface by simply forwarding.
func (d *EventStoreTimeoutDecorator) Error() error {
	return d.store.Error()
}

// Error implements the EventStore and io.Closer interfaces by simply forwarding.
func (d *EventStoreTimeoutDecorator) Close() error {
	return d.store.Close()
}

// Insert implements the EventStore interface by simply forwarding.
func (d *EventStoreTimeoutDecorator) Insert(ctx context.Context, event Event, causationID int32) (Envelope, error) {
	return d.store.Insert(ctx, event, causationID)
}

// RetrieveOne implements the EventStore interface by simply forwarding.
func (d *EventStoreTimeoutDecorator) RetrieveOne(ctx context.Context, id int32) (Envelope, error) {
	return d.store.RetrieveOne(ctx, id)
}

// LoadEvents implements the EventStore interface by simply forwarding.
func (d *EventStoreTimeoutDecorator) LoadEvents(ctx context.Context, start int32) (<-chan Envelope, error) {
	return d.store.LoadEvents(ctx, start)
}

// FollowNotifications implements the EventStore interface by simply forwarding.
func (d *EventStoreTimeoutDecorator) FollowNotifications(ctx context.Context) (<-chan Notification, error) {
	return d.store.FollowNotifications(ctx)
}

// FollowEvents implements the EventStore interface by simply forwarding.
func (d *EventStoreTimeoutDecorator) FollowEvents(ctx context.Context, start int32) (<-chan Envelope, error) {
	return d.store.FollowEvents(ctx, start)
}
