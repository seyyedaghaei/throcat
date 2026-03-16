package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/seyyedaghaei/throcat/internal/limit"
	"github.com/spf13/pflag"
)

func main() {
	listen := pflag.StringP("listen", "l", "", "Listen address")
	upstream := pflag.StringP("upstream", "u", "", "Upstream address")
	speed := pflag.StringP("speed", "s", "", "Speed in KB/s: fixed (e.g. 50), range (e.g. 30-60), or 0 / no-limit for plain relay")
	interval := pflag.StringP("interval", "i", "", "When speed is range: seconds between rate changes (e.g. 5 or 3-7); omit to change rate often so speed varies constantly")
	quiet := pflag.BoolP("quiet", "q", false, "Do not log listen address")
	verbose := pflag.BoolP("verbose", "v", false, "Log each connection open and close")
	timeout := pflag.DurationP("timeout", "t", 0, "Idle connection timeout (e.g. 30s, 5m); 0 = no timeout")
	pflag.Parse()

	if *listen == "" || *upstream == "" {
		fmt.Fprintln(os.Stderr, "must set -l/--listen and -u/--upstream")
		pflag.Usage()
		os.Exit(1)
	}
	if *speed == "" {
		fmt.Fprintln(os.Stderr, "must set -s/--speed")
		pflag.Usage()
		os.Exit(1)
	}

	speedCfg, err := parseSpeed(*speed, *interval)
	if err != nil {
		fmt.Fprintf(os.Stderr, "speed: %v\n", err)
		os.Exit(1)
	}

	ln, err := net.Listen("tcp", *listen)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
	defer func() { _ = ln.Close() }()
	if !*quiet {
		log.Printf("listening on %s", *listen)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Printf("accept: %v", err)
			continue
		}
		go handleConn(conn, *upstream, speedCfg, *verbose, *timeout)
	}
}

type speedConfig struct {
	bytesPerSec float64 // 0 = no limit (when !isRange)
	isRange     bool
	minKB       float64
	maxKB       float64
	intervalMin float64 // seconds
	intervalMax float64 // seconds
}

func handleConn(client net.Conn, upstream string, cfg speedConfig, verbose bool, idleTimeout time.Duration) {
	defer func() { _ = client.Close() }()
	if verbose {
		log.Printf("connection from %s", client.RemoteAddr())
		defer func() { log.Printf("connection closed %s", client.RemoteAddr()) }()
	}

	remote, err := net.Dial("tcp", upstream)
	if err != nil {
		log.Printf("dial %s: %v", upstream, err)
		return
	}
	defer func() { _ = remote.Close() }()

	if idleTimeout > 0 {
		client = &deadlineConn{Conn: client, timeout: idleTimeout}
		remote = &deadlineConn{Conn: remote, timeout: idleTimeout}
	}

	if !cfg.isRange && cfg.bytesPerSec <= 0 {
		go func() { _, _ = copyBytes(remote, client) }()
		_, _ = copyBytes(client, remote)
		return
	}
	if cfg.isRange {
		handleConnRateLimitedRange(client, remote, cfg)
		return
	}
	clientLim := limit.Reader(client, cfg.bytesPerSec)
	remoteLim := limit.Reader(remote, cfg.bytesPerSec)
	go func() { _, _ = copyBytesFromReader(remote, clientLim) }()
	_, _ = copyBytesFromReader(client, remoteLim)
}

func handleConnRateLimitedRange(client, remote net.Conn, cfg speedConfig) {
	initialKB := (cfg.minKB + cfg.maxKB) / 2
	initialBps := initialKB * 1024
	lim1 := limit.NewVariableLimiter(initialBps)
	lim2 := limit.NewVariableLimiter(initialBps)
	clientLim := lim1.Reader(client)
	remoteLim := lim2.Reader(remote)

	ctx, stop := context.WithCancel(context.Background())
	defer stop()
	go func() {
		for {
			intervalSec := cfg.intervalMin
			if cfg.intervalMax > cfg.intervalMin {
				intervalSec = cfg.intervalMin + rand.Float64()*(cfg.intervalMax-cfg.intervalMin)
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Duration(intervalSec * float64(time.Second))):
				kb := cfg.minKB + rand.Float64()*(cfg.maxKB-cfg.minKB)
				bps := kb * 1024
				lim1.SetLimit(bps)
				lim2.SetLimit(bps)
			}
		}
	}()

	go func() { _, _ = copyBytesFromReader(remote, clientLim) }()
	_, _ = copyBytesFromReader(client, remoteLim)
}

func copyBytesFromReader(dst net.Conn, src io.Reader) (int64, error) {
	return io.Copy(dst, src)
}

func copyBytes(dst, src net.Conn) (int64, error) {
	return io.Copy(dst, src)
}

// deadlineConn sets read/write deadlines before each Read/Write for idle timeout.
type deadlineConn struct {
	net.Conn
	timeout time.Duration
}

func (c *deadlineConn) Read(p []byte) (n int, err error) {
	_ = c.SetReadDeadline(time.Now().Add(c.timeout))
	return c.Conn.Read(p)
}

func (c *deadlineConn) Write(p []byte) (n int, err error) {
	_ = c.SetWriteDeadline(time.Now().Add(c.timeout))
	return c.Conn.Write(p)
}

func parseSpeed(speed, interval string) (speedConfig, error) {
	speed = strings.TrimSpace(strings.ToLower(speed))
	if speed == "0" || speed == "no-limit" {
		return speedConfig{}, nil
	}
	// Fixed: single number (KB/s)
	f, err := strconv.ParseFloat(speed, 64)
	if err == nil && f > 0 {
		return speedConfig{bytesPerSec: f * 1024}, nil
	}
	// Range: min-max (e.g. 30-60)
	if strings.Contains(speed, "-") {
		parts := strings.SplitN(speed, "-", 2)
		if len(parts) != 2 {
			return speedConfig{}, fmt.Errorf("invalid speed range %q", speed)
		}
		minKB, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		maxKB, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err1 != nil || err2 != nil || minKB <= 0 || maxKB < minKB {
			return speedConfig{}, fmt.Errorf("invalid speed range %q", speed)
		}
		intervalMin, intervalMax := 0.1, 0.3 // omit -i: change rate every 0.1–0.3s so speed varies constantly
		if interval != "" {
			var err error
			intervalMin, intervalMax, err = parseInterval(interval)
			if err != nil {
				return speedConfig{}, fmt.Errorf("interval: %w", err)
			}
		}
		return speedConfig{
			isRange:     true,
			minKB:       minKB,
			maxKB:       maxKB,
			intervalMin: intervalMin,
			intervalMax: intervalMax,
		}, nil
	}
	return speedConfig{}, fmt.Errorf("invalid speed %q", speed)
}

func parseInterval(s string) (minSec, maxSec float64, err error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, 0, fmt.Errorf("empty interval")
	}
	if strings.Contains(s, "-") {
		parts := strings.SplitN(s, "-", 2)
		if len(parts) != 2 {
			return 0, 0, fmt.Errorf("invalid interval range %q", s)
		}
		minSec, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
		maxSec, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
		if err1 != nil || err2 != nil || minSec <= 0 || maxSec < minSec {
			return 0, 0, fmt.Errorf("invalid interval range %q", s)
		}
		return minSec, maxSec, nil
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil || f <= 0 {
		return 0, 0, fmt.Errorf("invalid interval %q", s)
	}
	return f, f, nil
}
