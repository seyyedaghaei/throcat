package proxy

import (
	"context"
	"io"
	"net"
	"time"

	"github.com/seyyedaghaei/throcat/internal/limit"
	"github.com/seyyedaghaei/throcat/internal/logx"
	"github.com/seyyedaghaei/throcat/internal/netem"
)

type RelayConfig struct {
	Upstream    string
	SpeedBytes  float64
	Verbose     bool
	IdleTimeout time.Duration
	Latency     netem.Latency
	LossPercent float64
	Log         logx.Logger
}

func ServeRelay(ctx context.Context, ln net.Listener, cfg RelayConfig) error {
	defer func() { _ = ln.Close() }()

	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			cfg.Log.Event(map[string]interface{}{"event": "accept_error", "error": err.Error()})
			continue
		}
		go handleRelayConn(conn, cfg)
	}
}

func handleRelayConn(client net.Conn, cfg RelayConfig) {
	defer func() { _ = client.Close() }()
	remoteAddr := client.RemoteAddr().String()

	if cfg.Verbose {
		cfg.Log.Event(map[string]interface{}{"event": "connection", "remote": remoteAddr, "direction": "open"})
		defer func() {
			cfg.Log.Event(map[string]interface{}{"event": "connection", "remote": remoteAddr, "direction": "close"})
		}()
	}

	remote, err := net.Dial("tcp", cfg.Upstream)
	if err != nil {
		cfg.Log.Event(map[string]interface{}{"event": "dial_error", "upstream": cfg.Upstream, "error": err.Error()})
		return
	}
	defer func() { _ = remote.Close() }()

	if cfg.IdleTimeout > 0 {
		client = &deadlineConn{Conn: client, timeout: cfg.IdleTimeout}
		remote = &deadlineConn{Conn: remote, timeout: cfg.IdleTimeout}
	}
	if cfg.LossPercent > 0 {
		client = netem.WrapWriteLoss(client, netem.DropLoss{Percent: cfg.LossPercent})
		remote = netem.WrapWriteLoss(remote, netem.DropLoss{Percent: cfg.LossPercent})
	}
	if cfg.Latency.Enabled {
		delay := netem.Delay{Base: cfg.Latency.Base, Jitter: cfg.Latency.Jitter}
		client = netem.WrapWriteDelay(client, delay)
		remote = netem.WrapWriteDelay(remote, delay)
	}

	if cfg.SpeedBytes <= 0 {
		go func() { _, _ = io.Copy(remote, client) }()
		_, _ = io.Copy(client, remote)
		return
	}

	clientLim := limit.Reader(client, cfg.SpeedBytes)
	remoteLim := limit.Reader(remote, cfg.SpeedBytes)
	go func() { _, _ = io.Copy(remote, clientLim) }()
	_, _ = io.Copy(client, remoteLim)
}

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
