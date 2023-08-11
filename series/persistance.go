package series

import (
	"bufio"
	"compress/gzip"
	"context"
	"errors"
	"io"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/ft"
	"github.com/tychoish/fun/pubsub"
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

type CollectorBakendSocketOptionProvider = fun.OptionProvider[*CollectorBackendSocketConf]

type CollectorBackendSocketConf struct {
	Dialer  net.Dialer
	Network string // tcp or udb
	Address string

	DialWorkers       int
	IdleConns         int
	MinDialRetryDelay time.Duration
	MaxDialRetryDelay time.Duration
	DialErrorHandling CollectorBackendSocketErrorOption

	MessageWorkers       int
	NumMessageRetries    int
	MinMessageRetryDelay time.Duration
	MaxMessageRetryDelay time.Duration
	MessageErrorHandling CollectorBackendSocketErrorOption
}

func handleSocketBackedError(
	eh fun.Handler[error],
	op CollectorBackendSocketErrorOption,
	err error,
) (bool, error) {
	switch {
	case err == nil:
		return true, nil
	case ers.ContextExpired(err):
		return true, err
	case errors.Is(err, pubsub.ErrQueueClosed):
		return true, nil
	default:
		switch op {
		case CollectorBackendSocketErrorContinue:
			return false, nil
		case CollectorBackendSocketErrorAbort:
			return true, err
		case CollectorBackendSocketErrorCollect:
			return false, nil
		case CollectorBackendSocketErrorPanic:
			panic(err)
		}
	}
	return false, nil
}

func (conf CollectorBackendSocketConf) handleDialError(eh fun.Handler[error], err error) (bool, error) {
	return handleSocketBackedError(eh, conf.DialErrorHandling, err)
}

func (conf CollectorBackendSocketConf) handleMessageError(eh fun.Handler[error], err error) (bool, error) {
	return handleSocketBackedError(eh, conf.MessageErrorHandling, err)
}

type CollectorBackendSocketErrorOption int8

const (
	CollectorBackendSocketErrorINVALID CollectorBackendSocketErrorOption = iota
	CollectorBackendSocketErrorAbort
	CollectorBackendSocketErrorContinue
	CollectorBackendSocketErrorCollect
	CollectorBackendSocketErrorPanic
	CollectorBackendSocketErrorUNSPECIFIED
)

func (co CollectorBackendSocketErrorOption) poolErrorOptions() fun.OptionProvider[*fun.WorkerGroupConf] {
	return func(conf *fun.WorkerGroupConf) error {
		switch co {
		case CollectorBackendSocketErrorAbort:
			conf.ContinueOnError = false
			conf.ContinueOnPanic = false
		case CollectorBackendSocketErrorContinue:
			conf.ContinueOnError = true
			conf.ContinueOnPanic = true
		case CollectorBackendSocketErrorPanic:
			conf.ContinueOnError = false
			conf.ContinueOnPanic = false
		case CollectorBackendSocketErrorCollect:
			conf.ContinueOnError = true
			conf.ContinueOnPanic = false
		}
		return nil
	}
}

type connCacheItem struct {
	conn    net.Conn
	written uint64
	nerrs   int
	closed  bool
}

func (c *connCacheItem) Write(in []byte) (int, error) {
	n, err := c.conn.Write(in)
	if err != nil {
		c.nerrs++
	}
	c.written += uint64(n)
	return n, err
}

func (c *connCacheItem) Close() error { c.closed = true; return c.conn.Close() }

func SocketBackend(opts ...CollectorBakendSocketOptionProvider) (CollectorBackend, error) {
	conf := &CollectorBackendSocketConf{}
	if err := fun.JoinOptionProviders(opts...).Apply(conf); err != nil {
		return nil, err
	}

	// TODO have a pipe that we can use to put connections in?
	// maybe just a single worker?
	connCache, err := pubsub.NewDeque[*connCacheItem](pubsub.DequeOptions{Capacity: conf.IdleConns})
	if err != nil {
		return nil, err
	}

	ec := &erc.Collector{}
	conPoolWorker := fun.Worker(func(ctx context.Context) error {
		return fun.Worker(func(ctx context.Context) error {
			timer := time.NewTimer(0)
			defer timer.Stop()

			var isFinal bool

		LOOP:
			for {
				if connCache.Len() < conf.IdleConns {
					cc, err := conf.Dialer.DialContext(ctx, conf.Network, conf.Address)
					isFinal, err = conf.handleDialError(ec.Add, err)
					switch {
					case err == nil && cc != nil:
						if _, err = conf.handleDialError(ec.Add, connCache.WaitPushBack(ctx, &connCacheItem{conn: cc})); err != nil {
							return err
						}
					case err == nil && cc == nil:
						continue LOOP
					case isFinal:
						return err
					case !isFinal:
						continue
					}
				}

				timer.Reset(conf.MinDialRetryDelay + time.Duration(rand.Int63n(int64(conf.MaxDialRetryDelay))))
				if connCache.Len() >= conf.IdleConns {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-timer.C:
					}
				}

			}

		}).StartGroup(ctx, conf.DialWorkers).Run(ctx)
	})

	return func(ctx context.Context, iter *fun.Iterator[MetricPublisher]) error {
		return iter.ProcessParallel(
			func(ctx context.Context, pub MetricPublisher) (err error) {
				conn, err := connCache.WaitFront(ctx)
				if err != nil {
					return err
				}
				defer func() {
					if err == nil {
						go ft.Ignore(connCache.WaitPushBack(ctx, conn))
						return
					}
					if ers.ContextExpired(err) || conn == nil {
						return
					}

					err = erc.Join(err, ft.IgnoreFirst(conf.handleMessageError(ec.Add, conn.conn.Close())))
				}()

				timer := time.NewTimer(0)
				defer timer.Stop()
				var isFinal bool
				for i := 0; i < conf.NumMessageRetries; i++ {
					select {
					case <-ctx.Done():
						return ctx.Err()
					case <-timer.C:
					}

					err = pub(conn)

					isFinal, err = conf.handleMessageError(ec.Add, err)
					if isFinal {
						return err
					}

					_ = conn.conn.Close()

					conn, err = connCache.WaitFront(ctx)
					if err != nil {
						return err
					}

					timer.Reset(conf.MinMessageRetryDelay + time.Duration(rand.Int63n(int64(conf.MinMessageRetryDelay))))
				}

				return
			},
			fun.WorkerGroupConfErrorHandler(ec.Add),
			fun.WorkerGroupConfNumWorkers(conf.MessageWorkers),
			conf.MessageErrorHandling.poolErrorOptions(),
		).PreHook(conPoolWorker.Operation(ec.Add)).PostHook(func() { /* do cancel */ }).Run(ctx)
	}, nil
}
