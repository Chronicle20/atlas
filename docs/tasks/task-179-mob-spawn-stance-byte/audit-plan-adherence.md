# Plan-Adherence Audit — task-179-mob-spawn-stance-byte

**IMPORTANT:** The committed `plan.md` describes the ABANDONED "stance byte"
approach (fly-aware `IdleMoveAction` helper, `resolveSpawnStance` emit-boundary
guard, `libs/atlas-constants/monster/stance.go`). Live v79 tracing during
implementation proved that hypothesis wrong and produced a different,
root-cause fix. This audit verifies the ACTUAL shipped deliverable against the
root-cause correction documented at the top of `design.md`, not against
`plan.md`'s checkboxes.

**Branch:** `task-179-mob-spawn-stance-byte`
**Base:** `main` (b30c7bf2c)
**Commits:** 6 (`git log --oneline main..HEAD`)
**Audit date:** 2026-07-21

---

## Commit-by-commit verification

### 1. `1e7a1926f` — config root cause: movement `types` added to Monster/Pet/Summon handlers

**Claim:** every non-v92/v95 GMS+JMS template's Monster/Pet/Summon move handlers
now carry a `types` array identical to that template's own
`CharacterMoveHandle.types`; v92/v95 intentionally skipped because their
`CharacterMoveHandle.types` is also absent.

**Verified independently** with a script that recursively finds every
`{"handler": ...}` object in each of the 11 templates
(`services/atlas-configurations/seed-data/templates/template_*.json`),
extracts `options.types`, and deep-compares each Monster/Pet/Summon handler's
array against that template's `CharacterMoveHandle.types`:

| Template | CharacterMoveHandle.types len | Monster | Pet | Summon |
|---|---|---|---|---|
| gms_12 | 9 | MATCH (9) | n/a (no Pet handler in this template at all) | MATCH (9) |
| gms_48 | 23 | MATCH | MATCH | MATCH |
| gms_61 | 23 | MATCH | MATCH | MATCH |
| gms_72 | 23 | MATCH | MATCH | MATCH |
| gms_79 | 23 | MATCH | MATCH | MATCH |
| gms_83 | 23 | MATCH | MATCH | MATCH |
| gms_84 | 24 | MATCH | MATCH | MATCH |
| gms_87 | 25 | MATCH | MATCH | MATCH |
| gms_92 | NONE | MISSING (consistent — char handle also has no `types`) | (no Pet handler present) | MISSING (consistent) |
| gms_95 | NONE (no CharacterMoveHandle at all) | MISSING (consistent) | MISSING (consistent) | MISSING (consistent) |
| jms_185 | 33 | MATCH | MATCH | MATCH |

`template_gms_12_1.json` has no `PetMovementHandle`/`PetMoveHandle` entry at
all (confirmed via `grep -o '"handler": "[A-Za-z]*"'` — the template's handler
set is a minimal login-era subset with no pet feature), so "n/a" there is
correct, not a gap.

All 11 templates parse as valid JSON (`python3 json.load` on each — 0
errors).

**Status: DONE.** Matches the claim exactly, including the correctly-skipped
v92/v95 pair.

### 2. `2e3db1ee5` — fold pointer-type fix (`movement/processor.go`)

Read the full diff. `foldMovementSummary`'s `switch v := e.(type)` previously
matched `case model.JumpElement:` / `case model.TeleportElement:` /
`case model.StartFallDownElement:` by value, while the decoder
(`movement.go Movement.Decode`) always constructs `&NormalElement{}`,
`&TeleportElement{}`, etc. (pointers) — so those three cases were dead code; only
`*model.NormalElement` (already a pointer case) ever matched. The commit
converts all three to pointer-typed cases (`*model.TeleportElement`,
`*model.JumpElement`, `*model.StartFallDownElement`), makes Teleport apply
`X`/`Y`/`Fh` (not just stance), and documents Jump/StartFallDown as
stance-only (mid-air, no resting foothold). Also drops the stale
`Spawn monster wire:` debug log from `socket/writer/monster_spawn.go`, and adds
explanatory comments to `data/map/processor.go`'s pre-existing
`SnapMobPosition` (no logic change there).

`movement/fold_test.go` (new, 63 lines) pins the fix with 5 table tests:
Normal applies X/Y/Fh, Teleport applies X/Y/Fh (the dead-case regression
guard), Teleport with `Fh=0` preserves prior Fh, Jump updates stance only,
StartFallDown updates stance only.

**Status: DONE.** Test file present and exercises exactly the previously-dead
cases.

