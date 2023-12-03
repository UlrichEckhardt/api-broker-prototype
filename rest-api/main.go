package api

import (
	"api-broker-prototype/events"
	"context"
	"errors"
)

// serve a simple API to access requests
func Serve(ctx context.Context, store events.EventStore, port uint) error {
	if store == nil {
		return errors.New("the supplied store is nil")
	}
	if port == 0 {
		return errors.New("the supplied port is zero")
	}

	select {
	case <-ctx.Done():
		// cancelled by context
		return ctx.Err()
	}
}
