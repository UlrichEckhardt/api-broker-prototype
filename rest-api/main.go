package api

import (
	"api-broker-prototype/broker"
	"api-broker-prototype/events"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gofrs/uuid"
	"github.com/kataras/iris/v12"
)

// serve a simple API to access requests
func Serve(ctx context.Context, store events.EventStore, port uint) error {
	if store == nil {
		return errors.New("the supplied store is nil")
	}
	if port == 0 {
		return errors.New("the supplied port is zero")
	}

	app := iris.New()
	app.Use(iris.Compression)

	app.Get("/up", func(ctx iris.Context) {
		ctx.ResponseWriter().WriteHeader(http.StatusOK)
	})
	app.Head("/up", func(ctx iris.Context) {
		ctx.ResponseWriter().WriteHeader(http.StatusOK)
	})
	app.Get("/request/{external_uuid}", func(ctx iris.Context) {
		// extract external UUID from the path
		external_uuid := uuid.FromStringOrNil(ctx.Params().Get("external_uuid"))
		if external_uuid == uuid.Nil {
			ctx.ResponseWriter().WriteHeader(http.StatusBadRequest)
			ctx.WriteString("Bad Request")
			return
		}

		// convert UUID to internal ID
		id, err := store.ResolveUUID(ctx, external_uuid)
		if err != nil {
			ctx.ResponseWriter().WriteHeader(http.StatusNotFound)
			ctx.WriteString("failed to resolve external UUID")
			return
		}

		// load event
		envelope, err := store.RetrieveOne(ctx, id)
		if err != nil {
			ctx.ResponseWriter().WriteHeader(http.StatusInternalServerError)
			ctx.WriteString("failed to load event")
			return
		}

		response := newResponseDTO(envelope)
		ctx.ResponseWriter().WriteHeader(http.StatusOK)
		ctx.JSON(response)
	})
	app.Post("/request/:external_uuid", func(ctx iris.Context) {
		// extract external UUID from the path
		external_uuid := uuid.FromStringOrNil(ctx.Params().Get("external_uuid"))
		if external_uuid == uuid.Nil {
			ctx.ResponseWriter().WriteHeader(http.StatusBadRequest)
			ctx.WriteString("Bad Request")
			return
		}

		// extract event from the body
		request := requestDTO{}
		if err := ctx.ReadJSON(&request); err != nil {
			ctx.ResponseWriter().WriteHeader(http.StatusBadRequest)
			ctx.WriteString("Bad Request")
			return
		}

		// insert event
		envelope, err := store.Insert(ctx, external_uuid, request.asEvent(), 0)
		if err != nil {
			if err == events.DuplicateEventUUID {
				ctx.ResponseWriter().WriteHeader(http.StatusConflict)
				ctx.WriteString("Conflict")
				return
			}

			ctx.ResponseWriter().WriteHeader(http.StatusInternalServerError)
			ctx.WriteString("Internal Server Error")
			return
		}

		response := newResponseDTO(envelope)
		ctx.ResponseWriter().WriteHeader(http.StatusCreated)
		ctx.JSON(response)
	})

	go app.Listen(fmt.Sprint(":", port))

	select {
	case <-ctx.Done():
		// cancelled by context
		app.Shutdown(context.Background())
		return ctx.Err()
	}
}

// representation of the request during HTTP transfer
type requestDTO struct {
	Data string `json:"data"`
}

// asEvent converts the DTO to the internally used representation
func (dto requestDTO) asEvent() broker.RequestEvent {
	return broker.RequestEvent{
		Request: dto.Data,
	}
}

// representation of the (sucessful) response during HTTP transfer
type responseDTO struct {
	Created      time.Time `json:"created"`
	ExternalUUID uuid.UUID `json:"external_uuid"`
}

// newResponseDTO converts the envelope to the externally used DTO
func newResponseDTO(envelope events.Envelope) responseDTO {
	return responseDTO{
		Created:      envelope.Created(),
		ExternalUUID: envelope.ExternalUUID(),
	}
}
