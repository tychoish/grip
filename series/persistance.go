package series

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"iter"
	"math/rand"
	"net"
	"os"
	"time"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/fn"
	"github.com/tychoish/fun/fnx"
	"github.com/tychoish/fun/opt"
	"github.com/tychoish/fun/pubsub"
	"github.com/tychoish/fun/wpa"
	"github.com/tychoish/grip/send"
)

type MetricPublisher func(io.Writer, Renderer) error

type CollectorBackend fnx.Handler[iter.Seq[MetricPublisher]]

type Renderer struct {
	Metric    MetricValueRenderer
	Histogram MetricHistogramRenderer
}

func (cb CollectorBackend) Worker(iter iter.Seq[MetricPublisher]) fnx.Worker {
	return func(ctx context.Context) error { return cb(ctx, iter) }
}

type CollectorBakendFileOptionProvider = opt.Provider[*CollectorBackendFileConf]

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
	ec.If(conf.Megabytes < 1, ers.New("must specify at least 1mb rotation size"))
	ec.If(conf.CounterPadding < 1, ers.New("must specify at least 1 didget for counter padding"))
	stat, err := os.Stat(conf.Directory)
	ec.If(os.IsNotExist(err) || stat != nil && !stat.IsDir(), ers.New("directory must either not exist or be a directory"))
	ec.If(conf.FilePrefix == "", ers.New("must specify a prefix for data files"))
	ec.If(conf.Extension == "", ers.New("must specify at prefix for data files"))

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
	if err := opt.Join(opts...).Apply(conf); err != nil {
		return nil, err
	}
	targetSizeBytes := conf.Megabytes * 1024 * 1024
	getNextFn := conf.RotatingFileProducer()
	return func(ctx context.Context, seq iter.Seq[MetricPublisher]) (err error) {
		var file io.WriteCloser
		var saw *sizeAccountingWriter
		var buf *bufio.Writer
		var gzp *gzip.Writer
		var wr io.Writer

		ec := &erc.Collector{}

		defer func() { ec.Push(err); err = ec.Resolve() }()
		defer func() {
			if buf != nil {
				ec.Push(buf.Flush())
			}
			if gzp != nil {
				ec.Push(gzp.Close())
			}
			if file != nil {
				ec.Push(file.Close())
			}
		}()

		for op := range seq {
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
					wr = gzp
				} else {
					wr = saw
				}
			}

			if err := op(wr, conf.Renderer); err != nil {
				return err
			}

			if saw.Size() >= targetSizeBytes {
				if err := erc.Join(buf.Flush(), gzp.Close(), file.Close()); err != nil {
					return err
				}

				file = nil
				saw = nil
				buf = nil
				gzp = nil
				wr = nil
			}
		}
		return nil
	}, nil
}

func LoggerBackend(sender send.Sender, r Renderer) CollectorBackend {
	wr := send.MakeWriter(sender)
	return func(ctx context.Context, seq iter.Seq[MetricPublisher]) error {
		count := 0
		for op := range seq {
			if err := op(wr, r); err != nil {
				return erc.Join(err, wr.Close())
			}
			count++
		}
		return wr.Close()
	}
}

func PassthroughBackend(r Renderer, handler fnx.Handler[string], opts ...opt.Provider[*wpa.WorkerGroupConf]) CollectorBackend {
	pool := &adt.Pool[*bytes.Buffer]{}
	pool.SetConstructor(func() *bytes.Buffer { return &bytes.Buffer{} })
	pool.SetCleanupHook(func(buf *bytes.Buffer) *bytes.Buffer { buf.Reset(); return buf })

	return func(ctx context.Context, seq iter.Seq[MetricPublisher]) error {
		return pubsub.Convert(fnx.MakeConverter(
			func(mp MetricPublisher) string {
				buf := pool.Get()
				defer pool.Put(buf)
				erc.Invariant(mp(buf, r))
				return buf.String()
			})).
			Stream(pubsub.IteratorStream(seq)).
			Parallel(handler, opts...).
			Run(ctx)
	}
}

