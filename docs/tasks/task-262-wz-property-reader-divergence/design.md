# WZ Property Reader Divergence — Design

Task: task-262-wz-property-reader-divergence
PRD: `docs/tasks/task-262-wz-property-reader-divergence/prd.md`
Evidence: `docs/tasks/task-262-wz-property-reader-divergence/evidence-wz-parse-divergence-reactor.txt`
Status: Historical — see the provenance correction below
Created: 2026-08-26
Revised: 2026-08-26 — see `provenance.md` and `reference-fidelity.md`

---

> **Provenance correction.** §0, §1, §2.1, and §2.3 below were written against a
> HaRepacker `.img.xml` dump believed to be an export of the same
> `GMS/83.1/Reactor.wz` archive supplied for this task. Task 5's byte-level
> adjudication (`reference-fidelity.md`) found the dump was exported from a
> different, heavily customised WZ dataset — `provenance.md` has the
> independent count evidence. Both "Population A" (genuine decode defects) and
> "Population B" (reference-side link/UOL resolution) below turned out to be
> the same thing: `INPUT-MISMATCH`, 21/21. **§2.1 and §2.3 are kept as
> historical record of how the premise error was found and reasoned about; they
> are not a live design position.** §2.2's gate *method* was still the right
> one — Task 5 ran it and it correctly surfaced the mismatch — but its intended
> two-way `PARSER-DEFECT`/`REFERENCE-RESOLUTION` split needed a third label.
> §2.2 is annotated with that outcome below. The live design going forward is
> the self-consistency gate (Task R2, `plan.md`), which needs no external
> reference at all.

## 0. Summary of the architectural position — HISTORICAL, see correction above

The PRD frames the whole 19-image delta as one thing: "our parser is wrong,
HaRepacker is right, make the diff empty." Reading the evidence file against the
reader source says that framing is **not safe to build on**. The 19 divergent
images fall into at least two populations, and only one of them is a parser
defect:

- **Population A — genuine decode defects.** Nodes our parser loses outright, a
  property *name* mangled, a scalar read wrong. These are ours. `2406000`,
  `2408001`, `2502002`, `2618003`–`2618007`, `3002000`, `2006000`.
- **Population B — reference-side reference resolution.** Divergences that are
  fully explained by the HaRepacker XML dump *following* `link` / `_inlink` /
  `_outlink` / `UOL` references while our parser stays literal. `2519002`,
  `2519003`, `2519000`, `2519001`, and (partly) `2006001`.

The design therefore front-loads a **reference-fidelity gate** (§2) before any
fix work, splits the acceptance target accordingly (§3), and only then commits
to the fix, error-surface, and regression architecture (§5–§8).

Everything below is a design position derived from the reader source and the
evidence file. It is not the FR-1 byte-level diagnosis — that is Phase 4 work
and this document specifies the harness that produces it (§4).

---

## 1. What the code actually looks like today

Four files carry the whole property decode path:

| File | Role |
|---|---|
| `libs/atlas-wz/wz/directory.go` | `parseDirectory` — entry enumeration, lazy `*Image` construction |
| `libs/atlas-wz/wz/image.go` | `parse` / `parseWithKey` / `parsePropertyList` / `parsePropertyValue` / `parseExtendedProperty` / `parseCanvasProperty` / `parseSoundProperty` |
| `libs/atlas-wz/wz/reader.go` | `ReadWzInt` / `ReadWzLong` / `ReadWzString` / `ReadWzStringBlock` / `ReadWzOffset` / `Peek` |
| `libs/atlas-wz/wztest/builder.go` | the only fixture generator; emits a strict subset of the format |

Four structural properties of that code matter for this design:

1. **The parse is position-driven with exactly one recovery point.**
   `parsePropertyValue` case 9 reads the sub-object `size`, computes
   `endPos = pos + size`, and — success or failure — `Seek(endPos)`. That single
   unconditional reseek is the *only* thing that keeps a desynchronised stream
   from cascading. It is also precisely why the defect is silent: a child list
   that reads garbage still lands the parent back on a valid boundary, so the
   parent continues and the image "parses successfully" with a wrong subtree.

