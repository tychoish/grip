package series

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/risky"
)

type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGuage     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
)

type Metric struct {
	ID     string
	Type   MetricType
	labels dt.Set[dt.Pair[string, string]]

	labelsf    fun.Future[[]byte]
	labelCache fun.Future[*dt.Pairs[string, string]]
	labelstr   fun.Future[string]
	// pointer to the collector, for rendering
	// interaction. populated in publish event.
	coll *Collector
	// internal configuration
	hconf *HistogramConf
	dur   time.Duration
}

func Gauge(id string) *Metric   { return &Metric{ID: id, Type: MetricTypeGuage} }
func Counter(id string) *Metric { return &Metric{ID: id, Type: MetricTypeCounter} }
func Histogram(id string, opts ...HistogramOptionProvider) *Metric {
	conf := &HistogramConf{}
	fun.Invariant.Must(fun.JoinOptionProviders(opts...).Apply(conf))
	return &Metric{ID: id, Type: MetricTypeHistogram, hconf: conf}
}

func (m *Metric) Label(k, v string) *Metric { m.labels.Add(dt.MakePair(k, v)); return m }

func (m *Metric) Annotate(pairs ...dt.Pair[string, string]) *Metric {
	m.labels.Populate(dt.Sliceify(pairs).Iterator())
	return m
}

func (m *Metric) AddLabels(set *dt.Set[dt.Pair[string, string]]) { m.labels.Populate(set.Iterator()) }
func (m *Metric) Equal(two *Metric) bool {
	return m.Type == two.Type && m.ID == two.ID && m.labels.Equal(&two.labels)
}

func (m *Metric) Periodic(dur time.Duration) *Metric { m.dur = dur; return m }

type Event struct {
	m        *Metric
	value    int64
	resolved bool
	op       func(int64) int64
	ts       time.Time
}

func (e *Event) String() string {
	if e.m == nil {
		return "Metric<UNKNOWN>"
	}

	if !e.resolved {
		return fmt.Sprintf("Metric<%s> Event<UNRESOLVED>", e.m.ID)
	}

	return fmt.Sprintf("Metric<%s> Labels<%s> Event<%d>", e.m.ID, e.m.labelstr(), e.value)
}

func (e *Event) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID    string         `json:"id"`
		Type  MetricType     `json:"type"`
		Tags  json.Marshaler `json:"tags"`
		Value int64          `json:"value"`
	}{
		ID:    e.m.ID,
		Type:  e.m.Type,
		Tags:  e.m.labelCache(),
		Value: e.value,
	})
}

func (m *Metric) Dec() *Event { return m.Add(-1) }
func (m *Metric) Inc() *Event { return m.Add(1) }
func (m *Metric) Add(v int64) *Event {
	return &Event{m: m, ts: time.Now().UTC(), op: func(in int64) int64 { return in + v }}
}

func (m *Metric) Set(v int64) *Event {
	return &Event{m: m, ts: time.Now().UTC(), op: func(int64) int64 { return v }}
}

func (m *Metric) Collect(fn fun.Future[int64]) *Event {
	return &Event{m: m, ts: time.Now().UTC(), op: func(int64) int64 { return fn() }}
}

func (m *Metric) CollectAdd(fn fun.Future[int64]) *Event {
	return &Event{m: m, ts: time.Now().UTC(), op: func(in int64) int64 { return in + fn() }}
}

func (m *Metric) factory() localMetricValue {
	switch m.Type {
	case MetricTypeCounter:
		return &localDelta{metric: m}
	case MetricTypeGuage:
		return &localGauge{metric: m}
	case MetricTypeHistogram:
		fun.Invariant.OK(m.hconf != nil, "histograms must have configuration")
		conf := m.hconf.factory()()
		return conf
	default:
		panic(fmt.Errorf("%q is not a valid metric type: %w", m.Type, fun.ErrInvariantViolation))
	}
}

func (m *Metric) resolve() {
	if m.Type == MetricTypeHistogram && m.hconf != nil && m.hconf.Renderer == nil {
		if m.coll.DefaultHistogramRender != nil {
			m.hconf.Renderer = m.coll.DefaultHistogramRender
		} else {
			m.hconf.Renderer = MakeDefaultHistogramMetricRenderer(m.coll.MetricRenderer)
		}
	}

	if m.labelsf == nil {
		m.labelsf = fun.Futurize(func() []byte {
			if m.labels.Len() == 0 {
				return nil
			}

			builder := m.coll.pool.Get()
			defer m.coll.pool.Put(builder)
			m.coll.LabelRenderer(builder, m.labelCache().Slice())
			return builder.Bytes()
		}).Once()
	}

	if m.labelCache == nil {
		m.labelCache = fun.Futurize(func() *dt.Pairs[string, string] {
			ps := &dt.Pairs[string, string]{}

			fun.Invariant.Must(ps.Consume(context.Background(), m.labels.Iterator()))

			ps.SortQuick(func(a, b dt.Pair[string, string]) bool {
				return a.Key < b.Key && a.Value < b.Value
			})

			return ps
		}).Once()
	}

	if m.labelstr == nil {
		m.labelstr = fun.Futurize(func() string {
			ps := m.labelCache()
			buf := m.coll.pool.Get()
			defer m.coll.pool.Put(buf)
			risky.Observe(ps.Iterator(), func(p dt.Pair[string, string]) {
				if buf.Len() > 0 {
					buf.WriteByte(';')
				}
				buf.WriteString(p.Key)
				buf.WriteByte('=')
				buf.WriteString(p.Value)
			})

			return buf.String()
		}).Once()
	}
}