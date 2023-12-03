package rest_api

import (
	"api-broker-prototype/broker"
	"api-broker-prototype/events"
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/beego/beego/v2/server/web"
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

	// TODO: This doesn't work, the controller is cloned without copying
	// the store, so the store field is `nil` in the according handlers.
	// As workaround, we store it in a global instead.
	ctrl := &controller{
		store: store,
	}
	g_store = store

	// CopyRequestBody is required to to actually receive the body and to parse it as JSON
	web.BConfig.CopyRequestBody = true
	// AutoRender is suggested to be turned off for APIs
	web.BConfig.WebConfig.AutoRender = false

	web.Router("/up", ctrl, "get,head:HeadUp")
	web.Router("/request/:external_uuid", ctrl, "post:PostRequest")
	web.Router("/request/:external_uuid", ctrl, "get:GetRequest")

	web.Run(fmt.Sprintf(":%d", port))

	// TODO: reactivate
	// Getting at something that can be cancelled seems complicated,
	// so I'm leaving this feature out for now.
	select {
	case <-ctx.Done():
		// cancelled by context
		return ctx.Err()
	}
}

var g_store events.EventStore

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

type controller struct {
	web.Controller
	store events.EventStore
}

func (ctrl *controller) HeadUp() {
	ctrl.Ctx.WriteString("")
}

func (ctrl *controller) PostRequest() {
	// extract external UUID from the path
	external_uuid := uuid.FromStringOrNil(ctrl.GetString(":external_uuid"))
	if external_uuid == uuid.Nil {
		ctrl.Ctx.ResponseWriter.WriteHeader(http.StatusBadRequest)
		ctrl.Ctx.WriteString("Bad Request")
		return
	}

	// extract event from the body
	request := requestDTO{}
	if err := ctrl.BindJSON(&request); err != nil {
		ctrl.Ctx.ResponseWriter.WriteHeader(http.StatusBadRequest)
		ctrl.Ctx.WriteString("Bad Request")
		return
	}

	// insert event
	envelope, err := g_store.Insert(context.Background(), external_uuid, request.asEvent(), 0)
	if err != nil {
		if err == events.DuplicateEventUUID {
			ctrl.Ctx.ResponseWriter.WriteHeader(http.StatusConflict)
			ctrl.Ctx.WriteString("Conflict")
			return
		}

		ctrl.Ctx.ResponseWriter.WriteHeader(http.StatusInternalServerError)
		ctrl.Ctx.WriteString("Internal Server Error")
		return
	}

	response := newResponseDTO(envelope)
	ctrl.Ctx.ResponseWriter.WriteHeader(http.StatusCreated)
	ctrl.JSONResp(response)
}

func (ctrl *controller) GetRequest() {
	// extract external UUID from the path
	external_uuid := uuid.FromStringOrNil(ctrl.GetString(":external_uuid"))
	if external_uuid == uuid.Nil {
		ctrl.Ctx.ResponseWriter.WriteHeader(http.StatusBadRequest)
		ctrl.Ctx.WriteString(ctrl.GetString("external_uuid"))
		return
	}

	// convert UUID to internal ID
	id, err := g_store.ResolveUUID(context.Background(), external_uuid)
	if err != nil {
		ctrl.Ctx.ResponseWriter.WriteHeader(http.StatusNotFound)
		ctrl.Ctx.WriteString("failed to resolve external UUID")
		return
	}

	// load event
	envelope, err := g_store.RetrieveOne(context.Background(), id)
	if err != nil {
		ctrl.Ctx.ResponseWriter.WriteHeader(http.StatusInternalServerError)
		ctrl.Ctx.WriteString("failed to load event")
		return
	}

	response := newResponseDTO(envelope)
	ctrl.Ctx.ResponseWriter.WriteHeader(http.StatusOK)
	ctrl.Ctx.JSONResp(response)
}
