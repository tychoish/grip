package message

import (
	"errors"
	"go/build"
	"strings"
	"testing"
)

func TestStackTraceString(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() StackTrace
		wantSubstr []string
	}{
		{
			name: "EmptyFrames",
			setup: func() StackTrace {
				return StackTrace{
					Frames: StackFrames{},
				}
			},
			wantSubstr: []string{},
		},
		{
			name: "SingleFrame",
			setup: func() StackTrace {
				// Use GOPATH-relative path to avoid panic
				gopath := "/go/src/github.com/user/repo"
				return StackTrace{
					Frames: StackFrames{
						{Function: "main.test", File: gopath + "/file.go", Line: 42},
					},
				}
			},
			wantSubstr: []string{"file.go", "42"},
		},
		{
			name: "MultipleFrames",
			setup: func() StackTrace {
				gopath := "/go/src/github.com/user/repo"
				return StackTrace{
					Frames: StackFrames{
						{Function: "main.test", File: gopath + "/file1.go", Line: 10},
						{Function: "main.other", File: gopath + "/file2.go", Line: 20},
						{Function: "main.another", File: gopath + "/file3.go", Line: 30},
					},
				}
			},
			wantSubstr: []string{"file1.go", "10", "file2.go", "20", "file3.go", "30"},
		},
		{
			name: "WithContext",
			setup: func() StackTrace {
				gopath := "/go/src/github.com/user/repo"
				return StackTrace{
					Context: "test context",
					Frames: StackFrames{
						{Function: "test", File: gopath + "/test.go", Line: 1},
					},
				}
			},
			wantSubstr: []string{"test.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			st := tt.setup()
			str := st.String()

			for _, substr := range tt.wantSubstr {
				if !strings.Contains(str, substr) {
					t.Errorf("String() = %q, should contain %q", str, substr)
				}
			}
		})
	}
}

func TestWrapStack(t *testing.T) {
	tests := []struct {
		name     string
		skip     int
		msg      any
		validate func(*testing.T, Composer)
	}{
		{
			name: "WithString",
			skip: 1,
			msg:  "test message",
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("WrapStack should be loggable")
				}
				if !c.Structured() {
					t.Error("WrapStack should be structured")
				}
				str := c.String()
				if str == "" {
					t.Error("String should not be empty")
				}
			},
		},
		{
			name: "WithNilMessage",
			skip: 1,
			msg:  nil,
			validate: func(t *testing.T, c Composer) {
				if c.Loggable() {
					t.Error("WrapStack with nil should not be loggable")
				}
			},
		},
		{
			name: "WithError",
			skip: 1,
			msg:  errors.New("test error"),
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("WrapStack with error should be loggable")
				}
				str := c.String()
				if !strings.Contains(str, "test error") {
					t.Error("should contain error message")
				}
			},
		},
		{
			name: "WithComposer",
			skip: 1,
			msg:  MakeString("composer message"),
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("should be loggable")
				}
			},
		},
		{
			name: "WithZeroSkip",
			skip: 0,
			msg:  "message",
			validate: func(t *testing.T, c Composer) {
				str := c.String()
				if str == "" {
					t.Error("should have stack trace")
				}
			},
		},
		{
			name: "WithNegativeSkip",
			skip: -5,
			msg:  "message",
			validate: func(t *testing.T, c Composer) {
				str := c.String()
				if str == "" {
					t.Error("should handle negative skip")
				}
			},
		},
		{
			name: "WithLargeSkip",
			skip: 100,
			msg:  "message",
			validate: func(t *testing.T, c Composer) {
				// Should not panic with large skip
				_ = c.String()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := WrapStack(tt.skip, tt.msg)
			if c == nil {
				t.Fatal("WrapStack should not return nil")
			}
			tt.validate(t, c)
		})
	}
}