2. **There is no block-bound enforcement anywhere else.**
   `parsePropertyList` has no `endPos`. Nothing checks that a child's decode
   consumed exactly its declared extent, or that the list stopped inside its
   parent's block. `parseCanvasProperty` and `parseSoundProperty` likewise
   compute their skip from a declared length and trust it.

3. **String reads seek away and back.** `ReadWzStringBlock` tags `0x01`/`0x1B`
   `Seek(fileStart+offset)`, `ReadWzString()`, `Seek(pos)`. Every property name
   and every type tag in a real archive goes through this. A one-byte error in
   the *offset base* or in `ReadWzString`'s length/tag interpretation produces a
   wrong *name* while leaving the stream position perfectly intact — which is
   exactly the shape of `2408001`'s `repeat` → `rpeat`. This is the highest-prior
   suspect for Population A and §4 is built to prove or kill it from bytes.

4. **Failures are swallowed at three levels.**
   `parseDirectory` `Warnf`s a failed sub-directory and continues (losing every
   image beneath it); `Image.Properties()` logs and caches the error but returns
   the partial slice alongside it; and the ingest-side
   `wztoxml.serializeDirectory` (`services/atlas-data/atlas.com/data/data/wztoxml/adapter.go:45`)
   `Warnf`s a failed image and continues. FR-8 and the observability NFR are
   about closing the *first* and *third* of these, not the second — `Properties()`
   already returns an error and task-076 F6 deliberately made callers decide.

---

## 2. Decision 0 (blocking): establish what the reference actually is

### 2.1 Why this is a decision and not an assumption — HISTORICAL

This subsection reasons from the (false) premise that the dump is a same-archive
export. Task 5 found it is not (`reference-fidelity.md`); the link/UOL-resolution
theory below was never actually tested against a matching archive. Kept as the
record of the reasoning that led to building the §2.2 gate, which is the part
that survived.

`services/atlas-data/atlas.com/data/reactor/reader.go:65` does
`link := info.GetString("link", "")` and follows the link itself. So does
`services/atlas-data/atlas.com/data/map/reader.go:60`, and
`libs/atlas-wz/icons/extract.go:222` (`findInfoLink`). **Link resolution in Atlas
is a consumer responsibility and the parser is deliberately literal.**

Now read the evidence for `2519002`:

```
  -- present in HaRepacker dump, ABSENT from our parse:
      /imgdir:0 …  /imgdir:0/canvas:0 | height=121 width=100 … (the whole 2519000 body)
  -- present in our parse, ABSENT from HaRepacker dump:
      /imgdir:info/string:link | value=2519000
```

That is not a parse failure. That is the reference dump having **substituted the
link target's body and dropped the `link` node**, while our parse emitted the
literal image — which is a two-node image containing `info/link`, exactly what the
bytes say and exactly what `reactor/reader.go` expects to receive.

`2519000` / `2519001` are the same phenomenon one level down: `1/canvas:0` really
*is* a 1×1 stub in the archive (that is how `_inlink`/`_outlink` canvases are
stored); the reference printed the resolved target's `100×121` / `92×124`. And
`2006001`'s `/imgdir:1/uol:0 = ../0/0` appearing in the reference as a
`canvas:0` 1×1 is a resolved UOL.

If we accept FR-4/FR-13 literally ("0 divergent in either direction"), the only
way to satisfy them for these images is to **implement link/UOL resolution inside
`Image.Properties()`** — which would double-resolve against `reactor/reader.go`,
contradict the PRD's own non-goal list, and change the meaning of the property
tree for every existing consumer. That is the wrong fix.

### 2.2 The gate — method survived, outcome differed from what was anticipated

