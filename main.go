package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/urfave/cli/v2" // imports as package "cli"
	"go.mongodb.org/mongo-driver/bson"
	"os"
	"strconv"
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

					var lastProcessed int32
					if processStartAfter := c.String("start-after"); processStartAfter != "" {
						lp, err := strconv.ParseInt(processStartAfter, 10, 32)
						if err != nil {
							return err
						}
						lastProcessed = int32(lp)
					}
					return listMain(lastProcessed)
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

					var lastProcessed int32
					if processStartAfter := c.String("start-after"); processStartAfter != "" {
						lp, err := strconv.ParseInt(processStartAfter, 10, 32)
						if err != nil {
							return err
						}
						lastProcessed = int32(lp)
					}
					return processMain(lastProcessed)
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

	// insert an additional document
	var doc Envelope
	doc.Payload = bson.M{"event": event}
	id := store.Insert(ctx, doc)
	if store.Error() != nil {
		return store.Error()
	}

	fmt.Println("inserted new document", id)
	return nil
}

// list existing elements
func listMain(lastProcessed int32) error {
	store := NewEventStore()
	if store.Error() != nil {
		return store.Error()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := store.LoadEvents(ctx, lastProcessed)

	// process events from the channel
	for envelope := range ch {
		fmt.Println("received event", envelope.ID, envelope.Created.Time().Format(time.RFC3339), envelope.Payload)
	}

	return store.Error()
}

// process existing elements
func processMain(lastProcessed int32) error {
	store := NewEventStore()
	if store.Error() != nil {
		return store.Error()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := store.FollowEvents(ctx, lastProcessed)

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
