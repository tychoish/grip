package send

import (
	"bytes"

	"github.com/tychoish/grip/message"
)

// MakeBytesBuffer creates a sender that writes data to the provided
// bytes.Buffer. The sender uses the message formatter, and resepects
// level logging.
//
// A new line is added between each message as written.
func MakeBytesBuffer(buf *bytes.Buffer) Sender { return &bufsend{buffer: buf} }

type bufsend struct {
	Base
	buffer *bytes.Buffer
}

func (b *bufsend) Send(m message.Composer) {
	if ShouldLog(b, m) {
		if line, err := b.Format(m); b.HandleErrorOK(WrapError(err, m)) {
			b.buffer.WriteString(line)
			b.buffer.WriteByte('\n')
		}
	}
}
