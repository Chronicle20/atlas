# Version-Aware Race-Index → Job Mapping — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the character-creation race-ordinal → beginner-job mapping a function of the tenant's client version, derived from the client binaries, and turn the no-op `validJob` into a real rejection gate.

**Architecture:** A per-version `Carousel` (a `map[Slot]job.Id`) lives in `atlas-character-factory/job`, selected by an ordered `IsRegion`/`MajorAtLeast` predicate chain over `tenant.Model`. Absence from the map *is* rejection — there is no default branch and no `BeginnerId` fallback. The table's contents come from an IDA-derived evidence artifact written before any mapping code exists, and are projected into `docs/packets/race-carousels.json`, which both the Go and the TypeScript sides assert against so the admin UI cannot drift.

**Tech Stack:** Go 1.27.0 (multi-module, no `go.work` — each module built from its own root), `libs/atlas-tenant` version predicates, `libs/atlas-constants/job`; React 19 + Vitest 4 in `services/atlas-ui`; `mcp__ida-pro__*` MCP tools against 10 live IDBs.

**Spec:** `docs/tasks/task-283-race-index-job-mapping/design.md` (PRD: `docs/tasks/task-283-race-index-job-mapping/prd.md`)

## Global Constraints

- **No fallback coercion.** `FromIndex` returns `(job.Id, bool)`. `ok=false` MUST NOT be coerced to `BeginnerId`. There is no `default:` arm in the lookup. (FR-1, D-2)
- **No raw version comparison.** Every version predicate uses `tenant.Model`'s `IsRegion` / `MajorAtLeast` / `MajorAtMost` / `MajorInRange` (`libs/atlas-tenant/tenant.go:88-105`). A bare `> N` or `>= N` on `MajorVersion()` fails review. (FR-2, D-3)
- **Evidence before code.** No carousel entry may exist without a matching row in `docs/tasks/task-283-race-index-job-mapping/findings.md` carrying an IDA function name + address. A cell that cannot be derived is recorded `unverified`, never guessed. (FR-11, D-4)
- **Exactly one mapper survives.** After Task 6, `grep -rn "FromIndex" --include="*.go" .` shows the `atlas-character-factory/job` mapper and its callers only — no `libs/atlas-constants/job.FromIndex`, no `job2.JobFromIndex`. (FR-4, D-1)
- **Pre-Big-Bang behavior is frozen.** Every currently-seeded `(jobIndex, subJobIndex, gender)` row on `gms_12` … `gms_92` must resolve to the same `job.Id` after the change as before it, unless IDA *positively contradicts* the row — in which case `findings.md` records the contradiction and the change cites it. "The findings didn't mention it" is not a contradiction. (FR-21, NFR backward compatibility)
- **No new job id is invented.** A Resistance/Citizen or Dual Blade creation id is added only if IDA/WZ data yields the value. Never assumed from memory. (FR-14, FR-15)
- **No new REST endpoint, no schema change, no `jobId` field on `template.RestModel`.** (PRD §5, §6)
- **Multi-tenancy.** No package-level mutable version state. The carousel package vars are read-only after init. (NFR)
- Go modules in play and their build/test cwd:
  - `services/atlas-character-factory/atlas.com/character-factory` (module `atlas-character-factory`)
  - `libs/atlas-constants` (module `github.com/Chronicle20/atlas/libs/atlas-constants`)
  - `services/atlas-ui` (npm; `npm test` → `vitest run`)
  There is **no `go.work`**. Run `go build ./...` / `go test ./...` from each module root.

## Version key → IDB / export map (established, do not re-derive)

`tools/task-facts.sh` and `mcp__ida-pro__idb_list` were run during planning. All ten IDBs are **already open and adopted** in the IDA MCP server. Resolve a session from `idb_list` by `filename` (session ids are ephemeral — never hard-code them).

| version key | seed template | IDB filename | export JSON |
|---|---|---|---|
| `gms_12` | `template_gms_12_1.json` | *(none)* | *(none)* |
| `gms_v48` | `template_gms_48_1.json` | `GMS_v48_1_DEVM.exe.i64` | `gms_v48.json` |
| `gms_v61` | `template_gms_61_1.json` | `GMS_v61.1_U_DEVM.exe.i64` | `gms_v61.json` |
| `gms_v72` | `template_gms_72_1.json` | `GMS_v72.1_U_DEVM.exe.i64` | `gms_v72.json` |
| `gms_v79` | `template_gms_79_1.json` | `GMS_v79_1_DEVM.exe.i64` | `gms_v79.json` |
| `gms_v83` | `template_gms_83_1.json` | `MapleStory_dump.exe.i64` (v83_Me) | `gms_v83.json` |
| `gms_v84` | `template_gms_84_1.json` | `GMS_v84.1_U_DEVM.i64` | `gms_v84.json` |
| `gms_v87` | `template_gms_87_1.json` | `GMSv87_4GB.exe.i64` | `gms_v87.json` |
| `gms_v92` | `template_gms_92_1.json` | `GMS_v92_1_DEVM.exe.i64` | `gms_v92.json` |
| `gms_v95` | `template_gms_95_1.json` | `GMS_v95.0_U_DEVM.exe.i64` | `gms_v95.json` |
| `gms_jms_185` | `template_jms_185_1.json` | `MapleStory_dump_SCY.exe.i64` | `gms_jms_185.json` |

**Correction to the design's assumption, established during planning:** the checked-in
`docs/packets/ida-exports/*.json` files are **packet-handler registries**, not function
databases. Schema is `{binary, md5, generated_at, functions}` where `functions` maps a
symbol name → `{address, direction, calls}` (838 entries in `gms_v95.json`). **`CLogin::Update`
is not in any export, and no key in any of the ten files contains "race"** (verified by
case-insensitive scan of all ten). The exports are useful only to (a) confirm which binary +
md5 a version key corresponds to and (b) give starting addresses for `CLogin::*` handlers.
**The FR-6 derivation must go through the live IDBs via the `mcp__ida-pro__*` tools.**

## Current-state facts (established during planning — do not re-derive)

Current mapping, identical in both twins
(`services/atlas-character-factory/atlas.com/character-factory/job/model.go:5-21` and
`libs/atlas-constants/job/model.go:106-122`):

| raceIndex | subJobIndex | current job.Id |
|---|---|---|
| 0 | any | `NoblesseId` = 1000 |
| 1 | 0 | `BeginnerId` = 0 |
| 1 | 1 | `BeginnerId` = 0 *(empty `// jobId = job.BladeRecruit TODO` branch falls through)* |
| 2 | any | `LegendId` = 2000 |
| 3 | any | `EvanId` = 2001 |
| anything else | any | `BeginnerId` = 0 |

Seeded `(jobIndex, subJobIndex) → mapId` per template (identical for gender 0 and 1):

| template | rows |
|---|---|
| `gms_12` | (1,0)→10000 |
| `gms_48`, `gms_61`, `gms_72`, `gms_79`, `gms_83` | (0,0)→130010220, (1,0)→10000, (2,0)→140090000 |
| `gms_84` | (0,0)→130010220, (1,0)→10000, (2,0)→140090000, (3,0)→**100030102** |
| `gms_87` | (0,0)→130010220, (1,0)→10000, (2,0)→140090000, (3,0)→**100030100** |
| `gms_92`, `gms_95`, `jms_185` | (0,0)→130010220, (1,0)→10000, (1,1)→10000, (2,0)→140090000, (3,0)→100030100 |

Constants: `type Id uint16` (`libs/atlas-constants/job/constants.go:3`);
`BeginnerId = Id(0)` (`:95`), `NoblesseId = Id(1000)` (`:140`), `LegendId = Id(2000)` (`:161`),
`EvanId = Id(2001)` (`:166`). `Jobs` registry map starts at `constants.go:9`, entry shape
`BeginnerId: {id: BeginnerId},`. `IsBeginner` = `IsA(jobId, BeginnerId, NoblesseId, LegendId, EvanId)`
(`libs/atlas-constants/job/model.go:56-58`).

`tenant.Model`'s predicates are **pointer-receiver** (`func (m *Model) MajorAtLeast(v uint16) bool`,
`libs/atlas-tenant/tenant.go:93`). They are callable on an addressable local — which is what
`processor.go:104`'s `t := tenant.MustFromContext(ctx)` gives — but a `tenant.Model` **parameter**
is likewise addressable, so `func FromIndex(t tenant.Model, ...)` works. Do not take a pointer
parameter.

`buildCharacterCreationSaga(transactionId uuid.UUID, input RestModel, tmpl template.RestModel) saga.Saga`
(`factory/processor.go:185`) has **11 call sites**: `processor.go:167` and ten in
`factory/processor_test.go` (lines 230, 269, 342, 386, 466, 511, 993, 1486, 1527, 1603).

