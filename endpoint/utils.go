package endpoint

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"reflect"
	"strings"

	"gopkg.in/yaml.v2"

	"github.com/gorilla/schema"

	"github.com/wraix/device-flow-proxy/endpoint/problem"

	"github.com/hetiansu5/urlquery"
	"go.opentelemetry.io/otel"

	"github.com/go-playground/locales/en"
	ut "github.com/go-playground/universal-translator"
	"github.com/go-playground/validator/v10"
	en_translations "github.com/go-playground/validator/v10/translations/en"
)

var (
	validate *validator.Validate
	locale   string
	trans    ut.Translator
)

func init() {
	validate = validator.New()

	validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]

		if name == "-" {
			return ""
		}

		return name
	})

	locale = "en"

	enTranslator := en.New()
	uni := ut.New(enTranslator, enTranslator)

	trans, _ = uni.GetTranslator(locale)
	en_translations.RegisterDefaultTranslations(validate, trans)
}

func WithRequestValidation(ctx context.Context, i interface{}) error {
	tr := otel.Tracer("request")
	ctx, span := tr.Start(ctx, "request-validation")
	defer span.End()

	err := validate.Struct(i)
	if err == nil {
		// No validation error, continue
		return nil
	}

	prob := problem.NewValidationProblem(http.StatusBadRequest)
	for _, verr := range err.(validator.ValidationErrors) {
		prob.Add(verr.Field(), verr.Translate(trans))
	}

	return prob
}

func WithResponseValidation(ctx context.Context, i interface{}) error {
	tr := otel.Tracer("request")
	ctx, span := tr.Start(ctx, "response-validation")
	defer span.End()

	err := validate.Struct(i)
	if err == nil {
		// No validation error, continue
		return nil
	}

	prob := problem.NewValidationProblem(http.StatusInternalServerError)
	for _, verr := range err.(validator.ValidationErrors) {
		prob.Add(verr.Field(), verr.Translate(trans))
	}

	return prob
}

func WithJsonRequestParser(ctx context.Context, r *http.Request, i interface{}) error {
	tr := otel.Tracer("request")
	ctx, span := tr.Start(ctx, "request-parser")
	defer span.End()

	// Try to decode the request body into the struct. If there is an error,
	// respond to the client with the error message and a 400 status code.
	err := json.NewDecoder(r.Body).Decode(&i)
	if err != nil {
		return problem.New(http.StatusBadRequest).WithErr(err)
	}

	return nil
}

func WithRequestQueryParser(ctx context.Context, r *http.Request, i interface{}) error {
	tr := otel.Tracer("request")
	ctx, span := tr.Start(ctx, "request-parser-query")
	defer span.End()

	err := urlquery.Unmarshal([]byte(r.URL.RawQuery), i)
	if err != nil {
		return problem.New(http.StatusBadRequest).WithErr(err)
	}

	return nil
}

func WithFormRequestParser(ctx context.Context, r *http.Request, i interface{}) error {
	tr := otel.Tracer("request")
	ctx, span := tr.Start(ctx, "request-parser-form")
	defer span.End()

	err := r.ParseForm()
	if err != nil {
		return problem.New(http.StatusInternalServerError).WithErr(err)
	}

	// r.PostForm is a map of our POST form values
	var decoder = schema.NewDecoder()
	decoder.SetAliasTag("form")
	err = decoder.Decode(i, r.PostForm)
	if err != nil {
		return problem.New(http.StatusBadRequest).WithErr(err)
	}

	return nil
}

func WithJsonResponseWriter(ctx context.Context, w http.ResponseWriter, i interface{}) error {
	tr := otel.Tracer("request")
	ctx, span := tr.Start(ctx, "json-response-writer")
	defer span.End()

	d, err := json.Marshal(i)
	if err != nil {
		return problem.New(http.StatusInternalServerError).WithErr(err)
	}
	w.Write(d)

	return nil
}

func WithYamlResponseWriter(ctx context.Context, w http.ResponseWriter, i interface{}) error {
	tr := otel.Tracer("request")
	ctx, span := tr.Start(ctx, "yaml-response-writer")
	defer span.End()

	d, err := yaml.Marshal(i)
	if err != nil {
		return problem.New(http.StatusInternalServerError).WithErr(err)
	}
	w.Write(d)

	return nil
}

func WithResponseWriter(ctx context.Context, w http.ResponseWriter, tp string, i interface{}) error {
	switch tp {
	case "json":
		return WithJsonResponseWriter(ctx, w, i)
	case "yaml":
		return WithYamlResponseWriter(ctx, w, i)
	default:
		panic(fmt.Sprintf("Unknown response type given, %s", tp))
	}
}

// Shamelessly stolen from https://gist.github.com/dopey/c69559607800d2f2f90b1b1ed4e550fb

// GenerateRandomBytes returns securely generated random bytes.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	// Note that err == nil only if we read len(b) bytes.
	if err != nil {
		return nil, err
	}

	return b, nil
}

// GenerateRandomString returns a securely generated random string.
// It will return an error if the system's secure random
// number generator fails to function correctly, in which
// case the caller should not continue.
func GenerateRandomString(n int) (string, error) {
	const letters = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	ret := make([]byte, n)
	for i := 0; i < n; i++ {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			return "", err
		}
		ret[i] = letters[num.Int64()]
	}

	return string(ret), nil
}
