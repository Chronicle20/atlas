# WZ Property Reader Divergence — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Root-cause and fix the `libs/atlas-wz/wz` property-decode defects that make our parse of `GMS/83.1/Reactor.wz` diverge from HaRepacker on 19 of 421 images, make the parser enumerate all 421 images, and land a byte-level regression guard so the defect classes cannot silently return.

**Architecture:** Build the diagnostic harness first (a nil-by-default trace hook on `*File`, plus a `wzdiff` structural differ that consumes our own ingest serializer), use it to adjudicate every divergence against the archive bytes, then fix only what the bytes implicate. Strictness (`endPos` bounds, propagated directory errors) is added around — not instead of — the existing type-9 recovery reseek, so a single bad subtree is reported rather than hidden. Regression coverage comes from an extended `wztest.Builder` that can synthesize the byte patterns the defects depend on, not from committed archive blobs.

**Tech Stack:** Go 1.27 (`go1.27.0`), `libs/atlas-wz` module (`github.com/Chronicle20/atlas/libs/atlas-wz`), `logrus`, stdlib `encoding/xml`, `wztest.Builder` fixtures, `go test -race`.

**Spec:** `docs/tasks/task-262-wz-property-reader-divergence/design.md`
**PRD:** `docs/tasks/task-262-wz-property-reader-divergence/prd.md`
**Evidence:** `docs/tasks/task-262-wz-property-reader-divergence/evidence-wz-parse-divergence-reactor.txt`

---

## Global Constraints

Every task's requirements implicitly include this section.

- **Acceptance bar — WITHDRAWN and superseded again.** The bullet below (the
  user-approved restatement of PRD FR-4/FR-13, design §2.3) assumed Task 5's gate
  would split the 19 divergences into `PARSER-DEFECT` and `REFERENCE-RESOLUTION`.
  It did not: Task 5 found all 21 items (19 divergent + 2 un-enumerated)
  `INPUT-MISMATCH` — the HaRepacker dump was never exported from the supplied
  archive (`docs/tasks/task-262-wz-property-reader-divergence/provenance.md`,
  `reference-fidelity.md`). There is no allowlist and nothing to fix in
  `libs/atlas-wz/wz`. Tasks 6, 7, and 11-15 below, which all depend on that
  allowlist or on a fix landing, are marked **WITHDRAWN** in place. The live
  acceptance bar is `prd.md` §10 (re-scoped) and Task R2's self-consistency gate.
  ~~The post-fix whole-archive diff must show **zero `PARSER-DEFECT` deltas**.
  Every remaining delta must be an adjudicated `REFERENCE-RESOLUTION` entry on a
  committed, byte-justified allowlist in the task folder. The original "0
  divergent in either direction" bar is **superseded** — do not implement
  link/`_inlink`/`_outlink`/UOL resolution inside the parser.~~
- **`$WZ_ARCHIVE` is present and verified** (supplied by the user after the first execution session; this constraint previously read "NOT on this machine"). The 51.6 MiB PKG1 `GMS/83.1/Reactor.wz` lives at `tmp/83.1_wz/Reactor.wz` (repo-relative to the **main** checkout — see the worktree note below), alongside the other 83.1 archives (`Base.wz`, `Character.wz`, `Item.wz`, `Map.wz`, `Mob.wz`, `Npc.wz`, `Quest.wz`, `Skill.wz`, `String.wz`, `UI.wz`, and others). Verified: 51.6 MiB, first four bytes `50 4b 47 31` (`PKG1`). It is **not** committed and must not be added to git. Every command in this plan that needs it writes `"$WZ_ARCHIVE"`, so export it before running them.
- **Both external inputs live in the MAIN checkout's `tmp/`, not the task worktree's.** The worktree at `.worktrees/task-262-wz-property-reader-divergence/` has its own (empty) `tmp/`, so a bare relative `tmp/...` resolves to the wrong place from inside it. Set the variables to absolute paths, or reference the main checkout explicitly.
- **`$WZ_REFERENCE` is present and verified.** The HaRepacker XML dump lives at `tmp/083839c6-c47c-42a6-9585-76492795d123/GMS/83.1/Reactor.wz/` (repo-relative), 421 `.img.xml` files. It is **not** committed and must not be added to git.
- **Module root for Tasks 1–3, 8–14 is `libs/atlas-wz`.** Run `go build ./...` / `go test ./...` from there. Task 1 also touches `services/atlas-data/atlas.com/data` (a second module root).
- **`tools/verify.sh` on this branch fans out to all ~86 modules** because it touches `libs/` (`tools/verify.sh:196-224`). For per-task iteration gates pass `--base <last-gated-commit>`; the final flagless run is the only one that must be unscoped.
- **`tools/verify.sh` does not build or test `tools/*` Go modules** (`all_modules()`, `tools/verify.sh:181-184` scans only `services/` and `libs/`). This is why `wzdiff` lands under `libs/atlas-wz/`, not `tools/wzdiff/` — see `context.md`.
- **Locking discipline is unchanged.** `Image.Properties()` takes `i.wzFile.LockParse()` at `libs/atlas-wz/wz/image.go:89` and holds it across the whole parse chain. No new goroutines. Anything added to the parse path runs under that lock and must not re-enter the reader.
- **No per-property logging** anywhere (PRD §8 Observability). Per-image and per-archive only.
- **Do not land S3 strictness** (erroring on unknown property/extended/string-block tags). Out of scope per design §6.2.
- **Do not change `libs/atlas-wz/canvas`, `libs/atlas-wz/atlas`, or `propertyToElement`'s element mapping semantics.** Task 1 moves that mapping verbatim; it does not alter it.
- **One commit per identified defect**, each carrying its own failing fixture (design §6.1). A change that makes an image match without a byte offset explaining why is rejected at review.
- **`wztest` changes are additive only.** Every existing call site must keep compiling.

---

## Task 1: Extract the pure tree→XML mapping into `libs/atlas-wz/wz/wzxml`

`wzdiff` (Task 3) must serialize through the *ingest pipeline's own* mapping so the comparison is honest, but that mapping currently lives in module `atlas-data`, whose module path is the non-fetchable local name `atlas-data`. Depending on it from `libs/atlas-wz` is impossible (import cycle) and depending on it from a tool drags the whole service dependency graph in. The four mapping functions have **no atlas-data-local imports** (confirmed: `adapter.go:12-24` imports only stdlib, logrus, and `atlas-wz`), so the move is clean.

### Files

- `libs/atlas-wz/wz/wzxml/element.go` — **new file**; the moved pure mapping
- `libs/atlas-wz/wz/wzxml/element_test.go` — **new file**; the moved tests
- `services/atlas-data/atlas.com/data/data/wztoxml/adapter.go` — delete lines 88-163, delegate to `wzxml`
- `services/atlas-data/atlas.com/data/data/wztoxml/adapter_test.go` — drop the tests that moved; keep any that exercise `SerializeImage`
- `services/atlas-data/atlas.com/data/data/workers/runtime.go` — read-only; its `wztoxml.SerializeToDirectory(l, file, root)` call at line 101 must be unaffected

Patterns to copy: `services/atlas-data/atlas.com/data/data/wztoxml/adapter_test.go:14-60` (in-memory `property.New*` construction, no temp dirs, plus a table-driven `formatFloat` test).

Module roots: `libs/atlas-wz` **and** `services/atlas-data/atlas.com/data`. Both must build.

### Interfaces

- Produces:
  - `type wzxml.Element struct { XMLName xml.Name; Name, Value, Width, Height, X, Y string; Children []Element }` — same struct tags as today's `xmlElement` (`adapter.go:88-98`), field-for-field.
  - `func wzxml.PropertyToElement(p property.Property) Element`
  - `func wzxml.PropertiesToElements(props []property.Property) []Element`
  - `func wzxml.FormatFloat(v float64) string`
- Consumes: `property.Property` and the 13 concrete `property.*` types (`libs/atlas-wz/wz/property/property.go:30-201`).

- [x] **Step 1: Write the failing test**

`libs/atlas-wz/wz/wzxml/element_test.go`, package `wzxml`. Setup shape copied from `services/atlas-data/atlas.com/data/data/wztoxml/adapter_test.go:14-60` — construct `property.New*` values in memory, call the function, assert on returned fields. No temp dirs, no logrus.

`TestPropertyToElement` — table-driven, one row per concrete property type. Every row asserts `XMLName.Local`, `Name`, and the value-bearing attribute:

| case | input | want `XMLName.Local` | want attrs |
|---|---|---|---|
| null | `property.NewNull("a")` | `null` | Name=`a` |
| short | `property.NewShort("a", -3)` | `short` | Value=`-3` |
| int | `property.NewInt("state", 1)` | `int` | Value=`1` |
| long | `property.NewLong("a", 9000000000)` | `long` | Value=`9000000000` |
| float | `property.NewFloat("a", 1.5)` | `float` | Value=`1.5` |
| float integral | `property.NewFloat("a", 2)` | `float` | Value=`2.0` |
| double | `property.NewDouble("a", 0)` | `double` | Value=`0.0` |
| string | `property.NewString("name", "Red Potion")` | `string` | Value=`Red Potion` |
| sub | `property.NewSub("event", []property.Property{property.NewInt("state", 1)})` | `imgdir` | Name=`event`, one child `int` Value=`1` |
| canvas | `property.NewCanvas("0", 100, 121, 2, 0x1000, 64, []property.Property{property.NewVector("origin", 49, 121)})` | `canvas` | Width=`100`, Height=`121`, one child `vector` X=`49` Y=`121` |
| vector | `property.NewVector("lt", -100, -100)` | `vector` | X=`-100`, Y=`-100` |
| convex | `property.NewConvex("a", []property.Property{property.NewVector("0", 1, 2)})` | `extended` | one child |
| sound | `property.NewSound("a")` | `sound` | Name=`a`, no Value |
| uol | `property.NewUOL("0", "../0/0")` | `uol` | Value=`../0/0` |

`TestFormatFloat` — table-driven: `0 → "0.0"`, `1.5 → "1.5"`, `-2 → "-2.0"`, `100 → "100.0"`.

`TestPropertiesToElements` — `nil` input returns `nil`; a 2-element slice returns 2 elements in order.

