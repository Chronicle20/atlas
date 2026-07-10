# Mist Broadcast Writer Wiring Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make mist (affected-area) broadcasts reach the client by declaring the `AffectedAreaCreated`/`AffectedAreaRemoved` writers in atlas-channel, registering their opcodes in the five full seed templates, promoting SPAWN_MIST to ✅ in the packet coverage matrix, and rolling the fix out to live tenants.

**Architecture:** The codecs, the mist consumer, and the producers already work; only the name→opcode resolution chain is broken (Go declaration + template config, per the design). Sequencing per design §5: fixture-verify SPAWN_MIST first (matrix ❌→✅ ×5), then the two wiring fixes, then build gates, then rollout docs + live patch + e2e.

**Tech Stack:** Go (atlas-channel, libs/atlas-packet), JSON seed templates (atlas-configurations), `tools/packet-audit` (evidence pin + matrix), atlas-configurations REST (JSON:API) for live rollout.

## Global Constraints

- **No codec changes**: `affected_area_created.go` / `affected_area_removed.go` are frozen. If any per-version read order does not match the current codec, STOP and escalate (design §6) — do not adjust the codec.
- **No consumer/producer changes**: the mist consumer, atlas-maps mist domain, and atlas-monsters producers are out of scope.
- **gms_92 / gms_12 reduced templates are out of scope** (no registry/IDB to source opcodes).
- Every fixture byte must trace to a decompile line in `docs/packets/ida-exports/` (CLAUDE.md "Verification Over Memory") — never to MapleStory knowledge.
- Opcodes live only in tenant config, never hard-coded in Go (DOM-25 posture).
- No literal home/absolute paths in committed files; repo-relative paths or placeholders only.
- Baseline verified 2026-07-10: `go run ./tools/packet-audit matrix --check` exits **0** on this branch before any change. The bar for every matrix step is a clean exit 0 (the conflict backlog is already burned down — the "no new problems" fallback in VERIFYING_A_PACKET.md §8 does not apply here).
- All commands run from the worktree root (`.worktrees/task-165-mist-writer-template-wiring`) unless a `cd` is shown.

## Reference facts (verified against source, 2026-07-10)

Per-version IDA addresses and read orders for `CAffectedAreaPool::OnAffectedAreaCreated`, from the checked-in exports (`docs/packets/ida-exports/<file>.json`, key `functions."CAffectedAreaPool::OnAffectedAreaCreated"`):

| Version key | Export file | Address | tStart? | Body bytes |
|---|---|---|---|---|
| gms_v83 | `gms_v83.json` | `0x431a63` | no | 39 |
| gms_v84 | `gms_v84.json` | `0x4326ca` | no | 39 |
| gms_v87 | `gms_v87.json` | `0x432f3f` | no | 39 |
| gms_v95 | `gms_v95.json` | `0x437ec0` | **yes** | 43 |
| jms_v185 | `gms_jms_185.json` | `0x436572` | no | 39 |

Common read order (v83/v84/v87/jms_v185): `Decode4 dwId, Decode4 nType, Decode4 dwOwnerId, Decode4 nSkillID, Decode1 nSLV, Decode2 phase, DecodeBuf(16) rcArea (4×int32 absolute LTRB), Decode4 tEnd`. gms_v95 inserts `Decode4 tStart` between rcArea and tEnd. The jms_v185 export notes explicitly say "NO leading tStart" — this pins the design §3.3 ruling (only gms_v95 carries tStart; the PRD FR-3 parenthetical saying "v95/jms" is corrected by the design and by the export).

Stale-note caveat: the jms_v185 export entry's `notes` field still carries an old "STRUCTURAL DEFERRAL … matches NEITHER" remark from before the abs-RECT codec fix. The corresponding `_pending.md` entry (`AFFECTEDAREA-create-shape`) no longer exists — the deferral was cleared. The `calls` list is authoritative; do not treat the stale note as a live deferral, and do not edit the export.

