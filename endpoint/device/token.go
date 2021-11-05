package device

import (
	"context"
	"fmt"
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/charmixer/oas/api"

	"github.com/wraix/device-flow-proxy/app"
	"github.com/wraix/device-flow-proxy/endpoint"
	"github.com/wraix/device-flow-proxy/endpoint/problem"
)

type PostTokenRequest struct {
	ClientId   string `form:"client_id" validate:"required" oas-desc:"The client id"`
	DeviceCode string `form:"device_code" validate:"required" oas-desc:"The device code"`
	GrantType  string `form:"grant_type" validate:"required" oas-desc:"The grant type"`
}

type PostTokenError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

// https://golang.org/doc/effective_go#embedding
type PostTokenEndpoint struct {
	endpoint.Endpoint
}

func (ep PostTokenEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, span := tr.Start(ctx, fmt.Sprintf("%s execution", r.URL.Path))
	defer span.End()

	request := PostTokenRequest{}
	if err := endpoint.WithFormRequestParser(ctx, r, &request); err != nil {
		problem.MustWrite(w, err)
		return
	}

	if err := endpoint.WithRequestValidation(ctx, &request); err != nil {
		problem.MustWrite(w, err)
		return
	}

	deviceCode := request.DeviceCode

	// TODO add rate limiting in middleware

	// Check if the device code is in the cache
	_data, found := app.Env.Cache.Get(deviceCode)
	if !found {
		w.Header().Set("Content-Type", "application/json")
		e := PostTokenError{
			Error: "invalid_grant",
		}
		w.WriteHeader(http.StatusBadRequest)
		if err := endpoint.WithJsonResponseWriter(ctx, w, e); err != nil {
			log.Error().Err(err).Str("hint", "device code not found in cache").Msg("Unable to write json")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	data := _data.(map[string]string)

	if data["status"] == "pending" {
		w.Header().Set("Content-Type", "application/json")
		e := PostTokenError{
			Error: "authorization_pending",
		}
		w.WriteHeader(http.StatusBadRequest)
		if err := endpoint.WithJsonResponseWriter(ctx, w, e); err != nil {
			log.Error().Err(err).Str("status", data["status"]).Msg("Unable to write json")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	if data["status"] != "complete" {
		w.Header().Set("Content-Type", "application/json")
		e := PostTokenError{
			Error: "invalid_grant",
		}
		w.WriteHeader(http.StatusBadRequest)
		if err := endpoint.WithJsonResponseWriter(ctx, w, e); err != nil {
			log.Error().Err(err).Str("status", data["status"]).Msg("Unable to write json")
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}

	// Everything is awesome

	deleteCacheForDeviceCode(ctx, deviceCode)

	// Just return what hydra made as an access token. No output validation.
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(data["token_response"]))
}

func deleteCacheForDeviceCode(ctx context.Context, deviceCode string) {
	_, unitOfWork := tr.Start(ctx, "Delete cache for device code")
	defer unitOfWork.End()
	app.Env.Cache.Delete(deviceCode)
}

func NewPostTokenEndpoint() endpoint.EndpointHandler {
	ep := PostTokenEndpoint{}

	ep.Setup(
		endpoint.WithSpecification(api.Path{
			Summary:     "Exchange device code for access token",
			Description: ``,
			Tags:        OPENAPI_TAGS,

			Request: api.Request{
				Description: ``,
				Schema:      PostTokenRequest{},
			},

			Responses: []api.Response{{
				Description: "The access token from the OAuth2 provider",
				Code:        http.StatusOK,
			}, {
				Description: http.StatusText(http.StatusBadRequest),
				Code:        http.StatusBadRequest,
				Schema:      PostTokenError{},
			}, {
				Description: http.StatusText(http.StatusBadRequest),
				Code:        http.StatusBadRequest,
				Schema:      problem.ValidationProblem{},
			}},
		}),
	)

	return ep
}
