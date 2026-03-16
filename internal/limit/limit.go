package limit

import (
	"context"
	"io"

	"golang.org/x/time/rate"
)

// Reader wraps an io.Reader and rate-limits reads to approximately limitBytesPerSec bytes per second.
func Reader(r io.Reader, limitBytesPerSec float64) io.Reader {
	if limitBytesPerSec <= 0 {
		return r
	}
	burst := 64 * 1024
	if int(limitBytesPerSec)*2 < burst {
		burst = int(limitBytesPerSec) * 2
	}
	if burst < 1024 {
		burst = 1024
	}
	lim := rate.NewLimiter(rate.Limit(limitBytesPerSec), burst)
	return &limitedReader{r: r, lim: lim}
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
