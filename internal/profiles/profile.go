package profiles

import "time"

type RawProfile struct {
	Listen   string   `yaml:"listen" json:"listen"`
	Upstream string   `yaml:"upstream" json:"upstream"`
	Speed    *string  `yaml:"speed" json:"speed"`
	Interval *string  `yaml:"interval" json:"interval"`
	Latency  *string  `yaml:"latency" json:"latency"`
	Jitter   *string  `yaml:"jitter" json:"jitter"`
	Loss     *float64 `yaml:"loss" json:"loss"`
}

type Profile struct {
	Listen   string
	Upstream string
	Speed    *string
	Interval *string
	Latency  *time.Duration
	Jitter   *time.Duration
	Loss     *float64
}
