# v95 Skill `common` Formula Nodes — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-07
---

## 1. Overview

`services/atlas-data` derives every skill's per-level effects by enumerating the
`<imgdir name="level">` subtree of each `Skill.wz` skill node
(`skill/reader.go:127`), building one `effect.RestModel` per child
(`reader.go:160-167`), and setting `maxLevel` to the count of those children
(`reader.go:139-144`). GMS v95.1's `Skill.wz` does not encode most skills that
way. It encodes them as a flat `common` subtree holding an explicit `maxLevel`
plus **formula strings over the skill level** — `mpCon = "6+2*u(x/5)"` — which
the reader never reads. There is zero `common` handling anywhere in atlas-data.

The blast radius is measured, not estimated. A full census of the real archive
(MinIO `atlas-wz/shared/regions/GMS/versions/95.1/Skill.wz`, md5
`2d77583108eb928b65f2904196a894ef`, `wz.Open().GameVersion() == 95`) finds 954
player skills: **319 `level`-only, 633 `common`-only, 2 with both, 0 with
neither.** So **635 of 954 skills (66.6%)** currently persist as
`{"maxLevel": 0, "effects": []}` and every consumer — level caps, buff statups,
damage, cooldowns, MP cost — sees nothing for them. Name, description, `action`
and `element` still populate (they come from String.wz and from direct children
of the skill node), which is why the served document looks half-valid instead of
obviously broken.

This is a missing feature, not stale data. Re-ingest alone cannot fix it because
the derivation code does not read the node. The fix is a formula evaluator in
atlas-data that expands a `common` subtree into `maxLevel` synthetic per-level
effects, plus effect-model fields for the `common` keys the model does not carry,
followed by a re-ingest and a serving-pod restart.

`libs/atlas-script-core`'s `EvaluateArithmeticExpression`
(`context/arithmetic.go:12-88`) is **not** reusable. It accepts exactly two
integer operands and one operator — no variables, no function calls, no
parentheses, no precedence, no decimals. Every one of those is required here.

## 2. Goals

Primary goals:

- Parse and evaluate `Skill.wz` `common` subtrees into per-level skill effects,
  so the 635 affected v95 skills serve a correct `maxLevel` and a correct
  `effects` array.
- Model every `common` key the archive actually uses, so no ingested datum is
  silently discarded.
- Make an unevaluable expression a loud, attributable ERROR naming the skill id,
  key, and expression — never a silent empty-effect skill.
- Apply the capability universally (all regions/versions), gated on the presence
  of a `common` node, never on a version predicate.

Non-goals:

- Changing how any downstream service *interprets* effects. The contract of
  every existing field (units, defaults, seconds→ms conversion) is unchanged.
- Extending `common`-style formula handling to other WZ archives (Item.wz,
  Mob.wz, etc.). Scope is `Skill.wz`.
- Fixing which skills belong to which job on v95 — that is task-185.
- Deleting orphaned documents on re-ingest. Ingest remains an idempotent upsert
  (`document/db_storage.go:144-149`); this task does not change that.
- Backfilling the effect model's other never-populated fields (`Berserk`,
  `Booster`, `CardStats`) beyond what `common` keys require.

## 3. User Stories

- As a player on a v95 tenant, I want my skills to have levels and effects so
  that casting them applies buffs and deals damage instead of doing nothing.
- As a service consuming `GET /data/skills/{id}`, I want `maxLevel` to reflect
  the skill's real level cap so that level validation and SP allocation are
  correct.
- As an operator running a WZ ingest, I want an unparseable formula to appear as
  an ERROR log naming the skill and expression, and to be counted in the run
  summary, so that a grammar gap is visible immediately rather than surfacing
  months later as "that skill does nothing."
- As a maintainer bringing up a future client version, I want `common` handling
  to work without a version gate so that a newer archive does not silently
  regress to empty effects.

## 4. Functional Requirements

### FR-1 — `common` detection and precedence

- **FR-1.1** For each skill node, the reader MUST check for a `common` child in
  addition to `level`.
- **FR-1.2** **COMMON wins unconditionally.** When a skill has a `common` node,
  effects are derived from `common` and the `level` subtree is not read. `level`
  is used only when `common` is absent.
