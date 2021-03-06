package postgresql

// This file provides PostgreSQL codecs for the events defined in events.go.

import (
	"api-broker-prototype/events"

	"github.com/jackc/pgtype"
)

// in-memory representation for the JSON data we store in the DB
type dataRecord map[string]interface{}

// PostgreSQL codec for SimpleEvents.
type simpleEventCodec struct{}

// Class implements the PostgreSQLEventCodec interface.
func (codec *simpleEventCodec) Class() string {
	return "simple"
}

// Serialize implements the PostgreSQLEventCodec interface.
func (codec *simpleEventCodec) Serialize(ev events.Event) (pgtype.JSONB, error) {
	event := ev.(events.SimpleEvent)
	res := pgtype.JSONB{}
	err := res.Set(
		dataRecord{
			"message": event.Message,
		},
	)
	if err != nil {
		return pgtype.JSONB{}, err
	}
	return res, nil
}

// Deserialize implements the PostgreSQLEventCodec interface.
func (codec *simpleEventCodec) Deserialize(data pgtype.JSONB) (events.Event, error) {
	tmp := dataRecord{}
	err := data.AssignTo(&tmp)
	res := events.SimpleEvent{
		Message: tmp["message"].(string),
	}
	return res, err
}

// PostgreSQL codec for ConfigurationEvents.
type configurationEventCodec struct{}

// Class implements the PostgreSQLEventCodec interface.
func (codec *configurationEventCodec) Class() string {
	return "configuration"
}

// Serialize implements the PostgreSQLEventCodec interface.
func (codec *configurationEventCodec) Serialize(ev events.Event) (pgtype.JSONB, error) {
	event := ev.(events.ConfigurationEvent)
	res := pgtype.JSONB{}
	err := res.Set(
		dataRecord{
			"retries": event.Retries,
		},
	)
	if err != nil {
		return pgtype.JSONB{}, err
	}
	return res, nil
}

// Deserialize implements the PostgreSQLEventCodec interface.
func (codec *configurationEventCodec) Deserialize(data pgtype.JSONB) (events.Event, error) {
	tmp := dataRecord{}
	err := data.AssignTo(&tmp)
	res := events.ConfigurationEvent{
		Retries: (int32)(tmp["retries"].(float64)),
	}
	return res, err
}

type requestEventCodec struct{}

// Class implements the PostgreSQLEventCodec interface.
func (codec *requestEventCodec) Class() string {
	return "request"
}

// Serialize implements the PostgreSQLEventCodec interface.
func (codec *requestEventCodec) Serialize(ev events.Event) (pgtype.JSONB, error) {
	event := ev.(events.RequestEvent)
	res := pgtype.JSONB{}
	err := res.Set(
		dataRecord{
			"request": event.Request,
		},
	)
	if err != nil {
		return pgtype.JSONB{}, err
	}
	return res, nil
}

// Deserialize implements the PostgreSQLEventCodec interface.
func (codec *requestEventCodec) Deserialize(data pgtype.JSONB) (events.Event, error) {
	tmp := dataRecord{}
	err := data.AssignTo(&tmp)
	res := events.RequestEvent{
		Request: tmp["request"].(string),
	}
	return res, err
}

type responseEventCodec struct{}

// Class implements the PostgreSQLEventCodec interface.
func (codec *responseEventCodec) Class() string {
	return "response"
}

// Serialize implements the PostgreSQLEventCodec interface.
func (codec *responseEventCodec) Serialize(ev events.Event) (pgtype.JSONB, error) {
	event := ev.(events.ResponseEvent)
	res := pgtype.JSONB{}
	err := res.Set(
		dataRecord{
			"attempt":  event.Attempt,
			"response": event.Response,
		},
	)
	if err != nil {
		return pgtype.JSONB{}, err
	}
	return res, nil
}

// Deserialize implements the PostgreSQLEventCodec interface.
func (codec *responseEventCodec) Deserialize(data pgtype.JSONB) (events.Event, error) {
	tmp := dataRecord{}
	err := data.AssignTo(&tmp)
	res := events.ResponseEvent{
		Attempt:  (uint)(tmp["attempt"].(float64)),
		Response: tmp["response"].(string),
	}
	return res, err
}

type failureEventCodec struct{}

// Class implements the PostgreSQLEventCodec interface.
func (codec *failureEventCodec) Class() string {
	return "failure"
}

// Serialize implements the PostgreSQLEventCodec interface.
func (codec *failureEventCodec) Serialize(ev events.Event) (pgtype.JSONB, error) {
	event := ev.(events.FailureEvent)
	res := pgtype.JSONB{}
	err := res.Set(
		dataRecord{
			"attempt": event.Attempt,
			"failure": event.Failure,
		},
	)
	if err != nil {
		return pgtype.JSONB{}, err
	}
	return res, nil
}

// Deserialize implements the PostgreSQLEventCodec interface.
func (codec *failureEventCodec) Deserialize(data pgtype.JSONB) (events.Event, error) {
	tmp := dataRecord{}
	err := data.AssignTo(&tmp)
	res := events.FailureEvent{
		Attempt: (uint)(tmp["attempt"].(float64)),
		Failure: tmp["failure"].(string),
	}
	return res, err
}
