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

// the requestEvent represents a request that should be sent to the API
type requestEvent struct {
	request string
}

// Class implements the Event interface.
func (e *requestEvent) Class() string {
	return "request"
}

// the responseEvent represents a response received from the API
// Note that this does not discriminate between success or failure. Rather,
// any response is stored here.
type responseEvent struct {
	response string
}

// Class implements the Event interface.
func (e *responseEvent) Class() string {
	return "response"
}

// the failureEvent represents a failure while trying to send a request to the API
// By its very nature, these kinds of failure are generated locally, like e.g.
// the failure to resolve a remote name.
type failureEvent struct {
	failure string
}

// Class implements the Event interface.
func (e *failureEvent) Class() string {
	return "failure"
}