func TestStackMessageString(t *testing.T) {
	tests := []struct {
		name       string
		setup      func() *stackMessage
		wantSubstr []string
	}{
		{
			name: "SimpleMessage",
			setup: func() *stackMessage {
				return &stackMessage{
					Composer: MakeString("test message"),
					trace:    captureStack(1),
				}
			},
			wantSubstr: []string{"test message"},
		},
		{
			name: "EmptyMessage",
			setup: func() *stackMessage {
				return &stackMessage{
					Composer: MakeString(""),
					trace:    captureStack(1),
				}
			},
			wantSubstr: []string{},
		},
		{
			name: "WithStackTrace",
			setup: func() *stackMessage {
				gopath := "/go/src/github.com/user/repo"
				return &stackMessage{
					Composer: MakeString("msg"),
					trace: StackFrames{
						{Function: "test", File: gopath + "/test.go", Line: 1},
					},
				}
			},
			wantSubstr: []string{"test.go", "1", "msg"},
		},
		{
			name: "CachedString",
			setup: func() *stackMessage {
				sm := &stackMessage{
					Composer: MakeString("cached"),
					trace:    captureStack(1),
				}
				// Call String once to cache it
				_ = sm.String()
				return sm
			},
			wantSubstr: []string{"cached"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := tt.setup()
			str := sm.String()

			for _, substr := range tt.wantSubstr {
				if !strings.Contains(str, substr) {
					t.Errorf("String() = %q, should contain %q", str, substr)
				}
			}

			// Call String again to test caching
			str2 := sm.String()
			if str != str2 {
				t.Error("String() should return same result when cached")
			}
		})
	}
}

