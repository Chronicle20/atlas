# Packet-Audit Closeout — Four-Version Baseline — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close out every actionable deferral from the four-version (GMS v83/v87/v95, JMS v185) packet audit so the baseline has zero open actionable deferrals — fixing the real wire bugs with byte-level tests + IDA-verified version gates, resolving the IDA verification spikes to verdicts, routing JMS cash-shop purchases through the existing wallet, enhancing the `packet-audit` analyzer to suppress the known false-positive classes, and curating the docs/ledger into a trusted, reusable baseline.

**Architecture:** Layered phases (design §5). **Phase A** lands analyzer enhancements first so every later re-run reads a clean signal. **Phase B** fixes the real wire bugs (B1) and atlas-channel handler logic (B2). **Phase C** wires the JMS cash-shop bodies + template + verifies routing. **Phase D** runs the IDA verification spikes (B3/B4/B6) to verdicts, fixing in-task. **Phase E** regenerates SUMMARY/TOTAL and curates `_pending.md` to zero actionable items. **Phase F** runs the full CLAUDE.md verify gates + code review. Every wire change follows the cross-cutting mechanics (design §3): tenant-context version gates (`t.Region()=="GMS" && t.MajorVersion()>=N`), a **region-dispatched body** for >2-version divergences (never a 3rd nested guard), and a **byte-level wire-shape test as the oracle** (no change ships on analyzer verdict alone).

**Tech Stack:** Go 1.24 (multi-module `go.work`); `libs/atlas-packet` (encoders/decoders + `test` harness with `pt.CreateContext`/`pt.Variants`/`pt.RoundTrip`); `libs/atlas-tenant` (`tenant.MustFromContext`); Kafka via `libs/atlas-kafka`; `tools/packet-audit` (Go AST static analyzer); IDA-MCP for live binary verification (four IDBs: GMS v83/v87/v95, JMS v185); `docker buildx bake` for service image verification.

---

## Conventions used throughout this plan

- **Worktree root** (all paths relative to it): `<repo-root>/.worktrees/task-080-packet-audit-closeout`. Every subagent prompt MUST `cd` into this worktree first and verify `git branch --show-current` returns `task-080-packet-audit-closeout` after each commit.
- **Version gate idiom** (design §3.1): compute a local in the outer function (before the returned closure) from `t := tenant.MustFromContext(ctx)`, e.g. `v95Plus := t.Region() == "GMS" && t.MajorVersion() >= 95`. Apply the same condition symmetrically in Encode and Decode. Exemplar in tree: `libs/atlas-packet/stat/clientbound/changed.go`.
- **Region-dispatched body idiom** (design §3.2): for a divergence spanning >2 versions, dispatch at the top of the encode/decode closure to a per-region helper method (`m.decodeJMS(r)` / `m.decodeGMS(t, r)`); a GMS body may itself carry ≤2 guards. Never stack a 3rd nested `if`. The repo nesting `awk` must stay clean.
- **Byte-test idiom** (design §3.3): tests live beside the packet (`*_test.go`), use the model's own constructor/Builder (no `*_testhelpers.go`). Round-trip via `pt.RoundTrip(t, ctx, in.Encode, out.Decode, opts)`; exact-shape via `in.Encode(l, pt.CreateContext("GMS",95,1))(opts)` then `len(...)` + `bytes.Equal(got[a:b], []byte{...})`. `pt.Variants` = GMS v28/v83/v87/v95 + JMS v185.
- **IDA-evidence capture** (design §3.4): every fix and every resolved spike records, in the per-packet audit `.md` and/or the registry, the IDA `FName@address` and the read-order it was verified against.
- **Commit cadence:** one commit per task (a logical fix + its tests). Commit messages: `fix(<domain>): <what> (task-080 B<n>)` for fixes, `feat(packet-audit): <what> (task-080 §4.7)` for analyzer, `docs(packets): <what> (task-080 §4.8)` for ledger.
- **Per-module verify** after each code task: `go test -race ./...` and `go vet ./...` in every changed module; full `docker buildx bake` is batched into Phase F.

---

# PHASE A — Analyzer enhancements (`tools/packet-audit`)

Land these first (design §5 phase A). Each false-positive class gets a `.go.txt` fixture under `tools/packet-audit/internal/<pkg>/testdata/` plus a unit test asserting the fixture now produces ✅ instead of ❌/🔍. Module: `github.com/Chronicle20/atlas/tools/packet-audit`. Run tool tests with `GOWORK=off` only if a workspace conflict appears; default `go test ./...` from `tools/packet-audit`.

Current-state facts (verified):
- `internal/atlaspacket/analyzer.go` — guard stack (`callCtx.stack`, `callCtx.suffixGuards`, `conjoin()` ~237–405), `blockTerminatesWithReturn` (~335–360), suffix-taint (~395–405), `KindRepeat` for `RangeStmt`/`ForStmt` (~433–465). Early-return modeling **already exists and is tested** (`TestEarlyReturnThenTaintsSuffix`).
- `internal/diff/diff.go` — `primWidth` (~80–96), `idaWidth` (~98–114), width comparison at the `case primWidth(...) != idaWidth(...)` arm (~69), `FlattenWithRegistry`/`flattenWithRegistryGuarded` (~128–164). `Verdict` enum + `Symbol()` (~10–21).
- `internal/atlaspacket/registry.go` — Pass-2 method scan (~109–180) registers `Encode`/`EncodeEntry`/`EncodeBytes`/`EncodeForeign`/`Write`. **Gap:** a struct type with none of those methods is never pre-analyzed, so a `KindRecurse` into it falls through `diff.go` and surfaces as `VerdictDeferred` (🔍).
- `cmd/run.go` — `locateAtlasFile(root, name, pkg, dir)` (~1709–1744) **already takes a `pkg` disambiguation param**; `candidatesFromFName` (~192–1644) is a 359-case switch assigning `pkg`; `qualifiedWriterName(pkg,name)` (~134–139); `writeSummary` (~241–249).
- Verdict rendering: `internal/report/report.go` `renderMarkdown` (~42–70).
- Test pattern: `.go.txt` fixtures in `internal/atlaspacket/testdata/`; tests call `AnalyzeFile("testdata/<f>.go.txt", "<Type>", "Encode")` and assert on `[]Call`. Diff tests build `[]atlaspacket.Call` + `idasrc.Fields` literals and assert `Verdict`.

---

### Task A0: Baseline the analyzer — capture the current false-positive set

**Files:**
- Create: `docs/tasks/task-080-packet-audit-closeout/analyzer-baseline.md` (working note, not committed to the ledger)

- [ ] **Step 1: Build the tool**

Run: `cd tools/packet-audit && go build ./...`
Expected: clean build.

- [ ] **Step 2: Run the four-version audit against the checked-in IDA exports (no live IDA needed for a baseline)**

Run (from worktree root), for each version json under `docs/packets/ida-exports/`:
```bash
cd tools/packet-audit
go run . \
  -csv-clientbound "../../docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "../../docs/packets/MapleStory Ops - ServerBound.csv" \
  -atlas-packet ../../libs/atlas-packet \
  -ida-source ../../docs/packets/ida-exports/gms_v95.json \
  -output /tmp/audit-baseline/gms_v95
```
(Repeat for `gms_v83.json`, `gms_v87.json`, `gms_jms_185.json` → `/tmp/audit-baseline/<ver>`.)
Expected: SUMMARY.md + per-packet .md written under each `/tmp/audit-baseline/<ver>`.

- [ ] **Step 3: Record the current ❌/🔍 inventory**

Run: `grep -rE '\| (❌|🔍) \|' /tmp/audit-baseline/*/SUMMARY.md | sort > docs/tasks/task-080-packet-audit-closeout/analyzer-baseline.md`
This is the "before" set. After each Phase-A task, re-run and diff against this file to confirm the targeted entries flipped to ✅ and nothing regressed. Cross-reference each entry to the PRD §4.7 named classes; entries NOT in the named classes stay as-is (they are real findings or genuine residue).

- [ ] **Step 4: Commit the baseline note**

```bash
git add docs/tasks/task-080-packet-audit-closeout/analyzer-baseline.md
git commit -m "chore(packet-audit): capture analyzer false-positive baseline (task-080 §4.7)"
```

---

### Task A1: Opaque-buffer / width-label equivalence (`internal/diff/diff.go`)

**Files:**
- Modify: `tools/packet-audit/internal/diff/diff.go` (width comparison ~69, `primWidth`/`idaWidth` ~80–114)
- Test: `tools/packet-audit/internal/diff/diff_test.go`

Goal: stop flagging width mismatches where Atlas writes an opaque/composite buffer that is byte-equal to the IDA read. Cases from PRD §4.7: `WriteByteArray(N) ≡ DecodeBuf(N)`, `WriteLong ≡ EncodeBuffer(8)` / 8-byte buf, `WriteInt16+WriteShort(0) ≡ Decode4`, `WriteInt64 point ≡ EncodeBuffer(&pt,8)`.

- [ ] **Step 1: Write the failing test**

Add to `diff_test.go`:
```go
func TestDiffOpaqueBufferWidthEquivalence(t *testing.T) {
	// Atlas writes an 8-byte fixed buffer (WriteLong); IDA reads EncodeBuffer(8).
	atlas := []atlaspacket.Call{{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode8}}
	ida := idasrc.Fields{Calls: []idasrc.FieldCall{{Op: idasrc.DecodeBuf, BufLen: 8}}}
	rows := Diff(atlas, ida)
	if len(rows) != 1 || rows[0].Verdict != VerdictMatch {
		t.Fatalf("expected match for 8-byte buf == Encode8; got %+v", rows)
	}
}
```
(If `idasrc.FieldCall` has no `BufLen` field, the test must first be adjusted to whatever field carries the opaque length — read `internal/idasrc/idasrc.go` and use the real field name. Keep the assertion semantics identical.)

- [ ] **Step 2: Run it — expect FAIL**

Run: `cd tools/packet-audit && go test ./internal/diff/ -run TestDiffOpaqueBufferWidthEquivalence -v`
Expected: FAIL — current code reports a width mismatch (Encode8=8 vs DecodeBuf=-2).

