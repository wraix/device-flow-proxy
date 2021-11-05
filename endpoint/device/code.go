package device

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/charmixer/oas/api"

	"github.com/wraix/device-flow-proxy/app"
	"github.com/wraix/device-flow-proxy/endpoint"
	"github.com/wraix/device-flow-proxy/endpoint/problem"
)

type PostCodeRequest struct {
	ClientId string `form:"client_id" validate:"required" oas-desc:"The client id"`
	Scope    string `form:"client_id" oas-desc:"The scopes"`
}
type PostCodeResponse struct {
	DeviceCode      string `json:"device_code" validate:"required" oas-desc:"This is a long string that the device will use to eventually exchange for an access token"`
	VerificationUri string `json:"verification_uri" validate:"required" oas-desc:"This is the URL the user needs to enter into their phone to start logging in"`
	UserCode        string `json:"user_code" validate:"required" oas-desc:"This is the text the user will enter at the verification uri."`
	ExpiresIn       int    `json:"expires_in" validate:"required" oas-desc:"he number of seconds that this set of values is valid. After this amount of time, the device_code and user_code will expire and the device will have to start over"`
	Interval        int    `json:"interval" validate:"required" oas-desc:"The number of seconds the device should wait between polling to see if the user has finished logging in"`
}

// https://golang.org/doc/effective_go#embedding
type PostCodeEndpoint struct {
	endpoint.Endpoint
}

func (ep PostCodeEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, span := tr.Start(ctx, fmt.Sprintf("%s execution", r.URL.Path))
	defer span.End()

	request := PostCodeRequest{}
	if err := endpoint.WithFormRequestParser(ctx, r, &request); err != nil {
		problem.MustWrite(w, err)
		return
	}

	if err := endpoint.WithRequestValidation(ctx, &request); err != nil {
		problem.MustWrite(w, err)
		return
	}

	deviceCode, pkceVerifier, userCode, userCodeWithNoDash, err := createDeviceFlowCodes(ctx)
	if err != nil {
		e := problem.New(http.StatusInternalServerError).WithErr(err)
		problem.MustWrite(w, e)
		return
	}

	cache := map[string]string{
		"client_id":     request.ClientId,
		"scope":         request.Scope,
		"device_code":   deviceCode,
		"pkce_verifier": pkceVerifier, // TODO: This should be encryptet.
	}
	expiresIn := app.Env.CacheDefaultExpiration
	writeToCache(ctx, deviceCode, userCodeWithNoDash, expiresIn, cache)

	response := PostCodeResponse{
		DeviceCode:      deviceCode,
		UserCode:        userCode,
		VerificationUri: app.Env.BaseUrl + "/device",
		ExpiresIn:       expiresIn,
		Interval:        app.Env.PollIntervalInSeconds,
	}

	w.Header().Set("Content-Type", "application/json")

	if err := endpoint.WithResponseValidation(ctx, response); err != nil {
		problem.MustWrite(w, err)
		return
	}

	if err := endpoint.WithJsonResponseWriter(ctx, w, response); err != nil {
		problem.MustWrite(w, err)
		return
	}
}

func writeToCache(ctx context.Context, deviceCode string, userCodeWithNoDash string, expiresIn int, cache map[string]string) {
	_, unitOfWork := tr.Start(ctx, "Store in cache")
	defer unitOfWork.End()

	// Rely on the cache to remove entries upon expire to get the codes to expire.

	// Store without the hyphen
	app.Env.Cache.Set(userCodeWithNoDash, cache, time.Second*time.Duration(expiresIn))

	// Add a placeholder entry with the device code so that the token route knows the request is pending
	app.Env.Cache.Set(deviceCode, map[string]string{
		"iat":    strconv.FormatInt(time.Now().UnixNano(), 10),
		"status": "pending",
	}, time.Second*time.Duration(expiresIn))
}

func createDeviceFlowCodes(ctx context.Context) (deviceCode string, pkceVerifier string, userCode string, userCodeWithNoDash string, err error) {
	_, unitOfWork := tr.Start(ctx, "Create device code, pkce verifier and user code")
	defer unitOfWork.End()

	// Generate a verification code and cache it along with the other values in the request.
	_deviceCodeInBytes, err := endpoint.GenerateRandomBytes(32)
	if err != nil {
		return "", "", "", "", err
	}
	deviceCode = hex.EncodeToString(_deviceCodeInBytes)

	_pkceVerifierInBytes, err := endpoint.GenerateRandomBytes(32)
	if err != nil {
		return "", "", "", "", err
	}
	pkceVerifier = hex.EncodeToString(_pkceVerifierInBytes)

	// If more entropy in the user code is needed increase the number of characters in the generateRandomString call
	_userCode, err := endpoint.GenerateRandomString(8)
	if err != nil {
		return "", "", "", "", err
	}
	userCode = _userCode[:4] + "-" + _userCode[4:]
	userCodeWithNoDash = _userCode[:4] + _userCode[4:]

	return deviceCode, pkceVerifier, userCode, userCodeWithNoDash, nil
}

func NewPostCodeEndpoint() endpoint.EndpointHandler {
	ep := PostCodeEndpoint{}

	ep.Setup(
		endpoint.WithSpecification(api.Path{
			Summary:     "Generate a device code",
			Description: ``,
			Tags:        OPENAPI_TAGS,

			Request: api.Request{
				Description: ``,
				Schema:      PostCodeRequest{},
			},

			Responses: []api.Response{{
				Description: http.StatusText(http.StatusOK),
				Code:        http.StatusOK,
				Schema:      PostCodeResponse{},
			}, {
				Description: http.StatusText(http.StatusBadRequest),
				Code:        http.StatusBadRequest,
				Schema:      problem.ValidationProblem{}, // TODO fix oas to work with: problem.ValidationError{},
			}},
		}),
	)

	return ep
}
