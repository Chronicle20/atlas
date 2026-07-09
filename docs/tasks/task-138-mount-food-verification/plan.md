# Mount Food Packet Verification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Promote all five serverbound `USE_MOUNT_FOOD` coverage-matrix cells (gms_v83, gms_v84, gms_v87, gms_v95, jms_v185) from ❌ to ✅ with IDA-derived byte-level evidence.

**Architecture:** One tooling-linkage change (`candidatesFromFName` case in packet-audit) makes the packet gradeable, then five strictly-serialized per-version verification passes each produce three co-committed artifacts: an export splice, an audit report, and a `packet-audit:verify` byte-fixture marker. The matrix is regenerated (never hand-edited) after each pass.

**Tech Stack:** Go (libs/atlas-packet, tools/packet-audit), ida-pro-mcp (live decompilation), python3 (surgical JSON export splice).

## Global Constraints

- **Strict serialization:** Tasks 2–6 run one at a time, in order v83 → v84 → v87 → v95 → jms_v185. `select_instance` is shared global state on the IDA server; all five markers land in one `food_test.go`; matrix regen is global. Never run two versions in parallel.
- **Grounding:** every opcode and fixture byte must trace to a decompiled function address in the matching IDB. The integer in `COutPacket::COutPacket(&pkt, OPCODE)` is the opcode ground truth — distrust the IDB symbol name and the csv-seeded registry alike. Nothing may be inferred from version-shift patterns or MapleStory memory.
- **Export hygiene:** committed exports in `docs/packets/ida-exports/` are NOT idempotent. Never regenerate or overwrite one wholesale. Harvest to a temp file, splice surgically (overwrite the one sender entry; add helpers only if absent), strip any `{op: "Delegate", ref: "COutPacket..."}` artifact.
- **Matrix:** never hand-edit `docs/packets/audits/STATUS.md` or `status.json`; always regenerate via `go run ./tools/packet-audit matrix`. The `matrix --check` bar is **no new problems**: zero new orphan/dangling/stale/drift lines mentioning this packet, conflict count not increased (pre-existing 🟥 backlog may keep exit ≠ 0).
- **Tier-0 cell:** `USE_MOUNT_FOOD` is `tier1: false`. Do NOT pin evidence records (`packet-audit evidence pin`) — a tier-0 cell promotes on audit report + marker alone, and a pinned record is a standing freshness liability.
- **Commit unit is the cell:** each version's test marker + export splice + audit report + regenerated STATUS.md/status.json commit together. A marker must never land in a commit without its report (orphan-marker failure).
- **Hard out-of-scope:** anything v92 (no template edit, no matrix column, no opcode inference); the clientbound `SetTamingMobInfo` writer; handler/processor/Kafka behavior in atlas-channel and atlas-mounts.
- **Stop-and-ask escalations:** a send function genuinely unlocatable in an IDB after regex + byte-signature attempts; no live v84 IDA instance available. Never substitute a fname, borrow another version's address, or fake a hash.
- All commands run from the worktree root (`.worktrees/task-138-mount-food-verification`). Committed files must never contain literal home/absolute paths.

## Baseline facts (verified against the repo at planning time)

- Codec: `libs/atlas-packet/mount/serverbound/food.go` — `type Food struct` with `updateTime uint32, slot int16, itemId uint32`; Encode/Decode are unconditional `ts(4) LE, slot(2) LE, itemId(4) LE` (no version gates). Existing test: `TestFoodDecode` in `food_test.go` (keep it).
- Registry rows (`docs/packets/registry/<v>.yaml`): opcode 77 (v83:2181, v84:2844), 80 (v87:2311), 83 (v95:2529), 69 (jms:2306); all `fname: CWvsContext::SendTamingMobFoodItemUseRequest`, `provenance: csv-import`.
- Seed templates (`services/atlas-configurations/seed-data/templates/`): `MountFoodHandle` registered with `LoggedInValidator` at `0x4D` (template_gms_83_1.json:405, template_gms_84_1.json:409), `0x50` (template_gms_87_1.json:358), `0x53` (template_gms_95_1.json:238), `0x45` (template_jms_185_1.json:325). Registry and templates already agree — discrepancy branch (a) fires only if an IDB contradicts both.
- `CWvsContext::SendTamingMobFoodItemUseRequest` is absent from all five exports in `docs/packets/ida-exports/` and has no `candidatesFromFName` case in `tools/packet-audit/cmd/run.go` — that missing case is why every cell reads "no audit report".
- Direct analog: `CWvsContext::SendPetFoodItemUseRequest` (run.go:1069) → `{name: "Food", pkg: "pet", dir: csvpkg.DirServerbound}`; its v83 export entry is `Encode4(get_update_time) + Encode2(nPOS) + Encode4(nItemID)` — the same expected shape.
- `pt.Variants` indices (`libs/atlas-packet/test/context.go:18`): `[0]`=GMS v28, `[1]`=GMS v83, `[2]`=GMS v87, `[3]`=GMS v95, `[4]`=JMS v185, `[5]`=GMS v84, `[6]`=GMS v86.
- Export JSON format: top-level keys `binary, md5, generated_at, functions`; 2-space indent; trailing newline; `functions` keys are NOT sorted (append new entries at the end for a minimal diff).
- jms mapping: version key `gms_jms_185`, export `gms_jms_185.json`, but the audit dir is `jms_v185` — the root report command maps this automatically (root.go:204–208). The jms retail dump is SMC; only the `*_U_DEVM` IDB decompiles.

