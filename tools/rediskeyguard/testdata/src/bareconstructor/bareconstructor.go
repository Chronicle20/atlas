// Package bareconstructor is the analysistest fixture for the
// bare-constructor half of the rediskeyguard check. It must carry both a
// positive case (a bare libs/atlas-redis constructor, which must be flagged)
// and a negative case (its Tenant-scoped sibling, which must not be) — a
// fixture with only the positive case would pass trivially against a matcher
// too broad to distinguish the two.
package bareconstructor

import (
	goredis "github.com/redis/go-redis/v9"

	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
)

// bad calls the bare, env-global constructor — flagged.
func bad(client *goredis.Client) {
	_ = atlasredis.NewRegistry[string, string](client, "ns", func(k string) string { return k }) // want `rediskeyguard: NewRegistry is a bare \(non-tenant-scoped\) libs/atlas-redis constructor; use the Tenant-scoped equivalent \(D7\), or add this package to bareConstructorAllowlist with a written reason`
}

// good calls the Tenant-scoped sibling — not flagged.
func good(client *goredis.Client) {
	_ = atlasredis.NewTenantRegistry[string, string](client, "ns", func(k string) string { return k })
}

// The five constructors below are bare-shaped by name but have no
// bare/tenant-scoped split to migrate off of — every method they expose
// already takes a tenant.Model. Flagging them is a false positive; none of
// these are in bannedConstructors, so none should be reported.
func alreadyTenantScoped(client *goredis.Client) {
	_ = atlasredis.NewIndex(client, "ns", "idx")
	_ = atlasredis.NewUint32Index(client, "ns", "idx")
	_ = atlasredis.NewIDGenerator(client, "ns")
	_ = atlasredis.NewIDGeneratorWithStart(client, "ns", 1000)
	_ = atlasredis.NewTTLRegistry[string, string](client, "ns", func(k string) string { return k }, 0)
}
