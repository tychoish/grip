package series

import (
	"bytes"
	"sync/atomic"
	"time"
)

type localMetricValue interface {
	Apply(func(int64) int64) int64
	Resolve(*bytes.Buffer, Renderer)
	Last() int64
}

type localDelta struct {
	delta  atomic.Int64
	total  atomic.Int64
	metric *Metric
}

func (lv *localDelta) Last() int64 { return lv.total.Load() }

func (lv *localDelta) Apply(op func(int64) int64) int64 {
	var prev, curr int64
	for {
		prev = lv.delta.Load()
		curr = op(prev)
		if lv.delta.CompareAndSwap(prev, curr) {
			return curr
		}
	}
}

func (lv *localDelta) Resolve(wr *bytes.Buffer, r Renderer) {
	var delta int64
	var now time.Time
	for {
		delta = lv.delta.Load()
		if lv.delta.CompareAndSwap(delta, 0) {
			now = time.Now().UTC()
			break
		}
	}
	lv.total.Add(delta)
	r.Metric(wr, lv.metric.ID, lv.metric.labelCache, delta, now)
}

type localIntValue struct {
	value  atomic.Int64
	metric *Metric
}

func (lg *localIntValue) Last() int64 { return lg.value.Load() }
func (lg *localIntValue) Apply(op func(int64) int64) int64 {
	var prev, curr int64
	for {
		prev = lg.value.Load()
		curr = op(prev)
		if lg.value.CompareAndSwap(prev, curr) {
			return curr
		}
	}
}

func (lg *localIntValue) Resolve(wr *bytes.Buffer, r Renderer) {
	r.Metric(wr, lg.metric.ID, lg.metric.labelCache, lg.value.Load(), time.Now())
}
