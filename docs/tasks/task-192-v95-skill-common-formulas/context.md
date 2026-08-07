# task-192 — Implementation Context

Companion to [`plan.md`](plan.md). Inputs: [`prd.md`](prd.md), [`design.md`](design.md),
[`wz-common-grammar.md`](wz-common-grammar.md).

---

## 1. The one-paragraph problem

`atlas-data` derives every skill's per-level effects by enumerating the
`<imgdir name="level">` children of a skill node. GMS v95.1's `Skill.wz` encodes
most skills as a flat `common` subtree instead — an explicit `maxLevel` plus
formula strings over the skill level (`mpCon = "6+2*u(x/5)"`). No code anywhere
in atlas-data reads that node, so **635 of 954 v95 skills persist as
`{"maxLevel": 0, "effects": []}`**. Re-ingest cannot fix it; the derivation code
has to learn the node.

---

## 2. Key files, and what each one actually does

All paths are worktree-relative. Go module root for every `go` command:
`services/atlas-data/atlas.com/data`.

| File | Why it matters |
|---|---|
| `skill/reader.go:45` `Read` | Curried `l → ctx → nodeProvider → provider`. Reads one job image (`112.img`), iterates `skill/<skillId>` children, calls `produceSkill`. Task 5 changes its return type to `Derivation`. |
| `skill/reader.go:106` `produceSkill` | Derives element/action/buff, then effects. **Line 127-130 is the whole bug**: it only ever looks for `level`. Task 4 inserts the `common` branch ahead of it. |
| `skill/reader.go:169` `getEffect` | ~260 lines of post-processing: seconds→ms with the `OverTime` rule (196-201), `prop`/`hpR`/`mpR` ÷100, the statup block (213-230), the 40-arm per-skill statup / monsterStatus chain (251-420), `lt`/`rb`, mob info, abnormal statuses. **This function is not modified by the `common` path** — only Task 3's block of new key reads is added. That is the design's load-bearing decision. |
| `skill/reader.go:489` `levelFromNode` | Parses the effect node's *name* as its level. Synthesized nodes must therefore be named `strconv.Itoa(level)` or the per-level SpaceShip vehicle id breaks. |
| `xml/model.go:82` `GetIntegerWithDefault` | Looks in `IntegerNodes` then `StringNodes`, `ParseInt`, **silently returns the default on failure**. This is exactly how a formula string degrades to `0` today. FR-7.5 exists because of this function. |
| `xml/model.go:9` `Node` | `ChildNodes`, `IntegerNodes`, `StringNodes`, `PointNodes`, `DoubleNodes`, `CanvasNodes`. Synthesis builds one of these per level with only `Name`, `IntegerNodes` and `PointNodes` populated. |
| `skill/effect/rest.go` | The served JSON shape. 63 lines; append-only in this task. |
| `skill/effect/model.go` | Builder: private fields → `Set…` chain → `Build()` literal. Three edits per new field (field, setter, `Build()` line). |
| `skill/processor.go:55` `RegisterSkill` | Wraps `Read` in a DB transaction. One of two ingest entry points. |
| `data/workers/skill.go:50` | The current SKILL worker entry point. `countingRegister`/`logJobDocCount` (lines 113-137) are the **existing precedent** the stats plumbing mirrors. |
| `data/processor.go:153` | The *second*, legacy ingest entry point for `WorkerSkill`. Easy to miss; the plan wires the summary here too. |
| `data/workers/walk.go:29` | `RegisterFunc = func(path string) error` — the shape `StatsAccumulator.Wrap` adapts to. |
| `atlas-ui/src/lib/skills/level-table.ts:19` | `FIELD_LABELS` is an **explicit whitelist**. A served attribute not listed here renders nowhere. |
| `atlas-ui/src/services/api/skills.service.ts:14` | `SkillEffect` is a closed interface; a `FIELD_LABELS` key not declared here is a TS error. The two files must be edited together. |

