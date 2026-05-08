package redis

import (
	"fmt"
	"os"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const keyPrefixBase = "atlas"
const keySeparator = ":"

var keyPrefix = computeKeyPrefix(os.Getenv("ATLAS_ENV"))

// computeKeyPrefix returns "atlas" for empty atlasEnv (main env, legacy
// behavior) or "<atlasEnv>:atlas" for any non-empty value.
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

func CompositeKey(parts ...string) string {
	return strings.Join(parts, keySeparator)
}
