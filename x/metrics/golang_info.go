package metrics

import (
	"runtime"
	"sync"
	"time"

	"github.com/tychoish/birch"
	"github.com/tychoish/fun/adt"
	"github.com/tychoish/grip/message"
)

var goStatsCache *goStats

func init() { goStatsCache = &goStats{} }

type goStatsData struct {
	previous int64
	current  int64
}

func (d goStatsData) diff() int64 { return d.current - d.previous }

type goStats struct {
	cgoCalls           goStatsData
	mallocCounter      goStatsData
	freesCounter       goStatsData
	gcRate             goStatsData
	gcPause            uint64
	lastGC             time.Time
	lastCollection     time.Time
	durSinceLastUpdate time.Duration

	sync.Mutex
}

func (s *goStats) update() *runtime.MemStats {
	now := time.Now()

	m := runtime.MemStats{}
	runtime.ReadMemStats(&m)

	s.lastGC = time.Unix(0, int64(m.LastGC))
	s.gcPause = m.PauseNs[(m.NumGC+255)%256]

	s.cgoCalls.previous = s.cgoCalls.current
	s.cgoCalls.current = runtime.NumCgoCall()

	s.mallocCounter.previous = s.mallocCounter.current
	s.mallocCounter.current = int64(m.Mallocs)

	s.freesCounter.previous = s.freesCounter.current
	s.freesCounter.current = int64(m.Frees)

	s.gcRate.previous = s.gcRate.current
	s.gcRate.current = int64(m.NumGC)

	s.durSinceLastUpdate = now.Sub(s.lastCollection)
	s.lastCollection = now

	return &m
}

type statRate struct {
	Delta    int64         `bson:"delta" json:"delta" yaml:"delta"`
	Duration time.Duration `bson:"duration" json:"duration" yaml:"duration"`
}

func (s *goStats) getRate(stat int64) statRate {
	if s.durSinceLastUpdate == 0 {
		return statRate{}
	}

	return statRate{Delta: stat, Duration: s.durSinceLastUpdate}
}

func (s *goStats) cgo() statRate     { return s.getRate(s.cgoCalls.diff()) }
func (s *goStats) mallocs() statRate { return s.getRate(s.mallocCounter.diff()) }
func (s *goStats) frees() statRate   { return s.getRate(s.freesCounter.diff()) }
func (s *goStats) gcs() statRate     { return s.getRate(s.gcRate.diff()) }

func (s statRate) float() float64 {
	if s.Duration == 0 {
		return 0
	}
	return float64(s.Delta) / float64(s.Duration)
}

func (s statRate) int() int64 {
	if s.Duration == 0 {
		return 0
	}
	return s.Delta / int64(s.Duration)
}

// CollectBasicGoStats returns some very basic runtime statistics about the
// current go process, using runtime.MemStats and
// runtime.NumGoroutine.
//
// The data reported for the runtime event metrics (e.g. mallocs,
// frees, gcs, and cgo calls,) are the counts since the last time
// metrics were collected, and are reported as rates calculated since
// the last time the statics were collected.
//
// Values are cached between calls, to produce the deltas. For the
// best results, collect these messages on a regular interval.
//
// Internally, this uses message.Fields message type, which means the
// order of the fields when serialized is not defined and applications
// cannot manipulate the Raw value of this composer.
//
// The basic idea is taken from https://github.com/YoSmudge/go-stats.
func CollectBasicGoStats() message.Composer {
	goStatsCache.Lock()
	defer goStatsCache.Unlock()
	m := goStatsCache.update()

	return message.MakeFields(message.Fields{
		"memory.objects.heap":      m.HeapObjects,
		"memory.summary.alloc":     m.Alloc,
		"memory.summary.system":    m.HeapSys,
		"memory.heap.Idle":         m.HeapIdle,
		"memory.heap.InUse":        m.HeapInuse,
		"memory.counters.mallocs":  goStatsCache.mallocs().float(),
		"memory.counters.frees":    goStatsCache.frees().float(),
		"gc.rate":                  goStatsCache.gcs().float(),
		"gc.pause.duration.span":   int64(time.Since(goStatsCache.lastGC)),
		"gc.pause.duration.string": time.Since(goStatsCache.lastGC).String(),
		"gc.pause.last.span":       goStatsCache.gcPause,
		"gc.pause.last.string":     time.Duration(goStatsCache.gcPause).String(),
		"goroutines.total":         runtime.NumGoroutine(),
		"cgo.calls":                goStatsCache.cgo().float(),
	})
}

