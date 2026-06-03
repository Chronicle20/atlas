# PR-Env Teardown Leak Fixes — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Stop PR-env teardown from leaking Redis keys and MinIO tenant data into shared infra, and prevent recurrence with an automated guard.

**Architecture:** (1) Extend `libs/atlas-redis` with typed Set/Hash registries, migrate every bypassing call site onto them so all keys carry `KeyPrefix()`, and add a `go vet`-style analyzer that bans keyed ops on the raw client outside the lib. (2) Move MinIO tenant purge from a post-prune PostDelete bash phase to an Argo CD PreDelete hook that calls atlas-data's purge endpoint while the env is alive; delete the broken PostDelete phase; harden the sweep CronJob backstop.

**Tech Stack:** Go 1.25/1.26 (generics), `redis/go-redis/v9`, `alicebob/miniredis/v2`, `golang.org/x/tools/go/analysis`, Bash + bats, Argo CD lifecycle hooks, Kustomize.

**Read `context.md` first** — it records the locked decisions (D1–D5, P1–P3) and the corrections to the design's call-site table that this plan depends on.

---

## File Structure

### libs/atlas-redis (new types — Phase 1)
- Create `libs/atlas-redis/set.go` — `Set` (env-global SET) + `TenantSet`.
- Create `libs/atlas-redis/set_test.go`.
- Create `libs/atlas-redis/hash.go` — `Hash` (env-global HASH) + `KeyedHash[K]`.
- Create `libs/atlas-redis/hash_test.go`.
- Create `libs/atlas-redis/keyed_set.go` — `TenantKeyedSet[K]`.
- Create `libs/atlas-redis/keyed_set_test.go`.
- Create `libs/atlas-redis/keyed_hash.go` — `TenantKeyedHash[K]`.
- Create `libs/atlas-redis/keyed_hash_test.go`.

### Service migrations (Phase 2) — rewrite each registry to use the lib types
- Modify `services/atlas-world/atlas.com/world/channel/registry.go` (+ test).
- Modify `services/atlas-invites/atlas.com/invites/invite/registry.go` (+ test).
- Modify `services/atlas-guilds/atlas.com/guilds/coordinator/registry.go` (+ test).
- Modify `services/atlas-drops/atlas.com/drops/drop/registry.go` (+ test).
- Modify `services/atlas-reactors/atlas.com/reactors/reactor/registry.go` (+ test).
- Modify `services/atlas-transports/atlas.com/transports/instance/instance_registry.go` (+ test).
- Modify `services/atlas-transports/atlas.com/transports/instance/character_registry.go` (+ test).
- Modify `services/atlas-transports/atlas.com/transports/channel/registry.go` (+ test).
- Modify `services/atlas-rates/atlas.com/rates/character/item_tracker.go` (+ test).
- Modify `services/atlas-maps/atlas.com/maps/map/monster/registry.go` (+ test).

### Regression guard (Phase 3)
- Create `tools/rediskeyguard/go.mod`, `analyzer.go`, `analyzer_test.go`,
  `cmd/rediskeyguard/main.go`, `testdata/src/bad/bad.go`, `testdata/src/good/good.go`.
- Create `tools/redis-key-guard.sh`.
- Modify `CLAUDE.md` (verification list), `.github/workflows/` CI (one step).

### One-time reclaim (Phase 4)
- Create `services/atlas-pr-bootstrap/scripts/reclaim-main-bare-keys.sh` (+ bats).

### Leak #2 (Phases 5–7)
- Create `services/atlas-pr-bootstrap/scripts/predelete-purge.sh` (+ bats).
- Modify `services/atlas-pr-bootstrap/Dockerfile`.
- Create `deploy/k8s/overlays/pr/predelete-purge.yaml`; modify `kustomization.yaml`.
- Modify `services/atlas-pr-bootstrap/scripts/cleanup.sh` (+ bats); modify
  `deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml`.
- Modify `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh` (+ bats).
- Create `dev/cluster-infra-coordination/task-045-teardown.md` +
  `dev/cluster-infra-coordination/sweep-orphans-cronjob.example.yaml`.

---

## Phase 1 — libs/atlas-redis new types

All new types are thin wrappers over the existing `keys.go` helpers
(`namespacedKey`, `tenantEntityKey`, `TenantKey`). They live in the `redis`
package and reuse the test harness in `registry_test.go`
(`setupTestRedis`, `makeTenant`, `testTenant`) and `tenant_registry_test.go`
(`newTestTenant`).

Run all Phase-1 commands from `libs/atlas-redis/`.

### Task 1: `Set` and `TenantSet`

**Files:**
- Create: `libs/atlas-redis/set.go`
- Test: `libs/atlas-redis/set_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// libs/atlas-redis/set_test.go
package redis

import (
	"context"
	"testing"
)

func TestSet_AddMembersIsMemberSize(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("a3f7")

	client, mr := setupTestRedis(t)
	ctx := context.Background()
	s := NewSet(client, "drops:all")

	if err := s.Add(ctx, "x", "y"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	// Key-format assertion: must be <env>:atlas:<namespace>.
	if !mr.Exists("a3f7:atlas:drops:all") {
		t.Fatalf("expected key a3f7:atlas:drops:all to exist; keys=%v", mr.Keys())
	}
	ok, err := s.IsMember(ctx, "x")
	if err != nil || !ok {
		t.Fatalf("IsMember x = %v,%v want true,nil", ok, err)
	}
	n, err := s.Size(ctx)
	if err != nil || n != 2 {
		t.Fatalf("Size = %d,%v want 2,nil", n, err)
	}
	members, err := s.Members(ctx)
	if err != nil || len(members) != 2 {
		t.Fatalf("Members = %v,%v want len 2", members, err)
	}
	if err := s.Remove(ctx, "x"); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if n, _ := s.Size(ctx); n != 1 {
		t.Fatalf("Size after remove = %d want 1", n)
	}
}

func TestTenantSet_PerTenantKeyAndIsolation(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, mr := setupTestRedis(t)
	ctx := context.Background()
	s := NewTenantSet(client, "transport:channels")
	t1 := makeTenant("00000000-0000-0000-0000-000000000001", "GMS", 83, 1)
	t2 := makeTenant("00000000-0000-0000-0000-000000000002", "GMS", 83, 1)

	if err := s.Add(ctx, t1, "0:1"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	wantKey := "atlas:transport:channels:" + TenantKey(t1)
	if !mr.Exists(wantKey) {
		t.Fatalf("expected key %q; keys=%v", wantKey, mr.Keys())
	}
	if m2, _ := s.Members(ctx, t2); len(m2) != 0 {
		t.Fatalf("t2 must not see t1 members: %v", m2)
	}
	if m1, _ := s.Members(ctx, t1); len(m1) != 1 || m1[0] != "0:1" {
		t.Fatalf("t1 members = %v want [0:1]", m1)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./... -run 'TestSet_|TestTenantSet_' -v`
Expected: FAIL — `undefined: NewSet` / `undefined: NewTenantSet`.

- [ ] **Step 3: Implement `set.go`**

```go
// libs/atlas-redis/set.go
package redis

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

func toIfaces(members []string) []interface{} {
	out := make([]interface{}, len(members))
	for i, m := range members {
		out[i] = m
	}
	return out
}

// Set is an env-global Redis SET whose key is namespaced via KeyPrefix().
// Use for cross-tenant-within-env aggregate sets (e.g. "drops:all").
type Set struct {
	client    *goredis.Client
	namespace string
}

func NewSet(client *goredis.Client, namespace string) *Set {
	return &Set{client: client, namespace: namespace}
}

func (s *Set) key() string { return namespacedKey(s.namespace) }

func (s *Set) Add(ctx context.Context, members ...string) error {
	if len(members) == 0 {
		return nil
	}
	return s.client.SAdd(ctx, s.key(), toIfaces(members)...).Err()
}

func (s *Set) Remove(ctx context.Context, members ...string) error {
	if len(members) == 0 {
		return nil
	}
	return s.client.SRem(ctx, s.key(), toIfaces(members)...).Err()
}

func (s *Set) Members(ctx context.Context) ([]string, error) {
	return s.client.SMembers(ctx, s.key()).Result()
}

func (s *Set) IsMember(ctx context.Context, member string) (bool, error) {
	return s.client.SIsMember(ctx, s.key(), member).Result()
}

func (s *Set) Size(ctx context.Context) (int64, error) {
	return s.client.SCard(ctx, s.key()).Result()
}

// TenantSet is a tenant-scoped Redis SET: one SET per tenant under namespace.
type TenantSet struct {
	client    *goredis.Client
	namespace string
}

func NewTenantSet(client *goredis.Client, namespace string) *TenantSet {
	return &TenantSet{client: client, namespace: namespace}
}

func (s *TenantSet) key(t tenant.Model) string {
	return namespacedKey(s.namespace, TenantKey(t))
}

func (s *TenantSet) Add(ctx context.Context, t tenant.Model, members ...string) error {
	if len(members) == 0 {
		return nil
	}
	return s.client.SAdd(ctx, s.key(t), toIfaces(members)...).Err()
}

func (s *TenantSet) Remove(ctx context.Context, t tenant.Model, members ...string) error {
	if len(members) == 0 {
		return nil
	}
	return s.client.SRem(ctx, s.key(t), toIfaces(members)...).Err()
}

func (s *TenantSet) Members(ctx context.Context, t tenant.Model) ([]string, error) {
	return s.client.SMembers(ctx, s.key(t)).Result()
}
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./... -run 'TestSet_|TestTenantSet_' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-redis/set.go libs/atlas-redis/set_test.go
git commit -m "feat(atlas-redis): add Set and TenantSet types"
```

### Task 2: `Hash` and `KeyedHash[K]`

**Files:**
- Create: `libs/atlas-redis/hash.go`
- Test: `libs/atlas-redis/hash_test.go`

- [ ] **Step 1: Write the failing tests**

```go
// libs/atlas-redis/hash_test.go
package redis

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestHash_SetGetDelExistsGetAll(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("a3f7")

	client, mr := setupTestRedis(t)
	ctx := context.Background()
	h := NewHash(client, "transport:characters")

	if err := h.Set(ctx, "1001", "inst-a"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if !mr.Exists("a3f7:atlas:transport:characters") {
		t.Fatalf("expected key a3f7:atlas:transport:characters; keys=%v", mr.Keys())
	}
	v, err := h.Get(ctx, "1001")
	if err != nil || v != "inst-a" {
		t.Fatalf("Get = %q,%v want inst-a,nil", v, err)
	}
	ok, _ := h.Exists(ctx, "1001")
	if !ok {
		t.Fatalf("Exists 1001 want true")
	}
	if _, err := h.Get(ctx, "nope"); err != ErrNotFound {
		t.Fatalf("Get missing = %v want ErrNotFound", err)
	}
	all, _ := h.GetAll(ctx)
	if len(all) != 1 {
		t.Fatalf("GetAll len = %d want 1", len(all))
	}
	if err := h.Del(ctx, "1001"); err != nil {
		t.Fatalf("Del: %v", err)
	}
	if ok, _ := h.Exists(ctx, "1001"); ok {
		t.Fatalf("Exists after Del want false")
	}
}

func TestKeyedHash_PerKeyHashKeyAndOps(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, mr := setupTestRedis(t)
	ctx := context.Background()
	kh := NewKeyedHash[uuid.UUID](client, "transport:instance:chars", func(id uuid.UUID) string { return id.String() })
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")

	if got := kh.Key(id); got != "atlas:transport:instance:chars:"+id.String() {
		t.Fatalf("Key = %q", got)
	}
	if err := kh.Set(ctx, id, "1001", "entry"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	if !mr.Exists("atlas:transport:instance:chars:" + id.String()) {
		t.Fatalf("expected per-key hash; keys=%v", mr.Keys())
	}
	if n, _ := kh.Len(ctx, id); n != 1 {
		t.Fatalf("Len = %d want 1", n)
	}
	all, _ := kh.GetAll(ctx, id)
	if all["1001"] != "entry" {
		t.Fatalf("GetAll = %v", all)
	}
	_ = kh.Del(ctx, id, "1001")
	if n, _ := kh.Len(ctx, id); n != 0 {
		t.Fatalf("Len after Del = %d want 0", n)
	}
}

func TestKeyedHash_ClearByPrefix(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, _ := setupTestRedis(t)
	ctx := context.Background()
	// Maps shape: keyFn embeds the tenant uuid then the field segments.
	kh := NewKeyedHash[string](client, "maps:spawn", func(k string) string { return k })
	_ = kh.Set(ctx, "uuidA:0:1:100:nil", "1", "{}")
	_ = kh.Set(ctx, "uuidA:0:1:200:nil", "1", "{}")
	_ = kh.Set(ctx, "uuidB:0:1:100:nil", "1", "{}")

	// Clear only tenant uuidA's hashes.
	deleted, err := kh.Clear(ctx, "uuidA")
	if err != nil || deleted != 2 {
		t.Fatalf("Clear(uuidA) = %d,%v want 2,nil", deleted, err)
	}
	if n, _ := kh.Len(ctx, "uuidB:0:1:100:nil"); n != 1 {
		t.Fatalf("uuidB hash must survive; Len=%d", n)
	}
	// Clear everything.
	if _, err := kh.Clear(ctx); err != nil {
		t.Fatalf("Clear(all): %v", err)
	}
	if n, _ := kh.Len(ctx, "uuidB:0:1:100:nil"); n != 0 {
		t.Fatalf("Clear(all) should remove uuidB; Len=%d", n)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./... -run 'TestHash_|TestKeyedHash_' -v`
Expected: FAIL — `undefined: NewHash` / `undefined: NewKeyedHash`.

- [ ] **Step 3: Implement `hash.go`**

```go
// libs/atlas-redis/hash.go
package redis

import (
	"context"
	"errors"

	goredis "github.com/redis/go-redis/v9"
)

// Hash is an env-global Redis HASH whose key is namespaced via KeyPrefix().
type Hash struct {
	client    *goredis.Client
	namespace string
}

func NewHash(client *goredis.Client, namespace string) *Hash {
	return &Hash{client: client, namespace: namespace}
}

func (h *Hash) key() string { return namespacedKey(h.namespace) }

func (h *Hash) Set(ctx context.Context, field, value string) error {
	return h.client.HSet(ctx, h.key(), field, value).Err()
}

func (h *Hash) Get(ctx context.Context, field string) (string, error) {
	v, err := h.client.HGet(ctx, h.key(), field).Result()
	if errors.Is(err, goredis.Nil) {
		return "", ErrNotFound
	}
	return v, err
}

func (h *Hash) Del(ctx context.Context, fields ...string) error {
	if len(fields) == 0 {
		return nil
	}
	return h.client.HDel(ctx, h.key(), fields...).Err()
}

func (h *Hash) Exists(ctx context.Context, field string) (bool, error) {
	return h.client.HExists(ctx, h.key(), field).Result()
}

func (h *Hash) GetAll(ctx context.Context) (map[string]string, error) {
	return h.client.HGetAll(ctx, h.key()).Result()
}

// KeyedHash is a family of env-global HASHes, one per key K. The Lua-script
// callers (atlas-maps) obtain the concrete Redis key via Key(k) and run their
// scripts against it; Key construction stays inside the lib.
type KeyedHash[K comparable] struct {
	client    *goredis.Client
	namespace string
	keyFn     func(K) string
}

func NewKeyedHash[K comparable](client *goredis.Client, namespace string, keyFn func(K) string) *KeyedHash[K] {
	return &KeyedHash[K]{client: client, namespace: namespace, keyFn: keyFn}
}

// Key returns the fully-namespaced Redis key for k.
func (h *KeyedHash[K]) Key(k K) string { return namespacedKey(h.namespace, h.keyFn(k)) }

func (h *KeyedHash[K]) Set(ctx context.Context, k K, field, value string) error {
	return h.client.HSet(ctx, h.Key(k), field, value).Err()
}

func (h *KeyedHash[K]) Get(ctx context.Context, k K, field string) (string, error) {
	v, err := h.client.HGet(ctx, h.Key(k), field).Result()
	if errors.Is(err, goredis.Nil) {
		return "", ErrNotFound
	}
	return v, err
}

func (h *KeyedHash[K]) Del(ctx context.Context, k K, fields ...string) error {
	if len(fields) == 0 {
		return nil
	}
	return h.client.HDel(ctx, h.Key(k), fields...).Err()
}

func (h *KeyedHash[K]) Exists(ctx context.Context, k K, field string) (bool, error) {
	return h.client.HExists(ctx, h.Key(k), field).Result()
}

func (h *KeyedHash[K]) GetAll(ctx context.Context, k K) (map[string]string, error) {
	return h.client.HGetAll(ctx, h.Key(k)).Result()
}

func (h *KeyedHash[K]) Len(ctx context.Context, k K) (int64, error) {
	return h.client.HLen(ctx, h.Key(k)).Result()
}

// DeleteKey removes the entire hash for k.
func (h *KeyedHash[K]) DeleteKey(ctx context.Context, k K) error {
	return h.client.Del(ctx, h.Key(k)).Err()
}

// Clear deletes every hash whose key begins with
// namespacedKey(namespace, segments...). With no segments it clears the whole
// namespace. SCAN(COUNT=100) + pipelined DEL, mirroring TenantRegistry.Clear.
// Returns the number of keys deleted.
func (h *KeyedHash[K]) Clear(ctx context.Context, segments ...string) (int, error) {
	var pattern string
	if len(segments) == 0 {
		pattern = namespacedKey(h.namespace) + keySeparator + "*"
	} else {
		pattern = namespacedKey(h.namespace, segments...) + keySeparator + "*"
	}
	iter := h.client.Scan(ctx, 0, pattern, 100).Iterator()
	deleted := 0
	pipe := h.client.Pipeline()
	pipeSize := 0
	var firstErr error
	flush := func() {
		if pipeSize == 0 {
			return
		}
		if _, err := pipe.Exec(ctx); err != nil && firstErr == nil {
			firstErr = err
		}
		pipe = h.client.Pipeline()
		pipeSize = 0
	}
	for iter.Next(ctx) {
		pipe.Del(ctx, iter.Val())
		deleted++
		pipeSize++
		if pipeSize >= 100 {
			flush()
		}
	}
	flush()
	if err := iter.Err(); err != nil && firstErr == nil {
		firstErr = err
	}
	return deleted, firstErr
}
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./... -run 'TestHash_|TestKeyedHash_' -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-redis/hash.go libs/atlas-redis/hash_test.go
git commit -m "feat(atlas-redis): add Hash and KeyedHash types"
```

