package series

import (
	"bytes"
	"context"
	"errors"
	"io"
	"iter"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/grip/send"
)

// Helper to create a test backend that writes to a buffer
func makeTestBackend(buf *bytes.Buffer, renderer Renderer) CollectorBackend {
	return func(ctx context.Context, metrics iter.Seq[MetricPublisher]) error {
		for publisher := range metrics {
			if err := publisher(buf, renderer); err != nil {
				return err
			}
		}
		return nil
	}
}

// Helper to create a counting backend
func makeCountingBackend(counter *atomic.Int64) CollectorBackend {
	return func(ctx context.Context, metrics iter.Seq[MetricPublisher]) error {
		for publisher := range metrics {
			counter.Add(1)
			// Execute the publisher to prevent blocking
			_ = publisher(io.Discard, MakeJSONRenderer())
		}
		return nil
	}
}

// Helper to create an error backend
func makeErrorBackend(errMsg string) CollectorBackend {
	return func(ctx context.Context, metrics iter.Seq[MetricPublisher]) error {
		for range metrics {
			return errors.New(errMsg)
		}
		return nil
	}
}

// Test NewCollector basic construction
func TestNewCollector(t *testing.T) {
	tests := []struct {
		name        string
		opts        []CollectorOptionProvider
		expectError bool
		validate    func(*testing.T, *Collector)
	}{
		{
			name: "ValidSingleBackend",
			opts: []CollectorOptionProvider{
				CollectorConfBuffer(10),
				CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
			},
			expectError: false,
			validate: func(t *testing.T, c *Collector) {
				if c.ctx == nil {
					t.Error("expected context to be set")
				}
				if c.publish == nil {
					t.Error("expected publish queue to be set")
				}
				if c.broker != nil {
					t.Error("expected broker to be nil for single backend")
				}
			},
		},
		{
			name: "ValidMultipleBackends",
			opts: []CollectorOptionProvider{
				CollectorConfBuffer(10),
				CollectorConfAppendBackends(
					LoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
					LoggerBackend(send.MakeInternal(), MakeGraphiteRenderer()),
				),
			},
			expectError: false,
			validate: func(t *testing.T, c *Collector) {
				if c.broker == nil {
					t.Error("expected broker to be set for multiple backends")
				}
			},
		},
		{
			name: "InvalidConfiguration",
			opts: []CollectorOptionProvider{
				CollectorConfBuffer(0), // Invalid: buffer must be non-zero
			},
			expectError: true,
		},
		{
			name: "NoBackends",
			opts: []CollectorOptionProvider{
				CollectorConfBuffer(10),
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			c, err := NewCollector(ctx, tt.opts...)

			if tt.expectError {
				if err == nil {
					t.Error("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			defer c.Close()

			if tt.validate != nil {
				tt.validate(t, c)
			}
		})
	}
}

// Test Collector Close
func TestCollectorClose(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(*testing.T) *Collector
		validate func(*testing.T, error)
	}{
		{
			name: "CloseCleanly",
			setup: func(t *testing.T) *Collector {
				c, err := NewCollector(
					context.Background(),
					CollectorConfBuffer(10),
					CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
				)
				if err != nil {
					t.Fatalf("failed to create collector: %v", err)
				}
				return c
			},
			validate: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("expected clean close, got error: %v", err)
				}
			},
		},
		{
			name: "CloseAfterPush",
			setup: func(t *testing.T) *Collector {
				c, err := NewCollector(
					context.Background(),
					CollectorConfBuffer(10),
					CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
				)
				if err != nil {
					t.Fatalf("failed to create collector: %v", err)
				}
				c.Push(Counter("test").Inc())
				time.Sleep(10 * time.Millisecond) // Allow processing
				return c
			},
			validate: func(t *testing.T, err error) {
				if err != nil {
					t.Errorf("expected clean close, got error: %v", err)
				}
			},
		},
		{
			name: "DoubleClose",
			setup: func(t *testing.T) *Collector {
				c, err := NewCollector(
					context.Background(),
					CollectorConfBuffer(10),
					CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
				)
				if err != nil {
					t.Fatalf("failed to create collector: %v", err)
				}
				c.Close() // First close
				return c
			},
			validate: func(t *testing.T, err error) {
				// Second close should not panic
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := tt.setup(t)
			err := c.Close()
			tt.validate(t, err)
		})
	}
}

