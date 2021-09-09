package events

import (
	"context"
	"github.com/inconshreveable/log15"
	"time"
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
	store   EventStore // decorated event store
	logger  log15.Logger
	timeout *time.Duration
}

// NewTimeoutEventStoreDecorator creates a decorator for an event store.
func NewTimeoutEventStoreDecorator(store EventStore, logger log15.Logger) (*EventStoreTimeoutDecorator, error) {
	res := &EventStoreTimeoutDecorator{
		store:   store,
		logger:  logger,
		timeout: nil,
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

// Insert implements the EventStore interface.
func (d *EventStoreTimeoutDecorator) Insert(ctx context.Context, event Event, causationID int32) (Envelope, error) {
	// forward call to the decorated event store first
	env, err := d.store.Insert(ctx, event, causationID)
	if err != nil {
		return env, err
	}

	// if no timeout is configured, do nothing
	if d.timeout == nil {
		return env, err
	}

	// ignore all but API request events
	request, ok := env.Event().(APIRequestEvent)
	if !ok {
		return env, err
	}

	// trigger async creation of a timeout event
	// TODO: this accesses d.store asynchronously, which may need synchronization
	time.AfterFunc(
		*d.timeout,
		func() {
			_, err := d.store.Insert(
				ctx,
				APITimeoutEvent{
					Attempt: request.Attempt,
				},
				causationID,
			)
			if err != nil {
				d.logger.Error("timeout-decorator: failed to insert timeout event", "error", err)
			}
		},
	)

	// return result of inserting the initial event
	return env, err
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

// FollowEvents implements the EventStore interface.
func (d *EventStoreTimeoutDecorator) FollowEvents(ctx context.Context, start int32) (<-chan Envelope, error) {
	// forward call to the decorated event store first
	stream, err := d.store.FollowEvents(ctx, start)
	if err != nil {
		return stream, err
	}

	// create intermediate stream to intercept and log the events loaded
	res := make(chan Envelope)
	go func() {
		// close channel on finish
		defer close(res)

		for env := range stream {
			config, ok := env.Event().(ConfigurationEvent)
			if ok {
				// store the timeout value from the configuration event
				if config.Timeout > 0 {
					duration := time.Duration(float64(time.Second) * config.Timeout)
					d.timeout = &duration
				} else {
					d.timeout = nil
				}
				d.logger.Info(
					"timeout-decorator: adjusted timeout",
					"value", d.timeout,
				)
			}

			res <- env
		}
	}()

	return res, nil
}
