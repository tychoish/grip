package message

import (
	"strings"
	"testing"
)

func TestLineMessengerAnnotate(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *strln
		key      string
		value    any
		validate func(*testing.T, *strln)
	}{
		{
			name: "AnnotateWithoutFieldMessage",
			setup: func() *strln {
				return &strln{
					lines: []any{"test"},
				}
			},
			key:   "key1",
			value: "value1",
			validate: func(t *testing.T, lm *strln) {
				if lm.Context.Len() == 0 {
					t.Error("Context should be initialized")
				}
				if lm.Context.Get("key1") != "value1" {
					t.Error("should add to Base.Context")
				}
			},
		},
		{
			name: "AnnotateMultipleTimes",
			setup: func() *strln {
				return &strln{
					lines: []any{"message"},
				}
			},
			key:   "key1",
			value: 1,
			validate: func(t *testing.T, lm *strln) {
				lm.Annotate("key2", 2)
				lm.Annotate("key3", 3)

				if lm.Context.Len() != 3 {
					t.Errorf("should have 3 annotations, got %d", lm.Context.Len())
				}
			},
		},
		{
			name: "AnnotateWithNilValue",
			setup: func() *strln {
				return &strln{
					lines: []any{"test"},
				}
			},
			key:   "nilkey",
			value: nil,
			validate: func(t *testing.T, lm *strln) {
				if _, exists := lm.Context.Load("nilkey"); !exists {
					t.Error("should allow nil values")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := tt.setup()
			lm.Annotate(tt.key, tt.value)
			tt.validate(t, lm)
		})
	}
}

func TestMakeLines(t *testing.T) {
	tests := []struct {
		name     string
		args     []any
		validate func(*testing.T, Composer)
	}{
		{
			name: "EmptyArgs",
			args: []any{},
			validate: func(t *testing.T, c Composer) {
				if c.Loggable() {
					t.Error("empty args should not be loggable")
				}
			},
		},
		{
			name: "SingleArg",
			args: []any{"test"},
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("should be loggable")
				}
				str := c.String()
				if !strings.Contains(str, "test") {
					t.Error("should contain arg")
				}
			},
		},
		{
			name: "MultipleArgs",
			args: []any{"arg1", "arg2", "arg3"},
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("should be loggable")
				}
				str := c.String()
				if !strings.Contains(str, "arg1") || !strings.Contains(str, "arg2") {
					t.Error("should contain all args")
				}
			},
		},
		{
			name: "WithNilArgs",
			args: []any{"valid", nil, "args"},
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("should be loggable with non-nil args")
				}
			},
		},
		{
			name: "WithEmptyStringArgs",
			args: []any{"test", "", "value"},
			validate: func(t *testing.T, c Composer) {
				str := c.String()
				if !strings.Contains(str, "test") || !strings.Contains(str, "value") {
					t.Error("should contain non-empty args")
				}
			},
		},
		{
			name: "AllNilArgs",
			args: []any{nil, nil, nil},
			validate: func(t *testing.T, c Composer) {
				if c.Loggable() {
					t.Error("all nil should not be loggable")
				}
			},
		},
		{
			name: "AllEmptyStrings",
			args: []any{"", "", ""},
			validate: func(t *testing.T, c Composer) {
				if c.Loggable() {
					t.Error("all empty strings should not be loggable")
				}
			},
		},
		{
			name: "MixedTypes",
			args: []any{"string", 42, true, 3.14},
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("should be loggable")
				}
				str := c.String()
				if !strings.Contains(str, "string") || !strings.Contains(str, "42") {
					t.Error("should contain all values")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := MakeLines(tt.args...)
			if c == nil {
				t.Fatal("MakeLines should not return nil")
			}
			tt.validate(t, c)
		})
	}
}

