package mongodb

// This file provides MongoDB codecs for the events defined in events.go.

import (
	"api-broker-prototype/events"
	"go.mongodb.org/mongo-driver/bson"
)

// MongoDB codec for simpleEvents.
type SimpleEventCodec struct{}

// Class implements the MongoDBEventCodec interface.
func (codec *SimpleEventCodec) Class() string {
	return "simple"
}

// Serialize implements the MongoDBEventCodec interface.
func (codec *SimpleEventCodec) Serialize(e events.Event) (bson.M, error) {
	ev := e.(events.SimpleEvent)
	return bson.M{"message": ev.Message}, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *SimpleEventCodec) Deserialize(data bson.M) (events.Event, error) {
	res := events.SimpleEvent{
		Message: data["message"].(string),
	}
	return res, nil
}

// MongoDB codec for ConfigurationEvents.
type ConfigurationEventCodec struct{}

// Class implements the MongoDBEventCodec interface.
func (codec *ConfigurationEventCodec) Class() string {
	return "configuration"
}

// Serialize implements the MongoDBEventCodec interface.
func (codec *ConfigurationEventCodec) Serialize(e events.Event) (bson.M, error) {
	ev := e.(events.ConfigurationEvent)
	return bson.M{"retries": ev.Retries}, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *ConfigurationEventCodec) Deserialize(data bson.M) (events.Event, error) {
	res := events.ConfigurationEvent{
		Retries: data["retries"].(int32),
	}
	return res, nil
}

// MongoDB codec for requestEvents.
type RequestEventCodec struct{}

// Class implements the MongoDBEventCodec interface.
func (codec *RequestEventCodec) Class() string {
	return "request"
}

// Serialize implements the MongoDBEventCodec interface.
func (codec *RequestEventCodec) Serialize(e events.Event) (bson.M, error) {
	ev := e.(events.RequestEvent)
	res := bson.M{
		"request": ev.Request,
	}
	return res, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *RequestEventCodec) Deserialize(data bson.M) (events.Event, error) {
	res := events.RequestEvent{
		Request: data["request"].(string),
	}
	return res, nil
}

// MongoDB codec for ResponseEvents.
type ResponseEventCodec struct{}

// Class implements the MongoDBEventCodec interface.
func (codec *ResponseEventCodec) Class() string {
	return "response"
}

// Serialize implements the MongoDBEventCodec interface.
func (codec *ResponseEventCodec) Serialize(e events.Event) (bson.M, error) {
	ev := e.(events.ResponseEvent)
	res := bson.M{
		"response": ev.Response,
	}
	return res, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *ResponseEventCodec) Deserialize(data bson.M) (events.Event, error) {
	res := events.ResponseEvent{
		Response: data["response"].(string),
	}
	return res, nil
}

// MongoDB codec for FailureEvents.
type FailureEventCodec struct{}

// Class implements the MongoDBEventCodec interface.
func (codec *FailureEventCodec) Class() string {
	return "failure"
}

// Serialize implements the MongoDBEventCodec interface.
func (codec *FailureEventCodec) Serialize(e events.Event) (bson.M, error) {
	ev := e.(events.FailureEvent)
	res := bson.M{
		"failure": ev.Failure,
	}
	return res, nil
}

// Deserialize implements the MongoDBEventCodec interface.
func (codec *FailureEventCodec) Deserialize(data bson.M) (events.Event, error) {
	res := events.FailureEvent{
		Failure: data["failure"].(string),
	}
	return res, nil
}