- [x] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-wz && go test ./wz/wzxml/...`
Expected: FAIL — `no Go files in .../wz/wzxml` (the package does not exist yet).

- [x] **Step 3: Create the package by moving the code verbatim**

Create `libs/atlas-wz/wz/wzxml/element.go` with package doc, and move `adapter.go:88-163` into it **unchanged except for renames**: `xmlElement`→`Element`, `propertiesToElements`→`PropertiesToElements`, `propertyToElement`→`PropertyToElement`, `formatFloat`→`FormatFloat`, and `[]xmlElement`→`[]Element` inside the struct. Imports become `encoding/xml`, `fmt`, `strconv`, `strings`, and `github.com/Chronicle20/atlas/libs/atlas-wz/wz/property`.

Do **not** change any `XMLName.Local` value, any `omitempty`, or the `formatFloat` decimal-point rule — HaRepacker-compatible output is a wire contract with the existing `.img.xml` consumers.

- [x] **Step 4: Run test to verify it passes**

Run: `cd libs/atlas-wz && go test ./wz/wzxml/...`
Expected: PASS.

- [x] **Step 5: Delegate from `wztoxml` and delete the duplicate**

In `services/atlas-data/atlas.com/data/data/wztoxml/adapter.go`: delete lines 88-163, add the `wzxml` import, and replace the call inside `SerializeImage` that built elements with `wzxml.PropertiesToElements(...)` / `wzxml.PropertyToElement(...)`. `SerializeToDirectory` (`:30-40`), `serializeDirectory` (`:42-58`) and `SerializeImage` (`:61-86`) keep their signatures exactly — `runtime.go:101` must not change.

In `adapter_test.go`, delete the cases that now live in `wzxml`; keep anything exercising `SerializeImage` on disk.

- [x] **Step 6: Both modules build and test green**

Run: `cd libs/atlas-wz && go build ./... && go test ./...`
Run: `cd services/atlas-data/atlas.com/data && go build ./... && go test ./...`
Expected: PASS in both.

- [x] **Step 7: Commit**

```bash
git add libs/atlas-wz/wz/wzxml services/atlas-data/atlas.com/data/data/wztoxml
git commit -m "refactor(atlas-wz): move pure property->XML mapping to wz/wzxml"
```

---

## Task 2: Nil-by-default trace hook on `*File`

The defect is silent because the type-9 branch reseeks to `endPos` unconditionally (`libs/atlas-wz/wz/image.go:285-287`), healing any drift a child introduced. A trace that records where each node's decode started and ended is what turns "the tree is wrong" into "the stream diverged at offset X." This is the mechanism design §4.2 chose over printf tracing and over a second parser.

### Files

- `libs/atlas-wz/wz/trace.go` — **new file**; `TraceEvent`, `SetTrace`, the `emit` helper
- `libs/atlas-wz/wz/trace_test.go` — **new file**
- `libs/atlas-wz/wz/file.go` — add the `trace` field to `File` (`:29-50`); propagate in `NewSubFile` (`:115-129`)
- `libs/atlas-wz/wz/image.go` — emit sites in `parsePropertyList` (`:168-198`), `parsePropertyValue` (`:201-294`), `parseExtendedProperty` (`:297-361`), `parseCanvasProperty` (`:364-433`)
- `libs/atlas-wz/wz/reader.go` — read-only; `ReadWzStringBlock` at `:312-347` is what the `stringblock` detail describes
- `libs/atlas-wz/wz/parse_race_test.go` — read-only; its invariants must still hold

Patterns to copy: `libs/atlas-wz/wz/fixture_roundtrip_test.go:16-27` (`writeFixture`) and `:43-66` (build → `Open` → walk → `Properties()`).

Module root: `libs/atlas-wz`.

### Interfaces

- Produces:
  - `type wz.TraceEvent struct { Path, Kind, Name string; Type byte; StartOff, EndOff int64; Detail string }`
  - `func (wz *File) SetTrace(fn func(TraceEvent))`
- Consumes: `File.LockParse()` (`libs/atlas-wz/wz/file.go:93-99`), `Reader.Pos()` (`libs/atlas-wz/wz/reader.go:37`).

**Concurrency contract (must be in the doc comment, verbatim intent):** `SetTrace` must be called before any `Properties()` call on the file or any of its sub-files. The hook fires synchronously under `File.parseMu` and must not re-enter the reader or call back into `wz`. It is `nil` in production; every emit site is guarded by a single `if wz.trace != nil` with no allocation on the nil path.

`NewSubFile` (`libs/atlas-wz/wz/file.go:115-129`) copies fields explicitly and delegates `parseMu`/`registerImageKey` to `parent`. Follow the same delegation shape for `trace` so the hook fires once per node, not once per view.

- [x] **Step 1: Write the failing test**

`libs/atlas-wz/wz/trace_test.go`, package `wz`.

`TestSetTraceEmitsNodeEvents` — build a GMS fixture with `wztest.NewBuilder().SetVersion(83).SetEncryption(crypto.EncryptionGMS)` containing one image `Img("0200", Sub("info", Int("state", 1), Str("name", "x")))`, write it with `writeFixture`, `Open`, `SetTrace` collecting into a `[]TraceEvent` slice, then call `Properties()`. Assert:

| assertion | expected |
|---|---|
| an event exists with `Path == "/info"` and `Kind == "extended"` | true |
| an event exists with `Path == "/info/state"`, `Kind == "prop"`, `Type == 3` | true |
| an event exists with `Path == "/info/name"`, `Kind == "prop"`, `Type == 8` | true |
| for every event | `EndOff >= StartOff` |
| at least one event has `Kind == "stringblock"` and a non-empty `Detail` | true |
| events are emitted in decode order (`StartOff` non-decreasing across sibling `prop` events) | true |

`TestTraceNilByDefault` — same fixture, never call `SetTrace`, call `Properties()`, assert no panic and the returned tree matches the no-trace baseline (2 properties under `info`).

`TestSetTraceOnSubFileDelegatesToParent` — copy the sub-file construction shape from `libs/atlas-wz/wz/subfile_test.go`; assert a trace set on the parent fires for a sub-file's image and that events are not duplicated.

- [x] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-wz && go test ./wz/ -run 'TestSetTrace|TestTraceNil'`
Expected: FAIL — `undefined: TraceEvent`, `f.SetTrace undefined`.

- [x] **Step 3: Add the type, field, and setter**

`libs/atlas-wz/wz/trace.go`: declare `TraceEvent`, `SetTrace`, and an unexported `func (wz *File) emitTrace(ev TraceEvent)` that resolves through `wz.parent` when set and returns immediately when the resolved hook is nil.

`libs/atlas-wz/wz/file.go`: add `trace func(TraceEvent)` to the `File` struct (`:29-50`); no new mutex — the documented contract is set-before-use, matching how `encryptionKey`/`versionHash` are set once during `Open` and never mutated after publication.

- [x] **Step 4: Add the emit sites**

Thread a `path string` argument down the parse chain (`parsePropertyList` → `parsePropertyValue` → `parseExtendedProperty` → `parseCanvasProperty`), rooted at `""` from `parseWithKey` (`image.go:151`). Emit:

- `parsePropertyList` (`image.go:168-198`): one `Kind: "list"` event per list, `StartOff` = position before the `ReadWzInt` count, `EndOff` = position after the final child, `Detail` = `fmt.Sprintf("count=%d", count)`.
- `parsePropertyList` per child: one `Kind: "stringblock"` event for the name read at `:178`, with `Detail` carrying the tag byte, the resolved absolute offset when the tag is `0x01`/`0x1B`, and the resulting string.
- `parsePropertyValue` (`image.go:201-294`): one `Kind: "prop"` event per scalar case with `Type` = the raw `propType` byte. For the type-9 branch (`:264-289`) additionally emit `Kind: "sub"` with `Detail` = `fmt.Sprintf("declaredSize=%d endPos=%d actualEnd=%d", size, endPos, actualEnd)` where `actualEnd` is `r.Pos()` **immediately before** the recovery reseek at `:285`. **This one field is the whole point of the harness** — it is what makes an over- or under-read visible.
- `parseExtendedProperty` (`image.go:297-361`): `Kind: "extended"`, `Detail` = the tag string that was read at `:300`.
- `parseCanvasProperty` (`image.go:364-433`): `Kind: "canvas"`, `Detail` = `fmt.Sprintf("w=%d h=%d format=%d dataOffset=%d dataSize=%d hasProperty=%d", ...)`.

`ReadWzStringBlock` emission stays in `image.go` at each of its five call sites (`:141`, `:178`, `:258`, `:300`, `:349`) rather than inside `reader.go` — the `*Reader` has no access to `*File` and must not gain one.

- [x] **Step 5: Run the tests**

Run: `cd libs/atlas-wz && go test ./wz/ -run 'TestSetTrace|TestTraceNil'`
Expected: PASS.

- [x] **Step 6: Prove the existing concurrency contracts still hold**

Run: `cd libs/atlas-wz && go test -race ./...`
Expected: PASS, including `TestLockParseIsExclusive` (`parse_race_test.go:29-63`), `TestPropertiesFastPathSkipsLock` (`:77-92`), `TestPropertiesConcurrentParse` (`:105-138`), `TestImageNameStripsDotImg` (`iteration_contract_test.go:22-45`) and `TestNewFileWithRootRoundTrip` (`:52-79`).

`TestPropertiesFastPathSkipsLock` asserts `Properties()` on an already-parsed `NewParsedImage` never touches `wzFile`. Do not add a trace emit on that fast path — `Image.Properties()` (`image.go:81-102`) must keep returning early before touching `i.wzFile`.

- [x] **Step 7: Commit**

```bash
git add libs/atlas-wz/wz/trace.go libs/atlas-wz/wz/trace_test.go libs/atlas-wz/wz/file.go libs/atlas-wz/wz/image.go
git commit -m "feat(atlas-wz): opt-in parse trace hook for decode diagnosis"
```

---

## Task 3: `wzdiff` core — tree model, XML loader, structural differ

The pure, unit-testable half of the differ. It reads both sides into one tree shape and reports deltas in the same `/imgdir:0/imgdir:event/int:state | value=1` notation the evidence file already uses, so a `wzdiff` failure reads like `evidence-wz-parse-divergence-reactor.txt`.

### Files

- `libs/atlas-wz/wzdiff/node.go` — **new file**; the common tree type and path rendering
- `libs/atlas-wz/wzdiff/xmlload.go` — **new file**; HaRepacker `.img.xml` → `[]Node`
- `libs/atlas-wz/wzdiff/diff.go` — **new file**; bidirectional structural diff
- `libs/atlas-wz/wzdiff/node_test.go` — **new file**
- `libs/atlas-wz/wzdiff/xmlload_test.go` — **new file**
- `libs/atlas-wz/wzdiff/diff_test.go` — **new file**
- `libs/atlas-wz/wz/wzxml/element.go` — created by Task 1; read-only here. `Element` is the in-memory side's input
- `docs/tasks/task-262-wz-property-reader-divergence/evidence-wz-parse-divergence-reactor.txt` — read-only; the output format to match

Patterns to copy: `libs/atlas-wz/wz/wzxml/element_test.go` (Task 1) for the table-driven shape; `tools/packet-audit/cmd/diff_shape_test.go` for a structural-diff test's assertion style.

Module root: `libs/atlas-wz`.

### Interfaces

- Produces:
  - `type wzdiff.Node struct { Kind, Name string; Attrs map[string]string; Children []Node }` — `Kind` is the XML local name (`imgdir`, `int`, `canvas`, `vector`, `uol`, `string`, `short`, `long`, `float`, `double`, `sound`, `extended`, `null`).
  - `func wzdiff.FromElements(els []wzxml.Element) []Node`
  - `func wzdiff.LoadImageXML(path string) ([]Node, error)`
  - `type wzdiff.Delta struct { Path string; Attrs string; OnlyIn string }` — `OnlyIn` is `"reference"` or `"ours"`.
  - `func wzdiff.Diff(ours, reference []Node) []Delta`
  - `func (d Delta) String() string` — renders `      /imgdir:0/int:state | value=1` (6-space indent, matching the evidence file).
- Consumes: `wzxml.Element` (Task 1).

