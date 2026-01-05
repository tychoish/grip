package series

import (
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/opt"
	"github.com/tychoish/fun/pubsub"
	"github.com/tychoish/grip/send"
)

type CollectorConf struct {
	Backends      []CollectorBackend
	BrokerOptions pubsub.BrokerOptions
	Buffer        int
}

func (conf *CollectorConf) Validate() error {
	ec := &erc.Collector{}
	ec.When(len(conf.Backends) == 0, "must specify one or more backends")
	ec.When(conf.Buffer == 0, "must define buffer size (positive) or negative (unlimited)")
	// TODO validate broker options make sense with other buffer options
	return ec.Resolve()
}

type CollectorOptionProvider = opt.Provider[*CollectorConf]

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

func CollectorConfWithLoggerBackend(sender send.Sender, r Renderer) CollectorOptionProvider {
	return func(conf *CollectorConf) error {
		conf.Backends = append(conf.Backends, LoggerBackend(sender, r))
		return nil
	}
}

func CollectorConfWithFileBacked(opts ...CollectorBakendFileOptionProvider) CollectorOptionProvider {
	return func(conf *CollectorConf) error {
		be, err := FileBackend(opts...)
		if err != nil {
			return err
		}

		conf.Backends = append(conf.Backends, be)
		return nil
	}
}

func CollectorConfFileBackend(opts *CollectorBackendFileConf) CollectorOptionProvider {
	return func(conf *CollectorConf) error {
		be, err := FileBackend(CollectorBackendFileConfSet(opts))
		if err != nil {
			return err
		}
		conf.Backends = append(conf.Backends, be)
		return nil
	}
}
