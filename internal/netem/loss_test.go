package netem

import (
	"net"
	"testing"
	"time"
)

func TestWrapWriteLoss_zero(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	w := WrapWriteLoss(c1, DropLoss{Percent: 0})
	go func() { _, _ = w.Write([]byte("x")) }()

	_ = c2.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
	buf := make([]byte, 1)
	n, err := c2.Read(buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if n != 1 || buf[0] != 'x' {
		t.Fatalf("got n=%d buf=%q", n, buf)
	}
}

func TestWrapWriteLoss_full(t *testing.T) {
	c1, c2 := net.Pipe()
	defer c1.Close()
	defer c2.Close()

	w := WrapWriteLoss(c1, DropLoss{Percent: 100})
	writeErrCh := make(chan error, 1)
	go func() {
		_, err := w.Write([]byte("x"))
		writeErrCh <- err
	}()

	_ = c2.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	buf := make([]byte, 1)
	_, err := c2.Read(buf)
	if err == nil {
		t.Fatalf("expected read timeout/error; got nil")
	}
	if errw := <-writeErrCh; errw != nil {
		t.Fatalf("write should succeed, got %v", errw)
	}
}
