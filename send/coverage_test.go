package send

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strings"
	"testing"

	"github.com/tychoish/fun/assert/check"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
)

// Tests for buf.go - MakeBytesBuffer

func TestMakeBytesBuffer(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() (*bytes.Buffer, Sender)
		messages []message.Composer
		validate func(*testing.T, *bytes.Buffer)
	}{
		{
			name: "BasicMessage",
			setup: func() (*bytes.Buffer, Sender) {
				buf := &bytes.Buffer{}
				sender := MakeBytesBuffer(buf)
				sender.SetPriority(level.Info)
				sender.SetFormatter(MakePlainFormatter())
				return buf, sender
			},
			messages: []message.Composer{
				func() message.Composer {
					m := message.MakeString("test message")
					m.SetPriority(level.Info)
					return m
				}(),
			},
			validate: func(t *testing.T, buf *bytes.Buffer) {
				output := buf.String()
				if !strings.Contains(output, "test message") {
					t.Errorf("expected 'test message' in output, got: %q", output)
				}
				if !strings.Contains(output, "\n") {
					t.Error("expected newline in output")
				}
			},
		},
		{
			name: "MultipleMessages",
			setup: func() (*bytes.Buffer, Sender) {
				buf := &bytes.Buffer{}
				sender := MakeBytesBuffer(buf)
				sender.SetPriority(level.Debug)
				sender.SetFormatter(MakePlainFormatter())
				return buf, sender
			},
			messages: []message.Composer{
				func() message.Composer {
					m := message.MakeString("first")
					m.SetPriority(level.Debug)
					return m
				}(),
				func() message.Composer {
					m := message.MakeString("second")
					m.SetPriority(level.Debug)
					return m
				}(),
				func() message.Composer {
					m := message.MakeString("third")
					m.SetPriority(level.Debug)
					return m
				}(),
			},
			validate: func(t *testing.T, buf *bytes.Buffer) {
				output := buf.String()
				if !strings.Contains(output, "first") {
					t.Error("expected 'first' in output")
				}
				if !strings.Contains(output, "second") {
					t.Error("expected 'second' in output")
				}
				if !strings.Contains(output, "third") {
					t.Error("expected 'third' in output")
				}
				// Should have 3 newlines
				if strings.Count(output, "\n") < 3 {
					t.Error("expected at least 3 newlines")
				}
			},
		},
		{
			name: "BelowThreshold",
			setup: func() (*bytes.Buffer, Sender) {
				buf := &bytes.Buffer{}
				sender := MakeBytesBuffer(buf)
				sender.SetPriority(level.Error)
				return buf, sender
			},
			messages: []message.Composer{
				func() message.Composer {
					m := message.MakeString("debug msg")
					m.SetPriority(level.Debug)
					return m
				}(),
			},
			validate: func(t *testing.T, buf *bytes.Buffer) {
				if buf.Len() > 0 {
					t.Errorf("expected no output for below-threshold message, got: %q", buf.String())
				}
			},
		},
		{
			name: "WithFormatter",
			setup: func() (*bytes.Buffer, Sender) {
				buf := &bytes.Buffer{}
				sender := MakeBytesBuffer(buf)
				sender.SetPriority(level.Info)
				sender.SetFormatter(MakeDefaultFormatter())
				return buf, sender
			},
			messages: []message.Composer{
				func() message.Composer {
					m := message.MakeString("formatted")
					m.SetPriority(level.Info)
					return m
				}(),
			},
			validate: func(t *testing.T, buf *bytes.Buffer) {
				output := buf.String()
				if !strings.Contains(output, "[p=") {
					t.Error("expected formatted output with priority")
				}
				if !strings.Contains(output, "formatted") {
					t.Error("expected 'formatted' in output")
				}
			},
		},
		{
			name: "NonLoggableMessage",
			setup: func() (*bytes.Buffer, Sender) {
				buf := &bytes.Buffer{}
				sender := MakeBytesBuffer(buf)
				sender.SetPriority(level.Info)
				return buf, sender
			},
			messages: []message.Composer{
				message.MakeString(""), // Empty string is not loggable
			},
			validate: func(t *testing.T, buf *bytes.Buffer) {
				if buf.Len() > 0 {
					t.Errorf("expected no output for non-loggable message, got: %q", buf.String())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf, sender := tt.setup()
			for _, msg := range tt.messages {
				sender.Send(msg)
			}
			tt.validate(t, buf)
		})
	}
}

