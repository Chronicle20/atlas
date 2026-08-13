# Mist Broadcast Writer Wiring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make mist (affected-area) broadcasts reach the client on **every supported version** by declaring the `AffectedAreaCreated`/`AffectedAreaRemoved` writers in atlas-channel, registering their opcodes in every seed template with an IDB-sourced opcode, completing both mist rows of the packet coverage matrix, and rolling the fix out to live tenants.

**Architecture:** The codecs, the mist consumer, and the producers already work; only the name→opcode resolution chain is broken (Go declaration + template config, per the design). Sequencing per design §6: Tier A fixtures first, then the two wiring fixes across eight versions, then Tier B discovery for the three versions with no sourced opcode, then build gates, then rollout docs + live patch + e2e.

**Tech Stack:** Go (atlas-channel, libs/atlas-packet), JSON seed templates (atlas-configurations), `tools/packet-audit` (evidence pin + matrix), ida-pro-mcp (Tier B discovery), atlas-configurations REST (JSON:API) for live rollout.

## Global Constraints

- **No codec byte changes on any already-verified version.** `affected_area_created.go` / `affected_area_removed.go` produce identical output for v61–jms_185 at the end of this task. If a per-version read order does not match the current codec on a ✅ version, STOP and escalate (design §7). Tier B may add an *additive* version gate only (design §4.3), using the `MajorAtLeast` idiom — never a raw `> N` comparison.
- **No consumer/producer changes**: the mist consumer, atlas-maps mist domain, and atlas-monsters producers are out of scope.
- **No derived-unverified opcodes.** A template is wired only when its opcode was read out of that version's IDB. If discovery is inconclusive, the template stays unwired (Outcome C). Interpolating a band from neighbouring versions is prohibited — see design §4.3 approach B.
- Every fixture byte must trace to a decompile line in `docs/packets/ida-exports/` or a live decompile (CLAUDE.md "Verification Over Memory") — never to MapleStory knowledge.
- Opcodes live only in tenant config, never hard-coded in Go (DOM-25 posture).
- No literal home/absolute paths in committed files; repo-relative paths or placeholders only.
- **Baseline re-verified 2026-08-07 (post-merge with main):** `go run ./tools/packet-audit matrix --check` exits **0** on this branch before any change. The bar for every matrix step is a clean exit 0.
- All commands run from the worktree root (`.worktrees/task-165-mist-writer-template-wiring`) unless a `cd` is shown.

## Reference facts (verified against source, 2026-08-07)

### Version surface

Eleven seed templates; nine matrix columns (`tools/packet-audit/internal/matrix/model.go:14`). gms_92 and gms_12 are template-only. **No template currently has either writer** (`grep -c AffectedArea` = 0 in all eleven).

**Tier A — opcode IDB-sourced, wire directly.** From `docs/packets/registry/<ver>.yaml`, agreeing with `docs/packets/audits/STATUS.md:346-347`:

| Template | AffectedAreaCreated | AffectedAreaRemoved | Insert before |
|---|---|---|---|
| `template_gms_61_1.json` | `0x0D2` | `0x0D3` | `SpawnDoor` `0x0D4` |
| `template_gms_72_1.json` | `0x0F3` | `0x0F4` | `SpawnDoor` `0x0F5` |
| `template_gms_79_1.json` | `0x0FB` | `0x0FC` | `SpawnDoor` `0x0FD` |
| `template_gms_83_1.json` | `0x111` | `0x112` | `SpawnDoor` `0x113` |
| `template_gms_84_1.json` | `0x118` | `0x119` | `SpawnDoor` `0x11A` |
| `template_gms_87_1.json` | `0x122` | `0x123` | `SpawnDoor` `0x124` |
| `template_gms_95_1.json` | `0x148` | `0x149` | `SpawnDoor` `0x14A` |
| `template_jms_185_1.json` | `0x126` | `0x127` | `SpawnDoor` `0x128` |

The band layout is uniform in all eight: `DropDestroy` at (SPAWN_MIST − 4), a 2-slot gap at SPAWN_MIST/REMOVE_MIST, `SpawnDoor` at (SPAWN_MIST + 2), `RemoveDoor` at (+3). Both entries always slot immediately before `SpawnDoor`.

**Tier B — opcode unknown, discovery required:** `template_gms_48_1.json`, `template_gms_92_1.json`, `template_gms_12_1.json` (Task 8).

### IDA addresses

`CAffectedAreaPool::OnAffectedAreaCreated` (from `docs/packets/ida-exports/<ver>.json`, key `functions.<fname>.address`):

| Version | Address | tStart? | Body bytes | Current matrix |
|---|---|---|---|---|
| gms_v61 | `0x423edc` | no | 39 | ✅ (byte-pinned) |
| gms_v72 | `0x42e36c` | no | 39 | ✅ (byte-pinned) |
| gms_v79 | `0x42e7fc` | no | 39 | ✅ **length-only — must re-pin** |
| gms_v83 | `0x431a63` | no | 39 | ❌ |
| gms_v84 | `0x4326ca` | no | 39 | ❌ |
| gms_v87 | `0x432f3f` | no | 39 | ❌ |
| gms_v95 | `0x437ec0` | **yes** | 43 | ❌ |
| jms_v185 | `0x436572` | no | 39 | ❌ |

`CAffectedAreaPool::OnAffectedAreaRemoved`:

| Version | Address | Current matrix |
|---|---|---|
| gms_v61 | `0x4246b0` | 🟡ᶠ |
| gms_v72 | `0x42ec4e` | 🟡ᶠ |
| gms_v79 | `0x42f0de` | 🟡ᶠ |
| gms_v83 | `0x43234d` | ✅ |
| gms_v84 | `0x432fb4` | ✅ |
| gms_v87 | `0x43388c` | ✅ |
| gms_v95 | `0x4360a0` | ✅ |
| jms_v185 | `0x436eda` | ✅ |

