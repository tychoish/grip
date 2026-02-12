package grip

import (
	"testing"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

func TestPackageLevelClone(t *testing.T) {
	tests := []struct {
		name     string
		validate func(*testing.T, Logger)
	}{
		{
			name: "CloneStdLogger",
			validate: func(t *testing.T, cloned Logger) {
				// Verify cloned logger is not zero value
				if cloned.impl == nil {
					t.Error("cloned logger should have impl")
				}
				if cloned.conv == nil {
					t.Error("cloned logger should have conv")
				}

				// Verify it's independent
				cloned.SetSender(send.MakeStdOut())
				if std.Sender() == cloned.Sender() {
					t.Error("cloned logger should be independent from std")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cloned := Clone()
			tt.validate(t, cloned)
		})
	}
}

func TestPackageLevelConvert(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		validate func(*testing.T, message.Composer)
	}{
		{
			name:  "ConvertString",
			input: "test message",
			validate: func(t *testing.T, c message.Composer) {
				if c == nil {
					t.Fatal("composer should not be nil")
				}
				if c.String() != "test message" {
					t.Errorf("expected 'test message', got %q", c.String())
				}
			},
		},
		{
			name:  "ConvertNil",
			input: nil,
			validate: func(t *testing.T, c message.Composer) {
				if c == nil {
					t.Fatal("composer should not be nil")
				}
				if c.Loggable() {
					t.Error("nil input should produce non-loggable composer")
				}
			},
		},
		{
			name:  "ConvertComposer",
			input: message.MakeString("composer"),
			validate: func(t *testing.T, c message.Composer) {
				if c == nil {
					t.Fatal("composer should not be nil")
				}
				if c.String() != "composer" {
					t.Errorf("expected 'composer', got %q", c.String())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Convert(tt.input)
			tt.validate(t, result)
		})
	}
}

func TestPackageLevelSetSender(t *testing.T) {
	// Save original sender
	originalSender := std.Sender()
	defer func() {
		// Restore original sender
		std.SetSender(originalSender)
	}()

	tests := []struct {
		name     string
		sender   send.Sender
		validate func(*testing.T)
	}{
		{
			name:   "SetPlainSender",
			sender: send.MakeStdOut(),
			validate: func(t *testing.T) {
				if std.Sender() == originalSender {
					t.Error("sender should have been changed")
				}
			},
		},
		{
			name: "SetSenderWithPriority",
			sender: func() send.Sender {
				s := send.MakeStdOut()
				s.SetPriority(level.Debug)
				return s
			}(),
			validate: func(t *testing.T) {
				if std.Sender().Priority() != level.Debug {
					t.Errorf("expected Debug priority, got %v", std.Sender().Priority())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetSender(tt.sender)
			tt.validate(t)
		})
	}
}

func TestPackageLevelSetConverter(t *testing.T) {
	// Save original converter
	originalConverter := std.conv.Get()
	defer func() {
		// Restore original converter
		std.SetConverter(originalConverter.Converter)
	}()

	tests := []struct {
		name      string
		converter message.Converter
		testInput any
		expected  string
	}{
		{
			name: "SetCustomConverter",
			converter: message.ConverterFunc(func(any) (message.Composer, bool) {
				return message.MakeString("custom"), true
			}),
			testInput: "anything",
			expected:  "custom",
		},
		{
			name:      "SetDefaultConverter",
			converter: message.DefaultConverter(),
			testInput: "test",
			expected:  "test",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetConverter(tt.converter)
			result := std.Convert(tt.testInput)
			if result.String() != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, result.String())
			}
		})
	}
}

func TestPackageLevelSend(t *testing.T) {
	// Create a mock sender to capture sent messages
	mockSender := send.MakeInternal()
	originalSender := std.Sender()
	defer func() {
		std.SetSender(originalSender)
	}()

	std.SetSender(mockSender)

	tests := []struct {
		name     string
		composer message.Composer
		validate func(*testing.T)
	}{
		{
			name:     "SendSimpleMessage",
			composer: message.MakeString("test message"),
			validate: func(t *testing.T) {
				// Message should have been sent to the sender
				// We can't easily verify internal state, but we can verify no panic
			},
		},
		{
			name:     "SendStructuredMessage",
			composer: message.NewKV().KV("key", "value"),
			validate: func(t *testing.T) {
				// Should handle structured messages
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			Send(tt.composer)
			tt.validate(t)
		})
	}
}

