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
			"timeout": event.Timeout,
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
		Timeout: tmp["timeout"].(float64),
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

type apiRequestEventCodec struct{}

// Class implements the PostgreSQLEventCodec interface.
func (codec *apiRequestEventCodec) Class() string {
	return "api-request"
}

// Serialize implements the PostgreSQLEventCodec interface.
func (codec *apiRequestEventCodec) Serialize(ev events.Event) (pgtype.JSONB, error) {
	event := ev.(events.APIRequestEvent)
	res := pgtype.JSONB{}
	err := res.Set(
		dataRecord{
			"attempt":  event.Attempt,
		},
	)
	if err != nil {
		return pgtype.JSONB{}, err
	}
	return res, nil
}

// Deserialize implements the PostgreSQLEventCodec interface.
func (codec *apiRequestEventCodec) Deserialize(data pgtype.JSONB) (events.Event, error) {
	tmp := dataRecord{}
	err := data.AssignTo(&tmp)
	res := events.APIRequestEvent{
		Attempt:  (uint)(tmp["attempt"].(float64)),
	}
	return res, err
}

type apiResponseEventCodec struct{}

// Class implements the PostgreSQLEventCodec interface.
func (codec *apiResponseEventCodec) Class() string {
	return "api-response"
}

// Serialize implements the PostgreSQLEventCodec interface.
func (codec *apiResponseEventCodec) Serialize(ev events.Event) (pgtype.JSONB, error) {
	event := ev.(events.APIResponseEvent)
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
func (codec *apiResponseEventCodec) Deserialize(data pgtype.JSONB) (events.Event, error) {
	tmp := dataRecord{}
	err := data.AssignTo(&tmp)
	res := events.APIResponseEvent{
		Attempt:  (uint)(tmp["attempt"].(float64)),
		Response: tmp["response"].(string),
	}
	return res, err
}

type apiFailureEventCodec struct{}

// Class implements the PostgreSQLEventCodec interface.
func (codec *apiFailureEventCodec) Class() string {
	return "api-failure"
}

// Serialize implements the PostgreSQLEventCodec interface.
func (codec *apiFailureEventCodec) Serialize(ev events.Event) (pgtype.JSONB, error) {
	event := ev.(events.APIFailureEvent)
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
func (codec *apiFailureEventCodec) Deserialize(data pgtype.JSONB) (events.Event, error) {
	tmp := dataRecord{}
	err := data.AssignTo(&tmp)
	res := events.APIFailureEvent{
		Attempt: (uint)(tmp["attempt"].(float64)),
		Failure: tmp["failure"].(string),
	}
	return res, err
}

type apiTimeoutEventCodec struct{}

// Class implements the PostgreSQLEventCodec interface.
func (codec *apiTimeoutEventCodec) Class() string {
	return "api-timeout"
}

// Serialize implements the PostgreSQLEventCodec interface.
func (codec *apiTimeoutEventCodec) Serialize(ev events.Event) (pgtype.JSONB, error) {
	event := ev.(events.APITimeoutEvent)
	res := pgtype.JSONB{}
	err := res.Set(
		dataRecord{
			"attempt": event.Attempt,
		},
	)
	if err != nil {
		return pgtype.JSONB{}, err
	}
	return res, nil
}

// Deserialize implements the PostgreSQLEventCodec interface.
func (codec *apiTimeoutEventCodec) Deserialize(data pgtype.JSONB) (events.Event, error) {
	tmp := dataRecord{}
	err := data.AssignTo(&tmp)
	res := events.APITimeoutEvent{
		Attempt: (uint)(tmp["attempt"].(float64)),
	}
	return res, err
}
