package main

import (
	"fmt"
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

	_, _ = listen, upstream
	fmt.Printf("listen=%s upstream=%s speed=%s interval=%s\n", *listen, *upstream, *speed, *interval)
}
