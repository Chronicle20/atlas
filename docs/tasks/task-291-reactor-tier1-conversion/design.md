# Tier-1 Reactor Script Conversion — Design

Task: task-291-reactor-tier1-conversion
PRD: [prd.md](prd.md)
Status: Draft
Created: 2026-09-02

---

## 1. Scope of this document

The PRD fixes *what* must exist: a repaired `/convert-reactor` contract, a repaired
schema, 13 audited seeds, 159 converted reactors fanned out to eleven tenant
directories, and the removal of the legacy `minMeso`/`maxMeso` fallback. This
document fixes *how*: who generates the 1,749 files, what the review surface is,
what enforces the acceptance criteria mechanically, and in what order the pieces
land.

Three decisions carry the design:

1. Conversion is performed by a **committed, fail-closed generator** over a
   restricted grammar, not by 159 agent conversions and not by a general JS parser.
2. Verification is a **new `tools/reactor-seed-lint` Go tool** that validates the
   seed corpus against the JSON schema *and* asserts three properties the schema
   cannot express.
3. The generator's judgment calls — descriptions, PQ mappings, loop unrolls — live
   in a small **override table inside the generator**, so review targets ~100 lines
   of curated data plus a sampled diff, never 1,749 files.

## 2. Grounding: what the engine actually does

Read from the shipped code in this worktree, not from memory:

| Fact | Evidence |
|---|---|
| 11 operations dispatched | `services/atlas-reactor-actions/atlas.com/reactor/script/executor.go:45-84` |
| 2 condition types, 6 operators | `.../evaluator.go:30-40`, `:67-89`, `:125-146` |
| `drop_items` reads `dropType`, `meso`, `mesoChance`, `mesoMin`, `mesoMax`, `minItems`; legacy fallback reads `minMeso`/`maxMeso` only | `.../executor.go:101-150` |
| Defaults when a param is absent: `mesoChance=1, mesoMin=1, mesoMax=1, minItems=0`, `dropType="drop"` | `.../executor.go:114` |
| `spray_items` mutates `op.Params()["dropType"]="spray"` then delegates to `executeDropItems` — identical parameter set | `.../executor.go:247-256` |
| `spawn_monster` reads `monsterId` (required), `count` (default 1), `x`/`y` (default reactor position) | `.../executor.go:185-215` |
| `pq_custom_data` uses `cond.Step()` as the custom-data key; a missing key or non-numeric value compares as `0`; an unresolvable PQ instance is *not* an error, it is "condition not met" | `.../evaluator.go:94-123` |
| `update_pq_state`, `broadcast_pq_message`, `stage_clear_attempt` all fail the operation when the PQ instance lookup fails | `.../executor.go:351-358`, `:453-456`, `:478-482` |
| Seeding is delete-all-then-recreate per tenant per subdomain; `CATALOG_REVISION` is reported, not consulted for skipping | `libs/atlas-seeder/seed.go:86-96`, `:145-155` |

Two consequences worth stating up front, because they shape §5 and §9:

- **The schema as written cannot catch the meso-shift regression.** The
  `drop_items` `allOf` branch lists properties but does not set
  `additionalProperties: false`, and the top-level `params` allows any string
  value. `"mesoRange": "15"` validates today. The schema must be tightened, or the
  lint must carry a separate legacy-key assertion. This design does both (§5.2).
- **Seeding is idempotent by construction.** `runSubdomain` deletes every row for
  the tenant before inserting, so 159 new files change the row count and nothing
  else. No `CATALOG_REVISION` bump is required by this task; that file is bumped by
  the image-overlay automation, and touching it here would only produce a spurious
  drift warning.

## 3. Conversion architecture

### 3.1 Alternatives considered

**(A) 159 per-script agent conversions via the repaired `/convert-reactor` skill.**
This is what the skill was written for. It is also the worst fit: 159 independent
judgment calls with no mechanical guarantee of consistency, 1,749 files whose only
provenance is "an agent said so," and a review surface that is exactly the
unreviewable bulk diff the PRD calls out in §8. Rejected.

