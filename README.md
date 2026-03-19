# throcat

**throcat** is a rate-limiting TCP proxy (throttle + cat). It listens on an address, forwards TCP to an upstream, and can rate-limit traffic in both directions. Use it to simulate constrained or variable links (e.g. for benchmarking). With rate limit disabled it acts as a simple TCP relay (socat-style).

## Build

```bash
go build -o throcat ./cmd/throcat
```

Or use the Makefile: `make build`, `make test`, `make install`, `make lint`.

## Install

```bash
go install github.com/seyyedaghaei/throcat/cmd/throcat@latest
```

See [Releases](https://github.com/seyyedaghaei/throcat/releases) for pre-built binaries.

## Usage

All of `-l`/`--listen`, `-u`/`--upstream`, and `-s`/`--speed` are required. There are no defaults.

| Flag | Short | Description |
|------|--------|-------------|
| `--listen` | `-l` | Listen address (e.g. `127.0.0.1:10001`) |
| `--upstream` | `-u` | Upstream address to forward to (e.g. `127.0.0.1:1393`) |
| `--speed` | `-s` | Speed in KB/s: fixed (e.g. `50`), range (e.g. `30-60`), or `0` / `no-limit` for plain relay |
| `--interval` | `-i` | When speed is a range: seconds between rate changes (e.g. `5` or `3-7`). Omit to change rate often so speed varies constantly. |
| `--quiet` | `-q` | Do not log listen address |
| `--verbose` | `-v` | Log each connection open and close |
| `--timeout` | `-t` | Idle connection timeout (e.g. `30s`, `5m`); 0 = no timeout |
| `--loss` | `-p` | Loss percentage of forwarded bytes (0-100); 0 disables |
| `--latency` | `-L` | Base one-way latency (e.g. `100ms`); 0 disables |
| `--jitter` | `-J` | Additional random latency up to (e.g. `50ms`); 0 disables |
| `--version` | `-V` | Print version and exit |
| `--json` | `-j` | Log in JSON (one object per line) for scripting/monitoring |

With `-j`/`--json`, each log line is a JSON object with `time` (RFC3339), `event`, and event-specific fields (`addr`, `remote`, `direction`, `upstream`, `error`), for easy parsing by scripts or log aggregators.

### Examples

Plain TCP relay (no rate limit):

```bash
./throcat -l 127.0.0.1:10001 -u 127.0.0.1:1393 -s 0
```

Fixed rate 50 KB/s:

```bash
./throcat -l 127.0.0.1:10001 -u 127.0.0.1:1393 -s 50
```

Variable rate between 30 and 60 KB/s, new rate every 5 seconds:

```bash
./throcat -l 127.0.0.1:10001 -u 127.0.0.1:1393 -s 30-60 -i 5
```

Variable rate with random interval between 3 and 7 seconds:

```bash
./throcat -l 127.0.0.1:10001 -u 127.0.0.1:1393 -s 30-60 -i 3-7
```

Fixed rate with latency and jitter:

```bash
./throcat -l 127.0.0.1:10001 -u 127.0.0.1:1393 -s 50 -L 100ms -J 50ms
```

Fixed rate with loss, latency, and jitter:

```bash
./throcat -l 127.0.0.1:10001 -u 127.0.0.1:1393 -s 50 -p 2.5 -L 100ms -J 50ms
```

### Benchmarking (e.g. iperf through throcat)

Run the service (e.g. iperf server) on port 1393. Start throcat with a fixed 50 KB/s limit, then point the iperf client at the proxy:

```bash
# Terminal 1: iperf server on 1393
iperf3 -s -p 1393

# Terminal 2: proxy at 50 KB/s
./throcat -l 127.0.0.1:10001 -u 127.0.0.1:1393 -s 50

# Terminal 3: iperf client through proxy
iperf3 -c 127.0.0.1 -p 10001
```

Throughput should be capped at about 50 KB/s.

## How it works

- Listens for TCP on `-l`, accepts connections, and dials `-u` for each.
- Data is copied both ways. When `-s` is not `0`/`no-limit`, both directions are rate-limited.
- For a fixed speed, the same limit (KB/s) is applied in both directions.
- For a range speed, a new rate is chosen at random in the range. With `-i`, that happens every N seconds (fixed or range). Without `-i`, the rate is updated very often (every ~0.1–0.3s) so the effective speed keeps varying.

## Caveats

- Rate limiting is **per connection**. Multiple clients each get the configured limit; total throughput is (limit × number of connections).

## Testing

```bash
go test ./...
```

## License

MIT. See [LICENSE](LICENSE).
