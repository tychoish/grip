// Error Messages
//
// The error message composers underpin the Catch<> logging messages,
// which allow you to log error messages but let the logging system
// elide logging for nil errors.
package message

import (
	"errors"

	"github.com/tychoish/fun/dt"
)

type errorMessage struct {
	err        error
	ErrorValue string `bson:"error" json:"error" yaml:"error"`
	Base       `bson:"meta" json:"meta" yaml:"meta"`
}

func Noop() Composer {
	m := BuildKV().Composer()
	m.SetOption(OptionSkipCollectInfo, OptionSkipMetadata)
	return m
}

// MakeError returns a Composer, that wraps an error, and is only
// loggable for non-nil errors. The message also implements error
// methods (e.g. Error() string, Is() bool, and Unwrap() error).
func MakeError(err error) Composer {
	m := new(errorMessage)
	m.err = err
	return m
}

func (e *errorMessage) String() string {
	if e.err != nil {
		e.ErrorValue = e.err.Error()
	}
	return e.ErrorValue
}

func (e *errorMessage) Loggable() bool { return e.err != nil }
func (e *errorMessage) Unwrap() error  { return e.err }

func (e *errorMessage) Raw() any {
	e.Collect() // noop based on option

	if e.IncludeMetadata {
		_ = e.String()
		return e
	}

	return struct {
		Error                       string `bson:"error" json:"error" yaml:"error"`
		*dt.OrderedMap[string, any] `bson:",omitempty" json:",omitempty" yaml:",omitempty"`
	}{
		Error:      e.String(),
		OrderedMap: &e.Context,
	}
}

func (e *errorMessage) Error() string     { return e.String() }
func (e *errorMessage) Is(err error) bool { return errors.Is(e.err, err) }
func (e *errorMessage) As(err any) bool   { return errors.As(e.err, err) }
