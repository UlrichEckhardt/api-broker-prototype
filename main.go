package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/urfave/cli/v2" // imports as package "cli"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"os"
	"time"
)

//Envelope .
type Envelope struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	Created primitive.DateTime `bson:"created"`
	Payload bson.M             `bson:"payload"`
}

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

					var lastProcessed primitive.ObjectID
					if processStartAfter := c.String("start-after"); processStartAfter != "" {
						lp, err := primitive.ObjectIDFromHex(processStartAfter)
						if err != nil {
							return err
						}
						lastProcessed = lp
					}
					return processMain(lastProcessed)
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
	events, notifications, err := Connect()
	if err != nil {
		return err
	}

	// insert an additional document
	id, err := insertNewDocument(bson.M{"event": event}, events, notifications)
	if err != nil {
		return err
	}
	fmt.Println("inserted new document", id.Hex())
	return nil
}

// process existing elements
func processMain(lastProcessed primitive.ObjectID) error {
	events, notifications, err := Connect()
	if err != nil {
		return err
	}

	// iterate over existing events
	for {
		envelope, err := getNextDocument(lastProcessed, events, notifications)
		if err != nil {
			return err
		}
		if envelope == nil {
			break
		}
		fmt.Println("loaded envelope", envelope.ID.Hex())
		fmt.Println("created", envelope.Created.Time().Format(time.RFC3339))
		fmt.Println("payload", envelope.Payload)

		// move to next element
		lastProcessed = envelope.ID
	}

	return nil
}

func insertNewDocument(payload bson.M, events *mongo.Collection, notifications *mongo.Collection) (primitive.ObjectID, error) {
	var doc Envelope
	doc.Created = primitive.NewDateTimeFromTime(time.Now())
	doc.Payload = payload

	res, err := events.InsertOne(context.Background(), doc)
	if err != nil {
		return primitive.NilObjectID, err
	}
	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		return primitive.NilObjectID, errors.New("no ID returned from insert")
	}

	return id, nil
}

func getNextDocument(lastProcessed primitive.ObjectID, events *mongo.Collection, notifications *mongo.Collection) (*Envelope, error) {
	var filter interface{}
	if lastProcessed.IsZero() {
		filter = bson.M{}
	} else {
		filter = bson.M{"_id": bson.M{"$gt": lastProcessed}}
	}

	res := events.FindOne(nil, filter)
	if res.Err() == mongo.ErrNoDocuments {
		// not an error, there are no more documents left
		return nil, nil
	}
	if res.Err() != nil {
		return nil, res.Err()
	}

	var envelope Envelope
	if err := res.Decode(&envelope); err != nil {
		return nil, err
	}

	return &envelope, nil
}
