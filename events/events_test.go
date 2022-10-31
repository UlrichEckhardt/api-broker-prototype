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
