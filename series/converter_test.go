package series

import (
	"encoding/json"
	"strings"
	"sync"
	"testing"

	"github.com/tychoish/fun/irt"
	"github.com/tychoish/grip/message"
)

// Test Extract with various input types
func TestExtractVariousTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		validate func(*testing.T, []*Event)
	}{
		{
			name:  "SingleEvent",
			input: Counter("test").Inc(),
			validate: func(t *testing.T, events []*Event) {
				if len(events) != 1 {
					t.Errorf("expected 1 event, got %d", len(events))
				}
			},
		},
		{
			name:  "EventPointer",
			input: Counter("test").Inc(),
			validate: func(t *testing.T, events []*Event) {
				if len(events) != 1 {
					t.Errorf("expected 1 event, got %d", len(events))
				}
			},
		},
		{
			name:  "SliceOfEvents",
			input: []*Event{Counter("test1").Inc(), Counter("test2").Inc()},
			validate: func(t *testing.T, events []*Event) {
				if len(events) != 2 {
					t.Errorf("expected 2 events, got %d", len(events))
				}
			},
		},
		{
			name: "MapWithEvents",
			input: map[string]any{
				"metric": Counter("test").Inc(),
				"other":  "value",
			},
			validate: func(t *testing.T, events []*Event) {
				if len(events) != 1 {
					t.Errorf("expected 1 event, got %d", len(events))
				}
			},
		},
		{
			name:  "NonEventType",
			input: "plain string",
			validate: func(t *testing.T, events []*Event) {
				if len(events) != 0 {
					t.Errorf("expected 0 events for non-event type, got %d", len(events))
				}
			},
		},
		{
			name:  "NilInput",
			input: nil,
			validate: func(t *testing.T, events []*Event) {
				if len(events) != 0 {
					t.Errorf("expected empty slice, got %d events", len(events))
				}
			},
		},
		{
			name: "EventFunction",
			input: func() *Event {
				return Counter("test").Inc()
			},
			validate: func(t *testing.T, events []*Event) {
				if len(events) != 1 {
					t.Errorf("expected 1 event from function, got %d", len(events))
				}
			},
		},
		{
			name: "SliceOfEventsFunction",
			input: func() []*Event {
				return []*Event{Counter("test1").Inc(), Counter("test2").Inc()}
			},
			validate: func(t *testing.T, events []*Event) {
				if len(events) != 2 {
					t.Errorf("expected 2 events from function, got %d", len(events))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			events := Extract(tt.input)
			tt.validate(t, events)
		})
	}
}

