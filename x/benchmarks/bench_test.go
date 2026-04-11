// Package benchmarks_test measures the performance characteristics of the grip
// logging library. It covers:
//
//   - Sender implementations: nop, io.Discard writer, file, stdlog, slog, zap, zerolog
//   - Message types: string (short/long), format, lines, fields, KV, error, bytes
//   - MessageFormatter implementations: plain, default, JSON, callsite
//   - Native loggers (no grip layer) for apples-to-apples comparison
//   - Parallel throughput for the most common sender/message combinations
//
// Run with:
//
//	go test -bench=. -benchmem ./...
package benchmarks_test

import (
	"context"
	"errors"
	"io"
	"log"
	"log/slog"
	"os"
	"testing"

	"github.com/rs/zerolog"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
	gripslog "github.com/tychoish/grip/x/slog"
	"github.com/tychoish/grip/x/stdlog"
	gripzap "github.com/tychoish/grip/x/zap"
	gripzerolog "github.com/tychoish/grip/x/zerolog"
)

// ── constants ────────────────────────────────────────────────────────────────

const (
	shortMsg = "hello world"
	longMsg  = "this is a longer log message that carries substantial content, " +
		"simulating real-world scenarios where lines include request details, " +
		"identifiers, durations, status codes, and other contextual information"
)

var errBench = errors.New("benchmark test error: operation failed unexpectedly")

// ── sender constructors ──────────────────────────────────────────────────────

func discardSender() send.Sender {
	s := send.MakeWriter(io.Discard)
	s.SetPriority(level.Trace)
	return s
}

func fileSender(b *testing.B) (send.Sender, func()) {
	b.Helper()
	f, err := os.CreateTemp(b.TempDir(), "grip-bench-*.log")
	if err != nil {
		b.Fatal(err)
	}
	_ = f.Close()
	s, err := send.MakeFile(f.Name())
	if err != nil {
		b.Fatal(err)
	}
	s.SetPriority(level.Trace)
	return s, func() { _ = s.Close() }
}

func stdlogSender() send.Sender {
	s := stdlog.WrapWriter(io.Discard)
	s.SetPriority(level.Trace)
	return s
}

func slogSender() send.Sender {
	h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
	s := gripslog.MakeSender(context.Background(), slog.New(h))
	s.SetPriority(level.Trace)
	return s
}

func zapSender() send.Sender {
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		zapcore.AddSync(io.Discard),
		zapcore.DebugLevel,
	)
	s := gripzap.MakeSender(zap.New(core))
	s.SetPriority(level.Trace)
	return s
}

func zerologSender() send.Sender {
	s := gripzerolog.MakeSender(zerolog.New(io.Discard))
	s.SetPriority(level.Trace)
	return s
}

// withInfo sets level.Info on any Composer and returns it.
func withInfo(m message.Composer) message.Composer {
	m.SetPriority(level.Info)
	return m
}

// ── BenchmarkSenders ─────────────────────────────────────────────────────────

// BenchmarkSenders measures per-Send throughput for each Sender implementation
// using a pre-composed short string message at level.Info.
func BenchmarkWithDiscardSenderBackends(b *testing.B) {
	msg := withInfo(message.MakeString(shortMsg))

	cases := []struct {
		name  string
		setup func(b *testing.B) (send.Sender, func())
	}{
		{
			name: "nop",
			setup: func(b *testing.B) (send.Sender, func()) {
				s := send.NopSender()
				s.SetPriority(level.Trace)
				return s, func() { _ = s.Close() }
			},
		},
		{
			name: "devnull",
			setup: func(b *testing.B) (send.Sender, func()) {
				s := discardSender()
				return s, func() { _ = s.Close() }
			},
		},
		{
			name: "tempfile",
			setup: func(b *testing.B) (send.Sender, func()) {
				return fileSender(b)
			},
		},
		{
			name: "stdlog",
			setup: func(b *testing.B) (send.Sender, func()) {
				s := stdlogSender()
				return s, func() { _ = s.Close() }
			},
		},
		{
			name: "slog",
			setup: func(b *testing.B) (send.Sender, func()) {
				s := slogSender()
				return s, func() { _ = s.Close() }
			},
		},
		{
			name: "zap",
			setup: func(b *testing.B) (send.Sender, func()) {
				s := zapSender()
				return s, func() { _ = s.Close() }
			},
		},
		{
			name: "zerolog",
			setup: func(b *testing.B) (send.Sender, func()) {
				s := zerologSender()
				return s, func() { _ = s.Close() }
			},
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			sender, cleanup := tc.setup(b)
			defer cleanup()
			b.ResetTimer()
			for b.Loop() {
				sender.Send(msg)
			}
		})
	}
}

