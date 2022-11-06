package broker

import (
	"api-broker-prototype/events"
	"api-broker-prototype/mock_api"
	"context"
	"time"

	"github.com/inconshreveable/log15"
)

// state representing the state a request is in
// This is applied both to the initial request from the client and the single
// requests made to the API.
type requestState int

const (
	state_initial requestState = iota
	state_pending
	state_success
	state_failure
	state_timeout
)

func (d requestState) String() string {
	switch d {
	case state_initial:
		return "initial"
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
	envelope events.Envelope
	attempts []requestState
}

func newRequestData(request events.Envelope, retries uint) *requestData {
	return &requestData{
		envelope: request,
		attempts: make([]requestState, retries+1),
	}
}

// retrieve the request event
func (request *requestData) Event() RequestEvent {
	return request.envelope.Event().(RequestEvent)
}

// retrieve the ID from the envelope
func (request *requestData) ID() int32 {
	return request.envelope.ID()
}

// query the number of retries for this request
func (request *requestData) Retries() uint {
	return uint(len(request.attempts) - 1)
}

// query whether any attempt for the request succeeded
func (request *requestData) Succeeded() bool {
	for _, val := range request.attempts {
		if val == state_success {
			return true
		}
	}
	return false
}

// index of the next attempt
func (request *requestData) NextAttempt() uint {
	i := uint(0)
	for _, val := range request.attempts {
		if val == state_initial {
			break
		}
		i++
	}
	return i
}

// determine overall state of the request
func (request *requestData) State() requestState {
	// the default value is pending, which means no actual requests have been made
	res := state_pending
	for _, val := range request.attempts {
		switch val {
		case state_initial:
			// this and further attempts are initial, so the overall state is undetermined yet
			return state_pending
		case state_pending, state_failure, state_timeout:
			// temporary state, store it but keep looking
			res = val
		case state_success:
			// state is final, store it
			return val
		}
	}
	return res
}

// Working data for the request processor.
//
// The RequestProcessor processes API-related events. It controls communication
// with the API and related tasks like timeout handling and retries.
type RequestProcessor struct {
	store  events.EventStore
	logger log15.Logger
	// number of retries after a failed request
	retries uint
	// maximum duration before considering an attempt failed
	timeout time.Duration
}

func NewRequestProcessor(store events.EventStore, logger log15.Logger) (*RequestProcessor, error) {
	// wrap the actual event store with the timeout handling decorator
	timeoutDecorator, err := NewTimeoutEventStoreDecorator(store, logger)
	if err != nil {
		return nil, err
	}

	return &RequestProcessor{
		store:  timeoutDecorator,
		logger: logger,
	}, nil
}

// utility function to invoke the API and store the result as event
func (handler *RequestProcessor) startApiCall(ctx context.Context, request *requestData) {
	event := request.Event()
	causationID := request.ID()
	attempt := request.NextAttempt()

	// emit event that a request was started
	handler.store.Insert(
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
			handler.store.Insert(
				ctx,
				APIResponseEvent{
					Attempt:  attempt,
					Response: *response,
				},
				causationID,
			)
		} else if err != nil {
			handler.store.Insert(
				ctx,
				APIFailureEvent{
					Attempt: attempt,
					Failure: err.Error(),
				},
				causationID,
			)
		} else {
			handler.logger.Info("No response from API.")
		}
	}()
}

