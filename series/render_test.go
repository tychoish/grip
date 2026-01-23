package series

import (
	"bytes"
	"fmt"
	"iter"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tychoish/fun/fn"
)

// Test RenderMetricJSON
func TestRenderMetricJSON(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		labels         fn.Future[iter.Seq2[string, string]]
		value          int64
		ts             time.Time
		expectedSubstr []string
		notContains    []string
	}{
		{
			name: "BasicMetric",
			key:  "test.metric",
			labels: fn.MakeFuture(func() iter.Seq2[string, string] {
				return func(yield func(string, string) bool) {}
			}),
			value: 42,
			ts:    time.Unix(1234567890, 0),
			expectedSubstr: []string{
				`"metric":"test.metric"`,
				`"value":42`,
				`"ts":1234567890000`,
			},
		},
		{
			name: "MetricWithLabels",
			key:  "http.requests",
			labels: fn.MakeFuture(func() iter.Seq2[string, string] {
				return func(yield func(string, string) bool) {
					yield("method", "GET")
					yield("status", "200")
				}
			}),
			value: 100,
			ts:    time.Unix(1234567890, 0),
			expectedSubstr: []string{
				`"metric":"http.requests"`,
				`"tags":{`,
				`"method":"GET"`,
				`"status":"200"`,
				`"value":100`,
			},
		},
		{
			name:   "ZeroValue",
			key:    "counter",
			labels: fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} }),
			value:  0,
			ts:     time.Unix(0, 0),
			expectedSubstr: []string{
				`"value":0`,
			},
		},
		{
			name:   "NegativeValue",
			key:    "delta",
			labels: fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} }),
			value:  -50,
			ts:     time.Unix(1234567890, 0),
			expectedSubstr: []string{
				`"value":-50`,
			},
		},
		{
			name: "SingleLabel",
			key:  "metric",
			labels: fn.MakeFuture(func() iter.Seq2[string, string] {
				return func(yield func(string, string) bool) {
					yield("host", "localhost")
				}
			}),
			value: 1,
			ts:    time.Unix(1234567890, 0),
			expectedSubstr: []string{
				`"tags":{"host":"localhost"}`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			RenderMetricJSON(buf, tt.key, tt.labels, tt.value, tt.ts)

			output := buf.String()
			for _, substr := range tt.expectedSubstr {
				if !strings.Contains(output, substr) {
					t.Errorf("expected output to contain %q, got: %s", substr, output)
				}
			}

			for _, substr := range tt.notContains {
				if strings.Contains(output, substr) {
					t.Errorf("expected output to NOT contain %q, got: %s", substr, output)
				}
			}

			// Should end with newline
			if !strings.HasSuffix(output, "\n") {
				t.Error("expected output to end with newline")
			}
		})
	}
}

// Test RenderHistogramJSON
func TestRenderHistogramJSON(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		labels         fn.Future[iter.Seq2[string, string]]
		sample         iter.Seq2[float64, int64]
		ts             time.Time
		expectedSubstr []string
	}{
		{
			name:   "BasicHistogram",
			key:    "response.time",
			labels: fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} }),
			sample: func(yield func(float64, int64) bool) {
				yield(0.5, 100)
				yield(0.99, 500)
			},
			ts: time.Unix(1234567890, 0),
			expectedSubstr: []string{
				`"metric":"response.time"`,
				`"value":{`,
				`"50":100`,
				`"99":500`,
			},
		},
		{
			name: "HistogramWithLabels",
			key:  "latency",
			labels: fn.MakeFuture(func() iter.Seq2[string, string] {
				return func(yield func(string, string) bool) {
					yield("endpoint", "/api/users")
				}
			}),
			sample: func(yield func(float64, int64) bool) {
				yield(0.95, 250)
			},
			ts: time.Unix(1234567890, 0),
			expectedSubstr: []string{
				`"metric":"latency"`,
				`"tags":{"endpoint":"/api/users"}`,
				`"95":250`,
			},
		},
		{
			name:   "EmptyHistogram",
			key:    "empty",
			labels: fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} }),
			sample: func(yield func(float64, int64) bool) {},
			ts:     time.Unix(1234567890, 0),
			expectedSubstr: []string{
				`"metric":"empty"`,
				`"value":{}`,
			},
		},
		{
			name:   "MultipleQuantiles",
			key:    "histogram",
			labels: fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} }),
			sample: func(yield func(float64, int64) bool) {
				yield(0.1, 10)
				yield(0.5, 50)
				yield(0.9, 90)
				yield(0.99, 99)
			},
			ts: time.Unix(1234567890, 0),
			expectedSubstr: []string{
				`"10":10`,
				`"50":50`,
				`"90":90`,
				`"99":99`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			RenderHistogramJSON(buf, tt.key, tt.labels, tt.sample, tt.ts)

			output := buf.String()
			for _, substr := range tt.expectedSubstr {
				if !strings.Contains(output, substr) {
					t.Errorf("expected output to contain %q, got: %s", substr, output)
				}
			}

			if !strings.HasSuffix(output, "\n") {
				t.Error("expected output to end with newline")
			}
		})
	}
}

