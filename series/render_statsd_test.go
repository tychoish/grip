package series

import (
	"bytes"
	"fmt"
	"time"

	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/fn"
)

// RenderMetricStatsd writes a single StatsD-formatted gauge line.
//
// Format: <metric>:<value>|g|#tag1:val1,tag2:val2\n
func RenderMetricStatsd(
	buf *bytes.Buffer,
	key string,
	labels fn.Future[*dt.Pairs[string, string]],
	value int64,
	_ time.Time, // StatsD ignores the timestamp; the server applies arrival time.
) {
	buf.WriteString(key)
	buf.WriteByte(':')
	fmt.Fprint(buf, value)
	buf.WriteString("|g")

	if pairs := labels(); pairs != nil && pairs.Len() > 0 {
		buf.WriteString("|#")
		first := true
		for p := range pairs.Iterator() {
			if !first {
				buf.WriteByte(',')
			}
			first = false
			buf.WriteString(p.Key)
			buf.WriteByte(':')
			buf.WriteString(p.Value)
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
