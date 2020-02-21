package main

import (
	"go.mongodb.org/mongo-driver/bson"
	"reflect"
	"testing"
)

type testcase struct {
	event *simpleEvent
	data  bson.M
	err   error
}

func TestCodec(t *testing.T) {
	cases := map[string]testcase{
		"test 1": {
			event: new(simpleEvent),
			data:  bson.M{"message": ""},
		},
		"test 2": {
			event: &simpleEvent{message: "some message"},
			data:  bson.M{"message": "some message"},
		},
	}

	var codec simpleEventCodec
	for name, c := range cases {
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
}
