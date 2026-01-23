package message

import (
	"errors"
	"fmt"
	"io"
	"sync/atomic"
	"testing"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/assert/check"
	"github.com/tychoish/fun/testt"
)

func testConverter(t *testing.T, shouldCall bool) ConverterFunc {
	t.Helper()

	return func(m any) (Composer, bool) {
		t.Helper()
		return Convert(m), false
	}
}

func mockSender(t *testing.T, expected int) func(Composer) {
	t.Helper()
	count := &atomic.Int64{}
	t.Cleanup(func() {
		t.Helper()
		check.Equal(t, expected, int(count.Load()))
	})
	return func(Composer) { count.Add(1) }
}

func mockSenderMessage(t *testing.T, expected string) func(Composer) {
	t.Helper()
	count := &atomic.Int64{}
	value := &adt.Atomic[string]{}
	t.Cleanup(func() {
		t.Helper()
		check.Equal(t, int(count.Load()), 1)
		check.Equal(t, expected, value.Get())
	})
	return func(c Composer) {
		t.Helper()
		count.Add(1)
		value.Set(c.String())
		testt.Logf(t, "%d> %T", count.Load(), c)
	}
}

func TestBuilder(t *testing.T) {
	t.Run("Nil", func(t *testing.T) {
		b := NewBuilder(nil, nil)
		b.Send()
		check.Error(t, b.catcher.Resolve())
	})
	t.Run("ErrorsBecomeMessages", func(t *testing.T) {
		b := NewBuilder(mockSenderMessage(t, "kip"), testConverter(t, true))
		b.catcher.Push(errors.New("kip"))
		b.Send()
	})
	t.Run("ErrorsAreAnnotated", func(t *testing.T) {
		b := NewBuilder(mockSenderMessage(t, "bad cat [error='kip']"), testConverter(t, true)).Ln("bad cat").SetGroup(true)
		b.catcher.Push(errors.New("kip"))
		b.Send()
	})
	t.Run("SetLevelInvalidIsAnError", func(t *testing.T) {
		NewBuilder(mockSender(t, 1), testConverter(t, true)).Ln("msg").Level(0).Send()
		NewBuilder(mockSender(t, 1), testConverter(t, true)).Ln("msg").Level(200).Send()
		NewBuilder(mockSender(t, 1), testConverter(t, true)).Level(0).Send()
		NewBuilder(mockSender(t, 1), testConverter(t, true)).Level(200).Send()
	})
	t.Run("SingleString", func(t *testing.T) {
		NewBuilder(mockSender(t, 1), testConverter(t, true)).Ln("hello world").Send()
	})
	t.Run("Double", func(t *testing.T) {
		NewBuilder(mockSender(t, 2), testConverter(t, true)).Ln("hello").Ln("world").Send()
	})
	t.Run("DoubleGroup", func(t *testing.T) {
		NewBuilder(mockSender(t, 1), testConverter(t, true)).Ln("hello").Ln("world").Group().Send()
	})
	t.Run("DoubleGroupCallsAreSequential", func(t *testing.T) {
		NewBuilder(mockSender(t, 2), testConverter(t, true)).Ln("hello").Ln("world").Group().Ungroup().Send()
		NewBuilder(mockSender(t, 2), testConverter(t, true)).Ln("hello").Ln("world").Group().Group().Ungroup().Send()
		NewBuilder(mockSender(t, 1), testConverter(t, true)).Ln("hello").Ln("world").Ungroup().Group().Send()
		NewBuilder(mockSender(t, 1), testConverter(t, true)).Ln("hello").Ln("world").Ungroup().Group().Group().Send()
	})
	t.Run("SetGroup", func(t *testing.T) {
		NewBuilder(mockSender(t, 2), testConverter(t, true)).Ln("hello").Ln("world").Group().SetGroup(false).Send()
		NewBuilder(mockSender(t, 1), testConverter(t, true)).Ln("hello").Ln("world").Ungroup().SetGroup(true).Send()
	})

	t.Run("Values", func(t *testing.T) {
		t.Run("SingleStringValue", func(t *testing.T) {
			NewBuilder(mockSenderMessage(t, "1hello world"), testConverter(t, true)).Ln("1hello world").Send()
		})
		t.Run("SingleFormat", func(t *testing.T) {
			NewBuilder(mockSenderMessage(t, "hello 543 world"), testConverter(t, true)).F("hello %d world", 543).Send()
		})
		t.Run("SingleLines", func(t *testing.T) {
			NewBuilder(mockSenderMessage(t, "hello world 543"), testConverter(t, true)).Lns("hello", "world", 543).Send()
		})
		t.Run("SingleError", func(t *testing.T) {
			NewBuilder(mockSenderMessage(t, "kip: EOF"), testConverter(t, true)).Error(fmt.Errorf("kip: %w", io.EOF)).Send()
		})
		t.Run("SingleStringSlice", func(t *testing.T) {
			NewBuilder(mockSenderMessage(t, "hello world 543"), testConverter(t, true)).Strings([]string{"hello", "world", "543"}).Send()
		})
		t.Run("FromMap", func(t *testing.T) {
			NewBuilder(mockSenderMessage(t, "hello='world'"), testConverter(t, true)).StringMap(map[string]string{"hello": "world"}).Send()
		})
	})
	t.Run("Conditional", func(t *testing.T) {
		t.Run("True", func(t *testing.T) {
			NewBuilder(mockSenderMessage(t, "hi kip"), testConverter(t, true)).Ln("hi kip").When(true).Send()
		})
		t.Run("False", func(t *testing.T) {
			NewBuilder(mockSender(t, 1), testConverter(t, true)).Ln("hello").When(false).Group().Send()
		})
	})
}

