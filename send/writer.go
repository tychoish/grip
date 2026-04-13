package send

import (
	"bufio"
	"bytes"
	"io"
	"sync"

	"github.com/tychoish/fun/adt"
	"github.com/tychoish/fun/erc"
	"github.com/tychoish/grip/message"
)

var bufpool = adt.MakeBytesBufferPool(128)

type iowritersender struct {
	mtx sync.Mutex
	iwr *bufio.Writer
	Base
}

func newWriter(wr io.Writer) *iowritersender          { return &iowritersender{iwr: bufio.NewWriter(wr)} }
func (s *iowritersender) Write(in []byte) (err error) { _, err = s.iwr.Write(in); return }

func (s *iowritersender) Send(m message.Composer) {
	if ShouldLog(s, m) {
		if out, err := s.Format(m); s.HandleErrorOK(WrapError(err, m)) {
			buf := bufpool.Get()
			defer bufpool.Put(buf)

			_, err = buf.WriteString(out)

			s.mtx.Lock()
			defer s.mtx.Unlock()

			s.HandleErrorOK(erc.Join(err,
				s.Write(bytes.TrimSpace(buf.Bytes())),
				s.iwr.WriteByte('\n'),
				s.iwr.Flush(),
			))
		}
	}
}
