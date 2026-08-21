package redis

import (
	"fmt"
	"os"
	"strings"

	env "github.com/Chronicle20/atlas/libs/atlas-env"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const (
	keyPrefixBase = "atlas"
	keySeparator  = ":"
)

var keyPrefix = computeKeyPrefix(keyEnv())

// keyEnv resolves the environment discriminator for the Redis keyspace.
//
// It is deliberately NOT `ATLAS_ENV` alone. `ATLAS_ENV` names the
// *deployment*, and two things read it for two incompatible reasons:
//
//   - libs/atlas-lock (`leader.go`) scopes leader-election leases by it, and
//     MUST stay per-deployment — a sparse override sharing main's lease loses
//     every leader-gated sweep to main's pod (the task-200 defect).
//   - this package scopes the *keyspace* by it, and in a sparse environment
//     the keyspace must match the baseline's, because the override pods and
//     the baseline pods serve the same environments out of one Redis.
//
// `ATLAS_REDIS_ENV` splits the second use off the first. When unset — every
// deployment except a sparse overlay — the value falls back to `ATLAS_ENV`
// and behavior is byte-identical to before this variable existed.
func keyEnv() string {
	if v := os.Getenv("ATLAS_REDIS_ENV"); v != "" {
		return v
	}
	return os.Getenv("ATLAS_ENV")
}

// computeKeyPrefix returns "atlas" for empty atlasEnv (legacy behavior, and
// the value any deployment that sets neither variable gets) or
// "<atlasEnv>:atlas" for any non-empty value.
func computeKeyPrefix(atlasEnv string) string {
	if atlasEnv == "" {
		return keyPrefixBase
	}
	return atlasEnv + keySeparator + keyPrefixBase
}

// KeyPrefix returns the env-aware key prefix. Exported so callers
// composing keys outside the helper functions can avoid hardcoding "atlas:".
func KeyPrefix() string {
	return keyPrefix
}

func TenantKey(t tenant.Model) string {
	return fmt.Sprintf("%s%s%s%s%d.%d",
		t.Id().String(), keySeparator,
		t.Region(), keySeparator,
		t.MajorVersion(), t.MinorVersion())
}

func namespacedKey(namespace string, parts ...string) string {
	all := make([]string, 0, 2+len(parts))
	all = append(all, keyPrefix, namespace)
	all = append(all, parts...)
	return strings.Join(all, keySeparator)
}

func tenantEntityKey(namespace string, t tenant.Model, entityKey string) string {
	return namespacedKey(namespace, TenantKey(t), entityKey)
}

func tenantScanPattern(namespace string, t tenant.Model) string {
	return namespacedKey(namespace, TenantKey(t), "*")
}

// environmentEntityKey scopes a control-plane key by environment. The empty
// environment is the legacy value and produces exactly the key
// namespacedKey produced before environments existed, so main's existing
// state stays addressable (NFR-7).
func environmentEntityKey(namespace string, e env.Id, entityKey string) string {
	if e == "" {
		return namespacedKey(namespace, entityKey)
	}
	return namespacedKey(namespace, string(e), entityKey)
}

func environmentScanPattern(namespace string, e env.Id) string {
	if e == "" {
		return namespacedKey(namespace, "*")
	}
	return namespacedKey(namespace, string(e), "*")
}

func CompositeKey(parts ...string) string {
	return strings.Join(parts, keySeparator)
}
