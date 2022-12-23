package controller

var (
	ErrUnauthorized = newError("Unauthorized")
	ErrBadRequest   = newError("Body invalid")
)

// HTTPError is custom HTTP error for API
type HTTPError struct {
	Message string `json:"message"`
}

func (e *HTTPError) Error() string {
	return e.Message
}

func newError(msg string) *HTTPError {
	return &HTTPError{Message: msg}
}
