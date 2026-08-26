# task-262 — Implementation Context

Companion to `plan.md`. Everything an implementer or reviewer needs that is not a task step.

---

## 1. Decisions taken during planning

### 1.1 Acceptance bar — design §10 resolved (user-approved)

The design's blocking decision was signed off in favour of the **§2.3 restatement**:

> Zero `PARSER-DEFECT` deltas, plus a committed, byte-justified allowlist of
> `REFERENCE-RESOLUTION` deltas produced by the §2.2 gate.

PRD **FR-4 and FR-13's "0 divergent in either direction" bar is superseded.** The
reason, from the design and confirmed against the code: part of the 19-image delta is
the HaRepacker dump *resolving* `link` / `_inlink` / `_outlink` / `UOL` references while
our parser stays deliberately literal, and Atlas resolves those in consumers —
`services/atlas-data/atlas.com/data/reactor/reader.go:65`
(`info.GetString("link", "")`), `services/atlas-data/atlas.com/data/map/reader.go:60`,
`libs/atlas-wz/icons/extract.go:222` (`findInfoLink`). Satisfying FR-13 literally would
mean implementing resolution inside `Image.Properties()`, double-resolving against those
consumers and changing the property tree's meaning for every existing caller.

Consequence for review: the allowlist is a **deliverable with byte evidence**, not a
waiver. Task 15 Step 1 explicitly forbids adding an entry to make a `PARSER-DEFECT`
disappear.

### 1.2 `wzdiff` lives in `libs/atlas-wz`, not `tools/wzdiff` — deviation from design §4.3

Design §4.3 proposed `tools/wzdiff/cmd/wzdiff/main.go` following the `tools/packet-audit`
convention, and flagged the module edge as "confirm during planning." Confirmed, and the
placement changed. Three findings drove it:

1. **`tools/verify.sh` does not build or test `tools/*` Go modules.** `all_modules()`
   (`tools/verify.sh:181-184`) globs `go.mod` under `$ROOT/services` and `$ROOT/libs`
   only. A tool at `tools/wzdiff` would be invisible to the gate.
2. **Registering a new module trips a guard.** `service-registration-guard.sh` fires on
   any `go.work` edit (`tools/verify.sh:419`).
3. **The `wztoxml` dependency is not importable from `tools/`.** `wztoxml` lives in
   module `atlas-data` — a non-fetchable local module path. A `require`+`replace` would
   drag gorm, kafka and the whole service dependency graph into a diff tool.

Precedent for the chosen shape: `libs/atlas-constants/gen/cmd/mksnapshot/main.go` — a
`cmd/` binary inside a `libs/` module, covered by the gate for free.

