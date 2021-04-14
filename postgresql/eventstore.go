package postgresql

// Eventstore built on top of a PostgreSQL DB
// This uses Postgre's LISTEN/NOTIFY/pg_notify() feature to efficiently signal
// new events.

import (
	"api-broker-prototype/events"
	"context"
	"errors"
	"github.com/inconshreveable/log15"
	"strconv"
)

// PostgreSQLEventStore implements the EventStore interface using a PostgreSQL DB
type PostgreSQLEventStore struct {
	logger log15.Logger
	host   string
	err    error
}

// NewEventStore creates and connects a PostgreSQLEventStore instance.
func NewEventStore(logger log15.Logger, host string) (*PostgreSQLEventStore, error) {
	logger.Debug("creating event store", "host", host)
	s := PostgreSQLEventStore{
		logger: logger,
		host:   host,
		err:    errors.New("not implemented"),
	}

	return &s, nil
}

// ParseEventID implements the EventStore interface.
func (s *PostgreSQLEventStore) ParseEventID(str string) (int32, error) {
	lp, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(lp), nil
}

// Error implements the EventStore interface.
func (s *PostgreSQLEventStore) Error() error {
	return s.err
}

// Error implements the EventStore and io.Closer interfaces.
func (s *PostgreSQLEventStore) Close() error {
	s.logger.Debug("closing eventstore")

	// don't do anything if the error state of the store is set already
	if s.err != nil {
		return nil
	}

	return errors.New("not implemented")
}

// Insert implements the EventStore interface.
func (s *PostgreSQLEventStore) Insert(ctx context.Context, event events.Event, causationID int32) (events.Envelope, error) {
	s.logger.Debug("inserting event", "causation", causationID, "class", event.Class())

	return nil, errors.New("not implemented")
}

// RetrieveOne implements the EventStore interface.
func (s *PostgreSQLEventStore) RetrieveOne(ctx context.Context, id int32) (events.Envelope, error) {
	s.logger.Debug("loading event", "id", id)

	// The ID must be valid.
	if id == 0 {
		return nil, errors.New("provided document ID is null")
	}

	return nil, errors.New("not implemented")
}

// LoadEvents implements the EventStore interface.
func (s *PostgreSQLEventStore) LoadEvents(ctx context.Context, start int32) (<-chan events.Envelope, error) {
	s.logger.Debug("loading events", "start", start)

	return nil, errors.New("not implemented")
}

// FollowNotifications implements the EventStore interface.
func (s *PostgreSQLEventStore) FollowNotifications(ctx context.Context) (<-chan events.Notification, error) {
	s.logger.Debug("following notifications")

	return nil, errors.New("not implemented")
}

// FollowEvents implements the EventStore interface.
func (s *PostgreSQLEventStore) FollowEvents(ctx context.Context, start int32) (<-chan events.Envelope, error) {
	s.logger.Debug("following events", "start", start)

	return nil, errors.New("not implemented")
}
