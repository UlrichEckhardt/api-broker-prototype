package events

// This file collects error types.

import "errors"

// DuplicateEventUUID is used to signal that the UUID identifying an event is already in use
var DuplicateEventUUID = errors.New("duplicate event identifier UUID")
