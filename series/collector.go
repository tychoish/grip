package series

import (
	"bytes"
	"context"
	"io"
	"sync/atomic"
	"time"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/ft"
	"github.com/tychoish/fun/pubsub"
	"github.com/tychoish/grip"
)

// MetricValueRenderer takes an event and writes the output to a
// buffer. This provides the ability to add support arbitrary output
// formats and targets via dependency injection.
type MetricValueRenderer func(writer *bytes.Buffer, key string, labels fun.Future[*dt.Pairs[string, string]], value int64, ts time.Time)

// Collector maintains the local state of collected metrics: metric
// series are registered lazily when they are first sent, and the
// collector tracks the value and is responsible for orchestrating.
type Collector struct {
	CollectorConf

	// synchronized map tracking metrics and periodic collection operations.
	local adt.Map[string, *dt.List[*tracked]]
	loops adt.Map[time.Duration, fun.Handler[*tracked]]
	pool  adt.Pool[*bytes.Buffer]

	// broker is for cases where there are more than one output
	// system. the broker is backed by the publish deque, but we
	// use the deque directly when there's only one output.
	broker  *pubsub.Broker[MetricPublisher]
	publish *pubsub.Deque[MetricPublisher]

	// lifecycle an error collection.
	ctx    context.Context
	cancel context.CancelFunc
	wg     fun.WaitGroup
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
// that consumes (and processes!) fun.Iterator[series.MetricPublisher]
// objects. Metrics publishers, then are closures that write the
// metrics format to an io.Writer, while the formatting of a message
// is controlled by the <>Renderer function in the Collector
// configuration.
func NewCollector(ctx context.Context, opts ...CollectorOptionProvider) (*Collector, error) {
	c := &Collector{}

	if err := fun.JoinOptionProviders(opts...).Apply(&c.CollectorConf); err != nil {
		return nil, err
	}
	conf := c.CollectorConf

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.publish = pubsub.NewUnlimitedDeque[MetricPublisher]()
	c.local.Default.SetConstructor(func() *dt.List[*tracked] { return &dt.List[*tracked]{} })
	c.pool.SetConstructor(func() *bytes.Buffer { return &bytes.Buffer{} })
	c.pool.SetCleanupHook(func(buf *bytes.Buffer) *bytes.Buffer { buf.Reset(); return buf })
	ec := c.errs.Handler().Lock() // Join(func(err error) { ft.WhenCall(err != nil, c.cancel) }).

	if len(conf.Backends) == 1 {
		c.wg.Launch(c.ctx, func(ctx context.Context) {
			err := conf.Backends[0].Worker(c.publish.Distributor().Iterator()).Run(ctx)
			// .Operation(func(err error) { grip.Warning(err) }))
			grip.Critical(err)
		})

		return c, nil
	}

	pbopts := pubsub.BrokerOptions{
		BufferSize:       4,
		WorkerPoolSize:   16,
		ParallelDispatch: true,
	}

	c.broker = pubsub.NewDequeBroker(ctx, c.publish, pbopts)

	for idx := range conf.Backends {
		ch := c.broker.Subscribe(c.ctx)

		c.wg.Launch(c.ctx, conf.Backends[idx].Worker(pubsub.DistributorChannel(ch).Iterator()).
			Operation(ec).PostHook(func() { c.broker.Unsubscribe(c.ctx, ch) }))
	}

	return c, nil
}

func (c *Collector) Close() error {
	ft.SafeCall(c.cancel)
	c.wg.Operation().Wait()
	if c.broker != nil {
		c.broker.Stop()
	}
	c.errs.Add(c.publish.Close())
	return c.errs.Resolve()
}

func (c *Collector) Stream(
	iter *fun.Iterator[*Event],
	opts ...fun.OptionProvider[*fun.WorkerGroupConf],
) fun.Worker {
	return iter.ProcessParallel(fun.Handle(c.PushEvent).Processor(), opts...)
}

func (c *Collector) Push(events ...*Event)   { c.Publish(events) }
func (c *Collector) Publish(events []*Event) { dt.NewSlice(events).Observe(c.PushEvent) }

func (c *Collector) PushEvent(e *Event) {
	if e.m == nil {
		return
	}

	tr := c.getRegisteredTracked(e)
	val := tr.local.Apply(e.op)
	e.value = val
	e.resolved = true

	tr.lastMod.Set(dt.MakePair(val, e.ts))

	if tr.dur.Load() != 0 {
		return
	}

	c.distribute(func(wr io.Writer, r Renderer) error {
		buf := c.pool.Get()
		defer c.pool.Put(buf)

		r.Metric(buf,
			tr.meta.ID,
			tr.meta.labelCache,
			val, e.ts)

		return ft.IgnoreFirst(wr.Write(buf.Bytes()))
	})
}

// Register runs an event producing function,
func (c *Collector) Register(prod fun.Producer[[]*Event], dur time.Duration) {
	c.wg.Launch(c.ctx, prod.Operation(c.Publish, c.errs.Handler()).Interval(dur))
}

// Iterator iterates through every metric and label combination, and
// takes a (rough) snapshot of each metric. Rough only because the
// timestamps and last metric may not always be (exactly) synchronized
// with regards to eachother.
func (c *Collector) Iterator() *fun.Iterator[MetricSnapshot] {
	pipe := fun.Blocking(make(chan *tracked))
	proc := pipe.Send().Processor()
	ec := &erc.Collector{}

	// This is a pretty terse way of implementing this
	// transformation pipeline, but:
	//
	// map[ids][]trackedMetrics => join([][]trackedMetrics) => []trackedMetrics =>transformationFunction => []MetricSnapshot

	return fun.ConvertIterator(
		// the pipe is just a channel that we turn into an
		// iterator, with a "pre hook" that populates the pipe
		// by starting a goroutine. Errors are captured in the
		// collector.
		pipe.Producer().PreHook(
			c.local.Values().Process(func(ctx context.Context, list *dt.List[*tracked]) error {
				return list.Iterator().Process(proc).PostHook(pipe.Close).Run(ctx)
			}).Operation(ec.Handler()).Go().Once(),
		).IteratorWithHook(erc.IteratorHook[*tracked](ec)),
		// transformation function to convert the iterator of
		// trackedMetrics to metrics snapshots.
		fun.Converter(func(tr *tracked) MetricSnapshot {
			last := tr.lastMod.Load()
			return MetricSnapshot{Name: tr.meta.ID, Labels: tr.meta.labelstr(), Value: last.Key, Timestamp: last.Value}
		}),
	)
}

func (c *Collector) distribute(fn MetricPublisher) {
	switch {
	case c.broker != nil:
		c.broker.Publish(c.ctx, fn)
	case c.publish != nil:
		ers.Ignore(c.publish.PushBack(fn))
	default:
		fun.Invariant.Failure("configuration issue, publication error")
	}
}

////////////////////////////////////////////////////////////////////////
//
// constructor for tracked metrics

type tracked struct {
	meta    *Metric // must be immutable
	local   localMetricValue
	lastMod adt.Atomic[dt.Pair[int64, time.Time]]
	dur     atomic.Int64
}

func newTracked(m *Metric) *tracked {
	return &tracked{meta: m, local: m.factory()}
}

func (c *Collector) getRegisteredTracked(e *Event) *tracked {
	trl := c.local.Get(e.m.ID)

	if trl.Len() > 0 {
		for el := trl.Front(); el.OK(); el = el.Next() {
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
	fun.Invariant.OK(ok)
	handler(tr)
}

func (c *Collector) spawnBackground(dur time.Duration, tr *tracked) {
	pipe := pubsub.NewUnlimitedDeque[*tracked]()

	fun.Invariant.Must(pipe.PushBack(tr))

	for {
		if c.loops.Check(dur) {
			return
		}

		if c.loops.EnsureStore(dur, fun.MakeProcessor(pipe.PushBack).Force) {
			c.wg.Launch(c.ctx, func(ctx context.Context) {
				ticker := time.NewTicker(dur)
				defer ticker.Stop()

				for {
					select {
					case <-ctx.Done():
						return
					case <-ticker.C:
						if err := pipe.Iterator().Observe(func(tr *tracked) {
							c.broker.Publish(c.ctx, func(wr io.Writer, r Renderer) error {
								buf := c.pool.Get()
								defer c.pool.Put(buf)

								tr.local.Resolve(buf, r)

								return ft.IgnoreFirst(wr.Write(buf.Bytes()))
							})
						}).Run(ctx); err != nil {
							return
						}
					}
				}
			})

			return
		}
	}
}
