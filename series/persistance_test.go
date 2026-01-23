package series

import (
	"bytes"
	"context"
	"errors"
	"io"
	"iter"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tychoish/fun/assert/check"
	"github.com/tychoish/fun/fn"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/grip/send"
)

// Test LoggerBackend functionality
func TestLoggerBackendFunctionality(t *testing.T) {
	tests := []struct {
		name      string
		metrics   []MetricPublisher
		validate  func(*testing.T, send.Sender)
		expectErr bool
	}{
		{
			name: "SingleMetric",
			metrics: []MetricPublisher{
				func(w io.Writer, r Renderer) error {
					buf := &bytes.Buffer{}
					r.Metric(
						buf,
						"test.metric",
						fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} }),
						42,
						time.Unix(1234567890, 0),
					)
					_, err := w.Write(buf.Bytes())
					return err
				},
			},
			validate: func(t *testing.T, s send.Sender) {
				// Sender should have received message
			},
		},
		{
			name: "MultipleMetrics",
			metrics: []MetricPublisher{
				func(w io.Writer, r Renderer) error {
					buf := &bytes.Buffer{}
					r.Metric(
						buf,
						"metric1",
						fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} }),
						10,
						time.Now(),
					)
					_, err := w.Write(buf.Bytes())
					return err
				},
				func(w io.Writer, r Renderer) error {
					buf := &bytes.Buffer{}
					r.Metric(
						buf,
						"metric2",
						fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} }),
						20,
						time.Now(),
					)
					_, err := w.Write(buf.Bytes())
					return err
				},
			},
		},
		{
			name: "ErrorInPublisher",
			metrics: []MetricPublisher{
				func(w io.Writer, r Renderer) error {
					return errors.New("publisher error")
				},
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sender := send.MakeInternal()
			renderer := MakeJSONRenderer()
			backend := LoggerBackend(sender, renderer)

			seq := func(yield func(MetricPublisher) bool) {
				for _, m := range tt.metrics {
					if !yield(m) {
						return
					}
				}
			}

			err := backend(context.Background(), seq)

			if tt.expectErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if tt.validate != nil {
					tt.validate(t, sender)
				}
			}
		})
	}
}

// Test PassthroughBackend
func TestPassthroughBackend(t *testing.T) {
	tests := []struct {
		name      string
		metrics   []MetricPublisher
		validate  func(*testing.T, []string)
		expectErr bool
	}{
		{
			name: "BasicPassthrough",
			metrics: []MetricPublisher{
				func(w io.Writer, r Renderer) error {
					r.Metric(
						w.(*bytes.Buffer),
						"pass.through",
						fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} }),
						100,
						time.Unix(1234567890, 0),
					)
					return nil
				},
			},
			validate: func(t *testing.T, outputs []string) {
				if len(outputs) != 1 {
					t.Errorf("expected 1 output, got %d", len(outputs))
				}
				if len(outputs) > 0 && !strings.Contains(outputs[0], "pass.through") {
					t.Error("expected metric name in output")
				}
			},
		},
		{
			name: "MultipleMetrics",
			metrics: []MetricPublisher{
				func(w io.Writer, r Renderer) error {
					r.Metric(
						w.(*bytes.Buffer),
						"metric1",
						fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} }),
						10,
						time.Now(),
					)
					return nil
				},
				func(w io.Writer, r Renderer) error {
					r.Metric(
						w.(*bytes.Buffer),
						"metric2",
						fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} }),
						20,
						time.Now(),
					)
					return nil
				},
			},
			validate: func(t *testing.T, outputs []string) {
				if len(outputs) != 2 {
					t.Errorf("expected 2 outputs, got %d", len(outputs))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var outputs []string
			handler := fnx.NewHandler(func(ctx context.Context, s string) error {
				outputs = append(outputs, s)
				return nil
			})

			backend := PassthroughBackend(MakeJSONRenderer(), handler)

			seq := func(yield func(MetricPublisher) bool) {
				for _, m := range tt.metrics {
					if !yield(m) {
						return
					}
				}
			}

			err := backend(context.Background(), seq)

			if tt.expectErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if tt.validate != nil {
					tt.validate(t, outputs)
				}
			}
		})
	}
}

