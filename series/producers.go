package series

import (
	"iter"
	"runtime"
	"time"

	"github.com/tychoish/fun/dt"
	"github.com/tychoish/fun/fn"
	"github.com/tychoish/fun/irt"
)

func GoRuntimeEvents(labels ...irt.KV[string, string]) fn.Future[iter.Seq[*Event]] {
	ls := &dt.OrderedSet[irt.KV[string, string]]{}
	ls.Extend(irt.Slice(labels))

	return func() iter.Seq[*Event] {
		m := runtime.MemStats{}
		runtime.ReadMemStats(&m)

		return irt.Args(
			Gauge("memory").Labels(ls).Label("heap", "objects").Set(int64(m.HeapObjects)),
			Gauge("memory").Labels(ls).Label("heap", "alloc").Set(int64(m.HeapAlloc)),
			Gauge("memory").Labels(ls).Label("heap", "system").Set(int64(m.HeapSys)),
			Gauge("memory").Labels(ls).Label("heap", "idle").Set(int64(m.HeapIdle)),
			Gauge("memory").Labels(ls).Label("heap", "used").Set(int64(m.HeapInuse)),
			Delta("memory").Labels(ls).Label("runtime", "mallocs").Set(int64(m.Mallocs)),
			Delta("memory").Labels(ls).Label("runtime", "frees").Set(int64(m.Frees)),
			Gauge("goruntime").Labels(ls).Label("goroutines", "current").Set(int64(runtime.NumGoroutine())),
			Delta("goruntime").Labels(ls).Label("cgo", "calls").Set(runtime.NumCgoCall()),
			Delta("goruntime").Labels(ls).Label("gc", "latency").Set(int64(time.Since(time.Unix(0, int64(m.LastGC))))),
			Delta("goruntime").Labels(ls).Label("gc", "pause").Set(int64(m.PauseNs[(m.NumGC+255)%256])),
			Delta("goruntime").Labels(ls).Label("gc", "passes").Set(int64(m.NumGC)),
		)
	}
}