// BenchmarkSendersStructured repeats BenchmarkSenders with a structured KV
// message so that the structured code paths in slog/zap/zerolog senders are
// exercised.
func BenchmarkSendersStructured(b *testing.B) {
	msg := message.NewKV().
		KV("msg", "benchmark event").
		KV("component", "benchmarks").
		KV("count", 42).
		Level(level.Info)

	cases := []struct {
		name  string
		setup func(b *testing.B) (send.Sender, func())
	}{
		{
			name: "devnull",
			setup: func(b *testing.B) (send.Sender, func()) {
				s := discardSender()
				return s, func() { _ = s.Close() }
			},
		},
		{
			name: "stdlog",
			setup: func(b *testing.B) (send.Sender, func()) {
				s := stdlogSender()
				return s, func() { _ = s.Close() }
			},
		},
		{
			name: "slog",
			setup: func(b *testing.B) (send.Sender, func()) {
				s := slogSender()
				return s, func() { _ = s.Close() }
			},
		},
		{
			name: "zap",
			setup: func(b *testing.B) (send.Sender, func()) {
				s := zapSender()
				return s, func() { _ = s.Close() }
			},
		},
		{
			name: "zerolog",
			setup: func(b *testing.B) (send.Sender, func()) {
				s := zerologSender()
				return s, func() { _ = s.Close() }
			},
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			sender, cleanup := tc.setup(b)
			defer cleanup()
			b.ResetTimer()
			for b.Loop() {
				sender.Send(msg)
			}
		})
	}
}

// ── BenchmarkMessageTypes ────────────────────────────────────────────────────

// BenchmarkMessageTypes measures the cost of constructing and sending different
// message types. A devnull sender is used so sender overhead is minimal.
//
// A new Composer is created on every iteration to include construction cost.
func BenchmarkMessageTypes(b *testing.B) {
	sender := discardSender()
	defer func() { _ = sender.Close() }()

	cases := []struct {
		name string
		make func() message.Composer
	}{
		{
			name: "string/short",
			make: func() message.Composer { return withInfo(message.MakeString(shortMsg)) },
		},
		{
			name: "string/long",
			make: func() message.Composer { return withInfo(message.MakeString(longMsg)) },
		},
		{
			name: "format/short",
			make: func() message.Composer {
				return withInfo(message.MakeFormat("op=%s count=%d duration=%s", "read", 42, "1.5ms"))
			},
		},
		{
			name: "format/long",
			make: func() message.Composer {
				return withInfo(message.MakeFormat(
					"service=%s op=%s id=%s status=%d bytes=%d duration=%s host=%s",
					"api", "read", "req-abc-123", 200, 4096, "1.234ms", "host-01",
				))
			},
		},
		{
			name: "lines",
			make: func() message.Composer {
				return withInfo(message.MakeLines("line one", "line two", "line three"))
			},
		},
		{
			name: "error",
			make: func() message.Composer { return withInfo(message.MakeError(errBench)) },
		},
		{
			name: "bytes",
			make: func() message.Composer {
				return withInfo(message.MakeBytes([]byte("raw bytes payload for logging")))
			},
		},
		{
			name: "fields/small",
			make: func() message.Composer {
				return withInfo(message.MakeFields(message.Fields{
					"msg":  "small fields message",
					"key1": "value1",
					"key2": 42,
				}))
			},
		},
		{
			name: "fields/large",
			make: func() message.Composer {
				return withInfo(message.MakeFields(message.Fields{
					"msg":    "large fields message",
					"svc":    "api-server",
					"op":     "process_request",
					"req_id": "req-abc-123",
					"user":   "alice",
					"status": 200,
					"bytes":  4096,
					"dur_ms": 1.234,
					"host":   "host-01",
					"region": "us-east-1",
				}))
			},
		},
		{
			name: "kv/small",
			make: func() message.Composer {
				return message.NewKV().
					KV("msg", "small kv message").
					KV("key1", "value1").
					KV("key2", 42).
					Level(level.Info)
			},
		},
		{
			name: "kv/large",
			make: func() message.Composer {
				return message.NewKV().
					KV("msg", "large kv message").
					KV("svc", "api-server").
					KV("op", "process_request").
					KV("req_id", "req-abc-123").
					KV("user", "alice").
					KV("status", 200).
					KV("bytes", 4096).
					KV("dur_ms", 1.234).
					KV("host", "host-01").
					KV("region", "us-east-1").
					Level(level.Info)
			},
		},
	}

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			for b.Loop() {
				sender.Send(tc.make())
			}
		})
	}
}