var (
	_ message.Composer        = &GoRuntimeInfo{}
	_ birch.DocumentMarshaler = &GoRuntimeInfo{}
)

// GoRuntimeInfo provides a structured format for data about the
// current go runtime. Also implements the message composer interface.
type GoRuntimeInfo struct {
	Message string `bson:"msg" json:"msg" yaml:"msg"`
	Payload struct {
		HeapObjects uint64        `bson:"memory.objects.heap" json:"memory.objects.heap" yaml:"memory.objects.heap"`
		Alloc       uint64        `bson:"memory.summary.alloc" json:"memory.summary.alloc" yaml:"memory.summary.alloc"`
		HeapSystem  uint64        `bson:"memory.summary.system" json:"memory.summary.system" yaml:"memory.summary.system"`
		HeapIdle    uint64        `bson:"memory.heap.idle" json:"memory.heap.idle" yaml:"memory.heap.idle"`
		HeapInUse   uint64        `bson:"memory.heap.used" json:"memory.heap.used" yaml:"memory.heap.used"`
		Mallocs     int64         `bson:"memory.counters.mallocs" json:"memory.counters.mallocs" yaml:"memory.counters.mallocs"`
		Frees       int64         `bson:"memory.counters.frees" json:"memory.counters.frees" yaml:"memory.counters.frees"`
		GC          int64         `bson:"gc.rate" json:"gc.rate" yaml:"gc.rate"`
		GCPause     time.Duration `bson:"gc.pause.duration.last" json:"gc.pause.last" yaml:"gc.pause.last"`
		GCLatency   time.Duration `bson:"gc.pause.duration.latency" json:"gc.pause.duration.latency" yaml:"gc.pause.duration.latency"`
		Goroutines  int64         `bson:"goroutines.total" json:"goroutines.total" yaml:"goroutines.total"`
		CgoCalls    int64         `bson:"cgo.calls" json:"cgo.calls" yaml:"cgo.calls"`
	}
	message.Base `json:"meta,omitempty" bson:"meta,omitempty" yaml:"meta,omitempty"`

	ct CounterType

	loggable bool
	rendered adt.Once[string]
}

type CounterType int8

const (
	CounterTypeCurrent CounterType = iota
	CounterTypeDeltas
	CounterTypeRates
)

// CollectGoStatsTotals constructs a Composer, which is a
// GoRuntimeInfo internally, that contains data collected from the Go
// runtime about the state of memory use and garbage collection.
//
// The data reported for the runtime event metrics (e.g. mallocs,
// frees, gcs, and cgo calls,) are totals collected since the
// beginning on the runtime.
//
// GoRuntimeInfo also implements the message.Composer interface.
func CollectGoStatsTotals() *GoRuntimeInfo {
	s := &GoRuntimeInfo{}
	s.build()
	return s
}

// MakeGoStatsTotals has the same semantics as CollectGoStatsTotals,
// but additionally allows you to set a message string to annotate the
// data.
//
// GoRuntimeInfo also implements the message.Composer interface.
func MakeGoStatsTotals(msg string) *GoRuntimeInfo {
	s := &GoRuntimeInfo{ct: CounterTypeCurrent}
	s.Message = msg
	s.build()

	return s
}

// CollectGoStatsDeltas constructs a Composer, which is a
// GoRuntimeInfo internally, that contains data collected from the Go
// runtime about the state of memory use and garbage collection.
//
// The data reported for the runtime event metrics (e.g. mallocs,
// frees, gcs, and cgo calls,) are the counts since the last time
// metrics were collected.
//
// Values are cached between calls, to produce the deltas. For the
// best results, collect these messages on a regular interval.
//
// GoRuntimeInfo also implements the message.Composer interface.
func CollectGoStatsDeltas() *GoRuntimeInfo {
	s := &GoRuntimeInfo{ct: CounterTypeDeltas}
	s.build()

	return s
}

// MakeGoStatsDeltas has the same semantics as CollectGoStatsDeltas,
// but additionally allows you to set a message string to annotate the
// data.
//
// GoRuntimeInfo also implements the message.Composer interface.
func MakeGoStatsDeltas(msg string) *GoRuntimeInfo {
	s := &GoRuntimeInfo{ct: CounterTypeDeltas}
	s.Message = msg
	s.build()
	return s
}

