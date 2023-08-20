package series

import (
	"bytes"
	"fmt"
	"time"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/risky"
)

func RenderLabelsJSON(buf *bytes.Buffer, labels []dt.Pair[string, string], extra ...dt.Pair[string, string]) {
	buf.WriteByte('{')
	for _, label := range append(labels, extra...) {
		if buf.Len() != 1 {
			buf.WriteByte(',')
		}

		buf.WriteByte('"')
		buf.WriteString(label.Key)
		buf.WriteString(`":"`)
		buf.WriteString(label.Value)
		buf.WriteByte('"')
	}
	buf.WriteByte('}')
}

func RenderMetricJSON(buf *bytes.Buffer, key string, labels fun.Future[[]byte], value int64, ts time.Time) {
	buf.WriteString(`{"metric":"`)
	buf.WriteString(key)
	buf.WriteString(`","ts":`)
	fmt.Fprint(buf, ts.UTC().UnixMilli())
	buf.WriteByte(',')
	if tags := labels(); tags != nil {
		buf.WriteString(`"tags":{`)
		buf.Write(tags)
		buf.WriteString(`},`)
	}
	buf.WriteString(`"value":`)
	buf.WriteString(fmt.Sprint(value))
	buf.WriteByte('}')
	buf.WriteByte('\n')
}

func RenderHistogramJSON(
	buf *bytes.Buffer,
	key string,
	labels fun.Future[[]byte],
	sample *dt.Pairs[float64, int64],
	ts time.Time,
) {
	buf.WriteString(`{"metric":"`)

	buf.WriteString(key)
	buf.WriteString(`",`)
	if tags := labels(); tags != nil {
		buf.WriteString(`,"tags":{`)
		buf.Write(tags)
		buf.WriteByte('}')
		buf.WriteByte(',')
	}
	buf.WriteString(`"value":{`)
	first := true
	risky.Observe(sample.Iterator(), func(pair dt.Pair[float64, int64]) {
		if !first {
			buf.WriteByte(',')
		}
		first = false
		buf.WriteByte('"')
		fmt.Fprint(buf, int(pair.Key*100))
		buf.WriteString(`":`)
		fmt.Fprint(buf, pair.Value)
	})
	buf.WriteString("}}")
	buf.WriteByte('\n')
}

func RenderLabelsOpenTSB(builder *bytes.Buffer, labels []dt.Pair[string, string], extra ...dt.Pair[string, string]) {
	for _, label := range append(labels, extra...) {
		builder.WriteString(label.Key)
		builder.WriteByte('=')
		builder.WriteString(label.Value)
		builder.WriteByte(' ')
	}
}

func RenderMetricOpenTSB(buf *bytes.Buffer, key string, labels fun.Future[[]byte], value int64, ts time.Time) {
	buf.WriteString("put ")
	buf.WriteString(key)
	buf.WriteByte(' ')
	buf.WriteString(fmt.Sprint(ts.UTC().UnixMilli()))
	buf.WriteByte(' ')
	buf.WriteString(fmt.Sprint(value))
	if tags := labels(); tags != nil {
		buf.WriteByte(' ')
		buf.Write(tags)
	}
	buf.WriteByte('\n')
}

func RenderLabelsGraphite(builder *bytes.Buffer, labels []dt.Pair[string, string], extra ...dt.Pair[string, string]) {
	for _, label := range append(labels, extra...) {
		builder.WriteByte(';')
		builder.WriteString(label.Key)
		builder.WriteByte('=')
		builder.WriteString(label.Value)
	}
}

func RenderMetricGraphite(buf *bytes.Buffer, key string, labels fun.Future[[]byte], value int64, ts time.Time) {
	buf.WriteString(key)
	if tags := labels(); tags != nil {
		buf.Write(tags)
	}
	buf.WriteByte(' ')
	buf.WriteString(fmt.Sprint(value))
	buf.WriteByte(' ')
	buf.WriteString(fmt.Sprint(ts.UTC().UnixMilli()))
	buf.WriteByte('\n')
}
