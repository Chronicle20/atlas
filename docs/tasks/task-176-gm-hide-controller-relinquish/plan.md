# GM-Hide Relinquishes Monster & NPC Controller Eligibility — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** A GM who hides (SuperGmHide 9101004) relinquishes every monster and NPC they control and is excluded from controller candidacy until they reveal; NPC control becomes a single-controller-per-NPC election in atlas-channel (it is broadcast-to-everyone today).

**Architecture:** Monster half — atlas-monsters gains a Redis hidden-character registry (payload `Registry` + per-tenant SET index, mirroring the monster/puppet registry pairing), a buff-status consumer that mutates the set then relinquishes/re-elects, hidden filtering in the single election choke point `getControllerCandidate`, and a leader-gated reconciliation sweep. NPC half — atlas-channel gains a Redis `TenantKeyedHash` NPC-controller registry (claim = `HSETNX`, uncontrolled = absent), election in the spawn/exit/hide/reveal paths, a remove-controller packet arm, and NPC movement/animation relay to non-controller sessions.

**Tech Stack:** Go, `libs/atlas-redis` (Registry / TenantKeyedSet / TenantKeyedHash — `SetNX` already exists), atlas-kafka consumers, atlas-rest `requests`, miniredis for tests.

## Global Constraints

- Verification gates (CLAUDE.md): `go test -race ./...`, `go vet ./...`, `go build ./...` per changed module; `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check` from repo root; `docker buildx bake atlas-channel` (its `go.mod` changes). atlas-monsters `go.mod` is expected unchanged — verify with `git diff`; if it changed, bake it too.
- All keyed Redis access goes through `libs/atlas-redis` (redis-key-guard). All goroutines via `routine.Go` (goroutine-guard). This plan adds no new goroutines outside existing patterns.
- Hide filter keys on `skill.SuperGmHideId` (9101004, `libs/atlas-constants/skill/constants.go:3253`) ONLY. `GmHideId` (9001004) is absent from v83 WZ and MUST NOT be referenced. Dark Sight must be unaffected.
- Ordering is load-bearing (design D5): hidden-set mutation ALWAYS before any location-dependent action; every election reads Redis live; all failure paths fail-open to pre-task behavior (log + skip / unfiltered), never retry loops.
- No Cosmic citations in code comments; cite IDA/WZ instead. No literal home/absolute paths in committed files. Preserve line endings.
- New Kafka topic env `EVENT_TOPIC_CHARACTER_BUFF_STATUS` and `REDIS_URL` are already in the shared `atlas-env` configmap (`deploy/k8s/base/env-configmap.yaml`), and REST root URLs fall back to `BASE_SERVICE_URL` (`libs/atlas-rest/requests/url.go:14`) — **no deploy/k8s changes are needed**.
- Run `tools/lint.sh` (fix mode) before each commit.

**Plan-time facts confirmed** (design §9): (1) atlas-maps route is `GET /characters/{characterId}/location`, JSON:API type `character-locations`, attrs `worldId/channelId/mapId/instance` (`services/atlas-maps/atlas.com/maps/character/location/resource.go:35`, `rest.go`). (2) atlas-buffs removes buff state from its registry BEFORE emitting `EXPIRED` (`Cancel` at `character/processor.go:71` cancels registry then emits; `GetExpired` prunes then returns) — a winner-check on a just-revealed GM will not see the hide buff; no hedge needed. (3) Channel consumer groups are per-pod (`consumerGroupIdTemplate = "Channel Service - %s"` + `SERVICE_ID`, `atlas-channel/main.go:156-163`) — every pod sees every buff event and routes by session presence. (4) `CNpcPool::OnNpcChangeController` remove arm = `Decode1` flag `0` + `Decode4` npcId → `SetRemoteNpc` (v95 `0x679730`, v83 `0x6d9a83`, byte-identical) — version-stable, no gates. (5) atlas-buffs has no logout consumer (its only consumers are buff commands, `kafka/consumer/character/consumer.go`) — a hidden entry persists over logout, which is safe (offline chars are in no candidate pool) and self-corrects via reconciliation; the sweep prunes entries whose hide buff is gone.

**Contradiction with design noted:** design §5.1 says `KeyedHash` lacks `SetNX` and it must be added. Stale — `TenantKeyedHash.SetNX` already exists (`libs/atlas-redis/keyed_hash.go:33`) with a test. No atlas-redis change is needed; Task 6 does not exist for that reason.

---

## File Structure

**atlas-monsters** (`services/atlas-monsters/atlas.com/monsters/`):
- Create `character/hidden/registry.go` + `registry_test.go` — hidden-character registry (payload + tenant SET index).
- Create `character/hidden/task.go` + `task_test.go` — reconciliation sweep.
- Create `character/buff/requests.go`, `rest.go`, `model.go`, `processor.go` + `processor_test.go` — minimal atlas-buffs REST client for the sweep.
- Create `kafka/message/buff/kafka.go` — buff status-event message defs.
- Create `kafka/consumer/buff/consumer.go` + `consumer_test.go` — SuperGmHide APPLIED/EXPIRED handler.
- Modify `map/requests.go`, `map/rest.go`, `map/processor.go`, `map/mock/processor.go` — character-location client.
- Modify `monster/processor.go` + `processor_test.go` — hidden-aware election, sentinel, hide/reveal methods, DPS-leader guard.
- Modify `main.go` — registry init, consumer registration, sweep task.

**libs/atlas-packet**:
- Create `npc/clientbound/remove_controller.go` + `remove_controller_test.go` — remove arm of OnNpcChangeController.

**atlas-channel** (`services/atlas-channel/atlas.com/channel/`):
- Create `npc/controller/registry.go` + `registry_test.go` — NPC-controller Redis registry.
- Create `npc/controller/processor.go` + `processor_test.go` — claim/release/elect logic.
- Create `npc/controller/announce.go` — grant/revoke packet announcement helpers.
- Modify `go.mod` (+ redis deps), `main.go` (Redis connect + registry init).
- Modify `kafka/consumer/map/consumer.go` — spawn grant gating, exit release/reassign.
- Modify `kafka/consumer/buff/consumer.go` — hide/reveal branches.
- Modify `movement/processor.go`, `socket/handler/npc_action.go` — controller guard + relay.

**Task docs**: `docs/tasks/task-176-gm-hide-controller-relinquish/coverage-manifest.yaml`.

All paths below that start with `character/`, `monster/`, `map/`, `kafka/`, `npc/`, `movement/`, `socket/` are relative to the service module root named in the task. Run tests from the module root (`services/atlas-monsters/atlas.com/monsters` or `services/atlas-channel/atlas.com/channel` or `libs/atlas-packet`).

---

### Task 1: atlas-monsters hidden-character registry

**Files:**
- Create: `services/atlas-monsters/atlas.com/monsters/character/hidden/registry.go`
- Test: `services/atlas-monsters/atlas.com/monsters/character/hidden/registry_test.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/main.go` (import + init)

**Interfaces:**
- Produces: `hidden.InitRegistry(rc *goredis.Client)`, `hidden.GetRegistry() *Registry` (nil until Init), `(*Registry).Add(ctx context.Context, t tenant.Model, characterId uint32) error`, `(*Registry).Remove(ctx, t, characterId uint32) error`, `(*Registry).MemberSet(ctx, t) (map[uint32]struct{}, error)`, `(*Registry).GetAll(ctx) map[tenant.Model][]uint32`, `(*Registry).Clear(ctx)`.
- Consumed by: Task 2 (election filter), Task 4 (consumer mutation), Task 5 (sweep).

Two structures mirror the monster/puppet registry pairing (`monster/puppet_registry.go`): a payload `Registry` whose stored value carries full tenant identity (so the sweep can rebuild `tenant.Model` — same trick as `storedMonster`/`fromStored`, `monster/registry.go`), plus a per-tenant SET index for O(1) election reads.

- [ ] **Step 1: Write the failing test**

```go
package hidden

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/google/uuid"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func testTenant(t *testing.T) tenant.Model {
	t.Helper()
	ten, err := tenant.Create(uuid.New(), "GMS", 83, 1)
	if err != nil {
		t.Fatalf("tenant.Create: %v", err)
	}
	return ten
}

func setup(t *testing.T) *Registry {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })
	r := newRegistry(rc)
	t.Cleanup(func() { r.Clear(context.Background()) })
	return r
}

func TestAddRemoveMemberSet(t *testing.T) {
	r := setup(t)
	ctx := context.Background()
	ten := testTenant(t)

	ms, err := r.MemberSet(ctx, ten)
	if err != nil {
		t.Fatalf("MemberSet: %v", err)
	}
	if len(ms) != 0 {
		t.Fatalf("expected empty set, got %v", ms)
	}

	if err := r.Add(ctx, ten, 42); err != nil {
		t.Fatalf("Add: %v", err)
	}
	// Idempotent double-add (FR-1.4).
	if err := r.Add(ctx, ten, 42); err != nil {
		t.Fatalf("Add twice: %v", err)
	}
	ms, _ = r.MemberSet(ctx, ten)
	if _, ok := ms[42]; !ok || len(ms) != 1 {
		t.Fatalf("expected {42}, got %v", ms)
	}

	if err := r.Remove(ctx, ten, 42); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	// Idempotent double-remove (FR-1.4).
	if err := r.Remove(ctx, ten, 42); err != nil {
		t.Fatalf("Remove twice: %v", err)
	}
	ms, _ = r.MemberSet(ctx, ten)
	if len(ms) != 0 {
		t.Fatalf("expected empty after remove, got %v", ms)
	}
}

func TestTenantIsolationAndGetAll(t *testing.T) {
	r := setup(t)
	ctx := context.Background()
	tenA := testTenant(t)
	tenB := testTenant(t)

	_ = r.Add(ctx, tenA, 1)
	_ = r.Add(ctx, tenA, 2)
	_ = r.Add(ctx, tenB, 3)

	msA, _ := r.MemberSet(ctx, tenA)
	if len(msA) != 2 {
		t.Fatalf("tenant A expected 2 members, got %v", msA)
	}
	msB, _ := r.MemberSet(ctx, tenB)
	if _, ok := msB[3]; !ok || len(msB) != 1 {
		t.Fatalf("tenant B expected {3}, got %v", msB)
	}

	all := r.GetAll(ctx)
	if len(all[tenA]) != 2 || len(all[tenB]) != 1 {
		t.Fatalf("GetAll mismatch: %v", all)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run (from `services/atlas-monsters/atlas.com/monsters`): `go test ./character/hidden/... -run 'TestAddRemoveMemberSet|TestTenantIsolationAndGetAll' -v`
Expected: FAIL — package does not compile (`newRegistry` undefined).

- [ ] **Step 3: Write the implementation**

```go
// Package hidden tracks which characters are currently GM-hidden
// (SuperGmHide 9101004), shared across atlas-monsters replicas via Redis so
// any pod's controller election observes the same set (PRD FR-1.3).
package hidden

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	goredis "github.com/redis/go-redis/v9"
	"github.com/google/uuid"

	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// storedHidden carries full tenant identity alongside the character id so
// GetAll can rebuild tenant.Model for the reconciliation sweep — the same
// pattern as storedMonster in monster/registry.go.
type storedHidden struct {
	TenantId           string `json:"tenantId"`
	TenantRegion       string `json:"tenantRegion"`
	TenantMajorVersion uint16 `json:"tenantMajorVersion"`
	TenantMinorVersion uint16 `json:"tenantMinorVersion"`
	CharacterId        uint32 `json:"characterId"`
}

// Registry pairs a payload store with a per-tenant SET index (mirroring the
// monster registry's reg + mapIdx pairing):
//   - reg: atlas:hidden-character:<tenantId>:<characterId> -> storedHidden
//   - tenantIdx: atlas:hidden-characters:<tenantKey>:all -> SET of characterIds
type Registry struct {
	reg       *atlasredis.Registry[string, storedHidden]
	tenantIdx *atlasredis.TenantKeyedSet[string]
}

var (
	registry *Registry
	once     sync.Once
)

func newRegistry(rc *goredis.Client) *Registry {
	return &Registry{
		reg:       atlasredis.NewRegistry[string, storedHidden](rc, "hidden-character", func(s string) string { return s }),
		tenantIdx: atlasredis.NewTenantKeyedSet[string](rc, "hidden-characters", func(s string) string { return s }),
	}
}

func InitRegistry(rc *goredis.Client) {
	once.Do(func() {
		registry = newRegistry(rc)
	})
}

// GetRegistry returns the singleton, or nil before InitRegistry — callers
// must nil-check (same contract as GetPuppetRegistry).
func GetRegistry() *Registry {
	return registry
}

func payloadSuffix(t tenant.Model, characterId uint32) string {
	return fmt.Sprintf("%s:%d", t.Id().String(), characterId)
}

const tenantSetKey = "all"

// Add marks characterId as GM-hidden. Idempotent (SADD + Put overwrite).
func (r *Registry) Add(ctx context.Context, t tenant.Model, characterId uint32) error {
	if err := r.reg.Put(ctx, payloadSuffix(t, characterId), storedHidden{
		TenantId:           t.Id().String(),
		TenantRegion:       t.Region(),
		TenantMajorVersion: t.MajorVersion(),
		TenantMinorVersion: t.MinorVersion(),
		CharacterId:        characterId,
	}); err != nil {
		return err
	}
	return r.tenantIdx.Add(ctx, t, tenantSetKey, strconv.FormatUint(uint64(characterId), 10))
}

// Remove clears characterId's hidden mark. Idempotent (SREM + Remove of a
// missing key are both no-ops).
func (r *Registry) Remove(ctx context.Context, t tenant.Model, characterId uint32) error {
	if err := r.reg.Remove(ctx, payloadSuffix(t, characterId)); err != nil {
		return err
	}
	return r.tenantIdx.Remove(ctx, t, tenantSetKey, strconv.FormatUint(uint64(characterId), 10))
}

// MemberSet returns the hidden character ids for one tenant, fetched once
// per election (FR-4.1).
func (r *Registry) MemberSet(ctx context.Context, t tenant.Model) (map[uint32]struct{}, error) {
	members, err := r.tenantIdx.Members(ctx, t, tenantSetKey)
	if err != nil {
		return nil, err
	}
	out := make(map[uint32]struct{}, len(members))
	for _, m := range members {
		id, perr := strconv.ParseUint(m, 10, 32)
		if perr != nil {
			continue
		}
		out[uint32(id)] = struct{}{}
	}
	return out, nil
}

