package grip

import (
	"context"

	"github.com/tychoish/grip/send"
)

type ctxKey string

const defaultContextKey ctxKey = "__GRIP_STD_LOGGER"

// WithLogger attaches a Logger instance to the context.
func WithLogger(ctx context.Context, logger Logger) context.Context {
	return WithContextLogger(ctx, string(defaultContextKey), logger)
}

// Context resolves a logger from the given context, and if one does
// not exist (or the context is nil), produces the global Logger
// instance.
func Context(ctx context.Context) Logger { return ContextLogger(ctx, string(defaultContextKey)) }

// WithContextLogger attaches a logger with a specific name to the
// context. Your package should wrap this and use
// constants for logger names. In most cases, WithLogger to set a
// default logger, or even just using the standard Context is
// preferable.
//
// If this logger exists, WitContextLogger is a noop and the existing
// context is returned directly.
func WithContextLogger(ctx context.Context, name string, logger Logger) context.Context {
	if HasContextLogger(ctx, name) {
		return ctx
	}

	return context.WithValue(ctx, ctxKey(name), logger)
}

// ContextLogger produces a logger stored in the context by a given
// name. If such a context is not stored the standard/default logger
// is returned.
func ContextLogger(ctx context.Context, name string) Logger {
	if ctx == nil {
		return std
	}

	val := ctx.Value(ctxKey(name))
	if l, ok := val.(Logger); ok {
		return l
	}
	return std
}

// WithNewContextLogger checks if a logger is configured with a
// specific name in the current context. If this logger exists,
// WithNewContextLogger is a noop; otherwise, it constructs a logger
// with the sender produced by the provided function and attaches it
// to the context returning that context.
//
// The name provided controls the id of the logger in the context, not
// the name of the logger.
func WithNewContextLogger(ctx context.Context, name string, fn func() send.Sender) context.Context {
	if HasContextLogger(ctx, name) {
		return ctx
	}

	return WithContextLogger(ctx, name, NewLogger(fn()))
}

// HasContextLogger checks the provided context to see if a logger
// with the given name is attached to the provided context.
func HasContextLogger(ctx context.Context, name string) bool {
	_, ok := ctx.Value(ctxKey(name)).(Logger)
	return ok
}

// HasLogger returns true when the default context logger is
// attached.
func HasLogger(ctx context.Context) bool { return HasContextLogger(ctx, string(defaultContextKey)) }
