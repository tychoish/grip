package series

import (
	"strings"
	"testing"
	"time"

	"github.com/tychoish/fun/assert/check"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip/level"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

// Tests for metric.go

func TestMetricConstructors(t *testing.T) {
	tests := []struct {
		name     string
		create   func() *Metric
		validate func(*testing.T, *Metric)
	}{
		{
			name:   "Collect",
			create: func() *Metric { return Collect("test-metric") },
			validate: func(t *testing.T, m *Metric) {
				if m.ID != "test-metric" {
					t.Errorf("expected ID 'test-metric', got %q", m.ID)
				}
				if m.Type != "" {
					t.Errorf("expected empty type, got %q", m.Type)
				}
			},
		},
		{
			name:   "Gauge",
			create: func() *Metric { return Gauge("gauge-metric") },
			validate: func(t *testing.T, m *Metric) {
				if m.ID != "gauge-metric" {
					t.Error("expected ID to be set")
				}
				if m.Type != MetricTypeGuage {
					t.Errorf("expected gauge type, got %q", m.Type)
				}
			},
		},
		{
			name:   "Counter",
			create: func() *Metric { return Counter("counter-metric") },
			validate: func(t *testing.T, m *Metric) {
				if m.ID != "counter-metric" {
					t.Error("expected ID to be set")
				}
				if m.Type != MetricTypeCounter {
					t.Errorf("expected counter type, got %q", m.Type)
				}
			},
		},
		{
			name:   "Delta",
			create: func() *Metric { return Delta("delta-metric") },
			validate: func(t *testing.T, m *Metric) {
				if m.ID != "delta-metric" {
					t.Error("expected ID to be set")
				}
				if m.Type != MetricTypeDeltas {
					t.Errorf("expected deltas type, got %q", m.Type)
				}
			},
		},
		{
			name:   "Histogram",
			create: func() *Metric { return Histogram("histogram-metric") },
			validate: func(t *testing.T, m *Metric) {
				if m.ID != "histogram-metric" {
					t.Error("expected ID to be set")
				}
				if m.Type != MetricTypeHistogram {
					t.Errorf("expected histogram type, got %q", m.Type)
				}
				if m.hconf == nil {
					t.Error("expected histogram configuration to be set")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.create()
			if m == nil {
				t.Fatal("expected non-nil metric")
			}
			tt.validate(t, m)
		})
	}
}

func TestMetricLabels(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Metric
		validate func(*testing.T, *Metric)
	}{
		{
			name: "SingleLabel",
			setup: func() *Metric {
				return Counter("test").Label("env", "prod")
			},
			validate: func(t *testing.T, m *Metric) {
				labels := m.labels()
				if labels.Len() != 1 {
					t.Errorf("expected 1 label, got %d", labels.Len())
				}
				if !labels.Check(irt.MakeKV("env", "prod")) {
					t.Error("expected env=prod label")
				}
			},
		},
		{
			name: "MultipleLabels",
			setup: func() *Metric {
				return Counter("test").
					Label("env", "prod").
					Label("region", "us-west").
					Label("service", "api")
			},
			validate: func(t *testing.T, m *Metric) {
				labels := m.labels()
				if labels.Len() != 3 {
					t.Errorf("expected 3 labels, got %d", labels.Len())
				}
			},
		},
		{
			name: "AnnotateLabels",
			setup: func() *Metric {
				return Counter("test").Annotate(
					irt.MakeKV("key1", "val1"),
					irt.MakeKV("key2", "val2"),
				)
			},
			validate: func(t *testing.T, m *Metric) {
				labels := m.labels()
				if labels.Len() != 2 {
					t.Errorf("expected 2 labels, got %d", labels.Len())
				}
			},
		},
		{
			name: "EmptyLabels",
			setup: func() *Metric {
				return Counter("test")
			},
			validate: func(t *testing.T, m *Metric) {
				labels := m.labels()
				if labels.Len() != 0 {
					t.Errorf("expected 0 labels, got %d", labels.Len())
				}
			},
		},
		{
			name: "DuplicateLabels",
			setup: func() *Metric {
				return Counter("test").
					Label("env", "prod").
					Label("env", "prod")
			},
			validate: func(t *testing.T, m *Metric) {
				labels := m.labels()
				// OrderedSet should handle duplicates
				if labels.Len() > 2 {
					t.Error("expected set to handle duplicates")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup()
			tt.validate(t, m)
		})
	}
}

func TestMetricEqual(t *testing.T) {
	tests := []struct {
		name     string
		m1       *Metric
		m2       *Metric
		expected bool
	}{
		{
			name:     "SameIDNoLabels",
			m1:       Counter("test"),
			m2:       Counter("test"),
			expected: true,
		},
		{
			name:     "DifferentID",
			m1:       Counter("test1"),
			m2:       Counter("test2"),
			expected: false,
		},
		{
			name:     "DifferentType",
			m1:       Counter("test"),
			m2:       Gauge("test"),
			expected: false,
		},
		{
			name:     "SameLabels",
			m1:       Counter("test").Label("env", "prod"),
			m2:       Counter("test").Label("env", "prod"),
			expected: true,
		},
		{
			name:     "DifferentLabels",
			m1:       Counter("test").Label("env", "prod"),
			m2:       Counter("test").Label("env", "dev"),
			expected: false,
		},
		{
			name:     "DifferentLabelCount",
			m1:       Counter("test").Label("env", "prod"),
			m2:       Counter("test"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.m1.Equal(tt.m2)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestMetricPeriodic(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		validate func(*testing.T, *Metric)
	}{
		{
			name:     "SetPeriod",
			duration: 5 * time.Second,
			validate: func(t *testing.T, m *Metric) {
				if m.dur != 5*time.Second {
					t.Errorf("expected 5s, got %v", m.dur)
				}
			},
		},
		{
			name:     "ZeroPeriod",
			duration: 0,
			validate: func(t *testing.T, m *Metric) {
				if m.dur != 0 {
					t.Errorf("expected 0, got %v", m.dur)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Counter("test").Periodic(tt.duration)
			tt.validate(t, m)
		})
	}
}

func TestEvent(t *testing.T) {
	tests := []struct {
		name     string
		create   func() *Event
		validate func(*testing.T, *Event)
	}{
		{
			name:   "Inc",
			create: func() *Event { return Counter("test").Inc() },
			validate: func(t *testing.T, e *Event) {
				if e == nil {
					t.Fatal("expected non-nil event")
				}
				if e.m == nil {
					t.Fatal("expected metric to be set")
				}
				if e.op == nil {
					t.Fatal("expected op to be set")
				}
				// Test op adds 1
				result := e.op(0)
				if result != 1 {
					t.Errorf("expected 1, got %d", result)
				}
			},
		},
		{
			name:   "Dec",
			create: func() *Event { return Counter("test").Dec() },
			validate: func(t *testing.T, e *Event) {
				if e.op == nil {
					t.Fatal("expected op to be set")
				}
				result := e.op(10)
				if result != 9 {
					t.Errorf("expected 9, got %d", result)
				}
			},
		},
		{
			name:   "Add",
			create: func() *Event { return Counter("test").Add(5) },
			validate: func(t *testing.T, e *Event) {
				if e.op == nil {
					t.Fatal("expected op to be set")
				}
				result := e.op(10)
				if result != 15 {
					t.Errorf("expected 15, got %d", result)
				}
			},
		},
		{
			name:   "Set",
			create: func() *Event { return Gauge("test").Set(42) },
			validate: func(t *testing.T, e *Event) {
				if e.op == nil {
					t.Fatal("expected op to be set")
				}
				result := e.op(100)
				if result != 42 {
					t.Errorf("expected 42, got %d", result)
				}
			},
		},
		{
			name:   "Collect",
			create: func() *Event { return Gauge("test").Collect(func() int64 { return 99 }) },
			validate: func(t *testing.T, e *Event) {
				if e.op == nil {
					t.Fatal("expected op to be set")
				}
				result := e.op(0)
				if result != 99 {
					t.Errorf("expected 99, got %d", result)
				}
			},
		},
		{
			name:   "CollectAdd",
			create: func() *Event { return Counter("test").CollectAdd(func() int64 { return 10 }) },
			validate: func(t *testing.T, e *Event) {
				if e.op == nil {
					t.Fatal("expected op to be set")
				}
				result := e.op(5)
				if result != 15 {
					t.Errorf("expected 15, got %d", result)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := tt.create()
			tt.validate(t, e)
		})
	}
}

func TestEventString(t *testing.T) {
	tests := []struct {
		name     string
		event    *Event
		validate func(*testing.T, string)
	}{
		{
			name:  "NilMetric",
			event: &Event{},
			validate: func(t *testing.T, s string) {
				if s != "Metric<UNKNOWN>" {
					t.Errorf("expected 'Metric<UNKNOWN>', got %q", s)
				}
			},
		},
		{
			name:  "UnresolvedEvent",
			event: Counter("test").Inc(),
			validate: func(t *testing.T, s string) {
				if !strings.Contains(s, "UNRESOLVED") {
					t.Errorf("expected 'UNRESOLVED' in string, got %q", s)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.event.String()
			tt.validate(t, s)
		})
	}
}

func TestEventMarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		event    *Event
		validate func(*testing.T, []byte, error)
	}{
		{
			name:  "BasicEvent",
			event: Counter("test").Set(42),
			validate: func(t *testing.T, data []byte, err error) {
				if err != nil {
					t.Fatalf("expected no error, got %v", err)
				}
				s := string(data)
				if !strings.Contains(s, "test") {
					t.Error("expected metric ID in JSON")
				}
				if !strings.Contains(s, "value") {
					t.Error("expected value field in JSON")
				}
				if !strings.Contains(s, "type") {
					t.Error("expected type field in JSON")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := tt.event.MarshalJSON()
			tt.validate(t, data, err)
		})
	}
}

// Tests for histogram.go

func TestHistogramConfValidation(t *testing.T) {
	tests := []struct {
		name        string
		conf        *HistogramConf
		expectError bool
		errorMsg    string
	}{
		{
			name:        "ValidDefault",
			conf:        MakeDefaultHistogramConf(),
			expectError: false,
		},
		{
			name: "MinGreaterThanMax",
			conf: &HistogramConf{
				Min:        100,
				Max:        10,
				Quantiles:  []float64{0.5, 0.99},
				OutOfRange: HistogramOutOfRangeTruncate,
			},
			expectError: true,
			errorMsg:    "min cannot be",
		},
		{
			name: "TooFewQuantiles",
			conf: &HistogramConf{
				Min:        0,
				Max:        100,
				Quantiles:  []float64{0.5},
				OutOfRange: HistogramOutOfRangeTruncate,
			},
			expectError: true,
			errorMsg:    "more than one bucket",
		},
		{
			name: "InvalidOutOfRange",
			conf: &HistogramConf{
				Min:        0,
				Max:        100,
				Quantiles:  []float64{0.5, 0.99},
				OutOfRange: HistogramOutOfRangeINVALID,
			},
			expectError: true,
			errorMsg:    "valid behavior for out of range",
		},
		{
			name: "ValidConfiguration",
			conf: &HistogramConf{
				Min:               0,
				Max:               1000,
				SignificantDigits: 3,
				Quantiles:         []float64{0.5, 0.9, 0.99},
				OutOfRange:        HistogramOutOfRangeTruncate,
			},
			expectError: false,
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

func TestHistogramConfOptions(t *testing.T) {
	tests := []struct {
		name     string
		apply    func(*HistogramConf)
		validate func(*testing.T, *HistogramConf)
	}{
		{
			name: "OutOfRangeOption",
			apply: func(conf *HistogramConf) {
				check.NotError(t, conf.Apply(HistogramConfOutOfRange(HistogramOutOfRangePanic)))
			},
			validate: func(t *testing.T, conf *HistogramConf) {
				if conf.OutOfRange != HistogramOutOfRangePanic {
					t.Error("expected panic option to be set")
				}
			},
		},
		{
			name: "SetOption",
			apply: func(conf *HistogramConf) {
				newConf := &HistogramConf{
					Min:        10,
					Max:        100,
					Quantiles:  []float64{0.5, 0.99},
					OutOfRange: HistogramOutOfRangeIgnore,
				}
				check.NotError(t, conf.Apply(HistogramConfSet(newConf)))
			},
			validate: func(t *testing.T, conf *HistogramConf) {
				if conf.Min != 10 || conf.Max != 100 {
					t.Error("expected configuration to be replaced")
				}
			},
		},
		{
			name: "ResetOption",
			apply: func(conf *HistogramConf) {
				conf.Min = 999
				check.NotError(t, conf.Apply(HistogramConfReset()))
			},
			validate: func(t *testing.T, conf *HistogramConf) {
				if conf.Min != 0 {
					t.Error("expected configuration to be reset")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := MakeDefaultHistogramConf()
			tt.apply(conf)
			tt.validate(t, conf)
		})
	}
}

// Tests for collector_conf.go

func TestCollectorConfValidation(t *testing.T) {
	tests := []struct {
		name        string
		conf        *CollectorConf
		expectError bool
		errorMsg    string
	}{
		{
			name: "NoBackends",
			conf: &CollectorConf{
				Buffer: 10,
			},
			expectError: true,
			errorMsg:    "one or more backends",
		},
		{
			name: "NoBuffer",
			conf: &CollectorConf{
				Backends: []CollectorBackend{
					LoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
				},
				Buffer: 0,
			},
			expectError: true,
			errorMsg:    "buffer size",
		},
		{
			name: "ValidConfiguration",
			conf: &CollectorConf{
				Backends: []CollectorBackend{
					LoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
				},
				Buffer: 10,
			},
			expectError: false,
		},
		{
			name: "NegativeBuffer",
			conf: &CollectorConf{
				Backends: []CollectorBackend{
					LoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
				},
				Buffer: -1,
			},
			expectError: false, // Negative is allowed (unlimited)
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

func TestCollectorConfOptions(t *testing.T) {
	tests := []struct {
		name     string
		apply    func(*CollectorConf) error
		validate func(*testing.T, *CollectorConf)
	}{
		{
			name: "SetBuffer",
			apply: func(conf *CollectorConf) error {
				return CollectorConfBuffer(100)(conf)
			},
			validate: func(t *testing.T, conf *CollectorConf) {
				if conf.Buffer != 100 {
					t.Errorf("expected buffer 100, got %d", conf.Buffer)
				}
			},
		},
		{
			name: "AppendBackends",
			apply: func(conf *CollectorConf) error {
				backend := LoggerBackend(send.MakeInternal(), MakeJSONRenderer())
				return CollectorConfAppendBackends(backend)(conf)
			},
			validate: func(t *testing.T, conf *CollectorConf) {
				if len(conf.Backends) != 1 {
					t.Errorf("expected 1 backend, got %d", len(conf.Backends))
				}
			},
		},
		{
			name: "WithLoggerBackend",
			apply: func(conf *CollectorConf) error {
				return CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer())(conf)
			},
			validate: func(t *testing.T, conf *CollectorConf) {
				if len(conf.Backends) != 1 {
					t.Error("expected backend to be added")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &CollectorConf{}
			err := tt.apply(conf)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.validate(t, conf)
		})
	}
}

// Tests for converter.go

func TestExtractMetrics(t *testing.T) {
	tests := []struct {
		name     string
		composer message.Composer
		validate func(*testing.T, []*Event)
	}{
		{
			name:     "NoMetrics",
			composer: message.MakeString("plain message"),
			validate: func(t *testing.T, events []*Event) {
				if len(events) != 0 {
					t.Errorf("expected no events, got %d", len(events))
				}
			},
		},
		{
			name:     "NilComposer",
			composer: nil,
			validate: func(t *testing.T, events []*Event) {
				// Should handle nil gracefully
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := Extract(tt.composer)
			tt.validate(t, events)
		})
	}
}

func TestMessage(t *testing.T) {
	tests := []struct {
		name     string
		events   []*Event
		validate func(*testing.T, message.Composer)
	}{
		{
			name:   "SingleEvent",
			events: []*Event{Counter("test").Inc()},
			validate: func(t *testing.T, c message.Composer) {
				if c == nil {
					t.Fatal("expected non-nil composer")
				}
				if !c.Loggable() {
					t.Error("expected composer to be loggable")
				}
			},
		},
		{
			name:   "MultipleEvents",
			events: []*Event{Counter("test1").Inc(), Gauge("test2").Set(42)},
			validate: func(t *testing.T, c message.Composer) {
				if c == nil {
					t.Fatal("expected non-nil composer")
				}
			},
		},
		{
			name:   "EmptyEvents",
			events: []*Event{},
			validate: func(t *testing.T, c message.Composer) {
				if c == nil {
					t.Fatal("expected non-nil composer")
				}
				// Message with empty events but valid base composer is still loggable
			},
		},
		{
			name:   "NilEvents",
			events: nil,
			validate: func(t *testing.T, c message.Composer) {
				if c == nil {
					t.Fatal("expected non-nil composer")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := Message(message.MakeString("test message"), tt.events...)
			tt.validate(t, c)
		})
	}
}

func TestWithMetrics(t *testing.T) {
	tests := []struct {
		name     string
		composer message.Composer
		events   []*Event
		validate func(*testing.T, message.Composer)
	}{
		{
			name:     "AddMetricsToPlainMessage",
			composer: message.MakeString("test"),
			events:   []*Event{Counter("test").Inc()},
			validate: func(t *testing.T, c message.Composer) {
				if c == nil {
					t.Fatal("expected non-nil composer")
				}
				// Should be wrapped
			},
		},
		{
			name:     "NilComposer",
			composer: nil,
			events:   []*Event{Counter("test").Inc()},
			validate: func(t *testing.T, c message.Composer) {
				// Should handle nil gracefully
			},
		},
		{
			name:     "NoEvents",
			composer: message.MakeString("test"),
			events:   []*Event{},
			validate: func(t *testing.T, c message.Composer) {
				if c == nil {
					t.Fatal("expected non-nil composer")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := WithMetrics(tt.composer, tt.events...)
			tt.validate(t, c)
		})
	}
}

// Edge case and error handling tests

func TestMetricTypeEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Metric
		validate func(*testing.T, *Metric)
	}{
		{
			name: "EmptyID",
			setup: func() *Metric {
				return Counter("")
			},
			validate: func(t *testing.T, m *Metric) {
				if m.ID != "" {
					t.Error("expected empty ID to be preserved")
				}
			},
		},
		{
			name: "ChainedMethods",
			setup: func() *Metric {
				return Counter("test").
					Label("env", "prod").
					Label("region", "us").
					Periodic(5 * time.Second)
			},
			validate: func(t *testing.T, m *Metric) {
				if m.labels().Len() != 2 {
					t.Error("expected labels to be set")
				}
				if m.dur != 5*time.Second {
					t.Error("expected period to be set")
				}
			},
		},
		{
			name: "MetricTypeOverwrite",
			setup: func() *Metric {
				return Counter("test").MetricType(MetricTypeGuage)
			},
			validate: func(t *testing.T, m *Metric) {
				if m.Type != MetricTypeGuage {
					t.Error("expected type to be overwritten")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup()
			tt.validate(t, m)
		})
	}
}

func TestHistogramConfEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		conf     *HistogramConf
		validate func(*testing.T, *HistogramConf, error)
	}{
		{
			name: "ZeroInterval",
			conf: &HistogramConf{
				Min:        0,
				Max:        100,
				Quantiles:  []float64{0.5, 0.99},
				OutOfRange: HistogramOutOfRangeTruncate,
				Interval:   0,
			},
			validate: func(t *testing.T, conf *HistogramConf, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				// Validate should set minimum interval
				if conf.Interval < 100*time.Millisecond {
					t.Error("expected interval to be at least 100ms")
				}
			},
		},
		{
			name: "VerySmallInterval",
			conf: &HistogramConf{
				Min:        0,
				Max:        100,
				Quantiles:  []float64{0.5, 0.99},
				OutOfRange: HistogramOutOfRangeTruncate,
				Interval:   1 * time.Nanosecond,
			},
			validate: func(t *testing.T, conf *HistogramConf, err error) {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if conf.Interval < 100*time.Millisecond {
					t.Error("expected interval to be clamped to minimum")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.conf.Validate()
			tt.validate(t, tt.conf, err)
		})
	}
}

func TestCollectorConfEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *CollectorConf
		validate func(*testing.T, *CollectorConf)
	}{
		{
			name: "MultipleBackends",
			setup: func() *CollectorConf {
				conf := &CollectorConf{Buffer: 10}
				check.NotError(t, CollectorConfAppendBackends(
					LoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
					LoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
				)(conf))
				return conf
			},
			validate: func(t *testing.T, conf *CollectorConf) {
				if len(conf.Backends) != 2 {
					t.Errorf("expected 2 backends, got %d", len(conf.Backends))
				}
			},
		},
		{
			name: "SetEntireConfiguration",
			setup: func() *CollectorConf {
				newConf := &CollectorConf{
					Buffer: 50,
					Backends: []CollectorBackend{
						LoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
					},
				}
				conf := &CollectorConf{}
				check.NotError(t, CollectorConfSet(newConf)(conf))
				return conf
			},
			validate: func(t *testing.T, conf *CollectorConf) {
				if conf.Buffer != 50 {
					t.Error("expected configuration to be replaced")
				}
				if len(conf.Backends) != 1 {
					t.Error("expected backends to be replaced")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := tt.setup()
			tt.validate(t, conf)
		})
	}
}

func TestConverterEdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() message.Composer
		validate func(*testing.T, message.Composer)
	}{
		{
			name: "WithMetricsNonMetricMessage",
			setup: func() message.Composer {
				return WithMetrics(message.MakeString("plain"))
			},
			validate: func(t *testing.T, c message.Composer) {
				if c == nil {
					t.Fatal("expected non-nil composer")
				}
				// Should wrap non-metric messages
			},
		},
		{
			name: "MessageWithNilEvent",
			setup: func() message.Composer {
				return Message(message.MakeString("test"), []*Event{nil}...)
			},
			validate: func(t *testing.T, c message.Composer) {
				if c == nil {
					t.Fatal("expected non-nil composer")
				}
				// Should handle nil events
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.setup()
			tt.validate(t, c)
		})
	}
}

func TestEventOperationEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		create    func() *Event
		initial   int64
		expected  int64
		shouldRun bool
	}{
		{
			name:      "AddNegative",
			create:    func() *Event { return Counter("test").Add(-10) },
			initial:   100,
			expected:  90,
			shouldRun: true,
		},
		{
			name:      "AddZero",
			create:    func() *Event { return Counter("test").Add(0) },
			initial:   50,
			expected:  50,
			shouldRun: true,
		},
		{
			name:      "SetNegative",
			create:    func() *Event { return Gauge("test").Set(-1) },
			initial:   100,
			expected:  -1,
			shouldRun: true,
		},
		{
			name:      "SetZero",
			create:    func() *Event { return Gauge("test").Set(0) },
			initial:   100,
			expected:  0,
			shouldRun: true,
		},
		{
			name:      "LargeValue",
			create:    func() *Event { return Counter("test").Add(1000000) },
			initial:   0,
			expected:  1000000,
			shouldRun: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.shouldRun {
				t.Skip("test disabled")
			}
			e := tt.create()
			if e.op == nil {
				t.Fatal("expected op to be set")
			}
			result := e.op(tt.initial)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestMetricResolve(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() *Metric
		validate func(*testing.T, *Metric)
	}{
		{
			name: "ResolveInitializesBufferPool",
			setup: func() *Metric {
				m := Counter("test")
				m.resolve()
				return m
			},
			validate: func(t *testing.T, m *Metric) {
				// Should not panic when getting from buffer pool
				buf := m.bufferPool.Get()
				if buf == nil {
					t.Error("expected buffer pool to be initialized")
				}
				m.bufferPool.Put(buf)
			},
		},
		{
			name: "ResolveInitializesLabelCache",
			setup: func() *Metric {
				m := Counter("test").Label("env", "prod")
				m.resolve()
				return m
			},
			validate: func(t *testing.T, m *Metric) {
				if m.labelCache == nil {
					t.Error("expected label cache to be initialized")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := tt.setup()
			tt.validate(t, m)
		})
	}
}

func TestMessageComposerIntegration(t *testing.T) {
	tests := []struct {
		name     string
		setup    func() message.Composer
		validate func(*testing.T, message.Composer)
	}{
		{
			name: "MessageWithMultipleMetrics",
			setup: func() message.Composer {
				events := []*Event{
					Counter("requests").Inc(),
					Gauge("memory").Set(1024),
					Delta("bytes").Add(512),
				}
				return Message(message.MakeString("metrics"), events...)
			},
			validate: func(t *testing.T, c message.Composer) {
				if c == nil {
					t.Fatal("expected non-nil composer")
				}
				if !c.Loggable() {
					t.Error("expected composer with events to be loggable")
				}
				if !c.Structured() {
					t.Error("expected structured composer")
				}
			},
		},
		{
			name: "WithMetricsStructuredMessage",
			setup: func() message.Composer {
				return WithMetrics(message.BuildKV().KV("key", "value"))
			},
			validate: func(t *testing.T, c message.Composer) {
				if c == nil {
					t.Fatal("expected non-nil composer")
				}
			},
		},
		{
			name: "WithMetricsChaining",
			setup: func() message.Composer {
				base := message.MakeString("baseline")
				return WithMetrics(base,
					Counter("metric1").Inc(),
					Counter("metric2").Add(5),
				)
			},
			validate: func(t *testing.T, c message.Composer) {
				if c == nil {
					t.Fatal("expected non-nil composer")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.setup()
			tt.validate(t, c)
		})
	}
}

func TestRenderers(t *testing.T) {
	tests := []struct {
		name     string
		renderer Renderer
		validate func(*testing.T, Renderer)
	}{
		{
			name:     "JSONRenderer",
			renderer: MakeJSONRenderer(),
			validate: func(t *testing.T, r Renderer) {
				if r.Metric == nil {
					t.Fatal("expected non-nil metric renderer")
				}
			},
		},
		{
			name:     "GraphiteRenderer",
			renderer: MakeGraphiteRenderer(),
			validate: func(t *testing.T, r Renderer) {
				if r.Metric == nil {
					t.Fatal("expected non-nil metric renderer")
				}
			},
		},
		{
			name:     "OpenTSDBRenderer",
			renderer: MakeOpenTSBLineRenderer(),
			validate: func(t *testing.T, r Renderer) {
				if r.Metric == nil {
					t.Fatal("expected non-nil metric renderer")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.renderer)
		})
	}
}

func TestLoggerBackend(t *testing.T) {
	tests := []struct {
		name     string
		sender   send.Sender
		renderer Renderer
		validate func(*testing.T, CollectorBackend)
	}{
		{
			name:     "WithInternalSender",
			sender:   send.MakeInternal(),
			renderer: MakeJSONRenderer(),
			validate: func(t *testing.T, backend CollectorBackend) {
				if backend == nil {
					t.Fatal("expected non-nil backend")
				}
			},
		},
		{
			name: "WithConfiguredSender",
			sender: func() send.Sender {
				s := send.MakeInternal()
				s.SetPriority(level.Debug)
				s.SetName("metrics")
				return s
			}(),
			renderer: MakeGraphiteRenderer(),
			validate: func(t *testing.T, backend CollectorBackend) {
				if backend == nil {
					t.Fatal("expected non-nil backend")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			backend := LoggerBackend(tt.sender, tt.renderer)
			tt.validate(t, backend)
		})
	}
}