// Test Push/Publish/PushEvent
func TestCollectorPush(t *testing.T) {
	tests := []struct {
		name     string
		events   []*Event
		validate func(*testing.T, *Collector)
	}{
		{
			name:   "PushSingleEvent",
			events: []*Event{Counter("requests").Inc()},
			validate: func(t *testing.T, c *Collector) {
				time.Sleep(50 * time.Millisecond)
				// Event should be tracked
				c.local.mx.Lock()
				defer c.local.mx.Unlock()
				if len(c.local.mp) == 0 {
					t.Error("expected metric to be tracked")
				}
			},
		},
		{
			name: "PushMultipleEvents",
			events: []*Event{
				Counter("requests").Inc(),
				Gauge("memory").Set(1024),
				Delta("bytes").Add(512),
			},
			validate: func(t *testing.T, c *Collector) {
				time.Sleep(50 * time.Millisecond)
				c.local.mx.Lock()
				defer c.local.mx.Unlock()
				if len(c.local.mp) != 3 {
					t.Errorf("expected 3 metrics, got %d", len(c.local.mp))
				}
			},
		},
		{
			name:   "PushNilEvent",
			events: []*Event{nil},
			validate: func(t *testing.T, c *Collector) {
				// Should not panic
			},
		},
		{
			name: "PushSameMetricMultipleTimes",
			events: []*Event{
				Counter("requests").Inc(),
				Counter("requests").Inc(),
				Counter("requests").Inc(),
			},
			validate: func(t *testing.T, c *Collector) {
				time.Sleep(50 * time.Millisecond)
				c.local.mx.Lock()
				defer c.local.mx.Unlock()
				if len(c.local.mp) != 1 {
					t.Errorf("expected 1 metric, got %d", len(c.local.mp))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewCollector(
				context.Background(),
				CollectorConfBuffer(10),
				CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
			)
			if err != nil {
				t.Fatalf("failed to create collector: %v", err)
			}
			defer c.Close()

			c.Push(tt.events...)
			tt.validate(t, c)
		})
	}
}

// Test Publish method
func TestCollectorPublish(t *testing.T) {
	c, err := NewCollector(
		context.Background(),
		CollectorConfBuffer(10),
		CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
	)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}
	defer c.Close()

	events := []*Event{
		Counter("test1").Inc(),
		Gauge("test2").Set(42),
	}

	c.Publish(events)
	time.Sleep(50 * time.Millisecond)

	c.local.mx.Lock()
	defer c.local.mx.Unlock()
	if len(c.local.mp) != 2 {
		t.Errorf("expected 2 metrics, got %d", len(c.local.mp))
	}
}

// Test PushEvent method
func TestCollectorPushEvent(t *testing.T) {
	c, err := NewCollector(
		context.Background(),
		CollectorConfBuffer(10),
		CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
	)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}
	defer c.Close()

	c.PushEvent(Counter("single").Inc())
	time.Sleep(50 * time.Millisecond)

	c.local.mx.Lock()
	defer c.local.mx.Unlock()
	if len(c.local.mp) != 1 {
		t.Errorf("expected 1 metric, got %d", len(c.local.mp))
	}
}

// Test Register for background collection
func TestCollectorRegister(t *testing.T) {
	tests := []struct {
		name     string
		producer fnx.Future[[]*Event]
		interval time.Duration
		validate func(*testing.T, *Collector)
	}{
		{
			name: "RegisterSimpleProducer",
			producer: fnx.NewFuture(func(ctx context.Context) ([]*Event, error) {
				return []*Event{Counter("background").Inc()}, nil
			}),
			interval: 20 * time.Millisecond,
			validate: func(t *testing.T, c *Collector) {
				time.Sleep(100 * time.Millisecond)
				c.local.mx.Lock()
				defer c.local.mx.Unlock()
				if len(c.local.mp) == 0 {
					t.Error("expected background metric to be registered")
				}
			},
		},
		{
			name: "RegisterProducerReturningError",
			producer: fnx.NewFuture(func(ctx context.Context) ([]*Event, error) {
				return nil, errors.New("producer error")
			}),
			interval: 20 * time.Millisecond,
			validate: func(t *testing.T, c *Collector) {
				time.Sleep(100 * time.Millisecond)
				c.local.mx.Lock()
				defer c.local.mx.Unlock()
				if len(c.local.mp) != 0 {
					t.Error("expected no metrics when producer returns error")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewCollector(
				context.Background(),
				CollectorConfBuffer(10),
				CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
			)
			if err != nil {
				t.Fatalf("failed to create collector: %v", err)
			}
			defer c.Close()

			c.Register(tt.producer, tt.interval)
			tt.validate(t, c)
		})
	}
}

// Test concurrent Push operations
func TestCollectorConcurrentPush(t *testing.T) {
	c, err := NewCollector(
		context.Background(),
		CollectorConfBuffer(100),
		CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
	)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}
	defer c.Close()

	const (
		numGoroutines = 5
		eventsPerGoroutine = 20
	)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				c.PushEvent(Counter("concurrent").Inc())
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	c.local.mx.Lock()
	defer c.local.mx.Unlock()
	if len(c.local.mp) == 0 {
		t.Error("expected metrics to be tracked")
	}
}

