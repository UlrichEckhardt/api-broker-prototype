package broker

import (
	"api-broker-prototype/events"
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

// mock for the events.Envelope interface
type envelopeMock struct{}

func (envelope envelopeMock) ID() int32 {
	return 42
}

func (envelope envelopeMock) ExternalUUID() uuid.UUID {
	return uuid.FromStringOrNil("22428f46-a2d8-4d51-b6b5-bc8551bd0921")
}

func (envelope envelopeMock) Created() time.Time {
	return time.Now()
}

func (envelope envelopeMock) CausationID() int32 {
	return 23
}

func (envelope envelopeMock) Event() events.Event {
	return RequestEvent{
		Request: "test request data",
	}
}

func TestRequestData(t *testing.T) {
	envelope := envelopeMock{}
	retries := uint(1)
	timeout := time.Second

	t.Run("initial state", func(t *testing.T) {
		request := newRequestData(envelope, retries, &timeout)
		if request == nil {
			t.Errorf("request is nil")
		}

		event := request.Event()
		if event.Request != "test request data" {
			t.Errorf("event data is unexpected")
		}
		if request.ID() != 42 {
			t.Errorf("ID() is unexpected")
		}
		if len(request.attempts) != 2 {
			t.Errorf("attempt count is unexpected")
		}
		if request.Retries() != retries {
			t.Errorf("Retries() is unexpected")
		}
		if request.Timeout() != &timeout {
			t.Errorf("Timeout() is unexpected")
		}
		if request.Succeeded() {
			t.Errorf("Succeeded() is unexpected")
		}
		if request.NextAttempt() != 0 {
			t.Errorf("NextAttempt() is unexpected")
		}
		if request.State() != state_pending {
			t.Errorf("State() is unexpected")
		}
	})

	t.Run("first pending", func(t *testing.T) {
		request := newRequestData(envelope, retries, &timeout)
		if request == nil {
			t.Errorf("request is nil")
		}

		request.attempts[0] = state_pending

		if request.Succeeded() {
			t.Errorf("Succeeded() is unexpected")
		}
		if request.NextAttempt() != 1 {
			t.Errorf("NextAttempt() is unexpected")
		}
		if request.State() != state_pending {
			t.Errorf("State() is unexpected")
		}
	})

	t.Run("first success", func(t *testing.T) {
		request := newRequestData(envelope, retries, &timeout)
		if request == nil {
			t.Errorf("request is nil")
		}

		request.attempts[0] = state_success

		if !request.Succeeded() {
			t.Errorf("Succeeded() is unexpected")
		}
		if request.NextAttempt() != 1 {
			t.Errorf("NextAttempt() is unexpected")
		}
		if request.State() != state_success {
			t.Errorf("State() is unexpected")
		}
	})

	t.Run("first failure", func(t *testing.T) {
		request := newRequestData(envelope, retries, &timeout)
		if request == nil {
			t.Errorf("request is nil")
		}

		request.attempts[0] = state_failure

		if request.Succeeded() {
			t.Errorf("Succeeded() is unexpected")
		}
		if request.NextAttempt() != 1 {
			t.Errorf("NextAttempt() is unexpected")
		}
		if request.State() != state_pending {
			t.Errorf("State() is unexpected")
		}
	})

	t.Run("first timeout", func(t *testing.T) {
		request := newRequestData(envelope, retries, &timeout)
		if request == nil {
			t.Errorf("request is nil")
		}

		request.attempts[0] = state_timeout

		if request.Succeeded() {
			t.Errorf("Succeeded() is unexpected")
		}
		if request.NextAttempt() != 1 {
			t.Errorf("NextAttempt() is unexpected")
		}
		if request.State() != state_pending {
			t.Errorf("State() is unexpected")
		}
	})

	t.Run("second success", func(t *testing.T) {
		request := newRequestData(envelope, retries, &timeout)
		if request == nil {
			t.Errorf("request is nil")
		}

		request.attempts[0] = state_failure
		request.attempts[1] = state_success

		if !request.Succeeded() {
			t.Errorf("Succeeded() is unexpected")
		}
		if request.NextAttempt() != 2 {
			t.Errorf("NextAttempt() is unexpected")
		}
		if request.State() != state_success {
			t.Errorf("State() is unexpected")
		}
	})

	t.Run("second failure", func(t *testing.T) {
		request := newRequestData(envelope, retries, &timeout)
		if request == nil {
			t.Errorf("request is nil")
		}

		request.attempts[0] = state_failure
		request.attempts[1] = state_failure

		if request.Succeeded() {
			t.Errorf("Succeeded() is unexpected")
		}
		if request.NextAttempt() != 2 {
			t.Errorf("NextAttempt() is unexpected")
		}
		if request.State() != state_failure {
			t.Errorf("State() is unexpected")
		}
	})

	t.Run("second timeout", func(t *testing.T) {
		request := newRequestData(envelope, retries, &timeout)
		if request == nil {
			t.Errorf("request is nil")
		}

		request.attempts[0] = state_timeout
		request.attempts[1] = state_timeout

		if request.Succeeded() {
			t.Errorf("Succeeded() is unexpected")
		}
		if request.NextAttempt() != 2 {
			t.Errorf("NextAttempt() is unexpected")
		}
		if request.State() != state_timeout {
			t.Errorf("State() is unexpected")
		}
	})
}