Known audit-record artifact: `docs/packets/audits/<ver>/FieldAffectedAreaCreated.json` rows carry trailing `Verdict: 2` ("atlas: extra") rows and `FlatInvalid: true`. That is the flat-alignment artifact of the codec writing rcArea as 4×`WriteInt32` while the client reads one `DecodeBuf(16)` — the semantic read order matches (all real rows Verdict 0). These JSONs are inputs, not deliverables; do not regenerate or edit them.

Writer opcodes (pinned by `docs/packets/audits/STATUS.md` rows SPAWN_MIST/REMOVE_MIST and the per-version registries):

| Template | AffectedAreaCreated | AffectedAreaRemoved |
|---|---|---|
| `template_gms_83_1.json` | `0x111` | `0x112` |
| `template_gms_84_1.json` | `0x118` | `0x119` |
| `template_gms_87_1.json` | `0x122` | `0x123` |
| `template_gms_95_1.json` | `0x148` | `0x149` |
| `template_jms_185_1.json` | `0x126` | `0x127` |

In every one of the five templates, `socket.writers` currently has no entry for either name, and the immediate next-higher-opcode entry is `SpawnDoor` (`0x113`/`0x11A`/`0x124`/`0x14A`/`0x128` respectively) followed by `RemoveDoor` — so both new entries slot immediately **before the `SpawnDoor` entry** in each file.

---

### Task 1: SPAWN_MIST byte-fixture test (`TestAffectedAreaCreatedByteOutput`)

**Files:**
- Modify: `libs/atlas-packet/field/clientbound/affected_area_test.go` (insert new test after `TestAffectedAreaCreatedFields`, i.e. before the `TestAffectedAreaRemovedByteOutput` block that starts around line 92)

**Interfaces:**
- Consumes: `NewAffectedAreaCreated(mistId uuid.UUID, ownerId uint32, nType int32, skillId int32, skillLevel byte, phase int16, originX, originY, ltX, ltY, rbX, rbY int16, tStart, tEnd int32) AffectedAreaCreated` and `mistKey(id uuid.UUID) uint32` — both already exist in `affected_area_created.go`. Test helpers `pt.CreateContext(region, major, minor)` and `testlog.NewNullLogger()` as used by the sibling tests in this file.
- Produces: test name `TestAffectedAreaCreatedByteOutput` with five `packet-audit:verify` markers — Task 2's evidence records reference it (`verifies:` line) and Task 3's matrix regen consumes the markers.

- [ ] **Step 1: Hand-derive the expected bytes from the export read order**

Open each export entry (`python3 -c "import json; print(json.dumps(json.load(open('docs/packets/ida-exports/gms_v83.json'))['functions']['CAffectedAreaPool::OnAffectedAreaCreated'], indent=1))"` and likewise for `gms_v84.json`, `gms_v87.json`, `gms_v95.json`, `gms_jms_185.json`) and confirm the `calls` lists match the "Reference facts" table above (8 calls, no tStart, for all but gms_v95 which has 9 calls including tStart). **If any export disagrees with the table or with the codec's field order, STOP — this is the design §6 escalation, not a fix-it-yourself.**

Fixture inputs (chosen so every byte is distinct and hand-checkable):
- `mistId = uuid.MustParse("00010203-0000-0000-0000-000000000000")` → `mistKey` = `uuid.ID()` = first 4 UUID bytes big-endian = `0x00010203` (same convention as `TestAffectedAreaRemovedByteOutput`)
- `ownerId = 42`, `nType = 7`, `skillId = 2121006` (= `0x205D2E`), `skillLevel = 20`, `phase = 3`
- `origin (100, 200)`, `lt (-50, -30)`, `rb (50, 30)` → absolute RECT L=50 (`0x32`), T=170 (`0xAA`), R=150 (`0x96`), B=230 (`0xE6`)
- `tStart = 1234` (= `0x04D2`), `tEnd = 10000` (= `0x2710`)