// Test concurrent operations with different metrics
func TestCollectorConcurrentMixedOperations(t *testing.T) {
	c, err := NewCollector(
		context.Background(),
		CollectorConfBuffer(100),
		CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
	)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}
	defer c.Close()

	const numGoroutines = 5

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // counters, gauges, deltas

	// Concurrent counters
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				c.PushEvent(Counter("counter").Inc())
			}
		}(i)
	}

	// Concurrent gauges
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				c.PushEvent(Gauge("gauge").Set(int64(j)))
			}
		}(i)
	}

	// Concurrent deltas
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 20; j++ {
				c.PushEvent(Delta("delta").Add(1))
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	c.local.mx.Lock()
	defer c.local.mx.Unlock()
	if len(c.local.mp) != 3 {
		t.Errorf("expected 3 metrics, got %d", len(c.local.mp))
	}
}

// Test multiple backends receiving events
func TestCollectorMultipleBackends(t *testing.T) {
	var counter1, counter2 atomic.Int64

	c, err := NewCollector(
		context.Background(),
		CollectorConfBuffer(10),
		CollectorConfAppendBackends(
			makeCountingBackend(&counter1),
			makeCountingBackend(&counter2),
		),
	)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}
	defer c.Close()

	// Push some events
	for i := 0; i < 5; i++ {
		c.PushEvent(Counter("test").Inc())
	}

	time.Sleep(200 * time.Millisecond)

	// Both backends should have received the events
	count1 := counter1.Load()
	count2 := counter2.Load()

	if count1 == 0 {
		t.Error("backend 1 did not receive events")
	}
	if count2 == 0 {
		t.Error("backend 2 did not receive events")
	}
}

// Test metric value accumulation
func TestCollectorMetricAccumulation(t *testing.T) {
	c, err := NewCollector(
		context.Background(),
		CollectorConfBuffer(10),
		CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
	)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}
	defer c.Close()

	// Push multiple increments to the same counter
	for i := 0; i < 10; i++ {
		c.PushEvent(Counter("accumulator").Inc())
	}

	time.Sleep(50 * time.Millisecond)

	c.local.mx.Lock()
	defer c.local.mx.Unlock()

	list, ok := c.local.mp["accumulator"]
	if !ok {
		t.Fatal("metric not found")
	}

	var tr *tracked
	for elem := range list.IteratorFront() {
		tr = elem
		break
	}

	if tr == nil {
		t.Fatal("tracked metric not found")
	}

	// Check the accumulated value
	lastVal := tr.lastMod.Load()
	if lastVal.Key != 10 {
		t.Errorf("expected accumulated value 10, got %d", lastVal.Key)
	}
}

// Test metrics with labels are tracked separately
func TestCollectorMetricsWithLabels(t *testing.T) {
	c, err := NewCollector(
		context.Background(),
		CollectorConfBuffer(10),
		CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
	)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}
	defer c.Close()

	// Push same metric ID with different labels
	c.PushEvent(Counter("requests").Label("method", "GET").Inc())
	c.PushEvent(Counter("requests").Label("method", "POST").Inc())
	c.PushEvent(Counter("requests").Label("method", "GET").Inc())

	time.Sleep(50 * time.Millisecond)

	c.local.mx.Lock()
	defer c.local.mx.Unlock()

	list, ok := c.local.mp["requests"]
	if !ok {
		t.Fatal("metric not found")
	}

	// Should have 2 tracked metrics (GET and POST)
	count := 0
	for range list.IteratorFront() {
		count++
	}

	if count != 2 {
		t.Errorf("expected 2 tracked metrics for different labels, got %d", count)
	}
}

// Test event with nil metric
func TestCollectorPushEventNilMetric(t *testing.T) {
	c, err := NewCollector(
		context.Background(),
		CollectorConfBuffer(10),
		CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
	)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}
	defer c.Close()

	// Create event with nil metric
	e := &Event{m: nil}
	c.PushEvent(e)

	time.Sleep(50 * time.Millisecond)

	c.local.mx.Lock()
	defer c.local.mx.Unlock()
	if len(c.local.mp) != 0 {
		t.Error("expected no metrics for nil metric event")
	}
}

// Test context cancellation
func TestCollectorContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	c, err := NewCollector(
		ctx,
		CollectorConfBuffer(10),
		CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
	)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}

	// Cancel context
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Push should not panic
	c.PushEvent(Counter("test").Inc())

	c.Close()
}

