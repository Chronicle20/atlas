# Touch-Activated Reactors — Implementation Plan

Version: v1
Status: Draft
Created: 2026-08-21
PRD: [prd.md](prd.md)
Design: [design.md](design.md)
Context: [context.md](context.md)

---

## Ordering

Tasks 1–5 are the packet/coverage lane. Tasks 6–12 are the reactor lane. Task 13
is the channel edge. Tasks 14–15 are `atlas-reactor-actions`. Task 16 is docs.

Hard dependencies:

- Task 3, 4, 5 depend on **Task 1** (the derived opcodes).
- Task 5 depends on **Tasks 2, 3, 4** (codec + registry + routing must all be in
  place before `matrix --check` can pass).
- Task 7 depends on **Task 6** (the REST shape it decodes).
- Task 12 depends on **Tasks 7, 8, 9, 10, 11**.
- Task 15 depends on **Task 14**.

Tasks 6, 8, 9, 10, 11, 13, 14 have no cross-dependency and may run in any order.

---

## Task 1: Derive the TOUCHING_REACTOR send site on all ten versions

Read-only IDA work. Produces a findings document; changes no code.

Ten IDBs are already open. Confirm the current session ids with
`mcp__ida-pro__idb_list` before starting — the ids below were captured at plan
time and are not stable across restarts. Pin `database: <session id>` on every
`mcp__ida-pro__*` call; omitting it silently hits whichever IDB the server
thinks is active.

| version | session id (plan-time) |
|---|---|
| gms_v48 | `12a398ce` |
| gms_v61 | `921fdbb5` |
| gms_v72 | `99e435d8` |
| gms_v79 | `5a1cd4f3` |
| gms_v83 | `754107bf` |
| gms_v84 | `46c2a2eb` |
| gms_v87 | `c0829805` |
| gms_v92 | `019cd393` |
| gms_v95 | `ecc757f4` |
| jms_v185 | `a977912e` |

### Files

- `docs/tasks/task-249-touch-activated-reactors/opcode-derivation.md` — **new file**; the findings document this task produces
- `docs/tasks/task-249-touch-activated-reactors/design.md` — read-only; §1.1–§1.4 carry the addresses already measured
- `docs/packets/audits/VERIFYING_A_PACKET.md` — read-only; §10 "Naming unnamed senders", and "Is this cell `n-a`?" for the absence bar
- `docs/packets/registry/gms_v83.yaml` — read-only; line 3176 is the existing `TOUCHING_REACTOR` entry shape

No Go module is built by this task.

### Steps

- [ ] **Step 1: Re-confirm the five already-measured versions**

  For each of gms_v72 `@0x692bb0`, gms_v79 `@0x6b8362`, gms_v87 `@0x77bca7`,
  gms_v95 `@0x6cded0`, jms_v185 `@0x79f0aa`: `decompile` (or `insn_query` at the
  call site for v87/jms, whose bodies are large) and record **verbatim** the
  `COutPacket::COutPacket(&pkt, N)` integer and the sequence of `Encode*` calls
  that follow it, in both the entering and leaving arms.

  Expected, per design §1.1–§1.2: `COutPacket(N)` → `Encode4(dwID)` →
  `Encode1(1)` in the entering arm, and `COutPacket(N)` → `Encode4(dwID)` →
  `Encode1(0)` in the leaving arm. **If any version deviates, that is the
  finding — record it and do not smooth it over.**

  Expected opcodes: v72 = 196 (`0x0C4`), v79 = 198 (`0x0C6`), v87 = 219
  (`0x0DB`), v95 = 250 (`0x0FA`), jms_v185 = 217 (`0x0D9`).

- [ ] **Step 2: Name the unnamed sender in gms_v83**

  `CReactorPool::FindSkillReactor` ends at `0x735d13`. `find_bytes "68 CE 00 00
  00"` yields `0x735fb9` and `0x736021` — the expected enter/leave pair. Use
  `define_func` if the bytes are not already inside a function, then `rename`
  the containing function to `CReactorPool::FindTouchReactorAroundLocalUser`,
  decompile it, and confirm the two-field body. Save the IDB (`idb_save`).

  Record the function's start address — Task 5 needs it for the audit report and
  the `packet-audit:verify` marker.

- [ ] **Step 3: Derive the gms_v84 opcode from the binary**

  §1.4 records that `find_bytes "68 CE 00 00 00"` finds nothing near the v84
  `CReactorPool` cluster (`FindHitReactor` @`0x752cbc`, ending `0x7530ac`), so
  the registry's `0x0CE` is unconfirmed.

  **Hypothesis to test, not to assume:** `DAMAGE_REACTOR` is 205 on v83 and 211
  on v84 (`docs/packets/audits/status.json`), a +6 shift. If `TOUCHING_REACTOR`
  shifted with it, v84 is 212 = `0x0D4` and the signature is
  `find_bytes "68 D4 00 00 00"`. Test that first, but if it misses, sweep the
  general send-site signature from §10 — `6A 01 68 ?? ?? 00 00` (push 1; push
  opcode) — within the `CReactorPool` cluster, and structure-match the result
  against the v83 twin named in Step 2.

  Whatever address is found: `rename` it to
  `CReactorPool::FindTouchReactorAroundLocalUser`, decompile, record the
  `COutPacket` integer verbatim, and `idb_save`. **Do not copy v83's 0x0CE
  across on the registry's say-so.**

- [ ] **Step 4: Confirm the gms_v92 candidate**

  `find_bytes "6A 01 68 F3 00 00 00"` yields `0x79f903`, `0x79f9f4`, `0x828dad`.
  The first two are adjacent (enter/leave pair). Decompile the function
  containing `0x79f903`, confirm the `Encode4(dwID)` + `Encode1` body, `rename`
  it to `CReactorPool::FindTouchReactorAroundLocalUser`, and `idb_save`. Record
  the function start address and confirm the opcode is 243 (`0x0F3`).

- [ ] **Step 5: Re-verify the two `n-a` claims (gms_v48, gms_v61)**

  Design §1.3 measured both as genuinely absent: `func_query` for `CReactorPool`
  returns six symbols on each, none of them the touch function. That is a
  *name*-scoped search, which "Is this cell `n-a`?" step 1 explicitly rejects as
  sufficient on its own. Close the gap with the opcode-construction invariant:

  - Run `find_bytes` for the general send signature `6A 01 68 ?? ?? 00 00` across
    the whole binary and check whether any hit falls inside or adjacent to the
    `CReactorPool` cluster (v48 `FindHitReactor` @`0x5a5a32`; find the v61
    equivalent with `func_query`).
  - Perform the **mandatory sibling cross-check**: decompile the version's
    `CReactorPool::OnReactorChangeState` / `OnPacket` and record whether it
    carries any touch/proximity state that would imply a send side.

  Record the result. If either version turns out to define the opcode, say so —
  it moves that version into scope for Tasks 2–5 and the plan must be revised
  before continuing.

- [ ] **Step 6: Write the findings document**

  `opcode-derivation.md` MUST contain, per version, a row with: the function
  address, whether the symbol was pre-existing or named by this task, the
  `COutPacket` integer **as read from the pseudocode**, the field sequence, and
  the registry value it agrees or disagrees with. Quote the decompiled send
  block for every version whose address or opcode this task newly established
  (v83, v84, v92) — Task 5 cites these addresses and Task 3 writes these opcodes
  into the registry.

  End the document with an explicit **in-scope version list** and its opcodes.

### Verification

- Every in-scope version has an address and an opcode read from the binary.
- gms_v83, gms_v84 and gms_v92 have a named
  `CReactorPool::FindTouchReactorAroundLocalUser` and a saved IDB.
- gms_v48 and gms_v61 have positive absence evidence beyond a name search.

---

## Task 2: `TouchingRequest` codec

### Files

- `libs/atlas-packet/reactor/serverbound/touching.go` — **new file**; the codec
- `libs/atlas-packet/reactor/serverbound/touching_test.go` — **new file**; round-trip + byte fixtures
- `libs/atlas-packet/reactor/serverbound/hit.go` — read-only; the shape to copy
- `libs/atlas-packet/test/context.go` — read-only; `pt.Variants`, `pt.CreateContext`
- `libs/atlas-packet/test/roundtrip.go` — read-only; `pt.RoundTrip`, `pt.Encode`
- `tools/packet-audit/cmd/run.go` — add the `candidatesFromFName` case at line 1264's switch
- `docs/tasks/task-249-touch-activated-reactors/opcode-derivation.md` — **new file** created by Task 1; read-only here

Modules: `libs/atlas-packet` (run `go build ./... && go test ./...` there) and
`tools/packet-audit` (build only).

Patterns to copy: `libs/atlas-packet/reactor/serverbound/hit.go` (struct +
accessors + `Operation()` + `String()` + `Encode`/`Decode`);
`libs/atlas-packet/reactor/serverbound/hit_test.go:26-64` (round-trip over
`pt.Variants`) and `:76-96` (byte fixture with an evidence comment block).

