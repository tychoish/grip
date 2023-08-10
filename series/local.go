package series

import (
	"bytes"
	"sync/atomic"
	"time"
)

type localMetricValue interface {
	Apply(func(int64) int64) int64
	Resolve(*bytes.Buffer)
}

type localDelta struct {
	delta  atomic.Int64
	total  atomic.Int64
	metric *Metric
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

func (lv *localDelta) Resolve(wr *bytes.Buffer) {
	var delta int64
	for {
		delta = lv.delta.Load()
		if lv.delta.CompareAndSwap(delta, 0) {
			break
		}
	}
	lv.total.Add(delta)
	lv.metric.coll.MetricRenderer(wr, lv.metric.ID, lv.metric.labelsf, delta, time.Now())
}

type localGauge struct {
	value  atomic.Int64
	metric *Metric
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

func (lg *localGauge) Resolve(wr *bytes.Buffer) {
	lg.metric.coll.MetricRenderer(wr, lg.metric.ID, lg.metric.labelsf, lg.value.Load(), time.Now())
}
