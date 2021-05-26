package mongodb

import (
	"api-broker-prototype/events"
	"go.mongodb.org/mongo-driver/bson"
	"reflect"
	"testing"
)

type testcase struct {
	event events.Event
	data  bson.M
	err   error
}

func runTestcase(name string, c testcase, codec MongoDBEventCodec, t *testing.T) {
	if c.data != nil {
		// test deserializing
		t.Run(name, func(t *testing.T) {
			event, err := codec.Deserialize(c.data)

			if c.err == nil {
				// not expecting an error
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
			} else {
				// expecting an error
				if event != nil {
					t.Errorf("unexpected event %v", event)
					return
				}
				if err == nil {
					t.Error("expected error missing")
					return
				}
				if c.err != err {
					t.Log("expected error", c.err)
					t.Log("received error", err)
					t.Error("wrong error")
					return
				}
			}
		})
	}
	if c.event != nil {
		// test serializing
		t.Run(name, func(t *testing.T) {
			data, err := codec.Serialize(c.event)
			_ = data
			_ = err

			if c.err == nil {
				// not expecting an error
				if err != nil {
					t.Errorf("unexpected error %v", err)
					return
				}
				if data == nil {
					t.Error("expected data missing")
					return
				}
				if !reflect.DeepEqual(c.data, data) {
					t.Log("expected data", c.data)
					t.Log("received data", data)
					t.Errorf("data differ")
					return
				}
			} else {
				// expecting an error
				if data != nil {
					t.Errorf("unexpected data %v", data)
					return
				}
				if err == nil {
					t.Error("expected error missing")
					return
				}
				if c.err != err {
					t.Log("expected error", c.err)
					t.Log("received error", err)
					t.Error("wrong error")
					return
				}
			}
		})
	}
}

func TestSimpleCodec(t *testing.T) {
	cases := map[string]testcase{
		"test 1": {
			event: events.SimpleEvent{},
			data: bson.M{
				"message": "",
			},
		},
		"test 2": {
			event: events.SimpleEvent{
				Message: "some message",
			},
			data: bson.M{
				"message": "some message",
			},
		},
	}

	var codec SimpleEventCodec
	for name, c := range cases {
		runTestcase(name, c, &codec, t)
	}
}

func TestConfigurationCodec(t *testing.T) {
	cases := map[string]testcase{
		"test configuration": {
			event: events.ConfigurationEvent{
				Retries: 2,
			},
			data: bson.M{
				"retries": int32(2),
			},
		},
	}

	var codec ConfigurationEventCodec
	for name, c := range cases {
		runTestcase(name, c, &codec, t)
	}
}

func TestRequestCodec(t *testing.T) {
	cases := map[string]testcase{
		"test request": {
			event: events.RequestEvent{
				Request: "some request",
			},
			data: bson.M{
				"request": "some request",
			},
		},
	}

	var codec RequestEventCodec
	for name, c := range cases {
		runTestcase(name, c, &codec, t)
	}
}

func TestResponseCodec(t *testing.T) {
	cases := map[string]testcase{
		"test response": {
			event: events.ResponseEvent{
				Attempt:  uint(0),
				Response: "some response",
			},
			data: bson.M{
				"attempt":  int64(0),
				"response": "some response",
			},
		},
	}

	var codec ResponseEventCodec
	for name, c := range cases {
		runTestcase(name, c, &codec, t)
	}
}

func TestFailureCodec(t *testing.T) {
	cases := map[string]testcase{
		"test failure": {
			event: events.FailureEvent{
				Attempt: uint(0),
				Failure: "some failure",
			},
			data: bson.M{
				"attempt": int64(0),
				"failure": "some failure",
			},
		},
	}

	var codec FailureEventCodec
	for name, c := range cases {
		runTestcase(name, c, &codec, t)
	}
}
