package httpx

import "errors"

type APIError struct {
	Status  int
	Code    string
	Message string
	Details map[string]string
	Err     error // wrapped, never serialized
}

func (e *APIError) Error() string { return e.Message }
func (e *APIError) Unwrap() error { return e.Err }

func (e *APIError) WithDetails(details map[string]string) *APIError {
	e.Details = details
	return e
}

func ValidationFailed(details map[string]string) *APIError {
	return &APIError{Status: 400, Code: "validation_failed", Message: "Request validation failed", Details: details}
}
func BadRequest(msg string) *APIError {
	return &APIError{Status: 400, Code: "bad_request", Message: msg}
}
func Unauthenticated(msg string) *APIError {
	return &APIError{Status: 401, Code: "unauthenticated", Message: msg}
}
func Forbidden(msg string) *APIError { return &APIError{Status: 403, Code: "forbidden", Message: msg} }
func NotFound(msg string) *APIError  { return &APIError{Status: 404, Code: "not_found", Message: msg} }
func Conflict(msg string) *APIError  { return &APIError{Status: 409, Code: "conflict", Message: msg} }
func AlreadyExists(msg string) *APIError {
	return &APIError{Status: 409, Code: "already_exists", Message: msg}
}
func Unprocessable(msg string) *APIError {
	return &APIError{Status: 422, Code: "unprocessable", Message: msg}
}
func Internal(err error) *APIError {
	return &APIError{Status: 500, Code: "internal_error", Message: "Internal error", Err: err}
}

// As extracts an *APIError from err, defaulting to 500.
func As(err error) *APIError {
	var e *APIError
	if errors.As(err, &e) {
		return e
	}
	return Internal(err)
}
