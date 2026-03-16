package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"

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
		go handleConn(conn, *upstream, *speed, *interval)
	}
}

func handleConn(client net.Conn, upstream, speed, interval string) {
	defer client.Close()
	_ = speed
	_ = interval

	remote, err := net.Dial("tcp", upstream)
	if err != nil {
		log.Printf("dial %s: %v", upstream, err)
		return
	}
	defer remote.Close()

	go copyBytes(remote, client)
	copyBytes(client, remote)
}

func copyBytes(dst, src net.Conn) (int64, error) {
	return io.Copy(dst, src)
}
