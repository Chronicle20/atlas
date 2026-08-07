# Mount Food Packet Verification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Promote **all nine** serverbound `USE_MOUNT_FOOD` coverage-matrix cells (gms_v48, gms_v61, gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v95, jms_v185) from ❌/`n-a` to ✅ with IDA-derived byte-level evidence — and correct the false gms_v48 `n-a`.

**Architecture:** One tooling-linkage change (`candidatesFromFName` case) makes the packet gradeable; one test-context change adds the four legacy tenant variants; then nine strictly-serialized per-version verification passes each produce co-committed artifacts (export splice, audit report, `packet-audit:verify` marker, regenerated matrix). gms_v48 additionally gains a registry op and a seed-template `MountFoodHandle` registration (its cell is currently `n-a` because the op is absent from its registry). The matrix is regenerated (never hand-edited) after each pass.

**Tech Stack:** Go (libs/atlas-packet, tools/packet-audit), ida-pro-mcp (session-based decompilation), python3 (surgical JSON export splice).

**Scope note (2026-07-25):** v1 of this plan covered five columns; main has since brought up the legacy columns, so the row is nine wide. v48/v61/v72/v79 were added per maintainer direction. **gms_12** stays parked (no v12 IDB). **gms_92** mount food is unblocked (IDB exists, opcode `0x54` verified) but gms_92 is **not a matrix column** — closing it is an optional one-line template registration, out of this task's matrix scope unless folded in (see PRD §9).

## Global Constraints

- **Serialize the shared mutating steps.** Tasks 3–11 run one at a time, in order v48 → v61 → v72 → v79 → v83 → v84 → v87 → v95 → jms_v185. The session-based IDA API scopes reads per `database` id, but all nine markers land in one `food_test.go`, the matrix regen is global, and v48 mutates the shared registry/template — never run two version passes in parallel.
- **Grounding:** every opcode and fixture byte must trace to a decompiled function address in the matching IDB. The integer in `COutPacket::COutPacket(&pkt, OPCODE)` is the opcode ground truth — distrust the IDB symbol name and the csv-seeded registry alike. Nothing inferred from version-shift patterns or memory.
- **Codec is version-invariant:** the send body is `update_time u32 · slot i16 · itemId u32` on every version (v48 decompile confirms it matches v83+). `food.go` needs **no** version gating and is not expected to change (discrepancy branch (b) is a contingency). Every byte-fixture variant asserts the same 10-byte body; the per-version evidence is the marker address, not a differing layout.
- **Export hygiene:** committed exports in `docs/packets/ida-exports/` are NOT idempotent. Harvest to a temp file, splice surgically, strip any `{op:"Delegate", ref:"COutPacket..."}` artifact. Never regenerate wholesale.
- **Matrix:** never hand-edit `STATUS.md`/`status.json`; always regenerate via `go run ./tools/packet-audit matrix`. The `matrix --check` bar is **no new problems** (zero new orphan/dangling/stale/drift/n-a-consistency lines mentioning this packet; conflict count not increased).
- **Tier-0 cell:** `USE_MOUNT_FOOD` is `tier1: false`. Do NOT pin evidence records — a tier-0 cell promotes on audit report + marker alone.
- **Commit unit is the cell:** each version's test marker + export splice + audit report + regenerated matrix (+ v48's registry/template) commit together.
- **Parked / out-of-scope:** gms_12 (no IDB, no matrix column, no inference); gms_92 matrix work (not a column); the clientbound `SetTamingMobInfo` writer; handler/processor/Kafka behavior in atlas-channel and atlas-mounts.
- **Stop-and-ask escalations:** a send function genuinely unlocatable in an IDB after regex + byte-signature attempts. Never substitute a fname, borrow another version's address, or fake a hash.
- All commands run from the worktree root (`.worktrees/task-138-mount-food-verification`). Committed files must never contain literal home/absolute paths.

## Baseline facts (verified against the repo + IDBs at planning time)

