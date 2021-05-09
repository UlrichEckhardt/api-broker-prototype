package postgresql

import (
	"api-broker-prototype/events"
	"testing"
)

func TestEnvelope(t *testing.T) {
	var _ events.Envelope = &postgreSQLEnvelope{}
}

func TestEventstore(t *testing.T) {
	var _ events.EventStore = &PostgreSQLEventStore{}
}
