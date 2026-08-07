# Backend Guidelines Audit — task-167-homing-beacon-bullseye

- **Worktree:** `.worktrees/task-167-homing-beacon-bullseye` (repo-relative)
- **Branch:** `task-167-homing-beacon-bullseye` (confirmed via `git branch --show-current`)
- **Scope:** `git diff main..HEAD` (24 commits) across three modules:
  - `libs/atlas-packet`
  - `services/atlas-buffs/atlas.com/buffs`
  - `services/atlas-channel/atlas.com/channel`
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*` (ai-guidance, file-responsibilities, anti-patterns, testing-guide, patterns-provider, patterns-multitenancy-context, patterns-rest-jsonapi, patterns-functional, patterns-ingress-documentation, patterns-deploy, patterns-cache, scaffolding-checklist)
- **Date:** 2026-08-05
- **Build:** PASS (`go build ./...` clean in all three modules)
- **Tests:** PASS — `go test ./... -count=1` clean in all three modules (no failures; `[no test files]` packages listed as such, not failures)
- **Guard scripts:** `tools/goroutine-guard.sh` exit 0, `tools/redis-key-guard.sh` exit 0, `tools/skill-job-id-guard.sh` exit 0 ("14 divergent const(s) checked", clean)
- **Overall:** NEEDS-WORK — build/tests pass, but one Important and one Minor finding below.

## Build & Test Results

```
cd libs/atlas-packet && go build ./... && go test ./... -count=1   # PASS, all packages ok or [no test files]
cd services/atlas-buffs/atlas.com/buffs && go build ./... && go test ./... -count=1   # PASS
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./... -count=1   # PASS
```

No go.mod changes in any of the three modules (`git diff main..HEAD -- '*/go.mod'` empty) — DOM-22 (Dockerfile lib-mention count) and DOM-23 (Kafka topic naming/configmap) are N/A: no new `libs/atlas-X` dependency and no new `COMMAND_TOPIC_*`/`EVENT_TOPIC_*` env var was introduced by this diff. Confirmed via `git diff --name-status main..HEAD | grep -v '\.go$'` — no `deploy/k8s/*`, `services.json`, or Dockerfile touched. Scaffolding checklist (SCAFFOLD-01..08) is N/A — no new service directory added.

## Domain Discovery

This is a diff-scoped audit (not a full-service sweep): the checklist below is applied to every package the diff touches, per the task's explicit instruction set. Packages touched:

| Package | Classification | Notes |
|---|---|---|
| `libs/atlas-packet/character/clientbound` (buff_cancel.go) | Wire-codec (packet library, not DOM/SUB) | Writer structs, version-gated via `tenant.Model` |
| `libs/atlas-packet/model` (character_temporary_stat.go) | Wire-codec (packet library) | Core CTS mask/base-block encode/decode |
| `services/atlas-buffs/.../buff` | Domain package (`model.go` present) | Redis-backed, no `entity.go`/GORM — see DOM-01 note |
| `services/atlas-buffs/.../character` | Domain package (`model.go`, `registry.go` in place of `entity.go`/`provider.go`) | Redis registry, not GORM |
| `services/atlas-buffs/.../kafka/consumer/character`, `.../characterstatus` | Sub-domain (consumer.go, no model.go) | SUB checklist |
| `services/atlas-buffs/.../kafka/message/character` | Support (Kafka contract types) | FILE checklist |
| `services/atlas-channel/.../character/buff` | Domain package (`model.go` present) | Also holds new `beacon.go` (see findings) |
| `services/atlas-channel/.../kafka/consumer/buff` | Sub-domain (consumer.go, no model.go) | SUB checklist |
| `services/atlas-channel/.../kafka/message/buff` | Support (Kafka contract types) | FILE checklist |
| `services/atlas-channel/.../socket/handler` | Support (packet handlers) | `character_attack_common.go` |

## DOM-21 — Reuse of `libs/atlas-constants` (PASS)

- `skill3.OutlawHomingBeaconId` (`Id(5211006)`) and `skill3.CorsairBullseyeId` (`Id(5220011)`) are pre-existing constants in `libs/atlas-constants/skill/constants.go:3230,3241` (unchanged by this diff — `git diff main..HEAD -- libs/atlas-constants/` is empty). Used directly in `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:679` — no local redeclaration.
- `charconst.TemporaryStatTypeHomingBeacon` (`"HOMING_BEACON"`) is a pre-existing constant in `libs/atlas-constants/character/temporary_stat.go:121`, used consistently across `beacon.go`, `consumer.go` (both services), and `character_attack_common.go` — no local string-literal redeclaration of the stat name found.
- **PASS.**

## DOM-25 — Client wire values are config/version-resolved, never Go literals in a service (PASS)

- All version-dependent framing decisions (GMS v61's 6-member/88-byte two-state group vs. the 7-member/110-byte classic group; the movement-filter byte gating; the v92 Flying/Frozen inference) live entirely inside `libs/atlas-packet/model/character_temporary_stat.go`, gated via `tenant.Model` (`isGmsV61`, `movementAffectingStatNames`, `MovementAffectingMask`) — `libs/atlas-packet/model/character_temporary_stat.go:387-401` (`isGmsV61`), `:719-816` (`movementAffectingStatNames`).
- No service (`atlas-buffs`, `atlas-channel`) contains a hardcoded client wire byte, opcode, or raw version comparison for this feature — `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go` only calls `bp.ApplyNoExpiry(...)` with a domain-level stat name and the struck monster's object id; it never touches wire bytes.
- The cancel-mask fix (`EncodeCancelMask`, `CancelMask`, `libs/atlas-packet/model/character_temporary_stat.go:684-716`) computes the mask from the CTS's own `stats` map — no service-supplied byte.
- **PASS.**

## Kafka contract — `noExpiry` JSON tag parity (PASS)

- Producer side (`services/atlas-buffs/atlas.com/buffs/kafka/message/character/kafka.go`): `NoExpiry bool \`json:"noExpiry,omitempty"\`` added identically to `ApplyCommandBody`, `AppliedStatusEventBody`, `ExpiredStatusEventBody` (diff hunks confirmed via `git diff`).
- Consumer side (`services/atlas-channel/atlas.com/channel/kafka/message/buff/kafka.go`): same three structs, same field name, same tag `json:"noExpiry,omitempty"`.
- **PASS** — no silent mismatch.

## Immutable domain models (PASS, with one exception — see Findings)

| Type | File | Verdict |
|---|---|---|
| `buff.Model` (atlas-buffs) | `services/atlas-buffs/atlas.com/buffs/buff/model.go:12-21` | PASS — private fields (`id`, `sourceId`, `level`, `duration`, `changes`, `createdAt`, `expiresAt`, `noExpiry`), all access via getters, construction via `NewBuff`/`NewNoExpiryBuff` (`model.go:144,165`) |
| `buff.Model` (atlas-channel) | `services/atlas-channel/atlas.com/channel/character/buff/model.go:22-30` | PASS — private fields, getters (`SourceId()`, `Level()`, `Changes()`, `CreatedAt()`, `Expired()`, `ExpiresAt()`, `NoExpiry()`), construction via `NewBuff` (`model.go:63`) |
| `BeaconEntry` (atlas-channel) | `services/atlas-channel/atlas.com/channel/character/buff/beacon.go:13-17` | **FAIL — see Finding 1** |

## Concurrency — `character/buff/beacon.go` `BeaconMirror` (PASS on correctness, Minor pattern deviation — see Finding 2)

- `BeaconMirror.Set`/`Clear` take `m.mu.Lock()`/`defer m.mu.Unlock()` (`beacon.go:50-51, 62-63`); `Get` takes `m.mu.RLock()`/`defer m.mu.RUnlock()` (`beacon.go:71-72`). All three methods hold the lock for their entire critical section and release via `defer` — no unlocked window, no risk of a torn read on the inner `map[uint32]BeaconEntry`.
- Singleton init is via `sync.Once` (`beacon.go:34,40-44`) — correct double-checked-init idiom, no lazy-init race.
- Lock granularity is one `sync.RWMutex` for the whole `perTenant` map (not per-tenant) — this is a correctness non-issue (coarser than necessary, never wrong), so not flagged as a defect, only noted.
- **PASS on correctness.** See Finding 2 for the pattern-doc deviation (cache.go template: interface-typed return + exported test-reset helper).

## Findings

### Finding 1 (Important) — `BeaconEntry` violates the immutable-domain-model pattern

**File:** `services/atlas-channel/atlas.com/channel/character/buff/beacon.go:13-17`

```go
type BeaconEntry struct {
	SourceId int32
	Level    byte
	MobId    int32
}
```

`patterns-functional.md` ("Immutability"): *"Domain models have private fields. Public getters expose read-only state. All mutations occur via builders returning new instances."* `BeaconEntry` is constructed and read directly via exported fields at every call site (`consumer.go:153` `buff.BeaconEntry{SourceId: e.Body.SourceId, Level: e.Body.Level, MobId: bc.Amount}`; `consumer.go:372` `e.SourceId`, `e.Level`, `e.MobId`; `beacon_test.go:31,38,57` construct it as a struct literal). There is no constructor function and no getter methods — any caller in the package can mutate a `BeaconEntry` value's fields directly.

This sits in the same package (`character/buff`) as `Model` (`model.go:22-30`), which correctly implements the private-field+getter+constructor convention just a few files away — `BeaconEntry` is the outlier within its own domain package, not a package-wide idiom being applied consistently.

**Severity:** Important, per the audit brief's explicit instruction to grade `BeaconEntry`/`BeaconMirror` against the immutable-model requirement, and per the "structural violation defaults to Important" rule (this is a documented core pattern in `patterns-functional.md`, not an incidental style choice).

**Practical impact:** Low-to-moderate — `BeaconEntry` values are always passed by value into `BeaconMirror.Set` (`beacon.go:49`) and never aliased/shared as a pointer, so there is no live mutable-aliasing bug today. The finding is a guideline-compliance gap (exported mutable fields, no constructor/getters), not a demonstrated runtime defect.

### Finding 2 (Minor) — `BeaconMirror` deviates from the documented `patterns-cache.md` singleton template

**File:** `services/atlas-channel/atlas.com/channel/character/buff/beacon.go:27-79`, `beacon_test.go:22-23,51-52`

`BeaconMirror` is structurally the "Multi-Tenant Cache Pattern" documented in `patterns-cache.md` (§"Multi-Tenant Cache Pattern": `sync.Once` singleton, `sync.RWMutex`, `map[uuid.UUID]map[uint32]T]`, `GetCache()` accessor) — and correctly implements the core singleton-scope guarantee. Two documented specifics of that template are not followed:

1. `GetBeaconMirror()` returns the concrete type `*BeaconMirror` (`beacon.go:39`), not an interface (`patterns-cache.md`'s template returns `CacheInterface`, a `Get`/`Put`-shaped interface, specifically so callers can be tested against a substitutable type).
2. No exported test-reset/setter helper is provided (`patterns-cache.md` §"Testing Pattern": `SetCacheForTesting`/`ResetCache`). `beacon_test.go:22-23` and `:51-52` instead reset the package-private `beaconMirrorOnce`/`beaconMirror` vars directly from within the same package (white-box), which works today only because the test lives in `package buff` itself — it is not the documented pattern.

**Severity:** Minor — the load-bearing guarantees (application-wide singleton scope, thread safety via `RWMutex`, no per-request cache) are all correctly implemented; the deviation is from the template's specific shape, not from its functional requirement. Downgraded from the FILE-06 "no package-named catch-all" class of Important findings because `beacon.go` is a genuinely single-purpose file (it does not bundle Processor/RestModel/requests responsibilities) — the gap is against `patterns-cache.md`'s more detailed template, not `file-responsibilities.md`'s "which file holds which responsibility" table.

## Non-Findings Worth Recording (evaluated, no violation)

- **DOM-01 (builder.go) — `services/atlas-buffs/.../buff`:** This package has no `builder.go`; construction is via `NewBuff`/`NewNoExpiryBuff`. Confirmed via `git diff main..HEAD -- services/atlas-buffs/atlas.com/buffs/buff/model.go` that `NewBuff` and the constructor-not-builder shape pre-date this branch — the diff only adds a second, symmetric constructor (`NewNoExpiryBuff`) following the file's own pre-existing convention. Not a new deviation introduced by this diff; out of scope for a diff-scoped audit that is not re-architecting this file.
- **DOM-24 (Kafka producer stubbing in tests):** `services/atlas-buffs/atlas.com/buffs/character/testmain_test.go` and `.../kafka/consumer/characterstatus/testmain_test.go` both call `producertest.InstallNoop()` in `TestMain` (pre-existing, unmodified by this diff) — the new `characterstatus/consumer_test.go` tests (`TestHandleMapChangedCancelsHomingBeacon` etc., which exercise `character.NewProcessor(l, ctx).CancelByStatTypes` → `message.Emit`) run under that stub. No `t.Cleanup(producer.ResetInstance)` found anywhere in the diff. `services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer_test.go` does not need a stub: its handlers write via `writer.Producer` (socket writes, stubbed with `noOpWriterProducer`, `consumer_test.go:109-111`), not the Kafka producer — no emit path in scope.
- **Skill-id raw comparison guard:** `character_attack_common.go:679` (`sid != skill3.OutlawHomingBeaconId && sid != skill3.CorsairBullseyeId`) is a raw equality comparison against skill wire constants, which `tools/skill-job-id-guard.sh` would flag if either constant were on the task-187 divergence list. Ran the guard: exit 0, "14 divergent const(s) checked". Independently confirmed neither `5211006` nor `5220011` appears in `docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv`.
- **Expiry-ticker exclusion for no-expiry buffs:** Achieved entirely through `Model.Expired()`'s short-circuit on the `noExpiry` flag (`buff/model.go:38-43`), consulted uniformly at every registry expiry site (`character/registry.go:204,310,372` — `GetExpired`, plus two other `b.Expired()` call sites) — no separate/duplicated filter logic that could drift out of sync.
- **No stub `*_testhelpers.go` files** added anywhere in the diff (`git diff --name-status` shows none).
- **TODO hygiene:** the diff removes exactly one TODO (`// TODO Homing Beacon / Bullseye`, formerly at `character_attack_common.go`) and replaces it with the implemented `beaconTryApply` call; the ~6 unrelated sibling TODOs in the same switch block (Bandit Steal, Fire/Ice Demon weaken, Flame Thrower, Snow Charge, Hamstring) are untouched, as the task brief said was expected. No new TODO/stub/501 introduced anywhere in the diff (`git diff | grep TODO` shows only the pre-existing untouched lines).

## Summary

### Blocking (must fix)
- **Finding 1 (Important):** `BeaconEntry` (`services/atlas-channel/atlas.com/channel/character/buff/beacon.go:13-17`) uses exported mutable fields with no constructor/getters, violating the immutable-domain-model convention documented in `patterns-functional.md`.

### Non-Blocking (should fix)
- **Finding 2 (Minor):** `BeaconMirror`'s singleton accessor returns a concrete type rather than an interface, and its tests reset singleton state via direct package-private var access rather than an exported test helper, deviating from `patterns-cache.md`'s documented template (`beacon.go:27-79`).