- Codec: `libs/atlas-packet/mount/serverbound/food.go` — `type Food struct { updateTime uint32; slot int16; itemId uint32 }`; Encode/Decode unconditional `ts(4) LE, slot(2) LE, itemId(4) LE` (no gates). Existing test `TestFoodDecode` stays.
- Per-version senders (all symbol-named `has_type:true` in the DEVM IDBs where checked this turn):

  | Version | IDB session (re-enumerate at exec) | Address | Opcode | Template `MountFoodHandle` | Registry op |
  |---|---|---|---|---|---|
  | gms_v48 | `0bb5f11a` `GMS_v48_1_DEVM` | `0x70e00b` | **0x3D (61)** decompile-verified | **absent — add at 0x3D** | **absent — add** |
  | gms_v61 | `965202bf` `GMS_v61.1_U_DEVM` | `0x831f44` | 0x4C (76) | present `0x4C` | present `ida-discovered` |
  | gms_v72 | `90e36cb0` `GMS_v72.1_U_DEVM` | `0x904419` | 0x4C (76) | present `0x4C` | present `ida-discovered` |
  | gms_v79 | `9a7d3642` `GMS_v79_1_DEVM` | `0x955781` | 0x4B (75) | present `0x4B` | present `ida-discovered` |
  | gms_v83 | derive at exec | derive | 0x4D (77) | present `0x4D` | present |
  | gms_v84 | `79511a2a` `GMS_v84.1_U_DEVM` | derive | 0x4D (77) | present `0x4D` | present |
  | gms_v87 | derive at exec | derive | 0x50 (80) | present `0x50` | present |
  | gms_v95 | derive at exec | derive | 0x53 (83) | present `0x53` | present |
  | jms_v185 | derive at exec | derive | 0x45 (69) | present `0x45` | present |

- v48 decompile (`0x70e00b`): `COutPacket(v9, 61)` → `Encode4(update_time) · Encode2(slot) · Encode4(itemId)`; guard `nItemID/10000 == 226` (taming-mob food category). No collision: `0x3D` is unused in `template_gms_48_1.json` (78 handler entries).
- `CWvsContext::SendTamingMobFoodItemUseRequest` is absent from all nine exports and has no `candidatesFromFName` case — that missing case is why every cell reads "no audit report".
- Direct analog: `CWvsContext::SendPetFoodItemUseRequest` (run.go ~1069) → `{name:"Food", pkg:"pet", dir:DirServerbound}`; same expected shape.
- `pt.Variants` (`test/context.go`): `[0]`v28, `[1]`v83, `[2]`v87, `[3]`v95, `[4]`jms185, `[5]`v84, `[6]`v86. **No legacy variants yet** — Task 2 appends `[7]`v48, `[8]`v61, `[9]`v72, `[10]`v79.
- Export JSON: keys `binary, md5, generated_at, functions`; 2-space indent; trailing newline; `functions` NOT sorted (append at end).
- jms mapping: version key `gms_jms_185`, export `gms_jms_185.json`, audit dir `jms_v185`; DEVM IDB only (retail dump is SMC).
- n-a derivation: a cell is `n-a` (opcode -1) when the op is absent from that version's registry. Adding the op to `gms_v48.yaml` reclassifies the cell to `incomplete`; report + marker then promote it to `verified`.

---

### Task 1: Link the fname to the codec in packet-audit

**Files:** Modify `tools/packet-audit/cmd/run.go` (the `candidatesFromFName` switch, beside `CWvsContext::SendPetFoodItemUseRequest` ~line 1069); modify `tools/packet-audit/cmd/disambiguation_test.go`.

- [ ] **Step 1: Regression table entries.** In `disambiguation_test.go` add `{"mount", "Food", "MountFood"}` to `TestQualifiedWriterName` and `{"mount", "Food", csvpkg.DirServerbound, "/mount/serverbound/"}` to `TestLocateAtlasFileDisambiguatesByPkg`.
- [ ] **Step 2: Run** `cd tools/packet-audit && go test ./cmd/ -run 'TestQualifiedWriterName|TestLocateAtlasFileDisambiguatesByPkg' -v` → PASS.
- [ ] **Step 3: Add the case** in `run.go` after `SendPetFoodItemUseRequest`:
  ```go
  case "CWvsContext::SendTamingMobFoodItemUseRequest":
      // USE_MOUNT_FOOD — taming-mob (mount) food. Codec mount/serverbound/Food
      // (handler MountFoodHandle). update_time u32 + slot i16 + itemId u32.
      return []candidate{{name: "Food", pkg: "mount", dir: csvpkg.DirServerbound}}
  ```
- [ ] **Step 4: Build/vet/test** `cd tools/packet-audit && go build ./... && go vet ./... && go test ./...` → clean/PASS.
- [ ] **Step 5: Commit.**
  ```bash
  git add tools/packet-audit/cmd/run.go tools/packet-audit/cmd/disambiguation_test.go
  git commit -m "feat(packet-audit): link SendTamingMobFoodItemUseRequest to mount/serverbound Food"
  ```

---

