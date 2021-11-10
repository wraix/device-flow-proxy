package problem

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

// ContentProblemDetails is the correct MIME type to use when returning a
// problem details object as JSON.
const ContentProblemDetails = "application/problem+json"

// ProblemDetails provides a standard encapsulation for problems encountered
// in web applications and REST APIs.
type ProblemDetails struct {
	Status       int    `json:"status,omitempty" oas-desc:"HTTP status code for the response"`
	Title        string `json:"title,omitempty" oas-desc:"Title of the problem"`
	Detail       string `json:"detail,omitempty" oas-desc:"Detailed description of the problem"`
	Type         string `json:"type,omitempty" oas-desc:"Type of problem"`
	Instance     string `json:"instance,omitempty" oas-desc:"Instance affected by the problem"`
	wrappedError error
}

// HTTPError is the minimal interface needed to be able to Write a problem,
// defined so that ProblemDetails can be encapsulated and expanded as needed.
type HTTPError interface {
	GetStatus() int
}

// GetStatus implements the HTTPError interface
func (pd ProblemDetails) GetStatus() int {
	return pd.Status
}

// New implements the error interface, so ProblemDetails objects can be used
// as regular error return values.
func (pd ProblemDetails) Error() string {
	return pd.Title
}

// Unwrap implements the Go 1.13+ unwrapping interface for errors.
func (pd ProblemDetails) Unwrap() error {
	return pd.wrappedError
}

const rfcBase = "https://tools.ietf.org/html/"

//// Fluent API

// New returns a ProblemDetails error object with the given HTTP status code.
func New(status int) *ProblemDetails {
	return &ProblemDetails{
		Status: status,
		Title:  http.StatusText(status),
		Type:   "https://httpstatuses.com/" + strconv.Itoa(status),
	}
}

// Errorf uses fmt.Errorf to add a detail message to the ProblemDetails object.
// It supports the %w verb.
func (pd *ProblemDetails) Errorf(fmtstr string, args ...interface{}) *ProblemDetails {
	err := fmt.Errorf(fmtstr, args...)
	pd.wrappedError = errors.Unwrap(err)
	pd.Detail = err.Error()
	return pd
}

// WithDetail adds the supplied detail message to the problem details.
func (pd *ProblemDetails) WithDetail(msg string) *ProblemDetails {
	pd.Detail = msg
	return pd
}

// WithErr adds an error value as a wrapped error. If the error detail message
// is currently blank, it is initialized from the error's New() message.
func (pd *ProblemDetails) WithErr(err error) *ProblemDetails {
	pd.wrappedError = err
	if pd.Detail == "" {
		pd.Detail = err.Error()
	}
	return pd
}

// rawWrite implements writing anything which satisfies HTTPError, as a JSON
// problem details object.
func rawWrite(w http.ResponseWriter, obj HTTPError) error {
	w.Header().Set(http.CanonicalHeaderKey("Content-Type"), ContentProblemDetails)
	w.WriteHeader(obj.GetStatus())
	return json.NewEncoder(w).Encode(obj)
}

// Write sets the HTTP response code from the ProblemDetails and then sends the
// entire object as JSON.
func (pd *ProblemDetails) Write(w http.ResponseWriter) error {
	return rawWrite(w, pd)
}

//// Non-fluent API

// Write writes the supplied error if it's a ProblemDetails, returning nil;
// otherwise it returns the error untouched for the caller to handle.
func Write(w http.ResponseWriter, err error) error {
	if err == nil {
		return nil
	}
	switch r := err.(type) {
	/* case ProblemDetails:
	return r.Write(w) */
	case HTTPError:
		return rawWrite(w, r)
	case error:
		return r
	default:
		return fmt.Errorf("can't write non-error type %T", err)
	}
}

// MustWrite is like Write, but if the error isn't a ProblemDetails object
// the error is written as a new problem details object, HTTP Internal Server
// Error.
func MustWrite(w http.ResponseWriter, err error) error {
	err = Write(w, err)
	if err != nil {
		return New(http.StatusInternalServerError).WithErr(err).Write(w)
	}
	return nil
}

// Errorf is used like fmt.Errorf to create and return errors. It takes an
// extra first argument of the HTTP status to use.
func Errorf(status int, fmtstr string, args ...interface{}) *ProblemDetails {
	return New(status).Errorf(fmtstr, args...)
}

// Error is used just like http.Error to create and immediately issue an error.
func Error(w http.ResponseWriter, msg string, status int) error {
	return New(status).WithDetail(msg).Write(w)
}

// ValidationProblem is an example of extending the ProblemDetails structure
// as per the form validation example in section 3 of RFC 7807, to support
// reporting of server-side data validation errors.
type ValidationProblem struct {
	ProblemDetails
	ValidationErrors []ValidationError `json:"invalid-params,omitempty" oas-desc:"Validation errors"`
}

// ValidationError indicates a server-side validation error for data submitted
// as JSON or via a web form.
type ValidationError struct {
	FieldName string `json:"name" oas-desc:"Name of the field with failed validation"`
	Error     string `json:"reason" oas-desc:"Description of the error"`
}

// NewValidationProblem creates an object to represent a server-side validation error.
func NewValidationProblem(status int) *ValidationProblem {
	return &ValidationProblem{
		ProblemDetails:   ProblemDetails{Status: status, Detail: http.StatusText(status)},
		ValidationErrors: []ValidationError{},
	}
}

// Add adds a validation error message for the specified field to the ValidationProblem.
func (vp *ValidationProblem) Add(field string, errmsg string) {
	ve := ValidationError{field, errmsg}
	vp.ValidationErrors = append(vp.ValidationErrors, ve)
}
