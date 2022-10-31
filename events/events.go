package events

// This file defines event types and interfaces.

import ()

// The Event interface defines methods common to events.
type Event interface {
	// Class returns a string that identifies the event type.
	// This is distinct from Go's type name because it should be
	// agnostic of the language.
	Class() string
}

// SimpleEvent models a simple event with a message but without any
// further meaning for the API broker.
type SimpleEvent struct {
	Message string
}

// Class implements the Event interface.
func (e SimpleEvent) Class() string {
	return "simple"
}