### Task 2: Add the four legacy tenant variants to the test context

**Files:** Modify `libs/atlas-packet/test/context.go`.

- [ ] **Step 1:** Append (never insert — positional `Variants[N]` refs must hold) after the existing `[6]`=GMS v86 entry:
  ```go
  {Name: "GMS v48", Region: "GMS", MajorVersion: 48, MinorVersion: 1}, // [7]
  {Name: "GMS v61", Region: "GMS", MajorVersion: 61, MinorVersion: 1}, // [8]
  {Name: "GMS v72", Region: "GMS", MajorVersion: 72, MinorVersion: 1}, // [9]
  {Name: "GMS v79", Region: "GMS", MajorVersion: 79, MinorVersion: 1}, // [10]
  ```
- [ ] **Step 2: Build** `cd libs/atlas-packet && go build ./... && go vet ./...` → clean. (No behavior yet; existing positional refs unchanged.)
- [ ] **Step 3: Commit.**
  ```bash
  git add libs/atlas-packet/test/context.go
  git commit -m "test(atlas-packet): add v48/v61/v72/v79 tenant variants for legacy fixtures"
  ```

---

## Canonical per-version verify procedure (Tasks 3–11)

Each version task below specifies its **parameters** and any **special steps**, then runs this procedure. Substitute the parameters (`$VER` version key, `$SESSION` IDB session, `$ADDR` send-fn address, `$OP` opcode hex, `$EXPORT` export filename, `$AUDITDIR` audit dir, `$TMPL` template file, `$VARIDX` variant index, `$WANT` fixture bytes).

**A. Select + verify session.** `mcp__ida-pro__idb_list` → confirm `$SESSION` (or its relaunched equivalent) is the target version; quote metadata before reading. Never hardcode a session id.

**B. Locate + decompile.** `mcp__ida-pro__func_query` `name_regex:"SendTamingMobFoodItemUseRequest"` on `database:$SESSION`. Legacy four are pre-named at `$ADDR`. If unnamed (only possible for some original-five): `mcp__ida-pro__find` bytes `6A <op> 8D 8D ?? ?? ?? ?? E8`, structure-match to the pet-food twin, `rename` + `idb_save`. Unlocatable → STOP AND ASK. Decompile; record the `COutPacket(&pkt, N)` integer (must equal `$OP`) and the ordered encode list. Handle discrepancy branches (a)/(b)/(c) — see the decision table — before proceeding.

**C. Harvest to temp.**
```bash
printf 'CWvsContext::SendTamingMobFoodItemUseRequest\n' > /tmp/mountfood_roster.md
go run ./tools/packet-audit export -version $VER \
  -ida-url http://192.168.20.3:13337/mcp -ida-port <port for $SESSION> \
  -prior-export "" -pending /tmp/mountfood_roster.md -descent-depth 12 \
  -output /tmp/harvest_$VER.json
```

**D. Surgical splice** (parameterized; strips the COutPacket-delegate artifact, adds absent helpers only):
```bash
VER=$VER EXPORT=docs/packets/ida-exports/$EXPORT HARVEST=/tmp/harvest_$VER.json python3 - <<'EOF'
import json, os
COMMITTED, HARVEST = os.environ['EXPORT'], os.environ['HARVEST']
FNAME = 'CWvsContext::SendTamingMobFoodItemUseRequest'
c = json.load(open(COMMITTED)); h = json.load(open(HARVEST))
entry = h['functions'][FNAME]
entry['calls'] = [x for x in (entry.get('calls') or [])
                  if not (x.get('op') == 'Delegate' and 'COutPacket' in (x.get('ref') or ''))]
c['functions'][FNAME] = entry
added = [FNAME]
for k, v in h['functions'].items():
    if k != FNAME and k not in c['functions']:
        c['functions'][k] = v; added.append(k)
open(COMMITTED, 'w').write(json.dumps(c, indent=2) + '\n')
print('spliced:', added)
EOF
git diff --stat docs/packets/ida-exports/$EXPORT   # additions only; if an existing key changed, checkout + redo
```

**E. Generate + copy report.**
```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template $TMPL -ida-source docs/packets/ida-exports/$EXPORT -output /tmp/rpt_$VER
grep -o '"verdict": *"[^"]*"' /tmp/rpt_$VER/$AUDITDIR/MountFood.json   # expect Match
cp /tmp/rpt_$VER/$AUDITDIR/MountFood.json /tmp/rpt_$VER/$AUDITDIR/MountFood.md docs/packets/audits/$AUDITDIR/
```
A non-Match is a real divergence → back to step B's branches; do not copy a non-Match in.

