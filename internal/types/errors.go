package types

import (
	"errors"
	"fmt"
)

// AppError mirrors com.buy.group.easy.types.exception.AppException. It carries a
// business response code and message and satisfies the builtin error interface so
// it can flow through the RPC wire and be translated back at the gateway.
type AppError struct {
	Code string
	Info string
	// Cause optionally wraps an underlying error for logging; it is not part of
	// the wire contract.
	Cause error
}

// NewAppError builds an AppError from a ResponseCode.
func NewAppError(rc ResponseCode) *AppError {
	return &AppError{Code: rc.Code, Info: rc.Info}
}

// NewAppErrorf builds an AppError with a custom code and formatted message.
func NewAppErrorf(code, format string, args ...any) *AppError {
	return &AppError{Code: code, Info: fmt.Sprintf(format, args...)}
}

// WrapAppError attaches a cause to an existing AppError for logging.
func WrapAppError(rc ResponseCode, cause error) *AppError {
	return &AppError{Code: rc.Code, Info: rc.Info, Cause: cause}
}

func (e *AppError) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause != nil {
		return fmt.Sprintf("AppException{code='%s', info='%s', cause='%v'}", e.Code, e.Info, e.Cause)
	}
	return fmt.Sprintf("AppException{code='%s', info='%s'}", e.Code, e.Info)
}

func (e *AppError) Unwrap() error { return e.Cause }

// AsAppError extracts an *AppError from an error chain, returning nil when the
// error is not a business AppError.
//
// Defect fix: the lookup goes through errors.As instead of a direct type
// assertion, so an AppError wrapped with fmt.Errorf("%w", ...) is still
// recognized and its business code survives the trip across the RPC wire.
func AsAppError(err error) *AppError {
	var ae *AppError
	if errors.As(err, &ae) {
		return ae
	}
	return nil
}
