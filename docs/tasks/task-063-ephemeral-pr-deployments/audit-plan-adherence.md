# Plan Audit (Phases 1-5) — task-063-ephemeral-pr-deployments

**Plan Path:** `docs/tasks/task-063-ephemeral-pr-deployments/plan.md`
**Audit Date:** 2026-05-08
**Branch:** `task-063-ephemeral-pr-deployments`
**Base for review:** `10d61861f150fce94f01d7b5da32917c4c5b4c40` (plan amendment commit) → `1055cc725630cc86feeb0b6c3a2412bfc9377e00` (HEAD)
**Phases in scope:** 1, 2, 3, 4 (Tasks 4.1, 4.2, 4.3), 5 (Tasks 5.1–5.15)
**Phases out of scope:** 0 (already landed before BASE), 6–11 (not yet executed)

## Executive Summary

All Phase 1–5 plan tasks were faithfully implemented. Phases 1 and 2 land their library + tests with the implementation form documented in the plan amendment commit `10cc1d58f`. Phase 3 routes the two atlas-object-id key composers through `atlasredis.KeyPrefix()` and adds the env-prefix unit test. Phase 4 sweeps all 49 services into `consumergroup.Resolve()` (47 literal + 2 templated) across four batched commits matching the planned core/social/world/infra split. Phase 5 patches all 14 audited services; for atlas-monsters the implementer correctly expanded scope from the plan’s single line to all 12 hits enumerated in `audit-redis-prefix.txt` (which is the authoritative source per Phase 0). The Phase 5.15 final-grep gate is clean — only the doc-comment hit at `libs/atlas-redis/keys.go:26` remains, exactly as the plan permits. All 14 Phase 5 service modules and all three Phase 1–3 libraries build and test green at HEAD.

## Task Completion

### Phase 1 — `libs/atlas-redis` env-aware key prefix

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 1.1 | Failing tests for env-aware prefix | DONE | `libs/atlas-redis/keys_test.go:1-57` adds `TestComputeKeyPrefix_envUnset/envSet`, `TestKeyPrefix_returnsBaseWhenEnvUnset`, `TestNamespacedKey_useEnvAwarePrefix`, `TestTenantEntityKey_useEnvAwarePrefix` — all five tests exactly as the plan prescribed (commit `fc18f3fec`). |
| 1.2 | Implement env-aware prefix | DONE | `libs/atlas-redis/keys.go:11-29` introduces `keyPrefixBase`, `keySeparator`, `var keyPrefix = computeKeyPrefix(os.Getenv("ATLAS_ENV"))`, `computeKeyPrefix` and exported `KeyPrefix()`. Plan’s minimal-diff path was taken: original `fmt.Sprintf` `TenantKey` is preserved verbatim (line 31-36). Tests pass with `-race` (`go test -race ./libs/atlas-redis/...`). |

### Phase 2 — `libs/atlas-kafka/consumergroup`

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 2.1 | Failing tests | DONE | `libs/atlas-kafka/consumergroup/resolver_test.go:1-29` lands the three test functions. The third was renamed `TestResolve_envWhitespaceOnly_returnsVerbatim` (commit `10cc1d58f`) — name only, same assertions. |
| 2.2 | Implement resolver | DONE | `libs/atlas-kafka/consumergroup/resolver.go:11,19-24` exports `Resolve(defaultName)` using the `LookupEnv && v != ""` form. This matches the plan-amendment commit `10cc1d58f` body (which fixed the original §2.2 example so that `Setenv("","")` test in §2.1 actually passes). Build and tests green: `go test ./libs/atlas-kafka/consumergroup/...`. |

### Phase 3 — `libs/atlas-object-id` raw-key fix

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 3.1 | Patch allocator.go | DONE | `libs/atlas-object-id/allocator.go:104,108` both use `fmt.Sprintf("%s:oid:%s:next/free", atlasredis.KeyPrefix(), t.Id().String())`. Import alias `atlasredis "github.com/Chronicle20/atlas/libs/atlas-redis"` at line 21. Test `TestAllocator_keysRespectEnvPrefix` at `allocator_test.go:133-158` (with non-tautological `strings.Contains(gotNext, "atlas")` guard) — added in commit `9583ce6b7`. Build and test green. |