- **FR-1.3** When a skill has neither `common` nor `level`, behavior is unchanged
  from today: zero effects, `maxLevel` 0. (0 skills in the v95 archive; the case
  must still not panic.)
- **FR-1.4** Detection MUST be structural (presence of the node), never gated on
  region, major, or minor version.

> **Accepted known regression (FR-1.2).** Exactly two skills in the v95 archive
> carry both subtrees: **2211002** and **2211006** (job 2211, Evan). Their two
> subtrees are *not* alternative encodings of the same data — neither is a
> superset of the other:
>
> | | `common` | `level` |
> |---|---|---|
> | level count | `maxLevel` = 20 | 30 entries (1..30) |
> | keys only here | `damage` (+ `lt`/`rb`/`mobCount` on 2211002) | `mad`, `mastery`, `hs` |
> | 2211002 `mpCon` @ lvl 1 | `20+6*d(1/4)` = 20 | 21 |
> | 2211002 `mpCon` @ lvl 30 | `20+6*d(30/4)` = 62 | 50 |
>
> Applying COMMON-wins means these two skills **lose `mad` and `mastery`** and
> **cap at level 20 instead of 30**. `mad` feeds `SetMagicAttack`
> (`reader.go:205`), so both Evan breath skills will serve magic attack 0. This
> was reviewed against the archive evidence and accepted as the cost of a single
> unconditional rule. It is recorded here so it is not rediscovered as a bug.

### FR-2 — `common` node shape

Derived from an exhaustive walk of all 635 `common` nodes (2 946 leaf children):

- **FR-2.1** `common` is always at `/<jobId>/skill/<skillId>/common` and is
  **exactly one level deep** — zero non-leaf children archive-wide. The reader
  MUST NOT recurse into it.
- **FR-2.2** Leaf node types are **`string`, `int`, and `vector` only**.
- **FR-2.3** `maxLevel` is present on **all 635** nodes and is **never an
  expression**. It is `int` 632× and `string` 3× (`13100004`="10",
  `1320005`="30", `32120001`="30"), always a plain decimal integer. The parser
  MUST accept both node types. Observed values: 1, 5, 10, 15, 20, 25, 30.
- **FR-2.4** `lt` and `rb` are **always `vector`** (226 each) and MUST be passed
  through unevaluated, exactly as the `level` path does (`reader.go:232-235`).
- **FR-2.5** `action` (1 occurrence, `4311003` = `"slashStorm2"`) is an opaque
  client animation name, not an expression, and MUST NOT be evaluated.
- **FR-2.6** Two non-`maxLevel` leaves are `int`-typed rather than `string`:
  `3111003/common/time` = 0 and `33111005/common/x` = 1. The reader MUST accept
  an integer leaf wherever it accepts an expression string.

### FR-3 — Expression grammar and evaluation

The grammar below is **complete for this archive** — all 2 374 string values
under all 635 `common` nodes were tokenized, and the unclassified-character
bucket is empty.

```
expr    := term (op term)*
op      := '+' | '-' | '*' | '/'
term    := ['-'] atom
atom    := number | 'x' | func '(' expr ')'
func    := 'u' | 'd'
number  := [0-9]+ ('.' [0-9]+)?
```

- **FR-3.1** The only variable is **`x`, which is the skill level** (1-based).
  There is no `y`, `level`, or any other free identifier.
- **FR-3.2** Operators are exactly `+ - * /` (occurrences: `*` 1298, `+` 1198,
  `/` 684, `-` 204). Standard precedence MUST be implemented — `* /` bind tighter
  than `+ -`. The data depends on it (`"-1-1*u(x/10)"`).
- **FR-3.3** Functions are exactly two, both arity-1: **`u(...)` = ceiling**,
  **`d(...)` = floor** (`d` 398×, `u` 286×).
- **FR-3.4** **Division MUST be real (floating-point) division, not Go integer
  division.** This is the single most important implementation detail: `u(x/2)`
  at x=1 must be `ceil(0.5)` = **1**, not `ceil(0) = 0`; `d(x/4)` at x=1 must be
  `floor(0.25)` = **0**. Every one of the 684 call sites has argument form
  `x/N` (divisors 2 3 4 5 6 7 8 10 11 15 19), and `/` never appears outside a
  call argument in this archive.
