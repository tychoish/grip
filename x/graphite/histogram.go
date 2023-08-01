package graphite

import (
	"bytes"
	"fmt"
	"time"

	"github.com/tychoish/birch/x/ftdc/hdrhist"
	"github.com/tychoish/fun"
	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/intish"
)

type HistogramConf struct {
	Min               int64
	Max               int64
	SignificantDigits int
	Quantiles         []float64
	OutOfRange        HistogramOutOfRangeOption
	Interval          time.Duration
}

type HistogramOutOfRangeOption int8

const (
	HistogramOutOfRangeINVALID HistogramOutOfRangeOption = iota
	HistogramOutOfRangePanic
	HistogramOutOfRangeIgnore
	HistogramOutOfRangeTruncate
	HistogramOutOfRangeUNSPECIFIED
)

func MakeDefaultHistogramConf() *HistogramConf {
	return &HistogramConf{
		Max:               1000,
		Min:               0,
		SignificantDigits: 4,
		Quantiles:         []float64{.5, .8, .9, .99},
		OutOfRange:        HistogramOutOfRangeTruncate,
		Interval:          500 * time.Millisecond,
	}
}

func (conf *HistogramConf) factory() fun.Future[localMetricValue] {
	return func() localMetricValue {
		out := &localHistogram{}

		out.hdrh.Set(hdrhist.New(
			conf.Min,
			conf.Max,
			conf.SignificantDigits,
		))
		return out
	}
}

func (conf *HistogramConf) Validate() error {
	conf.Interval = intish.Max(conf.Interval, 100*time.Millisecond)

	ec := &erc.Collector{}
	erc.When(ec, conf.Min > conf.Max, "min cannot be larget than the max")
	erc.When(ec, len(conf.Quantiles) <= 1, "must specify more than one bucket")
	erc.When(ec, conf.OutOfRange <= HistogramOutOfRangeINVALID ||
		conf.OutOfRange >= HistogramOutOfRangeUNSPECIFIED,
		"must specify valid behavior for out of range",
	)
	// TODO decide if we need to validate: conf.SignificantDigits > math.Log10(float64(conf.Max-conf.Min))
	return ec.Resolve()
}

type HistogramOptionProvider = fun.OptionProvider[*HistogramConf]

func HistogramConfOutOfRange(spec HistogramOutOfRangeOption) HistogramOptionProvider {
	return func(conf *HistogramConf) error { conf.OutOfRange = spec; return nil }
}
func HistogramConfSet(arg *HistogramConf) HistogramOptionProvider {
	return func(conf *HistogramConf) error { *conf = *arg; return nil }
}
func HistogramConfLowerBound(in int64) HistogramOptionProvider {
	return func(conf *HistogramConf) error { conf.Min = in; return nil }
}
func HistogramConfBounds(min, max int64) HistogramOptionProvider {
	return func(conf *HistogramConf) error { conf.Min = min; conf.Max = max; return nil }
}
func HistogramConfUpperBound(in int64) HistogramOptionProvider {
	return func(conf *HistogramConf) error { conf.Max = in; return nil }
}
func HistogramConfSignifcantDigits(in int) HistogramOptionProvider {
	return func(conf *HistogramConf) error { conf.SignificantDigits = in; return nil }
}
func HistogramConfInterval(dur time.Duration) HistogramOptionProvider {
	return func(conf *HistogramConf) error { conf.Interval = intish.Max(dur, 100*time.Millisecond); return nil }
}
func HistogramConfSetQuantiles(quant []float64) HistogramOptionProvider {
	return func(conf *HistogramConf) (err error) {
		ec := &erc.Collector{}
		for idx, q := range quant {
			erc.Whenf(ec, q < 0, "quantile at index %d has value %f which is less than 0", idx, q)
			erc.Whenf(ec, q > 1, "quantile at index %d has value %f which is more than 1", idx, q)
			quant[idx] = float64(int(q*100+0.5)) / 100
		}
		if ec.HasErrors() {
			return ec.Resolve()
		}

		conf.Quantiles = quant
		return nil
	}
}

type localHistogram struct {
	hdrh adt.Synchronized[*hdrhist.Histogram]
	conf *HistogramConf
}

func (lh *localHistogram) Apply(op func(int64) int64) int64 {
	var val int64
	lh.hdrh.With(func(hist *hdrhist.Histogram) {
		val = op(0)

		switch lh.conf.OutOfRange {
		case HistogramOutOfRangeIgnore:
			if val > lh.conf.Max || val < lh.conf.Min {
				return
			}
		case HistogramOutOfRangeTruncate:
			switch {
			case val > lh.conf.Max:
				val = lh.conf.Max
			case val < lh.conf.Min:
				val = lh.conf.Min
			}
		case HistogramOutOfRangePanic:
			// pass
		}

		fun.Invariant.Must(hist.RecordValue(val))
	})
	return val
}

func (lh *localHistogram) Resolve(m *Metric, wr *bytes.Buffer) {
	now := time.Now().UTC().Round(time.Millisecond)
	lh.hdrh.With(func(in *hdrhist.Histogram) {
		for _, bucket := range lh.conf.Quantiles {
			m.RenderTo(fmt.Sprintf("%s.p%d", m.ID, int(bucket*100)), in.ValueAtQuantile(bucket), now, wr)
		}
	})
}