**(B) A general JavaScript parser (goja / esprima) walking each Cosmic script.**
Correct in principle and robust to corpus growth, but it buys generality this
corpus does not need and introduces an AST-walk layer that is itself the thing
needing review. Rejected as YAGNI — see the census below.

**(C, recommended) A committed Go generator over a restricted, whitelisted
grammar, reading `tier1-inventory.md` as its input, failing closed on anything it
does not recognize.**

The census justifies (C). Every statement in all 159 Tier-1 bodies, normalized
(`[0-9]+`→`N`, string literals→`"S"`):

```
  85  rm.dropItems();
  16  rm.dropItems(true, N, N, N);
  14  rm.sprayItems(true, N, N, N, N);
   8  rm.spawnMonster(N);
   7  rm.weakenAreaBoss(N, "S");
   5  rm.sprayItems();
   5  rm.dropItems(true, N, N, N, N);
   4  rm.weakenAreaBoss(N, "S")          [no semicolon]
   4  if (rm.getReactor().getState() !== N) { return }
   3  rm.spawnMonster(N, N);
   3  rm.spawnMonster(N);                [inside a for body]
   2  var eim = rm.getPlayer().getEventInstance();
   2  var now = eim.getIntProperty("S");
   2  var nextNum = now + N;
   2  eim.setIntProperty("S", nextNum);
   2  for (var i = N; i < N; i++) {
   1  rm.spawnMonster(N, N, N, N);
   1  rm.spawnMonster(N, N, -N, N);
   1  rm.dropItems(false, N, N, N, N);
   1  var eim = rm.getEventInstance();
   1  rm.getEventInstance().setProperty("S", "S");
   1  if (rm.getEventInstance() != null) { ... }
   1  eim.setProperty("S", "S");
```

Twenty-three forms, of which about eight are structural. A hand-written matcher
for that is smaller and far more auditable than an AST walker, and — critically —
it can be made *fail-closed*: any line that matches none of the whitelisted forms
aborts the whole generation with the reactor ID and the offending line. There is no
"skip and report," no partial output, no silent drop. That property is what makes
the tool trustworthy in a way (A) never is.

### 3.2 Generator shape

New Go module `tools/reactor-seed-gen/` (own `go.mod`, added to `go.work`),
following the `tools/catalog-lint` precedent exactly: `main.go`, focused
companions, `main_test.go`, `testdata/`.

```
tools/reactor-seed-gen/
  go.mod
  main.go        // CLI: parse flags, run pipeline, write or --check
  parse.go       // tier1-inventory.md → []sourceScript{id, comment, hitBody, actBody}
  convert.go     // sourceScript → scriptDoc (the whitelisted grammar lives here)
  describe.go    // comment → description, with the override table
  emit.go        // scriptDoc → JSON:API bytes; fan-out to the eleven directories
  *_test.go
  testdata/
```

**Pipeline.** `parse` splits the inventory on `### \`<id>\`` headings and captures
the fenced body verbatim; `convert` walks each body statement-by-statement against
the grammar; `emit` marshals and writes. Each stage is independently testable and
none knows about the others' formats beyond a plain struct.

**Grammar → rules mapping.**

