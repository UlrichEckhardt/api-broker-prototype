package broker

import (
	"api-broker-prototype/events"
	"testing"
	"time"
)

// mock for the events.Envelope interface
type envelopeMock struct{}

func (envelope envelopeMock) ID() int32 {
	return 42
}

func (envelope envelopeMock) Created() time.Time {
	return time.Now()
}

func (envelope envelopeMock) CausationID() int32 {
	return 23
}

func (envelope envelopeMock) Event() events.Event {
	return nil
}

func TestRequestData(t *testing.T) {
	envelope := envelopeMock{}
	retries := uint(1)

	t.Run("initial state", func(t *testing.T) {
		request := newRequestData(envelope, retries)
		if request == nil {
			t.Errorf("request is nil")
		}

		if len(request.attempts) != 2 {
			t.Errorf("attempt count is unexpected")
		}
		if request.Retries() != retries {
			t.Errorf("Retries() is unexpected")
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
		request := newRequestData(envelope, retries)
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
		request := newRequestData(envelope, retries)
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
		request := newRequestData(envelope, retries)
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
		request := newRequestData(envelope, retries)
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
		request := newRequestData(envelope, retries)
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
		request := newRequestData(envelope, retries)
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
		request := newRequestData(envelope, retries)
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
