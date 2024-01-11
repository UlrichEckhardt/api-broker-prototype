package api

import (
	"api-broker-prototype/broker"
	"api-broker-prototype/events"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofrs/uuid"
)

// serve a simple API to access requests
func Serve(ctx context.Context, store events.EventStore, port uint) error {
	if store == nil {
		return errors.New("the supplied store is nil")
	}
	if port == 0 {
		return errors.New("the supplied port is zero")
	}

	app := fiber.New()

	ctrl := requestController{
		store: store,
	}
	app.Get("/up", ctrl.up)
	app.Head("/up", ctrl.up)
	app.Get("/request/:external_uuid", ctrl.getRequest)
	app.Post("/request/:external_uuid", ctrl.postRequest)

	go app.Listen(fmt.Sprint(":", port))

	select {
	case <-ctx.Done():
		// cancelled by context
		app.Shutdown()
		return ctx.Err()
	}
}

type requestController struct {
	store events.EventStore
}

func (ctrl *requestController) up(ctx fiber.Ctx) error {
	ctx.SendStatus(http.StatusOK)
	return nil
}

func (ctrl *requestController) getRequest(ctx fiber.Ctx) error {
	// extract external UUID from the path
	external_uuid := uuid.FromStringOrNil(ctx.Params("external_uuid"))
	if external_uuid == uuid.Nil {
		ctx.SendStatus(http.StatusBadRequest)
		ctx.SendString("Bad Request")
		return nil
	}

	// convert UUID to internal ID
	id, err := ctrl.store.ResolveUUID(ctx.Context(), external_uuid)
	if err != nil {
		ctx.SendStatus(http.StatusNotFound)
		ctx.SendString("failed to resolve external UUID")
		return nil
	}

	// load event
	envelope, err := ctrl.store.RetrieveOne(ctx.Context(), id)
	if err != nil {
		ctx.SendStatus(http.StatusInternalServerError)
		ctx.SendString("failed to load event")
		return nil
	}

	response := newResponseDTO(envelope)
	ctx.SendStatus(http.StatusOK)
	ctx.JSON(response)
	return nil
}

func (ctrl *requestController) postRequest(ctx fiber.Ctx) error {
	// extract external UUID from the path
	external_uuid := uuid.FromStringOrNil(ctx.Params("external_uuid"))
	if external_uuid == uuid.Nil {
		ctx.SendStatus(http.StatusBadRequest)
		ctx.SendString("Bad Request")
		return nil
	}

	// extract event from the body
	request := requestDTO{}
	if err := ctx.Bind().JSON(&request); err != nil {
		ctx.SendStatus(http.StatusBadRequest)
		ctx.SendString("Bad Request")
		return nil
	}

	// insert event
	envelope, err := ctrl.store.Insert(ctx.Context(), external_uuid, request.asEvent(), 0)
	if err != nil {
		if err == events.DuplicateEventUUID {
			ctx.SendStatus(http.StatusConflict)
			ctx.SendString("Conflict")
			return nil
		}

		ctx.SendStatus(http.StatusInternalServerError)
		ctx.SendString("Internal Server Error")
		return nil
	}

	response := newResponseDTO(envelope)
	ctx.SendStatus(http.StatusCreated)
	ctx.JSON(response)

	return nil
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