Common Created read order (everything except gms_v95): `Decode4 dwId, Decode4 nType, Decode4 dwOwnerId, Decode4 nSkillID, Decode1 nSLV, Decode2 phase, DecodeBuf(16) rcArea (4×int32 absolute LTRB), Decode4 tEnd`. gms_v95 inserts `Decode4 tStart` between rcArea and tEnd. The jms_v185 export notes explicitly say "NO leading tStart" — only gms_v95 carries it (the v1 PRD's "v95/jms" parenthetical was wrong; design §4.4(a)).

Removed read order is a single `Decode4 dwId` on **every** version — the codec (`affected_area_removed.go:35`) takes no context and is version-independent, so extending its fixture to v61/v72/v79 is three table rows, not new derivation.

### Stale-note caveats (do not act on these)

- The jms_v185 export entry's `notes` field still carries an old "STRUCTURAL DEFERRAL … matches NEITHER" remark from before the abs-RECT codec fix. The corresponding `_pending.md` entry no longer exists. The `calls` list is authoritative; do not edit the export.
- `docs/packets/audits/<ver>/FieldAffectedAreaCreated.json` rows carry trailing `Verdict: 2` ("atlas: extra") rows and `FlatInvalid: true`. That is the flat-alignment artifact of the codec writing rcArea as 4×`WriteInt32` while the client reads one `DecodeBuf(16)` — the semantic read order matches (all real rows Verdict 0). These JSONs are inputs, not deliverables; do not regenerate or edit them.
- `docs/packets/ida-exports/gms_v48.json` carries `{"unresolved": true, "comment": "function not found in IDB"}` for **both** mist functions. That is an unnamed-function artifact, **not** absence evidence (Task 8).

### Test-file starting state

`libs/atlas-packet/field/clientbound/affected_area_test.go` currently holds:

| Test | Covers | Evidence target for |
|---|---|---|
| `TestAffectedAreaCreatedWireShape` | length-only (39/43) for v79/v83/v87/jms/v95 | gms_v79 Created ← **the soft ✅** |
| `TestAffectedAreaCreatedByteOutputV72` | v72, per-field offsets | gms_v72 Created |
| `TestAffectedAreaCreatedByteOutputV61` | v61, per-field offsets | gms_v61 Created |
| `TestAffectedAreaCreatedFields` | RECT math, v83 + v95 | — |
| `TestAffectedAreaRemovedByteOutput` | full body pin, v83/84/87/95/jms | Removed ×5 |
| `TestAffectedAreaRemoved_EncodeShape` | smoke | — |

Task 1 consolidates the three Created tests into one table-driven byte-output test covering all eight Tier A versions.

---

### Task 1: Consolidate + extend the SPAWN_MIST byte-output test (8 versions)

**Files:**
- Modify: `libs/atlas-packet/field/clientbound/affected_area_test.go`

**Interfaces:**
- Consumes: `NewAffectedAreaCreated(mistId uuid.UUID, ownerId uint32, nType int32, skillId int32, skillLevel byte, phase int16, originX, originY, ltX, ltY, rbX, rbY int16, tStart, tEnd int32)` and `mistKey(id uuid.UUID) uint32` from `affected_area_created.go`. Helpers `pt.CreateContext(region, major, minor)` and `testlog.NewNullLogger()` as used by the sibling tests.
- Produces: test name `TestAffectedAreaCreatedByteOutput` with **eight** `packet-audit:verify` markers — Task 3's evidence records reference it and Task 4's matrix regen consumes the markers.

- [ ] **Step 1: Confirm every export read order before writing a byte**

```bash
for v in gms_v61 gms_v72 gms_v79 gms_v83 gms_v84 gms_v87 gms_v95 gms_jms_185; do
  echo "=== $v"
  python3 -c "
import json,sys
d=json.load(open('docs/packets/ida-exports/$v.json'))['functions']['CAffectedAreaPool::OnAffectedAreaCreated']
print(' address:', d.get('address'))
for c in d.get('calls',[]): print('  ', c.get('op'), c.get('comment','')[:70])
"
done
```

Expected: 8 decode calls for every version except `gms_v95`, which has 9 (the extra `Decode4 tStart` between rcArea and tEnd). Addresses match the Reference-facts table. **If any export disagrees with the table or with the codec's field order, STOP — that is the design §7 escalation, not a fix-it-yourself.**

- [ ] **Step 2: Hand-derive the expected bytes**

Fixture inputs (chosen so every byte is distinct and hand-checkable):
- `mistId = uuid.MustParse("00010203-0000-0000-0000-000000000000")` → `mistKey` = `uuid.ID()` = first 4 UUID bytes big-endian = `0x00010203` (same convention as `TestAffectedAreaRemovedByteOutput`)
- `ownerId = 42`, `nType = 7`, `skillId = 2121006` (= `0x205D2E`), `skillLevel = 20`, `phase = 3`
- `origin (100, 200)`, `lt (-50, -30)`, `rb (50, 30)` → absolute RECT L=50 (`0x32`), T=170 (`0xAA`), R=150 (`0x96`), B=230 (`0xE6`)
- `tStart = 1234` (= `0x04D2`), `tEnd = 10000` (= `0x2710`)

Expected little-endian body (39 bytes; gms_v95 inserts the 4 tStart bytes where marked):

```text
03 02 01 00   dwId      = mistKey = 0x00010203
07 00 00 00   nType     = 7
2A 00 00 00   dwOwnerId = 42
2E 5D 20 00   nSkillID  = 2121006
14            nSLV      = 20
03 00         phase     = 3
32 00 00 00   rcArea.left   = 100 + (-50) = 50
AA 00 00 00   rcArea.top    = 200 + (-30) = 170
96 00 00 00   rcArea.right  = 100 + 50   = 150
E6 00 00 00   rcArea.bottom = 200 + 30   = 230
[D2 04 00 00  tStart = 1234 — gms_v95 ONLY]
10 27 00 00   tEnd   = 10000
```

- [ ] **Step 3: Replace the three Created tests with one table-driven test**

Delete `TestAffectedAreaCreatedByteOutputV72` and `TestAffectedAreaCreatedByteOutputV61` (their coverage is subsumed, and leaving them would mean three conventions in one file). **Keep** `TestAffectedAreaCreatedWireShape` and `TestAffectedAreaCreatedFields` — they still guard the RECT math and the version gate — but **move the `packet-audit:verify` marker off `WireShape`** (it is the length-only soft-✅; design §4.4(c)). `WireShape` must end up with zero verify markers.

Insert the following in place of the removed tests:

```go
// TestAffectedAreaCreatedByteOutput pins the full wire body of
// CAffectedAreaPool::OnAffectedAreaCreated (SPAWN_MIST) against the client
// read order per version (docs/packets/ida-exports/):
//
//	Decode4 dwId, Decode4 nType, Decode4 dwOwnerId, Decode4 nSkillID,
//	Decode1 nSLV, Decode2 phase, DecodeBuf(16) rcArea (4×int32 absolute LTRB),
//	[Decode4 tStart — gms_v95 only], Decode4 tEnd.
//
// Atlas encodes rcArea as origin+offset absolute LTRB (4×WriteInt32) matching
// the client's single DecodeBuf(16). Wire body: 39 bytes (43 on gms_v95).
// The tStart gate is Region()=="GMS" && MajorVersion()>=95, so JMS185 does NOT
// carry it (the jms export notes confirm: "NO leading tStart").
//
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=gms_v61 ida=0x423edc
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=gms_v72 ida=0x42e36c
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=gms_v79 ida=0x42e7fc
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=gms_v83 ida=0x431a63
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=gms_v84 ida=0x4326ca
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=gms_v87 ida=0x432f3f
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=gms_v95 ida=0x437ec0
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaCreated version=jms_v185 ida=0x436572
func TestAffectedAreaCreatedByteOutput(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	// mistId chosen so uuid.ID() (time_low, first 4 UUID bytes big-endian) is a
	// known value: bytes 00 01 02 03 → 0x00010203 (same trick as the Removed test).
	mistId := uuid.MustParse("00010203-0000-0000-0000-000000000000")
	wantKey := mistKey(mistId) // == 0x00010203

	// Common prefix: dwId nType dwOwnerId nSkillID nSLV phase rcArea (abs LTRB).
	prefix := []byte{
		byte(wantKey), byte(wantKey >> 8), byte(wantKey >> 16), byte(wantKey >> 24), // dwId LE
		0x07, 0x00, 0x00, 0x00, // nType = 7
		0x2A, 0x00, 0x00, 0x00, // dwOwnerId = 42
		0x2E, 0x5D, 0x20, 0x00, // nSkillID = 2121006
		0x14,       // nSLV = 20
		0x03, 0x00, // phase = 3
		0x32, 0x00, 0x00, 0x00, // rcArea.left   = 100-50  = 50
		0xAA, 0x00, 0x00, 0x00, // rcArea.top    = 200-30  = 170
		0x96, 0x00, 0x00, 0x00, // rcArea.right  = 100+50  = 150
		0xE6, 0x00, 0x00, 0x00, // rcArea.bottom = 200+30  = 230
	}
	tStart := []byte{0xD2, 0x04, 0x00, 0x00} // tStart = 1234 (gms_v95 only)
	tEnd := []byte{0x10, 0x27, 0x00, 0x00}   // tEnd = 10000

	for _, v := range []struct {
		Name, Region string
		Major, Minor uint16
		HasTStart    bool
	}{
		{"GMS v61", "GMS", 61, 1, false},
		{"GMS v72", "GMS", 72, 1, false},
		{"GMS v79", "GMS", 79, 1, false},
		{"GMS v83", "GMS", 83, 1, false},
		{"GMS v84", "GMS", 84, 1, false},
		{"GMS v87", "GMS", 87, 1, false},
		{"GMS v95", "GMS", 95, 1, true},
		{"JMS v185", "JMS", 185, 1, false},
	} {
		t.Run(v.Name, func(t *testing.T) {
			want := append([]byte{}, prefix...)
			if v.HasTStart {
				want = append(want, tStart...)
			}
			want = append(want, tEnd...)

			in := NewAffectedAreaCreated(mistId /*ownerId*/, 42 /*nType*/, 7,
				/*skillId*/ 2121006 /*skillLevel*/, 20 /*phase*/, 3,
				/*originX*/ 100 /*originY*/, 200,
				/*ltX*/ -50 /*ltY*/, -30 /*rbX*/, 50 /*rbY*/, 30,
				/*tStart*/ 1234 /*tEnd*/, 10000)
			got := in.Encode(l, pt.CreateContext(v.Region, v.Major, v.Minor))(nil)
			require.Equal(t, want, got, "%s wire bytes", v.Name)
		})
	}
}
```

- [ ] **Step 4: Run the test — must pass without touching the codec**

```bash
cd libs/atlas-packet && go test -race -run 'TestAffectedAreaCreated' ./field/clientbound/ -v && cd -
```

Expected: `TestAffectedAreaCreatedByteOutput` PASS for all eight subtests; `TestAffectedAreaCreatedWireShape` and `TestAffectedAreaCreatedFields` still PASS. **If any subtest fails, do NOT edit the codec** — recheck the Step 2 derivation first; a genuine codec/client divergence on a ✅ version is the design §7 STOP condition.

- [ ] **Step 5: Confirm no stray verify markers remain on the retired tests**

```bash
grep -n "packet-audit:verify.*FieldAffectedAreaCreated" libs/atlas-packet/field/clientbound/affected_area_test.go
```

Expected: exactly 8 lines, all inside the `TestAffectedAreaCreatedByteOutput` doc comment. If `TestAffectedAreaCreatedWireShape` still carries the `gms_v79` marker, remove it — leaving it means two tests claim the same cell and the v79 soft-✅ survives.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-packet/field/clientbound/affected_area_test.go
git commit -m "test(atlas-packet): pin SPAWN_MIST wire body across all eight versions"
```

---

### Task 2: Extend the REMOVE_MIST byte-output test to v61/v72/v79

**Files:**
- Modify: `libs/atlas-packet/field/clientbound/affected_area_test.go`

**Interfaces:**
- Consumes: `NewAffectedAreaRemoved(mistId uuid.UUID, ownerId uint32)`. The codec takes no context (`affected_area_removed.go:35` — `Encode(l, _ context.Context)`), so the body is a single LE uint32 on every version.
- Produces: three additional `packet-audit:verify` markers on `TestAffectedAreaRemovedByteOutput` (total 8) — Task 3 pins matching evidence.

- [ ] **Step 1: Confirm the v61/v72/v79 read order is the same single Decode4**

```bash
for v in gms_v61 gms_v72 gms_v79; do
  echo "=== $v"
  python3 -c "
import json
d=json.load(open('docs/packets/ida-exports/$v.json'))['functions']['CAffectedAreaPool::OnAffectedAreaRemoved']
print(' address:', d.get('address'))
for c in d.get('calls',[]): print('  ', c.get('op'), c.get('comment','')[:70])
"
done
```

Expected: addresses `0x4246b0` / `0x42ec4e` / `0x42f0de`, each with a single `Decode4` — matching the v83+ pattern already pinned. **If any version reads more than the one uint32, STOP** — that is a codec divergence on a version the codec currently treats as identical, and it is a design §7 escalation.

- [ ] **Step 2: Add the three markers and three table rows**

In the `TestAffectedAreaRemovedByteOutput` doc comment, add above the existing five markers:

```go
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaRemoved version=gms_v61 ida=0x4246b0
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaRemoved version=gms_v72 ida=0x42ec4e
// packet-audit:verify packet=field/clientbound/FieldAffectedAreaRemoved version=gms_v79 ida=0x42f0de
```

and add the matching rows at the top of the subtest table:

```go
		{"GMS v61", "GMS", 61, 1},
		{"GMS v72", "GMS", 72, 1},
		{"GMS v79", "GMS", 79, 1},
```

Also extend the doc comment's per-version read-order list with the three new addresses, matching the existing `v83 @0x43234d: …` style.

- [ ] **Step 3: Run and verify marker count**

```bash
cd libs/atlas-packet && go test -race -run 'TestAffectedAreaRemoved' ./field/clientbound/ -v && cd -
grep -c "packet-audit:verify.*FieldAffectedAreaRemoved" libs/atlas-packet/field/clientbound/affected_area_test.go
```

Expected: 8 subtests PASS; marker count `8`.

- [ ] **Step 4: Run the whole package**

```bash
cd libs/atlas-packet && go test -race ./field/... && cd -
```

Expected: `ok` for `field/clientbound` and all other field packages.

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/field/clientbound/affected_area_test.go
git commit -m "test(atlas-packet): pin REMOVE_MIST wire body on v61/v72/v79"
```

---

### Task 3: Pin and re-point evidence records

**Files:**
- Create: `docs/packets/evidence/{gms_v83,gms_v84,gms_v87,gms_v95,jms_v185}/field.clientbound.FieldAffectedAreaCreated.yaml` (5 new)
- Create: `docs/packets/evidence/{gms_v61,gms_v72,gms_v79}/field.clientbound.FieldAffectedAreaRemoved.yaml` (3 new)
- Modify: `docs/packets/evidence/{gms_v61,gms_v72,gms_v79}/field.clientbound.FieldAffectedAreaCreated.yaml` (re-point `verifies:` at the consolidated test — **v79 is the soundness fix**, v61/v72 are renames following Task 1's consolidation)

**Interfaces:**
- Consumes: the `packet-audit evidence pin` subcommand (VERIFYING_A_PACKET.md §7); Task 1's `TestAffectedAreaCreatedByteOutput` and Task 2's `TestAffectedAreaRemovedByteOutput`.
- Produces: sixteen consistent TIER1-FIXTURE records (8 Created + 8 Removed) that Task 4's matrix regen consumes.

- [ ] **Step 1: Pin the five new Created records**

```bash
for v in gms_v83 gms_v84 gms_v87 gms_v95 jms_v185; do
  go run ./tools/packet-audit evidence pin \
    --packet field/clientbound/FieldAffectedAreaCreated \
    --version "$v" \
    --ida "CAffectedAreaPool::OnAffectedAreaCreated" \
    --category TIER1-FIXTURE
done
```

Expected: five YAMLs created, each with `ida.address` matching the Reference-facts table and a tool-computed `decompile_sha256`. **If the tool cannot resolve the function or address for any version, STOP and ask (unresolved-fname rule) — never substitute an address or hash.**

- [ ] **Step 2: Pin the three new Removed records**

```bash
for v in gms_v61 gms_v72 gms_v79; do
  go run ./tools/packet-audit evidence pin \
    --packet field/clientbound/FieldAffectedAreaRemoved \
    --version "$v" \
    --ida "CAffectedAreaPool::OnAffectedAreaRemoved" \
    --category TIER1-FIXTURE
done
```

Expected: three YAMLs with addresses `0x4246b0` / `0x42ec4e` / `0x42f0de`.

- [ ] **Step 3: Add the `verifies:` field to all eight new records**

The pin tool does not write this field — mirror `docs/packets/evidence/gms_v83/field.clientbound.FieldAffectedAreaRemoved.yaml`:

```yaml
verifies:
    - libs/atlas-packet/field/clientbound/affected_area_test.go#TestAffectedAreaCreatedByteOutput
```

(…and `#TestAffectedAreaRemovedByteOutput` for the three Removed records.)

- [ ] **Step 4: Re-point the three existing Created records — the v79 soundness fix**

Edit the `verifies:` line in each:

| File | Was | Becomes |
|---|---|---|
| `evidence/gms_v61/field.clientbound.FieldAffectedAreaCreated.yaml` | `#TestAffectedAreaCreatedByteOutputV61` | `#TestAffectedAreaCreatedByteOutput` |
| `evidence/gms_v72/field.clientbound.FieldAffectedAreaCreated.yaml` | `#TestAffectedAreaCreatedByteOutputV72` | `#TestAffectedAreaCreatedByteOutput` |
| `evidence/gms_v79/field.clientbound.FieldAffectedAreaCreated.yaml` | `#TestAffectedAreaCreatedWireShape` | `#TestAffectedAreaCreatedByteOutput` |

Leave `ida.address` and `decompile_sha256` untouched in all three — the decompile did not change, only which test pins it. The v79 row is the whole point: it moves that cell from a length assertion to a full-body pin.

- [ ] **Step 5: Verify shape consistency across all sixteen records**

```bash
grep -L "verifies:" docs/packets/evidence/*/field.clientbound.FieldAffectedArea*.yaml; echo "--- (want: no output)"
grep -h "verifies:" -A1 docs/packets/evidence/*/field.clientbound.FieldAffectedArea*.yaml | grep -c "ByteOutput"
grep -rn "ByteOutputV61\|ByteOutputV72\|WireShape" docs/packets/evidence/; echo "--- (want: no output)"
```

Expected: first command prints nothing (every record has `verifies:`); the count is `16`; the third prints nothing (no record still points at a retired or length-only test).

- [ ] **Step 6: Commit**

```bash
git add docs/packets/evidence/
git commit -m "docs(packets): pin mist TIER1-FIXTURE evidence across all eight versions"
```

---

### Task 4: Regenerate the coverage matrix

**Files:**
- Modify (generated): `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json` (tool-owned; commit whatever the regen touches under `docs/packets/audits/`)

**Interfaces:**
- Consumes: Task 1's 8 Created markers, Task 2's 8 Removed markers, Task 3's 16 evidence records.
- Produces: both mist rows fully ✅ across the eight Tier A columns — the acceptance artifact for PRD FR-3.

- [ ] **Step 1: Regenerate and check**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check; echo "exit=$?"
```

Expected: `exit=0` (baseline was exit 0; the bar stays a clean 0 — no orphan/dangling/stale/drift lines mentioning `FieldAffectedArea`).

- [ ] **Step 2: Verify both rows**

```bash
grep -n "SPAWN_MIST\|REMOVE_MIST" docs/packets/audits/STATUS.md
```

Expected: SPAWN_MIST shows `⬜` for v48 and `✅` for v61/v72/v79/v83/v84/v87/v95/JMS185. REMOVE_MIST shows `⬜` for v48 and `✅` for all eight — the three 🟡ᶠ glyphs are gone. If a cell did not flip, read the tool's check output (marker/evidence mismatch on the exact `packet=`/`version=` strings is the usual cause) and fix the marker or record — not the tool.

- [ ] **Step 3: Full lib suite (markers are comments — this is a no-op guard)**

```bash
cd libs/atlas-packet && go test -race ./... && cd -
```

Expected: all packages `ok`.

- [ ] **Step 4: Commit**

```bash
git add docs/packets/audits/
git commit -m "docs(packets): both mist ops verified across all eight versions (matrix regen)"
```

---

### Task 5: Declare the writers in atlas-channel

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/main.go`

**Interfaces:**
- Consumes: `fieldcb.AffectedAreaCreatedWriter` / `fieldcb.AffectedAreaRemovedWriter` (string constants in `libs/atlas-packet/field/clientbound/affected_area_created.go:13` and `affected_area_removed.go:13`). The `fieldcb` import alias already exists in `main.go`.
- Produces: the declared-writer names that `BuildWriterProducer` (`libs/atlas-opcodes/producer.go`) intersects with the tenant config from Task 6. The mist consumer (`kafka/consumer/mist/consumer.go:64,72`) already announces by these names — no consumer change.

- [ ] **Step 1: Add the two writer declarations**

In `produceWriters()` (`services/atlas-channel/atlas.com/channel/main.go:608`), immediately after the line `fieldcb.KiteDestroyWriter,` (line 718 — kites are the adjacent-opcode neighbours below the mist pair), insert:

```go
		fieldcb.AffectedAreaCreatedWriter,
		fieldcb.AffectedAreaRemovedWriter,
```

- [ ] **Step 2: Verify the service builds and tests pass**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test -race ./... && cd -
```

Expected: all clean. (Inner-loop check; the full mandatory gate incl. bake runs in Task 9.)

- [ ] **Step 3: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/main.go
git commit -m "fix(atlas-channel): declare AffectedAreaCreated/Removed writers"
```

---

### Task 6: Wire the writers into the eight Tier A seed templates

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_{61,72,79,83,84,87,95}_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`

**Interfaces:**
- Consumes: the Tier A opcode table from Reference facts. Entry shape used throughout `socket.writers`: `{"opCode": "0xNNN", "writer": "<Name>"}` — no `options` key (the Encode closures ignore options).
- Produces: tenant-side name→opcode mappings that complete the `BuildWriterProducer` intersection with Task 5's declarations. Task 10's live-tenant patch reuses these exact per-version entries.

- [ ] **Step 1: Insert two entries per template, immediately before its `SpawnDoor` entry**

In each file, find the `socket.writers` entry whose writer is `SpawnDoor` and insert the two entries before it, matching the file's existing formatting exactly (copy a neighbouring line and edit it — quote style and indentation vary by file):

| File | Insert |
|---|---|
| `template_gms_61_1.json` | `{"opCode": "0x0D2", "writer": "AffectedAreaCreated"}`, `{"opCode": "0x0D3", "writer": "AffectedAreaRemoved"}` |
| `template_gms_72_1.json` | `{"opCode": "0x0F3", ...}`, `{"opCode": "0x0F4", ...}` |
| `template_gms_79_1.json` | `{"opCode": "0x0FB", ...}`, `{"opCode": "0x0FC", ...}` |
| `template_gms_83_1.json` | `{"opCode": "0x111", ...}`, `{"opCode": "0x112", ...}` |
| `template_gms_84_1.json` | `{"opCode": "0x118", ...}`, `{"opCode": "0x119", ...}` |
| `template_gms_87_1.json` | `{"opCode": "0x122", ...}`, `{"opCode": "0x123", ...}` |
| `template_gms_95_1.json` | `{"opCode": "0x148", ...}`, `{"opCode": "0x149", ...}` |
| `template_jms_185_1.json` | `{"opCode": "0x126", ...}`, `{"opCode": "0x127", ...}` |

Match each file's existing opcode-literal width convention (some files write `0x0D2`, others `0xD2`) — copy the neighbouring `SpawnDoor` entry's style. The order guard parses the value, not the width, but consistency keeps diffs clean.

- [ ] **Step 2: Verify — one occurrence per name per template, valid JSON, correct opcodes**

```bash
python3 -c "
import json
want={'gms_61':(0x0D2,0x0D3),'gms_72':(0x0F3,0x0F4),'gms_79':(0x0FB,0x0FC),
      'gms_83':(0x111,0x112),'gms_84':(0x118,0x119),'gms_87':(0x122,0x123),
      'gms_95':(0x148,0x149),'jms_185':(0x126,0x127)}
ok=True
for v,(c,r) in want.items():
    f='services/atlas-configurations/seed-data/templates/template_%s_1.json'%v
    d=json.load(open(f))
    def find(o):
        if isinstance(o,dict):
            if 'writers' in o: return o['writers']
            for x in o.values():
                q=find(x)
                if q is not None: return q
        if isinstance(o,list):
            for x in o:
                q=find(x)
                if q is not None: return q
    w=find(d)
    got={e['writer']:int(e['opCode'],16) for e in w if 'AffectedArea' in e['writer']}
    exp={'AffectedAreaCreated':c,'AffectedAreaRemoved':r}
    status='OK' if got==exp else 'MISMATCH'
    if got!=exp: ok=False
    print(v, status, {k:hex(x) for k,x in got.items()})
print('ALL OK' if ok else 'FAILURES PRESENT')
"
```

Expected: `OK` for all eight and `ALL OK`. This also proves the JSON parses (a syntax error raises) and that no name appears twice (dict collapse would show a wrong opcode).

- [ ] **Step 3: Run the template guards (ascending-order rule is CI-enforced)**

```bash
tools/template-opcode-order-guard.sh; echo "order guard exit=$?"
tools/template-movement-types-guard.sh; echo "movement guard exit=$?"
```

Expected: both `exit=0`. If the order guard fails it prints the exact `0xNN (writer) follows 0xMM (writer)` pair — move the entry to its sorted position.

- [ ] **Step 4: Run the seeder tests (they parse these JSON files from disk)**

```bash
cd services/atlas-configurations/atlas.com/configurations && go test -race ./... && cd -
```

Expected: all `ok`.

- [ ] **Step 5: Re-run the matrix check (templates are a matrix input)**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check; echo "exit=$?"
git status --porcelain docs/packets/audits/
```

Expected: `exit=0`. If the regen changed anything under `docs/packets/audits/` (route-state columns), include it in this task's commit.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/ docs/packets/audits/
git commit -m "fix(atlas-configurations): register AffectedArea writers in eight seed templates"
```

---

### Task 7: Tier A checkpoint — the task is shippable from here

**Files:** none (checkpoint only)

**Interfaces:**
- Consumes: Tasks 1–6.
- Produces: an explicit go/no-go boundary. Everything after Task 8 is additive; if Tier B stalls, the branch still delivers mists on eight of eleven versions.

- [ ] **Step 1: Confirm the Tier A end state**

```bash
grep -n "SPAWN_MIST\|REMOVE_MIST" docs/packets/audits/STATUS.md
grep -c "AffectedArea" services/atlas-configurations/seed-data/templates/*.json
grep -n "AffectedArea" services/atlas-channel/atlas.com/channel/main.go
```

Expected: both matrix rows ✅ on all eight Tier A columns (v48 still ⬜); `grep -c` returns `2` for the eight Tier A templates and `0` for `template_gms_12_1.json`, `template_gms_48_1.json`, `template_gms_92_1.json`; two writer lines in `main.go`.

- [ ] **Step 2: No commit** — checkpoint only.

---

### Task 8: Tier B opcode discovery (gms_48, gms_92, gms_12)

**Files:**
- Create: `docs/tasks/task-165-mist-writer-template-wiring/discovery.md`
- Modify (conditional, Outcome A): `services/atlas-configurations/seed-data/templates/template_gms_{48,92,12}_1.json`, `docs/packets/registry/gms_v48.yaml`, `docs/packets/ida-exports/gms_v48.json`, `libs/atlas-packet/field/clientbound/affected_area_test.go`, `docs/packets/evidence/gms_v48/`
- Modify (conditional, Outcome B for gms_48): `docs/packets/feature-na-evidence.yaml`

**Interfaces:**
- Consumes: ida-pro-mcp sessions. Confirm the live session id per version with `idb_list` and **match on binary NAME**, never a remembered port — the instance set rotates.
- Produces: a recorded outcome per version (A / B / C) in `discovery.md`, plus conditional wiring.

> **This task is deliberately last-but-one and independently abortable.** Each
> version terminates in one of three recorded outcomes. Outcome C (inconclusive)
> is an acceptable, documented result — falling back to a derived/interpolated
> opcode is **not** (design §4.3, Global Constraints).

- [ ] **Step 1: Per version, locate the affected-area pool dispatcher**

Work one version at a time; do not run two IDA write sessions in parallel (MCP serializes writes).

- **gms_48** — anchor: v61's `CAffectedAreaPool::OnPacket` @0x423eb7 is a two-arm dispatcher (`a2==210` → Created, `a2==211` → Removed). Decompile it, then locate the v48 analogue by positional correlation off the already-named v61 pool dispatchers (`CTownPortalPool::*` is named in v48 and is a sibling anchor). v48's CField base is Δ≈−20 vs v61 and the shift is non-uniform — **read** the case constants, never compute them.
- **gms_92** — anchor: `CField::OnPacket` v92=0x5406b0 / v95=0x546d50. Align arms by already-named neighbours (the documented method for this IDB). Opcode shift v92↔v95 is irregular. **Disasm is authoritative over Hex-Rays** for SEH functions.
- **gms_12** — no session is currently open; open the v12 IDB first (`idb_open`, path per the v12 naming record). Stage dispatcher `CField::OnPacket`@0x47502d. v12 drift vs v48 is non-uniform — match by class + payload fingerprint, never by opcode number.

Name the functions in the IDB as you resolve them (`CAffectedAreaPool::OnAffectedAreaCreated` / `::OnAffectedAreaRemoved`, using the v95 mangled-symbol convention) so the next task does not repeat this walk.

- [ ] **Step 2: Record the outcome per version in `discovery.md`**

Write `docs/tasks/task-165-mist-writer-template-wiring/discovery.md` with one section per version, each stating:

- What was walked (dispatcher address, arms enumerated, correlation basis).
- The outcome: **A** (opcodes found — list them with the dispatcher address and case constants), **B** (proven absent — with evidence meeting the `feature-na-evidence.yaml` bar), or **C** (inconclusive — with exactly what remains to walk).
- For Outcome A: the derived read order, and whether it matches the current codec.

A failed name lookup is **not** an Outcome B. `docs/packets/ida-exports/gms_v48.json` already contains exactly that failed lookup (`unresolved: true`), and it is what made this task think v48 was n-a.

- [ ] **Step 3 (Outcome A only): verify the read order against the codec**

For each discovered version, derive the client read order and compare against `affected_area_created.go` / `affected_area_removed.go`. If it matches, add a subtest row + `packet-audit:verify` marker to the Task 1/Task 2 tests and pin evidence exactly as Task 3 did. If it diverges, add an **additive** version gate using the `MajorAtLeast` idiom — and re-run Task 1's and Task 2's tests to prove **no byte changed on any already-✅ version**. A byte change on a ✅ version is a STOP.

- [ ] **Step 4 (Outcome A only): registry + export + template**

- gms_48: add both ops to `docs/packets/registry/gms_v48.yaml` (with `provenance: ida-discovered` and a `note` naming the dispatcher address); re-export so `gms_v48.json` no longer carries `unresolved: true`.
- gms_92 / gms_12: no registry exists and standing one up is out of scope — record the opcode in `discovery.md` as the provenance.
- Wire the template(s) exactly as Task 6 (ascending order, before the neighbouring higher-opcode entry), then re-run both template guards.

- [ ] **Step 5 (Outcome B, gms_48 only): record n-a evidence**

Add an entry to `docs/packets/feature-na-evidence.yaml` for `SPAWN_MIST` and `REMOVE_MIST` × `gms_v48`, meeting that file's stated bar (the opcode slot is a different op / a binary-wide search for the op's construction found nothing / the receive handler lacks the feature branch). gms_92 and gms_12 have no matrix column, so their absence record lives in `discovery.md` only.

- [ ] **Step 6: Re-run the matrix check and guards**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check; echo "exit=$?"
tools/template-opcode-order-guard.sh; echo "order guard exit=$?"
```

Expected: both `exit=0`.

- [ ] **Step 7: Commit**

```bash
git add docs/tasks/task-165-mist-writer-template-wiring/discovery.md \
        docs/packets/ services/atlas-configurations/seed-data/templates/ \
        libs/atlas-packet/field/clientbound/affected_area_test.go
git commit -m "feat(task-165): Tier B mist opcode discovery for v48/v92/v12"
```

(Adjust the staged paths to whatever the outcomes actually touched — an all-Outcome-C run commits only `discovery.md`.)

---

### Task 9: Full verification gate

**Files:** none (verification only; fix-and-recommit into the responsible task's files if anything fails)

**Interfaces:**
- Consumes: everything from Tasks 1–8.
- Produces: the CLAUDE.md "done" bar for the branch — required before code review / PR.

- [ ] **Step 1: Test + vet every changed module**

```bash
cd libs/atlas-packet && go test -race ./... && go vet ./... && cd -
cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./... && cd -
cd services/atlas-configurations/atlas.com/configurations && go test -race ./... && go vet ./... && cd -
```

Expected: all clean.

- [ ] **Step 2: Repo guards**

```bash
tools/redis-key-guard.sh; echo "redis exit=$?"
tools/template-opcode-order-guard.sh; echo "order exit=$?"
tools/template-movement-types-guard.sh; echo "movement exit=$?"
```

Expected: all `exit=0`.

- [ ] **Step 3: Docker bake for atlas-channel (mandatory — `go build` does not catch Dockerfile COPY gaps)**

```bash
docker buildx bake atlas-channel
```

Expected: image builds successfully. (Only atlas-channel's Go module was touched; atlas-configurations changed JSON only and libs/atlas-packet changed a test file, so `atlas-channel` is the one mandatory bake target.)

- [ ] **Step 4: Final matrix check + full-branch sanity**

```bash
go run ./tools/packet-audit matrix --check; echo "exit=$?"
git status --porcelain
```

Expected: `exit=0`; working tree clean (everything committed).

- [ ] **Step 5: No commit** — this task produces no changes; if any gate failed, fix within the owning task's scope, re-run this whole task from Step 1, and commit the fix with a `fix:` message referencing the gate.

---

### Task 10: Rollout procedure document (`rollout.md`)

**Files:**
- Create: `docs/tasks/task-165-mist-writer-template-wiring/rollout.md`

**Interfaces:**
- Consumes: the **actually wired** version set (Task 6 + any Task 8 Outcome A) and the atlas-configurations REST surface (`GET /configurations/tenants` list, `GET`/`PATCH /configurations/tenants/{tenantId}`; JSON:API type `"tenants"`; PATCH goes through `RegisterInputHandler`, so the JSON:API envelope is mandatory).
- Produces: the reproducible procedure Task 11 executes per environment, and the per-environment record it fills in.

> Write this **after** Task 8 so the per-version table reflects what was really
> wired rather than what was hoped for.

- [ ] **Step 1: Write the document**

Write `docs/tasks/task-165-mist-writer-template-wiring/rollout.md` covering:

1. **Why** — seed templates apply only at tenant creation (known pattern: new opcodes missing from live tenant config → packet silently dropped), so existing tenants must be patched directly and atlas-channel restarted (the configuration projection does not hot-reload writer maps).
2. **Per-version entries** — a table generated from the wired set, one row per version, giving both `{"opCode": "0xNNN", "writer": "..."}` entries verbatim. Explicitly list any version left unwired by Task 8 and why, so an operator does not go looking for it.
3. **Procedure** — enumerate all tenant configurations (`GET <configurations-service-base>/api/configurations/tenants`), select every tenant whose (region, majorVersion) is in the table (full sweep, not spot-check); per tenant read-modify-write with an idempotency guard (skip if `"AffectedAreaCreated"` already present) and PATCH with the JSON:API envelope `{"data": {"type": "tenants", "id": "<tenantId>", "attributes": {…}}}`; restart atlas-channel (`kubectl -n <namespace> rollout restart deployment atlas-channel` + `rollout status`); verify per tenant that the startup "no opcode mapping" warning no longer lists either writer and that no `writer not found` / `Unable to broadcast AffectedArea` lines appear after mist activity.
4. **Environment record** — an empty table with columns: Environment | Tenant id | Region/version | Patched (date) | Restarted | Verified. Placeholders in angle brackets stay as placeholders in the committed file.

- [ ] **Step 2: Placeholder check — no literal hostnames, namespaces, or home paths**

```bash
grep -nE "/home/|/Users/|https?://[a-z0-9.-]+\.(com|net|io|dev)" docs/tasks/task-165-mist-writer-template-wiring/rollout.md; echo "exit=$? (want 1 = no matches)"
```

Expected: no matches (exit 1).

- [ ] **Step 3: Commit**

```bash
git add docs/tasks/task-165-mist-writer-template-wiring/rollout.md
git commit -m "docs(task-165): live-tenant rollout procedure"
```

---

### Task 11: Code review, PR, live rollout, and end-to-end verification

**Files:**
- Modify: `docs/tasks/task-165-mist-writer-template-wiring/rollout.md` (fill in the environment record)

**Interfaces:**
- Consumes: the completed branch (Tasks 1–10); the rollout procedure (Task 10); a deploy environment via the `deploy-env` PR label flow.
- Produces: PRD acceptance criteria — live tenants patched + restarted, mists visibly rendering, clean logs.

> **Environment note:** Steps 3–6 require a running deploy environment and a
> game client; they cannot be completed inside the worktree alone. Do not mark
> them done on the strength of code inspection — each has an observable
> expected result. If the deploy environment is unavailable, stop after Step 2
> and report BLOCKED on the environment (not "done").

- [ ] **Step 1: Code review (mandatory before PR)**

Invoke `superpowers:requesting-code-review` for this branch (Go changes → it dispatches `plan-adherence-reviewer` + `backend-guidelines-reviewer`; the packet surface also warrants `packet-completeness-critic`). Address findings; findings land in `docs/tasks/task-165-mist-writer-template-wiring/audit.md`.

- [ ] **Step 2: Open the PR with the deploy-env label**

```bash
git push -u origin task-165-mist-writer-template-wiring
env -u GH_TOKEN -u GITHUB_TOKEN gh pr create \
  --title "fix(mist): wire AffectedArea writers across all supported versions" \
  --body-file - <<'EOF'
Mist broadcasts were silently dropped for every tenant on every version:
neither AffectedArea writer was declared in atlas-channel's produceWriters()
nor registered in any of the eleven seed templates, so session.Announce failed
with "writer not found".

- Declare AffectedAreaCreated/Removed in atlas-channel produceWriters()
- Register both writers (per-version opcodes) in the eight seed templates with
  an IDB-sourced opcode
- SPAWN_MIST ❌→✅ on v83/v84/v87/v95/jms185; REMOVE_MIST 🟡ᶠ→✅ on v61/v72/v79
- Re-pin the v79 SPAWN_MIST cell: its evidence pointed at a length-only test
- Tier B discovery outcomes for v48/v92/v12: docs/tasks/task-165-mist-writer-template-wiring/discovery.md
- Rollout procedure: docs/tasks/task-165-mist-writer-template-wiring/rollout.md

Task: docs/tasks/task-165-mist-writer-template-wiring/

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
env -u GH_TOKEN -u GITHUB_TOKEN gh pr edit --add-label deploy-env
```

Expected: PR created; the `deploy-env` label triggers the ephemeral build/deploy.

- [ ] **Step 3: Execute the rollout procedure in the deploy environment**

Follow `rollout.md` exactly: enumerate ALL tenant configurations, patch every tenant on every wired version (idempotency guard honored), restart atlas-channel, and fill in the environment-record table. Full sweep — the record must list every tenant on those versions, including "already patched" rows.

- [ ] **Step 4: Verify the deployed binary and config before testing (first-diagnostic rule)**

- Confirm the running atlas-channel image is built from this branch (check the pod image tag against the PR build).
- `GET .../api/configurations/tenants/<tenantId>` for one patched tenant per version band and confirm both writer entries are present with the right opcodes.
- Check atlas-channel startup logs: neither AffectedArea writer appears in the "no opcode mapping" warning for patched tenants.

- [ ] **Step 5: End-to-end — mob mist, on one modern and one legacy version**

Pick a mob with an AREA_POISON mob skill **from local WZ data or repo source at execution time** (Verification Over Memory — search the tenant's mob-skill data for AREA_POISON/mist entries and pick a mob+map from that). Run it on a patched **v83+** tenant and a patched **v61/v72/v79** tenant — the legacy band is a different opcode range and an older template generation, so a modern-only pass does not evidence it. Expected: the poison cloud renders visibly and disappears when the mist expires; no `Unable to broadcast AffectedArea` / `writer not found` lines.

- [ ] **Step 6: End-to-end — player mist + observer**

Pick a player mist skill by the same WZ/repo-verification rule. **Skill ids are version-specific** — re-resolve the id against each tenant's own WZ data rather than reusing the modern-version id on the legacy tenant. Cast it with one client while a second client watches on the same map. Expected: the mist renders for both caster and observer; same clean-log check.

- [ ] **Step 7: Commit the completed environment record**

```bash
git add docs/tasks/task-165-mist-writer-template-wiring/rollout.md
git commit -m "docs(task-165): record deploy-env rollout and e2e verification"
git push
```

---

## Task → requirement traceability

| PRD requirement | Task |
|---|---|
| FR-1 Go writer declaration | Task 5 |
| FR-2 Seed template wiring ×8 (Tier A) | Task 6 |
| FR-2b Opcode discovery for gms_48/gms_92/gms_12 (Tier B) | Task 8 |
| FR-3 SPAWN_MIST ❌→✅ ×5 | Tasks 1, 3, 4 |
| FR-3 REMOVE_MIST 🟡ᶠ→✅ ×3 | Tasks 2, 3, 4 |
| FR-3 v79 SPAWN_MIST re-pin (soft ✅ → byte pin) | Tasks 1, 3, 4 |
| FR-4 Live tenant rollout | Tasks 10–11 |
| FR-5 End-to-end verification (modern + legacy) | Task 11 |
| Build/verification gate (PRD §7, CLAUDE.md) | Task 9 (+ inner-loop checks in 1, 2, 5, 6) |
| Code review before PR | Task 11 Step 1 |
| Tier A shippable checkpoint | Task 7 |
