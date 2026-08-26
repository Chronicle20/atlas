# WZ Property Reader Divergence from HaRepacker — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-08-25
Issue: https://github.com/Chronicle20/atlas/issues/1496
---

## 1. Overview

`libs/atlas-wz/wz` is the single WZ binary reader for the entire Atlas server. Every
domain's game data flows through it: `services/atlas-data` opens the tenant's `.wz`
archives, walks `Directory.Images()`, calls `Image.Properties()`, serializes the result
through `data/wztoxml`, and hands the XML to the per-domain readers. If the property
reader mis-decodes an image, the corruption is silent — the reader returns a shorter
(or differently-shaped) property tree with no error, and the domain reader simply
imports whatever it was given.

That is happening today. Serializing all 419 images our parser enumerates from a
tenant's `GMS/83.1/Reactor.wz` (PKG1, 51.6 MiB) with the ingest pipeline's own
`wztoxml.SerializeToDirectory`, and structurally diffing each against a HaRepacker XML
dump of the *same archive file*, shows 400 images structurally identical and **19
divergent**. Same bytes in, different tree out — this is a parser defect, not a data
mismatch. The divergence runs in both directions and in several distinct shapes:
whole subtrees lost, a subtree gained that HaRepacker does not have, a property *name*
mangled (`repeat` → `rpeat`), scalar values wrong (`state` 0 read as 1), and canvases
decoded as 1×1 that are really 100×121. Separately, our parser enumerates 419 images
where HaRepacker enumerates 421: `9400300.img` and `9400301.img` are never seen at all.

