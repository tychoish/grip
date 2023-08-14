package series

import (
	"context"
	"runtime"
	"time"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/dt"
)

func GoRuntimeEventProducer(labels ...dt.Pair[string, string]) fun.Producer[[]*series.Event] {
	return func(context.Context) ([]*Event, error) {
		ls := &dt.Set[dt.Pair[string, string]]{}
		ls.Populate(fun.SliceIterator(labels))

		m := runtime.MemStats{}
		runtime.ReadMemStats(&m)

		return []*Event{
			series.Gauge("memory.heap.objects").SetLabels(ls).Set(int64(m.HeapObjects)),
			series.Gauge("memory.heap.alloc").SetLabels(ls).Set(int64(m.HeapAlloc)),
			series.Gauge("memory.heap.system").SetLabels(ls).Set(int64(m.HeapSys)),
			series.Gauge("memory.heap.idle").SetLabels(ls).Set(int64(m.HeapIdle)),
			series.Gauge("memory.heap.used").SetLabels(ls).Set(int64(m.HeapInuse)),
			series.Delta("memory.mallocs").SetLabels(ls).Set(int64(m.Mallocs)),
			series.Delta("memory.frees").SetLabels(ls).Set(int64(m.Frees)),
			series.Gauge("go.runtime.goroutines").SetLabels(ls).Set(int64(runtime.NumGoroutine())),
			series.Delta("go.runtime.cgo").SetLabels(ls).Set(int64(runtime.NumCgoCall())),
			series.Delta("go.runtime.gc.latency").SetLabels(ls).Set(int64(time.Since(time.Unix(0, int64(m.LastGC))))),
			series.Delta("go.runtime.gc.pause").SetLabels(ls).Set(int64(m.PauseNs[(m.NumGC+255)%256])),
			series.Delta("go.runtime.gc.passes").SetLabels(ls).Set(int64(m.NumGC)),
		}, nil
	}
}