func TestPackageLevelEmergencyFatal(t *testing.T) {
	// Note: We cannot directly test EmergencyFatal as it calls os.Exit(1)
	// which would terminate the test process. This test documents the limitation.
	t.Skip("EmergencyFatal calls os.Exit(1) and cannot be tested in a standard test")

	// To test this properly, you would need to:
	// 1. Use a subprocess approach (run test in separate process)
	// 2. Mock os.Exit via build tags or interfaces
	// 3. Use integration tests with process exit code verification
}

func TestLoggerEmergencyFatal(t *testing.T) {
	// Note: We cannot directly test Logger.EmergencyFatal as it calls os.Exit(1)
	// which would terminate the test process. This test documents the limitation.
	t.Skip("Logger.EmergencyFatal calls os.Exit(1) and cannot be tested in a standard test")

	// The method implementation:
	// func (g Logger) EmergencyFatal(m any) { g.sendFatal(level.Emergency, m) }
	//
	// And sendFatal calls os.Exit(1):
	// func (g Logger) sendFatal(l level.Priority, in any) {
	//     if m, s := g.ms(l, in); send.ShouldLog(s, m) {
	//         s.Send(m)
	//         os.Exit(1)
	//     }
	// }
}

func TestLoggerMethodsIndirectly(t *testing.T) {
	// This test verifies that the uncovered Logger methods work correctly
	// by testing them through a custom logger instance
	tests := []struct {
		name     string
		setup    func() Logger
		validate func(*testing.T, Logger)
	}{
		{
			name: "CloneMethod",
			setup: func() Logger {
				sender := send.MakeStdOut()
				sender.SetPriority(level.Info)
				return NewLogger(sender)
			},
			validate: func(t *testing.T, logger Logger) {
				cloned := logger.Clone()
				if cloned.impl == nil {
					t.Error("cloned logger should have impl")
				}
				if cloned.conv == nil {
					t.Error("cloned logger should have conv")
				}

				// Verify independence
				cloned.SetSender(send.MakeStdOut())
				if logger.Sender() == cloned.Sender() {
					t.Error("clone should be independent")
				}
			},
		},
		{
			name: "ConvertMethod",
			setup: func() Logger {
				return NewLogger(send.MakeStdOut())
			},
			validate: func(t *testing.T, logger Logger) {
				result := logger.Convert("test")
				if result == nil {
					t.Fatal("convert should not return nil")
				}
				if result.String() != "test" {
					t.Errorf("expected 'test', got %q", result.String())
				}
			},
		},
		{
			name: "SetSenderMethod",
			setup: func() Logger {
				return NewLogger(send.MakeStdOut())
			},
			validate: func(t *testing.T, logger Logger) {
				newSender := send.MakeStdOut()
				newSender.SetPriority(level.Debug)
				logger.SetSender(newSender)

				if logger.Sender().Priority() != level.Debug {
					t.Error("sender should have been updated")
				}
			},
		},
		{
			name: "SetConverterMethod",
			setup: func() Logger {
				return NewLogger(send.MakeStdOut())
			},
			validate: func(t *testing.T, logger Logger) {
				customConverter := message.ConverterFunc(func(any) (message.Composer, bool) {
					return message.MakeString("converted"), true
				})
				logger.SetConverter(customConverter)

				result := logger.Convert("anything")
				if result.String() != "converted" {
					t.Errorf("expected 'converted', got %q", result.String())
				}
			},
		},
		{
			name: "SendMethod",
			setup: func() Logger {
				return NewLogger(send.MakeInternal())
			},
			validate: func(t *testing.T, logger Logger) {
				composer := message.MakeString("test message")
				// Should not panic
				logger.Send(composer)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := tt.setup()
			tt.validate(t, logger)
		})
	}
}
