package events

import (
	"errors"
	"testing"
)

func TestDuplicateEventUUID(t *testing.T) {
	// make sure the type implements the `error` interface
	var err error = DuplicateEventUUID

	if !errors.Is(err, DuplicateEventUUID) {
		t.Error("error type is not recognized")
	}
}
