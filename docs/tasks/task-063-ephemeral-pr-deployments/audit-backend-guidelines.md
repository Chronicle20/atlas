# Backend Audit — task-063 Phases 1–5 (Ephemeral PR Deployments)

- **Worktree:** `<worktree-root>`
- **Branch:** `task-063-ephemeral-pr-deployments`
- **Range:** `10d61861f150fce94f01d7b5da32917c4c5b4c40` → `1055cc725630cc86feeb0b6c3a2412bfc9377e00` (24 commits)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-05-08
- **Build:** PASS (libs + 16 sampled services)
- **Tests:** PASS (default env)
- **Overall:** NEEDS-WORK

The task touches 70 production Go files and 3 test files across 3 libs and 49 service `main.go` files plus 14 service registry/cache files. The scope is purely infrastructure plumbing (Redis key prefix, Kafka consumer group resolver, object-id lib rewire, plus 49 main.go and 14 registry/cache call-site rewires). It does NOT introduce new domain types, new processors, new resources, new REST models, new builders, new entities, new providers, new administrators, or new producers. As a result, the standard DOM-* per-domain checklist mostly does not apply: the modified `registry.go` / `cache.go` / `cooldown_*.go` files in domain packages are infrastructure adjuncts, not the load-bearing DDD artifacts that DOM-01 through DOM-21 evaluate. Where DOM-* items are genuinely affected by the changes (DOM-12 no `os.Getenv` in handlers; DOM-21 no atlas-constants duplication), they are evaluated below.

The audit therefore focuses on:

1. The three new/modified library packages (`libs/atlas-redis`, `libs/atlas-kafka/consumergroup`, `libs/atlas-object-id`).
2. DOM-12 / DOM-21 spot-checks across the call-site changes.
3. The reviewer's six specific concerns (init-order, race-safety, import hygiene, alias collision, atlas-constants duplication, test coverage).

---

## Phase 1: Build and Test Gate

### Library builds

| Lib | Build | Tests |
|-----|-------|-------|
| `libs/atlas-redis` | PASS (`libs/atlas-redis/keys.go:1`) | PASS, including `-race` (`libs/atlas-redis/keys_test.go:1`) |
| `libs/atlas-kafka/consumergroup` | PASS (`libs/atlas-kafka/consumergroup/resolver.go:1`) | PASS (3 tests, `libs/atlas-kafka/consumergroup/resolver_test.go:1`) |
| `libs/atlas-object-id` | PASS (`libs/atlas-object-id/allocator.go:1`) | PASS (5 tests including the env-prefix guard) |

### Service builds (sampled across Phase 4 + Phase 5 surface)

Built clean: `atlas-account`, `atlas-buffs`, `atlas-channel`, `atlas-login`, `atlas-monsters`, `atlas-pets`, `atlas-rates`, `atlas-skills`, `atlas-storage`, `atlas-portals`, `atlas-maps`, `atlas-npc-shops`, `atlas-character`, `atlas-chairs`, `atlas-chalkboards`, `atlas-expressions` (16/49). Test runs across the same 16 services pass under default env (`ATLAS_ENV` unset).

### Test gate failure under non-default env (BLOCKING — see BG-01)

Running the same `atlas-monsters` test suite with `ATLAS_ENV=ci42` produces:

```
--- FAIL: TestAllocator_RecyclesLIFONearExhaustion (0.00s)
    id_allocator_test.go:94: Expected LIFO recycled 1000030, got 1000000
FAIL    atlas-monsters/monster
```

Caused by `services/atlas-monsters/atlas.com/monsters/monster/id_allocator_test.go:80` hardcoding `"atlas:oid:" + ten.Id().String() + ":next"` for direct miniredis priming. Under any non-empty `ATLAS_ENV` the production code now writes the counter to `<ATLAS_ENV>:atlas:oid:...:next` (per `libs/atlas-object-id/allocator.go:104`), so the test primes a key the allocator never reads. The allocator therefore initializes a fresh counter at `MinId` instead of seeing the test-primed `RecycleThreshold` value, which makes `Allocate` return `MinId` (1,000,000) instead of the LIFO-recycled `MinId+30` (1,000,030).

