package graphite

import (
	"io"
	"net"
	"sync/atomic"

	"github.com/tychoish/fun/erc"
	"github.com/tychoish/fun/ers"
	"github.com/tychoish/fun/pubsub"
)

type Pool struct {
	maxConns int
	active   atomic.Int64
	cache    *pubsub.Queue[net.Conn]
	dialer   func() net.Conn
}

type poolConn struct {
	p *Pool
	c net.Conn
}

func (p poolConn) Write(in []byte) (int, error) {
	n, err := p.c.Write(in)
	if err != nil {
		p.p.active.Add(-1)
		err = erc.Join(err, p.c.Close())
		p.c = nil
	}
	return n, err
}

const ErrAlreadyClosed ers.Error = ers.Error("connection already closed")

func (p poolConn) Close() error {
	if p.c == nil {
		return ErrAlreadyClosed
	}
	return p.c.Close()
}

func (p *Pool) GetConnection() io.WriteCloser { return nil }