> **Outcome (Task 5, `reference-fidelity.md`).** The gate below was framed as a
> two-way adjudication, `PARSER-DEFECT` vs. `REFERENCE-RESOLUTION`. Running it
> produced neither: all 21 items (19 divergent + 2 un-enumerated) came back a
> third label, `INPUT-MISMATCH` — the dump was never exported from the supplied
> archive at all (`provenance.md`). Byte adjudication (method 1 below) was
> exactly what surfaced this: 1136 type-9 sub-objects traced across the 19
> images with 0 instances of `actualEnd != endPos`, i.e. our reader is
> byte-faithful to the archive everywhere it was checked, which is
> incompatible with either of the two labels this section anticipated. The
> gate mechanism (byte adjudication first, ordered above the other two
> methods) is exactly right; what follows describes the two outcomes it was
> built to distinguish between, neither of which occurred.

Before any code change, Phase 4 must produce
`docs/tasks/task-262-wz-property-reader-divergence/reference-fidelity.md`
answering, per divergent image, one question: **is this delta explained by the
reference resolving a reference, or by our decode?**

Method, in order of authority:

1. **Byte adjudication.** For each disputed node, hexdump the image block at the
   relevant offset and hand-decode. The archive bytes are authoritative over both
   parsers. This is decisive and cheap for the ~8 disputed images.
2. **Re-dump with resolution disabled.** Confirm which HaRepacker export setting
   produced the reference and whether a literal-mode dump exists. Recorded, not
   trusted over (1).
3. **Cross-check against a third literal reader** only if (1) is ambiguous.

Output: each of the 19 images plus the 2 missing ones labelled `PARSER-DEFECT`,
`REFERENCE-RESOLUTION`, or `MIXED`, with the byte evidence for the label.

