package series

import (
	"bytes"
	"fmt"
	"time"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/dt"
)

func RenderLabelsJSON(labels []dt.Pair[string, string], builder *bytes.Buffer) {
	builder.WriteByte('{')
	defer builder.WriteByte('}')

	for _, label := range labels {
		if builder.Len() != 1 {
			builder.WriteByte(',')
		}

		builder.WriteByte('"')
		builder.WriteString(label.Key)
		builder.WriteByte('"')
		builder.WriteByte(':')
		builder.WriteString(label.Value)
	}
}

func RenderMetricJSON(key string, value int64, ts time.Time, labels fun.Future[[]byte], buf *bytes.Buffer) {
	buf.WriteString(`{"metric":"`)
	buf.WriteString(key)
	buf.WriteString(`",`)
	buf.WriteString(`"value":`)
	buf.WriteString(fmt.Sprint(value))
	if tags := labels(); tags != nil {
		buf.WriteString(`,"tags":{`)
		buf.Write(tags)
		buf.WriteByte('}')
	}
	buf.WriteByte('\n')
}

func RenderLabelsOpenTSB(labels []dt.Pair[string, string], builder *bytes.Buffer) {
	for _, label := range labels {
		builder.WriteString(label.Key)
		builder.WriteByte('=')
		builder.WriteString(label.Value)
		builder.WriteByte(' ')
	}
}

func RenderMetricOpenTSB(key string, value int64, ts time.Time, labels fun.Future[[]byte], buf *bytes.Buffer) {
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

func RenderLabelsGraphite(labels []dt.Pair[string, string], builder *bytes.Buffer) {
	for _, label := range labels {
		builder.WriteString(label.Key)
		builder.WriteByte('=')
		builder.WriteString(label.Value)
		builder.WriteByte(';')
	}
}

func RenderMetricGraphite(key string, value int64, ts time.Time, labels fun.Future[[]byte], buf *bytes.Buffer) {
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
