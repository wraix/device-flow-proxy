package browser

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/wraix/device-flow-proxy/app"
	"github.com/wraix/device-flow-proxy/endpoint"
	"github.com/wraix/device-flow-proxy/endpoint/problem"

  "github.com/charmixer/oas/api"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/contrib/propagators/jaeger"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
)

type GetRedirectRequest struct {
	Code  string `query:"code" validate:"required"`
	State string `query:"state" validate:"required"`
}

type GetRedirectEndpoint struct {
	endpoint.Endpoint
}

type Token struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
	TokenType   string `json:"token_type"`
}

type SignedInData struct {
	PageTitle string
}

type ErrorPage struct {
	PageTitle        string
	Error            string
	ErrorDescription string
}

type tracingTransport struct {
	originalTransport http.RoundTripper
}

func (c *tracingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// Convert from OTEL to Jaeger trace context
	prop := jaeger.Jaeger{}
	prop.Inject(r.Context(), propagation.HeaderCarrier(r.Header))

	return c.originalTransport.RoundTrip(r)
}

var client http.Client

func init() {
	timeout := time.Duration(5 * time.Second)
	client = http.Client{
		Timeout: timeout,
		Transport: &tracingTransport{
			originalTransport: otelhttp.NewTransport(&http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}),
		},
	}

}

func (ep GetRedirectEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tr := otel.Tracer("request")
	ctx, span := tr.Start(ctx, fmt.Sprintf("%s execution", r.URL.Path))
	defer span.End()

	request := GetRedirectRequest{}
	if err := endpoint.WithRequestQueryParser(ctx, r, &request); err != nil {
		problem.MustWrite(w, err)
		return
	}

	if err := endpoint.WithRequestValidation(ctx, &request); err != nil {
		problem.MustWrite(w, err)
		return
	}

	// Check that the state parameter matches
	cacheStateKey := "state:" + request.State
	_cachedState, found := app.Env.Cache.Get(cacheStateKey)
	if !found {
		prob := problem.New(http.StatusBadRequest).WithDetail("The state parameter is invalid")
		problem.MustWrite(w, prob)
		return
	}
	cachedState := _cachedState.(map[string]string)

	// Look up the info from the user code provided in the state parameter
	_cache, found := app.Env.Cache.Get(cachedState["user_code"])
	if !found {
		prob := problem.New(http.StatusInternalServerError).WithDetail("No user_code found i cached state")
		problem.MustWrite(w, prob)
		return
	}
	cache := _cache.(map[string]string)

	// Exchange the authorization code for an access token

	// Query params
	q := url.Values{}

	q.Add("grant_type", "authorization_code")
	q.Add("code", request.Code)
	q.Add("redirect_uri", app.Env.BaseUrl+"/auth/redirect")
	q.Add("client_id", cache["client_id"])
	q.Add("code_verifier", cache["pkce_verifier"])

	if cache["client_secret"] != "" {
		q.Add("client_secret", cache["client_secret"])
	}

	tokenRequest, err := http.NewRequestWithContext(ctx, "POST", app.Env.TokenEndpoint, bytes.NewBuffer([]byte(q.Encode())))
	if err != nil {
		prob := problem.New(http.StatusInternalServerError).WithErr(err)
		problem.MustWrite(w, prob)
		return
	}
	//	tokenRequest.Header.Set("Content-Type", "application/json")
	tokenRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(tokenRequest)
	if err != nil {
		prob := problem.New(http.StatusInternalServerError).WithErr(err)
		problem.MustWrite(w, prob)
		return
	}
	defer resp.Body.Close()

	tokenResponse, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		prob := problem.New(http.StatusInternalServerError).WithErr(err)
		problem.MustWrite(w, prob)
		return
	}

	token := Token{}
	err = json.Unmarshal(tokenResponse, &token)
	if err != nil {
		prob := problem.New(http.StatusInternalServerError).WithErr(err)
		problem.MustWrite(w, prob)
		return
	}

	if token.AccessToken == "" {
		app.Env.Cache.Delete(cachedState["user_code"])
		app.Env.Cache.Delete(cache["device_code"])

		w.WriteHeader(http.StatusBadRequest)

		tmpl := template.Must(template.ParseFiles("./endpoint/browser/error.html"))
		data := ErrorPage{
			PageTitle:        "Error",
			Error:            "Error Logging In",
			ErrorDescription: "There was an error getting an access token from the service <p><pre>" + string(tokenResponse) + "</pre></p>",
		}
		tmpl.Execute(w, data)
		return
	}

	// Stash the access token in the cache and display a success message
	s := map[string]string{
		"status":         "complete",
		"token_response": string(tokenResponse),
	}
	app.Env.Cache.Set(cache["device_code"], s, 120*time.Second)
	app.Env.Cache.Delete(cachedState["user_code"])

	tmpl := template.Must(template.ParseFiles("./endpoint/browser/signed-in.html"))
	data := SignedInData{
		PageTitle: "Signed In",
	}
	tmpl.Execute(w, data)
}

func NewGetRedirectEndpoint() endpoint.EndpointHandler {
	ep := GetRedirectEndpoint{}

	ep.Setup(
		endpoint.WithSpecification(api.Path{
			Summary:     "Redirect",
			Description: ``,
			Tags:        OPENAPI_TAGS,

			Request: api.Request{
				Description: ``,
				Schema:      GetRedirectRequest{},
			},
		}),
	)

	return ep
}
