package grip

import (
	"context"
	"testing"

	"github.com/tychoish/grip/send"
)

func TestContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if Context(ctx) != std {
		t.Fatal("context does not default to standard")
	}

	if Context(nil) != std { //nolint:staticcheck
		t.Fatal("context does not default to standard")
	}

	logger := NewLogger(send.MakeStdOutput())
	ctx = WithLogger(ctx, logger)

	if !HasContextLogger(ctx, string(defaultContextKey)) {
		t.Error(ctx)
	}

	if Context(ctx) == std {
		t.Fatal("context logger should not return standard if set")
	}

	if Context(ctx) != logger {
		t.Fatal("context should return expected value")
	}
}