- [ ] **Step 3: Implement width equivalence**

Replace the bare `primWidth(...) != idaWidth(...)` comparison with a helper `widthEquivalent(atlasOp, idaCall)` that returns true when the byte footprints provably match even though the labels differ. Add to `diff.go`:
```go
// widthEquivalent reports whether an Atlas write and an IDA read occupy the
// same number of wire bytes even when their op-labels differ — the opaque-buffer
// / width-label equivalence class (task-080 §4.7). A fixed-width Atlas primitive
// (1/2/4/8) matches an IDA DecodeBuf of the same byte length, and vice-versa.
func widthEquivalent(a atlaspacket.Primitive, ida idasrc.FieldCall) bool {
	aw := primWidth(a)
	iw := idaWidth(ida.Op)
	if aw == iw {
		return true
	}
	// Atlas fixed-width vs IDA opaque buffer of the same declared length.
	if aw > 0 && ida.Op == idasrc.DecodeBuf && ida.BufLen == aw {
		return true
	}
	return false
}
```
Then change the comparison arm from `case primWidth(atlas[i].Op) != idaWidth(ida.Calls[i].Op):` to `case !widthEquivalent(atlas[i].Op, ida.Calls[i]):`. (The Atlas-opaque-buffer-vs-IDA-fixed direction is handled by the composite-run rule in Step 5; Atlas-side buffer lengths live on the `Call`, not the `Primitive`.)

- [ ] **Step 4: Run it — expect PASS**

Run: `cd tools/packet-audit && go test ./internal/diff/ -run TestDiffOpaqueBufferWidthEquivalence -v`
Expected: PASS.

- [ ] **Step 5: Add the composite-run rule (WriteInt16+WriteShort(0) ≡ Decode4) with a failing test first**

Add test:
```go
func TestDiffCompositeRunEqualsWiderRead(t *testing.T) {
	// Atlas writes Encode2 + Encode2 (a 4-byte composite); IDA reads Decode4.
	atlas := []atlaspacket.Call{
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode2},
		{Kind: atlaspacket.KindWrite, Op: atlaspacket.Encode2},
	}
	ida := idasrc.Fields{Calls: []idasrc.FieldCall{{Op: idasrc.Decode4}}}
	rows := Diff(atlas, ida)
	for _, r := range rows {
		if r.Verdict == VerdictBlocker {
			t.Fatalf("composite 2+2 should equal Decode4, got blocker: %+v", rows)
		}
	}
}
```
Run it — expect FAIL. Then implement a coalescing pre-pass in `Diff` that merges adjacent fixed-width Atlas writes whose summed width equals the next IDA read width before the per-index compare. Keep the pre-pass conservative: only coalesce when the running Atlas sum exactly hits an IDA fixed width and the IDA side has fewer calls at that position. Re-run — expect PASS. Run the full diff suite: `go test ./internal/diff/ -v` — all PASS.

- [ ] **Step 6: Re-run the four-version audit; confirm the named width entries flipped**

Re-run Task A0 Step 2 into `/tmp/audit-A1/<ver>`; diff `SUMMARY.md` against `analyzer-baseline.md`. Confirm messenger AvatarLook, note `Display`, guild BBS FILETIME, fame `GiveResponse`, socket `Hello`/`ChannelConnect`, stat `Changed`, omok `MoveStone` (PRD §4.7 width list) flipped ✅ where they were width-mismatch artifacts. Note any that did NOT flip for the registry (§4.8).

- [ ] **Step 7: Commit**

```bash
git add tools/packet-audit/internal/diff/
git commit -m "feat(packet-audit): width-label equivalence for opaque buffers (task-080 §4.7)"
```

---

### Task A2: Qualified struct-name tracking — complete `candidatesFromFName` collisions

**Files:**
- Modify: `tools/packet-audit/cmd/run.go` (`candidatesFromFName` ~192–1644)
- Test: `tools/packet-audit/cmd/run_test.go` (or the existing `*_test.go` beside run.go)

Goal: the disambiguation machinery already exists (`locateAtlasFile` takes `pkg`; `qualifiedWriterName` prefixes the report). Gap is map completeness for the named same-name collisions: `ChannelChange` (buddy vs channel) and `Spawn`/`Destroy`/`Movement` across monster/drop/reactor/pet.

- [ ] **Step 1: Write a failing test pinning the collision routing**

Add to the cmd test file:
```go
func TestCandidatesQualifyCollidingNames(t *testing.T) {
	cases := []struct {
		fname    string
		wantPkg  string
		wantName string
	}{
		{"CMobPool::OnMobEnterField", "monster", "Spawn"},
		{"CDropPool::OnDropEnterField", "drop", "Spawn"},
		// add the reactor/pet Spawn/Destroy/Movement + buddy/channel ChannelChange
		// rows using the REAL IDA FNames found in the four version jsons.
	}
	for _, c := range cases {
		cands := candidatesFromFName(c.fname)
		if len(cands) == 0 {
			t.Fatalf("%s: no candidates", c.fname)
		}
		if cands[0].pkg != c.wantPkg || cands[0].name != c.wantName {
			t.Errorf("%s: got pkg=%q name=%q, want pkg=%q name=%q",
				c.fname, cands[0].pkg, cands[0].name, c.wantPkg, c.wantName)
		}
	}
}
```
Before writing the test rows, grep the four `docs/packets/ida-exports/*.json` for the actual FNames that resolve to `Spawn`/`Destroy`/`Movement`/`ChannelChange` so the test uses real inputs, not invented ones.

- [ ] **Step 2: Run it — expect FAIL** for any FName whose case is missing or lacks a `pkg`.

Run: `cd tools/packet-audit && go test ./cmd/ -run TestCandidatesQualifyCollidingNames -v`

- [ ] **Step 3: Add/repair the switch cases**

For each colliding FName, add or fix the `candidatesFromFName` case to return `[]candidate{{name: "<Name>", pkg: "<subdomain>", dir: csvpkg.Dir<...>}}`. Match the existing case style (see the `CShopDlg::SendRechargeRequest` exemplar that already sets `pkg: "npc"`).

- [ ] **Step 4: Run it — expect PASS**, then `go test ./cmd/ -v` — all PASS.

- [ ] **Step 5: Re-run the four-version audit; confirm no same-name file misroutes**

Re-run into `/tmp/audit-A2/<ver>`; confirm the previously-misrouted `Spawn`/`Destroy`/`Movement`/`ChannelChange` reports now point at the correct `libs/atlas-packet/<subdomain>/<dir>/` file (check the `**Atlas file:**` line in the per-packet `.md`).

- [ ] **Step 6: Commit**

```bash
git add tools/packet-audit/cmd/
git commit -m "feat(packet-audit): qualify colliding struct names in candidatesFromFName (task-080 §4.7)"
```

---

### Task A3: Sub-struct / loop descent for self-describing types (`registry.go` + `diff.go`)

**Files:**
- Modify: `tools/packet-audit/internal/atlaspacket/registry.go` (Pass-2 ~109–180)
- Modify: `tools/packet-audit/internal/diff/diff.go` (`flattenWithRegistryGuarded` ~128–164)
- Test: `tools/packet-audit/internal/atlaspacket/registry_test.go`, `tools/packet-audit/internal/atlaspacket/testdata/`

Goal (bounded, design §2/Q4): generalize descent so a field whose type **has an Encode/Write method OR decomposes into known primitives** is inlined; a type with neither is the explicit register boundary (→ §4.8 registry, not chased). Covers PRD §4.7 sub-struct list (party `WritePartyData`, npc `Action`/`ShopList`, character `CharacterInfo`/`CharacterSkillChange`/`AddCharacterEntry`/`CharacterViewAllCharacters`, inventory `Add`/`ChangeBatch`, storage `UpdateAssets`, pet bodies, `model.Asset`/`GW_ItemSlotBase`).

- [ ] **Step 1: Create a fixture that exercises a struct field with no Encode method but a flat layout**

Create `tools/packet-audit/internal/atlaspacket/testdata/substruct_no_encode.go.txt` modeling a top-level `Encode` that writes a sub-struct via a field whose type only has primitive writes inlined (mirroring `model.Asset`). Use the existing `.go.txt` fixtures as the template for the fake `response.Writer` shim.

- [ ] **Step 2: Write the failing test**

Add to `registry_test.go`:
```go
func TestRegistryDescendsDecomposableType(t *testing.T) {
	reg := buildTestRegistry(t, "testdata/substruct_no_encode.go.txt")
	calls, ok := reg.Calls("fixture.Outer")
	if !ok {
		t.Fatal("Outer not registered")
	}
	// The sub-struct's two primitive writes must be inlined, not left as a
	// single unresolved KindRecurse.
	flat := diff.FlattenWithRegistry(calls, testCtx(), reg)
	if got := countOp(flat, atlaspacket.Encode4); got != 2 {
		t.Fatalf("expected 2 inlined Encode4 from sub-struct, got %d", got)
	}
}
```
(Use the registry-construction helper the existing registry tests already use; if none exists, add a minimal `buildTestRegistry` mirroring how `registry.New` is called in production.)

- [ ] **Step 3: Run it — expect FAIL** (type has no Encode → not pre-analyzed → KindRecurse passes through).

- [ ] **Step 4: Implement the fallback descent**

In `registry.go` Pass-2, after the method scan, add a Pass-3 that, for any registered `TypeEntry` whose `Calls == nil`, attempts to synthesize `Calls` by walking the struct's fields and emitting the primitive/`KindRecurse` calls for each (a struct literal decomposition). Only synthesize when **every** field resolves to a known primitive or an already-registered type; otherwise leave `Calls == nil` (the register boundary). Add a `TypeEntry.Opaque bool` flag set true when synthesis fails, so `diff.go` can emit a stable deferred row that the registry curation (§4.8) keys on.

- [ ] **Step 5: Run it — expect PASS**, then `go test ./internal/... -v` — all PASS.