Path rendering rule, taken from the evidence file: a node contributes `/<Kind>:<Name>` and the root image element itself contributes nothing. Attribute rendering: `value=X` for value-bearing kinds, `height=H width=W` for `canvas` (that attribute order, matching the evidence), `x=X y=Y` for `vector`, empty string for containers.

- [x] **Step 1: Write the failing tests**

`node_test.go` — `TestNodePath`:

| case | node chain | want path |
|---|---|---|
| single container | `imgdir` named `0` | `/imgdir:0` |
| nested | `imgdir:0` → `imgdir:event` → `int:state` | `/imgdir:0/imgdir:event/int:state` |
| canvas child | `imgdir:1` → `canvas:0` → `vector:origin` | `/imgdir:1/canvas:0/vector:origin` |

`TestDeltaString`:

| case | Delta | want |
|---|---|---|
| int | Path `/imgdir:0/int:state`, Attrs `value=1` | `      /imgdir:0/int:state \| value=1` |
| container | Path `/imgdir:0/imgdir:event`, Attrs `""` | `      /imgdir:0/imgdir:event \| ` |
| canvas | Path `/imgdir:1/canvas:0`, Attrs `height=121 width=100` | `      /imgdir:1/canvas:0 \| height=121 width=100` |

`xmlload_test.go` — `TestLoadImageXML`: write a temp `.img.xml` with `t.TempDir()` containing exactly the header and body shape of a real dump:

```xml
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="2406000.img">
  <imgdir name="info">
    <string name="name" value="x"/>
    <int name="activateByTouch" value="1"/>
  </imgdir>
  <imgdir name="0">
    <canvas name="0" width="100" height="121">
      <vector name="origin" x="49" y="121"/>
    </canvas>
  </imgdir>
</imgdir>
```

Assert the loader returns 2 top-level nodes (`imgdir:info`, `imgdir:0`), that `info` has 2 children in document order with `Kind` `string` then `int`, and that the canvas node's `Attrs` carries `width=100` and `height=121` and one `vector` child with `x=49 y=121`. Assert the outer `<imgdir name="2406000.img">` wrapper is stripped (it is the image, not a property).

`diff_test.go` — `TestDiff`, table-driven, one row per shape the evidence contains:

| case | ours | reference | want |
|---|---|---|---|
| identical | `int:state=1` | `int:state=1` | no deltas |
| lost subtree | `[]` | `imgdir:event` + `imgdir:event/imgdir:0` + `int:state=1` | 3 deltas, all `OnlyIn=="reference"`, paths in document order |
| gained subtree | `imgdir:event`+`int:timeOut=2000` | `[]` | 2 deltas, all `OnlyIn=="ours"` |
| mangled name | `int:rpeat=1` | `int:repeat=1` | 2 deltas: `/int:repeat` only-in-reference, `/int:rpeat` only-in-ours |
| wrong scalar | `int:state=1` | `int:state=0` | 2 deltas at the same path, one each direction, `Attrs` `value=1` / `value=0` |
| collapsed canvas | `canvas:0 height=1 width=1` | `canvas:0 height=121 width=100` | 2 deltas at `/canvas:0`, one each direction |
| uol vs canvas | `uol:0 value=../0/0` | `canvas:0 height=1 width=1` | 2 deltas (different `Kind` at the same name ⇒ two distinct paths) |
| ordering-insensitive | `[int:a=1, int:b=2]` | `[int:b=2, int:a=1]` | no deltas |

The last row pins a decision: **the diff is set-based over `(path, attrs)`, not order-sensitive.** HaRepacker's dump order is not our decode order and the evidence file already compares as sets.

- [x] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-wz && go test ./wzdiff/...`
Expected: FAIL — `no Go files in .../wzdiff`.

- [x] **Step 3: Implement `node.go`, `xmlload.go`, `diff.go`**

`xmlload.go` uses `encoding/xml`'s `Decoder` token stream (not struct unmarshalling) so unknown attributes and element names are preserved into `Attrs`/`Kind` rather than dropped — a dropped attribute would make the differ lie.

`diff.go` flattens both trees to `map[string]string` (path → attrs), then walks both key sets. A path in one side only yields one `Delta`; a path in both with differing attrs yields two `Delta`s, one per direction. Emit in sorted path order for stable output.

- [x] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-wz && go test ./wzdiff/...`
Expected: PASS.

- [x] **Step 5: Commit**

```bash
git add libs/atlas-wz/wzdiff
git commit -m "feat(atlas-wz): wzdiff structural tree model, XML loader and differ"
```

---

## Task 4: `wzdiff` CLI and the reference-resolution allowlist

Makes the harness runnable end-to-end and gives design §2.3's allowlist a machine-checked home, so the acceptance bar is enforced by a command rather than by prose.

### Files

- `libs/atlas-wz/wzdiff/allowlist.go` — **new file**
- `libs/atlas-wz/wzdiff/allowlist_test.go` — **new file**
- `libs/atlas-wz/wzdiff/run.go` — **new file**; the archive-vs-dump orchestration
- `libs/atlas-wz/cmd/wzdiff/main.go` — **new file**; flag parsing only
- `libs/atlas-wz/wz/wzxml/element.go` — created by Task 1; read-only here
- `libs/atlas-wz/wzdiff/diff.go` — created by Task 3; read-only here

Patterns to copy: `libs/atlas-constants/gen/wzsnapshot/cmd/mksnapshot/main.go` — the repo's precedent for a `cmd/` binary living inside a `libs/` module, which is why `wzdiff` is placed here rather than under `tools/` (see Global Constraints).

Module root: `libs/atlas-wz`.

### Interfaces

- Produces:
  - `type wzdiff.AllowEntry struct { Image, Path, OnlyIn, Reason string }`
  - `func wzdiff.LoadAllowlist(path string) ([]AllowEntry, error)` — YAML-free; one entry per line, tab-separated `image<TAB>path<TAB>onlyIn<TAB>reason`, `#` comments and blank lines skipped. No new module dependency.
  - `func wzdiff.Allowed(entries []AllowEntry, image string, d Delta) bool`
  - `type wzdiff.Result struct { ImagesOurs, ImagesReference int; Divergent map[string][]Delta; Allowed int }`
  - `func wzdiff.Run(l logrus.FieldLogger, archivePath, referenceDir string, allow []AllowEntry) (Result, error)`
- Consumes: `wz.Open` (`libs/atlas-wz/wz/file.go:132`), `File.Root()` (`:181`), `Directory.Images()` / `Directory.Directories()` (`libs/atlas-wz/wz/directory.go:29,34`), `Image.Properties()` (`libs/atlas-wz/wz/image.go:81`), `wzxml.PropertiesToElements` (Task 1), `wzdiff.FromElements` / `LoadImageXML` / `Diff` (Task 3).

`Run` walks the opened archive's image tree, converts each image's `Properties()` through `wzxml.PropertiesToElements` → `wzdiff.FromElements`, loads `<referenceDir>/<image>.img.xml`, diffs, and drops any delta matched by the allowlist. It records images present on only one side separately from property deltas — the 419-vs-421 gap must surface as its own number, exactly as `evidence-wz-parse-divergence-reactor.txt:1-3` does.

CLI contract:

```
wzdiff --archive <path.wz> --reference <harepacker-dump-dir> [--allowlist <file>] [--trace <image>]
```

Exit **0** only when there are zero unallowlisted deltas and both sides enumerate the same image set. Exit **1** otherwise, after printing a report in the evidence file's format. `--trace <image>` installs `File.SetTrace` (Task 2) and dumps the trace for that one image to stdout — this is the FR-1 tool.

- [x] **Step 1: Write the failing test**

`allowlist_test.go` — `TestLoadAllowlist`: write a temp file with a comment line, a blank line, and two entries; assert 2 entries parsed with fields intact and that a malformed 3-field line returns an error naming the line number.

`TestAllowed` — table-driven:

| case | entry | delta | want |
|---|---|---|---|
| exact match | `2519002` / `/imgdir:0` / `reference` | same image, same path, `OnlyIn=reference` | true |
| wrong direction | same entry | `OnlyIn=ours` | false |
| wrong image | same entry | image `2519003` | false |
| prefix, not exact | entry path `/imgdir:0` | delta path `/imgdir:0/canvas:0` | **true** — a resolved reference substitutes a whole subtree, so an allowlist entry covers its descendants |
| unrelated | same entry | `/imgdir:1/int:state` | false |

`run_test.go` — `TestRunReportsImageSetMismatch`: build a 1-image archive with `wztest`, write a reference dir containing 2 `.img.xml` files, assert `Result.ImagesOurs == 1`, `Result.ImagesReference == 2`, and that `Run` reports the missing image by name.

`TestRunCleanArchiveHasNoDeltas`: build a `wztest` archive with `Sub("info", Int("state", 1))`, serialize a matching reference `.img.xml` by hand into `t.TempDir()`, assert `len(Result.Divergent) == 0`.

- [x] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-wz && go test ./wzdiff/ -run 'TestLoadAllowlist|TestAllowed|TestRun'`
Expected: FAIL — `undefined: LoadAllowlist`, `undefined: Run`.

- [x] **Step 3: Implement `allowlist.go`, `run.go`, and `cmd/wzdiff/main.go`**

`main.go` contains flag parsing, a `logrus.New()` logger, a call to `wzdiff.Run`, report printing, and `os.Exit`. No logic — all of it is in the testable package.

- [x] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-wz && go test ./wzdiff/...`
Expected: PASS.

- [x] **Step 5: Verify the binary builds and its help is correct**

Run: `cd libs/atlas-wz && go build ./... && go run ./cmd/wzdiff --help`
Expected: build succeeds; usage lists `--archive`, `--reference`, `--allowlist`, `--trace`.

- [x] **Step 6: Commit**

```bash
git add libs/atlas-wz/wzdiff libs/atlas-wz/cmd/wzdiff
git commit -m "feat(atlas-wz): wzdiff CLI and reference-resolution allowlist"
```

---

## Task 5: Reference-fidelity gate (design §2.2) — **requires `$WZ_ARCHIVE`**

**This is a diagnosis task. It changes no product code.** Its output decides which of the 19 divergences are ours to fix, and it is design §10's approved gate. Do not start Task 12 without it.

### Files

- `docs/tasks/task-262-wz-property-reader-divergence/reference-fidelity.md` — **new file**; the deliverable
- `docs/tasks/task-262-wz-property-reader-divergence/allowlist.tsv` — **new file**; the machine-readable §2.3 allowlist
- `docs/tasks/task-262-wz-property-reader-divergence/evidence-wz-parse-divergence-reactor.txt` — read-only; the 19 images and their exact deltas
- `libs/atlas-wz/cmd/wzdiff/main.go` — created by Task 4; read-only here. The tool you run
- `services/atlas-data/atlas.com/data/reactor/reader.go` — read-only; line 65 `info.GetString("link", "")` is the consumer that already resolves links
- `services/atlas-data/atlas.com/data/map/reader.go` — read-only; line 60, same
- `libs/atlas-wz/icons/extract.go` — read-only; line 222 `findInfoLink`, same

Module root: `libs/atlas-wz` (for running `wzdiff`).

### Interfaces

- Consumes: `wzdiff` (Tasks 3-4), the trace hook (Task 2).
- Produces: a `PARSER-DEFECT` / `REFERENCE-RESOLUTION` / `MIXED` label with byte evidence for each of the 19 images and the 2 missing ones; `allowlist.tsv` in `wzdiff.LoadAllowlist` format.

