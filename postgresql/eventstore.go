package postgresql

// Eventstore built on top of a PostgreSQL DB
// This uses Postgre's LISTEN/NOTIFY/pg_notify() feature to efficiently signal
// new events.

import (
	"api-broker-prototype/events"
	"errors"
	"github.com/inconshreveable/log15"
)

// NewEventStore creates and connects a PostgreSQLEventStore instance.
func NewEventStore(logger log15.Logger, host string) (events.EventStore, error) {
	logger.Debug("creating event store", "host", host)

	return nil, errors.New("not implemented")
}
