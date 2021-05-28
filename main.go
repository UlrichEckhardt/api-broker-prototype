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
	store              events.EventStore
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
				Usage:     "process events from the store",
				ArgsUsage: " ", // no arguments expected
				Flags: []cli.Flag{
					&cli.Float64Flag{
						Name:  "api-failure-rate",
						Value: 0.0,
						Usage: "Fraction of API requests that fail.",
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
				Usage:     "process notifications from the store",
				ArgsUsage: " ", // no arguments expected
				Action: func(c *cli.Context) error {
					if c.NArg() > 0 {
						return errors.New("no arguments expected")
					}

					return watchMain()
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
		c.Float64("api-min-latency"),
		c.Float64("api-max-latency"),
	)
}

func initEventStore() error {
	loglevel, err := log15.LvlFromString(eventStoreLoglevel)
	// setup log handler
	handler := log15.LvlFilterHandler(loglevel, logger.GetHandler())

	// setup logger for event store
	esLogger := log15.New("context", "event store")
	esLogger.SetHandler(handler)

	// create an event store facade
	var s events.EventStore
	switch eventStoreDriver {
	case "mongodb":
		s, err = mongodb.NewEventStore(eventStoreDBHost)
	case "postgresql":
		s, err = postgresql.NewEventStore(eventStoreDBHost)
	default:
		err = errors.New("invalid driver selected")
	}
	if err != nil {
		return err
	}

	// add a logging decorator in front
	s, err = logging.NewLoggingDecorator(s, esLogger)
	if err != nil {
		return err
	}

	store = s

	logger.Info("initialized event store", "host", eventStoreDBHost)

	return nil
}

func finalizeEventStore() {
	if err := store.Close(); err != nil {
		logger.Error("failed to close event store", "error", err)
	}
}

// insert a configuration event
func configureMain(retries int, timeout float64) error {
	if err := initEventStore(); err != nil {
		return err
	}
	defer finalizeEventStore()

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
	if err := initEventStore(); err != nil {
		return err
	}
	defer finalizeEventStore()

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
		event = events.ResponseEvent{Response: data}
	case "failure":
		event = events.FailureEvent{Failure: data}
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
	if err := initEventStore(); err != nil {
		return err
	}
	defer finalizeEventStore()

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
func callAPI(ctx context.Context, store events.EventStore, event events.RequestEvent, causationID int32, attempt uint) {
	// delegate to API stub
	response, err := ProcessRequest(event.Request)

	// store results as event
	if err == nil {
		store.Insert(
			ctx,
			events.ResponseEvent{
				Attempt:  attempt,
				Response: response,
			},
			causationID,
		)
	} else {
		store.Insert(
			ctx,
			events.FailureEvent{
				Attempt: attempt,
				Failure: err.Error(),
			},
			causationID,
		)
	}
}

// process existing elements
func processMain(lastProcessed string) error {
	if err := initEventStore(); err != nil {
		return err
	}
	defer finalizeEventStore()

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

	// map of requests being processed currently
	// This combines the request data and metadata used for processing it.
	type requestState struct {
		request     events.Envelope
		maxAttempts uint
		attempts    uint
	}
	calls := make(map[int32]*requestState)

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
			)

			// store configuration
			retries = uint(event.Retries)

		case events.RequestEvent:
			logger.Info("starting API call")

			// create record to correlate the results with it
			call := &requestState{
				request:     envelope,
				maxAttempts: 1 + retries,
			}
			calls[envelope.ID()] = call

			// try event processing asynchronously
			go callAPI(ctx, store, event, envelope.ID(), call.attempts)
			call.attempts++

		case events.ResponseEvent:
			// fetch the request event
			requestID := envelope.CausationID()
			if requestID == 0 {
				logger.Error("response event lacks a causation ID to locate the request")
				break
			}
			call := calls[requestID]
			if call == nil {
				logger.Error("failed to locate request event")
				break
			}

			delete(calls, requestID)
			logger.Info("completed API call")

		case events.FailureEvent:
			// fetch the request event
			requestID := envelope.CausationID()
			if requestID == 0 {
				logger.Error("failure event lacks a causation ID to locate the request")
				break
			}
			call := calls[requestID]
			if call == nil {
				logger.Error("failed to locate request event")
				break
			}

			// check if any retries remain
			if call.attempts == call.maxAttempts {
				delete(calls, requestID)
				logger.Info("failed API call")
				break
			}

			// retry event processing asynchronously
			go callAPI(ctx, store, call.request.Event().(events.RequestEvent), call.request.ID(), call.attempts)
			call.attempts++
		}
	}

	return store.Error()
}

// watch stream of notifications
func watchMain() error {
	if err := initEventStore(); err != nil {
		return err
	}
	defer finalizeEventStore()

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
