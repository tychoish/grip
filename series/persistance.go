package series

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/ft"
	"github.com/tychoish/fun/intish"
	"github.com/tychoish/fun/pubsub"
	"github.com/tychoish/grip/send"
)

type MetricPublisher func(io.Writer, Renderer) error

type CollectorBackend fun.Processor[*fun.Iterator[MetricPublisher]]

type Renderer struct {
	Metric    MetricValueRenderer
	Histogram MetricHistogramRenderer
}

func (cb CollectorBackend) Worker(iter *fun.Iterator[MetricPublisher]) fun.Worker {
	return func(ctx context.Context) error { return cb(ctx, iter) }
}

type CollectorBakendFileOptionProvider = fun.OptionProvider[*CollectorBackendFileConf]

type CollectorBackendFileConf struct {
	Directory      string
	FilePrefix     string
	Extension      string
	CounterPadding int
	Megabytes      int
	Gzip           bool
	Renderer       Renderer `json:"-" yaml:"-" db:"-" bson:"-"`
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
func CollectorBackendFileConfWithRenderer(r Renderer) CollectorBakendFileOptionProvider {
	return func(conf *CollectorBackendFileConf) error { conf.Renderer = r; return nil }
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

			if err := op(wr, conf.Renderer); err != nil {
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

func LoggerBackend(sender send.Sender, r Renderer) CollectorBackend {
	wr := send.MakeWriter(sender)
	return func(ctx context.Context, iter *fun.Iterator[MetricPublisher]) error {
		count := 0
		for iter.Next(ctx) {
			op := iter.Value()
			if err := op(wr, r); err != nil {
				return ers.Join(err, wr.Close())
			}
			count++
		}
		return wr.Close()
	}
}

func PassthroughBackend(r Renderer, handler fun.Handler[string], opts ...fun.OptionProvider[*fun.WorkerGroupConf]) CollectorBackend {
	return func(ctx context.Context, iter *fun.Iterator[MetricPublisher]) error {
		pool := &adt.Pool[*bytes.Buffer]{}
		pool.SetConstructor(func() *bytes.Buffer { return &bytes.Buffer{} })
		pool.SetCleanupHook(func(buf *bytes.Buffer) *bytes.Buffer { buf.Reset(); return buf })

		return fun.Converter(
			func(mp MetricPublisher) string {
				buf := pool.Get()
				defer pool.Put(buf)
				fun.Invariant.Must(mp(buf, r))
				return buf.String()
			}).
			ProcessParallel(iter, opts...).
			ProcessParallel(handler.Processor(), opts...).
			Run(ctx)
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

	Renderer Renderer
}

func (conf *CollectorBackendSocketConf) Validate() error {
	conf.DialWorkers = intish.Max(1, conf.DialWorkers)
	conf.MessageWorkers = intish.Max(conf.DialWorkers, conf.MessageWorkers)
	conf.NumMessageRetries = intish.Max(1, conf.NumMessageRetries)
	conf.IdleConns = intish.Max(2*conf.DialWorkers, conf.IdleConns)
	conf.MinDialRetryDelay = intish.Max(100*time.Millisecond, conf.MinDialRetryDelay)
	conf.MaxDialRetryDelay = intish.Max(conf.MinDialRetryDelay, conf.MaxDialRetryDelay)
	conf.MinMessageRetryDelay = intish.Max(100*time.Millisecond, conf.MinMessageRetryDelay)
	conf.MaxMessageRetryDelay = intish.Max(conf.MinMessageRetryDelay, conf.MaxMessageRetryDelay)

	ec := &erc.Collector{}
	ec.Add(conf.DialErrorHandling.Validate())
	ec.Add(conf.MessageErrorHandling.Validate())

	erc.When(ec, conf.Renderer.Histogram == nil, "must specify histogram renderer")
	erc.When(ec, conf.Renderer.Metric == nil, "must specify scalar metrics renderer")

	return ec.Resolve()
}

func CollectorBackendSocketConfWithRenderer(r Renderer) CollectorBakendSocketOptionProvider {
	return func(conf *CollectorBackendSocketConf) error { conf.Renderer = r; return nil }
}
func CollectorBackendSocketConfSet(c *CollectorBackendSocketConf) CollectorBakendSocketOptionProvider {
	return func(conf *CollectorBackendSocketConf) error { *conf = *c; return nil }
}
func CollectorBackendSocketConfDialer(d net.Dialer) CollectorBakendSocketOptionProvider {
	return func(conf *CollectorBackendSocketConf) error { conf.Dialer = d; return nil }
}
func CollectorBackendSocketConfNetowrkTCP() CollectorBakendSocketOptionProvider {
	return func(conf *CollectorBackendSocketConf) error { conf.Network = "tcp"; return nil }
}
func CollectorBackendSocketConfNetowrkUDP() CollectorBakendSocketOptionProvider {
	return func(conf *CollectorBackendSocketConf) error { conf.Network = "udp"; return nil }
}
func CollectorBackendSocketConfAddress(addr string) CollectorBakendSocketOptionProvider {
	return func(conf *CollectorBackendSocketConf) error { conf.Address = addr; return nil }
}
func CollectorBackendSocketConfDialWorkers(n int) CollectorBakendSocketOptionProvider {
	return func(conf *CollectorBackendSocketConf) error { conf.DialWorkers = n; return nil }
}
func CollectorBackendSocketConfIdleConns(n int) CollectorBakendSocketOptionProvider {
	return func(conf *CollectorBackendSocketConf) error { conf.IdleConns = n; return nil }
}
func CollectorBackendSocketConfMinDialRetryDelay(d time.Duration) CollectorBakendSocketOptionProvider {
	return func(conf *CollectorBackendSocketConf) error { conf.MinDialRetryDelay = d; return nil }
}
func CollectorBackendSocketConfMaxDialRetryDelay(d time.Duration) CollectorBakendSocketOptionProvider {
	return func(conf *CollectorBackendSocketConf) error { conf.MaxDialRetryDelay = d; return nil }
}
func CollectorBackendSocketConfDialErrorHandling(eh CollectorBackendSocketErrorOption) CollectorBakendSocketOptionProvider {
	return func(conf *CollectorBackendSocketConf) error { conf.DialErrorHandling = eh; return nil }
}
func CollectorBackendSocketConfMessageWorkers(n int) CollectorBakendSocketOptionProvider {
	return func(conf *CollectorBackendSocketConf) error { conf.MessageWorkers = n; return nil }
}
func CollectorBackendSocketConfNumMessageRetries(n int) CollectorBakendSocketOptionProvider {
	return func(conf *CollectorBackendSocketConf) error { conf.NumMessageRetries = n; return nil }
}
func CollectorBackendSocketConfMinMessageRetryDelay(d time.Duration) CollectorBakendSocketOptionProvider {
	return func(conf *CollectorBackendSocketConf) error { conf.MinMessageRetryDelay = d; return nil }
}
func CollectorBackendSocketConfMaxMessageRetryDelay(d time.Duration) CollectorBakendSocketOptionProvider {
	return func(conf *CollectorBackendSocketConf) error { conf.MaxMessageRetryDelay = d; return nil }
}
func CollectorBackendSocketConfMessageErrorHandling(eh CollectorBackendSocketErrorOption) CollectorBakendSocketOptionProvider {
	return func(conf *CollectorBackendSocketConf) error { conf.MessageErrorHandling = eh; return nil }
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

func (co CollectorBackendSocketErrorOption) Validate() error {
	if co <= CollectorBackendSocketErrorINVALID || co >= CollectorBackendSocketErrorUNSPECIFIED {
		return fmt.Errorf("%d is not a valid error handling option", co)
	}
	return nil
}

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

func (c *connCacheItem) Close() error { c.closed = true; return ft.SafeDo(c.conn.Close) }

func SocketBackend(opts ...CollectorBakendSocketOptionProvider) (CollectorBackend, error) {
	conf := &CollectorBackendSocketConf{}
	if err := fun.JoinOptionProviders(opts...).Apply(conf); err != nil {
		return nil, err
	}

	connCache := make(chan *connCacheItem, 2*(conf.IdleConns+conf.DialWorkers+conf.MessageWorkers))
	connCacheSize := &intish.Atomic[int]{}

	ec := &erc.Collector{}
	var dialOperation fun.Operation = func(ctx context.Context) {
		timer := time.NewTimer(0)
		defer timer.Stop()

		var isFinal bool
	LOOP:
		for {
			if connCacheSize.Load() < conf.IdleConns {
				cc, err := conf.Dialer.DialContext(ctx, conf.Network, conf.Address)
				isFinal, err = conf.handleDialError(ec.Add, err)
				switch {
				case err == nil && cc != nil:
					connCacheSize.Add(1)
					if !fun.BlockingSend(connCache).Check(ctx, &connCacheItem{conn: cc}) {
						return
					}

					// if _, err = conf.handleDialError(ec.Add, connCache.WaitPushBack(ctx, )); err != nil {
					// 	return
					// }
					continue LOOP
				case err == nil && cc == nil:
					continue LOOP
				case isFinal:
					return
				case !isFinal:
					continue LOOP
				}
			}

			timer.Reset(conf.MinDialRetryDelay +
				intish.Max(0, time.Duration(
					rand.Int63n(int64(conf.MaxDialRetryDelay)))-conf.MinDialRetryDelay,
				),
			)

			if connCacheSize.Load() >= conf.IdleConns {
				select {
				case <-ctx.Done():
					return
				case <-timer.C:
				}
			}
		}
	}

	return func(ctx context.Context, iter *fun.Iterator[MetricPublisher]) error {
		return iter.ProcessParallel(
			func(ctx context.Context, pub MetricPublisher) (err error) {
				var conn *connCacheItem
				defer func() {
					if conn == nil || ers.ContextExpired(err) {
						return
					}
					if err == nil {
						ec.Add(fun.BlockingSend(connCache).Write(ctx, conn))
						connCacheSize.Add(1)
						return
					}
					err = ers.Join(err, ft.IgnoreFirst(conf.handleMessageError(ec.Add, conn.conn.Close())))
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

					if conn == nil {
						conn, err = fun.BlockingReceive(connCache).Read(ctx)
						connCacheSize.Add(-1)
						if err != nil {
							return err
						}
					}

					err = pub(conn, conf.Renderer)
					isFinal, err = conf.handleMessageError(ec.Add, err)
					if err == nil {
						return nil
					}
					if isFinal {
						return err
					}

					_ = conn.conn.Close()
					conn = nil

					timer.Reset(conf.MinMessageRetryDelay +
						intish.Max(0, time.Duration(
							rand.Int63n(int64(conf.MaxMessageRetryDelay)))-conf.MinMessageRetryDelay,
						),
					)
				}

				return
			},
			fun.WorkerGroupConfWithErrorCollector(ec),
			fun.WorkerGroupConfNumWorkers(conf.MessageWorkers),
			conf.MessageErrorHandling.poolErrorOptions(),
		).PreHook(dialOperation.Go().Once()).Run(ctx)
	}, nil
}
