package broker

import (
	"api-broker-prototype/events"
	"testing"
)

func TestConfigurationEvent(t *testing.T) {

	// make sure the event implements the event interface
	var event events.Event = ConfigurationEvent{}

	if event.Class() != "configuration" {
		t.Error("unexpected class value")
		return
	}
}

func TestRequestEvent(t *testing.T) {
	// make sure the event implements the event interface
	var event events.Event = RequestEvent{}

	if event.Class() != "request" {
		t.Error("unexpected class value")
		return
	}
}

func TestAPIRequestEvent(t *testing.T) {
	// make sure the event implements the event interface
	var event events.Event = APIRequestEvent{}

	if event.Class() != "api-request" {
		t.Error("unexpected class value")
		return
	}
}

func TestAPIResponseEvent(t *testing.T) {
	// make sure the event implements the event interface
	var event events.Event = APIResponseEvent{}

	if event.Class() != "api-response" {
		t.Error("unexpected class value")
		return
	}
}

func TestAPIFailureEvent(t *testing.T) {
	// make sure the event implements the event interface
	var event events.Event = APIFailureEvent{}

	if event.Class() != "api-failure" {
		t.Error("unexpected class value")
		return
	}
}
