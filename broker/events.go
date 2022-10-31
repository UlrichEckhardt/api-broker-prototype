package broker

// This file defines various events handled or emitted here

import ()

// ConfigurationEvent models an event that contains configuration settings
// for the way the API is used.
type ConfigurationEvent struct {
	Retries int32   // number of retries after a failure
	Timeout float64 // timeout for each attempt
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

// the APIRequestEvent represents a communication attempt with the API
// When starting the communication attempt, this event is emitted.
type APIRequestEvent struct {
	Attempt uint // zero-based index of the attempt
}

// Class implements the Event interface.
func (e APIRequestEvent) Class() string {
	return "api-request"
}

// the APIResponseEvent represents a response received from the API
// Note that this does not discriminate between success or failure. Rather,
// any response is stored here without interpretation.
type APIResponseEvent struct {
	Attempt  uint // zero-based index of the attempt
	Response string
}

// Class implements the Event interface.
func (e APIResponseEvent) Class() string {
	return "api-response"
}

// the APIFailureEvent represents a failure while trying to send a request to the API
// By its very nature, these kinds of failure are generated locally, like e.g.
// the failure to resolve the remote DNS name to an IP. It does not represent a
// response received from remote which contains an error.
type APIFailureEvent struct {
	Attempt uint // zero-based index of the attempt
	Failure string
}

// Class implements the Event interface.
func (e APIFailureEvent) Class() string {
	return "api-failure"
}

// the APITimeoutEvent signals that the timeout for a response has elapsed
// Note that this event is only meaningful if neither response nor failure
// occurred. Further, if a response is received after this timeout event, an
// interpretation would most likely use that and ignore the timeout. This
// event type is only tied to the attempt that timed out and doesn't carry
// any further data, simply because it represents the absence of data.
type APITimeoutEvent struct {
	Attempt uint // zero-based index of the attempt
}

// Class implements the Event interface.
func (e APITimeoutEvent) Class() string {
	return "api-timeout"
}
