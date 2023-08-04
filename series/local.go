package series

import (
	"bytes"
	"sync/atomic"
	"time"
)

type localMetricValue interface {
	Apply(func(int64) int64) int64
	Resolve(*Metric, *bytes.Buffer)
}

type localDelta struct {
	delta atomic.Int64
	total atomic.Int64
}

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

func (lv *localDelta) Resolve(m *Metric, wr *bytes.Buffer) {
	var delta int64
	for {
		delta = lv.delta.Load()
		if lv.delta.CompareAndSwap(delta, 0) {
			break
		}
	}
	lv.total.Add(delta)
	m.RenderTo(m.ID, delta, time.Now(), wr)
}

type localGauge struct {
	value atomic.Int64
}

func (lg *localGauge) Apply(op func(int64) int64) int64 {
	var prev, curr int64
	for {
		prev = lg.value.Load()
		curr = op(prev)
		if lg.value.CompareAndSwap(prev, curr) {
			return curr
		}
	}
}

func (lg *localGauge) Resolve(m *Metric, wr *bytes.Buffer) {
	m.RenderTo(m.ID, lg.value.Load(), time.Now(), wr)
}
