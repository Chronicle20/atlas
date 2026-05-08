# Ephemeral Per-PR Deployments — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stand up Argo CD on the `bee` cluster, ship a Kustomize base/overlay layout for `deploy/k8s`, and add the per-environment isolation plumbing (`ATLAS_ENV` token through Postgres DB names, Kafka topics, Kafka consumer groups, Redis key prefix) so every open PR gets its own reachable, isolated review environment that auto-tears-down after a grace period.

**Architecture:** Four layers, three of them in this repo: `libs/atlas-redis` becomes env-aware via `KeyPrefix()`; a new `libs/atlas-kafka/consumergroup` resolver lets every service derive its consumer group from `KAFKA_CONSUMER_GROUP` with literal fallback; `deploy/k8s/` restructures into a `base` plus `overlays/main` (auto-synced by Argo) and `overlays/pr` (templated per PR); CI gains per-PR image builds and a PR-close cleanup hook. The fourth layer — Argo `Application(atlas-main)`, `ApplicationSet(atlas-pr)`, cleanup CronJob, Pi-hole secret — is staged in this repo at `deploy/argocd-bee/` for the maintainer to copy into the `tumidanski/k3s` infra repo.

**Tech Stack:** Go 1.25.5, Kustomize 4.5+, Argo CD 2.13.x with `goTemplate: true` and the GitHub PR generator, Traefik IngressRoute, Longhorn (existing on bee), Pi-hole v6 REST API, GitHub Actions, ghcr.io, kafka-tools and redis-cli (for hooks), `psql` 15.

---

## How to use this plan

- Tasks are TDD where the change is library code; for manifests, "tests" are `kustomize build` plus a diff against the live cluster; for shell scripts, `shellcheck` plus a `--dry-run` invocation.
- Every task ends with a `git commit` step. Don't batch commits across tasks.
- Phase 0 contains pre-flight checks that **must** be done before Phase 1. Their output goes into commit messages and the runbook (Phase 10), not into code.
- After every Go-package edit, run `go build ./...` from the package directory and `go test ./...` if the package has tests.
- Worktree: `<home>/source/atlas-ms/atlas/.worktrees/task-063-ephemeral-pr-deployments`. All paths below are relative to this worktree root.

---

## Phase 0: Pre-flight verification

These produce no committed code; they confirm the design's environmental assumptions and feed into the runbook (Phase 10). Capture the outputs in the runbook task. Run commands against the live `bee` cluster.

### Task 0.1: Verify Postgres role has CREATEDB

- [ ] **Step 1: Read the live `db-credentials` Secret to find the user**

```bash
kubectl get secret -n atlas db-credentials -o jsonpath='{.data.DB_USER}' | base64 -d
```

- [ ] **Step 2: Connect and query `pg_roles`**

```bash
PGPASSWORD=$(kubectl get secret -n atlas db-credentials -o jsonpath='{.data.DB_PASSWORD}' | base64 -d) \
  psql -h postgres.home -U "$(kubectl get secret -n atlas db-credentials -o jsonpath='{.data.DB_USER}' | base64 -d)" \
  -d postgres -c "SELECT rolname, rolcreatedb FROM pg_roles WHERE rolname = current_user;"
```

Expected: `rolcreatedb` is `t`. If `f`, run as a Postgres superuser:

```sql
ALTER ROLE <user> CREATEDB;
```

- [ ] **Step 3: Record outcome in `docs/tasks/task-063-ephemeral-pr-deployments/preflight.md`**

Create the file with one section:

```markdown
# Pre-flight findings (task-063)

## CREATEDB on db-credentials user
- Result: PASS / FAIL <date>
- Action taken (if FAIL): "ALTER ROLE <user> CREATEDB;" applied <date>
```

- [ ] **Step 4: Commit**

```bash
git add docs/tasks/task-063-ephemeral-pr-deployments/preflight.md
git commit -m "docs(task-063): record CREATEDB pre-flight outcome"
```

### Task 0.2: Verify Kafka auto-create topics

- [ ] **Step 1: Find the broker pod**

```bash
kubectl get pod -A -l app.kubernetes.io/name=kafka -o name | head -1
```

- [ ] **Step 2: Inspect broker config**

```bash
KAFKA_POD=<pod-from-step-1>
kubectl exec -n <kafka-ns> "$KAFKA_POD" -- \
  kafka-configs.sh --bootstrap-server localhost:9092 --describe --entity-type brokers --entity-name 0 \
  | grep -i auto.create
```

Expected: `auto.create.topics.enable=true` (the value visible should be `true`).

- [ ] **Step 3: Append outcome to `preflight.md`**

```markdown
## Kafka auto.create.topics.enable
- Result: true / false
- Action taken (if false): PreSync hook to create per-env topics is mandatory; document in plan §7
```

- [ ] **Step 4: Commit**

```bash
git add docs/tasks/task-063-ephemeral-pr-deployments/preflight.md
git commit -m "docs(task-063): record Kafka auto-create pre-flight outcome"
```

### Task 0.3: Re-confirm raw-redis-key audit list

The design enumerates a partial audit. Re-run grep and lock the list against drift before Phase 5.

- [ ] **Step 1: Grep services**

```bash
grep -rn '"atlas:' services/ libs/ --include='*.go' | grep -v _test.go > docs/tasks/task-063-ephemeral-pr-deployments/audit-redis-prefix.txt
```

- [ ] **Step 2: Grep for any other keyPrefix bypass**

```bash
grep -rn 'keyPrefix\|"atlas:"' libs/atlas-redis/ services/ --include='*.go' | grep -v _test.go >> docs/tasks/task-063-ephemeral-pr-deployments/audit-redis-prefix.txt
```

- [ ] **Step 3: Commit the audit snapshot**

```bash
git add docs/tasks/task-063-ephemeral-pr-deployments/audit-redis-prefix.txt
git commit -m "docs(task-063): snapshot raw-redis-key audit before sweep"
```

---

## Phase 1: `libs/atlas-redis` env-aware key prefix

**Files:**
- Modify: `libs/atlas-redis/keys.go`
- Create: `libs/atlas-redis/keys_test.go`

### Task 1.1: Failing tests for env-aware prefix

- [ ] **Step 1: Write failing tests in `libs/atlas-redis/keys_test.go`**

```go
package redis

import (
	"testing"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
)

func TestComputeKeyPrefix_envUnset(t *testing.T) {
	got := computeKeyPrefix("")
	if got != "atlas" {
		t.Fatalf("computeKeyPrefix(\"\") = %q, want %q", got, "atlas")
	}
}

func TestComputeKeyPrefix_envSet(t *testing.T) {
	got := computeKeyPrefix("a3f7")
	if got != "a3f7:atlas" {
		t.Fatalf("computeKeyPrefix(\"a3f7\") = %q, want %q", got, "a3f7:atlas")
	}
}

func TestKeyPrefix_returnsBaseWhenEnvUnset(t *testing.T) {
	if got := KeyPrefix(); got == "" {
		t.Fatalf("KeyPrefix() returned empty string")
	}
}

func TestNamespacedKey_useEnvAwarePrefix(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("a3f7")

	got := namespacedKey("buffs", "_tenants")
	want := "a3f7:atlas:buffs:_tenants"
	if got != want {
		t.Fatalf("namespacedKey = %q, want %q", got, want)
	}
}

func TestTenantEntityKey_useEnvAwarePrefix(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("a3f7")

	tm, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}

	got := tenantEntityKey("buffs", tm, "42")
	if want := "a3f7:atlas:buffs:"; len(got) < len(want) || got[:len(want)] != want {
		t.Fatalf("tenantEntityKey = %q, want prefix %q", got, want)
	}
}
```

- [ ] **Step 2: Run, expect failure (functions undefined)**

```bash
cd libs/atlas-redis && go test ./...
```

Expected: fails with `undefined: computeKeyPrefix` and `undefined: KeyPrefix`.

### Task 1.2: Implement env-aware prefix

- [ ] **Step 1: Replace `libs/atlas-redis/keys.go` with**

```go
package redis

import (
	"os"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
)

const keyPrefixBase = "atlas"
const keySeparator = ":"

// keyPrefix is computed once at package init from ATLAS_ENV.
// Empty env (the main env) yields the legacy "atlas" prefix.
var keyPrefix = computeKeyPrefix(os.Getenv("ATLAS_ENV"))

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
	return strings.Join([]string{
		t.Id().String(),
		t.Region(),
		// MajorVersion.MinorVersion serialised inline (matches existing format)
		formatVersion(t.MajorVersion(), t.MinorVersion()),
	}, keySeparator)
}

func formatVersion(major, minor uint16) string {
	var b strings.Builder
	b.Grow(8)
	b.WriteString(uintToString(uint64(major)))
	b.WriteByte('.')
	b.WriteString(uintToString(uint64(minor)))
	return b.String()
}

func uintToString(v uint64) string {
	if v == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
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
```

> **Note:** the original `keys.go` used `fmt.Sprintf` for `TenantKey`. Preserve byte-for-byte output (`"<uuid>:<region>:<major>.<minor>"`) — the rewrite above is mechanically equivalent but avoids `fmt` so no test imports drift. If the unit tests in §1.1 already pass with the original `fmt.Sprintf` body kept verbatim, KEEP IT and only change `keyPrefix` from `const` to `var` plus add `computeKeyPrefix` and `KeyPrefix`. Verify by checking `git diff libs/atlas-redis/keys.go` is minimal. If unsure, take the minimal-diff path:

```go
// Minimal diff:
// 1) `const keyPrefix = "atlas"` -> `var keyPrefix = computeKeyPrefix(os.Getenv("ATLAS_ENV"))`
// 2) Add helpers `computeKeyPrefix`, `KeyPrefix`
// 3) Add import "os"
// All other functions unchanged.
```

- [ ] **Step 2: Run tests, expect PASS**

```bash
cd libs/atlas-redis && go test ./...
```

Expected: all tests pass; `keys_test.go` and any pre-existing `*_test.go` green.

- [ ] **Step 3: Run race-detection tests for safety**

```bash
cd libs/atlas-redis && go test -race ./...
```

Expected: all tests pass.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-redis/keys.go libs/atlas-redis/keys_test.go
git commit -m "feat(atlas-redis): env-aware key prefix via ATLAS_ENV

Adds computeKeyPrefix() and exported KeyPrefix() so all helpers
(Registry, TenantRegistry, Index, Lock, coalesced, id, ttl) pick up
per-environment key isolation transparently. Empty ATLAS_ENV preserves
the legacy 'atlas' prefix, so the main env is unchanged.

Refs task-063."
```

---

## Phase 2: New library `libs/atlas-kafka/consumergroup`

**Files:**
- Create: `libs/atlas-kafka/consumergroup/resolver.go`
- Create: `libs/atlas-kafka/consumergroup/resolver_test.go`
- Modify: `libs/atlas-kafka/go.mod` only if a new dep is added (none expected)

### Task 2.1: Failing tests

- [ ] **Step 1: Create `libs/atlas-kafka/consumergroup/resolver_test.go`**

```go
package consumergroup

import (
	"testing"
)

func TestResolve_envUnset_returnsDefault(t *testing.T) {
	t.Setenv("KAFKA_CONSUMER_GROUP", "")
	if got := Resolve("Character Service"); got != "Character Service" {
		t.Fatalf("Resolve = %q, want %q", got, "Character Service")
	}
}

func TestResolve_envSet_returnsEnvValue(t *testing.T) {
	t.Setenv("KAFKA_CONSUMER_GROUP", "Character Service [a3f7]")
	if got := Resolve("Character Service"); got != "Character Service [a3f7]" {
		t.Fatalf("Resolve = %q, want %q", got, "Character Service [a3f7]")
	}
}

func TestResolve_envEmptyAfterTrim_returnsDefault(t *testing.T) {
	t.Setenv("KAFKA_CONSUMER_GROUP", "   ")
	// design §5.4 decision: do NOT trim. Whitespace-only is a config bug,
	// but we keep verbatim to avoid silently masking it.
	if got := Resolve("Character Service"); got != "   " {
		t.Fatalf("Resolve = %q, want verbatim whitespace", got)
	}
}
```

- [ ] **Step 2: Run, expect failure**

```bash
cd libs/atlas-kafka && go test ./consumergroup/...
```

Expected: `package consumergroup: cannot find package`.

### Task 2.2: Implement resolver

- [ ] **Step 1: Create `libs/atlas-kafka/consumergroup/resolver.go`**

```go
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
// If KAFKA_CONSUMER_GROUP is set (even to a non-trimmed value) it is
// returned verbatim. Otherwise defaultName is returned.
func Resolve(defaultName string) string {
	v, ok := os.LookupEnv(envVar)
	if !ok {
		return defaultName
	}
	return v
}
```

- [ ] **Step 2: Run tests, expect pass**

```bash
cd libs/atlas-kafka && go test ./consumergroup/...
```

Expected: all green.

- [ ] **Step 3: Build the whole module**

```bash
cd libs/atlas-kafka && go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add libs/atlas-kafka/consumergroup/
git commit -m "feat(atlas-kafka): consumergroup.Resolve for env-driven group IDs

Adds a small resolver so each service can pick its consumer group ID
from KAFKA_CONSUMER_GROUP with the historical literal as fallback.
Used by the per-PR Kustomize overlay to suffix every group with the
ATLAS_ENV token.

Refs task-063."
```

---

## Phase 3: Library `libs/atlas-object-id` raw-key fix

**Files:**
- Modify: `libs/atlas-object-id/allocator.go`

The library composes Redis keys outside the helpers. Route them through `KeyPrefix()`.

### Task 3.1: Patch `allocator.go`

- [ ] **Step 1: Read the current file**

```bash
sed -n '95,115p' libs/atlas-object-id/allocator.go
```

Expected lines (verbatim or close): two `fmt.Sprintf` calls hardcoding `"atlas:oid:..."`.

- [ ] **Step 2: Replace both literals with `redis.KeyPrefix()`**

For each occurrence of:

```go
return fmt.Sprintf("atlas:oid:%s:next", t.Id().String())
```

substitute:

```go
return fmt.Sprintf("%s:oid:%s:next", atlasredis.KeyPrefix(), t.Id().String())
```

For:

```go
return fmt.Sprintf("atlas:oid:%s:free", t.Id().String())
```

substitute:

```go
return fmt.Sprintf("%s:oid:%s:free", atlasredis.KeyPrefix(), t.Id().String())
```

Add the import (alias `atlasredis` to avoid collision with stdlib `redis` packages elsewhere):

```go
atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
```

- [ ] **Step 3: Add a unit test if one is missing**

If `libs/atlas-object-id/allocator_test.go` already covers key shape, extend it; otherwise add:

```go
package objectid