// TestBuilderMessageCreation tests various message creation methods using table-driven approach
func TestBuilderMessageCreation(t *testing.T) {
	tests := []struct {
		name     string
		build    func(*Builder) *Builder
		validate func(*testing.T, string)
	}{
		{
			name: "BytesMessage",
			build: func(b *Builder) *Builder {
				return b.Bytes([]byte("binary data"))
			},
			validate: func(t *testing.T, s string) {
				check.True(t, len(s) > 0)
			},
		},
		{
			name: "AnyMapMessage",
			build: func(b *Builder) *Builder {
				return b.AnyMap(map[string]any{"status": "ok", "code": 200})
			},
			validate: func(t *testing.T, s string) {
				check.True(t, len(s) > 0)
			},
		},
		{
			name: "FieldsWithExistingMessage",
			build: func(b *Builder) *Builder {
				return b.Ln("base").Fields(Fields{"key": "value", "count": 10})
			},
			validate: func(t *testing.T, s string) {
				check.True(t, len(s) > 0)
			},
		},
		{
			name: "FieldsAsFirstMessage",
			build: func(b *Builder) *Builder {
				return b.Fields(Fields{"only": "fields"})
			},
			validate: func(t *testing.T, s string) {
				check.True(t, len(s) > 0)
			},
		},
		{
			name: "ChainedKV",
			build: func(b *Builder) *Builder {
				return b.Ln("msg").KV("k1", "v1").KV("k2", 42).KV("k3", true)
			},
			validate: func(t *testing.T, s string) {
				check.True(t, len(s) > 0)
			},
		},
		{
			name: "ComposerMethod",
			build: func(b *Builder) *Builder {
				return b.Composer(MakeString("composed"))
			},
			validate: func(t *testing.T, s string) {
				check.True(t, len(s) > 0)
			},
		},
		{
			name: "AnyMethod",
			build: func(b *Builder) *Builder {
				return b.Any(struct{ Name string }{"test"})
			},
			validate: func(t *testing.T, s string) {
				check.True(t, len(s) > 0)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder(nil, DefaultConverter())
			builder = tt.build(builder)
			msg := builder.Message()
			check.NotZero(t, msg)
			tt.validate(t, msg.String())
		})
	}
}

