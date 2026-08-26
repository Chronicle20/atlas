# WZ Property Reader Archive-Byte Fidelity — Product Requirements Document

Version: v2
Status: Draft — re-scoped after oracle withdrawal
Created: 2026-08-25
Revised: 2026-08-26 — see `provenance.md` and `reference-fidelity.md`
Issue: https://github.com/Chronicle20/atlas/issues/1496
---

> **Provenance correction (read this first).** This PRD was originally written
> against a HaRepacker `.img.xml` dump of `GMS/83.1/Reactor.wz`, treated below
> as an independent structural oracle. Task 5's byte-level adjudication
> (`reference-fidelity.md`) found that dump was never exported from the
> archive supplied for this task: three independent counts of `Npc.wz`'s
> images (our reader, a raw `declaredCount` byte read, and HaRepacker run by
> the user on the archive itself) agree on **1620**, while the dump holds
> **6962**; the dump's `Reactor.wz` entries include two images (`9400300`,
> `9400301`) absent from the archive's own directory. The full evidence is
> `provenance.md`. **The archive's own bytes are now the authority.** Every
> requirement below that compared our parse against the dump is marked
> **WITHDRAWN** in place, with a pointer to the finding; the text is kept for
> the historical record, not as a live requirement. §10 has the current
> acceptance criteria.

## 1. Overview

`libs/atlas-wz/wz` is the single WZ binary reader for the entire Atlas server. Every
domain's game data flows through it: `services/atlas-data` opens the tenant's `.wz`
archives, walks `Directory.Images()`, calls `Image.Properties()`, serializes the result
through `data/wztoxml`, and hands the XML to the per-domain readers. If the property
reader mis-decodes an image, the corruption is silent — the reader returns a shorter
(or differently-shaped) property tree with no error, and the domain reader simply
imports whatever it was given.

**[WITHDRAWN — see the provenance correction above.]** That is what a whole-archive
structural diff originally appeared to show. Serializing all 419 images our parser
enumerates from `GMS/83.1/Reactor.wz` (PKG1, 51.6 MiB) with the ingest pipeline's own
`wztoxml.SerializeToDirectory`, and structurally diffing each against a HaRepacker XML
dump believed at the time to be of the same archive file, showed 400 images structurally
identical and **19 divergent**, in shapes that looked like a parser defect: whole
subtrees lost, a subtree gained that the dump did not have, a property *name* mangled
(`repeat` → `rpeat`), scalar values wrong (`state` 0 read as 1), canvases decoded as 1×1
where the dump showed 100×121, and 419 images enumerated where the dump held 421
(`9400300.img` and `9400301.img` never seen).

**That diagnosis does not hold.** Task 5's byte-level adjudication
(`reference-fidelity.md`) found the dump was never exported from this archive — the
archive's own bytes vindicate our reader on all 19 originally-disputed images and both
un-enumerated ones (`INPUT-MISMATCH`, 21/21; `PARSER-DEFECT` 0). `provenance.md` records
the independent count agreement establishing this. The paragraph above and the next one
are retained as the record of what was observed and believed at the time; they are not a
live parser-defect list.

