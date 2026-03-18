package netem

import "time"

type Config struct {
	Bandwidth Bandwidth
	Latency   Latency
	Loss      Loss
}

type Bandwidth struct {
	Enabled bool
	KBps    Range
}

type Latency struct {
	Enabled bool
	Base    time.Duration
	Jitter  time.Duration
}

type Loss struct {
	Enabled bool
	Percent Range
}

type Range struct {
	Min float64
	Max float64
}

func (r Range) IsZero() bool {
	return r.Min == 0 && r.Max == 0
}

func (r Range) IsRange() bool {
	return r.Min != 0 && r.Max != 0 && r.Min != r.Max
}