### Task 3: `TenantKeyedSet[K]`

**Files:**
- Create: `libs/atlas-redis/keyed_set.go`
- Test: `libs/atlas-redis/keyed_set_test.go`

- [ ] **Step 1: Write the failing test**

```go
// libs/atlas-redis/keyed_set_test.go
package redis

import (
	"context"
	"testing"
)

func TestTenantKeyedSet_PerTenantPerKey(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, mr := setupTestRedis(t)
	ctx := context.Background()
	s := NewTenantKeyedSet[string](client, "drops:map", func(k string) string { return k })
	tm := makeTenant("00000000-0000-0000-0000-000000000001", "GMS", 83, 1)

	if err := s.Add(ctx, tm, "0:1:100:nil", "42", "43"); err != nil {
		t.Fatalf("Add: %v", err)
	}
	wantKey := "atlas:drops:map:" + TenantKey(tm) + ":0:1:100:nil"
	if !mr.Exists(wantKey) {
		t.Fatalf("expected key %q; keys=%v", wantKey, mr.Keys())
	}
	members, _ := s.Members(ctx, tm, "0:1:100:nil")
	if len(members) != 2 {
		t.Fatalf("Members = %v want len 2", members)
	}
	_ = s.Remove(ctx, tm, "0:1:100:nil", "42")
	members, _ = s.Members(ctx, tm, "0:1:100:nil")
	if len(members) != 1 {
		t.Fatalf("Members after remove = %v want len 1", members)
	}
	if err := s.Clear(ctx, tm, "0:1:100:nil"); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	members, _ = s.Members(ctx, tm, "0:1:100:nil")
	if len(members) != 0 {
		t.Fatalf("Members after Clear = %v want empty", members)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./... -run TestTenantKeyedSet_ -v`
Expected: FAIL — `undefined: NewTenantKeyedSet`.

- [ ] **Step 3: Implement `keyed_set.go`**

```go
// libs/atlas-redis/keyed_set.go
package redis

import (
	"context"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

// TenantKeyedSet is a family of tenant-scoped SETs, one per key K.
// Key format: <prefix>:<namespace>:<tenantKey>:<keyFn(k)>.
type TenantKeyedSet[K comparable] struct {
	client    *goredis.Client
	namespace string
	keyFn     func(K) string
}

func NewTenantKeyedSet[K comparable](client *goredis.Client, namespace string, keyFn func(K) string) *TenantKeyedSet[K] {
	return &TenantKeyedSet[K]{client: client, namespace: namespace, keyFn: keyFn}
}

func (s *TenantKeyedSet[K]) key(t tenant.Model, k K) string {
	return tenantEntityKey(s.namespace, t, s.keyFn(k))
}

func (s *TenantKeyedSet[K]) Add(ctx context.Context, t tenant.Model, k K, members ...string) error {
	if len(members) == 0 {
		return nil
	}
	return s.client.SAdd(ctx, s.key(t, k), toIfaces(members)...).Err()
}

func (s *TenantKeyedSet[K]) Remove(ctx context.Context, t tenant.Model, k K, members ...string) error {
	if len(members) == 0 {
		return nil
	}
	return s.client.SRem(ctx, s.key(t, k), toIfaces(members)...).Err()
}

func (s *TenantKeyedSet[K]) Members(ctx context.Context, t tenant.Model, k K) ([]string, error) {
	return s.client.SMembers(ctx, s.key(t, k)).Result()
}

// Clear removes the entire SET for (t, k).
func (s *TenantKeyedSet[K]) Clear(ctx context.Context, t tenant.Model, k K) error {
	return s.client.Del(ctx, s.key(t, k)).Err()
}
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./... -run TestTenantKeyedSet_ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-redis/keyed_set.go libs/atlas-redis/keyed_set_test.go
git commit -m "feat(atlas-redis): add TenantKeyedSet type"
```

### Task 4: `TenantKeyedHash[K]`

**Files:**
- Create: `libs/atlas-redis/keyed_hash.go`
- Test: `libs/atlas-redis/keyed_hash_test.go`

- [ ] **Step 1: Write the failing test**

```go
// libs/atlas-redis/keyed_hash_test.go
package redis

import (
	"context"
	"testing"
)

func TestTenantKeyedHash_SetNXAndDeleteKey(t *testing.T) {
	prev := keyPrefix
	t.Cleanup(func() { keyPrefix = prev })
	keyPrefix = computeKeyPrefix("")

	client, mr := setupTestRedis(t)
	ctx := context.Background()
	h := NewTenantKeyedHash[string](client, "reactor:spot", func(k string) string { return k })
	tm := makeTenant("00000000-0000-0000-0000-000000000001", "GMS", 83, 1)
	mapKey := "0:1:100:nil"

	first, err := h.SetNX(ctx, tm, mapKey, "5000:10:20", "1")
	if err != nil || !first {
		t.Fatalf("first SetNX = %v,%v want true,nil", first, err)
	}
	wantKey := "atlas:reactor:spot:" + TenantKey(tm) + ":" + mapKey
	if !mr.Exists(wantKey) {
		t.Fatalf("expected key %q; keys=%v", wantKey, mr.Keys())
	}
	second, _ := h.SetNX(ctx, tm, mapKey, "5000:10:20", "1")
	if second {
		t.Fatalf("second SetNX on same field must be false")
	}
	_ = h.Set(ctx, tm, mapKey, "5000:30:40", "1")
	all, _ := h.GetAll(ctx, tm, mapKey)
	if len(all) != 2 {
		t.Fatalf("GetAll = %v want len 2", all)
	}
	_ = h.Del(ctx, tm, mapKey, "5000:10:20")
	if ok, _ := h.Exists(ctx, tm, mapKey, "5000:10:20"); ok {
		t.Fatalf("field should be gone")
	}
	_ = h.DeleteKey(ctx, tm, mapKey)
	all, _ = h.GetAll(ctx, tm, mapKey)
	if len(all) != 0 {
		t.Fatalf("DeleteKey should empty the hash; got %v", all)
	}
}
```

- [ ] **Step 2: Run to verify failure**

Run: `go test ./... -run TestTenantKeyedHash_ -v`
Expected: FAIL — `undefined: NewTenantKeyedHash`.

- [ ] **Step 3: Implement `keyed_hash.go`**

```go
// libs/atlas-redis/keyed_hash.go
package redis

import (
	"context"
	"errors"

	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

// TenantKeyedHash is a family of tenant-scoped HASHes, one per key K.
// Key format: <prefix>:<namespace>:<tenantKey>:<keyFn(k)>.
type TenantKeyedHash[K comparable] struct {
	client    *goredis.Client
	namespace string
	keyFn     func(K) string
}

func NewTenantKeyedHash[K comparable](client *goredis.Client, namespace string, keyFn func(K) string) *TenantKeyedHash[K] {
	return &TenantKeyedHash[K]{client: client, namespace: namespace, keyFn: keyFn}
}

func (h *TenantKeyedHash[K]) key(t tenant.Model, k K) string {
	return tenantEntityKey(h.namespace, t, h.keyFn(k))
}

func (h *TenantKeyedHash[K]) Set(ctx context.Context, t tenant.Model, k K, field, value string) error {
	return h.client.HSet(ctx, h.key(t, k), field, value).Err()
}

// SetNX sets field only if it does not yet exist; returns true if it was set.
func (h *TenantKeyedHash[K]) SetNX(ctx context.Context, t tenant.Model, k K, field, value string) (bool, error) {
	return h.client.HSetNX(ctx, h.key(t, k), field, value).Result()
}

func (h *TenantKeyedHash[K]) Get(ctx context.Context, t tenant.Model, k K, field string) (string, error) {
	v, err := h.client.HGet(ctx, h.key(t, k), field).Result()
	if errors.Is(err, goredis.Nil) {
		return "", ErrNotFound
	}
	return v, err
}

func (h *TenantKeyedHash[K]) Del(ctx context.Context, t tenant.Model, k K, fields ...string) error {
	if len(fields) == 0 {
		return nil
	}
	return h.client.HDel(ctx, h.key(t, k), fields...).Err()
}

func (h *TenantKeyedHash[K]) Exists(ctx context.Context, t tenant.Model, k K, field string) (bool, error) {
	return h.client.HExists(ctx, h.key(t, k), field).Result()
}

func (h *TenantKeyedHash[K]) GetAll(ctx context.Context, t tenant.Model, k K) (map[string]string, error) {
	return h.client.HGetAll(ctx, h.key(t, k)).Result()
}

// DeleteKey removes the entire hash for (t, k).
func (h *TenantKeyedHash[K]) DeleteKey(ctx context.Context, t tenant.Model, k K) error {
	return h.client.Del(ctx, h.key(t, k)).Err()
}
```

- [ ] **Step 4: Run to verify pass**

Run: `go test ./... -run TestTenantKeyedHash_ -v`
Expected: PASS.

- [ ] **Step 5: Verify the whole lib, then commit**

Run: `go test -race ./... && go vet ./... && go build ./...`
Expected: all clean.

```bash
git add libs/atlas-redis/keyed_hash.go libs/atlas-redis/keyed_hash_test.go
git commit -m "feat(atlas-redis): add TenantKeyedHash type"
```

---

## Phase 2 — Service migrations (Leak #1)

For each service, run `go test -race ./... && go vet ./... && go build ./...`
from that service's module dir (`services/atlas-<svc>/atlas.com/<svc>/`).
`InitRegistry(client)` signatures do not change, so no `main.go`/consumer edits.

Order: simplest first (world, invites) to shake out lib-API issues early.

### Task 5: atlas-world — `channel:tenants` → `Set`

**Files:**
- Modify: `services/atlas-world/atlas.com/world/channel/registry.go`
- Test: `services/atlas-world/atlas.com/world/channel/registry_test.go` (locate
  existing; if none, create)

- [ ] **Step 1: Update/add the failing test**

Add to the channel registry test (create the file if absent, mirroring the
existing builder/test style in the package):

```go
func TestRegistry_TenantSetIsPrefixed(t *testing.T) {
	prev := os.Getenv("ATLAS_ENV")
	// NOTE: keyPrefix in atlas-redis is computed at init from ATLAS_ENV; this
	// test asserts behavior, not the literal env-prefix (that is covered by
	// the lib's own tests). Here we assert tracked tenants round-trip.
	_ = prev
	client, _ := setupTestRedis(t) // add a miniredis helper to this test file
	InitRegistry(client)
	ctx := tenant.WithContext(context.Background(), testTenant())
	reg := GetChannelRegistry()
	reg.Register(ctx, NewModel(world.Id(0), channelConstant.Id(1)))
	tenants := reg.Tenants()
	if len(tenants) != 1 {
		t.Fatalf("Tenants() = %d want 1", len(tenants))
	}
}
```

> If the package has no miniredis helper, add one identical to
> `libs/atlas-redis/registry_test.go:setupTestRedis` and a `testTenant()`
> producing a valid `tenant.Model`. Match imports already used in the package.

- [ ] **Step 2: Run to verify it compiles and (pre-change) still uses bare key**

Run: `go test ./channel/ -run TestRegistry_TenantSetIsPrefixed -v`
Expected: PASS pre-change (behavior unchanged), but the key is bare. This test
guards behavior across the refactor.

- [ ] **Step 3: Rewrite the registry to use `Set`**

Replace the bare-key parts of `registry.go`. Full new file:

```go
package channel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	channelConstant "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	channels *atlas.TenantRegistry[string, Model]
	tenants  *atlas.Set
}

var channelRegistry *Registry

var ErrChannelNotFound = errors.New("channel not found")

func compositeKey(worldId world.Id, channelId channelConstant.Id) string {
	return fmt.Sprintf("%d:%d", worldId, channelId)
}

func InitRegistry(client *goredis.Client) {
	channelRegistry = &Registry{
		channels: atlas.NewTenantRegistry[string, Model](client, "channel", func(k string) string { return k }),
		tenants:  atlas.NewSet(client, "channel:tenants"),
	}
}

func GetChannelRegistry() *Registry {
	return channelRegistry
}

func (r *Registry) Register(ctx context.Context, m Model) Model {
	t := tenant.MustFromContext(ctx)
	key := compositeKey(m.worldId, m.channelId)
	_ = r.channels.Put(ctx, t, key, m)
	r.trackTenant(ctx, t)
	return m
}

func (r *Registry) ChannelServers(ctx context.Context) []Model {
	t := tenant.MustFromContext(ctx)
	vals, err := r.channels.GetAllValues(ctx, t)
	if err != nil {
		return nil
	}
	return vals
}

func (r *Registry) ChannelServer(ctx context.Context, ch channelConstant.Model) (Model, error) {
	t := tenant.MustFromContext(ctx)
	key := compositeKey(ch.WorldId(), ch.Id())
	m, err := r.channels.Get(ctx, t, key)
	if err != nil {
		return Model{}, ErrChannelNotFound
	}
	return m, nil
}

func (r *Registry) RemoveByWorldAndChannel(ctx context.Context, ch channelConstant.Model) error {
	t := tenant.MustFromContext(ctx)
	key := compositeKey(ch.WorldId(), ch.Id())
	exists, _ := r.channels.Exists(ctx, t, key)
	if !exists {
		return ErrChannelNotFound
	}
	_ = r.channels.Remove(ctx, t, key)
	return nil
}

func (r *Registry) Tenants() []tenant.Model {
	ctx := context.Background()
	members, err := r.tenants.Members(ctx)
	if err != nil {
		return nil
	}
	results := make([]tenant.Model, 0)
	for _, data := range members {
		var t tenant.Model
		if err := json.Unmarshal([]byte(data), &t); err != nil {
			continue
		}
		results = append(results, t)
	}
	return results
}

func (r *Registry) trackTenant(ctx context.Context, t tenant.Model) {
	data, err := json.Marshal(&t)
	if err != nil {
		return
	}
	_ = r.tenants.Add(ctx, string(data))
}
```

- [ ] **Step 4: Run tests + vet + build**

