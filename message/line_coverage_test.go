package message

import (
	"strings"
	"testing"
)

func TestLineMessengerSetupField(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *lineMessenger
		validate func(*testing.T, *lineMessenger)
	}{
		{
			name: "EmptyContext",
			setup: func() *lineMessenger {
				return &lineMessenger{
					lines: []any{"test"},
				}
			},
			validate: func(t *testing.T, lm *lineMessenger) {
				lm.setupField()
				if lm.fm == nil {
					t.Error("setupField should create fieldMessage")
				}
				if lm.Message == "" {
					t.Error("setupField should resolve message")
				}
			},
		},
		{
			name: "WithContext",
			setup: func() *lineMessenger {
				lm := &lineMessenger{
					lines: []any{"test message"},
				}
				lm.Context = map[string]any{"key": "value"}
				return lm
			},
			validate: func(t *testing.T, lm *lineMessenger) {
				lm.setupField()
				if lm.fm == nil {
					t.Fatal("setupField should create fieldMessage")
				}
				if len(lm.fm.fields) == 0 {
					t.Error("fieldMessage should have context fields")
				}
				if lm.fm.message == "" {
					t.Error("fieldMessage should have message")
				}
			},
		},
		{
			name: "WithExistingMessage",
			setup: func() *lineMessenger {
				lm := &lineMessenger{
					lines:   []any{"line1", "line2"},
					Message: "already set",
				}
				lm.Context = map[string]any{"field": "data"}
				return lm
			},
			validate: func(t *testing.T, lm *lineMessenger) {
				lm.setupField()
				if lm.fm == nil {
					t.Error("should create fieldMessage")
				}
				if lm.fm.message == "" {
					t.Error("fieldMessage should preserve message")
				}
			},
		},
		{
			name: "MultipleLines",
			setup: func() *lineMessenger {
				lm := &lineMessenger{
					lines: []any{"line1", "line2", "line3"},
				}
				lm.Context = map[string]any{"k1": "v1", "k2": "v2"}
				return lm
			},
			validate: func(t *testing.T, lm *lineMessenger) {
				lm.setupField()
				if lm.fm == nil {
					t.Error("should create fieldMessage")
				}
				if len(lm.fm.fields) != 2 {
					t.Error("should preserve all context fields")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lm := tt.setup()
			tt.validate(t, lm)
		})
	}
}