// TestBuilderLevels tests priority/level setting with table-driven approach
func TestBuilderLevels(t *testing.T) {
	tests := []struct {
		name     string
		build    func(*Builder) *Builder
		expected int
	}{
		{
			name: "InfoLevel",
			build: func(b *Builder) *Builder {
				return b.Ln("message").Level(40) // Info level
			},
			expected: 40,
		},
		{
			name: "ErrorLevel",
			build: func(b *Builder) *Builder {
				return b.Ln("message").Level(96) // Error level
			},
			expected: 96,
		},
		{
			name: "DebugLevel",
			build: func(b *Builder) *Builder {
				return b.Ln("message").Level(16) // Debug level
			},
			expected: 16,
		},
		{
			name: "SetPriorityMethod",
			build: func(b *Builder) *Builder {
				b.Ln("message")
				b.SetPriority(50) // Notice level
				return b
			},
			expected: 50,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder(nil, DefaultConverter())
			builder = tt.build(builder)
			msg := builder.Message()
			check.Equal(t, tt.expected, int(msg.Priority()))
		})
	}
}

// TestBuilderChaining tests message chaining behavior
func TestBuilderChaining(t *testing.T) {
	tests := []struct {
		name         string
		build        func(*Builder) *Builder
		expectedMsgs int
	}{
		{
			name: "SingleMessage",
			build: func(b *Builder) *Builder {
				return b.Ln("single")
			},
			expectedMsgs: 1,
		},
		{
			name: "TwoMessages",
			build: func(b *Builder) *Builder {
				return b.Ln("first").Ln("second")
			},
			expectedMsgs: 2,
		},
		{
			name: "ThreeMessages",
			build: func(b *Builder) *Builder {
				return b.Ln("a").Ln("b").Ln("c")
			},
			expectedMsgs: 3,
		},
		{
			name: "MixedTypes",
			build: func(b *Builder) *Builder {
				return b.Ln("text").Error(errors.New("err")).F("fmt %d", 1)
			},
			expectedMsgs: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder(nil, DefaultConverter())
			builder = tt.build(builder)
			msg := builder.Message()
			unwound := Unwind(msg)
			check.Equal(t, tt.expectedMsgs, len(unwound))
		})
	}
}

// TestBuilderAnnotations tests annotation and extension methods
func TestBuilderAnnotations(t *testing.T) {
	tests := []struct {
		name  string
		build func(*Builder) *Builder
	}{
		{
			name: "SingleAnnotation",
			build: func(b *Builder) *Builder {
				b.Annotate("key", "value")
				return b
			},
		},
		{
			name: "MultipleAnnotations",
			build: func(b *Builder) *Builder {
				b.Annotate("k1", "v1")
				b.Annotate("k2", 42)
				b.Annotate("k3", true)
				return b
			},
		},
		{
			name: "AnnotateAfterMessage",
			build: func(b *Builder) *Builder {
				b.Ln("base")
				b.Annotate("key", "value")
				return b
			},
		},
		{
			name: "KVChain",
			build: func(b *Builder) *Builder {
				return b.KV("a", 1).KV("b", 2).KV("c", 3)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder(nil, DefaultConverter())
			builder = tt.build(builder)
			msg := builder.Message()
			check.NotZero(t, msg)
			check.True(t, msg.Loggable())
		})
	}
}

