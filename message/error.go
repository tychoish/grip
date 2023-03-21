// Error Messages
//
// The error message composers underpin the Catch<> logging messages,
// which allow you to log error messages but let the logging system
// elide logging for nil errors.
package message

import (
	"errors"
)

type errorMessage struct {
	err        error
	ErrorValue string `bson:"error" json:"error" yaml:"error"`
	Base       `bson:"metadata" json:"metadata" yaml:"metadata"`
}

// MakeError returns an error composer, like NewErrorMessage, but
// without the requirement to specify priority, which you may wish to
// specify directly.
func MakeError(err error) Composer {
	return &errorMessage{err: err}
}

func (e *errorMessage) String() string {
	if e.ErrorValue != "" {
		return e.ErrorValue
	} else if e.err != nil {
		e.ErrorValue = e.err.Error()
	}
	return e.ErrorValue
}

func (e *errorMessage) Loggable() bool { return e.err != nil }
func (e *errorMessage) Unwrap() error  { return e.err }

func (e *errorMessage) Raw() any {
	e.Collect()
	_ = e.String()

	return e
}

func (e *errorMessage) Error() string     { return e.String() }
func (e *errorMessage) Is(err error) bool { return errors.Is(e.err, err) }
func (e *errorMessage) As(err any) bool   { return errors.As(e.err, err) }
