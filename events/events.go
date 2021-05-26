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

// ConfigurationEvent models an event that contains configuration settings
// for the way the API is used.
type ConfigurationEvent struct {
	Retries int32 // number of retries after a failure
}

// Class implements the Event interface.
func (e ConfigurationEvent) Class() string {
	return "configuration"
}

// the RequestEvent represents a request that should be sent to the API
type RequestEvent struct {
	Request string
}

// Class implements the Event interface.
func (e RequestEvent) Class() string {
	return "request"
}

// the ResponseEvent represents a response received from the API
// Note that this does not discriminate between success or failure. Rather,
// any response is stored here.
type ResponseEvent struct {
	Attempt  uint // zero-based index of the attempt
	Response string
}

// Class implements the Event interface.
func (e ResponseEvent) Class() string {
	return "response"
}

// the FailureEvent represents a failure while trying to send a request to the API
// By its very nature, these kinds of failure are generated locally, like e.g.
// the failure to resolve a remote name.
type FailureEvent struct {
	Attempt uint // zero-based index of the attempt
	Failure string
}

// Class implements the Event interface.
func (e FailureEvent) Class() string {
	return "failure"
}
