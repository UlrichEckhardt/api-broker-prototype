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
	"strconv"
	"time"
)

//Envelope .
type Envelope struct {
	ID      int32              `bson:"_id"`
	Created primitive.DateTime `bson:"created"`
	Payload bson.M             `bson:"payload"`
}

// Notification just carries the ID of a persisted event.
type Notification struct {
	ID int32 `bson:"_id"`
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
// returns the newly inserted event's ID. If it returns zero, the state of
// the event store carries the according error.
func (s *EventStore) Insert(ctx context.Context, env Envelope) int32 {
	// don't do anything if the error state of the store is set already
	if s.err != nil {
		return 0
	}

	// generate an ID
	env.ID = s.findNextID(ctx)
	if env.ID == 0 {
		return 0
	}

	for {
		// set creation date
		env.Created = primitive.NewDateTimeFromTime(time.Now())

		// insert new document
		res, err := s.events.InsertOne(ctx, env)
		if err != nil {
			// Check if the next free ID changed. In that case,
			// just try again with the new ID.
			id := s.findNextID(ctx)
			if id != env.ID {
				env.ID = id
				continue
			}
			s.err = err
			return 0
		}

		// decode the assigned object ID
		id, ok := res.InsertedID.(int32)
		if !ok {
			s.err = errors.New("no ID returned from insert")
			return 0
		}
		if id != env.ID {
			s.err = errors.New("returned ID differs from written ID")
		}
		break
	}

	// insert a notification with the created document's ID
	var note Notification
	note.ID = env.ID
	_, err := s.notifications.InsertOne(ctx, note)
	if err != nil {
		s.err = err
		return 0
	}

	return env.ID
}

// find next free ID to use for an insert
// This returns zero if an error occurs.
func (s *EventStore) findNextID(ctx context.Context) int32 {
	// find the event with the highest ID
	opts := options.FindOne().
		SetBatchSize(1).
		SetProjection(bson.M{"_id": 1}).
		SetSort(bson.M{"_id": -1})
	res := s.events.FindOne(ctx, bson.M{}, opts)
	if res.Err() == mongo.ErrNoDocuments {
		// not an error, the collection is only empty
		return 1
	}
	if res.Err() != nil {
		s.err = res.Err()
		return 0
	}

	// decode and return the envelope's ID increased by one
	var envelope Envelope
	if err := res.Decode(&envelope); err != nil {
		s.err = err
		return 0
	}

	return envelope.ID + 1
}

// RetrieveOne retrieves the event with the given ID.
// When the document is not found, it sets the error state of the store.
func (s *EventStore) RetrieveOne(ctx context.Context, id int32) *Envelope {
	// don't do anything if the error state of the store is set already
	if s.err != nil {
		return nil
	}

	// The ID must be valid.
	if id == 0 {
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
func (s *EventStore) RetrieveNext(ctx context.Context, id int32) *Envelope {
	// don't do anything if the error state of the store is set already
	if s.err != nil {
		return nil
	}

	var filter interface{}
	if id == 0 {
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
func (s *EventStore) LoadEvents(ctx context.Context, start int32) <-chan Envelope {
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
		if start != 0 {
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
			fmt.Println("loaded event", envelope.ID)
			out <- *envelope

			// move to next element
			id = envelope.ID
		}
	}()

	return out
}

// FollowEvents returns a channel that emits all events in turn.
func (s *EventStore) FollowEvents(ctx context.Context, start int32) <-chan Envelope {
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
		if start != 0 {
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
					fmt.Println("received notification", nid)
					continue
				}
			}

			// emit envelope
			fmt.Println("loaded event", envelope.ID)
			out <- *envelope

			// move to next element
			id = envelope.ID
		}
	}()

	return out
}

// FollowNotifications returns a channel that emits all notifications in turn.
func (s *EventStore) FollowNotifications(ctx context.Context) <-chan int32 {
	out := make(chan int32)

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
			fmt.Println("loaded notification", note.ID)
			out <- note.ID
		}
	}()

	return out
}
