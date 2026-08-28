# task-283 — Implementation Context

Companion to `plan.md`. Consumes `prd.md` (v1) and `design.md` (v1). Everything here was
established during the planning session against the actual tree at `22848da73`; the plan does not
require re-deriving any of it.

## Key files

| Path | Role |
|---|---|
| `services/atlas-character-factory/atlas.com/character-factory/job/model.go` | The surviving mapper's home today (`JobFromIndex`, 21 lines). Deleted in Task 6 and replaced by `job/carousel.go`. |
| `services/atlas-character-factory/atlas.com/character-factory/factory/processor.go` | `Create` (`:89`), sentinels (`:26-32`), `findCreationTemplate` (`:80`), saga builder (`:185`), `JobId` payload field (`:206`), `validJob` stub (`:649-651`). |
| `services/atlas-character-factory/atlas.com/character-factory/factory/resource.go` | `categorizeError` (`:130-157`) and the `errors.Is` sibling pattern (`:38-77`). |
| `services/atlas-character-factory/atlas.com/character-factory/factory/processor_test.go` | The two tautologies (`:400`, `:1069-1071`), `createMockContext` (`:949-959`), 10 `buildCharacterCreationSaga` call sites. |
| `libs/atlas-constants/job/constants.go` | `type Id uint16` (`:3`), `Jobs` map (`:9`), `BeginnerId`/`NoblesseId`/`LegendId`/`EvanId` (`:95`, `:140`, `:161`, `:166`). |
| `libs/atlas-constants/job/model.go` | `IsBeginner` (`:56-58`), the unreferenced `FromIndex` twin (`:106-122`). |
| `libs/atlas-tenant/tenant.go` | `IsRegion`/`MajorAtLeast`/`MajorAtMost`/`MajorInRange` (`:88-105`), all pointer-receiver. |
| `services/atlas-configurations/seed-data/templates/*.json` | 11 files; `characters.templates[]` rows carry `(jobIndex, subJobIndex, mapId, gender)` plus appearance/equipment option lists. |
| `services/atlas-ui/src/components/features/characters/templates/jobNames.ts` | `WORLD_NAMES`, `worldNameFromJobIndex`, `templateLabels`, `KNOWN_CLASSES`. |
| `services/atlas-ui/src/components/features/characters/templates/IdentitySection.tsx` | Class dropdown (`:24-78`), the only `KNOWN_CLASSES` consumer. |
| `docs/tasks/task-283-race-index-job-mapping/findings.md` | **New, Task 1.** The evidence of record; Tasks 4, 5, 7, 8 all read its `## Consequences for later tasks` section. |
| `docs/packets/race-carousels.json` | **New, Task 2.** Cross-language parity fixture; loaded only by tests, never at runtime. |

## Decisions carried in from `design.md`

- **D-1** mapper lives in `atlas-character-factory/job`; `libs/atlas-constants` keeps no version
  awareness and gains no `atlas-tenant` dependency.
- **D-2** a `map[Slot]job.Id`, not a switch. Absence *is* rejection; no default arm exists to
  coerce through.
- **D-3** `carouselFor` is an ordered `IsRegion`/`MajorAtLeast` chain; the **number of carousels is
  an output of Task 1's findings, not a plan input**. The three vars in Task 5's skeleton are a
  shape, not a prediction.
- **D-4** `findings.md` is a hard gate before any code.
- **D-7** race-index validation moves below the tenant fetch; name and gender validation still run
  first. Deliberate, and pinned by `TestCreate_ValidatorOrderIsPreserved`.
- **D-10** forward-only. No repair pass for characters already created wrong on a live v95 tenant.

## Corrections to the design, found during planning

1. **The IDA exports cannot answer FR-6.** `docs/packets/ida-exports/*.json` are packet-handler
   registries — `{binary, md5, generated_at, functions}` where `functions` maps a symbol name to
   `{address, direction, calls}`. `CLogin::Update` is **not present in any of the ten**, and no
   key in any file contains "race" (case-insensitive scan of all ten). The design's §5 step 1
   reads as though the exports are the source; they are not. Task 1 goes through the live IDBs
   via `mcp__ida-pro__*`, and the exports serve only to pin binary+md5 and give `CLogin::*` entry
   addresses.
2. **All ten IDBs are already open** and adopted in the IDA MCP server (`idb_list`, verified at
   plan time). Task 1 is executable; there is no external blocker. Session ids are ephemeral, so
   the plan resolves by `filename`.
3. **`tenant.Model`'s predicates are pointer-receiver**, not value-receiver as the design's
   snippets imply. `func FromIndex(t tenant.Model, ...)` still works — a parameter is addressable
   — but a `*tenant.Model` parameter would be wrong and `tenant.Model` cannot satisfy an interface
   by value.
4. **`buildCharacterCreationSaga` needs a signature change**, which the design does not mention.
   It takes `(transactionId, input, tmpl)` and computes `JobId` inline at `:206`; "reuse the
   resolved `jobId`" therefore means adding a `jobId job.Id` parameter and updating **11 call
   sites** (1 production, 10 in `processor_test.go`).