Expected little-endian body (39 bytes; v95 inserts the 4 tStart bytes where marked):

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

- [ ] **Step 2: Write the test**

Insert the following into `libs/atlas-packet/field/clientbound/affected_area_test.go` immediately after the closing brace of `TestAffectedAreaCreatedFields` (keeping the two Created tests and the new byte-output test contiguous, ahead of the Removed section):

```go
// TestAffectedAreaCreatedByteOutput pins the full wire body of
// CAffectedAreaPool::OnAffectedAreaCreated (SPAWN_MIST) against the client
// read order per version (docs/packets/ida-exports/):
//
//	v83  @0x431a63: Decode4 dwId, Decode4 nType, Decode4 dwOwnerId,
//	                Decode4 nSkillID, Decode1 nSLV, Decode2 phase,
//	                DecodeBuf(16) rcArea (4×int32 absolute LTRB), Decode4 tEnd.
//	v84  @0x4326ca: same 8 reads as v83 (no tStart).
//	v87  @0x432f3f: same 8 reads as v83 (no tStart; export notes "v87 has NO
//	                leading tStart (matches v83; v95 adds tStart)").
//	v95  @0x437ec0: same, plus Decode4 tStart inserted between rcArea and tEnd.
//	jms  @0x436572: same 8 reads as v83 (export notes "NO leading tStart
//	                (v95-only; absent in JMS like v83)").
//
// Atlas encodes rcArea as origin+offset absolute LTRB (4×WriteInt32) matching
// the client's single DecodeBuf(16). Wire body: 39 bytes (43 on v95).
//
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
	tStart := []byte{0xD2, 0x04, 0x00, 0x00} // tStart = 1234 (v95 only)
	tEnd := []byte{0x10, 0x27, 0x00, 0x00}   // tEnd = 10000

	for _, v := range []struct {
		Name, Region string
		Major, Minor uint16
		HasTStart    bool
	}{
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

- [ ] **Step 3: Run the test — must pass without touching the codec**

```bash
cd libs/atlas-packet && go test -race -run 'TestAffectedAreaCreated' ./field/clientbound/ -v
```

Expected: `TestAffectedAreaCreatedByteOutput` PASS for all five subtests (plus the two pre-existing Created tests still PASS). **If any subtest fails, do NOT edit the codec — that is the design §6 STOP-and-escalate condition** (a byte mismatch means either your hand-derivation is wrong — recheck Step 1 — or the codec diverges from the client, which is out of scope by PRD non-goal).

- [ ] **Step 4: Run the whole package to confirm nothing else broke**

```bash
cd libs/atlas-packet && go test -race ./field/...
```

Expected: `ok` for `field/clientbound` (and all other field packages).

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-packet/field/clientbound/affected_area_test.go
git commit -m "test(atlas-packet): pin SPAWN_MIST wire body across all five versions"
```

---

### Task 2: Pin SPAWN_MIST evidence records (×5)

**Files:**
- Create: `docs/packets/evidence/gms_v83/field.clientbound.FieldAffectedAreaCreated.yaml`
- Create: `docs/packets/evidence/gms_v84/field.clientbound.FieldAffectedAreaCreated.yaml`
- Create: `docs/packets/evidence/gms_v87/field.clientbound.FieldAffectedAreaCreated.yaml`
- Create: `docs/packets/evidence/gms_v95/field.clientbound.FieldAffectedAreaCreated.yaml`
- Create: `docs/packets/evidence/jms_v185/field.clientbound.FieldAffectedAreaCreated.yaml`

**Interfaces:**
- Consumes: the `packet-audit evidence pin` subcommand (VERIFYING_A_PACKET.md §7); Task 1's test name `TestAffectedAreaCreatedByteOutput`.
- Produces: five TIER1-FIXTURE evidence records whose `verifies:` lines point at `libs/atlas-packet/field/clientbound/affected_area_test.go#TestAffectedAreaCreatedByteOutput`; Task 3's matrix regen consumes them.

