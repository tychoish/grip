package series

import (
	"bytes"
	"fmt"
	"time"

	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/fn"
	"github.com/tychoish/fun/fnx"
)

func renderLabelsJSON(buf *bytes.Buffer, labels *dt.Pairs[string, string]) {
	if labels == nil || labels.Len() == 0 {
		return
	}

	buf.WriteString(`"tags":{`)
	first := true
	labels.Stream().ReadAll(fnx.FromHandler(func(label dt.Pair[string, string]) {
		switch {
		case first:
			first = false
		case !first:
			buf.WriteByte(',')
		}
		fmt.Fprintf(buf, `"%s":"%s"`, label.Key, label.Value)
	})).Ignore().Wait()
	buf.WriteString("},")
}

func RenderMetricJSON(buf *bytes.Buffer, key string, labels fn.Future[*dt.Pairs[string, string]], value int64, ts time.Time) {
	fmt.Fprintf(buf, `{"metric":"%s","ts":%d,`, key, ts.UTC().UnixMilli())
	renderLabelsJSON(buf, labels())
	fmt.Fprintf(buf, `"value":%d}`, value)
	buf.WriteByte('\n')
}

func RenderHistogramJSON(
	buf *bytes.Buffer,
	key string,
	labels fn.Future[*dt.Pairs[string, string]],
	sample *dt.Pairs[float64, int64],
	ts time.Time,
) {
	fmt.Fprintf(buf, `{"metric":"%s",`, key)
	renderLabelsJSON(buf, labels())
	buf.WriteString(`"value":{`)

	first := true
	sample.Stream().ReadAll(fnx.FromHandler(func(pair dt.Pair[float64, int64]) {
		switch {
		case first:
			first = false
		case !first:
			buf.WriteByte(',')
		}

		fmt.Fprintf(buf, `"%d":%d`, int(pair.Key*100), pair.Value)
	})).Ignore().Wait()
	fmt.Fprint(buf, "}}")
	buf.WriteByte('\n')
}

func RenderMetricOpenTSB(buf *bytes.Buffer, key string, labels fn.Future[*dt.Pairs[string, string]], value int64, ts time.Time) {
	buf.WriteString("put ")
	buf.WriteString(key)
	buf.WriteByte(' ')
	fmt.Fprint(buf, ts.UTC().UnixMilli())
	buf.WriteByte(' ')
	fmt.Fprint(buf, value)
	if tags := labels(); tags != nil && tags.Len() > 0 {
		buf.WriteByte(' ')
		tags.Stream().ReadAll(fnx.FromHandler(func(label dt.Pair[string, string]) {
			buf.WriteString(label.Key)
			buf.WriteByte('=')
			buf.WriteString(label.Value)
			buf.WriteByte(' ')
		})).Ignore().Wait()
	}
	buf.WriteByte('\n')
}

func RenderMetricGraphite(buf *bytes.Buffer, key string, labels fn.Future[*dt.Pairs[string, string]], value int64, ts time.Time) {
	buf.WriteString(key)
	if tags := labels(); tags != nil && tags.Len() > 0 {
		tags.Stream().ReadAll(fnx.FromHandler(func(label dt.Pair[string, string]) {
			fmt.Fprintf(buf, ";%s=%s", label.Key, label.Value)
		})).Ignore().Wait()
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
