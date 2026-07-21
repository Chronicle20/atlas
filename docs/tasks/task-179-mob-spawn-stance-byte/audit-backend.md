# Backend Audit — task-179-mob-spawn-stance-byte

- **Audit Scope:** `git diff main...HEAD -- '*.go'` (6 commits ahead of main, 0 behind). 14 files changed: 705 insertions / 135 deletions across
  - `libs/atlas-packet/character/data.go` + `data_test.go` (legacy CharacterData version-gate fix)
  - `libs/atlas-packet/inventory/clientbound/change_test.go`, `change_v48_test.go` (test assertions updated to match the above)
  - `libs/atlas-packet/model/asset.go` + new `asset_v72_test.go` (legacy equip-trailer version-gate fix)
  - `services/atlas-channel/atlas.com/channel/data/map/processor.go` (comments only on pre-existing `SnapMobPosition`)
  - `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go` (comment rewrite, no logic change)
  - `services/atlas-channel/atlas.com/channel/movement/processor.go` + new `fold_test.go` (pointer-match fold fix)
  - `services/atlas-channel/atlas.com/channel/socket/writer/monster_spawn.go` (debug log line removed)
  - `services/atlas-monsters/atlas.com/monsters/kafka/consumer/map/consumer.go` (call site swapped to `ControlOnEnter`)
  - `services/atlas-monsters/atlas.com/monsters/monster/processor.go` + new `control_on_enter_test.go` (new `ControlOnEnter` method)
- **Guidelines Source:** backend-dev-guidelines skill (ai-guidance.md, file-responsibilities.md, anti-patterns.md, testing-guide.md, DOM-*/SUB-*/SEC-* checklist)
- **Date:** 2026-07-21
- **Build:** PASS (all three affected modules)
- **Tests:** 3 modules, 0 failed
- **Overall:** PASS

## Build & Test Results

```
$ cd libs/atlas-packet && go build ./... && go vet ./...
(clean, no output)
$ go test ./... -count=1
All packages ok (character, character/clientbound, model, inventory/clientbound, etc.)

$ cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./...
(clean, no output)
$ go test ./... -count=1
89 "ok" package results, 0 FAIL (grep -c "^ok" = 89, grep FAIL = empty)

$ cd services/atlas-monsters/atlas.com/monsters && go build ./... && go vet ./...
(clean, no output)
$ go test ./... -count=1
ok  	atlas-monsters	2.043s
ok  	atlas-monsters/kafka/consumer/data	0.011s
ok  	atlas-monsters/kafka/consumer/monster	0.011s
ok  	atlas-monsters/map	0.028s
ok  	atlas-monsters/monster	12.531s
ok  	atlas-monsters/monster/drop	0.034s
ok  	atlas-monsters/monster/information	15.311s
ok  	atlas-monsters/world	0.039s
(0 FAIL)
```

Note on the two slow rows (`monster` 12.5s, `monster/information` 15.3s): traced to
`TestGetById_RedisDown_FallsThroughGracefully` in
`services/atlas-monsters/atlas.com/monsters/monster/information/cache_test.go:345`, a
pre-existing Redis-fallback-timeout test untouched by this diff (`git diff main...HEAD
--stat -- monster/` shows only `control_on_enter_test.go` and `processor.go` changed in
that package). Not a finding against this branch.

## Per-File / Per-Package Findings

### `libs/atlas-packet/character/data.go`, `libs/atlas-packet/model/asset.go`

Not a domain/sub-domain/support package under `services/*/atlas.com/*/internal` — this is
the shared packet-codec library (`Encode`/`Decode` pairs keyed off `tenant.Model`), which
has no `Processor`/`RestModel`/`resource.go` scaffold to begin with, so DOM-01..24 /
FILE-01..06 do not apply (no such files exist in this package family; e.g. `ls
libs/atlas-packet/character/` has no `resource.go`, `entity.go`, or `builder.go`).

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-25-analog | Client wire values config-resolved, not literals | PASS (not applicable to this diff) | The changed gates (`libs/atlas-packet/character/data.go:15,26,38` etc.) branch on `t.MajorAtLeast(N)` / `t.IsRegion("GMS")` — tenant *protocol-version* dispatch, the same idiom used pre-existing throughout this file (e.g. untouched `t.MajorAtLeast(84)` at `libs/atlas-packet/model/asset.go:593` survives unchanged). DOM-25 targets client-chosen dispatcher/opcode bytes resolved from a tenant writer-options table; this is wire-format version gating, a different and already-established mechanism. No new Go literal masquerading as a client wire code was introduced. |
| — | Build/vet/test | PASS | `go build ./...`, `go vet ./...`, `go test ./... -count=1` all clean (see above); new tests `TestCharacterDataLegacyFieldGate_V72` (`data_test.go:282`), `TestCharacterDataLegacyRoundTrip` (`data_test.go:347`), `TestCharacterDataLegacyStructure` (`data_test.go:409`), `TestEquipableLegacyTrailerTiers` / `TestCashEquipableLegacyTrailerTiers` (`asset_v72_test.go:19,52`) all pass and pin the fixed byte layout with IDA citations in-line. |
| — | Verification-over-memory (project rule) | PASS | Every new/changed gate cites an IDA address (e.g. `data.go:13 "@0x49d341"`, `data.go:37 "v72 @0x4cf30d"`, `asset.go:588 "@0x4d0172"`) rather than asserting from memory, consistent with CLAUDE.md's "Verification Over Memory" rule. |