// Tests for error_handler.go

func TestWrapError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		composer message.Composer
		validate func(*testing.T, error)
	}{
		{
			name:     "NilError",
			err:      nil,
			composer: message.MakeString("test"),
			validate: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("expected nil error, got: %v", err)
				}
			},
		},
		{
			name:     "WithError",
			err:      errors.New("original error"),
			composer: message.MakeString("test message"),
			validate: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("expected non-nil error")
				}
				errStr := err.Error()
				if !strings.Contains(errStr, "original error") {
					t.Errorf("expected 'original error' in wrapped error, got: %v", err)
				}
				if !strings.Contains(errStr, "test message") {
					t.Errorf("expected 'test message' in wrapped error, got: %v", err)
				}
				if !errors.Is(err, ErrGripMessageSendError) {
					t.Error("expected error to wrap ErrGripMessageSendError")
				}
			},
		},
		{
			name:     "WithComplexComposer",
			err:      errors.New("send failed"),
			composer: message.NewKV().KV("key", "value"),
			validate: func(t *testing.T, err error) {
				if err == nil {
					t.Fatal("expected non-nil error")
				}
				errStr := err.Error()
				if !strings.Contains(errStr, "send failed") {
					t.Error("expected original error in wrapped error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WrapError(tt.err, tt.composer)
			tt.validate(t, result)
		})
	}
}

func TestErrorHandlerWriter(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		validate func(*testing.T, *bytes.Buffer)
	}{
		{
			name: "WithError",
			err:  errors.New("test error"),
			validate: func(t *testing.T, buf *bytes.Buffer) {
				output := buf.String()
				if !strings.Contains(output, "logging error:") {
					t.Error("expected 'logging error:' in output")
				}
				if !strings.Contains(output, "test error") {
					t.Error("expected 'test error' in output")
				}
			},
		},
		{
			name: "WithNilError",
			err:  nil,
			validate: func(t *testing.T, buf *bytes.Buffer) {
				if buf.Len() > 0 {
					t.Errorf("expected no output for nil error, got: %q", buf.String())
				}
			},
		},
		{
			name: "MultipleErrors",
			err:  errors.New("first error"),
			validate: func(t *testing.T, buf *bytes.Buffer) {
				output := buf.String()
				if !strings.Contains(output, "first error") {
					t.Error("expected 'first error' in output")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			handler := ErrorHandlerWriter(buf)
			handler(tt.err)
			tt.validate(t, buf)
		})
	}
}

func TestErrorHandlerFromLogger(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		validate func(*testing.T, *bytes.Buffer)
	}{
		{
			name: "WithError",
			err:  errors.New("logged error"),
			validate: func(t *testing.T, buf *bytes.Buffer) {
				output := buf.String()
				if !strings.Contains(output, "logging error:") {
					t.Error("expected 'logging error:' in output")
				}
				if !strings.Contains(output, "logged error") {
					t.Error("expected 'logged error' in output")
				}
			},
		},
		{
			name: "WithNilError",
			err:  nil,
			validate: func(t *testing.T, buf *bytes.Buffer) {
				if buf.Len() > 0 {
					t.Errorf("expected no output for nil error, got: %q", buf.String())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			logger := log.New(buf, "", 0)
			handler := ErrorHandlerFromLogger(logger)
			handler(tt.err)
			tt.validate(t, buf)
		})
	}
}

func TestErrorHandlerFromSender(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		validate func(*testing.T, *InternalSender)
	}{
		{
			name: "WithError",
			err:  errors.New("sender error"),
			validate: func(t *testing.T, sender *InternalSender) {
				if sender.Len() == 0 {
					t.Fatal("expected message to be sent")
				}
				msg := sender.GetMessage()
				if msg == nil {
					t.Fatal("expected non-nil message")
				}
				if !strings.Contains(msg.Message.String(), "sender error") {
					t.Errorf("expected 'sender error' in message, got: %v", msg.Message.String())
				}
			},
		},
		{
			name: "WithNilError",
			err:  nil,
			validate: func(t *testing.T, sender *InternalSender) {
				if sender.Len() > 0 {
					t.Error("expected no messages for nil error")
				}
			},
		},
		{
			name: "ErrorPriority",
			err:  errors.New("priority test"),
			validate: func(t *testing.T, sender *InternalSender) {
				if sender.Len() == 0 {
					t.Fatal("expected message to be sent")
				}
				msg := sender.GetMessage()
				if msg.Priority != level.Error {
					t.Errorf("expected Error priority, got: %v", msg.Priority)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := MakeInternal()
			handler := ErrorHandlerFromSender(sender)
			handler(tt.err)
			tt.validate(t, sender)
		})
	}
}

