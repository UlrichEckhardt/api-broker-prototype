package main

// This file defines event types.

import ()

type simpleEvent struct {
	message string
}

// Class implements the Event interface.
func (e *simpleEvent) Class() string {
	return "simple"
}
