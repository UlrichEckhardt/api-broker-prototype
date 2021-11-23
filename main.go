package main

import (
	"api-broker-prototype/events"
	"api-broker-prototype/logging"
	"api-broker-prototype/mongodb"
	"api-broker-prototype/postgresql"
	"context"
	"errors"
	"github.com/inconshreveable/log15"
	"github.com/urfave/cli/v2" // imports as package "cli"
	"os"
	"time"
)

var (
	eventStoreDriver   string
	eventStoreDBHost   string
	eventStoreLoglevel string
	logger             log15.Logger
)

func main() {
	// setup logger
	logger = log15.New("context", "main")
	logger.SetHandler(log15.StdoutHandler)

	app := cli.App{
		Name:  "api-broker-prototype",
		Usage: "prototype for an event-sourcing inspired API binding",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "eventstore-driver",
				EnvVars:     []string{"EVENTSTORE_DRIVER"},
				Value:       "mongodb",
				Usage:       "Driver for the event store, one of [mongodb, postgresql].",
				Destination: &eventStoreDriver,
			},
			&cli.StringFlag{
				Name:        "eventstore-db-host",
				EnvVars:     []string{"EVENTSTORE_DB_HOST"},
				Value:       "localhost",
				Usage:       "Hostname of the DB server for the event store.",
				Destination: &eventStoreDBHost,
			},
			&cli.StringFlag{
				Name:        "eventstore-loglevel",
				EnvVars:     []string{"EVENTSTORE_LOGLEVEL"},
				Value:       "info",
				Usage:       "Minimum loglevel for event store operations.",
				Destination: &eventStoreLoglevel,
			},
		},
		Commands: []*cli.Command{
			{
				Name:      "configure",
				Usage:     "Insert a configuration event into the store.",
				ArgsUsage: " ",
				Flags: []cli.Flag{
					&cli.IntFlag{
						Name:  "retries",
						Value: -1,
						Usage: "number of times to retry a failed request",
					},
					&cli.Float64Flag{
						Name:  "timeout",
						Value: -1,
						Usage: "maximum duration for a request",
					},
				},
				Action: func(c *cli.Context) error {
					if c.NArg() > 0 {
						return errors.New("no arguments expected")
					}
					return configureMain(c.Int("retries"), c.Float64("timeout"))
				},
			},
			{
				Name:      "insert",
				Usage:     "Insert an event into the store.",
				ArgsUsage: "<event>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "causation",
						Value: "0",
						Usage: "`ID` of the event to register as causation",
					},
				},
				Action: func(c *cli.Context) error {
					args := c.Args()
					if args.Len() != 2 {
						return errors.New("exactly two arguments expected")
					}
					return insertMain(args.Get(0), args.Get(1), c.String("causation"))
				},
			},
			{
				Name:      "list",
				Usage:     "List all events in the store.",
				ArgsUsage: " ", // no arguments expected
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "start-after",
						Value: "",
						Usage: "`ID` of the event after which to start processing",
					},
				},
				Action: func(c *cli.Context) error {
					if c.NArg() > 0 {
						return errors.New("no arguments expected")
					}

					return listMain(c.String("start-after"))
				},
			},
			{
				Name:      "process",
				Usage:     "Process events from the store.",
				ArgsUsage: " ", // no arguments expected
				Flags: []cli.Flag{
					&cli.Float64Flag{
						Name:  "api-failure-rate",
						Value: 0.0,
						Usage: "Fraction of API requests that fail.",
					},
					&cli.Float64Flag{
						Name:  "api-silent-failure-rate",
						Value: 0.0,
						Usage: "Fraction of API requests that don't produce any response.",
					},
					&cli.Float64Flag{
						Name:  "api-min-latency",
						Value: 0.0,
						Usage: "Minimal API latency.",
					},
					&cli.Float64Flag{
						Name:  "api-max-latency",
						Value: 0.0,
						Usage: "Maximal API latency.",
					},
					&cli.StringFlag{
						Name:  "start-after",
						Value: "",
						Usage: "`ID` of the event after which to start processing",
					},
				},
				Action: func(c *cli.Context) error {
					if c.NArg() > 0 {
						return errors.New("no arguments expected")
					}
					configureAPIStub(c)

					return processMain(c.String("start-after"))
				},
			},
			{
				Name:      "watch",
				Usage:     "Watch notifications from the store.",
				ArgsUsage: " ", // no arguments expected
				Action: func(c *cli.Context) error {
					if c.NArg() > 0 {
						return errors.New("no arguments expected")
					}

					return watchNotificationsMain()
				},
			},
			{
				Name:      "watch-requests",
				Usage:     "Watch requests as they are processed.",
				ArgsUsage: " ", // no arguments expected
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "start-after",
						Value: "",
						Usage: "`ID` of the event after which to start watching",
					},
				},
				Action: func(c *cli.Context) error {
					if c.NArg() > 0 {
						return errors.New("no arguments expected")
					}

					return watchRequestsMain(c.String("start-after"))
				},
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		logger.Debug("command exited with error", "error", err)
		return
	}
}

