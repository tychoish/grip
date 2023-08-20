package series

import (
	"bytes"
	"fmt"
	"time"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/dt"
)

func renderLabelsJSON(buf *bytes.Buffer, labels *dt.Pairs[string, string]) {
	if labels == nil || labels.Len() == 0 {
		return
	}

	buf.WriteString(`",tags":{`)
	first := true
	labels.Observe(func(label dt.Pair[string, string]) {
		switch {
		case first:
			first = false
		case !first:
			buf.WriteByte(',')
		}
		buf.WriteByte('"')
		buf.WriteString(label.Key)
		buf.WriteString(`":"`)
		buf.WriteString(label.Value)
		buf.WriteByte('"')
	})
	buf.WriteByte('}')
}

func RenderMetricJSON(buf *bytes.Buffer, key string, labels fun.Future[*dt.Pairs[string, string]], value int64, ts time.Time) {
	buf.WriteString(`{"metric":"`)
	buf.WriteString(key)
	buf.WriteString(`","ts":`)
	fmt.Fprint(buf, ts.UTC().UnixMilli())
	renderLabelsJSON(buf, labels())
	buf.WriteString(`",value":`)
	buf.WriteString(fmt.Sprint(value))
	buf.WriteByte('}')
	buf.WriteByte('\n')
}

func RenderHistogramJSON(
	buf *bytes.Buffer,
	key string,
	labels fun.Future[*dt.Pairs[string, string]],
	sample *dt.Pairs[float64, int64],
	ts time.Time,
) {
	buf.WriteString(`{"metric":"`)
	buf.WriteString(key)
	renderLabelsJSON(buf, labels())
	buf.WriteString(`",value":{`)
	first := true
	sample.Observe(func(pair dt.Pair[float64, int64]) {
		switch {
		case first:
			first = false
		case !first:
			buf.WriteByte(',')
		}

		buf.WriteByte('"')
		fmt.Fprint(buf, int(pair.Key*100))
		buf.WriteString(`":`)
		fmt.Fprint(buf, pair.Value)
	})
	buf.WriteString("}}")
	buf.WriteByte('\n')
}

func RenderMetricOpenTSB(buf *bytes.Buffer, key string, labels fun.Future[*dt.Pairs[string, string]], value int64, ts time.Time) {
	buf.WriteString("put ")
	buf.WriteString(key)
	buf.WriteByte(' ')
	buf.WriteString(fmt.Sprint(ts.UTC().UnixMilli()))
	buf.WriteByte(' ')
	buf.WriteString(fmt.Sprint(value))
	if tags := labels(); tags != nil && tags.Len() > 0 {
		buf.WriteByte(' ')
		tags.Observe(func(label dt.Pair[string, string]) {
			buf.WriteString(label.Key)
			buf.WriteByte('=')
			buf.WriteString(label.Value)
			buf.WriteByte(' ')
		})
	}
	buf.WriteByte('\n')
}

func RenderMetricGraphite(buf *bytes.Buffer, key string, labels fun.Future[*dt.Pairs[string, string]], value int64, ts time.Time) {
	buf.WriteString(key)
	if tags := labels(); tags != nil && tags.Len() > 0 {
		tags.Observe(func(label dt.Pair[string, string]) {
			buf.WriteByte(';')
			buf.WriteString(label.Key)
			buf.WriteByte('=')
			buf.WriteString(label.Value)
		})
	}
	buf.WriteByte(' ')
	buf.WriteString(fmt.Sprint(value))
	buf.WriteByte(' ')
	buf.WriteString(fmt.Sprint(ts.UTC().Unix()))
	buf.WriteByte('\n')
}

func MakeOpenTSBLineRenderer() Renderer {
	return Renderer{
		Metric:    RenderMetricOpenTSB,
		Histogram: MakeDefaultHistogramMetricRenderer(RenderMetricOpenTSB),
	}

}
func MakeGraphiteRenderer() Renderer {
	return Renderer{
		Metric:    RenderMetricGraphite,
		Histogram: MakeDefaultHistogramMetricRenderer(RenderMetricGraphite),
	}
}

func MakeJSONRenderer() Renderer {
	return Renderer{
		Metric:    RenderMetricJSON,
		Histogram: MakeDefaultHistogramMetricRenderer(RenderMetricJSON),
	}
}