This was found while implementing touch-activated reactors (task-249, #1459). Reactor
`2406000` is a touch reactor in the WZ, but our parse drops both `info/activateByTouch`
and the entire state-0 `event` subtree, so the server can never treat it as
touch-activated. Reactor.wz is only where it surfaced — the same reader feeds Map.wz,
Mob.wz, Item.wz, Skill.wz and the rest, so the same class of silent corruption is
likely present across every domain and has never been swept for.

The evidence file for the full Reactor.wz diff is committed alongside this PRD as
`evidence-wz-parse-divergence-reactor.txt`.

## 2. Goals

Primary goals:

- Root-cause the divergence(s) in `libs/atlas-wz/wz` down to the specific decode step(s),
  with the failing byte sequence identified — not a plausible-looking patch.
- Make our parse of all 19 divergent Reactor.wz images structurally identical to
  HaRepacker's, in both directions (nothing lost, nothing gained, names and scalars equal).
- Make the parser enumerate `9400300.img` and `9400301.img` (421 images, not 419).
- Land a permanent, CI-runnable regression guard built from committed fixtures, so a
  future change to the reader cannot silently reintroduce any of these shapes.
- Cause no regression on the 400 images that already match, and no regression in any
  existing `libs/atlas-wz` consumer (`mapimage`, `icons`, `charparts`, `atlas` packer,
  `atlas-data` domain readers).

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

### 4.1 Root cause (required before any fix lands)

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

- **FR-4.** After the fix, for each of the 19 images the parsed tree must be structurally
  equal to the HaRepacker reference: same set of node paths, same node types, same property
  names, same scalar values, same canvas `width`/`height`, same vector `x`/`y`. Both
  directions of the diff must be empty.
- **FR-5.** The specific known divergences must each be resolved:

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
- **FR-6.** `Directory.Images()` must enumerate `9400300` and `9400301`, and those images
  must parse to trees structurally equal to the HaRepacker reference.
- **FR-7.** The 400 currently-identical images must remain identical after the fix. A fix
  verified only on the 19 is not verified.
- **FR-8.** If the root cause is a decode path our reader silently tolerates (e.g. a
  stream position that lands mid-record and then "successfully" reads plausible garbage),
  the fix must make that condition an error rather than a silent shorter tree, wherever
  doing so does not break a legitimate archive. Silent truncation is the property that made
  this defect invisible for so long.

### 4.3 Regression guard

- **FR-9.** Land byte-level fixtures and a table-driven test in `libs/atlas-wz/wz` that
  asserts the expected property tree for every failing shape. The test must run under
  `go test ./...` with no external tool, no network, and no real archive on disk.
- **FR-10.** Fixture construction, in priority order:
  1. Prefer `wztest.Builder`-synthesized archives that reproduce the *byte pattern* the
     root cause depends on. This is the existing convention — `libs/atlas-wz` has no
     committed `.wz` or `testdata/` today; `fixture_roundtrip_test.go` builds archives
     in-process into `t.TempDir()`.
  2. Where the pattern cannot be synthesized (e.g. a canvas whose header the builder
     cannot currently emit), extend `wztest.Builder` to emit it.
  3. Only if neither is achievable, commit a trimmed byte blob for that single image
     under a `testdata/` directory, kept as small as the reproduction requires, with a
     comment naming the source archive and the shape it pins.
- **FR-11.** Coverage: at minimum one fixture per distinct defect class — lost subtree
  (`imgdir`), lost `uol` node, lost `canvas` node, gained subtree, mangled property name,
  wrong scalar value, collapsed canvas dimensions, un-enumerated image.
- **FR-12.** Each fixture test must fail on the pre-fix parser. Demonstrate this: run the
  new tests against the unfixed reader and record the failure output in the task folder.
  A regression test that passes before the fix guards nothing.

### 4.4 Full-archive verification

- **FR-13.** Re-run the whole-archive structural diff for `GMS/83.1/Reactor.wz` after the
  fix and record the result: expected `421` images enumerated, `421` structurally
  identical, `0` divergent. Store the post-fix diff output in the task folder next to the
  pre-fix evidence.
- **FR-14.** The repro tooling used for FR-13 (serialize via
  `wztoxml.SerializeToDirectory`, structurally diff against a HaRepacker dump) must be
  reproducible by another engineer from what is written down — either checked in as a
  repo tool, or documented step-by-step in the task folder. The HaRepacker dump itself is
  an external input and is not committed.

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
- **Evidence discipline.** No claim of "fixed" without the post-fix whole-archive diff
  output (FR-13) and the pre-fix failing test output (FR-12) quoted in the task folder.
- **Licensing.** Committed fixtures (FR-10.3) must be the minimum bytes needed for the
  reproduction, not whole images or archives, wherever a synthesized fixture is impossible.

## 9. Open Questions

1. **Multiple root causes.** The five divergence shapes may or may not share one cause. If
   diagnosis shows two or more independent defects, does this task fix all of them, or fix
   the dominant one and spin the rest out? *Assumption for now: fix all of them — they are
   all in the same reader and the task's acceptance is a clean whole-archive diff.*
2. **Blast-radius sweep.** Explicitly deferred (§2). The follow-up task should run the same
   whole-archive structural diff across Map.wz, Mob.wz, Item.wz, Skill.wz, String.wz,
   Character.wz, Npc.wz, Quest.wz and record per-archive divergence counts, *after* this
   fix lands so it measures residual, not known, corruption.
3. **Re-ingest.** Out of scope (§2), but the corrected data does not reach a running
   environment until affected tenants re-ingest. Who schedules that, and does any tenant
   need a targeted Reactor re-import once this merges?
4. **`3002000`'s phantom `event` subtree.** Our parse invents a subtree HaRepacker does not
   have. Confirm during diagnosis whether HaRepacker is *skipping* something real (a
   record it doesn't model) rather than us fabricating one — the reference is
   authoritative for this task, but the distinction matters if a third parser is ever
   consulted.
5. **Canvas collapse and downstream renders.** If `2519000`/`2519001` canvases were being
   decoded 1×1, sprite/atlas output for those reactors was presumably also wrong. Confirm
   whether any committed render baseline changes as a result; if so, that baseline refresh
   needs an owner (likely the follow-up, not this task).

## 10. Acceptance Criteria

- [ ] A written diagnosis in `docs/tasks/task-262-wz-property-reader-divergence/` names the
      exact decode step and byte offset where `2406000.img` diverges, and classifies all 19
      divergent images plus the 2 un-enumerated images against the identified cause(s).
- [ ] Parsing `GMS/83.1/Reactor.wz` enumerates **421** images.
- [ ] The post-fix whole-archive structural diff of `Reactor.wz` against the HaRepacker
      reference reports **0 divergent images in either direction**, and its raw output is
      committed to the task folder.
- [ ] `2406000` parses `info/activateByTouch = 1` and `0/event/0/{type=6,state=1}`.
- [ ] Each row of the FR-5 table is satisfied, verified against
      `evidence-wz-parse-divergence-reactor.txt`.
- [ ] New fixture-backed tests in `libs/atlas-wz/wz` cover every distinct defect class
      (lost subtree, gained subtree, mangled name, wrong scalar, collapsed canvas,
      un-enumerated image), run with no external tool, and their **pre-fix failure output is
      recorded** in the task folder.
- [ ] `go test -race ./libs/atlas-wz/...` passes, including the existing `parse_race_test.go`
      and `iteration_contract_test.go`.
- [ ] `services/atlas-data` builds and its tests pass; no `wztoxml` change was required.
- [ ] `libs/atlas-wz/atlas` determinism tests still pass (no unintended packer output shift).
- [ ] Flagless `tools/verify.sh` exits 0.
- [ ] Code review completed before the PR opens.
- [ ] No reactor touch-activation *behavior* was added in this task (that stays in task-249),
      and no tenant re-ingest was performed.