| Source form | Emitted |
|---|---|
| `rm.dropItems()` | `drop_items`, no params |
| `rm.dropItems(m, c, lo, hi)` | `drop_items` with `meso`, `mesoChance`, `mesoMin`, `mesoMax` |
| `rm.dropItems(m, c, lo, hi, n)` | as above plus `minItems` |
| `rm.sprayItems(...)` | `spray_items`, same positional mapping |
| `rm.spawnMonster(id)` | `spawn_monster` `{monsterId}` |
| `rm.spawnMonster(id, n)` | `+ count` |
| `rm.spawnMonster(id, n, x, y)` | `+ count, x, y` |
| `rm.weakenAreaBoss(id, msg)` | `weaken_area_boss` `{monsterId, message}` |
| `if (getState() !== K) { return }` guard | hoists a `reactor_state` `=` `K` condition onto every rule built from the statements after it (FR-17) |
| `for (var i = 0; i < K; i++) { … }` | multiplies each `spawn_monster` inside by `count = K × (its own count)`; any other operation inside the loop is a generator error (FR-18) |
| `var eim = rm.getPlayer().getEventInstance()` / `rm.getEventInstance()` / `if (eim != null) {` | no output — binding and null guard are erased |
| `var now = eim.getIntProperty("k"); var nextNum = now + 1; eim.setIntProperty("k", nextNum)` | one `update_pq_state` `{increments: "k"}` — matched as a three-statement idiom, not statement by statement |
| `eim.setProperty("k", "v")` / `setIntProperty("k", <literal>)` | `update_pq_state` `{updates: "k=v"}` |
| anything else | **hard error, generation aborts** |

The `nextNum` idiom is matched as a *unit*. Matching `getIntProperty` alone would
lose the fact that the write is an increment of the read, which is the whole
semantic content. If a future script increments by something other than 1, the
idiom will not match and the generator will abort rather than guess — the correct
outcome.

**Rule identity and ordering.** Rules are emitted in source order. Rule `id` is
derived deterministically from the operation types in the rule (`drop_items`,
`update_pq_state_spawn_monster`, …), matching the convention in the existing
`reactor-2001.json` and `reactor-9102002.json` (`"id": "drop_items"`). Ordering
matters at runtime: `processor.go:193-215` takes the first matching rule, so a guarded rule
must precede an unguarded fallback. Tier-1 has no script with both, but the
generator preserves source order regardless so the property holds by construction.

**Empty bodies.** A `function act() {}` yields `[]`, not an omitted key (FR-19).
An absent function likewise yields `[]`. The six `9018000`–`9018005` flowers land
as `{"hitRules": [], "actRules": []}` with a description drawn from their
`@purpose Flower N` comment.

### 3.3 Descriptions (FR-15)

The inventory's `**Source comment:**` field is mostly AGPL boilerplate:

> `This file is part of the HeavenMS MapleStory Server Copyleft (L) 2016 - 2019 RonanLana … @Author Ronan * * 1021000.js: relic room fail www.gnu.org/licenses/>.`

`describe.go` applies a deterministic cleaner: strip the known license paragraph,
strip `@Author`/`@author` clauses, strip trailing URL debris (`www.gnu.org/...`,
`mymapleland.blogspot.com/...`), strip a leading `<id>.js:` prefix, collapse
whitespace, trim `*` decorations. `1021000` reduces to `relic room fail`, which is
then title-cased into `Relic room fail`.

