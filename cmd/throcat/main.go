package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/seyyedaghaei/throcat/internal/logx"
	"github.com/seyyedaghaei/throcat/internal/proxy"
	"github.com/spf13/pflag"
)

func main() {
	args := os.Args[1:]
	cmd := "relay"
	if len(args) > 0 {
		switch args[0] {
		case "relay", "server", "client":
			cmd = args[0]
			args = args[1:]
		}
	}

	switch cmd {
	case "relay":
		runRelay(args)
	case "server":
		fmt.Fprintln(os.Stderr, "server: not implemented")
		os.Exit(2)
	case "client":
		fmt.Fprintln(os.Stderr, "client: not implemented")
		os.Exit(2)
	default:
		fmt.Fprintln(os.Stderr, "unknown command")
		os.Exit(2)
	}
}

func runRelay(args []string) {
	fs := pflag.NewFlagSet("relay", pflag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	listen := fs.StringP("listen", "l", "", "Listen address")
	upstream := fs.StringP("upstream", "u", "", "Upstream address")
	speed := fs.StringP("speed", "s", "", "Speed in KB/s: fixed (e.g. 50), range (e.g. 30-60), or 0 / no-limit for plain relay")
	interval := fs.StringP("interval", "i", "", "When speed is range: seconds between rate changes (e.g. 5 or 3-7); omit to change rate often so speed varies constantly")
	quiet := fs.BoolP("quiet", "q", false, "Do not log listen address")
	verbose := fs.BoolP("verbose", "v", false, "Log each connection open and close")
	timeout := fs.DurationP("timeout", "t", 0, "Idle connection timeout (e.g. 30s, 5m); 0 = no timeout")
	jsonLog := fs.BoolP("json", "j", false, "Log in JSON format for scripting/monitoring")

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	if *listen == "" || *upstream == "" {
		fmt.Fprintln(os.Stderr, "must set -l/--listen and -u/--upstream")
		fs.Usage()
		os.Exit(1)
	}
	if *speed == "" {
		fmt.Fprintln(os.Stderr, "must set -s/--speed")
		fs.Usage()
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

	logger := logx.Logger{JSON: *jsonLog}
	if !*quiet {
		logger.Event(map[string]interface{}{"event": "listen", "addr": *listen})
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg := proxy.RelayConfig{
		Upstream:    *upstream,
		SpeedBytes:  speedCfg.bytesPerSec,
		Verbose:     *verbose,
		IdleTimeout: *timeout,
		Log:         logger,
	}
	if err := proxy.ServeRelay(ctx, ln, cfg); err != nil {
		log.Fatalf("relay: %v", err)
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
