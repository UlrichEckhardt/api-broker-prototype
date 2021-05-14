package postgresql

// Eventstore built on top of a PostgreSQL DB
// This uses Postgre's LISTEN/NOTIFY/pg_notify() feature to efficiently signal
// new events.

import (
	"api-broker-prototype/events"
	"context"
	"errors"
	"fmt"
	"github.com/inconshreveable/log15"
	"github.com/jackc/pgtype"
	"github.com/jackc/pgx/v4"
	"strconv"
	"time"
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

// ID implements the EventStore interface.
func (env *postgreSQLEnvelope) ID() int32 {
	return env.IDVal
}

// Created implements the EventStore interface.
func (env *postgreSQLEnvelope) Created() time.Time {
	return env.CreatedVal
}

// CausationID implements the EventStore interface.
func (env *postgreSQLEnvelope) CausationID() int32 {
	return env.CausationIDVal
}

// Event implements the EventStore interface.
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
	logger log15.Logger
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
	return conn
}

// NewEventStore creates and connects a PostgreSQLEventStore instance.
func NewEventStore(logger log15.Logger, host string) (*PostgreSQLEventStore, error) {
	logger.Debug("creating event store", "host", host)
	s := PostgreSQLEventStore{
		logger: logger,
		host:   host,
		codecs: make(map[string]PostgreSQLEventCodec),
		err:    nil,
	}

	// register codecs
	s.registerCodec(&simpleEventCodec{})
	s.registerCodec(&configurationEventCodec{})
	s.registerCodec(&requestEventCodec{})
	s.registerCodec(&responseEventCodec{})
	s.registerCodec(&failureEventCodec{})

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
	s.logger.Debug("closing eventstore")

	// nothing to do, the eventstore doesn't bind any temporary resources

	// set this error to block any further calls
	s.err = errors.New("eventstore is closed")

	return nil
}

// Insert implements the EventStore interface.
func (s *PostgreSQLEventStore) Insert(ctx context.Context, event events.Event, causationID int32) (events.Envelope, error) {
	s.logger.Debug("inserting event", "causation", causationID, "class", event.Class())

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

	// retrieve assigned ID from response
	var id int32
	if err = row.Scan(&id); err != nil {
		s.logger.Error("Unable to insert into the 'messages' table.", "error", err)
		return nil, err
	}

	s.logger.Debug("Inserted data into 'messages' table.", "id", id)

	res := &postgreSQLEnvelope{
		IDVal:          id,
		CreatedVal:     now,
		CausationIDVal: causationID,
		EventVal:       event,
	}

	return res, nil
}

// RetrieveOne implements the EventStore interface.
func (s *PostgreSQLEventStore) RetrieveOne(ctx context.Context, id int32) (events.Envelope, error) {
	s.logger.Debug("loading event", "id", id)

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
	var created time.Time
	var causationID int32
	var class string
	var payload pgtype.JSONB
	if err := row.Scan(&created, &causationID, &class, &payload); err != nil {
		return nil, err
	}

	// locate codec for the event class
	codec := s.codecs[class]
	if codec == nil {
		return nil, errors.New("failed to locate codec for event")
	}

	// decode event from storage
	event, err := codec.Deserialize(payload)
	if err != nil {
		return nil, err
	}

	res := &postgreSQLEnvelope{
		IDVal:          id,
		CreatedVal:     created,
		CausationIDVal: causationID,
		EventVal:       event,
	}
	return res, nil
}

// LoadEvents implements the EventStore interface.
func (s *PostgreSQLEventStore) LoadEvents(ctx context.Context, start int32) (<-chan events.Envelope, error) {
	s.logger.Debug("loading events", "start", start)

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
			start,
		)
		if err != nil {
			s.err = err
			return
		}

		for rows.Next() {
			// extract fields from response
			var id int32
			var created time.Time
			var causationID int32
			var class string
			var payload pgtype.JSONB
			if err := rows.Scan(&id, &created, &causationID, &class, &payload); err != nil {
				s.err = err
				return
			}

			// locate codec for the event class
			codec := s.codecs[class]
			if codec == nil {
				s.err = errors.New("failed to locate codec for event")
				return
			}

			// decode event from storage
			event, err := codec.Deserialize(payload)
			if err != nil {
				s.err = err
				return
			}

			out <- &postgreSQLEnvelope{
				IDVal:          id,
				CreatedVal:     created,
				CausationIDVal: causationID,
				EventVal:       event,
			}
		}
	}()

	return out, nil
}

// FollowNotifications implements the EventStore interface.
func (s *PostgreSQLEventStore) FollowNotifications(ctx context.Context) (<-chan events.Notification, error) {
	s.logger.Debug("following notifications")

	// establish connection
	conn := s.connect(ctx)
	if conn == nil {
		return nil, s.err
	}

	// close connection on finish
	defer conn.Close(ctx)

	return nil, errors.New("not implemented")
}

// FollowEvents implements the EventStore interface.
func (s *PostgreSQLEventStore) FollowEvents(ctx context.Context, start int32) (<-chan events.Envelope, error) {
	s.logger.Debug("following events", "start", start)

	// establish connection
	conn := s.connect(ctx)
	if conn == nil {
		return nil, s.err
	}

	// close connection on finish
	defer conn.Close(ctx)

	return nil, errors.New("not implemented")
}
