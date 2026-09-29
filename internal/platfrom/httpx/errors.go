package httpx

import "errors"

type Error struct {
	Status  int
	Code    string
	Message string
	Details map[string]string
	Err     error // wrapped, never serialized
}

func (e *Error) Error() string { return e.Message }
func (e *Error) Unwrap() error { return e.Err }

func ValidationFailed(details map[string]string) *Error {
	return &Error{Status: 400, Code: "validation_failed", Message: "Request validation failed", Details: details}
}
func BadRequest(msg string) *Error { return &Error{Status: 400, Code: "bad_request", Message: msg} }
func Unauthenticated(msg string) *Error {
	return &Error{Status: 401, Code: "unauthenticated", Message: msg}
}
func Forbidden(msg string) *Error { return &Error{Status: 403, Code: "forbidden", Message: msg} }
func NotFound(msg string) *Error  { return &Error{Status: 404, Code: "not_found", Message: msg} }
func Conflict(msg string) *Error  { return &Error{Status: 409, Code: "conflict", Message: msg} }
func AlreadyExists(msg string) *Error {
	return &Error{Status: 409, Code: "already_exists", Message: msg}
}
func Unprocessable(msg string) *Error {
	return &Error{Status: 422, Code: "unprocessable", Message: msg}
}
func Internal(err error) *Error {
	return &Error{Status: 500, Code: "internal_error", Message: "Internal error", Err: err}
}

// As extracts an *Error from err, defaulting to 500.
func As(err error) *Error {
	var e *Error
	if errors.As(err, &e) {
		return e
	}
	return Internal(err)
}