`categorizeError` (`factory/resource.go:130-157`) matches on **error-message substrings**, not
`errors.Is`. `ErrTemplateNotFound`'s message is *not* in its `validationErrors` list, so a missing
template currently surfaces as **HTTP 500**, not 4xx. `"must provide valid job index"` **is** in the
list. Sibling categorizers in the same file (`categorizePresetError`, `categorizeMapleLifeError`,
`resource.go:38-77`) use `errors.Is` switches — that is the pattern to follow.

---

## Task 1: Derive every version's race carousel from the client binaries

This is a **hard gate**. No code in Tasks 3–9 may be written until `findings.md` is committed.
It is a research task: its deliverable is an evidence document, not source.

### Files

- `docs/tasks/task-283-race-index-job-mapping/findings.md` — **new file**; the evidence of record
- `docs/packets/ida-exports/gms_v95.json` — read-only; `functions[<name>].address` gives entry points
- `docs/packets/ida-exports/gms_v48.json`, `gms_v61.json`, `gms_v72.json`, `gms_v79.json`, `gms_v83.json`, `gms_v84.json`, `gms_v87.json`, `gms_v92.json`, `gms_jms_185.json` — read-only, same use
- `docs/reverse-engineering.md` — read-only; **read this first**, it owns IDA session mechanics
- `services/atlas-configurations/seed-data/templates/` — read-only; the *hypothesis under test*, never a source

Patterns to copy: `docs/packets/audits/gms_v95/CreateCharacter.md` (evidence-doc tone: claim, function, address, read order).

No Go module is built by this task.

- [ ] **Step 1: Load the IDA tool schemas in one turn**

Per `docs/reverse-engineering.md` ("Load the tool schemas in one turn"), do this once, not per
unit of work:

```text
ToolSearch: select:mcp__ida-pro__idb_list,mcp__ida-pro__func_query,mcp__ida-pro__decompile,mcp__ida-pro__xrefs_to,mcp__ida-pro__insn_query,mcp__ida-pro__get_global_value,mcp__ida-pro__search_text
```

- [ ] **Step 2: Resolve all ten sessions by filename**

Call `mcp__ida-pro__idb_list` once. Build a filename → `session_id` map using the table in
"Version key → IDB / export map" above. Pass `session_id` as the `database` parameter on every
later call. **Do not hard-code a session id into `findings.md`** — they are ephemeral.

If a session for a version key is missing from `idb_list`, record that key as `unverified` with
reason `"IDB not available in this session"` and continue. Do not stall the task.

- [ ] **Step 3: Derive the v95 carousel independently, before looking at the lead**

The claim under test (FR-7), stated here so it can be *compared to*, not *copied from*:

> `CLogin::Update` at `0x5dee90` gives `0 = Resistance, 1 = Explorer (subJob 1 = Dual Blade),
> 2 = Cygnus, 3 = Aran, 4 = Evan`.

Reach your own ordering first. In `GMS_v95.0_U_DEVM.exe.i64`:

1. `func_query` with `name_regex` `CLogin::(SendNewCharPacket|OnCreateNewCharacterResult|Update|OnRaceSelect.*|ChangeStep.*)` and with `name_regex` `CRaceSelect|CUIRaceSelect|CLoginRaceSelect|.*Race.*`.
2. `decompile` the function that writes the creation request. `CLogin::SendNewCharPacket` is
   present in `gms_v95.json` and is the packet builder — find where `m_nCurSelectedRace` /
   `m_nCurSelectedSubJob` are read from and what sets them.
3. `xrefs_to` the member offset that `SendNewCharPacket` reads, to find every writer. The writers
   are the carousel slot handlers; their ordinal literals are the answer.
4. Bound the result — prefer `func_query` + targeted `xrefs_to` over a blanket `decompile`, and
   give `insn_query` a narrow address range (`docs/reverse-engineering.md`, "Bound the result").

Only after you have an ordering, compare it to the lead and record agreement or correction.

- [ ] **Step 4: Answer the Resistance selectability question (FR-12)**

Enum membership is not evidence. Find the race-**availability** flags the v95 login screen
consults to decide which slots it *draws* — the guard around the slot's creation/enable path in
the race-select UI. Record, with function + address:

- whether the Resistance/Citizen slot is drawn at all on v95.0
- if it is, what beginner job id the client expects for it (**read from data, never assumed**)

- [ ] **Step 5: Answer the Dual Blade question (FR-5 / PRD §9.3)**

`BladeRecruit` exists today only as `Identity = 430` (`libs/atlas-constants/job/identities_gen.go:41`),
not as a creation-time `job.Id`. Determine which of the three outcomes holds on each of `gms_v92`,
`gms_v95`, `gms_jms_185`, with cited evidence:

1. a distinct creation job id exists → record the id and where it was read from
2. the client expects Explorer-beginner with a sub-job marker → record `(1,1) → BeginnerId` as an
   **explicit, evidenced** entry (this is *not* the current silent fallback)
3. the slot is not offered → record `(1,1)` as absent, and note that `template_gms_92_1.json`,
   `template_gms_95_1.json`, and `template_jms_185_1.json` each carry a `(1,1)` row that Task 7
   must then remove or annotate

- [ ] **Step 6: Derive `gms_jms_185` independently (FR-9)**

Use `MapleStory_dump_SCY.exe.i64`. Its seed rows share `gms_95`'s shape; that is **suggestive and
is recorded in `notes` as such**, never used as a source. If the JMS ordering is derived and
matches v95, say so as a *result*.

- [ ] **Step 7: Derive the pre-Big-Bang columns (FR-8)**

For `gms_v48`, `gms_v61`, `gms_v72`, `gms_v79`, `gms_v83`, `gms_v84`, `gms_v87`, `gms_v92`, run
the same derivation. The claim under test is `0 = Cygnus, 1 = Explorer, 2 = Aran, 3 = Evan` —
which today is an **inference from seed data**, and deriving it *from* the seed rows would be
circular. Note the two facts the seed data already flags as version-sensitive and check them
against the binary:

- `(3,0)` first appears at `gms_84`, whose `mapId` is `100030102` while `gms_87`+ use `100030100`
- `(0,0)` is seeded on every version from `gms_48` up, including versions that predate Cygnus
  Knights — if the binary shows the slot is not drawn on an early version, that is a positive
  contradiction and must be recorded

- [ ] **Step 8: Record `gms_12` as unverified (FR-10)**

One row, `status: unverified`, reason `"no IDA export and no IDB"`, and the note that its lone
`(1,0)` slot is present in every candidate mapping and is therefore insensitive to the ambiguity.

- [ ] **Step 9: Write `findings.md`**

Exactly this structure. One `##` section per version key, each containing a table with **one row
per `(raceIndex, subJobIndex)`**, plus a `### Method` subsection naming the functions and
addresses walked.

```markdown
# Race-Index → Job Findings (task-283)

Derived from the client binaries listed below. Every row cites the function and address it was
read from. A row that could not be derived is `status: unverified` with a reason; no row is
populated from remembered MapleStory knowledge.

| version key | binary | md5 | IDB |
|---|---|---|---|
| gms_v95 | GMS_v95.0_U_DEVM.exe | 3c71fd8872d5efbe16183ae8c51f887d | GMS_v95.0_U_DEVM.exe.i64 |
| ... | | | |

## gms_v95

| raceIndex | subJobIndex | class | job id | IDA function | address | status | notes |
|---|---|---|---|---|---|---|---|
| 1 | 0 | Explorer | 0 (BeginnerId) | CLogin::SendNewCharPacket | 0x… | verified | |

### Method
Walked <function> at <address>; <what the ordinal literal was and where the writer lives>.

### Lead comparison (FR-7)
Independently derived ordering: <ordering>. The lead claimed <ordering>. <Confirmed | Corrected
because …>.

## gms_12

| raceIndex | subJobIndex | class | job id | IDA function | address | status | notes |
|---|---|---|---|---|---|---|---|
| 1 | 0 | Explorer | 0 (BeginnerId) | — | — | unverified | No IDA export and no IDB. Its lone (1,0) slot is present in every candidate mapping, so it is insensitive to the ambiguity. |

## Open questions resolved

- **Resistance selectable on v95.0?** <Yes/No>, from <function> at <address>. → applies <FR-13 | FR-14>.
- **Dual Blade creation job id?** <outcome 1|2|3>, from <function> at <address>.
- **Does jms_185 share the v95 carousel?** <derived answer, as a result — not an assumption>.

## Consequences for later tasks

- Carousels required (Task 5): <list of distinct carousels and which version keys map to each>
- New job constants required (Task 4): <none | id + where the value was read from>
- Seed rows to add (Task 7, FR-19): <list, with mapId and the WZ path it was read from>
- Seed rows to correct (Task 7, FR-20): <list, with the contradiction cited>
- Seed rows to remove/annotate (Task 7): <the (1,1) rows, if outcome 3>
```