Run (from `services/atlas-world/atlas.com/world/`):
`go test -race ./... && go vet ./... && go build ./...`
Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-world/atlas.com/world/channel/
git commit -m "fix(atlas-world): namespace channel:tenants via atlas-redis Set"
```

### Task 6: atlas-invites — `invite:active-tenants` → `Set`

**Files:**
- Modify: `services/atlas-invites/atlas.com/invites/invite/registry.go`
- Test: existing `registry_test.go` in the package (update if it asserts the
  bare key; otherwise add a round-trip test like Task 5).

- [ ] **Step 1: Update the test** — add (or adapt) a behavior test that
  `GetActiveTenants()` round-trips a tracked tenant after `Create`, mirroring
  Task 5's structure (use the package's existing miniredis test helper if
  present; otherwise add one).

- [ ] **Step 2: Run** `go test ./invite/ -run ActiveTenants -v` → PASS pre-change.

- [ ] **Step 3: Edit `registry.go`** — apply exactly these three changes:

1. Delete the const:
```go
const tenantTrackerKey = "invite:active-tenants"
```

2. Replace the `client` field and add a `tenants` Set in the struct + ctor:
```go
type Registry struct {
	invites         *atlas.TenantRegistry[uint32, Model]
	idGen           *atlas.IDGenerator
	targetTypeIndex *atlas.Index
	targetIndex     *atlas.Uint32Index
	originatorIndex *atlas.Uint32Index
	tenants         *atlas.Set
}
```
```go
func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		invites: atlas.NewTenantRegistry[uint32, Model](client, "invite", func(k uint32) string {
			return strconv.FormatUint(uint64(k), 10)
		}),
		idGen:           atlas.NewIDGenerator(client, "invite"),
		targetTypeIndex: atlas.NewIndex(client, "invite", "target-type"),
		targetIndex:     atlas.NewUint32Index(client, "invite", "target"),
		originatorIndex: atlas.NewUint32Index(client, "invite", "originator"),
		tenants:         atlas.NewSet(client, "invite:active-tenants"),
	}
}
```

3. Replace the two bare-client methods:
```go
func (r *Registry) trackTenant(ctx context.Context, t tenant.Model) {
	data, err := json.Marshal(&t)
	if err != nil {
		return
	}
	_ = r.tenants.Add(ctx, string(data))
}

func (r *Registry) GetActiveTenants() []tenant.Model {
	ctx := context.Background()
	members, err := r.tenants.Members(ctx)
	if err != nil {
		return nil
	}
	var tenants []tenant.Model
	for _, m := range members {
		var t tenant.Model
		if err := json.Unmarshal([]byte(m), &t); err != nil {
			continue
		}
		tenants = append(tenants, t)
	}
	return tenants
}
```

Remove the now-unused `goredis` import only if no other reference remains
(`InitRegistry`'s parameter still uses it — keep it).

- [ ] **Step 4: Run** (from `services/atlas-invites/atlas.com/invites/`):
`go test -race ./... && go vet ./... && go build ./...` → clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-invites/atlas.com/invites/invite/
git commit -m "fix(atlas-invites): namespace invite:active-tenants via atlas-redis Set"
```

### Task 7: atlas-guilds — coordinator registry

Migrate three keys: `coordinator:active`→`Set`, `coordinator:agreement:<uuid>`→
`Registry[uuid.UUID, Model]` (env-global), `coordinator:char:<tk>:<id>`→
`TenantRegistry[uint32, string]` (the stored value is the agreement-id string).

**Files:**
- Modify: `services/atlas-guilds/atlas.com/guilds/coordinator/registry.go`
- Test: existing `registry_test.go` in the package (update to new behavior).

- [ ] **Step 1: Add/adapt a behavior test** — `Initiate` then `GetExpired`
  (with a tiny timeout) returns the agreement; `Respond(false)` removes it.
  Use the package's miniredis helper (add one if absent).

- [ ] **Step 2: Run** the test → PASS pre-change.

- [ ] **Step 3: Rewrite `registry.go`**. Full new file:

```go
package coordinator

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	active     *atlas.Set                                 // agreement-id strings
	agreements *atlas.Registry[uuid.UUID, Model]          // agreement-id -> Model
	charAgree  *atlas.TenantRegistry[uint32, string]      // characterId -> agreement-id string
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{
		active:     atlas.NewSet(client, "coordinator:active"),
		agreements: atlas.NewRegistry[uuid.UUID, Model](client, "coordinator:agreement", func(id uuid.UUID) string { return id.String() }),
		charAgree:  atlas.NewTenantRegistry[uint32, string](client, "coordinator:char", func(id uint32) string { return strconv.FormatUint(uint64(id), 10) }),
	}
}

func GetRegistry() *Registry {
	return registry
}

func (r *Registry) Initiate(ctx context.Context, ch channel.Model, name string, leaderId uint32, members []uint32) error {
	t := tenant.MustFromContext(ctx)

	for _, m := range members {
		val, err := r.charAgree.Get(ctx, t, m)
		if err == nil && val != "" && val != uuid.Nil.String() {
			return errors.New("already attempting guild creation")
		}
	}

	agreementId := uuid.New()
	rm := make(map[uint32]bool)
	rm[leaderId] = true

	mdl := Model{
		tenant:    t,
		channel:   ch,
		leaderId:  leaderId,
		name:      name,
		requests:  members,
		responses: rm,
		age:       time.Now(),
	}

	for _, memberId := range members {
		if err := r.charAgree.Put(ctx, t, memberId, agreementId.String()); err != nil {
			return fmt.Errorf("track member agreement: %w", err)
		}
	}
	if err := r.agreements.Put(ctx, agreementId, mdl); err != nil {
		return fmt.Errorf("store agreement: %w", err)
	}
	return r.active.Add(ctx, agreementId.String())
}

func (r *Registry) Respond(ctx context.Context, characterId uint32, agree bool) (Model, error) {
	t := tenant.MustFromContext(ctx)

	agreementIdStr, err := r.charAgree.Get(ctx, t, characterId)
	if err != nil {
		return Model{}, fmt.Errorf("character not in agreement: %w", err)
	}
	agreementId, err := uuid.Parse(agreementIdStr)
	if err != nil {
		return Model{}, fmt.Errorf("parse agreement id: %w", err)
	}
	g, err := r.agreements.Get(ctx, agreementId)
	if err != nil {
		return Model{}, fmt.Errorf("agreement not found: %w", err)
	}

	if agree {
		g = g.Agree(characterId)
		_ = r.agreements.Put(ctx, agreementId, g)
		return g, nil
	}

	// Disagreed — delete the agreement and clear character mappings.
	_ = r.agreements.Remove(ctx, agreementId)
	_ = r.active.Remove(ctx, agreementId.String())
	for _, m := range g.requests {
		_ = r.charAgree.Put(ctx, t, m, uuid.Nil.String())
	}
	return g, nil
}

func (r *Registry) GetExpired(timeout time.Duration) ([]Model, error) {
	ctx := context.Background()
	members, err := r.active.Members(ctx)
	if err != nil {
		return nil, fmt.Errorf("get active agreements: %w", err)
	}
	now := time.Now()
	results := make([]Model, 0)
	for _, idStr := range members {
		id, err := uuid.Parse(idStr)
		if err != nil {
			continue
		}
		g, err := r.agreements.Get(ctx, id)
		if err != nil {
			continue
		}
		if now.Sub(g.Age()) > timeout {
			results = append(results, g)
		}
	}
	return results, nil
}
```

> Notes: the agreement `Model` must JSON round-trip through `Registry`'s
> default `json.Marshal`/`Unmarshal`. The original code already marshaled the
> same `Model` via `json.Marshal(&m)`, so its JSON tags are intact — confirm by
> running the package tests. `charAgree` values are agreement-id strings, not
> Models (this corrects the design table; see context.md).

