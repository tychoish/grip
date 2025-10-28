package metrics

import (
	"bytes"
	"fmt"
	"time"

	"github.com/tychoish/birch"
	"github.com/tychoish/fun"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/fn"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/ft"
	"github.com/tychoish/grip/series"
)

func SeriesRendererBSON() series.Renderer {
	return series.Renderer{
		Metric:    RenderMetricBSON,
		Histogram: RenderHistogramBSON,
	}
}

func RenderMetricBSON(buf *bytes.Buffer, key string, labels fn.Future[*dt.Pairs[string, string]], value int64, ts time.Time) {
	doc := birch.DC.Elements(birch.EC.String("metric", key))
	if tags := labels(); tags != nil {
		tagdoc := birch.DC.Make(tags.Len())
		tags.ReadAll(fnx.FromHandler(func(kv dt.Pair[string, string]) { tagdoc.Append(birch.EC.String(kv.Key, kv.Value)) })).Ignore().Wait()
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
	labels fn.Future[*dt.Pairs[string, string]],
	sample *dt.Pairs[float64, int64],
	ts time.Time,
) {
	doc := birch.DC.Elements(birch.EC.String("metric", key))

	if tags := labels(); tags != nil {
		tagdoc := birch.DC.Make(tags.Len())
		tags.ReadAll(fnx.FromHandler(func(kv dt.Pair[string, string]) { tagdoc.Append(birch.EC.String(kv.Key, kv.Value)) })).Ignore().Wait()
		doc.Append(birch.EC.SubDocument("labels", tagdoc))
	}

	doc.Append(birch.EC.Time("ts", ts))
	quants := birch.DC.Make(sample.Len())
	for key, value := range sample.Iterator2() {
		quants.Append(birch.EC.Int64(fmt.Sprint(int(key*100)), value))
	}

	fun.Invariant.Must(ft.IgnoreFirst(doc.WriteTo(wr)))
}