// ── BenchmarkFormatters ──────────────────────────────────────────────────────

// BenchmarkFormatters measures the formatting overhead for each
// send.MessageFormatter applied to the same devnull sender and a fixed short
// string message.
//
// The devnull sender calls Format(m) on every Send, so formatter cost is
// directly captured.
func BenchmarkFormatters(b *testing.B) {
	cases := []struct {
		name      string
		formatter send.MessageFormatter
	}{
		{name: "plain", formatter: send.MakePlainFormatter()},
		{name: "default", formatter: send.MakeDefaultFormatter()},
		{name: "json", formatter: send.MakeJSONFormatter()},
		{name: "callsite", formatter: send.MakeCallSiteFormatter(1)},
	}

	msg := withInfo(message.MakeString(shortMsg))

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			s := discardSender()
			defer func() { _ = s.Close() }()
			s.SetFormatter(tc.formatter)
			b.ResetTimer()
			for b.Loop() {
				s.Send(msg)
			}
		})
	}
}

// BenchmarkFormattersStructured repeats BenchmarkFormatters with a KV (structured)
// message to exercise JSON and plain formatters' structured code paths.
func BenchmarkFormattersStructured(b *testing.B) {
	cases := []struct {
		name      string
		formatter send.MessageFormatter
	}{
		{name: "plain", formatter: send.MakePlainFormatter()},
		{name: "default", formatter: send.MakeDefaultFormatter()},
		{name: "json", formatter: send.MakeJSONFormatter()},
	}

	msg := message.NewKV().
		KV("msg", "benchmark event").
		KV("component", "benchmarks").
		KV("count", 42).
		KV("host", "host-01").
		Level(level.Info)

	for _, tc := range cases {
		b.Run(tc.name, func(b *testing.B) {
			s := discardSender()
			defer func() { _ = s.Close() }()
			s.SetFormatter(tc.formatter)
			b.ResetTimer()
			for b.Loop() {
				s.Send(msg)
			}
		})
	}
}

// ── BenchmarkNative ──────────────────────────────────────────────────────────