// ProcessRequests processes request events from the store.
func (handler *RequestProcessor) Run(ctx context.Context, lastProcessedID int32) error {
	ch, err := handler.store.FollowEvents(ctx, lastProcessedID)
	if err != nil {
		return err
	}

	// map of requests being processed currently
	// Key is the event ID of the initial event (`RequestEvent`), which is
	// used as causation ID in future events associated with this request.
	requests := make(map[int32]*requestData)

	// process events from the channel
	for envelope := range ch {
		handler.logger.Info(
			"processing event",
			"id", envelope.ID(),
			"class", envelope.Event().Class(),
			"created", envelope.Created().Format(time.RFC3339),
			"causation_id", envelope.CausationID(),
			"data", envelope.Event(),
		)

		switch event := envelope.Event().(type) {
		case ConfigurationEvent:
			handler.logger.Info(
				"updating API configuration",
				"retries", event.Retries,
				"timeout", event.Timeout,
			)

			// store configuration
			if event.Retries >= 0 {
				handler.retries = uint(event.Retries)
			}
			if event.Timeout >= 0 {
				handler.timeout = time.Duration(event.Timeout * float64(time.Second))
			}
			handler.logger.Info(
				"updated API configuration",
				"retries", handler.retries,
				"timeout", handler.timeout,
			)

		case RequestEvent:
			handler.logger.Info("starting request processing")

			// create record to correlate the results with it
			request := newRequestData(envelope, handler.retries)
			requests[envelope.ID()] = request

			// try event processing asynchronously
			handler.startApiCall(ctx, request)

		case APIRequestEvent:
			// fetch the request data
			requestID := envelope.CausationID()
			if requestID == 0 {
				handler.logger.Error("event lacks a causation ID to locate the request")
				break
			}
			request := requests[requestID]
			if request == nil {
				handler.logger.Error("failed to locate request data")
				break
			}

			// mark request as pending
			request.attempts[event.Attempt] = state_pending

			handler.logger.Info(
				"starting API call",
				"attempt", event.Attempt,
			)

		case APIResponseEvent:
			// fetch the request data
			requestID := envelope.CausationID()
			if requestID == 0 {
				handler.logger.Error("event lacks a causation ID to locate the request")
				break
			}
			request := requests[requestID]
			if request == nil {
				handler.logger.Error("failed to locate request data")
				break
			}

			// mark request as successful
			request.attempts[event.Attempt] = state_success
			handler.logger.Info("completed API call")

		case APIFailureEvent:
			// fetch the request data
			requestID := envelope.CausationID()
			if requestID == 0 {
				handler.logger.Error("event lacks a causation ID to locate the request")
				break
			}
			request := requests[requestID]
			if request == nil {
				handler.logger.Error("failed to locate request event")
				break
			}

			// mark request as failed
			request.attempts[event.Attempt] = state_failure
			handler.logger.Info("failed API call")

			// check if any retries remain
			if event.Attempt == request.Retries() {
				handler.logger.Info("retries exhausted")
				break
			}

			// If a retry for this unsuccessful attempt was already made, there
			// is nothing to do here. This happens when the timeout elapsed
			// before the failure response was received.
			if event.Attempt+1 != request.NextAttempt() {
				handler.logger.Info("retry attempt already started")
				break
			}

			// check if a retry or a previous attempt succeeded in the meantime
			if request.Succeeded() {
				handler.logger.Info("request already succeeded, no need for a retry")
				break
			}

			// retry event processing asynchronously
			handler.startApiCall(ctx, request)

		case APITimeoutEvent:
			// fetch the request data
			requestID := envelope.CausationID()
			if requestID == 0 {
				handler.logger.Error("timeout event lacks a causation ID to locate the request")
				break
			}
			request := requests[requestID]
			if request == nil {
				handler.logger.Error("failed to locate request event")
				break
			}

			// A timeout event can only transition the state from "pending" to
			// "timeout". Other states like "failure" or "success" are final.
			if request.attempts[event.Attempt] != state_pending {
				break
			}

			// mark request as timed out
			request.attempts[event.Attempt] = state_timeout
			handler.logger.Info("API call timed out")

			// check if any retries remain
			if event.Attempt == request.Retries() {
				handler.logger.Info("retries exhausted")
				break
			}

			// If a retry for this unsuccessful attempt was already made, there
			// is nothing to do here. This happens when the failure response was
			// received before the timeout elapsed.
			if event.Attempt+1 != request.NextAttempt() {
				handler.logger.Info("retry attempt already started")
				break
			}

			// check if a retry or a previous attempt succeeded in the meantime
			if request.Succeeded() {
				handler.logger.Info("request already succeeded, no need for a retry")
				break
			}

			// retry event processing asynchronously
			handler.startApiCall(ctx, request)
		}
	}

	return handler.store.Error()
}

