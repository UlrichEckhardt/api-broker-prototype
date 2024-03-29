package mongodb

// This file provides MongoDB codecs for the events defined in events.go.

import (
	"api-broker-prototype/broker"
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
	ev := e.(broker.ConfigurationEvent)
	return bson.M{"retries": ev.Retries, "timeout": ev.Timeout}, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *configurationEventCodec) Deserialize(data bson.M) (events.Event, error) {
	res := broker.ConfigurationEvent{
		Retries: data["retries"].(int32),
		Timeout: data["timeout"].(float64),
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
	ev := e.(broker.RequestEvent)
	res := bson.M{
		"request": ev.Request,
	}
	return res, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *requestEventCodec) Deserialize(data bson.M) (events.Event, error) {
	res := broker.RequestEvent{
		Request: data["request"].(string),
	}
	return res, nil
}

// MongoDB codec for APIResponseEvents.
type apiRequestEventCodec struct{}

// Class implements the MongoDBEventCodec interface.
func (codec *apiRequestEventCodec) Class() string {
	return "api-request"
}

// Serialize implements the MongoDBEventCodec interface.
func (codec *apiRequestEventCodec) Serialize(e events.Event) (bson.M, error) {
	ev := e.(broker.APIRequestEvent)
	res := bson.M{
		"attempt": int64(ev.Attempt),
	}
	return res, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *apiRequestEventCodec) Deserialize(data bson.M) (events.Event, error) {
	res := broker.APIRequestEvent{
		Attempt: uint(data["attempt"].(int64)),
	}
	return res, nil
}

// MongoDB codec for APIResponseEvents.
type apiResponseEventCodec struct{}

// Class implements the MongoDBEventCodec interface.
func (codec *apiResponseEventCodec) Class() string {
	return "api-response"
}

// Serialize implements the MongoDBEventCodec interface.
func (codec *apiResponseEventCodec) Serialize(e events.Event) (bson.M, error) {
	ev := e.(broker.APIResponseEvent)
	res := bson.M{
		"attempt":  int64(ev.Attempt),
		"response": ev.Response,
	}
	return res, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *apiResponseEventCodec) Deserialize(data bson.M) (events.Event, error) {
	res := broker.APIResponseEvent{
		Attempt:  uint(data["attempt"].(int64)),
		Response: data["response"].(string),
	}
	return res, nil
}

// MongoDB codec for APIFailureEvents.
type apiFailureEventCodec struct{}

// Class implements the MongoDBEventCodec interface.
func (codec *apiFailureEventCodec) Class() string {
	return "api-failure"
}

// Serialize implements the MongoDBEventCodec interface.
func (codec *apiFailureEventCodec) Serialize(e events.Event) (bson.M, error) {
	ev := e.(broker.APIFailureEvent)
	res := bson.M{
		"attempt": int64(ev.Attempt),
		"failure": ev.Failure,
	}
	return res, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *apiFailureEventCodec) Deserialize(data bson.M) (events.Event, error) {
	res := broker.APIFailureEvent{
		Attempt: uint(data["attempt"].(int64)),
		Failure: data["failure"].(string),
	}
	return res, nil
}

// MongoDB codec for APITimeoutEvents.
type apiTimeoutEventCodec struct{}

// Class implements the MongoDBEventCodec interface.
func (codec *apiTimeoutEventCodec) Class() string {
	return "api-timeout"
}

// Serialize implements the MongoDBEventCodec interface.
func (codec *apiTimeoutEventCodec) Serialize(e events.Event) (bson.M, error) {
	ev := e.(broker.APITimeoutEvent)
	return bson.M{"attempt": int64(ev.Attempt)}, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *apiTimeoutEventCodec) Deserialize(data bson.M) (events.Event, error) {
	res := broker.APITimeoutEvent{
		Attempt: uint(data["attempt"].(int64)),
	}
	return res, nil
}
