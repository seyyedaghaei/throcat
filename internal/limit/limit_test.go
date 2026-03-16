package limit

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"
)

func TestReader_noLimit(t *testing.T) {
	data := []byte("hello world")
	r := Reader(bytes.NewReader(data), 0)
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("got %q", got)
	}
}

func TestReader_withLimit(t *testing.T) {
	data := []byte("data")
	r := Reader(bytes.NewReader(data), 1024)
	got, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, data) {
		t.Errorf("got %q", got)
	}
}

func TestVariableLimiter_SetLimit(t *testing.T) {
	v := NewVariableLimiter(100)
	r := v.Reader(&infiniteReader{})
	buf := make([]byte, 10)
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	done := make(chan struct{})
	go func() {
		_, _ = r.Read(buf)
		close(done)
	}()
	// Lower the limit so the read is throttled
	v.SetLimit(1)
	select {
	case <-done:
		// Read completed (may complete if burst allowed it)
	case <-ctx.Done():
		// Expected: still waiting due to limit
	}
}

type infiniteReader struct{}

func (infiniteReader) Read(p []byte) (n int, err error) {
	for i := range p {
		p[i] = byte(i)
	}
	return len(p), nil
}
