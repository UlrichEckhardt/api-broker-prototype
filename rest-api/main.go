package api

import (
	"api-broker-prototype/broker"
	"api-broker-prototype/events"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
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

	r := gin.Default()
	ctrl := requestController{
		store: store,
	}
	r.GET("/up", ctrl.up)
	r.HEAD("/up", ctrl.up)
	r.GET("/request/:external_uuid", ctrl.getRequest)
	r.POST("/request/:external_uuid", ctrl.postRequest)

	listener, err := net.Listen("tcp", fmt.Sprint(":", port))
	if err != nil {
		return err
	}
	defer listener.Close()

	go func() {
		println("starting HTTP server")
		r.RunListener(listener)
		println("stopped HTTP server")
	}()

	select {
	case <-ctx.Done():
		// cancelled by context
		return ctx.Err()
	}
}

type requestController struct {
	store events.EventStore
}

func (ctrl *requestController) up(c *gin.Context) {
	c.String(http.StatusOK, "")
}

func (ctrl *requestController) getRequest(c *gin.Context) {
	// extract external UUID from the path
	external_uuid := uuid.FromStringOrNil(c.Param("external_uuid"))
	if external_uuid == uuid.Nil {
		c.String(http.StatusBadRequest, "Bad Request")
		return
	}

	// convert UUID to internal ID
	id, err := ctrl.store.ResolveUUID(c.Request.Context(), external_uuid)
	if err != nil {
		c.String(http.StatusNotFound, "failed to resolve external UUID")
		return
	}

	// load event
	envelope, err := ctrl.store.RetrieveOne(c.Request.Context(), id)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to load event")
		return
	}

	response := newResponseDTO(envelope)
	c.JSON(http.StatusOK, response)
}

func (ctrl *requestController) postRequest(c *gin.Context) {
	// extract external UUID from the path
	external_uuid := uuid.FromStringOrNil(c.Param("external_uuid"))
	if external_uuid == uuid.Nil {
		c.String(http.StatusBadRequest, "Bad Request")
		return
	}

	// extract event from the body
	request := requestDTO{}
	if err := c.BindJSON(&request); err != nil {
		c.String(http.StatusBadRequest, "Bad Request")
		return
	}

	// insert event
	envelope, err := ctrl.store.Insert(c.Request.Context(), external_uuid, request.asEvent(), 0)
	if err != nil {
		if err == events.DuplicateEventUUID {
			c.String(http.StatusConflict, "Conflict")
			return
		}

		c.String(http.StatusInternalServerError, "Internal Server Error")
		return
	}

	response := newResponseDTO(envelope)
	c.JSON(http.StatusCreated, response)
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