### Phase 4 — Service consumer-group sweep (49 services)

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 4.1 (core, 5) | atlas-account, atlas-character, atlas-character-factory, atlas-tenants, atlas-world | DONE | Commit `8f528db73` modifies 5 main.go files. Plan’s "core" set lists 7 but explicitly defers atlas-channel/atlas-login to Task 4.2; the remaining 5 ship here. |
| 4.1 (social, 15) | atlas-buddies, atlas-buffs, atlas-chairs, atlas-chalkboards, atlas-expressions, atlas-fame, atlas-families, atlas-guilds, atlas-invites, atlas-marriages, atlas-merchant, atlas-messengers, atlas-notes, atlas-parties, atlas-party-quests | DONE | Commit `1700096db` modifies all 15 social main.go files. |
| 4.1 (world, 24) | atlas-asset-expiration, atlas-cashshop, atlas-consumables, atlas-data, atlas-drops, atlas-effective-stats, atlas-inventory, atlas-keys, atlas-map-actions, atlas-maps, atlas-monster-death, atlas-monsters, atlas-npc-conversations, atlas-npc-shops, atlas-pets, atlas-portal-actions, atlas-portals, atlas-quest, atlas-rates, atlas-reactor-actions, atlas-reactors, atlas-skills, atlas-storage, atlas-transports | DONE | Commit `0a457b73d` modifies all 24 world main.go files. |
| 4.1 (infra, 3) | atlas-ban, atlas-messages, atlas-saga-orchestrator | DONE | Commit `5e2121c7b` modifies all 3 infra main.go files. |
| 4.2 (templated, 2) | atlas-channel, atlas-login | DONE | Commit `889db0b6e`. Verified `services/atlas-channel/atlas.com/channel/main.go:117,148` and `services/atlas-login/atlas.com/login/main.go:31,66` wrap the templated `fmt.Sprintf(...)` in `consumergroup.Resolve(...)`. |
| 4.3 | Workspace build verify (no commit) | DONE | All 49 service modules build green at HEAD: per-module `go build ./...` passes for every `services/atlas-*/atlas.com/*/`. Plan explicitly says "no commit; the proof is the green build." |

Sweep total verification:

- `grep -l "consumergroup.Resolve" services/atlas-*/atlas.com/*/main.go | wc -l` = **49**
- `grep -lr "InitConsumers" services/ --include='main.go' | xargs grep -L "consumergroup.Resolve"` = **(empty)** — no service has Kafka consumers without resolving its group from env.

Acknowledged deviation #2: the plan suggested explicit `consumergroup` import alias. Since the alias matches the package name, the alias is technically redundant but harmless. Both forms are present across services and both compile.

### Phase 5 — Service raw-Redis-key audit fixes

| # | Task | Status | Evidence |
|---|------|--------|----------|
| 5.1 atlas-buffs | tenant key uses KeyPrefix() | DONE | `services/atlas-buffs/atlas.com/buffs/character/registry.go:46` uses `atlas.KeyPrefix() + ":" + r.characters.Namespace() + ":_tenants"`. Commit `9986ffbc5`. |
| 5.2 atlas-npc-shops | cache + registry keys | DONE | `services/atlas-npc-shops/atlas.com/npc/shops/cache.go:30` and `shops/registry.go:34` both use `atlasredis.KeyPrefix()`/`atlas.KeyPrefix()`. Commit `5730e067b`. |
| 5.3 atlas-portals | blocked-cache key | DONE | `services/atlas-portals/atlas.com/portals/blocked/cache.go:33`. Commit `4286cf9a2`. |
| 5.4 atlas-pets | three registry keys | DONE | `services/atlas-pets/atlas.com/pets/character/registry.go:40,69,70` — all three literals routed. Commit `34ad687e0`. |
| 5.5 atlas-skills | four cooldown keys | DONE | `services/atlas-skills/atlas.com/skills/skill/cooldown_registry.go:39,62,123,124`. Commit `d3aec2ca5`. |
| 5.6 atlas-expressions | tenant key | DONE | `services/atlas-expressions/atlas.com/expressions/expression/registry.go:31` uses `atlas.KeyPrefix() + ":expression:_tenants"`. Commit `d2cc9f306`. |
| 5.7 atlas-maps | spawn cache keys | DONE | `services/atlas-maps/atlas.com/maps/map/monster/registry.go:62,262`. Plan called out lines 60 and 260; the format-string literal moved by 2 lines because the format-string call was reformatted across multiple lines. Same hits, addressed correctly. Commit `2851943b2`. |
| 5.8 atlas-chairs | character registry keys | DONE | `services/atlas-chairs/atlas.com/chairs/character/registry.go:35,69`. Commit `788f028e9`. |
| 5.9 atlas-storage | npc-context cache key | DONE | `services/atlas-storage/atlas.com/storage/storage/cache.go:37`. Commit `e3a6818f1`. |
| 5.10 atlas-character | session tenant key | DONE | `services/atlas-character/atlas.com/character/session/registry.go:38`. Commit `5618aa483`. |
| 5.11 atlas-chalkboards | registry key | DONE | `services/atlas-chalkboards/atlas.com/chalkboards/character/registry.go:34`. Commit `1ed478230`. |
| 5.12 atlas-monsters | cooldown key (plan) → 12 hits across 4 files (per audit-redis-prefix.txt) | DONE | All 12 hits patched in commit `2fbe18d38`: `monster/cooldown.go:34,43`, `monster/attack_cooldown.go:34,43`, `monster/drop_timer_registry.go:68,157`, `monster/registry.go:277,282,287,707,744,745`. Verified by `grep -rn '"atlas:\|KeyPrefix' services/atlas-monsters/...` — every hit goes through `atlasredis.KeyPrefix()`. Plan §5 header explicitly defers to `audit-redis-prefix.txt` as the locked list, so the expansion is correct. |
| 5.13 atlas-account | three registry keys | DONE | `services/atlas-account/atlas.com/account/account/registry.go:84,248,249`. Commit `2597efcb6`. |
| 5.14 atlas-rates | item_tracker scan key | DONE | `services/atlas-rates/atlas.com/rates/character/item_tracker.go:216`. Commit `1055cc725`. |
| 5.15 Final audit grep | clean | DONE | `grep -rn '"atlas:' services/ libs/ --include='*.go' | grep -v _test.go` returns exactly one line: `libs/atlas-redis/keys.go:26:// composing keys outside the helper functions can avoid hardcoding "atlas:".` This is the doc-comment line the plan explicitly permits. No production composition remains. No commit, per plan. |