// Test RenderMetricOpenTSB
func TestRenderMetricOpenTSB(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		labels         fn.Future[iter.Seq2[string, string]]
		value          int64
		ts             time.Time
		expectedSubstr []string
	}{
		{
			name:   "BasicMetric",
			key:    "sys.cpu.user",
			labels: fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} }),
			value:  75,
			ts:     time.Unix(1234567890, 0),
			expectedSubstr: []string{
				"put sys.cpu.user",
				"1234567890000",
				"75",
			},
		},
		{
			name: "MetricWithTags",
			key:  "sys.mem.free",
			labels: fn.MakeFuture(func() iter.Seq2[string, string] {
				return func(yield func(string, string) bool) {
					yield("host", "server1")
					yield("dc", "us-east")
				}
			}),
			value: 2048,
			ts:    time.Unix(1234567890, 0),
			expectedSubstr: []string{
				"put sys.mem.free",
				"host=server1",
				"dc=us-east",
				"2048",
			},
		},
		{
			name:   "ZeroValue",
			key:    "metric",
			labels: fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} }),
			value:  0,
			ts:     time.Unix(1234567890, 0),
			expectedSubstr: []string{
				"put metric",
				"0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			RenderMetricOpenTSB(buf, tt.key, tt.labels, tt.value, tt.ts)

			output := buf.String()
			for _, substr := range tt.expectedSubstr {
				if !strings.Contains(output, substr) {
					t.Errorf("expected output to contain %q, got: %s", substr, output)
				}
			}

			if !strings.HasPrefix(output, "put ") {
				t.Error("expected output to start with 'put '")
			}

			if !strings.HasSuffix(output, "\n") {
				t.Error("expected output to end with newline")
			}
		})
	}
}

// Test RenderMetricGraphite
func TestRenderMetricGraphite(t *testing.T) {
	tests := []struct {
		name           string
		key            string
		labels         fn.Future[iter.Seq2[string, string]]
		value          int64
		ts             time.Time
		expectedSubstr []string
	}{
		{
			name:   "BasicMetric",
			key:    "servers.web01.cpu",
			labels: fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} }),
			value:  85,
			ts:     time.Unix(1234567890, 0),
			expectedSubstr: []string{
				"servers.web01.cpu",
				"85",
				"1234567890",
			},
		},
		{
			name: "MetricWithTags",
			key:  "app.requests",
			labels: fn.MakeFuture(func() iter.Seq2[string, string] {
				return func(yield func(string, string) bool) {
					yield("method", "POST")
					yield("status", "200")
				}
			}),
			value: 150,
			ts:    time.Unix(1234567890, 0),
			expectedSubstr: []string{
				"app.requests",
				";method=POST",
				";status=200",
				"150",
			},
		},
		{
			name:   "NegativeValue",
			key:    "delta.metric",
			labels: fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} }),
			value:  -10,
			ts:     time.Unix(1234567890, 0),
			expectedSubstr: []string{
				"delta.metric",
				"-10",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			RenderMetricGraphite(buf, tt.key, tt.labels, tt.value, tt.ts)

			output := buf.String()
			for _, substr := range tt.expectedSubstr {
				if !strings.Contains(output, substr) {
					t.Errorf("expected output to contain %q, got: %s", substr, output)
				}
			}

			if !strings.HasSuffix(output, "\n") {
				t.Error("expected output to end with newline")
			}

			// Verify format: key [;tag=value]* value timestamp
			parts := strings.Fields(output)
			if len(parts) < 3 {
				t.Errorf("expected at least 3 parts, got %d: %s", len(parts), output)
			}
		})
	}
}