// Test CollectorBackendFileConf validation
func TestCollectorBackendFileConfValidation(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		conf        *CollectorBackendFileConf
		expectError bool
		errorMsg    string
	}{
		{
			name: "ValidConfiguration",
			conf: &CollectorBackendFileConf{
				Directory:      tmpDir,
				FilePrefix:     "metrics",
				Extension:      ".json",
				CounterPadding: 3,
				Megabytes:      10,
				Renderer:       MakeJSONRenderer(),
			},
			expectError: false,
		},
		{
			name: "MissingMegabytes",
			conf: &CollectorBackendFileConf{
				Directory:      tmpDir,
				FilePrefix:     "metrics",
				Extension:      ".json",
				CounterPadding: 3,
				Megabytes:      0,
				Renderer:       MakeJSONRenderer(),
			},
			expectError: true,
			errorMsg:    "rotation size",
		},
		{
			name: "MissingCounterPadding",
			conf: &CollectorBackendFileConf{
				Directory:      tmpDir,
				FilePrefix:     "metrics",
				Extension:      ".json",
				CounterPadding: 0,
				Megabytes:      10,
				Renderer:       MakeJSONRenderer(),
			},
			expectError: true,
			errorMsg:    "counter padding",
		},
		{
			name: "NonexistentDirectory",
			conf: &CollectorBackendFileConf{
				Directory:      "/nonexistent/path/that/does/not/exist",
				FilePrefix:     "metrics",
				Extension:      ".json",
				CounterPadding: 3,
				Megabytes:      10,
				Renderer:       MakeJSONRenderer(),
			},
			expectError: true, // Current validation requires directory to exist
			errorMsg:    "directory",
		},
		{
			name: "MissingPrefix",
			conf: &CollectorBackendFileConf{
				Directory:      tmpDir,
				FilePrefix:     "",
				Extension:      ".json",
				CounterPadding: 3,
				Megabytes:      10,
				Renderer:       MakeJSONRenderer(),
			},
			expectError: true,
			errorMsg:    "prefix",
		},
		{
			name: "MissingExtension",
			conf: &CollectorBackendFileConf{
				Directory:      tmpDir,
				FilePrefix:     "metrics",
				Extension:      "",
				CounterPadding: 3,
				Megabytes:      10,
				Renderer:       MakeJSONRenderer(),
			},
			expectError: true,
			errorMsg:    "prefix",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.conf.Validate()

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

// Test CollectorBackendFileConf option providers
func TestCollectorBackendFileConfOptions(t *testing.T) {
	tests := []struct {
		name     string
		apply    func(*CollectorBackendFileConf)
		validate func(*testing.T, *CollectorBackendFileConf)
	}{
		{
			name: "SetDirectory",
			apply: func(conf *CollectorBackendFileConf) {
				check.NotError(t, CollectorBackendFileConfDirectory("/tmp/metrics")(conf))
			},
			validate: func(t *testing.T, conf *CollectorBackendFileConf) {
				if conf.Directory != "/tmp/metrics" {
					t.Errorf("expected directory /tmp/metrics, got %s", conf.Directory)
				}
			},
		},
		{
			name: "SetPrefix",
			apply: func(conf *CollectorBackendFileConf) {
				check.NotError(t, CollectorBackendFileConfPrefix("app")(conf))
			},
			validate: func(t *testing.T, conf *CollectorBackendFileConf) {
				if conf.FilePrefix != "app" {
					t.Errorf("expected prefix 'app', got %s", conf.FilePrefix)
				}
			},
		},
		{
			name: "SetExtension",
			apply: func(conf *CollectorBackendFileConf) {
				check.NotError(t, CollectorBackendFileConfExtension(".log")(conf))
			},
			validate: func(t *testing.T, conf *CollectorBackendFileConf) {
				if conf.Extension != ".log" {
					t.Errorf("expected extension '.log', got %s", conf.Extension)
				}
			},
		},
		{
			name: "SetCounterPadding",
			apply: func(conf *CollectorBackendFileConf) {
				check.NotError(t, CollectorBackendFileConfCounterPadding(5)(conf))
			},
			validate: func(t *testing.T, conf *CollectorBackendFileConf) {
				if conf.CounterPadding != 5 {
					t.Errorf("expected counter padding 5, got %d", conf.CounterPadding)
				}
			},
		},
		{
			name: "SetRotationSize",
			apply: func(conf *CollectorBackendFileConf) {
				check.NotError(t, CollectorBackendFileConfRotationSizeMB(50)(conf))
			},
			validate: func(t *testing.T, conf *CollectorBackendFileConf) {
				if conf.Megabytes != 50 {
					t.Errorf("expected 50 MB, got %d", conf.Megabytes)
				}
			},
		},
		{
			name: "SetRenderer",
			apply: func(conf *CollectorBackendFileConf) {
				renderer := MakeGraphiteRenderer()
				check.NotError(t, CollectorBackendFileConfWithRenderer(renderer)(conf))
			},
			validate: func(t *testing.T, conf *CollectorBackendFileConf) {
				if conf.Renderer.Metric == nil {
					t.Error("expected renderer to be set")
				}
			},
		},
		{
			name: "SetComplete",
			apply: func(conf *CollectorBackendFileConf) {
				newConf := &CollectorBackendFileConf{
					Directory:      "/data/metrics",
					FilePrefix:     "series",
					Extension:      ".txt",
					CounterPadding: 4,
					Megabytes:      25,
					Gzip:           true,
					Renderer:       MakeJSONRenderer(),
				}
				check.NotError(t, CollectorBackendFileConfSet(newConf)(conf))
			},
			validate: func(t *testing.T, conf *CollectorBackendFileConf) {
				if conf.Directory != "/data/metrics" {
					t.Error("expected full configuration to be set")
				}
				if !conf.Gzip {
					t.Error("expected Gzip to be true")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &CollectorBackendFileConf{}
			tt.apply(conf)
			tt.validate(t, conf)
		})
	}
}

// Test CollectorBackendSocketConf validation
func TestCollectorBackendSocketConfValidation(t *testing.T) {
	tests := []struct {
		name        string
		conf        *CollectorBackendSocketConf
		expectError bool
		errorMsg    string
		validate    func(*testing.T, *CollectorBackendSocketConf)
	}{
		{
			name: "ValidTCPConfiguration",
			conf: &CollectorBackendSocketConf{
				Network:              "tcp",
				Address:              "localhost:8080",
				Renderer:             MakeJSONRenderer(),
				DialErrorHandling:    CollectorBackendSocketErrorContinue,
				MessageErrorHandling: CollectorBackendSocketErrorContinue,
			},
			expectError: false,
			validate: func(t *testing.T, conf *CollectorBackendSocketConf) {
				// Check defaults are set
				if conf.DialWorkers < 1 {
					t.Error("expected DialWorkers to be set to minimum 1")
				}
				if conf.MinDialRetryDelay < 100*time.Millisecond {
					t.Error("expected MinDialRetryDelay to have minimum value")
				}
			},
		},
		{
			name: "ValidUDPConfiguration",
			conf: &CollectorBackendSocketConf{
				Network:              "udp",
				Address:              "127.0.0.1:9090",
				Renderer:             MakeJSONRenderer(),
				DialErrorHandling:    CollectorBackendSocketErrorAbort,
				MessageErrorHandling: CollectorBackendSocketErrorCollect,
			},
			expectError: false,
		},
		{
			name: "InvalidNetwork",
			conf: &CollectorBackendSocketConf{
				Network:  "http",
				Address:  "localhost:8080",
				Renderer: MakeJSONRenderer(),
			},
			expectError: true,
			errorMsg:    "network",
		},
		{
			name: "MissingMetricRenderer",
			conf: &CollectorBackendSocketConf{
				Network: "tcp",
				Address: "localhost:8080",
				Renderer: Renderer{
					Histogram: MakeJSONRenderer().Histogram,
				},
			},
			expectError: true,
			errorMsg:    "scalar metrics renderer",
		},
		{
			name: "MissingHistogramRenderer",
			conf: &CollectorBackendSocketConf{
				Network: "tcp",
				Address: "localhost:8080",
				Renderer: Renderer{
					Metric: MakeJSONRenderer().Metric,
				},
			},
			expectError: true,
			errorMsg:    "histogram renderer",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.conf.Validate()

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("expected error containing %q, got %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}

				if tt.validate != nil {
					tt.validate(t, tt.conf)
				}
			}
		})
	}
}

// Test CollectorBackendSocketConf option providers
func TestCollectorBackendSocketConfOptions(t *testing.T) {
	tests := []struct {
		name     string
		apply    func(*CollectorBackendSocketConf)
		validate func(*testing.T, *CollectorBackendSocketConf)
	}{
		{
			name: "SetNetworkTCP",
			apply: func(conf *CollectorBackendSocketConf) {
				check.NotError(t, CollectorBackendSocketConfNetowrkTCP()(conf))
			},
			validate: func(t *testing.T, conf *CollectorBackendSocketConf) {
				if conf.Network != "tcp" {
					t.Errorf("expected network 'tcp', got %s", conf.Network)
				}
			},
		},
		{
			name: "SetNetworkUDP",
			apply: func(conf *CollectorBackendSocketConf) {
				check.NotError(t, CollectorBackendSocketConfNetowrkUDP()(conf))
			},
			validate: func(t *testing.T, conf *CollectorBackendSocketConf) {
				if conf.Network != "udp" {
					t.Errorf("expected network 'udp', got %s", conf.Network)
				}
			},
		},
		{
			name: "SetAddress",
			apply: func(conf *CollectorBackendSocketConf) {
				check.NotError(t, CollectorBackendSocketConfAddress("localhost:9999")(conf))
			},
			validate: func(t *testing.T, conf *CollectorBackendSocketConf) {
				if conf.Address != "localhost:9999" {
					t.Errorf("expected address 'localhost:9999', got %s", conf.Address)
				}
			},
		},
		{
			name: "SetDialWorkers",
			apply: func(conf *CollectorBackendSocketConf) {
				check.NotError(t, CollectorBackendSocketConfDialWorkers(5)(conf))
			},
			validate: func(t *testing.T, conf *CollectorBackendSocketConf) {
				if conf.DialWorkers != 5 {
					t.Errorf("expected 5 dial workers, got %d", conf.DialWorkers)
				}
			},
		},
		{
			name: "SetIdleConns",
			apply: func(conf *CollectorBackendSocketConf) {
				check.NotError(t, CollectorBackendSocketConfIdleConns(10)(conf))
			},
			validate: func(t *testing.T, conf *CollectorBackendSocketConf) {
				if conf.IdleConns != 10 {
					t.Errorf("expected 10 idle conns, got %d", conf.IdleConns)
				}
			},
		},
		{
			name: "SetMinDialRetryDelay",
			apply: func(conf *CollectorBackendSocketConf) {
				check.NotError(t, CollectorBackendSocketConfMinDialRetryDelay(500*time.Millisecond)(conf))
			},
			validate: func(t *testing.T, conf *CollectorBackendSocketConf) {
				if conf.MinDialRetryDelay != 500*time.Millisecond {
					t.Errorf("expected 500ms, got %v", conf.MinDialRetryDelay)
				}
			},
		},
		{
			name: "SetMaxDialRetryDelay",
			apply: func(conf *CollectorBackendSocketConf) {
				check.NotError(t, CollectorBackendSocketConfMaxDialRetryDelay(5*time.Second)(conf))
			},
			validate: func(t *testing.T, conf *CollectorBackendSocketConf) {
				if conf.MaxDialRetryDelay != 5*time.Second {
					t.Errorf("expected 5s, got %v", conf.MaxDialRetryDelay)
				}
			},
		},
		{
			name: "SetMessageWorkers",
			apply: func(conf *CollectorBackendSocketConf) {
				check.NotError(t, CollectorBackendSocketConfMessageWorkers(8)(conf))
			},
			validate: func(t *testing.T, conf *CollectorBackendSocketConf) {
				if conf.MessageWorkers != 8 {
					t.Errorf("expected 8 message workers, got %d", conf.MessageWorkers)
				}
			},
		},
		{
			name: "SetNumMessageRetries",
			apply: func(conf *CollectorBackendSocketConf) {
				check.NotError(t, CollectorBackendSocketConfNumMessageRetries(3)(conf))
			},
			validate: func(t *testing.T, conf *CollectorBackendSocketConf) {
				if conf.NumMessageRetries != 3 {
					t.Errorf("expected 3 retries, got %d", conf.NumMessageRetries)
				}
			},
		},
		{
			name: "SetMinMessageRetryDelay",
			apply: func(conf *CollectorBackendSocketConf) {
				check.NotError(t, CollectorBackendSocketConfMinMessageRetryDelay(200*time.Millisecond)(conf))
			},
			validate: func(t *testing.T, conf *CollectorBackendSocketConf) {
				if conf.MinMessageRetryDelay != 200*time.Millisecond {
					t.Errorf("expected 200ms, got %v", conf.MinMessageRetryDelay)
				}
			},
		},
		{
			name: "SetMaxMessageRetryDelay",
			apply: func(conf *CollectorBackendSocketConf) {
				check.NotError(t, CollectorBackendSocketConfMaxMessageRetryDelay(10*time.Second)(conf))
			},
			validate: func(t *testing.T, conf *CollectorBackendSocketConf) {
				if conf.MaxMessageRetryDelay != 10*time.Second {
					t.Errorf("expected 10s, got %v", conf.MaxMessageRetryDelay)
				}
			},
		},
		{
			name: "SetRenderer",
			apply: func(conf *CollectorBackendSocketConf) {
				renderer := MakeOpenTSBLineRenderer()
				check.NotError(t, CollectorBackendSocketConfWithRenderer(renderer)(conf))
			},
			validate: func(t *testing.T, conf *CollectorBackendSocketConf) {
				if conf.Renderer.Metric == nil {
					t.Error("expected renderer to be set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &CollectorBackendSocketConf{}
			tt.apply(conf)
			tt.validate(t, conf)
		})
	}
}

// Test CollectorBackend.Worker
func TestCollectorBackendWorker(t *testing.T) {
	backend := LoggerBackend(send.MakeInternal(), MakeJSONRenderer())

	metrics := []MetricPublisher{
		func(w io.Writer, r Renderer) error {
			buf := &bytes.Buffer{}
			r.Metric(
				buf,
				"test",
				fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} }),
				42,
				time.Now(),
			)
			_, err := w.Write(buf.Bytes())
			return err
		},
	}

	seq := func(yield func(MetricPublisher) bool) {
		for _, m := range metrics {
			if !yield(m) {
				return
			}
		}
	}

	worker := backend.Worker(seq)
	err := worker.Run(context.Background())
	if err != nil {
		t.Errorf("worker failed: %v", err)
	}
}

// Test FileBackend creation
func TestFileBackend(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		opts        []CollectorBakendFileOptionProvider
		expectError bool
	}{
		{
			name: "ValidBackend",
			opts: []CollectorBakendFileOptionProvider{
				CollectorBackendFileConfDirectory(tmpDir),
				CollectorBackendFileConfPrefix("test"),
				CollectorBackendFileConfExtension(".json"),
				CollectorBackendFileConfCounterPadding(3),
				CollectorBackendFileConfRotationSizeMB(1),
				CollectorBackendFileConfWithRenderer(MakeJSONRenderer()),
			},
			expectError: false,
		},
		{
			name: "InvalidConfiguration",
			opts: []CollectorBakendFileOptionProvider{
				CollectorBackendFileConfDirectory(tmpDir),
				// Missing required fields
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend, err := FileBackend(tt.opts...)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if backend == nil {
					t.Error("expected non-nil backend")
				}
			}
		})
	}
}

// Test RotatingFilePath
func TestRotatingFilePath(t *testing.T) {
	conf := &CollectorBackendFileConf{
		Directory:      "/tmp/metrics",
		FilePrefix:     "app",
		Extension:      ".log",
		CounterPadding: 3,
	}

	pathFuture := conf.RotatingFilePath()
	path, err := pathFuture(context.Background())
	if err != nil {
		t.Fatalf("unexpected error getting path: %v", err)
	}

	expectedPattern := filepath.Join("/tmp/metrics", "app")
	if !strings.HasPrefix(path, expectedPattern) {
		t.Errorf("expected path to start with %s, got %s", expectedPattern, path)
	}

	// Verify the path has the counter (e.g., "app000")
	// Note: RotatingFilePath doesn't append the extension - that's done elsewhere
	if !strings.Contains(path, "app0") {
		t.Errorf("expected path to contain counter, got %s", path)
	}

	// Call again to get next file
	nextPath, err := pathFuture(context.Background())
	if err != nil {
		t.Fatalf("unexpected error getting next path: %v", err)
	}
	if path == nextPath {
		t.Error("expected different paths for sequential calls")
	}
}
