package tracing

import (
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
)

const (
	samplingRatioEnvVar  = "TRACE_SAMPLING_RATIO"
	defaultSamplingRatio = 1.0
)

// parseSamplingRatio reads TRACE_SAMPLING_RATIO from the environment and returns
// a value in [0.0, 1.0]. On any parse failure (missing, empty, non-numeric, out
// of range), it returns 1.0 and emits a WARN log line (except for the truly
// unset case, where 1.0 is the documented default and silence is correct).
func parseSamplingRatio(l logrus.FieldLogger) float64 {
	raw, ok := os.LookupEnv(samplingRatioEnvVar)
	if !ok {
		return defaultSamplingRatio
	}
	if raw == "" {
		l.Warnf("%s set but empty; defaulting to %.1f", samplingRatioEnvVar, defaultSamplingRatio)
		return defaultSamplingRatio
	}
	v, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		l.Warnf("%s=%q is not a valid float; defaulting to %.1f", samplingRatioEnvVar, raw, defaultSamplingRatio)
		return defaultSamplingRatio
	}
	if v < 0.0 || v > 1.0 {
		l.Warnf("%s=%v is outside [0.0, 1.0]; defaulting to %.1f", samplingRatioEnvVar, v, defaultSamplingRatio)
		return defaultSamplingRatio
	}
	return v
}
