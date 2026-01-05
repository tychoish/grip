package series

import (
	"bytes"
	"fmt"
	"iter"
	"time"

	"github.com/tychoish/fun/fn"
)

func renderLabelsJSON(buf *bytes.Buffer, labels iter.Seq2[string, string]) {
	first := true
	for k, v := range labels {
		switch {
		case first:
			first = false
			buf.WriteString(`"tags":{`)
		case !first:
			buf.WriteByte(',')
		}
		buf.WriteByte('"')
		buf.WriteString(k)
		buf.WriteString(`":"`)
		buf.WriteString(v)
		buf.WriteByte('"')
	}
	if !first {
		buf.WriteString("},")
	}
}

func RenderMetricJSON(buf *bytes.Buffer, key string, labels fn.Future[iter.Seq2[string, string]], value int64, ts time.Time) {
	fmt.Fprintf(buf, `{"metric":"%s","ts":%d,`, key, ts.UTC().UnixMilli())
	renderLabelsJSON(buf, labels())
	fmt.Fprintf(buf, `"value":%d}`, value)
	buf.WriteByte('\n')
}

func RenderHistogramJSON(
	buf *bytes.Buffer,
	key string,
	labels fn.Future[iter.Seq2[string, string]],
	sample iter.Seq2[float64, int64],
	ts time.Time,
) {
	fmt.Fprintf(buf, `{"metric":"%s",`, key)
	renderLabelsJSON(buf, labels())
	buf.WriteString(`"value":{`)

	first := true
	for k, v := range sample {
		switch {
		case first:
			first = false
		case !first:
			buf.WriteByte(',')
		}

		fmt.Fprintf(buf, `"%d":%d`, int(k*100), v)
	}
	fmt.Fprint(buf, "}}")
	buf.WriteByte('\n')
}

func RenderMetricOpenTSB(buf *bytes.Buffer, key string, labels fn.Future[iter.Seq2[string, string]], value int64, ts time.Time) {
	buf.WriteString("put ")
	buf.WriteString(key)
	buf.WriteByte(' ')
	fmt.Fprint(buf, ts.UTC().UnixMilli())
	buf.WriteByte(' ')
	fmt.Fprint(buf, value)

	if tags := labels(); tags != nil {
		for k, v := range labels() {
			buf.WriteByte(' ')
			buf.WriteString(k)
			buf.WriteByte('=')
			buf.WriteString(v)
			buf.WriteByte(' ')
		}
	}
	buf.WriteByte('\n')
}

func RenderMetricGraphite(buf *bytes.Buffer, key string, labels fn.Future[iter.Seq2[string, string]], value int64, ts time.Time) {
	buf.WriteString(key)
	if tags := labels(); tags != nil {
		for k, v := range labels() {
			fmt.Fprintf(buf, ";%s=%s", k, v)
		}
	}

	buf.WriteByte(' ')
	fmt.Fprint(buf, value)
	buf.WriteByte(' ')
	fmt.Fprint(buf, ts.UTC().Unix())
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
