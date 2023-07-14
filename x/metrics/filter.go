package metrics

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/tychoish/birch/x/ftdc"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/grip/message"
	"github.com/tychoish/grip/send"
)

// SchemaComposer wraps a message.Composer and adds a method for describing
// a message object's raw format.
type SchemaComposer interface {
	message.Composer
	// Schema describes the format of the Raw() method for a
	// message. This is provided as an alternative to simply using
	// fmt.Sprintf("%T", msg), and makes it possible to maintain
	// schema versions.
	Schema() string
}

// CollectOptions are the settings to provide the behavior of
// the collection process.
type CollectOptions struct {
	FlushInterval     time.Duration
	SampleCount       int
	BlockCount        int
	OutputFilePrefix  string
	CaptureStructured bool

	// The WriterConstructor returns a writable object (typically
	// a file) to stream metrics too. This can be a (simple)
	// wrapper around os.Create in the common case.
	WriterConstructor func(string) (io.WriteCloser, error)
}

func (opts CollectOptions) Validate() error {
	ec := &erc.Collector{}
	erc.When(ec, opts.FlushInterval < 10*time.Millisecond, "flush interval must be greater than 10ms")
	erc.When(ec, opts.SampleCount < 10, "sample count must be greater than 10")
	erc.When(ec, opts.BlockCount < 10, "block count must be greater than 10")
	erc.When(ec, opts.OutputFilePrefix == "", "must specify prefix for output files")
	erc.When(ec, opts.WriterConstructor == nil, "must specify a constructor ")
	return ec.Resolve()
}

// DefaultCollectionOptions produces a reasonable collection
// constructor for most basic use. The WriterConstructorField
func DefaultCollectionOptions() CollectOptions {
	return CollectOptions{
		FlushInterval:     100 * time.Millisecond,
		SampleCount:       100,
		BlockCount:        100,
		CaptureStructured: true,
		OutputFilePrefix:  "ftdc",
		WriterConstructor: func(f string) (io.WriteCloser, error) { return os.Create(f) },
	}
}

type metricsFilterImpl struct {
	send.Sender
	opts        CollectOptions
	mtx         sync.Mutex
	collectors  map[string]ftdc.Collector
	constructor func(string) ftdc.Collector
	closers     []func() error
}

// NewFilter produces a sender that persists metrics collection to
// ftdc files by filtering out appropratly typed messages to the
// logger. In this model, applications can send metrics to the logger
// without needing to configure any additional logging infrastructure
// or setup. Metrics get persisted to the timeseries files of the
// filter, and also (potentially) written to the logging event
// system. All messages are passed to the underlying sender
//
// The context passed to the constructor controls the life-cycle of
// the collectors, and the close method on the sender blocks until all
// resources are released. The options control the behavior of the
// collector: how often data is flushed, how many samples are in each
// compressed block, and if the metrics collector should include all
// structured messages or only those that implement SchemaComposer.
//
// Errors encountered are propagated to the underlying sender's
// ErrorHandling facility.
//
// While this implementation is robust, and the birch/FTDC data format
// is compact and useable, it is also minimal and is perhaps a better
// model for other kinds of integration: production systems might want
// to stream metrics to some kind of central service or use a
// different persistence format, but the general model is robust.
func NewFilter(ctx context.Context, sender send.Sender, opts CollectOptions) send.Sender {
	mf := &metricsFilterImpl{
		Sender:     sender,
		collectors: map[string]ftdc.Collector{},
		opts:       opts,
	}

	mf.constructor = func(schema string) ftdc.Collector {
		coll, closer, err := mf.rotatingCollector(ctx, schema)
		if err != nil {
			mf.ErrorHandler()(err, message.MakeFields(message.Fields{
				message.FieldsMsgName: "creating constructor",
				"schema":              schema,
				"prefix":              opts.OutputFilePrefix,
			}))
			return nil
		}
		go func() {
			mf.mtx.Lock()
			defer mf.mtx.Unlock()
			mf.closers = append(mf.closers, closer)
		}()
		return coll()
	}

	return mf
}