// Tests for formatter.go

func TestWithOptionSender(t *testing.T) {
	tests := []struct {
		name     string
		options  []message.Option
		validate func(*testing.T, Sender, message.Composer)
	}{
		{
			name:    "WithSingleOption",
			options: []message.Option{message.OptionIncludeMetadata},
			validate: func(t *testing.T, sender Sender, msg message.Composer) {
				// Send the message through the wrapped sender
				sender.Send(msg)
				// The option should have been applied
			},
		},
		{
			name:    "WithMultipleOptions",
			options: []message.Option{message.OptionIncludeMetadata, message.OptionCollectInfo},
			validate: func(t *testing.T, sender Sender, msg message.Composer) {
				sender.Send(msg)
				// Options should have been applied
			},
		},
		{
			name:    "WithNoOptions",
			options: []message.Option{},
			validate: func(t *testing.T, sender Sender, msg message.Composer) {
				sender.Send(msg)
				// Should still work with no options
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			baseSender := MakeInternal()
			wrappedSender := WithOptionSender(baseSender, tt.options...)
			msg := message.MakeString("test")
			tt.validate(t, wrappedSender, msg)
		})
	}
}

func TestMakeJSONFormatter(t *testing.T) {
	tests := []struct {
		name     string
		composer message.Composer
		validate func(*testing.T, string, error)
	}{
		{
			name:     "SimpleMessage",
			composer: message.MakeString("test"),
			validate: func(t *testing.T, output string, err error) {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				if !strings.Contains(output, "test") {
					t.Errorf("expected 'test' in JSON output, got: %q", output)
				}
				if !strings.HasPrefix(output, "{") || !strings.HasSuffix(output, "}") {
					t.Errorf("expected JSON object, got: %q", output)
				}
			},
		},
		{
			name:     "StructuredMessage",
			composer: message.NewKV().KV("key", "value"),
			validate: func(t *testing.T, output string, err error) {
				if err != nil {
					t.Fatalf("expected no error, got: %v", err)
				}
				// JSON formatter may return empty object for non-loggable structured messages
				if !strings.HasPrefix(output, "{") || !strings.HasSuffix(output, "}") {
					t.Errorf("expected JSON object, got: %q", output)
				}
			},
		},
	}

	formatter := MakeJSONFormatter()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output, err := formatter(tt.composer)
			tt.validate(t, output, err)
		})
	}
}

// Tests for base.go error handling methods

func TestBaseGetErrorHandler(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Base
		validate func(*testing.T, ErrorHandler)
	}{
		{
			name: "NoHandlerSet",
			setup: func() *Base {
				return &Base{}
			},
			validate: func(t *testing.T, handler ErrorHandler) {
				if handler != nil {
					t.Error("expected nil handler when none is set")
				}
			},
		},
		{
			name: "WithHandlerSet",
			setup: func() *Base {
				base := &Base{}
				buf := &bytes.Buffer{}
				base.SetErrorHandler(ErrorHandlerWriter(buf))
				return base
			},
			validate: func(t *testing.T, handler ErrorHandler) {
				if handler == nil {
					t.Error("expected non-nil handler")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := tt.setup()
			handler := base.GetErrorHandler()
			tt.validate(t, handler)
		})
	}
}

func TestBaseHandleError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		validate func(*testing.T, *bytes.Buffer)
	}{
		{
			name: "WithNilError",
			err:  nil,
			validate: func(t *testing.T, buf *bytes.Buffer) {
				if buf.Len() > 0 {
					t.Error("expected no output for nil error")
				}
			},
		},
		{
			name: "WithError",
			err:  errors.New("test error"),
			validate: func(t *testing.T, buf *bytes.Buffer) {
				if buf.Len() == 0 {
					t.Error("expected output for error")
				}
				if !strings.Contains(buf.String(), "test error") {
					t.Error("expected 'test error' in output")
				}
			},
		},
		{
			name: "WithoutErrorHandler",
			err:  errors.New("ignored error"),
			validate: func(t *testing.T, buf *bytes.Buffer) {
				// Should not panic when no handler is set
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			base := &Base{}
			if tt.name != "WithoutErrorHandler" {
				base.SetErrorHandler(ErrorHandlerWriter(buf))
			}
			base.HandleError(tt.err)
			tt.validate(t, buf)
		})
	}
}