- [ ] **Step 6: Re-run the four-version audit; confirm sub-struct entries flip**

Re-run into `/tmp/audit-A3/<ver>`; confirm the PRD §4.7 sub-struct list flips ✅ where the type is self-describing. Record every type that stayed `Opaque` for the §4.8 registry.

- [ ] **Step 7: Commit**

```bash
git add tools/packet-audit/internal/atlaspacket/ tools/packet-audit/internal/diff/
git commit -m "feat(packet-audit): descend self-describing sub-structs, flag opaque residue (task-080 §4.7)"
```

---

### Task A4: Early-return / exclusive-branch — verify coverage, extend only if gaps remain

**Files:**
- Modify (only if a gap is found): `tools/packet-audit/internal/atlaspacket/analyzer.go` (~237–405)
- Test: `tools/packet-audit/internal/atlaspacket/analyzer_test.go`, `testdata/`

Early-return modeling already exists (`blockTerminatesWithReturn`, suffix-taint, `TestEarlyReturnThenTaintsSuffix`). This task confirms the PRD §4.7 early-return list is actually covered and fills only real gaps.

- [ ] **Step 1: Enumerate the named early-return packets**

For each PRD §4.7 early-return entry (login `CharacterList`, character `CharacterSitResult`, monster/drop/reactor `Spawn`/`ReactorHitRequest`, cash `IncreaseInventory`/`IncreaseStorage`), find its current verdict in `/tmp/audit-A3/<ver>` SUMMARY. Any still ❌/🔍 due to over-counted conditional bytes is a gap.

- [ ] **Step 2: For each gap, add a `.go.txt` fixture reproducing the over-count and a failing test**

Mirror the structure of the real packet's guarded-return. Assert the expected `[]Call` guard set (the surviving-branch guard must taint the suffix). Run — expect FAIL.

- [ ] **Step 3: Extend the suffix-taint / guard-stack logic minimally to cover the gap**

Only touch `analyzer.go` if a fixture genuinely fails. Re-run — expect PASS. Run `go test ./internal/atlaspacket/ -v` — all PASS.

- [ ] **Step 4: Re-run the four-version audit (final Phase-A run)**

Re-run into `/tmp/audit-A4/<ver>`. The named early-return entries should be ✅. Capture the final post-Phase-A false-positive delta in `analyzer-baseline.md` (append a "post-Phase-A" section). Anything still ❌/🔍 is either a real bug (→ Phase B/D) or genuine residue (→ §4.8 registry).

- [ ] **Step 5: Commit** (even if only the baseline note changed)

```bash
git add tools/packet-audit/ docs/tasks/task-080-packet-audit-closeout/analyzer-baseline.md
git commit -m "feat(packet-audit): confirm/close early-return modeling gaps (task-080 §4.7)"
```

---

# PHASE B — Real wire bugs (B1) + atlas-channel handler fixes (B2)

B1.1 first (only multi-service change). Modules touched: `libs/atlas-packet`, `atlas-maps`, `atlas-channel`.

---

### Task B1.1a: Extend the mist `CreatedBody` event contract (atlas-maps)

**Files:**
- Modify: `services/atlas-maps/atlas.com/maps/kafka/message/mist/kafka.go` (`CreatedBody` struct)
- Modify: `services/atlas-maps/atlas.com/maps/mist/producer.go` (`createdEventProvider`)
- Modify: `services/atlas-maps/atlas.com/maps/mist/model.go` (add `mistType` field + getter + builder setter)
- Test: `services/atlas-maps/atlas.com/maps/mist/producer_test.go` (create if absent)

The model already carries `SourceSkillId()`, `SourceSkillLevel()`. `CreatedBody` drops them and has no `nType`.

- [ ] **Step 1: Write a failing producer test**

Create `producer_test.go` asserting `createdEventProvider` copies skill id/level/type onto the event body:
```go
func TestCreatedEventCarriesSkillAndType(t *testing.T) {
	tn, _ := tenant.Create(uuid.New(), "GMS", 83, 1)
	f := field.NewBuilder(0, 0, 100000000).Build()
	m := NewBuilder(uuid.New(), f).
		SetOwner("MONSTER", 42).
		SetSource(2121006, 20).     // skill id + level
		SetType(1).                 // NEW: mist/affected-area type
		SetOrigin(100, 200).
		SetBounds(-50, -30, 50, 30).
		SetDuration(10 * time.Second).
		Build()
	msgs, err := createdEventProvider(tn, m)()
	if err != nil {
		t.Fatal(err)
	}
	var ev mistKafka.Event[mistKafka.CreatedBody]
	if err := json.Unmarshal(msgs[0].Value, &ev); err != nil {
		t.Fatal(err)
	}
	if ev.Body.SourceSkillId != 2121006 || ev.Body.SourceSkillLevel != 20 || ev.Body.Type != 1 {
		t.Fatalf("body missing skill/type: %+v", ev.Body)
	}
}
```

- [ ] **Step 2: Run it — expect FAIL** (compile error: `SetType`/`Type`/`SourceSkillId` not present).

Run: `cd services/atlas-maps/atlas.com/maps && go test ./mist/ -run TestCreatedEventCarriesSkillAndType -v`

- [ ] **Step 3: Add the fields to `CreatedBody`**

In `kafka/message/mist/kafka.go`, extend `CreatedBody`:
```go
type CreatedBody struct {
	OwnerType        string `json:"ownerType"`
	OwnerId          uint32 `json:"ownerId"`
	SourceSkillId    uint32 `json:"sourceSkillId"`
	SourceSkillLevel uint32 `json:"sourceSkillLevel"`
	Type             int32  `json:"type"`
	OriginX          int16  `json:"originX"`
	OriginY          int16  `json:"originY"`
	LtX              int16  `json:"ltX"`
	LtY              int16  `json:"ltY"`
	RbX              int16  `json:"rbX"`
	RbY              int16  `json:"rbY"`
	Duration         int64  `json:"duration"`
}
```

- [ ] **Step 4: Add `type` to the model + builder**

In `mist/model.go` add a `mistType int32` field, a `Type() int32` getter, a `SetType(int32) *Builder` setter, and thread it through `Build()`. `SetType` defaults to 0 if unset (the create-path source is decided in Task B1.1b).

- [ ] **Step 5: Populate the event in `createdEventProvider`**

```go
Body: mistKafka.CreatedBody{
	OwnerType:        m.OwnerType(),
	OwnerId:          m.OwnerId(),
	SourceSkillId:    m.SourceSkillId(),
	SourceSkillLevel: m.SourceSkillLevel(),
	Type:             m.Type(),
	OriginX:          m.OriginX(),
	OriginY:          m.OriginY(),
	LtX:              m.LtX(),
	LtY:              m.LtY(),
	RbX:              m.RbX(),
	RbY:              m.RbY(),
	Duration:         int64(m.Duration() / time.Millisecond),
},
```

- [ ] **Step 6: Run it — expect PASS**, then `go test -race ./...` + `go vet ./...` in atlas-maps — clean.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-maps/
git commit -m "feat(maps): carry skill id/level + type on MIST_CREATED event (task-080 B1.1)"
```

---

### Task B1.1b: Determine the mist `Type` (nType) source — confined IDA + creation-path spike

**Files:**
- Modify (per finding): `services/atlas-maps/atlas.com/maps/mist/processor.go` and/or the create-command consumer
- Note: `docs/tasks/task-080-packet-audit-closeout/spike-affectedarea.md`

- [ ] **Step 1: Read the atlas-maps mist creation path**

Read `mist/processor.go` (the create handler) and `kafka/message/mist/kafka.go` `CreateCommandBody`. Determine whether a type/disease distinction is available at creation (the model has `disease`, `sourceSkillId`).

- [ ] **Step 2: Read all four IDBs' `CAffectedAreaPool::OnAffectedAreaCreated`**

Via IDA-MCP (`mcp__ida-pro__decompile_function`) at v83@0x431a63, v87@0x432f3f, v95@0x437ec0, JMS185@0x436572. Identify what `nType` selects on the client (mist vs other affected-area kinds) and confirm the full field read-order (this also feeds B1.1c, including the exact `tStart`/`tEnd` positions).

- [ ] **Step 3: Record the verdict**

Write `spike-affectedarea.md` with the four FName@address read-orders and the decision: `Type` is a fixed constant for skill-driven mist (set it in the create path) **or** derived from the skill/disease. Wire `SetType(...)` accordingly in `mist/processor.go`.

- [ ] **Step 4: Verify + commit**

`go test -race ./... && go vet ./...` in atlas-maps.
```bash
git add services/atlas-maps/ docs/tasks/task-080-packet-audit-closeout/spike-affectedarea.md
git commit -m "fix(maps): set mist nType from creation path per IDA (task-080 B1.1)"
```

---

### Task B1.1c: Rewrite `AffectedAreaCreated` to the client RECT-buffer layout

**Files:**
- Modify: `libs/atlas-packet/field/clientbound/affected_area_created.go`
- Test: `libs/atlas-packet/field/clientbound/affected_area_test.go`

Target layout (PRD B1.1, confirm exact field order + tStart position via the B1.1b spike read-order): `dwId(int4), nType(int4), dwOwnerId(int4), nSkillID(int4), nSLV(byte), phase(int16), rcArea(16-byte RECT = LT.x,LT.y,RB.x,RB.y as 4×int32 absolute), tEnd(int4)`, with `tStart(int4)` added (gated `GMS && MajorVersion>=95`) in the position the IDA read-order dictates. Absolute RECT = origin + offset.

- [ ] **Step 1: Write the failing byte-shape test**

Replace the existing round-trip-only test with version byte-shape assertions:
```go
func TestAffectedAreaCreatedWireShape(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	id := uuid.New()
	// origin (100,200), offsets lt(-50,-30) rb(50,30) → abs LT(50,170) RB(150,230)
	in := NewAffectedAreaCreated(id, /*ownerId*/ 42, /*nType*/ 1, /*skillId*/ 2121006,
		/*skillLevel*/ 20, /*phase*/ 0, /*originX*/ 100, /*originY*/ 200,
		/*ltX*/ -50, /*ltY*/ -30, /*rbX*/ 50, /*rbY*/ 30, /*tStart*/ 0, /*tEnd*/ 10000)

	// v83/v87/JMS185: 4+4+4+4+1+2+16+4 = 39 bytes (no tStart).
	for _, v := range []pt.TenantVariant{
		{Name: "GMS v83", Region: "GMS", MajorVersion: 83, MinorVersion: 1},
		{Name: "GMS v87", Region: "GMS", MajorVersion: 87, MinorVersion: 1},
		{Name: "JMS v185", Region: "JMS", MajorVersion: 185, MinorVersion: 1},
	} {
		b := in.Encode(l, pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion))(nil)
		if len(b) != 39 {
			t.Errorf("%s: got %d bytes, want 39: % x", v.Name, len(b), b)
		}
	}
	// v95: +4 for tStart = 43 bytes.
	b95 := in.Encode(l, pt.CreateContext("GMS", 95, 1))(nil)
	if len(b95) != 43 {
		t.Errorf("v95: got %d bytes, want 43: % x", len(b95), b95)
	}
}
```
(Adjust the expected lengths to the exact field order confirmed in B1.1b. The point is per-version byte-count + key-field assertions, matching `TestStatChangedV95WireWidths`.)

- [ ] **Step 2: Run it — expect FAIL** (compile error: new constructor signature).

- [ ] **Step 3: Rewrite the struct, constructor, and Encode**

Replace the struct fields with `mistId uuid.UUID, ownerId uint32, nType int32, skillId int32, skillLevel byte, phase int16, originX/originY/ltX/ltY/rbX/rbY int16, tStart, tEnd int32`. Keep `mistKey(id)` for `dwId`. Encode (compute absolute RECT, gate tStart):
```go
func (m AffectedAreaCreated) Encode(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	v95Plus := t.Region() == "GMS" && t.MajorVersion() >= 95
	return func(_ map[string]interface{}) []byte {
		w.WriteInt(mistKey(m.mistId))
		w.WriteInt32(m.nType)
		w.WriteInt(m.ownerId)
		w.WriteInt32(m.skillId)
		w.WriteByte(m.skillLevel)
		w.WriteInt16(m.phase)
		// rcArea — absolute RECT (origin + offset), 4×int32.
		w.WriteInt32(int32(m.originX + m.ltX))
		w.WriteInt32(int32(m.originY + m.ltY))
		w.WriteInt32(int32(m.originX + m.rbX))
		w.WriteInt32(int32(m.originY + m.rbY))
		if v95Plus {
			w.WriteInt32(m.tStart) // position per IDA read-order (B1.1b)
		}
		w.WriteInt32(m.tEnd)
		return w.Bytes()
	}
}
```
Drop the invented `originX/originY` from the wire (they remain constructor inputs to compute the RECT). Update `String()` and getters.

- [ ] **Step 4: Run it — expect PASS**; `go test -race ./field/... && go vet ./...` in libs/atlas-packet — clean.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/field/clientbound/affected_area_created.go libs/atlas-packet/field/clientbound/affected_area_test.go
git commit -m "fix(packet): AffectedAreaCreated client RECT-buffer layout + tStart gate (task-080 B1.1)"
```

