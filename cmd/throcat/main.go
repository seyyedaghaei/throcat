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
	"github.com/seyyedaghaei/throcat/internal/netem"
	"github.com/seyyedaghaei/throcat/internal/profiles"
	"github.com/seyyedaghaei/throcat/internal/proxy"
	"github.com/spf13/pflag"
)

func main() {
	runRelay(os.Args[1:])
}

func runRelay(args []string) {
	fs := pflag.NewFlagSet("relay", pflag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	listen := fs.StringP("listen", "l", "", "Listen address")
	upstream := fs.StringP("upstream", "u", "", "Upstream address")
	speed := fs.StringP("speed", "s", "", "Speed in KB/s: fixed (e.g. 50), range (e.g. 30-60), or 0 / no-limit for plain relay")
	interval := fs.StringP("interval", "i", "", "When speed is range: seconds between rate changes (e.g. 5 or 3-7); omit to change rate often so speed varies constantly")
	profilePath := fs.StringP("profile", "", "", "Path to network profile YAML (CLI flags override profile values)")
	quiet := fs.BoolP("quiet", "q", false, "Do not log listen address")
	verbose := fs.BoolP("verbose", "v", false, "Log each connection open and close")
	timeout := fs.DurationP("timeout", "t", 0, "Idle connection timeout (e.g. 30s, 5m); 0 = no timeout")
	loss := fs.Float64P("loss", "p", 0, "Loss percentage of forwarded bytes (0-100); 0 disables")
	latency := fs.DurationP("latency", "L", 0, "Base one-way latency (e.g. 100ms); 0 disables")
	jitter := fs.DurationP("jitter", "J", 0, "Additional random latency up to (e.g. 50ms); 0 disables")
	jsonLog := fs.BoolP("json", "j", false, "Log in JSON format for scripting/monitoring")

	if err := fs.Parse(args); err != nil {
		os.Exit(2)
	}

	if *profilePath != "" {
		p, err := profiles.Load(*profilePath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "profile: %v\n", err)
			os.Exit(1)
		}

		if !fs.Lookup("listen").Changed && p.Listen != "" {
			*listen = p.Listen
		}
		if !fs.Lookup("upstream").Changed && p.Upstream != "" {
			*upstream = p.Upstream
		}
		if !fs.Lookup("speed").Changed && p.Speed != nil {
			*speed = *p.Speed
		}
		if !fs.Lookup("interval").Changed && p.Interval != nil {
			*interval = *p.Interval
		}
		if !fs.Lookup("latency").Changed && p.Latency != nil {
			*latency = *p.Latency
		}
		if !fs.Lookup("jitter").Changed && p.Jitter != nil {
			*jitter = *p.Jitter
		}
		if !fs.Lookup("loss").Changed && p.Loss != nil {
			*loss = *p.Loss
		}
	}

	if *listen == "" || *upstream == "" {
		fmt.Fprintln(os.Stderr, "must set -l/--listen and -u/--upstream (or provide them in --profile)")
		fs.Usage()
		os.Exit(1)
	}
	if *speed == "" {
		fmt.Fprintln(os.Stderr, "must set -s/--speed (or provide it in --profile)")
		fs.Usage()
		os.Exit(1)
	}
	if *loss < 0 || *loss > 100 {
		fmt.Fprintln(os.Stderr, "loss: must be between 0 and 100")
		fs.Usage()
		os.Exit(1)
	}
	if *latency < 0 || *jitter < 0 {
		fmt.Fprintln(os.Stderr, "latency/jitter must be >= 0")
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
		LossPercent: *loss,
		Latency: netem.Latency{
			Enabled: *latency > 0 || *jitter > 0,
			Base:    *latency,
			Jitter:  *jitter,
		},
		Log: logger,
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