// TestBuilderEdgeCases tests edge cases and boundary conditions
func TestBuilderEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		build    func() *Builder
		validate func(*testing.T, *Builder)
	}{
		{
			name: "EmptyBuilder",
			build: func() *Builder {
				return NewBuilder(nil, DefaultConverter())
			},
			validate: func(t *testing.T, b *Builder) {
				msg := b.Message()
				check.NotZero(t, msg)
				// Empty builder with errors creates non-loggable message
			},
		},
		{
			name: "NoConverter",
			build: func() *Builder {
				return NewBuilder(nil, nil)
			},
			validate: func(t *testing.T, b *Builder) {
				msg := b.Message()
				check.NotZero(t, msg)
			},
		},
		{
			name: "InitBeforeMessage",
			build: func() *Builder {
				b := NewBuilder(nil, DefaultConverter())
				// Trigger init methods - these create a default composer
				_ = b.String()
				_ = b.Structured()
				_ = b.Priority()
				return b
			},
			validate: func(t *testing.T, b *Builder) {
				// After init, builder should have a composer
				msg := b.Message()
				check.NotZero(t, msg)
			},
		},
		{
			name: "OnlyKVNoMessage",
			build: func() *Builder {
				b := NewBuilder(nil, DefaultConverter())
				return b.KV("key", "value")
			},
			validate: func(t *testing.T, b *Builder) {
				msg := b.Message()
				check.NotZero(t, msg)
			},
		},
		{
			name: "OnlyAnnotationsNoMessage",
			build: func() *Builder {
				b := NewBuilder(nil, DefaultConverter())
				b.Annotate("k", "v")
				return b
			},
			validate: func(t *testing.T, b *Builder) {
				msg := b.Message()
				check.NotZero(t, msg)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := tt.build()
			tt.validate(t, builder)
		})
	}
}

// TestBuilderComposerInterface tests that Builder implements Composer interface properly
func TestBuilderComposerInterface(t *testing.T) {
	tests := []struct {
		structured bool
		name       string
		build      func(*Builder) *Builder
	}{
		{
			name: "StringMethod",
			build: func(b *Builder) *Builder {
				return b.Ln("test")
			},
		},
		{
			name:       "StructuredMethod",
			structured: true,
			build: func(b *Builder) *Builder {
				return b.Fields(Fields{"k": "v"})
			},
		},
		{
			name: "PriorityMethod",
			build: func(b *Builder) *Builder {
				return b.Ln("test").Level(40)
			},
		},
		{
			name: "RawMethod",
			build: func(b *Builder) *Builder {
				return b.Ln("test")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := NewBuilder(nil, DefaultConverter())
			builder = tt.build(builder)

			// Test Composer interface methods
			_ = builder.String()

			check.Equal(t, builder.Structured(), tt.structured)
			_ = builder.Priority()
			_ = builder.Raw()
			check.True(t, builder.Loggable())
		})
	}
}

// TestBuilderOptions tests option application
func TestBuilderOptions(t *testing.T) {
	tests := []struct {
		name         string
		build        func(*Builder) *Builder
		expectedSent int
	}{
		{
			name: "WithSingleOption",
			build: func(b *Builder) *Builder {
				return b.Ln("message").WithOptions(OptionIncludeMetadata)
			},
			expectedSent: 1,
		},
		{
			name: "WithMultipleOptions",
			build: func(b *Builder) *Builder {
				return b.Ln("message").WithOptions(OptionIncludeMetadata, OptionCollectInfo)
			},
			expectedSent: 1,
		},
		{
			name: "SetOptionMethod",
			build: func(b *Builder) *Builder {
				b.Ln("message")
				b.SetOption(OptionSkipMetadata)
				return b
			},
			expectedSent: 1,
		},
		{
			name: "OptionsWithGroupedSend",
			build: func(b *Builder) *Builder {
				return b.Ln("a").Ln("b").Group().WithOptions(OptionCollectInfo)
			},
			expectedSent: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sentCount := 0
			sender := func(c Composer) { sentCount++ }

			builder := NewBuilder(sender, DefaultConverter())
			builder = tt.build(builder)
			builder.Send()

			check.Equal(t, tt.expectedSent, sentCount)
		})
	}
}
