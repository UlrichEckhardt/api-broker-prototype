package events

import (
	"testing"
)

// make sure the decorator implements the event store interface
func TestInterface(t *testing.T) {
	var _ EventStore = &EventStoreTimeoutDecorator{}
}