// BenchmarkNative benchmarks each logging library's own API directly, without
// any grip layer. These results serve as a lower-bound reference for the
// grip-wrapped sender benchmarks above.
func BenchmarkNative(b *testing.B) {
	b.Run("stdlib/printf", func(b *testing.B) {
		logger := log.New(io.Discard, "", 0)
		for b.Loop() {
			logger.Printf("%s", shortMsg)
		}
	})

	b.Run("stdlib/printf/structured", func(b *testing.B) {
		logger := log.New(io.Discard, "", 0)
		for b.Loop() {
			logger.Printf("msg=%q svc=%q count=%d dur=%s", "benchmark event", "api", 42, "1.5ms")
		}
	})

	b.Run("slog/text/string", func(b *testing.B) {
		h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
		logger := slog.New(h)
		for b.Loop() {
			logger.Info(shortMsg)
		}
	})

	b.Run("slog/text/structured", func(b *testing.B) {
		h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
		logger := slog.New(h)
		for b.Loop() {
			logger.Info("benchmark event",
				slog.String("svc", "api"),
				slog.Int("count", 42),
				slog.String("dur", "1.5ms"),
			)
		}
	})

	b.Run("slog/json/string", func(b *testing.B) {
		h := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
		logger := slog.New(h)
		for b.Loop() {
			logger.Info(shortMsg)
		}
	})

	b.Run("slog/json/structured", func(b *testing.B) {
		h := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
		logger := slog.New(h)
		for b.Loop() {
			logger.Info("benchmark event",
				slog.String("svc", "api"),
				slog.Int("count", 42),
				slog.String("dur", "1.5ms"),
			)
		}
	})

	b.Run("zap/string", func(b *testing.B) {
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(io.Discard),
			zapcore.DebugLevel,
		)
		logger := zap.New(core)
		for b.Loop() {
			logger.Info(shortMsg)
		}
	})

	b.Run("zap/structured", func(b *testing.B) {
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(io.Discard),
			zapcore.DebugLevel,
		)
		logger := zap.New(core)
		for b.Loop() {
			logger.Info("benchmark event",
				zap.String("svc", "api"),
				zap.Int("count", 42),
				zap.String("dur", "1.5ms"),
			)
		}
	})

	b.Run("zerolog/string", func(b *testing.B) {
		logger := zerolog.New(io.Discard)
		for b.Loop() {
			logger.Info().Msg(shortMsg)
		}
	})

	b.Run("zerolog/structured", func(b *testing.B) {
		logger := zerolog.New(io.Discard)
		for b.Loop() {
			logger.Info().
				Str("svc", "api").
				Int("count", 42).
				Str("dur", "1.5ms").
				Msg("benchmark event")
		}
	})
}

// ── BenchmarkHeadToHead ──────────────────────────────────────────────────────

