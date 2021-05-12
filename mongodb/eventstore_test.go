package mongodb

import (
	"api-broker-prototype/events"
	"testing"
)

func TestEnvelope(t *testing.T) {
	var _ events.Envelope = &mongoDBEnvelope{}
}

func TestNotification(t *testing.T) {
	var _ events.Notification = &mongoDBNotification{}
}

func TestEventstore(t *testing.T) {
	var _ events.EventStore = &MongoDBEventStore{}
}