// GetAll returns every hidden character grouped by tenant — the
// reconciliation sweep's iteration source (mirrors Registry.GetMonsters).
func (r *Registry) GetAll(ctx context.Context) map[tenant.Model][]uint32 {
	result := make(map[tenant.Model][]uint32)
	all, err := r.reg.GetAll(ctx)
	if err != nil {
		return result
	}
	for _, sh := range all {
		tid, perr := uuid.Parse(sh.TenantId)
		if perr != nil {
			continue
		}
		t, terr := tenant.Create(tid, sh.TenantRegion, sh.TenantMajorVersion, sh.TenantMinorVersion)
		if terr != nil {
			continue
		}
		result[t] = append(result[t], sh.CharacterId)
	}
	return result
}

// Clear removes all hidden-character state (tests / operational reset).
// Deleting this state fail-opens to pre-task behavior (design D4).
func (r *Registry) Clear(ctx context.Context) {
	_, _ = r.reg.Clear(ctx)
	all := r.GetAll(ctx)
	for t := range all {
		_ = r.tenantIdx.Clear(ctx, t, tenantSetKey)
	}
}
```

Note on `Clear`: `GetAll` is read before `reg.Clear` wipes payloads — order the two calls `all := r.GetAll(ctx)` FIRST, then `r.reg.Clear(ctx)`, then the index loop. Write it that way:

```go
func (r *Registry) Clear(ctx context.Context) {
	all := r.GetAll(ctx)
	_, _ = r.reg.Clear(ctx)
	for t := range all {
		_ = r.tenantIdx.Clear(ctx, t, tenantSetKey)
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./character/hidden/... -v`
Expected: PASS (both tests).

- [ ] **Step 5: Init in main.go**

In `services/atlas-monsters/atlas.com/monsters/main.go`, add import `"atlas-monsters/character/hidden"` and after `monster.InitPuppetRegistry(rc)` (line ~64) add:

```go
	hidden.InitRegistry(rc)
```

- [ ] **Step 6: Verify build + guards**

Run (module root): `go build ./... && go vet ./...`
Run (repo root): `tools/redis-key-guard.sh`
Expected: clean — all Redis access is via atlas-redis types.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/character/hidden/ services/atlas-monsters/atlas.com/monsters/main.go
git commit -m "feat(task-176): hidden-character registry in atlas-monsters"
```

---

### Task 2: atlas-monsters election excludes hidden characters

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go` (`getControllerCandidate` ~line 260, `FindNextController` ~line 305, `NewProcessor` ~line 87, Damage DPS-switch ~line 473; delete `zeroValue`/`characterIdKey` helpers ~line 1391)
- Test: `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go` (append)

**Interfaces:**
- Consumes: `hidden.GetRegistry().MemberSet(ctx, t)` from Task 1.
- Produces: `monster.ErrNoControllerCandidate` (exported sentinel), a `hiddenFn func() (map[uint32]struct{}, error)` seam field on `ProcessorImpl` (tests override it), and the invariant that every election path (`Create`, `FindNextController` → enter/exit/hide/reveal) skips hidden characters. Task 4 relies on `FindNextController` treating "no candidate" as a logged no-op.

Behavior changes, all inside the one choke point:
1. Fetch the hidden set once per `getControllerCandidate` call via the seam; on error, warn and proceed **unfiltered** (fail-open, design §4.5).
2. Puppet-vicinity owner bias skips a hidden owner (FR-4.2).
3. The candidate pool drops hidden ids (FR-4.1). **Also fix the latent pool leak:** the current `controlCounts[m.ControlCharacterId()] += 1` inserts controllers that are NOT in the field pool (Go map insert-on-increment) — a hidden GM still controlling other mobs mid-relinquish would re-enter the pool through it. Only increment ids already seeded.
4. Empty pool returns typed `ErrNoControllerCandidate` instead of `errors.New("should not get here")` (FR-4.3); `FindNextController` treats it as a debug-logged no-op success so every call site (enter/exit/hide/reveal) inherits the behavior with no per-site changes.
5. The Damage DPS-leader switch (`processor.go:473`) also skips a hidden damage leader — acceptance says a hidden GM is **never** selected; this path assigns control outside `getControllerCandidate`.

- [ ] **Step 1: Write the failing tests**

Append to `monster/processor_test.go` (follow the file's existing setup helpers — it already boots miniredis via `InitMonsterRegistry` in its `TestMain`; reuse the file's tenant/context helpers; the snippets below show intent and must be adapted to the file's established builder/setup names):

```go
func testProcessorWithHidden(t *testing.T, hidden map[uint32]struct{}, hiddenErr error) *ProcessorImpl {
	t.Helper()
	p := NewProcessor(testLogger(), testContext(t)).(*ProcessorImpl)
	p.hiddenFn = func() (map[uint32]struct{}, error) { return hidden, hiddenErr }
	return p
}

func TestGetControllerCandidateExcludesHidden(t *testing.T) {
	// field with characters 1 (hidden) and 2 (visible), no controlled mobs:
	p := testProcessorWithHidden(t, map[uint32]struct{}{1: {}}, nil)
	f := testField()
	cid, err := p.getControllerCandidate(f, 0, 0, model.FixedProvider([]uint32{1, 2}))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cid != 2 {
		t.Fatalf("expected visible character 2, got %d", cid)
	}
}

func TestGetControllerCandidateOnlyHiddenIsSentinel(t *testing.T) {
	p := testProcessorWithHidden(t, map[uint32]struct{}{1: {}}, nil)
	f := testField()
	_, err := p.getControllerCandidate(f, 0, 0, model.FixedProvider([]uint32{1}))
	if !errors.Is(err, ErrNoControllerCandidate) {
		t.Fatalf("expected ErrNoControllerCandidate, got %v", err)
	}
}

func TestGetControllerCandidateEmptyPoolIsSentinel(t *testing.T) {
	p := testProcessorWithHidden(t, map[uint32]struct{}{}, nil)
	f := testField()
	_, err := p.getControllerCandidate(f, 0, 0, model.FixedProvider([]uint32{}))
	if !errors.Is(err, ErrNoControllerCandidate) {
		t.Fatalf("expected ErrNoControllerCandidate, got %v", err)
	}
}

func TestGetControllerCandidateRedisFailureFailsOpen(t *testing.T) {
	p := testProcessorWithHidden(t, nil, errors.New("redis down"))
	f := testField()
	cid, err := p.getControllerCandidate(f, 0, 0, model.FixedProvider([]uint32{1}))
	if err != nil || cid != 1 {
		t.Fatalf("fail-open expected candidate 1, got %d err %v", cid, err)
	}
}

func TestControlCountsDoNotResurrectNonPoolControllers(t *testing.T) {
	// Register a mob controlled by character 9 (not in pool), pool = {2}.
	// Candidate must be 2; 9 must never appear even though it controls a mob.
	// (Guards the pool-leak fix: increment only seeded ids.)
	// ... create mob via GetMonsterRegistry().CreateMonster + ControlMonster(t, id, 9)
	p := testProcessorWithHidden(t, map[uint32]struct{}{}, nil)
	cid, err := p.getControllerCandidate(f, 0, 0, model.FixedProvider([]uint32{2}))
	if err != nil || cid != 2 {
		t.Fatalf("expected 2, got %d err %v", cid, err)
	}
}

func TestFindNextControllerNoCandidateIsNoop(t *testing.T) {
	p := testProcessorWithHidden(t, map[uint32]struct{}{1: {}}, nil)
	// mob in field, pool contains only hidden char 1:
	err := p.FindNextController(model.FixedProvider([]uint32{1}))(mob)
	if err != nil {
		t.Fatalf("no-candidate must be a no-op success, got %v", err)
	}
	// mob must remain uncontrolled:
	got, _ := p.GetById(mob.UniqueId())
	if got.ControlCharacterId() != 0 {
		t.Fatalf("mob must stay uncontrolled, controller=%d", got.ControlCharacterId())
	}
}
```

Also add a puppet-bias test: seed `GetPuppetRegistry()` with an in-vicinity puppet owned by hidden character 1 (pool `{1, 2}`); expect candidate 2 (bias skipped), mirroring the existing puppet tests in `puppet_test.go`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./monster/... -run 'TestGetControllerCandidate|TestControlCounts|TestFindNextControllerNoCandidate' -v`
Expected: FAIL — `hiddenFn`/`ErrNoControllerCandidate` undefined.

- [ ] **Step 3: Implement**

In `monster/processor.go`:

(a) Add the sentinel near the top-level var/const declarations:

```go
// ErrNoControllerCandidate reports that an election found no eligible
// controller — a legitimate outcome when the field is empty of visible
// characters (e.g. only a GM-hidden character remains, FR-4.3). Callers
// treat it as "leave uncontrolled", not an error.
var ErrNoControllerCandidate = errors.New("no controller candidate")
```

(b) Add the seam to `ProcessorImpl` (next to `inFieldFn`) and default it in `NewProcessor`:

```go
	hiddenFn  func() (map[uint32]struct{}, error)
```

```go
	p.hiddenFn = func() (map[uint32]struct{}, error) {
		if r := hidden.GetRegistry(); r != nil {
			return r.MemberSet(p.ctx, p.t)
		}
		return map[uint32]struct{}{}, nil
	}
```

Import `"atlas-monsters/character/hidden"`.

(c) Add a helper:

```go
// hiddenSet reads the shared GM-hidden set. On failure it returns an empty
// set (fail-open: election degrades to pre-hide-awareness behavior rather
// than leaving monsters uncontrolled).
func (p *ProcessorImpl) hiddenSet() map[uint32]struct{} {
	hs, err := p.hiddenFn()
	if err != nil {
		p.l.WithError(err).Warnf("Unable to read hidden-character set; controller election proceeding unfiltered.")
		return map[uint32]struct{}{}
	}
	return hs
}
```

(d) Rewrite `getControllerCandidate` (replacing lines ~260-302):

```go
func (p *ProcessorImpl) getControllerCandidate(f field.Model, monsterX int16, monsterY int16, idp model.Provider[[]uint32]) (uint32, error) {
	p.l.Debugf("Identifying controller candidate for monsters in field [%s].", f.Id())

	hiddenSet := p.hiddenSet()

	// Puppet vicinity bias: prefer the owner of an in-vicinity puppet, but only
	// when that owner is actually a candidate in the field's character pool and
	// is not GM-hidden (FR-4.2).
	if pr := GetPuppetRegistry(); pr != nil {
		if owner, ok := pr.VicinityOwner(p.ctx, p.t, f, monsterX, monsterY); ok {
			if _, isHidden := hiddenSet[owner]; !isHidden {
				if ids, ierr := idp(); ierr == nil {
					for _, id := range ids {
						if id == owner {
							p.l.Debugf("Controller candidate biased to puppet owner [%d] in field [%s].", owner, f.Id())
							return owner, nil
						}
					}
				}
			}
		}
	}

	ids, err := idp()
	if err != nil {
		p.l.WithError(err).Errorf("Unable to initialize controller candidate map.")
		return 0, err
	}
	controlCounts := make(map[uint32]int, len(ids))
	for _, id := range ids {
		if _, isHidden := hiddenSet[id]; isHidden {
			continue
		}
		controlCounts[id] = 0
	}
	err = model.ForEachSlice(p.ControlledInFieldProvider(f), func(m Model) error {
		// Only count loads for seeded (in-pool, non-hidden) candidates —
		// incrementing an unseeded key would insert it and let a character
		// outside the pool (or a hidden one mid-relinquish) win the election.
		if _, ok := controlCounts[m.ControlCharacterId()]; ok {
			controlCounts[m.ControlCharacterId()] += 1
		}
		return nil
	})
	if err != nil {
		return 0, err
	}

	index := uint32(0)
	first := true
	for key, val := range controlCounts {
		if first {
			index = key
			first = false
		} else if val < controlCounts[index] {
			index = key
		}
	}

	if first {
		return 0, ErrNoControllerCandidate
	}
	p.l.Debugf("Controller candidate has been determined. Character [%d].", index)
	return index, nil
}
```

(Keep the existing `err` handling shape for `ForEachSlice` consistent with the current code, which ignores its error — preserve current behavior: assign to `err` and don't early-return if that is what the file does today; match it exactly.)

(e) In `FindNextController` (~line 305), swallow the sentinel:

```go
	return func(m Model) error {
		cid, err := p.getControllerCandidate(m.Field(), m.X(), m.Y(), idp)
		if errors.Is(err, ErrNoControllerCandidate) {
			p.l.Debugf("No eligible controller for monster [%d] in field [%s]; leaving uncontrolled.", m.UniqueId(), m.Field().Id())
			return nil
		}
		if err != nil {
			return err
		}
		...unchanged...
	}
```

(f) In `Damage` (~line 473), guard the DPS-leader switch:

```go
	if characterId != last.Monster.ControlCharacterId() && last.Monster.DamageLeader() == characterId {
		if _, isHidden := p.hiddenSet()[characterId]; isHidden {
			p.l.Debugf("Skipping DPS-leader controller switch to GM-hidden character [%d] for monster [%d].", characterId, last.Monster.UniqueId())
		} else {
			...existing attackerInField block unchanged, indented under this else...
		}
	}
```

(g) Delete the now-unused `zeroValue` and `characterIdKey` helpers (~line 1391) — confirm with `grep -rn "zeroValue\|characterIdKey"` that line 278's `CollectToMap` call was their only consumer; if tests reference them, update the tests.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -race ./monster/... -v`
Expected: PASS — new tests green, all existing election/exit tests still green (behavior for the no-hidden case is unchanged: same least-loaded pick, same puppet bias).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/monster/
git commit -m "feat(task-176): exclude GM-hidden characters from monster controller election"
```

---

### Task 3: atlas-monsters character-location REST client

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/map/requests.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/map/rest.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/map/processor.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/map/mock/processor.go`

**Interfaces:**
- Produces: `_map.Processor.GetCharacterField(characterId uint32) (field.Model, error)` — resolves the character's live field from atlas-maps `GET /characters/{characterId}/location`; mock gains `GetCharacterFieldFunc`.
- Consumed by: Task 4 (hide/reveal handlers, FR-2.1).

The upstream route is `GET {MAPS}/characters/{characterId}/location` returning JSON:API type `character-locations` with attributes `worldId`, `channelId`, `mapId`, `instance` (`services/atlas-maps/atlas.com/maps/character/location/rest.go`). Single resource → `requests.GetRequest` + `requests.Provider` (the `mobskill` pattern, `monster/mobskill/processor.go:30`), not `DrainProvider`. 404 (no location row → `requests.ErrNotFound`) is the "offline / in transition" case of FR-7.1.

- [ ] **Step 1: Add the request**

Append to `map/requests.go`:

```go
const characterLocationResource = "characters/%d/location"

func requestCharacterLocation(characterId uint32) requests.Request[LocationRestModel] {
	return requests.GetRequest[LocationRestModel](fmt.Sprintf(getBaseRequest()+characterLocationResource, characterId))
}
```

- [ ] **Step 2: Add the REST model**

Append to `map/rest.go` (match the JSON:API contract of the maps-side `RestModel` exactly):

```go
// LocationRestModel is the JSON:API projection of atlas-maps'
// GET /characters/{characterId}/location response.
type LocationRestModel struct {
	Id        uint32     `json:"-"`
	WorldId   world.Id   `json:"worldId"`
	ChannelId channel.Id `json:"channelId"`
	MapId     _map2.Id   `json:"mapId"`
	Instance  uuid.UUID  `json:"instance"`
}

func (r LocationRestModel) GetName() string {
	return "character-locations"
}

func (r LocationRestModel) GetID() string {
	return strconv.FormatUint(uint64(r.Id), 10)
}

func (r *LocationRestModel) SetID(s string) error {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(v)
	return nil
}

// ExtractLocation converts the REST projection to a field.Model.
func ExtractLocation(rm LocationRestModel) (field.Model, error) {
	return field.NewBuilder(rm.WorldId, rm.ChannelId, rm.MapId).SetInstance(rm.Instance).Build(), nil
}
```

Use whatever import aliases `map/rest.go` already has for `world`/`channel`/`map` constants (check the file — `rest.go` currently backs `CharacterIdsInFieldProvider`'s `RestModel`; add missing imports `strconv`, `uuid`, and the constants packages with non-colliding aliases, e.g. `_map2 "github.com/Chronicle20/atlas/libs/atlas-constants/map"`).

- [ ] **Step 3: Add the processor method + update the mock**

`map/processor.go` — extend the interface and impl:

```go
type Processor interface {
	CharacterIdsInFieldProvider(f field.Model) model.Provider[[]uint32]
	GetCharacterField(characterId uint32) (field.Model, error)
}
```

```go
// GetCharacterField resolves the character's CURRENT field from atlas-maps —
// the location authority — at call time. The buff event deliberately carries
// no field (PRD §8: a hide buff outlives any single map visit, so a field
// snapshot on the event would be stale by EXPIRED time).
func (p *ProcessorImpl) GetCharacterField(characterId uint32) (field.Model, error) {
	return requests.Provider[LocationRestModel, field.Model](p.l, p.ctx)(requestCharacterLocation(characterId), ExtractLocation)()
}
```

`map/mock/processor.go` — add the func field + method:

```go
	GetCharacterFieldFunc func(characterId uint32) (field.Model, error)
```

```go
func (m *ProcessorMock) GetCharacterField(characterId uint32) (field.Model, error) {
	if m.GetCharacterFieldFunc != nil {
		return m.GetCharacterFieldFunc(characterId)
	}
	return field.Model{}, nil
}
```

- [ ] **Step 4: Verify build**

Run: `go build ./... && go vet ./...`
Expected: clean. (No new unit test — the method is a thin `requests.Provider` composition, exercised through Task 4's seam tests; this matches how `CharacterIdsInFieldProvider` is covered today.)

- [ ] **Step 5: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/map/
git commit -m "feat(task-176): character-location client in atlas-monsters"
```

---

### Task 4: atlas-monsters hide/reveal — processor methods + buff consumer

**Files:**
- Modify: `services/atlas-monsters/atlas.com/monsters/monster/processor.go` (interface + two methods + `locationFn` seam)
- Create: `services/atlas-monsters/atlas.com/monsters/kafka/message/buff/kafka.go`
- Create: `services/atlas-monsters/atlas.com/monsters/kafka/consumer/buff/consumer.go`
- Test: `services/atlas-monsters/atlas.com/monsters/monster/processor_test.go` (append), `services/atlas-monsters/atlas.com/monsters/kafka/consumer/buff/consumer_test.go`
- Modify: `services/atlas-monsters/atlas.com/monsters/main.go`

**Interfaces:**
- Consumes: `hidden.GetRegistry()` (Task 1), `_map.Processor.GetCharacterField` (Task 3), `ErrNoControllerCandidate` semantics (Task 2).
- Produces: `monster.Processor.RelinquishControlOnHide(characterId uint32) error` and `monster.Processor.RestoreCandidacyOnReveal(characterId uint32) error`; consumer package `kafka/consumer/buff` with `InitConsumers`/`InitHandlers` (registered in `main.go` under the shared `consumerGroupId`); message package `kafka/message/buff` exposing `EnvEventStatusTopic`, `EventStatusTypeBuffApplied`, `EventStatusTypeBuffExpired`, `StatusEvent[E]`, `AppliedStatusEventBody`, `ExpiredStatusEventBody`.

**Ordering is load-bearing (design §3.1 / FR-7.2):** set mutation first, then location, then relinquish/re-elect. A location failure must leave the set mutated.

- [ ] **Step 1: Write the failing processor tests**

Append to `monster/processor_test.go`:

```go
func TestRelinquishOnHideMutatesSetBeforeLocationFailure(t *testing.T) {
	// hidden registry backed by the test miniredis (InitRegistry in setup)
	p := NewProcessor(testLogger(), testContext(t)).(*ProcessorImpl)
	p.locationFn = func(characterId uint32) (field.Model, error) {
		return field.Model{}, errors.New("maps unavailable")
	}
	if err := p.RelinquishControlOnHide(77); err != nil {
		t.Fatalf("location failure must not propagate: %v", err)
	}
	ms, _ := hidden.GetRegistry().MemberSet(context.Background(), testTenantModel(t))
	if _, ok := ms[77]; !ok {
		t.Fatalf("hidden-set mutation must apply even when location fails (FR-7.2)")
	}
}

func TestRelinquishOnHideReassignsControlledMobs(t *testing.T) {
	// field with chars {1 (hiding GM), 2}; two mobs controlled by 1.
	p := NewProcessor(testLogger(), testContext(t)).(*ProcessorImpl)
	p.locationFn = func(uint32) (field.Model, error) { return f, nil }
	p.inFieldFn = func(field.Model) ([]uint32, error) { return []uint32{1, 2}, nil }
	if err := p.RelinquishControlOnHide(1); err != nil {
		t.Fatalf("RelinquishControlOnHide: %v", err)
	}
	for _, id := range []uint32{mob1.UniqueId(), mob2.UniqueId()} {
		m, _ := p.GetById(id)
		if m.ControlCharacterId() != 2 {
			t.Fatalf("mob [%d] expected controller 2, got %d", id, m.ControlCharacterId())
		}
	}
}

func TestRelinquishOnHideOnlyHiddenLeftLeavesUncontrolled(t *testing.T) {
	// field with only char 1 (hiding); mob controlled by 1 → left uncontrolled (FR-4.3).
	...
	m, _ := p.GetById(mob.UniqueId())
	if m.ControlCharacterId() != 0 {
		t.Fatalf("expected uncontrolled, got %d", m.ControlCharacterId())
	}
}

func TestRestoreCandidacyOnRevealRemovesFromSetAndSweeps(t *testing.T) {
	// char 1 hidden and sole occupant; one uncontrolled mob. On reveal the
	// sweep must elect 1 (only candidate, now visible).
	_ = hidden.GetRegistry().Add(ctx, ten, 1)
	p.locationFn = func(uint32) (field.Model, error) { return f, nil }
	p.inFieldFn = func(field.Model) ([]uint32, error) { return []uint32{1}, nil }
	if err := p.RestoreCandidacyOnReveal(1); err != nil {
		t.Fatalf("RestoreCandidacyOnReveal: %v", err)
	}
	ms, _ := hidden.GetRegistry().MemberSet(ctx, ten)
	if len(ms) != 0 {
		t.Fatalf("set entry must be removed on reveal")
	}
	m, _ := p.GetById(mob.UniqueId())
	if m.ControlCharacterId() != 1 {
		t.Fatalf("reveal sweep must elect the revealed character, got %d", m.ControlCharacterId())
	}
}
```

(`p.inFieldFn` note: `getControllerCandidate` receives its pool via the `idp` argument, so the hide/reveal methods must pass a pool provider — implement them to use `p.inFieldFn`-backed providers as shown in Step 3, so tests inject the pool exactly as `handleStatusEventCharacterExit` tests do today. Reuse the file's existing helpers for creating mobs/fields.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./monster/... -run 'TestRelinquishOnHide|TestRestoreCandidacy' -v`
Expected: FAIL — methods undefined.

- [ ] **Step 3: Implement the processor methods**

In `monster/processor.go`: add to the `Processor` interface:

```go
	RelinquishControlOnHide(characterId uint32) error
	RestoreCandidacyOnReveal(characterId uint32) error
```

Add the seam field to `ProcessorImpl` next to `inFieldFn`, defaulted in `NewProcessor`:

```go
	locationFn func(characterId uint32) (field.Model, error)
```

```go
	p.locationFn = func(characterId uint32) (field.Model, error) {
		return _map.NewProcessor(p.l, p.ctx).GetCharacterField(characterId)
	}
```

Implement (place after `StopControl`):

```go
// RelinquishControlOnHide handles a SuperGmHide APPLIED event (FR-2): mark
// the character hidden (ALWAYS first — FR-7.2), resolve their live field,
// then release and reassign every monster they control there. Location
// failure (offline / in transition) skips the release; candidacy exclusion
// still holds via the set, and the next election trigger converges.
func (p *ProcessorImpl) RelinquishControlOnHide(characterId uint32) error {
	if r := hidden.GetRegistry(); r != nil {
		if err := r.Add(p.ctx, p.t, characterId); err != nil {
			p.l.WithError(err).Warnf("Unable to mark character [%d] hidden; election exclusion degraded until reconciliation.", characterId)
		}
	}

	f, err := p.locationFn(characterId)
	if err != nil {
		p.l.WithError(err).Debugf("GM-hide: unable to locate character [%d]; skipping monster relinquish (set mutation applied).", characterId)
		return nil
	}

	// Snapshot ONCE before StopControl mutates registry state — same
	// provider-re-evaluation race as handleStatusEventCharacterExit.
	mobs, err := p.ControlledByCharacterInFieldProvider(f, characterId)()
	if err != nil {
		p.l.WithError(err).Warnf("GM-hide: unable to fetch mobs controlled by [%d] in field [%s]; skipping relinquish.", characterId, f.Id())
		return nil
	}
	if len(mobs) == 0 {
		p.l.Debugf("GM-hide: character [%d] controls no monsters in field [%s].", characterId, f.Id())
		return nil
	}
	snapshot := model.FixedProvider(mobs)
	_ = model.ForEachSlice(snapshot, p.StopControl, model.ParallelExecute())
	idp := func() ([]uint32, error) { return p.inFieldFn(f) }
	_ = model.ForEachSlice(snapshot, p.FindNextController(idp), model.ParallelExecute())
	p.l.Debugf("GM-hide: character [%d] relinquished [%d] monsters in field [%s].", characterId, len(mobs), f.Id())
	return nil
}

// RestoreCandidacyOnReveal handles the SuperGmHide EXPIRED event (FR-3):
// unmark hidden (ALWAYS first), then re-run election for uncontrolled
// monsters in the character's live field so the revealed character is
// eligible again. No forced transfer (FR-3.2).
func (p *ProcessorImpl) RestoreCandidacyOnReveal(characterId uint32) error {
	if r := hidden.GetRegistry(); r != nil {
		if err := r.Remove(p.ctx, p.t, characterId); err != nil {
			p.l.WithError(err).Warnf("Unable to unmark character [%d] hidden; reconciliation will repair.", characterId)
		}
	}

	f, err := p.locationFn(characterId)
	if err != nil {
		p.l.WithError(err).Debugf("GM-reveal: unable to locate character [%d]; skipping re-election (set mutation applied).", characterId)
		return nil
	}

	idp := func() ([]uint32, error) { return p.inFieldFn(f) }
	_ = model.ForEachSlice(p.NotControlledInFieldProvider(f), p.FindNextController(idp), model.ParallelExecute())
	p.l.Debugf("GM-reveal: re-ran election for uncontrolled monsters in field [%s] after character [%d] revealed.", f.Id(), characterId)
	return nil
}
```

(Note: `inFieldFn` already exists on `ProcessorImpl` (line ~83) and defaults to the live `CharacterIdsInFieldProvider` — reusing it keeps one seam for tests. The reveal-race is closed by construction: `Remove` runs before the sweep, and the sweep covers exactly the mobs a concurrent stale-read election would have skipped — design D5.)

- [ ] **Step 4: Run processor tests**

Run: `go test -race ./monster/... -v`
Expected: PASS.

- [ ] **Step 5: Create the message defs**

`kafka/message/buff/kafka.go` (subset of the producer's contract, `services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go:60-90`):

```go
package buff

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

const (
	EnvEventStatusTopic        = "EVENT_TOPIC_CHARACTER_BUFF_STATUS"
	EventStatusTypeBuffApplied = "APPLIED"
	EventStatusTypeBuffExpired = "EXPIRED"
)

type StatusEvent[E any] struct {
	WorldId     world.Id `json:"worldId"`
	CharacterId uint32   `json:"characterId"`
	Type        string   `json:"type"`
	Body        E        `json:"body"`
}

type AppliedStatusEventBody struct {
	FromId    uint32       `json:"fromId"`
	SourceId  int32        `json:"sourceId"`
	Level     byte         `json:"level"`
	Duration  int32        `json:"duration"`
	Changes   []StatChange `json:"changes"`
	CreatedAt time.Time    `json:"createdAt"`
	ExpiresAt time.Time    `json:"expiresAt"`
}

type ExpiredStatusEventBody struct {
	SourceId  int32        `json:"sourceId"`
	Level     byte         `json:"level"`
	Duration  int32        `json:"duration"`
	Changes   []StatChange `json:"changes"`
	CreatedAt time.Time    `json:"createdAt"`
	ExpiresAt time.Time    `json:"expiresAt"`
}

type StatChange struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}
```

- [ ] **Step 6: Create the consumer**

`kafka/consumer/buff/consumer.go` (mirrors `kafka/consumer/map/consumer.go`):

```go
package buff

import (
	consumer2 "atlas-monsters/kafka/consumer"
	buff2 "atlas-monsters/kafka/message/buff"
	"atlas-monsters/monster"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/consumer"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/handler"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/message"
	"github.com/Chronicle20/atlas/libs/atlas-kafka/topic"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

func InitConsumers(l logrus.FieldLogger) func(func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
	return func(rf func(config consumer.Config, decorators ...model.Decorator[consumer.Config])) func(consumerGroupId string) {
		return func(consumerGroupId string) {
			rf(consumer2.NewConfig(l)("character_buff_status_event")(buff2.EnvEventStatusTopic)(consumerGroupId), consumer.SetHeaderParsers(consumer.SpanHeaderParser, consumer.TenantHeaderParser))
		}
	}
}

func InitHandlers(l logrus.FieldLogger) func(rf func(topic string, handler handler.Handler) (string, error)) error {
	return func(rf func(topic string, handler handler.Handler) (string, error)) error {
		var t string
		t, _ = topic.EnvProvider(l)(buff2.EnvEventStatusTopic)()
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventApplied))); err != nil {
			return err
		}
		if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventExpired))); err != nil {
			return err
		}
		return nil
	}
}

// handleStatusEventApplied reacts ONLY to SuperGmHide (9101004) APPLIED
// events (FR-1.1/FR-1.2). Dark Sight and every other buff pass through
// untouched. GmHideId (9001004) is absent from v83 game data and is
// deliberately not handled.
func handleStatusEventApplied(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.AppliedStatusEventBody]) {
	if e.Type != buff2.EventStatusTypeBuffApplied {
		return
	}
	if e.Body.SourceId != int32(skill.SuperGmHideId) {
		return
	}
	if err := monster.NewProcessor(l, ctx).RelinquishControlOnHide(e.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to relinquish monster control for hiding character [%d].", e.CharacterId)
	}
}

func handleStatusEventExpired(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.ExpiredStatusEventBody]) {
	if e.Type != buff2.EventStatusTypeBuffExpired {
		return
	}
	if e.Body.SourceId != int32(skill.SuperGmHideId) {
		return
	}
	if err := monster.NewProcessor(l, ctx).RestoreCandidacyOnReveal(e.CharacterId); err != nil {
		l.WithError(err).Errorf("Unable to restore controller candidacy for revealed character [%d].", e.CharacterId)
	}
}
```

- [ ] **Step 7: Write the SourceId-filter test**

`kafka/consumer/buff/consumer_test.go` — prove a non-SuperGmHide event does NOT mutate the hidden set (acceptance: Dark Sight unaffected):

```go
package buff

import (
	"atlas-monsters/character/hidden"
	buff2 "atlas-monsters/kafka/message/buff"
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestAppliedIgnoresNonSuperGmHideSources(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	hidden.InitRegistry(rc)
	t.Cleanup(func() { hidden.GetRegistry().Clear(context.Background()) })

	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	ctx := tenant.WithContext(context.Background(), ten)
	l, _ := logrus.NewNullLogger() // use the test-logger idiom already present in the module if different

	for _, sourceId := range []int32{int32(skill.RogueDarkSightId), 9001004} {
		handleStatusEventApplied(l, ctx, buff2.StatusEvent[buff2.AppliedStatusEventBody]{
			WorldId: 0, CharacterId: 5, Type: buff2.EventStatusTypeBuffApplied,
			Body: buff2.AppliedStatusEventBody{SourceId: sourceId},
		})
	}
	ms, _ := hidden.GetRegistry().MemberSet(context.Background(), ten)
	if len(ms) != 0 {
		t.Fatalf("non-SuperGmHide sources must not mutate the hidden set, got %v", ms)
	}
}
```

(`logrus.NewNullLogger` lives in `github.com/sirupsen/logrus/hooks/test` as `test.NewNullLogger()` — use `test.NewNullLogger()` with import `"github.com/sirupsen/logrus/hooks/test"`, matching existing monsters tests; check `monster/processor_test.go` for the module's idiom and copy it. Note `hidden.InitRegistry` is `sync.Once`-guarded — if another test file in this package (or an earlier test) already initialized it, route through a shared TestMain like `monster/registry_test.go` does. The APPLIED/EXPIRED act paths are covered by Task 4 Steps 1-4 at the processor layer; this test pins the filter.)

- [ ] **Step 8: Register in main.go**

In `main.go` add import `buffconsumer "atlas-monsters/kafka/consumer/buff"` and, beside the `_map` consumer registration (lines ~69, ~74):

```go
	buffconsumer.InitConsumers(l)(cmf)(consumerGroupId)
```

```go
	if err := buffconsumer.InitHandlers(l)(consumer.GetManager().RegisterHandler); err != nil {
		l.WithError(err).Fatal("Unable to register kafka handlers.")
	}
```

(Shared `consumerGroupId` is correct: the handler's effect is a shared-Redis mutation + Kafka-emitting election, so exactly-one-pod consumption is required — same reasoning as the `_map` consumer, PRD §8.)

- [ ] **Step 9: Run the full module**

Run: `go test -race ./... && go vet ./... && go build ./...`
Expected: PASS/clean.

- [ ] **Step 10: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/
git commit -m "feat(task-176): monster relinquish/re-elect on GM hide/reveal via buff-status consumer"
```

---

### Task 5: atlas-monsters hidden-set reconciliation sweep

**Files:**
- Create: `services/atlas-monsters/atlas.com/monsters/character/buff/requests.go`, `rest.go`, `model.go`, `processor.go`
- Create: `services/atlas-monsters/atlas.com/monsters/character/hidden/task.go`
- Test: `services/atlas-monsters/atlas.com/monsters/character/hidden/task_test.go`, `services/atlas-monsters/atlas.com/monsters/character/buff/processor_test.go` (only if the module's DrainProvider clients have test precedent — `map/processor_drain_test.go` exists; mirror it if practical, otherwise rely on task_test seams)
- Modify: `services/atlas-monsters/atlas.com/monsters/main.go`

**Interfaces:**
- Consumes: `hidden.GetRegistry().GetAll/Remove` (Task 1).
- Produces: `buff.Processor.GetByCharacterId(characterId uint32) ([]Model, error)` + `buff.HasActiveGmHide(bs []Model) bool` in monsters; `hidden.NewReconciliationTask(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *ReconciliationTask` satisfying `tasks.Task` (`Run()`, `SleepTime() time.Duration`), registered leader-gated in `main.go`.

Sweep direction is one-way (design §3.3): remove set members whose atlas-buffs state shows no active SuperGmHide buff. Members hidden in atlas-buffs but missing from the set are NOT swept (fail-open = pre-task behavior; self-heals on next event).

- [ ] **Step 1: Create the buffs REST client**

`character/buff/requests.go`:

```go
package buff

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const characterBuffsResource = "characters/%d/buffs"

func getBaseRequest() string {
	return requests.RootUrl("BUFFS")
}

// characterBuffsUrl is a bare URL because atlas-buffs' list is paginated
// (task-117) and consumed via requests.DrainProvider.
func characterBuffsUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+characterBuffsResource, characterId)
}
```

`character/buff/rest.go` (subset of atlas-buffs' projection — only the fields the sweep needs; extra attributes in the payload are ignored by JSON unmarshalling):

```go
package buff

import "time"

type RestModel struct {
	Id        string    `json:"-"`
	SourceId  int32     `json:"sourceId"`
	ExpiresAt time.Time `json:"expiresAt"`
}

func (r RestModel) GetName() string {
	return "buffs"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{sourceId: rm.SourceId, expiresAt: rm.ExpiresAt}, nil
}
```

`character/buff/model.go`:

```go
package buff

import (
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-constants/skill"
)

type Model struct {
	sourceId  int32
	expiresAt time.Time
}

func (m Model) SourceId() int32 {
	return m.sourceId
}

func (m Model) Expired() bool {
	return time.Now().After(m.expiresAt)
}

// HasActiveGmHide reports whether bs contains an unexpired SuperGmHide
// buff. Keying on SourceId, not the DARK_SIGHT stat type, so Rogue Dark
// Sight never matches.
func HasActiveGmHide(bs []Model) bool {
	for _, b := range bs {
		if b.SourceId() == int32(skill.SuperGmHideId) && !b.Expired() {
			return true
		}
	}
	return false
}
```

`character/buff/processor.go` (Drain pattern per `services/atlas-channel/atlas.com/channel/character/buff/processor.go:47`):

```go
package buff

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor interface {
	GetByCharacterId(characterId uint32) ([]Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &ProcessorImpl{l: l, ctx: ctx}
}

var _ Processor = (*ProcessorImpl)(nil)

// GetByCharacterId drains every page of the character's buffs (the
// upstream list is paginated, task-117).
func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(characterBuffsUrl(characterId), 250, Extract, model.Filters[Model]())()
}
```

- [ ] **Step 2: Write the failing sweep test**

`character/hidden/task_test.go`:

```go
package hidden

import (
	"context"
	"errors"
	"testing"
	"time"

	buff "atlas-monsters/character/buff"

	"github.com/sirupsen/logrus/hooks/test"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func TestReconciliationRemovesStaleKeepsActive(t *testing.T) {
	r := setup(t) // from registry_test.go
	ctx := context.Background()
	ten := testTenant(t)
	_ = r.Add(ctx, ten, 1) // stale: no hide buff upstream
	_ = r.Add(ctx, ten, 2) // active: hide buff upstream
	_ = r.Add(ctx, ten, 3) // fetch error: must be kept (fail-safe)

	l, _ := test.NewNullLogger()
	task := NewReconciliationTask(l, ctx, time.Minute)
	task.registry = r
	task.buffsFn = func(_ tenant.Model, characterId uint32) ([]buff.Model, error) {
		switch characterId {
		case 2:
			return []buff.Model{buff.NewModel(9101004, time.Now().Add(time.Hour))}, nil
		case 3:
			return nil, errors.New("buffs unavailable")
		default:
			return []buff.Model{}, nil
		}
	}
	task.Run()

	ms, _ := r.MemberSet(ctx, ten)
	if _, ok := ms[1]; ok {
		t.Fatalf("stale member 1 must be removed")
	}
	if _, ok := ms[2]; !ok {
		t.Fatalf("active member 2 must be kept")
	}
	if _, ok := ms[3]; !ok {
		t.Fatalf("member 3 must be kept when the buffs fetch fails")
	}
}
```

This requires a `buff.NewModel(sourceId int32, expiresAt time.Time) Model` test-visible constructor — the project's Builder-pattern rule says no `_testhelpers.go` constructors; instead give `Model` a plain exported constructor in `model.go` (it is a legitimate production constructor, used by `Extract`):

```go
func NewModel(sourceId int32, expiresAt time.Time) Model {
	return Model{sourceId: sourceId, expiresAt: expiresAt}
}
```

and have `Extract` use it.

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./character/hidden/... -run TestReconciliation -v`
Expected: FAIL — `NewReconciliationTask` undefined.

- [ ] **Step 4: Implement the task**

`character/hidden/task.go`:

```go
package hidden

import (
	"context"
	"time"

	buff "atlas-monsters/character/buff"

	"github.com/sirupsen/logrus"

	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// ReconcileInterval is how often the leader pod sweeps the hidden set
// against atlas-buffs (design D4: drift repair for lost EXPIRED events).
const ReconcileInterval = 5 * time.Minute

// ReconciliationTask prunes hidden-set members whose SuperGmHide buff no
// longer exists upstream. One-way on purpose: the inverse drift (hidden in
// atlas-buffs, absent here) degrades to pre-task behavior and self-heals on
// the next APPLIED/EXPIRED event.
type ReconciliationTask struct {
	l        logrus.FieldLogger
	ctx      context.Context
	interval time.Duration
	registry *Registry
	buffsFn  func(t tenant.Model, characterId uint32) ([]buff.Model, error)
}

func NewReconciliationTask(l logrus.FieldLogger, ctx context.Context, interval time.Duration) *ReconciliationTask {
	l.Infof("Initializing hidden-character reconciliation task to run every %dms.", interval.Milliseconds())
	t := &ReconciliationTask{l: l, ctx: ctx, interval: interval}
	t.registry = GetRegistry()
	t.buffsFn = func(ten tenant.Model, characterId uint32) ([]buff.Model, error) {
		return buff.NewProcessor(l, tenant.WithContext(ctx, ten)).GetByCharacterId(characterId)
	}
	return t
}

func (t *ReconciliationTask) Run() {
	if t.registry == nil {
		return
	}
	all := t.registry.GetAll(t.ctx)
	for ten, ids := range all {
		for _, id := range ids {
			bs, err := t.buffsFn(ten, id)
			if err != nil {
				t.l.WithError(err).Debugf("Hidden-set reconciliation: unable to fetch buffs for character [%d]; keeping entry.", id)
				continue
			}
			if !buff.HasActiveGmHide(bs) {
				// Warn: reaching here means an EXPIRED event was lost.
				t.l.Warnf("Hidden-set reconciliation: character [%d] has no active SuperGmHide buff; removing stale entry.", id)
				if err := t.registry.Remove(t.ctx, ten, id); err != nil {
					t.l.WithError(err).Warnf("Hidden-set reconciliation: unable to remove stale entry for character [%d].", id)
				}
			}
		}
	}
}

func (t *ReconciliationTask) SleepTime() time.Duration {
	return t.interval
}
```

- [ ] **Step 5: Run tests**

Run: `go test -race ./character/... -v`
Expected: PASS.

- [ ] **Step 6: Register leader-gated in main.go**

In `main.go`'s `registerSweepTasks` (line ~94-101) add:

```go
		tasks.Register(l, ctx)(hidden.NewReconciliationTask(l, ctx, hidden.ReconcileInterval))
```

(The function is already invoked under leader election when `leaderEnabled`; nothing else to wire.)

- [ ] **Step 7: Module gates + go.mod check**

Run: `go test -race ./... && go vet ./... && go build ./...`
Run: `git diff --stat services/atlas-monsters/atlas.com/monsters/go.mod`
Expected: tests/vet/build clean; go.mod diff EMPTY (all imports already present). If go.mod changed, run `docker buildx bake atlas-monsters` from the repo root and it must succeed.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-monsters/atlas.com/monsters/
git commit -m "feat(task-176): leader-gated hidden-set reconciliation sweep"
```

---

### Task 6: remove-controller packet arm in libs/atlas-packet

**Files:**
- Create: `libs/atlas-packet/npc/clientbound/remove_controller.go`
- Test: `libs/atlas-packet/npc/clientbound/remove_controller_test.go`
- Create: `docs/tasks/task-176-gm-hide-controller-relinquish/coverage-manifest.yaml`

**Interfaces:**
- Produces: `clientbound.RemoveController` struct, `clientbound.NewNpcRemoveController(id uint32) RemoveController`, `Operation()` returning the EXISTING writer name `NpcSpawnRequestControllerWriter` (`"SpawnNPCRequestController"`, `spawn_request_controller.go:13`) — same opcode, different leading byte. **No new opcode, no template routing, no writer registration** — the op `SPAWN_NPC_REQUEST_CONTROLLER` is already routed and verified (`docs/packets/audits/status.json`: verified on gms_v83/v84/v87/v95/jms_v185).
- Consumed by: Task 9's `AnnounceRevoke`.

Wire layout is IDA-derived (design companion change 2; confirmed this plan phase): `CNpcPool::OnNpcChangeController` reads `Decode1` (controller flag) then `Decode4` (npc object id); flag `0` → `SetRemoteNpc(id)` and NOTHING else is read. Verified byte-identical at GMS v95 `0x679730` and GMS v83 `0x6d9a83`. The existing grant codec hard-codes flag `1`; this struct is the flag-`0` arm.

- [ ] **Step 1: Write the failing fixture test**

```go
package clientbound

import (
	"bytes"
	"context"
	"testing"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
)

// Read-order fixture for the remove arm of CNpcPool::OnNpcChangeController:
// Decode1 (flag=0) + Decode4 (dwNpcId) -> SetRemoteNpc. IDA: GMS v95
// 0x679730, GMS v83 0x6d9a83 (byte-identical across the version set).
func TestNpcRemoveControllerEncode(t *testing.T) {
	l, _ := logrus.NewEntry(logrus.New()), context.Background()
	m := NewNpcRemoveController(0x01020304)
	out := m.Encode(l.Logger, context.Background())(map[string]interface{}{})
	want := []byte{0x00, 0x04, 0x03, 0x02, 0x01}
	if !bytes.Equal(out, want) {
		t.Fatalf("encode mismatch: got % X want % X", out, want)
	}
}

func TestNpcRemoveControllerDecodeRoundTrip(t *testing.T) {
	l := logrus.New()
	m := NewNpcRemoveController(42)
	raw := m.Encode(l, context.Background())(map[string]interface{}{})
	r := request.NewReader(&raw, 0) // match the reader-construction idiom used by spawn_request_controller_test.go
	var d RemoveController
	d.Decode(l, context.Background())(&r, map[string]interface{}{})
	if d.Id() != 42 {
		t.Fatalf("round-trip id mismatch: got %d", d.Id())
	}
}
```

(Copy the exact `request.Reader` construction and Encode invocation idiom from the neighboring `spawn_request_controller_test.go` — signatures there are authoritative; adjust the snippet's scaffolding accordingly, keeping the `want` bytes as written.)

- [ ] **Step 2: Run test to verify it fails**

Run (from `libs/atlas-packet`): `go test ./npc/clientbound/ -run TestNpcRemoveController -v`
Expected: FAIL — `NewNpcRemoveController` undefined.

- [ ] **Step 3: Implement the codec**

`npc/clientbound/remove_controller.go`:

```go
package clientbound

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-socket/request"
	"github.com/Chronicle20/atlas/libs/atlas-socket/response"
)

// packet-audit:fname CNpcPool::OnNpcChangeController
//
// RemoveController is the flag-0 (remove) arm of OnNpcChangeController: the
// client demotes the NPC to remote control (SetRemoteNpc) and stops running
// its AI/animation locally. Same opcode as SpawnRequestController (the
// flag-1 grant arm); read order: Decode1 flag, Decode4 npc object id, no
// further reads (GMS v95 0x679730, GMS v83 0x6d9a83).
type RemoveController struct {
	id uint32
}

func NewNpcRemoveController(id uint32) RemoveController {
	return RemoveController{id: id}
}

func (m RemoveController) Id() uint32 {
	return m.id
}

func (m RemoveController) Operation() string {
	return NpcSpawnRequestControllerWriter
}

func (m RemoveController) String() string {
	return fmt.Sprintf("id [%d] (remove controller)", m.id)
}

func (m RemoveController) Encode(l logrus.FieldLogger, _ context.Context) func(options map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(options map[string]interface{}) []byte {
		w.WriteByte(0)
		w.WriteInt(m.id)
		return w.Bytes()
	}
}

func (m *RemoveController) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, options map[string]interface{}) {
		_ = r.ReadByte() // always 0 (remove arm)
		m.id = r.ReadUint32()
	}
}
```

- [ ] **Step 4: Run tests**

Run: `go test -race ./npc/... -v`
Expected: PASS (new + existing spawn_request_controller tests).

- [ ] **Step 5: Coverage manifest + matrix check**

Write `docs/tasks/task-176-gm-hide-controller-relinquish/coverage-manifest.yaml`:

```yaml
# coverage-manifest
ops:
  - SPAWN_NPC_REQUEST_CONTROLLER
versions:
  - gms_v83
  - gms_v84
  - gms_v87
  - gms_v95
  - jms_v185
fields:
  - "npc/clientbound/RemoveController: new flag-0 (remove) arm of CNpcPool::OnNpcChangeController; same opcode as the verified grant arm; no wire change to any existing codec; read order IDA-derived (v95 0x679730, v83 0x6d9a83)"
out_of_scope: []
```

Then confirm the matrix is undisturbed — run the packet-audit checks the repo gates on (from repo root):

```bash
go run ./tools/packet-audit matrix --check
```

(If the invocation path differs, use the exact command CI runs — see `docs/packets/PROCESS.md`; the requirement is exit 0 with no cell drift, since this task adds a struct without touching any verified codec's bytes.)

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/npc/clientbound/remove_controller.go libs/atlas-packet/npc/clientbound/remove_controller_test.go docs/tasks/task-176-gm-hide-controller-relinquish/coverage-manifest.yaml
git commit -m "feat(task-176): NPC remove-controller packet arm (OnNpcChangeController flag 0)"
```

---

### Task 7: atlas-channel NPC-controller registry + Redis bootstrap

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/npc/controller/registry.go`
- Test: `services/atlas-channel/atlas.com/channel/npc/controller/registry_test.go`
- Modify: `services/atlas-channel/atlas.com/channel/go.mod`
- Modify: `services/atlas-channel/atlas.com/channel/main.go`

**Interfaces:**
- Produces: `controller.InitRegistry(rc *goredis.Client)`, `controller.GetRegistry() *Registry` (nil before init), `(*Registry).Claim(ctx, t, f field.Model, npcObjectId uint32, characterId uint32) (bool, error)` (HSETNX — atomic first-writer-wins), `(*Registry).Release(ctx, t, f, npcObjectIds ...uint32) error`, `(*Registry).ControllerOf(ctx, t, f, npcObjectId uint32) (uint32, bool, error)`, `(*Registry).GetAll(ctx, t, f) (map[uint32]uint32, error)`, `(*Registry).ControlledBy(ctx, t, f, characterId uint32) ([]uint32, error)`.
- Consumed by: Tasks 8-12.

State model (design D2/§5.1): one `TenantKeyedHash` per field, namespace `npc-controller`, key suffix `<world>:<channel>:<map>:<instance>`, hash field = NPC objectId (decimal), value = controller characterId (decimal). Uncontrolled = absent. Redis drops empty hashes automatically → no teardown sweep.

- [ ] **Step 1: Add the go.mod requirement**

In `services/atlas-channel/atlas.com/channel/go.mod` (the `replace` for atlas-redis already exists at line 90), add to the first `require` block:

```
	github.com/Chronicle20/atlas/libs/atlas-redis v0.0.0
```

then from the module root run:

```bash
go mod tidy
```

`go mod tidy` will pull `github.com/redis/go-redis/v9 v9.21.0` and (after the test in Step 2 exists) `github.com/alicebob/miniredis/v2 v2.38.0` — versions matching `services/atlas-monsters/atlas.com/monsters/go.mod`. (Run tidy again after writing the test file if the first pass ran before it existed. Per project memory: tidy AFTER imports exist, never `go work sync`.)

- [ ] **Step 2: Write the failing registry test**

`npc/controller/registry_test.go`:

```go
package controller

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

func setupRegistry(t *testing.T) (*Registry, tenant.Model, field.Model) {
	t.Helper()
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	t.Cleanup(mr.Close)
	rc := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rc.Close() })
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	f := field.NewBuilder(0, 1, 100000000).Build()
	return newRegistry(rc), ten, f
}

func TestClaimIsFirstWriterWins(t *testing.T) {
	r, ten, f := setupRegistry(t)
	ctx := context.Background()

	won, err := r.Claim(ctx, ten, f, 1000, 7)
	if err != nil || !won {
		t.Fatalf("first claim must win: won=%v err=%v", won, err)
	}
	won, err = r.Claim(ctx, ten, f, 1000, 8)
	if err != nil || won {
		t.Fatalf("second claim must lose: won=%v err=%v", won, err)
	}
	cur, ok, err := r.ControllerOf(ctx, ten, f, 1000)
	if err != nil || !ok || cur != 7 {
		t.Fatalf("controller must remain 7: cur=%d ok=%v err=%v", cur, ok, err)
	}
}

func TestReleaseAndAbsence(t *testing.T) {
	r, ten, f := setupRegistry(t)
	ctx := context.Background()

	_, ok, err := r.ControllerOf(ctx, ten, f, 1000)
	if err != nil || ok {
		t.Fatalf("absent entry must report ok=false: ok=%v err=%v", ok, err)
	}

	_, _ = r.Claim(ctx, ten, f, 1000, 7)
	_, _ = r.Claim(ctx, ten, f, 1001, 7)
	_, _ = r.Claim(ctx, ten, f, 1002, 9)

	got, err := r.ControlledBy(ctx, ten, f, 7)
	if err != nil || len(got) != 2 {
		t.Fatalf("expected 2 NPCs controlled by 7, got %v err %v", got, err)
	}

	if err := r.Release(ctx, ten, f, 1000, 1001); err != nil {
		t.Fatalf("Release: %v", err)
	}
	all, _ := r.GetAll(ctx, ten, f)
	if len(all) != 1 || all[1002] != 9 {
		t.Fatalf("expected only 1002->9 left, got %v", all)
	}
	// Idempotent double-release.
	if err := r.Release(ctx, ten, f, 1000); err != nil {
		t.Fatalf("double release must be a no-op: %v", err)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run (from `services/atlas-channel/atlas.com/channel`): `go test ./npc/controller/... -v`
Expected: FAIL — package does not exist.

- [ ] **Step 4: Implement the registry**

`npc/controller/registry.go`:

```go
// Package controller owns the single-controller-per-NPC election state
// (task-176, FR-5). Exactly one non-hidden character in a field is granted
// client-side control of each NPC; everyone else renders it as remote.
// State lives in Redis so every channel pod observes the same assignment
// (FR-5.5). Uncontrolled = absent from the hash (design D2) — there is no
// live-NPC record; static map data remains the NPC source of truth.
package controller

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"sync"

	goredis "github.com/redis/go-redis/v9"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Registry maps, per (tenant, field), NPC objectId -> controller
// characterId. Backing key:
// atlas:npc-controller:<tenantKey>:<world>:<channel>:<map>:<instance>.
type Registry struct {
	hash *atlasredis.TenantKeyedHash[string]
}

var (
	registry *Registry
	once     sync.Once
)

func newRegistry(rc *goredis.Client) *Registry {
	return &Registry{
		hash: atlasredis.NewTenantKeyedHash[string](rc, "npc-controller", func(s string) string { return s }),
	}
}

func InitRegistry(rc *goredis.Client) {
	once.Do(func() {
		registry = newRegistry(rc)
	})
}

// GetRegistry returns the singleton, or nil before InitRegistry — callers
// must nil-check and fail open.
func GetRegistry() *Registry {
	return registry
}

func fieldSuffix(f field.Model) string {
	return fmt.Sprintf("%d:%d:%d:%s", byte(f.WorldId()), byte(f.ChannelId()), uint32(f.MapId()), f.Instance().String())
}

// Claim atomically records characterId as npcObjectId's controller iff no
// controller is recorded (HSETNX). Returns true when this call won.
func (r *Registry) Claim(ctx context.Context, t tenant.Model, f field.Model, npcObjectId uint32, characterId uint32) (bool, error) {
	return r.hash.SetNX(ctx, t, fieldSuffix(f), strconv.FormatUint(uint64(npcObjectId), 10), strconv.FormatUint(uint64(characterId), 10))
}

// Release removes the controller entries for the given NPCs. Idempotent;
// Redis deletes the hash when its last field goes, so empty fields leak
// nothing.
func (r *Registry) Release(ctx context.Context, t tenant.Model, f field.Model, npcObjectIds ...uint32) error {
	if len(npcObjectIds) == 0 {
		return nil
	}
	fields := make([]string, 0, len(npcObjectIds))
	for _, id := range npcObjectIds {
		fields = append(fields, strconv.FormatUint(uint64(id), 10))
	}
	return r.hash.Del(ctx, t, fieldSuffix(f), fields...)
}

// ControllerOf returns (controllerId, true) when npcObjectId has a recorded
// controller, (0, false) when uncontrolled.
func (r *Registry) ControllerOf(ctx context.Context, t tenant.Model, f field.Model, npcObjectId uint32) (uint32, bool, error) {
	v, err := r.hash.Get(ctx, t, fieldSuffix(f), strconv.FormatUint(uint64(npcObjectId), 10))
	if err != nil {
		if errors.Is(err, goredis.Nil) {
			return 0, false, nil
		}
		return 0, false, err
	}
	id, perr := strconv.ParseUint(v, 10, 32)
	if perr != nil {
		return 0, false, perr
	}
	return uint32(id), true, nil
}

// GetAll returns the full npcObjectId -> controllerId map for the field.
func (r *Registry) GetAll(ctx context.Context, t tenant.Model, f field.Model) (map[uint32]uint32, error) {
	raw, err := r.hash.GetAll(ctx, t, fieldSuffix(f))
	if err != nil {
		return nil, err
	}
	out := make(map[uint32]uint32, len(raw))
	for k, v := range raw {
		nid, e1 := strconv.ParseUint(k, 10, 32)
		cid, e2 := strconv.ParseUint(v, 10, 32)
		if e1 != nil || e2 != nil {
			continue
		}
		out[uint32(nid)] = uint32(cid)
	}
	return out, nil
}

// ControlledBy lists the NPCs currently assigned to characterId in field f.
func (r *Registry) ControlledBy(ctx context.Context, t tenant.Model, f field.Model, characterId uint32) ([]uint32, error) {
	all, err := r.GetAll(ctx, t, f)
	if err != nil {
		return nil, err
	}
	var out []uint32
	for nid, cid := range all {
		if cid == characterId {
			out = append(out, nid)
		}
	}
	return out, nil
}
```

Check `TenantKeyedHash.Get`'s not-found behavior in `libs/atlas-redis/keyed_hash.go` before relying on `goredis.Nil` — if the lib wraps it (e.g. returns its own sentinel), match the lib's contract; `keyed_hash_test.go` shows the expected error.

- [ ] **Step 5: Run tests**

Run: `go test -race ./npc/controller/... -v`
Expected: PASS.

- [ ] **Step 6: Bootstrap Redis in main.go**

In `services/atlas-channel/atlas.com/channel/main.go`: add imports `controllernpc "atlas-channel/npc/controller"` and `atlas "github.com/Chronicle20/atlas/libs/atlas-redis"`; near the top of `main()` (after the bootstrap/logger setup, before consumers are registered) add:

```go
	rc := atlas.Connect(l)
	controllernpc.InitRegistry(rc)
```

(`atlas.Connect` reads `REDIS_URL`/`REDIS_PASSWORD`; both ship in the shared `atlas-env` configmap the channel deployment already mounts.)

- [ ] **Step 7: Module gates + bake**

Run: `go test -race ./... && go vet ./... && go build ./...` (module root)
Run (repo root): `tools/redis-key-guard.sh && docker buildx bake atlas-channel`
Expected: all clean — go.mod changed, so the bake is MANDATORY (CLAUDE.md item 4).

- [ ] **Step 8: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/go.mod services/atlas-channel/atlas.com/channel/go.sum services/atlas-channel/atlas.com/channel/main.go services/atlas-channel/atlas.com/channel/npc/controller/
git commit -m "feat(task-176): NPC-controller Redis registry + channel Redis bootstrap"
```

---

### Task 8: atlas-channel election processor

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/npc/controller/processor.go`
- Test: `services/atlas-channel/atlas.com/channel/npc/controller/processor_test.go`

**Interfaces:**
- Consumes: `Registry` (Task 7); `_map.NewProcessor(l, ctx).GetCharacterIdsInMap(f)` (`atlas-channel/map/processor.go:48`); `buff.NewProcessor(l, ctx).GetByCharacterId` + `buff.IsGmHidden` (`character/buff/processor.go:51`, `character/buff/hidden.go:13`).
- Produces:

```go
type Processor interface {
	TryClaim(f field.Model, npcObjectId uint32, characterId uint32) (bool, error)
	ReleaseFor(f field.Model, characterId uint32) ([]uint32, error)
	ElectFor(f field.Model, npcObjectIds []uint32, exclude ...uint32) (map[uint32]uint32, error)
	UncontrolledIn(f field.Model, npcObjectIds []uint32) ([]uint32, error)
}
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor
func IsController(ctx context.Context, t tenant.Model, f field.Model, characterId uint32, npcObjectId uint32) bool
```

plus test seams on `ProcessorImpl`: `fieldIdsFn func(f field.Model) ([]uint32, error)` and `hiddenFn func(characterId uint32) bool`.

Semantics:
- **TryClaim** (map-enter / reveal path, FR-5.2/FR-6.2): if the NPC's recorded controller is live in the field → no claim, EXCEPT when it is the claimant themself → return true so the caller re-issues the grant (mirrors `spawnMonsterForSession`'s re-issue rationale, `kafka/consumer/map/consumer.go:600-611`). If the entry is stale (controller not among the field's sessions) → release it, then race for it with `Claim` (SetNX) — concurrent enterers race safely: exactly one SetNX wins. A hidden claimant never claims (winner-check via `hiddenFn`, memoized).
- **ReleaseFor** (map-exit / hide): release all entries held by the character; returns the released NPC ids for reassignment/revocation.
- **ElectFor**: least-loaded election over the field's live sessions minus `exclude`, hidden winner-checked lazily (only actual winners get a buff fetch — typically 0-1 REST calls, design §3.2); claims via SetNX (losing a race to a concurrent claim is fine — that NPC just keeps the concurrent winner); returns npcId→winner for the caller to announce. Stale entries among the requested npcIds are released before claiming. No candidates → empty map (FR-5.3/FR-4.3 parity).
- **UncontrolledIn**: filters npcIds to those with no live controller (absent entry, or recorded controller not in the field's sessions).
- **IsController** (movement guard, Task 12): true iff the recorded controller is characterId; **fail-open true** on Redis error or nil registry (never freeze NPC motion on infrastructure failure), false when uncontrolled or another character.
- `ProcessorImpl` memoizes `hiddenFn` results per instance (`hiddenCache map[uint32]bool`); document NOT goroutine-safe — construct one per handler invocation (the codebase's per-invocation processor idiom).

- [ ] **Step 1: Write the failing tests**

`npc/controller/processor_test.go` (uses `setupRegistry` from `registry_test.go`; each test builds a processor with seams):

```go
func testProcessor(t *testing.T, r *Registry, ten tenant.Model, live []uint32, hidden map[uint32]bool) *ProcessorImpl {
	t.Helper()
	registry = r // package-level singleton for the test; restore after
	t.Cleanup(func() { registry = nil })
	l, _ := test.NewNullLogger()
	p := NewProcessor(l, tenant.WithContext(context.Background(), ten)).(*ProcessorImpl)
	p.fieldIdsFn = func(field.Model) ([]uint32, error) { return live, nil }
	p.hiddenFn = func(id uint32) bool { return hidden[id] }
	return p
}

func TestTryClaimClaimsUnclaimed(t *testing.T) {
	r, ten, f := setupRegistry(t)
	p := testProcessor(t, r, ten, []uint32{7}, nil)
	won, err := p.TryClaim(f, 1000, 7)
	if err != nil || !won {
		t.Fatalf("expected claim: won=%v err=%v", won, err)
	}
}

func TestTryClaimRespectsLiveController(t *testing.T) {
	r, ten, f := setupRegistry(t)
	_, _ = r.Claim(context.Background(), ten, f, 1000, 7)
	p := testProcessor(t, r, ten, []uint32{7, 8}, nil)
	won, err := p.TryClaim(f, 1000, 8)
	if err != nil || won {
		t.Fatalf("live controller must be kept: won=%v err=%v", won, err)
	}
}

func TestTryClaimReissuesForCurrentController(t *testing.T) {
	r, ten, f := setupRegistry(t)
	_, _ = r.Claim(context.Background(), ten, f, 1000, 7)
	p := testProcessor(t, r, ten, []uint32{7}, nil)
	won, err := p.TryClaim(f, 1000, 7)
	if err != nil || !won {
		t.Fatalf("current controller must get a re-issue: won=%v err=%v", won, err)
	}
}

func TestTryClaimReplacesStaleController(t *testing.T) {
	r, ten, f := setupRegistry(t)
	_, _ = r.Claim(context.Background(), ten, f, 1000, 99) // 99 not live
	p := testProcessor(t, r, ten, []uint32{7}, nil)
	won, err := p.TryClaim(f, 1000, 7)
	if err != nil || !won {
		t.Fatalf("stale entry must be re-claimed: won=%v err=%v", won, err)
	}
	cur, _, _ := r.ControllerOf(context.Background(), ten, f, 1000)
	if cur != 7 {
		t.Fatalf("expected new controller 7, got %d", cur)
	}
}

func TestTryClaimHiddenClaimsNothing(t *testing.T) {
	r, ten, f := setupRegistry(t)
	p := testProcessor(t, r, ten, []uint32{7}, map[uint32]bool{7: true})
	won, err := p.TryClaim(f, 1000, 7)
	if err != nil || won {
		t.Fatalf("hidden character must not claim: won=%v err=%v", won, err)
	}
}

func TestReleaseForReturnsReleasedIds(t *testing.T) {
	r, ten, f := setupRegistry(t)
	ctx := context.Background()
	_, _ = r.Claim(ctx, ten, f, 1000, 7)
	_, _ = r.Claim(ctx, ten, f, 1001, 7)
	_, _ = r.Claim(ctx, ten, f, 1002, 8)
	p := testProcessor(t, r, ten, []uint32{7, 8}, nil)
	released, err := p.ReleaseFor(f, 7)
	if err != nil || len(released) != 2 {
		t.Fatalf("expected 2 released, got %v err %v", released, err)
	}
	all, _ := r.GetAll(ctx, ten, f)
	if len(all) != 1 {
		t.Fatalf("only 1002 should remain, got %v", all)
	}
}

func TestElectForLeastLoadedSkipsHiddenAndExcluded(t *testing.T) {
	r, ten, f := setupRegistry(t)
	ctx := context.Background()
	// 8 already controls one NPC; 9 is hidden; 10 exiting (excluded); 11 free.
	_, _ = r.Claim(ctx, ten, f, 2000, 8)
	p := testProcessor(t, r, ten, []uint32{8, 9, 10, 11}, map[uint32]bool{9: true})
	got, err := p.ElectFor(f, []uint32{1000, 1001}, 10)
	if err != nil {
		t.Fatalf("ElectFor: %v", err)
	}
	// 11 is least-loaded (0 vs 8's 1) and visible: first NPC -> 11; then
	// counts tie 1-1, either 8 or 11 wins the second — assert both NPCs got
	// a visible, non-excluded winner.
	for npc, winner := range got {
		if winner == 9 || winner == 10 {
			t.Fatalf("npc %d assigned to hidden/excluded %d", npc, winner)
		}
	}
	if len(got) != 2 {
		t.Fatalf("both NPCs must be assigned, got %v", got)
	}
	if got[1000] != 11 && got[1001] != 11 {
		t.Fatalf("least-loaded 11 must win at least one NPC, got %v", got)
	}
}

func TestElectForNoCandidatesLeavesUncontrolled(t *testing.T) {
	r, ten, f := setupRegistry(t)
	p := testProcessor(t, r, ten, []uint32{9}, map[uint32]bool{9: true})
	got, err := p.ElectFor(f, []uint32{1000})
	if err != nil || len(got) != 0 {
		t.Fatalf("only-hidden field must elect nobody: %v err %v", got, err)
	}
	_, ok, _ := r.ControllerOf(context.Background(), ten, f, 1000)
	if ok {
		t.Fatalf("npc must stay uncontrolled")
	}
}

func TestUncontrolledIn(t *testing.T) {
	r, ten, f := setupRegistry(t)
	ctx := context.Background()
	_, _ = r.Claim(ctx, ten, f, 1000, 7)  // live
	_, _ = r.Claim(ctx, ten, f, 1001, 99) // stale
	p := testProcessor(t, r, ten, []uint32{7}, nil)
	unc, err := p.UncontrolledIn(f, []uint32{1000, 1001, 1002})
	if err != nil {
		t.Fatalf("UncontrolledIn: %v", err)
	}
	want := map[uint32]bool{1001: true, 1002: true}
	if len(unc) != 2 || !want[unc[0]] || !want[unc[1]] {
		t.Fatalf("expected {1001,1002}, got %v", unc)
	}
}

func TestIsControllerFailOpen(t *testing.T) {
	// nil registry (pre-init) must fail open to true.
	registry = nil
	ten, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	f := field.NewBuilder(0, 1, 100000000).Build()
	if !IsController(context.Background(), ten, f, 7, 1000) {
		t.Fatalf("nil registry must fail open")
	}
	// uncontrolled NPC -> not controller
	r, ten2, f2 := setupRegistry(t)
	registry = r
	t.Cleanup(func() { registry = nil })
	if IsController(context.Background(), ten2, f2, 7, 1000) {
		t.Fatalf("uncontrolled NPC must not pass the controller guard")
	}
	_, _ = r.Claim(context.Background(), ten2, f2, 1000, 7)
	if !IsController(context.Background(), ten2, f2, 7, 1000) {
		t.Fatalf("recorded controller must pass")
	}
	if IsController(context.Background(), ten2, f2, 8, 1000) {
		t.Fatalf("non-controller must not pass")
	}
}
```

(Direct assignment to the package-level `registry` var in tests bypasses the `sync.Once`; that is intentional and stays within the package.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./npc/controller/... -run 'TestTryClaim|TestReleaseFor|TestElectFor|TestUncontrolledIn|TestIsController' -v`
Expected: FAIL — `Processor` undefined.

- [ ] **Step 3: Implement the processor**

`npc/controller/processor.go`:

```go
package controller

import (
	"atlas-channel/character/buff"
	_map "atlas-channel/map"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	TryClaim(f field.Model, npcObjectId uint32, characterId uint32) (bool, error)
	ReleaseFor(f field.Model, characterId uint32) ([]uint32, error)
	ElectFor(f field.Model, npcObjectIds []uint32, exclude ...uint32) (map[uint32]uint32, error)
	UncontrolledIn(f field.Model, npcObjectIds []uint32) ([]uint32, error)
}

// ProcessorImpl decides NPC-controller assignments. NOT goroutine-safe
// (hiddenCache is unsynchronized) — construct one per handler invocation,
// matching the codebase's per-invocation processor idiom.
type ProcessorImpl struct {
	l           logrus.FieldLogger
	ctx         context.Context
	t           tenant.Model
	fieldIdsFn  func(f field.Model) ([]uint32, error)
	hiddenFn    func(characterId uint32) bool
	hiddenCache map[uint32]bool
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	p := &ProcessorImpl{
		l:           l,
		ctx:         ctx,
		t:           tenant.MustFromContext(ctx),
		hiddenCache: make(map[uint32]bool),
	}
	p.fieldIdsFn = func(f field.Model) ([]uint32, error) {
		return _map.NewProcessor(l, ctx).GetCharacterIdsInMap(f)
	}
	// Winner-check (design §3.2): fetch ONE candidate's buffs from
	// atlas-buffs and test IsGmHidden. Fail-open: an unreachable buffs
	// service must not stall NPC control, so errors read as "not hidden".
	p.hiddenFn = func(characterId uint32) bool {
		bs, err := buff.NewProcessor(l, ctx).GetByCharacterId(characterId)
		if err != nil {
			l.WithError(err).Warnf("Unable to winner-check hide state of [%d]; treating as visible.", characterId)
			return false
		}
		return buff.IsGmHidden(bs)
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) isHidden(characterId uint32) bool {
	if v, ok := p.hiddenCache[characterId]; ok {
		return v
	}
	v := p.hiddenFn(characterId)
	p.hiddenCache[characterId] = v
	return v
}

func contains(ids []uint32, id uint32) bool {
	for _, v := range ids {
		if v == id {
			return true
		}
	}
	return false
}

// TryClaim is the map-enter (and reveal) claim path (FR-5.2). It returns
// true when the caller should send the controller grant to characterId:
// either this call won a fresh/stale-replacement claim, or characterId is
// already the recorded controller (grant re-issue — same rationale as the
// MonsterControl re-issue in spawnMonsterForSession).
func (p *ProcessorImpl) TryClaim(f field.Model, npcObjectId uint32, characterId uint32) (bool, error) {
	r := GetRegistry()
	if r == nil {
		return false, nil
	}
	cur, ok, err := r.ControllerOf(p.ctx, p.t, f, npcObjectId)
	if err != nil {
		return false, err
	}
	if ok {
		if cur == characterId {
			return true, nil
		}
		live, lerr := p.fieldIdsFn(f)
		if lerr != nil {
			return false, lerr
		}
		if contains(live, cur) {
			return false, nil
		}
		// Stale (controller no longer in the field — missed exit or crashed
		// pod): release, then race for it below. Concurrent enterers both
		// reach Claim; SetNX lets exactly one win.
		if derr := r.Release(p.ctx, p.t, f, npcObjectId); derr != nil {
			return false, derr
		}
	}
	if p.isHidden(characterId) {
		p.l.Debugf("Character [%d] is GM-hidden; not claiming NPC [%d] in field [%s].", characterId, npcObjectId, f.Id())
		return false, nil
	}
	return r.Claim(p.ctx, p.t, f, npcObjectId, characterId)
}

// ReleaseFor drops every controller entry held by characterId in f and
// returns the released NPC ids (FR-5.3 / FR-6.1).
func (p *ProcessorImpl) ReleaseFor(f field.Model, characterId uint32) ([]uint32, error) {
	r := GetRegistry()
	if r == nil {
		return nil, nil
	}
	ids, err := r.ControlledBy(p.ctx, p.t, f, characterId)
	if err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	if err := r.Release(p.ctx, p.t, f, ids...); err != nil {
		return nil, err
	}
	p.l.Debugf("Released [%d] NPC controller entries held by character [%d] in field [%s].", len(ids), characterId, f.Id())
	return ids, nil
}

// ElectFor assigns a controller to each requested NPC using the same rule
// as monsters: least-loaded live session, hidden excluded (FR-5.2/FR-6.2),
// no forced transfer semantics — callers pass only NPCs known to need a
// controller. Returns npcId -> winner for announcement. NPCs that lose a
// SetNX race to a concurrent claim are simply omitted.
func (p *ProcessorImpl) ElectFor(f field.Model, npcObjectIds []uint32, exclude ...uint32) (map[uint32]uint32, error) {
	assignments := make(map[uint32]uint32)
	r := GetRegistry()
	if r == nil || len(npcObjectIds) == 0 {
		return assignments, nil
	}
	live, err := p.fieldIdsFn(f)
	if err != nil {
		return assignments, err
	}
	counts := make(map[uint32]int)
	for _, id := range live {
		if contains(exclude, id) {
			continue
		}
		counts[id] = 0
	}
	existing, err := r.GetAll(p.ctx, p.t, f)
	if err != nil {
		return assignments, err
	}
	for _, cid := range existing {
		if _, ok := counts[cid]; ok {
			counts[cid]++
		}
	}
	leastLoaded := func() (uint32, bool) {
		var best uint32
		bestCount := -1
		for id, c := range counts {
			if bestCount == -1 || c < bestCount {
				best = id
				bestCount = c
			}
		}
		return best, bestCount != -1
	}
	for _, npcId := range npcObjectIds {
		if cur, ok := existing[npcId]; ok && !contains(live, cur) {
			if derr := r.Release(p.ctx, p.t, f, npcId); derr != nil {
				p.l.WithError(derr).Warnf("Unable to release stale controller entry for NPC [%d]; skipping.", npcId)
				continue
			}
		}
		var winner uint32
		found := false
		for {
			cand, ok := leastLoaded()
			if !ok {
				break
			}
			if p.isHidden(cand) {
				delete(counts, cand)
				continue
			}
			winner = cand
			found = true
			break
		}
		if !found {
			p.l.Debugf("No eligible NPC controller candidate in field [%s]; NPC [%d] left uncontrolled.", f.Id(), npcId)
			continue
		}
		won, cerr := r.Claim(p.ctx, p.t, f, npcId, winner)
		if cerr != nil {
			p.l.WithError(cerr).Warnf("Unable to claim NPC [%d] for [%d]; skipping.", npcId, winner)
			continue
		}
		if won {
			assignments[npcId] = winner
			counts[winner]++
		}
	}
	return assignments, nil
}

// UncontrolledIn filters npcObjectIds to those with no live controller —
// absent entry, or an entry whose controller is no longer in the field.
func (p *ProcessorImpl) UncontrolledIn(f field.Model, npcObjectIds []uint32) ([]uint32, error) {
	r := GetRegistry()
	if r == nil {
		return nil, nil
	}
	existing, err := r.GetAll(p.ctx, p.t, f)
	if err != nil {
		return nil, err
	}
	live, err := p.fieldIdsFn(f)
	if err != nil {
		return nil, err
	}
	var out []uint32
	for _, npcId := range npcObjectIds {
		cur, ok := existing[npcId]
		if !ok || !contains(live, cur) {
			out = append(out, npcId)
		}
	}
	return out, nil
}

// IsController is the movement/animation guard (task-176 companion change
// 1). Fail-open TRUE on infrastructure failure or pre-init so NPC motion
// never freezes on a Redis outage; false for uncontrolled NPCs and
// non-controllers (spoof/stale-client suppression).
func IsController(ctx context.Context, t tenant.Model, f field.Model, characterId uint32, npcObjectId uint32) bool {
	r := GetRegistry()
	if r == nil {
		return true
	}
	cur, ok, err := r.ControllerOf(ctx, t, f, npcObjectId)
	if err != nil {
		return true
	}
	return ok && cur == characterId
}
```

- [ ] **Step 4: Run tests**

Run: `go test -race ./npc/controller/... -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/npc/controller/
git commit -m "feat(task-176): NPC-controller election processor (hide-aware, least-loaded)"
```

---

### Task 9: announce helpers + spawn-path gating (FR-5.4)

**Files:**
- Create: `services/atlas-channel/atlas.com/channel/npc/controller/announce.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go` (`spawnNPCForSession`, ~line 584)

**Interfaces:**
- Consumes: `Processor.TryClaim` (Task 8), `NewNpcRemoveController` (Task 6), existing `npcpkt.NpcSpawnRequestControllerWriter` / `NewNpcSpawnRequestController`, `session.Announce`, `data/npc` processor, `session.NewProcessor(...).IfPresentByCharacterId`.
- Produces: `controller.AnnounceGrant(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(f field.Model, characterId uint32, npcObjectId uint32) error` and `controller.AnnounceRevoke(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, npcObjectId uint32) error`. Consumed by Tasks 10-11.

- [ ] **Step 1: Implement the announcers**

`npc/controller/announce.go`:

```go
package controller

import (
	"atlas-channel/data/npc"
	"atlas-channel/session"
	"atlas-channel/socket/writer"
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-constants/field"
	npcpkt "github.com/Chronicle20/atlas/libs/atlas-packet/npc/clientbound"
)

// AnnounceGrant sends the controller grant (OnNpcChangeController flag 1)
// for npcObjectId to characterId's session, if present on this pod.
func AnnounceGrant(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(f field.Model, characterId uint32, npcObjectId uint32) error {
	return func(f field.Model, characterId uint32, npcObjectId uint32) error {
		return session.NewProcessor(l, ctx).IfPresentByCharacterId(f.Channel())(characterId, func(s session.Model) error {
			n, err := npc.NewProcessor(l, ctx).GetInMapByObjectId(f.MapId(), npcObjectId)
			if err != nil {
				l.WithError(err).Warnf("Unable to load NPC [%d] for controller grant to [%d].", npcObjectId, characterId)
				return err
			}
			l.Debugf("Granting NPC [%d] control to character [%d] in field [%s].", npcObjectId, characterId, f.Id())
			return session.Announce(l)(ctx)(wp)(npcpkt.NpcSpawnRequestControllerWriter)(npcpkt.NewNpcSpawnRequestController(n.Id(), n.Template(), n.X(), n.CY(), int32(n.F()), n.Fh(), n.RX0(), n.RX1(), true).Encode)(s)
		})
	}
}

// AnnounceRevoke sends the remove-controller arm (flag 0) for npcObjectId
// to s — the client demotes the NPC to remote control (FR-6.1).
func AnnounceRevoke(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) func(s session.Model, npcObjectId uint32) error {
	return func(s session.Model, npcObjectId uint32) error {
		l.Debugf("Revoking NPC [%d] control from character [%d].", npcObjectId, s.CharacterId())
		return session.Announce(l)(ctx)(wp)(npcpkt.NpcSpawnRequestControllerWriter)(npcpkt.NewNpcRemoveController(npcObjectId).Encode)(s)
	}
}
```

(Import-cycle check: `npc/controller` now imports `session`, `data/npc`, `socket/writer`, `map`, `character/buff` — none of those import `npc/controller`, and `movement` (Task 12) already imports `session` + `data/npc` the same way. If `socket/writer` ↔ `session` interplay surprises you, mirror the imports of `movement/processor.go`, which announces packets from a non-handler package today.)

- [ ] **Step 2: Gate the spawn path**

Rewrite `spawnNPCForSession` (`kafka/consumer/map/consumer.go:584-598`) — spawn to everyone, grant ONLY to the elected controller (FR-5.4). A processor is built once per session sweep so the entering character's hide winner-check costs at most one buffs fetch:

```go
func spawnNPCForSession(l logrus.FieldLogger) func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[npc2.Model] {
	return func(ctx context.Context) func(wp writer.Producer) func(s session.Model) model.Operator[npc2.Model] {
		return func(wp writer.Producer) func(s session.Model) model.Operator[npc2.Model] {
			return func(s session.Model) model.Operator[npc2.Model] {
				cp := controllernpc.NewProcessor(l, ctx)
				return func(n npc2.Model) error {
					err := session.Announce(l)(ctx)(wp)(npcpkt.NpcSpawnWriter)(npcpkt.NewNpcSpawn(n.Id(), n.Template(), n.X(), n.CY(), int32(n.F()), n.Fh(), n.RX0(), n.RX1()).Encode)(s)
					if err != nil {
						return err
					}
					// Single-controller election (task-176, FR-5.2/FR-5.4):
					// claim synchronously so NpcSpawn -> grant land on the
					// same session in order; non-controllers get spawn only.
					claimed, cerr := cp.TryClaim(s.Field(), n.Id(), s.CharacterId())
					if cerr != nil {
						l.WithError(cerr).Warnf("NPC-controller claim failed for NPC [%d]; session [%d] gets spawn only.", n.Id(), s.CharacterId())
						return nil
					}
					if !claimed {
						return nil
					}
					return session.Announce(l)(ctx)(wp)(npcpkt.NpcSpawnRequestControllerWriter)(npcpkt.NewNpcSpawnRequestController(n.Id(), n.Template(), n.X(), n.CY(), int32(n.F()), n.Fh(), n.RX0(), n.RX1(), true).Encode)(s)
				}
			}
		}
	}
}
```

Add import `controllernpc "atlas-channel/npc/controller"` to the consumer file.

- [ ] **Step 3: Build + full channel tests**

Run: `go build ./... && go test -race ./... `
Expected: clean/PASS.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/npc/controller/announce.go services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go
git commit -m "feat(task-176): grant NPC control only to the elected controller on spawn"
```

---

### Task 10: NPC reassignment on map exit (FR-5.3)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go` (`handleStatusEventCharacterExit`, ~line 554)

**Interfaces:**
- Consumes: `Processor.ReleaseFor/ElectFor` (Task 8), `AnnounceGrant` (Task 9).

- [ ] **Step 1: Extend the exit handler**

In `handleStatusEventCharacterExit`, after the existing despawn broadcast (line ~566-569), append:

```go
		// NPC-controller reassignment (task-176, FR-5.3): release the
		// exiting character's NPCs and hand them to the least-loaded
		// remaining non-hidden session; none left -> uncontrolled until the
		// next enter (lazy stale re-claim also covers a missed exit).
		cp := controllernpc.NewProcessor(l, ctx)
		released, rerr := cp.ReleaseFor(f, e.Body.CharacterId)
		if rerr != nil {
			l.WithError(rerr).Warnf("Unable to release NPC controller entries for exiting character [%d] in field [%s].", e.Body.CharacterId, f.Id())
			return
		}
		if len(released) == 0 {
			return
		}
		assignments, aerr := cp.ElectFor(f, released, e.Body.CharacterId)
		if aerr != nil {
			l.WithError(aerr).Warnf("Unable to re-elect NPC controllers after character [%d] left field [%s].", e.Body.CharacterId, f.Id())
			return
		}
		for npcId, winner := range assignments {
			if gerr := controllernpc.AnnounceGrant(l, ctx, wp)(f, winner, npcId); gerr != nil {
				l.WithError(gerr).Warnf("Unable to announce NPC [%d] controller grant to [%d].", npcId, winner)
			}
		}
		l.Debugf("NPC-controller exit: character [%d] released [%d] NPCs in field [%s]; reassigned [%d].", e.Body.CharacterId, len(released), f.Id(), len(assignments))
```

(Remove the bare `return` at the end of the current handler body if it would now be mid-function; keep control flow straightforward. `e.Body.CharacterId` is passed as `exclude` — the exiting character may still be present in the map service's session list at consumption time.)

- [ ] **Step 2: Build + tests + commit**

Run: `go build ./... && go test -race ./...`
Expected: clean.

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go
git commit -m "feat(task-176): reassign NPC controllers when the controller leaves the field"
```

---

### Task 11: channel hide/reveal branches — revoke + reassign (FR-6)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go`

**Interfaces:**
- Consumes: `Processor.ReleaseFor/ElectFor/UncontrolledIn` (Task 8), `AnnounceGrant/AnnounceRevoke` (Task 9), `data/npc` `InMapModelProvider`, `skill.SuperGmHideId`.
- Produces: two additional handlers registered on the same topic in `InitHandlers`.

The buff event itself is the trigger; the GM's field comes from their session (`s.Field()`) — no atlas-maps call on the channel side (design §3.2). If the GM's session is gone, the no-op is correct: map-exit already released their NPCs.

- [ ] **Step 1: Add the handlers**

Append to `kafka/consumer/buff/consumer.go` (imports to add: `controllernpc "atlas-channel/npc/controller"`, `npc2 "atlas-channel/data/npc"`, `skill2 "github.com/Chronicle20/atlas/libs/atlas-constants/skill"`):

```go
// handleStatusEventGmHideApplied relinquishes the hiding GM's NPCs
// (task-176, FR-6.1): revoke their client-side grants, then reassign to a
// visible session. Fires ONLY for SuperGmHide (9101004); Dark Sight and
// all other buffs are untouched.
func handleStatusEventGmHideApplied(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.AppliedStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.AppliedStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBuffApplied {
			return
		}
		if e.Body.SourceId != int32(skill2.SuperGmHideId) {
			return
		}
		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}
		session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			f := s.Field()
			cp := controllernpc.NewProcessor(l, ctx)
			released, err := cp.ReleaseFor(f, s.CharacterId())
			if err != nil {
				l.WithError(err).Warnf("GM-hide: unable to release NPC controller entries for [%d] in field [%s].", s.CharacterId(), f.Id())
				return nil
			}
			if len(released) == 0 {
				l.Debugf("GM-hide: character [%d] controlled no NPCs in field [%s].", s.CharacterId(), f.Id())
				return nil
			}
			for _, npcId := range released {
				if rerr := controllernpc.AnnounceRevoke(l, ctx, wp)(s, npcId); rerr != nil {
					l.WithError(rerr).Warnf("GM-hide: unable to revoke NPC [%d] control from [%d].", npcId, s.CharacterId())
				}
			}
			assignments, aerr := cp.ElectFor(f, released, s.CharacterId())
			if aerr != nil {
				l.WithError(aerr).Warnf("GM-hide: unable to re-elect NPC controllers in field [%s].", f.Id())
				return nil
			}
			for npcId, winner := range assignments {
				if gerr := controllernpc.AnnounceGrant(l, ctx, wp)(f, winner, npcId); gerr != nil {
					l.WithError(gerr).Warnf("GM-hide: unable to announce NPC [%d] grant to [%d].", npcId, winner)
				}
			}
			l.Debugf("GM-hide: character [%d] relinquished [%d] NPCs in field [%s]; reassigned [%d].", s.CharacterId(), len(released), f.Id(), len(assignments))
			return nil
		})
	}
}

// handleStatusEventGmHideExpired restores the revealed GM's candidacy
// (FR-6.3): elect controllers for currently-uncontrolled NPCs with the GM
// back in the pool. No forced transfer — live controllers keep their NPCs.
// (atlas-buffs prunes its registry BEFORE emitting EXPIRED, so the
// winner-check cannot see a stale hide buff.)
func handleStatusEventGmHideExpired(sc server.Model, wp writer.Producer) message.Handler[buff2.StatusEvent[buff2.ExpiredStatusEventBody]] {
	return func(l logrus.FieldLogger, ctx context.Context, e buff2.StatusEvent[buff2.ExpiredStatusEventBody]) {
		if e.Type != buff2.EventStatusTypeBuffExpired {
			return
		}
		if e.Body.SourceId != int32(skill2.SuperGmHideId) {
			return
		}
		if !sc.IsWorld(tenant.MustFromContext(ctx), e.WorldId) {
			return
		}
		session.NewProcessor(l, ctx).IfPresentByCharacterId(sc.Channel())(e.CharacterId, func(s session.Model) error {
			f := s.Field()
			npcIds := make([]uint32, 0)
			if err := npc2.NewProcessor(l, ctx).ForEachInMap(f.MapId(), func(n npc2.Model) error {
				npcIds = append(npcIds, n.Id())
				return nil
			}); err != nil {
				l.WithError(err).Warnf("GM-reveal: unable to enumerate NPCs in map [%d].", f.MapId())
				return nil
			}
			cp := controllernpc.NewProcessor(l, ctx)
			unc, err := cp.UncontrolledIn(f, npcIds)
			if err != nil {
				l.WithError(err).Warnf("GM-reveal: unable to compute uncontrolled NPCs in field [%s].", f.Id())
				return nil
			}
			if len(unc) == 0 {
				return nil
			}
			assignments, aerr := cp.ElectFor(f, unc)
			if aerr != nil {
				l.WithError(aerr).Warnf("GM-reveal: unable to elect NPC controllers in field [%s].", f.Id())
				return nil
			}
			for npcId, winner := range assignments {
				if gerr := controllernpc.AnnounceGrant(l, ctx, wp)(f, winner, npcId); gerr != nil {
					l.WithError(gerr).Warnf("GM-reveal: unable to announce NPC [%d] grant to [%d].", npcId, winner)
				}
			}
			l.Debugf("GM-reveal: elected controllers for [%d] of [%d] uncontrolled NPCs in field [%s].", len(assignments), len(unc), f.Id())
			return nil
		})
	}
}
```

- [ ] **Step 2: Register them**

In `InitHandlers` (same file, after the two existing registrations), add:

```go
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventGmHideApplied(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
				id, err = rf(t, message.AdaptHandler(message.PersistentConfig(handleStatusEventGmHideExpired(sc, wp))))
				if err != nil {
					return nil, err
				}
				handles = append(handles, listener.HandlerHandle{Topic: t, Id: id})
```

- [ ] **Step 3: Build + tests + commit**

Run: `go build ./... && go test -race ./...`
Expected: clean.

```bash
git add services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go
git commit -m "feat(task-176): NPC controller relinquish/revoke on GM hide, re-election on reveal"
```

---

### Task 12: NPC movement/animation guard + relay (design companion change 1)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/movement/processor.go` (`ForNPC`, line ~79)
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/npc_action.go` (animation branch, line ~28)

**Interfaces:**
- Consumes: `controller.IsController` (Task 8), `_map2.ForOtherSessionsInMap` (already imported in movement).

Without the relay, single-controller election would freeze NPCs for every non-controller client (today every client animates NPCs locally because everyone gets the grant). The guard also drops action packets from non-controllers/spoofers. `IsController` fails open (true) on Redis failure — motion is never frozen by infrastructure.

- [ ] **Step 1: Guard + relay in ForNPC**

Replace the body of `ForNPC` (`movement/processor.go:79-94`):

```go
func (p *ProcessorImpl) ForNPC(f field.Model, characterId uint32, objectId uint32, unk byte, unk2 byte, movement model.Movement) error {
	routine.Go(p.l, p.ctx, func(_ context.Context) {
		n, err := npc.NewProcessor(p.l, p.ctx).GetInMapByObjectId(f.MapId(), objectId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to retrieve npc moving.")
			return
		}
		// Only the elected controller animates an NPC (task-176); drop
		// non-controller (or stale/spoofed) action packets.
		if !controllernpc.IsController(p.ctx, p.t, f, characterId, objectId) {
			p.l.Debugf("Dropping NPC [%d] movement from non-controller [%d].", objectId, characterId)
			return
		}
		op := session.Announce(p.l)(p.ctx)(p.wp)(npcpkt.NpcActionWriter)(npcpkt.NewNpcActionMove(objectId, unk, unk2, movement).Encode)
		err = p.sp.IfPresentByCharacterId(f.Channel())(characterId, op)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to move npc [%d] for character [%d].", n.Template(), characterId)
		}
		// Relay to every other session (task-176): non-controllers no
		// longer run NPC AI locally, so the controller's actions are their
		// only source of NPC motion.
		if rerr := _map2.NewProcessor(p.l, p.ctx).ForOtherSessionsInMap(f, characterId, op); rerr != nil {
			p.l.WithError(rerr).Errorf("Unable to relay npc [%d] movement to field [%s].", objectId, f.Id())
		}
	})
	return nil
}
```

Add import `controllernpc "atlas-channel/npc/controller"`.

- [ ] **Step 2: Guard + relay the animation branch**

In `socket/handler/npc_action.go`, replace the non-movement branch (after the `GetInMapByObjectId` fetch):

```go
		if !controller.IsController(ctx, tenant.MustFromContext(ctx), s.Field(), s.CharacterId(), p.ObjectId()) {
			l.Debugf("Dropping NPC [%d] animation from non-controller [%d].", p.ObjectId(), s.CharacterId())
			return
		}
		op := session.Announce(l)(ctx)(wp)(npcpacket.NpcActionWriter)(npcpacket.NewNpcActionAnimation(p.ObjectId(), p.Unk(), p.Unk2()).Encode)
		if err = op(s); err != nil {
			l.WithError(err).Errorf("Unable to animate npc [%d] for character [%d].", n.Template(), s.CharacterId())
			return
		}
		if rerr := _map.NewProcessor(l, ctx).ForOtherSessionsInMap(s.Field(), s.CharacterId(), op); rerr != nil {
			l.WithError(rerr).Errorf("Unable to relay npc [%d] animation to field.", p.ObjectId())
		}
```

Add imports `controller "atlas-channel/npc/controller"`, `_map "atlas-channel/map"`, `tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"` (match the file's alias conventions; if `movement` uses `controllernpc` as the alias, use the same alias here for consistency).

- [ ] **Step 3: Build + tests + commit**

Run: `go build ./... && go test -race ./...`
Expected: clean.

```bash
git add services/atlas-channel/atlas.com/channel/movement/processor.go services/atlas-channel/atlas.com/channel/socket/handler/npc_action.go
git commit -m "feat(task-176): NPC action guard + relay — controller drives, others receive"
```

---

### Task 13: full verification sweep

**Files:** none created — gates only.

- [ ] **Step 1: Per-module gates**

From each changed module root (`services/atlas-monsters/atlas.com/monsters`, `services/atlas-channel/atlas.com/channel`, `libs/atlas-packet`):

```bash
go test -race ./... && go vet ./... && go build ./...
```

Expected: all clean.

- [ ] **Step 2: Repo-root guards**

```bash
tools/redis-key-guard.sh
tools/goroutine-guard.sh
tools/lint.sh --check
```

Expected: all exit 0. If `lint.sh --check` fails, run fix mode (`tools/lint.sh`), re-check, and fold the formatting into the offending commit or a `style(task-176)` commit.

- [ ] **Step 3: Docker bakes**

```bash
git diff main --stat -- '**/go.mod'
docker buildx bake atlas-channel
```

Expected: channel go.mod is the only changed go.mod; its bake succeeds. Bake any other service whose go.mod shows in the diff.

- [ ] **Step 4: Packet matrix check**

```bash
go run ./tools/packet-audit matrix --check
```

Expected: exit 0, no drift (Task 6 made no wire change to any verified codec).

- [ ] **Step 5: Acceptance criteria walkthrough (PRD §10)**

Verify each criterion has evidence, citing file:line or test name — monsters relinquish/exclude/reveal (Task 2/4 tests), Dark Sight untouched (Task 4 filter test), NPC single-controller + grant-gating (Task 8/9), revoke on hide (Task 11), multi-replica correctness (all state in Redis — Tasks 1, 7; buff consumption shared-group in monsters, per-pod session-routed in channel), fail-safe location handling (Task 4 mutation-before-location test). The ≥2-replica live walk (hide consumed on one pod, election on another) is an environment test for the PR review/deploy phase — note it in the PR description.

- [ ] **Step 6: Commit any residue, then request code review**

Per CLAUDE.md, run `superpowers:requesting-code-review` (plan-adherence + backend-guidelines reviewers; include `packet-completeness-critic` since `libs/atlas-packet` changed) before opening the PR.

---

## Self-Review

**Spec coverage** — design §2 D1 (channel owns NPC election): Tasks 7-11. §3.1 (monsters projection + handler ordering): Tasks 1, 4. §3.2 (no channel projection; winner-check): Task 8. §3.3 (lifecycle/reconciliation): Task 5. §4 (choke-point exclusion, sentinel, fail-open, pool-leak fix, DPS-guard): Task 2. §5.1 (registry, HSETNX, stale repair, no TTLs): Tasks 7-8. §5.2 (triggers): Tasks 9-11. §5.3 (packets): Tasks 6, 9. §5.4 (movement relay + guard): Task 12. §6 error table: distributed across Tasks 2/4/8 fail-open branches. §7 observability: debug/warn logs in every mutation path. §8 testing: per-task test steps. §9 facts: all five resolved in Global Constraints. PRD FR-1 → Tasks 1/4; FR-2/3 → Task 4; FR-4 → Task 2; FR-5 → Tasks 7-10; FR-6 → Task 11; FR-7 → Tasks 4/8 failure branches.

**Known deltas from design, intentional:** (1) `KeyedHash.SetNX` addition dropped — already exists (`keyed_hash.go:33`). (2) A DPS-leader-switch hidden guard was added (Task 2f) — the acceptance criterion "a hidden GM is never selected" covers a path the design's §4 choke-point list missed. (3) The monsters hidden registry stores tenant identity in the payload (monster-registry pattern) so the sweep can iterate tenants — the design's bare "TenantKeyedSet" alone cannot enumerate tenants for reconciliation.

**Type consistency** — `ErrNoControllerCandidate` (Task 2) consumed by Task 4 via `FindNextController` semantics; `GetCharacterField` (Task 3) consumed by Task 4's default `locationFn`; `hidden.GetRegistry()` API used identically in Tasks 2/4/5; `Registry` methods (Task 7) match every call in Task 8; `TryClaim/ReleaseFor/ElectFor/UncontrolledIn` signatures match Tasks 9-11 call sites; `NewNpcRemoveController` (Task 6) matches Task 9's `AnnounceRevoke`; `AnnounceGrant(l, ctx, wp)(f, winner, npcId)` matches Tasks 10-11.

**Placeholder scan** — test snippets that depend on the module's existing private helpers (`testField`, mob creation, reader construction) explicitly instruct adapting to the file's established idioms rather than inventing parallel scaffolding; no TBD/TODO markers; every code step shows the code.