- [ ] **Step 4: Run** (from `services/atlas-guilds/atlas.com/guilds/`):
`go test -race ./... && go vet ./... && go build ./...` → clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-guilds/atlas.com/guilds/coordinator/
git commit -m "fix(atlas-guilds): namespace coordinator keys via atlas-redis"
```

### Task 8: atlas-drops — drop registry

`drops:all`→`Set`; `drop:<t>:<id>`→`TenantRegistry[uint32, dropEntry]`;
`drops:map:<t>:<field>`→`TenantKeyedSet[field.Model]`.

**Files:**
- Modify: `services/atlas-drops/atlas.com/drops/drop/registry.go`
- Test: existing `drop/*_test.go` (update key-shape expectations).

- [ ] **Step 1: Adapt tests** — `CreateDrop` then `GetDropsForMap`,
  `GetDrop`, `GetAllDrops`, `RemoveDrop` round-trip. Keep the existing
  `allSetMember` `"<uuid>:<id>"` member encoding so `GetAllDrops`
  reconstruction is unchanged.

- [ ] **Step 2: Run** the package tests → PASS pre-change.

- [ ] **Step 3: Rewrite `registry.go`**. Full new file:

```go
package drop

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type dropEntry struct {
	Drop       Model  `json:"drop"`
	ReservedBy uint32 `json:"reservedBy"`
}

type DropRegistry struct {
	entries   *atlas.TenantRegistry[uint32, dropEntry]
	all       *atlas.Set
	mapSets   *atlas.TenantKeyedSet[field.Model]
	allocator objectid.Allocator
}

var registry *DropRegistry

func InitRegistry(client *goredis.Client) {
	registry = &DropRegistry{
		entries: atlas.NewTenantRegistry[uint32, dropEntry](client, "drop", func(id uint32) string {
			return strconv.FormatUint(uint64(id), 10)
		}),
		all: atlas.NewSet(client, "drops:all"),
		mapSets: atlas.NewTenantKeyedSet[field.Model](client, "drops:map", func(f field.Model) string {
			return fmt.Sprintf("%d:%d:%d:%s", f.WorldId(), f.ChannelId(), f.MapId(), f.Instance().String())
		}),
		allocator: objectid.NewRedisAllocator(client),
	}
}

func GetRegistry() *DropRegistry {
	return registry
}

func dropIdStr(id uint32) string {
	return strconv.FormatUint(uint64(id), 10)
}

// allSetMember encodes a tenant+id pair for the global drops:all set.
func allSetMember(t tenant.Model, id uint32) string {
	return fmt.Sprintf("%s:%d", t.Id().String(), id)
}

func (d *DropRegistry) loadEntry(t tenant.Model, id uint32) (dropEntry, bool) {
	entry, err := d.entries.Get(context.Background(), t, id)
	if err != nil {
		return dropEntry{}, false
	}
	return entry, true
}

func (d *DropRegistry) CreateDrop(mb *ModelBuilder) (Model, error) {
	t := mb.Tenant()
	ctx := context.Background()

	id, err := d.allocator.Allocate(ctx, t)
	if err != nil {
		return Model{}, fmt.Errorf("allocate drop oid: %w", err)
	}

	drop, err := mb.SetId(id).SetStatus(StatusAvailable).Build()
	if err != nil {
		_ = d.allocator.Release(ctx, t, id)
		return Model{}, err
	}

	entry := dropEntry{Drop: drop}
	if err := d.entries.Put(ctx, t, drop.Id(), entry); err != nil {
		_ = d.allocator.Release(ctx, t, id)
		return Model{}, err
	}

	_ = d.all.Add(ctx, allSetMember(t, drop.Id()))
	_ = d.mapSets.Add(ctx, t, mb.field, dropIdStr(drop.Id()))

	return drop, nil
}

func (d *DropRegistry) getDrop(t tenant.Model, dropId uint32) (Model, bool) {
	entry, ok := d.loadEntry(t, dropId)
	if !ok {
		return Model{}, false
	}
	return entry.Drop, true
}

func (d *DropRegistry) GetDrop(t tenant.Model, dropId uint32) (Model, error) {
	drop, ok := d.getDrop(t, dropId)
	if !ok {
		return Model{}, errors.New("drop not found")
	}
	return drop, nil
}

func (d *DropRegistry) ReserveDrop(t tenant.Model, dropId uint32, characterId uint32, partyId uint32, petSlot int8) (Model, error) {
	entry, ok := d.loadEntry(t, dropId)
	if !ok {
		return Model{}, errors.New("unable to locate drop")
	}
	if !entry.Drop.CanBeReservedBy(characterId, partyId) {
		return Model{}, errors.New("drop is not available for this character")
	}
	if entry.Drop.Status() == StatusAvailable {
		entry.Drop = entry.Drop.Reserve(petSlot)
		entry.ReservedBy = characterId
		if err := d.entries.Put(context.Background(), t, dropId, entry); err != nil {
			return Model{}, err
		}
		return entry.Drop, nil
	}
	if entry.ReservedBy == characterId {
		return entry.Drop, nil
	}
	return Model{}, errors.New("reserved by another party")
}

func (d *DropRegistry) CancelDropReservation(t tenant.Model, dropId uint32, characterId uint32) {
	entry, ok := d.loadEntry(t, dropId)
	if !ok {
		return
	}
	if entry.ReservedBy != characterId {
		return
	}
	if entry.Drop.Status() != StatusReserved {
		return
	}
	entry.Drop = entry.Drop.CancelReservation()
	entry.ReservedBy = 0
	_ = d.entries.Put(context.Background(), t, dropId, entry)
}

func (d *DropRegistry) RemoveDrop(t tenant.Model, dropId uint32) (Model, error) {
	entry, ok := d.loadEntry(t, dropId)
	if !ok {
		return Model{}, nil
	}
	drop := entry.Drop
	ctx := context.Background()

	_ = d.entries.Remove(ctx, t, dropId)
	_ = d.all.Remove(ctx, allSetMember(t, dropId))
	_ = d.mapSets.Remove(ctx, t, drop.Field(), dropIdStr(dropId))
	_ = d.allocator.Release(ctx, t, dropId)

	return drop, nil
}

func (d *DropRegistry) GetDropsForMap(t tenant.Model, f field.Model) ([]Model, error) {
	members, err := d.mapSets.Members(context.Background(), t, f)
	if err != nil {
		return make([]Model, 0), nil
	}
	drops := make([]Model, 0, len(members))
	for _, member := range members {
		id, err := strconv.ParseUint(member, 10, 32)
		if err != nil {
			continue
		}
		if drop, ok := d.getDrop(t, uint32(id)); ok {
			drops = append(drops, drop)
		}
	}
	return drops, nil
}

func (d *DropRegistry) GetAllDrops() []Model {
	members, err := d.all.Members(context.Background())
	if err != nil {
		return nil
	}
	drops := make([]Model, 0, len(members))
	for _, member := range members {
		// Members are "{tenantId}:{id}". Skip legacy "{id}"-only rows.
		sep := strings.LastIndex(member, ":")
		if sep < 0 {
			continue
		}
		id, err := strconv.ParseUint(member[sep+1:], 10, 32)
		if err != nil {
			continue
		}
		tenantId, err := uuid.Parse(member[:sep])
		if err != nil {
			continue
		}
		te, err := tenant.Create(tenantId, "", 0, 0)
		if err != nil {
			continue
		}
		if drop, ok := d.getDrop(te, uint32(id)); ok {
			drops = append(drops, drop)
		}
	}
	return drops
}
```

> The `mb.field` access stays valid (same package). `dropKey`/`mapSetKey`
> helpers are gone — their key construction now lives in the lib. Removed the
> raw `goredis` pipeline usage (the analyzer would flag `pipe.SAdd`).

- [ ] **Step 4: Run** (from `services/atlas-drops/atlas.com/drops/`):
`go test -race ./... && go vet ./... && go build ./...` → clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-drops/atlas.com/drops/drop/
git commit -m "fix(atlas-drops): namespace drops keys via atlas-redis"
```

### Task 9: atlas-reactors — reactor registry (incl. cooldown/spot semantic change)

`reactors:all`→`Set`; `reactor:<t>:<id>`→`TenantRegistry[uint32, Model]`;
`reactors:map`→`TenantKeyedSet[MapKey]`; `reactor:cd`→`TenantKeyedHash[MapKey]`
(field=`class:x:y`, value=expiry-unix-ms); `reactor:spot`→`TenantKeyedHash[MapKey]`
(field=`class:x:y`, value=`"1"`, via `SetNX`). See context.md **P2** for the
cooldown-TTL semantic change.

**Files:**
- Modify: `services/atlas-reactors/atlas.com/reactors/reactor/registry.go`
- Test: existing `reactor/*_test.go` (update; add cooldown-expiry test).

- [ ] **Step 1: Adapt/add tests**

Add a cooldown-expiry test reflecting the new timestamp semantics:

```go
func TestCooldown_ExpiresByTimestamp(t *testing.T) {
	client, _ := setupTestRedis(t) // package miniredis helper
	InitRegistry(client)
	r := GetRegistry()
	te := testTenant()
	// field/world/channel/_map are already imported by registry.go.
	mk := NewMapKey(field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000)).Build())
	r.RecordCooldown(te, mk, 9101000, 100, 200, 50) // 50ms delay
	if !r.IsOnCooldown(te, mk, 9101000, 100, 200) {
		t.Fatalf("expected on cooldown immediately after record")
	}
	time.Sleep(70 * time.Millisecond)
	if r.IsOnCooldown(te, mk, 9101000, 100, 200) {
		t.Fatalf("expected cooldown expired after delay")
	}
}

func TestSpot_ClaimIsExclusivePerPosition(t *testing.T) {
	client, _ := setupTestRedis(t)
	InitRegistry(client)
	r := GetRegistry()
	te := testTenant()
	mk := NewMapKey(field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100000)).Build())
	if !r.TryClaimSpot(te, mk, 9101000, 10, 20) {
		t.Fatalf("first claim should succeed")
	}
	if r.TryClaimSpot(te, mk, 9101000, 10, 20) {
		t.Fatalf("second claim on same spot must fail")
	}
	r.ReleaseSpot(te, mk, 9101000, 10, 20)
	if !r.TryClaimSpot(te, mk, 9101000, 10, 20) {
		t.Fatalf("claim after release should succeed")
	}
}
```

- [ ] **Step 2: Run** these → FAIL to compile (helpers not yet added) or behavior
  mismatch; that's expected until Step 3.

- [ ] **Step 3: Rewrite `registry.go`**. Full new file:

```go
package reactor

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	objectid "github.com/Chronicle20/atlas/libs/atlas-object-id"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	reactors  *atlas.TenantRegistry[uint32, Model]
	all       *atlas.Set
	mapSets   *atlas.TenantKeyedSet[MapKey]
	cooldowns *atlas.TenantKeyedHash[MapKey] // field=class:x:y -> expiry unix ms
	spots     *atlas.TenantKeyedHash[MapKey] // field=class:x:y -> "1"
	allocator objectid.Allocator
}

var reg *Registry

func mapKeyFn(mk MapKey) string {
	return fmt.Sprintf("%d:%d:%d:%s", mk.worldId, mk.channelId, mk.mapId, mk.instance.String())
}

func InitRegistry(client *goredis.Client) {
	reg = &Registry{
		reactors: atlas.NewTenantRegistry[uint32, Model](client, "reactor", func(id uint32) string {
			return strconv.FormatUint(uint64(id), 10)
		}),
		all:       atlas.NewSet(client, "reactors:all"),
		mapSets:   atlas.NewTenantKeyedSet[MapKey](client, "reactors:map", mapKeyFn),
		cooldowns: atlas.NewTenantKeyedHash[MapKey](client, "reactor:cd", mapKeyFn),
		spots:     atlas.NewTenantKeyedHash[MapKey](client, "reactor:spot", mapKeyFn),
		allocator: objectid.NewRedisAllocator(client),
	}
}

func GetRegistry() *Registry {
	return reg
}

type MapKey struct {
	worldId   world.Id
	channelId channel.Id
	mapId     _map.Id
	instance  uuid.UUID
}

func NewMapKey(f field.Model) MapKey {
	return MapKey{
		worldId:   f.WorldId(),
		channelId: f.ChannelId(),
		mapId:     f.MapId(),
		instance:  f.Instance(),
	}
}

func reactorIdStr(id uint32) string {
	return strconv.FormatUint(uint64(id), 10)
}

// allSetMember encodes a tenant+id pair for the global reactors:all set.
func allSetMember(t tenant.Model, id uint32) string {
	return fmt.Sprintf("%s:%d", t.Id().String(), id)
}

// posField is the hash field for cooldown/spot entries within a map hash.
func posField(classification uint32, x int16, y int16) string {
	return fmt.Sprintf("%d:%d:%d", classification, x, y)
}

func (r *Registry) load(t tenant.Model, id uint32) (Model, bool) {
	m, err := r.reactors.Get(context.Background(), t, id)
	if err != nil {
		return Model{}, false
	}
	return m, true
}

func (r *Registry) Get(t tenant.Model, id uint32) (Model, error) {
	m, ok := r.load(t, id)
	if !ok {
		return Model{}, errors.New("unable to locate reactor")
	}
	return m, nil
}

func (r *Registry) GetAll() map[tenant.Model][]Model {
	members, err := r.all.Members(context.Background())
	if err != nil {
		return make(map[tenant.Model][]Model)
	}
	res := make(map[tenant.Model][]Model)
	for _, member := range members {
		sep := -1
		for i := len(member) - 1; i >= 0; i-- {
			if member[i] == ':' {
				sep = i
				break
			}
		}
		if sep < 0 {
			continue
		}
		id, err := strconv.ParseUint(member[sep+1:], 10, 32)
		if err != nil {
			continue
		}
		tenantId, err := uuid.Parse(member[:sep])
		if err != nil {
			continue
		}
		te, err := tenant.Create(tenantId, "", 0, 0)
		if err != nil {
			continue
		}
		if m, ok := r.load(te, uint32(id)); ok {
			res[m.Tenant()] = append(res[m.Tenant()], m)
		}
	}
	return res
}

func (r *Registry) GetInField(t tenant.Model, f field.Model) []Model {
	mk := NewMapKey(f)
	members, err := r.mapSets.Members(context.Background(), t, mk)
	if err != nil {
		return make([]Model, 0)
	}
	result := make([]Model, 0, len(members))
	for _, member := range members {
		id, err := strconv.ParseUint(member, 10, 32)
		if err != nil {
			continue
		}
		if m, ok := r.load(t, uint32(id)); ok {
			result = append(result, m)
		}
	}
	return result
}

func (r *Registry) Create(t tenant.Model, b *ModelBuilder) (Model, error) {
	ctx := context.Background()
	id, err := r.allocator.Allocate(ctx, t)
	if err != nil {
		return Model{}, fmt.Errorf("allocate reactor oid: %w", err)
	}
	m, err := b.SetId(id).UpdateTime().Build()
	if err != nil {
		_ = r.allocator.Release(ctx, t, id)
		return Model{}, err
	}
	if err := r.reactors.Put(ctx, t, id, m); err != nil {
		_ = r.allocator.Release(ctx, t, id)
		return Model{}, err
	}
	mk := NewMapKey(m.Field())
	_ = r.all.Add(ctx, allSetMember(t, id))
	_ = r.mapSets.Add(ctx, t, mk, reactorIdStr(id))
	return m, nil
}

func (r *Registry) Update(t tenant.Model, id uint32, modifier func(*ModelBuilder)) (Model, error) {
	m, ok := r.load(t, id)
	if !ok {
		return Model{}, errors.New("unable to locate reactor")
	}
	b := NewFromModel(m)
	modifier(b)
	b.UpdateTime()
	updated, err := b.Build()
	if err != nil {
		return Model{}, err
	}
	if err := r.reactors.Put(context.Background(), t, id, updated); err != nil {
		return Model{}, err
	}
	return updated, nil
}

func (r *Registry) Remove(t tenant.Model, id uint32) {
	m, ok := r.load(t, id)
	if !ok {
		return
	}
	ctx := context.Background()
	mk := NewMapKey(m.Field())
	_ = r.reactors.Remove(ctx, t, id)
	_ = r.all.Remove(ctx, allSetMember(t, id))
	_ = r.mapSets.Remove(ctx, t, mk, reactorIdStr(id))
	_ = r.allocator.Release(ctx, t, id)
}

func (r *Registry) RecordCooldown(t tenant.Model, mk MapKey, classification uint32, x int16, y int16, delay uint32) {
	if delay == 0 {
		return
	}
	expiry := time.Now().Add(time.Millisecond * time.Duration(delay)).UnixMilli()
	_ = r.cooldowns.Set(context.Background(), t, mk, posField(classification, x, y), strconv.FormatInt(expiry, 10))
}

func (r *Registry) IsOnCooldown(t tenant.Model, mk MapKey, classification uint32, x int16, y int16) bool {
	v, err := r.cooldowns.Get(context.Background(), t, mk, posField(classification, x, y))
	if err != nil {
		return false
	}
	expiry, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return false
	}
	if time.Now().UnixMilli() >= expiry {
		// Lazily prune the stale field.
		_ = r.cooldowns.Del(context.Background(), t, mk, posField(classification, x, y))
		return false
	}
	return true
}

func (r *Registry) ClearCooldown(t tenant.Model, mk MapKey, classification uint32, x int16, y int16) {
	_ = r.cooldowns.Del(context.Background(), t, mk, posField(classification, x, y))
}

func (r *Registry) ClearAllCooldownsForMap(t tenant.Model, mk MapKey) {
	_ = r.cooldowns.DeleteKey(context.Background(), t, mk)
}

func (r *Registry) CleanupExpiredCooldowns() {
	// No-op: cooldowns are pruned lazily in IsOnCooldown and cleared per-map.
}

// TryClaimSpot atomically reserves a (classification, x, y) slot within a map
// instance. Returns true if this caller owns the slot.
func (r *Registry) TryClaimSpot(t tenant.Model, mk MapKey, classification uint32, x int16, y int16) bool {
	ok, err := r.spots.SetNX(context.Background(), t, mk, posField(classification, x, y), "1")
	if err != nil {
		return false
	}
	return ok
}

func (r *Registry) ReleaseSpot(t tenant.Model, mk MapKey, classification uint32, x int16, y int16) {
	_ = r.spots.Del(context.Background(), t, mk, posField(classification, x, y))
}

func (r *Registry) ClearAllSpotsForMap(t tenant.Model, mk MapKey) {
	_ = r.spots.DeleteKey(context.Background(), t, mk)
}
```

> Removed: `reactorKey`, `mapSetKey`, `cooldownKey`, `spotKey`,
> `cooldownPatternKey`, `spotPatternKey`, `store`/`load` raw helpers, and the two
> raw `client.Scan` loops (`ClearAllCooldownsForMap`/`ClearAllSpotsForMap` are now
> single `DeleteKey` calls). This eliminates every raw keyed client call.

- [ ] **Step 4: Run** (from `services/atlas-reactors/atlas.com/reactors/`):
`go test -race ./... && go vet ./... && go build ./...` → clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-reactors/atlas.com/reactors/reactor/
git commit -m "fix(atlas-reactors): namespace reactor keys via atlas-redis (cooldown/spot as hashes)"
```

### Task 10: atlas-transports — instance registry

`transport:instances`→`Set`; `transport:instance:<id>`→`Registry[uuid, transportInstanceJSON]`;
`transport:instance:<id>:chars`→`KeyedHash[uuid]`; `transport:route:<t>:<route>`→
`TenantKeyedSet[uuid]` (tenant reconstructed from the stored tenantId UUID).

**Files:**
- Modify: `services/atlas-transports/atlas.com/transports/instance/instance_registry.go`
- Test: existing `instance/*_test.go`.

- [ ] **Step 1: Adapt tests** — `FindOrCreateInstance`, `AddCharacter`,
  `RemoveCharacter`, `GetInstancesByRoute`, `ReleaseInstance` round-trip.

- [ ] **Step 2: Run** → PASS pre-change.

- [ ] **Step 3: Rewrite `instance_registry.go`**. Full new file:

```go
package instance

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type transportInstanceJSON struct {
	InstanceId    uuid.UUID     `json:"instanceId"`
	RouteId       uuid.UUID     `json:"routeId"`
	TenantId      uuid.UUID     `json:"tenantId"`
	State         InstanceState `json:"state"`
	BoardingUntil time.Time     `json:"boardingUntil"`
	ArrivalAt     time.Time     `json:"arrivalAt"`
	CreatedAt     time.Time     `json:"createdAt"`
}

func toJSON(inst TransportInstance) transportInstanceJSON {
	return transportInstanceJSON{
		InstanceId:    inst.instanceId,
		RouteId:       inst.routeId,
		TenantId:      inst.tenantId,
		State:         inst.state,
		BoardingUntil: inst.boardingUntil,
		ArrivalAt:     inst.arrivalAt,
		CreatedAt:     inst.createdAt,
	}
}

func fromJSON(j transportInstanceJSON) TransportInstance {
	return TransportInstance{
		instanceId:    j.InstanceId,
		routeId:       j.RouteId,
		tenantId:      j.TenantId,
		state:         j.State,
		boardingUntil: j.BoardingUntil,
		arrivalAt:     j.ArrivalAt,
		createdAt:     j.CreatedAt,
		characters:    make([]CharacterEntry, 0),
	}
}

// tenantFromId reconstructs a region-less tenant.Model from a bare UUID, used
// to scope the per-route SET. The region/version segments are unused by routing
// and repopulate; this only affects the (prefixed) Redis key shape.
func tenantFromId(id uuid.UUID) (tenant.Model, bool) {
	t, err := tenant.Create(id, "", 0, 0)
	if err != nil {
		return tenant.Model{}, false
	}
	return t, true
}

type InstanceRegistry struct {
	all   *atlas.Set
	meta  *atlas.Registry[uuid.UUID, transportInstanceJSON]
	chars *atlas.KeyedHash[uuid.UUID]
	routes *atlas.TenantKeyedSet[uuid.UUID]
}

var instanceRegistry *InstanceRegistry

func InitInstanceRegistry(client *goredis.Client) {
	instanceRegistry = &InstanceRegistry{
		all: atlas.NewSet(client, "transport:instances"),
		meta: atlas.NewRegistry[uuid.UUID, transportInstanceJSON](client, "transport:instance", func(id uuid.UUID) string {
			return id.String()
		}),
		chars: atlas.NewKeyedHash[uuid.UUID](client, "transport:instance:chars", func(id uuid.UUID) string {
			return id.String()
		}),
		routes: atlas.NewTenantKeyedSet[uuid.UUID](client, "transport:route", func(id uuid.UUID) string {
			return id.String()
		}),
	}
}

func getInstanceRegistry() *InstanceRegistry {
	return instanceRegistry
}

func (r *InstanceRegistry) storeMetadata(inst TransportInstance) {
	ctx := context.Background()
	_ = r.meta.Put(ctx, inst.instanceId, toJSON(inst))
	_ = r.all.Add(ctx, inst.instanceId.String())
	if t, ok := tenantFromId(inst.tenantId); ok {
		_ = r.routes.Add(ctx, t, inst.routeId, inst.instanceId.String())
	}
}

func (r *InstanceRegistry) loadMetadata(id uuid.UUID) (TransportInstance, bool) {
	j, err := r.meta.Get(context.Background(), id)
	if err != nil {
		return TransportInstance{}, false
	}
	return fromJSON(j), true
}

func (r *InstanceRegistry) loadCharacters(id uuid.UUID) []CharacterEntry {
	charMap, err := r.chars.GetAll(context.Background(), id)
	if err != nil {
		return nil
	}
	chars := make([]CharacterEntry, 0, len(charMap))
	for _, v := range charMap {
		var entry CharacterEntry
		if err := json.Unmarshal([]byte(v), &entry); err == nil {
			chars = append(chars, entry)
		}
	}
	return chars
}

func (r *InstanceRegistry) loadInstance(id uuid.UUID) (TransportInstance, bool) {
	inst, ok := r.loadMetadata(id)
	if !ok {
		return TransportInstance{}, false
	}
	chars := r.loadCharacters(id)
	if chars != nil {
		inst.characters = chars
	}
	return inst, true
}

func (r *InstanceRegistry) FindOrCreateInstance(tenantId uuid.UUID, route RouteModel, now time.Time) TransportInstance {
	ctx := context.Background()
	if t, ok := tenantFromId(tenantId); ok {
		members, err := r.routes.Members(ctx, t, route.Id())
		if err == nil {
			for _, member := range members {
				id, err := uuid.Parse(member)
				if err != nil {
					continue
				}
				inst, ok := r.loadMetadata(id)
				if !ok {
					continue
				}
				if inst.state != Boarding || !now.Before(inst.boardingUntil) {
					continue
				}
				count, err := r.chars.Len(ctx, id)
				if err != nil {
					continue
				}
				if uint32(count) < route.Capacity() {
					return inst
				}
			}
		}
	}

	instanceId := uuid.New()
	boardingUntil := now.Add(route.BoardingWindow())
	arrivalAt := boardingUntil.Add(route.TravelDuration())
	inst := NewTransportInstance(instanceId, route.Id(), tenantId, boardingUntil, arrivalAt)
	r.storeMetadata(inst)
	return inst
}

func (r *InstanceRegistry) AddCharacter(instanceId uuid.UUID, entry CharacterEntry) (bool, int) {
	ctx := context.Background()
	if _, ok := r.loadMetadata(instanceId); !ok {
		return false, 0
	}
	data, _ := json.Marshal(entry)
	_ = r.chars.Set(ctx, instanceId, strconv.FormatUint(uint64(entry.CharacterId), 10), string(data))
	count, _ := r.chars.Len(ctx, instanceId)
	return true, int(count)
}

func (r *InstanceRegistry) RemoveCharacter(instanceId uuid.UUID, characterId uint32) bool {
	ctx := context.Background()
	_ = r.chars.Del(ctx, instanceId, strconv.FormatUint(uint64(characterId), 10))
	count, err := r.chars.Len(ctx, instanceId)
	if err != nil {
		return false
	}
	return count == 0
}

func (r *InstanceRegistry) TransitionToInTransit(instanceId uuid.UUID) bool {
	inst, ok := r.loadMetadata(instanceId)
	if !ok || inst.state != Boarding {
		return false
	}
	inst.state = InTransit
	r.storeMetadata(inst)
	return true
}

func (r *InstanceRegistry) ReleaseInstance(instanceId uuid.UUID) {
	ctx := context.Background()
	inst, ok := r.loadMetadata(instanceId)
	if !ok {
		return
	}
	if t, ok := tenantFromId(inst.tenantId); ok {
		_ = r.routes.Remove(ctx, t, inst.routeId, instanceId.String())
	}
	_ = r.all.Remove(ctx, instanceId.String())
	_ = r.meta.Remove(ctx, instanceId)
	_ = r.chars.DeleteKey(ctx, instanceId)
}

func (r *InstanceRegistry) GetInstance(instanceId uuid.UUID) (TransportInstance, bool) {
	return r.loadInstance(instanceId)
}

func (r *InstanceRegistry) GetExpiredBoarding(now time.Time) []TransportInstance {
	return r.filterInstances(func(inst TransportInstance) bool {
		return inst.state == Boarding && now.After(inst.boardingUntil)
	})
}

func (r *InstanceRegistry) GetExpiredTransit(now time.Time) []TransportInstance {
	return r.filterInstances(func(inst TransportInstance) bool {
		return inst.state == InTransit && now.After(inst.arrivalAt)
	})
}

func (r *InstanceRegistry) GetAllActive() []TransportInstance {
	return r.filterInstances(func(inst TransportInstance) bool { return true })
}

func (r *InstanceRegistry) GetStuck(now time.Time, maxLifetime time.Duration) []TransportInstance {
	return r.filterInstances(func(inst TransportInstance) bool {
		return now.Sub(inst.createdAt) > maxLifetime
	})
}

func (r *InstanceRegistry) GetInstancesByRoute(tenantId, routeId uuid.UUID) []TransportInstance {
	t, ok := tenantFromId(tenantId)
	if !ok {
		return nil
	}
	members, err := r.routes.Members(context.Background(), t, routeId)
	if err != nil {
		return nil
	}
	var result []TransportInstance
	for _, member := range members {
		id, err := uuid.Parse(member)
		if err != nil {
			continue
		}
		inst, ok := r.loadInstance(id)
		if !ok {
			continue
		}
		result = append(result, inst)
	}
	return result
}

func (r *InstanceRegistry) filterInstances(predicate func(TransportInstance) bool) []TransportInstance {
	members, err := r.all.Members(context.Background())
	if err != nil {
		return nil
	}
	var result []TransportInstance
	for _, member := range members {
		id, err := uuid.Parse(member)
		if err != nil {
			continue
		}
		inst, ok := r.loadInstance(id)
		if !ok {
			continue
		}
		if predicate(inst) {
			result = append(result, inst)
		}
	}
	return result
}
```

> `NewTransportInstance`, `TransportInstance` fields, `InstanceState`,
> `RouteModel`, `CharacterEntry` are defined elsewhere in the package and
> unchanged. The old `marshalInstanceMetadata`/`unmarshalInstanceMetadata`
> free functions are replaced by `toJSON`/`fromJSON`; if other files in the
> package referenced the old names, update those call sites (grep the package).

- [ ] **Step 4: Run** (from `services/atlas-transports/atlas.com/transports/`):
`go test -race ./... && go vet ./... && go build ./...` → clean. Fix any
references to the removed `marshalInstanceMetadata`/`unmarshalInstanceMetadata`.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/instance/instance_registry.go
git commit -m "fix(atlas-transports): namespace transport instance keys via atlas-redis"
```

### Task 11: atlas-transports — character registry (`transport:characters` → `Hash`)

**Files:**
- Modify: `services/atlas-transports/atlas.com/transports/instance/character_registry.go`

- [ ] **Step 1: Adapt test** — `Add`/`IsInTransport`/`GetInstanceForCharacter`/
  `Remove` round-trip (likely an existing test).

- [ ] **Step 2: Run** → PASS pre-change.

- [ ] **Step 3: Rewrite `character_registry.go`**. Full new file:

```go
package instance

import (
	"context"
	"strconv"

	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
)

type CharacterRegistry struct {
	chars *atlas.Hash
}

var characterRegistry *CharacterRegistry

func InitCharacterRegistry(client *goredis.Client) {
	characterRegistry = &CharacterRegistry{chars: atlas.NewHash(client, "transport:characters")}
}

func getCharacterRegistry() *CharacterRegistry {
	return characterRegistry
}

func (r *CharacterRegistry) Add(characterId uint32, instanceId uuid.UUID) {
	_ = r.chars.Set(context.Background(), strconv.FormatUint(uint64(characterId), 10), instanceId.String())
}

func (r *CharacterRegistry) Remove(characterId uint32) {
	_ = r.chars.Del(context.Background(), strconv.FormatUint(uint64(characterId), 10))
}

func (r *CharacterRegistry) IsInTransport(characterId uint32) bool {
	ok, err := r.chars.Exists(context.Background(), strconv.FormatUint(uint64(characterId), 10))
	if err != nil {
		return false
	}
	return ok
}

func (r *CharacterRegistry) GetInstanceForCharacter(characterId uint32) (uuid.UUID, bool) {
	val, err := r.chars.Get(context.Background(), strconv.FormatUint(uint64(characterId), 10))
	if err != nil {
		return uuid.UUID{}, false
	}
	instanceId, err := uuid.Parse(val)
	if err != nil {
		return uuid.UUID{}, false
	}
	return instanceId, true
}
```

- [ ] **Step 4: Run** the transports module: `go test -race ./... && go vet ./... && go build ./...` → clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/instance/character_registry.go
git commit -m "fix(atlas-transports): namespace transport:characters via atlas-redis Hash"
```

### Task 12: atlas-transports — channel registry (`transport:channels:<tk>` → `TenantSet`)

**Files:**
- Modify: `services/atlas-transports/atlas.com/transports/channel/registry.go`

- [ ] **Step 1: Adapt test** — `Add`/`GetAll`/`Remove` round-trip per tenant.

- [ ] **Step 2: Run** → PASS pre-change.

- [ ] **Step 3: Rewrite `registry.go`**. Full new file:

```go
package channel

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	channelConstant "github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	atlas "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/Chronicle20/atlas/libs/atlas-tenant"
	goredis "github.com/redis/go-redis/v9"
)

type Registry struct {
	channels *atlas.TenantSet
}

var registry *Registry

func InitRegistry(client *goredis.Client) {
	registry = &Registry{channels: atlas.NewTenantSet(client, "transport:channels")}
}

func getRegistry() *Registry {
	return registry
}

func channelMember(ch channelConstant.Model) string {
	return fmt.Sprintf("%d:%d", ch.WorldId(), ch.Id())
}

func parseChannelMember(member string) (channelConstant.Model, bool) {
	parts := strings.SplitN(member, ":", 2)
	if len(parts) != 2 {
		return channelConstant.Model{}, false
	}
	worldId, err := strconv.Atoi(parts[0])
	if err != nil {
		return channelConstant.Model{}, false
	}
	channelId, err := strconv.Atoi(parts[1])
	if err != nil {
		return channelConstant.Model{}, false
	}
	return channelConstant.NewModel(world.Id(worldId), channelConstant.Id(channelId)), true
}

func (r *Registry) Add(ctx context.Context, model channelConstant.Model) {
	t := tenant.MustFromContext(ctx)
	_ = r.channels.Add(ctx, t, channelMember(model))
}

func (r *Registry) Remove(ctx context.Context, ch channelConstant.Model) {
	t := tenant.MustFromContext(ctx)
	_ = r.channels.Remove(ctx, t, channelMember(ch))
}

func (r *Registry) GetAll(ctx context.Context) []channelConstant.Model {
	t := tenant.MustFromContext(ctx)
	members, err := r.channels.Members(ctx, t)
	if err != nil {
		return nil
	}
	results := make([]channelConstant.Model, 0, len(members))
	for _, m := range members {
		if ch, ok := parseChannelMember(m); ok {
			results = append(results, ch)
		}
	}
	return results
}
```

- [ ] **Step 4: Run** the transports module → clean. Then run the full module
  `go test -race ./... && go vet ./... && go build ./...` once more for Tasks 10–12.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-transports/atlas.com/transports/channel/registry.go
git commit -m "fix(atlas-transports): namespace transport:channels via atlas-redis TenantSet"
```

### Task 13: atlas-rates — item tracker (`rates-items` raw scan → `TenantKeyedHash`)

Re-model per-character tracked items as one HASH per character (key by
characterId, field = templateId), eliminating the raw `client.Scan` in
`GetAllTrackedItems`.

**Files:**
- Modify: `services/atlas-rates/atlas.com/rates/character/item_tracker.go`
- Test: existing tracker test (update; assert `GetAllTrackedItems` returns
  tracked items without a raw scan).

- [ ] **Step 1: Adapt test** — `TrackItem` two items for one character; one for
  another; `GetAllTrackedItems(charA)` returns 2; `UntrackItem` removes one;
  `CleanupExpiredItems` removes expired coupons.

- [ ] **Step 2: Run** → PASS pre-change.

- [ ] **Step 3: Edit `item_tracker.go`** — replace the tracker type, ctor, and
  the five item methods. Keep `TrackedItem`, its `MarshalJSON`, `IsExpired`,
  `GetCouponMultiplier*`, `IsActiveAt`, and `itemSource` exactly as-is.

Remove `itemTrackerKey` struct. Replace from `type ItemTracker` through
`CleanupExpiredItems`:

```go
// ItemTracker tracks time-based rate items per character: one Redis HASH per
// (tenant, character), hash field = templateId, value = TrackedItem JSON.
type ItemTracker struct {
	items *atlas.TenantKeyedHash[uint32] // key = characterId
}

var itemTracker *ItemTracker

func GetItemTracker() *ItemTracker {
	return itemTracker
}

func InitItemTracker(client *goredis.Client) {
	itemTracker = &ItemTracker{
		items: atlas.NewTenantKeyedHash[uint32](client, "rates-items", func(characterId uint32) string {
			return strconv.FormatUint(uint64(characterId), 10)
		}),
	}
}

func templateField(templateId uint32) string {
	return strconv.FormatUint(uint64(templateId), 10)
}

func (t *ItemTracker) TrackItem(ctx context.Context, characterId uint32, item TrackedItem) {
	ten := tenant.MustFromContext(ctx)
	data, err := json.Marshal(item)
	if err != nil {
		return
	}
	_ = t.items.Set(ctx, ten, characterId, templateField(item.TemplateId), string(data))
}

func (t *ItemTracker) UntrackItem(ctx context.Context, characterId uint32, templateId uint32) {
	ten := tenant.MustFromContext(ctx)
	_ = t.items.Del(ctx, ten, characterId, templateField(templateId))
}

func (t *ItemTracker) UpdateEquippedSince(ctx context.Context, characterId uint32, templateId uint32, equippedSince *time.Time) {
	ten := tenant.MustFromContext(ctx)
	raw, err := t.items.Get(ctx, ten, characterId, templateField(templateId))
	if err != nil {
		return
	}
	var item TrackedItem
	if err := json.Unmarshal([]byte(raw), &item); err != nil {
		return
	}
	item.EquippedSince = equippedSince
	data, err := json.Marshal(item)
	if err != nil {
		return
	}
	_ = t.items.Set(ctx, ten, characterId, templateField(templateId), string(data))
}

func (t *ItemTracker) GetTrackedItem(ctx context.Context, characterId uint32, templateId uint32) (TrackedItem, bool) {
	ten := tenant.MustFromContext(ctx)
	raw, err := t.items.Get(ctx, ten, characterId, templateField(templateId))
	if err != nil {
		return TrackedItem{}, false
	}
	var item TrackedItem
	if err := json.Unmarshal([]byte(raw), &item); err != nil {
		return TrackedItem{}, false
	}
	return item, true
}

func (t *ItemTracker) GetAllTrackedItems(ctx context.Context, characterId uint32) []TrackedItem {
	ten := tenant.MustFromContext(ctx)
	all, err := t.items.GetAll(ctx, ten, characterId)
	if err != nil {
		return make([]TrackedItem, 0)
	}
	result := make([]TrackedItem, 0, len(all))
	for _, raw := range all {
		var item TrackedItem
		if err := json.Unmarshal([]byte(raw), &item); err != nil {
			continue
		}
		result = append(result, item)
	}
	return result
}

// ComputeItemRateFactors is unchanged below (keep existing body).
// CleanupExpiredItems is unchanged below (keep existing body — it calls
// GetAllTrackedItems and UntrackItem, both updated above).
```

Keep `ComputeItemRateFactors` and `CleanupExpiredItems` bodies as they are.
Remove the now-unused `goredis` import only if `InitItemTracker`'s param no
longer references it — it still does (`*goredis.Client`), so keep it. The
`strconv` import is still used.

- [ ] **Step 4: Run** (from `services/atlas-rates/atlas.com/rates/`):
`go test -race ./... && go vet ./... && go build ./...` → clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-rates/atlas.com/rates/character/item_tracker.go
git commit -m "fix(atlas-rates): route rates-items scan through atlas-redis TenantKeyedHash"
```

### Task 14: atlas-maps — spawn registry (fix scan literal; route raw ops through `KeyedHash`)

Keep the Lua scripts. Replace the raw `HGetAll`/`HSet`/`Scan`/`Del` calls and the
**bare scan literal `atlas:maps:spawn:%s:*`** (`registry.go:296`) with a
`KeyedHash[character.MapKey]` whose `keyFn` reproduces the **exact current
on-disk key** (`<uuid>:<world>:<channel>:<map>:<instance>`). Scripts run against
`kh.Key(mapKey)`. See context.md **P3**.

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/map/monster/registry.go`
- Test: existing `monster/registry_test.go` — add the regression test below.

- [ ] **Step 1: Add the failing regression test for the scan/write match**

```go
func TestFlushTenant_MatchesWriteKeyUnderEnvPrefix(t *testing.T) {
	// Reproduces the L296 bug: a write under <env>:atlas:maps:spawn:... must be
	// found and deleted by FlushTenant regardless of ATLAS_ENV.
	client, _ := setupTestRedis(t) // package miniredis helper
	InitRegistry(client)
	r := GetRegistry()
	ctx := context.Background()
	tid := uuid.New()
	te, err := tenant.Create(tid, "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	f := field.NewBuilder(world.Id(0), channel.Id(1), _map.Id(100100)).Build()
	mapKey := character.MapKey{Tenant: te, Field: f}
	if err := r.SetSpawnPointsForMap(ctx, mapKey, []*CooldownSpawnPoint{
		{SpawnPoint: monster2.SpawnPoint{Id: 1, Template: 100100}, NextSpawnAt: time.Now()},
	}); err != nil {
		t.Fatalf("SetSpawnPointsForMap: %v", err)
	}

	deleted, err := r.FlushTenant(ctx, logrus.New(), tid)
	if err != nil {
		t.Fatalf("FlushTenant: %v", err)
	}
	if deleted != 1 {
		t.Fatalf("FlushTenant deleted = %d, want 1 (scan/write key mismatch)", deleted)
	}
}
```

> Imports for this test: `character "atlas-maps/map/character"`,
> `monster2 "atlas-maps/data/map/monster"`, `field`, `world`, `channel`, `_map`
> (atlas-constants), `tenant`, `uuid`, `time`, `logrus`. If the package lacks a
> miniredis helper, add one (mirror
> `libs/atlas-redis/registry_test.go:setupTestRedis`).

- [ ] **Step 2: Run** `go test ./map/monster/ -run TestFlushTenant_ -v`
Expected: **FAIL — `deleted = 0, want 1`.** This is the real pre-existing bug:
the current write key embeds `mapKey.Tenant.String()`, which is the **verbose**
`tenant.Model.String()` form (`"Id [uuid] Region [GMS] Version [83.1]"`, with
spaces/brackets — confirmed at `libs/atlas-tenant/tenant.go:82`), while
`FlushTenant` scans the bare literal `atlas:maps:spawn:<bare-uuid>:*`. The two
never match, so `FlushTenant` deletes nothing today (worse than the `atlas:`
literal the design flagged). Step 3 fixes both halves to a single bare-uuid
representation produced by the lib.

- [ ] **Step 3: Edit `registry.go`** — make these precise changes:

1. Add the import and a `KeyedHash` field on the registry struct:
```go
import (
	// ...existing...
	"atlas-maps/map/character"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	"github.com/google/uuid"
	goredis "github.com/redis/go-redis/v9"
	// ...
)

type SpawnPointRegistry struct {
	client *goredis.Client
	hashes *atlasredis.KeyedHash[character.MapKey]
}
```

2. In `InitRegistry`, construct the `KeyedHash` with a keyFn reproducing the
   exact current key body (everything after `maps:spawn:`):
```go
func InitRegistry(rc *goredis.Client) {
	registryOnce.Do(func() {
		registryInstance = &SpawnPointRegistry{
			client: rc,
			hashes: atlasredis.NewKeyedHash[character.MapKey](rc, "maps:spawn", func(mk character.MapKey) string {
				// Bare tenant UUID (NOT mk.Tenant.String(), which is the verbose
				// "Id [..] Region [..] Version [..]" debug form). This is the
				// representation FlushTenant scans by, so write+flush match.
				return fmt.Sprintf("%s:%d:%d:%d:%s",
					mk.Tenant.Id().String(),
					mk.Field.WorldId(),
					mk.Field.ChannelId(),
					mk.Field.MapId(),
					mk.Field.Instance().String(),
				)
			}),
		}
	})
}
```

3. Replace `spawnHashKey` to delegate to the lib (keeps the script callers
   targeting the identical key):
```go
func spawnHashKey(mapKey character.MapKey) string {
	return registryInstance.hashes.Key(mapKey)
}
```

> Every Lua-script call site (`initializeScript.Run(ctx, r.client, []string{spawnHashKey(mapKey)}, …)`,
> `eligibleScript`, `updateCooldownsScript`, `resetCooldownScript`) already uses
> `spawnHashKey(...)`; they now target the lib-built key. `script.Run` passes the
> client as a value (analyzer-allowed). **No change to the scripts.**

4. Replace `GetSpawnPointsForMap`'s raw `HGetAll`:
```go
func (r *SpawnPointRegistry) GetSpawnPointsForMap(ctx context.Context, mapKey character.MapKey) ([]*CooldownSpawnPoint, bool) {
	entries, err := r.hashes.GetAll(ctx, mapKey)
	if err != nil || len(entries) == 0 {
		return nil, false
	}
	var spawnPoints []*CooldownSpawnPoint
	for _, value := range entries {
		var stored storedSpawnPoint
		if err := json.Unmarshal([]byte(value), &stored); err != nil {
			continue
		}
		spawnPoints = append(spawnPoints, fromStored(stored))
	}
	return spawnPoints, true
}
```

5. Replace `SetSpawnPointsForMap`'s raw pipeline HSet:
```go
func (r *SpawnPointRegistry) SetSpawnPointsForMap(ctx context.Context, mapKey character.MapKey, spawnPoints []*CooldownSpawnPoint) error {
	for _, csp := range spawnPoints {
		stored := toStored(csp.SpawnPoint, csp.NextSpawnAt)
		data, _ := json.Marshal(stored)
		if err := r.hashes.Set(ctx, mapKey, strconv.FormatUint(uint64(csp.SpawnPoint.Id), 10), string(data)); err != nil {
			return err
		}
	}
	return nil
}
```

6. Replace `Reset` (was a bare `KeyPrefix()+":maps:spawn:*"` scan):
```go
func (r *SpawnPointRegistry) Reset(ctx context.Context) {
	_, _ = r.hashes.Clear(ctx)
}
```

7. Replace `FlushTenant` (the L296 bug) with a lib-owned prefix clear:
```go
func (r *SpawnPointRegistry) FlushTenant(ctx context.Context, l logrus.FieldLogger, tenantId uuid.UUID) (int, error) {
	deleted, err := r.hashes.Clear(ctx, tenantId.String())
	if err != nil {
		l.WithError(err).Warnf("Spawn-registry flush failure for tenant [%s].", tenantId)
	}
	return deleted, err
}
```

> `Clear(ctx, tenantId.String())` scans `<prefix>:maps:spawn:<uuid>:*` —
> identical match to the write key's (now bare-uuid) tenant segment, fixing the
> read/write mismatch. Remove now-unused imports (`strconv` is still used in
> `SetSpawnPointsForMap`; check `fmt` still used by `InitRegistry`'s keyFn).

8. The log line at the old `registry.go:193` (a `Debugf`/`Infof` that
   interpolates `mapKey.Tenant.String()`) is cosmetic — leave it, or change it
   to `mapKey.Tenant.Id().String()` for tidiness. It does not affect any key.

> **Orphan note (main only):** changing the tenant segment from the verbose
> `tenant.Model.String()` form to the bare uuid orphans the pre-existing
> `atlas:maps:spawn:Id [<uuid>] Region [...]...` keys on `atlas-main`. Those are
> **atlas-prefixed**, so the FR-1.6 bare-key reclaim script (Task 18) must NOT
> match them (its `atlas:*` refusal guard protects live keys). They are dead,
> harmless, repopulating cache data. Document a one-time manual cleanup in the
> deploy runbook step that gates this PR:
> `redis-cli -u redis://redis.home:6379 --scan --pattern 'atlas:maps:spawn:Id *' | xargs -r redis-cli -u redis://redis.home:6379 DEL`
> (the literal space after `Id` cannot occur in the new bare-uuid keys, so this
> pattern only ever matches the old verbose orphans). This is operational, not
> code; note it in the PR description / runbook, not in `reclaim-main-bare-keys.sh`.

- [ ] **Step 4: Run** (from `services/atlas-maps/atlas.com/maps/`):
`go test -race ./... && go vet ./... && go build ./...` → clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-maps/atlas.com/maps/map/monster/registry.go
git commit -m "fix(atlas-maps): route spawn hash + flush scan through atlas-redis KeyedHash"
```

---

## Phase 3 — Regression guard analyzer (FR-1.5)

### Task 15: scaffold `tools/rediskeyguard` module

**Files:**
- Create: `tools/rediskeyguard/go.mod`
- Create: `tools/rediskeyguard/analyzer.go`
- Create: `tools/rediskeyguard/cmd/rediskeyguard/main.go`

- [ ] **Step 1: Create `go.mod`** (standalone — NOT added to `go.work`)

```
module github.com/Chronicle20/atlas/tools/rediskeyguard

go 1.25.5

require (
	github.com/redis/go-redis/v9 v9.19.0
	golang.org/x/tools v0.30.0
)
```

- [ ] **Step 2: Create `analyzer.go`**

```go
package rediskeyguard

import (
	"go/ast"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

const (
	libPkgPath     = "github.com/Chronicle20/atlas/libs/atlas-redis"
	goRedisPkgPath = "github.com/redis/go-redis/v9"
)

// bannedMethods are keyed Redis commands that take a key/field as their first
// argument. Calling any of these on the raw go-redis client/pipeliner outside
// the atlas-redis lib reintroduces the un-namespaced-key leak.
var bannedMethods = map[string]bool{
	"Set": true, "Get": true, "Del": true, "Exists": true, "Expire": true,
	"Scan": true, "Keys": true,
	"SAdd": true, "SRem": true, "SMembers": true, "SIsMember": true, "SCard": true,
	"HSet": true, "HSetNX": true, "HGet": true, "HDel": true, "HExists": true,
	"HGetAll": true, "HKeys": true, "HLen": true,
	"SetNX": true,
}

var Analyzer = &analysis.Analyzer{
	Name:     "rediskeyguard",
	Doc:      "bans keyed Redis commands on the raw go-redis client outside libs/atlas-redis",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

func run(pass *analysis.Pass) (interface{}, error) {
	// The lib itself is the sole allowlist.
	if pass.Pkg.Path() == libPkgPath {
		return nil, nil
	}
	insp := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	insp.Preorder([]ast.Node{(*ast.CallExpr)(nil)}, func(n ast.Node) {
		call := n.(*ast.CallExpr)
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return
		}
		if !bannedMethods[sel.Sel.Name] {
			return
		}
		tv, ok := pass.TypesInfo.Types[sel.X]
		if !ok {
			return
		}
		if !isGoRedisKeyedReceiver(tv.Type) {
			return
		}
		pass.Reportf(call.Pos(),
			"rediskeyguard: %s called on raw go-redis client/pipeliner; use a libs/atlas-redis type instead",
			sel.Sel.Name)
	})
	return nil, nil
}

func isGoRedisKeyedReceiver(t types.Type) bool {
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil || obj.Pkg().Path() != goRedisPkgPath {
		return false
	}
	switch obj.Name() {
	case "Client", "ClusterClient", "Conn", "Pipeliner", "Tx":
		return true
	}
	return false
}
```

- [ ] **Step 3: Create `cmd/rediskeyguard/main.go`**

```go
package main

import (
	"github.com/Chronicle20/atlas/tools/rediskeyguard"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	singlechecker.Main(rediskeyguard.Analyzer)
}
```

- [ ] **Step 4: Resolve deps and build**

Run (from `tools/rediskeyguard/`):
`go mod tidy && go build ./...`
Expected: clean; `go.sum` populated. (If `golang.org/x/tools` resolves to a
different patch, accept whatever `go mod tidy` pins.)

- [ ] **Step 5: Commit**

```bash
git add tools/rediskeyguard/go.mod tools/rediskeyguard/go.sum tools/rediskeyguard/analyzer.go tools/rediskeyguard/cmd/
git commit -m "feat(rediskeyguard): scaffold go vet-style redis key guard analyzer"
```

### Task 16: analyzer tests (`analysistest` good/bad)

**Files:**
- Create: `tools/rediskeyguard/analyzer_test.go`
- Create: `tools/rediskeyguard/testdata/src/bad/bad.go`
- Create: `tools/rediskeyguard/testdata/src/good/good.go`

- [ ] **Step 1: Write `testdata/src/bad/bad.go`** (must be flagged)

```go
package bad

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
)

func useRaw(client *goredis.Client) {
	ctx := context.Background()
	client.SAdd(ctx, "drops:all", "x")    // want `rediskeyguard: SAdd called on raw go-redis client`
	client.HSet(ctx, "h", "f", "v")       // want `rediskeyguard: HSet called on raw go-redis client`
	client.Scan(ctx, 0, "pat:*", 100)     // want `rediskeyguard: Scan called on raw go-redis client`
	_, _ = client.Get(ctx, "k").Result()  // want `rediskeyguard: Get called on raw go-redis client`
}
```

- [ ] **Step 2: Write `testdata/src/good/good.go`** (must NOT be flagged)

```go
package good

import (
	"context"

	goredis "github.com/redis/go-redis/v9"
)

// Passing the client as a value is allowed.
func wire(client *goredis.Client) *goredis.Client {
	return client
}

// Non-keyed commands and pipeline construction are allowed.
func allowed(client *goredis.Client) {
	ctx := context.Background()
	_ = client.Ping(ctx)
	_ = client.Pipeline()
}
```

- [ ] **Step 3: Write `analyzer_test.go`**

```go
package rediskeyguard_test

import (
	"testing"

	"github.com/Chronicle20/atlas/tools/rediskeyguard"
	"golang.org/x/tools/go/analysis/analysistest"
)

func TestAnalyzer(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, rediskeyguard.Analyzer, "bad", "good")
}
```

- [ ] **Step 4: Run** (from `tools/rediskeyguard/`)

`go mod tidy && go test ./...`
Expected: PASS — `bad` diagnostics match the `// want` regexps; `good` yields
none. If go-redis isn't fully resolved in testdata, `go mod tidy` pulls it
(the testdata imports it).

- [ ] **Step 5: Commit**

```bash
git add tools/rediskeyguard/analyzer_test.go tools/rediskeyguard/testdata/ tools/rediskeyguard/go.sum
git commit -m "test(rediskeyguard): analysistest good/bad fixtures"
```

### Task 17: runner script + CI + CLAUDE.md wiring

**Files:**
- Create: `tools/redis-key-guard.sh`
- Modify: `CLAUDE.md`
- Modify: a CI workflow under `.github/workflows/` (the Go verification job).

- [ ] **Step 1: Write `tools/redis-key-guard.sh`**

```bash
#!/usr/bin/env bash
# Build the rediskeyguard analyzer once, then run it over every Go service
# module. Non-empty diagnostics → non-zero exit. Run from the repo root.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
GUARD_SRC="$ROOT/tools/rediskeyguard"
BIN="$(mktemp -d)/rediskeyguard"

echo "building rediskeyguard..."
( cd "$GUARD_SRC" && go build -o "$BIN" ./cmd/rediskeyguard )

rc=0
# Every Go module that has a go.mod under services/ is a guard target.
while IFS= read -r modfile; do
    moddir="$(dirname "$modfile")"
    echo "rediskeyguard: $moddir"
    if ! ( cd "$moddir" && "$BIN" ./... ); then
        rc=1
    fi
done < <(find "$ROOT/services" -name go.mod -not -path '*/node_modules/*')

if [ "$rc" -ne 0 ]; then
    echo "rediskeyguard: FAIL — raw keyed redis client calls found (use a libs/atlas-redis type)"
fi
exit $rc
```

- [ ] **Step 2: Make it executable and run it**

Run: `chmod +x tools/redis-key-guard.sh && ./tools/redis-key-guard.sh`
Expected: PASS (exit 0) now that Phase 2 migrated all call sites. If any service
still trips it, fix that service (the diagnostic names the file:line + method).

- [ ] **Step 3: Add the CLAUDE.md verification line**

In `CLAUDE.md` under "Build & Verification", append item 5:
```markdown
5. **`tools/redis-key-guard.sh` clean from the repo root.** Bans keyed Redis
   commands on the raw `go-redis` client outside `libs/atlas-redis` (FR-1.5,
   task-045). Runs alongside `go vet ./...`.
```

- [ ] **Step 4: Wire into CI** — add a step to the Go verification workflow.
  Locate it (`grep -rl 'go vet' .github/workflows/`), then add, after the vet
  step, in the same job:
```yaml
      - name: redis key guard
        run: ./tools/redis-key-guard.sh
```
> Match the existing workflow's checkout/setup-go versions; do not add a new
> job. `tools/rediskeyguard` builds itself via `go build` inside the script, so
> no `go.work` change is needed.

- [ ] **Step 5: Commit**

```bash
git add tools/redis-key-guard.sh CLAUDE.md .github/workflows/
git commit -m "ci(rediskeyguard): run redis key guard in verification path"
```

---

## Phase 4 — FR-1.6 one-time reclaim of main's bare keys

### Task 18: `reclaim-main-bare-keys.sh` + bats

**Files:**
- Create: `services/atlas-pr-bootstrap/scripts/reclaim-main-bare-keys.sh`
- Create: `services/atlas-pr-bootstrap/test/reclaim_test.bats`
- Modify: `services/atlas-pr-bootstrap/Dockerfile` (COPY + chmod)

- [ ] **Step 1: Write `reclaim_test.bats`**

```bash
#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    SCRIPT="$PROJECT_ROOT/scripts/reclaim-main-bare-keys.sh"
    SHIM_DIR="$(mktemp -d)"
    # redis-cli shim: --scan prints fake keys per pattern; DEL records args.
    cat > "$SHIM_DIR/redis-cli" <<'EOF'
#!/usr/bin/env bash
args="$*"
# --scan --pattern <p> : echo one fake matching key for the bare namespaces,
# nothing for atlas:* (those must never be scanned by this script).
if [[ "$args" == *"--scan"* ]]; then
    for a in "$@"; do :; done
    pat=""
    while [ $# -gt 0 ]; do
        if [ "$1" = "--pattern" ]; then pat="$2"; fi
        shift
    done
    case "$pat" in
        atlas:*) ;;                       # must never happen
        "channel:tenants"|"drops:all"|"reactors:all"|"coordinator:active"|"invite:active-tenants"|"transport:instances"|"transport:characters")
            echo "$pat" ;;
        *) echo "${pat%\*}fake" ;;        # keyed families -> one fake key
    esac
    exit 0
fi
# DEL ... : record to a file so the test can assert.
if [ "$1" = "DEL" ]; then
    shift
    printf '%s\n' "$@" >> "$SHIM_DIR/deleted.txt"
    exit 0
fi
exit 0
EOF
    chmod +x "$SHIM_DIR/redis-cli"
    export PATH="$SHIM_DIR:$PATH"
    export SHIM_DIR
}

@test "reclaim: list mode (default) deletes nothing" {
    run env REDIS_URL="redis.test:6379" bash "$SCRIPT"
    [ "$status" -eq 0 ]
    [ ! -f "$SHIM_DIR/deleted.txt" ]
}

@test "reclaim: --apply deletes only allowlisted bare keys, never atlas:*" {
    run env REDIS_URL="redis.test:6379" bash "$SCRIPT" --apply
    [ "$status" -eq 0 ]
    [ -f "$SHIM_DIR/deleted.txt" ]
    # No deleted key may start with "atlas:" or contain a ":atlas:" segment.
    run grep -E '(^atlas:|:atlas:)' "$SHIM_DIR/deleted.txt"
    [ "$status" -ne 0 ]
    # A representative bare key was deleted.
    grep -qx "channel:tenants" "$SHIM_DIR/deleted.txt"
}

@test "reclaim: idempotent re-run" {
    run env REDIS_URL="redis.test:6379" bash "$SCRIPT" --apply
    [ "$status" -eq 0 ]
}
```

- [ ] **Step 2: Run** `bats services/atlas-pr-bootstrap/test/reclaim_test.bats`
Expected: FAIL — script does not exist.

- [ ] **Step 3: Write `reclaim-main-bare-keys.sh`**

```bash
#!/usr/bin/env bash
# One-time reclamation of atlas-main's now-dead BARE Redis keys after the
# task-045 namespacing fix lands. After the fix, main (ATLAS_ENV empty →
# prefix "atlas") writes prefixed keys (atlas:drops:all, …) and stops touching
# the bare forms. This script DELs only an explicit allowlist of bare
# namespaces; it MUST NEVER match atlas:* or <hash>:atlas:* (live keys).
#
# List-only by default; --apply to delete. Idempotent. Targets redis.home
# main DB 0 via REDIS_URL.
#
#   REDIS_URL  — host:port of the shared redis (e.g. redis.home:6379)
set -uo pipefail

. "$(dirname "$0")/lib.sh"

require_env REDIS_URL

APPLY=0
[ "${1:-}" = "--apply" ] && APPLY=1

# Bare namespaces the migrated services stop using. atlas-maps is intentionally
# excluded: its write side already used KeyPrefix() on main, so it has no bare
# orphans (see task-045 design §3.5).
EXACT_KEYS=(
    "channel:tenants"
    "drops:all"
    "reactors:all"
    "coordinator:active"
    "invite:active-tenants"
    "transport:instances"
    "transport:characters"
)
PREFIX_PATTERNS=(
    "coordinator:agreement:*"
    "coordinator:char:*"
    "transport:instance:*"
    "transport:route:*"
    "transport:channels:*"
    "drop:*"
    "reactor:*"
    "reactors:map:*"
    "drops:map:*"
    "reactor:cd:*"
    "reactor:spot:*"
)

reclaim_pattern() {
    local pat="$1"
    # Safety: refuse to ever scan an atlas-prefixed pattern.
    case "$pat" in
        atlas:*|*:atlas:*)
            ATLAS_STEP=reclaim log error "refusing to scan prefixed pattern: $pat"
            return 1 ;;
    esac
    local keys
    keys=$(redis-cli -u "redis://$REDIS_URL" --scan --pattern "$pat") || return 1
    [ -z "$keys" ] && return 0
    while IFS= read -r k; do
        [ -z "$k" ] && continue
        # Double-guard each concrete key.
        case "$k" in
            atlas:*|*:atlas:*)
                ATLAS_STEP=reclaim log warn "skipping prefixed key: $k"
                continue ;;
        esac
        echo "reclaim DEL $k"
        if [ "$APPLY" = "1" ]; then
            redis-cli -u "redis://$REDIS_URL" DEL "$k" >/dev/null 2>&1 \
                || ATLAS_STEP=reclaim log warn "DEL $k failed"
        fi
    done <<<"$keys"
}

ATLAS_STEP=reclaim log info "reclaim-main-bare-keys apply=${APPLY}"
rc=0
for k in "${EXACT_KEYS[@]}"; do
    reclaim_pattern "$k" || rc=1
done
for p in "${PREFIX_PATTERNS[@]}"; do
    reclaim_pattern "$p" || rc=1
done
ATLAS_STEP=done log info "reclaim complete"
exit $rc
```

- [ ] **Step 4: Run** `bats services/atlas-pr-bootstrap/test/reclaim_test.bats`
Expected: PASS.

- [ ] **Step 5: Add to Dockerfile + chmod, then commit**

In `services/atlas-pr-bootstrap/Dockerfile`, add to the COPY block (after
`sweep-orphans.sh`) and the chmod line:
```dockerfile
COPY scripts/reclaim-main-bare-keys.sh /atlas/reclaim-main-bare-keys.sh
```
```dockerfile
RUN chmod +x /atlas/bootstrap.sh /atlas/cleanup.sh /atlas/sweep-orphans.sh /atlas/reclaim-main-bare-keys.sh
```

```bash
git add services/atlas-pr-bootstrap/scripts/reclaim-main-bare-keys.sh \
        services/atlas-pr-bootstrap/test/reclaim_test.bats \
        services/atlas-pr-bootstrap/Dockerfile
git commit -m "feat(atlas-pr-bootstrap): one-time reclaim of main's bare redis keys (FR-1.6)"
```

---

## Phase 5 — Leak #2: PreDelete purge hook

### Task 19: `predelete-purge.sh` + bats + Dockerfile

**Files:**
- Create: `services/atlas-pr-bootstrap/scripts/predelete-purge.sh`
- Create: `services/atlas-pr-bootstrap/test/predelete_test.bats`
- Modify: `services/atlas-pr-bootstrap/Dockerfile`

- [ ] **Step 1: Write `predelete_test.bats`**

```bash
#!/usr/bin/env bats

setup() {
    PROJECT_ROOT="$(cd "$BATS_TEST_DIRNAME/.." && pwd)"
    SCRIPT="$PROJECT_ROOT/scripts/predelete-purge.sh"
    SHIM_DIR="$(mktemp -d)"
    export SHIM_DIR
    export PATH="$SHIM_DIR:$PATH"
    export ATLAS_INGRESS_BASE="http://atlas-ingress.test.svc.cluster.local"
    export PR_NUMBER="491"
}

# Write a curl shim driven by per-test CURL_MODE.
write_curl() {
    cat > "$SHIM_DIR/curl" <<EOF
#!/usr/bin/env bash
mode="$1"
url=""
method="GET"
while [ \$# -gt 0 ]; do
    case "\$1" in
        -X) method="\$2"; shift 2;;
        http*) url="\$1"; shift;;
        *) shift;;
    esac
done
case "\$mode:\$method:\$url" in
    *":GET:"*"/api/tenants")
        case "$CURL_MODE" in
            tenants_fail) echo ""; exit 22;;
            tenants_empty) echo '{"data":[]}';;
            *) echo '{"data":[{"id":"aaaaaaaa-0000-0000-0000-000000000001"},{"id":"bbbbbbbb-0000-0000-0000-000000000002"}]}';;
        esac
        ;;
    *":DELETE:"*"/api/data/tenants/"*)
        case "$CURL_MODE" in
            delete_500) echo "boom"; exit 22;;  # curl -f returns non-zero on 5xx
            *) exit 0;;
        esac
        ;;
esac
EOF
    chmod +x "$SHIM_DIR/curl"
}

@test "predelete: two tenants → two DELETEs, exit 0" {
    CURL_MODE=ok write_curl
    run bash "$SCRIPT"
    [ "$status" -eq 0 ]
    [[ "$output" == *"aaaaaaaa-0000-0000-0000-000000000001"* ]]
    [[ "$output" == *"bbbbbbbb-0000-0000-0000-000000000002"* ]]
}

@test "predelete: tenant-list fetch failure → non-zero, no silent skip" {
    CURL_MODE=tenants_fail write_curl
    run bash "$SCRIPT"
    [ "$status" -ne 0 ]
}

@test "predelete: empty tenant list → non-zero (env always has >=1 tenant)" {
    CURL_MODE=tenants_empty write_curl
    run bash "$SCRIPT"
    [ "$status" -ne 0 ]
}

@test "predelete: a DELETE failing → non-zero" {
    CURL_MODE=delete_500 write_curl
    run bash "$SCRIPT"
    [ "$status" -ne 0 ]
}
```

> The curl shim is intentionally simple; the assertion is on exit code +
> emitted tenant ids. Adjust the heredoc quoting if `bats` on the host
> mangles `$CURL_MODE` interpolation (use a file-based mode toggle if so).

- [ ] **Step 2: Run** `bats services/atlas-pr-bootstrap/test/predelete_test.bats`
Expected: FAIL — script missing.

- [ ] **Step 3: Write `predelete-purge.sh`**

```bash
#!/usr/bin/env bash
# Argo CD PreDelete hook. Runs in the per-PR namespace while atlas-data /
# atlas-tenants / atlas-ingress are still alive. Purges every tenant the env
# owns via atlas-data's DELETE /api/data/tenants/{id} (deletes Postgres rows +
# best-effort MinIO objects across atlas-wz/atlas-assets/atlas-renders). On any
# failure it exits non-zero so the hook Job fails visibly — NO silent skip.
#
# Required env:
#   PR_NUMBER            — for logging / ATLAS_ENV derivation
#   ATLAS_INGRESS_BASE   — http://atlas-ingress.<ns>.svc.cluster.local
set -uo pipefail

. "$(dirname "$0")/lib.sh"

require_env PR_NUMBER ATLAS_INGRESS_BASE

ATLAS_ENV="$(compute_atlas_env "$PR_NUMBER")"
export ATLAS_ENV

do_purge_tenants() {
    local ids
    if ! ids=$(curl -fsS -H 'Accept: application/vnd.api+json' \
            "${ATLAS_INGRESS_BASE}/api/tenants" 2>/dev/null | jq -r '.data[].id' 2>/dev/null); then
        record_error predelete-purge "could not enumerate tenants from ${ATLAS_INGRESS_BASE}/api/tenants"
        return 1
    fi
    if [ -z "$ids" ]; then
        record_error predelete-purge "no tenants returned; a PR env always owns >=1 tenant — refusing to report success"
        return 1
    fi

    local rc=0 id status
    while IFS= read -r id; do
        [ -z "$id" ] && continue
        ATLAS_STEP=predelete-purge log info "purging tenant ${id}"
        # atlas-data returns 202 on success; require operator header. -f makes
        # curl exit non-zero on >=400. Capture status for the log.
        status=$(curl -s -o /dev/null -w '%{http_code}' -X DELETE \
            -H 'X-Atlas-Operator: 1' \
            "${ATLAS_INGRESS_BASE}/api/data/tenants/${id}" 2>/dev/null || echo 000)
        case "$status" in
            202|200|204)
                ATLAS_STEP=predelete-purge log info "purged tenant ${id} (status ${status})" ;;
            *)
                ATLAS_STEP=predelete-purge log error "purge tenant ${id} failed (status ${status})"
                rc=1 ;;
        esac
    done <<<"$ids"
    return $rc
}

ATLAS_PHASE_ERRORS=()
run_phase predelete-purge do_purge_tenants
summarize_phases 1
exit $?
```

- [ ] **Step 4: Run** `bats services/atlas-pr-bootstrap/test/predelete_test.bats`
Expected: PASS.

- [ ] **Step 5: Add to Dockerfile, then commit**

In `services/atlas-pr-bootstrap/Dockerfile` COPY block + chmod line:
```dockerfile
COPY scripts/predelete-purge.sh /atlas/predelete-purge.sh
```
```dockerfile
RUN chmod +x /atlas/bootstrap.sh /atlas/cleanup.sh /atlas/sweep-orphans.sh /atlas/reclaim-main-bare-keys.sh /atlas/predelete-purge.sh
```

```bash
git add services/atlas-pr-bootstrap/scripts/predelete-purge.sh \
        services/atlas-pr-bootstrap/test/predelete_test.bats \
        services/atlas-pr-bootstrap/Dockerfile
git commit -m "feat(atlas-pr-bootstrap): PreDelete tenant-purge hook script (FR-2.1/2.2/2.4)"
```

### Task 20: PreDelete hook manifest + kustomization wiring

**Files:**
- Create: `deploy/k8s/overlays/pr/predelete-purge.yaml`
- Modify: `deploy/k8s/overlays/pr/kustomization.yaml`

- [ ] **Step 1: Create `predelete-purge.yaml`**

```yaml
# Argo CD PreDelete hook. Unlike the PostDelete cleanup (which runs in the
# `argocd` namespace AFTER the per-PR namespace is pruned), this hook runs IN
# the per-PR namespace while atlas-data / atlas-tenants / atlas-ingress are
# still Running, so it can call atlas-data's DELETE /api/data/tenants/{id} to
# purge each tenant's Postgres rows + MinIO objects. A failing hook blocks
# Application deletion and surfaces the error in Argo CD (the desired
# visibility); the sweep CronJob backstop reclaims storage if atlas-data is
# genuinely down. See task-045/design.md §4.1.
apiVersion: batch/v1
kind: Job
metadata:
  name: atlas-pr-predelete-purge
  namespace: atlas-pr-PLACEHOLDER_PR_NUMBER
  annotations:
    argocd.argoproj.io/hook: PreDelete
    # Retain failed jobs for inspection (visible-failure requirement); delete
    # only on success.
    argocd.argoproj.io/hook-delete-policy: HookSucceeded
spec:
  backoffLimit: 0
  template:
    spec:
      restartPolicy: Never
      # In-namespace networking only (reaches atlas-ingress ClusterIP); no
      # Kubernetes API access, so the default namespace SA suffices. See the
      # cluster-infra coordination note (task-045-teardown.md).
      imagePullSecrets:
        - name: ghcr-pull
      containers:
        - name: predelete-purge
          image: ghcr.io/chronicle20/atlas-pr-bootstrap/atlas-pr-bootstrap:latest
          command: ["/atlas/predelete-purge.sh"]
          env:
            - name: PR_NUMBER
              value: "PLACEHOLDER_PR_NUMBER"
            - name: ATLAS_INGRESS_BASE
              value: http://atlas-ingress.atlas-pr-PLACEHOLDER_PR_NUMBER.svc.cluster.local
```

> `PLACEHOLDER_PR_NUMBER` is substituted by `pr-validation.yml`'s
> update-pr-overlay step (same token used by `sync-bootstrap.yaml` and
> `postdelete-cleanup.yaml`). The image tag is bumped per-PR by CI like the
> other Jobs.

- [ ] **Step 2: Wire into `kustomization.yaml`** — add to the `resources:` list
  (after `sync-bootstrap.yaml`):
```yaml
  - predelete-purge.yaml
```

- [ ] **Step 3: Validate the overlay renders**

Run: `kustomize build deploy/k8s/overlays/pr >/dev/null && echo OK`
Expected: `OK` (no kustomize error). If `kustomize` isn't installed, use
`kubectl kustomize deploy/k8s/overlays/pr >/dev/null`.

> The overlay-level `namespace:` directive: confirm the Job's explicit
> `namespace: atlas-pr-PLACEHOLDER_PR_NUMBER` is consistent with how
> `sync-bootstrap.yaml` is namespaced (it relies on the overlay default). If the
> overlay sets a `namespace:` field, drop the explicit one to match siblings;
> if not, keep it. Inspect `kustomization.yaml` top for a `namespace:` line.

- [ ] **Step 4: Re-run render to confirm no regression** (same command).

- [ ] **Step 5: Commit**

```bash
git add deploy/k8s/overlays/pr/predelete-purge.yaml deploy/k8s/overlays/pr/kustomization.yaml
git commit -m "feat(deploy): wire PreDelete tenant-purge hook into pr overlay (FR-2.1)"
```

---

## Phase 6 — Leak #2: remove PostDelete `do_drop_tenant_storage`

### Task 21: delete the broken phase + update cleanup bats + drop dead secret mount

**Files:**
- Modify: `services/atlas-pr-bootstrap/scripts/cleanup.sh`
- Modify: `services/atlas-pr-bootstrap/test/cleanup_test.bats`
- Modify: `deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml`

- [ ] **Step 1: Update `cleanup_test.bats` first (TDD: assert the new shape)**

- Delete the three `do_drop_tenant_storage` tests (`cleanup_test.bats:360`,
  `:368`, `:413` — the `@test` blocks named
  `"cleanup.sh drop-tenant-storage …"`).
- Fix the phase-count assertion at `:356` from 8 phases to 7. Locate the test
  asserting the phase count / `phases_run=8` and change `8`→`7`. (Grep
  `phases_run` and `8 phases` in the file.)

- [ ] **Step 2: Run** `bats services/atlas-pr-bootstrap/test/cleanup_test.bats`
Expected: FAIL — the removed-phase tests are gone but `cleanup.sh` still defines
the phase, and the count assertion now expects 7 while the script still runs 8.

- [ ] **Step 3: Edit `cleanup.sh`**

1. Delete the entire `do_drop_tenant_storage()` function (`cleanup.sh:56-141`,
   including its leading comment block from `# do_drop_tenant_storage deletes…`).
2. Remove its `PHASES` entry (`cleanup.sh:333`):
```
    drop-tenant-storage  do_drop_tenant_storage
```
3. Delete the ordering comment block above `PHASES` that explains the
   drop-tenant-storage-before-drop-dbs constraint (`cleanup.sh:327-331`).

The resulting `PHASES` array:
```bash
PHASES=(
    drop-dbs             do_drop_dbs
    drop-topics          do_drop_topics
    drop-groups          do_drop_groups
    drop-redis           do_drop_redis
    drop-images          do_drop_images
    drop-dns             do_drop_dns
    drop-branch          do_drop_branch
)
```

- [ ] **Step 4: Run** `bats services/atlas-pr-bootstrap/test/cleanup_test.bats`
Expected: PASS (7 phases, no tenant-storage phase).

- [ ] **Step 5: Drop the now-dead `minio-root-creds` mount + commit**

In `deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml`, remove the
`minio-root-creds` `secretRef` envFrom block (`:72-83`, the `name:
minio-root-creds` + `optional: true` entry and its comment). PostDelete no
longer touches MinIO. (The Secret stays reflected into `argocd` for the sweep
CronJob — see coordination note.)

```bash
git add services/atlas-pr-bootstrap/scripts/cleanup.sh \
        services/atlas-pr-bootstrap/test/cleanup_test.bats \
        deploy/k8s/overlays/pr-cleanup/postdelete-cleanup.yaml
git commit -m "refactor(atlas-pr-bootstrap): remove broken PostDelete drop-tenant-storage (D3/FR-2.3/2.6)"
```

---

## Phase 7 — Leak #2: sweep allowlist + CronJob coordination

### Task 22: extend `sweep_minio` to protect live PR-env tenants (fail-closed)

**Files:**
- Modify: `services/atlas-pr-bootstrap/scripts/sweep-orphans.sh`
- Modify: `services/atlas-pr-bootstrap/test/sweep_test.bats`

- [ ] **Step 1: Add bats cases** to `sweep_test.bats` (mock `kubectl get ns` +
  per-ns `curl`):

```bash
@test "sweep_minio: protects a live PR-env tenant, deletes a true orphan" {
    SHIM_DIR="$(mktemp -d)"
    # kubectl get ns -> one live PR namespace.
    cat > "$SHIM_DIR/kubectl" <<'EOF'
#!/usr/bin/env bash
# get ns -l <selector> -o jsonpath -> one namespace name
echo "atlas-pr-700"
EOF
    chmod +x "$SHIM_DIR/kubectl"
    # curl: main tenants list = [MAIN]; live PR ns tenants = [LIVE]; mc handled
    # by a separate mc shim that lists MAIN, LIVE, ORPHAN under each bucket.
    cat > "$SHIM_DIR/curl" <<'EOF'
#!/usr/bin/env bash
for a in "$@"; do url="$a"; done
case "$url" in
    *atlas-main*) echo '{"data":[{"id":"11111111-1111-1111-1111-111111111111"}]}';;
    *atlas-pr-700*) echo '{"data":[{"id":"22222222-2222-2222-2222-222222222222"}]}';;
    *) echo '{"data":[]}';;
esac
EOF
    chmod +x "$SHIM_DIR/curl"
    cat > "$SHIM_DIR/mc" <<'EOF'
#!/usr/bin/env bash
case "$1 $2" in
    "alias set") exit 0;;
    "ls")
        # list tenants dir: MAIN, LIVE, ORPHAN
        echo "11111111-1111-1111-1111-111111111111/"
        echo "22222222-2222-2222-2222-222222222222/"
        echo "33333333-3333-3333-3333-333333333333/"
        ;;
esac
EOF
    chmod +x "$SHIM_DIR/mc"
    run env PATH="$SHIM_DIR:$PATH" \
        MINIO_ENDPOINT="minio.test:9000" MINIO_ROOT_USER=u MINIO_ROOT_PASSWORD=p \
        MINIO_TENANT_SAFETY_WINDOW_SEC=0 \
        bash "$SCRIPT" --minio --apply
    [ "$status" -eq 0 ] || [ "$status" -eq 1 ]
    # ORPHAN appears for deletion; MAIN and LIVE never do.
    [[ "$output" == *"33333333-3333-3333-3333-333333333333"* ]]
    [[ "$output" != *"drop-minio"*"22222222-2222-2222-2222-222222222222"* ]]
    [[ "$output" != *"drop-minio"*"11111111-1111-1111-1111-111111111111"* ]]
}

@test "sweep_minio: namespace-enumeration failure aborts (fail-closed)" {
    SHIM_DIR="$(mktemp -d)"
    cat > "$SHIM_DIR/kubectl" <<'EOF'
#!/usr/bin/env bash
exit 1
EOF
    chmod +x "$SHIM_DIR/kubectl"
    cat > "$SHIM_DIR/curl" <<'EOF'
#!/usr/bin/env bash
echo '{"data":[{"id":"11111111-1111-1111-1111-111111111111"}]}'
EOF
    chmod +x "$SHIM_DIR/curl"
    cat > "$SHIM_DIR/mc" <<'EOF'
#!/usr/bin/env bash
[ "$1 $2" = "alias set" ] && exit 0
exit 0
EOF
    chmod +x "$SHIM_DIR/mc"
    run env PATH="$SHIM_DIR:$PATH" \
        MINIO_ENDPOINT="minio.test:9000" MINIO_ROOT_USER=u MINIO_ROOT_PASSWORD=p \
        bash "$SCRIPT" --minio --apply
    [ "$status" -ne 0 ]
    [[ "$output" == *"abort"* ]] || [[ "$output" == *"enumerat"* ]]
}
```

- [ ] **Step 2: Run** `bats services/atlas-pr-bootstrap/test/sweep_test.bats`
Expected: FAIL — live-env allowlist not implemented.

- [ ] **Step 3: Edit `sweep_minio` in `sweep-orphans.sh`**

After the main-tenant allowlist is built (`sweep-orphans.sh:366-377`, where
`active_uuids` is populated and the empty/failed cases abort), insert live-PR-env
enumeration that UNIONs live PR-env tenant UUIDs into `active_uuids`,
fail-closed:

```bash
    # Extend the allowlist with LIVE PR-env tenants so the sweep never reclaims
    # data out from under a running-but-idle PR env (task-045 D5/FR-2.5).
    local ns_selector="${ATLAS_PR_NS_SELECTOR:-atlas.pr-number}"
    local pr_namespaces
    if ! pr_namespaces=$(kubectl get ns -l "$ns_selector" \
            -o jsonpath='{range .items[*]}{.metadata.name}{"\n"}{end}' 2>/dev/null); then
        ATLAS_STEP=drop-minio log warn \
            "could not enumerate PR namespaces (selector=${ns_selector}); aborting MinIO sweep (fail-closed)"
        return 1
    fi
    local ns pr_uuids
    while IFS= read -r ns; do
        [ -z "$ns" ] && continue
        if ! pr_uuids=$(curl -fsS -H 'Accept: application/vnd.api+json' \
                "http://atlas-tenants.${ns}.svc.cluster.local:8080/api/tenants" 2>/dev/null \
                | jq -r '.data[].id' 2>/dev/null); then
            ATLAS_STEP=drop-minio log warn \
                "could not fetch tenants for live ns ${ns}; aborting MinIO sweep (fail-closed)"
            return 1
        fi
        if [ -n "$pr_uuids" ]; then
            active_uuids=$(printf '%s\n%s\n' "$active_uuids" "$pr_uuids")
        fi
    done <<<"$pr_namespaces"
```

The downstream "Skip if this UUID is an active main tenant" check
(`sweep-orphans.sh:409-412`) now also protects live PR-env tenants because they
were unioned into `active_uuids`. Update that log/comment to say "active main or
live PR-env tenant".

Add the new env var to the script's usage/help text near the top if it
documents env vars (`ATLAS_PR_NS_SELECTOR`, default `atlas.pr-number`).

- [ ] **Step 4: Run** `bats services/atlas-pr-bootstrap/test/sweep_test.bats`
Expected: PASS (existing cases + the two new ones). Also re-run the full bats
suite: `bats services/atlas-pr-bootstrap/test/` → green.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-pr-bootstrap/scripts/sweep-orphans.sh \
        services/atlas-pr-bootstrap/test/sweep_test.bats
git commit -m "feat(atlas-pr-bootstrap): protect live PR-env tenants in MinIO sweep (D5/FR-2.5)"
```

### Task 23: cluster-infra coordination note + CronJob example

**Files:**
- Create: `dev/cluster-infra-coordination/task-045-teardown.md`
- Create: `dev/cluster-infra-coordination/sweep-orphans-cronjob.example.yaml`

- [ ] **Step 1: Write `sweep-orphans-cronjob.example.yaml`**

```yaml
# EXAMPLE — owned by the sibling cluster-infra repo, NOT applied from this repo.
# A cluster-wide SINGLETON CronJob (do not place under overlays/pr-cleanup,
# which CI renders once per PR). Runs the sweep backstop every 6h. The script
# logic lives in this repo's atlas-pr-bootstrap image; this manifest only
# schedules it. See task-045/design.md §4.4.
apiVersion: batch/v1
kind: CronJob
metadata:
  name: atlas-pr-sweep-orphans
  namespace: argocd
spec:
  schedule: "0 */6 * * *"
  concurrencyPolicy: Forbid
  successfulJobsHistoryLimit: 3
  failedJobsHistoryLimit: 3
  jobTemplate:
    spec:
      backoffLimit: 0
      template:
        spec:
          restartPolicy: Never
          # Reuses the existing PostDelete cleanup SA. cluster-infra must grant
          # it cluster-wide `list namespaces` + egress to cross-namespace
          # atlas-tenants services (for the live-PR-env allowlist).
          serviceAccountName: atlas-pr-cleanup
          containers:
            - name: sweep
              image: ghcr.io/chronicle20/atlas-pr-bootstrap/atlas-pr-bootstrap:latest
              command: ["/atlas/sweep-orphans.sh", "--minio", "--apply"]
              envFrom:
                - configMapRef:
                    name: atlas-pr-cleanup-env   # MINIO_ENDPOINT, ATLAS_MAIN_TENANTS_URL, …
                - secretRef:
                    name: minio-root-creds       # MINIO_ROOT_USER / MINIO_ROOT_PASSWORD
              env:
                - name: ATLAS_PR_NS_SELECTOR
                  value: "atlas.pr-number"
                - name: MINIO_TENANT_SAFETY_WINDOW_SEC
                  value: "7200"
