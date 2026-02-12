package grip

import (
	"context"
	"testing"

	"github.com/tychoish/fun/stw"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

func TestHasLogger(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() context.Context
		expected    bool
		shouldPanic bool
	}{
		{
			name: "NilContext",
			setup: func() context.Context {
				return nil
			},
			expected:    false,
			shouldPanic: true, // HasLogger panics on nil context
		},
		{
			name: "BackgroundWithoutLogger",
			setup: func() context.Context {
				return context.Background()
			},
			expected:    false,
			shouldPanic: false,
		},
		{
			name: "ContextWithLogger",
			setup: func() context.Context {
				ctx := context.Background()
				return WithLogger(ctx, NewLogger(send.MakeStdOut()))
			},
			expected:    true,
			shouldPanic: false,
		},
		{
			name: "ContextWithCanceledLogger",
			setup: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				ctx = WithLogger(ctx, NewLogger(send.MakeStdOut()))
				cancel()
				return ctx
			},
			expected:    true,
			shouldPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("expected panic but didn't get one")
					}
				}()
			}

			ctx := tt.setup()
			result := HasLogger(ctx)
			if !tt.shouldPanic && result != tt.expected {
				t.Errorf("HasLogger() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestWithContextLogger(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() (context.Context, string)
		shouldPanic bool
		validate    func(*testing.T, context.Context)
	}{
		{
			name: "NilContextWithNewKey",
			setup: func() (context.Context, string) {
				return nil, "test-key"
			},
			shouldPanic: true, // WithContextLogger panics on nil context
			validate:    nil,
		},
		{
			name: "BackgroundContextWithNewKey",
			setup: func() (context.Context, string) {
				return context.Background(), "new-logger"
			},
			shouldPanic: false,
			validate: func(t *testing.T, ctx context.Context) {
				if !HasContextLogger(ctx, "new-logger") {
					t.Error("expected logger to be set")
				}
			},
		},
		{
			name: "ExistingLoggerSameKey",
			setup: func() (context.Context, string) {
				ctx := context.Background()
				ctx = WithContextLogger(ctx, "same-key", NewLogger(send.MakeStdOut()))
				return ctx, "same-key"
			},
			shouldPanic: false,
			validate: func(t *testing.T, ctx context.Context) {
				if !HasContextLogger(ctx, "same-key") {
					t.Error("expected logger to still be set")
				}
			},
		},
		{
			name: "ExistingLoggerDifferentKey",
			setup: func() (context.Context, string) {
				ctx := context.Background()
				ctx = WithContextLogger(ctx, "first-key", NewLogger(send.MakeStdOut()))
				return ctx, "second-key"
			},
			shouldPanic: false, // WithContextLogger allows multiple loggers with different keys
			validate: func(t *testing.T, ctx context.Context) {
				if !HasContextLogger(ctx, "first-key") {
					t.Error("expected first-key logger to still exist")
				}
				if !HasContextLogger(ctx, "second-key") {
					t.Error("expected second-key logger to be set")
				}
			},
		},
		{
			name: "EmptyKeyString",
			setup: func() (context.Context, string) {
				return context.Background(), ""
			},
			shouldPanic: false,
			validate: func(t *testing.T, ctx context.Context) {
				if !HasContextLogger(ctx, "") {
					t.Error("expected logger with empty key to be set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shouldPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Error("expected panic but didn't get one")
					}
				}()
			}

			ctx, key := tt.setup()
			result := WithContextLogger(ctx, key, NewLogger(send.MakeStdOut()))

			if !tt.shouldPanic && tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestLoggerClone(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() Logger
		validate func(*testing.T, Logger, Logger)
	}{
		{
			name: "StandardLoggerClone",
			setup: func() Logger {
				return std
			},
			validate: func(t *testing.T, original, cloned Logger) {
				if stw.IsZero(cloned) {
					t.Fatal("cloned logger should not be nil")
				}
				// Cloned logger should be different instance
				if &original == &cloned {
					t.Error("cloned logger should be different instance")
				}
			},
		},
		{
			name: "CustomLoggerClone",
			setup: func() Logger {
				sender := send.MakeStdOut()
				sender.SetPriority(level.Debug)
				return NewLogger(sender)
			},
			validate: func(t *testing.T, original, cloned Logger) {
				if stw.IsZero(cloned) {
					t.Fatal("cloned logger should not be nil")
				}

				// Clone shares the same sender instance
				if original.Sender() != cloned.Sender() {
					t.Error("cloned logger should share the same sender")
				}

				// But they are independent loggers
				// Changing one shouldn't affect the other
				cloned.SetSender(send.MakeStdOut())
				if original.Sender() == cloned.Sender() {
					t.Error("after SetSender, they should have different senders")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			original := tt.setup()
			cloned := original.Clone()
			tt.validate(t, original, cloned)
		})
	}
}

func TestLoggerSetConverter(t *testing.T) {
	tests := []struct {
		name      string
		setup     func() Logger
		converter message.Converter
		validate  func(*testing.T, Logger)
	}{
		{
			name: "SetCustomConverter",
			setup: func() Logger {
				return NewLogger(send.MakeStdOut())
			},
			converter: message.ConverterFunc(func(any) (message.Composer, bool) {
				return message.MakeString("custom"), true
			}),
			validate: func(t *testing.T, logger Logger) {
				// Test that converter works
				result := logger.Convert("test")
				if result == nil {
					t.Error("converter should return non-nil composer")
				}
				if result.String() != "custom" {
					t.Errorf("expected 'custom', got %q", result.String())
				}
			},
		},
		{
			name: "ReplaceExistingConverter",
			setup: func() Logger {
				logger := NewLogger(send.MakeStdOut())
				logger.SetConverter(message.ConverterFunc(func(any) (message.Composer, bool) {
					return message.MakeString("first"), true
				}))
				return logger
			},
			converter: message.ConverterFunc(func(any) (message.Composer, bool) {
				return message.MakeString("second"), true
			}),
			validate: func(t *testing.T, logger Logger) {
				result := logger.Convert("test")
				if result.String() != "second" {
					t.Errorf("expected 'second', got %q", result.String())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := tt.setup()
			logger.SetConverter(tt.converter)
			tt.validate(t, logger)
		})
	}
}