// Test renderer constructors
func TestRendererConstructors(t *testing.T) {
	tests := []struct {
		name     string
		create   func() Renderer
		validate func(*testing.T, Renderer)
	}{
		{
			name:   "MakeJSONRenderer",
			create: MakeJSONRenderer,
			validate: func(t *testing.T, r Renderer) {
				if r.Metric == nil {
					t.Error("expected Metric renderer to be set")
				}
				if r.Histogram == nil {
					t.Error("expected Histogram renderer to be set")
				}

				// Test that it actually renders
				buf := &bytes.Buffer{}
				labels := fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} })
				r.Metric(buf, "test", labels, 100, time.Unix(1234567890, 0))
				if !strings.Contains(buf.String(), `"metric":"test"`) {
					t.Error("JSON renderer didn't produce expected output")
				}
			},
		},
		{
			name:   "MakeGraphiteRenderer",
			create: MakeGraphiteRenderer,
			validate: func(t *testing.T, r Renderer) {
				if r.Metric == nil {
					t.Error("expected Metric renderer to be set")
				}
				if r.Histogram == nil {
					t.Error("expected Histogram renderer to be set")
				}

				buf := &bytes.Buffer{}
				labels := fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} })
				r.Metric(buf, "test.metric", labels, 50, time.Unix(1234567890, 0))
				if !strings.Contains(buf.String(), "test.metric") {
					t.Error("Graphite renderer didn't produce expected output")
				}
			},
		},
		{
			name:   "MakeOpenTSBLineRenderer",
			create: MakeOpenTSBLineRenderer,
			validate: func(t *testing.T, r Renderer) {
				if r.Metric == nil {
					t.Error("expected Metric renderer to be set")
				}
				if r.Histogram == nil {
					t.Error("expected Histogram renderer to be set")
				}

				buf := &bytes.Buffer{}
				labels := fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} })
				r.Metric(buf, "test.metric", labels, 75, time.Unix(1234567890, 0))
				if !strings.HasPrefix(buf.String(), "put ") {
					t.Error("OpenTSB renderer didn't produce expected output")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.create()
			tt.validate(t, r)
		})
	}
}

// Test edge cases for label rendering
func TestRenderLabelsJSON(t *testing.T) {
	tests := []struct {
		name     string
		labels   iter.Seq2[string, string]
		expected string
	}{
		{
			name: "NoLabels",
			labels: func(yield func(string, string) bool) {
				// Empty iterator - no labels
			},
			expected: "",
		},
		{
			name: "SingleLabel",
			labels: func(yield func(string, string) bool) {
				yield("key", "value")
			},
			expected: `"tags":{"key":"value"},`,
		},
		{
			name: "MultipleLabels",
			labels: func(yield func(string, string) bool) {
				yield("key1", "value1")
				yield("key2", "value2")
			},
			expected: `"tags":{`,
		},
		{
			name: "SpecialCharacters",
			labels: func(yield func(string, string) bool) {
				yield("host", "server-01.example.com")
			},
			expected: `"host":"server-01.example.com"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			renderLabelsJSON(buf, tt.labels)

			output := buf.String()
			if tt.expected != "" && !strings.Contains(output, tt.expected) {
				t.Errorf("expected output to contain %q, got: %s", tt.expected, output)
			}
		})
	}
}

// Test concurrent rendering
func TestRenderConcurrent(t *testing.T) {
	r := MakeJSONRenderer()

	const numGoroutines = 10
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			buf := &bytes.Buffer{}
			labels := fn.MakeFuture(func() iter.Seq2[string, string] {
				return func(yield func(string, string) bool) {
					yield("goroutine", fmt.Sprintf("%d", id))
				}
			})

			for j := 0; j < 100; j++ {
				buf.Reset()
				r.Metric(buf, "concurrent.test", labels, int64(j), time.Now())
			}
		}(i)
	}

	wg.Wait()
}

// Test rendering with very long keys and values
func TestRenderLongValues(t *testing.T) {
	longKey := strings.Repeat("a", 1000)
	longValue := strings.Repeat("b", 1000)

	buf := &bytes.Buffer{}
	labels := fn.MakeFuture(func() iter.Seq2[string, string] {
		return func(yield func(string, string) bool) {
			yield("long", longValue)
		}
	})

	RenderMetricJSON(buf, longKey, labels, 42, time.Unix(1234567890, 0))

	output := buf.String()
	if !strings.Contains(output, longKey) {
		t.Error("expected long key to be present")
	}
	if !strings.Contains(output, longValue) {
		t.Error("expected long value to be present")
	}
}

// Test histogram renderer with edge cases
func TestRenderHistogramEdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		sample iter.Seq2[float64, int64]
	}{
		{
			name: "VerySmallQuantiles",
			sample: func(yield func(float64, int64) bool) {
				yield(0.001, 1)
				yield(0.01, 10)
			},
		},
		{
			name: "VeryLargeValues",
			sample: func(yield func(float64, int64) bool) {
				yield(0.99, 9999999999)
			},
		},
		{
			name: "ManyQuantiles",
			sample: func(yield func(float64, int64) bool) {
				for i := 1; i <= 100; i++ {
					yield(float64(i)/100.0, int64(i*10))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			labels := fn.MakeFuture(func() iter.Seq2[string, string] { return func(yield func(string, string) bool) {} })
			RenderHistogramJSON(buf, "test", labels, tt.sample, time.Now())

			if buf.Len() == 0 {
				t.Error("expected non-empty output")
			}
		})
	}
}
