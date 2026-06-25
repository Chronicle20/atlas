# Interaction Packet-Fixture Verification Campaign — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Drive the 12 remaining `incomplete` serverbound cells in the `interaction` packet family to `verified` (✅) across all applicable versions, each landed as the three coupled artifacts (byte-fixture marker + pinned evidence + audit report) with the matrix regenerated.

**Architecture:** Verification only — no new features. For each cell, follow `docs/packets/audits/VERIFYING_A_PACKET.md` §6–10. The wire codecs in `libs/atlas-packet/interaction/serverbound/` are uniform (un-gated) and already correct; the per-version work is adding `packet-audit:verify` marker rows to the existing tests, producing/copying the per-version audit report, pinning evidence, and regenerating the matrix. Two cell classes need a live-IDA harvest-and-splice first (TieAnswer v84; the eight merchant `#Merchant`-arm cells); the rest are report-gen-only.

**Tech Stack:** Go (`libs/atlas-packet`, single module — test files only), `tools/packet-audit` CLI, IDA Pro via ida-pro-mcp (`select_instance`), YAML evidence records.

**Read first:** `context.md` in this folder — it carries the confirmed 12-cell table, the export-presence grep, the exact tool invocations, the `pt.Variants` index map, and two corrections to design.md (template filenames have a `_1` suffix; the interaction route token varies by version).

---

## Conventions used in every task

- **Worktree.** All work happens in the `task-110-interaction-packet-fixtures` worktree (`<repo-root>/.worktrees/task-110-interaction-packet-fixtures`). Every subagent prompt must `cd` into that worktree first and, after each commit, verify `git rev-parse --show-toplevel` ends with `/.worktrees/task-110-interaction-packet-fixtures` and `git branch --show-current` is `task-110-interaction-packet-fixtures`.
- **Commands run from the worktree root** unless a step says otherwise. Go test/build commands run inside `libs/atlas-packet`.
- **Marker rows** are Go line comments placed directly above the test function they certify:
  `// packet-audit:verify packet=interaction/serverbound/<Struct> version=<key> ida=<0xaddr>`
  where `<Struct>` is the qualified writer name (e.g. `InteractionOperationInvite`) and `<key>` is one of `gms_v83 gms_v84 gms_v87 gms_v95 jms_v185`.