- [ ] **Step 1: Pin all five records with the tool (never hand-write the sha)**

```bash
for v in gms_v83 gms_v84 gms_v87 gms_v95 jms_v185; do
  go run ./tools/packet-audit evidence pin \
    --packet field/clientbound/FieldAffectedAreaCreated \
    --version "$v" \
    --ida "CAffectedAreaPool::OnAffectedAreaCreated" \
    --category TIER1-FIXTURE
done
```

Expected: five YAML files created, each with `ida.address` matching the Reference-facts table (`0x431a63` / `0x4326ca` / `0x432f3f` / `0x437ec0` / `0x436572`) and a tool-computed `decompile_sha256`. **If the tool cannot resolve the function or address for any version, STOP and ask (unresolved-fname rule) — never substitute an address or hash.**

- [ ] **Step 2: Add the `verifies:` field to each of the five YAMLs**

Append to each file (the pin tool does not write this field — mirror the existing `docs/packets/evidence/gms_v83/field.clientbound.FieldAffectedAreaRemoved.yaml`):

```yaml
verifies:
    - libs/atlas-packet/field/clientbound/affected_area_test.go#TestAffectedAreaCreatedByteOutput
```

- [ ] **Step 3: Sanity-check each record against the Removed sibling's shape**

```bash
head -20 docs/packets/evidence/gms_v83/field.clientbound.FieldAffectedAreaCreated.yaml
head -20 docs/packets/evidence/gms_v83/field.clientbound.FieldAffectedAreaRemoved.yaml
grep -c "verifies:" docs/packets/evidence/*/field.clientbound.FieldAffectedAreaCreated.yaml
```

Expected: the Created record has the same key set and layout as the Removed reference (`packet`, `direction: clientbound`, `version`, `category: TIER1-FIXTURE`, `ida.function/address/decompile_sha256`, `verifies:` list) with `packet: field/clientbound/FieldAffectedAreaCreated`; the grep shows count `1` for all five files. Fix any shape drift (indentation, key order) to match the Removed reference before committing.

- [ ] **Step 4: Commit**

```bash
git add docs/packets/evidence/*/field.clientbound.FieldAffectedAreaCreated.yaml
git commit -m "docs(packets): pin SPAWN_MIST TIER1-FIXTURE evidence for all five versions"
```

---

### Task 3: Regenerate the coverage matrix (SPAWN_MIST ❌→✅ ×5)

**Files:**
- Modify (generated): `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json` (paths are tool-owned; commit whatever the regen touches under `docs/packets/audits/`)

**Interfaces:**
- Consumes: Task 1's five markers + Task 2's five evidence records.
- Produces: STATUS.md SPAWN_MIST row ✅ ×5 — the acceptance artifact for PRD FR-3.

- [ ] **Step 1: Regenerate and check**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check; echo "exit=$?"
```

Expected: `exit=0` (baseline was exit 0; the bar stays a clean 0 — no orphan/dangling/stale/drift lines mentioning `FieldAffectedAreaCreated`).

- [ ] **Step 2: Verify the row flipped**

```bash
grep -n "SPAWN_MIST\|REMOVE_MIST" docs/packets/audits/STATUS.md
```

Expected: the SPAWN_MIST row shows `0x111 | ✅ | 0x118 | ✅ | 0x122 | ✅ | 0x148 | ✅ | 0x126 | ✅` (five ✅, opcodes unchanged); the REMOVE_MIST row is unchanged (still ✅ ×5). If SPAWN_MIST still shows ❌ anywhere, read the tool's check output for the reason (marker/evidence mismatch is the usual cause: exact `packet=`/`version=` string typos) and fix the marker or record — not the tool.

- [ ] **Step 3: Run the full lib test suite once more (markers are comments — this is a no-op guard)**

```bash
cd libs/atlas-packet && go test -race ./... && cd ../..
```

Expected: all packages `ok`.

- [ ] **Step 4: Commit (matrix + anything the regen touched, as one unit)**

```bash
git add docs/packets/audits/
git commit -m "docs(packets): SPAWN_MIST verified across all five versions (matrix regen)"
```

---

### Task 4: Declare the writers in atlas-channel

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/main.go` (the `produceWriters()` slice; the kite cluster sits at lines ~714–716)