### Steps

- [ ] **Step 1: Write the failing test**

`libs/atlas-packet/reactor/serverbound/touching_test.go`. Two test functions.
Imports, `pt` alias, and `bytes` follow `hit_test.go:1-8` exactly.

`TestTouchingRoundTrip` — iterates `pt.Variants` with `t.Run(v.Name, ...)`,
identical in shape to `TestHitRoundTrip`. **No version gate**: the body is the
same two fields on every variant (design §1.2), so every variant must round-trip
both fields exactly.

| case | input | assertion |
|---|---|---|
| every `pt.Variants` entry, `touching=true` | `TouchingRequest{oid: 100, touching: true}` | `output.Oid() == 100` and `output.Touching() == true` |
| every `pt.Variants` entry, `touching=false` | `TouchingRequest{oid: 100, touching: false}` | `output.Oid() == 100` and `output.Touching() == false` |

Run both sub-cases inside the one `t.Run` body (encode/decode twice), or add a
nested `t.Run("entering"/"leaving")` — either is fine.

`TestTouchingBytes` — table-driven byte fixture. Because the layout is
version-invariant the expected bytes are identical for every variant; assert the
exact five bytes on a representative pair.

| subtest | context | input | expected bytes |
|---|---|---|---|
| `entering` | `pt.CreateContext("GMS", 83, 1)` | `TouchingRequest{oid: 100, touching: true}` | `0x64 0x00 0x00 0x00 0x01` |
| `leaving` | `pt.CreateContext("GMS", 83, 1)` | `TouchingRequest{oid: 100, touching: false}` | `0x64 0x00 0x00 0x00 0x00` |
| `entering_v95` | `pt.CreateContext("GMS", 95, 1)` | `TouchingRequest{oid: 100, touching: true}` | `0x64 0x00 0x00 0x00 0x01` |
| `leaving_jms` | `pt.CreateContext("JMS", 185, 1)` | `TouchingRequest{oid: 100, touching: false}` | `0x64 0x00 0x00 0x00 0x00` |

Use `pt.Encode(t, ctx, input.Encode, nil)` and `bytes.Equal`, exactly as
`TestHitBytesV48` does.

Above `TestTouchingBytes`, write the marker block — **one line per in-scope
version**, with the address taken from Task 1's `opcode-derivation.md`, not from
this plan:

```
// packet-audit:verify packet=reactor/serverbound/ReactorTouchingRequest version=gms_v72 ida=0x<from Task 1>
// packet-audit:verify packet=reactor/serverbound/ReactorTouchingRequest version=gms_v79 ida=0x<from Task 1>
// packet-audit:verify packet=reactor/serverbound/ReactorTouchingRequest version=gms_v83 ida=0x<from Task 1>
// packet-audit:verify packet=reactor/serverbound/ReactorTouchingRequest version=gms_v84 ida=0x<from Task 1>
// packet-audit:verify packet=reactor/serverbound/ReactorTouchingRequest version=gms_v87 ida=0x<from Task 1>
// packet-audit:verify packet=reactor/serverbound/ReactorTouchingRequest version=gms_v92 ida=0x<from Task 1>
// packet-audit:verify packet=reactor/serverbound/ReactorTouchingRequest version=gms_v95 ida=0x<from Task 1>
// packet-audit:verify packet=reactor/serverbound/ReactorTouchingRequest version=jms_v185 ida=0x<from Task 1>
```

`ReactorTouchingRequest` is `qualifiedWriterName("reactor", "TouchingRequest")`
= TitleCase(pkg) + struct name; see `VERIFYING_A_PACKET.md` §9 "Linkage".

Also carry a comment block above the fixture recording the derivation, in the
style of `hit_test.go:66-79`: the `COutPacket(N)` per version, and the fact that
the body is `Encode4(dwID)` + `Encode1(touching)` on every version so no gate is
needed.

- [ ] **Step 2: Write `touching.go`**

```go
const TouchReactorHandle = "TouchReactorHandle"

// TouchingRequest - CReactorPool::FindTouchReactorAroundLocalUser
// packet-audit:fname CReactorPool::FindTouchReactorAroundLocalUser
type TouchingRequest struct {
    oid      uint32
    touching bool
}
```

Accessors `Oid() uint32` and `Touching() bool`. `Operation() string` returns
`TouchReactorHandle`. `String() string` returns
`fmt.Sprintf("oid [%d], touching [%t]", m.oid, m.touching)`.

`Encode(l logrus.FieldLogger, ctx context.Context) func(options map[string]interface{}) []byte`
writes `w.WriteInt(m.oid)` then `w.WriteByte(1)` / `w.WriteByte(0)`.
`Decode(_ logrus.FieldLogger, ctx context.Context) func(r *request.Reader, options map[string]interface{})`
sets `m.oid = r.ReadUint32()` and `m.touching = r.ReadByte() == 1`.

Neither method needs the tenant, so neither calls `tenant.MustFromContext` — do
**not** add a `MajorAtLeast` gate here; §1.2 measured the layout as
version-invariant, and an unused gate is a false claim about the wire.

- [ ] **Step 3: Add the packet-audit fname case**

In `tools/packet-audit/cmd/run.go`, in the `// --- Combat: reactor
(serverbound) ---` group immediately after the `CReactorPool::FindHitReactor`
case at line 1264:

```go
case "CReactorPool::FindTouchReactorAroundLocalUser":
    // CSV: TOUCHING_REACTOR — atlas TouchingRequest (handle = "TouchReactorHandle").
    return []candidate{{name: "TouchingRequest", pkg: "reactor", dir: csvpkg.DirServerbound}}
```

### Verification

- `cd libs/atlas-packet && go build ./... && go test ./reactor/...` passes.
- `cd tools/packet-audit && go build ./...` passes.

---

## Task 3: Registry and support-file corrections

Depends on Task 1.

### Files

- `docs/packets/registry/gms_v72.yaml` — add the `TOUCHING_REACTOR` serverbound entry
- `docs/packets/registry/gms_v79.yaml` — add the `TOUCHING_REACTOR` serverbound entry
- `docs/packets/registry/gms_v84.yaml` — correct the opcode at line 3936 if Task 1 measured something other than 206
- `docs/packets/audits/support/gms_v48.md` — line 719: keep `n-a`, state it was measured
- `docs/packets/audits/support/gms_v61.md` — same correction
- `docs/packets/audits/support/gms_v72.md` — line 671: `n-a` → the derived opcode
- `docs/packets/audits/support/gms_v79.md` — same correction
- `docs/tasks/task-249-touch-activated-reactors/opcode-derivation.md` — **new file** created by Task 1; read-only here

Also touch `docs/packets/registry/gms_v83.yaml`, `gms_v87.yaml`,
`gms_v92.yaml`, `gms_v95.yaml`, `jms_v185.yaml` **only if** Task 1 measured an
opcode that disagrees with the existing entry. If they agree, leave them alone.

No Go module. This task exceeds the ~6-file guidance because the edits are the
same one-line correction repeated across parallel per-version files; see
`context.md`.

### Steps

- [ ] **Step 1: Add the missing v72 / v79 registry entries**

  Entry shape, copied from `docs/packets/registry/gms_v83.yaml:3176`:

  ```yaml
  - op: TOUCHING_REACTOR
    direction: serverbound
    opcode: <decimal, from Task 1>
    fname: CReactorPool::FindTouchReactorAroundLocalUser
    provenance: ida-discovered
    ida:
      address: <decimal address, from Task 1>
  ```

  Insert at the sorted position for that opcode within the file's ordering.
  Opcodes 196 (v72) and 198 (v79) currently have no serverbound entry — verified
  at plan time — so there is no collision to resolve.

- [ ] **Step 2: Correct the four support files**

  In `gms_v72.md` and `gms_v79.md`, the row currently reads:

  ```
  | op | TOUCHING_REACTOR | serverbound |  | n-a |  |
  ```

  Replace the empty opcode column with the derived hex value and the state with
  the state the matrix will carry after Task 5; put the derivation address in the
  note column.

  In `gms_v48.md` and `gms_v61.md`, keep `n-a` and put the positive absence
  evidence from Task 1 Step 5 in the note column — the opcode-construction sweep
  result and the sibling cross-check — so the `n-a` reads as measured rather than
  interpolated.

- [ ] **Step 3: Correct the v84 opcode if it moved**

  If Task 1 measured a v84 opcode other than 206, update
  `docs/packets/registry/gms_v84.yaml:3936`'s `opcode:` field and add an `ida:`
  block with the address. Add a `note:` recording that the prior `csv-import`
  value was wrong and change `provenance:` to `ida-discovered`.

### Verification

- `grep -n "TOUCHING_REACTOR" docs/packets/registry/*.yaml` lists an entry for
  every in-scope version.
- Every opcode in the registry matches `opcode-derivation.md`.

---

## Task 4: Seed-template routing

Depends on Task 1.

### Files

