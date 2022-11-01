package main

import (
	"api-broker-prototype/broker"
	"api-broker-prototype/events"
	"api-broker-prototype/logging"
	"api-broker-prototype/mock_api"
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

// configure mock API from optional flags passed on the commandline
func configureAPIStub(c *cli.Context) {
	mock_api.Configure(
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

	event := broker.ConfigurationEvent{
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
		event = broker.RequestEvent{Request: data}
	case "response":
		event = broker.APIResponseEvent{Response: data}
	case "failure":
		event = broker.APIFailureEvent{Failure: data}
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

	return broker.ProcessRequests(ctx, store, logger, lastProcessedID)
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
func watchRequestsMain(lastProcessed string) error {
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

	return broker.WatchRequests(ctx, store, logger, lastProcessedID)
}