type CollectorBakendSocketOptionProvider = opt.Provider[*CollectorBackendSocketConf]

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
	conf.DialWorkers = max(1, conf.DialWorkers)
	conf.MessageWorkers = max(conf.DialWorkers, conf.MessageWorkers)
	conf.NumMessageRetries = max(1, conf.NumMessageRetries)
	conf.IdleConns = max(2*conf.DialWorkers, conf.IdleConns)
	conf.MinDialRetryDelay = max(100*time.Millisecond, conf.MinDialRetryDelay)
	conf.MaxDialRetryDelay = max(conf.MinDialRetryDelay, conf.MaxDialRetryDelay)
	conf.MinMessageRetryDelay = max(100*time.Millisecond, conf.MinMessageRetryDelay)
	conf.MaxMessageRetryDelay = max(conf.MinMessageRetryDelay, conf.MaxMessageRetryDelay)

	ec := &erc.Collector{}
	ec.Push(conf.DialErrorHandling.Validate())
	ec.Push(conf.MessageErrorHandling.Validate())

	ec.If(conf.Network != "tcp" && conf.Network != "udp", ers.New("network must be 'tcp' or 'udp'"))
	ec.If(conf.Renderer.Histogram == nil, ers.New("must specify histogram renderer"))
	ec.If(conf.Renderer.Metric == nil, ers.New("must specify scalar metrics renderer"))

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
	eh fn.Handler[error],
	op CollectorBackendSocketErrorOption,
	err error,
) (bool, error) {
	switch {
	case err == nil:
		return true, nil
	case ers.IsExpiredContext(err):
		return true, err
	case ers.Is(err, pubsub.ErrQueueClosed, io.EOF, ers.ErrCurrentOpAbort):
		return true, nil
	default:
		switch op {
		case CollectorBackendSocketErrorContinue:
			eh(err)
			return false, nil
		case CollectorBackendSocketErrorAbort:
			return true, err
		case CollectorBackendSocketErrorCollect:
			eh(err)
			return false, nil
		case CollectorBackendSocketErrorPanic:
			panic(err)
		}
	}
	return false, nil
}

func (conf CollectorBackendSocketConf) handleDialError(eh fn.Handler[error], err error) (bool, error) {
	return handleSocketBackedError(eh, conf.DialErrorHandling, err)
}

func (conf CollectorBackendSocketConf) handleMessageError(eh fn.Handler[error], err error) (bool, error) {
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

func (co CollectorBackendSocketErrorOption) poolErrorOptions() opt.Provider[*wpa.WorkerGroupConf] {
	return func(conf *wpa.WorkerGroupConf) error {
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
	if err := opt.Join(opts...).Apply(conf); err != nil {
		return nil, err
	}

	connCache := make(chan *connCacheItem, 2*(conf.IdleConns+conf.DialWorkers+conf.MessageWorkers))
	connCacheSize := &adt.AtomicInteger[int]{}

	ec := &erc.Collector{}
	var dialOperation fnx.Operation = func(ctx context.Context) {
		timer := time.NewTimer(0)
		defer timer.Stop()

		var isFinal bool
	LOOP:
		for {
			if connCacheSize.Load() < conf.IdleConns {
				cc, err := conf.Dialer.DialContext(ctx, conf.Network, conf.Address)
				isFinal, err = conf.handleDialError(ec.Push, err)
				switch {
				case err == nil && cc != nil:
					connCacheSize.Add(1)
					select {
					case <-ctx.Done():
						return
					case connCache <- &connCacheItem{conn: cc}:
					}
					continue LOOP
				case err == nil && cc == nil:
					continue LOOP
				case isFinal:
					return
				case !isFinal:
					continue LOOP
				}
			}

			if connCacheSize.Load() >= conf.IdleConns {
				timer.Reset(conf.MinDialRetryDelay +
					max(0, time.Duration(
						rand.Int63n(int64(conf.MaxDialRetryDelay)))-conf.MinDialRetryDelay,
					),
				)

				select {
				case <-ctx.Done():
					return
				case <-timer.C:
				}
			}
		}
	}
	dialOperation = dialOperation.Go().Once()
	return func(ctx context.Context, seq iter.Seq[MetricPublisher]) error {
		dialOperation.Run(ctx)
		return wpa.WithHandler(fnx.NewHandler(func(ctx context.Context, pub MetricPublisher) (err error) {
			var conn *connCacheItem
			defer func() {
				if conn == nil || ers.IsExpiredContext(err) {
					return
				}
				if err == nil {
					select {
					case <-ctx.Done():
						ec.Push(ctx.Err())
					case connCache <- conn:
						connCacheSize.Add(1)
					}
					return
				}
				_, err2 := conf.handleMessageError(ec.Push, conn.conn.Close())
				err = erc.Join(err, err2)
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
					select {
					case <-ctx.Done():
						return ctx.Err()
					case conn = <-connCache:
						connCacheSize.Add(-1)
						if err != nil {
							return err
						}
					}
				}
				err = pub(conn, conf.Renderer)
				isFinal, err = conf.handleMessageError(ec.Push, err)
				if err == nil {
					return nil
				}
				if isFinal {
					return err
				}

				_ = conn.conn.Close()
				conn = nil

				timer.Reset(conf.MinMessageRetryDelay +
					max(0, time.Duration(
						rand.Int63n(int64(conf.MaxMessageRetryDelay)))-conf.MinMessageRetryDelay,
					),
				)
			}

			return nil
		})).RunWithPool(
			wpa.WorkerGroupConfWithErrorCollector(ec),
			wpa.WorkerGroupConfNumWorkers(conf.MessageWorkers),
			conf.MessageErrorHandling.poolErrorOptions(),
		).For(seq).Run(ctx)
	}, nil
}