5. **`categorizeError` does not use `errors.Is`.** It substring-matches error messages
   (`resource.go:130-157`), and `ErrTemplateNotFound`'s message is not in its list — so a missing
   template currently returns **HTTP 500**. A new sentinel is unreachable through that function
   without converting it. Task 6 Step 6 converts it, and folds in the `ErrTemplateNotFound` → 400
   correction. That correction is flagged as separable in the task body if a reviewer objects.
6. **The v84/v87 `(3,0)` `mapId` differs** (`100030102` vs `100030100`) for the same slot. Task 1
   Step 7 checks this against the binaries rather than assuming one is a typo.
7. **`KNOWN_CLASSES` has no `(1,1)` entry** even though three templates seed that row, so Dual
   Blade already renders through the "unknown" label path in the editor today.
8. **`useTenant()`'s version fields are nested** — `activeTenant?.attributes.majorVersion` /
   `.region` (`services/atlas-ui/src/services/api/tenants.service.ts:14-24`), not top-level as
   the design's FR-22 note reads.
9. **No `services/atlas-ui/src` test reads a file from disk today** (repo-wide grep for
   `readFileSync`/`node:fs` under `src` is empty). The TS parity test introduces that pattern.

## Dependencies and ordering

Strict, per `design.md` §5. **Task 1 gates everything.** Tasks 4, 5, 7, and 8 all read
`findings.md`'s `## Consequences for later tasks` section, which is why Task 1 Step 9 makes that
section a required part of the artifact rather than a nicety.

```
1 findings.md  ──▶ 2 race-carousels.json ──▶ 5 mapper ──▶ 6 processor ──▶ 9 verify
       │                     │                  ▲            │
       │                     └──────────────────┼────────────┴──▶ 8 frontend
       ├──▶ 4 constants ─────────────────────────┘
       └──▶ 7 seed data (also needs 5's carousels for the correspondence test)
```

3 (freeze current behavior) is independent of 1 and can run at any point **before** 5 — but it
must run before 5, or the frozen literals get captured from the new mapper and the
backward-compatibility gate becomes circular.

## Deliberately large or unusual tasks

- **Task 1** is a research task with no code and no test cycle, which the plan format does not
  normally accommodate. It is one task because splitting the derivation per version would fan out
  eleven agents against a **single shared IDA MCP server** — the same serialization constraint
  that `dispatcher-family-implementer` carries. If it hits the 120-tool-call budget, it should
  hand back PARTIAL with the version keys already written into `findings.md`, and a continuation
  picks up the remaining keys; the artifact is append-structured to make that clean.
- **Task 6 touches 4 files across one service** and includes an 11-site mechanical signature
  sweep. Kept whole because the intermediate states do not compile: deleting `validJob`, threading
  `jobId`, and deleting `job/model.go` are one atomic change.
- **Task 7 spans two services** (`atlas-configurations` seed data + the `atlas-character-factory`
  correspondence test) and is the one `plan-lint` F4 warning the plan ships with. Kept whole
  deliberately: the test *is* the gate that decides which seed rows change, so splitting them
  produces a first task with no way to know what to edit and a second task whose test was written
  against edits it cannot see. Its file list names 11 seed templates, but 8 are marked read-only
  and are only *read* by the test — the expected edit surface is 2–3 files.
- **Task 5 also spans two modules** (`atlas-character-factory` and `libs/atlas-constants`), though
  both live under one service boundary so F4 does not fire. Same reasoning: the
  `libs/atlas-constants` half is a pure deletion of a function with zero callers, and separating
  it would produce a commit whose only content is removing dead code.

## Risks carried forward

| Risk | Where it is handled |
|---|---|
| Pre-Big-Bang regression (the PRD's highest-risk property) | Task 3's frozen literals, captured before the refactor |
| Task 1 confirms the v95 lead only because it was told the lead | Task 1 Step 3 requires an independent ordering before comparison; the lead is quoted as "the claim under test" |
| A version's ordering is underivable | Recorded `unverified`, falls to the default arm with a comment; Task 1 Step 2 says explicitly not to stall |
| A new beginner id omitted from `IsBeginner` — silent and downstream | Task 4 makes all four edit sites one commit, with `TestIsBeginner_CoversEveryBeginnerId` as the gate |
| Go/TS label drift after this task | `race-carousels.json` + parity tests on both sides (Tasks 5c, 8a) |
| Carousel/seed-row drift after this task | `TestCarouselMatchesSeedTemplates` (Task 7) |
| Seed-file edits normalizing CRLF→LF | Task 7 Step 6's `git diff --stat` check |
| A cross-service seam defect a green `verify.sh` cannot see | Task 9 Step 6 hands off to code review; the change crosses factory → configurations → ui |

## Out of scope (unchanged from `prd.md` §2 and `design.md` §8, §D-10)

No Resistance or Dual Blade gameplay. No `CreateCharacter` codec change (`docs/packets/audits/gms_v95/CreateCharacter.md`
is ✅ and the field decodes correctly — the bug is purely in interpretation). No new endpoint, no
schema change, no `jobId` field on `template.RestModel`. No data repair for already-created
characters.

**Flagged for the user, carried from design D-10:** if characters already exist on a live v95
tenant with a wrong `jobId` and start map, correcting them is a separate task with its own PRD.
The original `raceIndex` is not persisted, so any repair would be heuristic.