- **FR-3.5** Unary minus MUST be supported, both on a bare literal (`"-2"`) and
  leading an expression (`"-10-1*x"`). 45 occurrences.
- **FR-3.6** One decimal literal appears: `0.5`, in 5 values, all key `t`
  (`"0.5*x"` ×4, `"5+0.5*x"` ×1). Evaluation MUST be performed in `float64`.
- **FR-3.7** Values MUST be `TrimSpace`d before parsing. Exactly one value needs
  it: `2111002/common/damage` = `" 375+5*x"` (leading ASCII space) — the only
  whitespace-bearing value in the archive.
- **FR-3.8** Nested function calls and precedence parentheses **never occur**
  (0 of 2 374; every `(` is immediately preceded by `u` or `d`). The evaluator
  SHOULD still handle them correctly rather than assuming the shape, since the
  grammar admits them and a future archive may use them.
- **FR-3.9** Maximum observed complexity is 4 operators
  (`21001003`/`22141002` key `x` = `"-1-1*u(x/10)"`); longest is 14 chars
  (`11101004` key `range` = `"150+50*u(x/10)"`). These MUST be covered by tests.
- **FR-3.10** The final `float64` result is converted to the target field's
  integer type by **truncation toward zero**, except where the expression's
  outermost operation is already `u()`/`d()`. See OQ-1 for the `t` key.

### FR-4 — The `x` namespace hazard

- **FR-4.1** The key named `x` under `common` (315 occurrences) is a **skill
  parameter**, entirely unrelated to the variable `x` **inside** an expression,
  which is the **skill level**. The evaluator MUST NOT feed `common/x`'s value in
  as the variable binding. This is the most likely subtle implementation bug in
  the task and MUST have a dedicated regression test (e.g. `1101004` where
  `common/x = "-2"` while its sibling keys evaluate over level).

### FR-5 — Per-level expansion

- **FR-5.1** A `common` node expands to exactly `maxLevel` effects, evaluated at
  `x = 1, 2, … maxLevel`, in ascending level order — matching the document order
  the `level` path produces (`reader.go:160-167`).
- **FR-5.2** `RestModel.MaxLevel` for a `common` skill MUST be the declared
  `common/maxLevel`, not a derived count. (The existing `len(es)` clamp at
  `reader.go:139-144` yields the same number, but the declared value is
  authoritative.)
- **FR-5.3** Every derived effect MUST go through the **same** post-processing
  the `level` path applies, so the two paths are behaviorally identical for a
  key present in both. Specifically, and non-exhaustively:
  - `time` seconds→milliseconds conversion and the `OverTime` rule
    (`reader.go:196-201`);
  - `prop` and `hpR`/`mpR` divide-by-100 (`reader.go:173-178`);
  - the statup derivation block (`reader.go:213-230`), including the
    `produceBuffStatAmount` drop-zero behavior;
  - the per-skill statup / monsterStatus chain (`reader.go:251-420`);
  - `applyMobInformation`, `getMob`, `getAbnormalStatuses`, `getMapProtection`.
- **FR-5.4** Defaults for keys **absent** from a `common` node MUST match the
  `level` path's defaults exactly — notably `damage` 100, `mobCount` 1,
  `attackCount` 1, `bulletCount` 1, `fixdamage` -1, `moveTo` -1, `time` -1, and
  0 for the rest (`reader.go:169-247`).

### FR-6 — Effect model extension

Of the 65 distinct `common` keys, **28 already have an effect-model field** and
are populated by the `level` path today:

```
time hp mp hpCon mpCon prop mobCount cooltime morph pad pdd mad mdd acc eva
speed jump lt rb x y damage attackCount bulletCount bulletConsume moneyCon
itemCon itemConNo
```

`maxLevel` and `action` are skill-level, not per-effect. The remaining **35 keys
have no field anywhere in `effect.RestModel`** and MUST each be modeled
explicitly (typed field + `json` attribute + builder setter), with occurrence
counts:

| key | n | key | n | key | n |
|---|---|---|---|---|---|
| `range` | 46 | `mastery` | 32 | `z` | 29 |
| `dot` | 20 | `cr` | 20 | `dotInterval` | 20 |
| `dotTime` | 20 | `damR` | 17 | `criticaldamageMin` | 15 |
| `mhpR` | 13 | `v` | 12 | `ignoreMobpdpR` | 10 |
| `epad` | 10 | `w` | 9 | `u` | 9 |
| `epdd` | 8 | `emdd` | 8 | `selfDestruction` | 7 |
| `asrR` | 6 | `mmpR` | 5 | `t` | 5 |
| `er` | 5 | `pddR` | 5 | `terR` | 5 |
| `madX` | 4 | `subProp` | 4 | `emhp` | 3 |
| `criticaldamageMax` | 3 | `expR` | 3 | `emmp` | 3 |
| `itemConsume` | 2 | `mddR` | 2 | `subTime` | 1 |
| `padX` | 1 | `mesoR` | 1 | | |

- **FR-6.1** Each of the 35 keys above gets a typed field on
  `effect.RestModel` (`skill/effect/rest.go`), a matching private field +
  getter + builder setter on `effect.Model` (`skill/effect/model.go`), and is
  populated on **both** the `common` and `level` read paths where the key can
  occur.
- **FR-6.2** `damage` is **already modeled** — `effect/rest.go:51`
  (`Damage uint32`), read at `reader.go:239` with default 100. It requires no new
  field; it requires only that the `common` path populate it. It is called out
  explicitly because it is the highest-frequency post-BB percent key (277
  occurrences) and its absence is the most visible symptom.
- **FR-6.3** `mhpR` and `mmpR` SHOULD populate the existing but currently
  never-populated `MHPRRate` / `MMPRRate` fields (`effect/rest.go:20-21`) rather
  than adding duplicates, if the design phase confirms the semantics match.
- **FR-6.4** `itemConsume` (2 occurrences, values `2331000`/`2331001`) is a
  **distinct key** from `itemCon` (8 occurrences) and must not be folded into the
  existing `ItemConsume` field, which is fed from `itemCon` (`reader.go:245`).
  The design phase MUST resolve the naming collision explicitly.
- **FR-6.5** New fields are additive. No existing field is renamed, retyped, or
  removed — every current consumer of `GET /data/skills` keeps working
  unchanged.

### FR-7 — Failure handling

- **FR-7.1** A `common` value that fails to tokenize, parse, or evaluate MUST
  emit an **ERROR** log carrying, at minimum: tenant, skill id, job image id,
  key name, and the verbatim expression string.