### 3. `6a6c717a1` — `ControlOnEnter` (atlas-monsters)

Read the full diff. New `Processor.ControlOnEnter(enteringCharacterId, idp)`
method in `services/atlas-monsters/atlas.com/monsters/monster/processor.go`
reuses the existing `getControllerCandidate` helper (confirmed shared, not
duplicated — `grep -n getControllerCandidate` shows it backing
`FindNextController` at line 306 and the new `ControlOnEnter` at line 341,
i.e. genuine reuse, not a copy-paste fork). When the chosen controller equals
the entering character, it assigns via `GetMonsterRegistry().ControlMonster`
directly (no `StartControl` emit); otherwise it falls through to the existing
`StartControl` path (which does emit). The atlas-monsters consumer
(`kafka/consumer/map/consumer.go`) call site was switched from
`FindNextController` to `ControlOnEnter(e.Body.CharacterId, ...)`. The
atlas-channel consumer (`kafka/consumer/map/consumer.go`,
`spawnMonsterForSession`) comment was rewritten to explain the new
Spawn-then-Control invariant this enables (no logic change on the channel
side — channel already sent Spawn then Control; the fix is that atlas-monsters
no longer races an early Control ahead of it).

`control_on_enter_test.go` (new, 108 lines) covers both branches: entering
player assigned in-place with 0 `StartControl` emissions, and an
already-present player still getting exactly 1 `StartControl` emission via a
recording `ProcessorImpl` emit hook.

**Status: DONE.**

### 4. Legacy CharacterData/equip version-gate (`libs/atlas-packet`)

Read the full diff (445 insertions / 112 deletions across 6 files). Every
gate that was previously a flat `MajorVersion() > 28` / `> 12` is now split
into the correct per-field revision boundary (v61 dbcharFlag width + pet-id
array width, v72 monster-book cover-vs-cards split + gachaExp + equip
trailer + trailing stat int, v79 SN-list-size + linked-name + inventory
FILETIME + equip hammersApplied), with `Encode`/`Decode` kept symmetric (every
new `if` in `Encode` has a mirrored `if` in `Decode`, verified by side-by-side
read of both functions). `encodeMonsterBook`/`decodeMonsterBook` were split
into `encodeMonsterBookCover`/`encodeMonsterBookCards` and
`decodeMonsterBookCover`/`decodeMonsterBookCards` to gate the two pieces
independently (cover v61+, cards v72+) — existing callers
(`TestEncodeMonsterBook_Empty`/`_Populated`) updated to call both in sequence,
byte-identical output confirmed by the test's own comment ("byte-identical to
the old stub").

New/updated tests: `TestCharacterDataLegacyFieldGate_V72` (byte-diff +
byte-value assertion pinning the v72/v79 boundary),
`TestCharacterDataLegacyRoundTrip` (parameterized over v48/61/72/79/83,
encode→decode symmetry incl. tier-gated fields), `TestCharacterDataLegacyStructure`
(dbcharFlag width + monotonically increasing lengths), `TestEquipableLegacyTrailerTiers`
+ `TestCashEquipableLegacyTrailerTiers` (new `asset_v72_test.go`, length-delta
assertions for the three equip tiers), and the two `inventory/clientbound`
byte tests changed from exact-equality-with-v79 to length-delta assertions
(correctly, since the equip blob itself now differs by version).

**Status: DONE.** This commit is a legitimate, separately-motivated fix
("prerequisite that lets v79 clients enter the channel at all") bundled onto
the branch — not orphaned stance-byte work.

### 5. atlas-ui baseline operator header

Read the full diff. `BaselineService.restore` now calls
`headers.set("X-Atlas-Operator", "1")` after building `tenantHeaders(tenant)`,
with a comment explaining atlas-data's restore handler 403s without it. Test
updated from asserting the header is `null` to asserting it is `"1"`.

**Status: DONE.**

### 6. `b6aaff1e3` — docs correction

`design.md` and `plan.md` both carry the root-cause-correction preamble;
`context.md`, `grounding.md`, `prd.md` committed alongside. Commit message
states stale audit records from the abandoned approach were removed pending a
fresh review — consistent with there being no leftover audit-of-old-approach
file in the tree.

**Status: DONE.**

---

## Orphaned abandoned-approach check

Searched the full working tree for stance-byte-approach artifacts that should
have been removed: `IdleMoveAction`, `resolveSpawnStance`, `stance guard`,
`floor-to-5`, `FRAG-TRACE`, `STANCE-TRACE`, `SNAP-TRACE`.