---

## 3. Decisions already made (do not re-litigate)

- **COMMON wins unconditionally** (FR-1.2). Skills **2211002** and **2211006** carry
  both subtrees and are not supersets of each other; applying the rule costs them
  `mad` and `mastery` and caps them at level 20 instead of 30. Reviewed against the
  archive and accepted. Pinned by test so it stays a decision, not a future bug report.
- **Synthesize `xml.Node`s rather than refactoring `getEffect`** (design §2.1 option C).
  Duplicating `getEffect` guarantees drift; abstracting it behind a value-source
  interface touches ~60 read sites on the path this task must not regress. `xml.Node`
  already *is* that interface.
- **New keys are read once, in `getEffect`** — which is why they populate on *both*
  paths for free (FR-6.1).
- **Field naming: PascalCase of the wz key, JSON tag verbatim.** No invented
  descriptive names. The meanings of `z`/`u`/`v`/`w`/`t` and most of the abbreviations
  are unverified (OQ-4); a guessed name becomes permanent and load-bearing.
- **`itemConsume` (common) ≠ `itemCon` (level).** The existing `ItemConsume` field
  already *carries the JSON tag* `itemConsume` while being fed from `itemCon`, so the
  new key cannot use its own name. It becomes `ConsumeItemId` / `consumeItemId`.
  Whether the two are semantically the same key renamed post-Big-Bang is **unresolved**;
  they never co-occur, so merging later is safe and non-breaking.
- **`mhpR`/`mmpR` reuse the existing `MHPRRate`/`MMPRRate` fields** (never populated by
  any reader today, so nothing observes a change).
- **No version gate, ever** (FR-1.4). v92 has 931 `level`-only skills and **0** `common`
  nodes, so the new path is inert there at zero cost.

---

## 4. The client semantics — the single highest-risk area

Source: IDA session `79906a1e`, `GMS_v95.0_U_DEVM.exe.i64` (PDB-backed).
`SKILLLEVELDATA::GetParsedCommonData` at `0x6fe560`,
`SKILLLEVELDATA::GetArithmeticData` at `0x6f9300`.

The client is a **string-rewriting machine**, not an AST evaluator. Four consequences,
all of which the current archive **cannot** distinguish from the textbook version — which
is exactly why they are easy to get wrong and hard to notice:

1. **Precedence is `+` → `-` → `/` → `*`** (loosest to tightest), one operator per pass.
   Observable at `x/2*3`: client gives `20/(2*3) = 3`; left-to-right gives `30`. The
   archive never mixes `/` and `*` at the same level (every `/` is inside a `u(`/`d(`
   argument of the form `x/N`), so the two agree today. That is a property of the data,
   not of the grammar.
2. **Every binary operation truncates toward zero** before the next consumes it
   (the client `Format`s each intermediate with `"%d"`).
3. **`d(v) = trunc(v)`, `u(v) = trunc(v + 0.999999)`** — not `math.Floor`/`math.Ceil`.
   They agree for `v ≥ 0`; every argument in the archive is `x/N` with `x ≥ 1`, so no
   negative reaches either function today.
4. **The `u()` ceiling replaces the argument's outermost truncation**, it is not applied
   on top of it. The client passes `bCeiling` *into* the arithmetic pass that spans the
   whole argument. This is why `u(x/2)` at `x=1` is **1**. A "truncate the argument, then
   ceil" implementation gives **0** and fails FR-3.4.

Point 4 is a correction to `design.md` §3.2 as literally written — see plan Deviation 2.
The plan's `callNode`/`binNode` `ceiling` parameter is the mechanism; `TestEvaluate`'s
`ceil half at level 1` and `star binds tighter than slash` cases fail loudly if either is
"cleaned up" later. The package doc comment carries the IDB citation for the same reason.