func (mf *metricsFilterImpl) Send(msg message.Composer) {
	if mc, ok := msg.(SchemaComposer); ok {
		if coll := mf.getOrCreateFilter(mc.Schema()); coll != nil {
			if err := coll.Add(mc.Raw()); err != nil {
				mf.ErrorHandler()(err, message.MakeFields(message.Fields{
					message.FieldsMsgName: "adding metrics",
					"schema":              mc.Schema(),
					"prefix":              mf.opts.OutputFilePrefix,
				}))
			}
		}
	} else if mf.opts.CaptureStructured && msg.Structured() {
		if coll := mf.getOrCreateFilter(fmt.Sprintf("%T", msg)); coll != nil {
			m := msg.Raw()
			if err := coll.Add(m); err != nil {
				mf.ErrorHandler()(err, message.MakeFields(message.Fields{
					message.FieldsMsgName: "adding metrics",
					"schema":              fmt.Sprintf("%T", m),
					"prefix":              mf.opts.OutputFilePrefix,
				}))
			}
		}
	}

	mf.Sender.Send(msg)
}

func (mf *metricsFilterImpl) getOrCreateFilter(schema string) ftdc.Collector {
	mf.mtx.Lock()
	defer mf.mtx.Unlock()

	coll, ok := mf.collectors[schema]

	if !ok {
		coll = mf.constructor(schema)
		mf.collectors[schema] = coll
	}

	return coll
}

func (mf *metricsFilterImpl) Close() error {
	mf.mtx.Lock()
	defer mf.mtx.Unlock()

	catcher := &erc.Collector{}
	for _, fn := range mf.closers {
		erc.Check(catcher, fn)
	}
	catcher.Add(mf.Sender.Close())

	return catcher.Resolve()
}

func (mf *metricsFilterImpl) rotatingCollector(ctx context.Context, name string) (func() ftdc.Collector, func() error, error) {
	if err := mf.opts.Validate(); err != nil {
		return nil, nil, err
	}

	var outputCount int
	var blockCount int
	file, err := mf.opts.WriterConstructor(fmt.Sprintf("%s.%s.%d", mf.opts.OutputFilePrefix, name, outputCount))
	if err != nil {
		return nil, nil, fmt.Errorf("problem creating initial file: %w", err)
	}
	mtx := &sync.Mutex{}
	collector := ftdc.NewSynchronizedCollector(ftdc.NewStreamingCollector(mf.opts.SampleCount, file))
	type flushType bool
	const (
		closeCollector flushType = true
		reuseCollector flushType = false
	)

	flusher := func(op flushType) error {
		mtx.Lock()
		defer mtx.Unlock()

		if err = ftdc.FlushCollector(collector, file); err != nil {
			return err
		}
		blockCount++

		if op == reuseCollector {
			if blockCount < mf.opts.BlockCount {
				return nil
			}
		}

		if err = file.Close(); err != nil {
			return err
		}

		if op == closeCollector {
			return nil
		}

		outputCount++
		blockCount = 0
		file, err = mf.opts.WriterConstructor(fmt.Sprintf("%s.%s.%d", mf.opts.OutputFilePrefix, name, outputCount))
		if err != nil {
			return fmt.Errorf("problem creating subsequent file: %w", err)
		}

		collector = ftdc.NewSynchronizedCollector(ftdc.NewStreamingCollector(mf.opts.SampleCount, file))
		return nil
	}

	sig := make(chan struct{})
	bgFlushCtx, cancel := context.WithCancel(ctx)
	go func() {
		timer := time.NewTicker(mf.opts.FlushInterval)
		defer timer.Stop()
		defer close(sig)

		for {
			select {
			case <-bgFlushCtx.Done():
				_ = flusher(closeCollector)
				return
			case <-timer.C:
				if err := flusher(reuseCollector); err != nil {
					mf.ErrorHandler()(err, message.MakeFields(message.Fields{
						"operation": "flushing message",
						"name":      name,
						"count":     outputCount,
						"prefix":    mf.opts.OutputFilePrefix,
					}))
					return
				}
			}
		}
	}()

	return func() ftdc.Collector {
			mtx.Lock()
			defer mtx.Unlock()

			return collector
		},
		func() error {
			cancel()
			<-sig
			return nil
		}, nil
}
