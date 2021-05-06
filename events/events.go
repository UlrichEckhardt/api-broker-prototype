package events

// This file defines event types.

import ()

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
	Failure string
}

// Class implements the Event interface.
func (e FailureEvent) Class() string {
	return "failure"
}