import (
	"testing"

	"github.com/google/uuid"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestAllocator_keysRespectEnvPrefix(t *testing.T) {
	id := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	tm, err := tenant.Create(id, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	prefix := atlasredis.KeyPrefix()
	gotNext := nextKey(tm)
	want := prefix + ":oid:" + id.String() + ":next"
	if gotNext != want {
		t.Fatalf("nextKey = %q, want %q", gotNext, want)
	}
}
```

(If `nextKey`/`freeKey` are unexported, add a small exported test helper or use a `*_internal_test.go` file in the same package.)

- [ ] **Step 4: Build and test**

```bash
cd libs/atlas-object-id && go build ./... && go test ./...
```

Expected: all green.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-object-id/allocator.go libs/atlas-object-id/allocator_test.go
git commit -m "fix(atlas-object-id): use redis.KeyPrefix() for env isolation

Routes the two oid keys through libs/atlas-redis.KeyPrefix() so the
ATLAS_ENV token reaches the allocator. Closes a gap that would have
let two PR envs share the same oid keyspace.

Refs task-063."
```

---

## Phase 4: Service consumer-group sweep (49 services)

The pattern is mechanical: every `services/atlas-*/atlas.com/*/main.go` either has

```go
const consumerGroupId = "<Service Name>"
```

or (atlas-channel, atlas-login):

```go
const consumerGroupIdTemplate = "<Service Name> - %s"
// later: var consumerGroupId = fmt.Sprintf(consumerGroupIdTemplate, config.Id.String())
```

Every service needs an import and a one-line edit.

### Task 4.1: Sweep services with literal consumer group (47 services)

**Files (all 47):** see Appendix A at the bottom of this plan for the full list with literals.

- [ ] **Step 1: For each service in Appendix A, apply the following edit**

Add (or merge into the existing import block):

```go
import (
    consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
)
```

Replace the line:

```go
const consumerGroupId = "<Service Name>"
```

with:

```go
var consumerGroupId = consumergroup.Resolve("<Service Name>")
```

Keep the same literal string; only the keyword changes (`const` → `var`) and the right-hand side becomes `consumergroup.Resolve(...)`.

- [ ] **Step 2: Verify each service builds**

For every service in Appendix A:

```bash
cd services/<service>/atlas.com/<module>
go build ./...
```

Expected: no errors. If a service module declares `consumergroup` differently (case mismatch in import alias), reconcile to the canonical alias `consumergroup`.

- [ ] **Step 3: Run each service's tests**

```bash
cd services/<service>/atlas.com/<module>
go test ./...
```

Expected: existing tests still pass.

- [ ] **Step 4: Commit by service area**

Group commits to keep PR-review tractable. Suggested batches:
- "core" services: atlas-account, atlas-character, atlas-character-factory, atlas-tenants, atlas-world, atlas-channel, atlas-login (channel/login covered by Task 4.2 — leave for that batch)
- "social" services: atlas-buddies, atlas-buffs, atlas-chairs, atlas-chalkboards, atlas-expressions, atlas-fame, atlas-families, atlas-guilds, atlas-invites, atlas-marriages, atlas-merchant, atlas-messengers, atlas-notes, atlas-parties, atlas-party-quests
- "world" services: atlas-asset-expiration, atlas-cashshop, atlas-consumables, atlas-data, atlas-drops, atlas-effective-stats, atlas-inventory, atlas-keys, atlas-map-actions, atlas-maps, atlas-monster-death, atlas-monsters, atlas-npc-conversations, atlas-npc-shops, atlas-pets, atlas-portal-actions, atlas-portals, atlas-quest, atlas-rates, atlas-reactor-actions, atlas-reactors, atlas-skills, atlas-storage, atlas-transports
- "infra" services: atlas-ban, atlas-messages, atlas-saga-orchestrator

Commit each batch:

```bash
git add services/atlas-<svc>/atlas.com/<module>/main.go ...
git commit -m "feat(<batch-name>): resolve consumer group from KAFKA_CONSUMER_GROUP

Sweeps <N> services to read the consumer group ID from env via
libs/atlas-kafka/consumergroup.Resolve(), preserving the historical
literal as fallback. Enables per-PR consumer-group isolation when
the deployment sets KAFKA_CONSUMER_GROUP=\"<literal> [<ATLAS_ENV>]\".

Services in this batch: <comma-separated list>.

Refs task-063."
```

### Task 4.2: Sweep services with templated consumer group

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/main.go`
- Modify: `services/atlas-login/atlas.com/login/main.go`

Both compute the group ID at runtime from a template plus `config.Id`. Wrap the runtime computation with `Resolve` so a deployment-supplied env wins.

- [ ] **Step 1: In `services/atlas-channel/atlas.com/channel/main.go`, add the import**

```go
import (
    consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"
)
```

- [ ] **Step 2: Change the runtime line**

Find:

```go
var consumerGroupId = fmt.Sprintf(consumerGroupIdTemplate, config.Id.String())
```

Replace with:

```go
var consumerGroupId = consumergroup.Resolve(fmt.Sprintf(consumerGroupIdTemplate, config.Id.String()))
```

- [ ] **Step 3: Repeat for `services/atlas-login/atlas.com/login/main.go`**

Same pattern.

- [ ] **Step 4: Build both services**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./...
cd ../../../atlas-login/atlas.com/login && go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/main.go \
        services/atlas-login/atlas.com/login/main.go
git commit -m "feat(channel,login): consumer group resolves env override

Wraps the templated group computation so KAFKA_CONSUMER_GROUP wins
when set, preserving the templated default for the main env.

Refs task-063."
```

### Task 4.3: Build the entire workspace

- [ ] **Step 1: From repo root**

```bash
go build ./... 2>&1 | tee build.log
```

Expected: no errors. If any service fails to build (forgotten import, typo'd literal), fix it before continuing.

- [ ] **Step 2: Run all tests**

```bash
go test ./... 2>&1 | tee test.log
```

Expected: pre-existing failures stay (note them in `preflight.md`); no new failures.

- [ ] **Step 3: Discard the log files** (they don't ship in this commit)

```bash
rm build.log test.log
```

If the workspace passes, no commit; the proof is the green build.

---

## Phase 5: Service raw-Redis-key audit fixes

12 services compose Redis keys outside the helper, hardcoding `"atlas:"`. Each must call `atlasredis.KeyPrefix()` instead. The list is locked in Task 0.3's `audit-redis-prefix.txt`. The pattern is:

```go
// before
return fmt.Sprintf("atlas:%s:_tenants", r.reg.Namespace())
// after
return fmt.Sprintf("%s:%s:_tenants", atlasredis.KeyPrefix(), r.reg.Namespace())
```

Each task is one service.

### Task 5.1: atlas-buffs

**Files:** `services/atlas-buffs/atlas.com/buffs/character/registry.go:46`

- [ ] **Step 1: Replace the hardcoded prefix**

Change:

```go
return "atlas:" + r.characters.Namespace() + ":_tenants"
```

to:

```go
return atlasredis.KeyPrefix() + ":" + r.characters.Namespace() + ":_tenants"
```

(`atlasredis` is already aliased at the top of the file — verify and add if missing.)

- [ ] **Step 2: Build and test**

```bash
cd services/atlas-buffs/atlas.com/buffs && go build ./... && go test ./...
```

- [ ] **Step 3: Commit**

```bash
git add services/atlas-buffs/atlas.com/buffs/character/registry.go
git commit -m "fix(atlas-buffs): tenant key uses redis.KeyPrefix()

Fixes ATLAS_ENV bypass at character/registry.go:46.

Refs task-063."
```

### Task 5.2: atlas-npc-shops

**Files:**
- `services/atlas-npc-shops/atlas.com/npc/shops/cache.go:29` — `"atlas:npc-shop:consumables:%s"`
- `services/atlas-npc-shops/atlas.com/npc/shops/registry.go:34` — `"atlas:npc-shop-chars:%s:%d"`

- [ ] **Step 1: Replace both literals with `atlasredis.KeyPrefix()` followed by `:npc-shop:...`**

```go
// cache.go:29
return fmt.Sprintf("%s:npc-shop:consumables:%s", atlasredis.KeyPrefix(), tenantId.String())
// registry.go:34
return fmt.Sprintf("%s:npc-shop-chars:%s:%d", atlasredis.KeyPrefix(), atlas.TenantKey(t), shopId)
```

- [ ] **Step 2: Build and test**

```bash
cd services/atlas-npc-shops/atlas.com/npc && go build ./... && go test ./...
```

- [ ] **Step 3: Commit**

```bash
git add services/atlas-npc-shops/atlas.com/npc/shops/cache.go \
        services/atlas-npc-shops/atlas.com/npc/shops/registry.go
git commit -m "fix(atlas-npc-shops): cache and registry keys use redis.KeyPrefix()

Refs task-063."
```

### Task 5.3: atlas-portals

**Files:** `services/atlas-portals/atlas.com/portals/blocked/cache.go:33`

- [ ] **Step 1: Replace `"atlas:%s:%s:%s"` literal**

```go
return fmt.Sprintf("%s:%s:%s:%s", atlasredis.KeyPrefix(), r.namespace, atlas.TenantKey(t), strconv.FormatUint(uint64(characterId), 10))
```

- [ ] **Step 2: Build and test**

```bash
cd services/atlas-portals/atlas.com/portals && go build ./... && go test ./...
```

- [ ] **Step 3: Commit**

```bash
git add services/atlas-portals/atlas.com/portals/blocked/cache.go
git commit -m "fix(atlas-portals): blocked-cache key uses redis.KeyPrefix()

Refs task-063."
```

### Task 5.4: atlas-pets

**Files:** `services/atlas-pets/atlas.com/pets/character/registry.go:40,69,70`

- [ ] **Step 1: Replace three literals**

Lines 40, 69, 70 use `"atlas:%s:_tenants"`, `"atlas:%s:%s:*"`, `"atlas:%s:%s:"`. Convert each to use `atlasredis.KeyPrefix()` as the prefix expression.

- [ ] **Step 2: Build and test**

- [ ] **Step 3: Commit**

```bash
git add services/atlas-pets/atlas.com/pets/character/registry.go
git commit -m "fix(atlas-pets): registry keys use redis.KeyPrefix()

Refs task-063."
```

### Task 5.5: atlas-skills

**Files:** `services/atlas-skills/atlas.com/skills/skill/cooldown_registry.go:39,62,123,124`

- [ ] **Step 1: Replace four literals (`atlas:%s:_tenants`, `atlas:%s:%s:%s*`, `atlas:%s:%s:*`, `atlas:%s:%s:`)**

- [ ] **Step 2: Build and test**

- [ ] **Step 3: Commit**

```bash
git add services/atlas-skills/atlas.com/skills/skill/cooldown_registry.go
git commit -m "fix(atlas-skills): cooldown keys use redis.KeyPrefix()

Refs task-063."
```

### Task 5.6: atlas-expressions

**Files:** `services/atlas-expressions/atlas.com/expressions/expression/registry.go:31`

- [ ] **Step 1: Replace `tenantKey: "atlas:expression:_tenants"` with composed prefix**

```go
tenantKey: atlasredis.KeyPrefix() + ":expression:_tenants",
```

- [ ] **Step 2: Build and test**

- [ ] **Step 3: Commit**

```bash
git add services/atlas-expressions/atlas.com/expressions/expression/registry.go
git commit -m "fix(atlas-expressions): tenant key uses redis.KeyPrefix()

Refs task-063."
```

### Task 5.7: atlas-maps

**Files:** `services/atlas-maps/atlas.com/maps/map/monster/registry.go:60,260`

- [ ] **Step 1: Replace literals**

Line 60: `"atlas:maps:spawn:%s:%d:%d:%d:%s"` → `"%s:maps:spawn:%s:%d:%d:%d:%s"` with `atlasredis.KeyPrefix()` prepended.

Line 260: `r.client.Scan(ctx, 0, "atlas:maps:spawn:*", 0)` → `r.client.Scan(ctx, 0, atlasredis.KeyPrefix()+":maps:spawn:*", 0)`.

> **NOTE — atlas-maps spawn cache memory:** `reference_atlas_maps_spawn_cache.md` documents that this cache must be cleared after WZ data redeploy. The PR-env bootstrap starts from an empty Redis namespace, so this is moot for PR envs but call it out in the cleanup runbook (Phase 10).

- [ ] **Step 2: Build and test**

- [ ] **Step 3: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/map/monster/registry.go
git commit -m "fix(atlas-maps): spawn cache keys use redis.KeyPrefix()

Refs task-063."
```

### Task 5.8: atlas-chairs

**Files:** `services/atlas-chairs/atlas.com/chairs/character/registry.go:34,69`

- [ ] **Step 1: Replace literals**

Both lines use `"atlas:%s:..."` patterns; rewrite with `atlasredis.KeyPrefix()`.

- [ ] **Step 2: Build and test**

- [ ] **Step 3: Commit**

```bash
git add services/atlas-chairs/atlas.com/chairs/character/registry.go
git commit -m "fix(atlas-chairs): character registry keys use redis.KeyPrefix()

Refs task-063."
```

### Task 5.9: atlas-storage

**Files:** `services/atlas-storage/atlas.com/storage/storage/cache.go:36`

- [ ] **Step 1: Replace `"atlas:npc-context:%d"`**

```go
return fmt.Sprintf("%s:npc-context:%d", atlasredis.KeyPrefix(), characterId)
```

- [ ] **Step 2: Build and test**

- [ ] **Step 3: Commit**

```bash
git add services/atlas-storage/atlas.com/storage/storage/cache.go
git commit -m "fix(atlas-storage): npc-context cache key uses redis.KeyPrefix()

Refs task-063."
```

### Task 5.10: atlas-character

**Files:** `services/atlas-character/atlas.com/character/session/registry.go:38`

- [ ] **Step 1: Replace `"atlas:%s:_tenants"`**

- [ ] **Step 2: Build and test**

- [ ] **Step 3: Commit**

```bash
git add services/atlas-character/atlas.com/character/session/registry.go
git commit -m "fix(atlas-character): session tenant key uses redis.KeyPrefix()

Refs task-063."
```

### Task 5.11: atlas-chalkboards

**Files:** `services/atlas-chalkboards/atlas.com/chalkboards/character/registry.go:33`

- [ ] **Step 1: Replace `"atlas:%s:%s:%d:%d:%d:%s"`**

- [ ] **Step 2: Build and test**

- [ ] **Step 3: Commit**

```bash
git add services/atlas-chalkboards/atlas.com/chalkboards/character/registry.go
git commit -m "fix(atlas-chalkboards): registry key uses redis.KeyPrefix()

Refs task-063."
```

### Task 5.12: atlas-monsters

**Files:** `services/atlas-monsters/atlas.com/monsters/monster/cooldown.go:32`

- [ ] **Step 1: Replace `"atlas:monster-cooldown:%s:%s:%s"`**

```go
return fmt.Sprintf("%s:monster-cooldown:%s:%s:%s", atlasredis.KeyPrefix(), ...)
```

- [ ] **Step 2: Build and test**

- [ ] **Step 3: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/cooldown.go
git commit -m "fix(atlas-monsters): cooldown key uses redis.KeyPrefix()

Refs task-063."
```

### Task 5.13: Final audit grep

- [ ] **Step 1: Re-run the audit**

```bash
grep -rn '"atlas:' services/ libs/ --include='*.go' | grep -v _test.go
```

Expected: only matches inside test files OR inside string literals that are clearly documentation (e.g. `// example: "atlas:..."`). Every production composition must route through `atlasredis.KeyPrefix()`.

- [ ] **Step 2: If any production hit remains, repeat the pattern from Tasks 5.1–5.12.**

- [ ] **Step 3: No commit if audit clean.**

---

## Phase 6: Bootstrap container source

A new service `atlas-pr-bootstrap` produces a small image used by the PostSync bootstrap Job and the PostDelete cleanup Job. Two entrypoints share one image.

**Files:**
- Create: `services/atlas-pr-bootstrap/Dockerfile`
- Create: `services/atlas-pr-bootstrap/scripts/bootstrap.sh`
- Create: `services/atlas-pr-bootstrap/scripts/cleanup.sh`
- Create: `services/atlas-pr-bootstrap/scripts/lib.sh`
- Create: `services/atlas-pr-bootstrap/test/bootstrap_test.bats`
- Create: `services/atlas-pr-bootstrap/test/cleanup_test.bats`
- Create: `services/atlas-pr-bootstrap/README.md`
- Modify: `.github/config/services.json` (add the new service so detect-changes picks it up)

### Task 6.1: Scaffold

- [ ] **Step 1: Create `services/atlas-pr-bootstrap/Dockerfile`**

```dockerfile
FROM alpine:3.20

RUN apk add --no-cache \
        bash \
        curl \
        jq \
        postgresql-client \
        redis \
        kafka-tools \
        ca-certificates \
        github-cli

WORKDIR /atlas
COPY scripts/lib.sh /atlas/lib.sh
COPY scripts/bootstrap.sh /atlas/bootstrap.sh
COPY scripts/cleanup.sh /atlas/cleanup.sh

RUN chmod +x /atlas/bootstrap.sh /atlas/cleanup.sh

ENTRYPOINT ["/atlas/bootstrap.sh"]
```

- [ ] **Step 2: Create `services/atlas-pr-bootstrap/scripts/lib.sh`**

```bash
#!/usr/bin/env bash
# Shared helpers for bootstrap.sh and cleanup.sh.

set -euo pipefail

log() {
    local level="$1"; shift
    local step="${ATLAS_STEP:-init}"
    printf '{"ts":"%s","level":"%s","atlas.env":"%s","atlas.cleanup-step":"%s","msg":%s}\n' \
        "$(date -u +%Y-%m-%dT%H:%M:%SZ)" "$level" "${ATLAS_ENV:-}" "$step" \
        "$(printf '%s' "$*" | jq -Rs .)"
}

require_env() {
    for v in "$@"; do
        if [ -z "${!v:-}" ]; then
            log error "missing required env: $v"
            exit 1
        fi
    done
}

retry() {
    local max=$1; shift
    local sleep_s=$1; shift
    local n=0
    while ! "$@"; do
        n=$((n+1))
        if [ "$n" -ge "$max" ]; then
            log error "retry exhausted after $n attempts: $*"
            return 1
        fi
        sleep "$sleep_s"
    done
}

http_ok() {
    local url=$1
    local status
    status=$(curl -s -o /dev/null -w '%{http_code}' "$url" || echo 000)
    [ "$status" = "200" ] || [ "$status" = "204" ]
}
```

- [ ] **Step 3: Create `services/atlas-pr-bootstrap/scripts/bootstrap.sh`**

```bash
#!/usr/bin/env bash
# Atlas PR-env bootstrap. Idempotent — short-circuits each step that
# is already complete. Reads:
#   ATLAS_ENV          — env hash, REQUIRED
#   ATLAS_UI_BASE      — http://atlas-ingress.<ns>.svc.cluster.local
#   WZ_CANONICAL       — path to canonical zip (default /opt/wz/atlas.zip)
#   TENANT_ID/REGION/MAJOR_VERSION/MINOR_VERSION — required for tenant headers

set -euo pipefail

# shellcheck source=lib.sh
. "$(dirname "$0")/lib.sh"

require_env ATLAS_ENV ATLAS_UI_BASE TENANT_ID REGION MAJOR_VERSION MINOR_VERSION
WZ_CANONICAL="${WZ_CANONICAL:-/opt/wz/atlas.zip}"

post() {
    curl -fsS -X POST \
        -H "TENANT_ID: $TENANT_ID" \
        -H "REGION: $REGION" \
        -H "MAJOR_VERSION: $MAJOR_VERSION" \
        -H "MINOR_VERSION: $MINOR_VERSION" \
        -H "Content-Type: application/json" \
        "$@" -d '{}'
}

get_attr() {
    curl -fsS \
        -H "TENANT_ID: $TENANT_ID" \
        -H "REGION: $REGION" \
        -H "MAJOR_VERSION: $MAJOR_VERSION" \
        -H "MINOR_VERSION: $MINOR_VERSION" \
        -H "Accept: application/vnd.api+json" \
        "$1" | jq -r ".data.attributes.$2"
}

ATLAS_STEP=wait-ready log info "waiting for atlas-data and atlas-wz-extractor"
retry 60 5 http_ok "$ATLAS_UI_BASE/api/data/status"
retry 60 5 http_ok "$ATLAS_UI_BASE/api/wz/input"
retry 60 5 http_ok "$ATLAS_UI_BASE/api/wz/extractions"

# WZ upload: PATCH /api/wz/input
ATLAS_STEP=wz-upload
files=$(get_attr "$ATLAS_UI_BASE/api/wz/input" fileCount)
if [ "$files" = "0" ] || [ "$files" = "null" ]; then
    log info "uploading canonical WZ zip"
    curl -fsS -X PATCH \
        -H "TENANT_ID: $TENANT_ID" \
        -H "REGION: $REGION" \
        -H "MAJOR_VERSION: $MAJOR_VERSION" \
        -H "MINOR_VERSION: $MINOR_VERSION" \
        -F "zip_file=@$WZ_CANONICAL" \
        "$ATLAS_UI_BASE/api/wz/input"
else
    log info "WZ already uploaded (fileCount=$files), skipping"
fi

# WZ extraction
ATLAS_STEP=wz-extract
extracted=$(get_attr "$ATLAS_UI_BASE/api/wz/extractions" fileCount)
if [ "$extracted" = "0" ] || [ "$extracted" = "null" ]; then
    log info "running WZ extraction"
    post "$ATLAS_UI_BASE/api/wz/extractions"
    retry 240 10 sh -c "[ \"$(curl -fsS -H 'TENANT_ID: $TENANT_ID' -H 'REGION: $REGION' -H 'MAJOR_VERSION: $MAJOR_VERSION' -H 'MINOR_VERSION: $MINOR_VERSION' -H 'Accept: application/vnd.api+json' '$ATLAS_UI_BASE/api/wz/extractions' | jq -r '.data.attributes.fileCount')\" != '0' ]"
fi

# Data processing
ATLAS_STEP=data-process
docs=$(get_attr "$ATLAS_UI_BASE/api/data/status" documentCount)
if [ "$docs" = "0" ] || [ "$docs" = "null" ]; then
    log info "running data processing"
    post "$ATLAS_UI_BASE/api/data/process"
    retry 240 10 sh -c "[ \"$(curl -fsS -H 'TENANT_ID: $TENANT_ID' -H 'REGION: $REGION' -H 'MAJOR_VERSION: $MAJOR_VERSION' -H 'MINOR_VERSION: $MINOR_VERSION' -H 'Accept: application/vnd.api+json' '$ATLAS_UI_BASE/api/data/status' | jq -r '.data.attributes.documentCount')\" != '0' ]"
fi

# Per-domain seeds, in parallel
ATLAS_STEP=seed
log info "seeding domain data"
endpoints=(
    /api/drops/seed
    /api/gachapons/seed
    /api/npcs/conversations/seed
    /api/quests/conversations/seed
    /api/shops/seed
    /api/portals/scripts/seed
    /api/reactors/actions/seed
    /api/maps/actions/seed
)
for ep in "${endpoints[@]}"; do
    ( post "$ATLAS_UI_BASE$ep" >/dev/null && log info "seeded $ep" ) &
done
wait

ATLAS_STEP=done log info "bootstrap complete"
```

- [ ] **Step 4: Create `services/atlas-pr-bootstrap/scripts/cleanup.sh`**

```bash
#!/usr/bin/env bash
# Atlas PR-env cleanup. Each step is idempotent; failures stop the run
# and leave the env intact for inspection (ArgoCD Application stays in
# 'cleanup-failed' state).
#
# Required env:
#   ATLAS_ENV         — env hash
#   DB_HOST/USER/PASS — Postgres credentials
#   ATLAS_DB_NAMES    — comma-separated list of base DB names
#   BOOTSTRAP_SERVERS — kafka.home:9093
#   REDIS_URL         — redis.home:6379
#   PIHOLE_API_BASE_1, PIHOLE_TOKEN_1, PIHOLE_API_BASE_2, PIHOLE_TOKEN_2
#   GHCR_TOKEN        — for image-tag delete
#   PR_NUMBER         — for image-tag prefix
#   ATLAS_SERVICES    — comma-separated list of service names for image cleanup

set -euo pipefail

# shellcheck source=lib.sh
. "$(dirname "$0")/lib.sh"

require_env ATLAS_ENV DB_HOST DB_USER DB_PASSWORD ATLAS_DB_NAMES BOOTSTRAP_SERVERS REDIS_URL PR_NUMBER

ATLAS_STEP=drop-dbs log info "dropping per-env Postgres databases"
IFS=',' read -ra dbs <<< "$ATLAS_DB_NAMES"
for db in "${dbs[@]}"; do
    full="${db}-${ATLAS_ENV}"
    PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -U "$DB_USER" -d postgres \
        -c "DROP DATABASE IF EXISTS \"$full\";" || {
            log error "failed to drop $full"
            exit 1
        }
done

ATLAS_STEP=drop-topics log info "deleting per-env Kafka topics"
kafka-topics.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --list \
    | grep -E -- "-${ATLAS_ENV}\$" \
    | xargs -r -n 1 kafka-topics.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --delete --topic

ATLAS_STEP=drop-groups log info "deleting per-env consumer groups"
kafka-consumer-groups.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --list \
    | grep -E -- "\\[${ATLAS_ENV}\\]\$" \
    | xargs -r -n 1 kafka-consumer-groups.sh --bootstrap-server "$BOOTSTRAP_SERVERS" --delete --group

ATLAS_STEP=drop-redis log info "deleting per-env Redis keys"
redis-cli -u "redis://$REDIS_URL" --scan --pattern "${ATLAS_ENV}:*" \
    | xargs -r -n 1000 redis-cli -u "redis://$REDIS_URL" DEL

if [ -n "${ATLAS_SERVICES:-}" ] && [ -n "${GHCR_TOKEN:-}" ]; then
    ATLAS_STEP=drop-images log info "deleting per-PR ghcr image tags"
    IFS=',' read -ra svcs <<< "$ATLAS_SERVICES"
    for svc in "${svcs[@]}"; do
        gh api -H "Authorization: Bearer $GHCR_TOKEN" \
            "/users/chronicle20/packages/container/${svc}%2F${svc}/versions" \
            --jq ".[] | select(.metadata.container.tags[]? | startswith(\"pr-${PR_NUMBER}-\")) | .id" \
            | while read -r vid; do
                gh api --method DELETE -H "Authorization: Bearer $GHCR_TOKEN" \
                    "/users/chronicle20/packages/container/${svc}%2F${svc}/versions/${vid}" || true
            done
    done
fi

if [ -n "${PIHOLE_API_BASE_1:-}" ] && [ -n "${PIHOLE_TOKEN_1:-}" ]; then
    ATLAS_STEP=drop-dns log info "removing Pi-hole A records"
    for i in 1 2; do
        base_var="PIHOLE_API_BASE_$i"
        token_var="PIHOLE_TOKEN_$i"
        base="${!base_var:-}"
        token="${!token_var:-}"
        if [ -z "$base" ] || [ -z "$token" ]; then
            continue
        fi
        # Pi-hole v6: list existing hosts, find one matching <PR>.atlas.home, delete by id.
        host="${PR_NUMBER}.atlas.home"
        id=$(curl -fsS -H "X-Pi-Auth: $token" "$base/api/config/dns/hosts" \
            | jq -r ".hosts[] | select(.name == \"$host\") | .id" | head -1)
        if [ -n "$id" ]; then
            curl -fsS --request DELETE -H "X-Pi-Auth: $token" "$base/api/config/dns/hosts/$id" || \
                log warn "Pi-hole $i delete failed for $host"
        fi
    done
fi

ATLAS_STEP=done log info "cleanup complete"
```

- [ ] **Step 5: Create `services/atlas-pr-bootstrap/test/bootstrap_test.bats`**

```bash
#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
}

@test "bootstrap.sh fails without ATLAS_ENV" {
    run env -u ATLAS_ENV bash "$PROJECT_ROOT/scripts/bootstrap.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"missing required env: ATLAS_ENV"* ]]
}

@test "bootstrap.sh fails without ATLAS_UI_BASE" {
    run env ATLAS_ENV=test -u ATLAS_UI_BASE bash "$PROJECT_ROOT/scripts/bootstrap.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"missing required env: ATLAS_UI_BASE"* ]]
}
```

- [ ] **Step 6: Create `services/atlas-pr-bootstrap/test/cleanup_test.bats`**

```bash
#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
}

@test "cleanup.sh fails without ATLAS_ENV" {
    run env -u ATLAS_ENV bash "$PROJECT_ROOT/scripts/cleanup.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"missing required env: ATLAS_ENV"* ]]
}

@test "cleanup.sh fails without ATLAS_DB_NAMES" {
    run env ATLAS_ENV=test DB_HOST=h DB_USER=u DB_PASSWORD=p \
        BOOTSTRAP_SERVERS=k REDIS_URL=r PR_NUMBER=1 \
        -u ATLAS_DB_NAMES bash "$PROJECT_ROOT/scripts/cleanup.sh"
    [ "$status" -ne 0 ]
    [[ "$output" == *"missing required env: ATLAS_DB_NAMES"* ]]
}
```

- [ ] **Step 7: Run shellcheck and bats**

```bash
shellcheck services/atlas-pr-bootstrap/scripts/*.sh
cd services/atlas-pr-bootstrap && bats test/
```

Expected: shellcheck clean (or only style warnings on the `retry` polling sub-shells, which are accepted because they need eval-time variable expansion); bats green.

- [ ] **Step 8: Create `services/atlas-pr-bootstrap/README.md`**

```markdown
# atlas-pr-bootstrap

Image used by ephemeral per-PR environments for bootstrap (PostSync hook)
and cleanup (PostDelete hook). Two entrypoints share one image:

- `/atlas/bootstrap.sh` — uploads the canonical WZ zip and seeds every
  domain via the existing atlas-ui SetupPage endpoints.
- `/atlas/cleanup.sh` — drops per-env Postgres DBs, Kafka topics, Kafka
  consumer groups, Redis keys, ghcr image tags, and Pi-hole A records.

Both scripts read `ATLAS_ENV` and emit JSON-line logs to stdout for Loki.

See `docs/runbooks/ephemeral-pr-deployments.md` for operational docs.
```

- [ ] **Step 9: Add to `.github/config/services.json`**

Append a new entry (alphabetical order with other static services):

```json
{
  "name": "atlas-pr-bootstrap",
  "type": "support-image",
  "path": "services/atlas-pr-bootstrap",
  "docker_image": "ghcr.io/chronicle20/atlas-pr-bootstrap/atlas-pr-bootstrap",
  "docker_context": "services/atlas-pr-bootstrap"
}
```

- [ ] **Step 10: Commit**

```bash
git add services/atlas-pr-bootstrap/ .github/config/services.json
git commit -m "feat(atlas-pr-bootstrap): bootstrap and cleanup container

Adds the support image used by the per-PR PostSync and PostDelete
hooks. bootstrap.sh runs the existing atlas-ui SetupPage flow against
the per-PR namespace; cleanup.sh reclaims DBs, topics, consumer groups,
Redis keys, ghcr image tags, and Pi-hole DNS entries.

Includes bats tests for required-env validation and shellcheck-clean
shell. See services/atlas-pr-bootstrap/README.md for usage.

Refs task-063."
```

---

## Phase 7: `deploy/k8s/` Kustomize restructure

**Files (created):**
- `deploy/k8s/base/kustomization.yaml`
- `deploy/k8s/base/<every-current-yaml-moved-here>` — namespace stripped from each Deployment
- `deploy/k8s/overlays/main/kustomization.yaml`
- `deploy/k8s/overlays/pr/kustomization.yaml`
- `deploy/k8s/overlays/pr/ingress-route.yaml`
- `deploy/k8s/overlays/pr/presync-create-dbs.yaml`
- `deploy/k8s/overlays/pr/postsync-bootstrap.yaml`
- `deploy/k8s/overlays/pr/postsync-pihole-add.yaml`
- `deploy/k8s/overlays/pr/postdelete-cleanup.yaml`
- `deploy/k8s/overlays/pr/patches/db-name-suffix.yaml`
- `deploy/k8s/overlays/pr/patches/consumer-group-env.yaml`
- `deploy/k8s/overlays/pr/patches/atlas-env-env.yaml`
- `deploy/k8s/overlays/pr/atlas-env-tokens.yaml`
- `deploy/k8s/README.md`

### Task 7.1: Move base manifests

- [ ] **Step 1: Make the base directory and move every flat manifest**

```bash
mkdir -p deploy/k8s/base
git mv deploy/k8s/atlas-*.yaml deploy/k8s/base/
git mv deploy/k8s/env-configmap.yaml deploy/k8s/base/
git mv deploy/k8s/ingress.yaml deploy/k8s/base/atlas-ingress.yaml
git mv deploy/k8s/namespace.yaml deploy/k8s/base/
git mv deploy/k8s/secrets.example.yaml deploy/k8s/base/
```

- [ ] **Step 2: Strip `namespace: atlas` from every Deployment/Service/ConfigMap**

```bash
find deploy/k8s/base -name '*.yaml' -exec \
    sed -i '/^[[:space:]]*namespace: atlas$/d' {} +
```

- [ ] **Step 3: Verify no `namespace:` is left in base**

```bash
grep -nE '^[[:space:]]*namespace:' deploy/k8s/base/*.yaml
```

Expected: empty output. (The `atlas` Namespace resource itself doesn't have `metadata.namespace`; it has `metadata.name: atlas`. That stays.)

- [ ] **Step 4: Create `deploy/k8s/base/kustomization.yaml`**

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  - namespace.yaml
  - env-configmap.yaml
  - atlas-account.yaml
  - atlas-asset-expiration.yaml
  - atlas-assets.yaml
  - atlas-ban.yaml
  - atlas-buddies.yaml
  - atlas-buffs.yaml
  - atlas-cashshop.yaml
  - atlas-chairs.yaml
  - atlas-chalkboards.yaml
  - atlas-channel.yaml
  - atlas-character-factory.yaml
  - atlas-character.yaml
  - atlas-configurations.yaml
  - atlas-consumables.yaml
  - atlas-data.yaml
  - atlas-drop-information.yaml
  - atlas-drops.yaml
  - atlas-effective-stats.yaml
  - atlas-expressions.yaml
  - atlas-fame.yaml
  - atlas-gachapons.yaml
  - atlas-guilds.yaml
  - atlas-ingress.yaml
  - atlas-inventory.yaml
  - atlas-invites.yaml
  - atlas-keys.yaml
  - atlas-login.yaml
  - atlas-map-actions.yaml
  - atlas-maps.yaml
  - atlas-merchant.yaml
  - atlas-messages.yaml
  - atlas-messengers.yaml
  - atlas-monster-death.yaml
  - atlas-monsters.yaml
  - atlas-notes.yaml
  - atlas-npc-conversations.yaml
  - atlas-npc-shops.yaml
  - atlas-parties.yaml
  - atlas-party-quests.yaml
  - atlas-pets.yaml
  - atlas-portal-actions.yaml
  - atlas-portals.yaml
  - atlas-query-aggregator.yaml
  - atlas-quest.yaml
  - atlas-rates.yaml
  - atlas-reactor-actions.yaml
  - atlas-reactors.yaml
  - atlas-saga-orchestrator.yaml
  - atlas-skills.yaml
  - atlas-storage.yaml
  - atlas-tenants.yaml
  - atlas-transports.yaml
  - atlas-ui.yaml
  - atlas-world.yaml
  - atlas-wz-extractor.yaml
```

(Adjust to match the actual `ls deploy/k8s/base/*.yaml` minus `secrets.example.yaml`, which is documentation-only.)

- [ ] **Step 5: Verify base renders**

```bash
kustomize build deploy/k8s/base > /tmp/base.yaml
grep -c '^kind:' /tmp/base.yaml
```

Expected: same number of resources as the pre-move flat directory.

- [ ] **Step 6: Commit**

```bash
git add deploy/k8s/
git commit -m "refactor(deploy): move flat manifests into Kustomize base

Moves every deploy/k8s/atlas-*.yaml into deploy/k8s/base/ and strips
the hardcoded 'namespace: atlas' from each Deployment/Service so
overlays can inject the namespace. Renames ingress.yaml ->
atlas-ingress.yaml for consistency. No semantic change to the rendered
output of the main env.

Refs task-063."
```

### Task 7.2: Create the `main` overlay

- [ ] **Step 1: Create `deploy/k8s/overlays/main/kustomization.yaml`**

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: atlas
resources:
  - ../../base

# Pin every service image to :latest. main env is unsuffixed.
images:
  - name: ghcr.io/chronicle20/atlas-account/atlas-account
    newTag: latest
  - name: ghcr.io/chronicle20/atlas-asset-expiration/atlas-asset-expiration
    newTag: latest
  # … one entry per service. Generate from .github/config/services.json:
  #   jq -r '.services[] | select(.docker_image) | "  - name: \(.docker_image)\n    newTag: latest"'
  #   < .github/config/services.json
```

Generate the full list mechanically — script it into `deploy/k8s/overlays/main/kustomization.yaml`:

```bash
{
    cat <<EOF
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: atlas
resources:
  - ../../base

images:
EOF
    jq -r '.services[] | select(.docker_image) | "  - name: \(.docker_image)\n    newTag: latest"' \
        < .github/config/services.json
    cat <<EOF

commonLabels:
  atlas.env: main
EOF
} > deploy/k8s/overlays/main/kustomization.yaml
```

- [ ] **Step 2: Verify the overlay renders identical to current cluster state**

```bash
kustomize build deploy/k8s/overlays/main > /tmp/main-rendered.yaml
kubectl get -n atlas all,configmap -o yaml > /tmp/cluster-current.yaml

# Compare resource counts
grep -c '^kind:' /tmp/main-rendered.yaml
grep -c '^  kind:' /tmp/cluster-current.yaml
```

Expected: counts match (modulo Kubernetes-injected fields like `creationTimestamp`, `resourceVersion`, etc.). For a finer-grained diff, normalize with `yq`:

```bash
yq eval-all 'select(fileIndex == 0) - select(fileIndex == 1)' /tmp/main-rendered.yaml /tmp/cluster-current.yaml
```

Expected: only Kustomize-injected labels (`atlas.env: main`, `app.kubernetes.io/managed-by: kustomize`) are net new.

- [ ] **Step 3: Commit**

```bash
git add deploy/k8s/overlays/main/
git commit -m "feat(deploy): add Kustomize overlay for the main env

Renders to the same set of resources currently live in the atlas
namespace plus the 'atlas.env: main' label. Argo CD's Application(atlas-main)
syncs from this path.

Refs task-063."
```

### Task 7.3: Generate the per-PR consumer-group patch

This patch ships in the PR overlay (Task 7.4). Generate it now from the consumer-group literals so the script is committed for re-use.

- [ ] **Step 1: Create `deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh`**

```bash
#!/usr/bin/env bash
# Generates patches/consumer-group-env.yaml from the literal consumer
# group strings declared in each service's main.go.
#
# Output: a list of strategic-merge patches, one per Deployment, each
# adding KAFKA_CONSUMER_GROUP="<literal> [PLACEHOLDER_ATLAS_ENV]" to the
# container env. The PLACEHOLDER is rewritten by the kustomization's
# replacements: rule.

set -euo pipefail

ROOT="$(git rev-parse --show-toplevel)"
OUT="$ROOT/deploy/k8s/overlays/pr/patches/consumer-group-env.yaml"
mkdir -p "$(dirname "$OUT")"

{
    echo '# Generated by deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh'
    echo '# Do not edit by hand. Re-run after adding/removing services.'
    echo ''

    while IFS= read -r f; do
        svc=$(grep -E 'serviceName *= *"atlas-' "$f" | head -1 | sed -E 's/.*"atlas-([^"]*)".*/\1/')
        depname="atlas-$svc"

        literal=$(grep -E 'consumerGroupId(Template)? *= *"' "$f" | head -1 | sed -E 's/.*"([^"]*)".*/\1/')
        if [ -z "$literal" ]; then
            continue
        fi

        cat <<EOF
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: $depname
spec:
  template:
    spec:
      containers:
        - name: ${svc%%-*}
          env:
            - name: KAFKA_CONSUMER_GROUP
              value: "$literal [PLACEHOLDER_ATLAS_ENV]"
            - name: ATLAS_ENV
              value: "PLACEHOLDER_ATLAS_ENV"
EOF
    done < <(find services/atlas-*/atlas.com -maxdepth 2 -name 'main.go' | sort)
} > "$OUT"

echo "Wrote $(wc -l < "$OUT") lines to $OUT"
```

> **NOTE:** the `containers[].name` in each manifest is NOT always `<svc>` — for atlas-account, it's `account`; for atlas-character, it's `character`; for atlas-pr-bootstrap, no Deployment. The script's `${svc%%-*}` heuristic works for short service names but will be wrong for `atlas-asset-expiration` etc. **Decision:** rather than guess, the script reads the Deployment YAML in `deploy/k8s/base/` to extract the actual container name:

Replace the script's container-name extraction with:

```bash
# Read the actual container name from the base Deployment manifest.
cname=$(yq eval '.spec.template.spec.containers[0].name' "$ROOT/deploy/k8s/base/$depname.yaml")
```

(Requires `yq` 4+; install via `apk add yq` if missing.)

- [ ] **Step 2: Run it**

```bash
chmod +x deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh
./deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh
```

Expected: writes ~50 strategic-merge entries to `deploy/k8s/overlays/pr/patches/consumer-group-env.yaml`.

- [ ] **Step 3: Commit script and generated output**

```bash
git add deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh \
        deploy/k8s/overlays/pr/patches/consumer-group-env.yaml
git commit -m "feat(deploy): generator for per-PR consumer-group env patch

Produces deploy/k8s/overlays/pr/patches/consumer-group-env.yaml from
each service's consumerGroupId literal. Committed alongside the output
so adding a new service is one regenerate-and-commit step.

Refs task-063."
```

### Task 7.4: Create the per-PR overlay shell

- [ ] **Step 1: Create `deploy/k8s/overlays/pr/atlas-env-tokens.yaml`**

```yaml
# A tiny ConfigMap that carries the per-env token. Both the main
# manifests (via kustomization.replacements) and the Argo hook Jobs
# (via envFrom) read from here.
apiVersion: v1
kind: ConfigMap
metadata:
  name: atlas-env-tokens
data:
  ATLAS_ENV: "PLACEHOLDER_ATLAS_ENV"
```

- [ ] **Step 2: Create `deploy/k8s/overlays/pr/patches/db-name-suffix.yaml`**

```yaml
# Generated alongside consumer-group-env.yaml — same generator script
# reads each base manifest's existing DB_NAME env var and emits a
# patch suffixing it with -PLACEHOLDER_ATLAS_ENV.
```

> Extend the gen script in 7.3 to write `db-name-suffix.yaml` too. Each entry:

```yaml
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: atlas-<svc>
spec:
  template:
    spec:
      containers:
        - name: <container>
          env:
            - name: DB_NAME
              value: "<original>-PLACEHOLDER_ATLAS_ENV"
```

(Original DB_NAME is read from `deploy/k8s/base/atlas-<svc>.yaml` via `yq`.)

- [ ] **Step 3: Create `deploy/k8s/overlays/pr/kustomization.yaml`**

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

# namespace is rewritten by the ApplicationSet to atlas-pr-<N>
namespace: atlas-pr-PLACEHOLDER_PR_NUMBER

resources:
  - ../../base
  - atlas-env-tokens.yaml
  - ingress-route.yaml
  - presync-create-dbs.yaml
  - postsync-bootstrap.yaml
  - postsync-pihole-add.yaml
  - postdelete-cleanup.yaml

patches:
  - path: patches/db-name-suffix.yaml
  - path: patches/consumer-group-env.yaml

# Per-env atlas-env ConfigMap with topic suffixing.
configMapGenerator:
  - name: atlas-env
    behavior: replace
    literals:
      # Infrastructure (unchanged)
      - BASE_SERVICE_URL=http://atlas-ingress.atlas-pr-PLACEHOLDER_PR_NUMBER.svc.cluster.local:80/api/
      - BOOTSTRAP_SERVERS=kafka.home:9093
      - DB_HOST=postgres.home
      - DB_PORT=5432
      - REDIS_URL=redis.home:6379
      - TRACE_ENDPOINT=tempo.home:4317
      - TRACE_SAMPLING_RATIO=1.0
      - REST_PORT=8080
      # Topic env vars suffixed -PLACEHOLDER_ATLAS_ENV
      - COMMAND_TOPIC_ACCOUNT=COMMAND_TOPIC_ACCOUNT-PLACEHOLDER_ATLAS_ENV
      - COMMAND_TOPIC_ACCOUNT_LOGOUT=COMMAND_TOPIC_ACCOUNT_LOGOUT-PLACEHOLDER_ATLAS_ENV
      # … every other COMMAND_TOPIC_* and EVENT_TOPIC_* — generate from base/env-configmap.yaml.
      # Use deploy/k8s/overlays/pr/scripts/gen-topic-config.sh.

# Token replacement: ATLAS_ENV from atlas-env-tokens flows into every
# PLACEHOLDER_ATLAS_ENV slot in the rendered manifest set.
replacements:
  - source:
      kind: ConfigMap
      name: atlas-env-tokens
      fieldPath: data.ATLAS_ENV
    targets:
      - select:
          kind: Deployment
        fieldPaths:
          - spec.template.spec.containers.*.env.[name=ATLAS_ENV].value
          - spec.template.spec.containers.*.env.[name=KAFKA_CONSUMER_GROUP].value
          - spec.template.spec.containers.*.env.[name=DB_NAME].value
        options:
          create: false
      - select:
          kind: ConfigMap
          name: atlas-env
        fieldPaths:
          - data.[COMMAND_TOPIC_*]
          - data.[EVENT_TOPIC_*]

images:
  - name: ghcr.io/chronicle20/atlas-account/atlas-account
    newTag: pr-PLACEHOLDER_PR_NUMBER-PLACEHOLDER_SHA
  # … rewritten by the ApplicationSet for every service.

commonLabels:
  atlas.env: PLACEHOLDER_ATLAS_ENV
  atlas.pr-number: PLACEHOLDER_PR_NUMBER
```

- [ ] **Step 4: Add the topic-config generator script**

`deploy/k8s/overlays/pr/scripts/gen-topic-config.sh`:

```bash
#!/usr/bin/env bash
set -euo pipefail
ROOT="$(git rev-parse --show-toplevel)"
{
    yq -r '.data | to_entries | .[] | select(.key | test("^(COMMAND|EVENT)_TOPIC_")) | "      - " + .key + "=" + .value + "-PLACEHOLDER_ATLAS_ENV"' \
        "$ROOT/deploy/k8s/base/env-configmap.yaml"
} > "$ROOT/deploy/k8s/overlays/pr/_topics.snippet"
echo "Wrote topic snippet — paste into kustomization.yaml's literals: block."
```

- [ ] **Step 5: Run the generator and inline the topic literals**

Manually copy the generated snippet under the `# … every other …` placeholder in `deploy/k8s/overlays/pr/kustomization.yaml`. Once the literals: block is complete, delete the snippet file:

```bash
./deploy/k8s/overlays/pr/scripts/gen-topic-config.sh
# (manually edit kustomization.yaml to inline the contents of _topics.snippet)
rm deploy/k8s/overlays/pr/_topics.snippet
```

- [ ] **Step 6: Commit**

```bash
git add deploy/k8s/overlays/pr/
git commit -m "feat(deploy): per-PR Kustomize overlay shell

Adds the overlay skeleton: atlas-env-tokens ConfigMap, kustomization.yaml
with replacements/configMapGenerator wiring, and the gen-topic-config.sh
generator. PLACEHOLDER_ATLAS_ENV / PLACEHOLDER_PR_NUMBER / PLACEHOLDER_SHA
slots are rewritten by the Argo CD ApplicationSet at sync time.

Refs task-063."
```

### Task 7.5: Add the IngressRoute

- [ ] **Step 1: Create `deploy/k8s/overlays/pr/ingress-route.yaml`**

```yaml
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: atlas-pr-ingress
spec:
  entryPoints:
    - web
  routes:
    - match: Host(`PLACEHOLDER_PR_NUMBER.atlas.home`)
      kind: Rule
      services:
        - name: atlas-ingress
          port: 80
```

- [ ] **Step 2: Render the overlay locally to confirm the route appears**

```bash
kustomize build deploy/k8s/overlays/pr | grep -A4 'kind: IngressRoute'
```

- [ ] **Step 3: Commit**

```bash
git add deploy/k8s/overlays/pr/ingress-route.yaml
git commit -m "feat(deploy): per-PR Traefik IngressRoute

Adds <PR_NUMBER>.atlas.home routing to the namespace's atlas-ingress.

Refs task-063."
```

### Task 7.6: PreSync hook — create per-env Postgres DBs

- [ ] **Step 1: Create `deploy/k8s/overlays/pr/presync-create-dbs.yaml`**

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: atlas-pr-create-dbs
  annotations:
    argocd.argoproj.io/hook: PreSync
    argocd.argoproj.io/hook-delete-policy: HookSucceeded
spec:
  backoffLimit: 3
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: psql
          image: postgres:15-alpine
          envFrom:
            - configMapRef:
                name: atlas-env-tokens
            - configMapRef:
                name: atlas-db-names
            - secretRef:
                name: db-credentials
          env:
            - name: DB_HOST
              valueFrom:
                configMapKeyRef:
                  name: atlas-env
                  key: DB_HOST
            - name: DB_PORT
              valueFrom:
                configMapKeyRef:
                  name: atlas-env
                  key: DB_PORT
          command:
            - /bin/sh
            - -c
            - |
              set -e
              for raw in $ATLAS_DB_NAMES; do
                  full="${raw}-${ATLAS_ENV}"
                  PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d postgres \
                      -v "ON_ERROR_STOP=1" \
                      -c "SELECT 'CREATE DATABASE \"'||'$full'||'\"' WHERE NOT EXISTS (SELECT 1 FROM pg_database WHERE datname = '$full')\\gexec"
              done
```

- [ ] **Step 2: Create the `atlas-db-names` ConfigMap as a `configMapGenerator` entry**

In `deploy/k8s/overlays/pr/kustomization.yaml`, append to the `configMapGenerator:` block:

```yaml
  - name: atlas-db-names
    literals:
      - ATLAS_DB_NAMES=atlas-accounts atlas-asset-expiration atlas-bans atlas-buddies atlas-buffs atlas-cashshop atlas-chairs atlas-chalkboards atlas-channel atlas-character atlas-character-factory atlas-configurations atlas-consumables atlas-data atlas-drop-information atlas-drops atlas-effective-stats atlas-expressions atlas-fame atlas-families atlas-gachapons atlas-guilds atlas-inventory atlas-invites atlas-keys atlas-login atlas-map-actions atlas-maps atlas-marriages atlas-merchant atlas-messages atlas-messengers atlas-monster-death atlas-monsters atlas-notes atlas-npc-conversations atlas-npc-shops atlas-parties atlas-party-quests atlas-pets atlas-portal-actions atlas-portals atlas-quest atlas-rates atlas-reactor-actions atlas-reactors atlas-saga-orchestrator atlas-skills atlas-storage atlas-tenants atlas-transports atlas-world
```

> **NOTE:** the canonical list of base DB names lives in this generator entry. Each value matches the `DB_NAME` value currently set in the corresponding `deploy/k8s/base/atlas-<svc>.yaml`. Verify by `grep -A1 'name: DB_NAME' deploy/k8s/base/*.yaml`. Services without DB_NAME (atlas-assets, atlas-ui, atlas-ingress, atlas-query-aggregator, atlas-wz-extractor, atlas-pr-bootstrap) are excluded.

- [ ] **Step 3: Commit**

```bash
git add deploy/k8s/overlays/pr/presync-create-dbs.yaml \
        deploy/k8s/overlays/pr/kustomization.yaml
git commit -m "feat(deploy): PreSync Job that creates per-env Postgres DBs

Idempotent CREATE DATABASE for every entry in atlas-db-names suffixed
with -<ATLAS_ENV>. Runs before service Deployments cold-start so GORM
AutoMigrate can populate empty schemas.

Refs task-063."
```

### Task 7.7: PostSync hook — bootstrap

- [ ] **Step 1: Create `deploy/k8s/overlays/pr/postsync-bootstrap.yaml`**

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: atlas-pr-bootstrap
  annotations:
    argocd.argoproj.io/hook: PostSync
    argocd.argoproj.io/hook-delete-policy: HookSucceeded
spec:
  backoffLimit: 3
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: bootstrap
          image: ghcr.io/chronicle20/atlas-pr-bootstrap/atlas-pr-bootstrap:latest
          command: ["/atlas/bootstrap.sh"]
          env:
            - name: ATLAS_UI_BASE
              value: http://atlas-ingress.atlas-pr-PLACEHOLDER_PR_NUMBER.svc.cluster.local
            - name: TENANT_ID
              valueFrom:
                configMapKeyRef:
                  name: atlas-pr-bootstrap-tenant
                  key: TENANT_ID
            - name: REGION
              valueFrom:
                configMapKeyRef:
                  name: atlas-pr-bootstrap-tenant
                  key: REGION
            - name: MAJOR_VERSION
              valueFrom:
                configMapKeyRef:
                  name: atlas-pr-bootstrap-tenant
                  key: MAJOR_VERSION
            - name: MINOR_VERSION
              valueFrom:
                configMapKeyRef:
                  name: atlas-pr-bootstrap-tenant
                  key: MINOR_VERSION
          envFrom:
            - configMapRef:
                name: atlas-env-tokens
          volumeMounts:
            - name: wz-canonical
              mountPath: /opt/wz
              readOnly: true
      volumes:
        - name: wz-canonical
          persistentVolumeClaim:
            claimName: atlas-wz-canonical-readonly
```

- [ ] **Step 2: Add the `atlas-pr-bootstrap-tenant` ConfigMap to the overlay**

In `deploy/k8s/overlays/pr/kustomization.yaml` `configMapGenerator:`:

```yaml
  - name: atlas-pr-bootstrap-tenant
    literals:
      # The bootstrap Job's tenant headers. Matches the canonical
      # bootstrap tenant. If the canonical tenant isn't created until
      # after bootstrap (chicken/egg), the job is bound to a known
      # default tenant_id used by atlas-tenants' first-run seed.
      - TENANT_ID=00000000-0000-0000-0000-000000000001
      - REGION=GMS
      - MAJOR_VERSION=83
      - MINOR_VERSION=1
```

> **NOTE:** the actual canonical tenant id/region/version belong in `preflight.md` after Task 0.x — verify against atlas-tenants' first-run seed to lock the values. If atlas-tenants needs explicit seeding before bootstrap can run, add a presync-create-tenant.yaml hook that POSTs the tenant before the bootstrap Job starts.

- [ ] **Step 3: Commit**

```bash
git add deploy/k8s/overlays/pr/postsync-bootstrap.yaml \
        deploy/k8s/overlays/pr/kustomization.yaml
git commit -m "feat(deploy): PostSync bootstrap Job

Runs atlas-pr-bootstrap (the support image from Phase 6) against the
namespace's atlas-ingress. Mounts atlas-wz-canonical PVC for the WZ zip.

Refs task-063."
```

### Task 7.8: PostSync hook — Pi-hole DNS register

- [ ] **Step 1: Create `deploy/k8s/overlays/pr/postsync-pihole-add.yaml`**

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: atlas-pr-pihole-register
  annotations:
    argocd.argoproj.io/hook: PostSync
    argocd.argoproj.io/hook-delete-policy: HookSucceeded
    argocd.argoproj.io/sync-wave: "10"
spec:
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: pihole-register
          image: curlimages/curl:8.10.1
          command: ["/bin/sh", "-c"]
          args:
            - |
              set -eu
              host="${PR_NUMBER}.atlas.home"
              ok=0
              for i in 1 2; do
                  base_var="PIHOLE_API_BASE_${i}"
                  token_var="PIHOLE_TOKEN_${i}"
                  base=$(eval echo \$${base_var})
                  token=$(eval echo \$${token_var})
                  [ -z "$base" ] && continue
                  if curl -fsS -X POST \
                          -H "X-Pi-Auth: $token" \
                          -H "Content-Type: application/json" \
                          -d "{\"name\":\"$host\",\"address\":\"$TRAEFIK_LB_IP\"}" \
                          "$base/api/config/dns/hosts"; then
                      ok=$((ok+1))
                  fi
              done
              [ "$ok" -ge 1 ] || exit 1
          envFrom:
            - configMapRef:
                name: atlas-env-tokens
            - secretRef:
                name: pihole-credentials
          env:
            - name: PR_NUMBER
              value: "PLACEHOLDER_PR_NUMBER"
            - name: TRAEFIK_LB_IP
              value: "192.168.23.230"
```

- [ ] **Step 2: Commit**

```bash
git add deploy/k8s/overlays/pr/postsync-pihole-add.yaml
git commit -m "feat(deploy): PostSync Pi-hole DNS-register Job

Adds <PR>.atlas.home A records on both Pi-hole servers. Tolerates
single-server failure; fails only if both Pi-holes reject the request.

Refs task-063."
```

### Task 7.9: PostDelete cleanup hook

- [ ] **Step 1: Create `deploy/k8s/overlays/pr/postdelete-cleanup.yaml`**

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: atlas-pr-cleanup
  annotations:
    argocd.argoproj.io/hook: PostDelete
    argocd.argoproj.io/hook-delete-policy: HookSucceeded
spec:
  backoffLimit: 0
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: cleanup
          image: ghcr.io/chronicle20/atlas-pr-bootstrap/atlas-pr-bootstrap:latest
          command: ["/atlas/cleanup.sh"]
          envFrom:
            - configMapRef:
                name: atlas-env-tokens
            - configMapRef:
                name: atlas-db-names
            - secretRef:
                name: db-credentials
            - secretRef:
                name: pihole-credentials
            - secretRef:
                name: ghcr-pat
          env:
            - name: DB_HOST
              valueFrom:
                configMapKeyRef:
                  name: atlas-env
                  key: DB_HOST
            - name: BOOTSTRAP_SERVERS
              valueFrom:
                configMapKeyRef:
                  name: atlas-env
                  key: BOOTSTRAP_SERVERS
            - name: REDIS_URL
              valueFrom:
                configMapKeyRef:
                  name: atlas-env
                  key: REDIS_URL
            - name: PR_NUMBER
              value: "PLACEHOLDER_PR_NUMBER"
            - name: ATLAS_SERVICES
              value: "atlas-account,atlas-asset-expiration,atlas-ban,..."  # see Step 2
```

- [ ] **Step 2: Generate the `ATLAS_SERVICES` literal**

```bash
jq -r '[.services[] | select(.docker_image) | .name] | join(",")' \
    < .github/config/services.json
```

Paste the output into the `ATLAS_SERVICES` env value verbatim.

- [ ] **Step 3: Commit**

```bash
git add deploy/k8s/overlays/pr/postdelete-cleanup.yaml
git commit -m "feat(deploy): PostDelete cleanup Job

Drops per-env Postgres DBs, deletes Kafka topics and consumer groups,
clears Redis keys, removes ghcr image tags, and removes Pi-hole DNS.
backoffLimit: 0 so a failure leaves the env in cleanup-failed for
manual investigation.

Refs task-063."
```

### Task 7.10: Render and verify the PR overlay

- [ ] **Step 1: Render with PLACEHOLDER values**

```bash
kustomize build deploy/k8s/overlays/pr > /tmp/pr-rendered.yaml
```

Expected: completes without error. Resources:

```bash
grep -c '^kind:' /tmp/pr-rendered.yaml
```

Expected: roughly base count (~120) plus 4 hook Jobs plus 1 IngressRoute plus 1 atlas-env-tokens ConfigMap.

- [ ] **Step 2: Verify replacements would apply**

```bash
grep -c PLACEHOLDER_ATLAS_ENV /tmp/pr-rendered.yaml
grep -c PLACEHOLDER_PR_NUMBER /tmp/pr-rendered.yaml
grep -c PLACEHOLDER_SHA /tmp/pr-rendered.yaml
```

Expected: all three counts are non-zero (these are slots Argo CD's per-PR Application overrides via `replacements:` and `images:`).

- [ ] **Step 3: Lint with `kubeconform`**

```bash
kustomize build deploy/k8s/overlays/pr | kubeconform -strict -summary -kubernetes-version master
```

Expected: every resource validates against its schema. (Traefik/Argo CRDs require `-skip` flags or a CRD schema bundle; if not available locally, accept "schema not found" warnings for those kinds.)

- [ ] **Step 4: No commit** — proof is the green render.

---

## Phase 8: Argo CD bee-repo manifests (staged in this repo)

These belong in `tumidanski/k3s` but are committed here at `deploy/argocd-bee/` for version control. The runbook (Phase 10) tells the maintainer to copy them into the bee repo and apply.

**Files:**
- `deploy/argocd-bee/argocd.yml` — Argo install with patches
- `deploy/argocd-bee/argocd-atlas-main.yml`
- `deploy/argocd-bee/argocd-atlas-pr.yml`
- `deploy/argocd-bee/argocd-cleanup-cronjob.yml`
- `deploy/argocd-bee/argocd-pihole-secret.yml.example`
- `deploy/argocd-bee/argocd-ghcr-secret.yml.example`
- `deploy/argocd-bee/README.md`

### Task 8.1: Argo CD install manifest

- [ ] **Step 1: Render and commit `deploy/argocd-bee/argocd.yml`**

```bash
mkdir -p deploy/argocd-bee
curl -fsSL https://raw.githubusercontent.com/argoproj/argo-cd/v2.13.0/manifests/install.yaml \
    > deploy/argocd-bee/argocd-upstream.yml
```

Copy `argocd-upstream.yml` to `argocd.yml` and apply patches:

1. Add `--insecure` to the `argocd-server` deployment's `command:` list (Traefik terminates HTTP at the edge).
2. Append the Traefik `IngressRoute` for `argocd.bee.tumidanski`:

```yaml
---
apiVersion: traefik.io/v1alpha1
kind: IngressRoute
metadata:
  name: argocd
  namespace: argocd
spec:
  entryPoints:
    - web
  routes:
    - match: Host(`argocd.bee.tumidanski`)
      kind: Rule
      services:
        - name: argocd-server
          port: 80
```

- [ ] **Step 2: Delete the upstream copy after extracting**

```bash
rm deploy/argocd-bee/argocd-upstream.yml
```

- [ ] **Step 3: Commit**

```bash
git add deploy/argocd-bee/argocd.yml
git commit -m "chore(deploy): stage Argo CD install manifest for bee

Renders Argo CD v2.13.0 with --insecure and a Traefik IngressRoute at
argocd.bee.tumidanski. Maintainer copies this to tumidanski/k3s/bee/
and applies via kubectl. See deploy/argocd-bee/README.md.

Refs task-063."
```

### Task 8.2: Application(atlas-main)

- [ ] **Step 1: Create `deploy/argocd-bee/argocd-atlas-main.yml`**

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: atlas-main
  namespace: argocd
spec:
  project: default
  source:
    repoURL: https://github.com/Chronicle20/atlas.git
    targetRevision: main
    path: deploy/k8s/overlays/main
  destination:
    server: https://kubernetes.default.svc
    namespace: atlas
  syncPolicy:
    automated:
      selfHeal: true
      prune: false  # enable after stability window (see runbook §9)
    syncOptions:
      - ServerSideApply=true
      - CreateNamespace=false
```

- [ ] **Step 2: Commit**

```bash
git add deploy/argocd-bee/argocd-atlas-main.yml
git commit -m "chore(deploy): stage Application(atlas-main) for bee

Argo CD Application that auto-syncs deploy/k8s/overlays/main from
main branch into the atlas namespace. prune disabled until the
stability window completes; see runbook §9.

Refs task-063."
```

### Task 8.3: ApplicationSet(atlas-pr)

- [ ] **Step 1: Create `deploy/argocd-bee/argocd-atlas-pr.yml`**

```yaml
apiVersion: argoproj.io/v1alpha1
kind: ApplicationSet
metadata:
  name: atlas-pr
  namespace: argocd
spec:
  goTemplate: true
  goTemplateOptions: ["missingkey=error"]
  generators:
    - pullRequest:
        github:
          owner: Chronicle20
          repo: atlas
          tokenRef:
            secretName: argocd-repo-creds-chronicle20-atlas
            key: password
        requeueAfterSeconds: 30
  template:
    metadata:
      name: 'atlas-pr-{{.number}}'
      annotations:
        atlas.env: '{{ printf "%.4s" (sha256sum (printf "pr-%d" .number)) }}'
        atlas.pr-number: '{{.number}}'
        atlas.cleanup-grace: '24h'
        atlas.head-sha: '{{.head_sha}}'
    spec:
      project: default
      source:
        repoURL: https://github.com/Chronicle20/atlas.git
        targetRevision: '{{.head_sha}}'
        path: deploy/k8s/overlays/pr
        kustomize:
          commonAnnotations:
            atlas.env: '{{ printf "%.4s" (sha256sum (printf "pr-%d" .number)) }}'
          replacements:
            - source:
                value: '{{ printf "%.4s" (sha256sum (printf "pr-%d" .number)) }}'
              targets:
                - select:
                    kind: ConfigMap
                    name: atlas-env-tokens
                  fieldPaths:
                    - data.ATLAS_ENV
            - source:
                value: '{{.number}}'
              targets:
                - select:
                    kind: Namespace
                  fieldPaths:
                    - metadata.name
                # Plus IngressRoute, Job env, etc — covered by overlay's own replacements
          commonLabels:
            atlas.env: '{{ printf "%.4s" (sha256sum (printf "pr-%d" .number)) }}'
            atlas.pr-number: '{{.number}}'
          # images mapping rewritten per service-built list — see runbook §9.7
      destination:
        server: https://kubernetes.default.svc
        namespace: 'atlas-pr-{{.number}}'
      syncPolicy:
        automated:
          selfHeal: true
          prune: true
        syncOptions:
          - ServerSideApply=true
          - CreateNamespace=true
```

- [ ] **Step 2: Commit**

```bash
git add deploy/argocd-bee/argocd-atlas-pr.yml
git commit -m "chore(deploy): stage ApplicationSet(atlas-pr) for bee

GitHub PR generator polls Chronicle20/atlas every 30s and emits one
Application per open PR. ATLAS_ENV is computed deterministically as
sha256(\"pr-<N>\")[:4] via Argo's goTemplate sha256sum helper.

Refs task-063."
```

### Task 8.4: Cleanup CronJob

- [ ] **Step 1: Create `deploy/argocd-bee/argocd-cleanup-cronjob.yml`**

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: atlas-pr-cleanup
  namespace: argocd
spec:
  schedule: "5 * * * *"  # hourly at :05
  concurrencyPolicy: Forbid
  jobTemplate:
    spec:
      backoffLimit: 0
      template:
        spec:
          restartPolicy: Never
          serviceAccountName: atlas-pr-cleanup
          containers:
            - name: scan
              image: bitnami/kubectl:1.30
              command: ["/bin/bash", "-c"]
              args:
                - |
                  set -euo pipefail
                  PAT=$(cat /etc/github/token)
                  kubectl get applications -n argocd -l atlas.pr-number -o json \
                    | jq -c '.items[]' | while read -r app; do
                      name=$(echo "$app" | jq -r '.metadata.name')
                      pr=$(echo "$app" | jq -r '.metadata.annotations["atlas.pr-number"]')
                      grace=$(echo "$app" | jq -r '.metadata.annotations["atlas.cleanup-grace"] // "24h"')
                      deadline=$(echo "$app" | jq -r '.metadata.annotations["atlas.cleanup-deadline"] // ""')

                      state=$(curl -fsS -H "Authorization: Bearer $PAT" \
                        "https://api.github.com/repos/Chronicle20/atlas/pulls/$pr" \
                        | jq -r .state)

                      if [ "$state" = "open" ]; then
                          # PR re-opened or still open — clear deadline if set
                          if [ -n "$deadline" ]; then
                              kubectl annotate -n argocd application "$name" atlas.cleanup-deadline-
                          fi
                          continue
                      fi

                      if [ -z "$deadline" ]; then
                          # First time we noticed the PR is closed — set deadline
                          new=$(date -u -d "+${grace/h/ hours}" +%Y-%m-%dT%H:%M:%SZ)
                          kubectl annotate -n argocd application "$name" "atlas.cleanup-deadline=$new" --overwrite
                          continue
                      fi

                      now=$(date -u +%s)
                      due=$(date -u -d "$deadline" +%s)
                      if [ "$now" -ge "$due" ]; then
                          kubectl delete application -n argocd "$name"
                      fi
                  done
              volumeMounts:
                - name: github-token
                  mountPath: /etc/github
                  readOnly: true
          volumes:
            - name: github-token
              secret:
                secretName: argocd-repo-creds-chronicle20-atlas
                items:
                  - key: password
                    path: token
---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: atlas-pr-cleanup
  namespace: argocd
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  namespace: argocd
  name: atlas-pr-cleanup
rules:
  - apiGroups: ["argoproj.io"]
    resources: ["applications"]
    verbs: ["get", "list", "delete", "patch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: atlas-pr-cleanup
  namespace: argocd
roleBinding: {}
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: atlas-pr-cleanup
subjects:
  - kind: ServiceAccount
    name: atlas-pr-cleanup
    namespace: argocd
```

- [ ] **Step 2: Commit**

```bash
git add deploy/argocd-bee/argocd-cleanup-cronjob.yml
git commit -m "chore(deploy): stage cleanup CronJob for bee

Hourly job that polls each atlas-pr-* Application's GitHub state. On
PR close, sets cleanup-deadline; on grace expiry, deletes the
Application (which fires PostDelete cleanup hook).

Refs task-063."
```

### Task 8.5: Secret examples

- [ ] **Step 1: Create `deploy/argocd-bee/argocd-pihole-secret.yml.example`**

```yaml
# DO NOT COMMIT REAL VALUES.
# Apply via sealed-secrets or a one-time `kubectl apply -f`.
# This Secret is replicated into every atlas-pr-* namespace by the
# overlay (see deploy/k8s/overlays/pr/kustomization.yaml).
apiVersion: v1
kind: Secret
metadata:
  name: pihole-credentials
  namespace: argocd
type: Opaque
stringData:
  PIHOLE_API_BASE_1: "http://pihole-1.home/admin"
  PIHOLE_TOKEN_1: "REPLACE"
  PIHOLE_API_BASE_2: "http://pihole-2.home/admin"
  PIHOLE_TOKEN_2: "REPLACE"
```

- [ ] **Step 2: Create `deploy/argocd-bee/argocd-ghcr-secret.yml.example`**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: ghcr-pat
  namespace: argocd
type: Opaque
stringData:
  GHCR_TOKEN: "REPLACE"
```

- [ ] **Step 3: Commit (no real secrets)**

```bash
git add deploy/argocd-bee/argocd-pihole-secret.yml.example \
        deploy/argocd-bee/argocd-ghcr-secret.yml.example
git commit -m "chore(deploy): stage Pi-hole and ghcr secret examples for bee

Refs task-063."
```

### Task 8.6: README for bee delivery

- [ ] **Step 1: Create `deploy/argocd-bee/README.md`**

```markdown
# Argo CD bee-cluster artifacts

Files in this directory are version-controlled here but **belong in
`tumidanski/k3s`** under `bee/`. The Atlas repo carries them so the
manifests that drive Atlas live alongside the Atlas code that drives
them.

## Initial setup (one-time)

1. Install Argo CD on bee:
   ```sh
   cp argocd.yml ~/source/k3s/bee/argocd.yml
   kubectl apply -f ~/source/k3s/bee/argocd.yml
   ```

2. Provision repo credentials. Generate a fine-scoped GitHub PAT (read
   access to Chronicle20/atlas only). Apply:
   ```sh
   kubectl create secret generic argocd-repo-creds-chronicle20-atlas \
     --namespace argocd \
     --from-literal=type=git \
     --from-literal=url=https://github.com/Chronicle20/atlas.git \
     --from-literal=username=Chronicle20 \
     --from-literal=password=<PAT>
   ```

3. Apply Pi-hole and ghcr secrets (replace placeholders first):
   ```sh
   cp argocd-pihole-secret.yml.example ~/source/k3s/bee/argocd-pihole-secret.yml
   cp argocd-ghcr-secret.yml.example ~/source/k3s/bee/argocd-ghcr-secret.yml
   $EDITOR ~/source/k3s/bee/argocd-pihole-secret.yml
   $EDITOR ~/source/k3s/bee/argocd-ghcr-secret.yml
   kubectl apply -f ~/source/k3s/bee/argocd-pihole-secret.yml
   kubectl apply -f ~/source/k3s/bee/argocd-ghcr-secret.yml
   ```

4. Apply the Application and ApplicationSet:
   ```sh
   cp argocd-atlas-main.yml ~/source/k3s/bee/argocd-atlas-main.yml
   cp argocd-atlas-pr.yml ~/source/k3s/bee/argocd-atlas-pr.yml
   kubectl apply -f ~/source/k3s/bee/argocd-atlas-main.yml
   kubectl apply -f ~/source/k3s/bee/argocd-atlas-pr.yml
   ```

5. Apply the cleanup CronJob:
   ```sh
   cp argocd-cleanup-cronjob.yml ~/source/k3s/bee/argocd-cleanup-cronjob.yml
   kubectl apply -f ~/source/k3s/bee/argocd-cleanup-cronjob.yml
   ```

6. Wait for Application(atlas-main) to report Synced/Healthy with zero
   diffs. If non-zero, see runbook §9.

## Migration cutover

Application(atlas-main) ships with `prune: false` so the initial sync
adopts existing resources without risk of deletion. After ~1 week of
clean syncs, edit `~/source/k3s/bee/argocd-atlas-main.yml` to set
`prune: true` and reapply.

## Updating Argo CD

Re-render the upstream manifest, re-apply the patches, commit to the
Atlas repo (this directory) and to `tumidanski/k3s`:

```sh
curl -fsSL https://raw.githubusercontent.com/argoproj/argo-cd/v2.13.x/manifests/install.yaml \
    > argocd.yml
$EDITOR argocd.yml  # re-apply --insecure and IngressRoute patches
```
```

- [ ] **Step 2: Commit**

```bash
git add deploy/argocd-bee/README.md
git commit -m "docs(deploy): bee-cluster Argo CD delivery README

Refs task-063."
```

---

## Phase 9: GitHub Actions

### Task 9.1: Extend pr-validation.yml with per-PR image build

- [ ] **Step 1: Modify `.github/workflows/pr-validation.yml`**

After the existing `build-docker` job (validation-only), add:

```yaml
  # ============================================
  # Build and Push Per-PR Docker Images
  # ============================================
  build-docker-pr:
    name: Build PR Image - ${{ matrix.service.name }}
    needs: [detect-changes, test-go-services, test-go-libraries, test-ui]
    if: |
      always() &&
      needs.detect-changes.outputs.docker-services-matrix != '[]' &&
      (needs.test-go-services.result == 'success' || needs.test-go-services.result == 'skipped') &&
      (needs.test-go-libraries.result == 'success' || needs.test-go-libraries.result == 'skipped') &&
      (needs.test-ui.result == 'success' || needs.test-ui.result == 'skipped')
    runs-on: ubuntu-latest
    strategy:
      fail-fast: false
      matrix:
        service: ${{ fromJson(needs.detect-changes.outputs.docker-services-matrix) }}

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to GitHub Container Registry
        uses: docker/login-action@v3
        with:
          registry: ghcr.io
          username: ${{ github.actor }}
          password: ${{ secrets.GHCR_TOKEN }}

      - name: Compute short SHA
        id: sha
        run: echo "short=$(git rev-parse --short=7 ${{ github.event.pull_request.head.sha }})" >> $GITHUB_OUTPUT

      - name: Build and push PR image (amd64)
        uses: docker/build-push-action@v6
        with:
          context: ${{ matrix.service.docker_context }}
          file: ${{ matrix.service.path }}/Dockerfile
          platforms: linux/amd64
          push: true
          tags: ${{ matrix.service.docker_image }}:pr-${{ github.event.pull_request.number }}-${{ steps.sha.outputs.short }}
          cache-from: type=gha,scope=${{ matrix.service.name }}-amd64
          cache-to: type=gha,mode=max,scope=${{ matrix.service.name }}-amd64
          provenance: false
          sbom: false
```

> **NOTE:** the `build-docker` validation-only job above this can be left in place (no-push) for fast feedback; `build-docker-pr` does push. The duplication is the cost of preserving the validation-only fast path.

- [ ] **Step 2: Update the `pr-validation-complete` job**

Add `build-docker-pr` to its `needs:` list and to the result table.

- [ ] **Step 3: Lint with `actionlint`**

```bash
actionlint .github/workflows/pr-validation.yml
```

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/pr-validation.yml
git commit -m "ci(pr-validation): build and push per-PR images to ghcr

Adds build-docker-pr job that tags every changed service's image as
pr-<PR>-<sha7> and pushes to ghcr.io. The unchanged services stay on
:latest from main-publish, so the per-PR Argo Application's images:
list overrides only what was rebuilt.

Refs task-063."
```

### Task 9.2: pr-cleanup.yml

- [ ] **Step 1: Create `.github/workflows/pr-cleanup.yml`**

```yaml
name: PR Cleanup

on:
  pull_request:
    types: [closed]
  workflow_dispatch:
    inputs:
      pr-number:
        description: 'PR number to clean up'
        required: true
        type: string

concurrency:
  group: pr-cleanup-${{ github.event.pull_request.number || github.event.inputs.pr-number }}
  cancel-in-progress: false

env:
  PR_NUMBER: ${{ github.event.pull_request.number || github.event.inputs.pr-number }}

jobs:
  delete-images:
    name: Delete per-PR ghcr image tags
    runs-on: ubuntu-latest
    permissions:
      packages: write
    steps:
      - name: Checkout (for services.json)
        uses: actions/checkout@v4

      - name: Compute service list
        id: svc
        run: |
          jq -c '[.services[] | select(.docker_image) | .name]' \
            < .github/config/services.json > /tmp/svcs.json
          echo "list=$(cat /tmp/svcs.json)" >> $GITHUB_OUTPUT

      - name: Delete tags
        env:
          GH_TOKEN: ${{ secrets.GHCR_TOKEN }}
        run: |
          set -euo pipefail
          for svc in $(jq -r '.[]' /tmp/svcs.json); do
            echo "Cleaning up $svc tags matching pr-${PR_NUMBER}-*"
            gh api "/users/chronicle20/packages/container/${svc}%2F${svc}/versions" \
              --paginate \
              --jq '.[] | select(.metadata.container.tags[]? | startswith("pr-'"${PR_NUMBER}"'-")) | .id' \
              | while read -r vid; do
                  gh api --method DELETE \
                    "/users/chronicle20/packages/container/${svc}%2F${svc}/versions/${vid}" \
                    || echo "skip: $svc/$vid"
              done
          done

  notify-argo:
    name: Notify Argo of PR close
    runs-on: ubuntu-latest
    needs: delete-images
    steps:
      - name: Log notice
        run: |
          echo "PR ${PR_NUMBER} closed at $(date -u +%Y-%m-%dT%H:%M:%SZ)"
          echo "Argo CD's hourly cleanup CronJob will set cleanup-deadline within the hour."
          echo "Force-cleanup (bypassing grace) is documented in docs/runbooks/ephemeral-pr-deployments.md §9.2."
```

- [ ] **Step 2: Lint**

```bash
actionlint .github/workflows/pr-cleanup.yml
```

- [ ] **Step 3: Commit**

```bash
git add .github/workflows/pr-cleanup.yml
git commit -m "ci(pr-cleanup): delete per-PR ghcr image tags on PR close

Argo CD's cleanup CronJob is the canonical actor for env teardown
(it runs hourly and respects atlas.cleanup-grace). This workflow
removes ghcr image tags eagerly so PR-close doesn't leave hundreds
of orphaned tags accumulating during the grace window.

Refs task-063."
```

---

## Phase 10: Documentation

### Task 10.1: deploy/k8s/README.md

- [ ] **Step 1: Create `deploy/k8s/README.md`**

```markdown
# Atlas Kubernetes manifests

Atlas's manifests are organised as a Kustomize base plus two overlays:

```
deploy/k8s/
├── base/                # Per-service Deployment+Service (no namespace)
├── overlays/
│   ├── main/            # main env: namespace=atlas, images=:latest
│   └── pr/              # PR env: namespace=atlas-pr-<N>, hash-suffixed
└── README.md
```

## Adding a new service

1. Drop the service's manifest into `deploy/k8s/base/<svc>.yaml`.
2. Add the path to `deploy/k8s/base/kustomization.yaml`.
3. If the service uses Postgres, add its base DB name to the `ATLAS_DB_NAMES`
   literal in `deploy/k8s/overlays/pr/kustomization.yaml`.
4. Re-run the consumer-group patch generator:
   ```sh
   ./deploy/k8s/overlays/pr/scripts/gen-consumer-group-patch.sh
   ```
5. Commit `base/`, the updated kustomization, and the regenerated patches.

## ATLAS_ENV flow

The 4-character hex token `ATLAS_ENV` is the load-bearing isolation key.
For the **main** env it is unset (legacy behaviour preserved). For PR envs:

1. Argo CD's `ApplicationSet` computes
   `ATLAS_ENV = sha256("pr-<N>")[:4]` per PR.
2. The value is materialised as the literal in `atlas-env-tokens.ConfigMap`
   inside the rendered overlay.
3. Kustomize `replacements:` substitute `PLACEHOLDER_ATLAS_ENV` slots in:
   - Every Deployment's `DB_NAME`, `KAFKA_CONSUMER_GROUP`, `ATLAS_ENV` env
   - The `atlas-env` ConfigMap's topic values
4. Service code reads:
   - `DB_NAME` → `libs/atlas-database` connects to the suffixed DB
   - `KAFKA_CONSUMER_GROUP` → `libs/atlas-kafka/consumergroup.Resolve()`
     returns the suffixed group ID
   - `ATLAS_ENV` → `libs/atlas-redis.computeKeyPrefix()` prefixes every
     Redis key
   - Topic env vars → `topic.EnvProvider` returns the suffixed topic name

## Hooks

PR overlay only:

- `presync-create-dbs.yaml` — `CREATE DATABASE IF NOT EXISTS` per service DB
- `postsync-bootstrap.yaml` — runs the canonical WZ ingest + per-domain seed
- `postsync-pihole-add.yaml` — registers `<N>.atlas.home` on both Pi-holes
- `postdelete-cleanup.yaml` — drops DBs, topics, groups, Redis keys, ghcr tags, DNS

See `docs/runbooks/ephemeral-pr-deployments.md` for operational guidance.
```

- [ ] **Step 2: Commit**

```bash
git add deploy/k8s/README.md
git commit -m "docs(deploy): Kustomize structure and ATLAS_ENV flow

Refs task-063."
```

### Task 10.2: docs/runbooks/ephemeral-pr-deployments.md

- [ ] **Step 1: Create the runbook**

```markdown
# Ephemeral per-PR Deployments — Runbook

Reference: `docs/tasks/task-063-ephemeral-pr-deployments/`.

## §9.1 First-time setup: canonical WZ PVC

```sh
# Create the Longhorn ReadOnlyMany PVC (one-time)
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: atlas-wz-canonical
  namespace: longhorn-system
spec:
  accessModes: [ReadWriteOnce]
  resources:
    requests:
      storage: 8Gi
  storageClassName: longhorn
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: atlas-wz-canonical-readonly
  namespace: argocd
spec:
  accessModes: [ReadOnlyMany]
  resources:
    requests:
      storage: 8Gi
  storageClassName: longhorn
EOF

# Mount into a temporary writer pod and copy the canonical zip.
kubectl run wz-uploader --image=alpine -i --tty --rm \
    --overrides='{"spec":{"volumes":[{"name":"wz","persistentVolumeClaim":{"claimName":"atlas-wz-canonical"}}],"containers":[{"name":"wz","image":"alpine","stdin":true,"tty":true,"volumeMounts":[{"name":"wz","mountPath":"/opt/wz"}]}]}}' \
    -- /bin/sh
# inside: scp atlas.zip into /opt/wz/atlas.zip
```

## §9.2 Force-cleanup of a PR env

Bypass the grace period:

```sh
kubectl delete application -n argocd atlas-pr-<N>
```

Argo's PostDelete hook fires immediately. Verify:

```sh
kubectl get jobs -n atlas-pr-<N>
kubectl logs -n atlas-pr-<N> job/atlas-pr-cleanup -f
```

## §9.3 Inspecting a stuck env

```sh
argocd app get atlas-pr-<N>
kubectl get all,configmap,secret -n atlas-pr-<N>
kubectl logs -n atlas-pr-<N> job/atlas-pr-bootstrap
```

Loki query for env-scoped logs (`atlas.env=<token>`):

```logql
{atlas_env="a3f7"} |= ""
```

## §9.4 Re-running a failed PostDelete

If the cleanup Job fails, the Application stays in `cleanup-failed`. Re-run:

```sh
kubectl delete job -n atlas-pr-<N> atlas-pr-cleanup
kubectl create -n atlas-pr-<N> -f <(\
  kubectl get application atlas-pr-<N> -n argocd -o jsonpath='{...postdelete...}')
```

(The simpler path: `argocd app sync atlas-pr-<N> --force`, then re-delete.)

## §9.5 Rotating credentials

- **GitHub PAT for Argo:** generate a new fine-scoped PAT, then
  `kubectl edit secret argocd-repo-creds-chronicle20-atlas -n argocd`,
  replace `password`, save.
- **Pi-hole tokens:** edit `pihole-credentials` Secret similarly. The
  postsync-pihole-add Job reads at run time; rotation takes effect on
  the next sync.

## §9.6 Bootstrap-duration metrics

```promql
histogram_quantile(0.95,
  rate(atlas_bootstrap_step_duration_ms_bucket{atlas_env!="main"}[1h]))
```

Loki: filter `atlas.cleanup-step` field for stepwise breakdown.

## §9.7 Hash-collision resolution

Two PRs hash to the same 4-hex `ATLAS_ENV`. Symptom: second PR's
Application sync fails with a namespace conflict.

Workaround: close-and-reopen one PR — head SHA changes (or use a force
push to perturb the head). Long-term mitigation: bump suffix to 6 hex.

## §9.8 main env cutover (one-time)

1. Confirm `kustomize build deploy/k8s/overlays/main` matches the live
   cluster:
   ```sh
   kustomize build deploy/k8s/overlays/main > /tmp/built.yaml
   kubectl get -n atlas all,configmap -o yaml > /tmp/live.yaml
   yq eval-all 'select(fileIndex == 0) - select(fileIndex == 1)' \
       /tmp/built.yaml /tmp/live.yaml
   ```
   Expected: only Kustomize-injected labels are net new.
2. Apply Argo CD on bee per `deploy/argocd-bee/README.md`.
3. Apply `Application(atlas-main)` with `prune: false`. Confirm
   `Synced/Healthy` with zero changes.
4. Wait 7 days.
5. Edit `prune: true` on the Application, reapply.

## §9.9 Adding a service after cutover

Follow `deploy/k8s/README.md`'s "Adding a new service" — the
generators must be re-run so `consumer-group-env.yaml` and
`db-name-suffix.yaml` include the new entry.
```

- [ ] **Step 2: Commit**

```bash
git add docs/runbooks/ephemeral-pr-deployments.md
git commit -m "docs(runbooks): ephemeral PR deployments

Refs task-063."
```

### Task 10.3: docs/observability.md update

- [ ] **Step 1: Locate the file**

```bash
ls docs/observability.md 2>/dev/null || ls docs/observability* 2>/dev/null
```

If it exists, edit it. If not, create at `docs/observability.md`.

- [ ] **Step 2: Append (or insert) the env-label section**

```markdown
## Filtering by environment

Every per-environment pod carries the label `atlas.env=<token>`. Use
this label to scope queries:

- `main` env: `atlas.env=main`
- PR env: `atlas.env=<4-char-hex>` (deterministic per PR — see
  `docs/runbooks/ephemeral-pr-deployments.md`)

PR envs additionally carry `atlas.pr-number=<N>`.

### Loki

```logql
{atlas_env="a3f7"} |= "ERROR"
```

### Prometheus

```promql
sum by (pod) (rate(http_request_duration_seconds_count{atlas_env="a3f7"}[5m]))
```

### Grafana

The `atlas-pr-environments` dashboard summarises open envs, time-to-ready,
cleanup status, and bootstrap step durations.
```

- [ ] **Step 3: Commit**

```bash
git add docs/observability.md
git commit -m "docs(observability): atlas.env label filtering

Refs task-063."
```

---

## Phase 11: End-to-end validation (manual, post-merge)

These are not committable code; they are runbook acceptance steps the maintainer follows after the PR merges. Document them in the runbook (Phase 10) and tick them off during cutover. Listed here for completeness.

### Task 11.1: Apply Argo CD on bee
- See `deploy/argocd-bee/README.md` §1.

### Task 11.2: Verify Application(atlas-main) zero-diff sync

```bash
argocd app diff atlas-main
```

Expected: no output (zero diff). If non-zero, reconcile per runbook §9.8.

### Task 11.3: Open canary PR

Open a no-op PR (e.g. one-line whitespace tweak) against `Chronicle20/atlas`
and watch:

1. Within 30s, `atlas-pr-<N>` Application appears in Argo UI.
2. PreSync, Sync, PostSync hooks all green.
3. `<N>.atlas.home` resolves and serves atlas-ui.
4. `kubectl exec -n atlas-pr-<N> atlas-character-* -- env | grep ATLAS_ENV`
   shows the 4-char token.
5. `kafka-consumer-groups.sh --list | grep "[<token>]"` lists per-env groups.
6. `redis-cli --scan --pattern "<token>:*" | head` lists per-env keys.

### Task 11.4: Verify isolation across two PRs

Open a second canary PR. Repeat checks; confirm hashes differ.

### Task 11.5: Close canary, verify cleanup
- Close the PR.
- Wait for cleanup CronJob to set `atlas.cleanup-deadline` annotation.
- `kubectl annotate -n argocd application atlas-pr-<N> atlas.cleanup-deadline=$(date -u +%Y-%m-%dT%H:%M:%SZ) --overwrite` to force-immediate-cleanup.
- Verify PostDelete fires and namespace is gone.

### Task 11.6: Stability window and prune cutover
- After 7 days of green syncs, edit `argocd-atlas-main.yml` to set
  `prune: true`. Reapply.

---

## Self-Review

### Spec coverage

Walked the design's §3 through §13. Gaps and how they're addressed:

| Design § | Plan task |
|---|---|
| §3.1 Argo install | 8.1 |
| §3.2 Application(atlas-main) | 8.2 |
| §3.3 ApplicationSet | 8.3 |
| §3.4 Cleanup CronJob | 8.4 |
| §4.1 Directory restructure | 7.1 |
| §4.2 main overlay | 7.2 |
| §4.3 PR overlay shell | 7.3, 7.4 |
| §4.4 replacements: vs plugin (decision) | embedded in 7.4 |
| §4.5 Per-PR ingress | 7.5 |
| §5.1 ATLAS_ENV token | end-to-end through 7.4 + atlas-env-tokens ConfigMap |
| §5.2 Postgres DB-name | 7.6 (PreSync) + db-name-suffix patch in 7.4 |
| §5.3 Kafka topic | configMapGenerator in 7.4 |
| §5.4 Consumer group resolver | Phase 2 + Phase 4 sweep |
| §5.5 Redis prefix | Phase 1 |
| §5.6 Audit raw-key | Phase 5 + libs/atlas-object-id in Phase 3 |
| §6.1 Longhorn PVC | 10.2 §9.1 (one-time runbook step) |
| §6.2 PostSync bootstrap | 7.7 + Phase 6 image |
| §6.3 PostSync Pi-hole | 7.8 |
| §6.4 PostDelete cleanup | 7.9 + Phase 6 image |
| §7 main migration | 10.2 §9.8 |
| §8 Failure modes | 10.2 §9.2–9.7 |
| §9 Documentation | 10.1, 10.2, 10.3 |
| §12 CI changes | Phase 9 |
| §13 Decomposition preview | this plan |

PRD acceptance criteria (PRD §10.1–10.8) all map to one or more plan
tasks. Verified by walking each criterion against the plan.

### Placeholder scan

- "TODO/FIXME": none in plan.
- "implement later": none.
- "similar to Task N": Phase 5 tasks 5.1–5.12 reference a shared
  pattern; the *pattern is fully shown in Task 5.1* and each subsequent
  task lists its file paths and literal substitutions explicitly. No
  task says "similar to X" without citing the actual edit.
- "appropriate error handling" / "validation": none.

### Type consistency

- `KeyPrefix()` exported in Phase 1 §1.2; used by Phase 3, Phase 5
  consistently as `atlasredis.KeyPrefix()`.
- `consumergroup.Resolve(string) string` defined in Phase 2; used in
  Phase 4 with the same signature.
- `ATLAS_ENV` env var name: consistent across libs, manifests, hook
  scripts, ApplicationSet template.
- `KAFKA_CONSUMER_GROUP` env var name: consistent in resolver, kustomize
  patch, service main.go.
- `atlas-env-tokens` ConfigMap name: consistent across overlay,
  PostSync hooks, replacements rule.
- `atlas-pr-bootstrap` image name: consistent between Phase 6 (build),
  Phase 7 (consume), `.github/config/services.json`.

No drifts found.

---

## Appendix A: Service consumer-group literals

For Phase 4 Task 4.1, edit each `services/<svc>/atlas.com/<module>/main.go`
to convert:

```go
const consumerGroupId = "<literal>"
```

to:

```go
var consumerGroupId = consumergroup.Resolve("<literal>")
```

| Service path | Literal |
|---|---|
| atlas-account/atlas.com/account | `Account Service` |
| atlas-asset-expiration/atlas.com/asset-expiration | `Asset Expiration Service` |
| atlas-ban/atlas.com/ban | `Ban Service` |
| atlas-buddies/atlas.com/buddies | `Buddy Service` |
| atlas-buffs/atlas.com/buffs | `Buff Service` |
| atlas-cashshop/atlas.com/cashshop | `Cash Shop Service` |
| atlas-chairs/atlas.com/chairs | `Chairs Service` |
| atlas-chalkboards/atlas.com/chalkboards | `Chalkboard Service` |
| atlas-character-factory/atlas.com/character-factory | `Character Factory Service` |
| atlas-character/atlas.com/character | `Character Service` |
| atlas-consumables/atlas.com/consumables | `Consumables Service` |
| atlas-data/atlas.com/data | `Data Service` |
| atlas-drops/atlas.com/drops | `Drops Service` |
| atlas-effective-stats/atlas.com/effective-stats | `Effective Stats Service` |
| atlas-expressions/atlas.com/expressions | `Expression Service` |
| atlas-fame/atlas.com/fame | `Fame Service` |
| atlas-families/atlas.com/family | `Family Service` |
| atlas-guilds/atlas.com/guilds | `Guilds Service` |
| atlas-inventory/atlas.com/inventory | `Inventory Service` |
| atlas-invites/atlas.com/invites | `Invitation Service` |
| atlas-keys/atlas.com/keys | `Key Service` |
| atlas-map-actions/atlas.com/map-actions | `Map Actions Service` |
| atlas-maps/atlas.com/maps | `Map Service` |
| atlas-marriages/atlas.com/marriages | `Marriage Service` |
| atlas-merchant/atlas.com/merchant | `Merchant Service` |
| atlas-messages/atlas.com/messages | `Messages Service` |
| atlas-messengers/atlas.com/messengers | `Messenger Service` |
| atlas-monster-death/atlas.com/monster | `Monster Death Service` |
| atlas-monsters/atlas.com/monsters | `Monster Registry Service` |
| atlas-notes/atlas.com/notes | `Notes Service` |
| atlas-npc-conversations/atlas.com/npc | `NPC Conversation Service` |
| atlas-npc-shops/atlas.com/npc | `NPC Shops Service` |
| atlas-parties/atlas.com/parties | `Party Service` |
| atlas-party-quests/atlas.com/party-quests | `Party Quest Service` |
| atlas-pets/atlas.com/pets | `Pets Service` |
| atlas-portal-actions/atlas.com/portal | `Portal Actions Service` |
| atlas-portals/atlas.com/portals | `Portal Service` |
| atlas-quest/atlas.com/quest | `Quest Service` |
| atlas-rates/atlas.com/rates | `Rate Service` |
| atlas-reactor-actions/atlas.com/reactor | `Reactor Actions Service` |
| atlas-reactors/atlas.com/reactors | `Reactors Service` |
| atlas-saga-orchestrator/atlas.com/saga-orchestrator | `Saga Orchestrator Service` |
| atlas-skills/atlas.com/skills | `Skills Service` |
| atlas-storage/atlas.com/storage | `Storage Service` |
| atlas-tenants/atlas.com/tenants | `Tenant Service` |
| atlas-transports/atlas.com/transports | `Transport Service` |
| atlas-world/atlas.com/world | `World Orchestrator` |

Templated (Phase 4 Task 4.2):

| Service path | Template |
|---|---|
| atlas-channel/atlas.com/channel | `Channel Service - %s` (channel UUID) |
| atlas-login/atlas.com/login | `ChannelConnect Service - %s` (login UUID) |

49 services total.