---

### Task 1: Link the fname to the codec in packet-audit

**Files:**
- Modify: `tools/packet-audit/cmd/run.go` (the `candidatesFromFName` switch — insert near the mount/taming area or beside `CWvsContext::SendPetFoodItemUseRequest` at ~line 1069)
- Modify: `tools/packet-audit/cmd/disambiguation_test.go` (extend the two existing test tables)

**Interfaces:**
- Produces: `candidatesFromFName("CWvsContext::SendTamingMobFoodItemUseRequest")` → `[]candidate{{name: "Food", pkg: "mount", dir: csvpkg.DirServerbound}}`. Report/marker name derives as `qualifiedWriterName("mount", "Food")` = `MountFood`, so Tasks 2–6 use report file `MountFood.{json,md}` and marker path `packet=mount/serverbound/MountFood`.

- [ ] **Step 1: Add regression table entries to the existing tests**

In `tools/packet-audit/cmd/disambiguation_test.go`, add to the `TestQualifiedWriterName` cases slice:

```go
		{"mount", "Food", "MountFood"},
```

and to the `TestLocateAtlasFileDisambiguatesByPkg` cases slice (this one pins that the pkg hint resolves to the serverbound mount file, not `pet/serverbound/food.go`):

```go
		{"mount", "Food", csvpkg.DirServerbound, "/mount/serverbound/"},
```

- [ ] **Step 2: Run the disambiguation tests**

Run: `cd tools/packet-audit && go test ./cmd/ -run 'TestQualifiedWriterName|TestLocateAtlasFileDisambiguatesByPkg' -v`
Expected: PASS (both helpers are generic — these entries are regression pins, not failing tests; the behavior gap is the missing switch case, which no test enumerates).

- [ ] **Step 3: Add the `candidatesFromFName` case**

In `tools/packet-audit/cmd/run.go`, immediately after the `CWvsContext::SendPetFoodItemUseRequest` case (~line 1069):

```go
	case "CWvsContext::SendTamingMobFoodItemUseRequest":
		// USE_MOUNT_FOOD — taming-mob (mount) food. Codec mount/serverbound/Food
		// (handler MountFoodHandle). ts u32 + slot i16 + itemId u32.
		return []candidate{{name: "Food", pkg: "mount", dir: csvpkg.DirServerbound}}
```

- [ ] **Step 4: Build, vet, and full-package test**

Run: `cd tools/packet-audit && go build ./... && go vet ./... && go test ./...`
Expected: all clean/PASS.

- [ ] **Step 5: Confirm no matrix regression**

Run from worktree root: `go run ./tools/packet-audit matrix --check 2>&1 | grep -i "mount\|MountFood"`
Expected: no orphan/dangling/stale/drift lines mentioning MountFood (cells stay ❌ "no audit report" until Task 2 — the linkage only takes effect once the fname exists in an export).

- [ ] **Step 6: Commit**

```bash
git add tools/packet-audit/cmd/run.go tools/packet-audit/cmd/disambiguation_test.go
git commit -m "feat(packet-audit): link SendTamingMobFoodItemUseRequest to mount/serverbound Food"
```

---

### Task 2: Verify gms_v83 (opcode 77 / 0x4D expected)

**Files:**
- Modify: `docs/packets/ida-exports/gms_v83.json` (splice sender + helpers)
- Create: `docs/packets/audits/gms_v83/MountFood.json`, `docs/packets/audits/gms_v83/MountFood.md` (copied from report-gen output)
- Modify: `libs/atlas-packet/mount/serverbound/food_test.go` (new `TestFoodByteFixture` + v83 marker)
- Modify (regenerated): `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`
- Modify only if branch (b) fires: `libs/atlas-packet/mount/serverbound/food.go`

**Interfaces:**
- Consumes: Task 1's `candidatesFromFName` case (report name `MountFood`).
- Produces: `TestFoodByteFixture` with a `cases []struct{ variant pt.TenantVariant; want []byte }` table that Tasks 3–6 append one case each to, plus the marker block above it that Tasks 3–6 append one marker line each to.

- [ ] **Step 1: Select the v83 IDA instance**

Use `mcp__ida-pro__list_instances`, find the instance whose loaded IDB is GMS v83 (the checked-in export header says `"binary": "MapleStory_dump.exe (v83 Me)"`, md5 `80ff438ced539b831f0d2ed95099275d`), then `mcp__ida-pro__select_instance(port)`. Ports vary by launch order — never hardcode. Quote the instance metadata in your notes before reading anything.

- [ ] **Step 2: Locate the send function**

Run `mcp__ida-pro__func_query` with `name_regex: "SendTamingMobFoodItemUseRequest"`.
- Found → record the address and go to Step 3.
- Not found → the sender is unnamed: locate it via `mcp__ida-pro__find_bytes` with signature `6A 4D 8D 8D ?? ?? ?? ?? E8` (push 0x4D; lea ecx; call COutPacket ctor), decompile candidates, structure-match against the pet-food twin shape (`get_update_time` + Encode2 + Encode4), then `mcp__ida-pro__rename` it `CWvsContext::SendTamingMobFoodItemUseRequest` and `mcp__ida-pro__idb_save`. Unnamed ≠ unnameable — naming is producible work.
- Still unlocatable after both attempts → STOP AND ASK, reporting the exact regex and byte-signature searches attempted. Never substitute a fname.

- [ ] **Step 3: Decompile and record ground truth**

