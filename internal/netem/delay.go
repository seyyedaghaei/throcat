package netem

import (
	"math/rand"
	"net"
	"time"
)

type Delay struct {
	Base   time.Duration
	Jitter time.Duration
}

func (d Delay) Enabled() bool {
	return d.Base > 0 || d.Jitter > 0
}

// WrapWriteDelay returns a net.Conn wrapper that delays each Write to simulate one-way latency.
func WrapWriteDelay(c net.Conn, d Delay) net.Conn {
	if !d.Enabled() {
		return c
	}
	return &delayConn{Conn: c, delay: d}
}

type delayConn struct {
	net.Conn
	delay Delay
}

func (c *delayConn) Write(p []byte) (n int, err error) {
	delay := c.delay.Base
	if c.delay.Jitter > 0 {
		delay += time.Duration(rand.Float64() * float64(c.delay.Jitter))
	}
	if delay > 0 {
		time.Sleep(delay)
	}
	return c.Conn.Write(p)
}