// Test Message constructor
func TestMessageConstructor(t *testing.T) {
	tests := []struct {
		name     string
		composer message.Composer
		events   []*Event
		validate func(*testing.T, *MetricMessage)
	}{
		{
			name:     "SimpleMessage",
			composer: message.MakeString("test message"),
			events:   []*Event{Counter("test").Inc()},
			validate: func(t *testing.T, m *MetricMessage) {
				if m.Composer == nil {
					t.Error("expected composer to be set")
				}
				if len(m.Events) != 1 {
					t.Errorf("expected 1 event, got %d", len(m.Events))
				}
				if !m.Loggable() {
					t.Error("expected message to be loggable")
				}
			},
		},
		{
			name:     "MessageWithMultipleEvents",
			composer: message.MakeString("metrics"),
			events: []*Event{
				Counter("counter").Inc(),
				Gauge("gauge").Set(100),
				Delta("delta").Add(5),
			},
			validate: func(t *testing.T, m *MetricMessage) {
				if len(m.Events) != 3 {
					t.Errorf("expected 3 events, got %d", len(m.Events))
				}
			},
		},
		{
			name:     "MessageWithNoEvents",
			composer: message.MakeString("empty"),
			events:   []*Event{},
			validate: func(t *testing.T, m *MetricMessage) {
				if len(m.Events) != 0 {
					t.Errorf("expected 0 events, got %d", len(m.Events))
				}
			},
		},
		{
			name:     "MessageWithNilEvent",
			composer: message.MakeString("test"),
			events:   []*Event{nil, Counter("test").Inc()},
			validate: func(t *testing.T, m *MetricMessage) {
				if len(m.Events) != 2 {
					t.Errorf("expected 2 events (including nil), got %d", len(m.Events))
				}
			},
		},
		{
			name:     "StructuredComposer",
			composer: message.NewKV().KV("key", "value"),
			events:   []*Event{Counter("test").Inc()},
			validate: func(t *testing.T, m *MetricMessage) {
				if !m.Structured() {
					t.Error("expected structured message")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Message(tt.composer, tt.events...)
			tt.validate(t, m)
		})
	}
}

// Test MetricMessage.String()
func TestMetricMessageString(t *testing.T) {
	tests := []struct {
		name           string
		msg            *MetricMessage
		expectedSubstr []string
	}{
		{
			name: "WithEvents",
			msg: &MetricMessage{
				Composer: message.MakeString("base message"),
				Events: []*Event{
					Counter("test1").Inc(),
					Counter("test2").Inc(),
				},
			},
			expectedSubstr: []string{
				"base message",
			},
		},
		{
			name: "WithoutEvents",
			msg: &MetricMessage{
				Composer: message.MakeString("just message"),
				Events:   []*Event{},
			},
			expectedSubstr: []string{
				"just message",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := tt.msg.String()

			for _, substr := range tt.expectedSubstr {
				if !strings.Contains(output, substr) {
					t.Errorf("expected output to contain %q, got: %s", substr, output)
				}
			}
		})
	}
}

// Test MetricMessage.Raw()
func TestMetricMessageRaw(t *testing.T) {
	tests := []struct {
		name     string
		msg      *MetricMessage
		validate func(*testing.T, any)
	}{
		{
			name: "WithEvents",
			msg: &MetricMessage{
				Composer: message.MakeString("test"),
				Events: []*Event{
					Counter("counter").Set(42),
				},
			},
			validate: func(t *testing.T, raw any) {
				if raw == nil {
					t.Fatal("expected non-nil raw output")
				}

				// Try to marshal to JSON to verify structure
				data, err := json.Marshal(raw)
				if err != nil {
					t.Errorf("failed to marshal raw output: %v", err)
				}

				jsonStr := string(data)
				if !strings.Contains(jsonStr, "message") {
					t.Error("expected 'message' field in raw output")
				}
				if !strings.Contains(jsonStr, "events") {
					t.Error("expected 'events' field in raw output")
				}
			},
		},
		{
			name: "WithoutEvents",
			msg: &MetricMessage{
				Composer: message.MakeString("test"),
				Events:   []*Event{},
			},
			validate: func(t *testing.T, raw any) {
				data, err := json.Marshal(raw)
				if err != nil {
					t.Errorf("failed to marshal raw output: %v", err)
				}

				var result map[string]any
				if err := json.Unmarshal(data, &result); err != nil {
					t.Errorf("failed to unmarshal: %v", err)
				}

				// Events field will be omitted when empty due to omitempty tag
				// Just verify we have a message field
				if _, ok := result["message"]; !ok {
					t.Error("expected 'message' field in raw output")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Resolve metrics before calling Raw() which calls Export()
			for _, ev := range tt.msg.Events {
				if ev.m != nil {
					ev.m.resolve()
				}
			}

			raw := tt.msg.Raw()
			tt.validate(t, raw)
		})
	}
}

// Test WithMetrics with different input types
func TestWithMetricsVariousTypes(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		events   []*Event
		validate func(*testing.T, message.Composer)
	}{
		{
			name:   "StringWithEvents",
			input:  "base message",
			events: []*Event{Counter("test").Inc()},
			validate: func(t *testing.T, c message.Composer) {
				mm, ok := c.(*MetricMessage)
				if !ok {
					t.Fatal("expected MetricMessage")
				}
				if len(mm.Events) != 1 {
					t.Errorf("expected 1 event, got %d", len(mm.Events))
				}
				if !strings.Contains(mm.String(), "base message") {
					t.Error("expected base message in output")
				}
			},
		},
		{
			name: "MapWithEvents",
			input: map[string]any{
				"key": "value",
			},
			events: []*Event{Counter("test").Inc()},
			validate: func(t *testing.T, c message.Composer) {
				mm, ok := c.(*MetricMessage)
				if !ok {
					t.Fatal("expected MetricMessage")
				}
				if len(mm.Events) != 1 {
					t.Errorf("expected 1 event, got %d", len(mm.Events))
				}
			},
		},
		{
			name: "ExistingMetricMessage",
			input: &MetricMessage{
				Composer: message.MakeString("existing"),
				Events:   []*Event{Counter("existing").Inc()},
			},
			events: []*Event{Counter("new").Inc()},
			validate: func(t *testing.T, c message.Composer) {
				mm, ok := c.(*MetricMessage)
				if !ok {
					t.Fatal("expected MetricMessage")
				}
				// Should have both existing and new events
				if len(mm.Events) != 2 {
					t.Errorf("expected 2 events (1 existing + 1 new), got %d", len(mm.Events))
				}
			},
		},
		{
			name:   "NilInput",
			input:  nil,
			events: []*Event{Counter("test").Inc()},
			validate: func(t *testing.T, c message.Composer) {
				mm, ok := c.(*MetricMessage)
				if !ok {
					t.Fatal("expected MetricMessage")
				}
				if len(mm.Events) != 1 {
					t.Errorf("expected 1 event, got %d", len(mm.Events))
				}
			},
		},
		{
			name:   "ComposerWithNoEvents",
			input:  message.MakeString("test"),
			events: []*Event{},
			validate: func(t *testing.T, c message.Composer) {
				mm, ok := c.(*MetricMessage)
				if !ok {
					t.Fatal("expected MetricMessage")
				}
				if len(mm.Events) != 0 {
					t.Errorf("expected 0 events, got %d", len(mm.Events))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := WithMetrics(tt.input, tt.events...)
			tt.validate(t, c)
		})
	}
}

// Test Event.Export()
func TestEventExport(t *testing.T) {
	tests := []struct {
		name     string
		event    *Event
		validate func(*testing.T, Record)
	}{
		{
			name:  "SimpleCounter",
			event: Counter("test").Set(42),
			validate: func(t *testing.T, r Record) {
				if r.ID != "test" {
					t.Errorf("expected ID 'test', got %s", r.ID)
				}
				// Note: Value is not set until event is processed by collector
				// so we don't validate it here
				if r.Labels == nil {
					t.Error("expected labels to be initialized")
				}
			},
		},
		{
			name:  "WithLabels",
			event: Counter("labeled").Label("env", "prod").Label("host", "server1").Set(100),
			validate: func(t *testing.T, r Record) {
				if r.ID != "labeled" {
					t.Errorf("expected ID 'labeled', got %s", r.ID)
				}
				if r.Labels == nil {
					t.Fatal("expected labels to be present")
				}

				// Check labels

				if r.Labels.Get("env") != "prod" {
					t.Errorf("expected env=prod, got %s", r.Labels.Get("env"))
				}
				if r.Labels.Get("host") != "server1" {
					t.Errorf("expected host=server1, got %s", r.Labels.Get("host"))
				}
			},
		},
		{
			name:  "ZeroValue",
			event: Gauge("zero").Set(0),
			validate: func(t *testing.T, r Record) {
				if r.ID != "zero" {
					t.Errorf("expected ID 'zero', got %s", r.ID)
				}
				// Note: Value is not set until event is processed by collector
			},
		},
		{
			name:  "NegativeValue",
			event: Delta("negative").Set(-50),
			validate: func(t *testing.T, r Record) {
				if r.ID != "negative" {
					t.Errorf("expected ID 'negative', got %s", r.ID)
				}
				// Note: Value is not set until event is processed by collector
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Need to resolve the metric first to initialize labels
			if tt.event.m != nil {
				tt.event.m.resolve()
			}

			record := tt.event.Export()
			tt.validate(t, record)
		})
	}
}

// Test isEventTyped helper
func TestEventTypeDetection(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected bool
	}{
		{
			name:     "Event",
			input:    *Counter("test").Inc(),
			expected: true,
		},
		{
			name:     "EventPointer",
			input:    Counter("test").Inc(),
			expected: true,
		},
		{
			name:     "EventSlice",
			input:    []*Event{Counter("test").Inc()},
			expected: true,
		},
		{
			name:     "EventSliceValue",
			input:    []Event{*Counter("test").Inc()},
			expected: true,
		},
		{
			name:     "MetricMessage",
			input:    &MetricMessage{},
			expected: true,
		},
		{
			name:     "String",
			input:    "not an event",
			expected: false,
		},
		{
			name:     "Number",
			input:    42,
			expected: false,
		},
		{
			name:     "Map",
			input:    map[string]any{"key": "value"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isEventTyped(tt.input)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Test MetricMessage with complex nested structures
func TestMetricMessageComplexStructures(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		validate func(*testing.T, message.Composer)
	}{
		{
			name: "NestedMapWithEvents",
			input: map[string]any{
				"metrics": []*Event{Counter("nested").Inc()},
				"data":    "value",
			},
			validate: func(t *testing.T, c message.Composer) {
				mm, ok := c.(*MetricMessage)
				if !ok {
					t.Fatal("expected MetricMessage")
				}
				if len(mm.Events) == 0 {
					t.Error("expected events to be extracted from nested structure")
				}
			},
		},
		{
			name: "SliceWithMixedTypes",
			input: []any{
				"string",
				42,
				Counter("in-slice").Inc(),
			},
			validate: func(t *testing.T, c message.Composer) {
				mm, ok := c.(*MetricMessage)
				if !ok {
					t.Fatal("expected MetricMessage")
				}
				if len(mm.Events) == 0 {
					t.Error("expected event to be extracted from slice")
				}
			},
		},
		{
			name: "KVPairsWithEvents",
			input: []irt.KV[string, any]{
				{Key: "event", Value: Counter("kv").Inc()},
				{Key: "other", Value: "data"},
			},
			validate: func(t *testing.T, c message.Composer) {
				mm, ok := c.(*MetricMessage)
				if !ok {
					t.Fatal("expected MetricMessage")
				}
				if len(mm.Events) == 0 {
					t.Error("expected event from KV pairs")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := WithMetrics(tt.input)
			tt.validate(t, c)
		})
	}
}

// Test MetricMessage Structured() method
func TestMetricMessageStructured(t *testing.T) {
	mm := &MetricMessage{
		Composer: message.MakeString("test"),
		Events:   []*Event{},
	}

	if !mm.Structured() {
		t.Error("MetricMessage should always be structured")
	}
}

// Test concurrent Extract operations
func TestExtractConcurrent(t *testing.T) {
	const numGoroutines = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < 100; j++ {
				input := []*Event{
					Counter("concurrent").Inc(),
					Gauge("test").Set(int64(j)),
				}

				events := Extract(input)
				if len(events) != 2 {
					t.Errorf("expected 2 events, got %d", len(events))
				}
			}
		}(i)
	}

	wg.Wait()
}

// Test concurrent WithMetrics operations
func TestWithMetricsConcurrent(t *testing.T) {
	const numGoroutines = 10

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()

			for j := 0; j < 100; j++ {
				c := WithMetrics(
					message.MakeString("concurrent test"),
					Counter("metric").Inc(),
				)

				if c == nil {
					t.Error("expected non-nil composer")
				}
			}
		}(i)
	}

	wg.Wait()
}
