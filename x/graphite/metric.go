package graphite

import (
	"bytes"
	"fmt"
	"time"

	"github.com/tychoish/birch"
	"github.com/tychoish/fun"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/ft"
	"github.com/tychoish/fun/risky"
	"github.com/tychoish/grip"
	"github.com/tychoish/grip/message"
)

// TODO:
//   - filtering sender that will log normally, and also propagate the
//     messages.
//   - convert metrics into standard composers for non-metrics filter.
//   - move event code into root package
//   - implement prom formatting/renderer
//   - implement connection handling for tcp graphite connection
//   - adapters for current x/metrics package functionality/helpers

func example() { //nolint:unused
	grip.Info(WithMetrics(message.Fields{"op": "test"},
		Gauge("new_op").Label("key", "value").Inc(),
		Histogram("new_op").Label("key", "value").Inc(),
	))
	extractMetrics(fun.Futurize(func() message.Fields { return message.Fields{} }))

}

////////////////////////////////////////////////////////////////////////

type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGuage     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
)

type MetricOutputFormat string

const (
	MetricOutputFormatGraphite MetricOutputFormat = "graphite"
	MetricOutputFormatOpenTSDB MetricOutputFormat = "open-tsdb"
	MetricOutputFormatJSON     MetricOutputFormat = "ndjson"
	MetricOutputFormatBSON     MetricOutputFormat = "bson"
)

type Metric struct {
	ID      string
	Type    MetricType
	Format  MetricOutputFormat
	labels  dt.Set[dt.Pair[string, string]]
	labelsf fun.Future[[]byte]

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
	m  *Metric
	op func(int64) int64
	ts time.Time
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
		return &localDelta{}
	case MetricTypeGuage:
		return &localGauge{}
	case MetricTypeHistogram:
		fun.Invariant.OK(m.hconf != nil, "histograms must have configuration")
		return m.hconf.factory()()
	default:
		panic(fmt.Errorf("%q is not a valid metric type: %w", m.Type, fun.ErrInvariantViolation))
	}
}

func (m *Metric) resolve(renderer MetricLabelRenderer, getbuf func() *bytes.Buffer, putbuf func(*bytes.Buffer)) {
	if m.labelsf != nil {
		return
	}

	m.labelsf = fun.Futurize(func() []byte {
		if m.labels.Len() == 0 {
			return nil
		}

		ps := dt.Sliceify(risky.Slice(m.labels.Iterator()))
		ps.Sort(func(a, b dt.Pair[string, string]) bool {
			return a.Key < b.Key && a.Value < b.Value
		})

		builder := getbuf()
		defer putbuf(builder)

		switch m.Format {
		case MetricOutputFormatBSON:
			doc := birch.DC.Make(len(ps))
			ps.Observe(func(label dt.Pair[string, string]) {
				doc.Append(birch.EC.String(label.Key, label.Value))
			})
			return ft.Must(doc.MarshalBSON())
		case MetricOutputFormatJSON:
			builder.WriteByte('{')
			defer builder.WriteByte('}')
		}
		ps.Observe(func(label dt.Pair[string, string]) {
			switch m.Format {
			case MetricOutputFormatGraphite:
				builder.WriteString(label.Key)
				builder.WriteByte('=')
				builder.WriteString(label.Value)
				builder.WriteByte(';')
			case MetricOutputFormatOpenTSDB:
				builder.WriteString(label.Key)
				builder.WriteByte('=')
				builder.WriteString(label.Value)
				builder.WriteByte(' ')
			case MetricOutputFormatJSON:
				if builder.Len() != 1 {
					builder.WriteByte(',')
				}

				builder.WriteByte('"')
				builder.WriteString(label.Key)
				builder.WriteByte('"')
				builder.WriteByte(':')
				builder.WriteString(label.Value)
			}
		})
		return builder.Bytes()
	}).Once()
}

func (m *Metric) RenderTo(key string, value int64, ts time.Time, buf *bytes.Buffer) {
	ts = ts.Round(time.Millisecond)
	switch m.Format {
	case MetricOutputFormatGraphite:
		buf.WriteString(key)
		if tags := m.labelsf(); tags != nil {
			buf.Write(tags)
		}
		buf.WriteByte(' ')
		buf.WriteString(fmt.Sprint(value))
		buf.WriteByte(' ')
		buf.WriteString(fmt.Sprint(ts.UTC().UnixMilli()))
		buf.WriteByte('\n')
	case MetricOutputFormatOpenTSDB:
		buf.WriteString("put ")
		buf.WriteString(key)
		buf.WriteByte(' ')
		buf.WriteString(fmt.Sprint(ts.UTC().UnixMilli()))
		buf.WriteByte(' ')
		buf.WriteString(fmt.Sprint(value))
		if tags := m.labelsf(); tags != nil {
			buf.WriteByte(' ')
			buf.Write(tags)
		}
		buf.WriteByte('\n')
	case MetricOutputFormatJSON:
		buf.WriteString(`{"metric":"`)
		buf.WriteString(key)
		buf.WriteString(`",`)
		buf.WriteString(`"value":`)
		buf.WriteString(fmt.Sprint(value))
		if tags := m.labelsf(); tags != nil {
			buf.WriteString(`,"tags":{`)
			buf.Write(tags)
			buf.WriteByte('}')
		}
		buf.WriteByte('\n')
	case MetricOutputFormatBSON:
		doc := birch.DC.Elements(
			birch.EC.String("metric", key),
			birch.EC.Time("ts", ts),
			birch.EC.Int64("value", value),
		)
		if tags := m.labelsf(); tags != nil {
			doc.Append(birch.EC.SubDocumentFromReader("labels", birch.Reader(tags)))
		}
		fun.Invariant.Must(ft.IgnoreFirst(doc.WriteTo(buf)))
	}
}
