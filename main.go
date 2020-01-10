package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/urfave/cli/v2" // imports as package "cli"
	"go.mongodb.org/mongo-driver/bson"
	"os"
	"time"
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
		fmt.Println(err)
		return
	}
}

// insert a new event
func insertMain(event string) error {
	store := NewEventStore()
	if store.Error() != nil {
		return store.Error()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// insert a document
	payload := bson.M{"event": event}
	envelope := store.Insert(ctx, payload)
	if store.Error() != nil {
		return store.Error()
	}

	fmt.Println("inserted new document", envelope.ID)
	return nil
}

// list existing elements
func listMain(lastProcessed string) error {
	store := NewEventStore()
	if store.Error() != nil {
		return store.Error()
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
		fmt.Println("received event", envelope.ID, envelope.Created.Time().Format(time.RFC3339), envelope.Payload)
	}

	return store.Error()
}

// process existing elements
func processMain(lastProcessed string) error {
	store := NewEventStore()
	if store.Error() != nil {
		return store.Error()
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
		fmt.Println("received event", envelope.ID, envelope.Created.Time().Format(time.RFC3339), envelope.Payload)
	}

	return store.Error()
}

// watch stream of notifications
func watchMain() error {
	store := NewEventStore()
	if store.Error() != nil {
		return store.Error()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := store.FollowNotifications(ctx)

	// process notifications from the channel
	for oid := range ch {
		fmt.Println("received notification", oid)
	}

	return store.Error()
}
