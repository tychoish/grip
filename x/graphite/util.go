package graphite

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync/atomic"

	"github.com/tychoish/fun"
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
	w.Int64.Add(int64(out))
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

func (conf *CollectorBackendFileConf) RotatingFilePath() fun.Producer[string] {
	counter := &atomic.Int64{}
	tmpl := fmt.Sprintf("%%0%dd", conf.CounterPadding)
	counter.Add(-1)
	maxCounterVal := intPow(10, int64(conf.CounterPadding+1)) - 1

	return fun.BlockingProducer(func() (string, error) {
		var path string

		for i := counter.Add(1); i < maxCounterVal; i = counter.Add(1) {
			path = filepath.Join(conf.Directory,
				conf.FilePrefix,
				fmt.Sprintf(tmpl, i),
			)
			if _, err := os.Stat(path); os.IsNotExist(err) {
				return path, nil
			}
		}
		return "", fmt.Errorf("insufficient padding for %d elements %s %q", counter.Load(), conf.FilePrefix, path)
	})

}

func (conf *CollectorBackendFileConf) RotatingFileProducer() fun.Producer[io.WriteCloser] {
	getNextFileName := conf.RotatingFilePath().Block

	return fun.Producer[io.WriteCloser](func(ctx context.Context) (io.WriteCloser, error) {
		if err := os.MkdirAll(conf.Directory, 0755); err != nil {
			return nil, err
		}

		fn, err := getNextFileName()
		if err != nil {
			return nil, err
		}

		return os.Open(fn)
	})
}