- **FR-7.2** The failure is scoped to **that skill only**. It MUST NOT abort the
  job image (today's `reader.go:73-76` behavior) and MUST NOT abort the ingest
  run. The affected skill is persisted with whatever it successfully derived.
- **FR-7.3** The SKILL worker MUST emit a **run summary** at completion counting
  skills processed, skills derived from `common`, skills derived from `level`,
  and **skills with ≥1 evaluation failure**. A non-zero failure count MUST be
  logged at ERROR so the run is not mistaken for clean. This is what makes
  FR-7.2's permissiveness safe: a failure is visible in aggregate even if a
  single ERROR line scrolls past.
- **FR-7.4** A missing or non-integer `common/maxLevel` is an evaluation failure
  under FR-7.1 for the whole skill (no level count ⇒ no expansion possible).
- **FR-7.5** No `common` parse failure may be swallowed by
  `xml.GetIntegerWithDefault`'s silent-default behavior (`xml/model.go:82-102`),
  which is precisely how a formula string currently degrades to 0 without a
  trace. The `common` path MUST NOT route expression values through it.

### FR-8 — Evaluator placement

- **FR-8.1** The evaluator lives in a **new package inside atlas-data** (e.g.
  `services/atlas-data/atlas.com/data/skill/formula`). It is not a new shared
  lib (no second consumer exists) and does not extend
  `libs/atlas-script-core` (whose only consumer is atlas-npc-conversations and
  whose grammar is unrelated).
- **FR-8.2** The evaluator MUST be pure and free of tenant/context dependencies:
  `Evaluate(expr string, level int) (float64, error)` or equivalent, so it is
  unit-testable in isolation.
- **FR-8.3** Parsing SHOULD be separable from evaluation so a single expression
  is parsed once and evaluated `maxLevel` times, rather than re-parsed per level.

## 5. API Surface

No new or removed endpoints. The two existing read endpoints
(`skill/resource.go:19-29`) are unchanged in shape:

- `GET /data/skills` — search by `ids=` or `name=`, paginated
  (`skill/resource.go:31-98`)
- `GET /data/skills/{skillId}` — single skill (`skill/resource.go:100-118`)

Response changes, both additive:

- `maxLevel` becomes non-zero for the 635 affected v95 skills, and `effects`
  becomes a non-empty array of length `maxLevel`.
- Each `effects[]` object gains the 35 new attributes from FR-6. Existing
  attributes are unchanged in name, type, and units.

JSON:API resource type remains `"skills"` (`skill/rest.go:19-21`).

No new error cases are surfaced over HTTP — FR-7 failures occur at ingest time
and are reported through logs, not the read API.

## 6. Data Model

Skills are **not** stored in a per-skill relational table. They are persisted as
opaque JSON:API documents:

- Table `documents` (`document/entity.go:15-26`): `id, tenant_id, type,
  document_id, content json, updated_at`, unique index
  `(tenant_id, type, document_id)`.
- Document type is `"SKILL"` (`skill/processor.go:37-39`).
- Write is an upsert on conflict of `(tenant_id, type, document_id)` updating
  `content, updated_at` (`document/db_storage.go:144-149`).

Consequences for this task:

- **No schema migration is required.** The new effect attributes live inside the
  existing `content json` column.
- The change takes effect only when the `content` blob is rewritten — i.e. on
  re-ingest (FR-9). There is no in-place backfill path.
- Tenant scoping is already enforced by `tenant_id` on every row; the new fields
  inherit it with no additional work.
- The in-memory registry mirror (`skill/registry.go:13-17`,
  `document/storage.go:128-141`) is repopulated on write, but the ingest Job pod
  is a **separate process** from the serving pods — hence the restart in FR-9.

## 7. Service Impact

**`services/atlas-data`** — the only service with code changes.

| Area | Change |
|---|---|
| `skill/formula/` (new pkg) | Tokenizer, parser, evaluator for the FR-3 grammar |
| `skill/reader.go` | `produceSkill` gains `common` detection + precedence (FR-1); a `common`→effects expansion path (FR-5); `maxLevel` from the declared value (FR-5.2); refactor so `getEffect`'s post-processing (`:169-426`) is shared by both paths (FR-5.3) |
| `skill/effect/model.go`, `effect/rest.go` | 35 new fields + getters + builder setters (FR-6) |
| `data/workers/skill.go` | Run-summary counters and the ERROR-on-nonzero-failures line (FR-7.3) |
| `skill/reader_test.go` | Grammar unit tests, expansion tests, the FR-4 `x`-namespace regression test, and the FR-1.2 dual-node test |

**No other Go service changes.** Consumers read the new fields only if they
choose to; `atlas-channel`'s `character_cash_item_use_point_reset.go:121` uses
`len(ds.Effects())` as a level cap and will begin returning the correct cap for
v95 skills as a direct consequence — that is the intended fix, not a regression.

**`libs/atlas-script-core`** — untouched. Its
`EvaluateArithmeticExpression` is explicitly not reused (FR-8.1).

**Operational impact** — the v95 tenant requires a data step (FR-9). Other
tenants are unaffected at runtime: v92's `Skill.wz` has 931 `level`-only skills
and **0** `common` nodes, so the new path is inert there. Applying the change
universally (FR-1.4) costs nothing on those versions and prevents a future
version from silently regressing.

## 8. Non-Functional Requirements

- **NFR-1 (Performance).** Expansion adds ~635 × `maxLevel` (≈ 12 700) effect
  constructions and ~1 829 expression parses per v95 ingest. Parse-once /
  evaluate-per-level (FR-8.3) keeps this negligible against the existing WZ
  serialization cost. The SKILL worker's wall time MUST NOT regress by more than
  a small constant; ingest already runs under `INGEST_MAX_PARALLEL` bounded
  concurrency (`data/runwz.go:48`).
- **NFR-2 (Multi-tenancy).** Unchanged. Derivation is per-archive and the
  resulting documents are written tenant-scoped through the existing path. The
  evaluator itself is tenant-agnostic (FR-8.2).
- **NFR-3 (Observability).** FR-7's ERROR lines and run summary are the required
  signal. Log fields MUST be structured enough to grep by skill id.
- **NFR-4 (Determinism).** Evaluation MUST be deterministic — same archive in,
  byte-identical documents out. No map-iteration order may leak into the
  serialized `effects` array.
- **NFR-5 (Security).** The evaluator parses a fixed, tiny grammar from a
  trusted operator-supplied archive. It MUST NOT be a general expression
  interpreter, MUST NOT support identifiers beyond `x`, and MUST be bounded
  against pathological input (expression length / recursion depth) so a
  malformed archive cannot hang or stack-overflow the ingest job.
- **NFR-6 (Testing).** Grammar coverage MUST include, as explicit cases: real
  division (FR-3.4), unary minus on a bare literal and leading an expression,
  the `0.5` decimal, the leading-space `damage` value, both int-typed non-
  `maxLevel` leaves, the string-typed `maxLevel` values, and the 4-operator
  maximum-complexity expression.

## 9. Open Questions

- **OQ-1 — Final rounding for `t`.** FR-3.10 specifies truncation toward zero
  for a result not already wrapped in `u()`/`d()`. The only key where this is
  observable is `t` (5 occurrences, `"0.5*x"` / `"5+0.5*x"`), which at odd levels
  yields a `.5` fraction. The archive does not reveal the client's rounding, and
  `t` is one of the 35 newly-modeled keys with no current consumer. Truncation is
  the assumed default; if the design phase can source the client's behavior from
  the v95 IDB, it should.
- **OQ-2 — `itemConsume` vs `itemCon` naming (FR-6.4).** Whether `common`'s
  `itemConsume` is semantically the same field as `itemCon` under a different
  name, or a distinct concept, is unresolved. Both appear only in `common`
  (`itemCon` also appears under `level`). 2 occurrences.
- **OQ-3 — `mhpR`/`mmpR` → `MHPRRate`/`MMPRRate` (FR-6.3).** The existing fields
  are declared but never populated by any reader, so their intended semantics are
  undocumented. Confirm before reusing rather than adding new fields.
- **OQ-4 — Semantics of the low-frequency single-letter keys.** `u` (9), `v`
  (12), `w` (9), `z` (29), `t` (5) are modeled as opaque integers by FR-6. Their
  meaning is unknown and is not required to be resolved for this task — carrying
  the value correctly is sufficient. Named here so it is not mistaken for an
  oversight.
- **OQ-5 — Whether other regions' archives contain `common` nodes.** Only GMS
  v95.1 and v92 were censused (635 and 0 respectively). JMS v185 in particular is
  post-Big-Bang and plausibly uses `common`, which would mean this task fixes it
  too. Worth a census during design; does not block, since FR-1.4 makes the
  handling universal either way.

## 10. Acceptance Criteria

**Evaluator correctness**

- [ ] The FR-3 grammar is fully implemented, including `* /` precedence over
      `+ -`, unary minus, the `0.5` decimal, and arity-1 `u`/`d`.
- [ ] Division is real, not integer: `u(x/2)` at x=1 → 1; `d(x/4)` at x=1 → 0.
      Covered by an explicit unit test.
- [ ] `" 375+5*x"` (leading space, skill 2111002 `damage`) parses.
- [ ] `"-1-1*u(x/10)"` (maximum observed complexity) evaluates correctly at
      x=1 and x=20.
- [ ] `common/x` is never bound as the expression variable `x` — regression test
      on skill 1101004 (FR-4.1).

**Expansion and precedence**

- [ ] A skill with a `common` node produces exactly `common/maxLevel` effects,
      evaluated at x = 1..maxLevel.
- [ ] `maxLevel` is taken from the declared value, accepting both `int` and
      `string` node types (verified against 13100004, 1320005, 32120001).
- [ ] `lt`/`rb` vectors pass through unevaluated; `action` is not evaluated.
- [ ] A skill with only `level` behaves **byte-identically** to today — proven by
      a golden test over at least one v92 job image, whose serialized documents
      must be unchanged.
- [ ] Skills 2211002 and 2211006 derive from `common`, yielding `maxLevel` 20;
      the documented loss of `mad`/`mastery` is asserted by test so the tradeoff
      is pinned rather than accidental.

**Model coverage**

- [ ] All 35 previously-unmodeled `common` keys have a typed field, a `json`
      attribute, and a builder setter, and are populated from `common`.
- [ ] `damage` is populated from `common` (277 occurrences) — the flagship fix.
- [ ] No existing `effect.RestModel` field is renamed, retyped, or removed.

**Failure handling**

- [ ] An unparseable expression produces an ERROR log naming tenant, skill id,
      key, and verbatim expression, and does **not** abort the job image or the
      run.
- [ ] The SKILL worker logs a run summary with processed / `common` / `level` /
      failure counts, at ERROR when failures > 0.
- [ ] No `common` expression value is read through
      `xml.GetIntegerWithDefault` (FR-7.5).

**End-to-end, on the v95 tenant**

- [ ] Re-ingest completes with **0** evaluation failures against the real v95.1
      `Skill.wz`. A non-zero count blocks acceptance — the grammar is claimed
      complete, so any failure is a real gap.
- [ ] After re-ingest and a serving-pod restart,
      `GET /api/data/skills/1001003` returns `maxLevel: 20` with 20 effects,
      where level 1 has `MPConsume` = 8 (`6+2*u(1/5)` = 6+2·1) and
      `duration` = 110000 (`100+10*1` = 110 s → ms), and level 20 has
      `MPConsume` = 14 (`6+2*u(20/5)` = 6+2·4) and `duration` = 300000.
- [ ] A census of `GET /data/skills` on the v95 tenant shows **0 of 954** skills
      with `maxLevel: 0` and empty `effects` (down from 635).
- [ ] The 9 non-v95 tenants show **no diff** in their served skill documents
      before vs. after, confirming FR-1.4's universal handling is inert where no
      `common` node exists.

**Build & verification gates** (per `CLAUDE.md`)

- [ ] `go test -race ./...` clean in `services/atlas-data`.
- [ ] `go vet ./...` clean in `services/atlas-data`.
- [ ] `go build ./...` clean in `services/atlas-data`.
- [ ] `tools/lint.sh --check` clean from the repo root.
- [ ] `tools/goroutine-guard.sh` and `tools/redis-key-guard.sh` clean.
- [ ] `docker buildx bake atlas-data` succeeds **if** `go.mod` was touched.
- [ ] Code review run before the PR is opened.

## 11. Rollout Runbook

The code change alone changes nothing that is served — persisted documents are
only rewritten by an ingest pass.

1. Merge and deploy the atlas-data image.
2. Trigger a re-ingest for the v95.1 archive: `POST /data/process` with the
   appropriate scope (`runtime/rest/resource.go:31-74`; `shared` scope requires
   the `X-Atlas-Operator: 1` header). This creates a Kubernetes Job.
3. Watch the job to completion (`GET /data/process`) and **read the run summary**
   (FR-7.3). A non-zero failure count means the grammar is incomplete — stop and
   investigate before proceeding.
4. Restart the atlas-data **serving** pods. The ingest Job pod is a separate
   process; its in-memory registry mirror is not the one serving traffic.
5. Verify with the two end-to-end acceptance checks above (skill 1001003 values,
   and the 0-of-954 census).

Note that re-ingest is an idempotent upsert, **not** a truncate
(`document/db_storage.go:144-149`) — documents whose ids no longer appear are not
deleted. That is pre-existing behavior and out of scope here, but it means this
runbook is safe to re-run.

## 12. Evidence Provenance

Every count, key, operator, and value in this document comes from a full-archive
census of the real object, not from sampling or from general MapleStory
knowledge:

- Source: MinIO bucket `atlas-wz`, key
  `shared/regions/GMS/versions/95.1/Skill.wz` (163.7 MB, md5
  `2d77583108eb928b65f2904196a894ef`).
- Parsed with `libs/atlas-wz` `wz.Open`; `File.GameVersion()` reports **95**,
  confirming the archive is not a substituted version.
- 118 root images walked recursively; **635** `common` nodes found, **all** at
  `/<jobId>/skill/<skillId>/common` and nowhere else.
- **All 2 374** string values under those nodes were tokenized — not sampled —
  and the tokenizer's unclassified-character bucket is empty, which is what
  licenses the claim that the FR-3 grammar is complete.
- The v92 comparison figure (931 `level`-only, 0 `common`) comes from the same
  method applied to that version's archive.
