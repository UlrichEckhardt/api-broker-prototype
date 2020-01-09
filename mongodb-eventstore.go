package main

// Implementation of the EventStore interface on top of a MongoDB.
// The implementation uses two collections. The "events" collection just serves
// as a store for the event data (envelope). The "notifications" collection is
// a bit more elaborated. It is used to build a queue that is used to wake up a
// process waiting for new events. For that, the collection is capped, i.e. has
// a maximum size. This is necessary in order to allow creation of a tailable
// cursor, which is fundamental for the required blocking behaviour.

import (
	"context"
	"errors"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"strconv"
	"time"
)

// mongoDBEventStore implements the EventStore interface using a MongoDB.
type mongoDBEventStore struct {
	events        *mongo.Collection
	notifications *mongo.Collection
	err           error
}

// NewEventStore creates and connects a mongoDBEventStore instance.
func NewEventStore() EventStore {
	var s mongoDBEventStore
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

// ParseEventID implements the EventStore interface.
func (s *mongoDBEventStore) ParseEventID(str string) (int32, error) {
	lp, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(lp), nil
}

// Error implements the EventStore interface.
func (s *mongoDBEventStore) Error() error {
	return s.err
}

// Insert implements the EventStore interface.
func (s *mongoDBEventStore) Insert(ctx context.Context, env Envelope) int32 {
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
// This returns zero and sets the error state if an error occurs.
func (s *mongoDBEventStore) findNextID(ctx context.Context) int32 {
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

// RetrieveOne implements the EventStore interface.
func (s *mongoDBEventStore) RetrieveOne(ctx context.Context, id int32) *Envelope {
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

// retrieveNext retrieves the event following the one with the given ID.
func (s *mongoDBEventStore) retrieveNext(ctx context.Context, id int32) *Envelope {
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

// LoadEvents implements the EventStore interface.
func (s *mongoDBEventStore) LoadEvents(ctx context.Context, start int32) <-chan Envelope {
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
			envelope := s.retrieveNext(ctx, id)
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

// FollowNotifications implements the EventStore interface.
func (s *mongoDBEventStore) FollowNotifications(ctx context.Context) <-chan int32 {
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

// FollowEvents implements the EventStore interface.
func (s *mongoDBEventStore) FollowEvents(ctx context.Context, start int32) <-chan Envelope {
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
			envelope := s.retrieveNext(ctx, id)
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
