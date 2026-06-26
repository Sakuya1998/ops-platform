package service

import (
	"errors"
	"fmt"
)

type ErrorKind string

const (
	ErrInvalidArgument ErrorKind = "invalid_argument"
	ErrNotFound        ErrorKind = "not_found"
	ErrConflict        ErrorKind = "conflict"
	ErrForbidden       ErrorKind = "forbidden"
	ErrUnavailable     ErrorKind = "unavailable"
)

type ServiceError struct {
	Kind    ErrorKind
	Message string
	Err     error
}

func (e *ServiceError) Error() string {
	if e == nil {
		return ""
	}
	if e.Message != "" {
		return e.Message
	}
	return string(e.Kind)
}

func (e *ServiceError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func ErrorKindOf(err error) (ErrorKind, bool) {
	var svcErr *ServiceError
	if errors.As(err, &svcErr) {
		return svcErr.Kind, true
	}
	return "", false
}

func invalidArgument(field string) error {
	return &ServiceError{Kind: ErrInvalidArgument, Message: fmt.Sprintf("invalid %s", field)}
}

func notFound(resource string, err error) error {
	return &ServiceError{Kind: ErrNotFound, Message: fmt.Sprintf("%s not found", resource), Err: err}
}

func unavailable(message string) error {
	return &ServiceError{Kind: ErrUnavailable, Message: message}
}
