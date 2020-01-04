package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/urfave/cli/v2" // imports as package "cli"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"os"
	"time"
)

//Envelope .
type Envelope struct {
	ID      primitive.ObjectID `bson:"_id,omitempty"`
	Created primitive.DateTime `bson:"created"`
	Payload bson.M             `bson:"payload"`
}

// Notification just carries the ID of a persisted event.
type Notification struct {
	ID primitive.ObjectID `bson:"_id,omitempty"`
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

					var lastProcessed primitive.ObjectID
					if processStartAfter := c.String("start-after"); processStartAfter != "" {
						lp, err := primitive.ObjectIDFromHex(processStartAfter)
						if err != nil {
							return err
						}
						lastProcessed = lp
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

	fmt.Println("inserted new document", id.Hex())
	return nil
}

// list existing elements
func listMain(lastProcessed primitive.ObjectID) error {
	store := NewEventStore()
	if store.Error() != nil {
		return store.Error()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := store.LoadEvents(ctx, lastProcessed)

	// process events from the channel
	for envelope := range ch {
		fmt.Println("received event", envelope.ID.Hex(), envelope.Created.Time().Format(time.RFC3339), envelope.Payload)
	}

	return store.Error()
}

// process existing elements
func processMain(lastProcessed primitive.ObjectID) error {
	store := NewEventStore()
	if store.Error() != nil {
		return store.Error()
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch := store.FollowEvents(ctx, lastProcessed)

	// process events from the channel
	for envelope := range ch {
		fmt.Println("received event", envelope.ID.Hex(), envelope.Created.Time().Format(time.RFC3339), envelope.Payload)
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
		fmt.Println("received notification", oid.Hex())
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
func (s *EventStore) Insert(ctx context.Context, env Envelope) primitive.ObjectID {
	// don't do anything if the error state of the store is set already
	if s.err != nil {
		return primitive.NilObjectID
	}

	// set creation date
	env.Created = primitive.NewDateTimeFromTime(time.Now())

	// insert new document
	res, err := s.events.InsertOne(ctx, env)
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

	// insert a notification with the created document's ID
	var note Notification
	note.ID = id
	_, err = s.notifications.InsertOne(ctx, note)
	if err != nil {
		s.err = err
		return primitive.NilObjectID
	}

	return id
}

// RetrieveOne retrieves the event with the given ID.
// When the document is not found, it sets the error state of the store.
func (s *EventStore) RetrieveOne(ctx context.Context, id primitive.ObjectID) *Envelope {
	// don't do anything if the error state of the store is set already
	if s.err != nil {
		return nil
	}

	// The ID must be valid.
	if id.IsZero() {
		s.err = errors.New("provided document ID is null")
		return nil
	}

	// retrieve the document from the DB
	filter := bson.M{"_id": bson.M{"$eq": id}}
	res := s.events.FindOne(ctx, filter)
	if res.Err() == mongo.ErrNoDocuments {
		s.err = errors.New("document not found")
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

// RetrieveNext retrieves the event following the one with the given ID.
func (s *EventStore) RetrieveNext(ctx context.Context, id primitive.ObjectID) *Envelope {
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
	res := s.events.FindOne(ctx, filter)
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

// LoadEvents returns a channel that emits all events in turn.
func (s *EventStore) LoadEvents(ctx context.Context, start primitive.ObjectID) <-chan Envelope {
	out := make(chan Envelope)

	// run code to retrieve events in a goroutine
	go func() {
		// close channel on finish
		defer close(out)

		// don't do anything if the error state of the store is set already
		if s.Error() != nil {
			return
		}

		// load the referenced start object to verify the ID is valid
		if !start.IsZero() {
			ref := s.RetrieveOne(ctx, start)
			if ref == nil {
				return
			}
		}

		// pump events
		id := start
		for {
			// retrieve next envelope
			envelope := s.RetrieveNext(ctx, id)
			if s.Error() != nil {
				return
			}
			if envelope == nil {
				// Not an error: There are no more documents after "id".
				return
			}

			// emit envelope
			fmt.Println("loaded event", envelope.ID.Hex())
			out <- *envelope

			// move to next element
			id = envelope.ID
		}
	}()

	return out
}

// FollowEvents returns a channel that emits all events in turn.
func (s *EventStore) FollowEvents(ctx context.Context, start primitive.ObjectID) <-chan Envelope {
	out := make(chan Envelope)

	// run code to retrieve events in a goroutine
	go func() {
		// close channel on finish
		defer close(out)

		// don't do anything if the error state of the store is set already
		if s.Error() != nil {
			return
		}

		// load the referenced start object to verify the ID is valid
		if !start.IsZero() {
			ref := s.RetrieveOne(ctx, start)
			if ref == nil {
				return
			}
		}

		// create notification channel
		nch := s.FollowNotifications(ctx)

		// pump events
		id := start
		for {
			// retrieve next envelope
			envelope := s.RetrieveNext(ctx, id)
			if s.Error() != nil {
				return
			}
			if envelope == nil {
				// no more documents after "id"
				// When this happens, we just wait for notifications,
				// which are emitted when new events are queued.
				select {
				case <-ctx.Done():
					fmt.Println("cancelled by context:", ctx.Err())
					return
				case nid := <-nch:
					fmt.Println("received notification", nid.Hex())
					continue
				}
			}

			// emit envelope
			fmt.Println("loaded event", envelope.ID.Hex())
			out <- *envelope

			// move to next element
			id = envelope.ID
		}
	}()

	return out
}

// FollowNotifications returns a channel that emits all notifications in turn.
func (s *EventStore) FollowNotifications(ctx context.Context) <-chan primitive.ObjectID {
	out := make(chan primitive.ObjectID)

	// run code to pump notifications in a goroutine
	go func() {
		// close channel on finish
		defer close(out)

		// don't do anything if the error state of the store is set already
		if s.Error() != nil {
			return
		}

		// create a tailable cursor on the notifications collection
		filter := bson.M{}
		var opts options.FindOptions
		opts.SetCursorType(options.TailableAwait)
		cursor, err := s.notifications.Find(ctx, filter, &opts)
		if err != nil {
			s.err = err
			return
		}
		defer cursor.Close(ctx)

		// pump notifications
		for cursor.Next(ctx) {
			var note Notification
			if err := cursor.Decode(&note); err != nil {
				s.err = err
				return
			}
			fmt.Println("loaded notification", note.ID.Hex())
			out <- note.ID
		}
	}()

	return out
}
