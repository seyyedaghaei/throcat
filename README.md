# throcat

**throcat** is a rate-limiting TCP proxy (throttle + cat). It listens on an address, forwards TCP to an upstream, and can rate-limit traffic in both directions. Use it to simulate constrained or variable links (e.g. for benchmarking). With rate limit disabled it acts as a simple TCP relay (socat-style).

## Build

```bash
go build -o throcat ./cmd/throcat
```

## Install:

```bash
go install github.com/seyyedaghaei/throcat/cmd/throcat@latest
```

## Usage

All of `-l`/`--listen`, `-u`/`--upstream`, and `-s`/`--speed` are required. There are no defaults.

| Flag | Short | Description |
|------|--------|-------------|
| `--listen` | `-l` | Listen address (e.g. `127.0.0.1:10001`) |
| `--upstream` | `-u` | Upstream address to forward to (e.g. `127.0.0.1:1393`) |
| `--speed` | `-s` | Speed in KB/s: fixed (e.g. `50`), range (e.g. `30-60`), or `0` / `no-limit` for plain relay |
| `--interval` | `-i` | Required when speed is a range. Interval in seconds to pick a new rate: one number (e.g. `5`) or range (e.g. `3-7`) |

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

## How it works

- Listens for TCP on `-l`, accepts connections, and dials `-u` for each.
- Data is copied both ways. When `-s` is not `0`/`no-limit`, both directions are rate-limited.
- For a fixed speed, the same limit (KB/s) is applied in both directions.
- For a range speed, a new rate is chosen at random in the given interval (in seconds). The interval itself can be fixed (e.g. `-i 5`) or a range (e.g. `-i 3-7`).

## Caveats

- Rate limiting is **per connection**. Multiple clients each get the configured limit; total throughput is (limit × number of connections).

## Testing

```bash
go test ./...
```

## License

MIT. See [LICENSE](LICENSE).