**F. Fixture case + marker.** In `libs/atlas-packet/mount/serverbound/food_test.go`, add the evidence-comment line `//   $VER ...@0x$ADDR: op $OP; Encode4(update_time)·Encode2(slot)·Encode4(itemId)`, the marker `// packet-audit:verify packet=mount/serverbound/MountFood version=$VER ida=0x$ADDR`, and the `cases` entry `{pt.Variants[$VARIDX], $WANT}, // $VER`. (Task 3 creates `TestFoodByteFixture`; Tasks 4–11 append.)

**G. Test.** `cd libs/atlas-packet && go test -race ./mount/... -v` → the new subtest + `TestFoodDecode` PASS.

**H. Regen + verify + commit.**
```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check 2>&1 | grep -i "mount\|MountFood" || true
grep "USE_MOUNT_FOOD" docs/packets/audits/STATUS.md
git add libs/atlas-packet/mount/serverbound/food_test.go docs/packets/ida-exports/$EXPORT \
  docs/packets/audits/$AUDITDIR/MountFood.json docs/packets/audits/$AUDITDIR/MountFood.md \
  docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "verify(packets): USE_MOUNT_FOOD $VER byte-verified ($OP @0x$ADDR)"
```
Expected: the `$VER` cell flips ✅; no new problem lines; conflict count unchanged.

**Discrepancy decision table (step B):**

| Finding | Action (same task, same branch) |
|---|---|
| (a) opcode ≠ registry/template | Fix `docs/packets/registry/<v>.yaml` + `$TMPL` to the IDB value as its own commit; note the live-tenant config-patch callout for the PR. |
| (b) order ≠ update_time u32/slot i16/itemId u32 | Version-gate `food.go` Encode+Decode FIRST as its own commit (`MajorAtLeast(N)` per atlas-packet patterns; update `TestFoodDecode`). **Not expected** (§version-invariant). |
| (c) function absent from the IDB | STOP AND ASK with the exact search evidence. |

The fixture body is `$WANT = []byte{0x64,0x00,0x00,0x00, 0x03,0x00, 0x80,0x84,0x1E,0x00}` for **every** version (ts=100, slot=3, itemId=2000000, LE) — identical because the codec is version-invariant; only the marker address differs. If branch (b) ever fired, hand-compute `$WANT` from that version's decompile instead.

---

### Task 3: Verify gms_v48 — **correction pass** (opcode 0x3D)

Parameters: `$VER=gms_v48`, `$SESSION=0bb5f11a`, `$ADDR=70e00b`, `$OP=0x3D`, `$EXPORT=gms_v48.json`, `$AUDITDIR=gms_v48`, `$TMPL=services/atlas-configurations/seed-data/templates/template_gms_48_1.json`, `$VARIDX=7`.

**Special steps — run these BEFORE procedure step E (report needs the registry op + template registration to grade the cell):**

