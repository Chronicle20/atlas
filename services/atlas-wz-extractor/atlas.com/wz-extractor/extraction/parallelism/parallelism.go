// Package parallelism centralizes how worker pools across the wz-extractor
// service derive their size from operator configuration. The outer file-level
// pool, the inner map-render pool, the character-parts pool, the equipment-
// icon pool, and the Kafka consumer's MaxInFlight all call FromEnv so a
// single env var bounds memory and concurrency in lockstep.
//
// This lives in its own leaf package because both extraction/ and image/
// need it, and extraction/ already imports image/ — putting the helper here
// avoids an import cycle without duplicating the parsing logic.
package parallelism

import (
	"os"
	"runtime"
	"strconv"

	"github.com/sirupsen/logrus"
)

// EnvVar is the operator-facing name. Documented in
// deploy/k8s/overlays/pr/patches/wz-extractor-pr.yaml.
const EnvVar = "WZ_EXTRACT_PARALLELISM"

// FromEnv reads EnvVar and returns its integer value. If unset, empty, or
// not a positive integer, falls back to runtime.NumCPU() and (for malformed
// values) logs a warning so the operator notices the typo. Always returns
// at least 1.
func FromEnv(l logrus.FieldLogger) int {
	v := os.Getenv(EnvVar)
	if v == "" {
		return floor(runtime.NumCPU())
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		l.WithField("value", v).Warnf("invalid %s; using runtime.NumCPU()", EnvVar)
		return floor(runtime.NumCPU())
	}
	return n
}

func floor(n int) int {
	if n < 1 {
		return 1
	}
	return n
}
