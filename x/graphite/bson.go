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
	"github.com/tychoish/grip/series"
)

func CollectorConfOutputBSON() series.CollectorOptionProvider {
	return func(conf *series.CollectorConf) error {
		conf.LabelRenderer = RenderLabelsBSON
		conf.MetricRenderer = RenderMetricBSON
		conf.DefaultHistogramRender = RenderHistogramBSON
		return nil
	}
}

func RenderLabelsBSON(output *bytes.Buffer, labels []dt.Pair[string, string], extra ...dt.Pair[string, string]) {
	doc := birch.DC.Make(len(labels))
	dt.Sliceify(append(labels, extra...)).Observe(func(label dt.Pair[string, string]) {
		doc.Append(birch.EC.String(label.Key, label.Value))
	})

	fun.Invariant.Must(ft.IgnoreFirst(doc.WriteTo(output)))
}

func RenderMetricBSON(buf *bytes.Buffer, key string, labels fun.Future[[]byte], value int64, ts time.Time) {
	doc := birch.DC.Elements(birch.EC.String("metric", key))
	if tags := labels(); tags != nil {
		doc.Append(birch.EC.SubDocumentFromReader("labels", birch.Reader(tags)))
	}
	doc.Append(
		birch.EC.Time("ts", ts),
		birch.EC.Int64("value", value),
	)
	fun.Invariant.Must(ft.IgnoreFirst(doc.WriteTo(buf)))
}

func RenderHistogramBSON(
	wr *bytes.Buffer,
	key string,
	labels fun.Future[[]byte],
	sample *dt.Pairs[float64, int64],
	ts time.Time,
) {
	doc := birch.DC.Elements(birch.EC.String("metric", key))

	if tags := labels(); tags != nil {
		doc.Append(birch.EC.SubDocumentFromReader("labels", birch.Reader(tags)))
	}
	doc.Append(birch.EC.Time("ts", ts))
	quants := birch.DC.Make(sample.Len())
	risky.Observe(sample.Iterator(), func(pair dt.Pair[float64, int64]) {
		quants.Append(birch.EC.Int64(fmt.Sprint(int(pair.Key*100)), pair.Value))
	})
	fun.Invariant.Must(ft.IgnoreFirst(doc.WriteTo(wr)))
}
