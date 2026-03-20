package profiles

import (
	"os"
	"testing"
	"time"
)

func TestLoad_basic(t *testing.T) {
	tmp := t.TempDir() + "/profile.yaml"
	data := []byte(`
listen: "127.0.0.1:10001"
upstream: "127.0.0.1:1393"
speed: "30-60"
interval: "5"
latency: "100ms"
jitter: "50ms"
drop: 2.5
`)
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		t.Fatal(err)
	}

	p, err := Load(tmp)
	if err != nil {
		t.Fatal(err)
	}
	if p.Listen != "127.0.0.1:10001" {
		t.Fatalf("listen=%q", p.Listen)
	}
	if p.Upstream != "127.0.0.1:1393" {
		t.Fatalf("upstream=%q", p.Upstream)
	}
	if p.Speed == nil || *p.Speed != "30-60" {
		t.Fatalf("speed=%v", p.Speed)
	}
	if p.Interval == nil || *p.Interval != "5" {
		t.Fatalf("interval=%v", p.Interval)
	}
	if p.Latency == nil || *p.Latency != 100*time.Millisecond {
		t.Fatalf("latency=%v", p.Latency)
	}
	if p.Jitter == nil || *p.Jitter != 50*time.Millisecond {
		t.Fatalf("jitter=%v", p.Jitter)
	}
	if p.Drop == nil || *p.Drop != 2.5 {
		t.Fatalf("drop=%v", p.Drop)
	}
}
