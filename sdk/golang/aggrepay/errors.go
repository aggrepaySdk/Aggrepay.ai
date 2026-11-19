package aggrepay

import (
	"fmt"
	"time"
)

// ValidationError: ошибка до HTTP.
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("aggrepay: invalid %s: %s", e.Field, e.Message)
}

// TransportError: сеть, таймаут, битый ответ, лимит тела.
type TransportError struct {
	Op          string
	Err         error
	HTTPStatus  int
	BodySnippet string
	RetryAfter  time.Duration
}

func (e *TransportError) Error() string {
	if e.HTTPStatus > 0 {
		return fmt.Sprintf("aggrepay: %s: http %d: %v", e.Op, e.HTTPStatus, e.Err)
	}
	return fmt.Sprintf("aggrepay: %s: %v", e.Op, e.Err)
}

func (e *TransportError) Unwrap() error { return e.Err }

// APIError: ответ API (status=error или HTTP >= 400).
type APIError struct {
	Message    string
	Code       string
	HTTPStatus int
	Payload    any
	RetryAfter time.Duration
}

func (e *APIError) Error() string {
	if e.Code != "" {
		return fmt.Sprintf("aggrepay: %s (%s)", e.Message, e.Code)
	}
	return "aggrepay: " + e.Message
}
