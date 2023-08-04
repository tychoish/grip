package graphite

import (
	"bytes"
	"time"

	"github.com/tychoish/birch"
	"github.com/tychoish/fun"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/ft"
	"github.com/tychoish/grip/series"
)

func CollectorConfOutputBSON() series.CollectorOptionProvider {
	return func(conf *series.CollectorConf) error {
		conf.LabelRenderer = RenderLabelsBSON
		conf.MetricRenderer = RenderMetricBSON
		return nil
	}
}

func RenderLabelsBSON(labels []dt.Pair[string, string], output *bytes.Buffer) {
	doc := birch.DC.Make(len(labels))
	dt.Sliceify(labels).Observe(func(label dt.Pair[string, string]) {
		doc.Append(birch.EC.String(label.Key, label.Value))
	})

	fun.Invariant.Must(ft.IgnoreFirst(doc.WriteTo(output)))
}

func RenderMetricBSON(key string, value int64, ts time.Time, labels fun.Future[[]byte], buf *bytes.Buffer) {
	doc := birch.DC.Elements(
		birch.EC.String("metric", key),
		birch.EC.Time("ts", ts),
		birch.EC.Int64("value", value),
	)
	if tags := labels(); tags != nil {
		doc.Append(birch.EC.SubDocumentFromReader("labels", birch.Reader(tags)))
	}
	fun.Invariant.Must(ft.IgnoreFirst(doc.WriteTo(buf)))
}
