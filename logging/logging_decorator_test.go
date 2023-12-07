package logging

import (
	"api-broker-prototype/events"
	"context"
	"errors"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/inconshreveable/log15"
)

var notImplemented error = errors.New("not implemented")

// mock for the events.EventStore interface
type eventstoreMock struct{}

func (store *eventstoreMock) ParseEventID(str string) (int32, error) {
	return 0, notImplemented
}

func (store *eventstoreMock) Error() error {
	return notImplemented
}

func (store *eventstoreMock) Close() error {
	return notImplemented
}

func (store *eventstoreMock) Insert(ctx context.Context, externalUUID uuid.UUID, event events.Event, causationID int32) (events.Envelope, error) {
	return nil, notImplemented
}

func (store *eventstoreMock) ResolveUUID(ctx context.Context, externalUUID uuid.UUID) (int32, error) {
	return 0, notImplemented
}

func (store *eventstoreMock) RetrieveOne(ctx context.Context, id int32) (events.Envelope, error) {
	return nil, notImplemented
}

func (store *eventstoreMock) LoadEvents(ctx context.Context, startAfter int32) (<-chan events.Envelope, error) {
	return nil, notImplemented
}

func (store *eventstoreMock) FollowNotifications(ctx context.Context) (<-chan events.Notification, error) {
	return nil, notImplemented
}

func (store *eventstoreMock) FollowEvents(ctx context.Context, startAfter int32) (<-chan events.Envelope, error) {
	return nil, notImplemented
}

// mock for the events.Event interface
type eventMock struct{}

func (event *eventMock) Class() string {
	return "mock"
}

// mock for the log15.Logger interface
type loggerMock struct{}

func (logger *loggerMock) New(ctx ...interface{}) log15.Logger {
	return logger
}

func (logger *loggerMock) GetHandler() log15.Handler {
	return nil
}

func (logger *loggerMock) SetHandler(h log15.Handler) {

}

func (logger *loggerMock) Debug(msg string, ctx ...interface{}) {

}

func (logger *loggerMock) Info(msg string, ctx ...interface{}) {

}

func (logger *loggerMock) Warn(msg string, ctx ...interface{}) {

}

func (logger *loggerMock) Error(msg string, ctx ...interface{}) {

}

func (logger *loggerMock) Crit(msg string, ctx ...interface{}) {

}

func createMock() *LoggingDecoratorEventStore {
	logger := &loggerMock{}
	eventstore := &eventstoreMock{}
	decorator, err := NewLoggingDecorator(eventstore, logger)
	if err != nil {
		panic(err)
	}
	return decorator
}

// make sure the decorator implements the event store interface
func TestInterface(t *testing.T) {
	var _ events.EventStore = &LoggingDecoratorEventStore{}
}

func TestParseEventID(t *testing.T) {
	decorator := createMock()

	res, err := decorator.ParseEventID("abc")

	if res != 0 {
		t.Errorf("expected zero as result")
	}
	if err != notImplemented {
		t.Errorf("unexpected error")
	}
}

func TestError(t *testing.T) {
	decorator := createMock()

	err := decorator.Error()

	if err != notImplemented {
		t.Errorf("unexpected error")
	}
}

func TestClose(t *testing.T) {
	decorator := createMock()

	err := decorator.Close()

	if err != notImplemented {
		t.Errorf("unexpected error")
	}
}

func TestInsert(t *testing.T) {
	decorator := createMock()

	ctx := context.Background()

	externalUUID, err := uuid.NewV4()
	if err != nil {
		t.Errorf("failed to generate UUID")
	}

	event := eventMock{}

	res, err := decorator.Insert(ctx, externalUUID, &event, 0)

	if res != nil {
		t.Errorf("expected nil as result")
	}
	if err != notImplemented {
		t.Errorf("unexpected error")
	}
}

func TestResolveUUID(t *testing.T) {
	decorator := createMock()

	ctx := context.Background()

	externalUUID, err := uuid.NewV4()
	if err != nil {
		t.Errorf("failed to generate UUID")
	}

	res, err := decorator.ResolveUUID(ctx, externalUUID)
	if res != 0 {
		t.Errorf("expected zero as result")
	}
	if err != notImplemented {
		t.Errorf("unexpected error")
	}
}

func TestRetrieveOne(t *testing.T) {
	decorator := createMock()

	ctx := context.Background()

	res, err := decorator.RetrieveOne(ctx, 42)

	if res != nil {
		t.Errorf("expected nil as result")
	}
	if err != notImplemented {
		t.Errorf("unexpected error")
	}
}

func TestLoadEvents(t *testing.T) {
	decorator := createMock()

	ctx := context.Background()

	res, err := decorator.LoadEvents(ctx, 42)

	if res != nil {
		t.Errorf("expected nil as result")
	}
	if err != notImplemented {
		t.Errorf("unexpected error")
	}
}

func TestFollowEvents(t *testing.T) {
	decorator := createMock()

	ctx := context.Background()

	res, err := decorator.FollowEvents(ctx, 0)

	if res != nil {
		t.Errorf("expected nil as result")
	}
	if err != notImplemented {
		t.Errorf("unexpected error")
	}
}

func TestFollowNotifications(t *testing.T) {
	decorator := createMock()

	ctx := context.Background()

	res, err := decorator.FollowNotifications(ctx)

	if res != nil {
		t.Errorf("expected nil as result")
	}
	if err != notImplemented {
		t.Errorf("unexpected error")
	}
}
