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

func TestAPIRequestEvent(t *testing.T) {
	// make sure the event implements the event interface
	var event Event = APIRequestEvent{}

	if event.Class() != "api-request" {
		t.Error("unexpected class value")
		return
	}
}

func TestAPIResponseEvent(t *testing.T) {
	// make sure the event implements the event interface
	var event Event = APIResponseEvent{}

	if event.Class() != "api-response" {
		t.Error("unexpected class value")
		return
	}
}

func TestAPIFailureEvent(t *testing.T) {
	// make sure the event implements the event interface
	var event Event = APIFailureEvent{}

	if event.Class() != "api-failure" {
		t.Error("unexpected class value")
		return
	}
}
