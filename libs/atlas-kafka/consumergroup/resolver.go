// Package consumergroup resolves a service's Kafka consumer group ID.
//
// The default name is the service's historical literal (e.g. "Character Service").
// In environments where consumer-group isolation is required, the deployment
// sets KAFKA_CONSUMER_GROUP to a suffixed value such as
// "Character Service [a3f7]" and the env value is returned verbatim.
package consumergroup

import "os"

const envVar = "KAFKA_CONSUMER_GROUP"

// Resolve returns the consumer group ID this service must use.
// If KAFKA_CONSUMER_GROUP is set to a non-empty value (even to a
// non-trimmed whitespace-only value) it is returned verbatim.
// Otherwise defaultName is returned. See design §5.4: empty string
// is treated as unset for safety; whitespace-only is preserved
// verbatim to surface config bugs rather than silently mask them.
func Resolve(defaultName string) string {
	if v, ok := os.LookupEnv(envVar); ok && v != "" {
		return v
	}
	return defaultName
}