- `services/atlas-configurations/seed-data/templates/template_gms_72_1.json` — add the handler entry
- `services/atlas-configurations/seed-data/templates/template_gms_79_1.json` — add the handler entry
- `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` — add the handler entry
- `services/atlas-configurations/seed-data/templates/template_gms_84_1.json` — add the handler entry
- `services/atlas-configurations/seed-data/templates/template_gms_87_1.json` — add the handler entry
- `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` — add the handler entry
- `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` — add the handler entry
- `services/atlas-configurations/seed-data/templates/template_jms_185_1.json` — add the handler entry
- `docs/packets/TEMPLATE_CONVENTIONS.md` — read-only; entry conventions
- `tools/template-opcode-order-guard.sh` — read-only; the sorted-insert rule this task must satisfy

`template_gms_12_1.json`, `template_gms_48_1.json` and `template_gms_61_1.json`
gain nothing.

No Go module. Eight files, but one identical JSON object inserted into each; see
`context.md`.

### Steps

- [ ] **Step 1: Insert the handler entry in each in-scope template**

  Into the `socket.handlers` array, at its **sorted opCode position** (the array
  is strictly ascending by opcode and `tools/template-opcode-order-guard.sh`
  enforces it — never append):

  ```json
  {
    "opCode": "0x<HEX from Task 1>",
    "validator": "LoggedInValidator",
    "handler": "TouchReactorHandle",
    "fname": "CReactorPool::FindTouchReactorAroundLocalUser",
    "services": [
      "channel"
    ]
  }
  ```

  Shape copied from the `ReactorHitHandle` entry in the same file (e.g.
  `template_gms_83_1.json:1954-1961`).

  Plan-time opcodes, to be replaced by Task 1's measured values if they differ:
  gms_72 `0xC4`, gms_79 `0xC6`, gms_83 `0xCE`, gms_84 **unknown — from Task 1**,
  gms_87 `0xDB`, gms_92 `0xF3`, gms_95 `0xFA`, jms_185 `0xD9`. All eight slots
  were confirmed free in `socket.handlers` at plan time.

  Note: `template_gms_92_1.json` routes no `ReactorHitHandle`. That is a
  pre-existing gap; this task adds only `TouchReactorHandle` there and does not
  fix it (design §8).

- [ ] **Step 2: Run the guards**

  ```sh
  tools/template-opcode-order-guard.sh
  tools/template-duplicate-binding-guard.sh
  ```

  `tools/template-symbol-check.sh` requires the codec's
  `TouchReactorHandle` string literal to exist, so run it only after Task 2 has
  landed:

  ```sh
  tools/template-symbol-check.sh services/atlas-configurations/seed-data/templates/template_gms_83_1.json
  ```

### Verification

- All three guard scripts exit 0.
- `grep -c TouchReactorHandle services/atlas-configurations/seed-data/templates/*.json`
  shows 1 for each of the eight in-scope templates and 0 for gms_12/48/61.

---

## Task 5: Audit reports, export splice, and matrix promotion

Depends on Tasks 1, 2, 3, 4.

`TOUCHING_REACTOR` is **tier-0** (`status.json` `"tier1": false`), so per
`VERIFYING_A_PACKET.md` §7 this task must **not** pin evidence records — a
tier-0 cell promotes on the tool ✅ plus the marker. What it does need, per §9,
is the audit report per version, generated from the export.

### Files

- `docs/packets/ida-exports/gms_v83.json` — splice in the newly-named function entry
- `docs/packets/ida-exports/gms_v84.json` — splice in the newly-named function entry
- `docs/packets/ida-exports/gms_v92.json` — splice in the newly-named function entry
- `docs/packets/ida-exports/gms_v72.json` — splice only if the fname is absent
- `docs/packets/ida-exports/gms_v79.json` — splice only if the fname is absent
- `docs/packets/audits/status.json` — regenerated
- `docs/packets/audits/STATUS.md` — regenerated
- `docs/packets/audits/VERIFYING_A_PACKET.md` — read-only; §7–§10 are the procedure
- `docs/tasks/task-249-touch-activated-reactors/opcode-derivation.md` — **new file** created by Task 1; read-only here

Report outputs land in `docs/packets/audits/<version>/ReactorTouchingRequest.{json,md}`
— eight new files, one per in-scope version.

Module: `tools/packet-audit`.

### Steps

- [ ] **Step 1: Confirm the fname resolves in each export**

  ```sh
  for v in gms_v72 gms_v79 gms_v83 gms_v84 gms_v87 gms_v92 gms_v95 gms_jms_185; do
    printf '%s: ' "$v"
    grep -c 'FindTouchReactorAroundLocalUser' "docs/packets/ida-exports/$v.json"
  done
  ```

  (`jms_v185`'s export file is named `gms_jms_185.json`.)

  Any version returning 0 needs a splice.

- [ ] **Step 2: Splice the missing entries**

  Per §10, **never overwrite a committed export**. Harvest to a temp file with
  `-prior-export "" -pending <roster.md> -descent-depth 12` against that
  version's IDB (`-ida-database <session id>` — its own, never another's), then
  copy only the `CReactorPool::FindTouchReactorAroundLocalUser` entry (and any
  helper it descends into that the committed export lacks) into the committed
  file. Absent-only for helpers; overwrite only the sender entry if a stub for
  it already exists.

  If report generation later fails with "delegate to COutPacket: not in export",
  strip that one `{op: Delegate, ref: COutPacket}` call from the spliced entry —
  it is the packet constructor, not a wire read (§10).

- [ ] **Step 3: Generate the audit reports**

  For each in-scope version, run the ROOT command to a temp output and copy only
  the one report pair into the committed tree:

  ```sh
  go run ./tools/packet-audit \
    -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
    -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
    -template services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
    -ida-source docs/packets/ida-exports/gms_v83.json \
    -output /tmp/rpt
  ```

  then copy `/tmp/rpt/gms_v83/ReactorTouchingRequest.json` and `.md` into
  `docs/packets/audits/gms_v83/`. Repeat per version with that version's
  template and export.

  Shape reference: `docs/packets/audits/gms_v83/ReactorHitRequest.json`.

- [ ] **Step 4: Regenerate and check the matrix**

  ```sh
  go run ./tools/packet-audit matrix
  go run ./tools/packet-audit matrix --check
  ```

  `--check` must exit **0**. Each in-scope `TOUCHING_REACTOR` cell must read
  verified; `gms_v48` and `gms_v61` stay `n-a`.

  If `--check` reports an `n-a` consistency failure — a `TOUCHING_REACTOR` cell
  is `n-a` while a same-family sibling is verified on that version — record the
  positive absence proof from Task 1 Step 5 in
  `docs/packets/feature-na-evidence.yaml`. Never weaken the gate. (Note:
  `docs/packets/feature-families.yaml` currently declares no reactor family, so
  this may not fire.)

- [ ] **Step 5: Commit the artifacts together**

  Codec test + reports + STATUS.md + status.json in one commit, per §8.

### Verification

- `go run ./tools/packet-audit matrix --check` exits 0.
- `grep -A2 '"op": "TOUCHING_REACTOR"' docs/packets/audits/status.json` shows
  `verified` for all eight in-scope cells.

---

## Task 6: `atlas-data` — expose `activateByTouch` and `touchAreaInfo`

### Files

- `services/atlas-data/atlas.com/data/reactor/rest.go` — add two fields and the `AreaRestModel` type
- `services/atlas-data/atlas.com/data/reactor/reader.go` — populate them; `activateByTouch` is already read at line 80 as the `loadArea` local
- `services/atlas-data/atlas.com/data/reactor/reader_test.go` — new cases
- `services/atlas-data/atlas.com/data/xml/model.go` — read-only; `Node.CanvasNodes`, `CanvasNode.Width/Height`, `CanvasNode.GetPoint`
- `services/atlas-data/atlas.com/data/point/rest.go` — read-only; `point.RestModel{X, Y int16}`

Module root: `services/atlas-data/atlas.com/data`.

### Steps

- [ ] **Step 1: Confirm the canvas-origin sign convention**

  Design §5.1 flags this as an **implementation gate**: the formula below is
  inferred from canvas anchor semantics, not measured. Before writing the
  reader, confirm it against `CReactorPool::LoadReactorLayer` (gms_v83
  `@0x7348a0`; session id from `mcp__ida-pro__idb_list`) — specifically how the
  layer's `lt`/`rb` relate to the reactor's map position and the canvas
  `origin`. Record the confirmation (or the corrected formula) in
  `docs/tasks/task-249-touch-activated-reactors/opcode-derivation.md` under a
  "Canvas origin convention" heading.

  If the convention differs, only the formula in Step 3 changes — no other part
  of this plan moves.

- [ ] **Step 2: Write the failing test**

  Add to `services/atlas-data/atlas.com/data/reactor/reader_test.go`. Test
  fixtures are `const` XML strings plus a `var xxxNodeProvider = func(path
  string, id uint32) model.Provider[xml.Node] { return
  xml.FromByteArrayProvider([]byte(xxxXML)) }` — pattern at `reader_test.go:341`.
  Assertions are plain `if`/`t.Fatalf`, as in `TestReader` at `:357`.

  `TestReaderActivateByTouch` — table-driven over two providers.

