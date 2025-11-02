package types

import "fmt"

type ErrorCode int

const (
	ErrCodeGeneral ErrorCode = iota + 1000
	ErrCodePermission
	ErrCodeNotFound
	ErrCodeInvalidInput
	ErrCodeDISM
	ErrCodeDiskSpace
	ErrCodeNetwork
)

type BuildError struct {
	Code    ErrorCode
	Message string
	Cause   error
	Context map[string]interface{}
}

func (e *BuildError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func NewError(code ErrorCode, message string, cause error) *BuildError {
	return &BuildError{
		Code:    code,
		Message: message,
		Cause:   cause,
		Context: make(map[string]interface{}),
	}
}

func (e *BuildError) WithContext(key string, value interface{}) *BuildError {
	e.Context[key] = value
	return e
}