package postgresql

// Eventstore built on top of a PostgreSQL DB
// This uses Postgre's LISTEN/NOTIFY/pg_notify() feature to efficiently signal
// new events.

import (
	"api-broker-prototype/events"
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gofrs/uuid"
	"github.com/jackc/pgtype"
	pgxuuid "github.com/jackc/pgx-gofrs-uuid"
	"github.com/jackc/pgx/v5"
)

const (
	POSTGRES_USER     = "postgres"
	POSTGRES_PASSWORD = "postgres"
	POSTGRES_PORT     = "5432"
	POSTGRES_DATABASE = "postgres"
)

// The PostgreSQLEventCodec interface defines methods common to event codecs.
// The codecs convert between the internal representation (Event) and the
// general-purpose representation for PostgreSQL (JSONB).
// See also the Event interface, which it is closely related to.
type PostgreSQLEventCodec interface {
	// Class returns a string that identifies the event type this codec handles.
	Class() string
	// Serialize the event in a way that allows writing it to a PostgreSQL DB.
	Serialize(event events.Event) (pgtype.JSONB, error)
	// Deserialize an event from data from a PostgreSQL DB.
	Deserialize(data pgtype.JSONB) (events.Event, error)
}

// postgreSQLEnvelope implements the Envelope interface.
type postgreSQLEnvelope struct {
	IDVal          int32
	CreatedVal     time.Time
	CausationIDVal int32
	EventVal       events.Event
}

// ID implements the Envelope interface.
func (env *postgreSQLEnvelope) ID() int32 {
	return env.IDVal
}

// Created implements the Envelope interface.
func (env *postgreSQLEnvelope) Created() time.Time {
	return env.CreatedVal
}

// ExternalUUID implements the Envelope interface.
func (env *postgreSQLEnvelope) ExternalUUID() uuid.UUID {
	// TODO: implement UUID support
	return uuid.Nil
}

// CausationID implements the Envelope interface.
func (env *postgreSQLEnvelope) CausationID() int32 {
	return env.CausationIDVal
}

// Event implements the Envelope interface.
func (env *postgreSQLEnvelope) Event() events.Event {
	return env.EventVal
}

// postgreSQLNotification implements the Notification interface.
type postgreSQLNotification struct {
	IDVal int32
}

// ID implements the Notification interface.
func (note *postgreSQLNotification) ID() int32 {
	return note.IDVal
}

// PostgreSQLEventStore implements the EventStore interface using a PostgreSQL DB
type PostgreSQLEventStore struct {
	host   string
	codecs map[string]PostgreSQLEventCodec
	err    error
}

// connect to the PostgreSQL database
// This will return the created connection instance. It will set the error
// state of the eventstore instance and return `nil` on failure. Release the
// returned connection using its `Close()` method.
func (s *PostgreSQLEventStore) connect(ctx context.Context) *pgx.Conn {
	// do nothing when there's already an error present
	if s.err != nil {
		return nil
	}

	cs := fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		POSTGRES_USER,
		POSTGRES_PASSWORD,
		s.host,
		POSTGRES_PORT,
		POSTGRES_DATABASE,
	)
	conn, err := pgx.Connect(ctx, cs)
	if err != nil {
		s.err = err
		return nil
	}

	pgxuuid.Register(conn.TypeMap())

	return conn
}

// NewEventStore creates and connects a PostgreSQLEventStore instance.
func NewEventStore(host string) (*PostgreSQLEventStore, error) {
	s := PostgreSQLEventStore{
		host:   host,
		codecs: make(map[string]PostgreSQLEventCodec),
		err:    nil,
	}

	// register codecs
	s.registerCodec(&simpleEventCodec{})
	s.registerCodec(&configurationEventCodec{})
	s.registerCodec(&requestEventCodec{})
	s.registerCodec(&apiRequestEventCodec{})
	s.registerCodec(&apiResponseEventCodec{})
	s.registerCodec(&apiFailureEventCodec{})
	s.registerCodec(&apiTimeoutEventCodec{})

	return &s, nil
}