| case | fixture | assertion |
|---|---|---|
| flag set | new `touchTestXML` (below) | `rm.ActivateByTouch == true` |
| flag absent | existing `fixedNodeProvider` (reactor 1002000, no `activateByTouch` node) | `rm.ActivateByTouch == false` |

  `TestReaderTouchAreaInfo` — uses the same new `touchTestXML`, which is
  reactor `2406000` verbatim from `Reactor.wz` (three states; state 0 canvas
  `115×45` origin `(53,-24)`, state 1 canvas `122×137` origin `(56,68)`, state 2
  canvas `1×1` origin `(0,0)`). Include only the state-level `canvas name="0"`
  nodes and the `event` subtrees — the `hit` subdirectories are irrelevant to
  this derivation and make the fixture unreadable.

| state | canvas | expected TL | expected BR |
|---|---|---|---|
| 0 | `115×45`, origin `(53,-24)` | `(-53, 24)` | `(62, 69)` |
| 1 | `122×137`, origin `(56,68)` | `(-56, -68)` | `(66, 69)` |
| 2 | `1×1`, origin `(0,0)` | `(0, 0)` | `(1, 1)` |

  Assert `len(rm.TouchAreaInfo) == 3` and each state's four coordinates.

  Add a third case to `TestReaderTouchAreaInfo` (or a sibling function): a state
  directory with **no** canvas node contributes **no** map entry.

  **If Step 1 corrects the formula, correct this table with it before writing
  the code — the table is the spec, and a table that matches a wrong formula is
  worse than no test.**

- [ ] **Step 3: Extend the REST model**

  In `rest.go`:

  ```go
  type AreaRestModel struct {
      TL point.RestModel `json:"tl"`
      BR point.RestModel `json:"br"`
  }
  ```

  and on `RestModel`, after `BR`:

  ```go
  ActivateByTouch bool                    `json:"activateByTouch"`
  TouchAreaInfo   map[int8]AreaRestModel  `json:"touchAreaInfo"`
  ```

- [ ] **Step 4: Populate them in the reader**

  In `reader.go`, the value is already computed at line 80 as `loadArea`.
  Rename nothing; just set `ActivateByTouch: loadArea` on the `RestModel`
  literal at line 82, and initialise `TouchAreaInfo: map[int8]AreaRestModel{}`
  alongside the other maps.

  In the `for rid != nil` loop (line 87), before or after the `event` handling,
  derive the rectangle from the state directory's own canvas:

  ```go
  for _, c := range rid.CanvasNodes {
      if c.Name != "0" {
          continue
      }
      w, err := strconv.Atoi(c.Width)
      if err != nil {
          break
      }
      h, err := strconv.Atoi(c.Height)
      if err != nil {
          break
      }
      ox, oy := c.GetPoint("origin", 0, 0)
      m.TouchAreaInfo[i] = AreaRestModel{
          TL: point.RestModel{X: int16(-ox), Y: int16(-oy)},
          BR: point.RestModel{X: int16(int32(w) - ox), Y: int16(int32(h) - oy)},
      }
      break
  }
  ```

  Emit the map for every reactor, not only touch ones (design §5.1). The
  `if t == 100` `loadArea` use at line 111 is **untouched**.

  Also set `ActivateByTouch: false` and `TouchAreaInfo: map[int8]AreaRestModel{}`
  on the early-return `RestModel` for the no-`info` case at line 46, so the field
  is never a nil map on the wire.

### Verification

- `cd services/atlas-data/atlas.com/data && go build ./... && go test ./reactor/... ./point/...` passes.

---

## Task 7: `atlas-reactors` — plumb the flag and the touch areas into `reactor/data`

Depends on Task 6 (the JSON shape).

### Files

- `services/atlas-reactors/atlas.com/reactors/reactor/data/area/model.go` — **new file**
- `services/atlas-reactors/atlas.com/reactors/reactor/data/area/rest.go` — **new file**
- `services/atlas-reactors/atlas.com/reactors/reactor/data/area/model_json.go` — **new file**
- `services/atlas-reactors/atlas.com/reactors/reactor/data/model.go` — two fields, two accessors
- `services/atlas-reactors/atlas.com/reactors/reactor/data/rest.go` — two `RestModel` fields, populated in `Extract`
- `services/atlas-reactors/atlas.com/reactors/reactor/data/model_json.go` — two fields on `modelJSON`
- `services/atlas-reactors/atlas.com/reactors/reactor/data/rest_test.go` — **new file**
- `services/atlas-reactors/atlas.com/reactors/reactor/data/point/model.go` — read-only; the package to mirror

Module root: `services/atlas-reactors/atlas.com/reactors`.

Patterns to copy: the `point` package is the exact three-file shape the new
`area` package takes (`model.go` + `rest.go` + `model_json.go`, unexported
fields, `Extract`, `MarshalJSON`/`UnmarshalJSON`).

### Steps

- [ ] **Step 1: Write the failing test**

`services/atlas-reactors/atlas.com/reactors/reactor/data/rest_test.go`, package
`data`. Plain `if`/`t.Fatalf` assertions; no builder or fixture helper is needed
— `data.Extract` takes a literal `RestModel`.

`TestExtractTouchFields` — table-driven.

| case | input `RestModel` | assertion |
|---|---|---|
| flag set with areas | `ActivateByTouch: true`, `TouchAreaInfo: map[int8]area.RestModel{0: {TL: point.RestModel{X: -53, Y: 24}, BR: point.RestModel{X: 62, Y: 69}}}` | `m.ActivateByTouch() == true`; `a, ok := m.TouchArea(0)` → `ok == true`, `a.TL().X() == -53`, `a.TL().Y() == 24`, `a.BR().X() == 62`, `a.BR().Y() == 69` |
| fields absent (FR-12) | `RestModel{Name: "x"}` — no `ActivateByTouch`, no `TouchAreaInfo` | `m.ActivateByTouch() == false`; `_, ok := m.TouchArea(0)` → `ok == false` (no panic on the nil map) |
| unknown state | flag set, `TouchAreaInfo` has only state 0 | `_, ok := m.TouchArea(1)` → `ok == false` |

`TestModelJSONRoundTripTouchFields` — marshal a `data.Model` built by
`Extract` from the first case above, unmarshal into a fresh `data.Model`, and
assert `ActivateByTouch()` and `TouchArea(0)`'s four coordinates survive. This
guards the Redis registry round-trip, which serialises `data.Model` through
`model_json.go`.

- [ ] **Step 2: Create the `area` package**

`model.go`:

```go
package area

import "atlas-reactors/reactor/data/point"

type Model struct {
    tl point.Model
    br point.Model
}

func (m Model) TL() point.Model { return m.tl }
func (m Model) BR() point.Model { return m.br }
```

`rest.go`: `RestModel{TL, BR point.RestModel}` with json tags `tl` / `br`, and
`Extract(rm RestModel) (Model, error)` delegating to `point.Extract` for each.

`model_json.go`: `modelJSON{TL, BR point.Model}` with tags `tl` / `br`, plus
`MarshalJSON`/`UnmarshalJSON` on `Model` — same shape as
`reactor/data/point/model_json.go`.

- [ ] **Step 3: Extend `data.Model`**

Add `activateByTouch bool` and `touchAreaInfo map[int8]area.Model` fields, plus:

```go
func (m Model) ActivateByTouch() bool { return m.activateByTouch }

func (m Model) TouchArea(state int8) (area.Model, bool) {
    if m.touchAreaInfo == nil {
        return area.Model{}, false
    }
    v, ok := m.touchAreaInfo[state]
    return v, ok
}
```

The nil guard mirrors `Timeout`/`TimeoutNextState` in the same file.

- [ ] **Step 4: Extend `RestModel` and `Extract`**

Add `ActivateByTouch bool \`json:"activateByTouch"\`` and
`TouchAreaInfo map[int8]area.RestModel \`json:"touchAreaInfo"\`` to `RestModel`.

In `Extract`, build the area map the same way the existing code builds `si`:
iterate `rm.TouchAreaInfo`, call `area.Extract` per entry, and propagate the
error. Leave the map `nil` when `rm.TouchAreaInfo` is nil — `TouchArea`'s guard
handles it, and a nil map is what FR-12 requires for old cached data.

- [ ] **Step 5: Extend `modelJSON`**

Add `ActivateByTouch bool \`json:"activateByTouch"\`` and
`TouchAreaInfo map[int8]area.Model \`json:"touchAreaInfo"\`` and wire both in
`MarshalJSON` and `UnmarshalJSON`.

### Verification

- `cd services/atlas-reactors/atlas.com/reactors && go build ./... && go test ./reactor/...` passes.

---

## Task 8: `atlas-reactors` — read-only `character` client

### Files

