package metrics

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

func SeriesRendererBSON() series.Renderer {
	return series.Renderer{
		Metric:    RenderMetricBSON,
		Histogram: RenderHistogramBSON,
	}
}

func RenderMetricBSON(buf *bytes.Buffer, key string, labels fun.Future[*dt.Pairs[string, string]], value int64, ts time.Time) {
	doc := birch.DC.Elements(birch.EC.String("metric", key))
	if tags := labels(); tags != nil {
		tagdoc := birch.DC.Make(tags.Len())
		tags.Observe(func(kv dt.Pair[string, string]) { tagdoc.Append(birch.EC.String(kv.Key, kv.Value)) })
		doc.Append(birch.EC.SubDocument("labels", tagdoc))
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
	labels fun.Future[*dt.Pairs[string, string]],
	sample *dt.Pairs[float64, int64],
	ts time.Time,
) {
	doc := birch.DC.Elements(birch.EC.String("metric", key))

	if tags := labels(); tags != nil {
		tagdoc := birch.DC.Make(tags.Len())
		tags.Observe(func(kv dt.Pair[string, string]) { tagdoc.Append(birch.EC.String(kv.Key, kv.Value)) })
		doc.Append(birch.EC.SubDocument("labels", tagdoc))
	}

	doc.Append(birch.EC.Time("ts", ts))
	quants := birch.DC.Make(sample.Len())
	risky.Observe(sample.Iterator(), func(pair dt.Pair[float64, int64]) {
		quants.Append(birch.EC.Int64(fmt.Sprint(int(pair.Key*100)), pair.Value))
	})
	fun.Invariant.Must(ft.IgnoreFirst(doc.WriteTo(wr)))
}