**Interfaces:**
- Consumes: `fieldcb.AffectedAreaCreatedWriter` / `fieldcb.AffectedAreaRemovedWriter` (string constants `"AffectedAreaCreated"` / `"AffectedAreaRemoved"` in `libs/atlas-packet/field/clientbound/affected_area_created.go:13` and `affected_area_removed.go`). The `fieldcb` import alias already exists in `main.go`.
- Produces: the declared-writer names that `BuildWriterProducer` (`libs/atlas-opcodes/producer.go`) intersects with the tenant config from Task 5. The mist consumer (`kafka/consumer/mist/consumer.go:63,71`) already announces by these names — no consumer change.

- [ ] **Step 1: Add the two writer declarations**

In `produceWriters()` in `services/atlas-channel/atlas.com/channel/main.go`, immediately after the line `fieldcb.KiteDestroyWriter,` (kites are the adjacent-opcode neighbors 0x10E–0x110; the mists are 0x111/0x112 on v83), insert:

```go
		fieldcb.AffectedAreaCreatedWriter,
		fieldcb.AffectedAreaRemovedWriter,
```

- [ ] **Step 2: Verify the service builds and tests pass**

```bash
cd services/atlas-channel/atlas.com/channel && go build ./... && go vet ./... && go test -race ./... && cd -
```

Expected: all clean. (This is the quick inner-loop check; the full mandatory gate incl. bake runs in Task 6.)

- [ ] **Step 3: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/main.go
git commit -m "fix(atlas-channel): declare AffectedAreaCreated/Removed writers"
```

---

### Task 5: Wire the writers into the five seed templates

**Files:**
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_83_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_gms_95_1.json`
- Modify: `services/atlas-configurations/seed-data/templates/template_jms_185_1.json`

**Interfaces:**
- Consumes: the opcode table from Reference facts. Entry shape used throughout `socket.writers`: `{"opCode": "0xNNN", "writer": "<Name>"}` — no `options` key (the Encode closures ignore options).
- Produces: tenant-side name→opcode mappings that complete the `BuildWriterProducer` intersection with Task 4's declarations. Task 8's live-tenant patch reuses these exact per-version entries.

- [ ] **Step 1: Insert two entries per template, immediately before its `SpawnDoor` entry**

In each file, find the `socket.writers` entry `{"opCode": "0x113", "writer": "SpawnDoor"}` (opcode varies per file: `0x113`/`0x11A`/`0x124`/`0x14A`/`0x128`) and insert before it, matching the file's existing one-line-per-entry formatting exactly:

`template_gms_83_1.json`:
```json
        {"opCode": "0x111", "writer": "AffectedAreaCreated"},
        {"opCode": "0x112", "writer": "AffectedAreaRemoved"},
```

`template_gms_84_1.json`:
```json
        {"opCode": "0x118", "writer": "AffectedAreaCreated"},
        {"opCode": "0x119", "writer": "AffectedAreaRemoved"},
```

`template_gms_87_1.json`:
```json
        {"opCode": "0x122", "writer": "AffectedAreaCreated"},
        {"opCode": "0x123", "writer": "AffectedAreaRemoved"},
```

`template_gms_95_1.json`:
```json
        {"opCode": "0x148", "writer": "AffectedAreaCreated"},
        {"opCode": "0x149", "writer": "AffectedAreaRemoved"},
```

