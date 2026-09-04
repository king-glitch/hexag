package ports

import (
	"net/http"

	"github.com/pkg/errors"
)

var (
	ErrForbidden                    = errors.New("forbidden")
	ErrUnknownTransformerActionType = errors.New("unknown action type in transformer")
)

type Violation struct {
	Code    ServiceErrorCode `json:"code"`
	Message string           `json:"message"`
	Stack   string           `json:"stack,omitempty"`
}

type ServiceError struct {
	StatusCode int                  `json:"-"`
	Message    string               `json:"message"`
	Code       ServiceErrorCode     `json:"code"`
	Stack      string               `json:"stack,omitempty"`
	Violations map[string]Violation `json:"violations,omitempty"`

	cause error
}

func NewServiceError(code ServiceErrorCode, err error) *ServiceError {
	message := "failed to process the request"
	var stack string
	if err != nil {
		message = errors.Cause(err).Error()
		stack = err.Error()
	}

	return &ServiceError{
		Message:    message,
		Code:       code,
		Stack:      stack,
		Violations: make(map[string]Violation),
		cause:      err,
	}
}

func (e *ServiceError) WithStatusCode(code int) *ServiceError {
	e.StatusCode = code
	return e
}

func (e *ServiceError) Wrap(message string) *ServiceError {
	if e.cause != nil {
		e.cause = errors.Wrap(e.cause, message)
	} else {
		e.cause = errors.New(message)
		e.Message = message
	}
	e.Stack = e.cause.Error()

	return e
}

func (e *ServiceError) Unwrap() error {
	if e == nil {
		return nil
	}

	return e.cause
}

func (e *ServiceError) AddError(field string, message string, err error) *ServiceError {
	violation := Violation{
		Code:    e.Code,
		Message: message,
	}

	if err != nil {
		violation.Stack = err.Error()
	}

	e.Violations[field] = violation

	return e
}

func (e *ServiceError) Error() string {
	return e.Message
}

type ServiceErrorCode string

const (
	ServiceErrorCodeInternal     ServiceErrorCode = "Service.Internal.Error"
	ServiceErrorCodeValidation   ServiceErrorCode = "Request.Validation.Invalid"
	ServiceErrorCodeNotFound     ServiceErrorCode = "Service.Resource.NotFound"
	ServiceErrorCodeUnauthorized ServiceErrorCode = "Service.Authentication.Unauthorized"
	ServiceErrorCodeInvalidToken ServiceErrorCode = "Service.Authentication.InvalidToken"
	ServiceErrorCodeConflict     ServiceErrorCode = "Service.Resource.Conflict"
	ServiceErrorCodeForbidden    ServiceErrorCode = "Service.Authorization.Forbidden"
)

func (c ServiceErrorCode) DefaultStatusCode() int {
	switch c {
	case ServiceErrorCodeValidation:
		return http.StatusBadRequest
	case ServiceErrorCodeUnauthorized, ServiceErrorCodeInvalidToken:
		return http.StatusUnauthorized
	case ServiceErrorCodeNotFound:
		return http.StatusNotFound
	case ServiceErrorCodeConflict:
		return http.StatusConflict
	case ServiceErrorCodeForbidden:
		return http.StatusForbidden
	case ServiceErrorCodeInternal:
		return http.StatusInternalServerError
	default:
		return http.StatusBadRequest
	}
}

// sentinelServiceErrorCodes maps a project's domain sentinel errors to their
// HTTP-facing ServiceErrorCode. Framework code only pre-registers the two
// sentinels it defines itself (ErrForbidden, ErrUnknownTransformerActionType);
// every project registers its own ports.Err* values via RegisterSentinel in
// an init(), so this file never needs to know about a specific project's
// domain errors.
var sentinelServiceErrorCodes = map[error]ServiceErrorCode{
	ErrForbidden: ServiceErrorCodeForbidden,
}

// RegisterSentinel maps a project-defined sentinel error to the
// ServiceErrorCode it should resolve to via ServiceErrorCodeFor. Call this
// from an init() in the consuming project, once per sentinel.
func RegisterSentinel(err error, code ServiceErrorCode) {
	sentinelServiceErrorCodes[err] = code
}

func ServiceErrorCodeFor(err error) ServiceErrorCode {
	for sentinel, code := range sentinelServiceErrorCodes {
		if errors.Is(err, sentinel) {
			return code
		}
	}

	return ServiceErrorCodeInternal
}

func NewServiceErrorFromCause(err error, message string) *ServiceError {
	return NewServiceError(ServiceErrorCodeFor(err), errors.Wrap(err, message))
}