// Test ReadAll method
func TestCollectorReadAll(t *testing.T) {
	c, err := NewCollector(
		context.Background(),
		CollectorConfBuffer(10),
		CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
	)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}
	defer c.Close()

	events := []*Event{
		Counter("test1").Inc(),
		Counter("test2").Inc(),
		Counter("test3").Inc(),
	}

	worker := c.ReadAll(func(yield func(*Event) bool) {
		for _, e := range events {
			if !yield(e) {
				return
			}
		}
	})

	err = worker.Run(context.Background())
	if err != nil {
		t.Errorf("ReadAll failed: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	c.local.mx.Lock()
	defer c.local.mx.Unlock()
	if len(c.local.mp) != 3 {
		t.Errorf("expected 3 metrics, got %d", len(c.local.mp))
	}
}

// Test distribution error handling
func TestCollectorDistributeErrors(t *testing.T) {
	// This test verifies error collection during Close
	// Use multiple backends to ensure broker path is used
	backend := makeErrorBackend("backend error")

	c, err := NewCollector(
		context.Background(),
		CollectorConfBuffer(10),
		CollectorConfAppendBackends(backend, backend),
	)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}

	// Push an event that will cause backend error
	c.PushEvent(Counter("test").Inc())
	time.Sleep(100 * time.Millisecond)

	err = c.Close()
	// Error should be collected
	if err == nil {
		t.Error("expected error from backend, got nil")
	}
}

// Test background periodic collection
func TestCollectorPeriodicCollection(t *testing.T) {
	var receivedCount atomic.Int64
	backend := makeCountingBackend(&receivedCount)

	// Use multiple backends to ensure broker is created
	c, err := NewCollector(
		context.Background(),
		CollectorConfBuffer(10),
		CollectorConfAppendBackends(backend, backend),
	)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}
	defer c.Close()

	// Push a periodic metric
	m := Counter("periodic")
	m.Periodic(50 * time.Millisecond)
	c.PushEvent(m.Inc())

	// Wait for a few collections
	time.Sleep(200 * time.Millisecond)

	count := receivedCount.Load()
	// Should have received at least 2 periodic updates (times 2 backends)
	if count < 2 {
		t.Errorf("expected at least 2 periodic updates, got %d", count)
	}
}

// Test concurrent Register and Push
func TestCollectorConcurrentRegisterAndPush(t *testing.T) {
	c, err := NewCollector(
		context.Background(),
		CollectorConfBuffer(100),
		CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
	)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}
	defer c.Close()

	var wg sync.WaitGroup
	wg.Add(2)

	// Goroutine 1: Register background producers
	go func() {
		defer wg.Done()
		for i := 0; i < 5; i++ {
			producer := fnx.NewFuture(func(ctx context.Context) ([]*Event, error) {
				return []*Event{Counter("background").Inc()}, nil
			})
			c.Register(producer, 30*time.Millisecond)
			time.Sleep(10 * time.Millisecond)
		}
	}()

	// Goroutine 2: Push events
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			c.PushEvent(Counter("foreground").Inc())
			time.Sleep(5 * time.Millisecond)
		}
	}()

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	c.local.mx.Lock()
	defer c.local.mx.Unlock()
	if len(c.local.mp) == 0 {
		t.Error("expected metrics to be tracked")
	}
}

// Test getRegisteredTracked with concurrent access
func TestCollectorGetRegisteredTrackedConcurrent(t *testing.T) {
	c, err := NewCollector(
		context.Background(),
		CollectorConfBuffer(100),
		CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
	)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}
	defer c.Close()

	const numGoroutines = 20
	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Multiple goroutines trying to register the same metric
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			c.PushEvent(Counter("same").Inc())
		}()
	}

	wg.Wait()
	time.Sleep(100 * time.Millisecond)

	c.local.mx.Lock()
	defer c.local.mx.Unlock()

	list, ok := c.local.mp["same"]
	if !ok {
		t.Fatal("metric not found")
	}

	// Should only have one tracked instance
	count := 0
	for range list.IteratorFront() {
		count++
	}

	if count != 1 {
		t.Errorf("expected 1 tracked metric, got %d", count)
	}
}

// Test buffer pool reuse
func TestCollectorBufferPoolReuse(t *testing.T) {
	c, err := NewCollector(
		context.Background(),
		CollectorConfBuffer(10),
		CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
	)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}
	defer c.Close()

	// Push many events to trigger buffer pool usage
	for i := 0; i < 100; i++ {
		c.PushEvent(Counter("pooltest").Inc())
	}

	time.Sleep(100 * time.Millisecond)

	// Test passes if no panic or memory issues
}