- `services/atlas-reactors/atlas.com/reactors/character/processor.go` — **new file**
- `services/atlas-reactors/atlas.com/reactors/character/requests.go` — **new file**
- `services/atlas-reactors/atlas.com/reactors/character/rest.go` — **new file**
- `services/atlas-reactors/atlas.com/reactors/character/processor_test.go` — **new file**
- `services/atlas-reactors/atlas.com/reactors/character/mock/processor.go` — **new file**
- `services/atlas-maps/atlas.com/maps/character/processor.go` — read-only; the precedent to copy verbatim
- `services/atlas-maps/atlas.com/maps/character/processor_test.go` — read-only; the `baseURLProvider` httptest seam

Module root: `services/atlas-reactors/atlas.com/reactors`. `atlas-rest` is
already a dependency (`go.mod:10`).

Patterns to copy: `services/atlas-maps/atlas.com/maps/character/` is this exact
package, one narrower field. Copy `processor.go`, `requests.go`, `rest.go` and
`processor_test.go` and drop `Hp` — `atlas-reactors` needs only position.

### Steps

- [ ] **Step 1: Write the failing test**

`character/processor_test.go`, copying the httptest + `baseURLProvider` override
setup from `services/atlas-maps/atlas.com/maps/character/processor_test.go`
verbatim.

`TestSnapshot` — table-driven.

| case | handler response | assertion |
|---|---|---|
| success | JSON:API `characters` resource, id `1`, attributes `{"x": 250, "y": -300}` | `x == 250`, `y == -300`, `err == nil` |
| not found | HTTP 404 | `err != nil` |
| malformed body | HTTP 200 with `not json` | `err != nil` |

- [ ] **Step 2: Write the package**

`rest.go` — `RestModel{Id uint32 \`json:"-"\`; X int16 \`json:"x"\`; Y int16 \`json:"y"\`}`
with `GetName() == "characters"`, `GetID`, `SetID`, and the two no-op
`SetToOneReferenceID` / `SetToManyReferenceIDs` shims the maps copy carries
(they are required by api2go when the upstream sends a `relationships` block).

`requests.go` — `Resource = "characters"`, `ById = Resource + "/%d"`, the
`baseURLProvider` test seam resolving `requests.RootUrlFor(ctx, "CHARACTERS")`,
and `requestById`.

`processor.go` — `Processor` interface with
`Position(characterId uint32) (int16, int16, error)`, `ProcessorImpl`,
`NewProcessor(l, ctx)`, `var _ Processor = (*ProcessorImpl)(nil)`.

Name the method `Position`, not `Snapshot`: this package returns only position,
and `Snapshot` in `atlas-maps` means "position + HP".

`mock/processor.go` — `ProcessorMock{PositionFunc func(uint32) (int16, int16, error)}`
implementing `Processor`, shaped like
`services/atlas-reactors/atlas.com/reactors/reactor/data/mock/processor.go`.
Task 12 injects it.

### Verification

- `cd services/atlas-reactors/atlas.com/reactors && go build ./... && go test ./character/...` passes.

---

## Task 9: `atlas-reactors` — the touch latch on `Registry`

### Files

- `services/atlas-reactors/atlas.com/reactors/reactor/registry.go` — add the `touches` hash, three methods, and the clears
- `services/atlas-reactors/atlas.com/reactors/reactor/registry_touch_test.go` — **new file**
- `libs/atlas-redis/keyed_hash.go` — read-only; `SetNX`, `Del`, `DeleteKey`
- `services/atlas-reactors/atlas.com/reactors/reactor/processor_test.go` — read-only; `setupTestRegistry` at line 25 is the miniredis harness

Module root: `services/atlas-reactors/atlas.com/reactors`.

Patterns to copy: `TryClaimSpot` / `ReleaseSpot` / `ClearAllSpotsForMap`
(`registry.go`, near the end) are the same `SetNX`-based claim/release/wipe trio
this task adds, keyed on reactor id instead of map key.

### Steps

- [ ] **Step 1: Write the failing test**

`registry_touch_test.go`, package `reactor`. Uses `setupTestRegistry(t)` and
`setupTestTenant()` from `processor_test.go` (same package, no import needed).

`TestRegistry_TouchLatch` — sequential assertions in one function, since the
latch is stateful:

| step | call | expect |
|---|---|---|
| 1 | `GetRegistry().TryLatchTouch(ten, 42, 1000)` | `true` (first claim wins) |
| 2 | `GetRegistry().TryLatchTouch(ten, 42, 1000)` | `false` (already latched) |
| 3 | `GetRegistry().TryLatchTouch(ten, 42, 2000)` | `true` (different character) |
| 4 | `GetRegistry().ClearTouch(ten, 42, 1000)` then `TryLatchTouch(ten, 42, 1000)` | `true` (cleared, re-latchable) |
| 5 | `GetRegistry().ClearAllTouches(ten, 42)` then `TryLatchTouch(ten, 42, 2000)` | `true` (wipe released every character) |

`TestRegistry_TouchLatchIsTenantScoped` — latch `(tenA, 42, 1000)`, then assert
`TryLatchTouch(tenB, 42, 1000)` returns `true`. Build the second tenant with
`tenant.Create(uuid.New(), "GMS", 83, 1)` as `setupTestTenant` does.

- [ ] **Step 2: Add the hash to `Registry`**

Field on the `Registry` struct, alongside `spots`:

```go
touches *atlas.TenantKeyedHash[uint32] // field=characterId -> "1"
```

In `InitRegistry`:

```go
touches: atlas.NewTenantKeyedHash[uint32](client, "reactor:touch", reactorIdStr),
```

`reactorIdStr` is already defined in this file and has the right signature.

- [ ] **Step 3: Add the three methods**

```go
// TryLatchTouch atomically records that characterId has entered reactorId's
// touch area. Returns true if this caller set the latch -- false means the
// character is already inside and the touch must be ignored (FR-18).
func (r *Registry) TryLatchTouch(t tenant.Model, reactorId uint32, characterId uint32) bool

// ClearTouch releases one character's latch, on the client's touching=0.
func (r *Registry) ClearTouch(t tenant.Model, reactorId uint32, characterId uint32)

// ClearAllTouches drops every latch for a reactor, on removal or teardown.
func (r *Registry) ClearAllTouches(t tenant.Model, reactorId uint32)
```

`TryLatchTouch` wraps `r.touches.SetNX(context.Background(), t, reactorId,
strconv.FormatUint(uint64(characterId), 10), "1")` and returns `false` on error,
mirroring `TryClaimSpot`. `ClearTouch` wraps `Del`; `ClearAllTouches` wraps
`DeleteKey`.

- [ ] **Step 4: Clear the latch on reactor teardown**

Add `GetRegistry().ClearAllTouches(t, id)` inside `Registry.Remove`, next to the
existing `r.mapSets.Remove` / `r.allocator.Release` calls (use `r.touches`
directly there, since it is a method on `*Registry`). That single insertion
covers `Destroy`, `DestroyInField` and the teardown sweep, all of which route
through `Remove`.

### Verification

- `cd services/atlas-reactors/atlas.com/reactors && go test ./reactor/...` passes,
  including the pre-existing tests.

---

## Task 10: `atlas-reactors` — split `Hit` into `selectNextState` + `advance`

Behaviour-preserving refactor. No test is added; the **existing**
`processor_test.go` suite is the guard and must pass **unedited**.

### Files

- `services/atlas-reactors/atlas.com/reactors/reactor/processor.go` — split `Hit`
- `services/atlas-reactors/atlas.com/reactors/reactor/processor_test.go` — read-only; must not be modified

Module root: `services/atlas-reactors/atlas.com/reactors`.

### Steps

- [ ] **Step 1: Extract `selectNextState`**

```go
// selectNextState applies the hit path's skill-gating predicate to a state's
// events. Returns (-1, 0) when no event matches. Touch does NOT use this --
// see Touch's own selection (FR-16).
func selectNextState(stateEvents []state.Model, skillId uint32) (int8, int32) {
    for _, event := range stateEvents {
        if len(event.ActiveSkills()) == 0 || containsSkill(event.ActiveSkills(), skillId) {
            return event.NextState(), event.Type()
        }
    }
    return -1, 0
}
```

This is the loop currently inline in `Hit` (`processor.go`, the
`var nextState int8 = -1` block), moved verbatim.

- [ ] **Step 2: Extract `advance`**

```go
func (p *ProcessorImpl) advance(r Model, characterId uint32, nextState int8, matchedEventType int32) error
```

Its body is **everything in `Hit` from `_, hasNextState := stateInfo[nextState]`
onward**, moved verbatim: the `persistsAtEndState` branch, the
`isTerminalState` branch, the `GetRegistry().Update` calls,
`scheduleStateTimeout`, `p.Trigger`, `p.TriggerAndDestroy`, and the
`hitStatusEventProvider` emissions.

`advance` re-derives what the moved code closes over:
`t := tenant.MustFromContext(p.ctx)`, `reactorId := r.Id()`,
`stateInfo := r.Data().StateInfo()`. Keep every log message byte-identical.