This was found while implementing touch-activated reactors (task-249, #1459). Reactor
`2406000` is a touch reactor in the WZ, and our parse was originally believed to drop
both `info/activateByTouch` and the entire state-0 `event` subtree, so the server could
never treat it as touch-activated. Reactor.wz is only where it surfaced — the same
reader feeds Map.wz, Mob.wz, Item.wz, Skill.wz and the rest, so the same class of silent
corruption was thought likely present across every domain and unswept for.

**Correction:** `reference-fidelity.md`'s adjudication of `2406000` found the supplied
archive genuinely does not contain `info/activateByTouch` or the state-0 `event`
subtree in this image — the reader is not dropping them; they were never in these
bytes. Whether task-249's touch-reactor work needs a different tenant archive or a
different reactor ID is outside this task's scope; see `provenance.md`.

The evidence file for the full Reactor.wz diff is committed alongside this PRD as
`evidence-wz-parse-divergence-reactor.txt`. It remains useful evidence about two
unrelated datasets (our archive's parse, and the unidentified dataset the dump came
from) — see `provenance.md`.

## 2. Goals

Primary goals:

- **[WITHDRAWN]** ~~Root-cause the divergence(s) in `libs/atlas-wz/wz` down to the
  specific decode step(s), with the failing byte sequence identified — not a
  plausible-looking patch.~~ Task 5 found none of the 19 divergences is a decode defect
  (`reference-fidelity.md`, 21/21 `INPUT-MISMATCH`); there is no root cause to find.
- **[WITHDRAWN]** ~~Make our parse of all 19 divergent Reactor.wz images structurally
  identical to HaRepacker's, in both directions.~~ Same reason — every disputed image is
  a faithful read of a different archive than the one the dump came from.
- **[WITHDRAWN]** ~~Make the parser enumerate `9400300.img` and `9400301.img` (421
  images, not 419).~~ Both images are absent from the supplied archive's own directory
  (`reference-fidelity.md`); there is nothing to enumerate.
- Land a permanent, CI-runnable **self-consistency** guard, built from `wztest`
  fixtures, that asserts every type-9 sub-object consumes its declared size exactly and
  no bytes are orphaned or double-read — no external reference archive required. See
  `plan.md` Task R2.
- Cause no regression in any existing `libs/atlas-wz` consumer (`mapimage`, `icons`,
  `charparts`, `atlas` packer, `atlas-data` domain readers).

Non-goals:

- Sweeping the other archives (Map.wz, Mob.wz, Item.wz, Skill.wz, …) for the same class of
  divergence. Fix first; the blast-radius sweep is a follow-up task, seeded by §9.
- Re-ingesting or refreshing any tenant's already-imported data. The deliverable is a
  correct parser; making a running environment pick up corrected data is operational.
- Implementing reactor touch-activation behavior. This task stops at "the correct
  properties parse"; task-249 owns the behavior.
- Changing the canvas *pixel* decoders (`libs/atlas-wz/canvas`), the WZ writer
  (`libs/atlas-wz/atlas`), or `wztoxml`'s element mapping. `propertyToElement` in
  `services/atlas-data/atlas.com/data/data/wztoxml/adapter.go` has a case for every
  `property.*` type and no drop path; the loss is upstream in `Image.Properties()`.
- Any UI or API surface change.

## 3. User Stories

- As a server operator, I want WZ-sourced game data to match the archive I shipped, so
  that content behaves the way the client's own data says it should.
- As an Atlas engineer implementing a WZ-backed feature, I want `Image.Properties()` to
  either return the full property tree or return an error, so that a missing field means
  "the archive doesn't have it," not "our parser lost it."
- As the engineer on task-249, I want `2406000`'s `info/activateByTouch` and state-0
  `event` subtree to parse, so that touch-reactor work can be built on real data.
- As a future maintainer of `libs/atlas-wz`, I want a test that fails loudly if a decode
  change drops, gains, renames, or mis-values a property, so that I find out in CI rather
  than in a bug report six months later.

## 4. Functional Requirements

> **[WITHDRAWN, this section]** FR-1 through FR-7 and FR-9 through FR-14 below all
> assume the HaRepacker dump is an independent, same-archive oracle. That assumption is
> false (`provenance.md`, `reference-fidelity.md`). Each is marked **WITHDRAWN** in
> place with a one-line reason; the text is kept as the historical record of what was
> required under the false premise. **FR-8 is the exception and survives** — it is
> restated, not withdrawn, and is now the whole of the live requirement, carried forward
> as Task R2's self-consistency gate (see `plan.md`).

### 4.1 Root cause (required before any fix lands) — **WITHDRAWN, all of §4.1**

There is no root cause: Task 5 found zero `PARSER-DEFECT` images. FR-1/FR-2/FR-3 asked
for a diagnosis of defects that do not exist.

- **FR-1.** Diagnose the divergence from the bytes, not from inspection alone. For at
  least the cleanest failing image (`2406000.img` — pure loss, five nodes) the task must
  identify the exact decode step and byte offset at which our reader's stream position or
  interpretation departs from the correct one, and record it in a written diagnosis in the
  task folder.
- **FR-2.** Classify every one of the 19 divergent images against the identified root
  cause(s). If more than one distinct defect exists, each must be separately diagnosed;
  a fix that repairs some images and is *assumed* to repair the rest is not acceptable.
- **FR-3.** Separately diagnose the enumeration gap (`9400300.img`, `9400301.img` absent).
  Note that `parseDirectory` currently swallows sub-directory parse failures with a
  `Warnf` (`libs/atlas-wz/wz/directory.go`), and skips `elemType == 1` entries as 10 bytes;
  either is a candidate, but the cause must be confirmed, not assumed.

### 4.2 Parser correctness

- **FR-4. [WITHDRAWN]** ~~After the fix, for each of the 19 images the parsed tree must
  be structurally equal to the HaRepacker reference: same set of node paths, same node
  types, same property names, same scalar values, same canvas `width`/`height`, same
  vector `x`/`y`. Both directions of the diff must be empty.~~ There is no fix, and the
  reference is not this archive; "structurally equal to the HaRepacker reference" is not
  a meaningful bar. See §10 for the live acceptance criteria.
- **FR-5. WITHDRAWN.** The specific known divergences below were believed to be defects
  requiring resolution. Task 5's byte-level adjudication (`reference-fidelity.md`)
  found all 21 items — the 19 originally-flagged plus the 2 un-enumerated images — are
  `INPUT-MISMATCH`: the dump was exported from a different, heavily customised WZ
  dataset, not from the supplied archive. Zero rows are `PARSER-DEFECT`. The `rpeat`
  row specifically is withdrawn on **positive evidence**, not mere unverifiability: the
  archive's own name-length bytes (`0xfb`=5 chars at one path, `0xfa`=6 chars at another)
  show the archive genuinely spells the property both `rpeat` and `repeat` at different
  paths — see `provenance.md` and `reference-fidelity.md`. The table is retained below
  as the original (withdrawn) requirement text.

  | Image(s) | Required post-fix result |
  |---|---|
  | `2406000` | `info/activateByTouch = 1` present; `0/event/0/{type=6,state=1}` present |
  | `2006000`, `2006001`, `2618003`–`2618007`, `9202005`, `9208003`, `9208004` | the missing `event` subtree (and its `int`/`string`/`vector` children) present, plus the sibling `hit` subtree where the evidence shows one |
  | `2519002`, `2519003` | the entire missing `imgdir:0` frame present — its `canvas:0` (100×121 / 92×124, with `z` and `origin`), its `event` subtree, and its `hit` subtree |
  | `2618000` | the missing `6/hit/uol:0` and `6/hit/uol:1` (`../0`, `../1`) present, and the missing `7/canvas:0` (72×145) and `7/canvas:1` (75×143, `origin (33,80)`) with their `delay`/`z` children, plus `7/event/0` |
  | `3002000` | the `event` subtree with `timeOut=2000` that only *our* parse produces is gone |
  | `2408001` | property named `repeat`, not `rpeat` |
  | `2502002` | `0/event/0/state = 0` and `0/event/1/state = 0` (we currently read 1); `0/event/1/2 = 1` present |
  | `2519000` | `1/canvas:0` is 100×121 with `origin = (49,121)`, not 1×1 at `(0,0)` |
  | `2519001` | `1/canvas:0` is 92×124 with `origin = (47,124)`, not 1×1 at `(0,0)` |

  (The authoritative per-image expectation is `evidence-wz-parse-divergence-reactor.txt`;
  the table is a summary, and the design phase must re-derive each row from that file.)
- **FR-6. [WITHDRAWN]** ~~`Directory.Images()` must enumerate `9400300` and `9400301`,
  and those images must parse to trees structurally equal to the HaRepacker
  reference.~~ Both images are absent from `$WZ_ARCHIVE`'s own directory
  (`reference-fidelity.md`) — enumerating them would mean fabricating entries the
  archive does not have.
- **FR-7. [WITHDRAWN]** ~~The 400 currently-identical images must remain identical
  after the fix. A fix verified only on the 19 is not verified.~~ There is no fix
  driven by this comparison; the general no-regression bar is carried forward as a
  Goal (§2) and in §10.
- **FR-8. Restated, not withdrawn.** If the reader tolerates a decode path silently
  (e.g. a stream position that lands mid-record and then "successfully" reads
  plausible garbage), it must make that condition an error rather than a silent
  shorter tree, wherever doing so does not break a legitimate archive. Silent
  truncation is exactly the property that made the false 19-image "divergence"
  diagnosis plausible for as long as it was. This requirement did not depend on the
  withdrawn oracle and is carried forward whole as Task R2's self-consistency gate
  (`plan.md`): every type-9 sub-object's declared size must be consumed exactly, with
  a violation reported as an error rather than silently healed by the recovery reseek.

### 4.3 Regression guard

- **FR-9. [WITHDRAWN]** ~~Land byte-level fixtures and a table-driven test in
  `libs/atlas-wz/wz` that asserts the expected property tree for every failing
  shape.~~ There is no failing shape (§4.1). The general form — a fixture-backed,
  external-tool-free test in `libs/atlas-wz/wz` — is carried forward for the
  self-consistency gate (Task R2).
- **FR-10. Partially retained.** Fixture construction, in priority order — prefer
  `wztest.Builder`-synthesized archives; extend the builder where a pattern cannot be
  synthesized; commit a trimmed byte blob only as a last resort. This priority order is
  a general `libs/atlas-wz` testing convention, not specific to the withdrawn
  divergence shapes, and it is exactly what Tasks 8-10 already landed (`wztest.Builder`
  now emits scalars, vectors, UOLs, convexes, dimensioned canvases, and
  offset-referenced string blocks). Task R2 reuses this machinery; it does not repeat
  this requirement.
- **FR-11. [WITHDRAWN]** ~~Coverage: at minimum one fixture per distinct defect
  class — lost subtree (`imgdir`), lost `uol` node, lost `canvas` node, gained
  subtree, mangled property name, wrong scalar value, collapsed canvas dimensions,
  un-enumerated image.~~ None of these are defect classes (§4.1); the fixture rows in
  the withdrawn Task 11 (`plan.md`) were never landed. Task R2 defines its own
  coverage (declared-size overrun/underrun on `wztest`-built archives).
- **FR-12. [WITHDRAWN]** ~~Each fixture test must fail on the pre-fix parser.~~ There
  is no pre-fix parser to fail against; nothing in `libs/atlas-wz/wz` is being fixed.
  Task R2's own TDD evidence (red-then-green against its own gate implementation)
  replaces this.

### 4.4 Full-archive verification — **WITHDRAWN, all of §4.4**

- **FR-13. [WITHDRAWN]** ~~Re-run the whole-archive structural diff for
  `GMS/83.1/Reactor.wz` after the fix and record the result: expected `421` images
  enumerated, `421` structurally identical, `0` divergent.~~ Unreachable with these
  inputs (`reference-fidelity.md`) and would remain unreachable after any correct
  change to the reader, because the dump is not this archive.
- **FR-14. [WITHDRAWN, as written; the tool itself survives].** ~~The repro tooling
  used for FR-13 (serialize via `wztoxml.SerializeToDirectory`, structurally diff
  against a HaRepacker dump) must be reproducible by another engineer.~~ `wzdiff`
  (Tasks 3-4) is landed, reviewed, and kept — it is exactly the checked-in repo tool
  this requirement asked for, and it remains useful for any *future* comparison
  against a dump that genuinely matches the archive it is run against (see
  `provenance.md`, "what remains open"). What is withdrawn is its use here to verify
  FR-13, since no such matching dump exists for this task.

## 5. API Surface

No HTTP/JSON:API surface changes; no Kafka message changes.

Go API surface in `libs/atlas-wz/wz`:

- `Image.Properties() ([]property.Property, error)` — signature unchanged. Behavior change
  only: it returns the correct tree, and (per FR-8) may return a non-nil error in cases
  where it previously returned a silently truncated tree with `nil`.
- `Directory.Images() []*Image` — signature unchanged; may now return more images for
  affected archives.
- `wztest.Builder` — additive only, if extended per FR-10.2. Existing builder call sites
  must keep compiling.
- Any new decoding helper stays unexported within `libs/atlas-wz/wz`.

Error-surface note: `atlas-data` callers already handle the `Properties()` error
(task-076 F6 forced the explicit decision). Any newly-surfaced error must be checked at
each call site so a hard-failing image is logged with the image name, not swallowed.

## 6. Data Model

No database entities, no migrations, no `tenant_id` scoping changes — this is a binary
decoder fix. The in-memory model (`libs/atlas-wz/wz/property`) is unchanged: no new
`property.*` types, no changed constructors.

If a new `testdata/` directory is introduced (FR-10.3), it lives under
`libs/atlas-wz/wz/testdata/` and is covered by the module's existing test conventions.

## 7. Service Impact

| Component | Change |
|---|---|
| `libs/atlas-wz/wz` | The fix. `image.go` (`parsePropertyList` / `parsePropertyValue` / `parseExtendedProperty` / `parseCanvasProperty`), `reader.go` (`ReadWzString*`, offset-referenced string blocks), and/or `directory.go` (`parseDirectory`) — whichever the root cause implicates. |
| `libs/atlas-wz/wztest` | Possibly extended to emit the fixture byte patterns (additive). |
| `libs/atlas-wz/{mapimage,icons,charparts,atlas}` | No intended change, but they consume canvas offsets and property trees from the same reader. Their tests must stay green; if canvas dimension decoding changes, verify no packer/atlas output shifts (`atlas.Pack` determinism is load-bearing — see `libs/atlas-wz/atlas/README.md`). |
| `services/atlas-data` | No intended code change. It consumes the corrected trees; more properties may now appear in every domain reader's input. Its build and tests must stay green. Behavior in a running environment changes only on re-ingest, which is out of scope. |
| Consumers of ingested data (atlas-reactor, atlas-map, …) | No change in this task. |
| `services/atlas-ui` | None. |

## 8. Non-Functional Requirements

- **Correctness over tolerance.** Where the reader currently guesses past a malformed or
  unexpected record, prefer failing loudly (FR-8). A hard error on one image is a bug
  report; a silently short tree is corrupt game data.
- **Performance.** WZ parsing is on the ingest path and `Image.Properties()` is lazy and
  cached under `File.parseMu`. The fix must not add a per-property allocation or an extra
  seek per property on the hot path; a full `Reactor.wz` serialize must not regress
  measurably. No new goroutines, no change to the locking discipline documented on
  `Image.Properties()` / `parsePropertyList` (task-172 C-2 concurrency invariants hold).
- **Thread safety.** `go test -race ./libs/atlas-wz/...` must pass. The existing
  `parse_race_test.go` and `iteration_contract_test.go` contracts must not be weakened.
- **Multi-tenancy.** Unaffected — the reader is tenant-agnostic; tenancy is applied by
  `atlas-data` around it. The per-image key fallback (task-172 C-2, JMS mixed encryption)
  must keep working; do not regress `errBadImageTag` retry semantics.
- **Observability.** Newly-surfaced parse failures must log the archive path and image
  name at a level that ingest operators will actually see, and must not spam per-property.
- **Evidence discipline.** ~~No claim of "fixed" without the post-fix whole-archive diff
  output (FR-13) and the pre-fix failing test output (FR-12) quoted in the task
  folder.~~ **[WITHDRAWN wording; principle retained.]** FR-13/FR-12 are withdrawn, but
  the underlying discipline is not: no claim of "done" for Task R2 without its own
  TDD evidence (red-then-green, quoted in the task folder) and the self-consistency
  gate's own output, per §10.
- **Licensing.** Committed fixtures (FR-10.3) must be the minimum bytes needed for the
  reproduction, not whole images or archives, wherever a synthesized fixture is impossible.

## 9. Open Questions

1. **Multiple root causes. WITHDRAWN — moot.** There are no root causes; Task 5 found
   zero `PARSER-DEFECT` images. The question of "fix all vs. fix the dominant one" does
   not arise.
2. **Blast-radius sweep.** Still open, unaffected by the oracle withdrawal. The
   follow-up sweep should run `wzdiff` in self-consistency mode (Task R2's gate, not a
   HaRepacker comparison) across Map.wz, Mob.wz, Item.wz, Skill.wz, String.wz,
   Character.wz, Npc.wz, Quest.wz, recording per-archive size-accounting violations.
3. **Re-ingest.** Still open, unaffected. Not applicable to this task since no reader
   fix is landing, but remains a live question for whatever Task R2 or a future sweep
   does find.
4. **`3002000`'s phantom `event` subtree — ANSWERED.** Confirmed from the archive bytes
   (`reference-fidelity.md`): our parse of `$WZ_ARCHIVE` is **right**. `/1`'s sub-object
   declares two children; the second is `event`; `event` declares `declaredSize=54`,
   consumed exactly (`endPos=41983721 actualEnd=41983721`); `timeOut` is present as its
   literal second child. This is **not** "the reference is lossy" — the dump's source
   file never had `/1/event` at all; it is a different image in a different archive.
   The originally anticipated framing ("the reference is authoritative for this task")
   was wrong in a way that matters: there was no authoritative reference to defer to.
5. **Canvas collapse and downstream renders — DISSOLVED.** `2519000`/`2519001` came back
   `INPUT-MISMATCH`: the archive's `1/canvas:0` is genuinely a 1×1 stub with no
   `_inlink`/`_outlink` in play (`reference-fidelity.md`). No canvas was ever
   mis-decoded, so no committed render baseline is affected and no baseline-refresh
   owner is needed.

## 10. Acceptance Criteria

The criteria below the withdrawal notice depended on the HaRepacker dump being an
independent, same-archive oracle. It is not (`provenance.md`,
`reference-fidelity.md`). They are **withdrawn** — deleted, not merely marked, per the
brief for this re-scope, because the available inputs make them permanently
unreachable rather than temporarily unmet:

- ~~A written diagnosis naming the exact decode step and byte offset where
  `2406000.img` diverges, classifying all 19 + 2 images against the identified
  cause(s).~~ Withdrawn — no cause exists to identify (§4.1).
- ~~Parsing `GMS/83.1/Reactor.wz` enumerates 421 images.~~ Withdrawn — the archive
  itself has 419; 421 was the dump's count, from a different archive.
- ~~The post-fix whole-archive structural diff of `Reactor.wz` against the HaRepacker
  reference reports 0 divergent images in either direction.~~ Withdrawn — unreachable;
  see FR-13.
- ~~`2406000` parses `info/activateByTouch = 1` and `0/event/0/{type=6,state=1}`.~~
  Withdrawn — these bytes are not in the supplied archive's `2406000` (§1, correction).
- ~~Each row of the FR-5 table is satisfied.~~ Withdrawn along with FR-5 in full.

**Live acceptance criteria, re-scoped to archive-byte self-consistency (Task R2):**

- [ ] Parsing every image of a supplied `.wz` archive with the self-consistency gate
      reports **zero size-accounting violations** — every type-9 sub-object's declared
      end equals its actual end (`declaredSize`/`endPos`/`actualEnd` agree, the same
      measurement `reference-fidelity.md` already took across 1136 sub-objects in the
      **19 divergent images** with zero mismatches, and which Task R2's whole-archive
      self-check corroborated at 15428 sub-objects across **all 419 images in
      `$WZ_ARCHIVE`**, also zero mismatches).
- [ ] The gate is a repo tool, runnable against any archive on disk, and exits
      non-zero on a size-accounting violation.
- [ ] The gate is covered by synthetic `wztest` fixtures — a clean archive passes and a
      deliberately corrupted one (a sub-object whose declared size under- or
      over-states its actual body) fails — so it is verifiable in CI with no external
      archive.
- [ ] `go test -race ./libs/atlas-wz/...` passes, including the existing
      `parse_race_test.go` and `iteration_contract_test.go`.
- [ ] `services/atlas-data` builds and its tests pass; no `wztoxml` change was
      required.
- [ ] `libs/atlas-wz/atlas` determinism tests still pass (no unintended packer output
      shift).
- [ ] Flagless `tools/verify.sh` exits 0. **Note:** currently blocked in this
      environment by `golangci-lint` being built against go1.26 while the repo
      toolchain is go1.27.0 — a toolchain issue independent of this branch.
- [ ] Code review completed before the PR opens.
- [ ] No reactor touch-activation *behavior* was added in this task (that stays in
      task-249), and no tenant re-ingest was performed.