```

- [ ] **Step 2: Write `task-045-teardown.md`** (mirror the existing
  `atlas-pr-cleanup-env.example.yaml` note's structure):

```markdown
# cluster-infra coordination — task-045 teardown leak fixes

This repo's task-045 PR changes the per-PR teardown. Two pieces live in the
sibling `cluster-infra` repo. **This repo's PR can land independently** — the
PreDelete hook is self-contained and needs nothing from cluster-infra; the
sweep CronJob simply gains the live-env allowlist once the image bump
propagates.

## 1. PreDelete hook — no RBAC change
`deploy/k8s/overlays/pr/predelete-purge.yaml` runs a Job in the per-PR
namespace using the default namespace ServiceAccount. It only needs in-cluster
networking to reach `atlas-ingress` (no Kubernetes API). **Confirm** the default
SA is acceptable; no Role/RoleBinding required.

## 2. Sweep CronJob — cluster-infra owned (singleton)
Apply `sweep-orphans-cronjob.example.yaml` (in this folder) in cluster-infra.
It is a cluster-wide singleton in `argocd`; do NOT add it to this repo's
per-PR `overlays/pr-cleanup` (CI renders that once per PR → N copies).

### Required SA changes for `atlas-pr-cleanup`
The sweep's live-PR-env allowlist (task-045 §4.3) enumerates PR namespaces and
queries each namespace's `atlas-tenants`. Grant the `atlas-pr-cleanup` SA:
- ClusterRole: `list` on `namespaces`.
- Network egress to cross-namespace `atlas-tenants.<ns>.svc:8080`.
Without these, the sweep **fails closed** (aborts rather than deleting with a
partial allowlist) — safe but it won't reclaim anything.