---

### Task B1.1d: Wire the channel consumer to the new packet + event fields

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer.go`
- Modify: `services/atlas-channel/atlas.com/channel/kafka/message/mist/kafka.go` (mirror the new `CreatedBody` fields)
- Test: `services/atlas-channel/atlas.com/channel/kafka/consumer/mist/consumer_test.go` (create; use the `affectedAreaCreatedBroadcaster` swap seam)

- [ ] **Step 1: Mirror the new event fields on the channel-side `CreatedBody`**

In atlas-channel's `kafka/message/mist/kafka.go`, add `SourceSkillId uint32`, `SourceSkillLevel uint32`, `Type int32` to `CreatedBody` (same json tags as atlas-maps).

- [ ] **Step 2: Write a failing test using the broadcaster seam**

The consumer already exposes `affectedAreaCreatedBroadcaster` as a swappable package var. Write a test that swaps in a recorder, feeds a `CreatedBody` with skill id/level/type, and asserts the constructed packet carries them (not 0):
```go
func TestHandleMistCreatedPassesSkillAndType(t *testing.T) {
	var got fieldpkt.AffectedAreaCreated
	orig := affectedAreaCreatedBroadcaster
	affectedAreaCreatedBroadcaster = func(_ logrus.FieldLogger, _ context.Context, _ writer.Producer, _ field.Model, body fieldpkt.AffectedAreaCreated) {
		got = body
	}
	defer func() { affectedAreaCreatedBroadcaster = orig }()
	// build ctx with tenant + a server.Model that Is(...) true, feed the event,
	// then assert got.SkillId()==2121006, got.SkillLevel()==20, got.NType()==1.
}
```

- [ ] **Step 3: Run it — expect FAIL**.

- [ ] **Step 4: Update `handleMistCreated`**

Replace the hardcoded `0` constructor call with the full event fields (arg order must match the final B1.1c constructor signature):
```go
body := fieldpkt.NewAffectedAreaCreated(
	e.MistId,
	e.Body.OwnerId,
	e.Body.Type,
	int32(e.Body.SourceSkillId),
	byte(e.Body.SourceSkillLevel),
	0, // phase (per B1.1b)
	e.Body.OriginX, e.Body.OriginY,
	e.Body.LtX, e.Body.LtY,
	e.Body.RbX, e.Body.RbY,
	0,               // tStart (server leaves 0 / per IDA)
	int32(e.Body.Duration),
)
```
(Map duration→tEnd if the IDA read-order treats tEnd as duration-relative; confirm in B1.1b.)

- [ ] **Step 5: Run it — expect PASS**; `go test -race ./... && go vet ./...` in atlas-channel — clean.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/
git commit -m "fix(channel): pass skill id/level/type to AffectedAreaCreated (task-080 B1.1)"
```

---

### Task B1.2: chat `Multi` serverbound — leading `updateTime` (gated GMS>83)

**Files:**
- Modify: `libs/atlas-packet/chat/serverbound/multi.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_chat_multi.go` (decode caller; verify only)
- Test: `libs/atlas-packet/chat/serverbound/multi_test.go`

IDA: `CUIStatusBar::SendGroupMessage@0x87f7f0` prepends `Encode4(update_time)` before the chat-type byte.

- [ ] **Step 1: Write the failing byte-shape test**

```go
func TestMultiUpdateTimeGate(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := Multi{updateTime: 0x11223344, chatType: 1, recipients: []uint32{7}, chatText: "hi"}
	// GMS v83: no updateTime → first byte is chatType.
	b83 := in.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	if b83[0] != 0x01 {
		t.Errorf("v83 first byte = 0x%02x, want chatType 0x01", b83[0])
	}
	// GMS v87: leading 4-byte updateTime little-endian.
	b87 := in.Encode(l, pt.CreateContext("GMS", 87, 1))(nil)
	want := []byte{0x44, 0x33, 0x22, 0x11}
	if !bytes.Equal(b87[:4], want) {
		t.Errorf("v87 leading updateTime = % x, want % x", b87[:4], want)
	}
	// Round-trip every variant.
	for _, v := range pt.Variants {
		ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
		out := Multi{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	}
}
```

- [ ] **Step 2: Run it — expect FAIL** (no `updateTime` field).

- [ ] **Step 3: Add the gated field**

Add `updateTime uint32` to `Multi`, a getter, and gate it in Encode/Decode:
```go
func (m Multi) Encode(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	hasUpdateTime := t.Region() != "GMS" || t.MajorVersion() > 83
	return func(_ map[string]interface{}) []byte {
		if hasUpdateTime {
			w.WriteInt(m.updateTime)
		}
		w.WriteByte(m.chatType)
		w.WriteByte(byte(len(m.recipients)))
		for _, r := range m.recipients {
			w.WriteInt(r)
		}
		w.WriteAsciiString(m.chatText)
		return w.Bytes()
	}
}
```
Mirror in Decode (read `updateTime` first when `hasUpdateTime`). Confirm the JMS gate: `t.Region() != "GMS" || t.MajorVersion() > 83` means all non-GMS (incl. JMS185) carry it — verify against the IDA gate; if JMS differs, switch to an explicit region check.

- [ ] **Step 4: Verify the handler still decodes correctly**

`character_chat_multi.go` calls `p.Decode(l, ctx)(...)` — it picks up the new field automatically. Confirm no separate construction site needs the new field. `go build ./...` in atlas-channel catches any break.

