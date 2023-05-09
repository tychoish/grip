package send

import (
	"context"
	"sync/atomic"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

// Base provides most of the functionality of the Sender interface,
// except for the Send method, to facilitate writing novel Sender
// implementations. All implementations of the functions
type Base struct {
	// data exposed via the interface and tools to track them
	name     adt.Atomic[string]
	priority adt.Atomic[level.Priority]
	closed   atomic.Bool

	// function literals which allow customizable functionality.
	// they set via the SetErrorHandler/SetFormatter injectors.
	errHandler adt.Atomic[ErrorHandler]
	reset      adt.Atomic[func()]
	closer     adt.Atomic[func() error]
	formatter  adt.Atomic[MessageFormatter]
	converter  adt.Atomic[CustomMessageConverter]
}

// Close calls the closer function if it is defined and it has not already been
// closed.
func (b *Base) Close() error {
	if swapped := b.closed.CompareAndSwap(false, true); !swapped {
		return nil
	}

	if closer := b.closer.Get(); closer != nil {
		if err := closer(); err != nil {
			return err
		}
	}

	return nil
}

// Name returns the name of the Sender.
func (b *Base) Name() string { return b.name.Get() }

// SetName allows clients to change the name of the Sender.
func (b *Base) SetName(name string) { b.name.Set(name); b.doReset() }

func (b *Base) SetResetHook(f func()) { b.reset.Set(f) }

func (b *Base) doReset() {
	if reset := b.reset.Get(); reset != nil {
		reset()
	}
}

func (b *Base) SetCloseHook(f func() error) { b.closer.Set(f) }

// SetFormatter users to set the formatting function used to construct log messages.
func (b *Base) SetFormatter(mf MessageFormatter) { b.formatter.Set(mf); b.doReset() }

// Formatter returns the formatter, defaulting to using the string
// form of the message if no formatter is configured.
func (b *Base) Formatter() MessageFormatter {
	return func(m message.Composer) (string, error) {
		fn := b.formatter.Get()

		if fn == nil {
			return m.String(), nil
		}

		return fn(m)
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

		fn := b.errHandler.Get()
		if fn == nil {
			return
		}

		fn(err, m)
	}
}

func (b *Base) SetConverter(mc CustomMessageConverter) {
	b.converter.Set(mc)
}

func (b *Base) Converter() MessageConverter {
	return func(in any) message.Composer {
		converter := b.converter.Get()
		if converter != nil {
			if out, ok := converter(in); ok {
				return out
			}
		}
		return message.Convert(in)
	}
}

// SetPriority configures the level (default levels and threshold levels)
// for the Sender.
func (b *Base) SetPriority(p level.Priority) { b.priority.Set(p) }

// Level reports the currently configured level for the Sender.
func (b *Base) Priority() level.Priority { return b.priority.Get() }

// Flush provides a default implementation of the Flush method for
// senders that don't cache messages locally.
func (b *Base) Flush(ctx context.Context) error { return nil }