- [ ] **Step 10: Self-check before committing**

Confirm each of these by inspection of the file you just wrote:

- every version key in the map table has a `##` section
- every row has a non-empty `IDA function` + `address`, **or** `status: unverified` with a reason
- no row's justification is "matches the seed data" or "matches gms_v95"
- the `## Open questions resolved` section answers all three questions
- the `## Consequences for later tasks` section is filled in — Tasks 4, 5, and 7 read it directly

- [ ] **Step 11: Commit**

```bash
git add docs/tasks/task-283-race-index-job-mapping/findings.md
git commit -m "docs(task-283): IDA-derived race-carousel findings for all version columns"
```

---

## Task 2: Project the findings into `docs/packets/race-carousels.json`

### Files

- `docs/packets/race-carousels.json` — **new file**; the cross-language parity fixture
- `docs/tasks/task-283-race-index-job-mapping/findings.md` — **new file**, created by Task 1; read-only here, the source
- `docs/packets/gates.yaml` — read-only; sibling registry, for header/comment tone only

No Go module is built by this task.

- [ ] **Step 1: Write the fixture**

One object per version key. `slots` is an array so JSON key ordering is not load-bearing;
`jobId` is the numeric `job.Id`. A version key recorded `unverified` in `findings.md` still
appears, with `"verified": false`, so the file is a complete inventory rather than a partial one.

```json
{
  "_comment": "Machine-readable projection of docs/tasks/task-283-race-index-job-mapping/findings.md. Neither the Go server nor the admin UI loads this at runtime; both assert their compiled tables against it in tests (task-283 design D-9). Adding a slot here without adding it to BOTH tables fails those tests, which is the point.",
  "versions": {
    "gms_12": {
      "region": "GMS",
      "majorVersion": 12,
      "verified": false,
      "slots": [
        { "raceIndex": 1, "subJobIndex": 0, "jobId": 0, "class": "Explorer" }
      ]
    },
    "gms_v95": {
      "region": "GMS",
      "majorVersion": 95,
      "verified": true,
      "slots": []
    }
  }
}
```

Populate every version key from the corresponding `findings.md` section. `class` is a
human-readable label and is what the TypeScript side renders; it must match the `class` column in
`findings.md` character-for-character.

- [ ] **Step 2: Verify the projection is faithful and well-formed**

```bash
python3 -c "import json;d=json.load(open('docs/packets/race-carousels.json'));print(sorted(d['versions']));print({k:len(v['slots']) for k,v in d['versions'].items()})"
```

Expected: all eleven version keys (`gms_12`, `gms_v48`, `gms_v61`, `gms_v72`, `gms_v79`,
`gms_v83`, `gms_v84`, `gms_v87`, `gms_v92`, `gms_v95`, `gms_jms_185`), and a slot count per key
equal to the row count of that key's table in `findings.md`. If a count differs, the projection
dropped or invented a row — fix the JSON, never the findings.

- [ ] **Step 3: Commit**

```bash
git add docs/packets/race-carousels.json
git commit -m "docs(task-283): machine-readable race-carousel parity fixture"
```

---

## Task 3: Freeze current behavior and kill the tautological assertions

Done **before** the mapper changes, so the frozen expectations describe today's behavior rather
than whatever the new code happens to produce (design §4, risk row 1).

### Files

- `services/atlas-character-factory/atlas.com/character-factory/factory/processor_test.go` — modify lines 400 and 1070; add one new test
- `services/atlas-character-factory/atlas.com/character-factory/job/model.go` — read-only this task; the behavior being frozen

Module root: `services/atlas-character-factory/atlas.com/character-factory`.

Patterns to copy: `factory/processor_test.go:358-436` (`TestBuildCharacterCreationSaga_AllFieldsPresent` — input/template construction and `result.Steps[0].Payload.(saga.CharacterCreatePayload)` assertion shape); `factory/processor_test.go:949-959` (`createMockContext`, the only tenant-context builder in the file).

- [ ] **Step 1: Replace the two tautologies with literals**

At `processor_test.go:400` and `processor_test.go:1069-1071` the current text is, identically:

```go
expectedJobId := job2.JobFromIndex(input.JobIndex, input.SubJobIndex)
if payload.JobId != expectedJobId {
	t.Errorf("JobId mismatch: expected %d, got %d", expectedJobId, payload.JobId)
}
```

This asserts the function under test against itself and cannot fail. Replace each occurrence
with a literal keyed to that test's own `input.JobIndex`/`input.SubJobIndex`. Read those two
values out of each test's `input` literal and pick the row from the "Current-state facts" table
above — for example, if the test's input is `JobIndex: 1, SubJobIndex: 0`:

```go
// task-283: literal expectation. (1,0) is the Explorer slot -> BeginnerId (0)
// on every verified carousel; see docs/tasks/task-283-race-index-job-mapping/findings.md.
const expectedJobId = job.BeginnerId
if payload.JobId != expectedJobId {
	t.Errorf("JobId mismatch: expected %d, got %d", expectedJobId, payload.JobId)
}
```

`job` (`github.com/Chronicle20/atlas/libs/atlas-constants/job`) is the constants package; `job2`
is the service-local `atlas-character-factory/job`. After this step **no test in the file calls
`job2.JobFromIndex`** — that is what lets Task 6 delete it.

- [ ] **Step 2: Write the frozen pre-Big-Bang regression test**

New test in the same file. It enumerates every currently-seeded `(jobIndex, subJobIndex)` on
every pre-Big-Bang template and pins the job id. These are **frozen literals captured from
today's behavior**; if `findings.md` positively contradicts a row, change the literal in the same
commit as the mapper change and cite the finding in the comment — do not let the test drift
silently.

`TestFromIndex_PreBigBangFrozen` — table-driven, tenant built with the `createMockContext`
pattern at `processor_test.go:949-959` (`tenantModel.Create(uuid.New(), region, major, minor)`).
Until Task 5 lands, assert against `job2.JobFromIndex(jobIndex, subJobIndex)`; Task 5 rewrites the
call to `job2.FromIndex(t, jobIndex, subJobIndex)` and adds an `ok` assertion.

| subtest | region | major | minor | raceIndex | subJobIndex | expect jobId | expect ok |
|---|---|---|---|---|---|---|---|
| `gms_12/1.0` | GMS | 12 | 1 | 1 | 0 | `job.BeginnerId` (0) | true |
| `gms_48/0.0` | GMS | 48 | 1 | 0 | 0 | `job.NoblesseId` (1000) | true |
| `gms_48/1.0` | GMS | 48 | 1 | 1 | 0 | `job.BeginnerId` (0) | true |
| `gms_48/2.0` | GMS | 48 | 1 | 2 | 0 | `job.LegendId` (2000) | true |
| `gms_61/0.0` | GMS | 61 | 1 | 0 | 0 | `job.NoblesseId` (1000) | true |
| `gms_61/1.0` | GMS | 61 | 1 | 1 | 0 | `job.BeginnerId` (0) | true |
| `gms_61/2.0` | GMS | 61 | 1 | 2 | 0 | `job.LegendId` (2000) | true |
| `gms_72/0.0` | GMS | 72 | 1 | 0 | 0 | `job.NoblesseId` (1000) | true |
| `gms_72/1.0` | GMS | 72 | 1 | 1 | 0 | `job.BeginnerId` (0) | true |
| `gms_72/2.0` | GMS | 72 | 1 | 2 | 0 | `job.LegendId` (2000) | true |
| `gms_79/0.0` | GMS | 79 | 1 | 0 | 0 | `job.NoblesseId` (1000) | true |
| `gms_79/1.0` | GMS | 79 | 1 | 1 | 0 | `job.BeginnerId` (0) | true |
| `gms_79/2.0` | GMS | 79 | 1 | 2 | 0 | `job.LegendId` (2000) | true |
| `gms_83/0.0` | GMS | 83 | 1 | 0 | 0 | `job.NoblesseId` (1000) | true |
| `gms_83/1.0` | GMS | 83 | 1 | 1 | 0 | `job.BeginnerId` (0) | true |
| `gms_83/2.0` | GMS | 83 | 1 | 2 | 0 | `job.LegendId` (2000) | true |
| `gms_84/0.0` | GMS | 84 | 1 | 0 | 0 | `job.NoblesseId` (1000) | true |
| `gms_84/1.0` | GMS | 84 | 1 | 1 | 0 | `job.BeginnerId` (0) | true |
| `gms_84/2.0` | GMS | 84 | 1 | 2 | 0 | `job.LegendId` (2000) | true |
| `gms_84/3.0` | GMS | 84 | 1 | 3 | 0 | `job.EvanId` (2001) | true |
| `gms_87/0.0` | GMS | 87 | 1 | 0 | 0 | `job.NoblesseId` (1000) | true |
| `gms_87/1.0` | GMS | 87 | 1 | 1 | 0 | `job.BeginnerId` (0) | true |
| `gms_87/2.0` | GMS | 87 | 1 | 2 | 0 | `job.LegendId` (2000) | true |
| `gms_87/3.0` | GMS | 87 | 1 | 3 | 0 | `job.EvanId` (2001) | true |
| `gms_92/0.0` | GMS | 92 | 1 | 0 | 0 | `job.NoblesseId` (1000) | true |
| `gms_92/1.0` | GMS | 92 | 1 | 1 | 0 | `job.BeginnerId` (0) | true |
| `gms_92/2.0` | GMS | 92 | 1 | 2 | 0 | `job.LegendId` (2000) | true |
| `gms_92/3.0` | GMS | 92 | 1 | 3 | 0 | `job.EvanId` (2001) | true |