No findings.

### `libs/atlas-packet/inventory/clientbound/change_test.go`, `change_v48_test.go`

Test-only changes: assertions were loosened from `bytes.Equal` to a length-delta check
now that the v48/v61/v72 equip trailer is shorter than v79 (consequence of the asset.go
fix). Re-verified the arithmetic against the new asset.go gates:

- v72: `filler`/exp gate now v72+ only, no hammersApplied until v79+ → v72 is exactly 4
  bytes shorter than v79 (`change_test.go:486-488`). Confirmed against
  `asset.go:596-601` (hammersApplied gated `MajorAtLeast(79)`).
- v61/v48: whole extended trailer now gated v72+ (`asset.go:589`), so v61/v48 lose
  levelType(1)+level(1)+exp(4)+2nd-buf(8)+int(4) = 18, plus hammersApplied(4) = 22 bytes
  vs v79 (`change_test.go:520`, `change_v48_test.go:555`). Matches `asset.go` gates
  exactly. PASS — arithmetic is internally consistent with the production code change.

### `services/atlas-channel/atlas.com/channel/movement/processor.go` (`foldMovementSummary`)

Package classification: support package (`ls services/atlas-channel/atlas.com/channel/movement/`
→ `action.go, processor.go, producer.go`, no `model.go`/`resource.go`).

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor logic in processor.go | PASS | `foldMovementSummary` lives in `movement/processor.go:280`, the only processor-named file in the package. |
| FILE-06 | No package-named catch-all file | PASS | No `movement.go` bundling responsibilities; `ls` above shows single-purpose files only. |
| — | Bug-fix correctness | PASS | Verified `model.MovementCodec` methods are declared on pointer receivers only — `grep -n "func (m \*NormalElement)" libs/atlas-packet/model/movement.go:118` — confirming the pre-fix `case model.TeleportElement` (value type) at old `processor.go:295` could never match a `*TeleportElement` produced by the decoder, making the pre-fix code a dead branch exactly as the new comment (`processor.go:942-947`) claims. The fix changes all four cases to pointer types (`processor.go:949,959,970,974`) and additionally applies `X`/`Y` for `*TeleportElement` (`processor.go:962-963`), which was previously entirely unset (a stance-only update) — a materially different, and correct per the doc comment, behavior change. |
| DOM-20 | Table-driven tests | WARN (non-blocking) | `movement/fold_test.go` (new) uses four separate `Test*` functions (`fold_test.go:9,26,37,47`) rather than a single `[]struct{...}` + `t.Run` table. `testing-guide.md:18` phrases this as "Prefer table-driven tests" (soft preference, not a MUST), and the four cases exercise materially different type-switch arms (Normal/Teleport/Teleport-zero-fh/Jump/StartFallDown) that don't collapse cleanly into one table shape — noted, not blocking. |

### `services/atlas-channel/atlas.com/channel/kafka/consumer/map/consumer.go`, `data/map/processor.go`, `socket/writer/monster_spawn.go`

Comment-only / debug-log-removal changes, no new symbols, no behavior change beyond the
comment content itself.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | No functional regression | PASS | `diff` for all three files (see full diff) shows only comment text and one deleted `l.Debugf(...)` call (`monster_spawn.go:41-46` in the old version); no code path altered. |
| DOM-12 | No `os.Getenv()` in handlers | N/A | Not resource.go / not touched. |

No findings.

### `services/atlas-monsters/atlas.com/monsters/monster/processor.go` (`ControlOnEnter`)