- [x] **Step 1: Produce the baseline diff**

Run:

```bash
cd libs/atlas-wz && go run ./cmd/wzdiff \
  --archive "$WZ_ARCHIVE" \
  --reference ../../tmp/083839c6-c47c-42a6-9585-76492795d123/GMS/83.1/Reactor.wz \
  > /tmp/wzdiff-baseline.txt; echo "exit=$?"
```

Expected: exit 1, 19 divergent images, image-set mismatch 419 vs 421. **Confirm this reproduces `evidence-wz-parse-divergence-reactor.txt` before trusting the tool.** If the counts differ, the tool is wrong, not the evidence — stop and fix Task 3/4.

- [x] **Step 2: Adjudicate each disputed image from the bytes**

The disputed set is the one design §2.1 names: `2519000`, `2519001`, `2519002`, `2519003`, `2006001`, plus `2618000` (MIXED-suspect) and `3002000` (our parse *gains* a subtree).

For each, in this order of authority:

1. **Byte adjudication.** Run `go run ./cmd/wzdiff --archive "$WZ_ARCHIVE" --reference <dir> --trace <image>` to get the node-by-node offsets, then `xxd -s <StartOff> -l 128 "$WZ_ARCHIVE"` at the offset of the disputed node and hand-decode against `libs/atlas-wz/wz/image.go:201-361`. The archive bytes outrank both parsers.
2. Record which HaRepacker export setting produced the reference dump, and whether a literal-mode dump exists. Recorded, **not** trusted over (1).
3. Only if (1) is ambiguous, cross-check against a third literal reader.

The specific questions the bytes must answer:

| image | question |
|---|---|
| `2519002`, `2519003` | Does the archive contain a literal `/imgdir:info/string:link = 2519000` and no `imgdir:0` body? If yes → `REFERENCE-RESOLUTION`; our two-node parse is correct. |
| `2519000`, `2519001` | Is `1/canvas:0` genuinely a 1×1 stub in the bytes (`_inlink`/`_outlink` storage)? If yes → `REFERENCE-RESOLUTION`. |
| `2006001` | Two separate questions. (a) Is `/imgdir:1/uol:0 = ../0/0` literally in the bytes, with the reference showing a resolved `canvas:0`? (b) Do `imgdir:2` … `imgdir:7` exist in the bytes? We emit them and the reference does not — if the bytes have them, that half is `REFERENCE-RESOLUTION`; if not, it is `PARSER-DEFECT` and this image is `MIXED`. |
| `2618000` | Our `/imgdir:6/hit/canvas:*` content is byte-identical to the reference's `/imgdir:7/hit/canvas:*`. Confirm from the bytes whether the *names* `6`/`7` are misread (a name defect, same family as `rpeat`) rather than the structure being shifted. Also adjudicate `7/canvas:0` `delay`: we read `50000`, the reference reads `150`. |
| `3002000` | Do the bytes contain the `/imgdir:1/imgdir:event` subtree with `timeOut=2000` that only our parse produces? If yes → **we are right and the reference is skipping a record it does not model** (PRD Open Question 4, answered in the opposite direction). |

The remaining 12 images (`2406000`, `2408001`, `2502002`, `2618003`–`2618007`, `9202005`, `9208003`, `9208004`, `2006000`) are pure loss or a mangled name with no `link`/`uol` in play. Confirm each is `PARSER-DEFECT` from its trace — do not assume it. Sweep all 19; a spot-check presented as a full adjudication is a false verified.

- [x] **Step 3: Write `reference-fidelity.md`**

One section per image: the label, the byte offset(s) inspected, the hexdump excerpt, and the decode that justifies the label. Include an explicit table of all 21 items (19 divergent + `9400300` + `9400301`) with their labels, so the count is auditable at a glance.

State plainly which of PRD FR-5's table rows are **withdrawn** as reference-resolution rather than defects. That is the concrete consequence of the §2.3 restatement and a reviewer must be able to see it without re-deriving it.

- [ ] **Step 4: Write `allowlist.tsv`**

One line per `REFERENCE-RESOLUTION` delta, `image<TAB>path<TAB>onlyIn<TAB>reason`, the reason naming the byte evidence section in `reference-fidelity.md`. A `MIXED` image contributes allowlist lines only for its reference-resolution half.

- [ ] **Step 5: Verify the allowlist parses and shrinks the diff**

Run:

```bash
cd libs/atlas-wz && go run ./cmd/wzdiff \
  --archive "$WZ_ARCHIVE" \
  --reference ../../tmp/083839c6-c47c-42a6-9585-76492795d123/GMS/83.1/Reactor.wz \
  --allowlist ../../docs/tasks/task-262-wz-property-reader-divergence/allowlist.tsv
```

Expected: still exit 1 (the parser defects are unfixed), but the divergent count is reduced by exactly the allowlisted images, and no `REFERENCE-RESOLUTION` delta remains in the report.

- [ ] **Step 6: Commit**

```bash
git add docs/tasks/task-262-wz-property-reader-divergence/reference-fidelity.md \
        docs/tasks/task-262-wz-property-reader-divergence/allowlist.tsv
git commit -m "docs(task-262): reference-fidelity adjudication and resolution allowlist"
```

- [ ] **Step 7: CHECKPOINT — report to the controller**

Report the label counts and, specifically, whether any PRD FR-5 row was withdrawn. If `3002000` came back "we are right," say so explicitly — it inverts an acceptance criterion.

---

## Task 6: Byte-level diagnosis of the parser defects (FR-1, FR-2) — **WITHDRAWN**

**WITHDRAWN.** Depended on Task 5 producing a `PARSER-DEFECT` set to diagnose. Task 5
found zero (`reference-fidelity.md`, 21/21 `INPUT-MISMATCH`) — there are no parser
defects to diagnose an offset for. Text below kept as the original task description.

**Diagnosis only; changes no product code.** Task 12's patch follows these offsets, and design §6.1 rejects at review any fix without one.

### Files

- `docs/tasks/task-262-wz-property-reader-divergence/diagnosis.md` — **new file**; the deliverable
- `docs/tasks/task-262-wz-property-reader-divergence/reference-fidelity.md` — created by Task 5; read-only here. Its labels scope this work to the `PARSER-DEFECT` set
- `libs/atlas-wz/wz/image.go` — read-only
- `libs/atlas-wz/wz/reader.go` — read-only
- `libs/atlas-wz/cmd/wzdiff/main.go` — created by Task 4; read-only here. `--trace`

Module root: `libs/atlas-wz`.

### Interfaces

- Consumes: Task 2's trace, Task 4's `--trace`, Task 5's labels.
- Produces: `diagnosis.md` — per defect, the decode step (`file.go:line`), the byte offset, the hexdump, and the class label; plus a mapping from every `PARSER-DEFECT` image to the defect that explains it.