// registerCodec registers a codec that allows conversion of Events.
func (s *PostgreSQLEventStore) registerCodec(codec PostgreSQLEventCodec) {
	if codec == nil {
		s.err = errors.New("nil codec registered")
		return
	}
	s.codecs[codec.Class()] = codec
}

// decode event from the class name and JSON data
func (s *PostgreSQLEventStore) decodeEvent(class string, payload pgtype.JSONB) (events.Event, error) {
	// locate codec for the event class
	codec := s.codecs[class]
	if codec == nil {
		return nil, errors.New("failed to locate codec for event")
	}

	// decode event from storage
	return codec.Deserialize(payload)
}

// ParseEventID implements the EventStore interface.
func (s *PostgreSQLEventStore) ParseEventID(str string) (int32, error) {
	lp, err := strconv.ParseInt(str, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(lp), nil
}

// Error implements the EventStore interface.
func (s *PostgreSQLEventStore) Error() error {
	return s.err
}

// Error implements the EventStore and io.Closer interfaces.
// Note that this must correctly release any resources even with the error state set!
func (s *PostgreSQLEventStore) Close() error {
	// nothing to do, the eventstore doesn't bind any temporary resources

	// set this error to block any further calls
	s.err = errors.New("eventstore is closed")

	return nil
}

// Insert implements the EventStore interface.
func (s *PostgreSQLEventStore) Insert(ctx context.Context, externalUUID uuid.UUID, event events.Event, causationID int32) (events.Envelope, error) {
	// TODO: implement UUID support
	if externalUUID != uuid.Nil {
		return nil, errors.New("UUID support not implemented")
	}

	// locate codec for the event class
	class := event.Class()
	codec := s.codecs[class]
	if codec == nil {
		return nil, errors.New("failed to locate codec for event")
	}

	// encode event for storage
	payload, err := codec.Serialize(event)
	if err != nil {
		return nil, err
	}

	// establish connection
	conn := s.connect(ctx)
	if conn == nil {
		return nil, s.err
	}
	defer conn.Close(ctx)

	// insert the event into the DB
	now := time.Now()
	row := conn.QueryRow(
		ctx,
		`INSERT INTO events (created, causation_id, class, payload) VALUES ($1, $2, $3, $4) RETURNING id;`,
		now,
		causationID,
		class,
		payload,
	)

	// init envelope with input values
	res := postgreSQLEnvelope{
		CreatedVal:     now,
		CausationIDVal: causationID,
		EventVal:       event,
	}

	// retrieve assigned ID from response
	if err = row.Scan(&res.IDVal); err != nil {
		// unable to insert into the 'messages' table.
		return nil, err
	}

	return &res, nil
}

// ResolveUUID implements the EventStore interface.
func (s *PostgreSQLEventStore) ResolveUUID(ctx context.Context, externalUUID uuid.UUID) (int32, error) {
	// TODO: implement UUID support
	return 0, errors.New("UUID support not implemented")
}

// RetrieveOne implements the EventStore interface.
func (s *PostgreSQLEventStore) RetrieveOne(ctx context.Context, id int32) (events.Envelope, error) {
	// The ID must be valid.
	if id == 0 {
		return nil, errors.New("provided document ID is null")
	}

	// establish connection
	conn := s.connect(ctx)
	if conn == nil {
		return nil, s.err
	}
	defer conn.Close(ctx)

	// retrieve row from DB
	row := conn.QueryRow(
		ctx,
		`SELECT created, causation_id, class, payload FROM events WHERE id = $1;`,
		id,
	)

	// extract fields from response
	res := postgreSQLEnvelope{
		IDVal: id,
	}
	var class string
	var payload pgtype.JSONB
	if err := row.Scan(&res.CreatedVal, &res.CausationIDVal, &class, &payload); err != nil {
		return nil, err
	}

	// decode event
	if ev, err := s.decodeEvent(class, payload); err != nil {
		return nil, err
	} else {
		res.EventVal = ev
		return &res, nil
	}
}

// LoadEvents implements the EventStore interface.
func (s *PostgreSQLEventStore) LoadEvents(ctx context.Context, startAfter int32) (<-chan events.Envelope, error) {
	// establish connection
	conn := s.connect(ctx)
	if conn == nil {
		return nil, s.err
	}

	// run code to pump events in a goroutine
	out := make(chan events.Envelope)
	go func() {
		// close connection on finish
		defer conn.Close(ctx)
		// close channel on finish
		defer close(out)

		// retrieve rows from DB
		rows, err := conn.Query(
			ctx,
			`SELECT id, created, causation_id, class, payload FROM events WHERE id > $1;`,
			startAfter,
		)
		if err != nil {
			s.err = err
			return
		}

		for rows.Next() {
			// extract fields from response
			var res postgreSQLEnvelope
			var class string
			var payload pgtype.JSONB
			if err := rows.Scan(&res.IDVal, &res.CreatedVal, &res.CausationIDVal, &class, &payload); err != nil {
				s.err = err
				return
			}

			// decode event
			if ev, err := s.decodeEvent(class, payload); err != nil {
				s.err = err
				return
			} else {
				res.EventVal = ev
			}

			out <- &res
		}
	}()

	return out, nil
}

// FollowNotifications implements the EventStore interface.
func (s *PostgreSQLEventStore) FollowNotifications(ctx context.Context) (<-chan events.Notification, error) {
	// establish connection
	conn := s.connect(ctx)
	if conn == nil {
		return nil, s.err
	}

	// run code to pump events in a goroutine
	out := make(chan events.Notification)
	go func() {
		// close connection on finish
		defer conn.Close(ctx)
		// close channel on finish
		defer close(out)

		// register as listening to notification channel
		_, err := conn.Exec(
			ctx,
			"LISTEN notification;",
		)
		if err != nil {
			s.err = err
			return
		}

		// wait for notifications
		for {
			notification, err := conn.WaitForNotification(ctx)
			if err != nil {
				s.err = err
				return
			}

			// extract event ID from the notification
			id, err := s.ParseEventID(notification.Payload)
			if err != nil {
				s.err = err
				return
			}
			out <- &postgreSQLNotification{
				IDVal: id,
			}
		}
	}()

	return out, nil
}

// FollowEvents implements the EventStore interface.
func (s *PostgreSQLEventStore) FollowEvents(ctx context.Context, startAfter int32) (<-chan events.Envelope, error) {
	// establish connection
	conn := s.connect(ctx)
	if conn == nil {
		return nil, s.err
	}

	// run code to pump events in a goroutine
	out := make(chan events.Envelope)
	go func() {
		// close connection on finish
		defer conn.Close(ctx)
		// close channel on finish
		defer close(out)

		// register as listening to notification channel
		_, err := conn.Exec(
			ctx,
			"LISTEN notification;",
		)
		if err != nil {
			s.err = err
			return
		}

		for {
			// retrieve rows from DB
			rows, err := conn.Query(
				ctx,
				`SELECT id, created, causation_id, class, payload FROM events WHERE id > $1;`,
				startAfter,
			)
			if err != nil {
				s.err = err
				return
			}

			for rows.Next() {
				// extract fields from response
				var res postgreSQLEnvelope
				var class string
				var payload pgtype.JSONB
				if err := rows.Scan(&res.IDVal, &res.CreatedVal, &res.CausationIDVal, &class, &payload); err != nil {
					s.err = err
					return
				}

				// decode event
				if ev, err := s.decodeEvent(class, payload); err != nil {
					s.err = err
					return
				} else {
					res.EventVal = ev
				}

				out <- &res

				// remember new position in stream
				startAfter = res.IDVal
			}

			// wait for notifications of new events
			_, err = conn.WaitForNotification(ctx)
			if err != nil {
				s.err = err
				return
			}
		}
	}()

	return out, nil
}
