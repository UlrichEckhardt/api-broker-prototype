package mongodb

// Implementation of the EventStore interface on top of a MongoDB.
// The implementation uses two collections. The "events" collection just serves
// as a store for the event data (envelope). The "notifications" collection is
// a bit more elaborated. It is used to build a queue that is used to wake up a
// process waiting for new events. For that, the collection is capped, i.e. has
// a maximum size. This is necessary in order to allow creation of a tailable
// cursor, which is fundamental for the required blocking behaviour.

import (
	"api-broker-prototype/events"
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/bsoncodec"
	"go.mongodb.org/mongo-driver/bson/bsonrw"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Common DB settings.
const (
	AppName                    = "api-broker-prototype" // Name registered at MongoDB for better log readability.
	DBName                     = "test"                 // Name of the DB.
	EventCollectionName        = "events"               // Name of the collection with actual events and payload.
	NotificationCollectionName = "notifications"        // Name of the capped collection with notifications.
)

// The MongoDBEventCodec interface defines methods common to event codecs.
// The codecs convert between the internal representation (Event) and the
// general-purpose representation for MongoDB (bson.M).
// See also the Event interface, which it is closely related to.
type MongoDBEventCodec interface {
	// Class returns a string that identifies the event type this codec handles.
	Class() string
	// Serialize the event in a way that allows writing it to a MongoDB.
	Serialize(event events.Event) (bson.M, error)
	// Deserialize an event from data from a MongoDB.
	Deserialize(data bson.M) (events.Event, error)
}

// mongoDBRawEnvelope is the type representing the envelope in MongoDB
type mongoDBRawEnvelope struct {
	ID              int32              `bson:"_id"`
	ExternalUUIDVal *uuid.UUID         `bson:"external_uuid"`
	Created         primitive.DateTime `bson:"created"`
	CausationID     int32              `bson:"causation_id"`
	Class           string             `bson:"class"`
	Data            bson.M             `bson:"data"`
}

// mongoDBEnvelope implements the Envelope interface.
type mongoDBEnvelope struct {
	IDVal           int32
	ExternalUUIDVal uuid.UUID
	CreatedVal      primitive.DateTime
	CausationIDVal  int32
	EventVal        events.Event
}

// ID implements the Envelope interface.
func (env *mongoDBEnvelope) ID() int32 {
	return env.IDVal
}

// Created implements the Envelope interface.
func (env *mongoDBEnvelope) Created() time.Time {
	return env.CreatedVal.Time()
}

// ExternalUUID implements the Envelope interface.
func (env *mongoDBEnvelope) ExternalUUID() uuid.UUID {
	return env.ExternalUUIDVal
}

// CausationID implements the Envelope interface.
func (env *mongoDBEnvelope) CausationID() int32 {
	return env.CausationIDVal
}

// Event implements the Envelope interface.
func (env *mongoDBEnvelope) Event() events.Event {
	return env.EventVal
}

// mongoDBNotification implements the Notification interface.
type mongoDBNotification struct {
	IDVal int32 `bson:"_id"`
}

// ID implements the Notification interface.
func (note *mongoDBNotification) ID() int32 {
	return note.IDVal
}

// MongoDBEventStore implements the EventStore interface using a MongoDB.
type MongoDBEventStore struct {
	host          string
	events        *mongo.Collection
	notifications *mongo.Collection
	err           error
	codecs        map[string]MongoDBEventCodec
}

func buildTypeRegistry() *bsoncodec.Registry {
	UUIDType := reflect.TypeOf(uuid.UUID{})

	res := bson.NewRegistry()
	res.RegisterTypeEncoder(
		UUIDType,
		bsoncodec.ValueEncoderFunc(
			func(ec bsoncodec.EncodeContext, writer bsonrw.ValueWriter, value reflect.Value) error {
				if value.Type() != UUIDType {
					return bsoncodec.ValueEncoderError{Name: "UUIDCodec", Types: []reflect.Type{UUIDType}, Received: value}
				}
				uuid := value.Interface().(uuid.UUID)
				return writer.WriteBinaryWithSubtype(uuid.Bytes(), bson.TypeBinaryUUID)
			},
		),
	)
	res.RegisterTypeDecoder(
		UUIDType,
		bsoncodec.ValueDecoderFunc(
			func(ec bsoncodec.DecodeContext, reader bsonrw.ValueReader, value reflect.Value) error {
				if !value.CanSet() || value.Type() != UUIDType {
					return bsoncodec.ValueDecoderError{Name: "UUIDCodec", Types: []reflect.Type{UUIDType}, Received: value}
				}

				switch value_type := reader.Type(); value_type {
				case bson.TypeBinary:
					data, subtype, err := reader.ReadBinary()
					if err != nil {
						return err
					}
					if subtype != bson.TypeBinaryUUID {
						return fmt.Errorf("unsupported binary subtype %v for UUID", subtype)
					}
					uuid, err := uuid.FromBytes(data)
					if err != nil {
						return err
					}
					value.Set(reflect.ValueOf(uuid))
					return nil
				case bson.TypeNull:
					return reader.ReadNull()
				case bson.TypeUndefined:
					return reader.ReadUndefined()
				default:
					return fmt.Errorf("cannot decode %v into a UUID", value_type)
				}
			},
		),
	)

	return res
}

// connect establishes an on-demand connection to the DB
func (s *MongoDBEventStore) connect(ctx context.Context) {
	// do nothing if state is already establish
	if s.events != nil || s.err != nil {
		return
	}

	opts := options.
		Client().
		SetRegistry(buildTypeRegistry()).
		ApplyURI("mongodb://" + s.host).
		SetAppName(AppName).SetConnectTimeout(1 * time.Second)
	if err := opts.Validate(); err != nil {
		s.err = err
		return
	}

	client, err := mongo.Connect(ctx, opts)
	if err != nil {
		s.err = err
		return
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		s.err = err
		return
	}

	db := client.Database(DBName)
	s.events = db.Collection(EventCollectionName)
	s.notifications = db.Collection(NotificationCollectionName)
}

// NewEventStore creates and connects a MongoDBEventStore instance.
func NewEventStore(host string) (*MongoDBEventStore, error) {
	s := MongoDBEventStore{
		host:   host,
		codecs: make(map[string]MongoDBEventCodec),
	}

	// register codecs
	s.registerCodec(&configurationEventCodec{})
	s.registerCodec(&simpleEventCodec{})
	s.registerCodec(&requestEventCodec{})
	s.registerCodec(&apiRequestEventCodec{})
	s.registerCodec(&apiResponseEventCodec{})
	s.registerCodec(&apiFailureEventCodec{})
	s.registerCodec(&apiTimeoutEventCodec{})

	return &s, nil
}

// registerCodec registers a codec that allows conversion of Events.
func (s *MongoDBEventStore) registerCodec(codec MongoDBEventCodec) {
	if codec == nil {
		s.err = errors.New("nil codec registered")
		return
	}
	s.codecs[codec.Class()] = codec
}

// ParseEventID implements the EventStore interface.
func (s *MongoDBEventStore) ParseEventID(str string) (int32, error) {
	lp, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(lp), nil
}

// Error implements the EventStore interface.
func (s *MongoDBEventStore) Error() error {
	return s.err
}

// Close implements the EventStore and io.Closer interfaces.
func (s *MongoDBEventStore) Close() error {
	// don't do anything if the error state of the store is set already
	if s.err != nil {
		return nil
	}

	// reset fields so the GC can reclaim them
	s.events = nil
	s.notifications = nil

	// set this error to block any further calls
	s.err = errors.New("eventstore is closed")

	return nil
}

// Insert implements the EventStore interface.
func (s *MongoDBEventStore) Insert(ctx context.Context, externalUUID uuid.UUID, event events.Event, causationID int32) (events.Envelope, error) {
	// don't do anything if the error state of the store is set already
	s.connect(ctx)
	if s.err != nil {
		return nil, s.err
	}

	//  locate codec for the event class
	class := event.Class()
	codec := s.codecs[class]
	if codec == nil {
		return nil, errors.New("failed to locate codec for event")
	}

	// encode event for MongoDB storage
	payload, err := codec.Serialize(event)
	if err != nil {
		return nil, err
	}

	env := mongoDBRawEnvelope{
		ExternalUUIDVal: uuidAsDBValue(externalUUID),
		CausationID:     causationID,
		Class:           class,
		Data:            payload,
	}

	// generate an ID
	env.ID = s.findNextID(ctx)
	if env.ID == 0 {
		return nil, s.err
	}

	for {
		// set creation date
		env.Created = primitive.NewDateTimeFromTime(time.Now())

		// insert new document
		res, err := s.events.InsertOne(ctx, env)
		if err != nil {
			var write_exc mongo.WriteException
			if !errors.As(err, &write_exc) {
				return nil, err
			}

			// E11000 seems to be MongoDB's "duplicate key error", a unique
			// constraint violation. We now only need to distinguish by the
			// index that prevented the insert, the ID or the external UUID.
			if write_exc.HasErrorCodeWithMessage(11000, "index: unique_external_uuid_constraint") {
				return nil, events.DuplicateEventUUID
			}
			if write_exc.HasErrorCodeWithMessage(11000, "index: _id_") {
				// The ID is already used, just generate a new one.
				id := s.findNextID(ctx)
				if id == 0 {
					return nil, s.err
				}
				if id != env.ID {
					// retrying with the new ID
					env.ID = id
					continue
				}
			}

			return nil, err
		}

		// decode the assigned object ID
		id, ok := res.InsertedID.(int32)
		if !ok {
			s.err = errors.New("no ID returned from insert")
			return nil, s.err
		}
		if id != env.ID {
			s.err = errors.New("returned ID differs from written ID")
			return nil, s.err
		}
		break
	}

	// insert a notification with the created document's ID
	var note mongoDBNotification
	note.IDVal = env.ID
	_, err = s.notifications.InsertOne(ctx, note)
	if err != nil {
		s.err = err
		return nil, s.err
	}

	res := &mongoDBEnvelope{
		IDVal:           env.ID,
		ExternalUUIDVal: dbValueAsUUID(env.ExternalUUIDVal),
		CreatedVal:      env.Created,
		CausationIDVal:  env.CausationID,
		EventVal:        event,
	}
	return res, nil
}

// convert UUID to a parameter for the DB
// We write nil UUID as `null`, so that the index ignores the value.
// All internal events have a nil external UUID, because they don't need
// idempotent insert operations, but we also don't want those to be flagged
// as duplicate values.
func uuidAsDBValue(val uuid.UUID) *uuid.UUID {
	if val == uuid.Nil {
		return nil
	}

	return &val
}

// convert response from the DB to UUID
// See `uuidAsDBValue()`.
func dbValueAsUUID(val *uuid.UUID) uuid.UUID {
	if val == nil {
		return uuid.Nil
	}

	return *val
}

// ResolveUUID implements the EventStore interface.
func (s *MongoDBEventStore) ResolveUUID(ctx context.Context, externalUUID uuid.UUID) (int32, error) {
	if externalUUID == uuid.Nil {
		return 0, errors.New("provided external UUID is null")
	}

	// don't do anything if the error state of the store is set already
	s.connect(ctx)
	if s.err != nil {
		return 0, s.err
	}

	// retrieve the document from the DB
	filter := bson.M{"external_uuid": bson.M{"$eq": externalUUID}}
	res := s.events.FindOne(ctx, filter)
	if res.Err() == mongo.ErrNoDocuments {
		return 0, errors.New("document not found")
	}
	if res.Err() != nil {
		s.err = res.Err()
		return 0, s.err
	}

	// decode envelope from the document
	envelope := s.decodeEnvelope(res)
	if envelope == nil {
		return 0, s.err
	}

	return envelope.ID(), nil
}

// find next free ID to use for an insert
// This returns zero and sets the error state if an error occurs.
func (s *MongoDBEventStore) findNextID(ctx context.Context) int32 {
	// find the event with the highest ID
	opts := options.FindOne().
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
	var envelope mongoDBRawEnvelope
	if err := res.Decode(&envelope); err != nil {
		s.err = err
		return 0
	}

	return envelope.ID + 1
}

// RetrieveOne implements the EventStore interface.
func (s *MongoDBEventStore) RetrieveOne(ctx context.Context, id int32) (events.Envelope, error) {
	// don't do anything if the error state of the store is set already
	s.connect(ctx)
	if s.err != nil {
		return nil, s.err
	}

	// The ID must be valid.
	if id == 0 {
		return nil, errors.New("provided document ID is null")
	}

	// retrieve the document from the DB
	filter := bson.M{"_id": bson.M{"$eq": id}}
	res := s.events.FindOne(ctx, filter)
	if res.Err() == mongo.ErrNoDocuments {
		return nil, errors.New("document not found")
	}
	if res.Err() != nil {
		s.err = res.Err()
		return nil, s.err
	}

	return s.decodeEnvelope(res), s.err
}

// retrieveNext retrieves the event following the one with the given ID.
// This will return the decoded envelope or nil if there is no next event. In
// case of failure, it sets the error state.
func (s *MongoDBEventStore) retrieveNext(ctx context.Context, id int32) *mongoDBEnvelope {
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

	return s.decodeEnvelope(res)
}

// decode the envelope from a MongoDB lookup
// This will return the decoded envelope. In case of failure, it sets the error
// state. Errors here are non-recoverable, because they are caused by a mismatch
// between the actual and expected DB structure.
func (s *MongoDBEventStore) decodeEnvelope(raw *mongo.SingleResult) *mongoDBEnvelope {
	var envelope mongoDBRawEnvelope
	if err := raw.Decode(&envelope); err != nil {
		s.err = err
		return nil
	}

	codec := s.codecs[envelope.Class]
	if codec == nil {
		s.err = errors.New("no codec found for class " + envelope.Class)
		return nil
	}

	event, err := codec.Deserialize(envelope.Data)
	if err != nil {
		s.err = err
		return nil
	}

	return &mongoDBEnvelope{
		IDVal:           envelope.ID,
		ExternalUUIDVal: dbValueAsUUID(envelope.ExternalUUIDVal),
		CreatedVal:      envelope.Created,
		CausationIDVal:  envelope.CausationID,
		EventVal:        event,
	}
}

// LoadEvents implements the EventStore interface.
func (s *MongoDBEventStore) LoadEvents(ctx context.Context, startAfter int32) (<-chan events.Envelope, error) {
	// don't do anything if the error state of the store is set already
	s.connect(ctx)
	if s.err != nil {
		return nil, s.err
	}

	out := make(chan events.Envelope)

	// run code to retrieve events in a goroutine
	go func() {
		// close channel on finish
		defer close(out)

		// don't do anything if the error state of the store is set already
		if s.Error() != nil {
			return
		}

		// load the referenced start object to verify the ID is valid
		if startAfter != 0 {
			// TODO: actually handle the error here
			ref, _ := s.RetrieveOne(ctx, startAfter)
			if ref == nil {
				return
			}
		}

		// pump events
		id := startAfter
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
			out <- envelope

			// move to next element
			id = envelope.IDVal
		}
	}()

	return out, nil
}