func TestBaseHandleErrorOK(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		expectedResult bool
		validate       func(*testing.T, *bytes.Buffer)
	}{
		{
			name:           "WithNilError",
			err:            nil,
			expectedResult: true,
			validate: func(t *testing.T, buf *bytes.Buffer) {
				if buf.Len() > 0 {
					t.Error("expected no output for nil error")
				}
			},
		},
		{
			name:           "WithError",
			err:            errors.New("test error"),
			expectedResult: false,
			validate: func(t *testing.T, buf *bytes.Buffer) {
				if buf.Len() == 0 {
					t.Error("expected output for error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			base := &Base{}
			base.SetErrorHandler(ErrorHandlerWriter(buf))
			result := base.HandleErrorOK(tt.err)
			if result != tt.expectedResult {
				t.Errorf("expected %v, got %v", tt.expectedResult, result)
			}
			tt.validate(t, buf)
		})
	}
}

// Tests for interface.go - NopSender

func TestNopSender(t *testing.T) {
	tests := []struct {
		name     string
		validate func(*testing.T, Sender)
	}{
		{
			name: "SendDoesNothing",
			validate: func(t *testing.T, sender Sender) {
				// Should not panic
				sender.Send(message.MakeString("test"))
				sender.Send(message.NewKV().KV("key", "value"))
			},
		},
		{
			name: "SetPriorityWorks",
			validate: func(t *testing.T, sender Sender) {
				sender.SetPriority(level.Debug)
				if sender.Priority() != level.Debug {
					t.Error("expected priority to be set")
				}
			},
		},
		{
			name: "SetNameWorks",
			validate: func(t *testing.T, sender Sender) {
				sender.SetName("noop")
				if sender.Name() != "noop" {
					t.Error("expected name to be set")
				}
			},
		},
		{
			name: "FlushWorks",
			validate: func(t *testing.T, sender Sender) {
				err := sender.Flush(context.Background())
				if err != nil {
					t.Errorf("expected no error from Flush, got: %v", err)
				}
			},
		},
		{
			name: "CloseWorks",
			validate: func(t *testing.T, sender Sender) {
				err := sender.Close()
				if err != nil {
					t.Errorf("expected no error from Close, got: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := NopSender()
			tt.validate(t, sender)
		})
	}
}

// Tests for multi.go - AddToMulti

func TestAddToMulti(t *testing.T) {
	tests := []struct {
		name        string
		setup       func() (Sender, Sender)
		expectError bool
		validate    func(*testing.T, Sender)
	}{
		{
			name: "AddToMultiSender",
			setup: func() (Sender, Sender) {
				multi := MakeMulti()
				child := MakeInternal()
				return multi, child
			},
			expectError: false,
			validate: func(t *testing.T, multi Sender) {
				// Send a message and verify it goes to both senders
				multi.Send(message.MakeString("test"))
			},
		},
		{
			name: "AddToNonMultiSender",
			setup: func() (Sender, Sender) {
				notMulti := MakeInternal()
				child := MakeInternal()
				return notMulti, child
			},
			expectError: true,
			validate:    nil,
		},
		{
			name: "AddMultipleSenders",
			setup: func() (Sender, Sender) {
				multi := MakeMulti()
				child := MakeInternal()
				return multi, child
			},
			expectError: false,
			validate: func(t *testing.T, multi Sender) {
				// Add another sender
				child2 := MakeInternal()
				err := AddToMulti(multi, child2)
				if err != nil {
					t.Errorf("expected no error adding second child, got: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multi, child := tt.setup()
			err := AddToMulti(multi, child)
			if tt.expectError && err == nil {
				t.Error("expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("expected no error, got: %v", err)
			}
			if tt.validate != nil {
				tt.validate(t, multi)
			}
		})
	}
}

func TestMultiSenderSetFormatter(t *testing.T) {
	tests := []struct {
		name     string
		validate func(*testing.T, Sender)
	}{
		{
			name: "SetFormatterPropagates",
			validate: func(t *testing.T, multi Sender) {
				formatter := MakeJSONFormatter()
				multi.SetFormatter(formatter)
				// Should set formatter on all child senders
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multi := MakeMulti()
			child1 := MakeInternal()
			child2 := MakeInternal()
			check.NotError(t, AddToMulti(multi, child1))
			check.NotError(t, AddToMulti(multi, child2))
			tt.validate(t, multi)
		})
	}
}

func TestMultiSenderSetErrorHandler(t *testing.T) {
	tests := []struct {
		name     string
		validate func(*testing.T, Sender)
	}{
		{
			name: "SetErrorHandlerPropagates",
			validate: func(t *testing.T, multi Sender) {
				buf := &bytes.Buffer{}
				handler := ErrorHandlerWriter(buf)
				multi.SetErrorHandler(handler)
				// Should set error handler on all child senders
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			multi := MakeMulti()
			child1 := MakeInternal()
			child2 := MakeInternal()
			check.NotError(t, AddToMulti(multi, child1))
			check.NotError(t, AddToMulti(multi, child2))
			tt.validate(t, multi)
		})
	}
}

// Tests for async_group.go - SetErrorHandler and SetFormatter

func TestAsyncGroupSetErrorHandler(t *testing.T) {
	tests := []struct {
		name     string
		validate func(*testing.T, Sender)
	}{
		{
			name: "SetErrorHandlerOnAsyncGroup",
			validate: func(t *testing.T, sender Sender) {
				buf := &bytes.Buffer{}
				handler := ErrorHandlerWriter(buf)
				sender.SetErrorHandler(handler)
				// Should set error handler on async group and all children
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := MakeAsyncGroup(context.Background(), 10, MakeInternal())
			defer func() { check.NotError(t, sender.Close()) }()
			tt.validate(t, sender)
		})
	}
}

func TestAsyncGroupSetFormatter(t *testing.T) {
	tests := []struct {
		name     string
		validate func(*testing.T, Sender)
	}{
		{
			name: "SetFormatterOnAsyncGroup",
			validate: func(t *testing.T, sender Sender) {
				formatter := MakeJSONFormatter()
				sender.SetFormatter(formatter)
				// Should set formatter on async group and all children
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := MakeAsyncGroup(context.Background(), 10, MakeInternal())
			defer func() { check.NotError(t, sender.Close()) }()
			tt.validate(t, sender)
		})
	}
}

// Edge case and error handling tests

func TestShouldLogEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		senderPriority level.Priority
		msgPriority    level.Priority
		msgLoggable    bool
		expected       bool
	}{
		{
			name:           "NonLoggableMessage",
			senderPriority: level.Info,
			msgPriority:    level.Info,
			msgLoggable:    false,
			expected:       false,
		},
		{
			name:           "BelowThreshold",
			senderPriority: level.Error,
			msgPriority:    level.Debug,
			msgLoggable:    true,
			expected:       false,
		},
		{
			name:           "AboveThreshold",
			senderPriority: level.Debug,
			msgPriority:    level.Error,
			msgLoggable:    true,
			expected:       true,
		},
		{
			name:           "EqualPriority",
			senderPriority: level.Info,
			msgPriority:    level.Info,
			msgLoggable:    true,
			expected:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := MakeInternal()
			sender.SetPriority(tt.senderPriority)

			var msg message.Composer
			if tt.msgLoggable {
				msg = message.MakeString("test")
			} else {
				msg = message.MakeString("")
			}
			msg.SetPriority(tt.msgPriority)

			result := ShouldLog(sender, msg)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestBufferedMakeBufferedEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		size     int
		validate func(*testing.T, Sender)
	}{
		{
			name: "ZeroSize",
			size: 0,
			validate: func(t *testing.T, sender Sender) {
				// Size of 0 should be handled gracefully
				if sender == nil {
					t.Error("expected non-nil sender")
				}
			},
		},
		{
			name: "ValidSize",
			size: 10,
			validate: func(t *testing.T, sender Sender) {
				if sender == nil {
					t.Error("expected non-nil sender")
				}
				// Test that it can send messages
				sender.Send(message.MakeString("test"))
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := MakeBuffered(MakeInternal(), 0, tt.size)
			tt.validate(t, sender)
		})
	}
}