**Completion Rate (Phases 1–5):** 5 + 1 + 1 + 5 + 15 = **27/27 task items DONE (100%)**
**Skipped without approval:** 0
**Partial implementations:** 0

## Skipped / Deferred Tasks

None. Every task in Phases 1–5 has direct file:line evidence of implementation.

## Build & Test Results

### Libraries

| Library | Build | Tests | Notes |
|---------|-------|-------|-------|
| libs/atlas-redis | PASS | PASS | `go test -count=1 ./...` and `go test -race ./...` both green. |
| libs/atlas-kafka | PASS | PASS | `consumergroup`, `consumer`, `producer`, `retry` all pass. |
| libs/atlas-object-id | PASS | PASS | Includes new `TestAllocator_keysRespectEnvPrefix` with non-tautological guard. |

### Phase 5 services

| Service | Build | Tests |
|---------|-------|-------|
| atlas-buffs | PASS | PASS |
| atlas-npc-shops | PASS | PASS |
| atlas-portals | PASS | PASS |
| atlas-pets | PASS | PASS |
| atlas-skills | PASS | PASS |
| atlas-expressions | PASS | PASS |
| atlas-maps | PASS | PASS |
| atlas-chairs | PASS | PASS |
| atlas-storage | PASS | PASS |
| atlas-character | PASS | PASS |
| atlas-chalkboards | PASS | PASS |
| atlas-monsters | PASS | PASS |
| atlas-account | PASS | PASS |
| atlas-rates | PASS | PASS |

### Phase 4 — workspace build

All 49 service modules under `services/atlas-*/atlas.com/*/` build green via per-module `go build ./...`. Note: `go build ./...` from the workspace root is not meaningful for Go workspaces; per-module is the correct verification surface (the plan’s Step 1 is informational — Steps 2–3 strip the artifacts).

## Verification of Acknowledged Deviations

| # | Deviation | Verified |
|---|-----------|----------|
| 1 | Phase 2 resolver form uses `LookupEnv && v != ""` (plan amendment `10cc1d58f`) | YES — `libs/atlas-kafka/consumergroup/resolver.go:20`. Form matches amended plan §2.2. |
| 2 | Phase 4 import alias is the package name (alias redundant, harmless) | YES — both styles present across services, all compile and pass tests. |
| 3 | Phase 5 imports use `atlas` alias (not `atlasredis`) for atlas-redis | YES — verified across atlas-buffs, atlas-pets, atlas-skills, atlas-expressions, atlas-chairs, atlas-character, atlas-chalkboards, atlas-account, atlas-rates, atlas-portals, atlas-npc-shops/registry. atlas-maps, atlas-storage, atlas-monsters use the `atlasredis` alias (function call differs, target identical). |
| 4 | Phase 5.12 atlas-monsters scope expanded to 12 hits per audit-redis-prefix.txt | YES — all 12 hits patched in commit `2fbe18d38`; final grep is clean. |
| 5 | Phase 4.3 produces no commit | YES — last commit before workspace verify was `889db0b6e` (4.2). No 4.3 commit, builds green. |
| 6 | Phase 5.15 produces no commit | YES — last commit was `1055cc725` (5.14). Final grep clean (only doc-comment line). |

## Overall Assessment

- **Plan Adherence:** FULL
- **Recommendation:** READY_TO_MERGE (for Phases 1–5; Phases 6–11 are out of scope and unimplemented as expected)

## Action Items

None. The plan was faithfully executed; the acknowledged deviations are documented and verified.
