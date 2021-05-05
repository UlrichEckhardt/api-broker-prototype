package main

import (
	"context"
	"errors"
	"github.com/inconshreveable/log15"
	"github.com/urfave/cli/v2" // imports as package "cli"
	"os"
	"time"
)

var (
	eventStoreDBHost string
	logger           log15.Logger
	store            EventStore
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
				Name:    "eventstore-db-host",
				EnvVars: []string{"EVENTSTORE_DB_HOST"},
				Value:   "localhost",
				Usage:   "Hostname of the DB server for the event store.",
				Destination: &eventStoreDBHost,
			},
		},
		Commands: []*cli.Command{
			{
				Name:      "configure",
				Usage:     "Insert a configuration event into the store.",
				ArgsUsage: " ",
				Flags: []cli.Flag{
					&cli.UintFlag{
						Name:  "retries",
						Value: 0,
						Usage: "number of times to retry a failed request",
					},
				},
				Action: func(c *cli.Context) error {
					if c.NArg() > 0 {
						return errors.New("no arguments expected")
					}
					return configureMain(c.Uint("retries"))
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
	// setup log handler
	handler := log15.LvlFilterHandler(log15.LvlInfo, logger.GetHandler())

	// setup logger for event store
	esLogger := log15.New("context", "event store")
	esLogger.SetHandler(handler)

	s := NewEventStore(esLogger, eventStoreDBHost)
	s.RegisterCodec(&configurationEventCodec{})
	s.RegisterCodec(&simpleEventCodec{})
	s.RegisterCodec(&requestEventCodec{})
	s.RegisterCodec(&responseEventCodec{})
	s.RegisterCodec(&failureEventCodec{})
	if e := s.Error(); e != nil {
		return e
	}
	store = s

	logger.Info("initialized event store", "host", eventStoreDBHost)

	return nil
}

// insert a configuration event
func configureMain(retries uint) error {
	if err := initEventStore(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	event := configurationEvent{
		retries: int32(retries),
	}

	envelope := store.Insert(ctx, event, 0)
	if store.Error() != nil {
		return store.Error()
	}

	logger.Debug("inserted configuration event", "id", envelope.ID())
	return nil
}

// insert a new event
func insertMain(class string, data string, causation string) error {
	if err := initEventStore(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// create the event from the commandline arguments
	var event Event
	switch class {
	case "simple":
		event = simpleEvent{message: data}
	case "request":
		event = requestEvent{request: data}
	case "response":
		event = responseEvent{response: data}
	case "failure":
		event = failureEvent{failure: data}
	default:
		return errors.New("unrecognized event class")
	}

	// parse causation ID
	causationID, err := store.ParseEventID(causation)
	if err != nil {
		return err
	}

	// insert a document
	envelope := store.Insert(ctx, event, causationID)
	if store.Error() != nil {
		return store.Error()
	}

	logger.Debug("inserted new document", "id", envelope.ID())
	return nil
}

// list existing elements
func listMain(lastProcessed string) error {
	if err := initEventStore(); err != nil {
		return err
	}

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

	ch := store.LoadEvents(ctx, lastProcessedID)

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
func callAPI(ctx context.Context, store EventStore, event requestEvent, causationID int32) {
	// delegate to API stub
	response, err := ProcessRequest(event.request)

	// store results as event
	if err == nil {
		store.Insert(ctx, responseEvent{response: response}, causationID)
	} else {
		store.Insert(ctx, failureEvent{failure: err.Error()}, causationID)
	}
}

// process existing elements
func processMain(lastProcessed string) error {
	if err := initEventStore(); err != nil {
		return err
	}

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

	ch := store.FollowEvents(ctx, lastProcessedID)

	// number of retries after a failed request
	retries := uint(0)

	// map of requests being processed currently
	type requestState struct {
		request Envelope
		retries uint // number of retries left
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
		case configurationEvent:
			logger.Info(
				"updating API configuration",
				"retries", event.retries,
			)

			// store configuration
			retries = uint(event.retries)

		case requestEvent:
			logger.Info("starting API call")

			// create record to correlate the response with it
			calls[envelope.ID()] = &requestState{
				request: envelope,
				retries: retries,
			}

			// trigger event processing asynchronously
			go callAPI(ctx, store, event, envelope.ID())

		case responseEvent:
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

		case failureEvent:
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
			if call.retries == 0 {
				delete(calls, requestID)
				logger.Info("failed API call")
				break
			}

			// decrement retry counter and try again asynchronously
			call.retries--
			go callAPI(ctx, store, call.request.Event().(requestEvent), call.request.ID())
		}
	}

	return store.Error()
}

// watch stream of notifications
func watchMain() error {
	if err := initEventStore(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := store.FollowNotifications(ctx)

	// process notifications from the channel
	for notification := range ch {
		logger.Info("received notification", "id", notification.ID())
	}

	return store.Error()
}
