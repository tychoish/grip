package series

import (
	"bytes"
	"context"
	"io"
	"iter"
	"sync/atomic"
	"time"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/fn"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/irt"
	"github.com/tychoish/fun/opt"
	"github.com/tychoish/fun/pubsub"
	"github.com/tychoish/fun/wpa"
	"github.com/tychoish/grip"
)

// MetricValueRenderer takes an event and writes the output to a
// buffer. This provides the ability to add support arbitrary output
// formats and targets via dependency injection.
type MetricValueRenderer func(writer *bytes.Buffer, key string, labels fn.Future[iter.Seq2[string, string]], value int64, ts time.Time)

// Collector maintains the local state of collected metrics: metric
// series are registered lazily when they are first sent, and the
// collector tracks the value and is responsible for orchestrating.
type Collector struct {
	CollectorConf

	// synchronized map tracking metrics and periodic collection operations.
	local adt.Map[string, *dt.List[*tracked]]
	loops adt.Map[time.Duration, func(*tracked) error]
	pool  adt.Pool[*bytes.Buffer]

	// broker is for cases where there are more than one output
	// system. the broker is backed by the publish deque, but we
	// use the deque directly when there's only one output.
	broker  *pubsub.Broker[MetricPublisher]
	publish *pubsub.Queue[MetricPublisher]

	// lifecycle an error collection.
	ctx    context.Context
	cancel context.CancelFunc
	wg     fnx.WaitGroup
	errs   erc.Collector
}

// MetricSnapshot is the export format for a metric series at a given
// point of time.
type MetricSnapshot struct {
	Name      string
	Labels    string
	Value     int64
	Timestamp time.Time
}

// NewCollector constructs a collector service that is responsible for
// collecting and distributing metric events. There are several basic
// modes of operation:
//
// - Embedded: Use series.Sender to create in a grip/send.Sender: here
// the collector wraps the sender and intercepts events from normal
// logger messages. The series.WithMetrics helper can attach metrics.
//
// - Directly: You can use the Push/Publish/Stream/PushEvent methods
// to send events to the collector.
//
// - Background: Using the Register() method you can add a function to
// the Collector which will collect its result and distribute them on
// the provided backend.
//
// Output from a collector is managed by CollectorBackends, which may
// be implemented externally (a backend is a fun.Processor function
// that consumes (and processes!) *pubsub.Stream[series.MetricPublisher]
// objects. Metrics publishers, then are closures that write the
// metrics format to an io.Writer, while the formatting of a message
// is controlled by the <>Renderer function in the Collector
// configuration.
func NewCollector(ctx context.Context, opts ...CollectorOptionProvider) (*Collector, error) {
	c := &Collector{}

	if err := opt.Join(opts...).Apply(&c.CollectorConf); err != nil {
		return nil, err
	}
	conf := c.CollectorConf

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.publish = pubsub.NewUnlimitedQueue[MetricPublisher]()
	c.local.Default.SetConstructor(func() *dt.List[*tracked] { return &dt.List[*tracked]{} })
	c.pool.SetConstructor(func() *bytes.Buffer { return &bytes.Buffer{} })
	c.pool.SetCleanupHook(func(buf *bytes.Buffer) *bytes.Buffer { buf.Reset(); return buf })

	if len(conf.Backends) == 1 {
		c.wg.Launch(c.ctx, func(ctx context.Context) {
			grip.Critical(conf.Backends[0].Worker(c.publish.IteratorWait(ctx)).Run(ctx))
		})

		return c, nil
	}

	pbopts := pubsub.BrokerOptions{
		BufferSize:       1,
		WorkerPoolSize:   16,
		ParallelDispatch: true,
	}

	c.broker = pubsub.NewQueueBroker(ctx, c.publish, pbopts)

	for idx := range conf.Backends {
		ch := c.broker.Subscribe(c.ctx)

		c.wg.Launch(c.ctx,
			conf.Backends[idx].
				Worker(irt.Channel(ctx, ch)).
				Operation(c.errs.Push).
				PostHook(func() { c.broker.Unsubscribe(c.ctx, ch) }))
	}

	return c, nil
}

func (c *Collector) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	c.errs.Push(c.publish.Close())
	if c.broker != nil {
		c.broker.Stop()
	}
	c.wg.Operation().Wait()
	return c.errs.Resolve()
}

// ReadAll ingests all events from the input stream (in parallel)
func (c *Collector) ReadAll(st iter.Seq[*Event], opts ...opt.Provider[*wpa.WorkerGroupConf]) fnx.Worker {
	return func(ctx context.Context) error {
		return erc.JoinSeq(wpa.PullWithPool(ctx, st, fnx.NewHandler(func(ctx context.Context, e *Event) error { return c.pushHandler(ctx, e) }), opts...))
	}
}

func (c *Collector) Push(events ...*Event) { c.Publish(events) }
func (c *Collector) Publish(events []*Event) {
	_ = (c.ReadAll(irt.Slice(events)).Run(c.ctx))
}
func (c *Collector) PushEvent(e *Event) { _ = c.pushHandler(c.ctx, e) }

func (c *Collector) publishHandler(ctx context.Context, events []*Event) error {
	return pubsub.SliceStream(events).ReadAll(fnx.FromHandler(c.PushEvent)).Run(ctx)
}