`template_jms_185_1.json`:
```json
        {"opCode": "0x126", "writer": "AffectedAreaCreated"},
        {"opCode": "0x127", "writer": "AffectedAreaRemoved"},
```

(Adjust leading whitespace/quote style to whatever the surrounding lines in each file actually use — copy a neighboring line and edit it.)

- [ ] **Step 2: Verify — exactly one occurrence per name per template, valid JSON, correct opcodes**

```bash
for t in 83 84 87 95; do f=services/atlas-configurations/seed-data/templates/template_gms_${t}_1.json; echo "== $f"; grep -c '"AffectedAreaCreated"' $f; grep -c '"AffectedAreaRemoved"' $f; python3 -m json.tool $f > /dev/null && echo "json ok"; done
f=services/atlas-configurations/seed-data/templates/template_jms_185_1.json; echo "== $f"; grep -c '"AffectedAreaCreated"' $f; grep -c '"AffectedAreaRemoved"' $f; python3 -m json.tool $f > /dev/null && echo "json ok"
grep -n "AffectedArea" services/atlas-configurations/seed-data/templates/template_*_1.json
```

Expected: count `1` for each name in each of the five files, `json ok` ×5, and the final grep shows exactly the ten opcode/name pairs from the table (and nothing in `template_gms_92_1.json` / `template_gms_12_1.json`).

- [ ] **Step 3: Run the seeder tests (they parse these JSON files from disk)**

```bash
cd services/atlas-configurations/atlas.com/configurations && go test -race ./... && cd -
```

Expected: all `ok`.

- [ ] **Step 4: Re-run the matrix check (templates are a matrix input)**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check; echo "exit=$?"
git status --porcelain docs/packets/audits/
```

Expected: `exit=0`. If the regen changed anything under `docs/packets/audits/` (route-state columns), include it in this task's commit.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/ docs/packets/audits/
git commit -m "fix(atlas-configurations): register AffectedArea writers in five seed templates"
```

---

### Task 6: Full verification gate