// configure API stub from optional flags passed on the commandline
func configureAPIStub(c *cli.Context) {
	ConfigureStub(
		c.Float64("api-failure-rate"),
		c.Float64("api-silent-failure-rate"),
		c.Float64("api-min-latency"),
		c.Float64("api-max-latency"),
	)
}

func initEventStore() (events.EventStore, error) {
	loglevel, err := log15.LvlFromString(eventStoreLoglevel)
	// setup log handler
	handler := log15.LvlFilterHandler(loglevel, logger.GetHandler())

	// setup logger for event store
	esLogger := log15.New("context", "event store")
	esLogger.SetHandler(handler)

	// create an event store facade
	var store events.EventStore
	switch eventStoreDriver {
	case "mongodb":
		store, err = mongodb.NewEventStore(eventStoreDBHost)
	case "postgresql":
		store, err = postgresql.NewEventStore(eventStoreDBHost)
	default:
		err = errors.New("invalid driver selected")
	}
	if err != nil {
		return nil, err
	}

	// add a logging decorator in front
	store, err = logging.NewLoggingDecorator(store, esLogger)
	if err != nil {
		return nil, err
	}

	logger.Info("initialized event store", "host", eventStoreDBHost)

	return store, nil
}

func finalizeEventStore(store events.EventStore) {
	if err := store.Close(); err != nil {
		logger.Error("failed to close event store", "error", err)
	}
}

// insert a configuration event
func configureMain(retries int, timeout float64) error {
	store, err := initEventStore()
	if err != nil {
		return err
	}
	defer finalizeEventStore(store)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	event := events.ConfigurationEvent{
		Retries: int32(retries),
		Timeout: timeout,
	}

	envelope, err := store.Insert(ctx, event, 0)
	if err != nil {
		return err
	}

	logger.Debug("inserted configuration event", "id", envelope.ID())
	return nil
}

// insert a new event
func insertMain(class string, data string, causation string) error {
	store, err := initEventStore()
	if err != nil {
		return err
	}
	defer finalizeEventStore(store)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create the event from the commandline arguments
	var event events.Event
	switch class {
	case "simple":
		event = events.SimpleEvent{Message: data}
	case "request":
		event = events.RequestEvent{Request: data}
	case "response":
		event = events.APIResponseEvent{Response: data}
	case "failure":
		event = events.APIFailureEvent{Failure: data}
	default:
		return errors.New("unrecognized event class")
	}

	// parse causation ID
	causationID, err := store.ParseEventID(causation)
	if err != nil {
		return err
	}

	// insert a document
	envelope, err := store.Insert(ctx, event, causationID)
	if err != nil {
		return err
	}

	logger.Debug("inserted new document", "id", envelope.ID())
	return nil
}

