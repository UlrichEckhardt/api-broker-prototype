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
			},
			data: `{"retries":2}`,
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

func TestResponseCodec(t *testing.T) {
	var codec PostgreSQLEventCodec = &responseEventCodec{}

	cases := map[string]successCase{
		"test 1": {
			event: events.ResponseEvent{
				Response: "some response",
			},
			data: `{"response":"some response"}`,
		},
	}

	for name, c := range cases {
		runSuccessCase(name, c, codec, t)
	}
}

func TestFailureCodec(t *testing.T) {
	var codec PostgreSQLEventCodec = &failureEventCodec{}

	cases := map[string]successCase{
		"test 1": {
			event: events.FailureEvent{
				Failure: "some failure",
			},
			data: `{"failure":"some failure"}`,
		},
	}

	for name, c := range cases {
		runSuccessCase(name, c, codec, t)
	}
}