**Files:** none (verification only; fix-and-recommit into the responsible task's files if anything fails)

**Interfaces:**
- Consumes: everything from Tasks 1–5.
- Produces: the CLAUDE.md "done" bar for the branch — required before code review / PR.

- [ ] **Step 1: Test + vet every changed module**

```bash
cd libs/atlas-packet && go test -race ./... && go vet ./... && cd -
cd services/atlas-channel/atlas.com/channel && go test -race ./... && go vet ./... && go build ./... && cd -
cd services/atlas-configurations/atlas.com/configurations && go test -race ./... && go vet ./... && cd -
```

Expected: all clean.

- [ ] **Step 2: Redis key guard (from worktree root, no GOWORK prefix)**

```bash
tools/redis-key-guard.sh
```

Expected: clean exit.

- [ ] **Step 3: Docker bake for atlas-channel (mandatory — go build does not catch Dockerfile COPY gaps)**

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

### Task 7: Rollout procedure document (`rollout.md`)

**Files:**
- Create: `docs/tasks/task-165-mist-writer-template-wiring/rollout.md`

**Interfaces:**
- Consumes: the per-version opcode table (Task 5) and the atlas-configurations REST surface (verified: `GET /configurations/tenants` list, `GET /configurations/tenants/{tenantId}`, `PATCH /configurations/tenants/{tenantId}` — `services/atlas-configurations/atlas.com/configurations/tenants/resource.go:24-28`; JSON:API type `"tenants"` per `rest.go:24`; PATCH goes through `RegisterInputHandler`, so the JSON:API envelope is mandatory).
- Produces: the reproducible procedure Task 8 executes per environment, and the per-environment record it fills in.

- [ ] **Step 1: Write the document**

Write `docs/tasks/task-165-mist-writer-template-wiring/rollout.md` with exactly this content (placeholders in angle brackets stay as placeholders in the committed file):

````markdown
# task-165 live-tenant rollout: AffectedArea writer entries

Seed templates apply only at tenant creation (known pattern: new opcodes
missing from live tenant config → packet silently dropped). Every existing
tenant on the five affected versions must be patched directly, then
atlas-channel restarted (the configuration projection does not hot-reload
writer maps).

## Per-version entries

Append to the tenant configuration's `socket.writers` array (no `options`):

| Region/version | AffectedAreaCreated | AffectedAreaRemoved |
|---|---|---|
| GMS 83  | `{"opCode": "0x111", "writer": "AffectedAreaCreated"}` | `{"opCode": "0x112", "writer": "AffectedAreaRemoved"}` |
| GMS 84  | `{"opCode": "0x118", "writer": "AffectedAreaCreated"}` | `{"opCode": "0x119", "writer": "AffectedAreaRemoved"}` |
| GMS 87  | `{"opCode": "0x122", "writer": "AffectedAreaCreated"}` | `{"opCode": "0x123", "writer": "AffectedAreaRemoved"}` |
| GMS 95  | `{"opCode": "0x148", "writer": "AffectedAreaCreated"}` | `{"opCode": "0x149", "writer": "AffectedAreaRemoved"}` |
| JMS 185 | `{"opCode": "0x126", "writer": "AffectedAreaCreated"}` | `{"opCode": "0x127", "writer": "AffectedAreaRemoved"}` |

GMS 92 and GMS 12 (reduced templates) are intentionally NOT patched.

## Procedure (per environment)

1. **Enumerate** all tenant configurations:

       GET <configurations-service-base>/api/configurations/tenants

   For each resource note `id`, `attributes.region`, `attributes.majorVersion`.
   Select every tenant whose (region, majorVersion) is in the table above.
   Record the full selected list below — this is a full sweep, not a
   spot-check.

2. **Patch** each selected tenant (read-modify-write, never blind overwrite):

   a. `GET <configurations-service-base>/api/configurations/tenants/<tenantId>`
   b. Idempotency guard: if `attributes.socket.writers` already contains an
      entry with `"writer": "AffectedAreaCreated"`, skip this tenant (record
      as "already patched").
   c. Append the two version-correct entries to `attributes.socket.writers`
      (keep every other attribute exactly as returned).
   d. `PATCH <configurations-service-base>/api/configurations/tenants/<tenantId>`
      with the JSON:API envelope (RegisterInputHandler rejects bare bodies):

          {"data": {"type": "tenants", "id": "<tenantId>",
                    "attributes": { ...full modified attributes object... }}}

3. **Restart** atlas-channel in the environment (writer maps load at startup):

       kubectl -n <namespace> rollout restart deployment atlas-channel
       kubectl -n <namespace> rollout status deployment atlas-channel

4. **Verify** per tenant:
   - atlas-channel startup log: the "Service declares writer [...] but tenant
     config has no opcode mapping" warning no longer lists either
     AffectedArea writer for patched tenants.
   - No `writer not found` / `Unable to broadcast AffectedArea` lines after
     mist activity.

## Environment record

| Environment | Tenant id | Region/version | Patched (date) | Restarted | Verified |
|---|---|---|---|---|---|
| <env> | <tenant-uuid> | <e.g. GMS 83> | <date or "already patched"> | <yes/no> | <yes/no> |
````

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

### Task 8: Code review, PR, live rollout, and end-to-end verification

**Files:**
- Modify: `docs/tasks/task-165-mist-writer-template-wiring/rollout.md` (fill in the environment record)

**Interfaces:**
- Consumes: the completed branch (Tasks 1–7); the rollout procedure (Task 7); a deploy environment via the `deploy-env` PR label flow.
- Produces: PRD acceptance criteria — live tenants patched + restarted, mists visibly rendering, clean logs.

> **Environment note:** Steps 3–6 require a running deploy environment and a
> game client; they cannot be completed inside the worktree alone. Do not mark
> them done on the strength of code inspection — each has an observable
> expected result. If the deploy environment is unavailable, stop after Step 2
> and report BLOCKED on the environment (not "done").

- [ ] **Step 1: Code review (mandatory before PR)**

Invoke `superpowers:requesting-code-review` for this branch (Go changes → it dispatches `plan-adherence-reviewer` + `backend-guidelines-reviewer`). Address findings; findings land in `docs/tasks/task-165-mist-writer-template-wiring/audit.md`.

- [ ] **Step 2: Open the PR with the deploy-env label**

```bash
git push -u origin task-165-mist-writer-template-wiring
env -u GH_TOKEN -u GITHUB_TOKEN gh pr create \
  --title "fix(mist): wire AffectedArea writers (channel + templates) and verify SPAWN_MIST ×5" \
  --body-file - <<'EOF'
Mist broadcasts were silently dropped for every tenant: neither AffectedArea
writer was declared in atlas-channel's produceWriters() nor registered in any
seed template, so session.Announce failed with "writer not found".

- Declare AffectedAreaCreated/Removed in atlas-channel produceWriters()
- Register both writers (per-version opcodes) in the five full seed templates
- Promote SPAWN_MIST ❌→✅ ×5 (byte fixtures + evidence + matrix regen)
- Rollout procedure for live tenants: docs/tasks/task-165-mist-writer-template-wiring/rollout.md

Task: docs/tasks/task-165-mist-writer-template-wiring/

🤖 Generated with [Claude Code](https://claude.com/claude-code)
EOF
env -u GH_TOKEN -u GITHUB_TOKEN gh pr edit --add-label deploy-env
```

Expected: PR created; the `deploy-env` label triggers the ephemeral build/deploy.

- [ ] **Step 3: Execute the rollout procedure in the deploy environment**

Follow `rollout.md` exactly: enumerate ALL tenant configurations, patch every tenant on the five versions (idempotency guard honored), restart atlas-channel, and fill in the environment-record table. Full sweep — the record must list every tenant on those versions, including "already patched" rows.

- [ ] **Step 4: Verify the deployed binary and config before testing (first-diagnostic rule)**

- Confirm the running atlas-channel image is built from this branch (check the pod image tag against the PR build).
- `GET .../api/configurations/tenants/<tenantId>` for one patched tenant and confirm both writer entries are present.
- Check atlas-channel startup logs: neither AffectedArea writer appears in the "no opcode mapping" warning for patched tenants.

- [ ] **Step 5: End-to-end — mob mist**

Pick a mob with an AREA_POISON mob skill **from local WZ data or repo source at execution time** (Verification Over Memory — do not choose from MapleStory memory; search the tenant's mob-skill data for AREA_POISON/mist entries and pick a mob+map from that). On a patched tenant: stand near the mob until it casts. Expected: the poison cloud renders visibly and disappears when the mist expires; atlas-channel logs show no `Unable to broadcast AffectedArea` / `writer not found` lines.

- [ ] **Step 6: End-to-end — player mist + observer**

Pick a player mist skill by the same WZ/repo-verification rule (the codec fixture uses 2121006 as an input value, but the e2e skill must be re-verified against the tenant's WZ data before use). Cast it with one client while a second client watches on the same map. Expected: the mist renders for both caster and observer; same clean-log check.

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
| FR-1 Go writer declaration | Task 4 |
| FR-2 Seed template wiring ×5 | Task 5 |
| FR-3 SPAWN_MIST matrix promotion ×5 | Tasks 1–3 |
| FR-4 Live tenant rollout | Tasks 7–8 |
| FR-5 End-to-end verification | Task 8 |
| Build/verification gate (PRD §7, CLAUDE.md) | Task 6 (+ inner-loop checks in 1, 4, 5) |
| Code review before PR | Task 8 Step 1 |