`gms_92/1.1` is deliberately **absent** from this table: `(1,1)` currently resolves to
`BeginnerId` only by falling through the empty `// jobId = job.BladeRecruit TODO` branch, which
FR-5 declares an unacceptable outcome. Its expectation is set by `findings.md` in Task 5, not
frozen here.

Head the test with this comment verbatim:

```go
// TestFromIndex_PreBigBangFrozen pins today's mapping for every currently-seeded
// (jobIndex, subJobIndex) row on the pre-Big-Bang templates. These literals were captured
// from the behavior of JobFromIndex BEFORE task-283's refactor; they are the backward-
// compatibility gate (PRD §8) and must not be regenerated from the new mapper. A literal
// changes only when docs/tasks/task-283-race-index-job-mapping/findings.md positively
// contradicts the row, and the change cites the finding.
```

- [ ] **Step 3: Run the tests**

```bash
go test ./factory/... -run 'TestFromIndex_PreBigBangFrozen|TestBuildCharacterCreationSaga_AllFieldsPresent|TestCharacterCreationOrchestrationFlow' -v
```

from `services/atlas-character-factory/atlas.com/character-factory`.
Expected: PASS. The new test passes against the *current* mapper — that is the point of
capturing it now.

- [ ] **Step 4: Commit**

```bash
git add services/atlas-character-factory/atlas.com/character-factory/factory/processor_test.go
git commit -m "test(task-283): freeze pre-Big-Bang mapping and remove tautological job assertions"
```

---

## Task 4: Job constants — new beginner ids and the beginner set

**Conditional on Task 1.** Read `findings.md` → `## Consequences for later tasks` → "New job
constants required". If it says `none`, skip to Step 4 and record the skip in the task report; do
not invent a constant to have something to do.

### Files

- `libs/atlas-constants/job/constants.go` — add the id constant and its `Jobs` registry entry
- `libs/atlas-constants/job/model.go:56-58` — extend `IsBeginner`
- `libs/atlas-constants/job/advancement_test.go` — extend the beginner rows in the case table at lines 14-18
- `libs/atlas-constants/job/identities_gen.go` — read-only; `BladeRecruit` is `Identity = 430` here, **not** a `job.Id`; do not repurpose it
- `docs/tasks/task-283-race-index-job-mapping/findings.md` — **new file**, created by Task 1; read-only here, the source of the id value

Module root: `libs/atlas-constants`.

- [ ] **Step 1: Write the failing test**

Add rows to the existing case table at `advancement_test.go:14-18`, which currently reads:

```go
{"Beginner", job.BeginnerId, 0},
{"Noblesse", job.NoblesseId, 0},
{"Legend (Aran beginner)", job.LegendId, 0},
{"Evan beginner (2001)", job.EvanId, 0},
```

For each new beginner id `findings.md` requires, add one row with `want: 0`. Then add a new test
in the same file asserting membership in the beginner set — this is the check D-6 exists for,
because `IsBeginner` is a hand-maintained allow-list and omission is silent:

`TestIsBeginner_CoversEveryBeginnerId` — table-driven, no fixtures needed.

| case | jobId | expect |
|---|---|---|
| Beginner | `job.BeginnerId` | true |
| Noblesse | `job.NoblesseId` | true |
| Legend | `job.LegendId` | true |
| Evan | `job.EvanId` | true |
| *(new id per findings.md, one row each)* | `job.<NewId>` | true |
| Warrior (not a beginner) | `job.Id(100)` | false |
| Fighter (not a beginner) | `job.Id(110)` | false |

Also assert the registry entry exists, since `Jobs[id]` misses are what silently break
`IsFourthJob` and `FromSkillId`:

| case | expr | expect |
|---|---|---|
| *(new id per findings.md)* | `_, present := job.Jobs[job.<NewId>]` | `present == true` |

- [ ] **Step 2: Run it and confirm it fails**

```bash
go test ./job/... -run 'TestIsBeginner_CoversEveryBeginnerId|TestAdvancement' -v
```

from `libs/atlas-constants`.
Expected: FAIL — `undefined: job.<NewId>` (compile error) before the constant exists.

- [ ] **Step 3: Add the constant, the registry entry, and the beginner-set membership**

All four sites, in one commit (D-6 — omitting any one produces a silent downstream failure):

1. `constants.go` const block — `<NewId> = Id(<value read from game data>)`, placed in numeric
   order among the existing ids. The comment cites where the value was read from.
2. `constants.go` `Jobs` map (starts line 9) — `<NewId>: {id: <NewId>},`
3. `model.go:57` — `return IsA(jobId, BeginnerId, NoblesseId, LegendId, EvanId, <NewId>)`
4. `advancement_test.go` — the rows added in Step 1

- [ ] **Step 4: Run the module's tests**

```bash
go build ./... && go test ./... 
```

from `libs/atlas-constants`.
Expected: PASS. If Task 1 concluded no new constant is needed, this still runs and passes
unchanged — record that in the task report and make no commit for Steps 1–3.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-constants/job/constants.go libs/atlas-constants/job/model.go libs/atlas-constants/job/advancement_test.go
git commit -m "feat(atlas-constants): add <name> beginner job id and beginner-set membership"
```

---

## Task 5: The version-aware carousel mapper

### Files

- `services/atlas-character-factory/atlas.com/character-factory/job/carousel.go` — **new file**; `Slot`, `Carousel`, the per-version tables, `carouselFor`, `FromIndex`
- `services/atlas-character-factory/atlas.com/character-factory/job/carousel_test.go` — **new file**; table tests + the parity test
- `libs/atlas-constants/job/model.go:106-122` — **delete** `FromIndex` (zero callers repo-wide; verified during planning)
- `docs/tasks/task-283-race-index-job-mapping/findings.md` — **new file**, created by Task 1; read-only here, the table contents
- `docs/packets/race-carousels.json` — **new file**, created by Task 2; read-only here, the parity fixture
- `libs/atlas-tenant/tenant.go` — read-only; the predicate idiom at lines 88-105

Module roots: `services/atlas-character-factory/atlas.com/character-factory` **and**
`libs/atlas-constants` (the deletion). Build and test both.

Patterns to copy: `services/atlas-channel/atlas.com/channel/battleship/processor.go:59-101`
(`func ShipHP(t tenant.Model, ...)` — a pure, version-gated function taking `tenant.Model` by
value and switching on `IsRegion`/`MajorAtLeast`); `tools/packet-audit/internal/template/real_test.go:9-20`
(locating a repo file from a test via `runtime.Caller(0)` + `filepath.Join` — the only such
pattern in the repo, and what the parity and correspondence tests need).

`job2.JobFromIndex` in `job/model.go` stays in place through this task so `factory/processor.go`
keeps compiling; Task 6 deletes it.

- [ ] **Step 1: Write the failing tests**

`carousel_test.go`, package `job`.

**(a) `TestFromIndex_PerVersionCarousel`** — table-driven. **One case per row of every `##`
version section in `findings.md`.** Build the tenant with
`tenant.Create(uuid.New(), region, major, minor)` (returns `(Model, error)`; `t.Fatalf` on error).
Case fields: `name` (`"<version key>/<raceIndex>.<subJobIndex>"`), `region`, `major`, `minor`,
`raceIndex`, `subJobIndex`, `wantJobId`, `wantOk`.

Expected values are **literals transcribed from `findings.md`**. Never call `carouselFor`,
`FromIndex`, or read `race-carousels.json` to compute an expectation in this test — that would
reintroduce the tautology Task 3 just removed. The `wantJobId`/`wantOk` for the pre-Big-Bang rows
must equal the frozen literals in Task 3's table; if `findings.md` contradicts one, that row's
literal changes in *both* tests in this same commit, with the finding cited.

**(b) `TestFromIndex_RejectsOffCarouselSlots`** — table-driven, every case `wantOk == false`
and `wantJobId == job.Id(0)` is **not** asserted (an `ok=false` result's job id is meaningless;
assert only `ok`). Cases, for a `GMS/95/0` tenant and a `GMS/83/1` tenant each:

| case | raceIndex | subJobIndex | expect ok |
|---|---|---|---|
| ordinal beyond any carousel | 9 | 0 | false |
| ordinal at uint32 max | 4294967295 | 0 | false |
| valid ordinal, bogus subjob | 1 | 7 | false |
| valid ordinal, subjob at uint32 max | 1 | 4294967295 | false |
| ordinal 4 on a v83 tenant *(pre-Evan-carousel; only if findings show v83 has no slot 4)* | 4 | 0 | false |

Plus, keyed to `findings.md`: for **every** version key, one case per `raceIndex` in
`0..(max verified raceIndex + 2)` that has **no** row in that version's findings table, asserting
`ok == false`. This is the structural guarantee that FR-1 holds — a fallback would light up
every one of these.

**(c) `TestCarouselsMatchParityFixture`** — reads `docs/packets/race-carousels.json` from disk and
asserts, per version key, that `carouselFor` for that key's `(region, majorVersion)` returns a map
whose key set and values equal the fixture's `slots` **exactly** — both directions, so a slot
present in one and not the other fails. Locate the repo root with the
`runtime.Caller(0)` + `filepath.Join` pattern from
`tools/packet-audit/internal/template/real_test.go:9-20`; from
`services/atlas-character-factory/atlas.com/character-factory/job/` the repo root is
`filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "..", "..")`. Skip version keys whose
fixture entry has `"verified": false` only if `findings.md` records no rows for them; `gms_12`
has a row and must still match.

**(d) `TestFromIndex_IsPerTenant`** — the NFR multi-tenancy proof (design §4.6). Within one test
body, build two tenants on different versions whose carousels differ for the same ordinal
(pick the pair from `findings.md` — `GMS/95/0` vs `GMS/83/1` if their orderings diverge), call
`FromIndex` for each, and assert the two results differ. Then call each again in the opposite
order and assert the same results, proving no order-dependent package state.

- [ ] **Step 2: Run them and confirm they fail**

```bash
go test ./job/... -v
```

from `services/atlas-character-factory/atlas.com/character-factory`.
Expected: FAIL — `undefined: FromIndex`, `undefined: Slot`, `undefined: carouselFor`.

- [ ] **Step 3: Write `carousel.go`**

```go
package job

import (
	"github.com/Chronicle20/atlas/libs/atlas-constants/job"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

// Slot is a (raceIndex, subJobIndex) pair exactly as the client sends it in the
// character-creation request. raceIndex is an ordinal into the race carousel the login
// screen drew -- it is NOT a job id, and its meaning changes between client versions.
type Slot struct {
	RaceIndex   uint32
	SubJobIndex uint32
}

// Carousel is one client version's race-selection screen: the exact set of slots that
// client can send, and the beginner job each one creates. Absence from the map means the
// client could not have sent that slot.
//
// Every entry traces to a row in docs/tasks/task-283-race-index-job-mapping/findings.md
// with a cited IDA function and address. Do not add an entry without one.
type Carousel map[Slot]job.Id

// One table per distinct carousel the findings established. These are read-only after
// init; nothing mutates them, which is what makes FromIndex safe for concurrent
// multi-tenant use.
var (
	// preBigBangCarousel: contents per findings.md, FR-8.
	preBigBangCarousel = Carousel{}

	// gmsV95Carousel: contents per findings.md, FR-7.
	gmsV95Carousel = Carousel{}

	// jmsV185Carousel: contents per findings.md, FR-9. Derived independently from
	// MapleStory_dump_SCY.exe.i64 -- NOT copied from gmsV95Carousel, even if it matches.
	jmsV185Carousel = Carousel{}
)

// carouselFor selects the carousel for the tenant's client version. The chain is ordered
// most-specific-first and uses only the tenant.Model version predicates (task-283 FR-2);
// a raw `> N` comparison here is a review failure.
func carouselFor(t tenant.Model) Carousel {
	switch {
	case t.IsRegion("JMS") && t.MajorAtLeast(185):
		return jmsV185Carousel
	case t.IsRegion("GMS") && t.MajorAtLeast(95):
		return gmsV95Carousel
	default:
		// The verified pre-Big-Bang carousel, not an empty map. gms_12 has no IDA export
		// and cannot be verified (findings.md, FR-10), but its lone seeded slot is
		// (1,0) -> Explorer, which is present in every candidate mapping; an empty
		// default would break it.
		return preBigBangCarousel
	}
}

// FromIndex maps a client-sent race ordinal to the beginner job it creates for this
// tenant's client version.
//
// ok=false means the tenant's client could not have sent this slot. The caller MUST
// reject; there is deliberately no default branch and no fallback to job.BeginnerId,
// because coercing an unknown ordinal is the bug task-283 exists to fix (FR-1).
func FromIndex(t tenant.Model, raceIndex uint32, subJobIndex uint32) (job.Id, bool) {
	id, ok := carouselFor(t)[Slot{RaceIndex: raceIndex, SubJobIndex: subJobIndex}]
	return id, ok
}
```

Then populate the three (or however many `findings.md` §Consequences names) carousel literals,
one entry per findings row:

```go
	preBigBangCarousel = Carousel{
		{RaceIndex: 0, SubJobIndex: 0}: job.NoblesseId, // findings.md gms_v83, <fn> @ <addr>
		{RaceIndex: 1, SubJobIndex: 0}: job.BeginnerId, // findings.md gms_v83, <fn> @ <addr>
		{RaceIndex: 2, SubJobIndex: 0}: job.LegendId,   // findings.md gms_v83, <fn> @ <addr>
		{RaceIndex: 3, SubJobIndex: 0}: job.EvanId,     // findings.md gms_v84, <fn> @ <addr>
	}
```

Add or remove `carouselFor` arms to match the carousel count `findings.md` established — the
arm list above is the shape, not a prediction of the count. If the findings show a version
column needs its own carousel (e.g. `gms_v84` diverging from `gms_v87`), add a named var and an
arm using `MajorInRange`.

- [ ] **Step 4: Delete the unreferenced twin in `atlas-constants`**

Delete `func FromIndex` at `libs/atlas-constants/job/model.go:106-122` entirely. It has **zero
callers repo-wide** (verified during planning), so this is a pure deletion. Check whether the
`skill` import at the top of that file is still used after the deletion; if `go build` complains,
remove the now-unused import.

- [ ] **Step 5: Run both modules' tests**

```bash
go test ./job/... -v
```
from `services/atlas-character-factory/atlas.com/character-factory` — Expected: PASS, all four tests.

```bash
go build ./... && go test ./...
```
from `libs/atlas-constants` — Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-character-factory/atlas.com/character-factory/job/carousel.go \
        services/atlas-character-factory/atlas.com/character-factory/job/carousel_test.go \
        libs/atlas-constants/job/model.go
