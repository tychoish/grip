// Error Messages
//
// The error message composers underpin the Catch<> logging messages,
// which allow you to log error messages but let the logging system
// elide logging for nil errors.
package message

import (
	"errors"
	"fmt"

	"github.com/tychoish/grip/level"
)

type errorMessage struct {
	err        error
	ErrorValue string `bson:"error" json:"error" yaml:"error"`
	Extended   string `bson:"extended,omitempty" json:"extended,omitempty" yaml:"extended,omitempty"`
	Base       `bson:"metadata" json:"metadata" yaml:"metadata"`
}

// NewError takes an error object and returns a Composer
// instance that only renders a loggable message when the error is
// non-nil.
//
// These composers also implement the error interface and the
// pkg/errors.Causer and errors.Unwrapper interface and so can be
// passed as errors and used with existing error-wrapping mechanisms.
func NewError(p level.Priority, err error) Composer {
	m := &errorMessage{err: err}

	_ = m.SetPriority(p)
	return m
}

// MakeError returns an error composer, like NewErrorMessage, but
// without the requirement to specify priority, which you may wish to
// specify directly.
func MakeError(err error) Composer {
	return &errorMessage{err: err}
}

func (e *errorMessage) String() string {
	if e.err == nil {
		return ""
	}
	e.ErrorValue = e.err.Error()
	return e.ErrorValue
}

func (e *errorMessage) Loggable() bool { return e.err != nil }
func (e *errorMessage) Unwrap() error  { return e.err }

func (e *errorMessage) Raw() any {
	e.Collect()
	_ = e.String()

	extended := fmt.Sprintf("%+v", e.err)
	if extended != e.ErrorValue {
		e.Extended = extended
	}

	return e
}

func unwrapCause(err error) error {
	// stolen from pkg/errors
	type causer interface {
		Cause() error
	}

	for err != nil {
		cause, ok := err.(causer)
		if !ok {
			break
		}
		err = cause.Cause()
	}
	return err
}

func (e *errorMessage) Error() string     { return e.String() }
func (e *errorMessage) Is(err error) bool { return errors.Is(e.err, err) }
func (e *errorMessage) As(err any) bool   { return errors.As(e.err, err) }