// BenchmarkHeadToHead pairs grip-sender and native-API benchmarks that share
// the same underlying logger instance. Both sides write to io.Discard with
// identical encoder configuration, so the only variable is whether the call
// flows through grip's Send/Format path or goes directly to the logger's API.
//
// Each sub-benchmark is structured as:
//
//	<logger>/<payload>/<grip|native>
//
// Compare the grip and native numbers for the same logger+payload pair to
// measure the overhead that grip adds.
func BenchmarkHeadToHead(b *testing.B) {
	// shared KV fields used by every structured sub-benchmark
	structuredGripMsg := func() message.Composer {
		return message.NewKV().
			KV("msg", "benchmark event").
			KV("svc", "api").
			KV("count", 42).
			KV("dur", "1.5ms").
			Level(level.Info)
	}
	stringGripMsg := withInfo(message.MakeString(shortMsg))

	b.Run("stdlog", func(b *testing.B) {
		// Both sides use the same log.Logger writing to io.Discard.
		stdLogger := log.New(io.Discard, "", log.LstdFlags)
		sender := send.FromStandard(stdLogger)
		sender.SetPriority(level.Trace)
		defer func() { _ = sender.Close() }()

		b.Run("string/grip", func(b *testing.B) {
			for b.Loop() {
				sender.Send(stringGripMsg)
			}
		})
		b.Run("string/native", func(b *testing.B) {
			for b.Loop() {
				stdLogger.Print(shortMsg)
			}
		})
		// stdlog has no structured API; simulate with Printf key=value pairs.
		b.Run("structured/grip", func(b *testing.B) {
			msg := structuredGripMsg()
			for b.Loop() {
				sender.Send(msg)
			}
		})
		b.Run("structured/native", func(b *testing.B) {
			for b.Loop() {
				stdLogger.Printf("msg=%q svc=%q count=%d dur=%s",
					"benchmark event", "api", 42, "1.5ms")
			}
		})
	})

	b.Run("slog/text", func(b *testing.B) {
		h := slog.NewTextHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
		slogLogger := slog.New(h)
		sender := gripslog.MakeSender(context.Background(), slogLogger)
		sender.SetPriority(level.Trace)
		defer func() { _ = sender.Close() }()

		b.Run("string/grip", func(b *testing.B) {
			for b.Loop() {
				sender.Send(stringGripMsg)
			}
		})
		b.Run("string/native", func(b *testing.B) {
			for b.Loop() {
				slogLogger.Info(shortMsg)
			}
		})
		b.Run("structured/grip", func(b *testing.B) {
			msg := structuredGripMsg()
			for b.Loop() {
				sender.Send(msg)
			}
		})
		b.Run("structured/native", func(b *testing.B) {
			for b.Loop() {
				slogLogger.Info("benchmark event",
					slog.String("svc", "api"),
					slog.Int("count", 42),
					slog.String("dur", "1.5ms"),
				)
			}
		})
	})

	b.Run("slog/json", func(b *testing.B) {
		h := slog.NewJSONHandler(io.Discard, &slog.HandlerOptions{Level: slog.LevelDebug})
		slogLogger := slog.New(h)
		sender := gripslog.MakeSender(context.Background(), slogLogger)
		sender.SetPriority(level.Trace)
		defer func() { _ = sender.Close() }()

		b.Run("string/grip", func(b *testing.B) {
			for b.Loop() {
				sender.Send(stringGripMsg)
			}
		})
		b.Run("string/native", func(b *testing.B) {
			for b.Loop() {
				slogLogger.Info(shortMsg)
			}
		})
		b.Run("structured/grip", func(b *testing.B) {
			msg := structuredGripMsg()
			for b.Loop() {
				sender.Send(msg)
			}
		})
		b.Run("structured/native", func(b *testing.B) {
			for b.Loop() {
				slogLogger.Info("benchmark event",
					slog.String("svc", "api"),
					slog.Int("count", 42),
					slog.String("dur", "1.5ms"),
				)
			}
		})
	})

	b.Run("zap", func(b *testing.B) {
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
			zapcore.AddSync(io.Discard),
			zapcore.DebugLevel,
		)
		zapLogger := zap.New(core)
		sender := gripzap.MakeSender(zapLogger)
		sender.SetPriority(level.Trace)
		defer func() { _ = sender.Close() }()

		b.Run("string/grip", func(b *testing.B) {
			for b.Loop() {
				sender.Send(stringGripMsg)
			}
		})
		b.Run("string/native", func(b *testing.B) {
			for b.Loop() {
				zapLogger.Info(shortMsg)
			}
		})
		b.Run("structured/grip", func(b *testing.B) {
			msg := structuredGripMsg()
			for b.Loop() {
				sender.Send(msg)
			}
		})
		b.Run("structured/native", func(b *testing.B) {
			for b.Loop() {
				zapLogger.Info("benchmark event",
					zap.String("svc", "api"),
					zap.Int("count", 42),
					zap.String("dur", "1.5ms"),
				)
			}
		})
	})

	b.Run("zerolog", func(b *testing.B) {
		zerologLogger := zerolog.New(io.Discard)
		sender := gripzerolog.MakeSender(zerologLogger)
		sender.SetPriority(level.Trace)
		defer func() { _ = sender.Close() }()

		b.Run("string/grip", func(b *testing.B) {
			for b.Loop() {
				sender.Send(stringGripMsg)
			}
		})
		b.Run("string/native", func(b *testing.B) {
			for b.Loop() {
				zerologLogger.Info().Msg(shortMsg)
			}
		})
		b.Run("structured/grip", func(b *testing.B) {
			msg := structuredGripMsg()
			for b.Loop() {
				sender.Send(msg)
			}
		})
		b.Run("structured/native", func(b *testing.B) {
			for b.Loop() {
				zerologLogger.Info().
					Str("svc", "api").
					Int("count", 42).
					Str("dur", "1.5ms").
					Msg("benchmark event")
			}
		})
	})
}

// ── BenchmarkParallel ────────────────────────────────────────────────────────

// BenchmarkParallel measures concurrent Send throughput using GOMAXPROCS
// goroutines. This reveals lock contention and synchronisation costs in each
// Sender implementation.
func BenchmarkParallel(b *testing.B) {
	msg := withInfo(message.MakeString(shortMsg))

	cases := []struct {
		name   string
		sender send.Sender
	}{
		{name: "nop", sender: func() send.Sender { s := send.NopSender(); s.SetPriority(level.Trace); return s }()},
		{name: "devnull", sender: discardSender()},
		{name: "stdlog", sender: stdlogSender()},
		{name: "slog", sender: slogSender()},
		{name: "zap", sender: zapSender()},
		{name: "zerolog", sender: zerologSender()},
	}

	for _, tc := range cases {
		sender := tc.sender
		b.Run(tc.name, func(b *testing.B) {
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					sender.Send(msg)
				}
			})
		})
		_ = sender.Close()
	}
}