git commit -m "feat(character-factory): version-aware race carousel mapper; drop unreferenced constants twin"
```

---

## Task 6: Wire the mapper into creation and make rejection real

### Files

- `services/atlas-character-factory/atlas.com/character-factory/factory/processor.go` — sentinel at :26-32, `Create` at :89-114, saga builder at :185/:206, delete `validJob` at :649-651
- `services/atlas-character-factory/atlas.com/character-factory/factory/resource.go:130-157` — `categorizeError` must map the new sentinel to 400
- `services/atlas-character-factory/atlas.com/character-factory/factory/processor_test.go` — the 10 `buildCharacterCreationSaga` call sites (230, 269, 342, 386, 466, 511, 993, 1486, 1527, 1603) + new rejection tests
- `services/atlas-character-factory/atlas.com/character-factory/job/model.go` — **delete the file**; `JobFromIndex` is its only declaration and Task 5 replaced it

Module root: `services/atlas-character-factory/atlas.com/character-factory`.

Patterns to copy: `factory/resource.go:38-77` (`categorizePresetError` — the `errors.Is` switch
shape that `categorizeError` should adopt); `factory/processor.go:111-114` (the log-then-return-
sentinel style the new rejection must match); `factory/processor_test.go:949-959`
(`createMockContext`).

- [ ] **Step 1: Write the failing tests**

**(a) `TestCreate_RejectsOffCarouselRaceIndex`** — table-driven, calls `Create` through a
processor built the way `TestCharacterCreationOrchestrationFlow` (`processor_test.go:965`)
builds one, with a context from `createMockContext`-style
`tenantModel.Create(uuid.New(), region, major, minor)`.

| case | region/major/minor | JobIndex | SubJobIndex | expect |
|---|---|---|---|---|
| ordinal beyond carousel, v95 | GMS/95/0 | 9 | 0 | `errors.Is(err, ErrInvalidRaceIndex)`, `transactionId == ""` |
| uint32-max ordinal, v95 | GMS/95/0 | 4294967295 | 0 | `errors.Is(err, ErrInvalidRaceIndex)` |
| bogus subjob on a valid ordinal, v95 | GMS/95/0 | 1 | 7 | `errors.Is(err, ErrInvalidRaceIndex)` |
| ordinal beyond carousel, v83 | GMS/83/1 | 9 | 0 | `errors.Is(err, ErrInvalidRaceIndex)` |

All other input fields are a valid creation request — copy the `input RestModel{...}` literal
from `TestBuildCharacterCreationSaga_AllFieldsPresent` (`processor_test.go:358-436`) and override
only `JobIndex`/`SubJobIndex`. **Assert the error is the sentinel, and assert it is NOT
`ErrTemplateNotFound`** — the two gates must stay distinguishable (FR-18, D-7).

**(b) `TestCreate_ValidatorOrderIsPreserved`** — the deliberate behavior change in D-7 needs a
test that pins the *new* order, since race-index validation now happens after the tenant fetch:

| case | Name | Gender | JobIndex | expect error |
|---|---|---|---|---|
| bad name beats bad race index | `""` | 0 | 9 | message contains `"character name must be between 1 and 12 characters"` |
| bad gender beats bad race index | `"Valid"` | 5 | 9 | message contains `"gender must be 0 or 1"` |
| bad race index alone | `"Valid"` | 0 | 9 | `errors.Is(err, ErrInvalidRaceIndex)` |

**(c) `TestCategorizeError_InvalidRaceIndexIsBadRequest`** — in `resource.go`'s test file
(create `factory/resource_test.go` if none exists; check first).

| case | err | expect status |
|---|---|---|
| invalid race index | `ErrInvalidRaceIndex` | `http.StatusBadRequest` (400) |
| wrapped invalid race index | `fmt.Errorf("create: %w", ErrInvalidRaceIndex)` | `http.StatusBadRequest` |
| template not found | `ErrTemplateNotFound` | `http.StatusBadRequest` |
| name duplicate | `ErrNameDuplicate` | `http.StatusConflict` (409) |
| nil | `nil` | `http.StatusOK` |
| unknown | `errors.New("boom")` | `http.StatusInternalServerError` |

Note the `ErrTemplateNotFound` → 400 row: today that error falls through `categorizeError`'s
substring list to a **500**, because the list contains only message literals and not that
sentinel. A client asking for a slot with no seeded template is making a bad request, not
triggering a server fault. Fixing it here is in scope — FR-17 requires the client receive a clean
failure, and moving `categorizeError` to `errors.Is` is what makes the new sentinel reachable at
all. If a reviewer disputes the `ErrTemplateNotFound` status change, keep it at 500 and say so in
the task report; the `ErrInvalidRaceIndex` row is the non-negotiable one.

- [ ] **Step 2: Run them and confirm they fail**

```bash
go test ./factory/... -run 'TestCreate_RejectsOffCarouselRaceIndex|TestCreate_ValidatorOrderIsPreserved|TestCategorizeError_InvalidRaceIndexIsBadRequest' -v
```

Expected: FAIL — `undefined: ErrInvalidRaceIndex`.

- [ ] **Step 3: Add the sentinel**

In the `var (...)` block at `processor.go:26-32`, alongside `ErrTemplateNotFound`:

```go
	ErrInvalidRaceIndex     = errors.New("race index is not selectable on this client version")
```

- [ ] **Step 4: Resolve once in `Create`, reject, and reuse**

Delete the `validJob` call at `processor.go:100-102`:

```go
	if !validJob(input.JobIndex, input.SubJobIndex) {
		return "", errors.New("must provide valid job index")
	}
```

and delete the stub itself at `processor.go:649-651`. It is unreachable dead code once the call
is gone, and leaving a `func validJob(_ uint32, _ uint32) bool { return true }` behind is exactly
the stub CLAUDE.md forbids landing.

Insert, immediately **after** `t := tenant.MustFromContext(ctx)` (currently `processor.go:104`)
and before `configuration.GetTenantConfig`:

```go
	// task-283: the race ordinal is an index into the carousel THIS client version drew,
	// so it can only be resolved once the tenant is in hand. Resolved once here and
	// reused at the saga payload, so the validator and the payload cannot disagree.
	jobId, ok := job2.FromIndex(t, input.JobIndex, input.SubJobIndex)
	if !ok {
		p.l.Errorf("Race index [%d] subJobIndex [%d] is not selectable on region [%s] version [%d.%d]; rejecting creation so the client receives a failure instead of hanging.", input.JobIndex, input.SubJobIndex, t.Region(), t.MajorVersion(), t.MinorVersion())
		return "", ErrInvalidRaceIndex
	}
