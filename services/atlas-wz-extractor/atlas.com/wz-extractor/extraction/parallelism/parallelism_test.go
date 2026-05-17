package parallelism

import (
	"runtime"
	"testing"

	"github.com/sirupsen/logrus"
)

func TestFromEnv(t *testing.T) {
	l := logrus.New()
	cpus := runtime.NumCPU()
	if cpus < 1 {
		cpus = 1
	}

	tests := []struct {
		name string
		env  string
		set  bool
		want int
	}{
		{"unset falls back to NumCPU", "", false, cpus},
		{"empty falls back to NumCPU", "", true, cpus},
		{"positive integer wins", "3", true, 3},
		{"one is honored, not floored away", "1", true, 1},
		{"non-numeric falls back to NumCPU", "abc", true, cpus},
		{"zero falls back to NumCPU", "0", true, cpus},
		{"negative falls back to NumCPU", "-2", true, cpus},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.set {
				t.Setenv(EnvVar, tc.env)
			} else {
				// t.Setenv is the only API that auto-restores; for "unset"
				// we just rely on the Go test process not having it set.
				// In CI this is the common case; locally with the var
				// already set, t.Setenv("") would still set it to "".
				t.Setenv(EnvVar, "")
			}
			if got := FromEnv(l); got != tc.want {
				t.Fatalf("FromEnv = %d, want %d", got, tc.want)
			}
		})
	}
}