// Working data for a request observer.
//
// The RequestWatcher observes API-related events and sends the resulting
// status to the logger.
type RequestWatcher struct {
	store  events.EventStore
	logger log15.Logger
	// number of retries after a failed request
	retries uint
}

func NewRequestWatcher(store events.EventStore, logger log15.Logger) (*RequestWatcher, error) {
	return &RequestWatcher{
		store:  store,
		logger: logger,
	}, nil
}

// WatchRequests watches requests as they are processed
func (handler *RequestWatcher) Run(ctx context.Context, lastProcessedID int32) error {
	ch, err := handler.store.FollowEvents(ctx, lastProcessedID)
	if err != nil {
		return err
	}

	// map of requests being processed currently
	// Key is the event ID of the initial event (`RequestEvent`), which is
	// used as causation ID in future events associated with this request.
	requests := make(map[int32]*requestData)

	// process events from the channel
	for envelope := range ch {

		switch event := envelope.Event().(type) {
		case ConfigurationEvent:
			// store configuration
			if event.Retries >= 0 {
				handler.retries = uint(event.Retries)
			}

		case RequestEvent:
			// create record to correlate the results with it
			request := newRequestData(envelope, handler.retries)
			requests[envelope.ID()] = request

			// create record to correlate the results with it
			handler.logger.Info(
				"request received",
				"request ID", request.ID(),
				"state", request.State(),
			)

		case APIRequestEvent:
			// fetch the request data
			requestID := envelope.CausationID()
			if requestID == 0 {
				handler.logger.Error("event lacks a causation ID to locate the request")
				break
			}
			request := requests[requestID]
			if request == nil {
				handler.logger.Error("failed to locate request data")
				break
			}

			// mark request as pending
			request.attempts[event.Attempt] = state_pending

			handler.logger.Info(
				"API request starting",
				"request ID", request.ID(),
				"state", request.State(),
				"attempt", event.Attempt,
			)

		case APIResponseEvent:
			// fetch the request data
			requestID := envelope.CausationID()
			if requestID == 0 {
				handler.logger.Error("event lacks a causation ID to locate the request")
				break
			}
			request := requests[requestID]
			if request == nil {
				handler.logger.Error("failed to locate request data")
				break
			}

			// mark request as successful
			request.attempts[event.Attempt] = state_success

			handler.logger.Info(
				"API request succeeded",
				"request ID", request.ID(),
				"state", request.State(),
				"attempt", event.Attempt,
			)

		case APIFailureEvent:
			// fetch the request data
			requestID := envelope.CausationID()
			if requestID == 0 {
				handler.logger.Error("event lacks a causation ID to locate the request")
				break
			}
			request := requests[requestID]
			if request == nil {
				handler.logger.Error("failed to locate request data")
				break
			}

			// mark request as failed
			request.attempts[event.Attempt] = state_failure

			handler.logger.Info(
				"API request failed",
				"request ID", request.ID(),
				"state", request.State(),
				"attempt", event.Attempt,
			)

		case APITimeoutEvent:
			// fetch the request data
			requestID := envelope.CausationID()
			if requestID == 0 {
				handler.logger.Error("event lacks a causation ID to locate the request")
				break
			}
			request := requests[requestID]
			if request == nil {
				handler.logger.Error("failed to locate request data")
				break
			}

			// A timeout event can only transition the state from "pending" to
			// "timeout". Other states like "failure" or "success" are final.
			if request.attempts[event.Attempt] == state_pending {
				request.attempts[event.Attempt] = state_timeout
			}

			handler.logger.Info(
				"API request timeout elapsed",
				"request ID", request.ID(),
				"state", request.State(),
				"attempt", event.Attempt,
			)
		}
	}

	return handler.store.Error()
}
