package graphite

import (
	"context"
	"io"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/grip/send"
)

type MetricPublisher func(io.Writer) error
type CollectorBackend func(context.Context, *fun.Iterator[MetricPublisher]) error

func (cb CollectorBackend) Worker(iter *fun.Iterator[MetricPublisher]) fun.Worker {
	return func(ctx context.Context) error {
		return ers.Join(cb(ctx, iter), iter.Close())
	}
}

type CollectorBakendFileOptionProvider = fun.OptionProvider[*CollectorBakendFileConf]

type CollectorBakendFileConf struct {
	Directory      string
	FilePrefix     string
	Extension      string
	CounterPadding int
	Megabytes      int
}

func (conf *CollectorBakendFileConf) Validate() error { return nil }

func CollectorBackendFileConfSet(c *CollectorBakendFileConf) CollectorBakendFileOptionProvider {
	return nil
}
func CollectorBackendFileConfDirectory(path string) CollectorBakendFileOptionProvider { return nil }
func CollectorBackendFileConfPrefix(prefix string) CollectorBakendFileOptionProvider  { return nil }
func CollectorBackendFileConfExtension(ext string) CollectorBakendFileOptionProvider  { return nil }
func CollectorBackendFileConfCounterPadding(v int) CollectorBakendFileOptionProvider  { return nil }
func CollectorBackendFileConfRotationSizeMB(v int) CollectorBakendFileOptionProvider  { return nil }

func FileBackend(opts ...CollectorBakendFileOptionProvider) (CollectorBackend, error) {
	conf := &CollectorBakendFileConf{}
	if err := fun.JoinOptionProviders(opts...).Apply(conf); err != nil {
		return nil, err
	}
	targetSizeBytes := conf.Megabytes * 1024 * 1024
	getNextFn := conf.RotatingFileProducer()
	return func(ctx context.Context, iter *fun.Iterator[MetricPublisher]) error {
		var file io.WriteCloser
		var saw *sizeAccountingWriter
		for iter.Next(ctx) {
			if file == nil {
				var err error
				file, err = getNextFn(ctx)
				if err != nil {
					return err
				}
				saw = newSizeAccountingWriter(file)
			}

			op := iter.Value()

			if err := op(saw); err != nil {
				return err
			}

			if saw.Size() >= targetSizeBytes {
				if err := file.Close(); err != nil {
					return err
				}
				file = nil
				saw = nil
			}
		}
		return nil
	}, nil
}

func LoggerBackend(sender send.Sender) CollectorBackend {
	return func(ctx context.Context, iter *fun.Iterator[MetricPublisher]) error {
		wr := send.MakeWriter(sender)
		for iter.Next(ctx) {
			op := iter.Value()
			if err := op(wr); err != nil {
				return err
			}
		}
		return nil
	}
}
