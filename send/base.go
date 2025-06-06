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
// contained in this type should embed a Base value and expose its
// methods without further modification. See sender implementations in
// this package for examples.
//
// Historically Base exposed exported fields, but this made maintaining
// invariants challenging; prefer the accessor methods.
//
// Error handling and formatting are fully pluggable via the
// SetErrorHandler and SetFormatter methods.
//
// Close semantics: Base.Close() ensures the configured close hook runs
// exactly once. Subsequent Close calls return the hook’s original
// error (if any) joined with ErrAlreadyClosed so callers can detect
// double-close situations.
//
// All methods are safe for concurrent use unless otherwise noted.
type Base struct {
	// data exposed/configuration via the interface.
	name     adt.Atomic[string]
	priority adt.Atomic[level.Priority]

	// function literals which allow customizable functionality.
	// they set via the SetErrorHandler/SetFormatter injectors.
	errHandler adt.Atomic[ErrorHandler]
	formatter  adt.Atomic[MessageFormatter]

	// internal methods to support close ops.
	close  adt.Once[error]
	closer adt.Atomic[func() error]
	closed atomic.Bool
}

// ErrAlreadyClosed is a component of the error returned from
// Base.Close() when Close is called more than once.
var ErrAlreadyClosed = errors.New("sender already closed")

// Close calls the closer function if it is defined and it has not
// already been closed. A sender, relying on the Base.Close
// infrastructure is closed once closed and cannot be reopened or
// re-used. Subsequent attempts to close a sender will return an error
// that contains both the original error and ErrAlreadyClosed.
func (b *Base) Close() error {
	if swapped := b.closed.CompareAndSwap(false, true); !swapped {
		return erc.Join(fmt.Errorf("sender %q is already closed: %w", b.Name(), ErrAlreadyClosed), b.doClose())
	}

	return b.doClose()
}

func (b *Base) doClose() error {
	b.close.Set(func() error {
		if closer := b.closer.Get(); closer != nil {
			return closer()
		}
		return nil
	})
	return b.close.Resolve()
}

// Name returns the name of the Sender.
func (b *Base) Name() string { return b.name.Get() }

// SetName allows clients to change the name of the Sender.
func (b *Base) SetName(name string) { b.name.Set(name) }

// SetCloseHook defines the behavior of the Close() method in the Base
// implementation.
//
// The Base implementation ensures that this function is called exactly
// once when the Sender is closed. However, the error returned by this
// function is cached, and part of the return value of subsequent calls
// to Close().
func (b *Base) SetCloseHook(f func() error) { b.closer.Set(f) }

// SetFormatter lets users set the formatting function used to construct log messages.
func (b *Base) SetFormatter(mf MessageFormatter) { b.formatter.Set(mf) }

// GetFormatter returns the formatter, defaulting to using the string
// form of the message if no formatter is configured.
func (b *Base) GetFormatter() MessageFormatter { return b.Format }

// SetErrorHandler configures the error handling function for this Sender.
func (b *Base) SetErrorHandler(eh ErrorHandler) { b.errHandler.Set(eh) }

// GetErrorHandler returns the current error handler or nil if none has been set.
func (b *Base) GetErrorHandler() ErrorHandler { return b.errHandler.Get() }

// HandleError invokes the configured error handler when err is non-nil.
func (b *Base) HandleError(err error) {
	if err == nil {
		return
	}
	if fn := b.errHandler.Get(); fn != nil {
		fn(err)
	}
}

// Format renders a message using the configured formatter or falls back to
// m.String().
func (b *Base) Format(m message.Composer) (string, error) {
	if fn := b.formatter.Get(); fn != nil {
		return fn(m)
	}
	return m.String(), nil
}

// HandleErrorOK calls the error handler when err is non-nil and returns true
// when err is nil and false otherwise. Useful for early-return guard clauses.
func (b *Base) HandleErrorOK(err error) bool {
	if err == nil {
		return true
	}
	if fn := b.errHandler.Get(); fn != nil {
		fn(err)
	}
	return false
}

// SetPriority configures the level threshold for the Sender.
func (b *Base) SetPriority(p level.Priority) { b.priority.Set(p) }

// Priority reports the currently configured level for the Sender.
func (b *Base) Priority() level.Priority { return b.priority.Get() }

// Flush provides a default implementation for senders that don't cache messages locally.
// This default implementation is a noop.
func (b *Base) Flush(ctx context.Context) error { return nil }
