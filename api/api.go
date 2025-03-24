package api

// This file implements access to the exemplary "brittle-api"

import (
	"context"
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
func ProcessRequest(ctx context.Context, request string) (*string, error) {
	// assemble request
	req, err := http.NewRequestWithContext(
		ctx,
		"POST",
		conf.api_url+"/api",
		strings.NewReader(request),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// delegate to API
	response, err := http.DefaultClient.Do(req)
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