- [ ] **Step 3: Rewrite `Hit` in terms of both**

`Hit` keeps, in order and unchanged: the `GetById` lookup and its error return,
`cancelStateTimeout(reactorId)`, the `hitActionsCommandProvider` emission with
its warn-and-continue, the `!ok || len(stateEvents) == 0` →
`TriggerAndDestroy` fall-through, and the `nextState == -1` →
`TriggerAndDestroy` fall-through. It then calls
`p.advance(r, characterId, nextState, matchedEventType)`.

**Do not change `Hit`'s signature, its error values, or any log string.** FR-19
is the acceptance bar and `processor_test.go` is how it is measured.

### Verification

- `cd services/atlas-reactors/atlas.com/reactors && go build ./... && go test ./reactor/...` passes.
- `git diff --stat services/atlas-reactors/atlas.com/reactors/reactor/processor_test.go`
  shows **no change**.

---

## Task 11: `atlas-reactors` — TOUCH command types, producer, and consumer arm

### Files

- `services/atlas-reactors/atlas.com/reactors/reactor/kafka.go` — two new command-type constants and two body structs
- `services/atlas-reactors/atlas.com/reactors/reactor/producer.go` — `touchActionsCommandProvider`
- `services/atlas-reactors/atlas.com/reactors/kafka/consumer/reactor/consumer.go` — register and implement `handleTouch`
- `services/atlas-reactors/atlas.com/reactors/kafka/consumer/reactor/consumer_test.go` — new case
- `services/atlas-reactors/atlas.com/reactors/reactor/processor.go` — add `Touch` to the `Processor` interface only (implementation is Task 12)
- `services/atlas-reactors/atlas.com/reactors/reactor/mock/processor.go` — add `TouchFunc`

Module root: `services/atlas-reactors/atlas.com/reactors`.

Patterns to copy: `handleHit` in `consumer.go`;
`triggerActionsCommandProvider` in `producer.go`.

### Steps

- [ ] **Step 1: Write the failing test**

Add to `services/atlas-reactors/atlas.com/reactors/kafka/consumer/reactor/consumer_test.go`,
following whatever harness the existing cases in that file use.

`TestHandleTouch` — table-driven.

| case | command | expect |
|---|---|---|
| routed | `Type: reactor.CommandTypeTouch`, `Body: TouchCommandBody{ReactorId: 42, CharacterId: 1000, Touching: true}` | `Touch(42, 1000, true)` called exactly once |
| wrong type ignored | `Type: reactor.CommandTypeHit` with a `TouchCommandBody` | `Touch` not called |
| leaving forwarded | `Type: reactor.CommandTypeTouch`, `Touching: false` | `Touch(42, 1000, false)` called exactly once |

- [ ] **Step 2: Add the constants and bodies**

In `reactor/kafka.go`, in the reactor-command const block:

```go
CommandTypeTouch = "TOUCH"
```

and the body:

```go
type TouchCommandBody struct {
    ReactorId   uint32 `json:"reactorId"`
    CharacterId uint32 `json:"characterId"`
    Touching    bool   `json:"touching"`
}
```

In the reactor-actions const block:

```go
CommandTypeActionsTouch = "TOUCH"
```

and:

```go
// touchActionsBody represents the body of a TOUCH command to atlas-reactor-actions
type touchActionsBody struct {
    CharacterId uint32 `json:"characterId"`
}
```

`CommandTypeActionsTouch` and `CommandTypeTouch` are both `"TOUCH"` but live on
different topics; they are separate constants for the same reason
`CommandTypeHit` and `CommandTypeActionsHit` are.

- [ ] **Step 3: Add the actions producer**

`touchActionsCommandProvider(r Model, characterId uint32) model.Provider[[]kafka.Message]`
— a verbatim copy of `triggerActionsCommandProvider` with
`Type: CommandTypeActionsTouch` and `Body: touchActionsBody{CharacterId: characterId}`.

- [ ] **Step 4: Add `Touch` to the interface and the mock**

On the `Processor` interface in `processor.go`, after `Hit`:

```go
Touch(reactorId uint32, characterId uint32, touching bool) error
```

Add a matching `TouchFunc` field and method to
`reactor/mock/processor.go`. **Task 12 writes the real implementation** — to
keep this task compilable, implement `ProcessorImpl.Touch` here as the
`touching == false` short-circuit only:

```go
func (p *ProcessorImpl) Touch(reactorId uint32, characterId uint32, touching bool) error {
    t := tenant.MustFromContext(p.ctx)
    if !touching {
        GetRegistry().ClearTouch(t, reactorId, characterId)
        p.l.Debugf("Character [%d] left touch area of reactor [%d].", characterId, reactorId)
        return nil
    }
    return nil
}
```

That is a real, correct implementation of the leave path (design §6.1), not a
stub — Task 12 replaces the trailing `return nil` with the rejection ladder.
Sequence Task 12 immediately after this one.

- [ ] **Step 5: Register and implement `handleTouch`**

In `consumer.go`'s `InitHandlers`, after the `handleHit` registration:

```go
if _, err := rf(t, message.AdaptHandler(message.PersistentConfig(handleTouch))); err != nil {
    return err
}
```

and:

```go
func handleTouch(l logrus.FieldLogger, ctx context.Context, c reactor.Command[reactor.TouchCommandBody]) {
    if c.Type != reactor.CommandTypeTouch {
        return
    }
    err := reactor.NewProcessor(l, ctx).Touch(c.Body.ReactorId, c.Body.CharacterId, c.Body.Touching)
    if err != nil {
        l.WithError(err).Errorf("Failed to process touch for reactor [%d].", c.Body.ReactorId)
    }
}
```

### Verification

- `cd services/atlas-reactors/atlas.com/reactors && go build ./... && go test ./...` passes.

---

## Task 12: `atlas-reactors` — `ProcessorImpl.Touch`

Depends on Tasks 7, 8, 9, 10, 11.

### Files

- `services/atlas-reactors/atlas.com/reactors/reactor/processor.go` — the real `Touch` body
- `services/atlas-reactors/atlas.com/reactors/reactor/processor_touch_test.go` — **new file**
- `services/atlas-reactors/atlas.com/reactors/character/processor.go` — read-only; `Position`
- `services/atlas-reactors/atlas.com/reactors/character/mock/processor.go` — **new file** created by Task 8; read-only here — the injection point
- `services/atlas-reactors/atlas.com/reactors/reactor/processor_test.go` — read-only; `newTestData` (line 388) and `setupTestRegistry` (line 25)

Module root: `services/atlas-reactors/atlas.com/reactors`.

`Touch` needs an injectable character client for tests. Add a package-level
seam in `processor.go` next to the existing helpers:

```go
// characterProcessor is the seam tests use to stand in for the atlas-character
// REST read. Production resolves the real client.
var characterProcessor = func(l logrus.FieldLogger, ctx context.Context) character.Processor {
    return character.NewProcessor(l, ctx)
}
```

Tests swap it and restore it with `t.Cleanup`.

### Steps

- [ ] **Step 1: Write the failing test**

`processor_touch_test.go`, package `reactor`. Reuses `setupTestRegistry`,
`setupTestLogger`, `setupTestTenant`, `setupTestContext` and `newTestData` from
`processor_test.go` (same package). Reactor construction copies
`TestHit_SkillReactorPersistsAtTerminal` (`processor_test.go:485-510`):
`NewModelBuilder(...).SetState(0).SetPosition(x, y).SetDelay(0).SetData(d)` then
`GetRegistry().Create(ten, builder)`.

Producer errors are tolerated the same way the `Hit` tests tolerate them
(`_ = NewProcessor(l, ctx).Touch(...)`); assertions are on registry state.

Every case places the reactor at `SetPosition(100, 100)` and gives it a state-0
touch area of `TL(-50, -50)` / `BR(50, 50)`, i.e. a world-space box of
`[50,150] × [50,150]`. Build the `data.Model` with `data.Extract` on a
`data.RestModel` carrying both `StateInfo` and `TouchAreaInfo` — extend
`newTestData` with a `touchAreaInfo map[int8]area.RestModel` parameter and an
`activateByTouch bool` parameter, updating its five existing call sites to pass
`false, nil`.

`TestTouch` — table-driven.

| case | reactor data | character position | expect |
|---|---|---|---|
| `accepts` | `activateByTouch: true`, area on state 0, state 0 → 1 via `{Type: 6, NextState: 1, ActiveSkills: []uint32{}}` | `(100, 100)` | reactor exists, `State() == 1` |
| `rejects_flag_unset` | `activateByTouch: false`, same area, same events | `(100, 100)` | `State() == 0`, reactor exists |
| `rejects_outside_bounds` | `activateByTouch: true`, same area, same events | `(500, 500)` | `State() == 0` |
| `rejects_on_left_edge` | as `accepts` | `(49, 100)` | `State() == 0` |
| `accepts_on_boundary` | as `accepts` | `(50, 100)` | `State() == 1` (bounds are inclusive) |
| `rejects_missing_area` | `activateByTouch: true`, `TouchAreaInfo` empty | `(100, 100)` | `State() == 0` |
| `rejects_unknown_reactor` | — | `(100, 100)` | `Touch(999999999, …)` returns an error; nothing panics |
| `rejects_position_error` | as `accepts`; mock `PositionFunc` returns an error | — | `State() == 0` |