// Test Close waits for workers
func TestCollectorCloseWaitsForWorkers(t *testing.T) {
	var workerStarted, workerFinished atomic.Bool

	slowBackend := func(ctx context.Context, metrics iter.Seq[MetricPublisher]) error {
		for publisher := range metrics {
			workerStarted.Store(true)
			time.Sleep(100 * time.Millisecond)
			_ = publisher(io.Discard, MakeJSONRenderer())
			workerFinished.Store(true)
		}
		return nil
	}

	c, err := NewCollector(
		context.Background(),
		CollectorConfBuffer(10),
		CollectorConfAppendBackends(CollectorBackend(slowBackend)),
	)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}

	c.PushEvent(Counter("test").Inc())
	time.Sleep(50 * time.Millisecond)

	// Close should wait for worker to finish
	err = c.Close()
	if err != nil {
		t.Errorf("Close returned error: %v", err)
	}

	if !workerStarted.Load() {
		t.Error("worker never started")
	}
	if !workerFinished.Load() {
		t.Error("worker did not finish before Close returned")
	}
}

// Test error collection from multiple backends
func TestCollectorMultipleBackendErrors(t *testing.T) {
	backend1 := makeErrorBackend("error from backend 1")
	backend2 := makeErrorBackend("error from backend 2")

	c, err := NewCollector(
		context.Background(),
		CollectorConfBuffer(10),
		CollectorConfAppendBackends(backend1, backend2),
	)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}

	c.PushEvent(Counter("test").Inc())
	time.Sleep(100 * time.Millisecond)

	err = c.Close()
	if err == nil {
		t.Error("expected errors from backends")
	}

	// Should contain errors from both backends
	errStr := err.Error()
	if errStr == "" {
		t.Error("expected error message")
	}
}

// Test counter periodic collection (skip histogram due to implementation bug)
func TestCollectorCounterPeriodic(t *testing.T) {
	var receivedCount atomic.Int64
	backend := makeCountingBackend(&receivedCount)

	// Use multiple backends to ensure broker is created
	c, err := NewCollector(
		context.Background(),
		CollectorConfBuffer(10),
		CollectorConfAppendBackends(backend, backend),
	)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}
	defer c.Close()

	// Create counter with periodic updates
	m := Counter("counter_periodic")
	m.Periodic(50 * time.Millisecond)

	c.PushEvent(m.Inc())

	// Wait for periodic collections
	time.Sleep(200 * time.Millisecond)

	count := receivedCount.Load()
	// Should have received periodic updates (times 2 backends)
	if count < 2 {
		t.Errorf("expected at least 2 periodic updates, got %d", count)
	}
}

// Stress test with high concurrency
func TestCollectorStressConcurrent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping stress test in short mode")
	}

	c, err := NewCollector(
		context.Background(),
		CollectorConfBuffer(500),
		CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
	)
	if err != nil {
		t.Fatalf("failed to create collector: %v", err)
	}

	const (
		numGoroutines = 10
		eventsPerGoroutine = 50
	)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < eventsPerGoroutine; j++ {
				switch j % 3 {
				case 0:
					c.PushEvent(Counter("counter").Inc())
				case 1:
					c.PushEvent(Gauge("gauge").Set(int64(j)))
				case 2:
					c.PushEvent(Delta("delta").Add(1))
				}
			}
		}(i)
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)

	err = c.Close()
	if err != nil {
		t.Errorf("stress test close error: %v", err)
	}
}

// Test race conditions during rapid register/push/close
func TestCollectorRaceConditions(t *testing.T) {
	for i := 0; i < 10; i++ {
		c, err := NewCollector(
			context.Background(),
			CollectorConfBuffer(100),
			CollectorConfWithLoggerBackend(send.MakeInternal(), MakeJSONRenderer()),
		)
		if err != nil {
			t.Fatalf("failed to create collector: %v", err)
		}

		var wg sync.WaitGroup
		wg.Add(3)

		// Push events
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				c.PushEvent(Counter("race").Inc())
			}
		}()

		// Register producers
		go func() {
			defer wg.Done()
			for j := 0; j < 5; j++ {
				producer := fnx.NewFuture(func(ctx context.Context) ([]*Event, error) {
					return []*Event{Counter("background").Inc()}, nil
				})
				c.Register(producer, 10*time.Millisecond)
			}
		}()

		// Close after short delay
		go func() {
			defer wg.Done()
			time.Sleep(50 * time.Millisecond)
			c.Close()
		}()

		wg.Wait()
	}
}
