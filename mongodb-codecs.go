package main

// This file provides MongoDB codecs for the events defined in events.go.

import (
	"api-broker-prototype/events"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDB codec for simpleEvents.
type simpleEventCodec struct{}

// Class implements the MongoDBEventCodec interface.
func (codec *simpleEventCodec) Class() string {
	return "simple"
}

// Serialize implements the MongoDBEventCodec interface.
func (codec *simpleEventCodec) Serialize(e events.Event) (bson.M, error) {
	ev := e.(events.SimpleEvent)
	return bson.M{"message": ev.Message}, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *simpleEventCodec) Deserialize(data bson.M) (events.Event, error) {
	res := events.SimpleEvent{
		Message: data["message"].(string),
	}
	return res, nil
}

// MongoDB codec for ConfigurationEvents.
type configurationEventCodec struct{}

// Class implements the MongoDBEventCodec interface.
func (codec *configurationEventCodec) Class() string {
	return "configuration"
}

// Serialize implements the MongoDBEventCodec interface.
func (codec *configurationEventCodec) Serialize(e events.Event) (bson.M, error) {
	ev := e.(events.ConfigurationEvent)
	return bson.M{"retries": ev.Retries}, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *configurationEventCodec) Deserialize(data bson.M) (events.Event, error) {
	res := events.ConfigurationEvent{
		Retries: data["retries"].(int32),
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
func (codec *requestEventCodec) Serialize(e events.Event) (bson.M, error) {
	ev := e.(events.RequestEvent)
	res := bson.M{
		"request": ev.Request,
	}
	return res, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *requestEventCodec) Deserialize(data bson.M) (events.Event, error) {
	res := events.RequestEvent{
		Request: data["request"].(string),
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
func (codec *responseEventCodec) Serialize(e events.Event) (bson.M, error) {
	ev := e.(events.ResponseEvent)
	res := bson.M{
		"response": ev.Response,
	}
	return res, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *responseEventCodec) Deserialize(data bson.M) (events.Event, error) {
	res := events.ResponseEvent{
		Response: data["response"].(string),
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
func (codec *failureEventCodec) Serialize(e events.Event) (bson.M, error) {
	ev := e.(events.FailureEvent)
	res := bson.M{
		"failure": ev.Failure,
	}
	return res, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *failureEventCodec) Deserialize(data bson.M) (events.Event, error) {
	res := events.FailureEvent{
		Failure: data["failure"].(string),
	}
	return res, nil
}
