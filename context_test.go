package grip

import (
	"context"
	"testing"

	"github.com/tychoish/fun/assert/check"
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

	ctx = WithNewContextLogger(ctx, string(defaultContextKey), func() send.Sender { panic("should not panic") })
	check.True(t, HasContextLogger(ctx, string(defaultContextKey)))
	check.Panic(t, func() {
		ctx = WithNewContextLogger(ctx, "novel-key", func() send.Sender { panic("should not panic") })
	})

}