// list existing elements
func listMain(lastProcessed string) error {
	store, err := initEventStore()
	if err != nil {
		return err
	}
	defer finalizeEventStore(store)

	// parse optional event ID
	var lastProcessedID int32
	if lastProcessed != "" {
		id, err := store.ParseEventID(lastProcessed)
		if err != nil {
			return err
		}
		lastProcessedID = id
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := store.LoadEvents(ctx, lastProcessedID)
	if err != nil {
		return err
	}

	// process events from the channel
	for envelope := range ch {
		logger.Info(
			"event",
			"id", envelope.ID(),
			"class", envelope.Event().Class(),
			"created", envelope.Created().Format(time.RFC3339),
			"causation_id", envelope.CausationID(),
			"data", envelope.Event(),
		)
	}

	return store.Error()
}

// utility function to invoke the API and store the result as event
func startApiCall(ctx context.Context, store events.EventStore, event events.RequestEvent, causationID int32, attempt uint) {
	// emit event that a request was started
	store.Insert(
		ctx,
		events.APIRequestEvent{
			Attempt: attempt,
		},
		causationID,
	)

	// TODO: this accesses `store` asynchronously, which may need synchronization
	go func() {
		// delegate to API stub
		response, err := ProcessRequest(event.Request)

		// store results as event
		if response != nil {
			store.Insert(
				ctx,
				events.APIResponseEvent{
					Attempt:  attempt,
					Response: *response,
				},
				causationID,
			)
		} else if err != nil {
			store.Insert(
				ctx,
				events.APIFailureEvent{
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

// process existing elements
func processMain(lastProcessed string) error {
	store, err := initEventStore()
	if err != nil {
		return err
	}
	defer finalizeEventStore(store)

	// parse optional event ID
	var lastProcessedID int32
	if lastProcessed != "" {
		id, err := store.ParseEventID(lastProcessed)
		if err != nil {
			return err
		}
		lastProcessedID = id
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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
		case events.ConfigurationEvent:
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

		case events.RequestEvent:
			logger.Info("starting request processing")

			// create record to correlate the results with it
			request := newRequestData(envelope, retries)
			requests[envelope.ID()] = request

			// try event processing asynchronously
			startApiCall(ctx, store, event, envelope.ID(), 0)

		case events.APIRequestEvent:
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

		case events.APIResponseEvent:
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

			delete(requests, requestID)
			logger.Info("completed API call")

		case events.APIFailureEvent:
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
			if event.Attempt == request.retries {
				delete(requests, requestID)
				break
			}

			// retry event processing asynchronously
			startApiCall(ctx, store, request.request.Event().(events.RequestEvent), request.request.ID(), event.Attempt+1)
		}
	}

	return store.Error()
}

// watch stream of notifications
func watchNotificationsMain() error {
	store, err := initEventStore()
	if err != nil {
		return err
	}
	defer finalizeEventStore(store)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := store.FollowNotifications(ctx)
	if err != nil {
		return err
	}

	// process notifications from the channel
	for notification := range ch {
		logger.Info("received notification", "id", notification.ID())
	}

	return store.Error()
}

// watch requests as they are processed
func watchRequestsMain(startAfter string) error {
	store, err := initEventStore()
	if err != nil {
		return err
	}
	defer finalizeEventStore(store)

	// parse optional event ID
	var startAfterID int32
	if startAfter != "" {
		id, err := store.ParseEventID(startAfter)
		if err != nil {
			return err
		}
		startAfterID = id
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := store.FollowEvents(ctx, startAfterID)
	if err != nil {
		return err
	}

	// extract summary from request state
	// pending: API is being queried
	// success: response received
	// failure: request failed
	summary := func(state *requestData) requestState {
		// the default value is pending, which means no actual requests have been made
		res := state_pending

		for i, attempt := range state.attempts {
			res = attempt
			if res == state_success {
				// first successful response makes the whole request successful
				break
			}
			if i < state.retries {
				// if there are still some attempts left, the overall result is still pending
				res = state_pending
			}
		}
		return res
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
		case events.ConfigurationEvent:
			if event.Retries >= 0 {
				retries = uint(event.Retries)
			}

		case events.RequestEvent:
			// create record to correlate the results with it
			request := newRequestData(envelope, retries)
			requests[envelope.ID()] = request

			// create record to correlate the results with it
			logger.Info(
				"request received",
				"request ID", envelope.ID(),
				"summary", summary(request),
			)

		case events.APIRequestEvent:
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
				"summary", summary(request),
				"attempt", event.Attempt,
			)

		case events.APIResponseEvent:
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
				"summary", summary(request),
				"attempt", event.Attempt,
			)

		case events.APIFailureEvent:
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
				"summary", summary(request),
				"attempt", event.Attempt,
			)
		}
	}

	return store.Error()
}
