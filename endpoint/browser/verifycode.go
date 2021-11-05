package browser

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/wraix/device-flow-proxy/app"
	"github.com/wraix/device-flow-proxy/endpoint"
	"github.com/wraix/device-flow-proxy/endpoint/problem"

	"go.opentelemetry.io/otel"
)

type GetVerifyCodeRequest struct {
	Code string `query:"code" validate:"required"`
}

type GetVerifyCodeEndpoint struct {
	endpoint.Endpoint
}

type VerifyCodePageData struct {
	Code       string
	FormAction string
	PageTitle  string
}

func (ep GetVerifyCodeEndpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tr := otel.Tracer("request")
	ctx, span := tr.Start(ctx, fmt.Sprintf("%s execution", r.URL.Path))
	defer span.End()

	request := GetVerifyCodeRequest{}
	if err := endpoint.WithRequestQueryParser(ctx, r, &request); err != nil {
		problem.MustWrite(w, err)
		return
	}

	if err := endpoint.WithRequestValidation(ctx, &request); err != nil {
		problem.MustWrite(w, err)
		return
	}

	// 	Remove hyphens and convert to uppercase to make it easier for users to enter the code
	userCode := strings.ToUpper(strings.ReplaceAll(request.Code, "-", ""))

	_cache, found := app.Env.Cache.Get(userCode)
	if !found {
		prob := problem.New(http.StatusBadRequest).WithDetail("Code not found")
		problem.MustWrite(w, prob)
	}
	cache := _cache.(map[string]string)

	_state, err := endpoint.GenerateRandomBytes(16)
	if err != nil {
		prob := problem.New(http.StatusBadRequest).WithErr(err)
		problem.MustWrite(w, prob)
	}
	state := hex.EncodeToString(_state)

	expiresIn := app.Env.CacheDefaultExpiration

	obj := map[string]string{
		"user_code": userCode,
		"iat":       strconv.FormatInt(time.Now().UnixNano(), 10),
	}

	app.Env.Cache.Set("state:"+state, obj, time.Second*time.Duration(expiresIn))

	pkceVerifier := CodeVerifier{
		Value: cache["pkce_verifier"],
	}
	pkceChallenge := pkceVerifier.CodeChallengeS256() // base64_urlencode(hash('sha256', $cache->pkce_verifier, true))

	base, err := url.Parse(app.Env.AuthorizationEndpoint)
	if err != nil {
		prob := problem.New(http.StatusInternalServerError).WithErr(err)
		problem.MustWrite(w, prob)
	}

	// Query params
	q := url.Values{}

	q.Add("response_type", "code")
	q.Add("client_id", cache["client_id"])
	q.Add("redirect_uri", app.Env.BaseUrl+"/auth/redirect")
	q.Add("state", state)
	q.Add("code_challenge", pkceChallenge)
	q.Add("code_challenge_method", "S256")

	if cache["scope"] != "" {
		q.Add("scope", cache["scope"])
	}

	base.RawQuery = q.Encode()

	authUrl := base.String()

	http.Redirect(w, r, authUrl, http.StatusFound)
}

func NewGetVerifyCodeEndpoint() endpoint.EndpointHandler {
	ep := GetVerifyCodeEndpoint{}

	return ep
}

type CodeVerifier struct {
	Value string
}

func (v *CodeVerifier) CodeChallengeS256() string {
	h := sha256.New()
	h.Write([]byte(v.Value))
	return encode(h.Sum(nil))
}

func encode(msg []byte) string {
	encoded := base64.StdEncoding.EncodeToString(msg)
	encoded = strings.Replace(encoded, "+", "-", -1)
	encoded = strings.Replace(encoded, "/", "_", -1)
	encoded = strings.Replace(encoded, "=", "", -1)
	return encoded
}
