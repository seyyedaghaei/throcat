package netem

import (
	"net"
	"testing"
	"time"
)

func TestWrapWriteDelay(t *testing.T) {
	c1, c2 := net.Pipe()
	defer func() { _ = c1.Close() }()
	defer func() { _ = c2.Close() }()

	d := Delay{Base: 30 * time.Millisecond, Jitter: 0}
	w := WrapWriteDelay(c1, d)

	start := time.Now()
	writeErr := make(chan error, 1)
	go func() {
		_, err := w.Write([]byte("x"))
		writeErr <- err
	}()

	buf := make([]byte, 1)
	if _, err := c2.Read(buf); err != nil {
		t.Fatalf("read: %v", err)
	}
	<-writeErr

	elapsed := time.Since(start)
	if elapsed < 20*time.Millisecond {
		t.Fatalf("elapsed=%v, want >= 20ms", elapsed)
	}
}

