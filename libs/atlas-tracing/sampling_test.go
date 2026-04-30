package tracing

import (
	"testing"

	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
)

func TestParseSamplingRatio(t *testing.T) {
	tests := []struct {
		name      string
		envValue  string
		envSet    bool
		want      float64
		wantWarn  bool
	}{
		{name: "unset defaults to 1.0", envSet: false, want: 1.0, wantWarn: false},
		{name: "valid 1.0", envSet: true, envValue: "1.0", want: 1.0, wantWarn: false},
		{name: "valid 0.5", envSet: true, envValue: "0.5", want: 0.5, wantWarn: false},
		{name: "valid 0.0", envSet: true, envValue: "0.0", want: 0.0, wantWarn: false},
		{name: "empty string warns and defaults", envSet: true, envValue: "", want: 1.0, wantWarn: true},
		{name: "garbage warns and defaults", envSet: true, envValue: "abc", want: 1.0, wantWarn: true},
		{name: "above range warns and defaults", envSet: true, envValue: "1.5", want: 1.0, wantWarn: true},
		{name: "negative warns and defaults", envSet: true, envValue: "-0.1", want: 1.0, wantWarn: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv("TRACE_SAMPLING_RATIO", "")
			if tc.envSet {
				t.Setenv("TRACE_SAMPLING_RATIO", tc.envValue)
			} else {
				if err := unsetForTest(); err != nil {
					t.Fatal(err)
				}
			}

			logger, hook := logtest.NewNullLogger()
			got := parseSamplingRatio(logger)

			if got != tc.want {
				t.Errorf("parseSamplingRatio() = %v, want %v", got, tc.want)
			}

			gotWarn := false
			for _, e := range hook.AllEntries() {
				if e.Level == logrus.WarnLevel {
					gotWarn = true
					break
				}
			}
			if gotWarn != tc.wantWarn {
				t.Errorf("warn emitted = %v, want %v", gotWarn, tc.wantWarn)
			}
		})
	}
}

func unsetForTest() error {
	return osUnsetenv("TRACE_SAMPLING_RATIO")
}