- [ ] **Step 1: Diagnose `2406000` — the cleanest case (FR-1's named requirement)**

`2406000` loses exactly five nodes and gains nothing: `/imgdir:info/int:activateByTouch = 1`, `/imgdir:0/imgdir:event`, `/imgdir:0/imgdir:event/imgdir:0`, and that node's `int:state = 1` and `int:type = 6`.

Note the shape before you start: **both losses are the last child of their list.** `activateByTouch` is the final entry of `info`; `event` is the final entry of `imgdir:0`. That points at either a short `count` from `ReadWzInt` (`image.go:171`) or a child whose decode over-consumed and left the list loop unable to read one more entry. The trace's `declaredSize`/`actualEnd` pair on the type-9 branch (Task 2, Step 4) distinguishes these directly.

Run `go run ./cmd/wzdiff --archive "$WZ_ARCHIVE" --reference <dir> --trace 2406000` and record: the `count` the `info` list read, the `count` the reference implies, and the `StartOff`/`EndOff` of every child. Then `xxd` the list header and hand-decode the count field. **Write the offset down.**

- [ ] **Step 2: Diagnose `2408001` — the mangled name**

`/imgdir:1/int:repeat` reads as `/imgdir:1/int:rpeat`. Names come from `ReadWzStringBlock` at `image.go:178`, which for tags `0x01`/`0x1B` seeks to `fileStart + offset` where `fileStart` is always `imageOffset` (`reader.go:333`). Capture the trace's `stringblock` `Detail` for that node — it carries the tag byte and the resolved absolute offset. `xxd` that offset and decode the length-prefixed string by hand against `ReadWzString` (`reader.go:218-241`, the `int8` tag switch). Record whether the defect is in the offset base, in the length byte, or in the ASCII/unicode branch selection.

- [ ] **Step 3: Diagnose `2502002` — the wrong scalars**

`/imgdir:0/imgdir:event/imgdir:0/int:state` and `.../imgdir:1/int:state` read `1` where the archive should give `0`, and `/imgdir:0/imgdir:event/imgdir:1/int:2 = 1` is lost entirely. A wrong scalar *plus* a lost sibling in the same list is the signature of a misaligned read, not a bad int decode — get the offsets and confirm.

- [ ] **Step 4: Diagnose the remaining `PARSER-DEFECT` images**

Every image Task 5 labelled `PARSER-DEFECT` or `MIXED`, individually: `2006000`, the `PARSER-DEFECT` half of `2006001`, `2618000`, `2618003`, `2618004`, `2618005`, `2618006`, `2618007`, `9202005`, `9208003`, `9208004`, `3002000` (if the bytes said the reference is right), and any of `2519000`–`2519003` not fully allowlisted. Assign each to a defect from Steps 1-3 or open a new defect entry with its own offset. **FR-2 forbids "fixed some, assumed the rest"** — an image with no offset recorded is not diagnosed.

- [ ] **Step 5: Write `diagnosis.md`**

Structure: one `## Defect N` section per distinct root cause, each with the decode step (`libs/atlas-wz/wz/<file>.go:<line>`), the reproducing byte offset with hexdump, the hand-decode, and the list of images it explains. Close with a coverage table: every `PARSER-DEFECT` image → the defect number that explains it, with no blanks.

- [ ] **Step 6: Commit**

```bash
git add docs/tasks/task-262-wz-property-reader-divergence/diagnosis.md
git commit -m "docs(task-262): byte-level diagnosis of wz property decode defects"
```

- [ ] **Step 7: CHECKPOINT — report to the controller**

Report the defect count, each defect's one-line cause and `file:line`, and the image→defect coverage. The controller re-plans Task 12 from this.

---

## Task 7: Diagnose the enumeration gap (FR-3) — **WITHDRAWN**

**WITHDRAWN.** `9400300.img` and `9400301.img` are absent from `$WZ_ARCHIVE`'s own
directory entirely (`reference-fidelity.md`) — they are not silently dropped by
`parseDirectory`; there is no enumeration bug to diagnose. Text below kept as the
original task description.

`9400300.img` and `9400301.img` are never enumerated. Design §5 gives three candidates in elimination order; the cheapest one is first and must actually be run, not reasoned about.

### Files

- `docs/tasks/task-262-wz-property-reader-divergence/diagnosis.md` — append an `## Enumeration gap` section
- `libs/atlas-wz/wz/directory.go` — read-only; `parseDirectory` at `:39-137`
- `libs/atlas-wz/wz/file.go` — read-only; `parseRoot` at `:525-538`, called from `Open` at `:156`

Module root: `libs/atlas-wz`.

- [ ] **Step 1: Eliminate the swallowed sub-directory**

`parseDirectory` logs and drops a failed recursive parse at `libs/atlas-wz/wz/directory.go:122` (`Warnf("Unable to parse sub-directory [%s].", entryName)`) and continues, losing every image beneath it. Open the archive with a debug-level logger and grep the output:

```bash
cd libs/atlas-wz && go run ./cmd/wzdiff --archive "$WZ_ARCHIVE" --reference <dir> 2>&1 \
  | grep -i "unable to parse sub-directory"
```

If any line appears, that is the cause. Record the sub-directory name and the underlying error.

- [ ] **Step 2: Eliminate the `elemType == 1` width**

`directory.go:57-62` skips 10 bytes and `continue`s for `elemType == 1`, taking no size/checksum/offset trailer. Confirm from the bytes whether any type-1 entry precedes real entries in this archive's listing, and whether the 10-byte width matches what actually follows. A width mismatch desyncs every subsequent entry.

- [ ] **Step 3: Eliminate a short `count`**

`count` is read at `directory.go:42`. 419 vs 421 with the two missing images being the *last* two is consistent with a short count. Compare the decoded `count` against the reference's 421 and `xxd` the count field.

- [ ] **Step 4: Record the finding**

Append to `diagnosis.md`: which candidate was confirmed, the offset or log line proving it, and which candidates were eliminated and how. State explicitly whether the fix also requires propagating the `:122` swallow (design §5 says it does regardless — "we silently enumerated 419 of 421 images" is the enumeration-level instance of the same corruption class).

- [ ] **Step 5: Commit**

```bash
git add docs/tasks/task-262-wz-property-reader-divergence/diagnosis.md
git commit -m "docs(task-262): diagnose the 419-vs-421 enumeration gap"
```

---

## Task 8: `wztest.Builder` — scalar, vector, UOL and convex property kinds

The builder today emits only `KindInt`, `KindString`, `KindSub`, `KindCanvas` (`libs/atlas-wz/wztest/builder.go:18-23`). The divergences are full of `vector`, `uol`, and typed scalars, so none of the regression fixtures in Task 11 can be synthesized without these. Additive only: every new name is net-new (verified — no existing `Null`/`Short`/`Long`/`Float`/`Double`/`Vector`/`UOL`/`Convex` in `libs/atlas-wz`).

### Files

- `libs/atlas-wz/wztest/builder.go` — add `Kind` constants and constructors; add `case` arms in `writePropList` (`:148-198`)
- `libs/atlas-wz/wz/wztest_kinds_test.go` — **new file**; round-trip each new kind through the real parser
- `libs/atlas-wz/wz/image.go` — read-only; `parsePropertyValue:201-294` and `parseExtendedProperty:297-361` define the exact encodings

Patterns to copy: `libs/atlas-wz/wztest/builder.go:154-163` (the `KindInt`/`KindString` switch arms) for scalars, and `:164-174` (`KindSub`) for anything wrapped in a type-9 extended object. Test setup: `libs/atlas-wz/wz/fixture_roundtrip_test.go:16-27` (`writeFixture`), `:29-38` (`findProp`), `:43-66` (build → `Open` → `Properties()`).

Module root: `libs/atlas-wz`.

### Interfaces

- Produces (all in package `wztest`):
  - `KindNull`, `KindShort`, `KindLong`, `KindFloat`, `KindDouble`, `KindVector`, `KindUOL`, `KindConvex` added to the `Kind` enum
  - `func Null(name string) Prop`
  - `func Short(name string, v int16) Prop`
  - `func Long(name string, v int64) Prop`
  - `func Float(name string, v float32) Prop`
  - `func Double(name string, v float64) Prop`
  - `func Vector(name string, x, y int32) Prop`
  - `func UOL(name, target string) Prop`
  - `func Convex(name string, children ...Prop) Prop`
  - new `Prop` fields to carry the payloads (`Short int16`, `Long int64`, `Float float32`, `Double float64`, `X, Y int32`, reusing `Str` for the UOL target and `Children` for convex)
- Consumes: nothing from earlier tasks.

**Exact encodings the writer must emit** (read off `image.go`; get these wrong and the fixture proves nothing):

| kind | bytes after the name's string block |
|---|---|
| `Null` | tag byte `0`, no payload (`image.go:205-207`) |
| `Short` | tag byte `2`, then `int16` little-endian (`:209-215`) |
| `Int` | tag byte `3`, then WzInt (existing, `:217-223`) |
| `Long` | tag byte `20`, then WzLong (`:225-231`) |
| `Float` | tag byte `4`, then byte `0x80`, then `float32` little-endian (`:233-245` — a leading byte other than `0x80` means zero, so the fixture must write `0x80`) |
| `Double` | tag byte `5`, then `float64` little-endian (`:247-253`) |
| `String` | tag byte `8`, then a string block (existing, `:255-261`) |
| `Vector` | tag byte `9`, `int32` inner length, then inner = string block `"Shape2D#Vector2D"` + WzInt x + WzInt y (`:319-328`) |
| `UOL` | tag byte `9`, `int32` inner length, then inner = string block `"UOL"` + **1 skipped byte** + string block target (`:345-353`) |
| `Convex` | tag byte `9`, `int32` inner length, then inner = string block `"Shape2D#Convex2D"` + WzInt child count + each child encoded as a bare extended property (string block tag + payload, **no name, no length prefix** — `:330-343` recurses into `parseExtendedProperty` directly) |

- [x] **Step 1: Write the failing test**

`libs/atlas-wz/wz/wztest_kinds_test.go`, package `wz`. `TestBuilderEmitsAllPropertyKinds` — one GMS fixture containing every new kind, round-tripped through the real parser.

Setup copied from `libs/atlas-wz/wz/fixture_roundtrip_test.go:43-66`. Build:

```
wztest.NewBuilder().SetVersion(83).SetEncryption(crypto.EncryptionGMS).
  AddImage(wztest.Img("kinds",
    wztest.Null("n"),
    wztest.Short("s", -3),
    wztest.Int("i", 42),
    wztest.Long("l", 9000000000),
    wztest.Float("f", 1.5),
    wztest.Double("d", 2.25),
    wztest.Str("str", "hello"),
    wztest.Vector("lt", -100, -100),
    wztest.UOL("u", "../0/0"),
    wztest.Convex("cv", wztest.Vector("0", 1, 2), wztest.Vector("1", 3, 4)),
  ))
```

Then `Open`, take `f.Root().Images()[0].Properties()`, and assert with `findProp`:

| property | assertion |
|---|---|
| `n` | type-asserts to `*property.NullProperty` |
| `s` | `*property.ShortProperty`, `Value() == int16(-3)` |
| `i` | `*property.IntProperty`, `Value() == int32(42)` |
| `l` | `*property.LongProperty`, `Value() == int64(9000000000)` |
| `f` | `*property.FloatProperty`, `Value() == float32(1.5)` |
| `d` | `*property.DoubleProperty`, `Value() == float64(2.25)` |
| `str` | `*property.StringProperty`, `Value() == "hello"` |
| `lt` | `*property.VectorProperty`, `X() == -100`, `Y() == -100` |
| `u` | `*property.UOLProperty`, `Value() == "../0/0"` |
| `cv` | `*property.ConvexProperty`, `len(Children()) == 2`, children named `"0"` and `"1"`, both `*property.VectorProperty` with `(1,2)` and `(3,4)` |

`TestBuilderFloatZeroWithoutMarker` — assert `wztest.Float("f", 0)` round-trips to `Value() == 0`.

- [x] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-wz && go test ./wz/ -run TestBuilderEmitsAllPropertyKinds -v`
Expected: FAIL — `undefined: wztest.Null`, `undefined: wztest.Short`, etc.

- [x] **Step 3: Extend `Prop`, `Kind`, and the constructors**

Add the `Kind` constants after `KindCanvas` (`builder.go:18-23`), the payload fields on `Prop` (`:26-33`), and the constructors alongside `Int`/`Str`/`Sub`/`Canvas` (`:35-45`). Keep the existing one-line style.

- [x] **Step 4: Add the `writePropList` arms**

New `case` arms inside the switch at `builder.go:154-195`, each emitting exactly the byte layout in the table above. The extended kinds (`Vector`, `UOL`, `Convex`) build an `inner` buffer and wrap it with tag `9` + `int32(len(inner))`, mirroring `KindSub` at `:164-174`. Convex children are written *bare* — string-block type tag then payload, with no property-name string block and no length prefix, because `parseExtendedProperty` is re-entered directly at `image.go:337`.

- [x] **Step 5: Run tests to verify they pass**

Run: `cd libs/atlas-wz && go test ./wz/ -run TestBuilder -v`
Expected: PASS.

- [x] **Step 6: Prove the extension is additive**

Run: `cd libs/atlas-wz && go build ./... && go test ./...`
Expected: PASS. `TestFixtureRoundTripGMS` (`fixture_roundtrip_test.go:43-104`) and every other existing `wztest` call site must still compile and pass untouched.

- [x] **Step 7: Commit**

```bash
git add libs/atlas-wz/wztest/builder.go libs/atlas-wz/wz/wztest_kinds_test.go
git commit -m "test(atlas-wz): wztest builder emits null/short/long/float/double/vector/uol/convex"
```

---

## Task 9: `wztest.Builder` — dimensioned canvases with children

`writePropList`'s canvas arm hard-codes width=1, height=1 and `hasProperty = 0` (`libs/atlas-wz/wztest/builder.go:180-189`). Half the evidence is about canvases: `2519000`'s `1/canvas:0` should be 100×121 with `origin (49,121)`, `2618000`'s `7/canvas:1` should be 75×143 with `origin (33,80)` and `delay`/`z` children. A 1×1 childless canvas cannot pin any of that.

### Files

- `libs/atlas-wz/wztest/builder.go` — extend `Prop` for canvas dimensions/children; rewrite the `KindCanvas` arm (`:176-192`)
- `libs/atlas-wz/wz/wztest_canvas_test.go` — **new file**
- `libs/atlas-wz/wz/image.go` — read-only; `parseCanvasProperty:364-433` is the layout being matched
- `libs/atlas-wz/wz/file.go` — read-only; `ReadCanvasData:217-222` does the `offset+1`/`size-1` flag-byte adjustment

Patterns to copy: `libs/atlas-wz/wz/fixture_roundtrip_test.go:43-104` — it already exercises `Canvas` end to end including `f.ReadCanvasData(cp.DataOffset(), cp.DataSize())`.

Module root: `libs/atlas-wz`.

### Interfaces

- Produces:
  - `func CanvasWith(name string, w, h int32, payload []byte, children ...Prop) Prop` — `Canvas(name, payload)` stays and becomes `CanvasWith(name, 1, 1, payload)` internally, so existing call sites are byte-identical.
  - `Prop` gains `W, H int32` (canvas dimensions).
- Consumes: `Vector`, `Int` from Task 8.

The canvas inner layout `parseCanvasProperty` expects, in order (`image.go:368-428`): 1 skipped byte, `hasProperty` byte, and **if `hasProperty > 0`** 2 skipped bytes followed by a full property list; then WzInt width, WzInt height, WzInt format, byte format2, 4 reserved zero bytes, `int32` dataSize, then `dataSize` bytes of payload whose first byte is the flag. Writing children means setting `hasProperty = 1` and emitting `0x00 0x00` + `writePropList(children)` before the dimensions.

**Do not change `dataOffset` semantics.** `parseCanvasProperty` records `dataOffset` at `image.go:422` pointing *at* the `0xAB` flag byte, and `File.ReadCanvasData` compensates with `offset+1`/`size-1`. That pairing is load-bearing for `libs/atlas-wz/atlas`'s pack determinism (design §6.4). The builder keeps writing `int32(len(payload)+1)` as `dataSize` followed by `0xAB` then the payload, exactly as `builder.go:187-189` does today. If Task 6 implicates it, it becomes its own defect with its own fixture — not a drive-by here.

- [x] **Step 1: Write the failing test**

`libs/atlas-wz/wz/wztest_canvas_test.go`, package `wz`. `TestBuilderCanvasWithDimensionsAndChildren`:

Build a GMS fixture with:

```
wztest.Img("frames",
  wztest.Sub("1",
    wztest.CanvasWith("0", 100, 121, []byte{0x01, 0x02, 0x03, 0x04},
      wztest.Vector("origin", 49, 121),
      wztest.Int("z", 0),
      wztest.Int("delay", 150),
    ),
  ),
)
```

Assertions on the parsed `*property.CanvasProperty` at `/1/0`:

| assertion | expected |
|---|---|
| `Width()` | `100` |
| `Height()` | `121` |
| `len(Children())` | `3` |
| child `origin` | `*property.VectorProperty`, `X()==49`, `Y()==121` |
| child `z` | `*property.IntProperty`, `Value()==0` |
| child `delay` | `*property.IntProperty`, `Value()==150` |
| `f.ReadCanvasData(cp.DataOffset(), cp.DataSize())` | returns exactly `[]byte{0x01,0x02,0x03,0x04}` |

`TestBuilderCanvasBackCompat` — `wztest.Canvas("icon", payload)` still parses to `Width()==1`, `Height()==1`, `len(Children())==0`, and `ReadCanvasData` returns the payload. This is the additive-only proof for `TestFixtureRoundTripGMS`.

- [x] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-wz && go test ./wz/ -run TestBuilderCanvas -v`
Expected: FAIL — `undefined: wztest.CanvasWith`.

- [x] **Step 3: Implement `CanvasWith` and rewrite the canvas arm**

Keep `Canvas` as a wrapper so no existing call site changes.

- [x] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-wz && go test ./wz/ -run TestBuilderCanvas -v`
Expected: PASS.

- [x] **Step 5: Full module green**

Run: `cd libs/atlas-wz && go test ./...`
Expected: PASS, including `libs/atlas-wz/icons` and `libs/atlas-wz/mapimage` (both consume canvas offsets from this reader) and `libs/atlas-wz/atlas`'s determinism tests.

- [x] **Step 6: Commit**

```bash
git add libs/atlas-wz/wztest/builder.go libs/atlas-wz/wz/wztest_canvas_test.go
git commit -m "test(atlas-wz): wztest builder emits dimensioned canvases with children"
```

---

## Task 10: `wztest.Builder` — offset-referenced string blocks (`SetStringDedup`)

`writeStringBlock` (`libs/atlas-wz/wztest/builder.go:141-146`) always emits tag `0x73` (inline). Real client-produced archives deduplicate repeated names and type tags into tag `0x01`/`0x1B` offset references, which `ReadWzStringBlock` resolves by seeking to `fileStart + offset` (`libs/atlas-wz/wz/reader.go:324-342`) — a code path **no fixture exercises today**. `2408001`'s `repeat` → `rpeat` is a wrong *name* with an intact stream position, which is the exact signature of this path, so a fixture that cannot emit it cannot pin the defect. This is design §7.2's "the one that actually matters" and PRD FR-10.2 by name.

### Files

- `libs/atlas-wz/wztest/builder.go` — string pool, `SetStringDedup`, offset patching in `Build()` (`:278-341`) and `writeStringBlock` (`:141-146`)
- `libs/atlas-wz/wz/wztest_dedup_test.go` — **new file**
- `libs/atlas-wz/wz/reader.go` — read-only; `ReadWzStringBlock:312-347` and `ReadWzString:218-241`
- `libs/atlas-wz/wz/image.go` — read-only; the five `ReadWzStringBlock` call sites at `:141`, `:178`, `:258`, `:300`, `:349`

Patterns to copy: `libs/atlas-wz/wztest/builder.go:263-264` (recording a `patch{pos, target}` for a not-yet-known offset) and `:324-339` (the patch-application loop in `Build`) — the existing directory-offset machinery generalizes directly to a string pool.

Module root: `libs/atlas-wz`.

### Interfaces

- Produces: `func (b *Builder) SetStringDedup(on bool) *Builder` — default **off**, so every existing fixture's bytes are unchanged.
- Consumes: the existing `chunk`/`patch` layout machinery in `Build()`.

Semantics: with dedup on, the *first* occurrence of a string within an image is emitted inline (tag `0x73`) and its position recorded; the *second and later* occurrences emit tag `0x01` for property names and `0x1B` for extended type tags, followed by an `int32` offset **relative to the image's own `dataOffset`** — because every `ReadWzStringBlock` call site passes `imageOffset` as `fileStart` (`image.go:141,178,258,300,349`), never the archive's `contentStart`. Getting the base wrong here produces a fixture that tests nothing.

The string table is per image, not per archive.

- [x] **Step 1: Write the failing test**

`libs/atlas-wz/wz/wztest_dedup_test.go`, package `wz`. `TestBuilderStringDedupRoundTrip`:

Build with `SetStringDedup(true)` an image whose property names and type tags repeat, so the second occurrences must go through the offset path:

```
wztest.Img("dedup",
  wztest.Sub("0",
    wztest.Int("state", 1),
    wztest.Int("repeat", 1),
    wztest.Vector("lt", -48, -48),
  ),
  wztest.Sub("1",
    wztest.Int("state", 2),
    wztest.Int("repeat", 1),
    wztest.Vector("lt", -48, -48),
  ),
)
```

Assertions:

| assertion | expected |
|---|---|
| `/0/state` | `*property.IntProperty`, `Value()==1` |
| `/1/state` | `*property.IntProperty`, `Value()==2` |
| `/0/repeat` and `/1/repeat` | both present, named exactly `repeat` (not `rpeat`), `Value()==1` |
| `/0/lt` and `/1/lt` | both `*property.VectorProperty`, `(-48,-48)` — proves the repeated `"Shape2D#Vector2D"` type tag resolved through tag `0x1B` |
| built bytes | contain at least one `0x01` and one `0x1B` string-block tag (assert by scanning `b.Build()` output, so a silently-inline-only implementation fails) |

`TestBuilderStringDedupOffByDefault` — same image without `SetStringDedup`, assert the built bytes contain no `0x01`/`0x1B` string-block tag and the tree still parses identically. This is the additive-only proof.

- [x] **Step 2: Run test to verify it fails**

Run: `cd libs/atlas-wz && go test ./wz/ -run TestBuilderStringDedup -v`
Expected: FAIL — `undefined: SetStringDedup` on `*wztest.Builder`.

- [x] **Step 3: Implement the per-image string table and offset patching**

Thread a `map[string]int` (string → offset within the image) through `buildImage`, record a `patch` at each deferred offset field, and resolve them in the same pass as the existing directory-offset patching in `Build()` (`:324-339`).

- [x] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-wz && go test ./wz/ -run TestBuilderStringDedup -v`
Expected: PASS.

- [x] **Step 5: Full module green**

Run: `cd libs/atlas-wz && go test ./...`
Expected: PASS. Existing fixtures are byte-identical because dedup defaults off.

- [x] **Step 6: Commit**

```bash
git add libs/atlas-wz/wztest/builder.go libs/atlas-wz/wz/wztest_dedup_test.go
git commit -m "test(atlas-wz): wztest builder emits offset-referenced string blocks"
```

---

## Task 11: The failing regression fixtures (FR-9, FR-11, FR-12) — **WITHDRAWN**

**WITHDRAWN.** Depended on `diagnosis.md` (Tasks 6-7, withdrawn) naming byte patterns
to reproduce for defects that Task 5 found do not exist. Task R2 defines its own
fixture coverage (declared-size overrun/underrun) independent of this task. Text
below kept as the original task description.

FR-12: a regression test that passes before the fix guards nothing. These tests must be committed **red**, with their failure output committed alongside, so a reviewer can verify from `git log` rather than from a claim.

### Files

- `libs/atlas-wz/wz/property_divergence_test.go` — **new file**; the table-driven guard
- `docs/tasks/task-262-wz-property-reader-divergence/pre-fix-test-failures.txt` — **new file**; the recorded red output
- `docs/tasks/task-262-wz-property-reader-divergence/diagnosis.md` — created by Tasks 6-7; read-only here. Supplies the exact byte pattern each row must reproduce
- `docs/tasks/task-262-wz-property-reader-divergence/reference-fidelity.md` — created by Task 5; read-only here. Only `PARSER-DEFECT` shapes get a row
- `libs/atlas-wz/wztest/builder.go` — read-only; Tasks 8-10 supply the emitters

Patterns to copy: `libs/atlas-wz/wz/fixture_roundtrip_test.go:16-38` (`writeFixture`, `findProp`) and `libs/atlas-wz/wz/reader_test.go:10-27` (`tempFileWithBytes`, for any row that needs raw bytes rather than a builder program).

Module root: `libs/atlas-wz`.

### Interfaces

- Consumes: every `wztest` constructor from Tasks 8-10, `wz.Open`, `Image.Properties()`.
- Produces: `TestPropertyDivergence` — one subtest per defect class.

Required rows (FR-11's minimum, each named after the evidence image it pins):

| subtest | shape | expected tree after the fix |
|---|---|---|
| `lost_imgdir_subtree` | the byte pattern from `diagnosis.md` that drops `2406000`'s `/imgdir:0/imgdir:event` | `event` present with one child `imgdir:0` carrying `int:state=1` and `int:type=6` |
| `lost_last_scalar_in_list` | same pattern applied to `2406000`'s `info` list | `int:activateByTouch=1` present as the final child of `info` |
| `lost_uol_node` | `2618000`'s `/imgdir:6/hit/uol:0` and `uol:1` | both present, `Value()` `../0` and `../1` |
| `lost_canvas_node` | `2618000`'s `/imgdir:7/canvas:0` | present, 72×145, child `int:delay=150` |
| `gained_subtree` | `3002000`'s phantom `/imgdir:1/imgdir:event` with `int:timeOut=2000` | **absent** — assert `findProp` does not find `event` under `imgdir:1` |
| `mangled_property_name` | `2408001`'s offset-referenced name, built with `SetStringDedup(true)` | property named exactly `repeat`, `Value()==1`; assert no property named `rpeat` exists |
| `wrong_scalar_value` | `2502002`'s `/imgdir:0/imgdir:event/imgdir:0/int:state` | `Value()==0` (not `1`), and sibling `/imgdir:0/imgdir:event/imgdir:1/int:2` present with `Value()==1` |
| `collapsed_canvas_dimensions` | `2519000`'s `/imgdir:1/canvas:0` | `Width()==100`, `Height()==121`, child `vector:origin` `X()==49`, `Y()==121` |
| `unenumerated_image` | the directory-level pattern from Task 7 — this row builds a whole archive | `f.Root()` walk yields **both** images; assert by name |

If `3002000` came back "we are right" in Task 5, replace the `gained_subtree` row with a row asserting our tree is preserved, and say so in the row comment — do not silently drop a required defect class.

Comparison helper: a shared deep-equality function that reports the **first differing path** in the `/imgdir:0/imgdir:event/int:state` notation the evidence file uses, so a failure reads like the evidence. Put it in this file. **No `*_testhelpers.go`** (repo convention); use the project Builder pattern for setup.

- [ ] **Step 1: Write the table-driven test**

One `t.Run` per row above, each building its fixture with `wztest`, writing it via `writeFixture`, opening with `wz.Open`, and asserting the full expected tree.

- [ ] **Step 2: Run it against the unfixed reader and confirm every row fails**

Run:

```bash
cd libs/atlas-wz && go test ./wz/ -run TestPropertyDivergence -v \
  > ../../docs/tasks/task-262-wz-property-reader-divergence/pre-fix-test-failures.txt 2>&1; echo "exit=$?"
```

Expected: exit 1, and **every** subtest FAILs. A row that passes here is not pinning its defect — either the fixture does not reproduce the byte pattern, or the shape was mislabelled in Task 5. Fix the fixture; do not delete the row.

- [ ] **Step 3: Commit the red tests and their output together**

```bash
git add libs/atlas-wz/wz/property_divergence_test.go \
        docs/tasks/task-262-wz-property-reader-divergence/pre-fix-test-failures.txt
git commit -m "test(atlas-wz): failing property-divergence fixtures (pre-fix, FR-12)"
```

This commit is expected to leave `go test ./libs/atlas-wz/...` red. That is the point of FR-12 and must be stated in the commit body.

---

## Task 12: CHECKPOINT — the decode fix — **WITHDRAWN**

**WITHDRAWN.** There is no decode fix: Task 6 (its prerequisite) is withdrawn because
Task 5 found zero `PARSER-DEFECT` images. A patch written without a `diagnosis.md`
offset is exactly the "plausible-looking patch" this checkpoint's own text (below)
forbids. Text below kept as the original task description.

**Stop here and re-plan from `diagnosis.md`.** Design §6.1 deliberately leaves the patch unspecified: the offsets come from Task 6 and the patch follows the offsets. Writing speculative code here would be exactly the "plausible-looking patch" PRD §2 rules out.

The controller splits this into one implementer task **per defect** in `diagnosis.md`, each with:

### Files (per resulting sub-task)

- `libs/atlas-wz/wz/image.go` and/or `libs/atlas-wz/wz/reader.go` and/or `libs/atlas-wz/wz/directory.go` — whichever `diagnosis.md` names
- `libs/atlas-wz/wz/property_divergence_test.go` — created by Task 11; read-only here. The row that must go green
- `docs/tasks/task-262-wz-property-reader-divergence/diagnosis.md` — created by Tasks 6-7; read-only here. The brief

Module root: `libs/atlas-wz`.

### Constraints every sub-task inherits

- The commit message names the byte offset from `diagnosis.md`. **A change that makes an image match without an offset explaining why is rejected at review.**
- One commit per defect, each turning exactly its own `TestPropertyDivergence` rows green and leaving the others red until their own commit.
- No wire or behaviour change to the 400 already-matching images. This is proven by Task 15's full-archive run, not by inspection (FR-7).
- `go test -race ./libs/atlas-wz/...` stays green, including `parse_race_test.go` and `iteration_contract_test.go`.
- `Image.Properties()` keeps its signature (`libs/atlas-wz/wz/image.go:81`). A newly-strict image returns `(partialProps, err)` — the error is already cached and already returned; callers already decide (task-076 F6).

- [ ] **Step 1: Controller re-plans this task from `diagnosis.md` into N implementer sub-tasks**
- [ ] **Step 2: Each sub-task lands its fix, turns its rows green, and commits with the offset in the message**
- [ ] **Step 3: `cd libs/atlas-wz && go test ./wz/ -run TestPropertyDivergence -v` — all rows PASS**
- [ ] **Step 4: `cd libs/atlas-wz && go test -race ./...` — PASS**

---

## Task 13: S1 strictness — bound `parsePropertyList` by its parent's extent — **WITHDRAWN**

**WITHDRAWN.** This task would change `Image.Properties()`'s production error surface
(a new `ErrPropertyOverrun`) to catch a defect class that Task 5 found no live instance
of in the one archive this task's evidence base covers — 1136/1136 type-9 sub-objects
traced clean across the **19 divergent images** (`reference-fidelity.md`); Task R2's
later whole-archive self-check corroborates this at larger scope — 15428/15428 type-9
sub-objects clean across **all 419 images in `$WZ_ARCHIVE`**, 0 violations, 0 parse
errors. Landing a new production error path with no
known real-world trigger, on the strength of one archive, is a larger and riskier change
than this re-scoped task should make unilaterally. Task R2 (`plan.md`, below) delivers
the equivalent detection **without** changing `Image.Properties()`'s behavior: it is a
separate, read-only `wzdiff --selfcheck` report built on the existing trace hook, so the
same size-accounting check is available in CI without any production error-surface
change. If a future sweep (PRD Open Question 2) finds a real violation, *that* is the
evidence to reopen S1 as a production change. Text below kept as the original task
description.

`parsePropertyList` (`libs/atlas-wz/wz/image.go:168-198`) has **no end-bound at all**: it reads a `ReadWzInt` count and then trusts it, with nothing checking that the children stayed inside the enclosing block. The only recovery in the whole parser is the type-9 branch's unconditional `Seek(endPos)` at `image.go:285-287`, which silently heals any drift a child introduced — that reseek is precisely why this defect stayed invisible. S1 makes the drift **recorded** instead of hidden. Per design §6.2 it lands unconditionally; S2 is gated (Task 15) and S3 is out of scope.

### Files

- `libs/atlas-wz/wz/image.go` — thread an `endPos` into `parsePropertyList` (`:168-198`); check it in the type-9 branch (`:264-289`) and the canvas child list (`:379-387`)
- `libs/atlas-wz/wz/strictness_test.go` — **new file**
- `libs/atlas-wz/wz/parse_race_test.go` — read-only; invariants that must survive

Patterns to copy: `libs/atlas-wz/wz/reader_test.go:10-27` (`tempFileWithBytes`) for a hand-built corrupt-bytes fixture; `libs/atlas-wz/wz/fixture_roundtrip_test.go:16-27` for the builder path.

Module root: `libs/atlas-wz`.

### Interfaces

- Produces: an exported sentinel `var wz.ErrPropertyOverrun = errors.New("property list overran its declared extent")`, so tests and callers can match with `errors.Is`.
- Consumes: nothing from earlier tasks.

**S1 rule, exactly:** in the type-9 branch, after `parseExtendedProperty` returns, compare `r.Pos()` against `endPos` **before** the recovery reseek at `image.go:285`. If the position is **greater** than `endPos`, the child over-consumed: wrap and return `ErrPropertyOverrun` naming the property, the declared size, and both offsets — after still performing the reseek, so the parent stays on a valid boundary and the rest of the image is reportable. An overrun is unambiguously corrupt; an *underrun* is S2 and is **not** an error in this task.

- [ ] **Step 1: Write the failing test**

`libs/atlas-wz/wz/strictness_test.go`, package `wz`. `TestPropertyListOverrunIsAnError` — build a valid fixture with `wztest`, then patch the built bytes to shrink one type-9 sub-object's declared `int32` size below what its body actually consumes (locate it by scanning for the known inner tag string, or by building a one-property image so the offset is deterministic). Assert:

| assertion | expected |
|---|---|
| `Image.Properties()` second return | non-nil, `errors.Is(err, ErrPropertyOverrun)` |
| the error message | contains the property name and both offsets |
| first return | the partial slice, not nil — a truncated tree is still returned alongside the error |
| the *sibling* property after the corrupt one | still parsed, proving the recovery reseek survived |

`TestPropertyListUnderrunIsNotAnErrorYet` — same fixture but with the declared size *larger* than the body. Assert `Properties()` returns a nil error. This pins S2 as deliberately absent so Task 15 can turn it on as a visible change.

`TestValidArchiveUnaffectedByS1` — `TestFixtureRoundTripGMS`'s fixture parses with a nil error and an unchanged tree.

- [ ] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-wz && go test ./wz/ -run 'TestPropertyList|TestValidArchiveUnaffected' -v`
Expected: FAIL — `undefined: ErrPropertyOverrun`, and the overrun currently returns nil.

- [ ] **Step 3: Implement S1**

Add the sentinel and the `r.Pos() > endPos` check. Keep the unconditional reseek — S1 is an **additive** check around the existing recovery, not a replacement for it (design §6.2).

- [ ] **Step 4: Run tests to verify they pass**

Run: `cd libs/atlas-wz && go test ./wz/ -run 'TestPropertyList|TestValidArchiveUnaffected' -v`
Expected: PASS.

- [ ] **Step 5: Prove no false positives across the module**

Run: `cd libs/atlas-wz && go test -race ./...`
Expected: PASS, including `icons`, `mapimage`, `charparts`, `atlas` determinism, and `TestPropertyDivergence` from Tasks 11-12.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-wz/wz/image.go libs/atlas-wz/wz/strictness_test.go
git commit -m "feat(atlas-wz): error on property-list overrun instead of silent truncation"
```

---

## Task 14: Propagate directory-parse failures and make ingest failures countable (FR-8, Observability NFR)

**KEPT — ruled independent of the withdrawn diagnosis (Task R1 judgment call).**
Unlike Tasks 6/7/11/12/13/15, this task does not consume `diagnosis.md`,
`reference-fidelity.md`'s labels, or any allowlist — its Files section names none of
them, and its test fixtures (below) are synthetic archives built and corrupted by the
task itself, not reproductions of a `Reactor.wz`-specific byte pattern. It is a direct
instance of FR-8, which the re-scoped `prd.md` §4.2 keeps as the one requirement that
survived whole: "make a silently-tolerated decode path an error, wherever doing so does
not break a legitimate archive." A sub-directory that fails to parse and is silently
dropped, with every image beneath it disappearing and no error reaching the caller, is
exactly that failure mode — independent of whether *this* archive happens to trigger it.
The one thing entangled with the withdrawn narrative is the opening sentence's framing
("the enumeration-level instance of the exact corruption class this task exists to
kill," citing design §5/§6.3, which reasoned from the false 19-image diagnosis) — that
motivating story is wrong, but the engineering conclusion it reaches for does not depend
on it, so the task proceeds on FR-8's restated grounds instead.

`parseDirectory` logs and drops a failed sub-directory at `libs/atlas-wz/wz/directory.go:122`, losing every image beneath it with no error to the caller — the enumeration-level instance of the exact corruption class this task exists to kill (design §5, §6.3, historical framing; see the note above). On the ingest side, `wztoxml.serializeDirectory` does the same per image at `adapter.go:44-46`, producing a wall of individual warnings that an operator cannot act on.

### Files

- `libs/atlas-wz/wz/directory.go` — replace the `Warnf`-and-continue at `:122` with a returned error
- `libs/atlas-wz/wz/directory_error_test.go` — **new file**
- `services/atlas-data/atlas.com/data/data/wztoxml/adapter.go` — per-image failure logging plus a per-archive summary (`SerializeToDirectory:30-40`, `serializeDirectory:42-58`)
- `services/atlas-data/atlas.com/data/data/wztoxml/adapter_test.go` — assert the counting behaviour
- `services/atlas-data/atlas.com/data/data/workers/runtime.go` — read-only; `monolithFile:144-148` and `OpenArchive:209-213` are the two `wz.Open` call sites that already handle an error and will now see this new failure class

Patterns to copy: `libs/atlas-wz/wz/fixture_roundtrip_test.go:16-27` for building a corrupt archive; `services/atlas-data/atlas.com/data/data/wztoxml/adapter_test.go:14-60` for the adapter test shape.

Module roots: `libs/atlas-wz` **and** `services/atlas-data/atlas.com/data`.

### Interfaces

- Produces: `parseDirectory` returns the sub-directory error wrapped with the entry name; `wz.Open` (`libs/atlas-wz/wz/file.go:132`) consequently fails for archives that today open with a silently-missing subtree.
- Consumes: nothing from earlier tasks.

**Blast-radius note, already surveyed:** `parseDirectory` is reached only from `File.parseRoot` (`file.go:525-538`), itself called only from `Open` (`file.go:156`). Every non-test `wz.Open` call site already checks the error — `services/atlas-data/atlas.com/data/data/workers/runtime.go:144-148` and `:209-213`, and `services/atlas-renders/atlas.com/renders/storage/wzcache.go:135` (confirmed safe by the Task 14 reviewer: it checks the error, removes the partial download, and never touches a partial file). No new error handling is needed at any `f.Root()` consumer (`libs/atlas-wz/charparts/extract.go:156`, `zmap.go:34`, `smap.go:32`, `mapimage/layers.go:142`, `index.go:33`, `icons/extract.go:43,116,161`, and the `atlas-data` workers). The behaviour change is entirely in whether `Open` itself now fails hard. Say this out loud in the PR description — it is an operator-visible change.

- [x] **Step 1: Write the failing tests**

`libs/atlas-wz/wz/directory_error_test.go`, package `wz`. `TestSubDirectoryParseFailurePropagates` — build a two-level archive with `wztest` (`AddDir` a `Dir` containing a nested `Dir`), then corrupt the nested directory's entry-count bytes in the built output. Assert `wz.Open` returns a non-nil error whose message names the failing sub-directory, and that no partially-populated `*File` is returned.

`TestValidNestedArchiveStillOpens` — the same archive uncorrupted opens cleanly and enumerates every image.

`adapter_test.go` — `TestSerializeDirectoryCountsFailures`: serialize a directory containing images where one fails, and assert the logger saw exactly one per-image line for the failure and exactly one summary line per archive reporting `N of M images failed`. **Assert there is no per-property line** — per-property logging is forbidden by the NFR. Capture logrus output with a `logrus.New()` writing to a `bytes.Buffer` (standard logrus test-capture, no new dependency).

- [x] **Step 2: Run tests to verify they fail**

Run: `cd libs/atlas-wz && go test ./wz/ -run TestSubDirectoryParseFailure -v`
Run: `cd services/atlas-data/atlas.com/data && go test ./data/wztoxml/ -run TestSerializeDirectoryCountsFailures -v`
Expected: both FAIL — the error is swallowed, and no summary line exists.

- [x] **Step 3: Propagate the directory error**

Replace `directory.go:122`'s `Warnf`-and-continue with `return nil, fmt.Errorf("parse sub-directory [%s]: %w", entryName, err)`.

- [x] **Step 4: Add per-image and per-archive accounting to `wztoxml`**

`serializeDirectory` accumulates a failure count and an image count and returns them; `SerializeToDirectory` logs one summary line per archive naming the archive path and `N of M`. The existing per-image `Warnf` keeps the image name and gains the archive path. Signatures of the three exported functions do not change — `runtime.go:101` must not be touched.

- [x] **Step 5: Run tests to verify they pass**

Run: `cd libs/atlas-wz && go test -race ./...`
Run: `cd services/atlas-data/atlas.com/data && go build ./... && go test ./...`
Expected: PASS in both.

- [x] **Step 6: Commit**

```bash
git add libs/atlas-wz/wz/directory.go libs/atlas-wz/wz/directory_error_test.go \
        services/atlas-data/atlas.com/data/data/wztoxml
git commit -m "feat(atlas-wz): propagate sub-directory parse failures; count ingest image failures"
```

---

## Task 15: Full-archive re-verification, S2 decision, and the final gate (FR-7, FR-13, FR-14) — **WITHDRAWN**

**WITHDRAWN.** Depends on Task 12's fix (withdrawn) and Task 5's allowlist (never
produced — `reference-fidelity.md` found `INPUT-MISMATCH` everywhere, not
`REFERENCE-RESOLUTION`). FR-7/FR-13/FR-14 as written are withdrawn in `prd.md` §4;
there is no post-fix diff to run because there is no fix. The S2 strictness decision
(Step 3) is moot for the same reason S1 (Task 13) is withdrawn — no known trigger in
this task's evidence base. The final gate itself (flagless `tools/verify.sh`, code
review) is still required, but it now belongs to whichever task lands Task R2, not to
this withdrawn full-archive-diff step. Text below kept as the original task
description.

The acceptance evidence. FR-7 is proven here — that the 400 already-matching images are untouched is a measurement, not an inspection.

### Files

- `docs/tasks/task-262-wz-property-reader-divergence/post-fix-diff.txt` — **new file**; the raw FR-13 output
- `docs/tasks/task-262-wz-property-reader-divergence/allowlist.tsv` — created by Task 5; read-only here unless S2/Task 12 changed an adjudication
- `docs/tasks/task-262-wz-property-reader-divergence/reference-fidelity.md` — created by Task 5; read-only here
- `libs/atlas-wz/wz/image.go` — the S2 check, only if Step 3 says it is safe
- `libs/atlas-wz/wz/strictness_test.go` — flip `TestPropertyListUnderrunIsNotAnErrorYet` only if S2 lands
- `README` note: **do not commit the reference dump or the archive.**

Module root: `libs/atlas-wz`.

- [ ] **Step 1: Run the full-archive diff**

```bash
cd libs/atlas-wz && go run ./cmd/wzdiff \
  --archive "$WZ_ARCHIVE" \
  --reference ../../tmp/083839c6-c47c-42a6-9585-76492795d123/GMS/83.1/Reactor.wz \
  --allowlist ../../docs/tasks/task-262-wz-property-reader-divergence/allowlist.tsv \
  > ../../docs/tasks/task-262-wz-property-reader-divergence/post-fix-diff.txt 2>&1; echo "exit=$?"
```

Expected: **exit 0**. Both sides enumerate **421** images. Zero unallowlisted deltas. Every remaining delta is an allowlisted `REFERENCE-RESOLUTION` entry.

If the exit is non-zero, the task is not done — quote the failing block and go back to Task 12. Do not adjust the allowlist to make a `PARSER-DEFECT` disappear; that converts the allowlist from a deliverable into a waiver, which design §2.3 explicitly forbids.

- [ ] **Step 2: Confirm the FR-5 rows the adjudication kept**

Against `reference-fidelity.md` and `evidence-wz-parse-divergence-reactor.txt`, confirm each surviving PRD FR-5 row. Specifically assert from the post-fix output that `2406000` now carries `info/activateByTouch = 1` and `0/event/0/{type=6,state=1}`, and that `2408001`'s property is named `repeat`.

- [ ] **Step 3: Decide S2 from evidence, not preference**

Temporarily enable the underrun check (`r.Pos() < endPos` after `parseExtendedProperty`) and re-run Step 1's command. Design §6.2 gates S2 on **zero new failures across the full archive**, and asks for a second archive before enabling.

- If the run stays exit 0, land S2: flip `TestPropertyListUnderrunIsNotAnErrorYet` into a `TestPropertyListUnderrunIsAnError` with the same fixture, commit it, and re-run.
- If the run produces any new failure, **do not land S2.** Record the failing images and the count in `post-fix-diff.txt`, and note S2 as deferred alongside the design §9 sweep and S3.

Either outcome is acceptable; an unrecorded decision is not.

- [ ] **Step 4: Commit the evidence**

```bash
git add docs/tasks/task-262-wz-property-reader-divergence/post-fix-diff.txt
git commit -m "docs(task-262): post-fix full-archive diff, 421/421 with adjudicated allowlist"
```

- [ ] **Step 5: Run the full verification gate**

Run: `tools/verify.sh`
Expected: **exit 0**, flagless. `--quick`/`--no-docker` do not count (CLAUDE.md). Note this run fans out to all ~86 modules because the branch touches `libs/`.

- [ ] **Step 6: Lint the plan's own artifacts and confirm the acceptance ledger**

Confirm, and write into `context.md`:

- `Reactor.wz` enumerates 421 images.
- Zero `PARSER-DEFECT` deltas; the allowlist contains only byte-justified `REFERENCE-RESOLUTION` entries.
- `go test -race ./libs/atlas-wz/...` passes, including `parse_race_test.go` and `iteration_contract_test.go`.
- `services/atlas-data` builds and tests pass; `propertyToElement`'s mapping semantics are unchanged (Task 1 moved it verbatim).
- `libs/atlas-wz/atlas` determinism tests pass.
- `pre-fix-test-failures.txt` and `post-fix-diff.txt` are both committed.
- No reactor touch-activation *behaviour* was added (that stays with task-249) and no tenant re-ingest was performed.

- [ ] **Step 7: Code review before the PR**

Dispatch `atlas-reviewer` per plan task and `backend-guidelines-reviewer` over the changed Go packages. Both are required gates; a green `verify.sh` cannot see a cross-module seam defect, and this branch crosses `libs/atlas-wz` → `services/atlas-data`.

---

## Task R2: Whole-archive size-accounting self-check

**Added by Task R1's re-scope, replacing the withdrawn Tasks 6-7 and 11-15.** With the
HaRepacker dump withdrawn as an oracle (`provenance.md`), this task gives task-262 a
reference that needs no external dump at all: **the archive's own bytes**. Every WZ
type-9 sub-object declares its own byte length; if a decode ends anywhere other than
where that declaration says it should, that is a defect, and today it is silently
healed by the type-9 branch's unconditional recovery reseek (`wz/image.go:285-287`)
and never surfaces. Task 5 already ran this check by hand across the **19 divergent images** and found
1136/1136 sub-objects clean (`reference-fidelity.md`); this task turns it into a
repo-tool gate — a new `wzdiff --selfcheck` mode built on the existing trace hook
(Task 2), with `wztest`-fixture coverage (a clean archive passes; a sub-object with a
deliberately corrupted declared size fails) so it runs in CI with no external archive.
Deliberately **not** a change to `Image.Properties()`'s production error surface — see
Task 13's withdrawal note for why a new production error path is out of scope on this
task's evidence base. Full detail: `task-R2-brief.md`.