This is the exact failure mode `task-063` was designed to fix in production. The test file was not on Phase 5's audit-redis-prefix list (it was filtered out as a `_test.go` file), but it lives in the same package as `services/atlas-monsters/atlas.com/monsters/monster/registry.go` which Phase 5 *did* migrate. The implementer was already in this directory and missed it.

A second residual test reference is `services/atlas-monsters/atlas.com/monsters/monster/attack_cooldown_test.go:49`, which asserts a key does *not* equal `"atlas:monster-cooldown:..."`. Under non-default `ATLAS_ENV` the assertion is vacuous (no key in miniredis matches the now-stale "atlas:" literal anyway), so the test passes by accident rather than by intent. Not currently a regression, but brittle.

---

## Phase 2: Domain Discovery

The diff touches 14 directories that contain `model.go` (and are therefore domain packages by the audit's normal definition):

```
services/atlas-account/atlas.com/account/account
services/atlas-buffs/atlas.com/buffs/character
services/atlas-chairs/atlas.com/chairs/character
services/atlas-chalkboards/atlas.com/chalkboards/character
services/atlas-character/atlas.com/character/session
services/atlas-expressions/atlas.com/expressions/expression
services/atlas-maps/atlas.com/maps/map/monster
services/atlas-monsters/atlas.com/monsters/monster
services/atlas-npc-shops/atlas.com/npc/shops
services/atlas-pets/atlas.com/pets/character
services/atlas-portals/atlas.com/portals/blocked
services/atlas-rates/atlas.com/rates/character
services/atlas-skills/atlas.com/skills/skill
services/atlas-storage/atlas.com/storage/storage
```

In every case the **only** files modified within the package are `registry.go`, `cache.go`, `cooldown.go`, `attack_cooldown.go`, `drop_timer_registry.go`, `cooldown_registry.go`, or `item_tracker.go` — i.e. infrastructure adjuncts to the domain, not the DDD load-bearing files (`model.go`, `entity.go`, `builder.go`, `processor.go`, `provider.go`, `administrator.go`, `producer.go`, `resource.go`, `rest.go`). Verified via `git diff --name-only 10d61861f150fce94f01d7b5da32917c4c5b4c40 1055cc725630cc86feeb0b6c3a2412bfc9377e00 -- '*.go'`.

Therefore DOM-01 through DOM-11 and DOM-13 through DOM-20 are **not in scope** for this audit — they evaluate the structure of files this task did not touch. DOM-12 (no `os.Getenv` in handlers) and DOM-21 (no atlas-constants duplication) ARE in scope and evaluated below.

---

## Phase 3: Per-Lib Mechanical Checks

### `libs/atlas-redis` (Phase 1 of plan)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| LIB-RD-01 | `KeyPrefix()` exported with godoc explaining contract | PASS | `libs/atlas-redis/keys.go:25-29` ("// KeyPrefix returns the env-aware key prefix.") |
| LIB-RD-02 | `computeKeyPrefix(atlasEnv string)` is pure (testable without env mutation) | PASS | `libs/atlas-redis/keys.go:18-23` (takes string, returns string, no side effects) |
| LIB-RD-03 | Empty-string env yields legacy literal "atlas" (back-compat) | PASS | `libs/atlas-redis/keys.go:19-21` and tested at `libs/atlas-redis/keys_test.go:10-15` |
| LIB-RD-04 | Non-empty env yields `<atlasEnv>:atlas` shape | PASS | `libs/atlas-redis/keys.go:22` and tested at `libs/atlas-redis/keys_test.go:17-22` |
| LIB-RD-05 | `keyPrefixBase` constant exists and is "atlas" | PASS | `libs/atlas-redis/keys.go:11` |
| LIB-RD-06 | Existing `namespacedKey` / `tenantEntityKey` / `tenantScanPattern` use the env-aware `keyPrefix` | PASS | `libs/atlas-redis/keys.go:38-51` route through `keyPrefix` (no longer the literal "atlas") |
| LIB-RD-07 | Tests cover `namespacedKey` env-aware behavior end-to-end | PASS | `libs/atlas-redis/keys_test.go:30-40` (`TestNamespacedKey_useEnvAwarePrefix`) |
| LIB-RD-08 | Tests cover `tenantEntityKey` env-aware behavior | PASS | `libs/atlas-redis/keys_test.go:42-56` |
| LIB-RD-09 | `KeyPrefix()` test guards against empty return | PASS | `libs/atlas-redis/keys_test.go:24-28` |
| LIB-RD-10 | Test mutation of `keyPrefix` uses `t.Cleanup` for restoration | PASS | `libs/atlas-redis/keys_test.go:31-32, 43-44` |
| LIB-RD-11 | No dead code left behind (legacy `const keyPrefix = "atlas"` removed) | PASS | `libs/atlas-redis/keys.go:11` only carries `keyPrefixBase`; old top-level `keyPrefix` const is gone |
| LIB-RD-12 | Race-safe: tests pass with `-race` | PASS | `cd libs/atlas-redis && go test -race ./... -count=1` returns ok |

### `libs/atlas-kafka/consumergroup` (Phase 2 of plan)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| LIB-CG-01 | New package has package godoc | PASS | `libs/atlas-kafka/consumergroup/resolver.go:1-7` |
| LIB-CG-02 | `Resolve(default string) string` is pure-with-env | PASS | `libs/atlas-kafka/consumergroup/resolver.go:19-24` |
| LIB-CG-03 | Empty `KAFKA_CONSUMER_GROUP` returns default | PASS | `libs/atlas-kafka/consumergroup/resolver.go:20-22`, tested `resolver_test.go:7-12` |
| LIB-CG-04 | Non-empty value returned verbatim (not trimmed) | PASS | `resolver.go:20-22`, tested `resolver_test.go:14-19` |
| LIB-CG-05 | Whitespace-only value returned verbatim (design §5.4 — surface config bugs) | PASS | tested `resolver_test.go:21-28` (named `TestResolve_envWhitespaceOnly_returnsVerbatim`) |
| LIB-CG-06 | Single env-var literal `KAFKA_CONSUMER_GROUP` referenced via `const envVar` | PASS | `libs/atlas-kafka/consumergroup/resolver.go:11` |
| LIB-CG-07 | Tests use `t.Setenv` (auto-restored, race-safe) | PASS | `resolver_test.go:8, 15, 22` |

### `libs/atlas-object-id` (Phase 3 of plan)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| LIB-OID-01 | `counterKey` uses `atlasredis.KeyPrefix()` not literal "atlas" | PASS | `libs/atlas-object-id/allocator.go:103-105` |
| LIB-OID-02 | `freeKey` uses `atlasredis.KeyPrefix()` not literal "atlas" | PASS | `libs/atlas-object-id/allocator.go:107-109` |
| LIB-OID-03 | New regression test asserts env-prefix shape | PASS | `libs/atlas-object-id/allocator_test.go:133-158` (`TestAllocator_keysRespectEnvPrefix`) |
| LIB-OID-04 | Regression test is non-tautological (independent literal guard) | PASS | `allocator_test.go:155-157` (`if !strings.Contains(gotNext, "atlas")`); the test recomputes `prefix := atlasredis.KeyPrefix()` and then re-asserts the "atlas" literal independently to catch a regression where `KeyPrefix()` drifts from documented shape |
| LIB-OID-05 | Pre-existing tests still pass | PASS | `cd libs/atlas-object-id && go test ./... -count=1` ok |

---

## Phase 4: Per-Service Mechanical Checks (49 service `main.go`)

The change pattern is uniform: `const consumerGroupId = "<literal>"` becomes `var consumerGroupId = consumergroup.Resolve("<literal>")`. Sampled spot-checks:

| Sample | Check | Status | Evidence |
|--------|-------|--------|----------|
| atlas-account | `var consumerGroupId = consumergroup.Resolve(...)` at top-level scope | PASS | `services/atlas-account/atlas.com/account/main.go:23` |
| atlas-channel | Runtime template wrapped in `Resolve()` | PASS | `services/atlas-channel/atlas.com/channel/main.go:148` |
| atlas-login | Runtime template wrapped in `Resolve()` | PASS | `services/atlas-login/atlas.com/login/main.go:66` |
| All 49 | `consumergroup.Resolve` referenced exactly 49 times across `services/` | PASS | `grep -rln 'consumergroup.Resolve' services/` returns 49 |
| Full sweep | Zero stragglers using `const consumerGroupId = "..."` | PASS | `grep -rln 'const consumerGroupId' services/` returns only `atlas-channel` and `atlas-login`, both of which keep `const consumerGroupIdTemplate = "..."` (template, not the resolved value), and define `var consumerGroupId = consumergroup.Resolve(fmt.Sprintf(consumerGroupIdTemplate, ...))` |

### DOM-12 (no new `os.Getenv` in handlers) — Phase 4

| Check | Status | Evidence |
|-------|--------|----------|
| No new `os.Getenv` introduced in any `resource.go` | PASS | `grep -rn 'os.Getenv' --include='resource.go' services/` returns zero matches |
| Env reads localized to `libs/atlas-redis/keys.go:14` and `libs/atlas-kafka/consumergroup/resolver.go:20` | PASS | Confirmed via `grep -rn 'os.Getenv("ATLAS_ENV")\|os.Getenv("KAFKA_CONSUMER_GROUP")' libs/ services/` |

### Import hygiene — Phase 4

| Check | Status | Evidence |
|-------|--------|----------|
| Some main.go files use bare `consumergroup` import, others use the explicit alias `consumergroup "..."` — both forms valid since alias matches package name | PASS | Sampled across `services/atlas-buffs/atlas.com/buffs/main.go`, `services/atlas-fame/atlas.com/fame/main.go`, etc.; all use `consumergroup "github.com/Chronicle20/atlas/libs/atlas-kafka/consumergroup"` |
| No collision with another `consumergroup` symbol in any service's main.go | PASS | The only `consumergroup` references in `services/*/atlas.com/*/main.go` are this new import alias and the call sites; no shadow / collision risk |

---

## Phase 5: Per-Service Mechanical Checks (14 services routing through KeyPrefix)

The change pattern is uniform: hardcoded `"atlas:..."` literal Redis-key formats become `<KeyPrefix()>":..."` formats.

| Service | File | Status | Evidence |
|---------|------|--------|----------|
| atlas-account | `account/registry.go` | PASS | `services/atlas-account/atlas.com/account/account/registry.go:84, 248-249` |
| atlas-buffs | `character/registry.go` | PASS | `services/atlas-buffs/atlas.com/buffs/character/registry.go:46` |
| atlas-chairs | `character/registry.go` | PASS | `services/atlas-chairs/atlas.com/chairs/character/registry.go:34-36, 69` |
| atlas-chalkboards | `character/registry.go` | PASS | `services/atlas-chalkboards/atlas.com/chalkboards/character/registry.go:33-35` |
| atlas-character | `session/registry.go` | PASS | `services/atlas-character/atlas.com/character/session/registry.go:38` |
| atlas-expressions | `expression/registry.go` | PASS | `services/atlas-expressions/atlas.com/expressions/expression/registry.go:31` |
| atlas-maps | `map/monster/registry.go` | PASS | `services/atlas-maps/atlas.com/maps/map/monster/registry.go:61-67, 262` |
| atlas-monsters | `monster/registry.go`, `monster/cooldown.go`, `monster/attack_cooldown.go`, `monster/drop_timer_registry.go` | PASS (production); see BG-01 for test residue | `services/atlas-monsters/atlas.com/monsters/monster/registry.go:277, 282, 287, 707, 744-745` etc. |
| atlas-npc-shops | `shops/cache.go`, `shops/registry.go` | PASS | `services/atlas-npc-shops/atlas.com/npc/shops/cache.go:30`, `services/atlas-npc-shops/atlas.com/npc/shops/registry.go:34` |
| atlas-pets | `character/registry.go` | PASS | `services/atlas-pets/atlas.com/pets/character/registry.go:40, 69-70` |
| atlas-portals | `blocked/cache.go` | PASS | `services/atlas-portals/atlas.com/portals/blocked/cache.go:33` |
| atlas-rates | `character/item_tracker.go` | PASS | `services/atlas-rates/atlas.com/rates/character/item_tracker.go:216` |
| atlas-skills | `skill/cooldown_registry.go` | PASS | `services/atlas-skills/atlas.com/skills/skill/cooldown_registry.go:39, 62, 123-124` |
| atlas-storage | `storage/cache.go` | PASS | `services/atlas-storage/atlas.com/storage/storage/cache.go:37` |

### Final-grep validation

`grep -rn '"atlas:' --include='*.go' services/ libs/ | grep -v '_test.go' | grep -v 'docs/'` returns exactly one match: `libs/atlas-redis/keys.go:26` — a comment ("// composing keys outside the helper functions can avoid hardcoding "atlas:"."). That is documentation, not a key literal. Production code is clean. (See BG-01 for the two residual test-file matches.)

### Alias-collision check (reviewer concern #4)

Phase 5 uses two import aliases inconsistently — `atlas` in some files, `atlasredis` in others. Verified across all 14 Phase 5 production files:

- `atlas` alias: used in `services/atlas-account/.../registry.go`, `services/atlas-buffs/.../registry.go`, `services/atlas-chairs/.../registry.go`, `services/atlas-chalkboards/.../registry.go`, `services/atlas-character/.../session/registry.go`, `services/atlas-expressions/.../expression/registry.go`, `services/atlas-pets/.../character/registry.go`, `services/atlas-portals/.../blocked/cache.go`, `services/atlas-rates/.../character/item_tracker.go`, `services/atlas-skills/.../skill/cooldown_registry.go`, `services/atlas-npc-shops/.../shops/registry.go`.
- `atlasredis` alias: used in `services/atlas-storage/.../storage/cache.go`, `services/atlas-monsters/.../monster/*.go` (4 files), `services/atlas-maps/.../map/monster/registry.go`, `services/atlas-npc-shops/.../shops/cache.go`.

In every `atlas`-aliased file, the only co-imported atlas-prefixed lib is `github.com/Chronicle20/atlas/libs/atlas-tenant` (imported as bare `tenant`, not as `atlas`) — verified via `awk '/^import \(/,/^\)/' <file>` for all 11 atlas-aliased files. **Zero ambiguity.** The cosmetic inconsistency is real but does not produce any compile-time or readability hazard. Documenting only because the reviewer asked.

---

## Phase 6: Reviewer's Six Specific Concerns

### Concern 1: Init-order safety of `var keyPrefix = computeKeyPrefix(os.Getenv("ATLAS_ENV"))`

**Verdict: PASS.** Every call site of `KeyPrefix()` in production code is at function-call time, not at package-level init time. Verified by:

```
grep -rn 'KeyPrefix' --include='*.go' services/ libs/ | grep -v '_test.go' | grep -E '^[^:]+:[0-9]+:(var|const)'
```

returns zero matches. Equivalently: no production package has `var x = atlasredis.KeyPrefix() + "..."` at init time. All 28 hits across services are inside function bodies (`func (r *Registry) tenantSetKey() string { return ... atlas.KeyPrefix() ... }`). Init order is not a concern because the resolved value is captured per-call, after the importing package's init has run, after `libs/atlas-redis` init has run.

The single `init`-time read is in `libs/atlas-redis/keys.go:14` itself, which is a self-contained no-import-cycle expression that resolves before any consumer can call `KeyPrefix()`.

### Concern 2: Race safety on package-level mutable `keyPrefix`

**Verdict: PASS (production), CAVEAT (tests).**

- Production: writes occur exactly once at `libs/atlas-redis/keys.go:14` package init, before any goroutine reads. No mutex needed.
- Tests: `keys_test.go:31-33` and `keys_test.go:43-45` mutate `keyPrefix` directly under `t.Cleanup` restoration. The two mutating tests do not call `t.Parallel()`. `go test -race ./... -count=1` passes (verified). If anyone later adds a third test that calls `t.Parallel()` against `keyPrefix`, race detector will flag immediately. Acceptable as-is.

### Concern 3: Phase 4 import hygiene

**Verdict: PASS.** Already covered above.

### Concern 4: `atlas` vs `atlasredis` alias inconsistency

**Verdict: COSMETIC NIT.** No real collision; documented above.

### Concern 5: DOM-21 — atlas-constants reuse

**Verdict: PASS.** Phase 5 introduces no new domain types or numeric constants. Phase 1's `keyPrefixBase = "atlas"` (`libs/atlas-redis/keys.go:11`) is a string literal serving as a Redis key segment; `libs/atlas-constants/` contains domain numeric/type constants (item ids, world ids, channel ids, classifications), not Redis-key string segments. No overlap. No DOM-21 violation.

### Concern 6: Test coverage of env-aware behavior

**Verdict: PARTIAL — see BG-01 for the discovered regression.**

- `libs/atlas-redis` has 5 tests proving env-aware behavior end-to-end, including coverage of `namespacedKey` and `tenantEntityKey`. PASS.
- `libs/atlas-kafka/consumergroup` has 3 tests (default, set, whitespace-only). PASS.
- `libs/atlas-object-id` has a non-tautological env-prefix guard at `allocator_test.go:133-158`. PASS.
- Phase 4 service main.go changes don't add per-service tests. Acceptable — the resolver's tests cover the contract and the change is a literal-literal substitution at the call site.
- Phase 5 service changes don't add per-service key-shape regression tests. **Acceptable for the production code that was changed**, but caused the auditor to miss a residual test-file straggler in `services/atlas-monsters/atlas.com/monsters/monster/id_allocator_test.go` (see BG-01). The implementer was already in this directory for the Phase 5 production migration of `registry.go` / `cooldown.go` / `attack_cooldown.go` / `drop_timer_registry.go` and should have caught the test-file residue at the same time.

---

## Summary

### Blocking (must fix)

- **BG-01 (Test breaks under non-default `ATLAS_ENV`):** `services/atlas-monsters/atlas.com/monsters/monster/id_allocator_test.go:80` hardcodes the `"atlas:oid:" + ten.Id().String() + ":next"` Redis key for direct miniredis priming. Under any non-empty `ATLAS_ENV` the production path now writes to `<env>:atlas:oid:...:next` (per `libs/atlas-object-id/allocator.go:104`), making the test prime a key the allocator never reads. `TestAllocator_RecyclesLIFONearExhaustion` then fails because the counter is fresh-initialized at `MinId` instead of seeing the test-primed `RecycleThreshold` value. **Reproduced**: `ATLAS_ENV=ci42 go test ./monster/ -run TestAllocator_Recycles -count=1` returns `FAIL`. Fix: route the test's hardcoded literal through `atlasredis.KeyPrefix()` (matching the production rewire in the same package). This must ship before any CI environment ever sets `ATLAS_ENV` — currently the repo's CI doesn't, but the entire point of task-063 is to provision per-PR environments where `ATLAS_ENV` *is* set, and the allocator test would silently break in those environments without ever being noticed in main-CI.

### Non-Blocking (should fix)

- **BG-02 (Brittle test assertion):** `services/atlas-monsters/atlas.com/monsters/monster/attack_cooldown_test.go:49` asserts a key does not equal a hardcoded `"atlas:monster-cooldown:..."` literal. Under non-default `ATLAS_ENV` the assertion becomes vacuous (no key in miniredis matches the stale "atlas:" literal anyway). Test still passes by accident. Recommend routing the literal through `atlasredis.KeyPrefix()` so the assertion remains meaningful.

- **BG-03 (Cosmetic alias inconsistency):** Phase 5 uses both `atlas` and `atlasredis` as the import alias for `libs/atlas-redis` across the 14 modified files. No functional impact. If consistency is desired, normalize to `atlasredis` (the alias used by both library code and the new files in atlas-storage/atlas-maps/atlas-monsters/atlas-npc-shops).

- **BG-04 (Phase 5 should have grepped test files):** The audit-redis-prefix.txt list (referenced in the task folder) appears to have filtered out `_test.go` files. As Phase 5 is the precise moment to clean up *all* hardcoded `"atlas:"` literals, the implementer should have re-grepped including test files and either rewired them or filed a follow-up task. This is what Phase 5's "final-grep validation" step (per plan §2.5.15) was supposed to catch — and it missed two test-file matches because the existing audit grep was production-only.

### Out-of-Scope DOM Checks

DOM-01 through DOM-11, DOM-13 through DOM-20: not evaluated. The task does not modify the load-bearing DDD files (`model.go`, `entity.go`, `builder.go`, `processor.go`, `provider.go`, `administrator.go`, `producer.go`, `resource.go`, `rest.go`) in any package. Re-running these checks against unchanged structure is out of scope for a diff-targeted audit.