## 3. Confirm reflected secrets/configmaps remain in `argocd`
- `minio-root-creds` (reflected from `minio` ns) — still consumed by the sweep
  CronJob (PostDelete no longer uses it; that envFrom was removed this PR).
- `atlas-pr-cleanup-env` — `MINIO_ENDPOINT`, `ATLAS_MAIN_TENANTS_URL`, etc.

## Merge ordering
1. This repo's PR (image/scripts/PreDelete manifest/sweep logic) — land first.
2. cluster-infra: SA RBAC + CronJob — land any time after the image bump
   propagates. The CronJob is inert-safe before then (fails closed).
```

- [ ] **Step 3: Verify the example YAML parses**

Run: `kubectl apply --dry-run=client -f dev/cluster-infra-coordination/sweep-orphans-cronjob.example.yaml 2>/dev/null && echo OK || python3 -c "import yaml,sys;yaml.safe_load(open('dev/cluster-infra-coordination/sweep-orphans-cronjob.example.yaml'));print('OK')"`
Expected: `OK` (dry-run if a cluster is reachable, else YAML parse).

- [ ] **Step 4: (no test) — docs/example only.**

- [ ] **Step 5: Commit**

```bash
git add dev/cluster-infra-coordination/task-045-teardown.md \
        dev/cluster-infra-coordination/sweep-orphans-cronjob.example.yaml
