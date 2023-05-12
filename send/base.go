package send

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

// Base provides most of the functionality of the Sender interface,
// except for the Send method, to facilitate writing novel Sender
// implementations. All implementations of the functions
type Base struct {
	// data exposed/configuration via the interface.
	name     adt.Atomic[string]
	priority adt.Atomic[level.Priority]

	// function literals which allow customizable functionality.
	// they set via the SetErrorHandler/SetFormatter injectors.
	errHandler adt.Atomic[ErrorHandler]
	formatter  adt.Atomic[MessageFormatter]

	// internal methods to support close ops. close as
	close  adt.Once[error]
	closer adt.Atomic[func() error]
	closed atomic.Bool
}

// ErrAlreadyClosed is a component of the err returend from
// Base.Close() when Close is called more than once.
var ErrAlreadyClosed = errors.New("sender already closed")

// Close calls the closer function if it is defined and it has not
// already been closed. A sender, relying on the Base.Close
// infrastructure can is closed once closed cannot be reopened or
// re-used. Subsequent attempts to close a sender will return an error
// object that contains both the original error and
// an error rooted in ErrAlreadyClosed.
func (b *Base) Close() error {
	if swapped := b.closed.CompareAndSwap(false, true); !swapped {
		return erc.Merge(fmt.Errorf("sender %q is already closed: %w", b.Name(), ErrAlreadyClosed), b.doClose())
	}

	return b.doClose()
}

func (b *Base) doClose() error {
	return b.close.Do(func() error {
		if closer := b.closer.Get(); closer != nil {
			return closer()
		}
		return nil
	})
}

// Name returns the name of the Sender.
func (b *Base) Name() string { return b.name.Get() }

// SetName allows clients to change the name of the Sender.
//
// Previously this also called the ResetHook, but Sender implementors.
// should now do this manually, if/when needed.
func (b *Base) SetName(name string) { b.name.Set(name) }

// SetCloseHook defines the behavior of the Close() method in the Base
// implementation.
//
// The Base implementation ensures that this function is called
// exactly once when the Sender is closed. However, the error returned
// by this function is cached, and part of the return value of
// subsequent calls to Close().
func (b *Base) SetCloseHook(f func() error) { b.closer.Set(f) }

// SetFormatter users to set the formatting function used to construct log messages.
func (b *Base) SetFormatter(mf MessageFormatter) { b.formatter.Set(mf) }

// Formatter returns the formatter, defaulting to using the string
// form of the message if no formatter is configured.
func (b *Base) Formatter() MessageFormatter {
	return func(m message.Composer) (string, error) {
		if fn := b.formatter.Get(); fn != nil {
			return fn(m)
		}
		return m.String(), nil
	}
}

// SetErrorHandler configures the error handling function for this Sender.
func (b *Base) SetErrorHandler(eh ErrorHandler) { b.errHandler.Set(eh) }

// ErrorHandler returns an error handling functioncalls the error
// handler, and is a wrapper around the embedded ErrorHandler
// function.
func (b *Base) ErrorHandler() ErrorHandler {
	return func(err error, m message.Composer) {
		if err == nil {
			return
		}

		if fn := b.errHandler.Get(); fn != nil {
			fn(err, m)
		}
	}
}

// SetPriority configures the level (default levels and threshold levels)
// for the Sender.
func (b *Base) SetPriority(p level.Priority) { b.priority.Set(p) }

// Level reports the currently configured level for the Sender.
func (b *Base) Priority() level.Priority { return b.priority.Get() }

// Flush provides a default implementation of the Flush method for
// senders that don't cache messages locally. This is a noop by
// default.
func (b *Base) Flush(ctx context.Context) error { return nil }
