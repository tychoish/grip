package series

import (
	"bufio"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"os"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/ft"
	"github.com/tychoish/grip/send"
)

type MetricPublisher func(io.Writer) error
type CollectorBackend func(context.Context, *fun.Iterator[MetricPublisher]) error

func (cb CollectorBackend) Worker(iter *fun.Iterator[MetricPublisher]) fun.Worker {
	return func(ctx context.Context) error {
		return ers.Join(cb(ctx, iter), iter.Close())
	}
}

type CollectorBakendFileOptionProvider = fun.OptionProvider[*CollectorBackendFileConf]

type CollectorBackendFileConf struct {
	Directory      string
	FilePrefix     string
	Extension      string
	CounterPadding int
	Megabytes      int
	Gzip           bool
}

func (conf *CollectorBackendFileConf) Validate() error {
	ec := &erc.Collector{}
	erc.When(ec, conf.Megabytes < 1, "must specify at least 1mb rotation size")
	erc.When(ec, conf.CounterPadding < 1, "must specify at least 1 didget for counter padding")
	stat, err := os.Stat(conf.Directory)
	erc.When(ec, os.IsNotExist(err) || stat != nil && !stat.IsDir(), "directory must either not exist or be a directory")
	erc.When(ec, conf.FilePrefix == "", "must specify a prefix for data files")
	erc.When(ec, conf.Extension == "", "must specify at prefix for data files")

	return ec.Resolve()
}

func CollectorBackendFileConfSet(c *CollectorBackendFileConf) CollectorBakendFileOptionProvider {
	return func(conf *CollectorBackendFileConf) error { *conf = *c; return nil }
}
func CollectorBackendFileConfDirectory(path string) CollectorBakendFileOptionProvider {
	return func(conf *CollectorBackendFileConf) error { conf.Directory = path; return nil }
}
func CollectorBackendFileConfPrefix(prefix string) CollectorBakendFileOptionProvider {
	return func(conf *CollectorBackendFileConf) error { conf.FilePrefix = prefix; return nil }
}
func CollectorBackendFileConfExtension(ext string) CollectorBakendFileOptionProvider {
	return func(conf *CollectorBackendFileConf) error { conf.Extension = ext; return nil }
}
func CollectorBackendFileConfCounterPadding(v int) CollectorBakendFileOptionProvider {
	return func(conf *CollectorBackendFileConf) error { conf.CounterPadding = v; return nil }
}
func CollectorBackendFileConfRotationSizeMB(v int) CollectorBakendFileOptionProvider {
	return func(conf *CollectorBackendFileConf) error { conf.Megabytes = v; return nil }
}

func FileBackend(opts ...CollectorBakendFileOptionProvider) (CollectorBackend, error) {
	conf := &CollectorBackendFileConf{}
	if err := fun.JoinOptionProviders(opts...).Apply(conf); err != nil {
		return nil, err
	}
	targetSizeBytes := conf.Megabytes * 1024 * 1024
	getNextFn := conf.RotatingFileProducer()
	return func(ctx context.Context, iter *fun.Iterator[MetricPublisher]) error {
		var file io.WriteCloser
		var saw *sizeAccountingWriter
		var buf *bufio.Writer
		var gzp *gzip.Writer
		var wr io.Writer

		for iter.Next(ctx) {
			if file == nil {
				var err error
				file, err = getNextFn(ctx)
				if err != nil {
					return err
				}
				buf = bufio.NewWriter(file)
				saw = newSizeAccountingWriter(buf)
				if conf.Gzip {
					gzp = gzip.NewWriter(saw)
				} else {
					wr = saw
				}
			}

			op := iter.Value()

			if err := op(wr); err != nil {
				return errors.Join(err, ft.SafeDo(buf.Flush), ft.SafeDo(gzp.Close), ft.SafeDo(file.Close))
			}

			if saw.Size() >= targetSizeBytes {
				if err := errors.Join(ft.SafeDo(buf.Flush), ft.SafeDo(gzp.Close), ft.SafeDo(file.Close)); err != nil {
					return err
				}

				file = nil
				saw = nil
				buf = nil
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