`mcp__ida-pro__decompile` the function (descend into helper writes, address-based). Write down, quoting the actual decompiled lines:
1. The integer in `COutPacket::COutPacket(&pkt, OPCODE)` — must be 77 (0x4D) to match registry `docs/packets/registry/gms_v83.yaml:2183` and template `template_gms_83_1.json:405`.
2. The full ordered encode list with widths (expected: `Encode4(update-time), Encode2(slot/nPOS), Encode4(nItemID)`).

Discrepancy branches (handle NOW, in this task, before proceeding):
- **(a) opcode ≠ 77:** fix `docs/packets/registry/gms_v83.yaml` AND `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` to the IDB value, as their own commit `fix(packets): correct USE_MOUNT_FOOD gms_v83 opcode to 0x<hex> per IDB`, and record in the task notes that the PR must call out that existing v83 tenants need a live config patch (new-opcodes-not-in-live-config incident class).
- **(b) encode order ≠ ts u32/slot i16/itemId u32:** fix `libs/atlas-packet/mount/serverbound/food.go` Encode+Decode FIRST as its own commit, version-gating via `tenant.MustFromContext(ctx).MajorVersion()` branches per existing atlas-packet patterns. Gate rule: a shape that differs between v83 and v87 gates at `>= 87`, NOT `> 83` (v84 matches v83 unless the v84 IDB says otherwise — Task 3 confirms). Update `TestFoodDecode` in the same commit. atlas-channel handler wiring changes only if the decoded field set changes meaning.
- **(c) function absent from the IDB entirely:** STOP AND ASK with the search evidence.

- [ ] **Step 4: Harvest the sender to a temp export**

Write the roster file, then run a TARGETED harvest (empty `-prior-export` = harvest only the roster fnames):

```bash
printf 'CWvsContext::SendTamingMobFoodItemUseRequest\n' > /tmp/mountfood_roster.md
go run ./tools/packet-audit export \
  -version gms_v83 \
  -ida-url http://192.168.20.3:13337/mcp -ida-port <v83 port from Step 1> \
  -prior-export "" -pending /tmp/mountfood_roster.md \
  -descent-depth 12 \
  -output /tmp/harvest_gms_v83.json
```

Expected: `/tmp/harvest_gms_v83.json` contains a `functions` entry for the fname whose `address` matches Step 2 and whose `calls` match the Step 3 encode order.

- [ ] **Step 5: Surgically splice into the committed export**

```bash
python3 - <<'EOF'
import json

COMMITTED = 'docs/packets/ida-exports/gms_v83.json'
HARVEST = '/tmp/harvest_gms_v83.json'
FNAME = 'CWvsContext::SendTamingMobFoodItemUseRequest'

c = json.load(open(COMMITTED))
h = json.load(open(HARVEST))

entry = h['functions'][FNAME]
# Strip the COutPacket-delegate harvest artifact (report-gen killer): the
# packet ctor is not a wire write; other versions' sender entries omit it.
entry['calls'] = [x for x in (entry.get('calls') or [])
                  if not (x.get('op') == 'Delegate' and 'COutPacket' in (x.get('ref') or ''))]
c['functions'][FNAME] = entry  # overwrite-or-add the one sender

# Helpers: absent-only (never overwrite an existing committed entry).
added = [FNAME]
for k, v in h['functions'].items():
    if k != FNAME and k not in c['functions']:
        c['functions'][k] = v
        added.append(k)

with open(COMMITTED, 'w') as f:
    f.write(json.dumps(c, indent=2) + '\n')
print('spliced:', added)
EOF
git diff --stat docs/packets/ida-exports/gms_v83.json
```

Expected: diff shows ONLY additions at the end of the `functions` map (plus the one closing-brace context line). If any existing key changed, `git checkout` the file and redo the splice — that is the export-drift failure mode.

- [ ] **Step 6: Generate the audit report and copy it in**

```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
  -ida-source docs/packets/ida-exports/gms_v83.json \
  -output /tmp/rpt_gms_v83
grep -o '"verdict": *"[^"]*"' /tmp/rpt_gms_v83/gms_v83/MountFood.json
cp /tmp/rpt_gms_v83/gms_v83/MountFood.json /tmp/rpt_gms_v83/gms_v83/MountFood.md docs/packets/audits/gms_v83/
```

Expected: verdict is `Match`. A non-Match is a real divergence — go back to Step 3's discrepancy branches, fix, re-generate. Do not copy a non-Match report in and do not proceed with a non-Match.

- [ ] **Step 7: Write the byte-fixture test with the v83 marker**

