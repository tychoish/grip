package metrics

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/tychoish/birch"
	"github.com/tychoish/birch/x/ftdc/util"
	a "github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/testt"
	"github.com/tychoish/grip"
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

type buf struct {
	closed atomic.Bool

	mtx sync.Mutex
	bytes.Buffer
}

func (b *buf) Write(in []byte) (int, error) { defer a.With(a.Lock(&b.mtx)); return b.Buffer.Write(in) }
func (b *buf) Close() error                 { b.closed.Store(true); return nil }

func TestFilter(t *testing.T) {
	t.Run("CollectionBasicEndToEnd", func(t *testing.T) {
		t.Parallel()
		ctx := testt.Context(t)
		b := &buf{}
		opts := DefaultCollectionOptions()
		opts.FlushInterval = 10 * time.Millisecond
		opts.SampleCount = 10
		opts.BlockCount = 10
		opts.CaptureStructured = false
		opts.WriterConstructor = func(f string) (io.WriteCloser, error) {
			fmt.Println(f)
			return b, nil
		}

		sender := send.MakeInternal()
		sender.SetErrorHandler(send.ErrorHandlerFromSender(grip.Sender()))
		filter := NewFilter(ctx, sender, opts)
		for i := 0; i < 10; i++ {
			filter.Send(CollectGoStatsTotals())
		}

		if n := sender.Len(); n != 10 {
			t.Error("there should be one message:", n)
		}
		time.Sleep(100 * time.Millisecond)
		impl := filter.(*metricsFilterImpl)
		if len(impl.collectors) != 1 {
			t.Error("should have one collector")
		}

		if err := filter.Close(); err != nil {
			t.Error(err)
		}
		if !b.closed.Load() {
			t.Error("buffer should be closed")
		}
		if b.Len() == 0 {
			t.Error("buffer should have content", b.Len())
		}
	})
	t.Run("MultiStreamCollector", func(t *testing.T) {
		t.Parallel()
		ctx := testt.Context(t)

		mtx := &sync.Mutex{}
		bufs := map[string]*buf{}

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

			out = &buf{}
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
			if !b.closed.Load() {
				t.Errorf("buffer %q failed to close", n)
				continue
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
		t.Parallel()
		ctx := testt.Context(t)

		mtx := &sync.Mutex{}
		bufs := map[string]*buf{}

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

			out = &buf{}
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
		t.Parallel()
		ctx := testt.Context(t)

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