`2618000` is deliberately called out as **MIXED-suspect**: it shows a whole-index
shift (our `/imgdir:6/hit/canvas:*` is byte-identical in content to the
reference's `/imgdir:7/hit/canvas:*`), plus a `7/canvas:0` whose `delay` we read
as `50000` and the reference reads as `150`. An index shift with intact content
is a *name* defect, not a structure defect — same family as `rpeat`. Do not let
its size make it look like a different bug.

Open Question 4 in the PRD (`3002000`'s phantom `event` subtree) is answered by
this same gate, in the opposite direction: if the bytes contain that subtree, the
reference is skipping a record it does not model and **we are right**.

### 2.3 Consequence for acceptance — HISTORICAL

This restatement assumed the gate would produce a `REFERENCE-RESOLUTION`
allowlist. It did not (§2.2 outcome, above) — `allowlist.tsv` was never
produced because there is nothing to allowlist. Kept as the record of the
decision the user was asked to sign off on; superseded by the re-scoped §10 of
`prd.md` and by Task R2's self-consistency criteria.

FR-4/FR-13 are restated as: the post-fix whole-archive diff must show
**zero `PARSER-DEFECT` deltas**, and every remaining delta must be an
already-adjudicated `REFERENCE-RESOLUTION` entry on a committed allowlist, with
its byte justification. The allowlist is checked into the task folder and is
part of the deliverable, not a waiver.

> **This is the one thing in this design that needs the user's sign-off before
> Phase 3.** See §10.

---

## 3. Scope decomposition

| Workstream | Depends on | Output |
|---|---|---|
| W0 Reference fidelity gate | — | `reference-fidelity.md`, per-image labels, allowlist |
| W1 Diagnostic harness | — (parallel with W0, and feeds it) | `wzTrace` instrumentation + `wzdiff` tool |
| W2 Byte-level diagnosis (FR-1/FR-2) | W0, W1 | `diagnosis.md` with offsets |
| W3 Enumeration gap (FR-3/FR-6) | W1 | diagnosis + fix |
| W4 Decode fix | W2 | `libs/atlas-wz/wz` changes |
| W5 Strictness / error surface (FR-8) | W4 | error-on-desync + call-site handling |
| W6 Builder extension + fixtures (FR-9–FR-12) | W2 | `wztest` additions, table tests |
| W7 Full-archive re-verify (FR-13/FR-14) | W4, W5 | post-fix diff output, documented repro |

W1 and W6 are the two largest engineering surfaces; W4 is expected to be small
once W2 lands.

---

## 4. Diagnostic architecture (W1) — the piece that makes W2 possible

### 4.1 The problem with debugging this by inspection

There are ~11k property nodes in `Reactor.wz`. "Read `image.go` until the bug is
obvious" has already failed once — this defect has been latent since the reader
was written. What is needed is a mechanical answer to *"at which byte does our
stream position first mean something different from what the format says?"*

### 4.2 Chosen approach: a trace hook on the reader, plus a structural differ

**Option A — printf/logrus tracing in `parsePropertyList`.** Rejected: it
pollutes the hot path, the log is unstructured, and it can't be diffed against a
tree.

**Option B — a second, independent "audit" parser written alongside.** Rejected:
a from-scratch second implementation is the same work as the fix plus a new
source of disagreement, and it doesn't tell you where the *existing* one goes
wrong.

**Option C (chosen) — an opt-in trace sink on `*File`, emitting one record per
decoded node, and an offline differ.**

```go
// libs/atlas-wz/wz — new, unexported-by-default surface
type TraceEvent struct {
    Path     string // "/0/event/0/state"
    Kind     string // "list", "prop", "extended", "canvas", "stringblock"
    Name     string
    Type     byte
    StartOff int64
    EndOff   int64
    Detail   string // e.g. "stringblock tag=0x1B off=0x1a2c -> \"repeat\""
}

func (wz *File) SetTrace(fn func(TraceEvent)) // nil by default
```

Design constraints this satisfies:

- **Zero hot-path cost when unset.** The emit sites are guarded by a single
  `if wz.trace != nil`. No allocation, no seek, no goroutine — satisfies the
  performance NFR literally.
- **Locking unchanged.** The hook fires under the already-held `parseMu`; it must
  be documented as "called synchronously under `parseMu`; must not re-enter the
  reader." Preserves the task-172 C-2 / task-172 concurrency invariants and keeps
  `parse_race_test.go` meaningful.
- **The string-block detail field is the point.** Every `ReadWzStringBlock` emits
  its tag, its resolved absolute offset, and the string it produced. If `rpeat`
  comes from an offset-ref, the trace names the exact source offset — FR-1 falls
  straight out.

The differ (`wzdiff`) consumes (a) the trace, (b) our serialized XML, (c) the
HaRepacker XML, and reports the **first path where the two trees disagree,
annotated with the byte range our parser was in at that point**. That annotated
first-divergence offset *is* the FR-1 deliverable.

### 4.3 Tool placement

`tools/` already hosts Go tooling with a `cmd/` subdir per tool
(`tools/packet-audit/cmd`, `tools/atlasguards/cmd`, …). Follow it:

```
tools/wzdiff/
  cmd/wzdiff/main.go
  diff.go        // structural tree diff, both directions
  xmlload.go     // parse HaRepacker XML and our wztoxml XML into a common tree
  allowlist.go   // §2.3 REFERENCE-RESOLUTION allowlist
```

`wzdiff` takes `--archive <path.wz> --reference <harepacker-dump-dir>
[--allowlist <file>]` and exits non-zero on any unallowlisted delta. This
satisfies FR-14 as a *checked-in repo tool* rather than as prose, which is the
stronger of the two options the PRD offers. It is not wired into `verify.sh`
(it needs a real archive and an external dump); it is a developer/operator tool.

**Reuse note:** it must serialize via `wztoxml.SerializeToDirectory` to keep the
comparison honest to the ingest path. `wztoxml` lives in `services/atlas-data`,
so `tools/wzdiff` depends on the `atlas-data` module. If `go.work` makes that
awkward, the fallback is to move nothing and instead have `wzdiff` shell out —
**rejected**; instead `wzdiff` reimplements nothing and imports the package
directly. Confirm the module edge during planning; if it is genuinely blocked,
the alternative is to promote `wztoxml`'s pure tree→XML mapping into
`libs/atlas-wz/wz/wzxml` and have `atlas-data` re-export it. Prefer the
straightforward move over an alias shim, per repo convention.

---

## 5. Enumeration gap (W3) — `9400300`, `9400301`

Three candidate causes, in the order the harness should eliminate them:

1. **A swallowed sub-directory.** `parseDirectory` `Warnf`s and continues on a
   failed recursive parse (`directory.go`, the `elemType == 3` branch). If those
   two images live under a sub-directory that fails, they vanish with no error to
   the caller and only a warning in the log. **Cheapest to test: re-run the
   enumeration with the log at debug and look for the `Unable to parse
   sub-directory` line.** Do this first.
2. **`elemType == 1` skip.** We skip 10 bytes and `continue`. HaRepacker's
   equivalent consumes int32 + int16 + offset(4) = 10 bytes, so the *width* is
   right; but if a type-1 entry ever precedes real entries the widths must match
   exactly or every subsequent entry desyncs. The trace makes this visible.
3. **A short `count`.** Would show as a clean truncation at the end of the
   listing — 419 vs 421 with the two missing images being the *last* two is
   consistent with this, and it is the reason not to assume (1).

The fix, whichever it is, must also address the swallow: **a sub-directory parse
failure must propagate**, because "we silently enumerated 419 of 421 images" is
the enumeration-level instance of exactly the corruption class this task exists
to kill. See §6.

---

## 6. Fix + strictness architecture (W4/W5)

### 6.1 The decode fix itself

Deliberately unspecified here. W2 produces the offsets; the patch follows the
offsets. This document's job is to constrain *how* it lands, not to pre-guess it:

- **No speculative patch.** A change that makes an image match without a byte
  offset explaining why is rejected at review. (PRD FR-1; CLAUDE.md evidence
  discipline.)
- **One commit per identified defect**, each carrying its failing fixture.
  FR-2 forbids "fixed some, assumed the rest"; separate commits make that
  auditable.
- **No wire/behaviour change to the 400 matching images.** Enforced by W7's
  full-archive re-run, not by inspection (FR-7).

### 6.2 Strictness — the design decision inside FR-8

FR-8 says: make silent truncation an error "wherever doing so does not break a
legitimate archive." That qualifier is the whole design problem. Three levels of
strictness, escalating:

| Level | Rule | Risk |
|---|---|---|
| S1 | `parsePropertyList` carries an `endPos`; error if a child decode overruns it | Low. An overrun is unambiguously corrupt. |
| S2 | Additionally error if a sub-object decode *underruns* `endPos` (stopped early) | Medium. Some legitimate archives pad; needs the full-archive run to confirm zero false positives. |
| S3 | Error on any unknown property type / extended tag / string-block tag instead of tolerating | Medium-high. HaRepacker returns `""` for an unknown string-block tag rather than failing; matching its tolerance may be load-bearing for other archives. |

**Recommendation: land S1 unconditionally; land S2 gated on the full-archive run
across `Reactor.wz` showing zero new failures, and re-check it against at least
one other archive before enabling; do not land S3 in this task** — it is a
tolerance policy change with a blast radius across every archive and every
tenant, and this task's evidence base is one archive. Record S3 as the follow-up
alongside the §9 sweep.

Note that S1/S2 are *additive* checks around the existing `Seek(endPos)`
recovery, not a replacement for it. The recovery reseek must stay: it is what
lets a single bad subtree be reported without destroying the rest of the image.
What changes is that it now also **records** the desync instead of hiding it.

### 6.3 Error surface and observability

- `Image.Properties()` signature is unchanged (PRD §5). A newly-strict image
  returns `(partialProps, err)`; the error is already cached and already
  returned.
- `parseDirectory`'s sub-directory `Warnf` becomes a returned error (§5).
  This *is* an API behaviour change for `File.Open` and needs a sweep of
  `libs/atlas-wz/{mapimage,icons,charparts,atlas}` and `atlas-data` call sites.
- `wztoxml.serializeDirectory`'s per-image `Warnf` (adapter.go:45) must log the
  **archive path and image name** and must be countable — the ingest operator
  needs "N of 421 images failed," not 421 individual lines to eyeball. Emit one
  line per failed image plus one summary line per archive. Per-property logging
  is forbidden by the NFR.
- No new error is introduced on the canvas *pixel* path; `libs/atlas-wz/canvas`
  is untouched (PRD non-goal).

### 6.4 Canvas `dataOffset` note (observation, not scope)

`parseCanvasProperty` comments "skip 1 byte header before actual data" and then
records `dataOffset` *without* skipping it, before `Skip(dataSize)`. The total
extent consumed is correct, so property-tree structure is unaffected — but the
recorded offset points at the flag byte, and `wztest`'s builder writes a `0xAB`
flag byte that `ReadCanvasData` is documented to skip. The two are consistent
today. **Do not "fix" this as a drive-by**: `libs/atlas-wz/atlas`'s pack
determinism is load-bearing (`libs/atlas-wz/atlas/README.md`). If W2 implicates
it, it becomes a tracked defect with its own fixture; otherwise it is out of
scope and gets a comment correcting the misleading wording only.

---

## 7. Regression guard architecture (W6)

### 7.1 The constraint that shapes this

`libs/atlas-wz` has **no `testdata/` and no committed `.wz` today**;
`fixture_roundtrip_test.go` builds archives in-process into `t.TempDir()`. The
PRD's FR-10 priority order (synthesize > extend builder > commit bytes) is the
existing convention and should hold.

But the current `wztest.Builder` cannot express most of the shapes this task is
about. From `builder.go`:

| Format feature | Builder support today |
|---|---|
| string block, inline (`0x73`) | yes |
| **string block, offset-ref (`0x01`/`0x1B`)** | **no** |
| `imgdir` / `Property` sub-object (type 9) | yes |
| int (type 3), string (type 8) | yes |
| **null (0), short (2/11), long (20), float (4), double (5)** | **no** |
| **`Shape2D#Vector2D`** | **no** |
| **`UOL`** | **no** |
| **`Shape2D#Convex2D`, `Sound_DX8`** | **no** |
| canvas | yes, but hard-coded 1×1, no children, no origin |
| directory `elemType == 1` entry | no |

Since the leading Population-A hypothesis is an offset-referenced-string defect
(§1.3) and the divergences are full of `vector`, `uol`, and dimensioned canvases,
**extending the builder is not optional — it is the bulk of W6.**

### 7.2 Builder extension (additive only)

Add, mirroring the existing `Int`/`Str`/`Sub`/`Canvas` constructor style:

```go
func Null(name string) Prop
func Short(name string, v int16) Prop
func Long(name string, v int64) Prop
func Float(name string, v float32) Prop
func Double(name string, v float64) Prop
func Vector(name string, x, y int32) Prop
func UOL(name, target string) Prop
func Convex(name string, children ...Prop) Prop
func CanvasWith(name string, w, h int, payload []byte, children ...Prop) Prop
```

plus the one that actually matters:

```go
// SetStringDedup makes the builder emit the SECOND and later occurrence of any
// string as an offset reference (tag 0x1B for type tags, 0x01 for names),
// exactly as a real client-produced archive does.
func (b *Builder) SetStringDedup(on bool) *Builder
```

Every existing `wztest` call site must keep compiling (PRD §5) — all of the above
are new symbols and one new opt-in setter, default off, so they do.

`SetStringDedup` requires the builder to thread a per-image string table and
patch offsets after layout. That is real work but it is the only way to
synthesize the byte pattern the primary hypothesis depends on, and it is FR-10.2
by name.

### 7.3 Test shape

A single table-driven test in `libs/atlas-wz/wz`, one row per defect class from
FR-11, each row = (builder program → expected `[]property.Property` tree). Use
the project Builder pattern for setup; no `*_testhelpers.go` (repo convention).
Compare with a shared deep-equality helper that reports the first differing path
in the same `/a/b:name` notation the evidence file uses, so a failure reads like
the evidence.

Rows required at minimum:
lost `imgdir` subtree · lost `uol` node · lost `canvas` node · gained subtree ·
mangled property name · wrong scalar value · collapsed canvas dimensions ·
un-enumerated image (directory-level, so this row builds a whole archive).

### 7.4 Committed bytes — the escape hatch and its bar

If a shape resists synthesis, commit a trimmed blob under
`libs/atlas-wz/wz/testdata/`, minimum bytes for the reproduction, with a header
comment naming the source archive, the image, and the shape it pins. This is
FR-10.3 and is **last resort**: prefer extending the builder. Any use of it must
say in the commit message why synthesis was not achievable.

### 7.5 FR-12 — proving the tests fail first

Order of operations, enforced by commit order:

1. Land the builder extension + the failing tests **against the unfixed reader**.
2. Capture `go test ./libs/atlas-wz/wz/... -run TestPropertyDivergence` output to
   `docs/tasks/task-262-wz-property-reader-divergence/pre-fix-test-failures.txt`
   and commit it.
3. Land the fix; the same tests go green.

A reviewer can then verify FR-12 from `git log` alone, not from a claim.

---

## 8. Blast radius

| Consumer | Exposure | Mitigation |
|---|---|---|
| `libs/atlas-wz/mapimage`, `icons`, `charparts` | consume canvas offsets + trees from the same reader | their tests must stay green; if canvas dims change, re-check `icons` output |
| `libs/atlas-wz/atlas` | `atlas.Pack` determinism is load-bearing | determinism tests are a hard gate; §6.4 forbids drive-by canvas-offset changes |
| `services/atlas-data` domain readers | receive *more* properties after the fix | no code change expected; build + tests green. `reactor/reader.go` and `map/reader.go` already follow `info/link` themselves — §2's whole point |
| `wztoxml/adapter.go` | `propertyToElement` has a case per `property.*` type and no drop path | unchanged (PRD non-goal); only the per-image error logging changes (§6.3) |
| Running environments | corrected data needs re-ingest | out of scope; flag to the operator in the PR description |

Concurrency: `SetTrace` is the only new surface touching the parse path. It is
documented as called under `parseMu` and is nil in production. `go test -race
./libs/atlas-wz/...` including `parse_race_test.go` and
`iteration_contract_test.go` is a hard gate.

---

## 9. What this task deliberately does not do

- No sweep of Map.wz / Mob.wz / Item.wz / Skill.wz / String.wz / Character.wz /
  Npc.wz / Quest.wz (PRD §2, §9.2). `tools/wzdiff` is built so that sweep is a
  one-liner per archive; seeding it is this task's contribution to the follow-up.
- No S3 tolerance-policy change (§6.2).
- No link/UOL resolution in the parser (§2.1) — that stays with consumers.
- No reactor touch-activation behaviour (task-249 owns it).
- No re-ingest.
- No render-baseline refresh (PRD §9.5); if `2519000`/`2519001` turn out to be
  `REFERENCE-RESOLUTION` rather than defects — which §2 expects — then no
  baseline moves and the question dissolves.

---

## 10. Decision needed before Phase 3 — RESOLVED, superseded

This decision was made (user approved the §2.3 restatement, `context.md` §1.1) and
then superseded again by the finding it did not anticipate: §2.2's gate returned
`INPUT-MISMATCH`, not `REFERENCE-RESOLUTION`, so there was no allowlist to commit.
The live acceptance criteria are `prd.md` §10 and Task R2. Kept as the historical
record of the decision point.

**§2.3 — the acceptance restatement.** The PRD's FR-4/FR-13 demand a
literally-empty bidirectional diff. This design argues that is unreachable
*and undesirable*, because part of the delta is the reference dump resolving
`link`/`_inlink`/`UOL` while our parser is correctly literal, and Atlas resolves
those in the consumer (`reactor/reader.go:65`, `map/reader.go:60`,
`icons/extract.go:222`).

Proposed replacement: **zero `PARSER-DEFECT` deltas, plus a committed,
byte-justified allowlist of `REFERENCE-RESOLUTION` deltas** produced by the §2.2
gate.

The alternative — hold FR-13 as written and implement resolution in the parser —
is a substantially larger and, in my read, wrong change. Confirm the
restatement, or say to hold the original bar, before planning.

Secondary, non-blocking: PRD §9.3 (who schedules the Reactor re-ingest for
affected tenants once this merges) is an operational question this design does
not answer.