func TestStackMessageStructured(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *stackMessage
		expected bool
	}{
		{
			name: "AlwaysTrue",
			setup: func() *stackMessage {
				return &stackMessage{
					Composer: MakeString("test"),
					trace:    captureStack(1),
				}
			},
			expected: true,
		},
		{
			name: "WithStructuredComposer",
			setup: func() *stackMessage {
				return &stackMessage{
					Composer: BuildKV().KV("key", "value"),
					trace:    captureStack(1),
				}
			},
			expected: true,
		},
		{
			name: "WithNonStructuredComposer",
			setup: func() *stackMessage {
				return &stackMessage{
					Composer: MakeString("plain"),
					trace:    captureStack(1),
				}
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := tt.setup()
			result := sm.Structured()
			if result != tt.expected {
				t.Errorf("Structured() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStackMessageRaw(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *stackMessage
		validate func(*testing.T, any)
	}{
		{
			name: "WithStructuredComposer",
			setup: func() *stackMessage {
				gopath := "/go/src/github.com/user/repo"
				return &stackMessage{
					Composer: BuildKV().KV("key", "value"),
					trace: StackFrames{
						{Function: "test", File: gopath + "/test.go", Line: 1},
					},
				}
			},
			validate: func(t *testing.T, raw any) {
				if raw == nil {
					t.Error("Raw() should not return nil")
				}
				// Should annotate composer with stack.frames
			},
		},
		{
			name: "WithNonStructuredComposer",
			setup: func() *stackMessage {
				gopath := "/go/src/github.com/user/repo"
				return &stackMessage{
					Composer: MakeString("plain message"),
					trace: StackFrames{
						{Function: "test", File: gopath + "/test.go", Line: 1},
					},
				}
			},
			validate: func(t *testing.T, raw any) {
				st, ok := raw.(StackTrace)
				if !ok {
					t.Fatalf("Raw() should return StackTrace, got %T", raw)
				}
				if len(st.Frames) == 0 {
					t.Error("StackTrace should have frames")
				}
				if st.Context == nil {
					t.Error("StackTrace should have context")
				}
			},
		},
		{
			name: "MultipleRawCalls",
			setup: func() *stackMessage {
				gopath := "/go/src/github.com/user/repo"
				return &stackMessage{
					Composer: BuildKV().KV("k", "v"),
					trace:    StackFrames{{Function: "test", File: gopath + "/test.go", Line: 1}},
				}
			},
			validate: func(t *testing.T, raw any) {
				// First call annotates, second call should still work
				if raw == nil {
					t.Error("Raw() should not return nil")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sm := tt.setup()
			raw := sm.Raw()
			tt.validate(t, raw)

			// Call Raw again to test idempotency
			raw2 := sm.Raw()
			if raw2 == nil {
				t.Error("second Raw() call should not return nil")
			}
		})
	}
}

func TestStackFramesString(t *testing.T) {
	tests := []struct {
		name       string
		frames     StackFrames
		wantSubstr []string
	}{
		{
			name:       "EmptyFrames",
			frames:     StackFrames{},
			wantSubstr: []string{},
		},
		{
			name: "SingleFrame",
			frames: StackFrames{
				{Function: "main.test", File: "/go/src/github.com/user/repo/file.go", Line: 42},
			},
			wantSubstr: []string{"file.go", "42"},
		},
		{
			name: "MultipleFrames",
			frames: StackFrames{
				{Function: "pkg.Func1", File: "/go/src/github.com/user/repo/file1.go", Line: 10},
				{Function: "pkg.Func2", File: "/go/src/github.com/user/repo/file2.go", Line: 20},
			},
			wantSubstr: []string{"file1.go", "10", "file2.go", "20"},
		},
		{
			name: "FrameWithLongFunction",
			frames: StackFrames{
				{Function: "github.com/user/repo/package.Function", File: "/go/src/github.com/user/repo/file.go", Line: 5},
			},
			wantSubstr: []string{"file.go", "5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.frames.String()

			for _, substr := range tt.wantSubstr {
				if !strings.Contains(str, substr) {
					t.Errorf("String() = %q, should contain %q", str, substr)
				}
			}
		})
	}
}

func TestStackFrameString(t *testing.T) {
	tests := []struct {
		name       string
		frame      StackFrame
		wantSubstr []string
		notWant    []string
	}{
		{
			name: "SimpleFrame",
			frame: StackFrame{
				Function: "main.test",
				File:     "/go/src/github.com/user/repo/file.go",
				Line:     42,
			},
			wantSubstr: []string{"file.go", "42"},
			notWant:    []string{},
		},
		{
			name: "FrameInGoRoot",
			frame: StackFrame{
				Function: "runtime.main",
				File:     build.Default.GOROOT + "/src/runtime/proc.go",
				Line:     250,
			},
			wantSubstr: []string{"/src/runtime/proc.go", "250"},
			notWant:    []string{},
		},
		{
			name: "FrameWithPackagePath",
			frame: StackFrame{
				Function: "github.com/user/repo.Function",
				File:     build.Default.GOPATH + "/src/github.com/user/repo/file.go",
				Line:     100,
			},
			wantSubstr: []string{"file.go", "100", "Function"},
			notWant:    []string{},
		},
		{
			name: "FrameWithNestedFunction",
			frame: StackFrame{
				Function: "package.Type.Method",
				File:     "/go/src/github.com/user/repo/file.go",
				Line:     1,
			},
			wantSubstr: []string{"Method", "file.go", "1"},
			notWant:    []string{},
		},
		{
			name: "FrameWithEmptyFunction",
			frame: StackFrame{
				Function: "",
				File:     "/go/src/github.com/user/repo/file.go",
				Line:     1,
			},
			wantSubstr: []string{"file.go", "1"},
			notWant:    []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			str := tt.frame.String()

			for _, substr := range tt.wantSubstr {
				if !strings.Contains(str, substr) {
					t.Errorf("String() = %q, should contain %q", str, substr)
				}
			}

			for _, substr := range tt.notWant {
				if strings.Contains(str, substr) {
					t.Errorf("String() = %q, should not contain %q", str, substr)
				}
			}
		})
	}
}

func TestMakeStack(t *testing.T) {
	tests := []struct {
		name     string
		skip     int
		message  string
		validate func(*testing.T, Composer)
	}{
		{
			name:    "BasicMessage",
			skip:    1,
			message: "test message",
			validate: func(t *testing.T, c Composer) {
				if !c.Loggable() {
					t.Error("should be loggable")
				}
				str := c.String()
				if !strings.Contains(str, "test message") {
					t.Error("should contain message")
				}
			},
		},
		{
			name:    "EmptyMessage",
			skip:    1,
			message: "",
			validate: func(t *testing.T, c Composer) {
				if c.Loggable() {
					t.Error("empty message should not be loggable")
				}
			},
		},
		{
			name:    "WithSkipZero",
			skip:    0,
			message: "msg",
			validate: func(t *testing.T, c Composer) {
				str := c.String()
				if str == "" {
					t.Error("should have content")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := MakeStack(tt.skip, tt.message)
			if c == nil {
				t.Fatal("MakeStack should not return nil")
			}
			tt.validate(t, c)
		})
	}
}
