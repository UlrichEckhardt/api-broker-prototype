package broker

import (
	"api-broker-prototype/events"
	"api-broker-prototype/mock_api"
	"context"
	"time"

	"github.com/inconshreveable/log15"
)

// utility function to invoke the API and store the result as event
func startApiCall(ctx context.Context, store events.EventStore, logger log15.Logger, event RequestEvent, causationID int32, attempt uint) {
	// emit event that a request was started
	store.Insert(
		ctx,
		APIRequestEvent{
			Attempt: attempt,
		},
		causationID,
	)

	// TODO: this accesses `store` asynchronously, which may need synchronization
	go func() {
		// delegate to mock API
		response, err := mock_api.ProcessRequest(event.Request)

		// store results as event
		if response != nil {
			store.Insert(
				ctx,
				APIResponseEvent{
					Attempt:  attempt,
					Response: *response,
				},
				causationID,
			)
		} else if err != nil {
			store.Insert(
				ctx,
				APIFailureEvent{
					Attempt: attempt,
					Failure: err.Error(),
				},
				causationID,
			)
		} else {
			logger.Info("No response from API.")
		}
	}()
}

// state representing the state a request is in
// This is applied both to the initial request from the client and the single
// requests made to the API.
type requestState int

const (
	state_pending requestState = iota
	state_success
	state_failure
	state_timeout
)

func (d requestState) String() string {
	switch d {
	case state_pending:
		return "pending"
	case state_success:
		return "success"
	case state_failure:
		return "failure"
	case state_timeout:
		return "timeout"
	default:
		return ""
	}
}

// metadata for a request
type requestData struct {
	request  events.Envelope
	retries  uint
	attempts map[uint]requestState
}

func newRequestData(request events.Envelope, retries uint) *requestData {
	return &requestData{
		request:  request,
		retries:  retries,
		attempts: make(map[uint]requestState),
	}
}

// query the number of retries for this request
func (request *requestData) Retries() uint {
	return request.retries
}

// query whether any attempt for the request succeeded
func (d *requestData) Succeeded() bool {
	for _, val := range d.attempts {
		if val == state_success {
			return true
		}
	}
	return false
}

// index of the next attempt
func (request *requestData) NextAttempt() uint {
	return uint(len(request.attempts))
}

// determine overall state of the request
// pending: API is being queried
// success: response received
// failure: request failed
func (request *requestData) State() requestState {
	// the default value is pending, which means no actual requests have been made
	res := state_pending

	for i, attempt := range request.attempts {
		res = attempt
		if res == state_success {
			// first successful response makes the whole request successful
			break
		}
		if i < request.Retries() {
			// if there are still some attempts left, the overall result is still pending
			res = state_pending
		}
	}
	return res
}

// ProcessRequests processes request events from the store.
func ProcessRequests(ctx context.Context, store events.EventStore, logger log15.Logger, lastProcessedID int32) error {
	// wrap the actual event store with the timeout handling decorator
	store, err := NewTimeoutEventStoreDecorator(store, logger)
	if err != nil {
		return err
	}

	ch, err := store.FollowEvents(ctx, lastProcessedID)
	if err != nil {
		return err
	}

	// number of retries after a failed request
	retries := uint(0)
	// maximum duration before considering an attempt failed
	timeout := 5 * time.Second

	// map of requests being processed currently
	// Key is the event ID of the initial event (`RequestEvent`), which is
	// used as causation ID in future events associated with this request.
	requests := make(map[int32]*requestData)

	// process events from the channel
	for envelope := range ch {
		logger.Info(
			"processing event",
			"id", envelope.ID(),
			"class", envelope.Event().Class(),
			"created", envelope.Created().Format(time.RFC3339),
			"causation_id", envelope.CausationID(),
			"data", envelope.Event(),
		)

		switch event := envelope.Event().(type) {
		case ConfigurationEvent:
			logger.Info(
				"updating API configuration",
				"retries", event.Retries,
				"timeout", event.Timeout,
			)

			// store configuration
			if event.Retries >= 0 {
				retries = uint(event.Retries)
			}
			if event.Timeout >= 0 {
				timeout = time.Duration(event.Timeout * float64(time.Second))
			}
			logger.Info(
				"updated API configuration",
				"retries", retries,
				"timeout", timeout,
			)

		case RequestEvent:
			logger.Info("starting request processing")

			// create record to correlate the results with it
			request := newRequestData(envelope, retries)
			requests[envelope.ID()] = request

			// try event processing asynchronously
			startApiCall(ctx, store, logger, event, envelope.ID(), 0)

		case APIRequestEvent:
			// fetch the request data
			requestID := envelope.CausationID()
			if requestID == 0 {
				logger.Error("event lacks a causation ID to locate the request")
				break
			}
			request := requests[requestID]
			if request == nil {
				logger.Error("failed to locate request data")
				break
			}

			// mark request as pending
			request.attempts[event.Attempt] = state_pending

			logger.Info(
				"starting API call",
				"attempt", event.Attempt,
			)

		case APIResponseEvent:
			// fetch the request data
			requestID := envelope.CausationID()
			if requestID == 0 {
				logger.Error("event lacks a causation ID to locate the request")
				break
			}
			request := requests[requestID]
			if request == nil {
				logger.Error("failed to locate request data")
				break
			}

			// mark request as successful
			request.attempts[event.Attempt] = state_success
			logger.Info("completed API call")

		case APIFailureEvent:
			// fetch the request data
			requestID := envelope.CausationID()
			if requestID == 0 {
				logger.Error("event lacks a causation ID to locate the request")
				break
			}
			request := requests[requestID]
			if request == nil {
				logger.Error("failed to locate request event")
				break
			}

			// mark request as failed
			request.attempts[event.Attempt] = state_failure
			logger.Info("failed API call")

			// check if any retries remain
			if event.Attempt == request.Retries() {
				logger.Info("retries exhausted")
				break
			}

			// If a retry for this unsuccessful attempt was already made, there
			// is nothing to do here.
			if event.Attempt+1 != request.NextAttempt() {
				logger.Info("retry attempt already started")
				break
			}

			// check if a retry or a previous attempt succeeded in the meantime
			if request.Succeeded() {
				break
			}

			// retry event processing asynchronously
			startApiCall(ctx, store, logger, request.request.Event().(RequestEvent), request.request.ID(), event.Attempt+1)

		case APITimeoutEvent:
			// fetch the request data
			requestID := envelope.CausationID()
			if requestID == 0 {
				logger.Error("timeout event lacks a causation ID to locate the request")
				break
			}
			request := requests[requestID]
			if request == nil {
				logger.Error("failed to locate request event")
				break
			}

			// A timeout event can only transition the state from "pending" to
			// "timeout". Other states like "failure" or "success" are final.
			if request.attempts[event.Attempt] != state_pending {
				break
			}

			// mark request as timed out
			request.attempts[event.Attempt] = state_timeout
			logger.Info("API call timed out")

			// check if any retries remain
			if event.Attempt == request.Retries() {
				logger.Info("retries exhausted")
				break
			}

			// If a retry for this unsuccessful attempt was already made, there
			// is nothing to do here.
			if event.Attempt+1 != request.NextAttempt() {
				logger.Info("retry attempt already started")
				break
			}

			// check if a retry or a previous attempt succeeded in the meantime
			if request.Succeeded() {
				break
			}

			// retry event processing asynchronously
			startApiCall(ctx, store, logger, request.request.Event().(RequestEvent), request.request.ID(), event.Attempt+1)
		}
	}

	return store.Error()
}

