package api

// This file implements access to the exemplary "brittle-api"

import (
	"errors"
	"net/http"
	"strings"
)

// configuration values
var conf struct {
	api_url string
}

// Configure can be used to configure the behaviour of the mock API.
func Configure(api_url string) {
	conf.api_url = api_url
}

// ProcessRequest handles a single request
// Note that this can return an actual response, an error (when the HTTP
// response doesn't have status 200) or neither. The latter is used to
// mimick the behaviour of a remote API not responding at all.
// TODO: `ctx context.Context` parameter
func ProcessRequest(request string) (*string, error) {
	// delegate to API
	response, err := http.Post(conf.api_url+"/api", "application/x-www-form-urlencoded", strings.NewReader(request))
	if err != nil {
		// no response at all
		return nil, nil
	}
	defer response.Body.Close()

	// check response status
	if response.StatusCode != 200 {
		return nil, errors.New(response.Status)
	}

	// extract body as result
	len := 100
	buf := make([]byte, len)
	n, err := response.Body.Read(buf[0:])
	if n == 0 && err != nil {
		return nil, errors.New("empty response body from API.")
	}
	body := string(buf[:n])

	return &body, nil
}
