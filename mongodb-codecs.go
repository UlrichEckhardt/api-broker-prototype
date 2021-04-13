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
	ev := e.(simpleEvent)
	return bson.M{"message": ev.message}, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *simpleEventCodec) Deserialize(data bson.M) (Event, error) {
	res := simpleEvent{
		message: data["message"].(string),
	}
	return res, nil
}

// MongoDB codec for configurationEvents.
type configurationEventCodec struct{}

// Class implements the MongoDBEventCodec interface.
func (codec *configurationEventCodec) Class() string {
	return "configuration"
}

// Serialize implements the MongoDBEventCodec interface.
func (codec *configurationEventCodec) Serialize(e Event) (bson.M, error) {
	ev := e.(configurationEvent)
	return bson.M{"retries": ev.retries}, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *configurationEventCodec) Deserialize(data bson.M) (Event, error) {
	res := configurationEvent{
		retries: data["retries"].(int32),
	}
	return res, nil
}

// MongoDB codec for requestEvents.
type requestEventCodec struct{}

// Class implements the MongoDBEventCodec interface.
func (codec *requestEventCodec) Class() string {
	return "request"
}

// Serialize implements the MongoDBEventCodec interface.
func (codec *requestEventCodec) Serialize(e Event) (bson.M, error) {
	ev := e.(requestEvent)
	res := bson.M{
		"request": ev.request,
	}
	return res, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *requestEventCodec) Deserialize(data bson.M) (Event, error) {
	res := requestEvent{
		request: data["request"].(string),
	}
	return res, nil
}

// MongoDB codec for responseEvents.
type responseEventCodec struct{}

// Class implements the MongoDBEventCodec interface.
func (codec *responseEventCodec) Class() string {
	return "response"
}

// Serialize implements the MongoDBEventCodec interface.
func (codec *responseEventCodec) Serialize(e Event) (bson.M, error) {
	ev := e.(responseEvent)
	res := bson.M{
		"response": ev.response,
	}
	return res, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *responseEventCodec) Deserialize(data bson.M) (Event, error) {
	res := responseEvent{
		response: data["response"].(string),
	}
	return res, nil
}

// MongoDB codec for failureEvents.
type failureEventCodec struct{}

// Class implements the MongoDBEventCodec interface.
func (codec *failureEventCodec) Class() string {
	return "failure"
}

// Serialize implements the MongoDBEventCodec interface.
func (codec *failureEventCodec) Serialize(e Event) (bson.M, error) {
	ev := e.(failureEvent)
	res := bson.M{
		"failure": ev.failure,
	}
	return res, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *failureEventCodec) Deserialize(data bson.M) (Event, error) {
	res := failureEvent{
		failure: data["failure"].(string),
	}
	return res, nil
}
