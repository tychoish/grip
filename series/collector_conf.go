package series

import (
	"github.com/tychoish/fun"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/pubsub"
	"github.com/tychoish/grip/send"
)

type CollectorConf struct {
	Backends      []CollectorBackend
	BrokerOptions pubsub.BrokerOptions
	Buffer        int

	LabelRenderer          MetricLabelRenderer
	MetricRenderer         MetricValueRenderer
	DefaultHistogramRender MetricHistogramRenderer
}

func (conf *CollectorConf) Validate() error {
	ec := &erc.Collector{}
	erc.When(ec, len(conf.Backends) == 0, "must specify one or more backends")
	erc.When(ec, conf.MetricRenderer == nil, "must define a metric renderer")
	erc.When(ec, conf.LabelRenderer == nil, "must define a label renderer")
	erc.When(ec, conf.Buffer == 0, "must define buffer size (positive) or negative (unlimited)")
	return ec.Resolve()
}

type CollectorOptionProvider = fun.OptionProvider[*CollectorConf]

func CollectorConfSet(c *CollectorConf) CollectorOptionProvider {
	return func(conf *CollectorConf) error {
		*conf = *c
		return nil
	}
}

func CollectorConfBuffer(size int) CollectorOptionProvider {
	return func(conf *CollectorConf) error {
		conf.Buffer = size
		return nil
	}
}

func CollectorConfAppendBackends(bs ...CollectorBackend) CollectorOptionProvider {
	return func(conf *CollectorConf) error {
		conf.Backends = append(conf.Backends, bs...)
		return nil
	}
}

func CollectorConfWithLoggerBackend(sender send.Sender) CollectorOptionProvider {
	return func(conf *CollectorConf) error {
		conf.Backends = append(conf.Backends, LoggerBackend(sender))
		return nil
	}
}

func CollectorConfWithFileLogger(opts ...CollectorBakendFileOptionProvider) CollectorOptionProvider {
	return func(conf *CollectorConf) error {
		be, err := FileBackend(opts...)
		if err != nil {
			return err
		}

		conf.Backends = append(conf.Backends, be)
		return nil
	}
}

func CollectorConfFileLoggerBackend(opts *CollectorBackendFileConf) CollectorOptionProvider {
	return func(conf *CollectorConf) error {
		be, err := FileBackend(CollectorBackendFileConfSet(opts))
		if err != nil {
			return err
		}
		conf.Backends = append(conf.Backends, be)
		return nil
	}
}

func CollectorConfOutputOpenTSB() CollectorOptionProvider {
	return func(conf *CollectorConf) error {
		conf.LabelRenderer = RenderLabelsOpenTSB
		conf.MetricRenderer = RenderMetricOpenTSB
		conf.DefaultHistogramRender = MakeDefaultHistogramMetricRenderer(RenderMetricOpenTSB)
		return nil
	}
}

func CollectorConfOutputGraphite() CollectorOptionProvider {
	return func(conf *CollectorConf) error {
		conf.LabelRenderer = RenderLabelsGraphite
		conf.MetricRenderer = RenderMetricGraphite
		conf.DefaultHistogramRender = MakeDefaultHistogramMetricRenderer(RenderMetricGraphite)
		return nil
	}
}

func CollectorConfOutputJSON() CollectorOptionProvider {
	return func(conf *CollectorConf) error {
		conf.LabelRenderer = RenderLabelsJSON
		conf.MetricRenderer = RenderMetricJSON
		conf.DefaultHistogramRender = RenderHistogramJSON
		return nil
	}
}

func CollectorConfWithOutput(
	lr MetricLabelRenderer,
	mr MetricValueRenderer,
	hr MetricHistogramRenderer,
) CollectorOptionProvider {
	return func(conf *CollectorConf) error {
		ec := &erc.Collector{}

		erc.When(ec, lr == nil, "unspecified label renderer")
		erc.When(ec, mr == nil, "unspecified metric renderer")
		erc.When(ec, hr == nil, "unspecified histogram renderer")

		if err := ec.Resolve(); err != nil {
			return err
		}

		conf.LabelRenderer = lr
		conf.MetricRenderer = mr
		conf.DefaultHistogramRender = hr

		return nil
	}
}
