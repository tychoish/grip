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
	payload struct {
		err        error
		ErrorValue string `bson:"error" json:"error" yaml:"error"`
	}
	Base `bson:"meta" json:"meta" yaml:"meta"`
}

// MakeError returns a Composer, that wraps an error, and is only
// loggable for non-nil errors. The message also implements error
// methods (e.g. Error() string, Is() bool, and Unwrap() error).
func MakeError(err error) Composer {
	m := new(errorMessage)
	m.payload.err = err
	return m
}

func (e *errorMessage) String() string {
	if e.payload.ErrorValue != "" {
		return e.payload.ErrorValue
	} else if e.payload.err != nil {
		e.payload.ErrorValue = e.payload.err.Error()
	}
	return e.payload.ErrorValue
}

func (e *errorMessage) Loggable() bool { return e.payload.err != nil }
func (e *errorMessage) Unwrap() error  { return e.payload.err }

func (e *errorMessage) Raw() any {
	_ = e.String()

	if e.SkipMetadata {
		return e.payload
	}

	e.Collect()

	return e
}

func (e *errorMessage) Error() string     { return e.String() }
func (e *errorMessage) Is(err error) bool { return errors.Is(e.payload.err, err) }
func (e *errorMessage) As(err any) bool   { return errors.As(e.payload.err, err) }
