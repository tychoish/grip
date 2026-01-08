package metrics

import (
	"bytes"
	"fmt"
	"iter"
	"time"

	"github.com/tychoish/birch"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/fn"
	"github.com/tychoish/grip/series"
)

func SeriesRendererBSON() series.Renderer {
	return series.Renderer{
		Metric:    RenderMetricBSON,
		Histogram: RenderHistogramBSON,
	}
}

func RenderMetricBSON(buf *bytes.Buffer, key string, labels fn.Future[iter.Seq2[string, string]], value int64, ts time.Time) {
	doc := birch.DC.Elements(birch.EC.String("metric", key))
	if tags := labels(); tags != nil {
		tagdoc := birch.DC.Make(0)
		for k, v := range tags {
			tagdoc.Append(birch.EC.String(k, v))
		}
		doc.Append(birch.EC.SubDocument("labels", tagdoc))
	}
	doc.Append(
		birch.EC.Time("ts", ts),
		birch.EC.Int64("value", value),
	)
	erc.Must(doc.WriteTo(buf))
}

func RenderHistogramBSON(
	wr *bytes.Buffer,
	key string,
	labels fn.Future[iter.Seq2[string, string]],
	sample *dt.OrderedMap[float64, int64],
	ts time.Time,
) {
	doc := birch.DC.Elements(birch.EC.String("metric", key))

	if tags := labels(); tags != nil {
		tagdoc := birch.DC.Make(0)
		for k, v := range tags {
			tagdoc.Append(birch.EC.String(k, v))
		}
		doc.Append(birch.EC.SubDocument("labels", tagdoc))
	}

	doc.Append(birch.EC.Time("ts", ts))
	quants := birch.DC.Make(sample.Len())
	for key, value := range sample.Iterator() {
		quants.Append(birch.EC.Int64(fmt.Sprint(int(key*100)), value))
	}

	erc.Must(doc.WriteTo(wr))
}