git commit -m "docs(cluster-infra): task-045 teardown coordination note + sweep CronJob example"
```

---

## Phase 8 — Full verification

### Task 24: build/test/vet/guard/bake matrix + image build

**Files:** none (verification only).

- [ ] **Step 1: Go modules — test/vet/build each changed module**

Run from the worktree root:
```bash
for m in libs/atlas-redis \
         services/atlas-world/atlas.com/world \
         services/atlas-invites/atlas.com/invites \
         services/atlas-guilds/atlas.com/guilds \
         services/atlas-drops/atlas.com/drops \
         services/atlas-reactors/atlas.com/reactors \
         services/atlas-transports/atlas.com/transports \
         services/atlas-rates/atlas.com/rates \
         services/atlas-maps/atlas.com/maps; do
  echo "=== $m ===";
  ( cd "$m" && go test -race ./... && go vet ./... && go build ./... ) || break
done
( cd tools/rediskeyguard && go test ./... && go vet ./... && go build ./... )
```
Expected: every module clean.

- [ ] **Step 2: Run the regression guard**

Run: `./tools/redis-key-guard.sh`
Expected: exit 0 (no raw keyed client calls remain in any service).

- [ ] **Step 3: bats suite + bootstrap image build**

```bash
bats services/atlas-pr-bootstrap/test/
docker buildx bake atlas-pr-bootstrap
```
Expected: bats green; image builds (covers the new COPY lines for
`predelete-purge.sh` and `reclaim-main-bare-keys.sh`).

- [ ] **Step 4: `docker buildx bake` every touched service**

Run from the worktree root (mandatory per CLAUDE.md — go.work won't catch a
missing Dockerfile COPY):
```bash
for svc in atlas-world atlas-invites atlas-guilds atlas-drops atlas-reactors \
           atlas-transports atlas-rates atlas-maps; do
  echo "=== bake $svc ===";
  docker buildx bake "$svc" || break
