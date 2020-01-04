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
	store := NewEventStore()
	if store.Error() != nil {
		return store.Error()
	}

	// insert an additional document
	var doc Envelope
	doc.Payload = bson.M{"event": event}
	id := store.Insert(doc)
	if store.Error() != nil {
		return store.Error()
	}

	fmt.Println("inserted new document", id.Hex())
	return nil
}

// process existing elements
func processMain(lastProcessed primitive.ObjectID) error {
	store := NewEventStore()
	if store.Error() != nil {
		return store.Error()
	}

	ch := make(chan Envelope)

	// run code to retrieve events in a goroutine
	go func(id primitive.ObjectID, sink chan<- Envelope) {
		defer close(sink)
		for {
			// retrieve next envelope
			envelope := store.RetrieveNext(id)
			if store.Error() != nil {
				return
			}
			if envelope == nil {
				break
			}

			// emit envelope
			sink <- *envelope

			// move to next element
			id = envelope.ID
		}
	}(lastProcessed, ch)

	// process events from the channel
	for envelope := range ch {
		fmt.Println("received event", envelope.ID.Hex(), envelope.Created.Time().Format(time.RFC3339), envelope.Payload)
	}

	return store.Error()
}

// EventStore represents the storage and retrieval of events. The content is
// represented using The envelope type above.
type EventStore struct {
	events        *mongo.Collection
	notifications *mongo.Collection
	err           error
}

// NewEventStore connects an EventStore instance. In case of errors, the event
// store is nonfunctional and the according error state is set.
func NewEventStore() *EventStore {
	var s EventStore
	events, notifications, err := Connect()
	if err == nil {
		// initialize collections
		s.events = events
		s.notifications = notifications
	} else {
		// set error state
		s.err = err
	}
	return &s
}

// Retrieve the error state of the event store.
func (s *EventStore) Error() error {
	return s.err
}

// Insert a new event (wrapped in the envelope) into the event store. It
// returns the newly inserted event's ID. If it returns nil, the state of
// the event store carries the according error.
func (s *EventStore) Insert(env Envelope) primitive.ObjectID {
	// don't do anything if the error state of the store is set already
	if s.err != nil {
		return primitive.NilObjectID
	}

	// set creation date
	env.Created = primitive.NewDateTimeFromTime(time.Now())

	// insert new document
	res, err := s.events.InsertOne(context.Background(), env)
	if err != nil {
		s.err = err
		return primitive.NilObjectID
	}

	// decode and return the assigned object ID
	id, ok := res.InsertedID.(primitive.ObjectID)
	if !ok {
		s.err = errors.New("no ID returned from insert")
		return primitive.NilObjectID
	}

	return id
}

// RetrieveNext retrieves the event following the one with the given ID.
func (s *EventStore) RetrieveNext(id primitive.ObjectID) *Envelope {
	// don't do anything if the error state of the store is set already
	if s.err != nil {
		return nil
	}

	var filter interface{}
	if id.IsZero() {
		filter = bson.M{}
	} else {
		filter = bson.M{"_id": bson.M{"$gt": id}}
	}

	// retrieve the actual document from the DB
	res := s.events.FindOne(nil, filter)
	if res.Err() == mongo.ErrNoDocuments {
		// not an error, there are no more documents left
		return nil
	}
	if res.Err() != nil {
		s.err = res.Err()
		return nil
	}

	// decode and return the envelope
	var envelope Envelope
	if err := res.Decode(&envelope); err != nil {
		s.err = err
		return nil
	}

	return &envelope
}
