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
)

// MetricLabelRenderer provides an implementation for an ordered set
// of labels (tags) for a specific metric series. MetricLabels are
// rendered and cached in the Collector, and the buffered output, is
// passed as a future to the MetricRenderer function.
type MetricLabelRenderer func(output *bytes.Buffer, labels []dt.Pair[string, string], extra ...dt.Pair[string, string])

// MetricValueRenderer takes an event and writes the output to a
// buffer. This makes it possible to use the metrics system with
// arbitrary output formats and targets.
type MetricValueRenderer func(writer *bytes.Buffer, key string, labels fun.Future[[]byte], value int64, ts time.Time)

// Collector maintains the local state of collected metrics: metric
// series are registered lazily when they are first sent, and the
// collector tracks the value and is responsible for orchestrating.
type Collector struct {
	local adt.Map[string, *dt.List[*tracked]]
	loops adt.Map[time.Duration, fun.Handler[*tracked]]
	pool  adt.Pool[*bytes.Buffer]

	broker  *pubsub.Broker[MetricPublisher]
	publish *pubsub.Deque[MetricPublisher]

	CollectorConf

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

func NewCollector(ctx context.Context, opts ...CollectorOptionProvider) (*Collector, error) {
	conf := &CollectorConf{}
	if err := fun.JoinOptionProviders(opts...).Apply(conf); err != nil {
		return nil, err
	}

	c := &Collector{broker: pubsub.NewDequeBroker(
		ctx,
		pubsub.NewUnlimitedDeque[MetricPublisher](),
		conf.BrokerOptions,
	)}

	c.ctx, c.cancel = context.WithCancel(ctx)
	c.local.Default.SetConstructor(func() *dt.List[*tracked] { return &dt.List[*tracked]{} })
	c.pool.SetConstructor(func() *bytes.Buffer { return &bytes.Buffer{} })
	c.pool.SetCleanupHook(func(buf *bytes.Buffer) *bytes.Buffer { buf.Reset(); return buf })
	ec := c.errs.Handler().Join(func(err error) { ft.WhenCall(err != nil, c.cancel) }).Lock()

	if len(conf.Backends) == 1 {
		conf.Backends[0].Worker(c.publish.Distributor().Iterator()).
			Lock().
			Operation(ec).
			Add(ctx, &c.wg)

		return c, nil
	}

	c.broker = pubsub.NewDequeBroker[MetricPublisher](ctx, c.publish, c.BrokerOptions)
	for idx := range conf.Backends {
		ch := c.broker.Subscribe(ctx)
		conf.Backends[idx].Worker(fun.ChannelIterator(ch)).
			Lock().
			Operation(ec).
			PostHook(func() { c.broker.Unsubscribe(ctx, ch) }).
			Add(ctx, &c.wg)
	}

	ticker := time.NewTicker(500 * time.Microsecond)
	for {
		select {
		case <-ticker.C:
			if c.wg.Num() < len(conf.Backends) && !c.errs.HasErrors() {
				continue
			}
		case <-ctx.Done():
			c.errs.Add(ers.Wrap(ctx.Err(), "did not complete startup"))
		}
		break
	}
	if c.errs.HasErrors() {
		c.wg.Operation().Wait()
		return nil, c.errs.Resolve()
	}
	return c, nil
}

func (c *Collector) Close() error {
	c.cancel()
	if c.broker != nil {
		c.broker.Stop()
	}

	c.errs.Add(c.publish.Close())
	c.wg.Operation().Wait()

	return c.errs.Resolve()
}

func (c *Collector) Push(events ...*Event)   { c.Publish(events) }
func (c *Collector) Publish(events []*Event) { dt.Sliceify(events).Observe(c.PushEvent) }

func (c *Collector) Stream(
	iter *fun.Iterator[*Event],
	opts ...fun.OptionProvider[*fun.WorkerGroupConf],
) fun.Worker {
	return iter.ProcessParallel(fun.Handle(c.PushEvent).Processor(), opts...)
}

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

	c.distribute(func(wr io.Writer) error {
		buf := c.pool.Get()
		defer c.pool.Put(buf)

		c.MetricRenderer(buf, e.m.ID, e.m.labelsf, val, e.ts)

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
				return list.Iterator().Process(proc).Run(ctx)
			}).Operation(ec.Handler()).Launch().Once(),
		).IteratorWithHook(func(iter *fun.Iterator[*tracked]) { iter.AddError(ec.Resolve()) }),
		// transformation function to convert the iterator of
		// tracked
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
		for el := trl.Front(); el.Ok(); el = el.Next() {
			if el.Value().meta.Equal(e.m) {
				return el.Value()
			}
		}
	}

	e.m.coll = ft.Default(e.m.coll, c)
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
						if err := pipe.Iterator().Observe(ctx, func(tr *tracked) {
							buf := c.pool.Get()

							tr.local.Resolve(buf)

							fun.Invariant.Must(c.publish.PushBack(func(wr io.Writer) error {
								defer c.pool.Put(buf)
								return ft.IgnoreFirst(wr.Write(buf.Bytes()))
							}))
						}); err != nil {
							return
						}
					}
				}
			})

			return
		}
	}
}