- **Per-cell artifact triple** (commit together): the marker row, the evidence YAML, and the regenerated `STATUS.md`/`status.json` — plus the copied audit report `.json`/`.md`.
- **`--check` bar:** `matrix --check` (and `fname-doc`/`operations --check`) may exit 1 from the pre-existing registry-seed conflict backlog. The bar is "no NEW problems": zero orphan/dangling/stale/drift lines mentioning an interaction packet, and the conflict count must not increase.
- **Address in the marker** is the per-version function address from the export (the `ida.address` the `evidence pin` resolves, or read it directly from the export JSON's function entry). Never invent an address — read it from `docs/packets/ida-exports/<export>.json`.

---

## Task 0: Baseline the matrix and the build

**Files:** none modified — this is a read-only baseline so later `--check` deltas are attributable.

- [ ] **Step 1: Confirm worktree + branch**

Run (from the task-110 worktree root):
```bash
git rev-parse --show-toplevel   # must end with /.worktrees/task-110-interaction-packet-fixtures
git branch --show-current        # must be task-110-interaction-packet-fixtures
```
Expected: both correct. If cwd is not the worktree, `cd` into `<repo-root>/.worktrees/task-110-interaction-packet-fixtures` first.

- [ ] **Step 2: Capture the pre-change matrix-check baseline**

Run:
```bash
go run ./tools/packet-audit matrix --check 2>&1 | tee /tmp/t110-matrix-baseline.txt; echo "exit ${PIPESTATUS[0]}"
go run ./tools/packet-audit fnamedoc --check 2>&1 | tee /tmp/t110-fnamedoc-baseline.txt; echo "exit ${PIPESTATUS[0]}"
go run ./tools/packet-audit operations --check 2>&1 | tee /tmp/t110-operations-baseline.txt; echo "exit ${PIPESTATUS[0]}"
```
Expected: each may exit 1 (pre-existing backlog). Record the conflict line count and confirm **no line mentions any interaction packet**. This is the reference for "no new problems".

> Note: subcommand spelling is `fnamedoc` / `operations` as registered in `tools/packet-audit/cmd/`. If a subcommand name differs at runtime, list `go run ./tools/packet-audit -h` and use the registered name; do not guess.

- [ ] **Step 3: Capture the baseline test/vet/build state of `libs/atlas-packet`**

Run:
```bash
cd libs/atlas-packet
go test -race ./... 2>&1 | tail -5
go vet ./... 2>&1 | tail -5
go build ./... 2>&1 | tail -5
cd ../..
```
Expected: clean (or record any pre-existing failure so it is not blamed on this task).

- [ ] **Step 4: Confirm the 12-cell starting state**

Run:
```bash
for p in InteractionOperationInvite InteractionOperationMemoryGameTieAnswer InteractionOperationMerchantPutItem InteractionOperationMerchantRemoveItem; do
  echo "=== $p ==="
  jq -r --arg p "$p" '.rows[] | select((.packet|type=="string") and (.packet|endswith($p))) | (.cells | to_entries | map("\(.key)=\(.value.state)") | join(" "))' docs/packets/audits/status.json
done
```
Expected (matches `context.md`):
```
InteractionOperationInvite:                gms_v83=incomplete gms_v84=verified gms_v87=incomplete gms_v95=verified jms_v185=incomplete
InteractionOperationMemoryGameTieAnswer:   gms_v83=verified gms_v84=incomplete gms_v87=verified gms_v95=verified jms_v185=verified
InteractionOperationMerchantPutItem:       gms_v83=incomplete gms_v84=incomplete gms_v87=incomplete gms_v95=verified jms_v185=incomplete
InteractionOperationMerchantRemoveItem:    gms_v83=incomplete gms_v84=incomplete gms_v87=incomplete gms_v95=verified jms_v185=incomplete
```

- [ ] **Step 5: No commit** (read-only baseline).

---

## Phase A — Invite (Class A, 3 cells: v83, v87, jms)

No live IDA. `CField::SendInviteTradingRoomMsg` is already in every committed export. Establishes the report-gen → marker → evidence → matrix loop before the harder phases.

### Task A1: Invite — gms_v83

**Files:**
- Modify: `libs/atlas-packet/interaction/serverbound/operation_invite_test.go` (add marker row)
- Create: `docs/packets/evidence/gms_v83/interaction.serverbound.InteractionOperationInvite.yaml`
- Create: `docs/packets/audits/gms_v83/InteractionOperationInvite.json` + `.md`
- Regenerate: `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

- [ ] **Step 1: Read the per-version function address from the export**

Run:
```bash
jq -r '.functions["CField::SendInviteTradingRoomMsg"].address // .functions["CField::SendInviteTradingRoomMsg"].Address' docs/packets/ida-exports/gms_v83.json
```
Expected: a `0x…` address. If the JSON shape differs, run
`jq '.functions["CField::SendInviteTradingRoomMsg"]' docs/packets/ida-exports/gms_v83.json`
and read the address field. Record it as `<ADDR_V83>`.

- [ ] **Step 2: Generate the audit report to a temp dir**

Run:
```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
  -ida-source docs/packets/ida-exports/gms_v83.json \
  -output /tmp/t110-rpt-gms_v83
```
Expected: writes `/tmp/t110-rpt-gms_v83/gms_v83/InteractionOperationInvite.{json,md}` (among other reports). Confirm:
```bash
cat /tmp/t110-rpt-gms_v83/gms_v83/InteractionOperationInvite.json
```
Expected: a report with `"Verdict": 0` (✅) and a single row for `targetCharacterId`. If `Verdict` is non-zero, STOP — a non-✅ verdict means a real wire delta or linkage problem; surface it (this is the §6 fix-first contingency), do not paper over it.

- [ ] **Step 3: Copy the report into the committed audit dir**

Run:
```bash
mkdir -p docs/packets/audits/gms_v83
cp /tmp/t110-rpt-gms_v83/gms_v83/InteractionOperationInvite.json docs/packets/audits/gms_v83/
cp /tmp/t110-rpt-gms_v83/gms_v83/InteractionOperationInvite.md   docs/packets/audits/gms_v83/
```

- [ ] **Step 4: Add the marker row to the test**

In `libs/atlas-packet/interaction/serverbound/operation_invite_test.go`, add one line to the existing marker block above `TestOperationInviteRoundTrip` (using `<ADDR_V83>` from Step 1):
```go
// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=gms_v83 ida=<ADDR_V83>
```
The block then reads (order not significant):
```go
// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=gms_v95 ida=0x52e9e0
// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=gms_v84 ida=0x53bc2a
// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=gms_v83 ida=<ADDR_V83>
func TestOperationInviteRoundTrip(t *testing.T) {
```

- [ ] **Step 5: Run the test to confirm it still passes**

Run:
```bash
cd libs/atlas-packet && go test ./interaction/serverbound/ -run TestOperationInviteRoundTrip -v 2>&1 | tail -20; cd ../..
```
Expected: PASS for all variants (the codec is uniform; the marker is metadata, not new assertions).

- [ ] **Step 6: Pin evidence**

Run:
```bash
go run ./tools/packet-audit evidence pin \
  --packet interaction/serverbound/InteractionOperationInvite \
  --version gms_v83 --ida "CField::SendInviteTradingRoomMsg" --category TIER1-FIXTURE
```
Expected: writes `docs/packets/evidence/gms_v83/interaction.serverbound.InteractionOperationInvite.yaml`. If it errors "not in export", STOP — the fname citation is unresolvable (it should resolve; the export grep confirmed presence).

- [ ] **Step 7: Hand-add the `verifies:` line to the evidence YAML**

Append to `docs/packets/evidence/gms_v83/interaction.serverbound.InteractionOperationInvite.yaml`:
```yaml
verifies:
    - libs/atlas-packet/interaction/serverbound/operation_invite_test.go#TestOperationInviteRoundTrip
```

- [ ] **Step 8: Regenerate the matrix and check promotion**

Run:
```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check 2>&1 | tee /tmp/t110-matrix-A1.txt; echo "exit ${PIPESTATUS[0]}"
jq -r '.rows[] | select((.packet|type=="string") and (.packet|endswith("InteractionOperationInvite"))) | .cells.gms_v83.state' docs/packets/audits/status.json
```
Expected: the `gms_v83` cell now prints `verified`. Diff `/tmp/t110-matrix-A1.txt` against `/tmp/t110-matrix-baseline.txt`: **no new** orphan/dangling/stale/drift line mentioning an interaction packet, conflict count unchanged.

- [ ] **Step 9: Commit the artifact triple**

Run:
```bash
git add libs/atlas-packet/interaction/serverbound/operation_invite_test.go \
        docs/packets/evidence/gms_v83/interaction.serverbound.InteractionOperationInvite.yaml \
        docs/packets/audits/gms_v83/InteractionOperationInvite.json \
        docs/packets/audits/gms_v83/InteractionOperationInvite.md \
        docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "verify(interaction): InteractionOperationInvite gms_v83 byte-fixture + evidence"
git rev-parse --show-toplevel; git branch --show-current
```
Expected: commit succeeds; toplevel ends `/.worktrees/task-110-interaction-packet-fixtures`; branch is `task-110-interaction-packet-fixtures`.

### Task A2: Invite — gms_v87

**Files:** same shape as A1 but `gms_v87` / `template_gms_87_1.json` / `docs/packets/ida-exports/gms_v87.json` / `docs/packets/audits/gms_v87/` / `docs/packets/evidence/gms_v87/`.

- [ ] **Step 1: Read the address** — `jq -r '.functions["CField::SendInviteTradingRoomMsg"].address' docs/packets/ida-exports/gms_v87.json` → `<ADDR_V87>`.
- [ ] **Step 2: Report-gen** — same command as A1 Step 2 with `-template services/atlas-configurations/seed-data/templates/template_gms_87_1.json -ida-source docs/packets/ida-exports/gms_v87.json -output /tmp/t110-rpt-gms_v87`. Confirm `/tmp/t110-rpt-gms_v87/gms_v87/InteractionOperationInvite.json` has `"Verdict": 0`; STOP if not.
- [ ] **Step 3: Copy report** — `mkdir -p docs/packets/audits/gms_v87 && cp /tmp/t110-rpt-gms_v87/gms_v87/InteractionOperationInvite.{json,md} docs/packets/audits/gms_v87/`.
- [ ] **Step 4: Add marker row** to `operation_invite_test.go`:
  `// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=gms_v87 ida=<ADDR_V87>`.
- [ ] **Step 5: Run test** — `cd libs/atlas-packet && go test ./interaction/serverbound/ -run TestOperationInviteRoundTrip -v 2>&1 | tail -20; cd ../..` → PASS.
- [ ] **Step 6: Pin evidence** — `go run ./tools/packet-audit evidence pin --packet interaction/serverbound/InteractionOperationInvite --version gms_v87 --ida "CField::SendInviteTradingRoomMsg" --category TIER1-FIXTURE`.
- [ ] **Step 7: Hand-add `verifies:`** line (same test path as A1 Step 7) to `docs/packets/evidence/gms_v87/interaction.serverbound.InteractionOperationInvite.yaml`.
- [ ] **Step 8: Regenerate + check** — `go run ./tools/packet-audit matrix && go run ./tools/packet-audit matrix --check`; confirm `gms_v87` cell = `verified`; no new interaction lines vs baseline.
- [ ] **Step 9: Commit** — `git add` the six paths (`gms_v87` variants), `git commit -m "verify(interaction): InteractionOperationInvite gms_v87 byte-fixture + evidence"`, verify toplevel + branch.

### Task A3: Invite — jms_v185

**Files:** same shape, `jms_v185` / `template_jms_185_1.json` / `docs/packets/ida-exports/gms_jms_185.json` / `docs/packets/audits/jms_v185/` / `docs/packets/evidence/jms_v185/`.

> Note the asymmetry: the export file is `gms_jms_185.json` but the version **key** (template, audit dir, evidence dir, marker `version=`) is `jms_v185`. Do not conflate them.

- [ ] **Step 1: Read the address** — `jq -r '.functions["CField::SendInviteTradingRoomMsg"].address' docs/packets/ida-exports/gms_jms_185.json` → `<ADDR_JMS>`.
- [ ] **Step 2: Report-gen** — same command with `-template services/atlas-configurations/seed-data/templates/template_jms_185_1.json -ida-source docs/packets/ida-exports/gms_jms_185.json -output /tmp/t110-rpt-jms_v185`. Confirm `/tmp/t110-rpt-jms_v185/jms_v185/InteractionOperationInvite.json` has `"Verdict": 0`; STOP if not.
- [ ] **Step 3: Copy report** — `mkdir -p docs/packets/audits/jms_v185 && cp /tmp/t110-rpt-jms_v185/jms_v185/InteractionOperationInvite.{json,md} docs/packets/audits/jms_v185/`.
- [ ] **Step 4: Add marker row** — `// packet-audit:verify packet=interaction/serverbound/InteractionOperationInvite version=jms_v185 ida=<ADDR_JMS>`.
- [ ] **Step 5: Run test** → PASS.
- [ ] **Step 6: Pin evidence** — `--version jms_v185 --ida "CField::SendInviteTradingRoomMsg"`.
- [ ] **Step 7: Hand-add `verifies:`** to `docs/packets/evidence/jms_v185/interaction.serverbound.InteractionOperationInvite.yaml`.
- [ ] **Step 8: Regenerate + check** — confirm `jms_v185` cell = `verified`; no new interaction lines.
- [ ] **Step 9: Commit** — `git commit -m "verify(interaction): InteractionOperationInvite jms_v185 byte-fixture + evidence"`, verify toplevel + branch.

### Task A4: Invite gate — all five versions ✅

- [ ] **Step 1: Confirm full-row promotion**

Run:
```bash
jq -r '.rows[] | select((.packet|type=="string") and (.packet|endswith("InteractionOperationInvite"))) | (.cells | to_entries | map("\(.key)=\(.value.state)") | join(" "))' docs/packets/audits/status.json
```
Expected: `gms_v83=verified gms_v84=verified gms_v87=verified gms_v95=verified jms_v185=verified`.

- [ ] **Step 2: No commit** (verification gate only).

---

## Phase B — TieAnswer v84 (Class E, 1 cell)

`CMemoryGameDlg::OnTieRequest` is absent from the v84 export but ✅ on the byte-identical v83 twin. Name the present-but-unnamed v84 function against the v83 twin, surgically splice the single absent entry into `gms_v84.json`, then run the Class-A loop.

### Task B1: Name `OnTieRequest` in the v84 IDB and splice it into the export

**Files:**
- Modify: `docs/packets/ida-exports/gms_v84.json` (absent-only splice of `CMemoryGameDlg::OnTieRequest`)

- [ ] **Step 1: Select the v84 IDA instance**

Use ida-pro-mcp. List instances and select the one whose loaded IDB is the v84 binary (do not trust the port number blindly):
```
list_instances    → identify the GMS v84 binary by name
select_instance(<port for v84>)    # PRD says 13337; confirm by binary name
```
Expected: the active instance is the v84 binary.

- [ ] **Step 2: Confirm the v83 twin's read order (reference)**

Read the verified v83 `OnTieRequest`:
```bash
jq -r '.functions["CMemoryGameDlg::OnTieRequest"].address' docs/packets/ida-exports/gms_v83.json
```
Then in the v83 IDB (or from the committed export's `calls`), confirm the body is a single `Decode1`/`ReadBool` for `response` (matches `operation_memory_game_tie_answer.go` → `WriteBool(response)`). This is the structural template for the v84 match.

- [ ] **Step 3: Locate the present-but-unnamed v84 function**

In the v84 IDB, find the `OnTieRequest` send/handler by byte-signature + twin-structure match to v83 (per `VERIFYING_A_PACKET.md` §10: the `6A <op> 8D 8D ?? ?? ?? ?? E8` send signature uniquely locates a send site; structure-match to the named v83 twin). Confirm via `decompile` that the body matches the v83 twin (single bool/byte read) and that the `COutPacket(&pkt, OPCODE)` opcode matches PLAYER_INTERACTION's tie-answer sub-op. Record the v84 address as `<ADDR_TIE_V84>`.

- [ ] **Step 4: Name it in the IDB**

```
rename(<ADDR_TIE_V84>, "CMemoryGameDlg::OnTieRequest")
idb_save()
```
Expected: the function now resolves by name.

- [ ] **Step 5: Harvest the single function to a temp export**

Run the harvest pointed at the v84 instance, to a TEMP file (never overwrite the committed export):
```bash
# roster.md lists only CMemoryGameDlg::OnTieRequest
go run ./tools/packet-audit export --version gms_v84 \
  -prior-export "" -pending /tmp/t110-tie-roster.md -descent-depth 12 \
  -ida-url http://127.0.0.1:13337/mcp -ida-port 13337 \
  --output /tmp/t110-export-v84-tie.json
```
> The exact `export` flag names/spelling come from `go run ./tools/packet-audit export -h`; mirror the registered flags. The intent: harvest ONLY `OnTieRequest` from the v84 instance into a temp file. Confirm `/tmp/t110-export-v84-tie.json` contains a `CMemoryGameDlg::OnTieRequest` entry.

- [ ] **Step 6: Surgically splice the absent entry into the committed export**

Add ONLY `CMemoryGameDlg::OnTieRequest` (absent-only) from the temp file into `docs/packets/ida-exports/gms_v84.json`. Do not touch any other key. Strip any `COutPacket`-delegate artifact (`{op: Delegate, ref: COutPacket}`) from the spliced entry. First assert it is absent:
```bash
jq -e '.functions["CMemoryGameDlg::OnTieRequest"]' docs/packets/ida-exports/gms_v84.json && echo "ALREADY PRESENT — STOP" || echo "absent, ok to splice"
```
Expected: "absent, ok to splice". Then merge the single entry from the temp file and verify only that one key changed:
```bash
git diff --stat docs/packets/ida-exports/gms_v84.json   # after the splice: 1 file changed, additions only
jq -r '.functions["CMemoryGameDlg::OnTieRequest"].address' docs/packets/ida-exports/gms_v84.json
```
Expected: the address resolves; `git diff` shows an addition of exactly the one function key (no drift of unrelated keys).

- [ ] **Step 7: Sanity-check no unrelated drift**

Run:
```bash
git diff docs/packets/ida-exports/gms_v84.json | grep -E '^\-' | grep -v '^---' | head
```
Expected: **no removed lines** other than benign reformatting of the single spliced object. If many unrelated keys changed, the splice was a full re-export — revert and redo absent-only.

- [ ] **Step 8: Commit the export splice on its own**

Run:
```bash
git add docs/packets/ida-exports/gms_v84.json
git commit -m "export(gms_v84): splice CMemoryGameDlg::OnTieRequest (named from v83 twin)"
git rev-parse --show-toplevel; git branch --show-current
```

### Task B2: TieAnswer v84 — verify the cell

**Files:**
- Modify: `libs/atlas-packet/interaction/serverbound/operation_memory_game_tie_answer_test.go`
- Create: `docs/packets/evidence/gms_v84/interaction.serverbound.InteractionOperationMemoryGameTieAnswer.yaml`
- Create: `docs/packets/audits/gms_v84/InteractionOperationMemoryGameTieAnswer.{json,md}`
- Regenerate: `STATUS.md`, `status.json`

- [ ] **Step 1: Report-gen**

Run:
```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/template_gms_84_1.json \
  -ida-source docs/packets/ida-exports/gms_v84.json \
  -output /tmp/t110-rpt-gms_v84
cat /tmp/t110-rpt-gms_v84/gms_v84/InteractionOperationMemoryGameTieAnswer.json
```
Expected: `"Verdict": 0`, single bool/byte row. STOP if non-✅.

- [ ] **Step 2: Copy report** — `mkdir -p docs/packets/audits/gms_v84 && cp /tmp/t110-rpt-gms_v84/gms_v84/InteractionOperationMemoryGameTieAnswer.{json,md} docs/packets/audits/gms_v84/`.

- [ ] **Step 3: Add marker row** to `operation_memory_game_tie_answer_test.go` (address = `<ADDR_TIE_V84>` from B1 Step 3):
```go
// packet-audit:verify packet=interaction/serverbound/InteractionOperationMemoryGameTieAnswer version=gms_v84 ida=<ADDR_TIE_V84>
```

- [ ] **Step 4: Run test** — `cd libs/atlas-packet && go test ./interaction/serverbound/ -run TestOperationMemoryGameTieAnswerRoundTrip -v 2>&1 | tail -20; cd ../..` → PASS.

- [ ] **Step 5: Pin evidence** — `go run ./tools/packet-audit evidence pin --packet interaction/serverbound/InteractionOperationMemoryGameTieAnswer --version gms_v84 --ida "CMemoryGameDlg::OnTieRequest" --category TIER1-FIXTURE`.

- [ ] **Step 6: Hand-add `verifies:`** to the new YAML:
```yaml
verifies:
    - libs/atlas-packet/interaction/serverbound/operation_memory_game_tie_answer_test.go#TestOperationMemoryGameTieAnswerRoundTrip
```

- [ ] **Step 7: Regenerate + check** — `go run ./tools/packet-audit matrix && go run ./tools/packet-audit matrix --check`; confirm:
```bash
jq -r '.rows[] | select((.packet|type=="string") and (.packet|endswith("InteractionOperationMemoryGameTieAnswer"))) | (.cells | to_entries | map("\(.key)=\(.value.state)") | join(" "))' docs/packets/audits/status.json
```
Expected: all five = `verified`. No new interaction lines vs baseline.

- [ ] **Step 8: Commit the artifact triple**

Run:
```bash
git add libs/atlas-packet/interaction/serverbound/operation_memory_game_tie_answer_test.go \
        docs/packets/evidence/gms_v84/interaction.serverbound.InteractionOperationMemoryGameTieAnswer.yaml \
        docs/packets/audits/gms_v84/InteractionOperationMemoryGameTieAnswer.json \
        docs/packets/audits/gms_v84/InteractionOperationMemoryGameTieAnswer.md \
        docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "verify(interaction): InteractionOperationMemoryGameTieAnswer gms_v84 byte-fixture + evidence"
git rev-parse --show-toplevel; git branch --show-current
```

---

## Phase C — Merchant put/remove (Class E-arm, 8 cells)

`CPersonalShopDlg::PutItem` / `::MoveItemToInventory` are named on every version, but the entrusted-merchant arm (`#Merchant` case key) was harvested only on v95. For each of v83/v84/v87/jms, decompile the base function, locate the merchant send arm, confirm its write order matches the Atlas codec, splice the `#Merchant` entry, then run the Class-A loop.

**Serialize by IDB:** one `select_instance` at a time. Process all four cells for a single version (PutItem + RemoveItem) while that IDB is selected, then move to the next version. Order: **v83 → v87 → jms → v84** (v84 last; it is the byte-identical v83 twin, so v83's confirmed arm is the structural template).

**Reference precedent:** `operation_merchant_buy_test.go` — `TestOperationMerchantBuyBytes` pins the hex bytes for the `#Merchant` buy arm and carries v83/v95 markers; its comment documents that the merchant arm shares the base `CPersonalShopDlg::BuyItem` and carries the same body across versions. The two new fixtures mirror this exactly.

### Task C0: Add hex-pin byte fixtures to the two merchant tests (shared, version-independent)

The merchant codecs are uniform, so a single deterministic hex-pin per struct certifies the wire bytes for all versions (MerchantBuy precedent). Add these once, up front; the per-version markers attach to them in C1–C4.

**Files:**
- Modify: `libs/atlas-packet/interaction/serverbound/operation_merchant_put_item_test.go`
- Modify: `libs/atlas-packet/interaction/serverbound/operation_merchant_remove_item_test.go`

- [ ] **Step 1: Write the hex-pin test for MerchantPutItem**

Append to `operation_merchant_put_item_test.go` (compute the expected hex from the codec: `WriteByte(2)` → `02`; `WriteInt16(7)` LE → `0700`; `WriteShort(15)` LE → `0f00`; `WriteShort(4)` LE → `0400`; `WriteInt(2000=0x7d0)` LE → `d0070000`):
```go
// TestOperationMerchantPutItemBytes pins the wire bytes for the entrusted-merchant
// put-item arm: byte inventoryType, int16 slot (LE), uint16 quantity (LE),
// uint16 set (LE), uint32 price (LE). The #Merchant arm shares the base
// CPersonalShopDlg::PutItem and carries the same body as the all-versions-verified
// OperationPersonalStorePutItem; the codec has no MajorVersion() gate.
func TestOperationMerchantPutItemBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	input := OperationMerchantPutItem{inventoryType: 2, slot: 7, quantity: 15, set: 4, price: 2000}
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	// 02 | 0700 | 0f00 | 0400 | d0070000
	want := "0207000f000400d0070000"
	if got != want {
		t.Errorf("bytes: got %s, want %s", got, want)
	}
}
```
Add the imports `"encoding/hex"` and `testlog "github.com/sirupsen/logrus/hooks/test"` (mirror `operation_merchant_buy_test.go`).

- [ ] **Step 2: Run it to confirm the expected hex is right**

Run:
```bash
cd libs/atlas-packet && go test ./interaction/serverbound/ -run TestOperationMerchantPutItemBytes -v 2>&1 | tail -15; cd ../..
```
Expected: PASS. If it FAILS with a different `got`, the hand-computed `want` was wrong — replace `want` with the actual `got` (the codec is the source of truth) and re-run to green. Do not change the codec.

- [ ] **Step 3: Write the hex-pin test for MerchantRemoveItem**

Append to `operation_merchant_remove_item_test.go` (`WriteShort(42=0x2a)` LE → `2a00`):
```go
// TestOperationMerchantRemoveItemBytes pins the wire bytes for the entrusted-merchant
// remove-item arm: a single uint16 index (LE). The #Merchant arm shares the base
// CPersonalShopDlg::MoveItemToInventory and carries the same body as the
// all-versions-verified OperationPersonalStoreRemoveItem; no MajorVersion() gate.
func TestOperationMerchantRemoveItemBytes(t *testing.T) {
	l, _ := testlog.NewNullLogger()
	ctx := pt.CreateContext("GMS", 83, 1)
	input := OperationMerchantRemoveItem{index: 42}
	got := hex.EncodeToString(input.Encode(l, ctx)(nil))
	// 2a00
	want := "2a00"
	if got != want {
		t.Errorf("bytes: got %s, want %s", got, want)
	}
}
```
Add the same two imports.

- [ ] **Step 4: Run it** — `cd libs/atlas-packet && go test ./interaction/serverbound/ -run TestOperationMerchantRemoveItemBytes -v 2>&1 | tail -15; cd ../..` → PASS (fix `want` from actual `got` if needed).

- [ ] **Step 5: vet + full package test**

Run:
```bash
cd libs/atlas-packet && go vet ./interaction/serverbound/ && go test ./interaction/serverbound/ 2>&1 | tail -5; cd ../..
```
Expected: clean PASS.

- [ ] **Step 6: Commit the two hex-pin fixtures (no markers yet)**

Run:
```bash
git add libs/atlas-packet/interaction/serverbound/operation_merchant_put_item_test.go \
        libs/atlas-packet/interaction/serverbound/operation_merchant_remove_item_test.go
git commit -m "test(interaction): hex-pin byte fixtures for merchant put/remove arms"
git rev-parse --show-toplevel; git branch --show-current
```

### Task C1: Merchant arms — gms_v83 (PutItem + RemoveItem)

**Files:**
- Modify: `docs/packets/ida-exports/gms_v83.json` (splice two `#Merchant` arms, absent-only)
- Modify: the two merchant test files (add `gms_v83` marker rows)
- Create: `docs/packets/evidence/gms_v83/interaction.serverbound.InteractionOperationMerchant{PutItem,RemoveItem}.yaml`
- Create: `docs/packets/audits/gms_v83/InteractionOperationMerchant{PutItem,RemoveItem}.{json,md}`
- Regenerate: `STATUS.md`, `status.json`

- [ ] **Step 1: Select the v83 IDA instance** — `list_instances` → select the v83 binary by name (PRD port 13341; confirm by name).

- [ ] **Step 2: Decompile the two base functions and locate the merchant arms**

```bash
jq -r '.functions["CPersonalShopDlg::PutItem"].address, .functions["CPersonalShopDlg::MoveItemToInventory"].address' docs/packets/ida-exports/gms_v83.json
```
`decompile` each. In each, locate the entrusted-merchant send arm — the same `#case`-keyed arm structure that produced the all-versions-present `BuyItem#Merchant`. Confirm:
- `PutItem#Merchant` write order = `byte inventoryType · int16 slot · uint16 quantity · uint16 set · uint32 price` (matches `operation_merchant_put_item.go` and the hex-pin in C0).
- `MoveItemToInventory#Merchant` write order = single `uint16 index` (matches `operation_merchant_remove_item.go`).
- The `COutPacket(&pkt, OPCODE)` opcode is the PLAYER_INTERACTION sub-op for the merchant put/remove mode (distrust the IDB symbol — the integer is truth). Record both arm addresses (`<ADDR_PUT_V83>`, `<ADDR_REMOVE_V83>`).

> **Wire-delta contingency (§6 of design):** if either arm's write order does NOT match the Atlas codec, STOP. That is a real wire delta — the codec fix is its own commit + its own review FIRST (fix-first), then resume. Watch for it but it is not expected (uniform codec, personal-store twin ✅ on all versions). If the arm is IDA-confirmed identical to personal-store with no distinct merchant send, that is the only path to a justified `n-a` — record the IDB-confirmed reason; do not infer `n-a` from the export's absence.

- [ ] **Step 3: Confirm the arms are absent in the committed export**

```bash
jq -e '.functions["CPersonalShopDlg::PutItem#Merchant"]' docs/packets/ida-exports/gms_v83.json && echo "PutItem#Merchant PRESENT — STOP" || echo "PutItem#Merchant absent, ok"
jq -e '.functions["CPersonalShopDlg::MoveItemToInventory#Merchant"]' docs/packets/ida-exports/gms_v83.json && echo "MoveItem#Merchant PRESENT — STOP" || echo "MoveItem#Merchant absent, ok"
```
Expected: both "absent, ok".

- [ ] **Step 4: Harvest the two arms to a temp export**

```bash
# roster lists CPersonalShopDlg::PutItem and CPersonalShopDlg::MoveItemToInventory
# (the harvester's #case-arm splitting emits the #Merchant entries, same as it did for BuyItem#Merchant)
go run ./tools/packet-audit export --version gms_v83 \
  -prior-export "" -pending /tmp/t110-merchant-roster.md -descent-depth 12 \
  -ida-url http://127.0.0.1:13341/mcp -ida-port 13341 \
  --output /tmp/t110-export-v83-merchant.json
jq -r '.functions | keys[] | select(test("#Merchant"))' /tmp/t110-export-v83-merchant.json
```
Expected: the temp export contains `CPersonalShopDlg::PutItem#Merchant` and `CPersonalShopDlg::MoveItemToInventory#Merchant`. (Confirm exact `export` flag names via `-h`.)

- [ ] **Step 5: Surgically splice both `#Merchant` arms (absent-only) into the committed export**

Add ONLY those two keys from the temp file into `docs/packets/ida-exports/gms_v83.json`. Leave the base `PutItem`/`MoveItemToInventory` keys untouched. Strip any `COutPacket`-delegate artifact from each spliced entry. Verify:
```bash
jq -r '.functions["CPersonalShopDlg::PutItem#Merchant"].address, .functions["CPersonalShopDlg::MoveItemToInventory#Merchant"].address' docs/packets/ida-exports/gms_v83.json
git diff docs/packets/ida-exports/gms_v83.json | grep -E '^\-' | grep -v '^---' | head
```
Expected: both addresses resolve; no removed lines (additions only — no unrelated drift).

- [ ] **Step 6: Commit the export splice on its own**

```bash
git add docs/packets/ida-exports/gms_v83.json
git commit -m "export(gms_v83): splice CPersonalShopDlg::{PutItem,MoveItemToInventory}#Merchant arms"
git rev-parse --show-toplevel; git branch --show-current
```

- [ ] **Step 7: Report-gen for both merchant ops (v83)**

```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/template_gms_83_1.json \
  -ida-source docs/packets/ida-exports/gms_v83.json \
  -output /tmp/t110-rpt-gms_v83-merchant
cat /tmp/t110-rpt-gms_v83-merchant/gms_v83/InteractionOperationMerchantPutItem.json
cat /tmp/t110-rpt-gms_v83-merchant/gms_v83/InteractionOperationMerchantRemoveItem.json
```
Expected: both `"Verdict": 0`. STOP on any non-✅ (wire delta — §6 fix-first).

- [ ] **Step 8: Copy reports**

```bash
cp /tmp/t110-rpt-gms_v83-merchant/gms_v83/InteractionOperationMerchantPutItem.{json,md}    docs/packets/audits/gms_v83/
cp /tmp/t110-rpt-gms_v83-merchant/gms_v83/InteractionOperationMerchantRemoveItem.{json,md} docs/packets/audits/gms_v83/
```

- [ ] **Step 9: Add marker rows** (addresses from Step 2):
  - In `operation_merchant_put_item_test.go`, above `TestOperationMerchantPutItemBytes` (the hex-pin from C0):
    `// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantPutItem version=gms_v83 ida=<ADDR_PUT_V83>`
  - In `operation_merchant_remove_item_test.go`, above `TestOperationMerchantRemoveItemBytes`:
    `// packet-audit:verify packet=interaction/serverbound/InteractionOperationMerchantRemoveItem version=gms_v83 ida=<ADDR_REMOVE_V83>`

- [ ] **Step 10: Run both tests** — `cd libs/atlas-packet && go test ./interaction/serverbound/ -run 'TestOperationMerchant(PutItem|RemoveItem)' -v 2>&1 | tail -20; cd ../..` → PASS.

- [ ] **Step 11: Pin evidence for both**

```bash
go run ./tools/packet-audit evidence pin --packet interaction/serverbound/InteractionOperationMerchantPutItem \
  --version gms_v83 --ida "CPersonalShopDlg::PutItem#Merchant" --category TIER1-FIXTURE
go run ./tools/packet-audit evidence pin --packet interaction/serverbound/InteractionOperationMerchantRemoveItem \
  --version gms_v83 --ida "CPersonalShopDlg::MoveItemToInventory#Merchant" --category TIER1-FIXTURE
```

- [ ] **Step 12: Hand-add `verifies:`** to each new YAML:
  - put: `- libs/atlas-packet/interaction/serverbound/operation_merchant_put_item_test.go#TestOperationMerchantPutItemBytes`
  - remove: `- libs/atlas-packet/interaction/serverbound/operation_merchant_remove_item_test.go#TestOperationMerchantRemoveItemBytes`

- [ ] **Step 13: Regenerate + check**

```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check 2>&1 | tee /tmp/t110-matrix-C1.txt; echo "exit ${PIPESTATUS[0]}"
for p in InteractionOperationMerchantPutItem InteractionOperationMerchantRemoveItem; do
  jq -r --arg p "$p" '.rows[] | select((.packet|type=="string") and (.packet|endswith($p))) | "\($p): \(.cells.gms_v83.state)"' docs/packets/audits/status.json
done
```
Expected: both print `verified` for `gms_v83`. No new interaction lines vs `/tmp/t110-matrix-baseline.txt`.

- [ ] **Step 14: Commit the two artifact triples**

```bash
git add libs/atlas-packet/interaction/serverbound/operation_merchant_put_item_test.go \
        libs/atlas-packet/interaction/serverbound/operation_merchant_remove_item_test.go \
        docs/packets/evidence/gms_v83/interaction.serverbound.InteractionOperationMerchantPutItem.yaml \
        docs/packets/evidence/gms_v83/interaction.serverbound.InteractionOperationMerchantRemoveItem.yaml \
        docs/packets/audits/gms_v83/InteractionOperationMerchantPutItem.json \
        docs/packets/audits/gms_v83/InteractionOperationMerchantPutItem.md \
        docs/packets/audits/gms_v83/InteractionOperationMerchantRemoveItem.json \
        docs/packets/audits/gms_v83/InteractionOperationMerchantRemoveItem.md \
        docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "verify(interaction): merchant put/remove arms gms_v83 byte-fixtures + evidence"
git rev-parse --show-toplevel; git branch --show-current
```

### Task C2: Merchant arms — gms_v87

Repeat Task C1's Steps 1–14 with the v87 instance and `gms_v87` paths:
- IDA instance: v87 binary (PRD port 13340; confirm by name).
- Export file: `docs/packets/ida-exports/gms_v87.json`; template: `template_gms_87_1.json`; audit/evidence dirs: `gms_v87`.
- Marker `version=gms_v87`; addresses `<ADDR_PUT_V87>`, `<ADDR_REMOVE_V87>` read from the v87 decompile.
- Export-splice commit message: `export(gms_v87): splice CPersonalShopDlg::{PutItem,MoveItemToInventory}#Merchant arms`.
- Verify commit message: `verify(interaction): merchant put/remove arms gms_v87 byte-fixtures + evidence`.

- [ ] **Step 1:** Select v87 instance; decompile base funcs; locate + confirm both merchant arms (write order matches codec; opcode = COutPacket truth). STOP on wire delta (fix-first) or record IDB-confirmed `n-a`.
- [ ] **Step 2:** Confirm both `#Merchant` keys absent in `gms_v87.json`; harvest to temp (`-ida-port 13340`); splice absent-only; verify additions-only diff.
- [ ] **Step 3:** Commit the export splice alone.
- [ ] **Step 4:** Report-gen (template_gms_87_1.json) → confirm both `"Verdict": 0` → copy into `docs/packets/audits/gms_v87/`.
- [ ] **Step 5:** Add `version=gms_v87` marker rows above the two hex-pin tests; run both tests → PASS.
- [ ] **Step 6:** Pin evidence (both, `--version gms_v87`); hand-add `verifies:` lines (same test#func targets as C1 Step 12).
- [ ] **Step 7:** Regenerate matrix + check; confirm both ops `gms_v87=verified`; no new interaction lines.
- [ ] **Step 8:** Commit the two artifact triples (the six `gms_v87` doc paths + the two test files + STATUS/status); verify toplevel + branch.

### Task C3: Merchant arms — jms_v185

Repeat with the jms instance and `jms_v185` version key (export file `gms_jms_185.json`, template `template_jms_185_1.json`).

> jms caveat (`VERIFYING_A_PACKET.md` §10): the retail dump is SMC/control-flow-virtualized — use the clean `*_U_DEVM` build. Confirm the selected instance is the DEVM build by binary name before decompiling.

- [ ] **Step 1:** Select the jms `*_U_DEVM` instance (PRD port 13338; confirm by name it is the DEVM build, not the SMC retail dump). Decompile base funcs; locate + confirm both merchant arms. STOP on wire delta (fix-first) or record IDB-confirmed `n-a`.
- [ ] **Step 2:** Confirm both `#Merchant` keys absent in `gms_jms_185.json`; harvest to temp (`-ida-port 13338`); splice absent-only into `docs/packets/ida-exports/gms_jms_185.json`; verify additions-only diff.
- [ ] **Step 3:** Commit the export splice alone — `export(jms_v185): splice CPersonalShopDlg::{PutItem,MoveItemToInventory}#Merchant arms`.
- [ ] **Step 4:** Report-gen `-template services/atlas-configurations/seed-data/templates/template_jms_185_1.json -ida-source docs/packets/ida-exports/gms_jms_185.json -output /tmp/t110-rpt-jms_v185-merchant` → reports land under `/tmp/t110-rpt-jms_v185-merchant/jms_v185/` → confirm both `"Verdict": 0` → copy into `docs/packets/audits/jms_v185/`.
- [ ] **Step 5:** Add `version=jms_v185` marker rows above the two hex-pin tests (addresses `<ADDR_PUT_JMS>`, `<ADDR_REMOVE_JMS>`); run both tests → PASS.
- [ ] **Step 6:** Pin evidence (both, `--version jms_v185`); hand-add `verifies:` lines.
- [ ] **Step 7:** Regenerate matrix + check; confirm both ops `jms_v185=verified`; no new interaction lines.
- [ ] **Step 8:** Commit the two artifact triples (`jms_v185` doc paths + STATUS/status); `verify(interaction): merchant put/remove arms jms_v185 byte-fixtures + evidence`; verify toplevel + branch.

### Task C4: Merchant arms — gms_v84

Repeat with the v84 instance and `gms_v84` paths. v84 is byte-identical to v83, so v83's confirmed arm (C1) is the structural template — but still decompile the v84 IDB and confirm rather than assuming.

- [ ] **Step 1:** Select the v84 instance (PRD port 13337; confirm by name). Decompile base funcs; locate + confirm both merchant arms; structure-match to the v83 twin from C1. Confirm write order matches the codec and the COutPacket opcode. STOP on wire delta (fix-first) or record IDB-confirmed `n-a`.
- [ ] **Step 2:** Confirm both `#Merchant` keys absent in `gms_v84.json`; harvest to temp (`-ida-port 13337`); splice absent-only into `docs/packets/ida-exports/gms_v84.json` (this file already received the `OnTieRequest` splice in B1 — do not disturb that entry); verify additions-only diff.
- [ ] **Step 3:** Commit the export splice alone — `export(gms_v84): splice CPersonalShopDlg::{PutItem,MoveItemToInventory}#Merchant arms`.
- [ ] **Step 4:** Report-gen (`template_gms_84_1.json`, `gms_v84.json`) → confirm both `"Verdict": 0` → copy into `docs/packets/audits/gms_v84/`.
- [ ] **Step 5:** Add `version=gms_v84` marker rows above the two hex-pin tests (addresses `<ADDR_PUT_V84>`, `<ADDR_REMOVE_V84>`); run both tests → PASS.
- [ ] **Step 6:** Pin evidence (both, `--version gms_v84`); hand-add `verifies:` lines.
- [ ] **Step 7:** Regenerate matrix + check; confirm both ops `gms_v84=verified`; no new interaction lines.
- [ ] **Step 8:** Commit the two artifact triples (`gms_v84` merchant doc paths + STATUS/status); `verify(interaction): merchant put/remove arms gms_v84 byte-fixtures + evidence`; verify toplevel + branch.

### Task C5: Merchant gate — both ops ✅ on all five versions

- [ ] **Step 1: Confirm full-row promotion**

```bash
for p in InteractionOperationMerchantPutItem InteractionOperationMerchantRemoveItem; do
  echo "=== $p ==="
  jq -r --arg p "$p" '.rows[] | select((.packet|type=="string") and (.packet|endswith($p))) | (.cells | to_entries | map("\(.key)=\(.value.state)") | join(" "))' docs/packets/audits/status.json
done
```
Expected: each prints `gms_v83=verified gms_v84=verified gms_v87=verified gms_v95=verified jms_v185=verified`.

- [ ] **Step 2: No commit** (gate only).

---

## Task D: Final verification gate (branch "done")

**Files:** none modified — this is the CLAUDE.md build/verify gate plus the campaign acceptance check.

- [ ] **Step 1: All 12 cells verified**

```bash
for p in InteractionOperationInvite InteractionOperationMemoryGameTieAnswer InteractionOperationMerchantPutItem InteractionOperationMerchantRemoveItem; do
  echo "=== $p ==="
  jq -r --arg p "$p" '.rows[] | select((.packet|type=="string") and (.packet|endswith($p))) | (.cells | to_entries | map("\(.key)=\(.value.state)") | join(" "))' docs/packets/audits/status.json
done
```
Expected: every cell `verified` (no `incomplete` remaining for any of the four packets).

- [ ] **Step 2: `libs/atlas-packet` test/vet/build clean**

```bash
cd libs/atlas-packet
go test -race ./... 2>&1 | tail -8
go vet ./... 2>&1 | tail -8
go build ./... 2>&1 | tail -8
cd ../..
```
Expected: all clean (no new failures vs the Task 0 baseline).

- [ ] **Step 3: redis-key-guard clean**

```bash
GOWORK=off tools/redis-key-guard.sh 2>&1 | tail -5; echo "exit ${PIPESTATUS[0]}"
```
Expected: clean exit 0 (no redis surface in this task, but the gate runs).

- [ ] **Step 4: No `go.mod` touched (confirms no bake needed)**

```bash
git diff --name-only main...HEAD -- '**/go.mod' 'go.mod' | head
```
Expected: empty. If any `go.mod` appears, the no-bake assumption is void — run `docker buildx bake atlas-<svc>` for each touched service per CLAUDE.md before claiming done.

- [ ] **Step 5: matrix / fnamedoc / operations `--check` — no new problems**

```bash
go run ./tools/packet-audit matrix --check 2>&1 | tee /tmp/t110-matrix-final.txt; echo "exit ${PIPESTATUS[0]}"
go run ./tools/packet-audit fnamedoc --check 2>&1 | tee /tmp/t110-fnamedoc-final.txt; echo "exit ${PIPESTATUS[0]}"
go run ./tools/packet-audit operations --check 2>&1 | tee /tmp/t110-operations-final.txt; echo "exit ${PIPESTATUS[0]}"
diff <(grep -c conflict /tmp/t110-matrix-baseline.txt) <(grep -c conflict /tmp/t110-matrix-final.txt) && echo "conflict count unchanged"
```
Expected: each may still exit 1 from the pre-existing backlog, but **zero** orphan/dangling/stale/drift lines mention an interaction packet, and the conflict count is not higher than the Task 0 baseline. If a new interaction line appears, fix that cell before claiming done.

- [ ] **Step 6: Confirm the export splices are additions-only (no unrelated drift)**

```bash
git diff main...HEAD --stat -- docs/packets/ida-exports/
git diff main...HEAD -- docs/packets/ida-exports/ | grep -E '^\-' | grep -v '^---' | head
```
Expected: only `gms_v83.json`, `gms_v84.json`, `gms_v87.json`, `gms_jms_185.json` changed; the removed-lines view shows nothing beyond benign reformatting of the spliced objects (no ~150-key drift). If unrelated keys were removed, an export was accidentally re-run — fix before claiming done.

- [ ] **Step 7: Update the PRD acceptance checklist**

Tick the boxes in `docs/tasks/task-110-interaction-packet-fixtures/prd.md` §10 (all four packets ✅ on all versions; every promoted cell has marker + evidence + report; `--check` no-new-problems; module clean). Commit:
```bash
git add docs/tasks/task-110-interaction-packet-fixtures/prd.md
git commit -m "docs(task-110): tick acceptance criteria — interaction family verified"
git rev-parse --show-toplevel; git branch --show-current
```

- [ ] **Step 8: Code review before PR**

Per CLAUDE.md "Code Review Before PR", invoke `superpowers:requesting-code-review`. It dispatches `plan-adherence-reviewer` (and `backend-guidelines-reviewer` since Go test files changed). Address findings via `superpowers:receiving-code-review` before opening the PR. Do not skip.

---

## Self-Review notes (for the executor)

- **Spec coverage:** PRD FR lists exactly four packets / 12 cells → Phase A (Invite ×3), Phase B (TieAnswer ×1), Phase C (Merchant ×8). PRD acceptance criteria → Task D Steps 1/5/2. PRD non-goals (no features, no clientbound changes, no silent reshift) are respected — wire codecs are untouched absent a proven delta, and a delta triggers fix-first surface-don't-patch.
- **No new linkage:** all four ops already in `candidatesFromFName` (`context.md`); the plan adds none.
- **`n-a` is not expected:** `BuyItem#Merchant` present on all versions proves the merchant arms are producible; `n-a` only via IDB-confirmed reason after attempting the splice (Task C1 Step 2 note).
- **Export hygiene:** every splice is absent-only, harvested to a temp file, committed on its own, and diff-checked for drift (C1 Step 5/6, Task D Step 6).
- **Stop-and-ask triggers:** a non-✅ report verdict, a wire delta in a decompile, an unresolvable fname, or an SMC-only jms binary are genuine blockers — surface them, don't paper over (per CLAUDE.md grounding/honesty and the playbook's producible-vs-blocker rule).
