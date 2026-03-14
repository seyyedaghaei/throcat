package limit

import (
	"context"
	"io"
	"math"

	"golang.org/x/time/rate"
)

// Reader wraps an io.Reader and rate-limits reads to approximately limitBytesPerSec bytes per second.
func Reader(r io.Reader, limitBytesPerSec float64) io.Reader {
	if limitBytesPerSec <= 0 {
		return r
	}
	return NewVariableLimiter(limitBytesPerSec).Reader(r)
}

// VariableLimiter is a rate limiter whose limit can be updated at runtime.
type VariableLimiter struct {
	lim *rate.Limiter
}

// NewVariableLimiter creates a limiter with the given initial bytes per second.
func NewVariableLimiter(bytesPerSec float64) *VariableLimiter {
	burst := burstSize(bytesPerSec)
	lim := rate.NewLimiter(rate.Limit(bytesPerSec), burst)
	return &VariableLimiter{lim: lim}
}

// SetLimit updates the rate to the given bytes per second.
func (v *VariableLimiter) SetLimit(bytesPerSec float64) {
	v.lim.SetLimit(rate.Limit(bytesPerSec))
	v.lim.SetBurst(burstSize(bytesPerSec))
}

// Reader returns an io.Reader that rate-limits reads using this limiter.
func (v *VariableLimiter) Reader(r io.Reader) io.Reader {
	return &limitedReader{r: r, lim: v.lim}
}

func burstSize(bytesPerSec float64) int {
	burst := 64 * 1024
	if int(bytesPerSec)*2 < burst {
		burst = int(bytesPerSec) * 2
	}
	if burst < 1024 {
		burst = 1024
	}
	return int(math.Min(float64(burst), 64*1024))
}

type limitedReader struct {
	r   io.Reader
	lim *rate.Limiter
}

func (l *limitedReader) Read(p []byte) (n int, err error) {
	n, err = l.r.Read(p)
	if n > 0 {
		if err := l.lim.WaitN(context.Background(), n); err != nil {
			return n, err
		}
	}
	return n, err
}
