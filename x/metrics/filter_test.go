package metrics

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/tychoish/birch"
	"github.com/tychoish/birch/x/ftdc/util"
	"github.com/tychoish/grip/send"
)

func init() {
	util.RegisterGlobalMarshaler(func(in any) ([]byte, error) {
		return birch.DC.Interface(in).MarshalBSON()
	})
}
func TestCollectOptions(t *testing.T) {
	t.Run("DefaultValid", func(t *testing.T) {
		dco := DefaultCollectionOptions()
		if err := dco.Validate(); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("ZeroValueInvalid", func(t *testing.T) {
		dco := CollectOptions{}
		if err := dco.Validate(); err == nil {
			t.Fatal("zero object should not validate")
		}
	})
}

func TestSchemaComposer(t *testing.T) {
	sc := []any{
		CollectSystemInfo(),
		CollectGoStatsTotals(),
		CollectGoStatsRates(),
		CollectGoStatsDeltas(),
		CollectProcessInfoSelf(),
	}
	for _, m := range CollectAllProcesses() {
		sc = append(sc, m)
	}

	for idx, m := range sc {
		mtype := strings.Split(fmt.Sprintf("%T", m), ".")[1]
		t.Run(fmt.Sprint(mtype, "/", idx), func(t *testing.T) {
			if _, ok := m.(SchemaComposer); !ok {
				t.Fatal("should be a schema composer")
			}
		})
	}
}

type bufCloser struct {
	mtx sync.Mutex
	bytes.Buffer
	isClosed bool
}

func (b *bufCloser) withLock() func() { b.mtx.Lock(); return b.mtx.Unlock }

func (b *bufCloser) Write(in []byte) (int, error) { defer b.withLock()(); return b.Buffer.Write(in) }
func (b *bufCloser) Close() error                 { defer b.withLock()(); b.isClosed = true; return nil }

func TestFilter(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	t.Run("CollectionBasicEndToEnd", func(t *testing.T) {
		buf := &bufCloser{}
		opts := DefaultCollectionOptions()
		opts.FlushInterval = 10 * time.Millisecond
		opts.SampleCount = 10
		opts.BlockCount = 10
		opts.CaptureStructured = false
		opts.WriterConstructor = func(f string) (io.WriteCloser, error) {
			return buf, nil
		}

		sender := send.MakeInternal()
		filter := NewFilter(ctx, sender, opts)
		for i := 0; i < 20; i++ {
			filter.Send(CollectGoStatsTotals())
		}

		if n := sender.Len(); n != 20 {
			t.Fatal("there should be one message:", n)
		}
		impl := filter.(*metricsFilterImpl)
		if len(impl.collectors) != 1 {
			t.Fatal("should have one collector")
		}

		if buf.Buffer.Len() == 0 {
			t.Fatal("buffer should have content")
		}
		if err := filter.Close(); err != nil {
			t.Fatal(err)
		}
		if !buf.isClosed {
			t.Fatal("buffer should be closed")
		}
	})
	t.Run("MultiStreamCollector", func(t *testing.T) {
		mtx := &sync.Mutex{}
		bufs := map[string]*bufCloser{}

		opts := DefaultCollectionOptions()
		opts.FlushInterval = 100 * time.Millisecond
		opts.SampleCount = 10
		opts.BlockCount = 10
		opts.CaptureStructured = false
		opts.WriterConstructor = func(f string) (io.WriteCloser, error) {
			mtx.Lock()
			defer mtx.Unlock()
			out := bufs[f]
			if out != nil {
				return out, nil
			}

			out = &bufCloser{}
			bufs[f] = out
			return out, nil
		}

		sender := send.MakeInternal()
		filter := NewFilter(ctx, sender, opts)
		for i := 0; i < 20; i++ {
			filter.Send(CollectGoStatsTotals())
			filter.Send(CollectProcessInfoSelf())
			filter.Send(CollectSystemInfo())
			time.Sleep(5 * time.Millisecond)
		}

		if n := sender.Len(); n != 60 {
			t.Fatal("there should be one message:", n)
		}
		impl := filter.(*metricsFilterImpl)
		if len(impl.collectors) != 3 {
			t.Fatal("should have three collectors")
		}
		if err := filter.Close(); err != nil {
			t.Fatal(err)
		}
		if len(bufs) < 3 {
			t.Fatal("there should be more than 3 writers", len(bufs))
		}
		seenData := 0
		for n, b := range bufs {
			if !b.isClosed {
				t.Errorf("buffer %q failed to close", n)
			}
			if b.Len() > 0 {
				seenData++
			}
		}
		if seenData < len(bufs) {
			t.Error("not all writers saw data", len(bufs), seenData)
		}
	})
	t.Run("Rotation", func(t *testing.T) {
		mtx := &sync.Mutex{}
		bufs := map[string]*bufCloser{}

		opts := DefaultCollectionOptions()
		opts.FlushInterval = 10 * time.Millisecond
		opts.SampleCount = 10
		opts.BlockCount = 10
		opts.CaptureStructured = false
		opts.OutputFilePrefix = t.Name()
		opts.WriterConstructor = func(f string) (io.WriteCloser, error) {
			mtx.Lock()
			defer mtx.Unlock()
			out := bufs[f]
			if out != nil {
				return out, nil
			}

			out = &bufCloser{}
			bufs[f] = out
			return out, nil
		}

		sender := send.MakeInternal()
		filter := NewFilter(ctx, sender, opts)
		sender.SetErrorHandler(send.ErrorHandlerFromLogger(log.Default()))
		for i := 0; i < opts.SampleCount*opts.BlockCount*5; i++ {
			time.Sleep(time.Millisecond)
			filter.Send(CollectGoStatsTotals())
		}

		if err := filter.Close(); err != nil {
			t.Fatal(err)
		}
		if len(bufs) < 5 {
			t.Fatal("collectors should rotate", len(bufs))
		}
	})
	t.Run("Internals", func(t *testing.T) {
		t.Run("Constructor", func(t *testing.T) {
			opts := DefaultCollectionOptions()
			opts.WriterConstructor = func(f string) (io.WriteCloser, error) { return nil, io.EOF }

			sender := send.MakeInternal()
			filter := NewFilter(ctx, sender, opts)
			sender.SetErrorHandler(send.ErrorHandlerFromLogger(log.Default()))

			impl := filter.(*metricsFilterImpl)
			coll := impl.constructor("foo")
			if coll != nil {
				t.Fatal("should not have produced an collector")
			}
		})

	})
}
