package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strconv"
	"strings"

	"github.com/seyyedaghaei/throcat/internal/limit"
	"github.com/spf13/pflag"
)

func main() {
	listen := pflag.StringP("listen", "l", "", "Listen address")
	upstream := pflag.StringP("upstream", "u", "", "Upstream address")
	speed := pflag.StringP("speed", "s", "", "Speed in KB/s: fixed (e.g. 50), range (e.g. 30-60), or 0 / no-limit")
	interval := pflag.StringP("interval", "i", "", "When speed is range: interval in seconds to pick new rate (e.g. 5 or 3-7)")
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
	defer ln.Close()
	log.Printf("listening on %s", *listen)

	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("accept: %v", err)
			continue
		}
		go handleConn(conn, *upstream, speedCfg)
	}
}

type speedConfig struct {
	bytesPerSec float64 // 0 = no limit
}

func handleConn(client net.Conn, upstream string, cfg speedConfig) {
	defer client.Close()

	remote, err := net.Dial("tcp", upstream)
	if err != nil {
		log.Printf("dial %s: %v", upstream, err)
		return
	}
	defer remote.Close()

	if cfg.bytesPerSec <= 0 {
		go copyBytes(remote, client)
		copyBytes(client, remote)
		return
	}
	clientLim := limit.Reader(client, cfg.bytesPerSec)
	remoteLim := limit.Reader(remote, cfg.bytesPerSec)
	go copyBytesFromReader(remote, clientLim)
	copyBytesFromReader(client, remoteLim)
}

func copyBytesFromReader(dst net.Conn, src io.Reader) (int64, error) {
	return io.Copy(dst, src)
}

func copyBytes(dst, src net.Conn) (int64, error) {
	return io.Copy(dst, src)
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
	// Range: min-max (handled in next step)
	if strings.Contains(speed, "-") {
		return speedConfig{}, fmt.Errorf("range speed requires -i/--interval (not yet implemented)")
	}
	return speedConfig{}, fmt.Errorf("invalid speed %q", speed)
}