// FollowNotifications implements the EventStore interface.
func (s *MongoDBEventStore) FollowNotifications(ctx context.Context) (<-chan events.Notification, error) {
	// don't do anything if the error state of the store is set already
	s.connect(ctx)
	if s.err != nil {
		return nil, s.err
	}

	out := make(chan events.Notification)

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
			var note mongoDBNotification
			if err := cursor.Decode(&note); err != nil {
				s.err = err
				return
			}
			out <- &note
		}
	}()

	return out, nil
}

// FollowEvents implements the EventStore interface.
func (s *MongoDBEventStore) FollowEvents(ctx context.Context, startAfter int32) (<-chan events.Envelope, error) {
	// don't do anything if the error state of the store is set already
	s.connect(ctx)
	if s.err != nil {
		return nil, s.err
	}

	out := make(chan events.Envelope)

	// run code to retrieve events in a goroutine
	go func() {
		// close channel on finish
		defer close(out)

		// don't do anything if the error state of the store is set already
		if s.Error() != nil {
			return
		}

		// load the referenced start object to verify the ID is valid
		if startAfter != 0 {
			// TODO: actually handle the error here
			ref, _ := s.RetrieveOne(ctx, startAfter)
			if ref == nil {
				return
			}
		}

		// create notification channel
		nch, err := s.FollowNotifications(ctx)
		if err != nil {
			return
		}

		// pump events
		id := startAfter
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
					// cancelled by context
					return
				case notification := <-nch:
					if notification == nil {
						// notification channel closed
						return
					}
					continue
				}
			}

			// emit envelope
			out <- envelope

			// move to next element
			id = envelope.IDVal
		}
	}()

	return out, nil
}