- [ ] **Step 5: Run it — expect PASS**; `go test -race ./chat/... && go vet ./...` in libs/atlas-packet — clean; `go build ./...` in atlas-channel — clean.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/chat/serverbound/ services/atlas-channel/
git commit -m "fix(packet): chat Multi leading updateTime gated GMS>83 (task-080 B1.2)"
```

---

### Task B1.3: quest `ActionStart`/`ActionComplete` — insert `nItemPos`

**Files:**
- Modify: `libs/atlas-packet/quest/serverbound/action_start.go`, `action_complete.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/quest_action.go`
- Test: `libs/atlas-packet/quest/serverbound/action_start_test.go`, `action_complete_test.go`

IDA: `CQuest::StartQuest@0x6b40a0` (actions 1/2) reads `Encode4(nItemPos)` (delivery-item slot, 0 normal) between `npcId` and the conditional `x,y`.

- [ ] **Step 1: Write the failing byte-shape test (ActionStart)**

```go
func TestActionStartItemPos(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// autoStart=false → npcId(4) + nItemPos(4) = 8 bytes, no x,y.
	in := ActionStart{npcId: 9000000, itemPos: 3, autoStart: false}
	b := in.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	if len(b) != 8 {
		t.Fatalf("got %d bytes, want 8: % x", len(b), b)
	}
	// bytes 4..8 are nItemPos little-endian = 3.
	if got := int32(binary.LittleEndian.Uint32(b[4:8])); got != 3 {
		t.Errorf("nItemPos = %d, want 3", got)
	}
	for _, v := range pt.Variants {
		ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
		out := NewActionStart(false)
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	}
}
```
Add the analogous `TestActionCompleteItemPos` (npcId + nItemPos + [x,y if autoStart] + selection int32).

- [ ] **Step 2: Run it — expect FAIL**.

- [ ] **Step 3: Insert `itemPos int32` between npcId and the x,y guard**

`action_start.go`:
```go
func (m ActionStart) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(_ map[string]interface{}) []byte {
		w.WriteInt(m.npcId)
		w.WriteInt32(m.itemPos)
		if m.autoStart {
			w.WriteInt16(m.x)
			w.WriteInt16(m.y)
		}
		return w.Bytes()
	}
}
```
Decode reads `m.itemPos = r.ReadInt32()` right after `npcId`. Add field + `ItemPos()` getter. Same for `action_complete.go` (keep the trailing `selection` int32 last).

- [ ] **Step 4: Update the handler**

In `quest_action.go`, log/forward `sp.ItemPos()` where relevant (delivery-item slot). The decode picks up the new field automatically; ensure no positional reader assumption breaks.

- [ ] **Step 5: Verify the `autoStart` gate against IDA**

Confirm atlas `q.AutoStart()` ↔ IDA `!CQuestMan::IsAutoAlertQuest(questId)` in `CQuest::StartQuest@0x6b40a0`. Record the read-order in the per-packet audit note. If the gate is inverted, fix `NewActionStart`/`NewActionComplete` callers.

- [ ] **Step 6: Run it — expect PASS**; `go test -race ./quest/... && go vet ./...` clean; `go build ./...` in atlas-channel clean.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet/quest/serverbound/ services/atlas-channel/
git commit -m "fix(packet): quest ActionStart/Complete insert nItemPos (task-080 B1.3)"
```

---

### Task B1.4: quest `ActionRestoreLostItem` — count-prefixed id array