Package classification: domain package (`ls .../monster/` → has `model.go`). Diff only
touches `processor.go` (new method) and adds `control_on_enter_test.go`; the rest of the
package (builder.go, entity absence, registry.go, resource.go, rest.go) is pre-existing
and unmodified by this branch, so only the new method and its call site are graded.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | Processor logic in processor.go | PASS | `ControlOnEnter` defined at `monster/processor.go:1163`, added to the `Processor` interface at `monster/processor.go:47`. Both interface and implementation live in `processor.go`. |
| DOM-13 / DOM-15 (write-path analog) | No direct entity creation in handlers / cross-domain logic | PASS | `ControlOnEnter` is invoked from a Kafka consumer (`kafka/consumer/map/consumer.go:55`), not a REST handler; the consumer calls only `monster.NewProcessor(...).ControlOnEnter(...)` — no direct registry/DB access from the consumer file itself. |
| Registry-write pattern | Direct `GetMonsterRegistry().ControlMonster(...)` call from processor.go | PASS (established precedent, not a new violation) | This domain has no GORM entity/administrator.go — it is a Redis-backed in-memory registry domain where `registry.go` is the sole read/write gateway; `administrator.go`'s file-responsibilities.md definition is scoped to `*gorm.DB` mutations and does not apply here. Confirmed this exact call shape pre-exists in the untouched sibling method `StartControl` (`processor.go:376`: `GetMonsterRegistry().ControlMonster(p.t, uniqueId, controllerId)`) and in `Create` (`processor.go:229`). `ControlOnEnter` (`processor.go:1172`) mirrors both precisely — same registry call, same error-handling shape (`if _, err = ...; err != nil { p.l.WithError(err).Errorf(...) }`). |
| Error handling | `err` correctly threaded and returned | PASS | `processor.go:1170-1176`: `err` from `getControllerCandidate` is reused (`=`, not `:=`) inside `if _, err = GetMonsterRegistry().ControlMonster(...); err != nil` and returned; on success `err` is `nil` and returned as such. Verified by reading the full function body. |
| Test Helper Pattern (CLAUDE.md) | No new `*_testhelpers.go` file | PASS | `control_on_enter_test.go`'s `recordingProcessor` reuses pre-existing `newTestTenant` (`monster/cooldown_test.go:28`) and `testField` (`monster/model_test.go:15`); no new test-only constructor file was created. |
| DOM-24 | Kafka producer stubbed in tests that emit | PASS | `control_on_enter_test.go:16-22`'s `recordingProcessor` builds a bare `*ProcessorImpl{... emit: func(topic string, _ model.Provider[[]kafka.Message]) error {...}}`, bypassing the real `producer.ProviderImpl` entirely (no `BOOTSTRAP_SERVERS` write is ever attempted). This is not the canonical `producertest.InstallNoop()` / `WithProducer(...)` builder named in the DOM-24 checklist, but it is the package's own pre-existing, pervasive DI idiom — the same `emit func(topic string, provider model.Provider[[]kafka.Message]) error` field (`processor.go:67`) is already stubbed identically by ~15 pre-existing test call sites in `processor_test.go`, `aggro_task_test.go`, and `picker_test.go` (confirmed via `grep -n "emit:"`). No `t.Cleanup(producer.ResetInstance)` is used (there is nothing to reset — the struct is never routed through the singleton `producer.Manager` in the first place). Test run time for the two new tests is negligible (package total 12.5s attributed entirely to an unrelated pre-existing Redis test, see Build & Test Results). |
| DOM-20 | Table-driven tests | WARN (non-blocking) | `control_on_enter_test.go` has two separate `Test*` functions (`:22`, `:60`) covering the two branches (`cid == entering` vs `cid != entering`) rather than a `[]struct{}` + `t.Run` table. Same soft-preference caveat as the movement fold tests above — each branch asserts a different emission-count invariant and a different owner assignment, and collapsing two cases into a table buys little; not blocking. |
| Comment/doc accuracy | `ControlOnEnter` doc comment | PASS | `processor.go:1145-1162` doc comment's claims (in-place assignment for entering, StartControl+emit for already-present, "mirroring Create's in-place assignment") were independently verified against `Create` (`processor.go:191-233`, esp. the in-place block at `:211-231`) — the described precedent is real, not asserted from memory. |

### `services/atlas-monsters/atlas.com/monsters/kafka/consumer/map/consumer.go`

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | Consumer calls processor only, no direct registry/DB access | PASS | `consumer.go:55` calls `monster.NewProcessor(l, ctx)` then `.ControlOnEnter(...)`; no `GetMonsterRegistry()` or DB call appears in this file. |
| — | Type correctness | PASS | `e.Body.CharacterId` (used identically as `uint32` at `consumer.go:78` in the untouched `handleStatusEventCharacterExit`) matches `ControlOnEnter(enteringCharacterId uint32, ...)`; confirmed by clean `go build ./...`. |

## Sub-Domain Checklist Results

No sub-domain (action-event, no-`model.go`-with-`resource.go`) packages were touched by
this diff.

## Security Review

Not applicable — no auth/token/session-management code in this diff.

## Summary

### Blocking (must fix)

None.

### Non-Blocking (should fix)

- DOM-20: `services/atlas-channel/atlas.com/channel/movement/fold_test.go` uses four
  separate `Test*` functions instead of a table-driven `[]struct{}` + `t.Run` (soft
  "prefer" guideline, not a MUST).
- DOM-20: `services/atlas-monsters/atlas.com/monsters/monster/control_on_enter_test.go`
  uses two separate `Test*` functions instead of a table-driven form (same soft
  guideline).

### Verified Clean

- `go build ./...`, `go vet ./...`, `go test ./... -count=1` all pass with zero failures
  across `libs/atlas-packet`, `services/atlas-channel/atlas.com/channel`, and
  `services/atlas-monsters/atlas.com/monsters`.
- No bare `go` statements introduced in any changed file (`grep -nE '^\s*go (func|[A-Za-z_])'`
  against every changed file returned no matches).
- No new `os.Getenv`, manual JSON decoding, direct DB/registry writes from
  handlers/consumers, or hardcoded client wire-code literals were introduced.
- The `foldMovementSummary` pointer-match fix and the `ControlOnEnter` in-place-assignment
  fix are both independently verified correct against the codebase (interface receiver
  types; `Create`'s pre-existing precedent), not merely "looks plausible."
