package logging

// logging decorator for the eventstore interface
// The goal of this is to add logging to the eventstores without having to
// repeat it for every implementation independently. This is an implementation
// of the Decorator Pattern, see also http://wiki.c2.com/?DecoratorPattern.

import (
	"api-broker-prototype/events"
	"context"
	"errors"
	"github.com/inconshreveable/log15"
)

// LoggingDecoratorEventStore implements the EventStore interface
type LoggingDecoratorEventStore struct {
	logger     log15.Logger
	eventstore events.EventStore
}

func NewLoggingDecorator(eventstore events.EventStore, logger log15.Logger) (*LoggingDecoratorEventStore, error) {
	if eventstore == nil {
		return nil, errors.New("eventstore is nil")
	}
	if logger == nil {
		return nil, errors.New("logger is nil")
	}
	res := &LoggingDecoratorEventStore{
		eventstore: eventstore,
		logger:     logger,
	}
	return res, nil
}

func (s *LoggingDecoratorEventStore) ParseEventID(str string) (int32, error) {
	return s.eventstore.ParseEventID(str)
}

func (s *LoggingDecoratorEventStore) Error() error {
	return s.eventstore.Error()
}

func (s *LoggingDecoratorEventStore) Close() error {
	return s.eventstore.Close()
}

func (s *LoggingDecoratorEventStore) Insert(ctx context.Context, event events.Event, causationID int32) (events.Envelope, error) {
	s.logger.Debug("Inserting event.", "class", event.Class(), "causation_id", causationID)
	env, err := s.eventstore.Insert(ctx, event, causationID)
	if err == nil {
		s.logger.Debug("Inserted event.", "id", env.ID())
	} else {
		s.logger.Debug("Failed to insert event.", "error", err)
	}
	return env, err
}

func (s *LoggingDecoratorEventStore) RetrieveOne(ctx context.Context, id int32) (events.Envelope, error) {
	s.logger.Debug("Loading event.", "id", id)
	env, err := s.eventstore.RetrieveOne(ctx, id)
	if err == nil {
		s.logger.Debug("Loaded event.", "class", env.Event().Class(), "causation_id", env.CausationID(), "created", env.Created())
	} else {
		s.logger.Debug("Failed to load event.", "error", err)
	}
	return env, err
}

func (s *LoggingDecoratorEventStore) LoadEvents(ctx context.Context, start int32) (<-chan events.Envelope, error) {
	s.logger.Debug("Loading events.", "start", start)
	str, err := s.eventstore.LoadEvents(ctx, start)
	if err == nil {
		s.logger.Debug("Loaded events.")
	} else {
		s.logger.Debug("Failed to load events.", "error", err)
	}
	return str, err
}

func (s *LoggingDecoratorEventStore) FollowNotifications(ctx context.Context) (<-chan events.Notification, error) {
	s.logger.Debug("Loading notification stream.")
	str, err := s.eventstore.FollowNotifications(ctx)
	if err == nil {
		s.logger.Debug("Loaded notification stream.")
	} else {
		s.logger.Debug("Failed to load notification stream.", "error", err)
	}
	return str, err
}

func (s *LoggingDecoratorEventStore) FollowEvents(ctx context.Context, start int32) (<-chan events.Envelope, error) {
	s.logger.Debug("Loading event stream.", "start", start)
	str, err := s.eventstore.FollowEvents(ctx, start)
	if err == nil {
		s.logger.Debug("Loaded event stream.")
	} else {
		s.logger.Debug("Failed to load event stream.", "error", err)
	}
	return str, err
}