func TestLineMessengerAnnotate(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *lineMessenger
		key      string
		value    any
		validate func(*testing.T, *lineMessenger)
	}{
		{
			name: "AnnotateWithoutFieldMessage",
			setup: func() *lineMessenger {
				return &lineMessenger{
					lines: []any{"test"},
				}
			},
			key:   "key1",
			value: "value1",
			validate: func(t *testing.T, lm *lineMessenger) {
				if lm.Context == nil {
					t.Error("Context should be initialized")
				}
				if lm.Context["key1"] != "value1" {
					t.Error("should add to Base.Context")
				}
				if lm.fm != nil {
					t.Error("should not create fieldMessage yet")
				}
			},
		},
		{
			name: "AnnotateWithFieldMessage",
			setup: func() *lineMessenger {
				lm := &lineMessenger{
					lines: []any{"test"},
				}
				lm.Context = map[string]any{"existing": "field"}
				lm.setupField()
				return lm
			},
			key:   "newkey",
			value: "newvalue",
			validate: func(t *testing.T, lm *lineMessenger) {
				if lm.fm == nil {
					t.Error("fieldMessage should exist")
				}
				if lm.fm.fields["newkey"] != "newvalue" {
					t.Error("should add to fieldMessage fields")
				}
			},
		},
		{
			name: "AnnotateMultipleTimes",
			setup: func() *lineMessenger {
				return &lineMessenger{
					lines: []any{"message"},
				}
			},
			key:   "key1",
			value: 1,
			validate: func(t *testing.T, lm *lineMessenger) {
				lm.Annotate("key2", 2)
				lm.Annotate("key3", 3)

				if len(lm.Context) != 3 {
					t.Errorf("should have 3 annotations, got %d", len(lm.Context))
				}
			},
		},
		{
			name: "AnnotateWithNilValue",
			setup: func() *lineMessenger {
				return &lineMessenger{
					lines: []any{"test"},
				}
			},
			key:   "nilkey",
			value: nil,
			validate: func(t *testing.T, lm *lineMessenger) {
				if _, exists := lm.Context["nilkey"]; !exists {
					t.Error("should allow nil values")
				}
			},
		},
		{
			name: "AnnotateAfterSetupField",
			setup: func() *lineMessenger {
				lm := &lineMessenger{
					lines: []any{"msg"},
				}
				lm.Annotate("first", "value")
				lm.setupField()
				return lm
			},
			key:   "second",
			value: "value2",
			validate: func(t *testing.T, lm *lineMessenger) {
				if lm.fm == nil {
					t.Error("should have fieldMessage")
				}
				if lm.fm.fields["second"] != "value2" {
					t.Error("should annotate fieldMessage after setup")
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
		setup    func() *lineMessenger
		expected bool
	}{
		{
			name: "WithLines",
			setup: func() *lineMessenger {
				return &lineMessenger{
					lines: []any{"test"},
				}
			},
			expected: true,
		},
		{
			name: "WithContext",
			setup: func() *lineMessenger {
				lm := &lineMessenger{
					lines: []any{},
				}
				lm.Context = map[string]any{"key": "value"}
				return lm
			},
			expected: true,
		},
		{
			name: "WithFieldMessage",
			setup: func() *lineMessenger {
				lm := &lineMessenger{
					lines: []any{"msg"},
				}
				lm.Context = map[string]any{"k": "v"}
				lm.setupField()
				return lm
			},
			expected: true,
		},
		{
			name: "Empty",
			setup: func() *lineMessenger {
				return &lineMessenger{
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
		setup      func() *lineMessenger
		wantSubstr []string
	}{
		{
			name: "SimpleLines",
			setup: func() *lineMessenger {
				return &lineMessenger{
					lines: []any{"line1", "line2"},
				}
			},
			wantSubstr: []string{"line1", "line2"},
		},
		{
			name: "WithFieldMessage",
			setup: func() *lineMessenger {
				lm := &lineMessenger{
					lines: []any{"test"},
				}
				lm.Context = map[string]any{"key": "value"}
				lm.setupField()
				return lm
			},
			wantSubstr: []string{"test"},
		},
		{
			name: "WithContextNoFieldMessage",
			setup: func() *lineMessenger {
				lm := &lineMessenger{
					lines: []any{"message"},
				}
				lm.Context = map[string]any{"field": "data"}
				return lm
			},
			wantSubstr: []string{"message"},
		},
		{
			name: "WithExistingMessage",
			setup: func() *lineMessenger {
				return &lineMessenger{
					Message: "already set",
					lines:   []any{"ignored"},
				}
			},
			wantSubstr: []string{"already set"},
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
		setup    func() *lineMessenger
		validate func(*testing.T, any)
	}{
		{
			name: "WithFieldMessage",
			setup: func() *lineMessenger {
				lm := &lineMessenger{
					lines: []any{"test"},
				}
				lm.Context = map[string]any{"key": "value"}
				lm.setupField()
				return lm
			},
			validate: func(t *testing.T, raw any) {
				if raw == nil {
					t.Error("Raw() should not return nil")
				}
			},
		},
		{
			name: "WithContextNoFieldMessage",
			setup: func() *lineMessenger {
				lm := &lineMessenger{
					lines: []any{"msg"},
				}
				lm.Context = map[string]any{"field": "data"}
				return lm
			},
			validate: func(t *testing.T, raw any) {
				if raw == nil {
					t.Error("should create fieldMessage and return raw")
				}
			},
		},
		{
			name: "WithIncludeMetadata",
			setup: func() *lineMessenger {
				lm := &lineMessenger{
					lines: []any{"test"},
				}
				lm.IncludeMetadata = true
				return lm
			},
			validate: func(t *testing.T, raw any) {
				if lm, ok := raw.(*lineMessenger); ok {
					if lm.Message == "" {
						t.Error("should resolve message")
					}
				} else {
					t.Error("with IncludeMetadata should return lineMessenger")
				}
			},
		},
		{
			name: "WithoutIncludeMetadata",
			setup: func() *lineMessenger {
				return &lineMessenger{
					lines: []any{"test message"},
				}
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
		setup    func() *lineMessenger
		opts     []Option
		validate func(*testing.T, *lineMessenger)
	}{
		{
			name: "WithoutFieldMessage",
			setup: func() *lineMessenger {
				return &lineMessenger{
					lines: []any{"test"},
				}
			},
			opts: []Option{OptionIncludeMetadata},
			validate: func(t *testing.T, lm *lineMessenger) {
				if !lm.IncludeMetadata {
					t.Error("should set IncludeMetadata on Base")
				}
			},
		},
		{
			name: "WithFieldMessage",
			setup: func() *lineMessenger {
				lm := &lineMessenger{
					lines: []any{"test"},
				}
				lm.Context = map[string]any{"k": "v"}
				lm.setupField()
				return lm
			},
			opts: []Option{OptionCollectInfo},
			validate: func(t *testing.T, lm *lineMessenger) {
				if lm.fm == nil {
					t.Error("should have fieldMessage")
				}
				if !lm.fm.CollectInfo {
					t.Error("should set CollectInfo on fieldMessage")
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
			c := newLinesFromStrings(tt.args)
			if c == nil {
				t.Fatal("newLinesFromStrings should not return nil")
			}
			tt.validate(t, c)
		})
	}
}
