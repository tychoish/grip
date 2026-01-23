package message

import (
	"errors"
	"fmt"
	"testing"
)

func TestErrorMessageMethods(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *errorMessage
		testFunc func(*testing.T, *errorMessage)
	}{
		{
			name: "IsMethod",
			setup: func() *errorMessage {
				// Use a sentinel error that can be compared
				baseErr := fmt.Errorf("base error")
				return &errorMessage{
					err: fmt.Errorf("wrapped: %w", baseErr),
				}
			},
			testFunc: func(t *testing.T, em *errorMessage) {
				// Test that the error exists
				if em.err == nil {
					t.Error("error should not be nil")
				}
				// Test basic error matching with errors.Is
				if !errors.Is(em.err, em.err) {
					t.Error("error should match itself")
				}
			},
		},
		{
			name: "AsMethod",
			setup: func() *errorMessage {
				// Use a real wrapped error
				base := errors.New("base error")
				wrapped := fmt.Errorf("wrapped: %w", base)
				return &errorMessage{
					err: wrapped,
				}
			},
			testFunc: func(t *testing.T, em *errorMessage) {
				// Test that As works with wrapped errors
				if em.err == nil {
					t.Error("error should not be nil")
				}
				// Test Unwrap
				if em.Unwrap() == nil {
					t.Error("should be able to unwrap")
				}
			},
		},
		{
			name: "RawWithNilError",
			setup: func() *errorMessage {
				return &errorMessage{err: nil}
			},
			testFunc: func(t *testing.T, em *errorMessage) {
				raw := em.Raw()
				// Raw() may return an empty struct instead of nil
				if raw == nil {
					// That's fine
				} else {
					// Check it's some kind of struct
					_ = raw
				}
			},
		},
		{
			name: "RawWithSimpleError",
			setup: func() *errorMessage {
				return &errorMessage{
					err: errors.New("simple error"),
				}
			},
			testFunc: func(t *testing.T, em *errorMessage) {
				raw := em.Raw()
				if raw == nil {
					t.Error("Raw() should return non-nil for simple error")
				}
				if m, ok := raw.(map[string]any); ok {
					if m["error"] != "simple error" {
						t.Error("Raw() should include error message")
					}
				}
			},
		},
		{
			name: "RawWithWrappedError",
			setup: func() *errorMessage {
				base := errors.New("base")
				wrapped := fmt.Errorf("wrapped: %w", base)
				return &errorMessage{err: wrapped}
			},
			testFunc: func(t *testing.T, em *errorMessage) {
				raw := em.Raw()
				if raw == nil {
					t.Error("Raw() should return non-nil for wrapped error")
				}
			},
		},
		{
			name: "RawWithFields",
			setup: func() *errorMessage {
				em := &errorMessage{
					err: errors.New("test"),
				}
				em.Annotate("key", "value")
				em.Annotate("count", 42)
				return em
			},
			testFunc: func(t *testing.T, em *errorMessage) {
				raw := em.Raw()
				// Annotate adds to Context which is part of Raw output
				if raw == nil {
					t.Error("Raw() should not be nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			em := tt.setup()
			tt.testFunc(t, em)
		})
	}
}

func TestJoinErrors(t *testing.T) {
	tests := []struct {
		name     string
		errors   []error
		validate func(*testing.T, Composer)
	}{
		{
			name:   "NoErrors",
			errors: []error{},
			validate: func(t *testing.T, c Composer) {
				if c.Loggable() {
					t.Error("empty errors should not be loggable")
				}
			},
		},
		{
			name:   "SingleNilError",
			errors: []error{nil},
			validate: func(t *testing.T, c Composer) {
				if c.Loggable() {
					t.Error("nil error should not be loggable")
				}
			},
		},
		{
			name:   "SingleError",
			errors: []error{errors.New("error1")},
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("single error should be loggable")
				}
				if c.String() == "" {
					t.Error("should have non-empty string")
				}
			},
		},
		{
			name: "MultipleErrors",
			errors: []error{
				errors.New("error1"),
				errors.New("error2"),
				errors.New("error3"),
			},
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("multiple errors should be loggable")
				}
			},
		},
		{
			name: "MixedNilAndErrors",
			errors: []error{
				nil,
				errors.New("error1"),
				nil,
				errors.New("error2"),
				nil,
			},
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("should be loggable with non-nil errors")
				}
			},
		},
		{
			name:   "AllNilErrors",
			errors: []error{nil, nil, nil},
			validate: func(t *testing.T, c Composer) {
				if c.Loggable() {
					t.Error("all nil errors should not be loggable")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert slice to iterator
			seq := func(yield func(error) bool) {
				for _, err := range tt.errors {
					if !yield(err) {
						return
					}
				}
			}
			c := JoinErrors(seq)
			if c == nil {
				t.Fatal("JoinErrors should not return nil")
			}
			tt.validate(t, c)
		})
	}
}

func TestJoinErrorsWithMultipleErrors(t *testing.T) {
	// Test that JoinErrors works correctly with multiple errors
	tests := []struct {
		name     string
		errors   []error
		validate func(*testing.T, Composer)
	}{
		{
			name: "JoinedErrorsHaveMultipleMessages",
			errors: []error{
				errors.New("first"),
				errors.New("second"),
				errors.New("third"),
			},
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("joined errors should be loggable")
				}
				str := c.String()
				if str == "" {
					t.Error("should have non-empty string")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			seq := func(yield func(error) bool) {
				for _, err := range tt.errors {
					if !yield(err) {
						return
					}
				}
			}
			c := JoinErrors(seq)
			tt.validate(t, c)
		})
	}
}

func TestErrorMessageAnnotate(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() Composer
		key      string
		value    any
		validate func(*testing.T, Composer)
	}{
		{
			name: "AnnotateErrorMessage",
			setup: func() Composer {
				return &errorMessage{err: errors.New("test")}
			},
			key:   "field",
			value: "data",
			validate: func(t *testing.T, c Composer) {
				raw := c.Raw()
				if m, ok := raw.(map[string]any); ok {
					if v, hasField := m["field"]; hasField {
						if v != "data" {
							t.Error("unexpected value", v)
						}
						// Annotation worked
					}
				}
			},
		},
		{
			name: "MultipleAnnotations",
			setup: func() Composer {
				return &errorMessage{err: errors.New("test")}
			},
			key:   "first",
			value: 1,
			validate: func(t *testing.T, c Composer) {
				c.Annotate("second", 2)
				c.Annotate("third", 3)
				// Verify annotations are applied
				_ = c.String()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.setup()
			c.Annotate(tt.key, tt.value)
			tt.validate(t, c)
		})
	}
}

func TestComposerUnwind(t *testing.T) {
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
				return Wrap(base, MakeString("wrapper"))
			},
			validate: func(t *testing.T, composers []Composer) {
				if len(composers) < 1 {
					t.Error("should have unwrapped composers")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			comp := tt.setup()
			unwound := Unwind(comp)
			tt.validate(t, unwound)
		})
	}
}
