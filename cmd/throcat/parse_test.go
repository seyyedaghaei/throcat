package main

import (
	"testing"
)

func TestParseSpeed(t *testing.T) {
	tests := []struct {
		speed    string
		interval string
		wantZero bool
		wantRange bool
		wantErr  bool
	}{
		{"0", "", true, false, false},
		{"no-limit", "", true, false, false},
		{"NO-LIMIT", "", true, false, false},
		{"50", "", false, false, false},
		{" 50 ", "", false, false, false},
		{"30-60", "", false, false, true}, // needs interval
		{"30-60", "5", false, true, false},
		{"30-60", "3-7", false, true, false},
		{"invalid", "", false, false, true},
		{"", "", false, false, true},
	}
	for _, tt := range tests {
		cfg, err := parseSpeed(tt.speed, tt.interval)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseSpeed(%q, %q) wanted error", tt.speed, tt.interval)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseSpeed(%q, %q): %v", tt.speed, tt.interval, err)
			continue
		}
		if tt.wantZero && (cfg.bytesPerSec != 0 || cfg.isRange) {
			t.Errorf("parseSpeed(%q): expected no-limit config, got %+v", tt.speed, cfg)
		}
		if tt.wantRange && !cfg.isRange {
			t.Errorf("parseSpeed(%q, %q): expected range config", tt.speed, tt.interval)
		}
		if !tt.wantZero && !tt.wantRange && cfg.bytesPerSec <= 0 {
			t.Errorf("parseSpeed(%q): expected fixed speed > 0", tt.speed)
		}
	}
}

func TestParseInterval(t *testing.T) {
	tests := []struct {
		s     string
		wantA float64
		wantB float64
		wantErr bool
	}{
		{"5", 5, 5, false},
		{"3-7", 3, 7, false},
		{"1.5", 1.5, 1.5, false},
		{"", 0, 0, true},
		{"x", 0, 0, true},
		{"7-3", 0, 0, true},
	}
	for _, tt := range tests {
		a, b, err := parseInterval(tt.s)
		if tt.wantErr {
			if err == nil {
				t.Errorf("parseInterval(%q) wanted error", tt.s)
			}
			continue
		}
		if err != nil {
			t.Errorf("parseInterval(%q): %v", tt.s, err)
			continue
		}
		if a != tt.wantA || b != tt.wantB {
			t.Errorf("parseInterval(%q) = %v, %v; want %v, %v", tt.s, a, b, tt.wantA, tt.wantB)
		}
	}
}
