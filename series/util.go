package series

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/tychoish/fun"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/fnx"
)

type sizeAccountingWriter struct {
	io.Writer
	*atomic.Int64
}

func newSizeAccountingWriter(base io.Writer) *sizeAccountingWriter {
	return &sizeAccountingWriter{
		Writer: base,
		Int64:  &atomic.Int64{},
	}
}

func (w *sizeAccountingWriter) Write(in []byte) (out int, err error) {
	out, err = w.Writer.Write(in)
	w.Add(int64(out))
	return
}

func (w *sizeAccountingWriter) Size() int { return int(w.Load()) }

func intPow(val, exp int64) int64 {
	fun.Invariant.IsTrue(exp > 0)
	if val == 0 {
		return 1
	}
	result := val
	for i := int64(2); i <= exp; i++ {
		result *= val
	}
	return result
}

func (conf *CollectorBackendFileConf) RotatingFilePath() fnx.Future[string] {
	counter := &atomic.Int64{}
	tmpl := fmt.Sprintf("%s%%0%dd", conf.FilePrefix, conf.CounterPadding)
	counter.Add(-1)
	maxCounterVal := intPow(10, int64(conf.CounterPadding+1)) - 1

	return func(ctx context.Context) (string, error) {
		var path string

		for i := counter.Add(1); i < maxCounterVal; i = counter.Add(1) {
			path = filepath.Join(conf.Directory, fmt.Sprintf(tmpl, i))

			if _, err := os.Stat(path); os.IsNotExist(err) {
				return path, nil
			}
		}

		return "", fmt.Errorf("insufficient padding for %d elements %s %q", counter.Load(), conf.FilePrefix, path)
	}
}

func (conf *CollectorBackendFileConf) RotatingFileProducer() fnx.Future[io.WriteCloser] {
	getNextFileName := conf.RotatingFilePath().Wait

	return fnx.Future[io.WriteCloser](func(ctx context.Context) (io.WriteCloser, error) {
		if err := os.MkdirAll(conf.Directory, 0o755); err != nil {
			return nil, err
		}

		fn, err := getNextFileName()
		if err != nil {
			return nil, err
		}

		out, err := os.Create(fn)
		if err != nil {
			return nil, ers.Wrapf(err, "os.Create<%s>", fn)
		}

		return out, nil
	})
}