Some comments reduce to nothing (`8001000`'s comment is only
`www.gnu.org/licenses/>.`). For those, and for any case where the cleaned comment
is shorter than a threshold or is judged uninformative, `describe.go` consults an
**override table** — a `map[string]string` keyed by reactor ID, committed in the
source file. An entry is required whenever the cleaner produces an empty or
sub-threshold result; the generator aborts if a reactor needs one and does not have
one. This is where the human judgment for FR-15 ("state the reactor's observable
behaviour") lives, and it is the part of the change a reviewer should actually
read: roughly 20–40 short strings, not 1,749 files.

The cleaner is not permitted to emit AGPL boilerplate into a seed file. A test
asserts no generated description contains `gnu.org`, `Copyleft`, `@author`, or
`WITHOUT ANY WARRANTY`.

### 3.4 Emission and fan-out (FR-14)

`emit.go` marshals a `map[string]any` envelope with `json.MarshalIndent(v, "", "  ")`
and appends `\n`. Go sorts map keys, which reproduces the existing files'
alphabetical key order (`actRules`, `description`, `hitRules`, `reactorId`;
`attributes`, `id`, `type`) byte for byte — verified against
`reactor-2001.json`, which is 2-space indented, alphabetically keyed, LF, with a
single trailing newline.

The eleven destinations are a literal list in the generator
(`gms/{12_1,48_1,61_1,72_1,79_1,83_1,84_1,87_1,92_1,95_1}`, `jms/185_1`). Each
reactor is marshalled **once** and the same `[]byte` written to all eleven paths,
so byte-identity is structural rather than something to be checked after the fact —
though §5 checks it anyway, because the seed corpus can also be edited by hand.

`--check` mode regenerates in memory and diffs against disk, exiting non-zero on
any difference. That gives CI a way to catch a hand-edit that drifts from the
inventory without granting the generator write access during verification.

### 3.5 Why the inventory and not `Cosmic/scripts/reactor/`

`tier1-inventory.md` is committed in this task folder, carries verbatim bodies, and
is already the artifact the PRD points reviewers at (FR-13). Reading it keeps the
generator runnable in CI and in any clone, with no dependency on a sibling
reference-server checkout whose path is not repo-relative. The cost is that the
inventory becomes a build input rather than a document; the `--check` mode makes
that honest by failing when the two disagree.

## 4. The 13 existing seeds (FR-10, FR-11, FR-12)

These are **not** generator output. `reactor-2001` is not in the Tier-1 159, and
the twelve PQ scripts (`9102002`–`9102007`, `9108000`–`9108005`) use PQ shapes the
Tier-1 grammar does not cover. Folding them into the generator would mean teaching
it a grammar for thirteen files. They are corrected by hand.

- `reactor-2001.json` — the known regression. Source is
  `rm.dropItems(true, 2, 8, 15, 1)`; current params are
  `meso/minMeso=2/maxMeso=8/mesoRange=15/item=1`; corrected params are
  `meso=true, mesoChance=2, mesoMin=8, mesoMax=15, minItems=1`. Applied to all
  eleven copies.
- The other twelve — each read against its Cosmic source and given a **recorded
  outcome**, not an assertion that it was checked. The record is a table in
  `docs/tasks/task-291-reactor-tier1-conversion/existing-seed-audit.md`, one row
  per reactor: source path, source body, current seed, verdict (`correct` /
  `corrected: <specific change>`). The PRD's acceptance criterion is the recorded
  outcome, so the artifact is the deliverable.

Post-condition for §4 and §3 together, and the gate for §6:
`grep -rn 'minMeso\|maxMeso\|mesoRange\|"item"' deploy/seed/*/reactor-actions/`
returns nothing.

## 5. Verification

### 5.1 Alternatives considered

**(A) Hand-rolled Go validator with the schema's rules transcribed.** No new
dependency, but it duplicates the schema and drifts from it, which directly
contradicts the PRD's goal of an *authoritative* schema. Rejected.

**(B) `npx ajv` in a shell script.** Requires network at verify time and puts a
Node toolchain in a Go repo's gate path. Rejected.

**(C, recommended) A new `tools/reactor-seed-lint` Go module that loads
`reactor_script_schema.json` with a real JSON-Schema library
(`santhosh-tekuri/jsonschema/v6`) and validates every seed file against it.** The
dependency is confined to that module's own `go.mod`, exactly as
`tools/catalog-lint` confines its `atlas-seeder` dependency. The schema file stays
the single source of truth, which is the point.

### 5.2 What the lint asserts

The seed files are JSON:API envelopes; the schema describes the bare script object.
The lint unwraps `.data.attributes` and validates *that* against the schema — no
envelope wrapper is added to the schema, because the schema also describes the REST
resource and duplicating the envelope there would couple two contracts that are
deliberately separate.

Four assertions, run over every file under
`deploy/seed/*/reactor-actions/reactors/`:

1. **Schema conformance** of `.data.attributes` against `reactor_script_schema.json`.
2. **Envelope well-formedness** — `data.type == "reactor-action"`, and
   `data.id == attributes.reactorId ==` the `reactor-<id>.json` filename stem. This
   mirrors the seeder's own checks (`libs/atlas-seeder/seed.go:168-182`), so a file
   that fails the lint would also have failed at seed time.
3. **Non-empty `description`** (FR-15) — the schema declares `description` optional,
   and making it required would be wrong for the resource generally; it is a
   corpus rule, so it belongs in the lint.
4. **Cross-tenant byte identity** (FR-14) — every reactor ID present in any of the
   eleven directories must be present in all eleven with an identical SHA-256.
   This catches both a missing fan-out and a one-off hand edit.

The legacy-key ban is enforced **in the schema**, not in the lint: `emit.go`'s
output is only as safe as the schema is strict, so the `drop_items` and
`spray_items` `allOf` branches gain `"additionalProperties": false` around the
five-parameter set. That converts FR-9 from a documentation change into an
enforced one — `"mesoRange": "15"` becomes a validation failure rather than a
silently-ignored key. The `grep` in the acceptance criteria remains as a
belt-and-braces check, but the schema is the real gate.

`dropType` is deliberately **not** added to the declared `drop_items` parameters
(PRD §9.2). It is injected at runtime by `executeSprayItems` and must never appear
in a seed file; `additionalProperties: false` now enforces that.

### 5.3 Wiring

`tools/reactor-seed-lint.sh` wraps `go run ./tools/reactor-seed-lint deploy/seed`
with the repo's standard script preamble, and `tools/verify.sh` gains a
path-gated step following the existing idiom:

```sh
if touched '^deploy/seed/.*/reactor-actions/|^services/atlas-reactor-actions/docs/reactor_script_schema\.json$|^tools/reactor-seed-lint'; then
    step "reactor seed lint" ./tools/reactor-seed-lint.sh
else
    skip "reactor seed lint (no reactor seed or schema changed)"
fi
```

The generator's `--check` mode is *not* wired into `verify.sh`. It depends on
`tier1-inventory.md`, a task-folder artifact that will not survive as a permanent
build input once the task is merged; the lint, which depends only on the seed
corpus and the schema, is the durable gate. The generator ships with unit tests
that run under the existing Go test sweep.

A `tools/reactor-seed-lint_test.sh` accompanies the wrapper, per the
`changed_tool_suites()` convention in `verify.sh:225-243`.

### 5.4 Sampled human review (PRD §8 gate 2)

One converted reactor per shape in PRD §6 — ten shapes — read against its inventory
body, recorded in `existing-seed-audit.md` alongside the §4 audit. The mandated
sample (`reactor-2001`, one `reactor_state`-guarded script, both unrolled-loop
scripts, one empty-body script) is a subset of that.

## 6. Sequencing

The ordering constraint is real, not cosmetic: FR-20's removal of the legacy
fallback is safe only after no seed depends on it.

1. **Contract repair.** `/convert-reactor` skill (FR-1…FR-7) and
   `reactor_script_schema.json` (FR-8, FR-9, plus the `additionalProperties: false`
   tightening from §5.2). Landing the schema first means every later step is
   validated against the corrected contract.
2. **Lint tool.** `tools/reactor-seed-lint` + wrapper + `verify.sh` wiring, with
   tests. At this point it runs green over the 13 existing files only if they are
   already clean — which `reactor-2001` is not, so this step's tests use
   `testdata/`, and the corpus run is expected to fail until step 3.
3. **Existing-seed correction and audit.** §4. After this, the lint passes over the
   13-file corpus.
4. **Generator.** `tools/reactor-seed-gen` with tests, exercised against
   `testdata/` fixtures covering every grammar form.
5. **Generation.** Run it; 1,749 files appear; the lint passes over 1,762 files.
6. **Legacy removal.** The `grep` gate returns empty → delete the `minMeso`/
   `maxMeso` fallback branches and the backward-compatibility comment from
   `executeDropItems`; update the function's doc comment; `atlas-reactor-actions`
   tests pass.
7. **Sampled review** (§5.4) and flagless `tools/verify.sh`.

Steps 1–2 and 4 are independent of each other and can be built in parallel; 3
depends on 1, 5 depends on 3 and 4, 6 depends on 5.

## 7. Testing

| Unit | Tests |
|---|---|
| `parse.go` | inventory fixture with headings, comment/no-comment, multi-function bodies → expected `sourceScript` structs |
| `convert.go` | one case per grammar row in §3.2, including both loop scripts, all four guard scripts, the `nextNum` increment idiom, and the `eim != null` wrapper; plus **negative cases** asserting an unrecognized statement aborts with the reactor ID in the error |
| `describe.go` | boilerplate stripping across the four comment shapes in the corpus; override-table hit; missing-override abort; the no-boilerplate-leaks assertion |
| `emit.go` | golden-file test asserting byte equality with a checked-in expected envelope, including key order and trailing newline |
| `reactor-seed-lint` | `testdata/` corpus with a valid file, a `mesoRange` file, a description-less file, a type/id mismatch, and an eleven-directory set with one divergent copy — each asserting the specific failure |
| `atlas-reactor-actions` | existing `executor` tests updated for the removed fallback; a test asserting `mesoMin`/`mesoMax` are read and that a bare `minMeso` no longer has any effect |

The generator is a data transformer with a pure core, so TDD applies cleanly: write
the grammar table as test cases first, then make them pass.

## 8. What this design does not do

- No new operation or condition type; no engine behaviour change beyond deleting
  the fallback branches.
- No `untouchRules`; no Tier-2 or Tier-3 conversion.
- No migration to the shared catalog root. §9 records why this is the one decision
  most likely to be revisited.
- No change to the drop, monster, saga, or party-quest services. `reactor-2001` is
  the only existing reactor whose emitted `SpawnReactorDropsPayload` values change.

## 9. Risks and open items

**The eleven-copy fan-out is the load-bearing regret.** `libs/atlas-seeder`'s
`NewFilesystemCatalogSourceWithShared` already exists and would reduce this task's
output from 1,749 files to 159. The PRD defers it (§9.1) and this design honours
that, but the generator's fan-out is deliberately a *single literal list in
`emit.go`* — adopting the shared root later is then a one-function change plus a
`git rm`, not a rewrite. Doing it in this task would fold a seeder-behaviour change
(and the documented one-time fleet-wide "seed catalog drift detected" warning,
`libs/atlas-seeder/catalog.go:42-45`) into a data change; keeping them separate is
correct.

**PQ operations outside a PQ instance** (PRD §9.3). Four Tier-1 scripts call
`update_pq_state`, which returns an error when `getPqInstanceByCharacter` fails —
unlike the `pq_custom_data` *condition*, which explicitly downgrades the same
failure to "not met" (`evaluator.go:101-104`). The asymmetry is pre-existing and
this task does not change it, but the new call sites make it reachable in normal
play. Flagged for a follow-up decision; not fixed here, because changing an
operation's failure semantics is an engine change and this task is data.

**`9018000`–`9018005` semantics** (PRD §9.4). Seeded as explicitly empty per FR-19.
If Cosmic has a default behaviour for a script with no operations, it is not visible
in the script, and nothing in this design pretends otherwise.

**`weaken_area_boss`, `move_environment`, and `kill_all_monsters` are logging
stubs** (`executor.go:259-305`). Seven Tier-1 reactors emit `weaken_area_boss` and
will log rather than weaken anything. This is honest — the seed data becomes correct
ahead of the engine, and the player-facing user story in PRD §3 ("Lich … defeatable
as designed") is not satisfied by this task alone. Worth stating plainly at PR time
rather than discovering it in QA.

**Inventory-as-build-input.** `tier1-inventory.md` lives in the task folder and is
the generator's only input. After merge, re-running the generator requires that
file to still exist. Mitigation: the lint, not the generator, is the durable gate,
and the generator's `--check` is a task-lifetime tool.
