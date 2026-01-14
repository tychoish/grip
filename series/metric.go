package series

import (
	"bytes"
	"encoding/json"
	"fmt"
	"iter"
	"sort"
	"time"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/fn"
	"github.com/tychoish/fun/irt"
)

// MetricType determines the kind of metric, in particular how the
// state of the metric is tracked over the lifetime of the
// application.
type MetricType string

const (
	// MetricTypeDeltas represents an integer/numeric value that
	// is rendered as deltas: the difference since the last time
	// the metric was reported.
	MetricTypeDeltas MetricType = "deltas"
	// MetricTypeCounter represents an incrementing metric that
	// increases over the lifetime of the process' lifespan. When
	// reported, the total value of the counter is displayed.
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGuage     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
)

type Metric struct {
	ID       string
	Type     MetricType
	labelSet adt.Once[*adt.OrderedSet[irt.KV[string, string]]]

	// TODO: labelCache should be => adt.Once[fn.Future[iter.Seq2[string, string]]]
	labelCache fn.Future[iter.Seq2[string, string]]
	labelstr   fn.Future[string]

	bufferPool maybeBufferPool

	// internal configuration
	hconf *HistogramConf
	dur   time.Duration
}

func (m *Metric) labels() *adt.OrderedSet[irt.KV[string, string]] {
	return m.labelSet.Do(m.initLabels)
}

func (m *Metric) initLabels() *adt.OrderedSet[irt.KV[string, string]] {
	o := &adt.OrderedSet[irt.KV[string, string]]{}
	return o
}

type maybeBufferPool struct {
	pool *adt.Pool[*bytes.Buffer]
}

func (mbp *maybeBufferPool) Get() *bytes.Buffer {
	if mbp == nil || mbp.pool == nil {
		return &bytes.Buffer{}
	}

	return mbp.pool.Get()
}

func (mbp *maybeBufferPool) Put(buf *bytes.Buffer) {
	if mbp == nil || mbp.pool == nil {
		return
	}
	mbp.pool.Put(buf)
}

func (mbp *maybeBufferPool) Make() *bytes.Buffer {
	if mbp == nil || mbp.pool == nil {
		return &bytes.Buffer{}
	}
	return mbp.pool.Make()
}

func Collect(id string) *Metric { return &Metric{ID: id} }
func Gauge(id string) *Metric   { return &Metric{ID: id, Type: MetricTypeGuage} }
func Counter(id string) *Metric { return &Metric{ID: id, Type: MetricTypeCounter} }
func Delta(id string) *Metric   { return &Metric{ID: id, Type: MetricTypeDeltas} }

func Histogram(id string, opts ...HistogramOptionProvider) *Metric {
	conf := MakeDefaultHistogramConf()
	erc.Invariant(conf.Apply())
	return &Metric{ID: id, Type: MetricTypeHistogram, hconf: conf}
}

func labelCmp(lhs, rhs irt.KV[string, string]) bool {
	switch {
	case lhs.Key < rhs.Key:
		return true
	case lhs.Key == rhs.Key && lhs.Value < rhs.Value:
		return true
	default:
		return false
	}
}

func (m *Metric) Label(k, v string) *Metric {
	m.labels().Add(irt.MakeKV(k, v))
	return m
}
func (m *Metric) MetricType(t MetricType) *Metric { m.Type = t; return m }
func (m *Metric) Annotate(pairs ...irt.KV[string, string]) *Metric {
	m.labels().Extend(irt.Slice(pairs))
	return m
}

func (m *Metric) Labels(set *dt.OrderedSet[irt.KV[string, string]]) *Metric {
	m.labels().Extend(set.Iterator())
	return m
}

func (m *Metric) Equal(two *Metric) bool {
	if m.Type != two.Type || m.ID != two.ID || m.labels().Len() != two.labels().Len() {
		return false
	}

	for p := range m.labels().Iterator() {
		if two.labels().Check(p) {
			continue
		}
		return false
	}
	return true
}

// Periodic sets an interval for the metrics to be reported: new
// events aren't reported for this metric (id+labels) regardless of
// periodic being set on future matching events, but the periodic
// reporting remains.
//
// This periodicity only refers to the _reporting_ of the event, not
// the collection of the event. Register a fun.Producer[[]*Events]
// on the series.Collector for periodic collection.
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
		Tags:  json.RawMessage("{}"),
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

func (m *Metric) Collect(fp fn.Future[int64]) *Event {
	return &Event{m: m, ts: time.Now().UTC(), op: func(int64) int64 { return fp() }}
}

func (m *Metric) CollectAdd(fp fn.Future[int64]) *Event {
	return &Event{m: m, ts: time.Now().UTC(), op: func(in int64) int64 { return in + fp() }}
}

func (m *Metric) factory() localMetricValue {
	switch m.Type {
	case MetricTypeDeltas:
		return &localDelta{metric: m}
	case MetricTypeGuage:
		return &localIntValue{metric: m}
	case MetricTypeCounter:
		return &localIntValue{metric: m}
	case MetricTypeHistogram:
		erc.InvariantOk(m.hconf != nil, "histograms must have configuration")
		conf := m.hconf.factory()()
		return conf
	default:
		panic(fmt.Errorf("%q is not a valid metric type", m.Type))
	}
}

func (m *Metric) resolve() {
	if m.labelCache == nil {
		m.labelCache = fn.MakeFuture(func() iter.Seq2[string, string] {
			var ps []irt.KV[string, string]
			for elem := range m.labels().Iterator() {
				ps = append(ps, elem)
			}

			sort.Slice(ps, func(i, j int) bool { return labelCmp(ps[i], ps[j]) })

			return func(yield func(string, string) bool) {
				for _, p := range ps {
					if !yield(p.Key, p.Value) {
						return
					}
				}
			}
		}).Once()
	}

	if m.labelstr == nil {
		m.labelstr = fn.MakeFuture(func() string {
			buf := m.bufferPool.Get()
			defer m.bufferPool.Put(buf)

			for k, v := range m.labelCache() {
				if buf.Len() > 0 {
					buf.WriteByte(',')
				}
				buf.WriteString(k)
				buf.WriteByte('=')
				buf.WriteString(v)
			}

			return buf.String()
		}).Once()
	}
}