// CollectGoStatsRates constructs a Composer, which is a
// GoRuntimeInfo internally, that contains data collected from the Go
// runtime about the state of memory use and garbage collection.
//
// The data reported for the runtime event metrics (e.g. mallocs,
// frees, gcs, and cgo calls,) are the counts since the last time
// metrics were collected, divided by the time since the last
// time the metric was collected, to produce a rate, which is
// calculated using integer division.
//
// For the best results, collect these messages on a regular interval.
func CollectGoStatsRates() *GoRuntimeInfo {
	s := &GoRuntimeInfo{ct: CounterTypeRates}
	s.build()

	return s
}

// MakeGoStatsRates has the same semantics as CollectGoStatsRates,
// but additionally allows you to set a message string to annotate the
// data.
func MakeGoStatsRates(msg string) *GoRuntimeInfo {
	s := &GoRuntimeInfo{ct: CounterTypeRates}
	s.Message = msg
	s.build()
	return s
}

func (s *GoRuntimeInfo) build() {
	goStatsCache.Lock()
	defer goStatsCache.Unlock()
	m := goStatsCache.update()

	s.Payload.HeapObjects = m.HeapObjects
	s.Payload.Alloc = m.Alloc
	s.Payload.HeapSystem = m.HeapSys
	s.Payload.HeapIdle = m.HeapIdle
	s.Payload.HeapInUse = m.HeapInuse
	s.Payload.Goroutines = int64(runtime.NumGoroutine())

	s.Payload.GCLatency = time.Since(goStatsCache.lastGC)
	s.Payload.GCPause = time.Duration(goStatsCache.gcPause)

	switch s.ct {
	case CounterTypeDeltas:
		s.Payload.Mallocs = goStatsCache.mallocs().Delta
		s.Payload.Frees = goStatsCache.frees().Delta
		s.Payload.GC = goStatsCache.gcs().Delta
		s.Payload.CgoCalls = goStatsCache.cgo().Delta
	case CounterTypeRates:
		s.Payload.Mallocs = goStatsCache.mallocs().int()
		s.Payload.Frees = goStatsCache.frees().int()
		s.Payload.GC = goStatsCache.gcs().int()
		s.Payload.CgoCalls = goStatsCache.cgo().int()
	case CounterTypeCurrent:
		s.Payload.Mallocs = goStatsCache.mallocCounter.current
		s.Payload.Frees = goStatsCache.freesCounter.current
		s.Payload.GC = goStatsCache.gcRate.current
		s.Payload.CgoCalls = goStatsCache.cgoCalls.current
	default:
	}

	s.loggable = true
	s.rendered.Set(func() string {
		s.Collect()
		return renderStatsString(s.Message, s.Payload)
	})
}

// Loggable returns true when the GoRuntimeInfo structure is
// populated. Loggable is part of the Composer interface.
func (s *GoRuntimeInfo) Loggable() bool { return s.loggable }
func (*GoRuntimeInfo) Structured() bool { return true }
func (*GoRuntimeInfo) Schema() string   { return "runtime.0" }

// Raw is part of the Composer interface and returns the GoRuntimeInfo
// object itself.
func (s *GoRuntimeInfo) Raw() any {
	s.Collect()

	if s.IncludeMetadata {
		return s
	}

	return s.Payload
}
func (s *GoRuntimeInfo) String() string { return s.rendered.Resolve() }

func (s *GoRuntimeInfo) MarshalDocument() (*birch.Document, error) {
	return birch.DC.Elements(
		birch.EC.Int64("memory.objects.heap", int64(s.Payload.HeapObjects)),
		birch.EC.Int64("memory.summary.alloc", int64(s.Payload.Alloc)),
		birch.EC.Int64("memory.summary.system", int64(s.Payload.HeapSystem)),
		birch.EC.Int64("memory.heap.idle", int64(s.Payload.HeapIdle)),
		birch.EC.Int64("memory.heap.used", int64(s.Payload.HeapInUse)),
		birch.EC.Int64("memory.counters.mallocs", s.Payload.Mallocs),
		birch.EC.Int64("memory.counters.frees", s.Payload.Frees),
		birch.EC.Int64("gc.rate", s.Payload.GC),
		birch.EC.Duration("gc.pause.duration.last", s.Payload.GCPause),
		birch.EC.Duration("gc.pause.duration.latency", s.Payload.GCLatency),
		birch.EC.Int64("goroutines.total", s.Payload.Goroutines),
		birch.EC.Int64("cgo.calls", s.Payload.CgoCalls)), nil
}
