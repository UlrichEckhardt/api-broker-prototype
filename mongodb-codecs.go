package main

// This file provides MongoDB codecs for the events defined in events.go.

import (
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDB codec for simpleEvents.
type simpleEventCodec struct{}

// Class implements the MongoDBEventCodec interface.
func (codec *simpleEventCodec) Class() string {
	return "simple"
}

// Serialize implements the MongoDBEventCodec interface.
func (codec *simpleEventCodec) Serialize(e Event) (bson.M, error) {
	ev := e.(*simpleEvent)
	return bson.M{"message": ev.message}, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *simpleEventCodec) Deserialize(data bson.M) (Event, error) {
	res := new(simpleEvent)
	res.message = data["message"].(string)
	return res, nil
}
