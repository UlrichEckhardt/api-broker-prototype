package postgresql

import (
	"api-broker-prototype/events"
	"testing"
)

func TestEnvelope(t *testing.T) {
	var _ events.Envelope = &postgreSQLEnvelope{}
}

func TestNotification(t *testing.T) {
	var _ events.Notification = &postgreSQLNotification{}
}

func TestEventstore(t *testing.T) {
	var _ events.EventStore = &PostgreSQLEventStore{}
}
