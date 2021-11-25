package postgresql

import (
	"api-broker-prototype/events"
	"github.com/jackc/pgtype"
	"reflect"
	"testing"
)

type successCase struct {
	event events.Event
	data  string
}

func runSuccessCase(name string, c successCase, codec PostgreSQLEventCodec, t *testing.T) {
	if codec.Class() != c.event.Class() {
		t.Error("codec/event class mismatch")
		return
	}

	// test deserializing
	t.Run(name+" deserialising", func(t *testing.T) {
		record := pgtype.JSONB{
			Bytes:  []byte(c.data),
			Status: pgtype.Present,
		}

		event, err := codec.Deserialize(record)

		if err != nil {
			t.Errorf("unexpected error %v", err)
			return
		}
		if event == nil {
			t.Error("expected event missing")
			return
		}
		if !reflect.DeepEqual(c.event, event) {
			t.Log("expected event", c.event)
			t.Log("received event", event)
			t.Errorf("events differ")
			return
		}
	})

	// test serializing
	t.Run(name+" serialising", func(t *testing.T) {
		record := pgtype.JSONB{
			Bytes:  []byte(c.data),
			Status: pgtype.Present,
		}

		data, err := codec.Serialize(c.event)

		if err != nil {
			t.Errorf("unexpected error %v", err)
			return
		}
		if data.Status != pgtype.Present {
			t.Error("expected data missing")
			return
		}
		if !reflect.DeepEqual(record, data) {
			t.Log("expected data", record)
			t.Log("received data", data)
			t.Errorf("data differ")
			return
		}
	})
}

func TestSimpleCodec(t *testing.T) {
	var codec PostgreSQLEventCodec = &simpleEventCodec{}

	cases := map[string]successCase{
		"test 1": {
			event: events.SimpleEvent{},
			data:  `{"message":""}`,
		},
		"test 2": {
			event: events.SimpleEvent{
				Message: "some message",
			},
			data: `{"message":"some message"}`,
		},
	}

	for name, c := range cases {
		runSuccessCase(name, c, codec, t)
	}
}

func TestConfigurationCodec(t *testing.T) {
	var codec PostgreSQLEventCodec = &configurationEventCodec{}

	cases := map[string]successCase{
		"test 1": {
			event: events.ConfigurationEvent{
				Retries: 2,
				Timeout: 2.5,
			},
			data: `{"retries":2,"timeout":2.5}`,
		},
	}

	for name, c := range cases {
		runSuccessCase(name, c, codec, t)
	}
}

func TestRequestCodec(t *testing.T) {
	var codec PostgreSQLEventCodec = &requestEventCodec{}

	cases := map[string]successCase{
		"test 1": {
			event: events.RequestEvent{
				Request: "some request",
			},
			data: `{"request":"some request"}`,
		},
	}

	for name, c := range cases {
		runSuccessCase(name, c, codec, t)
	}
}

func TestAPIResponseCodec(t *testing.T) {
	var codec PostgreSQLEventCodec = &apiResponseEventCodec{}

	cases := map[string]successCase{
		"test 1": {
			event: events.APIResponseEvent{
				Response: "some response",
			},
			data: `{"attempt":0,"response":"some response"}`,
		},
		"test 2": {
			event: events.APIResponseEvent{
				Attempt:  uint(4),
				Response: "some response",
			},
			data: `{"attempt":4,"response":"some response"}`,
		},
	}

	for name, c := range cases {
		runSuccessCase(name, c, codec, t)
	}
}

func TestAPIFailureCodec(t *testing.T) {
	var codec PostgreSQLEventCodec = &apiFailureEventCodec{}

	cases := map[string]successCase{
		"test 1": {
			event: events.APIFailureEvent{
				Failure: "some failure",
			},
			data: `{"attempt":0,"failure":"some failure"}`,
		},
		"test 2": {
			event: events.APIFailureEvent{
				Attempt: uint(4),
				Failure: "some failure",
			},
			data: `{"attempt":4,"failure":"some failure"}`,
		},
	}

	for name, c := range cases {
		runSuccessCase(name, c, codec, t)
	}
}

func TestAPITimeoutCodec(t *testing.T) {
	var codec PostgreSQLEventCodec = &apiTimeoutEventCodec{}

	cases := map[string]successCase{
		"test timeout": {
			event: events.APITimeoutEvent{
				Attempt: uint(0),
			},
			data: `{"attempt":0}`,
		},
	}

	for name, c := range cases {
		runSuccessCase(name, c, codec, t)
	}
}
