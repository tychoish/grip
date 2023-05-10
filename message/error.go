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
	Payload struct {
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
	m.Payload.err = err
	return m
}

func (e *errorMessage) String() string {
	if e.Payload.ErrorValue != "" {
		return e.Payload.ErrorValue
	} else if e.Payload.err != nil {
		e.Payload.ErrorValue = e.Payload.err.Error()
	}
	return e.Payload.ErrorValue
}

func (e *errorMessage) Loggable() bool { return e.Payload.err != nil }
func (e *errorMessage) Unwrap() error  { return e.Payload.err }

func (e *errorMessage) Raw() any {
	_ = e.String()

	if e.SkipMetadata {
		return e.Payload
	}
	if !e.SkipCollection {
		e.Collect()
	}

	return e
}

func (e *errorMessage) Error() string     { return e.String() }
func (e *errorMessage) Is(err error) bool { return errors.Is(e.Payload.err, err) }
func (e *errorMessage) As(err any) bool   { return errors.As(e.Payload.err, err) }