func (c *Collector) pushHandler(ctx context.Context, e *Event) error {
	if e.m == nil {
		return nil
	}

	tr := c.getRegisteredTracked(e)
	val := tr.local.Apply(e.op)
	e.value = val
	e.resolved = true

	tr.lastMod.Set(irt.MakeKV(val, e.ts))

	if tr.dur.Load() != 0 {
		return nil
	}

	return c.distribute(ctx, func(wr io.Writer, r Renderer) error {
		buf := c.pool.Get()
		defer c.pool.Put(buf)

		r.Metric(buf,
			tr.meta.ID,
			tr.meta.labelCache,
			val, e.ts)
		_, err := wr.Write(buf.Bytes())

		return err
	})
}

// Register starts an event-generating function at the specified interval.
func (c *Collector) Register(prod fnx.Future[[]*Event], dur time.Duration) {
	c.wg.Launch(
		c.ctx,
		fnx.Operation(func(ctx context.Context) {
			events, ok := prod.Check(ctx)
			if !ok {
				return
			}
			_ = erc.JoinSeq(wpa.WithHandler(fn.NewHandler(c.PushEvent)).ForEach(irt.Slice(events)).PullAll(ctx))
		}).Interval(dur),
	)
}

// Stream returns a *pubsub.Stream that emits a MetricSnapshot for every
// metric series currently known to the Collector.
//
// The stream is created on demand. When Stream is invoked it walks the
// Collector's internal map of tracked metrics exactly once, converts each
// tracked series to a MetricSnapshot (capturing the most recent value and
// timestamp), sends it downstream, and then closes.  Because the Collector's
// data structures are concurrency-safe, it is safe to call Stream at any
// time—even while metrics are actively being collected or published from other
// goroutines.
//
// The returned stream inherits the Collector's context, so canceling the
// Collector or exhausting the stream will release all underlying resources.
func (c *Collector) Stream() iter.Seq[MetricSnapshot] {
	return func(yield func(MetricSnapshot) bool) {
		for list := range c.local.Values() {
			for tr := range list.IteratorFront() {
				last := tr.lastMod.Load()
				if !yield(MetricSnapshot{
					Name:      tr.meta.ID,
					Labels:    tr.meta.labelstr(),
					Value:     last.Key,
					Timestamp: last.Value,
				}) {
					return
				}
			}
		}
	}
}

func (c *Collector) distribute(ctx context.Context, fn MetricPublisher) error {
	switch {
	case c.broker != nil:
		return c.broker.Send(ctx, fn)
	case c.publish != nil:
		return c.publish.Push(fn)
	default:
		return ers.New("configuration issue, publication error")
	}
}

////////////////////////////////////////////////////////////////////////
//
// constructor for tracked metrics

type tracked struct {
	meta    *Metric // must be immutable
	local   localMetricValue
	lastMod adt.Atomic[irt.KV[int64, time.Time]]
	dur     atomic.Int64
}

func newTracked(m *Metric) *tracked {
	return &tracked{meta: m, local: m.factory()}
}

func (c *Collector) getRegisteredTracked(e *Event) *tracked {
	trl := c.local.Get(e.m.ID)

	if trl.Len() > 0 {
		for el := trl.Front(); el.Ok(); el = el.Next() {
			if el.Value().meta.Equal(e.m) {
				return el.Value()
			}
		}
	}

	e.m.resolve()
	tr := newTracked(e.m)
	trl.PushBack(tr)
	c.addBackground(tr)
	return tr
}

////////////////////////////////////////////////////////////////////////
//
// background metrics collection.

func (c *Collector) addBackground(tr *tracked) {
	dur := time.Duration(tr.dur.Load())
	switch {
	case dur != 0:
		return
	case tr.meta.dur != 0:
		dur = tr.meta.dur
	case tr.meta.Type == MetricTypeHistogram:
		dur = tr.meta.hconf.Interval
	default:
		return
	}
	if dur == 0 {
		return
	}

	if !tr.dur.CompareAndSwap(0, int64(dur)) {
		return
	}

	c.submitBackground(dur, tr)
}

func (c *Collector) submitBackground(dur time.Duration, tr *tracked) {
	if !c.loops.Check(dur) {
		c.spawnBackground(dur, tr)
	}

	handler, ok := c.loops.Load(dur)
	erc.InvariantOk(ok)
	erc.Invariant(handler(tr))
}

func (c *Collector) spawnBackground(dur time.Duration, tr *tracked) {
	pipe := pubsub.NewUnlimitedDeque[*tracked]()

	erc.Invariant(pipe.PushBack(tr))

	for {
		if c.loops.Check(dur) {
			return
		}

		if !c.loops.Set(dur, pipe.PushBack) {
			c.wg.Launch(c.ctx, func(ctx context.Context) {
				ticker := time.NewTicker(dur)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						for tr := range pipe.IteratorFront(c.ctx) {
							c.broker.Publish(c.ctx, func(wr io.Writer, r Renderer) error {
								buf := c.pool.Get()
								defer c.pool.Put(buf)

								tr.local.Resolve(buf, r)
								_, err := wr.Write(buf.Bytes())

								return err
							})
						}
					}
				}
			})

			return
		}
	}
}
