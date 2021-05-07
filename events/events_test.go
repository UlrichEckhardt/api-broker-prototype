package events

import (
	"testing"
)

func TestSimpleEvent(t *testing.T) {
	// make sure the event implements the event interface
	var event Event = SimpleEvent{}

	if event.Class() != "simple" {
		t.Error("unexpected class value")
		return
	}
}

func TestConfigurationEvent(t *testing.T) {
	// make sure the event implements the event interface
	var event Event = ConfigurationEvent{}

	if event.Class() != "configuration" {
		t.Error("unexpected class value")
		return
	}
}

func TestRequestEvent(t *testing.T) {
	// make sure the event implements the event interface
	var event Event = RequestEvent{}

	if event.Class() != "request" {
		t.Error("unexpected class value")
		return
	}
}

func TestResponseEvent(t *testing.T) {
	// make sure the event implements the event interface
	var event Event = ResponseEvent{}

	if event.Class() != "response" {
		t.Error("unexpected class value")
		return
	}
}

func TestFailureEvent(t *testing.T) {
	// make sure the event implements the event interface
	var event Event = FailureEvent{}

	if event.Class() != "failure" {
		t.Error("unexpected class value")
		return
	}
}