- [ ] **S1: Add the registry op.** In `docs/packets/registry/gms_v48.yaml`, add the `USE_MOUNT_FOOD` serverbound op:
  ```yaml
  - op: USE_MOUNT_FOOD
    direction: serverbound
    opcode: 61
    fname: CWvsContext::SendTamingMobFoodItemUseRequest
    provenance: ida-discovered
    ida:
      address: 7397387   # 0x70e00b
    note: 'v48 SendTamingMobFoodItemUseRequest COutPacket(61) @0x70e00b; Encode4(update_time)·Encode2(slot)·Encode4(itemId); guard nItemID/10000==226 (taming-mob food). Corrects prior false n-a (op was absent from registry). task-138.'
  ```
  (Match the file's existing op-entry field order/style; place it in opcode order.)
- [ ] **S2: Register the handler.** In `template_gms_48_1.json`, add to the channel recv-handler list (opcode order; verified `0x3D` is unused):
  ```json
  { "opCode": "0x3D", "validator": "LoggedInValidator", "handler": "MountFoodHandle", "services": ["channel"] }
  ```
- [ ] **S3: Confirm no collision** before committing: `grep -niE '0x0?3d' template_gms_48_1.json` returns only your new entry.

**Then run the canonical procedure A–H.** Notes:
- Procedure step D splices `SendTamingMobFoodItemUseRequest` into `gms_v48.json` (currently absent).
- Procedure step F **creates** `TestFoodByteFixture` (the first version) — include the imports `bytes` and `pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"` if absent, and the full function skeleton:
  ```go
  // TestFoodByteFixture pins the exact serverbound wire bytes per version,
  // hand-computed from each version's decompiled send order (full body, never
  // opcode-only). Body is version-invariant (update_time u32·slot i16·itemId u32);
  // only the opcode differs, and it is config-resolved (template), not in the codec.
  // IDA evidence:
  //   gms_v48 SendTamingMobFoodItemUseRequest@0x70e00b: op 0x3D; Encode4(update_time)·Encode2(slot)·Encode4(itemId)
  // packet-audit:verify packet=mount/serverbound/MountFood version=gms_v48 ida=0x70e00b
  func TestFoodByteFixture(t *testing.T) {
      cases := []struct {
          variant pt.TenantVariant
          want    []byte
      }{
          {pt.Variants[7], []byte{0x64, 0x00, 0x00, 0x00, 0x03, 0x00, 0x80, 0x84, 0x1E, 0x00}}, // gms_v48
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
- Procedure step H additionally `git add`s `docs/packets/registry/gms_v48.yaml` and the template file; expect the gms_v48 cell to go `n-a → ✅`. Commit message: `verify(packets): USE_MOUNT_FOOD gms_v48 registered + byte-verified (0x3D @0x70e00b, corrects false n-a)`.

---

### Task 4: Verify gms_v61 (opcode 0x4C)

Parameters: `$VER=gms_v61`, `$SESSION=965202bf`, `$ADDR=831f44`, `$OP=0x4C`, `$EXPORT=gms_v61.json`, `$AUDITDIR=gms_v61`, `$TMPL=services/atlas-configurations/seed-data/templates/template_gms_61_1.json`, `$VARIDX=8`. Template + registry already carry the op — run canonical procedure A–H; step F **appends** the v61 case + marker to `TestFoodByteFixture`.

---

### Task 5: Verify gms_v72 (opcode 0x4C)

Parameters: `$VER=gms_v72`, `$SESSION=90e36cb0`, `$ADDR=904419`, `$OP=0x4C`, `$EXPORT=gms_v72.json`, `$AUDITDIR=gms_v72`, `$TMPL=services/atlas-configurations/seed-data/templates/template_gms_72_1.json`, `$VARIDX=9`. Run canonical procedure A–H (append case + marker).

---

### Task 6: Verify gms_v79 (opcode 0x4B)

Parameters: `$VER=gms_v79`, `$SESSION=9a7d3642`, `$ADDR=955781`, `$OP=0x4B`, `$EXPORT=gms_v79.json`, `$AUDITDIR=gms_v79`, `$TMPL=services/atlas-configurations/seed-data/templates/template_gms_79_1.json`, `$VARIDX=10`. Run canonical procedure A–H (append case + marker).

---

### Task 7: Verify gms_v83 (opcode 0x4D)

Parameters: `$VER=gms_v83`, `$SESSION=ce4ff298` (`MapleStory_dump.exe` v83 Me), `$ADDR=`derive, `$OP=0x4D`, `$EXPORT=gms_v83.json`, `$AUDITDIR=gms_v83`, `$TMPL=services/atlas-configurations/seed-data/templates/template_gms_83_1.json`, `$VARIDX=1`. Send fn may be unnamed → procedure step B naming path (`6A 4D 8D 8D ?? ?? ?? ?? E8`). Run A–H (append case + marker).

---

### Task 8: Verify gms_v84 (opcode 0x4D)

Parameters: `$VER=gms_v84`, `$SESSION=79511a2a` (`GMS_v84.1_U_DEVM`), `$ADDR=`derive, `$OP=0x4D`, `$EXPORT=gms_v84.json`, `$AUDITDIR=gms_v84`, `$TMPL=services/atlas-configurations/seed-data/templates/template_gms_84_1.json`, `$VARIDX=5`. **If no v84 instance can be brought up, STOP AND ASK** — a missing IDB is a genuine blocker; never copy v83's bytes/address. Run A–H (append case + marker).

---

### Task 9: Verify gms_v87 (opcode 0x50)

Parameters: `$VER=gms_v87`, `$SESSION=81f32170` (`GMSv87_4GB`), `$ADDR=`derive, `$OP=0x50`, `$EXPORT=gms_v87.json`, `$AUDITDIR=gms_v87`, `$TMPL=services/atlas-configurations/seed-data/templates/template_gms_87_1.json`, `$VARIDX=2`. Send fn may carry a mangled name → byte signature `6A 50 8D 8D ?? ?? ?? ?? E8` finds it. Run A–H (append case + marker).

---

### Task 10: Verify gms_v95 (opcode 0x53)

Parameters: `$VER=gms_v95`, `$SESSION=e4abcb98` (`GMS_v95.0_U_DEVM`), `$ADDR=`derive, `$OP=0x53`, `$EXPORT=gms_v95.json`, `$AUDITDIR=gms_v95`, `$TMPL=services/atlas-configurations/seed-data/templates/template_gms_95_1.json`, `$VARIDX=3`. Run A–H (append case + marker).

---

### Task 11: Verify jms_v185 (opcode 0x45)

Parameters: `$VER=gms_jms_185` (export key), `$SESSION=3c4bb8b1` — **verify it is the `*_U_DEVM` build, not the SMC retail dump** (`MapleStory_dump_SCY` is present as session `3c4bb8b1`; if Hex-Rays fails, the DEVM build is required — STOP AND ASK if only the SMC dump is available), `$ADDR=`derive, `$OP=0x45`, `$EXPORT=gms_jms_185.json`, `$AUDITDIR=jms_v185`, `$TMPL=services/atlas-configurations/seed-data/templates/template_jms_185_1.json`, `$VARIDX=4`. The root command maps `gms_jms_185` → audit dir `jms_v185` automatically. Marker version key is `jms_v185` (matches the audit dir). Run A–H (append case + marker).

---

### Task 12: Final gates and bookkeeping

**Files:** Modify `docs/tasks/task-138-mount-food-verification/context.md`. No code files.

- [ ] **Step 1: Full-module gates.**
  ```bash
  cd libs/atlas-packet && go test -race ./... && go vet ./...
  cd ../../tools/packet-audit && go test -race ./... && go vet ./...
  cd ../.. && tools/lint.sh --check
  ```
  All clean/PASS. No `docker buildx bake` (no `go.mod` touched).
- [ ] **Step 2: Repo-root guards + final matrix state.**
  ```bash
  tools/redis-key-guard.sh
  go run ./tools/packet-audit matrix --check 2>&1 | tail -20
  grep "USE_MOUNT_FOOD" docs/packets/audits/STATUS.md
  grep -A 30 '"op": "USE_MOUNT_FOOD"' docs/packets/audits/status.json | head -40
  ```
  Expected: guards clean; STATUS.md row ✅ ×9; status.json has no `incomplete`/`n-a` cells for USE_MOUNT_FOOD; `matrix --check` shows no orphan/dangling/stale/drift/n-a-consistency lines for MountFood and the same conflict count as before Task 1.
- [ ] **Step 3: Record results in context.md.** Append a "Results" section: per version — IDB session, function address, `COutPacket` opcode as decompiled, derived encode order, and any discrepancy branch that fired. State explicitly: "All nine `USE_MOUNT_FOOD` cells are byte-verified. gms_12 remains parked solely on the absence of a v12 IDB. gms_92 mount food is unblocked (v92 IDB, opcode 0x54 verified) and reduced to a one-line `template_gms_92_1.json` registration that is out of this task's matrix scope (gms_92 is not a matrix column). No opcodes were inferred."
- [ ] **Step 4: Commit bookkeeping.**
  ```bash
  git add docs/tasks/task-138-mount-food-verification/context.md
  git commit -m "docs(task-138): record 9-version verification results; narrowed v12/v92 gaps"
  ```
- [ ] **Step 5: Orchestrator-only memory update (not a repo commit).** Update project memory: all nine `USE_MOUNT_FOOD` matrix cells byte-verified (task-138), v48 false-n-a corrected; gms_12 parked on IDB; gms_92 reduced to a template-registration line (already reflected in `project_v92_mount_food_parked`).

---

## Execution notes for the orchestrator

- Dispatch Tasks 3–11 **sequentially, never in parallel** (shared `food_test.go`, global matrix, and — for v48 — shared registry/template).
- Every implementer subagent prompt must `cd` into this worktree first and verify `git branch --show-current` = `task-138-mount-food-verification` after each commit.
- If any task reports a stop-and-ask condition (missing IDB, unlocatable function), halt at that version and surface it — the earlier per-version commits stand on their own.
- Before the PR: run `superpowers:requesting-code-review` (plan-adherence + backend reviewers; `packet-completeness-critic` for the coverage-manifest delta), per CLAUDE.md, pinned to the cheaper model per project memory.
- **gms_92 fold-in** (PRD §9) is deliberately excluded; if the maintainer wants it, add one template line (opcode `0x54`) — no matrix cell results.