done
```
Expected: all succeed. (`libs/atlas-redis` and `tools/rediskeyguard` have no
bake targets — validated via consumers above.)

- [ ] **Step 5: Final commit (if any verification fixups were needed)**

```bash
git add -A
git commit -m "test(task-045): verification fixups" || echo "nothing to commit"
```

> After this task, run `superpowers:requesting-code-review` (it will dispatch
> `plan-adherence-reviewer` + `backend-guidelines-reviewer`) BEFORE opening the
> PR, per CLAUDE.md. The acceptance-criteria end-to-end checks in PRD §10
> (fresh PR env → keys prefixed; close PR → redis/MinIO/Postgres clean) are
> manual on a test env and are not gated by this plan's automated steps.

---

## Spec coverage map (self-review)

| PRD / Design req | Task(s) |
|---|---|
| FR-1.1 all keys via KeyPrefix() | Tasks 1–14 (lib types + migrations) |
| FR-1.2 confirmed call sites (guilds/drops/reactors/transports/world/invites) | Tasks 5–12 |
| FR-1.3 broader audit (rates, maps, guard-surfaced) | Tasks 13, 14, 17 |
| FR-1.4 env-global aggregation preserved | `Set`/`Hash` (env-global) used for `*:all`, `channel:tenants`, etc. (Tasks 1–2, 5–12) |
| FR-1.5 regression guard in CI/go test | Tasks 15–17 |
| FR-1.6 one-time reclaim, idempotent | Task 18 |
| FR-2.1 purge moves to PreDelete | Tasks 19–20 |
| FR-2.2 enumerate tenants + DELETE /api/data/tenants/{id} | Task 19 |
| FR-2.3 remaining PostDelete phases unchanged | Task 21 (only drop-tenant-storage removed) |
| FR-2.4 PreDelete fails loudly, no silent skip | Task 19 (record_error + non-zero) |
| FR-2.5 sweep CronJob backstop + live-env safety | Tasks 22–23 |
| FR-2.6 no silent-success best-effort path remains | Task 21 (phase deleted entirely) |
| NFR isolation/security | Tasks 1–17 (the guard is the defense) |
| NFR observability (visible failures) | Tasks 19, 21, 22 |
| NFR idempotency | Tasks 18 (reclaim), 19 (purge), 22 (sweep) |
| Design §3.4 analyzer (D2) | Tasks 15–17 |
| Design §3.5 reclaim excludes maps | Task 18 (allowlist omits maps) + Task 14 (maps keeps shape) |
| Design §4.5 coordination note | Task 23 |
| Build/verification (CLAUDE.md) | Task 24 |
