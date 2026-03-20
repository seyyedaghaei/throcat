package profiles

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

func Load(path string) (Profile, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return Profile{}, fmt.Errorf("read profile: %w", err)
	}

	var raw RawProfile
	if err := yaml.Unmarshal(b, &raw); err != nil {
		return Profile{}, fmt.Errorf("parse profile yaml: %w", err)
	}

	var prof Profile
	prof.Listen = raw.Listen
	prof.Upstream = raw.Upstream
	prof.Speed = raw.Speed
	prof.Interval = raw.Interval
	prof.Drop = raw.Drop

	if raw.Latency != nil {
		d, err := time.ParseDuration(*raw.Latency)
		if err != nil {
			return Profile{}, fmt.Errorf("parse latency %q: %w", *raw.Latency, err)
		}
		prof.Latency = &d
	}
	if raw.Jitter != nil {
		d, err := time.ParseDuration(*raw.Jitter)
		if err != nil {
			return Profile{}, fmt.Errorf("parse jitter %q: %w", *raw.Jitter, err)
		}
		prof.Jitter = &d
	}

	return prof, nil
}
