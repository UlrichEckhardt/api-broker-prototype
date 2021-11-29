package main

// This file implements an API stub.
// Normally, this would communicate with a service using e.g. HTTP. This code
// here only serves as example.

import (
	"errors"
	"math/rand"
	"time"
)

// configuration values
var conf struct {
	failureRate       float64
	silentFailureRate float64
	minDuration       float64
	maxDuration       float64
}

// ConfigureStub can be used to configure the behaviour of the API stub.
func ConfigureStub(failureRate float64, silentFailureRate float64, minDuration, maxDuration float64) {
	conf.failureRate = failureRate
	conf.silentFailureRate = silentFailureRate
	conf.minDuration = minDuration
	conf.maxDuration = maxDuration
}

// ProcessRequest handles a single request. It returns a response string on
// success or an error on failure. Note that similar to the APIResponseEvent,
// it doesn't distinguish between an answer that signals success or failure
// so any error returned was caused locally not received from remote.
func ProcessRequest(request string) (*string, error) {
	// add a random delay
	delay := time.Duration((conf.minDuration + rand.Float64()*(conf.maxDuration-conf.minDuration)) * float64(time.Second))
	time.Sleep(delay)

	if rand.Float64() >= conf.failureRate {
		// successful call
		res := "response"
		return &res, nil
	}

	if rand.Float64() >= conf.silentFailureRate {
		// verbose failure
		return nil, errors.New("failure")
	}

	// silent failure
	return nil, nil
}
