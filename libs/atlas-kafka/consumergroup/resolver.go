// Package consumergroup resolves a service's Kafka consumer group ID.
//
// The default name is the service's historical literal (e.g. "Character Service").
// In environments where consumer-group isolation is required, the deployment
// sets KAFKA_CONSUMER_GROUP to a suffixed value such as
// "Character Service [a3f7]" and the env value is returned verbatim.
//
// Templated callers (atlas-channel, atlas-login) pass the per-channel /
// per-login ID as variadic args and a format string carrying "%s" in either
// the default or the env value; Resolve fmt.Sprintf's at runtime so the
// substitution happens after the PR-overlay patch has been applied.
package consumergroup

import (
	"fmt"
	"os"
)

const envVar = "KAFKA_CONSUMER_GROUP"

// Resolve returns the consumer group ID this service must use.
//
// Behaviour matrix:
//
//	KAFKA_CONSUMER_GROUP    args     result
//	------------------------------------------------------------
//	unset / ""              none     defaultName (verbatim)
//	unset / ""              N>0      fmt.Sprintf(defaultName, args...)
//	non-empty               none     env value (verbatim)
//	non-empty               N>0      fmt.Sprintf(envValue, args...)
//
// Whitespace-only env values are non-empty by this rule and therefore
// returned verbatim; design §5.4 keeps that semantic to surface
// config bugs rather than mask them.
//
// Existing zero-args callers (e.g. atlas-account, atlas-data) are
// source-compatible — they hit the verbatim paths above.
func Resolve(defaultName string, args ...any) string {
	v, ok := os.LookupEnv(envVar)
	if ok && v != "" {
		if len(args) > 0 {
			return fmt.Sprintf(v, args...)
		}
		return v
	}
	if len(args) > 0 {
		return fmt.Sprintf(defaultName, args...)
	}
	return defaultName
}