**Deliberate divergence:** the client mis-slices nested calls (`u(d(x/2))` — each rewrite
loop takes the first `(` and the first following `)`). The plan's recursive-descent parser
handles nesting correctly. The archive contains zero nesting, so this is a superset on
input that does not exist, and the client's behaviour there is a bug rather than a
contract.

---

## 5. Deviations from `design.md` decided while planning

1. **`Expr.EvaluateFloat` is not implemented.** Per-operation truncation is intrinsic to
   the client's machine, so a float-faithful evaluator is a *second* semantics (the
   client's `GetParsedCommonDataFloat` sibling at `0x6fd950`), not a `float64` variant of
   this one. Shipping it now would be dead code under a misleading name — and the project
   forbids landing stubs. If a consumer ever needs sub-second `t`, that is a small,
   honest, separate implementation.
2. **Design §3.2's rounding rule is restated**, per §4 point 4 above.
3. **The run summary is emitted at both ingest entry points**, not only the SKILL worker.
   `data/processor.go:153` is a second `RegisterSkill` call site; wiring only the worker
   would silently drop the summary on that path. Hence `StatsAccumulator` (mutex-guarded,
   since it is shared across whatever concurrency the walkers use) rather than a
   worker-local counter.

---

## 6. Dependencies and ordering

```
Task 1 (formula)  ──┐
Task 2 (model)  ────┼──> Task 3 (getEffect reads) ──> Task 4 (common expansion) ──> Task 5 (stats plumbing)
                    │                                                                      │
                    └──> Task 6 (atlas-ui, needs Task 2's JSON tags) <─────────────────────┘
Task 7 (census + corpus) needs Task 1 only, but its corpus test should land after Task 4.
Task 8 (gates + e2e + review) is last.
```

Task 3's test is written against today's `Read` shape and converted by Task 5 — the plan
spells out both forms so a subagent seeing only one task is not blocked.

---

## 7. External dependencies

- **The archives live in MinIO**, not the repo: bucket `atlas-wz`, key
  `shared/regions/<REGION>/versions/<VERSION>/Skill.wz`. GMS 95.1 is 163.7 MB, md5
  `2d77583108eb928b65f2904196a894ef`, and `wz.Open().GameVersion()` reports `95`.
  Reached via `kubectl -n minio port-forward svc/minio 19000:9000` plus
  `curl --aws-sigv4`, credentials in Secret `atlas-minio-credentials` (ns `atlas-main`).
- **Analysis programs are scratch, not committed** — a standalone module with `replace`
  directives to the repo libs, following the precedent recorded in
  `wz-common-grammar.md`. The committed artifacts are `archive-census.md` and
  `skill/formula/testdata/common_corpus.csv`.
- **IDA**: resolve the session from `idb_list` by binary **name** (`GMS_v95.0_U_DEVM.exe`)
  and pass it as the `database` parameter — port-based `select_instance` has been dead
  since task-138.
- **atlas-ui needs Node 22** (`nvm use 22`); `tools/lint.sh --check` false-fails without
  it. `npm run build` — not `vitest` alone — is the atlas-ui verification, because the
  build is what type-checks the test files.

---

## 8. Verification quick reference

From `services/atlas-data/atlas.com/data`: `go build ./...`, `go vet ./...`,
`go test -race ./...`.
From the worktree root: `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`,
`tools/lint.sh --check`.
From `services/atlas-ui`: `npm run build`.
`docker buildx bake atlas-data` **only if `go.mod` changed** — it should not; the
evaluator adds no dependency.

No template guards apply (no socket-config template is touched). No schema migration:
skills are opaque JSON:API documents in the `documents` table
(`document/entity.go:15-26`), upserted on `(tenant_id, type, document_id)`
(`document/db_storage.go:144-149`).

**The change serves nothing until a re-ingest rewrites the `content` blobs, and the
ingest Job pod is a different process from the serving pods — the serving pods need a
restart afterwards.** That is the most common way this kind of change looks "broken"
after deploy.
