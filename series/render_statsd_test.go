package series

import (
	"bytes"
	"fmt"
	"iter"
	"time"

	"github.com/tychoish/fun/fn"
)

// RenderMetricStatsd writes a single StatsD-formatted gauge line.
//
// Format: <metric>:<value>|g|#tag1:val1,tag2:val2\n
func RenderMetricStatsd(
	buf *bytes.Buffer,
	key string,
	labels fn.Future[iter.Seq2[string, string]],
	value int64,
	_ time.Time, // StatsD ignores the timestamp; the server applies arrival time.
) {
	buf.WriteString(key)
	buf.WriteByte(':')
	fmt.Fprint(buf, value)
	buf.WriteString("|g")

	if pairs := labels(); pairs != nil {
		first := true
		for k, v := range pairs {
			if first {
				buf.WriteString("|#")
				first = false
			} else {
				buf.WriteByte(',')
			}
			buf.WriteString(k)
			buf.WriteByte(':')
			buf.WriteString(v)
		}
	}

	buf.WriteByte('\n')
}

// MakeStatsdRenderer returns a Renderer that produces StatsD lines.
func MakeStatsdRenderer() Renderer {
	return Renderer{
		Metric:    RenderMetricStatsd,
		Histogram: MakeDefaultHistogramMetricRenderer(RenderMetricStatsd),
	}
}
