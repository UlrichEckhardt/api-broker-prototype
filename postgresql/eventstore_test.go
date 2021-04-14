package postgresql

import (
	"api-broker-prototype/events"
	"testing"
)

func TestEventstore(t *testing.T) {
	var _ events.EventStore = &PostgreSQLEventStore{}
}