In `libs/atlas-packet/mount/serverbound/food_test.go`, add below the existing `TestFoodDecode` (which stays — it pins the handler-facing decode contract). Fill `<v83 addr>` with the Step 2 address (same value the export entry carries — the marker's `ida=` must match the audit report address or `matrix --check` flags an orphan marker). The `want` bytes below assume the expected v83 layout; if branch (b) fired, hand-compute them from the IDB-derived order instead, one comment per field citing its decompile line:

```go
// TestFoodByteFixture pins the exact serverbound wire bytes per version,
// hand-computed from each version's decompiled send order (full body, never
// opcode-only). IDA evidence (COutPacket opcode + encode order):
//   v83 SendTamingMobFoodItemUseRequest@0x<v83 addr>: op 0x4D;
//       Encode4(get_update_time) + Encode2(nPOS) + Encode4(nItemID)
// packet-audit:verify packet=mount/serverbound/MountFood version=gms_v83 ida=0x<v83 addr>
func TestFoodByteFixture(t *testing.T) {
	// ts=100 (64 00 00 00), slot=3 (03 00), itemId=2000000 (80 84 1E 00) — LE.
	cases := []struct {
		variant pt.TenantVariant
		want    []byte
	}{
		{pt.Variants[1], []byte{0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x80, 0x84, 0x1E, 0x00}}, // GMS v83
	}
	for _, tc := range cases {
		t.Run(tc.variant.Name, func(t *testing.T) {
			ctx := pt.CreateContext(tc.variant.Region, tc.variant.MajorVersion, tc.variant.MinorVersion)
			input := Food{updateTime: 100, slot: 3, itemId: 2000000}
			got := input.Encode(logrus.New(), ctx)(nil)
			if !bytes.Equal(got, tc.want) {
				t.Errorf("bytes: got % X, want % X", got, tc.want)
			}
			output := Food{}
			req := request.Request(tc.want)
			reader := request.NewRequestReader(&req, 0)
			output.Decode(logrus.New(), ctx)(&reader, nil)
			if output.UpdateTime() != 100 || output.Slot() != 3 || output.ItemId() != 2000000 {
				t.Errorf("decode round-trip mismatch: %s", output.String())
			}
		})
	}
}
```

Add the imports the file doesn't have yet: `bytes` and `pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"` (`context`, `testing`, `request`, `logrus` are already imported).

- [ ] **Step 8: Run the test**

Run: `cd libs/atlas-packet && go test -race ./mount/... -v`
Expected: `TestFoodByteFixture/GMS_v83` and `TestFoodDecode` PASS.

- [ ] **Step 9: Regenerate the matrix and verify promotion**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check 2>&1 | grep -i "mount\|MountFood" || true
grep "USE_MOUNT_FOOD" docs/packets/audits/STATUS.md
```

Expected: the gms_v83 cell is ✅ (others still ❌); no orphan/dangling/stale/drift lines mentioning MountFood; conflict count not increased vs before this task (compare `matrix --check` tail summaries).

- [ ] **Step 10: Commit the cell atomically**

```bash
git add libs/atlas-packet/mount/serverbound/food_test.go \
  docs/packets/ida-exports/gms_v83.json \
  docs/packets/audits/gms_v83/MountFood.json docs/packets/audits/gms_v83/MountFood.md \
  docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "verify(packets): USE_MOUNT_FOOD gms_v83 byte-verified (0x4D @<v83 addr>)"
```

---

### Task 3: Verify gms_v84 (opcode 77 / 0x4D expected)

**Files:**
- Modify: `docs/packets/ida-exports/gms_v84.json`
- Create: `docs/packets/audits/gms_v84/MountFood.json`, `docs/packets/audits/gms_v84/MountFood.md`
- Modify: `libs/atlas-packet/mount/serverbound/food_test.go` (append v84 case + marker)
- Modify (regenerated): `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

**Interfaces:**
- Consumes: Task 2's `TestFoodByteFixture` cases table and marker block.

- [ ] **Step 1: Select the v84 IDA instance**

`mcp__ida-pro__list_instances` → find the GMS v84.1 IDB → `mcp__ida-pro__select_instance(port)`. Quote the instance metadata. **If no v84 instance exists and one cannot be brought up, STOP AND ASK — a missing IDB is a genuine blocker.** The v84 cell must be derived from the v84 binary; never copy v83's bytes or address (the PRD forbids assumed byte-identity even if the answer turns out identical).

- [ ] **Step 2: Locate the send function**

`mcp__ida-pro__func_query` with `name_regex: "SendTamingMobFoodItemUseRequest"`. If unnamed: `mcp__ida-pro__find_bytes` signature `6A 4D 8D 8D ?? ?? ?? ?? E8`, decompile candidates, structure-match to the v83 twin from Task 2, `mcp__ida-pro__rename` + `mcp__ida-pro__idb_save`. Still unlocatable → STOP AND ASK with search evidence.

- [ ] **Step 3: Decompile and record ground truth**

Record the `COutPacket` opcode integer (expected 77 per `docs/packets/registry/gms_v84.yaml:2846` / `template_gms_84_1.json:409`) and the ordered encode list, quoting decompiled lines. Discrepancy branches, handled now in this task:
- **(a) opcode ≠ 77:** fix `docs/packets/registry/gms_v84.yaml` + `template_gms_84_1.json` in their own commit; note the live-tenant config-patch callout for the PR.
- **(b) order ≠ ts u32/slot i16/itemId u32:** version-gate `food.go` FIRST as its own commit. If v84 diverges from v83 here, the gate is `MajorVersion() >= 84` (the summon-attack precedent), not `>= 87` — the gate boundary is whatever the two IDBs prove.
- **(c) absent:** STOP AND ASK.

- [ ] **Step 4: Harvest to a temp export**

```bash
printf 'CWvsContext::SendTamingMobFoodItemUseRequest\n' > /tmp/mountfood_roster.md
go run ./tools/packet-audit export \
  -version gms_v84 \
  -ida-url http://192.168.20.3:13337/mcp -ida-port <v84 port from Step 1> \
  -prior-export "" -pending /tmp/mountfood_roster.md \
  -descent-depth 12 \
  -output /tmp/harvest_gms_v84.json
```

- [ ] **Step 5: Surgically splice into the committed export**

Same script as Task 2 Step 5 with `COMMITTED = 'docs/packets/ida-exports/gms_v84.json'` and `HARVEST = '/tmp/harvest_gms_v84.json'`:

```bash
python3 - <<'EOF'
import json

COMMITTED = 'docs/packets/ida-exports/gms_v84.json'
HARVEST = '/tmp/harvest_gms_v84.json'
FNAME = 'CWvsContext::SendTamingMobFoodItemUseRequest'

c = json.load(open(COMMITTED))
h = json.load(open(HARVEST))

entry = h['functions'][FNAME]
entry['calls'] = [x for x in (entry.get('calls') or [])
                  if not (x.get('op') == 'Delegate' and 'COutPacket' in (x.get('ref') or ''))]
c['functions'][FNAME] = entry

added = [FNAME]
for k, v in h['functions'].items():
    if k != FNAME and k not in c['functions']:
        c['functions'][k] = v
        added.append(k)

with open(COMMITTED, 'w') as f:
    f.write(json.dumps(c, indent=2) + '\n')
print('spliced:', added)
EOF
git diff --stat docs/packets/ida-exports/gms_v84.json
```

Expected: additions only; existing keys untouched.

- [ ] **Step 6: Generate and copy the report**

```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/template_gms_84_1.json \
  -ida-source docs/packets/ida-exports/gms_v84.json \
  -output /tmp/rpt_gms_v84
grep -o '"verdict": *"[^"]*"' /tmp/rpt_gms_v84/gms_v84/MountFood.json
cp /tmp/rpt_gms_v84/gms_v84/MountFood.json /tmp/rpt_gms_v84/gms_v84/MountFood.md docs/packets/audits/gms_v84/
```

Expected: verdict `Match`; non-Match → back to Step 3 branches.

- [ ] **Step 7: Append the v84 fixture case and marker**

In `food_test.go`: add to the evidence comment block a line `//   v84 SendTamingMobFoodItemUseRequest@0x<v84 addr>: op 0x4D; <derived order>`, add below the v83 marker:

```go
// packet-audit:verify packet=mount/serverbound/MountFood version=gms_v84 ida=0x<v84 addr>
```

and append to the `cases` slice (bytes per the v84-derived order — identical to v83's only if the v84 decompile says so):

```go
		{pt.Variants[5], []byte{0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x80, 0x84, 0x1E, 0x00}}, // GMS v84
```

- [ ] **Step 8: Run the test**

Run: `cd libs/atlas-packet && go test -race ./mount/... -v`
Expected: `TestFoodByteFixture/GMS_v84` PASS (plus v83 still green).

- [ ] **Step 9: Regenerate the matrix and verify promotion**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check 2>&1 | grep -i "mount\|MountFood" || true
grep "USE_MOUNT_FOOD" docs/packets/audits/STATUS.md
```

Expected: gms_v83 ✅ and gms_v84 ✅; no new problem lines; conflict count unchanged.

- [ ] **Step 10: Commit**

```bash
git add libs/atlas-packet/mount/serverbound/food_test.go \
  docs/packets/ida-exports/gms_v84.json \
  docs/packets/audits/gms_v84/MountFood.json docs/packets/audits/gms_v84/MountFood.md \
  docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "verify(packets): USE_MOUNT_FOOD gms_v84 byte-verified (0x4D @<v84 addr>)"
```

---

### Task 4: Verify gms_v87 (opcode 80 / 0x50 expected)

**Files:**
- Modify: `docs/packets/ida-exports/gms_v87.json`
- Create: `docs/packets/audits/gms_v87/MountFood.json`, `docs/packets/audits/gms_v87/MountFood.md`
- Modify: `libs/atlas-packet/mount/serverbound/food_test.go` (append v87 case + marker)
- Modify (regenerated): `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

**Interfaces:**
- Consumes: Task 2's `TestFoodByteFixture` cases table and marker block.

- [ ] **Step 1: Select the v87 IDA instance**

`mcp__ida-pro__list_instances` → GMS v87 IDB → `mcp__ida-pro__select_instance(port)`. Quote metadata.

- [ ] **Step 2: Locate the send function**

`mcp__ida-pro__func_query` `name_regex: "SendTamingMobFoodItemUseRequest"`. If unnamed: byte-signature `6A 50 8D 8D ?? ?? ?? ?? E8` (0x50 is the expected v87 opcode), structure-match to the v83 twin, rename + `idb_save`. Note: many v87 names came from mangled-symbol demangling (task-085 groundwork) — the function may exist under a mangled name; the byte signature finds it regardless. Still unlocatable → STOP AND ASK.

- [ ] **Step 3: Decompile and record ground truth**

Opcode expected 80 (`docs/packets/registry/gms_v87.yaml:2313` / `template_gms_87_1.json:358`); record the encode order. Branches: **(a)** opcode mismatch → fix registry + template own commit + PR callout; **(b)** order mismatch vs v83 → version-gate `food.go` at `>= 87` (v84 = v83-shaped per Task 3's evidence) as its own commit, update `TestFoodDecode`; **(c)** absent → STOP AND ASK.

- [ ] **Step 4: Harvest to a temp export**

```bash
printf 'CWvsContext::SendTamingMobFoodItemUseRequest\n' > /tmp/mountfood_roster.md
go run ./tools/packet-audit export \
  -version gms_v87 \
  -ida-url http://192.168.20.3:13337/mcp -ida-port <v87 port from Step 1> \
  -prior-export "" -pending /tmp/mountfood_roster.md \
  -descent-depth 12 \
  -output /tmp/harvest_gms_v87.json
```

- [ ] **Step 5: Surgically splice into the committed export**

```bash
python3 - <<'EOF'
import json

COMMITTED = 'docs/packets/ida-exports/gms_v87.json'
HARVEST = '/tmp/harvest_gms_v87.json'
FNAME = 'CWvsContext::SendTamingMobFoodItemUseRequest'

c = json.load(open(COMMITTED))
h = json.load(open(HARVEST))

entry = h['functions'][FNAME]
entry['calls'] = [x for x in (entry.get('calls') or [])
                  if not (x.get('op') == 'Delegate' and 'COutPacket' in (x.get('ref') or ''))]
c['functions'][FNAME] = entry

added = [FNAME]
for k, v in h['functions'].items():
    if k != FNAME and k not in c['functions']:
        c['functions'][k] = v
        added.append(k)

with open(COMMITTED, 'w') as f:
    f.write(json.dumps(c, indent=2) + '\n')
print('spliced:', added)
EOF
git diff --stat docs/packets/ida-exports/gms_v87.json
```

Expected: additions only.

- [ ] **Step 6: Generate and copy the report**

```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/template_gms_87_1.json \
  -ida-source docs/packets/ida-exports/gms_v87.json \
  -output /tmp/rpt_gms_v87
grep -o '"verdict": *"[^"]*"' /tmp/rpt_gms_v87/gms_v87/MountFood.json
cp /tmp/rpt_gms_v87/gms_v87/MountFood.json /tmp/rpt_gms_v87/gms_v87/MountFood.md docs/packets/audits/gms_v87/
```

Expected: verdict `Match`.

- [ ] **Step 7: Append the v87 fixture case and marker**

Evidence-comment line `//   v87 ...@0x<v87 addr>: op 0x50; <derived order>`, marker:

```go
// packet-audit:verify packet=mount/serverbound/MountFood version=gms_v87 ida=0x<v87 addr>
```

case (bytes per the v87-derived order):

```go
		{pt.Variants[2], []byte{0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x80, 0x84, 0x1E, 0x00}}, // GMS v87
```

- [ ] **Step 8: Run the test**

Run: `cd libs/atlas-packet && go test -race ./mount/... -v`
Expected: PASS including `TestFoodByteFixture/GMS_v87`.

- [ ] **Step 9: Regenerate the matrix and verify promotion**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check 2>&1 | grep -i "mount\|MountFood" || true
grep "USE_MOUNT_FOOD" docs/packets/audits/STATUS.md
```

Expected: v83/v84/v87 ✅; no new problem lines.

- [ ] **Step 10: Commit**

```bash
git add libs/atlas-packet/mount/serverbound/food_test.go \
  docs/packets/ida-exports/gms_v87.json \
  docs/packets/audits/gms_v87/MountFood.json docs/packets/audits/gms_v87/MountFood.md \
  docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "verify(packets): USE_MOUNT_FOOD gms_v87 byte-verified (0x50 @<v87 addr>)"
```

---

### Task 5: Verify gms_v95 (opcode 83 / 0x53 expected)

**Files:**
- Modify: `docs/packets/ida-exports/gms_v95.json`
- Create: `docs/packets/audits/gms_v95/MountFood.json`, `docs/packets/audits/gms_v95/MountFood.md`
- Modify: `libs/atlas-packet/mount/serverbound/food_test.go` (append v95 case + marker)
- Modify (regenerated): `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

**Interfaces:**
- Consumes: Task 2's `TestFoodByteFixture` cases table and marker block.

- [ ] **Step 1: Select the v95 IDA instance**

`mcp__ida-pro__list_instances` → GMS v95 IDB → `mcp__ida-pro__select_instance(port)`. Quote metadata.

- [ ] **Step 2: Locate the send function**

`mcp__ida-pro__func_query` `name_regex: "SendTamingMobFoodItemUseRequest"`. If unnamed: byte-signature `6A 53 8D 8D ?? ?? ?? ?? E8`, structure-match, rename + `idb_save`. Still unlocatable → STOP AND ASK.

- [ ] **Step 3: Decompile and record ground truth**

Opcode expected 83 (`docs/packets/registry/gms_v95.yaml:2531` / `template_gms_95_1.json:238`); record the encode order. Branches as in Task 4 ((a) registry+template fix own commit + PR callout; (b) version-gated codec fix own commit; (c) STOP AND ASK).

- [ ] **Step 4: Harvest to a temp export**

```bash
printf 'CWvsContext::SendTamingMobFoodItemUseRequest\n' > /tmp/mountfood_roster.md
go run ./tools/packet-audit export \
  -version gms_v95 \
  -ida-url http://192.168.20.3:13337/mcp -ida-port <v95 port from Step 1> \
  -prior-export "" -pending /tmp/mountfood_roster.md \
  -descent-depth 12 \
  -output /tmp/harvest_gms_v95.json
```

- [ ] **Step 5: Surgically splice into the committed export**

```bash
python3 - <<'EOF'
import json

COMMITTED = 'docs/packets/ida-exports/gms_v95.json'
HARVEST = '/tmp/harvest_gms_v95.json'
FNAME = 'CWvsContext::SendTamingMobFoodItemUseRequest'

c = json.load(open(COMMITTED))
h = json.load(open(HARVEST))

entry = h['functions'][FNAME]
entry['calls'] = [x for x in (entry.get('calls') or [])
                  if not (x.get('op') == 'Delegate' and 'COutPacket' in (x.get('ref') or ''))]
c['functions'][FNAME] = entry

added = [FNAME]
for k, v in h['functions'].items():
    if k != FNAME and k not in c['functions']:
        c['functions'][k] = v
        added.append(k)

with open(COMMITTED, 'w') as f:
    f.write(json.dumps(c, indent=2) + '\n')
print('spliced:', added)
EOF
git diff --stat docs/packets/ida-exports/gms_v95.json
```

Expected: additions only.

- [ ] **Step 6: Generate and copy the report**

```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/template_gms_95_1.json \
  -ida-source docs/packets/ida-exports/gms_v95.json \
  -output /tmp/rpt_gms_v95
grep -o '"verdict": *"[^"]*"' /tmp/rpt_gms_v95/gms_v95/MountFood.json
cp /tmp/rpt_gms_v95/gms_v95/MountFood.json /tmp/rpt_gms_v95/gms_v95/MountFood.md docs/packets/audits/gms_v95/
```

Expected: verdict `Match`.

- [ ] **Step 7: Append the v95 fixture case and marker**

Evidence-comment line `//   v95 ...@0x<v95 addr>: op 0x53; <derived order>`, marker:

```go
// packet-audit:verify packet=mount/serverbound/MountFood version=gms_v95 ida=0x<v95 addr>
```

case (bytes per the v95-derived order):

```go
		{pt.Variants[3], []byte{0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x80, 0x84, 0x1E, 0x00}}, // GMS v95
```

- [ ] **Step 8: Run the test**

Run: `cd libs/atlas-packet && go test -race ./mount/... -v`
Expected: PASS including `TestFoodByteFixture/GMS_v95`.

- [ ] **Step 9: Regenerate the matrix and verify promotion**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check 2>&1 | grep -i "mount\|MountFood" || true
grep "USE_MOUNT_FOOD" docs/packets/audits/STATUS.md
```

Expected: v83/v84/v87/v95 ✅; no new problem lines.

- [ ] **Step 10: Commit**

```bash
git add libs/atlas-packet/mount/serverbound/food_test.go \
  docs/packets/ida-exports/gms_v95.json \
  docs/packets/audits/gms_v95/MountFood.json docs/packets/audits/gms_v95/MountFood.md \
  docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "verify(packets): USE_MOUNT_FOOD gms_v95 byte-verified (0x53 @<v95 addr>)"
```

---

### Task 6: Verify jms_v185 (opcode 69 / 0x45 expected)

**Files:**
- Modify: `docs/packets/ida-exports/gms_jms_185.json` (NOTE: jms's export filename)
- Create: `docs/packets/audits/jms_v185/MountFood.json`, `docs/packets/audits/jms_v185/MountFood.md`
- Modify: `libs/atlas-packet/mount/serverbound/food_test.go` (append jms case + marker)
- Modify (regenerated): `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

**Interfaces:**
- Consumes: Task 2's `TestFoodByteFixture` cases table and marker block.

- [ ] **Step 1: Select the jms DEVM IDA instance**

`mcp__ida-pro__list_instances` → the JMS v185 **`*_U_DEVM`** build (the retail dump is SMC/control-flow-virtualized — Hex-Rays fails on it; verify the instance metadata names the DEVM binary before decompiling) → `mcp__ida-pro__select_instance(port)`. Quote metadata.

- [ ] **Step 2: Locate the send function**

`mcp__ida-pro__func_query` `name_regex: "SendTamingMobFoodItemUseRequest"`. If unnamed: byte-signature `6A 45 8D 8D ?? ?? ?? ?? E8`, structure-match to the v83 twin, rename + `idb_save`. Still unlocatable → STOP AND ASK.

- [ ] **Step 3: Decompile and record ground truth**

Opcode expected 69 (`docs/packets/registry/jms_v185.yaml:2308` / `template_jms_185_1.json:325`); record the encode order. Branches as in Task 4 ((a) registry+template fix own commit + PR callout; (b) version-gated codec fix own commit — a jms-only divergence gates on `Region == "JMS"` per existing atlas-packet patterns; (c) STOP AND ASK).

- [ ] **Step 4: Harvest to a temp export**

The jms version key is `gms_jms_185` (its committed export is `gms_jms_185.json`):

```bash
printf 'CWvsContext::SendTamingMobFoodItemUseRequest\n' > /tmp/mountfood_roster.md
go run ./tools/packet-audit export \
  -version gms_jms_185 \
  -ida-url http://192.168.20.3:13337/mcp -ida-port <jms DEVM port from Step 1> \
  -prior-export "" -pending /tmp/mountfood_roster.md \
  -descent-depth 12 \
  -output /tmp/harvest_gms_jms_185.json
```

- [ ] **Step 5: Surgically splice into the committed export**

```bash
python3 - <<'EOF'
import json

COMMITTED = 'docs/packets/ida-exports/gms_jms_185.json'
HARVEST = '/tmp/harvest_gms_jms_185.json'
FNAME = 'CWvsContext::SendTamingMobFoodItemUseRequest'

c = json.load(open(COMMITTED))
h = json.load(open(HARVEST))

entry = h['functions'][FNAME]
entry['calls'] = [x for x in (entry.get('calls') or [])
                  if not (x.get('op') == 'Delegate' and 'COutPacket' in (x.get('ref') or ''))]
c['functions'][FNAME] = entry

added = [FNAME]
for k, v in h['functions'].items():
    if k != FNAME and k not in c['functions']:
        c['functions'][k] = v
        added.append(k)

with open(COMMITTED, 'w') as f:
    f.write(json.dumps(c, indent=2) + '\n')
print('spliced:', added)
EOF
git diff --stat docs/packets/ida-exports/gms_jms_185.json
```

Expected: additions only.

- [ ] **Step 6: Generate and copy the report**

The root command maps version `gms_jms_185` to audit dir `jms_v185` automatically (`tools/packet-audit/cmd/root.go:204-208`) — copy from that subdir explicitly:

```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/template_jms_185_1.json \
  -ida-source docs/packets/ida-exports/gms_jms_185.json \
  -output /tmp/rpt_jms_v185
grep -o '"verdict": *"[^"]*"' /tmp/rpt_jms_v185/jms_v185/MountFood.json
cp /tmp/rpt_jms_v185/jms_v185/MountFood.json /tmp/rpt_jms_v185/jms_v185/MountFood.md docs/packets/audits/jms_v185/
```

Expected: verdict `Match`.

- [ ] **Step 7: Append the jms fixture case and marker**

Evidence-comment line `//   jms185 ...@0x<jms addr>: op 0x45; <derived order>`, marker (the marker version key is `jms_v185`, matching the audit dir — same convention as the pet/storage markers):

```go
// packet-audit:verify packet=mount/serverbound/MountFood version=jms_v185 ida=0x<jms addr>
```

case (bytes per the jms-derived order):

```go
		{pt.Variants[4], []byte{0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x80, 0x84, 0x1E, 0x00}}, // JMS v185
```

- [ ] **Step 8: Run the test**

Run: `cd libs/atlas-packet && go test -race ./mount/... -v`
Expected: PASS including `TestFoodByteFixture/JMS_v185`.

- [ ] **Step 9: Regenerate the matrix and verify full promotion**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check 2>&1 | grep -i "mount\|MountFood" || true
grep "USE_MOUNT_FOOD" docs/packets/audits/STATUS.md
```

Expected: the `USE_MOUNT_FOOD` row reads ✅ in all five columns; no new problem lines; conflict count not increased.

- [ ] **Step 10: Commit**

```bash
git add libs/atlas-packet/mount/serverbound/food_test.go \
  docs/packets/ida-exports/gms_jms_185.json \
  docs/packets/audits/jms_v185/MountFood.json docs/packets/audits/jms_v185/MountFood.md \
  docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "verify(packets): USE_MOUNT_FOOD jms_v185 byte-verified (0x45 @<jms addr>)"
```

---

### Task 7: Final gates and parked-v92 bookkeeping

**Files:**
- Modify: `docs/tasks/task-138-mount-food-verification/context.md` (record final per-version addresses/opcodes and the narrowed v92 gap)
- No code files — verification and bookkeeping only.

**Interfaces:**
- Consumes: all five ✅ cells and the five per-version commits.

- [ ] **Step 1: Full-module test and vet gates**

```bash
cd libs/atlas-packet && go test -race ./... && go vet ./...
cd ../../tools/packet-audit && go test -race ./... && go vet ./...
```

Expected: all clean/PASS. (No `docker buildx bake` — no service `go.mod` was touched. If branch (a) fired anywhere, template JSON edits still don't touch `go.mod`; if branch (b) fired, only `libs/atlas-packet` changed.)

- [ ] **Step 2: Repo-root guard + final matrix state**

```bash
cd ../.. && tools/redis-key-guard.sh
go run ./tools/packet-audit matrix --check 2>&1 | tail -20
grep "USE_MOUNT_FOOD" docs/packets/audits/STATUS.md
grep -A 3 '"op": "USE_MOUNT_FOOD"' docs/packets/audits/status.json | head -20
```

Expected: redis-key-guard clean; STATUS.md row shows ✅×5; the status.json cells no longer read `incomplete`/"no audit report"; `matrix --check` tail shows no orphan/dangling/stale/drift lines for MountFood and the same conflict count as before Task 1.

- [ ] **Step 3: Record the verification summary in context.md**

Append to `docs/tasks/task-138-mount-food-verification/context.md` a "Results" section listing, per version: the IDB instance used, the function address, the `COutPacket` opcode as decompiled, the derived encode order, and which (if any) discrepancy branches fired. State explicitly: "The only remaining mount-food gap is the v92 inbound registration (`MountFoodHandle` absent from `template_gms_92_1.json`), still blocked solely on the absence of a v92 IDB. No v92 values were inferred."

- [ ] **Step 4: Commit the bookkeeping**

```bash
git add docs/tasks/task-138-mount-food-verification/context.md
git commit -m "docs(task-138): record verification results and narrowed v92 gap"
```

- [ ] **Step 5: Orchestrator-only memory update (not a repo commit)**

The session orchestrator (not an implementation subagent) updates the project-memory topic `project_v92_mount_food_parked`: all five `USE_MOUNT_FOOD` cells are now byte-verified (task-138); the sole remaining gap is the v92 inbound handler registration, blocked on a v92 IDB.

---

## Execution notes for the orchestrator

- Dispatch Tasks 2–6 **sequentially, never in parallel** (shared IDA instance selection, shared `food_test.go`, global matrix).
- Every implementer subagent prompt must `cd` into this worktree first and verify `git branch --show-current` = `task-138-mount-food-verification` after each commit.
- If any task reports a stop-and-ask condition (missing IDB, unlocatable function), halt the campaign at that version and surface it — the earlier versions' commits stand on their own (per-version cell commits keep the tree green).
- Before the PR: run `superpowers:requesting-code-review` (plan-adherence + backend reviewers), per CLAUDE.md.