`TestTouch_SkillGatedStateAdvances` — **the FR-16 regression guard**, its own
function so it cannot be diluted. `activateByTouch: true`, area on state 0,
state 0's only event is `{Type: 6, NextState: 1, ActiveSkills: []uint32{9001000}}`
and state 1 exists in `StateInfo` with `{Type: 7, NextState: 0, ActiveSkills: []uint32{9001000}}`.
Character at `(100, 100)`, `skillId` is not in play at all on the touch path.
Assert: the reactor **still exists** in the registry and `State() == 1`. If it
was destroyed, `Touch` reused the hit predicate and fell through to
`TriggerAndDestroy` — the exact inversion this task exists to prevent.

`TestTouch_EmptyStateIsNoOp` — **the OQ-6 guard**. `activateByTouch: true`,
area on state 0, `StateInfo` has **no** entry for state 0. Character inside.
Assert `Touch` returns `nil`, the reactor **still exists**, and `State() == 0`.

`TestTouch_Idempotence` — sequential, one function:

| step | call | expect |
|---|---|---|
| 1 | `Touch(id, 1000, true)` | `State() == 1` |
| 2 | `Touch(id, 1000, true)` again | still `State() == 1` (latched) |
| 3 | `Touch(id, 1000, false)` | still `State() == 1`; no error |
| 4 | `Touch(id, 1000, true)` | `State() == 0` (state 1's type-7 event cycles back) |

The reactor for this case is the cyclic `6109013` shape: state 0 →(type 6)→
state 1, state 1 →(type 7)→ state 0, both with an empty `ActiveSkills`, and a
touch area on **both** states.

`TestTouch_LeavingIsCheap` — `Touch(id, 1000, false)` against a reactor whose
`activateByTouch` is `false`, with `characterProcessor` swapped for a mock whose
`PositionFunc` calls `t.Fatal`. Asserts the leave path short-circuits before any
character read (design §6.1).

- [ ] **Step 2: Implement `Touch`**

Replace Task 11's trailing `return nil` with the rejection ladder, in this exact
order (design §6.2), each rejection logging reactor id, character id and reason
before returning:

1. `r, err := p.GetById(reactorId)` — on error, log and `return err`.
2. `!r.Data().ActivateByTouch()` — log, `return nil`.
3. `a, ok := r.Data().TouchArea(r.State())`; `!ok` — log, `return nil`.
4. `cx, cy, err := characterProcessor(p.l, p.ctx).Position(characterId)` — on
   error, log and `return nil` (a REST failure is not a reactor error).
5. AABB, inclusive on all four edges:
   ```go
   if cx < r.X()+a.TL().X() || cx > r.X()+a.BR().X() ||
       cy < r.Y()+a.TL().Y() || cy > r.Y()+a.BR().Y() {
       // log and return nil
   }
   ```
6. `if !GetRegistry().TryLatchTouch(t, reactorId, characterId) { /* log at debug; return nil */ }`
   — already inside, not an error (FR-18).

Then the acceptance path:

```go
cancelStateTimeout(reactorId)

err = producer.ProviderImpl(p.l)(p.ctx)(EnvCommandReactorActionsTopic)(touchActionsCommandProvider(r, characterId))
if err != nil {
    p.l.WithError(err).Warnf("Failed to emit TOUCH command to reactor-actions for reactor [%d].", reactorId)
}

stateEvents, ok := r.Data().StateInfo()[r.State()]
if !ok || len(stateEvents) == 0 {
    p.l.Debugf("No state events for reactor [%d] state [%d] on touch. No-op.", reactorId, r.State())
    return nil
}
return p.advance(r, characterId, stateEvents[0].NextState(), stateEvents[0].Type())
```

Three things this must **not** do, each an explicit design decision:

- It must **not** consult `event.ActiveSkills()` (FR-16). The template's
  `activateByTouch` flag is the gate.
- It must **not** call `TriggerAndDestroy` on an empty state (OQ-6) — that is
  `Hit`'s behaviour and stays `Hit`'s alone.
- It must **not** emit a `HIT` actions command (FR-17).

The actions command is emitted **before** `advance`, matching `Hit`'s ordering
so a script reading `reactorState` sees the state at time of activation
(design §6.5).

- [ ] **Step 3: Add the `character` import and the seam**

Import `atlas-reactors/character` in `processor.go` and declare
`characterProcessor` as specified above.

### Verification

- `cd services/atlas-reactors/atlas.com/reactors && go build ./... && go test ./...` passes.
- The pre-existing `processor_test.go` still passes unedited.

---

## Task 13: `atlas-channel` — handler, processor, producer

### Files

- `services/atlas-channel/atlas.com/channel/socket/handler/touch_reactor.go` — **new file**
- `services/atlas-channel/atlas.com/channel/reactor/processor.go` — add `Touch` to the interface and `ProcessorImpl`
- `services/atlas-channel/atlas.com/channel/reactor/producer.go` — `TouchCommandProvider`
- `services/atlas-channel/atlas.com/channel/kafka/message/reactor/kafka.go` — `CommandTypeTouch` and `TouchCommandBody`
- `services/atlas-channel/atlas.com/channel/reactor/mock/processor.go` — add `TouchFunc`
- `services/atlas-channel/atlas.com/channel/main.go` — register at line 1017's block
- `services/atlas-channel/atlas.com/channel/socket/handler/reactor_hit.go` — read-only; the shape to mirror

Module root: `services/atlas-channel/atlas.com/channel`.

### Steps

- [ ] **Step 1: Add the Kafka message types**

In `kafka/message/reactor/kafka.go`, in the command const block:

```go
CommandTypeTouch = "TOUCH"
```

and, after `HitCommandBody`:

```go
type TouchCommandBody struct {
    ReactorId   uint32 `json:"reactorId"`
    CharacterId uint32 `json:"characterId"`
    Touching    bool   `json:"touching"`
}
```

These must stay byte-compatible with `atlas-reactors`' `TouchCommandBody`
(Task 11) — same json tags, same types. The two services declare the envelope
independently, as they already do for `HitCommandBody`.

- [ ] **Step 2: Add the producer**

In `reactor/producer.go`, mirroring `HitCommandProvider` exactly, including
`key := producer.CreateKey(int(reactorId))` — keying on reactor id keeps a
reactor's touch and hit commands on the same partition and therefore ordered
relative to each other (design §4).

```go
func TouchCommandProvider(f field.Model, reactorId uint32, characterId uint32, touching bool) model.Provider[[]kafka.Message]
```

- [ ] **Step 3: Add `Touch` to the processor and the mock**

On the `Processor` interface in `reactor/processor.go`, after `Hit`:

```go
Touch(f field.Model, reactorId uint32, characterId uint32, touching bool) error
```

Implementation mirrors `Hit`: one `p.l.Debugf` then
`producer.ProviderImpl(p.l)(p.ctx)(reactor2.EnvCommandTopic)(TouchCommandProvider(f, reactorId, characterId, touching))`.

Add `TouchFunc` to `reactor/mock/processor.go`.

- [ ] **Step 4: Write the handler**

`socket/handler/touch_reactor.go`, mirroring `reactor_hit.go` — decode, debug
log, delegate. No validation, no session lookup beyond `s.Field()` and
`s.CharacterId()`; the authority does the validating (design §2).

```go
func TouchReactorHandleFunc(l logrus.FieldLogger, ctx context.Context, _ writer.Producer) func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
    return func(s session.Model, r *request.Reader, readerOptions map[string]interface{}) {
        p := reactor2.TouchingRequest{}
        p.Decode(l, ctx)(r, readerOptions)
        l.Debugf("[%s] read [%s]", p.Operation(), p.String())

        err := reactor.NewProcessor(l, ctx).Touch(s.Field(), p.Oid(), s.CharacterId(), p.Touching())
        if err != nil {
            l.WithError(err).Errorf("Unable to send touch command for reactor [%d].", p.Oid())
        }
    }
}
```

- [ ] **Step 5: Register the handler**

In `main.go`, immediately after line 1017:

```go
handlerMap[reactorsb.TouchReactorHandle] = handler.TouchReactorHandleFunc
```

### Verification

- `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...` passes.
- `tools/template-symbol-check.sh services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
  exits 0 (the `TouchReactorHandle` literal now resolves).

---

## Task 14: `atlas-reactor-actions` — `touchRules` model plumbing

### Files

- `services/atlas-reactor-actions/atlas.com/reactor/script/model.go` — `touchRules` field and `TouchRules()` accessor
- `services/atlas-reactor-actions/atlas.com/reactor/script/builder.go` — `touchRules` and `AddTouchRule`
- `services/atlas-reactor-actions/atlas.com/reactor/script/entity.go` — `TouchRules` on `jsonReactorScript`, both directions
- `services/atlas-reactor-actions/atlas.com/reactor/script/rest.go` — `TouchRules` on the REST model, both directions
- `services/atlas-reactor-actions/atlas.com/reactor/script/entity_test.go` — new cases
- `services/atlas-reactor-actions/atlas.com/reactor/script/rest_test.go` — new cases

Module root: `services/atlas-reactor-actions/atlas.com/reactor`.

No migration: rules live in the JSONB `data` column (`entity.go`'s
`Data string \`gorm:"column:data;type:jsonb"\``), so a third key is additive
(design §1.6).

