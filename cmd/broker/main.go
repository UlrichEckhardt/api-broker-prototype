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
	"os"
	"os/signal"
	"time"

	"github.com/gofrs/uuid"
	"github.com/inconshreveable/log15"
	"github.com/urfave/cli/v2" // imports as package "cli"
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

	// setup CLI handler
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
					&cli.StringFlag{
						Name:  "external-uuid",
						Value: "",
						Usage: "external identifier for the request",
					},
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
					externalUUID, err := parseUUID(c.String("external-uuid"))
					if err != nil {
						return err
					}
					return configureMain(c.Context, externalUUID, c.Int("retries"), c.Float64("timeout"))
				},
			},
			{
				Name:      "insert",
				Usage:     "Insert an event into the store.",
				ArgsUsage: "<event>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "external-uuid",
						Value: "",
						Usage: "external identifier for the request",
					},
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
					externalUUID, err := parseUUID(c.String("external-uuid"))
					if err != nil {
						return err
					}
					return insertMain(c.Context, args.Get(0), args.Get(1), externalUUID, c.String("causation"))
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

					return listMain(c.Context, c.String("start-after"))
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

					return processMain(c.Context, c.String("start-after"))
				},
			},
			{
				Name:      "resolve-external-uuid",
				Usage:     "resolve a UUID to the according internal ID",
				ArgsUsage: "<external UUID>",
				Action: func(c *cli.Context) error {
					args := c.Args()
					if args.Len() != 1 {
						return errors.New("exactly one argument expected")
					}
					externalUUID, err := parseUUID(args.Get(0))
					if err != nil {
						return err
					}
					return resolveExternalUUIDMain(c.Context, externalUUID)
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

					return watchNotificationsMain(c.Context)
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

					return watchRequestsMain(c.Context, c.String("start-after"))
				},
			},
		},
	}

	// setup a context for coordinated shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	// run the actual commandline application
	if err := app.RunContext(ctx, os.Args); err != nil {
		logger.Debug("command exited with error", "error", err)
		return
	}
}

// parse a string to a UUID
// Only a malformed string creates an error, an empty one will return a
// nil UUID value.
func parseUUID(arg string) (uuid.UUID, error) {
	if arg == "" {
		return uuid.Nil, nil
	}

	return uuid.FromString(arg)
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
	// setup log handler
	loglevel, err := log15.LvlFromString(eventStoreLoglevel)
	if err != nil {
		return nil, err
	}
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
func configureMain(ctx context.Context, externalUUID uuid.UUID, retries int, timeout float64) error {
	store, err := initEventStore()
	if err != nil {
		return err
	}
	defer finalizeEventStore(store)

	event := broker.ConfigurationEvent{
		Retries: int32(retries),
		Timeout: timeout,
	}

	envelope, err := store.Insert(ctx, externalUUID, event, 0)
	if err != nil {
		return err
	}

	logger.Debug("inserted configuration event", "id", envelope.ID())
	return nil
}

// insert a new event
func insertMain(ctx context.Context, class string, data string, externalUUID uuid.UUID, causation string) error {
	store, err := initEventStore()
	if err != nil {
		return err
	}
	defer finalizeEventStore(store)

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
	envelope, err := store.Insert(ctx, externalUUID, event, causationID)
	if err != nil {
		return err
	}

	logger.Debug("inserted new document", "id", envelope.ID())
	return nil
}

// list existing elements
func listMain(ctx context.Context, lastProcessed string) error {
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

	ch, err := store.LoadEvents(ctx, lastProcessedID)
	if err != nil {
		return err
	}

	// process events from the channel
	for envelope := range ch {
		logger.Info(
			"event",
			"id", envelope.ID(),
			"external_uuid", envelope.ExternalUUID(),
			"class", envelope.Event().Class(),
			"created", envelope.Created().Format(time.RFC3339),
			"causation_id", envelope.CausationID(),
			"data", envelope.Event(),
		)
	}

	return store.Error()
}

// process existing elements
func processMain(ctx context.Context, startAfter string) error {
	store, err := initEventStore()
	if err != nil {
		return err
	}
	defer finalizeEventStore(store)

	handler, err := broker.NewRequestProcessor(store, logger)
	if err != nil {
		return err
	}

	// parse optional event ID
	var startAfterID int32
	if startAfter != "" {
		id, err := store.ParseEventID(startAfter)
		if err != nil {
			return err
		}
		startAfterID = id
	}

	return handler.Run(ctx, startAfterID)
}

// resolve an event's external UUID to the according internal ID
func resolveExternalUUIDMain(ctx context.Context, externalUUID uuid.UUID) error {
	store, err := initEventStore()
	if err != nil {
		return err
	}
	defer finalizeEventStore(store)

	id, err := store.ResolveUUID(ctx, externalUUID)
	if err != nil {
		return err
	}

	logger.Info("resolved UUID", "id", id)

	return store.Error()
}

// watch stream of notifications
func watchNotificationsMain(ctx context.Context) error {
	store, err := initEventStore()
	if err != nil {
		return err
	}
	defer finalizeEventStore(store)

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
func watchRequestsMain(ctx context.Context, startAfter string) error {
	store, err := initEventStore()
	if err != nil {
		return err
	}
	defer finalizeEventStore(store)

	handler, err := broker.NewRequestWatcher(store, logger)
	if err != nil {
		return err
	}

	// parse optional event ID
	var startAfterID int32
	if startAfter != "" {
		id, err := store.ParseEventID(startAfter)
		if err != nil {
			return err
		}
		startAfterID = id
	}

	return handler.Run(ctx, startAfterID)
}
