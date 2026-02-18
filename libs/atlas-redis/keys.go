package redis

import (
	"fmt"
	"strings"

	"github.com/Chronicle20/atlas-tenant"
)

const keyPrefix = "atlas"
const keySeparator = ":"

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