### Steps

- [ ] **Step 1: Write the failing test**

Add to `entity_test.go`, following its existing case style.

`TestMakeTouchRules` — table-driven over `Entity.Data` JSON strings.

| case | `data` JSON | assertion |
|---|---|---|
| touchRules present | `{"reactorId":"6109013","hitRules":[],"actRules":[],"touchRules":[{"id":"r1","conditions":[],"operations":[]}]}` | `len(m.TouchRules()) == 1`, `m.TouchRules()[0].Id() == "r1"` |
| touchRules absent | `{"reactorId":"6109013","hitRules":[{"id":"h1","conditions":[],"operations":[]}],"actRules":[]}` | `len(m.TouchRules()) == 0`, `len(m.HitRules()) == 1` — no error |

`TestToEntityRoundTripsTouchRules` — build a `ReactorScript` with one touch rule
via `NewReactorScriptBuilder().AddTouchRule(...)`, call `ToEntity`, unmarshal
`Entity.Data`, and assert `touchRules` carries the rule.

Add to `rest_test.go` the equivalent for `Transform`/`Extract` (whichever
directions `rest.go` defines): a REST model carrying `touchRules` survives the
round trip, and one omitting it yields an empty slice.

- [ ] **Step 2: Extend the domain model and builder**

Add `touchRules []Rule` to `ReactorScript` and `TouchRules() []Rule`. Add
`touchRules` to `ReactorScriptBuilder` (initialised to `make([]Rule, 0)` in
`NewReactorScriptBuilder`), `AddTouchRule(rule Rule) *ReactorScriptBuilder`, and
the field in `Build()`.

- [ ] **Step 3: Extend the JSON entity**

Add `TouchRules []jsonRule \`json:"touchRules,omitempty"\`` to
`jsonReactorScript`. In `Make`, loop `data.TouchRules` through
`convertJsonRule` into `builder.AddTouchRule`. In `ToEntity`, build
`jsonTouchRules` the same way `jsonHitRules` is built and set it on the `data`
literal.

`omitempty` keeps every existing stored script's serialised form byte-identical.

- [ ] **Step 4: Extend the REST model**

Add `TouchRules []RestRuleModel \`json:"touchRules"\`` next to `HitRules`
(`rest.go:22-23`), populate it in the transform (`rest.go:100-116` is the
hit/act pattern) and consume it in the extract (`rest.go:158-170`).

### Verification

- `cd services/atlas-reactor-actions/atlas.com/reactor && go build ./... && go test ./script/...` passes.

---

## Task 15: `atlas-reactor-actions` — `TOUCH` consumer arm and `ProcessTouch`

Depends on Task 14.

### Files

- `services/atlas-reactor-actions/atlas.com/reactor/script/kafka.go` — `CommandTypeTouch`
- `services/atlas-reactor-actions/atlas.com/reactor/script/consumer.go` — the `case` arm and `handleTouchCommand`
- `services/atlas-reactor-actions/atlas.com/reactor/script/processor.go` — `ProcessTouch` on the interface and impl
- `services/atlas-reactor-actions/atlas.com/reactor/script/mock/processor.go` — `ProcessTouchFunc`
- `services/atlas-reactor-actions/atlas.com/reactor/script/processor_test.go` — new cases

Module root: `services/atlas-reactor-actions/atlas.com/reactor`.

Patterns to copy: `handleTriggerCommand` in `consumer.go`; `ProcessTrigger` in
`processor.go`.

### Steps

- [ ] **Step 1: Write the failing test**

Add to `processor_test.go`, following its existing DB harness.

`TestProcessTouch` — table-driven.

| case | stored script | assertion |
|---|---|---|
| `touchRules wins` | `touchRules` has rule `t1` (no conditions), `hitRules` has rule `h1` (no conditions) | `result.MatchedRule == "t1"` |
| `falls back to hitRules` | `touchRules` empty/absent, `hitRules` has `h1` (no conditions) | `result.MatchedRule == "h1"` |
| `no rules at all` | `hitRules` and `touchRules` both empty | `result.MatchedRule == "no_match"`, `result.Error == nil` |
| `no script` | nothing stored for the reactor id | `result.MatchedRule == "no_script"`, `result.Error == nil` |

An empty-conditions rule matches unconditionally — `evaluateRules` returns on
the first rule whose conditions all evaluate true, and a rule with none is
vacuously true.

`TestHandleCommandFuncRoutesTouch` — if `consumer.go` has an existing dispatch
test, add a case asserting `Type: "TOUCH"` reaches `ProcessTouch` and an unknown
type still warns and is ignored (design §1.6). If it has none, assert the
routing through the processor mock.

- [ ] **Step 2: Add the command type**

In `script/kafka.go`:

```go
CommandTypeTouch = "TOUCH"
```

- [ ] **Step 3: Add `ProcessTouch`**

On the `Processor` interface, after `ProcessTrigger`:

```go
ProcessTouch(reactorId string, reactorState int8, characterId uint32) ProcessResult
```

The implementation is `ProcessTrigger` with the rule selection changed:

```go
rules := script.TouchRules()
if len(rules) == 0 {
    p.l.Debugf("Reactor script [%s] declares no touchRules; falling back to hitRules.", reactorId)
    rules = script.HitRules()
}
return p.evaluateRules(rules, reactorId, reactorState, characterId, "touch")
```

The fallback is design §7's deliberate call: none of the ten `activateByTouch`
templates has a script yet and authoring them is a PRD non-goal, so without it
the mechanism ships observably inert. Distinguishability is preserved by the
command type, the `"touch"` event label in the log line, and the fact that a
script declaring `touchRules` never reaches the hit path.

Add `ProcessTouchFunc` to `script/mock/processor.go`.

- [ ] **Step 4: Route the command**

In `handleCommandFunc`'s switch, after the `CommandTypeTrigger` case:

```go
case CommandTypeTouch:
    handleTouchCommand(l, ctx, db, command)
```

`handleTouchCommand` is `handleTriggerCommand` with `ProcessTouch` substituted
and the log strings changed from "trigger" to "touch". The `default:
l.Warnf("Unknown command type: %s", ...)` arm stays unchanged.

### Verification

- `cd services/atlas-reactor-actions/atlas.com/reactor && go build ./... && go test ./...` passes.

---

## Task 16: Documentation

### Files

- `docs/TODO.md` — remove the deferred item at line 280
- `docs/tasks/task-249-touch-activated-reactors/touch-templates.md` — **new file**; the authoritative template list

### Steps

- [ ] **Step 1: Record the authoritative template list**

`touch-templates.md` records the ten `activateByTouch` templates measured from
the Cosmic `Reactor.wz` set (design §1.5):

```
2406000  6109013  6109014  6109021  6109022  6109023
6109024  6109025  6109026  6109027
```

State explicitly: the nine-item lists in `docs/TODO.md` and
`docs/tasks/task-019-reactor-type-semantics/prd.md:32` undercount by omitting
`2406000` (나인스피릿의둥지, a Horntail prequest reactor rather than a GPQ one);
the ten-item list in `docs/research/missing-features/maps-portals-reactors.md:35`
is correct. Note that the count is per-mounted-WZ — other sets on this machine
carry 9, 10 and 19 — so this list is authoritative for the Cosmic/v83-era data
Atlas reads, not universally.

Confirm the list before writing it:

```sh
grep -l 'activateByTouch' <WZ_ROOT>/Reactor.wz/*.img.xml
```

Also record that **none** of the ten contains a type-100 event, which is why
`TL`/`BR` are zero for all of them and why Task 6 adds `touchAreaInfo` instead
of reusing them.

- [ ] **Step 2: Remove the TODO entry**

Delete the `- [ ] Implement \`activateByTouch\` reactor activation…` bullet under
`### Reactors Service` in `docs/TODO.md` (line 280 at plan time — locate it with
`grep -n activateByTouch docs/TODO.md` rather than by line number).

### Verification

- `grep -n activateByTouch docs/TODO.md` returns nothing.

---

## Final gate

After Task 16:

- [ ] `tools/verify.sh` (flagless) exits 0.
- [ ] `go run ./tools/packet-audit matrix --check` exits 0.
- [ ] Code review (`superpowers:requesting-code-review`) completed before the PR
      is opened — `backend-guidelines-reviewer` over the five changed Go
      services, `plan-adherence-reviewer` over this plan, and
      `packet-completeness-critic` for the packet lane.
