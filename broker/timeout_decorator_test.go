package broker

import (
	"api-broker-prototype/events"
	"testing"
)

// make sure the decorator implements the event store interface
func TestInterface(t *testing.T) {
	var _ events.EventStore = &EventStoreTimeoutDecorator{}
}