```

This moves race-index validation from before the tenant fetch to after it. Name and gender
validation still run first, so ordering among the pre-existing validators is unchanged — that is
the deliberate, tested behavior change from D-7, pinned by test (b).

- [ ] **Step 5: Thread the resolved `jobId` to the saga payload**

Change the signature at `processor.go:185` to:

```go
func buildCharacterCreationSaga(transactionId uuid.UUID, input RestModel, tmpl template.RestModel, jobId job.Id) saga.Saga {
```

and at `processor.go:206` replace

```go
		JobId:        job2.JobFromIndex(input.JobIndex, input.SubJobIndex),
```

with

```go
		JobId:        jobId,
```

Update the production call site at `processor.go:167` to pass `jobId`, and all **ten** test call
sites (`processor_test.go` lines 230, 269, 342, 386, 466, 511, 993, 1486, 1527, 1603) to pass the
literal expected job id for that test's `input.JobIndex`/`SubJobIndex` — the same literals Task 3
established. This is a mechanical sweep; the compiler finds every site.

- [ ] **Step 6: Switch `categorizeError` to sentinel matching**

Replace `categorizeError` (`resource.go:130-157`) with an `errors.Is` switch in the shape of
`categorizePresetError` (`resource.go:38-77`), keeping the existing substring list as a trailing
fallback so the un-sentinelled validators (`"character name must be…"`, `"gender must be 0 or 1"`,
the face/hair/skin/top/bottom/shoes/weapon messages) still map to 400. Drop
`"must provide valid job index"` from that list — no code produces it any more. Add:

```go
	switch {
	case errors.Is(err, ErrInvalidRaceIndex):
		return http.StatusBadRequest
	case errors.Is(err, ErrTemplateNotFound):
		return http.StatusBadRequest
	case errors.Is(err, ErrNameDuplicate):
		return http.StatusConflict
	}
```

before the substring loop. Confirm `errors` and `net/http` are imported in `resource.go`.

- [ ] **Step 7: Delete the service-local twin**

```bash
git rm services/atlas-character-factory/atlas.com/character-factory/job/model.go
```

`JobFromIndex` is that file's only declaration and has no remaining callers after Steps 5 and
Task 3.

- [ ] **Step 8: Run the module's tests and confirm the twin is gone**

```bash
go build ./... && go test ./... 
```
from `services/atlas-character-factory/atlas.com/character-factory` — Expected: PASS.

```bash
grep -rn "FromIndex" --include="*.go" .
```
from the worktree root — Expected: matches in
`services/atlas-character-factory/atlas.com/character-factory/job/carousel.go`,
`job/carousel_test.go`, `factory/processor.go`, and `factory/processor_test.go` **only**. Any
match in `libs/atlas-constants/` or any `JobFromIndex` match is FR-4 unsatisfied.

- [ ] **Step 9: Commit**

```bash
git add -A services/atlas-character-factory/atlas.com/character-factory/factory \
             services/atlas-character-factory/atlas.com/character-factory/job
git commit -m "feat(character-factory): reject off-carousel race indices and resolve job once"
```

---

## Task 7: Seed templates and the carousel↔template correspondence gate

**Driven by `findings.md` → `## Consequences for later tasks`.** Only versions whose verification
demands a change are touched. FR-21: pre-Big-Bang rows are not touched unless IDA **positively
contradicts** them.

### Files

- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — likely: add the `(4,0)` row pair, correct ordinal 0 and 2 `mapId`s
- `services/atlas-configurations/seed-data/templates/template_jms_185_1.json` — per findings
- `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` — per findings; the `(1,1)` row pair if Dual Blade outcome 3
- `services/atlas-configurations/seed-data/templates/template_gms_12_1.json`, `template_gms_48_1.json`, `template_gms_61_1.json`, `template_gms_72_1.json`, `template_gms_79_1.json`, `template_gms_83_1.json`, `template_gms_84_1.json`, `template_gms_87_1.json` — **read-only unless `findings.md` positively contradicts a row** (FR-21)
- `services/atlas-character-factory/atlas.com/character-factory/job/correspondence_test.go` — **new file**; the FR-19/FR-20 gate
- `docs/tasks/task-283-race-index-job-mapping/findings.md` — **new file**, created by Task 1; read-only here

Module root for the test: `services/atlas-character-factory/atlas.com/character-factory`.

Patterns to copy: `tools/packet-audit/internal/template/real_test.go:9-20` (locating a seed
template from a Go test via `runtime.Caller(0)`); an existing template row literal is quoted
below.

- [ ] **Step 1: Write the failing correspondence test**

`correspondence_test.go`, package `job`.

`TestCarouselMatchesSeedTemplates` — for every `(version key → template filename)` pair in the
table below, load the template JSON, project its `characters.templates[]` rows to a set of
`(jobIndex, subJobIndex, gender)`, and assert **both directions**:

1. every `Slot` in that version's carousel has a template row for gender `0` **and** gender `1`
2. every template row's `(jobIndex, subJobIndex)` is a `Slot` in that version's carousel

| version key | region | major | minor | template file |
|---|---|---|---|---|
| `gms_12` | GMS | 12 | 1 | `template_gms_12_1.json` |
| `gms_v48` | GMS | 48 | 1 | `template_gms_48_1.json` |
| `gms_v61` | GMS | 61 | 1 | `template_gms_61_1.json` |
| `gms_v72` | GMS | 72 | 1 | `template_gms_72_1.json` |
| `gms_v79` | GMS | 79 | 1 | `template_gms_79_1.json` |
| `gms_v83` | GMS | 83 | 1 | `template_gms_83_1.json` |
| `gms_v84` | GMS | 84 | 1 | `template_gms_84_1.json` |
| `gms_v87` | GMS | 87 | 1 | `template_gms_87_1.json` |
| `gms_v92` | GMS | 92 | 1 | `template_gms_92_1.json` |
| `gms_v95` | GMS | 95 | 0 | `template_gms_95_1.json` |
| `gms_jms_185` | JMS | 185 | 1 | `template_jms_185_1.json` |

**Exception, stated in the test as a named constant with this comment:** a version whose carousel
is broader than its seeded rows because the version genuinely never seeded a slot is not a bug —
but the only such case established at plan time is `gms_12`, whose carousel is the pre-Big-Bang
default while its template seeds only `(1,0)`. Direction (1) is therefore skipped for `gms_12`
alone, with the reason inline; direction (2) still runs for it. If `findings.md` establishes any
other version key needs the same exemption, add it to the same constant with its own cited
reason. Do not silence a failure by widening the exemption without a finding.

Minimal decode shape (only the fields the test needs):

```go
type seedTemplate struct {
	Region       string `json:"region"`
	MajorVersion uint16 `json:"majorVersion"`
	MinorVersion uint16 `json:"minorVersion"`
	Characters   struct {
		Templates []struct {
			JobIndex    uint32 `json:"jobIndex"`
			SubJobIndex uint32 `json:"subJobIndex"`
			MapId       uint32 `json:"mapId"`
			Gender      byte   `json:"gender"`
		} `json:"templates"`
	} `json:"characters"`
}
```

Also assert each file's `region`/`majorVersion`/`minorVersion` equal the table's values, so a
renamed or re-keyed template file fails loudly instead of being silently skipped.

- [ ] **Step 2: Run it and confirm it fails**

```bash
go test ./job/... -run TestCarouselMatchesSeedTemplates -v
```

Expected: FAIL, listing the missing rows — at minimum the v95 slots `findings.md` verified that
have no template row (FR-19's known instance is `(4,0)`), and any row whose slot is not in the
carousel (FR-20 / Dual Blade outcome 3).

- [ ] **Step 3: Add the missing rows**

For each missing `(jobIndex, subJobIndex)`, add **two** entries (gender 0 and gender 1) to that
file's `characters.templates` array. Row shape, verbatim from
`template_gms_95_1.json`'s first entry:

```json
{
  "jobIndex": 0,
  "subJobIndex": 0,
  "mapId": 130010220,
  "gender": 0,
  "faces": [20000, 20001, 20002],
  "hairs": [30030, 30020, 30000],
  "hairColors": [0, 2, 3, 7],
  "skinColors": [0, 1, 2, 3],
  "tops": [1040002, 1040006, 1040010],
  "bottoms": [1060002, 1060006],
  "shoes": [1072001, 1072005, 1072037, 1072038],
  "weapons": [1302000, 1322005, 1312004],
  "items": [4161047]
}
```

`mapId` for a new row is **read from WZ data** for that class's start map and recorded in
`findings.md` § Consequences with the WZ path — never assumed. Copy the appearance/equipment
option lists from the same file's existing row for the nearest comparable class, and say in the
task report which row you copied from.

- [ ] **Step 4: Correct the contradicted `mapId`s**

Only where `findings.md` cites a positive contradiction. The candidates flagged in the PRD are
`template_gms_95_1.json` ordinal 2 (`mapId 140090000`, Aran's start map) and ordinal 0
(`130010220`) — both wrong if the v95 carousel binds those ordinals to different classes than
pre-Big-Bang. Change the `mapId` for **both** gender rows of each corrected ordinal.

- [ ] **Step 5: Handle the `(1,1)` rows per the Dual Blade outcome**

- Outcome 1 or 2 (Dual Blade maps to something) → carousel has a `{1,1}` entry and the existing
  `(1,1)` rows in `template_gms_92_1.json`, `template_gms_95_1.json`, `template_jms_185_1.json`
  stay. Verify their `mapId` against the findings.
- Outcome 3 (slot not offered) → **remove** the `(1,1)` row pair from all three files. Direction
  (2) of the correspondence test is what forces this.

- [ ] **Step 6: Confirm every touched file is still valid JSON and line endings survived**

```bash
python3 -c "
import json,glob
for p in sorted(glob.glob('services/atlas-configurations/seed-data/templates/*.json')):
    d=json.load(open(p))
    rows=d['characters']['templates']
    print(p.split('/')[-1], d['region'], d['majorVersion'], sorted({(r['jobIndex'],r['subJobIndex'],r['mapId']) for r in rows}))
"
```

Expected: each file parses, and the printed slot set matches that version's carousel.

```bash
git diff --stat -- services/atlas-configurations/seed-data/templates/
```

Expected: only the files `findings.md` named appear, and the changed-line count is small. A file
showing every line changed means the editor normalized CRLF→LF — revert it and redo the edit
preserving line endings.

- [ ] **Step 7: Run the test**

```bash
go test ./job/... -v
```

from `services/atlas-character-factory/atlas.com/character-factory`. Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/ \
        services/atlas-character-factory/atlas.com/character-factory/job/correspondence_test.go
git commit -m "fix(configurations): align creation templates with verified race carousels"
```

---

## Task 8: Version-aware class labels in the template editor

### Files

- `services/atlas-ui/src/components/features/characters/templates/jobNames.ts` — replace the version-invariant `WORLD_NAMES`/`KNOWN_CLASSES` with per-version tables
- `services/atlas-ui/src/components/features/characters/templates/IdentitySection.tsx` — the class dropdown at lines 24-78; consume the tenant's version
- `services/atlas-ui/src/components/features/characters/templates/__tests__/jobNames.test.ts` — update; it currently asserts the version-invariant mapping
- `services/atlas-ui/src/components/features/characters/templates/__tests__/raceCarousels.parity.test.ts` — **new file**; the D-9 TS side of the parity gate
- `services/atlas-ui/src/components/features/characters/templates/TemplateSelector.tsx:5` — read-only unless `templateLabels`' signature changes
- `services/atlas-ui/src/components/features/characters/templates/CharacterTemplatesEditor.tsx:13` — read-only unless `templateLabels`' signature changes
- `services/atlas-ui/src/context/tenant-context.tsx:216` — read-only; `useTenant()`
- `docs/packets/race-carousels.json` — **new file**, created by Task 2; read-only here, the fixture

Module root: `services/atlas-ui` (`npm test` → `vitest run`; config lives in `vite.config.ts:27-33`,
there is no separate `vitest.config.*`).

Patterns to copy: `services/atlas-ui/src/components/features/characters/templates/__tests__/jobNames.test.ts`
(the `describe`/`it`/`expect` shape for a pure-function test in this directory). There is **no
existing test in `services/atlas-ui/src` that reads a file from disk** — a repo-wide grep for
`readFileSync`/`node:fs` under `src` returns nothing — so the parity test introduces that
pattern; `vitest`'s `jsdom` environment still permits `node:fs` in the test process.

**Tenant shape, established during planning:** `useTenant()` returns a context whose
`activeTenant` is `Tenant | null` where `Tenant = TenantBasic`
(`services/atlas-ui/src/services/api/tenants.service.ts:143`), and the version fields are nested:
`activeTenant?.attributes.majorVersion` and `activeTenant?.attributes.region`
(`tenants.service.ts:14-24`). **Not** top-level fields.

- [ ] **Step 1: Write the failing tests**

**(a) `raceCarousels.parity.test.ts`** — reads `docs/packets/race-carousels.json` and asserts the
TS tables cover exactly the same slots as the fixture, per version:

```ts
import { readFileSync } from "node:fs";
import path from "node:path";
import { describe, it, expect } from "vitest";
import { classesForVersion } from "../jobNames";

const FIXTURE = path.resolve(
  __dirname, "..", "..", "..", "..", "..", "..", "..",
  "docs", "packets", "race-carousels.json",
);
```

Resolve the relative depth by running the test once and printing `FIXTURE` if it does not
resolve — `__dirname` is
`services/atlas-ui/src/components/features/characters/templates/__tests__`, so the repo root is
seven levels up.

For each version key in the fixture, assert
`classesForVersion(region, majorVersion).map(c => [c.jobIndex, c.subJobIndex, c.label])`, sorted,
equals the fixture's `slots.map(s => [s.raceIndex, s.subJobIndex, s.class])`, sorted. Labels must
match the fixture's `class` string exactly — that is what makes the fixture the single source for
both sides.

**(b) rewrite `jobNames.test.ts`** — it currently asserts `worldNameFromJobIndex(0..3)` against
the hardcoded pre-Big-Bang names and asserts the file "mirrors `JobFromIndex`". Replace with:

`describe("worldNameFromJobIndex")`, cases keyed to version:

| case | region | major | jobIndex | expect |
|---|---|---|---|---|
| pre-Big-Bang slot 0 | GMS | 83 | 0 | the `class` for `gms_v83` raceIndex 0 in `findings.md` |
| pre-Big-Bang slot 1 | GMS | 83 | 1 | the `class` for `gms_v83` raceIndex 1 |
| pre-Big-Bang slot 2 | GMS | 83 | 2 | the `class` for `gms_v83` raceIndex 2 |
| v95 slot 1 | GMS | 95 | 1 | the `class` for `gms_v95` raceIndex 1 |
| v95 slot 2 | GMS | 95 | 2 | the `class` for `gms_v95` raceIndex 2 |
| unknown ordinal falls back (FR-23) | GMS | 95 | 42 | `"Job 42"` |
| unknown ordinal, pre-Big-Bang (FR-23) | GMS | 83 | 42 | `"Job 42"` |
| no tenant selected falls back | `undefined` | `undefined` | 1 | `"Job 1"` |

`genderLabel` and `templateLabels` cases from the existing file are kept as-is except that
`templateLabels` now takes the version — keep its `" (2)"`/`" (3)"` duplicate-ordinal behavior and
its existing cases, with the version threaded through.

- [ ] **Step 2: Run them and confirm they fail**

```bash
npm test -- src/components/features/characters/templates
```

from `services/atlas-ui`. Expected: FAIL — `classesForVersion` is not exported.

- [ ] **Step 3: Rewrite `jobNames.ts`**

Delete the stale header comment at `jobNames.ts:1-4` (`"World names mirror atlas-character-factory
job/model.go JobFromIndex"`) — after task-283 there is no version-invariant mirror to claim, and
the parity test is what enforces the relationship now.

Replace `WORLD_NAMES` and `KNOWN_CLASSES` with a per-version structure whose contents are
transcribed from `docs/packets/race-carousels.json`, and export:

```ts
export type RaceClass = {
  jobIndex: number;
  subJobIndex: number;
  label: string;
};

// Per-version race carousels, transcribed from docs/packets/race-carousels.json, which is
// the machine-readable projection of the IDA findings in
// docs/tasks/task-283-race-index-job-mapping/findings.md. raceCarousels.parity.test.ts
// asserts this table against that file; a drift here fails that test rather than silently
// mislabelling a tenant admin's editor.
export function classesForVersion(
  region: string | undefined,
  majorVersion: number | undefined,
): readonly RaceClass[];

export function worldNameFromJobIndex(
  jobIndex: number,
  region: string | undefined,
  majorVersion: number | undefined,
): string;

export function templateLabels(
  templates: Pick<CharacterTemplate, "jobIndex" | "gender">[],
  region: string | undefined,
  majorVersion: number | undefined,
): string[];
```

`classesForVersion` mirrors the Go `carouselFor` predicate chain — same arms, same order,
most-specific first, and the same pre-Big-Bang default. `worldNameFromJobIndex` keeps the
`` `Job ${jobIndex}` `` fallback (FR-23) and returns it whenever `region`/`majorVersion` are
`undefined` or the ordinal has no entry. `genderLabel` is unchanged.

Note the `(1,1)` gap this closes: `KNOWN_CLASSES` today has no `(1,1)` entry even though
`gms_92`, `gms_95`, and `jms_185` all seed a `(1,1)` row, so that class falls to the "unknown"
label path. Whether it gains a label is decided by the Dual Blade outcome in `findings.md`.

- [ ] **Step 4: Thread the tenant version through the callers**

`IdentitySection.tsx:24-78` — add `const { activeTenant } = useTenant();` and pass
`activeTenant?.attributes.region` / `activeTenant?.attributes.majorVersion` into
`classesForVersion` (replacing the `KNOWN_CLASSES` import and its `.find`) and into
`worldNameFromJobIndex`. The `known ? classValue : ""` select-value guard and the
`` `${worldNameFromJobIndex(...)} (${classValue})` `` fallback label stay — an ordinal with no
label must still render (FR-23).

`TemplateSelector.tsx:5` and `CharacterTemplatesEditor.tsx:13` import `templateLabels`; add the
two arguments at both call sites from `useTenant()` in the same way.

- [ ] **Step 5: Run the UI tests and the type check**

```bash
npm test
npm run build
```

from `services/atlas-ui`. Expected: PASS, and a clean TypeScript build.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-ui/src/components/features/characters/templates
git commit -m "fix(atlas-ui): label creation-template classes per tenant client version"
```

---

## Task 9: Full verification gate

### Files

- `tools/verify.sh` — read-only; the gate
- `docs/verification.md` — read-only; read this if a guard fails
- `docs/tasks/task-283-race-index-job-mapping/plan.md` — check off every box

No file is edited by this task unless the gate demands it.

- [ ] **Step 1: Confirm FR-4 — exactly one mapper survives**

```bash
grep -rn "FromIndex" --include="*.go" .
```

from the worktree root. Expected: matches only in
`services/atlas-character-factory/atlas.com/character-factory/job/carousel.go`,
`job/carousel_test.go`, `factory/processor.go`, `factory/processor_test.go`. Any hit in
`libs/atlas-constants/`, or any `JobFromIndex` hit at all, fails the acceptance criterion.

- [ ] **Step 2: Confirm no raw version comparison crept in (FR-2)**

```bash
grep -rn "MajorVersion() >\|MajorVersion() <\|MajorVersion() ==" --include="*.go" services/atlas-character-factory
```

Expected: no output. Every version predicate must be `IsRegion` / `MajorAtLeast` /
`MajorAtMost` / `MajorInRange`.

- [ ] **Step 3: Confirm no stub or placeholder landed**

```bash
grep -rn "TODO\|FIXME\|BladeRecruit TODO" --include="*.go" services/atlas-character-factory/atlas.com/character-factory/job services/atlas-character-factory/atlas.com/character-factory/factory
```

Expected: no output. The `// jobId = job.BladeRecruit TODO` comment must be gone with the deleted
`job/model.go`, and `validJob`'s always-true stub must be gone with Task 6.

- [ ] **Step 4: Run the flagless gate**

```bash
tools/verify.sh
```

from the worktree root. **Flagless — `--quick` and `--no-docker` skip the docker bake and
`go test -race` and do not count** (CLAUDE.md, "Done means verified"). Expected: exit 0.

If a guard fails, read `docs/verification.md` before changing anything; a script/CI disagreement
is covered there.

Dispatch this via `task-verifier` in its own context rather than running it inline — the build
and lint output should not land in an implementer's window.

- [ ] **Step 5: Confirm every acceptance criterion in the PRD**

Walk `prd.md` §10 item by item and record PASS + the evidence (file:line, test name, or command
output) for each in the task report. The three that are easy to leave unproven:

- "the pre-Big-Bang mapping is IDA-verified rather than inferred" — points at `findings.md`, not
  at a passing test
- "`(1,1)` Dual Blade no longer produces a plain Beginner" — must cite the specific findings
  outcome and the test that pins it
- "Code review completed before PR" — Task 9 does not satisfy this; it is the next gate

- [ ] **Step 6: Commit any gate-driven fixes, then hand off to review**

```bash
git add -A
git commit -m "chore(task-283): verification gate fixes"
```

Then run code review before opening a PR — `tools/verify.sh` cannot see a cross-service seam
defect, and this change crosses `atlas-character-factory` → `atlas-configurations` (seed data)
and → `atlas-ui` (labels).
