package netem

import (
	"math/rand"
	"net"
)

// DropLoss drops writes at the given probability.
// It returns success to the caller even when bytes are not forwarded.
type DropLoss struct {
	Percent float64
	RNG     *rand.Rand
}

func (l DropLoss) shouldDrop() bool {
	if l.Percent <= 0 {
		return false
	}
	if l.Percent >= 100 {
		return true
	}
	r := l.RNG
	if r == nil {
		r = rand.New(rand.NewSource(1))
	}
	return r.Float64() < (l.Percent / 100.0)
}

func WrapWriteLoss(c net.Conn, d DropLoss) net.Conn {
	if d.Percent <= 0 {
		return c
	}
	return &lossConn{Conn: c, loss: d}
}

type lossConn struct {
	net.Conn
	loss DropLoss
}

func (c *lossConn) Write(p []byte) (n int, err error) {
	if c.loss.shouldDrop() {
		return len(p), nil
	}
	return c.Conn.Write(p)
}