func TestLineMessengerLoggable(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *strln
		expected bool
	}{
		{
			name: "WithLines",
			setup: func() *strln {
				return &strln{
					lines: []any{"test"},
				}
			},
			expected: true,
		},
		{
			name: "WithContext",
			setup: func() *strln {
				lm := &strln{
					lines: []any{},
				}
				lm.Context.Store("key", "value")
				return lm
			},
			expected: true,
		},
		{
			name: "WithFieldMessage",
			setup: func() *strln {
				lm := &strln{
					lines: []any{"msg"},
				}
				lm.Context.Store("k", "v")
				return lm
			},
			expected: true,
		},
		{
			name: "Empty",
			setup: func() *strln {
				return &strln{
					lines: []any{},
				}
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := tt.setup()
			result := lm.Loggable()
			if result != tt.expected {
				t.Errorf("Loggable() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestLineMessengerString(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() *strln
		wantSubstr []string
	}{
		{
			name: "SimpleLines",
			setup: func() *strln {
				return MakeLines([]any{"line1", "line2"}...).(*strln)
			},
			wantSubstr: []string{"line1", "line2"},
		},
		{
			name: "WithFieldMessage",
			setup: func() *strln {
				return MakeLines([]any{"test"}...).(*strln)
			},
			wantSubstr: []string{"test"},
		},
		{
			name: "WithContextNoFieldMessage",
			setup: func() *strln {
				lm := MakeLines("message").(*strln)
				lm.Context.Store("field", "data")
				return lm
			},
			wantSubstr: []string{"message"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := tt.setup()
			str := lm.String()

			for _, substr := range tt.wantSubstr {
				if !strings.Contains(str, substr) {
					t.Errorf("String() = %q, should contain %q", str, substr)
				}
			}
		})
	}
}

func TestLineMessengerRaw(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *strln
		validate func(*testing.T, any)
	}{
		{
			name: "WithFieldMessage",
			setup: func() *strln {
				lm := MakeLines("test")
				lm.Annotate("key", "value")
				return lm.(*strln)
			},
			validate: func(t *testing.T, raw any) {
				if raw == nil {
					t.Error("Raw() should not return nil")
				}
			},
		},
		{
			name: "WithContextNoFieldMessage",
			setup: func() *strln {
				lm := MakeLines("msg")
				lm.Annotate("field", "data")
				return lm.(*strln)
			},
			validate: func(t *testing.T, raw any) {
				if raw == nil {
					t.Error("should create fieldMessage and return raw")
				}
			},
		},
		{
			name: "WithoutIncludeMetadata",
			setup: func() *strln {
				lm := MakeLines("test message")
				return lm.(*strln)
			},
			validate: func(t *testing.T, raw any) {
				// Should return anonymous struct with Msg field
				if raw == nil {
					t.Error("should return struct")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := tt.setup()
			raw := lm.Raw()
			tt.validate(t, raw)
		})
	}
}

func TestLineMessengerSetOption(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *strln
		opts     []Option
		validate func(*testing.T, *strln)
	}{
		{
			name: "WithoutFieldMessage",
			setup: func() *strln {
				return &strln{
					lines: []any{"test"},
				}
			},
			opts: []Option{OptionIncludeMetadata},
			validate: func(t *testing.T, lm *strln) {
				if !lm.IncludeMetadata {
					t.Error("should set IncludeMetadata on Base")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := tt.setup()
			lm.SetOption(tt.opts...)
			tt.validate(t, lm)
		})
	}
}

func TestNewLinesFromStrings(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		validate func(*testing.T, Composer)
	}{
		{
			name: "EmptyArgs",
			args: []string{},
			validate: func(t *testing.T, c Composer) {
				if c.Loggable() {
					t.Error("empty should not be loggable")
				}
			},
		},
		{
			name: "SingleString",
			args: []string{"test"},
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("should be loggable")
				}
				str := c.String()
				if !strings.Contains(str, "test") {
					t.Error("should contain string")
				}
			},
		},
		{
			name: "MultipleStrings",
			args: []string{"str1", "str2", "str3"},
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("should be loggable")
				}
			},
		},
		{
			name: "WithEmptyStrings",
			args: []string{"valid", "", "string"},
			validate: func(t *testing.T, c Composer) {
				str := c.String()
				if !strings.Contains(str, "valid") || !strings.Contains(str, "string") {
					t.Error("should contain non-empty strings")
				}
			},
		},
		{
			name: "AllEmptyStrings",
			args: []string{"", "", ""},
			validate: func(t *testing.T, c Composer) {
				if c.Loggable() {
					t.Error("all empty should not be loggable")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewLines(tt.args)
			if c == nil {
				t.Fatal("newLinesFromStrings should not return nil")
			}
			tt.validate(t, c)
		})
	}
}
