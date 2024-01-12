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
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

// serve a simple API to access requests
func Serve(ctx context.Context, store events.EventStore, port uint) error {
	if store == nil {
		return errors.New("the supplied store is nil")
	}
	if port == 0 {
		return errors.New("the supplied port is zero")
	}

	e := echo.New()

	ctrl := requestController{
		store: store,
	}
	// GET /up -- server status check
	e.GET("/up", ctrl.up)

	// HEAD /up -- server status check
	e.HEAD("/up", ctrl.up)

	// GET /request/<external UUID> -- fetch request instance
	e.GET("/request/:external_uuid", ctrl.getRequest)

	// POST /request/<external UUID> with JSON body -- add a request instance
	// TODO: Allow an optional UUID which serves as causation ID
	e.POST("/request/:external_uuid", ctrl.postRequest)

	// GET /response/<ID> -- fetch current state of the request <ID>
	// GET /response -- fetch current state of all requests
	// PUT /configuration -- fetch current configuration
	// GET /configuration -- set configuration
	// PATCH /configuration?

	e.Logger.SetLevel(log.DEBUG)
	return e.Start(fmt.Sprint(":", port))
}

type requestController struct {
	store events.EventStore
}

func (ctrl *requestController) up(c echo.Context) error {
	return c.NoContent(http.StatusOK)
}

func (ctrl *requestController) getRequest(c echo.Context) error {
	// extract external UUID from the path
	external_uuid := uuid.FromStringOrNil(c.Param("external_uuid"))
	if external_uuid == uuid.Nil {
		return c.String(http.StatusBadRequest, "bad request")
	}

	id, err := ctrl.store.ResolveUUID(c.Request().Context(), external_uuid)
	if err != nil {
		return c.String(http.StatusNotFound, "failed to resolve external UUID")
	}

	envelope, err := ctrl.store.RetrieveOne(c.Request().Context(), id)
	if err != nil {
		return c.String(http.StatusInternalServerError, "failed to load event")
	}

	response := newResponseDTO(envelope)
	return c.JSON(http.StatusOK, response)
}

func (ctrl *requestController) postRequest(c echo.Context) error {
	// extract external UUID from the path
	external_uuid := uuid.FromStringOrNil(c.Param("external_uuid"))
	if external_uuid == uuid.Nil {
		return c.String(http.StatusBadRequest, "bad request")
	}

	// extract event from the body
	var request requestDTO
	if err := c.Bind(&request); err != nil {
		c.Logger().Error(err)
		return c.String(http.StatusBadRequest, "bad request")
	}

	// insert event
	envelope, err := ctrl.store.Insert(c.Request().Context(), external_uuid, request.asEvent(), 0)
	if err != nil {
		c.Logger().Error(err)
		if err == events.DuplicateEventUUID {
			return c.String(http.StatusConflict, "duplicate event")
		}
		return c.String(http.StatusInternalServerError, "failed to insert new event")
	}

	response := newResponseDTO(envelope)
	return c.JSON(http.StatusCreated, response)
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