**Files:**
- Modify: `libs/atlas-packet/quest/serverbound/action_restore_lost_item.go`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/quest_action.go`
- Test: `libs/atlas-packet/quest/serverbound/action_restore_lost_item_test.go`

IDA: `CQuest::OnCompleteQuestFailed@0x6b1fc0` (action 0). The base `Action` already consumes `action(byte)+questId(short)` upstream; this sub-packet model carries the remaining `count(int4) + count×itemId(int4)`. (The PRD's `Encode1(0)+Encode2(questId)+...` describes the whole frame including the base `Action` prefix.)

- [ ] **Step 1: Confirm read-order in IDA**

`mcp__ida-pro__decompile_function` at `CQuest::OnCompleteQuestFailed@0x6b1fc0`. Record whether `count` is int32 and the array element width (item ids = int32). Note in the per-packet audit `.md`.

- [ ] **Step 2: Write the failing test**

```go
func TestActionRestoreLostItemCountArray(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := ActionRestoreLostItem{itemIds: []uint32{4000001, 4000002}}
	b := in.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	// count(4) + 2×itemId(4) = 12 bytes.
	if len(b) != 12 {
		t.Fatalf("got %d bytes, want 12: % x", len(b), b)
	}
	if got := binary.LittleEndian.Uint32(b[0:4]); got != 2 {
		t.Errorf("count = %d, want 2", got)
	}
	for _, v := range pt.Variants {
		ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
		out := ActionRestoreLostItem{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	}
}
```

- [ ] **Step 3: Run it — expect FAIL**.

- [ ] **Step 4: Redesign the model**

Replace `unk1`/`itemId` with `itemIds []uint32`:
```go
type ActionRestoreLostItem struct {
	itemIds []uint32
}

func (m ActionRestoreLostItem) ItemIds() []uint32 { return m.itemIds }

func (m ActionRestoreLostItem) Encode(l logrus.FieldLogger, _ context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	return func(_ map[string]interface{}) []byte {
		w.WriteInt(uint32(len(m.itemIds)))
		for _, id := range m.itemIds {
			w.WriteInt(id)
		}
		return w.Bytes()
	}
}

func (m *ActionRestoreLostItem) Decode(_ logrus.FieldLogger, _ context.Context) func(r *request.Reader, options map[string]interface{}) {
	return func(r *request.Reader, _ map[string]interface{}) {
		count := r.ReadUint32()
		m.itemIds = make([]uint32, count)
		for i := range m.itemIds {
			m.itemIds[i] = r.ReadUint32()
		}
	}
}
```

- [ ] **Step 5: Update the handler**

In `quest_action.go` `QuestActionRestoreLostItem` case, iterate `sp.ItemIds()` and call `RestoreItem` per id (or extend the processor to accept a slice — read `quest.NewProcessor(...).RestoreItem` signature and adapt). Remove references to `sp.ItemId()`/`sp.Unk1()`.

- [ ] **Step 6: Run it — expect PASS**; module tests + vet clean; atlas-channel build clean.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet/quest/serverbound/ services/atlas-channel/
git commit -m "fix(packet): quest ActionRestoreLostItem count-prefixed id array (task-080 B1.4)"
```

---

### Task B1.5: `EffectWeather` JMS branch (region-dispatched body)

**Files:**
- Modify: `libs/atlas-packet/field/clientbound/effect_weather.go`
- Test: `libs/atlas-packet/field/clientbound/effect_weather_test.go`

IDA: JMS185 `CField::OnPacket@0x56e721` case 0x8B → `sub_5723E6`. JMS drops the leading `!active`/`m_nBlowType` byte; reads `Decode4 itemId` first, optional `Decode4 extra` when `get_consume_cash_item_type(itemId)==51`, optional `DecodeStr message` when `itemId!=0`. GMS/v83/v87 already correct (keep as-is).

- [ ] **Step 1: Confirm the JMS read-order in IDA** (`sub_5723E6`). Record in the per-packet note, especially whether `extra` precedes or follows `message` and the exact `consume_cash_item_type==51` condition.

- [ ] **Step 2: Write the failing test**

```go
func TestEffectWeatherJMSBranch(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewFieldEffectWeatherStart(5120000, "Happy holidays")
	// JMS185: itemId(4) first (no leading bool), then message (itemId!=0).
	b := in.Encode(l, pt.CreateContext("JMS", 185, 1))(nil)
	if got := binary.LittleEndian.Uint32(b[0:4]); got != 5120000 {
		t.Errorf("JMS leading itemId = %d, want 5120000 (no leading bool)", got)
	}
	// GMS v83 unchanged: leading bool then itemId.
	g := in.Encode(l, pt.CreateContext("GMS", 83, 1))(nil)
	if g[0] != 0x00 { // !active == false for a start packet
		t.Errorf("GMS leading byte = 0x%02x, want 0x00", g[0])
	}
	for _, v := range pt.Variants {
		ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
		out := EffectWeather{}
		pt.RoundTrip(t, ctx, in.Encode, out.Decode, nil)
	}
}
```

- [ ] **Step 3: Run it — expect FAIL**.

- [ ] **Step 4: Region-dispatch the body (design §3.2 — no 3rd nested guard)**

Add an `extra uint32` + `hasExtra bool` field (the type-51 case; set by the constructor when the server sends a cash weather item — confirm in spike). Implement:
```go
func (m EffectWeather) Encode(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(_ map[string]interface{}) []byte {
		if t.Region() == "JMS" {
			m.encodeJMS(w)
		} else {
			m.encodeGMS(w)
		}
		return w.Bytes()
	}
}

func (m EffectWeather) encodeGMS(w *response.Writer) {
	w.WriteBool(!m.active)
	w.WriteInt(m.itemId)
	if m.active {
		w.WriteAsciiString(m.message)
	}
}

func (m EffectWeather) encodeJMS(w *response.Writer) {
	w.WriteInt(m.itemId)
	if m.hasExtra {
		w.WriteInt(m.extra)
	}
	if m.itemId != 0 {
		w.WriteAsciiString(m.message)
	}
}
```
Mirror with `decodeGMS`/`decodeJMS`. The JMS decode cannot call the client's `get_consume_cash_item_type`; set `hasExtra` from the constructor on the encode side, and on decode read `extra` only per the confirmed deterministic condition. Document that decode exists for tests/round-trip; the server is the encoder. Confirm the round-trip stays byte-symmetric for the test inputs (pick test inputs without the ambiguous `extra` field, or set `hasExtra` deterministically).

- [ ] **Step 5: Run it — expect PASS**; module tests + vet clean. Confirm nesting `awk` stays clean (Phase F runs it repo-wide).

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/field/clientbound/
git commit -m "fix(packet): EffectWeather JMS region-dispatched body (task-080 B1.5)"
```

---

### Task B2.1: NPC continue-conversation discriminator (atlas-channel)

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/npc_continue_conversation.go`
- Test: `services/atlas-channel/atlas.com/channel/socket/handler/npc_continue_conversation_test.go` (create)

IDA: `OnSay@0x6dc110` (msgType 0), `OnAskYesNo@0x6dc5a0` (2/13), `OnAskText@0x6dc790` (3), `OnAskMenu@0x6dce00` (5), `OnAskAvatar@0x6dcff0` (8). Correct routing: text reply = msgType **3** (AskText) / **14** (AskBoxText) → `ContinueConversationText`; 5/8/9 → `ContinueConversationSelection`; 0/1/2/13 → no trailing body. The current handler wrongly treats msgType **2** as text.

- [ ] **Step 1: Write a failing test for a pure discriminator helper (per the monster_book_cover_test.go pattern)**

```go
func TestContinueConversationBodyKind(t *testing.T) {
	cases := []struct {
		msgType byte
		want    bodyKind
	}{
		{0, bodyNone}, {1, bodyNone}, {2, bodyNone}, {13, bodyNone},
		{3, bodyText}, {14, bodyText},
		{5, bodySelection}, {8, bodySelection}, {9, bodySelection},
	}
	for _, c := range cases {
		if got := bodyKindFor(c.msgType); got != c.want {
			t.Errorf("msgType %d: got %v, want %v", c.msgType, got, c.want)
		}
	}
}
```

- [ ] **Step 2: Run it — expect FAIL** (`bodyKindFor`/`bodyKind` undefined).

- [ ] **Step 3: Extract a pure discriminator + rewrite routing**

Add to the handler file:
```go
type bodyKind int

const (
	bodyNone bodyKind = iota
	bodyText
	bodySelection
)

// bodyKindFor maps the client's lastMessageType to the trailing body the
// serverbound continue-conversation packet carries (task-080 B2.1).
//   3 (OnAskText) / 14 (OnAskBoxText) → text reply
//   5 (OnAskMenu) / 8 (OnAskAvatar) / 9 → selection
//   0/1/2/13 (Say/AskYesNo) → no trailing body
func bodyKindFor(msgType byte) bodyKind {
	switch msgType {
	case 3, 14:
		return bodyText
	case 5, 8, 9:
		return bodySelection
	default:
		return bodyNone
	}
}
```
Rewrite `NPCContinueConversationHandleFunc` to switch on `bodyKindFor(lastMessageType)`:
- `bodyText` (and `action != 0`): decode `ContinueConversationText`, continue conversation with the text.
- `bodySelection`: decode `ContinueConversationSelection`, continue with `selection`.
- `bodyNone`: no trailing decode; dispose or continue per `action` exactly as the prior `lastMessageType==2 && action==0` path did.
Remove the hardcoded `== 2` check.

- [ ] **Step 4: Run it — expect PASS**; `go test -race ./... && go vet ./...` in atlas-channel — clean.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel/
git commit -m "fix(channel): NPC continue-conversation discriminator 3/14/5/8/9 (task-080 B2.1)"
```

---

### Task B2.2: Hired-merchant serverbound decode + handler

**Files:**
- Modify: `libs/atlas-packet/merchant/serverbound/operation.go` (currently a bare constant)
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/hired_merchant_operation.go` (currently a TODO stub)
- Test: `libs/atlas-packet/merchant/serverbound/operation_test.go` (create)

- [ ] **Step 1: Enumerate the hired-merchant serverbound op family in IDA**

Identify the op-byte dispatcher for the hired-merchant/entrusted-shop serverbound packet across the four IDBs. Record each op byte + body read-order in the per-packet note. (The clientbound side already has OpenShop/ErrorSimple/ShopSearch/ShopRename/RemoteShopWarp/ConfirmManage/FreeFormNotice as a shape reference.)

- [ ] **Step 2: Write a failing decode test for the operation discriminator**

```go
func TestHiredMerchantOperationDecode(t *testing.T) {
	// op byte + minimal body per the confirmed IDA shape.
	raw := []byte{ /* op */ 0x00 /* ...body... */ }
	req := request.Request(raw)
	reader := request.NewRequestReader(&req, 0)
	p := Operation{}
	p.Decode(logrus.New(), context.Background())(&reader, map[string]interface{}{})
	if p.Mode() != 0 {
		t.Fatalf("mode = %d, want 0", p.Mode())
	}
}
```

- [ ] **Step 3: Run it — expect FAIL**.

- [ ] **Step 4: Implement the `Operation` struct + Decode**

Add a struct carrying the op `mode` byte and the per-op fields the dispatcher needs (model only the ops the channel handler will act on; document the rest). Keep `HiredMerchantOperationHandle` as the `Operation()` name.

- [ ] **Step 5: Implement the channel handler**

Replace the TODO in `hired_merchant_operation.go` with a decode + dispatch that checks with the merchant processor whether the character may open/operate a merchant (read how other handlers call their processors). Where a sub-op is out of scope, log a `Debugf` and return (no silent panic, no `// TODO` left behind).

- [ ] **Step 6: Run it — expect PASS**; module tests + vet clean; atlas-channel build clean.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet/merchant/serverbound/ services/atlas-channel/
git commit -m "feat(channel): hired-merchant serverbound decode + handler (task-080 B2.2)"
```

---

### Task B2.3: Merchant modes 1 / 8 / 11 disposition

**Files:**
- Modify: `libs/atlas-packet/merchant/clientbound/operation.go` (mode 8 emitter, mode 11 constant)
- Modify: the atlas-channel merchant handler (emit mode 8 where appropriate)
- Test: `libs/atlas-packet/merchant/clientbound/operation_test.go`
- Note: `docs/tasks/task-080-packet-audit-closeout/spike-merchant-mode1.md`

IDA: `OnEntrustedShopCheckResult` (mode 8 = `Decode4 shopId + Decode1 channelId`; mode 1 absent in v95; mode 11 present, StringPool 3508).

- [ ] **Step 1: Spike mode 1 across all four IDBs (design §2/Q3)**

Read `OnEntrustedShopCheckResult` (and the entrusted-shop check-result dispatch) in v83/v87/v95/JMS185. If mode 1 is absent in all four → document client/KMS-only in `spike-merchant-mode1.md`, do not implement. If present in any → implement gated to those versions with a byte test.

- [ ] **Step 2: Write a failing test for mode 8**

```go
func TestEntrustedShopUnknownChannel(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := NewEntrustedShopUnknownChannel(123456, 5) // shopId, channelId
	b := in.Encode(l, pt.CreateContext("GMS", 95, 1))(nil)
	// mode(1) + shopId(4) + channelId(1) = 6 bytes.
	if len(b) != 6 || b[0] != 8 {
		t.Fatalf("mode-8 packet = % x, want mode 8 + 6 bytes", b)
	}
}
```

- [ ] **Step 3: Run it — expect FAIL**.

- [ ] **Step 4: Implement mode 8 emitter + mode 11 constant**

Add an `EntrustedShopUnknownChannel{mode=8, shopId uint32, channelId byte}` clientbound type (Encode writes mode, int shopId, byte channelId). Add a named constant for mode 11 (StringPool 3508); add its emitter only if a server-side path exercises it — otherwise register it as a defined-but-unused constant (note in the registry, §4.8). Wire the mode-8 emit into the channel merchant handler where the "unknown channel" notice belongs.

- [ ] **Step 5: Run it — expect PASS**; module tests + vet clean.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/merchant/ services/atlas-channel/ docs/tasks/task-080-packet-audit-closeout/spike-merchant-mode1.md
git commit -m "feat(packet): merchant mode 8 emitter; mode 1/11 disposition (task-080 B2.3)"
```

---

# PHASE C — JMS cash-shop NX-payment (B5)

Path is already region-agnostic and wired: `CashShopOperationHandleFunc` → `RequestPurchase` → Kafka `REQUEST_PURCHASE` → atlas-cashshop consumer → `PurchaseAndEmit` → wallet. `RequestPurchaseCommandBody` carries `Currency` + `SerialNumber`. The cash serverbound bodies already region-dispatch inline (`ShopOperationBuy` branches `GMS>=87`). Gaps: (1) JMS-correct serverbound bodies for the 5 ops; (2) the JMS template has no `CashShopOperationHandle` entry / op-byte map.

IDA: `CCashShop::OnBuy@0x47eaa7` (op 3), `SendGiftsPacket@0x47bced` (0x2E), `OnBuyCouple@0x48085a` (0x1E), `OnBuyFriendship@0x481184` (0x24), `OnRebateLockerItem@0x47c059` (0x1B).

---

### Task B5.1a: JMS body for `ShopOperationBuy`

**Files:**
- Modify: `libs/atlas-packet/cash/serverbound/shop_operation_buy.go`
- Test: `libs/atlas-packet/cash/serverbound/shop_operation_buy_test.go`

- [ ] **Step 1: Read `CCashShop::OnBuy@0x47eaa7` (JMS185 IDB)** and record the SPW-string + serial-number read-order.

- [ ] **Step 2: Write the failing JMS byte test**

```go
func TestShopOperationBuyJMS(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	in := ShopOperationBuy{ /* fields per JMS shape */ }
	b := in.Encode(l, pt.CreateContext("JMS", 185, 1))(nil)
	// assert the JMS layout length + key field offsets per OnBuy read-order.
	_ = b
}
```
(Fill the assertion from the confirmed read-order.)

- [ ] **Step 3: Run it — expect FAIL**.

- [ ] **Step 4: Add the JMS branch via region-dispatched body (NOT a 3rd nested guard)**

The current `ShopOperationBuy` branches `GMS>=87` inline (2 levels). Adding JMS would risk a 3rd level — instead refactor to dispatch:
```go
func (m ShopOperationBuy) Encode(l logrus.FieldLogger, ctx context.Context) func(map[string]interface{}) []byte {
	w := response.NewWriter(l)
	t := tenant.MustFromContext(ctx)
	return func(_ map[string]interface{}) []byte {
		if t.Region() == "JMS" {
			m.encodeJMS(w)
		} else {
			m.encodeGMS(t, w) // existing GMS body, ≤2 guards
		}
		return w.Bytes()
	}
}
```
Implement `encodeJMS`/`decodeJMS` per the IDA read-order. Keep the GMS body identical to today's.

- [ ] **Step 5: Run it — expect PASS**; `go test -race ./cash/... && go vet ./...` clean; nesting `awk` clean.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/cash/serverbound/shop_operation_buy.go libs/atlas-packet/cash/serverbound/shop_operation_buy_test.go
git commit -m "fix(packet): JMS ShopOperationBuy body (task-080 B5.1)"
```

---

### Task B5.1b–e: JMS bodies for gift / couple / friendship / rebate

**Files (one task each — repeat the B5.1a structure):**
- `shop_operation_gift.go` ← `SendGiftsPacket@0x47bced` (op 0x2E)
- `shop_operation_buy_couple.go` ← `OnBuyCouple@0x48085a` (op 0x1E)
- `shop_operation_buy_friendship.go` ← `OnBuyFriendship@0x481184` (op 0x24)
- `shop_operation_rebate_locker_item.go` ← `OnRebateLockerItem@0x47c059` (op 0x1B)

For each: (1) read the IDA function, record the read-order; (2) write the failing JMS byte test; (3) run → FAIL; (4) add `encodeJMS`/`decodeJMS` via region dispatch (refactor the existing inline GMS guards into `encodeGMS` if adding JMS would create a 3rd nesting level); (5) run → PASS + module tests + vet + nesting `awk` clean; (6) commit `fix(packet): JMS Shop<Op> body (task-080 B5.1)`.

- [ ] **Task B5.1b — gift** (steps 1–6 above)
- [ ] **Task B5.1c — buy_couple** (steps 1–6 above)
- [ ] **Task B5.1d — buy_friendship** (steps 1–6 above)
- [ ] **Task B5.1e — rebate_locker_item** (steps 1–6 above)

---

### Task B5.1f: JMS template op-byte map + interaction remaps

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`
- Note: per-op IDA citations in the per-packet audit notes.

- [ ] **Step 1: Confirm there is no `CashShopOperationHandle` handler entry in the JMS template** (the explorer confirmed the writer `CashShopOperation` exists at op `0x164` but no serverbound handler entry).

- [ ] **Step 2: Add the `CashShopOperationHandle` handler entry**

Add a handler entry mapping the JMS cash-shop serverbound opcode to `CashShopOperationHandle`, matching the schema of existing handler entries (opCode → handler name). Then map the per-op sub-bytes: buy=3, gift=0x2E, couple=0x1E, friendship=0x24, rebate=0x1B — each justified by its cited `Encode1(...)`. Use whatever sub-op-config mechanism the template uses for other op families (read how `CashShopOperationHandle` is configured in a GMS v95 template as the reference schema).

- [ ] **Step 3: Remap the two template-only interaction ops** (bodies already match): PersonalStore `BuyItem` op 0x14/0x1F (GMS 0x17/0x22), `DeliverBlackList` op 0x1B (GMS 0x1E). Cite each `Encode1(...)`.

- [ ] **Step 4: Validate the JSON parses**

Run: `python3 -m json.tool services/atlas-configurations/seed-data/templates/template_jms_185_1.json > /dev/null`
Expected: valid JSON (no output, exit 0).

- [ ] **Step 5: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/template_jms_185_1.json
git commit -m "feat(config): JMS cash-shop op-byte map + interaction remaps (task-080 B5.1)"
```

---

### Task B5.1g: Verify JMS routing into the existing wallet flow

**Files:**
- Read-only verification (+ minimal `RequestPurchaseCommandBody` extension only if a JMS field can't be carried)
- Test: extend `services/atlas-channel/.../cashshop` or `services/atlas-cashshop/.../cashshop` tests as needed

- [ ] **Step 1: Trace the path with JMS inputs**

Confirm `CashShopOperationHandleFunc` decodes the JMS `ShopOperationBuy` (region dispatch) and calls `RequestPurchase(characterId, serialNumber, isPoints, currency, zero)` → `RequestPurchaseCommandProvider(characterId, serialNumber, currency)` → consumer `PurchaseAndEmit(characterId, currency, serialNumber)` → wallet debit. No new topic/command.

- [ ] **Step 2: Scope-guard check**

If a JMS body needs a field the current `RequestPurchaseCommandBody{Currency, SerialNumber}` can't carry (design §4.5 scope guard), surface it and extend the command body **minimally** (add the one field + its producer/consumer plumbing) — do NOT invent a new topic. Default expectation: currency+serial suffices.

- [ ] **Step 3: Add a routing test**

Assert the JMS-decoded buy produces a `RequestPurchaseCommandBody` with the right currency + serial (mirror the existing channel cashshop producer test).

- [ ] **Step 4: Verify + commit**

`go test -race ./... && go vet ./...` in atlas-channel and atlas-cashshop.
```bash
git add services/atlas-channel/ services/atlas-cashshop/
git commit -m "test(cashshop): verify JMS cash purchase routes to wallet (task-080 B5.1)"
```

---

# PHASE D — Verification spikes (B3, B4, B6)

Each spike ends in a verdict recorded in the per-packet audit `.md` (and feeds §4.8). Where a real divergence surfaces, fix it in-task with a byte test + gate (reuse the Phase-B task shape). Standard procedure per spike: (a) `mcp__ida-pro__decompile_function` the cited FName in each relevant IDB; (b) enumerate the modes/op-bytes; (c) diff against the Atlas template/router/struct; (d) record verdict + IDA evidence; (e) fix-in-task if divergent.

---

### Task B3.1: messenger serverbound `Operation` full enum
- [ ] Enumerate the messenger serverbound modes in IDA; confirm no modes beyond 0/2/3/5/6. Verify `atlas-messengers` routing matches. File: `libs/atlas-packet/messenger/serverbound/operation.go`. Record verdict; fix-in-task if a mode is missing. Commit `audit(packet): messenger serverbound enum verdict (task-080 B3.1)`.

### Task B3.2: messenger `declineMode` sub-enum
- [ ] In `OnBlocked` (mode=5) confirm `if v3` → StringPool 0x31A vs 0x31B; confirm only 0/1 vs more. File: `libs/atlas-packet/messenger/clientbound/invite_declined.go`. Record verdict; fix-in-task if divergent. Commit.

### Task B3.3: npc shop-operation clientbound mode enum
- [ ] Cross-check the `operations` resolver vs every `CShopDlg::OnPacket@0x6eb7d0` case (`nType==365`); confirm modes 4/6/7/0xB/0xC carry no emitter. Files: `libs/atlas-packet/npc/clientbound/shop_operation.go`, `shop_operation_body.go`. Record verdict; fix-in-task if divergent. Commit.

### Task B3.4: npc shop serverbound op-byte values (esp. LEAVE)
- [ ] Confirm channel `operations` config BUY=0/SELL=1/RECHARGE=2 against IDA (`SendBuyRequest@0x6e9bb0` etc.); locate/confirm the LEAVE op value and that no body trails it. Files: `libs/atlas-packet/npc/serverbound/{shop,shop_buy,shop_sell,shop_recharge}.go` + `services/atlas-channel/.../npc_shop.go`. Record verdict; fix-in-task if divergent. Commit.

### Task B3.5: 7 interaction serverbound sub-ops (no located IDA sender)
- [ ] Focused IDA spike per sub-op: `operation_{create,open,cash_trade_open,invite_decline,visit,merchant_name_change,personal_store_set_visitor}.go`. Assign a verdict each; fix any real divergence in-task with a byte test. Commit one per sub-op or batched as `audit(packet): interaction serverbound sub-op verdicts (task-080 B3.5)`.

### Task B3.6: social-domain sub-op enum-drift cross-version pass
- [ ] Verify template-configured sub-op VALUE spaces (mode/op numbers) match the client across v83/v87/v95/JMS185 for buddy/chat/guild/party/note dispatchers + templates, incl. `BuddyError` conditional-string arms (modes 0x10/0x11/0x13/0x16). Per-struct wire shapes are already ✅ — this is a config-value audit. Any fix is config-or-constant level. Record the per-version comparison table in the note. Commit `audit(packet): social sub-op enum-drift four-version verdict (task-080 B3.6)`.

---

### Task B4.1: confirm v87 provisional gates (stat `Changed`, ui `Lock`)

**Files:**
- Modify (per finding): `libs/atlas-packet/stat/clientbound/changed.go`, `libs/atlas-packet/ui/clientbound/lock.go`
- Test: `libs/atlas-packet/stat/clientbound/changed_test.go`, `libs/atlas-packet/ui/clientbound/lock_test.go`

- [ ] **Step 1: Read the v87 IDB** for `GW_CharacterStat::DecodeChangeStat` (HP/MP width) and `CUserLocal::OnSetDirectionMode` (ui Lock int32). Determine whether the v87 boundary matches the current `>=95` (stat) and `>=90` (ui Lock) gates.

- [ ] **Step 2: Add a v87 byte assertion to each test**

Extend `TestStatChangedV95WireWidths` and `TestUiLockWireShape` with explicit v87 expectations (e.g. v87 stat single-HP byte count; v87 ui-Lock size). Run — they encode the current behavior; if the IDB shows the gate is wrong at v87, the assertion will reflect the corrected expectation and FAIL first.

- [ ] **Step 3: Tighten/keep the gate per evidence**

If v87 matches the gate, keep it and the new tests pass. If v87 needs a different boundary, adjust the `v95Plus`/`>=90` condition and update the tests. Record the v87 read-order in the per-packet notes.

- [ ] **Step 4: Run → PASS; module tests + vet clean; commit**

```bash
git add libs/atlas-packet/stat/ libs/atlas-packet/ui/
git commit -m "fix(packet): confirm v87 stat-Changed + ui-Lock gates (task-080 B4.1)"
```

---

### Task B6.1: Login IDA-export backlog — export + audit + verdicts

**Files:**
- Modify: `docs/packets/ida-exports/gms_v83.json`, `gms_v87.json`, `gms_v95.json`, `gms_jms_185.json` (add exported FNames)
- Read/modify: `libs/atlas-packet/login/` writers/handlers (fix if divergent)
- Note: `docs/tasks/task-080-packet-audit-closeout/spike-login.md`

- [ ] **Step 1: Export the addressed + bare-handler FNames**

Via the `packet-audit export` (live IDA-MCP) path, export to the four version jsons:
- Addressed: `CLogin::OnViewAllCharResult@0x5de120`, `SendSelectCharPacketByVAC@0x5d7550`, `OnSelectCharacterByVACResult@0x5de670`, `OnDenyLicense@0x5d45d0`, `CLicenseDlg::OnButtonClicked@0x5ff870`, `LoginAuth`.
- Bare handlers: `AfterLoginHandle` (0x09), `RegisterPinHandle` (0x0A), PIC family (0x15–0x1E), `SetGenderHandle` (0x08), `WorldCharacterListRequest` (0x05), `ServerStatus` (clientbound), `PicResult` (clientbound).

- [ ] **Step 2: Run the audit over the new exports** and assign verdicts per login packet (re-run the Task A0 audit invocation; the new FNames now resolve).

- [ ] **Step 3: Resolve `LoginAuth` (design §2/Q2)**

Absent in all four → remove the writer + template entry (record "removed, not in any baseline"). Present only in JMS185 → gate `Region()=="JMS"`, audit, verdict. Present in GMS → audit normally.

- [ ] **Step 4: Resolve v87 login quirks**

`SendCheckPasswordPacket@0x62dfb4` v87 appends `Encode4(PartnerCode)` (zero functional impact → read-and-discard or document). `SendSelectCharPacket` 0x1D/0x1E v87 PIC opcode layout differs → add v87-specific handler variants or opcode-keyed dispatch (whichever the export shows is minimal). Add byte tests for any code change.

- [ ] **Step 5: Document bare handlers** that map to a real client function (audited) vs no client counterpart (documented as intentional) in `spike-login.md`.

- [ ] **Step 6: Verify + commit**

`go test -race ./... && go vet ./...` in libs/atlas-packet (and atlas-channel if login handlers changed).
```bash
git add docs/packets/ida-exports/ libs/atlas-packet/login/ services/atlas-channel/ docs/tasks/task-080-packet-audit-closeout/spike-login.md
git commit -m "fix(packet): login IDA-export backlog audited + v87 quirks resolved (task-080 B6)"
```

---

# PHASE E — Docs + ledger curation (§4.8)

Run last — regenerated artifacts must reflect the final code + analyzer.

---

### Task E1: Regenerate the four `SUMMARY.md`

**Files:**
- Modify: `docs/packets/audits/{gms_v83,gms_v87,gms_v95,jms_v185}/SUMMARY.md` (+ per-packet `.md` regenerated)

- [ ] **Step 1: Run the four-version audit into the real output dir**

For each version, run the Task A0 invocation with `-output docs/packets/audits/<ver>` (the live tree, not `/tmp`). Use the live IDA-MCP source (`-ida-source mcp`) if the spikes added FNames not yet in the jsons; otherwise the updated jsons.

- [ ] **Step 2: Confirm no spurious ❌/🔍**

`grep -rE '\| (❌|🔍) \|' docs/packets/audits/*/SUMMARY.md`. Every remaining ❌/🔍 must be either a real finding already fixed (re-run should now be ✅) or an entry that appears verbatim in the §4.8 accepted-exception registry (Task E2). No other ❌/🔍 may remain.

- [ ] **Step 3: Commit**

```bash
git add docs/packets/audits/
git commit -m "docs(packets): regenerate four-version SUMMARY after fixes + analyzer (task-080 §4.8)"
```

---

### Task E2: Curate `_pending.md` (both copies) → accepted-exclusions registry

**Files:**
- Rewrite: `docs/packets/ida-exports/_pending.md`
- Rewrite: `docs/packets/audits/gms_v95/_pending.md`

- [ ] **Step 1: Reconcile every deferral against this task's outcomes**

For each entry in both `_pending.md` files, mark it resolved (✅/fixed, cite the task above) or move it to the **accepted permanent exclusions** registry with IDA evidence + a one-line justification (genuinely-unanalyzable opaque buffers from Task A3's `Opaque` set, removed-legacy FNames from B6, client/KMS-only modes from B2.3/B6).

- [ ] **Step 2: Replace deferral content with the registry**

Both files end with: zero actionable items, the accepted-exclusions table, and a pointer to task-080 as closeout of record. No entry may require future code or audit action.

- [ ] **Step 3: Cross-check zero actionable items**

`grep -nE 'DEFERRED|pending|TODO|🔍' docs/packets/ida-exports/_pending.md docs/packets/audits/gms_v95/_pending.md` — every hit must be inside the accepted-exclusions registry (a blessed permanent exclusion), not an open action.

- [ ] **Step 4: Commit**

```bash
git add docs/packets/ida-exports/_pending.md docs/packets/audits/gms_v95/_pending.md
git commit -m "docs(packets): curate _pending to accepted-exclusions registry (task-080 §4.8)"
```

---

### Task E3: Update `TOTAL.md` + add the new-version-pass guide

**Files:**
- Modify: `docs/packets/audits/gms_v95/TOTAL.md`
- Create: `docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md`

- [ ] **Step 1: Update `TOTAL.md`**

Flip task-080 (and any sibling) statuses to shipped, recompute the §2 verdict roll-up from the regenerated SUMMARYs, and replace §3/§5 with "**baseline complete — zero open actionable deferrals**".

- [ ] **Step 2: Write the new-version-pass guide**

Document: where IDBs go; how to run `packet-audit export`/audit (the exact invocation from Task A0); how SUMMARY/TOTAL/_pending relate; the gate-naming convention `Region()=="GMS" && MajorVersion()>=N`; and the region-dispatched body strategy for >2-version divergences (design §3.2). Reference the analyzer enhancements as the de-noising baseline.

- [ ] **Step 3: Commit**

```bash
git add docs/packets/audits/gms_v95/TOTAL.md docs/packets/audits/STARTING_A_NEW_VERSION_PASS.md
git commit -m "docs(packets): TOTAL baseline-complete + new-version-pass guide (task-080 §4.8)"
```

---

# PHASE F — Verify gates + code review

### Task F1: Full CLAUDE.md verify gates

- [ ] **Step 1: Per-module tests + vet**

For every changed module (`libs/atlas-packet`, `tools/packet-audit`, `services/atlas-maps`, `services/atlas-channel`, `services/atlas-cashshop`, `services/atlas-configurations` if touched):
```bash
go test -race ./...   # in each module dir
go vet ./...          # in each module dir
```
Expected: clean in every changed module.

- [ ] **Step 2: Builds**

`go build ./...` in each changed service. Expected: clean.

- [ ] **Step 3: Nesting `awk` guard**

Run the repo nesting-cap `awk` check (the 2-nested-guard policy) across `libs/atlas-packet`. Expected: no encoder/decoder exceeds two nested `if` guards (the region-dispatched bodies in B1.5/B5.1 must keep it clean).

- [ ] **Step 4: redis-key-guard**

Run: `tools/redis-key-guard.sh` from the repo root (use `GOWORK=off` if needed). Expected: clean.

- [ ] **Step 5: docker buildx bake per touched go.mod**

From the worktree root, for every service whose `go.mod` was touched (at minimum atlas-maps, atlas-channel, atlas-cashshop; add atlas-configurations if its go.mod changed):
```bash
docker buildx bake atlas-maps
docker buildx bake atlas-channel
docker buildx bake atlas-cashshop
```
Expected: each image builds (catches a missing `COPY libs/...` the workspace build can't). This is mandatory, not optional.

- [ ] **Step 6: Regression guard on closed items**

Confirm storage `Show`, `MonsterControl`, and SETFIELD/WarpToMap gate files are untouched (`git diff --stat main -- <those files>` empty) and their tests green.

---

### Task F2: Code review before PR

- [ ] **Step 1: Run the review orchestration**

Invoke `superpowers:requesting-code-review` (dispatches `plan-adherence-reviewer` + `backend-guidelines-reviewer`; no frontend changes here). Each writes to `docs/tasks/task-080-packet-audit-closeout/audit.md`.

- [ ] **Step 2: Address findings**

Use `superpowers:receiving-code-review`. Fix real issues; for each fix re-run the affected module's `go test -race ./...` + `go vet ./...`.

- [ ] **Step 3: Final verification statement**

Confirm all PRD §10 acceptance criteria are met (see the checklist below). Only then proceed to `superpowers:finishing-a-development-branch`.

---

## Acceptance criteria coverage (PRD §10)

| Criterion | Task(s) |
|---|---|
| B1.1–B1.5 real wire bugs fixed + byte tests + gates | B1.1a–d, B1.2, B1.3, B1.4, B1.5 |
| B2.1–B2.3 handler fixes (continue-conversation, hired-merchant, merchant mode 8; 1/11 dispositioned) | B2.1, B2.2, B2.3 |
| B3.1–B3.6 verification deferrals → verdicts; social four-version enum-drift | B3.1–B3.6 |
| B4.1 v87 stat-Changed + ui-Lock gates confirmed | B4.1 |
| B5.1 JMS cash bodies + template remaps + wallet routing; interaction op remaps; no 3rd nested guard | B5.1a–g |
| B6 login export + audit + verdicts; bare handlers; v87 quirks | B6.1 |
| §4.7 analyzer enhanced; clean four-version re-run | A0–A4, E1 |
| §4.8 four SUMMARYs + zeroed `_pending` (both) + baseline-complete TOTAL + new-version-pass guide | E1, E2, E3 |
| Closed items untouched + green | F1 Step 6 |
| All build/verify gates pass | F1 |
| Code review run before PR | F2 |

---

## Self-review notes (run before execution)

- **Spec coverage:** every PRD bucket (B1–B6), analyzer §4.7, and docs §4.8 maps to a task above (see the table). The four PRD §9 open questions are resolved in the design and operationalized here: Q1→B1.1a/b (extend `CreatedBody`, confined `Type` spike), Q2→B6.1 Step 3 (`LoginAuth` export-then-decide), Q3→B2.3 Step 1 (mode-1 confirm-then-dispose), Q4→A1–A4 (fix tractable classes, register opaque residue).
- **Type consistency:** the AffectedAreaCreated constructor signature is defined once in B1.1c and consumed in B1.1d (keep the arg order identical — adjust B1.1d to whatever final order B1.1c commits). `CreatedBody`'s new fields (`SourceSkillId`, `SourceSkillLevel`, `Type`) are added in B1.1a (atlas-maps) and mirrored + read in B1.1d (atlas-channel). `bodyKind`/`bodyKindFor` defined and tested in B2.1. `widthEquivalent` defined in A1. `encodeJMS`/`decodeJMS`/`encodeGMS` naming is consistent across B1.5 and B5.1.
- **Spikes:** B3/B4/B6 are genuine IDA investigations; their tasks specify the exact FName@address to decompile, the diff target, the verdict-recording location, and a byte-test scaffold for the fix-if-divergent path — not predetermined code, because the outcome is the verdict.
- **No silent caps / no TODOs in deliverables:** any analyzer residue that stays ❌/🔍 after Phase A must be logged into the §4.8 accepted-exclusions registry (E2), never silently left in SUMMARY; the hired-merchant handler (B2.2) replaces the existing `// TODO` stub rather than leaving it.