Design §4.3 named the fallback if the module edge was blocked ("promote `wztoxml`'s pure
tree→XML mapping into `libs/atlas-wz/wz/wzxml` and have `atlas-data` re-export it").
Task 1 does exactly that, and the move is clean: `adapter.go`'s import block
(`services/atlas-data/atlas.com/data/data/wztoxml/adapter.go:12-24`) has **no
atlas-data-local imports** — only stdlib, logrus, and `atlas-wz` packages. The four
functions being moved (`xmlElement:88-98`, `propertiesToElements:100-109`,
`propertyToElement:111-153`, `formatFloat:155-163`) touch none of logrus, `os`, or
`filepath`. Per repo convention this is a straightforward move, not a re-exported alias
shim — `wztoxml` delegates and keeps its three exported functions' signatures so
`services/atlas-data/atlas.com/data/data/workers/runtime.go:101` is untouched.

### 1.3 Strictness ladder

Landing **S1** (overrun is an error) unconditionally in Task 13; **S2** (underrun) gated
on the full-archive run in Task 15 Step 3; **S3** (erroring on unknown tags) explicitly
out of scope, per design §6.2. S3 is a tolerance-policy change with a blast radius across
every archive and tenant, and this task's evidence base is one archive.

Task 13 deliberately ships `TestPropertyListUnderrunIsNotAnErrorYet` asserting the
*absence* of S2, so Task 15's decision is a visible flip rather than an unrecorded
default.

### 1.4 Canvas `dataOffset` is observed, not fixed

`parseCanvasProperty` records `dataOffset` at `libs/atlas-wz/wz/image.go:422` pointing
*at* the flag byte, despite the comment at `:421` saying it skips it;
`File.ReadCanvasData` (`libs/atlas-wz/wz/file.go:217-222`) compensates with
`offset+1`/`size-1`, and `wztest` writes the matching `0xAB` flag byte
(`libs/atlas-wz/wztest/builder.go:188`). The pairing is self-consistent and
`libs/atlas-wz/atlas`'s pack determinism depends on it. Per design §6.4 this is **not** a
drive-by fix. Task 9 preserves the layout exactly. If Task 6's diagnosis implicates it, it
becomes a tracked defect with its own fixture under Task 12.

---

## 2. External inputs

| Input | Status |
|---|---|
| **`$WZ_REFERENCE`** — HaRepacker XML dump | **Present and verified.** `tmp/083839c6-c47c-42a6-9585-76492795d123/GMS/83.1/Reactor.wz/`, 421 `.img.xml` files. Confirmed `2406000.img.xml` contains `<int name="activateByTouch" value="1"/>`, matching the evidence. **Not committed; must not be added to git.** |
| **`$WZ_ARCHIVE`** — the 51.6 MiB PKG1 `GMS/83.1/Reactor.wz` binary | **NOT on this machine.** `find / -xdev -type f -name '*.wz' -size +10M` returned nothing; the same search over `/mnt/c`, `/mnt/d`, `/mnt/e` found only directories named `Reactor.wz`. **Tasks 5, 6, 7 and 15 are blocked until it is supplied.** |

Tasks 1-4 and 8-11 have no external dependency and can proceed immediately. Task 11's
fixture byte patterns come from Task 6's `diagnosis.md`, so in practice the runnable
prefix without the archive is Tasks 1-4 and 8-10.

The evidence file records no provenance for either input. Whoever supplies the archive
should record its size and a hash in `reference-fidelity.md` so a future re-run is
reproducible (FR-14).

---

## 3. Key files and what they do

| File | Role |
|---|---|
| `libs/atlas-wz/wz/image.go` | The whole property decode path. `parse:104`, `parseWithKey:132`, `parsePropertyList:168`, `parsePropertyValue:201`, `parseExtendedProperty:297`, `parseCanvasProperty:364`, `parseSoundProperty:436` |
| `libs/atlas-wz/wz/reader.go` | Primitives. `ReadWzString:218` (`int8` tag switch, never errors on tag), `ReadWzStringBlock:312` (`0x00`/`0x73` inline, `0x01`/`0x1B` offset-ref, everything else errors) |
| `libs/atlas-wz/wz/directory.go` | `parseDirectory:39-137`; `count` at `:42`, `elemType` switch at `:56-92`, the swallowed sub-directory failure at `:122` |
| `libs/atlas-wz/wz/file.go` | `Open:132`, `Root:181`, `ReadCanvasData:217`, `LockParse:93`, `NewSubFile:115`, `parseRoot:525` |
| `libs/atlas-wz/wztest/builder.go` | The only fixture generator. `writeStringBlock:141` (always inline `0x73` today), `writePropList:148-198`, `buildDir:220-268`, `Build:278-341` |
| `services/atlas-data/atlas.com/data/data/wztoxml/adapter.go` | Ingest serializer. Per-image swallow at `:44-46`; pure mapping at `:88-163` (moves in Task 1) |

## 4. Structural facts the plan depends on

- **`parsePropertyList` has no end-bound whatsoever.** It reads a `ReadWzInt` count at
  `image.go:171` and loops, with no size field, no `endPos`, and no bounds check. This is
  why Task 13 exists.
- **The type-9 branch's `Seek(endPos)` at `image.go:285-287` is unconditional** — it runs
  on success as well as failure. That single reseek is both the only thing preventing a
  desynchronised child from cascading *and* the reason the corruption is silent. Task 13
  keeps it and adds a recorded check around it; it does not replace it.
- **`ReadWzStringBlock`'s `fileStart` is always the image's own `dataOffset`.** All five
  call sites pass `imageOffset` (`image.go:141, 178, 258, 300, 349`); there are no callers
  outside `image.go`. Directory entry names use `ReadWzString` directly and never take the
  offset-referenced path. Task 10's `SetStringDedup` must write offsets relative to the
  image, not to `contentStart`.
- **`parseMu` is taken once, in `Image.Properties()` at `image.go:89`**, and held across
  the entire chain. Nothing below it locks; every doc comment states "Caller holds
  File.parseMu" as convention, not enforcement. The Task 2 trace hook fires under that
  lock and must not re-enter the reader.
- **`Properties()`'s fast path must stay lock-free.** `parse_race_test.go:77-92`
  (`TestPropertiesFastPathSkipsLock`) asserts an already-parsed `NewParsedImage` never
  touches `wzFile`. No trace emit belongs on that path.
- **`parseDirectory` is reachable only via `Open`.** `parseRoot` (`file.go:525-538`) is
  called only from `Open` (`file.go:156`). Every non-test `wz.Open` call site already
  handles an error: `runtime.go:144-148` and `runtime.go:209-213`. Task 14's propagation
  therefore needs no new error handling at any `f.Root()` consumer — but it does change
  whether `Open` itself fails, which is operator-visible.

## 5. The 19 divergent images

From `evidence-wz-parse-divergence-reactor.txt` (line numbers for slicing):

`2006000:6`, `2006001:18`, `2406000:66`, `2408001:75`, `2502002:81`, `2519000:90`,
`2519001:98`, `2519002:106`, `2519003:163`, `2618000:220`, `2618003:311`, `2618004:319`,
`2618005:327`, `2618006:335`, `2618007:716`, `3002000:724`, `9202005:733`, `9208003:742`,
`9208004:750`. Plus `9400300.img` and `9400301.img`, never enumerated (`:1-3`).

**A shape worth noticing before diagnosis starts** (a lead, not a finding): in `2406000`
both lost nodes are the **last child of their list** — `activateByTouch` is the final
entry of `info`, `event` the final entry of `imgdir:0`. That points at either a short
`count` from `ReadWzInt` (`image.go:171`) or a child that over-consumed. The Task 2
trace's `declaredSize`/`actualEnd` pair on the type-9 branch distinguishes them directly.
Task 6 must confirm this from bytes; do not treat it as diagnosed.

## 6. Task sizing notes

Task sizes are within the ~6-file / one-module guidance except where noted:

- **Task 1** crosses two module roots (`libs/atlas-wz` and
  `services/atlas-data/atlas.com/data`) deliberately. It is one indivisible move: leaving
  the extraction and the delegation in separate commits would leave the tree with
  duplicate mapping code, and reviewing half of a move is not useful.
- **Task 14** likewise spans both modules, for the same reason — the error-propagation
  change and the ingest-side accounting are the two halves of FR-8's "surface it rather
  than swallow it," and splitting them would land a build where `Open` fails hard with no
  operator-facing accounting.
- **Task 12 is intentionally a checkpoint, not an implementable task.** Design §6.1
  refuses to pre-guess the patch; the controller splits it into one implementer sub-task
  per defect in `diagnosis.md`, each ≤2 files. Writing speculative code there would be the
  "plausible-looking patch" PRD §2 rules out.
- **Tasks 5, 6, 7 are diagnosis tasks producing documents, not code.** They are separate
  from Task 12 precisely so the fix cannot start before the evidence exists.

## 7. Verification notes

- **`tools/verify.sh` on this branch fans out to all ~86 modules** (`tools/verify.sh:196-224`):
  a change anywhere under `libs/` reaches every module, and `CHANGED` is the whole branch
  against its merge base. For per-task iteration gates, pass
  `--base <last-gated-commit>` so the change set is the increment under test. Only the
  final gate runs flagless and unscoped.
- **`tools/verify.sh` does not cover `tools/*` Go modules** — see §1.2. Irrelevant given
  the chosen placement, but it is why the placement was chosen.
- **Task 11's commit is expected to leave `go test ./libs/atlas-wz/...` red.** That is
  FR-12, and the commit body must say so. Any gate run between Task 11 and the end of
  Task 12 will fail on `TestPropertyDivergence` by design.
- Existing contracts that must survive every task: `parse_race_test.go`
  (`TestLockParseIsExclusive:29-63`, `TestPropertiesFastPathSkipsLock:77-92`,
  `TestPropertiesConcurrentParse:105-138`) and `iteration_contract_test.go`
  (`TestImageNameStripsDotImg:22-45`, `TestNewFileWithRootRoundTrip:52-79`).

## 8. Deliberately out of scope

- The blast-radius sweep across Map.wz / Mob.wz / Item.wz / Skill.wz / String.wz /
  Character.wz / Npc.wz / Quest.wz. `wzdiff` is built so that sweep is one invocation per
  archive; seeding it is this task's contribution to the follow-up (design §9).
- S3 tolerance-policy change (§1.3 above).
- Link / UOL resolution in the parser (§1.1).
- Reactor touch-activation behaviour — task-249 owns it.
- Tenant re-ingest. The corrected parser does not reach a running environment until
  affected tenants re-import; PRD Open Question 3 (who schedules it) is unanswered and
  belongs in the PR description as an operator note.
- Render-baseline refresh. If `2519000`/`2519001` come back `REFERENCE-RESOLUTION` — which
  design §2 expects — no baseline moves and PRD Open Question 5 dissolves. Task 5 answers
  this.
