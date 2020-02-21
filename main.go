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
	logger log15.Logger
	store  EventStore
)

func main() {
	app := cli.App{
		Name:  "api-broker-prototype",
		Usage: "prototype for an event-sourcing inspired API binding",
		Commands: []*cli.Command{
			{
				Name:      "insert",
				Usage:     "Insert an event into the store.",
				ArgsUsage: "<event>",
				Action: func(c *cli.Context) error {
					args := c.Args()
					if args.Len() != 1 {
						return errors.New("exactly one argument expected")
					}
					return insertMain(args.First())
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

func initEventStore() error {
	logger = log15.New("context", "event-store")
	logger.SetHandler(log15.StdoutHandler)

	s := NewEventStore(logger)
	s.RegisterCodec(&simpleEventCodec{})
	if e := s.Error(); e != nil {
		return e
	}
	store = s
	return nil
}

// insert a new event
func insertMain(message string) error {
	if err := initEventStore(); err != nil {
		return err
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// insert a document
	event := simpleEvent{message: message}
	envelope := store.Insert(ctx, &event)
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
		logger.Info("received event", "id", envelope.ID(), "created", envelope.Created().Format(time.RFC3339), "event", envelope.Event())
	}

	return store.Error()
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

	// process events from the channel
	for envelope := range ch {
		logger.Info("received event", "id", envelope.ID(), "created", envelope.Created().Format(time.RFC3339), "event", envelope.Event())
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
