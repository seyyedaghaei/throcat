package main

import (
	"bufio"
	"net"
	"os/exec"
	"testing"
	"time"
)

func TestProxy_relay(t *testing.T) {
	// Build the binary (from module root)
	bin := t.TempDir() + "/throcat"
	out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput()
	if err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	// Start echo server
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	echoAddr := ln.Addr().String()
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				for {
					n, err := c.Read(buf)
					if n > 0 {
						_, _ = c.Write(buf[:n])
					}
					if err != nil {
						return
					}
				}
			}(conn)
		}
	}()

	// Start proxy on a fixed port (random might need parsing from log)
	proxyListen := "127.0.0.1:19998"
	cmd := exec.Command(bin, "-l", proxyListen, "-u", echoAddr, "-s", "0")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer cmd.Process.Kill()
	time.Sleep(200 * time.Millisecond)

	conn, err := net.Dial("tcp", proxyListen)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	msg := "hello\n"
	if _, err := conn.Write([]byte(msg)); err != nil {
		t.Fatal(err)
	}
	got, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if got != msg {
		t.Errorf("got %q, want %q", got, msg)
	}
}
