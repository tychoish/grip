package message

import (
	"errors"
	"strings"
	"testing"
)

func TestWrapConstructors(t *testing.T) {
	tests := []struct {
		name     string
		create   func() Composer
		validate func(*testing.T, Composer)
	}{
		{
			name: "WrapWithNilParent",
			create: func() Composer {
				return Wrap(nil, "wrapper message")
			},
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("Wrap with message should be loggable")
				}
			},
		},
		{
			name: "WrapComposerWithString",
			create: func() Composer {
				base := MakeError(errors.New("base error"))
				return Wrap(base, "context")
			},
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("Wrap with error should be loggable")
				}
				str := c.String()
				if str == "" {
					t.Error("wrapped error should have non-empty string")
				}
			},
		},
		{
			name: "WrapWithComposerAndError",
			create: func() Composer {
				base := MakeString("base message")
				return Wrap(base, errors.New("error"))
			},
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("should be loggable")
				}
			},
		},
		{
			name: "WrapWithMultipleWraps",
			create: func() Composer {
				base := MakeError(errors.New("err1"))
				wrapped1 := Wrap(base, "context1")
				wrapped2 := Wrap(wrapped1, "context2")
				return wrapped2
			},
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("should be loggable")
				}
			},
		},
		{
			name: "WrapNilWithNil",
			create: func() Composer {
				return Wrap(nil, nil)
			},
			validate: func(t *testing.T, c Composer) {
				// Should not panic
				_ = c.String()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.create()
			if c == nil {
				t.Fatal("constructor should not return nil")
			}
			tt.validate(t, c)
		})
	}
}

func TestIsMulti(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() Composer
		expected bool
	}{
		{
			name: "SimpleComposer",
			setup: func() Composer {
				return MakeError(errors.New("simple"))
			},
			expected: false,
		},
		{
			name: "JoinedErrors",
			setup: func() Composer {
				seq := func(yield func(error) bool) {
					yield(errors.New("e1"))
					yield(errors.New("e2"))
				}
				return JoinErrors(seq)
			},
			expected: false, // JoinErrors returns errorMessage, not a wrapped/grouped composer
		},
		{
			name: "SingleError",
			setup: func() Composer {
				seq := func(yield func(error) bool) {
					yield(errors.New("single"))
				}
				return JoinErrors(seq)
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := tt.setup()
			result := IsMulti(comp)
			if result != tt.expected {
				t.Errorf("IsMulti() = %v, want %v for %v", result, tt.expected, tt.name)
			}
		})
	}
}

func TestUnwindFunction(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() Composer
		validate func(*testing.T, []Composer)
	}{
		{
			name: "SimpleComposer",
			setup: func() Composer {
				return MakeError(errors.New("simple"))
			},
			validate: func(t *testing.T, composers []Composer) {
				if len(composers) != 1 {
					t.Errorf("should have 1 composer, got %d", len(composers))
				}
			},
		},
		{
			name: "WrappedComposer",
			setup: func() Composer {
				base := MakeError(errors.New("base"))
				return Wrap(base, "wrapper")
			},
			validate: func(t *testing.T, composers []Composer) {
				if len(composers) < 1 {
					t.Error("should have unwrapped composers")
				}
			},
		},
		{
			name: "MultipleWraps",
			setup: func() Composer {
				c1 := MakeString("msg1")
				c2 := Wrap(c1, "msg2")
				c3 := Wrap(c2, "msg3")
				return c3
			},
			validate: func(t *testing.T, composers []Composer) {
				if len(composers) < 2 {
					t.Error("should unwind multiple levels")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := tt.setup()
			result := Unwind(comp)
			tt.validate(t, result)
		})
	}
}

func TestWrapLoggable(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() Composer
		expected bool
	}{
		{
			name: "WrapNilComposer",
			setup: func() Composer {
				return Wrap(nil, nil)
			},
			expected: false,
		},
		{
			name: "WrapLoggableComposer",
			setup: func() Composer {
				return Wrap(MakeError(errors.New("test")), "context")
			},
			expected: true,
		},
		{
			name: "WrapNonLoggableComposer",
			setup: func() Composer {
				return Wrap(MakeString(""), nil)
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.setup()
			result := c.Loggable()
			if result != tt.expected {
				t.Errorf("Loggable() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestWrapString(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() Composer
		wantSubstr []string
	}{
		{
			name: "WrapWithMessage",
			setup: func() Composer {
				base := MakeError(errors.New("base error"))
				return Wrap(base, "wrapper context")
			},
			wantSubstr: []string{"base error"},
		},
		{
			name: "WrapWithoutMessage",
			setup: func() Composer {
				return Wrap(MakeString("just msg"), nil)
			},
			wantSubstr: []string{"just msg"},
		},
		{
			name: "MultipleWraps",
			setup: func() Composer {
				c := MakeString("original")
				c = Wrap(c, "layer1")
				c = Wrap(c, "layer2")
				return c
			},
			wantSubstr: []string{"original"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.setup()
			str := c.String()

			for _, substr := range tt.wantSubstr {
				if !strings.Contains(str, substr) {
					t.Errorf("String() = %q, should contain %q", str, substr)
				}
			}
		})
	}
}