// WatchRequests watches requests as they are processed
func WatchRequests(ctx context.Context, store events.EventStore, logger log15.Logger, lastProcessedID int32) error {
	ch, err := store.FollowEvents(ctx, lastProcessedID)
	if err != nil {
		return err
	}

	// map of requests being processed currently
	// Key is the event ID of the initial event (`RequestEvent`), which is
	// used as causation ID in future events associated with this request.
	requests := make(map[int32]*requestData)

	// number of retries after a failed request
	retries := uint(0)

	// process events from the channel
	for envelope := range ch {

		switch event := envelope.Event().(type) {
		case ConfigurationEvent:
			if event.Retries >= 0 {
				retries = uint(event.Retries)
			}

		case RequestEvent:
			// create record to correlate the results with it
			request := newRequestData(envelope, retries)
			requests[envelope.ID()] = request

			// create record to correlate the results with it
			logger.Info(
				"request received",
				"request ID", envelope.ID(),
				"state", request.State(),
			)

		case APIRequestEvent:
			// fetch the request data
			requestID := envelope.CausationID()
			if requestID == 0 {
				logger.Error("event lacks a causation ID to locate the request")
				break
			}
			request := requests[requestID]
			if request == nil {
				logger.Error("failed to locate request data")
				break
			}

			// mark request as pending
			request.attempts[event.Attempt] = state_pending

			logger.Info(
				"API request starting",
				"request ID", envelope.CausationID(),
				"state", request.State(),
				"attempt", event.Attempt,
			)

		case APIResponseEvent:
			// fetch the request data
			requestID := envelope.CausationID()
			if requestID == 0 {
				logger.Error("event lacks a causation ID to locate the request")
				break
			}
			request := requests[requestID]
			if request == nil {
				logger.Error("failed to locate request data")
				break
			}

			// mark request as successful
			request.attempts[event.Attempt] = state_success

			logger.Info(
				"API request succeeded",
				"request ID", envelope.CausationID(),
				"state", request.State(),
				"attempt", event.Attempt,
			)

		case APIFailureEvent:
			// fetch the request data
			requestID := envelope.CausationID()
			if requestID == 0 {
				logger.Error("event lacks a causation ID to locate the request")
				break
			}
			request := requests[requestID]
			if request == nil {
				logger.Error("failed to locate request data")
				break
			}

			// mark request as failed
			request.attempts[event.Attempt] = state_failure

			logger.Info(
				"API request failed",
				"request ID", envelope.CausationID(),
				"state", request.State(),
				"attempt", event.Attempt,
			)

		case APITimeoutEvent:
			// fetch the request data
			requestID := envelope.CausationID()
			if requestID == 0 {
				logger.Error("event lacks a causation ID to locate the request")
				break
			}
			request := requests[requestID]
			if request == nil {
				logger.Error("failed to locate request data")
				break
			}

			// A timeout event can only transition the state from "pending" to
			// "timeout". Other states like "failure" or "success" are final.
			if request.attempts[event.Attempt] == state_pending {
				request.attempts[event.Attempt] = state_timeout
			}

			logger.Info(
				"API request timeout elapsed",
				"request ID", envelope.CausationID(),
				"state", request.State(),
				"attempt", event.Attempt,
			)
		}
	}

	return store.Error()
}