```
grep -rn "IdleMoveAction\|resolveSpawnStance\|stance guard\|floor-to-5\|FRAG-TRACE\|STANCE-TRACE\|SNAP-TRACE" --include="*.go" .
```

Zero matches in any `.go` file. `libs/atlas-constants/monster/stance.go` (the
design's originally-planned new file) does not exist in the diff or the tree —
confirmed abandoned cleanly, not left as dead code.

## Stub / TODO / regression check

`git diff main...HEAD --name-only | xargs grep -ln "TODO\|FIXME\|not implemented\|panic(\"unimplemented"` found two hits:

- `docs/tasks/task-179-mob-spawn-stance-byte/plan.md` — expected; the
  superseded plan document itself, not code.
- `services/atlas-channel/atlas.com/channel/movement/processor.go:98` —
  `// TODO look up pet.` Pre-existing, `git blame` attributes it to commit
  `8bc75851a2` (2025-04-21), unrelated to this branch. Not introduced by
  task-179.

No new `// TODO`, stub, or `501`-style placeholder was introduced by any of
the 6 commits.

## Config JSON validity

All 11 templates under `services/atlas-configurations/seed-data/templates/`
(the 9 touched by commit 1, plus `gms_92`/`gms_95` used as the correctly-skipped
control group) parse as valid JSON via `python3 json.load` — 0 parse errors.

---

## Build & Test Results

| Module | `go build ./...` | `go vet ./...` | `go test -race ./...` |
|---|---|---|---|
| `libs/atlas-packet` | PASS | PASS | PASS (all packages `ok`, 0 FAIL; `character`, `character/clientbound`, `model`, `inventory/clientbound` explicitly confirmed) |
| `services/atlas-channel/atlas.com/channel` | PASS | PASS | PASS (exit 0; 89 `ok` package results via `grep -c "^ok"`, 0 `FAIL`; `atlas-channel/movement` and `atlas-channel/socket/writer` explicitly confirmed) |
| `services/atlas-monsters/atlas.com/monsters` | PASS | PASS | PASS (exit 0; 8 `ok` package results, 0 `FAIL`; `atlas-monsters/monster` and `atlas-monsters/monster/information` explicitly confirmed) |

| atlas-ui | Result |
|---|---|
| `npm test -- --run` | PASS — 154 test files, 1121 tests, 0 failed |
| `npm run build` | PASS — clean build (one pre-existing >500 kB chunk-size warning on `ConversationEditorPanel-*.js`, unrelated to this diff) |

### Repo-level guards (mandatory per CLAUDE.md)

| Guard | Result |
|---|---|
| `tools/redis-key-guard.sh` | exit 0, clean |
| `tools/goroutine-guard.sh` | exit 0, clean |
| `tools/lint.sh --check` | exit 0, "lint.sh: OK" (with Node 22 sourced via nvm; a first run without Node in `PATH` failed only on `ui:node-missing`, a local-shell-setup issue, not a code issue). Pre-existing dangling golangci-lint warnings reference a stale `task-124-teleport-rocks` worktree path that no longer exists on disk — an unrelated environment artifact from a shared lint cache, not attributable to this branch's files. |
| `docker buildx bake` | Not run — CLAUDE.md mandates this only "for every service whose `go.mod` was touched"; `git diff main...HEAD --name-only \| grep -i 'go.mod\|go.sum'` returned no matches, so no service's dependency graph changed. |
| `tools/service-registration-guard.sh` | Not applicable — no `services.json`, `deploy/k8s`, `docker-bake.hcl`, `go.work`, or `tools/db-bootstrap.sh` changes on this branch. |

---

## Cross-check against other reviewer artifacts already in the worktree

Two other reviewer outputs were already present as untracked files
(`audit-backend.md`, `audit-frontend.md`), evidently from a parallel
guideline-review pass. Both independently report PASS with build/test numbers
matching this audit's own measurements (89 `ok` / 0 FAIL for atlas-channel;
1121 tests / 154 files passed for atlas-ui) — corroborating rather than
duplicating this plan-adherence check.

---

## Overall Assessment

- **Deliverable adherence:** FULL. All 6 commits match their stated intent
  with file:line-level evidence; no gaps, no partial implementations.
- **Stub/regression check:** CLEAN. No orphaned stance-byte code, no new
  TODOs/stubs, no leftover diagnostic tracing.
- **Build/Test:** PASS across all three affected Go modules and atlas-ui
  (build + full test suite).
- **Recommendation:** READY_TO_MERGE.

## Action Items

None. No gaps found.
