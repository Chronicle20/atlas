# Login Packet-Fixture Verification Campaign — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Drive all **20 `incomplete` login-family cells** (`docs/packets/audits/status.json`) to `verified` (✅) — or to an IDB-justified `n-a` for the one jms user-limit cell — landing each promotion as its coupled artifacts (per-version audit report + `packet-audit:verify` marker + evidence record where the grader needs it, plus any export splice or codec fix), with `STATUS.md`/`status.json` regenerated.

**Architecture:** This is a packet *verification* campaign, not a feature. The login wire codecs live in `libs/atlas-packet/login/{clientbound,serverbound}/` and are already cross-version byte-tested; `gms_v95` (and `gms_v83` for most) are verified references. Each cell is promoted by: confirming the version's client read/write order, ensuring the receiver/send function (and any `#suffix` branch) is in that version's committed IDA export (surgical absent-only splice if missing), generating the per-version audit report from that export, stacking a `packet-audit:verify` marker on the byte-test (extending the fixture for missing-fixture cells), pinning evidence where the grader needs it, regenerating the matrix, and committing the cell's artifacts together. Live-IDA work is grouped **by IDB** (`select_instance` is shared global state — never interleave two versions).

**Tech Stack:** Go (`libs/atlas-packet` — the only module touched), the `tools/packet-audit` CLI (root report-gen pipeline, `export` harvest, `evidence pin`, `matrix`, `matrix --check`, `fname-doc --check`, `operations --check`), and the `mcp__ida-pro__*` MCP tools (decompile / rename / address lookup against the live IDBs).

> **Path convention:** `<worktree-root>` = `.worktrees/task-108-login-packet-fixtures/`. All paths are repo-relative from there. Run every command from `<worktree-root>`.

---

## §A. Planning-phase corrections to design.md (verified from source — READ FIRST)

The design (`design.md`) is correct on the goal, the two-stage pipeline shape, the `candidatesFromFName` fan-out (§3), and the jms `ServerStatusRequest` n-a fork (§6). **But five concrete claims were falsified during planning by grepping the committed exports / reports.** This plan is built on the verified facts below, not the design's optimistic classification. Each is evidenced so the executor trusts the plan over the design where they differ.

1. **Report filenames are BARE struct names, NOT `Login`-prefixed.** Login routing uses an empty `pkg` in `tools/packet-audit/cmd/run.go`, so `qualifiedWriterName("", name)` returns just `name` (run.go:222). On disk the reports are `docs/packets/audits/<v>/AllCharacterListSelect.json`, `AuthLoginFailed.json`, `ServerStatus.json`, etc. — **never** `LoginAllCharacterListSelect.json`. The marker `packet=` path keeps the `login/<dir>/` prefix (e.g. `packet=login/serverbound/AllCharacterListSelect`); only the report/evidence filenames drop it. (Design §2/§4 said `Login<Struct>` — wrong.)

2. **The "login is mostly deterministic report-gen, tiny IDA surface" thesis is wrong.** Design §5 grepped only the *base* function names and concluded the functions are present. They are present **but the analyzer needs the `#suffix` synthetic export entries**, and those are ABSENT in several target exports. Verified key-presence:
   - **v87 `SendSelectCharPacketByVAC#AllCharacterListSelect{,WithPic,WithPicRegister}` — ABSENT** (only the base `SendSelectCharPacketByVAC` is present). → the three v87 cells need a **harvest+splice from the v87 IDB**, not report-gen-only.
   - **`CLogin::ChangeStepImmediate` — ABSENT from gms_v83.json AND gms_v84.json** (PRESENT in v87/v95/jms, which is why those are verified). → `ServerListRequest` v83/v84 need a **harvest+splice from the v83 and v84 IDBs**, not report-gen-only. (Design §4.1/§14 called these Class A "no fresh decompile expected" — wrong.)
   - **jms `SendSelectCharPacket#CharacterSelectWithPic` / `#CharacterSelectRegisterPic` — ABSENT** (base `SendSelectCharPacket` present). → jms `CharacterSelect` RegisterPic/WithPic need a **harvest+splice from the jms IDB**.
   - **jms `OnWorldInformation#ServerListEnd` — ABSENT** (base `OnWorldInformation` present). → jms `ServerListEnd` needs a **harvest+splice from the jms IDB**.
   - **jms `SendCheckUserLimitPacket` — ABSENT** (confirms the design §6 n-a candidate; resolve via the jms IDB check).

3. **Several committed reports for "Class A" cells are stale or ❌** — "regen → ✅" is NOT guaranteed; each cell must read the post-regen verdict and branch. Verified current committed-report verdicts:
   - `gms_v84/AuthLoginFailed.json`, `gms_v84/AllCharacterListSelect{,WithPic,WithPicRegister}.json` — row 0 `verdict 4: "function not found in IDB"` with **empty Address**. These are **stale** (the `#suffix` keys ARE present in the current v84 export) → a fresh report-gen should resolve the function; the resulting verdict is then unknown (likely ✅ since v84 body ≡ v83, but **read it**).
   - `gms_v84/ServerStatus.json` — real Address `0x60e275`, `verdict 2: "width mismatch"` → a real comparison against the present `OnCheckUserLimitResult`; decompile to adjudicate (v84 ServerStatus body should match v83's verified report).
   - `gms_v83/AllCharacterListRequest.json` — `flatInvalid: true`, all rows `"atlas: extra — client never reads this field"` (tier-1). The function `SendViewAllCharPacket` IS present in the v83 export → this is a read-order/linkage issue to adjudicate by decompile, not a missing function.
   - `gms_v83/AuthLoginFailed.json` — `bad: 0` (clean verdict). The cell is `incomplete` only with note "marker present but no fresh evidence record"; clientbound tier-0 promotes on report+marker, so a matrix regen may already flip it — confirm, and if it still won't promote, treat the note's evidence requirement as authoritative and pin one.

4. **`ServerListRequest` grades on report + marker only — NO evidence record.** The verified versions (v87/v95/jms) each have a `ServerListRequest.json` report and a stacked marker but **no `login.serverbound.ServerListRequest.yaml` evidence file**. It is a `sub-struct` row (`op: null`, `tier1: false`). Mirror that: report + marker, do not pin evidence. (Design §4 said "serverbound: evidence always" — too strong; the matrix is the arbiter, and the verified siblings prove report+marker suffices here.)

5. **`pt.Variants` already includes v84 and JMS v185** (`libs/atlas-packet/test/context.go:18-32`). So existing `pt.RoundTrip(t, ..., pt.Variants)` tests already exercise those versions. A `CharacterSelect` cell graded `🔍 "tier-1 without fixture"` is missing a **golden byte-fixture assertion** for that version, not a round-trip — the base `character_select_byte_test.go` uses a hardcoded 3-version slice (`v83/v87/v95`) that must gain `v84` and `jms` rows.

**Net effect on work shape:** all four non-v95 IDBs are needed (v83, v84, v87, jms), each with at least one surgical export splice. The two-stage pipeline still holds — Stage 1 is the genuinely-regen-only cells; Stage 2 is the (larger-than-design-claimed) IDA surface, grouped by IDB.

---

## §B. Reference facts (read once before any task)

### Per-version mapping

| Version key (marker `version=`) | Seed template | Committed export json | Audit report dir | IDA port (per memory IDBs_v9 — **confirm by binary NAME via `list_instances`, never trust the number**) |
|---|---|---|---|---|
| `gms_v83` | `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` | `docs/packets/ida-exports/gms_v83.json` | `docs/packets/audits/gms_v83/` | 13341 |
| `gms_v84` | `services/atlas-configurations/seed-data/templates/template_gms_84_1.json` | `docs/packets/ida-exports/gms_v84.json` | `docs/packets/audits/gms_v84/` | 13337 |
| `gms_v87` | `services/atlas-configurations/seed-data/templates/template_gms_87_1.json` | `docs/packets/ida-exports/gms_v87.json` | `docs/packets/audits/gms_v87/` | 13340 |
| `gms_v95` | `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` | `docs/packets/ida-exports/gms_v95.json` | `docs/packets/audits/gms_v95/` | 13339 (verified reference; no cells here) |
| `jms_v185` | `services/atlas-configurations/seed-data/templates/template_jms_185_1.json` | `docs/packets/ida-exports/gms_jms_185.json` | `docs/packets/audits/jms_v185/` | 13338 (**use the clean `*_U_DEVM` build**, not the SMC retail dump) |

### Promotion mechanism (what makes a cell ✅)

- **Clientbound tier-0** (`AuthLoginFailed`, `ServerStatus`, `ServerListEnd`): per-version **audit report** (verdict clean) + stacked **`packet-audit:verify` marker**. Do **not** pin evidence unless `matrix --check` demands it for that cell (playbook §7: evidence on a tier-0 cell is a standing freshness liability).
- **Serverbound op rows** (`AllCharacterListSelect ×3`, `CharacterSelect ×3`, `ServerStatusRequest`): report + marker + **pinned evidence** (playbook §9 — three artifacts that agree) AND the op **routed** in that version's seed template.
- **Serverbound tier-1** (`AllCharacterListRequest`): report + marker + **pinned evidence** with `verifies:`.
- **Serverbound sub-struct** (`ServerListRequest`): report + marker, **no evidence** (verified-sibling pattern, §A.4).

The **arbiter is `go run ./tools/packet-audit matrix --check`** — a cell is done when it reads ✅ in regenerated `status.json` and introduces no new `--check` problem mentioning a `login/*` packet, and the global conflict count does not increase (§E).

### Report-gen command (deterministic, no live IDA) — used in every Stage-1 cell

```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/template_<V>.json \
  -ida-source docs/packets/ida-exports/<EXPORT>.json \
  -output /tmp/rpt-<V>
# then copy ONLY the needed report into the committed audit dir:
cp /tmp/rpt-<V>/<version-key>/<Struct>.json docs/packets/audits/<version-key>/<Struct>.json
cp /tmp/rpt-<V>/<version-key>/<Struct>.md   docs/packets/audits/<version-key>/<Struct>.md
```

Where `<V>` is the template-file infix (`gms_83_1`, `gms_84_1`, `gms_87_1`, `jms_185_1`), `<EXPORT>` is the export filename stem (`gms_v83`, `gms_v84`, `gms_v87`, `gms_jms_185`), and `<version-key>` is the marker/audit-dir key (`gms_v83`, `gms_v84`, `gms_v87`, `jms_v185`).

### Export splice procedure (Stage-2 cells, playbook §10) — surgical, absent-only

**Never overwrite a committed export.** To add ONE function (or `#suffix` branch):

```bash
# 1. Harvest to a TEMP file (target the right IDB port; confirm by binary name first):
go run ./tools/packet-audit export \
  --version <version-key> \
  --prior-export "" \
  --pending /tmp/roster-<version-key>.md \
  --descent-depth 12 \
  --ida-url http://127.0.0.1:<PORT>/mcp --ida-port <PORT> \
  --output /tmp/harvest-<version-key>.json
# 2. Surgically splice ONLY the needed key(s) into the committed export with a small
#    Python merge (absent-only for helpers; overwrite a stub only for the one sender).
#    Strip any {op: Delegate, ref: COutPacket} ctor artifact from a spliced send entry.
# 3. git diff docs/packets/ida-exports/<EXPORT>.json — confirm ONLY the intended keys changed.
```

`/tmp/roster-<version-key>.md` lists the fname(s) to harvest (one `- CLogin::Fn#Branch` per line). The `#suffix` synthetic entries are produced by the exporter's branch descent; harvest the base function and the descent emits the `#suffix` keys.

### Marker idiom

Stack one `// packet-audit:verify` line per version above the existing test function (examples: `libs/atlas-packet/login/serverbound/character_select_register_pic_test.go` has four stacked markers; `libs/atlas-packet/login/serverbound/all_character_list_request_test.go` has five). Format:

```
// packet-audit:verify packet=login/<dir>/<Struct> version=<version-key> ida=0x<addr>
```

The `ida=` address is the **function's own address** from the report (`Address` field) / decompile, NOT the dispatch opcode.

### Evidence pin idiom

```bash
go run ./tools/packet-audit evidence pin \
  --packet login/<dir>/<Struct> --version <version-key> \
  --ida "<FName-incl-#suffix>" --category TIER1-FIXTURE
# then OPEN docs/packets/evidence/<version-key>/login.<dir>.<Struct>.yaml and add:
#   verifies:
#     - <test file path>#<TestName>
```

Template evidence record: `docs/packets/evidence/gms_v84/login.serverbound.CharacterSelectWithPic.yaml`.

---

## §C. The 20 cells — verified state and recipe

Keyed by (packet, op, version). "Report?" / "Suffix in export?" / "Marker?" / "Evidence?" are the **verified planning-time** states. "Stage" is 1 (regen-only) or 2 (needs the named IDB).

| # | packet / op | version | report (current verdict) | `#suffix` in export | marker | evidence | Stage / IDB | recipe |
|---|---|---|---|---|---|---|---|---|
| 1 | `clientbound/AuthLoginFailed` LOGIN_STATUS | gms_v83 | YES (clean, bad=0) | n/a (present) | YES | — | **1** | regen → confirm ✅; if won't promote, pin evidence per note |
| 2 | `clientbound/AuthLoginFailed` LOGIN_STATUS | gms_v84 | YES (stale: fn-not-found) | present | YES | — | **2 / v84** | regen first; if still ❌, decompile `OnCheckPasswordResult#AuthLoginFailed`; adjudicate |
| 3 | `clientbound/ServerStatus` SERVERSTATUS | gms_v84 | YES (❌ width-mismatch) | present (`OnCheckUserLimitResult`) | YES | — | **2 / v84** | decompile `OnCheckUserLimitResult`; adjudicate vs v83 verified report |
| 4 | `clientbound/ServerListEnd` WORLD_INFORMATION | jms_v185 | NO | **ABSENT** (`OnWorldInformation#ServerListEnd`) | YES | — | **2 / jms** | harvest+splice suffix from jms IDB → regen → adjudicate (most likely real delta) |
| 5 | `serverbound/ServerStatusRequest` SERVERSTATUS_REQUEST | jms_v185 | NO | **ABSENT** (`SendCheckUserLimitPacket`) | YES | — | **2 / jms** | §6 fork: name+splice if present-but-unnamed → Class-A; else **n-a** with IDB justification |
| 6 | `serverbound/AllCharacterListRequest` VIEW_ALL_CHAR (tier1) | gms_v83 | YES (❌ flatInvalid) | present (`SendViewAllCharPacket`) | YES | — | **2 / v83** | decompile `SendViewAllCharPacket`; adjudicate the "atlas: extra" rows; pin tier-1 evidence |
| 7 | `serverbound/ServerListRequest` (sub-struct) | gms_v83 | NO | **ABSENT** (`ChangeStepImmediate`) | NO (v83) | — (none) | **2 / v83** | harvest+splice `ChangeStepImmediate` → regen → add v83 marker; NO evidence |
| 8 | `serverbound/ServerListRequest` (sub-struct) | gms_v84 | NO | **ABSENT** (`ChangeStepImmediate`) | NO (v84) | — (none) | **2 / v84** | harvest+splice `ChangeStepImmediate` → regen → add v84 marker; NO evidence |
| 9 | `serverbound/AllCharacterListSelect` PICK_ALL_CHAR | gms_v84 | YES (stale: fn-not-found) | present | YES | NO | **1** | regen → adjudicate; pin evidence |
| 10 | `serverbound/AllCharacterListSelect` VIEW_ALL_PIC_REGISTER | gms_v84 | YES (stale) | present (`#...WithPicRegister`) | YES | NO | **1** | regen → adjudicate; pin evidence |
| 11 | `serverbound/AllCharacterListSelect` VIEW_ALL_WITH_PIC | gms_v84 | YES (stale) | present (`#...WithPic`) | YES | NO | **1** | regen → adjudicate; pin evidence |
| 12 | `serverbound/AllCharacterListSelect` PICK_ALL_CHAR | gms_v87 | NO | **ABSENT** | YES | NO | **2 / v87** | harvest+splice 3 VAC `#suffix` branches → regen → pin evidence |
| 13 | `serverbound/AllCharacterListSelect` VIEW_ALL_PIC_REGISTER | gms_v87 | NO | **ABSENT** | YES | NO | **2 / v87** | (same v87 splice as #12) → regen → pin evidence |
| 14 | `serverbound/AllCharacterListSelect` VIEW_ALL_WITH_PIC | gms_v87 | NO | **ABSENT** | YES | NO | **2 / v87** | (same v87 splice) → regen → pin evidence |
| 15 | `serverbound/CharacterSelect` CHAR_SELECT | gms_v84 | YES (🔍 flatInvalid) | present (base) | NO (base test) | NO | **1** | add v84 golden row to `character_select_byte_test.go` + v84 marker → regen → pin evidence |
| 16 | `serverbound/CharacterSelect` REGISTER_PIC | gms_v84 | YES | present | YES | YES | **1** | confirm fixture exercises v84 → regen → pin/refresh evidence |
| 17 | `serverbound/CharacterSelect` CHAR_SELECT_WITH_PIC | gms_v84 | YES | present | YES | YES | **1** | confirm fixture exercises v84 → regen → pin/refresh evidence |
| 18 | `serverbound/CharacterSelect` CHAR_SELECT | jms_v185 | YES (🔍 flatInvalid) | present (base) | NO | NO | **2 / jms** | add jms golden row + marker → regen → pin evidence |
| 19 | `serverbound/CharacterSelect` REGISTER_PIC | jms_v185 | NO | **ABSENT** | NO | NO | **2 / jms** | harvest+splice `SendSelectCharPacket#CharacterSelectRegisterPic` → fixture+marker → regen → pin evidence |
| 20 | `serverbound/CharacterSelect` CHAR_SELECT_WITH_PIC | jms_v185 | NO | **ABSENT** | NO | NO | **2 / jms** | harvest+splice `SendSelectCharPacket#CharacterSelectWithPic` → fixture+marker → regen → pin evidence |

> **Verdict-clean rule (applies to every cell):** a cell is ✅ only when its regenerated report has `FlatInvalid: false` and every `Rows[].Verdict == 0`. A surviving non-zero verdict is either (a) a **stale report** → re-run report-gen against the up-to-date export, or (b) a **real wire delta** → STOP and take the fix-first path (§D.4). Never re-pin a ❌ as ✅ without a decompile that explains it.

---

## §D. Canonical per-cell procedure (referenced by the tasks)

Every cell follows this skeleton. Tasks below give the concrete substitutions; do not improvise beyond them.

**D.1 — Ensure the function is in the export.** If §C says `#suffix`/fname is ABSENT, run the §B export-splice procedure against the cell's IDB FIRST. Confirm with the python key-presence check (context.md "key-presence snippet"). If present, skip.

**D.2 — Generate the report.** Run the §B report-gen command for the version; copy `<Struct>.{json,md}` into `docs/packets/audits/<version-key>/`. Read the JSON: `FlatInvalid` and every `Rows[].Verdict`.

**D.3 — If verdict clean (all 0, FlatInvalid false):** add/confirm the stacked marker (§B), extend the byte-fixture if §C says a golden row is missing, pin evidence if the cell's tier requires it (§B promotion mechanism), then go to D.5.

**D.4 — If verdict NOT clean:** `select_instance` the cell's IDB (confirm by binary name), decompile the cell's fname (descend into helper reads), write the full ordered read/write list, and compare against the Atlas codec in `libs/atlas-packet/login/<dir>/<file>.go`.
  - **Stale/cosmetic** (codec matches the decompile; the ❌ was a pre-splice artifact) → the splice/regen in D.1–D.2 already fixed it; re-read the report, it should now be clean → D.3.
  - **Real wire delta** (an inserted field / changed width / different guard) → **STOP, surface it** in the PR description and in `docs/tasks/task-108-login-packet-fixtures/audit.md`. Land the **codec fix first** as its own commit in `libs/atlas-packet/login/<dir>/` (add the version branch; update the cross-version byte-test to expect the divergence; `go test -race ./...` green), then resume D.2 against the corrected codec. (No delta is *expected* for v84 — body ≡ v83; the jms `ServerListEnd` cell is the prime real-delta suspect.)

**D.5 — Regenerate the matrix and verify promotion.**
```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```
Confirm the cell reads `verified` in `docs/packets/audits/status.json` and that `--check` reports no new `login/*` problem and no increase in the global conflict count (§E baseline).

**D.6 — Commit the cell's coupled artifacts together** (report `.json`+`.md`, marker/test change, evidence yaml if any, export splice if any, regenerated `STATUS.md`+`status.json`):
```bash
git add docs/packets/audits/<version-key>/<Struct>.json docs/packets/audits/<version-key>/<Struct>.md \
        libs/atlas-packet/login/<dir>/<test>.go \
        docs/packets/evidence/<version-key>/login.<dir>.<Struct>.yaml \
        docs/packets/ida-exports/<EXPORT>.json \
        docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "verify(login): <packet>/<op> <version-key> ✅"
git rev-parse --show-toplevel   # must end with /.worktrees/task-108-login-packet-fixtures
git branch --show-current       # must be task-108-login-packet-fixtures
```
(Drop the `git add` lines for artifacts a given cell doesn't produce.)

---

## Task 0: Baseline snapshot (no code change)

**Files:** none (read-only baseline capture).

- [ ] **Step 1: Confirm worktree + green build**

Run from `<worktree-root>`:
```bash
pwd   # .../.worktrees/task-108-login-packet-fixtures
git branch --show-current   # task-108-login-packet-fixtures
( cd libs/atlas-packet && go build ./... && go test -race ./login/... && go vet ./... )
```
Expected: build/test/vet clean (the login codecs already pass).

- [ ] **Step 2: Capture the matrix-check baseline (the §E "no new problems" reference)**

```bash
go run ./tools/packet-audit matrix --check 2>&1 | tee /tmp/matrix-check-baseline.txt
grep -c "conflict" /tmp/matrix-check-baseline.txt   # record this number
grep -i "login/" /tmp/matrix-check-baseline.txt      # record any pre-existing login lines
```
Expected: a non-zero exit from the pre-existing registry-seed conflict backlog (playbook §8). **Record the conflict count and any `login/` lines** — the acceptance bar is "no increase, no new login line" (§E).

- [ ] **Step 3: Confirm the 20-cell work list matches this plan**

```bash
python3 - <<'EOF'
import json
d=json.load(open("docs/packets/audits/status.json"))
n=0
for r in d["rows"]:
    if not r.get("packet","").startswith("login/"): continue
    for v,c in r["cells"].items():
        if c.get("state") not in ("verified","n-a"):
            n+=1; print(r["packet"], r.get("op"), v, c.get("note"))
print("TOTAL:", n)
EOF
```
Expected: `TOTAL: 20`, matching §C. If it differs, the matrix has drifted since planning — re-reconcile §C before proceeding. **No commit.**

---

## Stage 1 — Report-gen-only cells (no live IDA)

These cells' functions (incl. `#suffix`) are already in the committed export; the work is report-gen + marker + evidence. **Each is its own commit** via the §D skeleton.

### Task 1: `AuthLoginFailed` gms_v83 (cell #1)

**Files:** `docs/packets/audits/gms_v83/AuthLoginFailed.{json,md}`, `libs/atlas-packet/login/clientbound/auth_login_failed_test.go` (marker already present).

- [ ] **Step 1: Regenerate the report (§D.2)** — run the §B report-gen with `<V>=gms_83_1`, `<EXPORT>=gms_v83`, `<version-key>=gms_v83`; copy `AuthLoginFailed.{json,md}`.
- [ ] **Step 2: Read the verdict** — `python3 -c "import json;d=json.load(open('docs/packets/audits/gms_v83/AuthLoginFailed.json'));print(d['FlatInvalid'],[r['Verdict'] for r in d['Rows']])"`. Expected: clean (`False []`-of-zeros).
- [ ] **Step 3: Confirm marker present** — `grep AuthLoginFailed.*gms_v83 libs/atlas-packet/login/clientbound/auth_login_failed_test.go` (already there per planning).
- [ ] **Step 4: Regenerate matrix + check (§D.5)** — confirm `clientbound/AuthLoginFailed gms_v83` → `verified`. If it still won't promote, the note's "no fresh evidence record" is authoritative: `evidence pin --packet login/clientbound/AuthLoginFailed --version gms_v83 --ida "CLogin::OnCheckPasswordResult#AuthLoginFailed" --category TIER1-FIXTURE`, add `verifies: - libs/atlas-packet/login/clientbound/auth_login_failed_test.go#TestAuthLoginFailed...`, re-run matrix.
- [ ] **Step 5: Commit (§D.6).**

### Task 2: `AllCharacterListSelect` ×3 gms_v84 (cells #9, #10, #11)

**Files:** `docs/packets/audits/gms_v84/{AllCharacterListSelect,AllCharacterListSelectWithPic,AllCharacterListSelectWithPicRegister}.{json,md}`; markers already on `all_character_list_select_test.go`, `all_character_list_select_with_pic_test.go`, `all_character_list_select_with_pic_register_test.go` (add the `gms_v84` line — verify it's not already there); evidence `docs/packets/evidence/gms_v84/login.serverbound.AllCharacterListSelect{,WithPic,WithPicRegister}.yaml`.

- [ ] **Step 1: Regenerate reports (§D.2)** — report-gen with `<V>=gms_84_1`, `<EXPORT>=gms_v84`, `<version-key>=gms_v84`; copy all three structs. (The committed reports are stale "fn-not-found"; the `#suffix` keys ARE in the v84 export, so regen resolves them.)
- [ ] **Step 2: Read each verdict.** Expected clean (v84 body ≡ v83; v83/v95 are verified). If any is non-clean → escalate that struct to **Task 9 (v84 IDB)** §D.4; do not force it.
- [ ] **Step 3: Stack the `gms_v84` marker** on each of the three tests (mirror the existing `gms_v83`/`gms_v95` lines; `ida=` = each report's `Address`).
- [ ] **Step 4: Pin evidence** for each (serverbound op rows need it): `evidence pin --packet login/serverbound/AllCharacterListSelect --version gms_v84 --ida "CLogin::SendSelectCharPacketByVAC#AllCharacterListSelect" --category TIER1-FIXTURE` (and the two `#...WithPic` / `#...WithPicRegister` variants), add `verifies:` lines.
- [ ] **Step 5: Run `go test -race ./login/...`** in `libs/atlas-packet` — green.
- [ ] **Step 6: Regenerate matrix + check (§D.5);** confirm all three `gms_v84` cells → `verified`.
- [ ] **Step 7: Commit (§D.6)** — one commit for the three v84 VAC cells (shared send function, shared splice-free regen).

### Task 3: `CharacterSelect` RegisterPic + WithPic gms_v84 (cells #16, #17)

**Files:** `docs/packets/audits/gms_v84/{CharacterSelectRegisterPic,CharacterSelectWithPic}.{json,md}`; tests already carry the `gms_v84` marker and evidence exists.

- [ ] **Step 1: Regenerate the two reports (§D.2)** — `<V>=gms_84_1`, `<EXPORT>=gms_v84`. Read verdicts; expected clean.
- [ ] **Step 2: Confirm the round-trip fixtures exercise v84** — `character_select_with_pic_test.go` / `character_select_register_pic_test.go` loop over `pt.Variants` (includes v84, §A.5). Confirm the existing v84 marker `ida=` matches the regenerated `Address`; fix if drifted.
- [ ] **Step 3: Refresh evidence if hash drifted** — re-run `evidence pin` for each (`--ida "CLogin::SendSelectCharPacket#CharacterSelectWithPic"` / `#CharacterSelectRegisterPic`), keep the `verifies:` line.
- [ ] **Step 4: Regenerate matrix + check (§D.5);** confirm both `gms_v84` cells → `verified`.
- [ ] **Step 5: Commit (§D.6).**

### Task 4: `CharacterSelect` base CHAR_SELECT gms_v84 (cell #15)

**Files:** `libs/atlas-packet/login/serverbound/character_select_byte_test.go` (add v84 golden row + marker), `docs/packets/audits/gms_v84/CharacterSelect.{json,md}`, `docs/packets/evidence/gms_v84/login.serverbound.CharacterSelect.yaml`.

- [ ] **Step 1: Add the v84 golden row to the byte-fixture.** In `character_select_byte_test.go`, the `TestCharacterSelectByteOutput` slice is hardcoded `{GMS v83},{GMS v87},{GMS v95}`. Append `{"GMS v84", "GMS", 84, 1}`. The expected `want` bytes are identical (v84 body ≡ v83: `Encode4(charId)+EncodeStr(mac)+EncodeStr(hwid)`):
```go
	{"GMS v84", "GMS", 84, 1},
```
- [ ] **Step 2: Run the byte-test, expect PASS** — `( cd libs/atlas-packet && go test -race ./login/serverbound/ -run TestCharacterSelectByteOutput -v )`. Expected: the new `GMS_v84` subtest passes (same body).
- [ ] **Step 3: Stack the `gms_v84` marker** above `TestCharacterSelectByteOutput` (after the existing v95 line): `// packet-audit:verify packet=login/serverbound/CharacterSelect version=gms_v84 ida=0x<addr>` where `<addr>` = the regenerated report `Address`.
- [ ] **Step 4: Regenerate the report (§D.2)** `<V>=gms_84_1`; read verdict. If `FlatInvalid`/non-zero persists after the fixture exists, decompile `SendSelectCharPacket` base branch in the **v84 IDB** (Task 9) to adjudicate (the 🔍 should clear once a fixture+evidence exist).
- [ ] **Step 5: Pin evidence** — `evidence pin --packet login/serverbound/CharacterSelect --version gms_v84 --ida "CLogin::SendSelectCharPacket" --category TIER1-FIXTURE`; add `verifies: - libs/atlas-packet/login/serverbound/character_select_byte_test.go#TestCharacterSelectByteOutput`.
- [ ] **Step 6: Regenerate matrix + check (§D.5);** confirm `serverbound/CharacterSelect CHAR_SELECT gms_v84` → `verified`.
- [ ] **Step 7: Commit (§D.6).**

---

## Stage 2 — Live-IDA cells, grouped by IDB

**Discipline (playbook §10):** `select_instance` is shared global state. Do ALL of one IDB's cells, then move on. Before any decompile/harvest: `mcp__ida-pro__list_instances`, pick the instance whose loaded binary NAME matches the target version, `select_instance(<port>)`. Never hardcode the port from the table — confirm by name.

### Task 5: v83 IDB — `ServerListRequest` v83 + `AllCharacterListRequest` v83 (cells #7, #6)

**IDB:** v83 (port ~13341 — confirm by name). **Files:** `docs/packets/ida-exports/gms_v83.json` (splice), `docs/packets/audits/gms_v83/{ServerListRequest,AllCharacterListRequest}.{json,md}`, `libs/atlas-packet/login/serverbound/server_list_request_test.go` (add v83 marker), `docs/packets/evidence/gms_v83/login.serverbound.AllCharacterListRequest.yaml`.

- [ ] **Step 1: Select the v83 instance** — `list_instances` → `select_instance(<v83 port>)`; confirm binary name.
- [ ] **Step 2: `ServerListRequest` — harvest+splice `CLogin::ChangeStepImmediate`** (ABSENT from v83 export, §A.2). Roster `/tmp/roster-gms_v83.md` = `- CLogin::ChangeStepImmediate`; run the §B export harvest at the v83 port; splice the `ChangeStepImmediate` key (and the `#ServerListRequest` suffix the descent emits, if any) into `gms_v83.json`; `git diff` shows only those keys. Model: the verified `gms_v87/v95/jms` exports contain `ChangeStepImmediate`.
- [ ] **Step 3: `ServerListRequest` — report-gen (§D.2)** `<V>=gms_83_1`; copy `ServerListRequest.{json,md}`; read verdict (expect clean — substruct mirrors verified siblings).
- [ ] **Step 4: `ServerListRequest` — add the v83 marker** to `server_list_request_test.go` (it currently has v95/v87/jms; add `// packet-audit:verify packet=login/serverbound/ServerListRequest version=gms_v83 ida=0x<addr>`). **No evidence** (§A.4).
- [ ] **Step 5: `AllCharacterListRequest` — decompile `CLogin::SendViewAllCharPacket`** (present in export but report is `flatInvalid` "atlas: extra", §A.3). Write the client write order; compare to `libs/atlas-packet/login/serverbound/all_character_list_request.go`. Decide stale-vs-real (§D.4). v83 is the reference version, so the codec is likely right and the report linkage stale — regen after confirming. If a real delta, fix-first (§D.4).
- [ ] **Step 6: `AllCharacterListRequest` — report-gen (§D.2)** `<V>=gms_83_1`; copy; confirm clean verdict.
- [ ] **Step 7: `AllCharacterListRequest` — pin tier-1 evidence** — `evidence pin --packet login/serverbound/AllCharacterListRequest --version gms_v83 --ida "CLogin::SendViewAllCharPacket" --category TIER1-FIXTURE`; add `verifies: - libs/atlas-packet/login/serverbound/all_character_list_request_test.go#<TestName>` (the v83 marker is already present).
- [ ] **Step 8: `go test -race ./login/...`** green; **regenerate matrix + check (§D.5)** — confirm both v83 cells → `verified`.
- [ ] **Step 9: Commit** each cell separately (§D.6): one commit for `ServerListRequest gms_v83`, one for `AllCharacterListRequest gms_v83`.

### Task 6: v84 IDB — `ServerListRequest` v84 + `AuthLoginFailed` v84 + `ServerStatus` v84 (cells #8, #2, #3) + any escalations from Tasks 2/4

**IDB:** v84 (port ~13337 — confirm by name). **Files:** `docs/packets/ida-exports/gms_v84.json` (splice for ServerListRequest), `docs/packets/audits/gms_v84/{ServerListRequest,AuthLoginFailed,ServerStatus}.{json,md}`, `libs/atlas-packet/login/serverbound/server_list_request_test.go` (add v84 marker), possibly `libs/atlas-packet/login/clientbound/{server_status,auth_login_failed}.go` (only if a real wire delta, §D.4).

- [ ] **Step 1: Select the v84 instance** — `list_instances` → `select_instance(<v84 port>)`; confirm name.
- [ ] **Step 2: `ServerListRequest` v84 — harvest+splice `CLogin::ChangeStepImmediate`** (ABSENT from v84 export, §A.2). Same procedure as Task 5 Step 2 but against the v84 port/export. Report-gen `<V>=gms_84_1`; add v84 marker; NO evidence; confirm clean.
- [ ] **Step 3: `AuthLoginFailed` v84 — decompile `CLogin::OnCheckPasswordResult#AuthLoginFailed`** (committed report stale "fn-not-found", but suffix present → regen first). Report-gen `<V>=gms_84_1`; read verdict. If clean (expected, v84≡v83) → marker `ida=` already present, just confirm address; pin not required (clientbound tier-0). If still ❌ → adjudicate via the decompile (§D.4).
- [ ] **Step 4: `ServerStatus` v84 — decompile `CLogin::OnCheckUserLimitResult`** (real "width mismatch" ❌, §A.3). Compare to `libs/atlas-packet/login/clientbound/server_status.go` and the verified `gms_v83/ServerStatus.json` read order. v84 body should equal v83. If the mismatch is a stale report → regen clears it. If a real width delta → fix-first (§D.4): add the version branch to `server_status.go`, update `server_status_test.go` cross-version expectation, own commit, then regen.
- [ ] **Step 5: Handle any Task 2 / Task 4 escalations** — if a v84 VAC or base-CharacterSelect report was non-clean in Stage 1, decompile its fname here in the same v84 IDB session and adjudicate.
- [ ] **Step 6: `go test -race ./login/...`** green; **regenerate matrix + check (§D.5)** — confirm all v84 cells handled here → `verified`.
- [ ] **Step 7: Commit** each cell separately (§D.6).

### Task 7: v87 IDB — `AllCharacterListSelect` ×3 v87 (cells #12, #13, #14)

**IDB:** v87 (port ~13340 — confirm by name). **Files:** `docs/packets/ida-exports/gms_v87.json` (splice 3 `#suffix` branches), `docs/packets/audits/gms_v87/{AllCharacterListSelect,AllCharacterListSelectWithPic,AllCharacterListSelectWithPicRegister}.{json,md}`, markers already on the three tests (verify), evidence `docs/packets/evidence/gms_v87/login.serverbound.AllCharacterListSelect{,WithPic,WithPicRegister}.yaml`.

- [ ] **Step 1: Select the v87 instance** — `list_instances` → `select_instance(<v87 port>)`; confirm name.
- [ ] **Step 2: Harvest+splice the three VAC `#suffix` branches** (ABSENT from v87 export, only base present, §A.2). Roster `/tmp/roster-gms_v87.md` = `- CLogin::SendSelectCharPacketByVAC` (the descent emits the three `#AllCharacterListSelect{,WithPic,WithPicRegister}` synthetic entries; model: v83/v95 exports which carry them). Splice all three suffix keys into `gms_v87.json`; strip any `COutPacket` delegate artifact; `git diff` shows only those keys.
- [ ] **Step 3: Report-gen (§D.2)** `<V>=gms_87_1`, `<EXPORT>=gms_v87`; copy the three structs; read each verdict (expect clean — v87 is a verified-on-other-modes version).
- [ ] **Step 4: Confirm/stack the `gms_v87` marker** on each test (`ida=` = each report `Address`).
- [ ] **Step 5: Pin evidence** for each (serverbound op rows) with the `#suffix` fname; add `verifies:` lines.
- [ ] **Step 6: `go test -race ./login/...`** green; **regenerate matrix + check (§D.5)** — confirm the three v87 cells → `verified`.
- [ ] **Step 7: Commit (§D.6)** — one commit for the three v87 VAC cells (shared splice).

### Task 8: jms IDB — `ServerListEnd` + `CharacterSelect` ×3 + `ServerStatusRequest` (cells #4, #18, #19, #20, #5)

**IDB:** jms `*_U_DEVM` (port ~13338 — confirm by name; NOT the SMC retail dump, playbook §10). **Files:** `docs/packets/ida-exports/gms_jms_185.json` (splices), `docs/packets/audits/jms_v185/{ServerListEnd,CharacterSelect,CharacterSelectRegisterPic,CharacterSelectWithPic}.{json,md}`, `libs/atlas-packet/login/{clientbound/server_list_end_test.go,serverbound/character_select_byte_test.go,serverbound/character_select_register_pic_test.go,serverbound/character_select_with_pic_test.go}` (markers/fixtures), evidence for the serverbound CharacterSelect cells, and the `ServerStatusRequest` n-a/splice resolution.

- [ ] **Step 1: Select the jms `*_U_DEVM` instance** — `list_instances` → confirm the `_U_DEVM` binary name → `select_instance(<jms port>)`.
- [ ] **Step 2: `ServerListEnd` jms — harvest+splice `CLogin::OnWorldInformation#ServerListEnd`** (suffix ABSENT, base present, §A.2). Roster = `- CLogin::OnWorldInformation`; the descent emits `#ServerListEnd`; splice it. Report-gen `<V>=jms_185_1`, `<EXPORT>=gms_jms_185`; read verdict. **This is the prime real-delta suspect** (jms login structure genuinely differs, design §7) — if ❌ after splice, decompile carefully and take the fix-first path (§D.4) on `libs/atlas-packet/login/clientbound/server_list_end.go`. The marker is already present; confirm `ida=` matches.
- [ ] **Step 3: `CharacterSelect` base jms (cell #18) — add jms golden row + marker.** Decompile `CLogin::SendSelectCharPacket` base (`m_bLoginOpt<=3`) branch; derive the body bytes; append `{"JMS v185", "JMS", 185, 1}` to the `character_select_byte_test.go` slice with the JMS-correct `want` (JMS may differ from GMS — derive from the decompile, do NOT assume GMS shape). Add the jms marker. Pin evidence (`--ida "CLogin::SendSelectCharPacket"`).
- [ ] **Step 4: `CharacterSelect` RegisterPic + WithPic jms (cells #19, #20) — harvest+splice the two `#suffix` branches** (ABSENT, §A.2). Roster = `- CLogin::SendSelectCharPacket`; descent emits `#CharacterSelectRegisterPic` and `#CharacterSelectWithPic`; splice both. Add the jms marker to each test; the round-trip tests already loop `pt.Variants` (incl. jms). Report-gen each; confirm clean (decompile-adjudicate if not). Pin evidence for each.
- [ ] **Step 5: `ServerStatusRequest` jms (cell #5) — resolve the n-a fork (design §6).** In the jms IDB, look for `CLogin::SendCheckUserLimitPacket` (ABSENT from export, §A.2):
  - **Present-but-unnamed** → name it (byte-signature `6A <op> … E8`, structure-match to a named GMS twin), harvest+splice, report-gen `<V>=jms_185_1`, pin evidence — promote to ✅.
  - **Genuinely absent** (IDB-confirmed) → record `serverbound/ServerStatusRequest jms_v185 = n-a` with the IDB justification in the status note / a short note in `audit.md`; the clientbound twin `ServerStatus` is already `n-a` for jms, consistent. **Do not fabricate a fixture.** Update `status.json` to `n-a` for this cell (via the matrix tool's n-a mechanism — check how sibling jms n-a cells like `serverbound/AllCharacterListSelect jms` are recorded and mirror it).
- [ ] **Step 6: `go test -race ./login/...`** green; **regenerate matrix + check (§D.5)** — confirm all jms cells → `verified` (or `ServerStatusRequest` → justified `n-a`).
- [ ] **Step 7: Commit** each cell separately (§D.6).

---

## Task 9: Final verification gate (acceptance)

**Files:** none new (final regen + gate).

- [ ] **Step 1: Full matrix regen** — `go run ./tools/packet-audit matrix`.
- [ ] **Step 2: Confirm zero incomplete login cells**
```bash
python3 - <<'EOF'
import json
d=json.load(open("docs/packets/audits/status.json"))
bad=[(r["packet"],r.get("op"),v) for r in d["rows"] if r["packet"].startswith("login/")
     for v,c in r["cells"].items() if c.get("state") not in ("verified","n-a")]
print("remaining incomplete login cells:", bad)
assert not bad, bad
print("OK — all login cells verified or n-a")
EOF
```
Expected: `OK`.
- [ ] **Step 2b: Sanity-check the `n-a`** — if `ServerStatusRequest jms` ended `n-a`, confirm the status note carries the IDB justification (not a bare downgrade).
- [ ] **Step 3: `matrix --check` no-new-problems gate (§E)**
```bash
go run ./tools/packet-audit matrix --check 2>&1 | tee /tmp/matrix-check-final.txt
grep -i "login/" /tmp/matrix-check-final.txt    # expect: no orphan/dangling/stale/drift lines for login
grep -c "conflict" /tmp/matrix-check-final.txt   # expect: <= baseline from Task 0 Step 2
```
- [ ] **Step 4: `fname-doc` and `operations` checks introduce no new failures**
```bash
go run ./tools/packet-audit fname-doc --check 2>&1 | tail -5
go run ./tools/packet-audit operations --check 2>&1 | tail -5
```
- [ ] **Step 5: Go module gates (`libs/atlas-packet` — the only module touched)**
```bash
( cd libs/atlas-packet && go test -race ./... && go vet ./... && go build ./... )
```
Expected: all clean. **No `go.mod` was touched → no `docker buildx bake` required** (CLAUDE.md gate is conditional on a `go.mod` change; design §10). `tools/redis-key-guard.sh` is unaffected (login codecs touch no Redis) but run it from repo root if in doubt.
- [ ] **Step 6: If a real wire delta was found (§D.4)** — confirm `docs/tasks/task-108-login-packet-fixtures/audit.md` records it and the PR description will surface it. Otherwise note "no wire deltas; verification-only."
- [ ] **Step 7: Final commit** of `STATUS.md`/`status.json` if not already coupled into a cell commit, then proceed to code review (`superpowers:requesting-code-review`) before any PR (CLAUDE.md "Code Review Before PR").

---

## §E. `matrix --check` exit-code bar (design §9 / playbook §8)

`matrix --check` exits 1 from a pre-existing 🟥 registry-seed conflict backlog unrelated to login. The acceptance bar is **"no new problems," not a clean exit 0**:
- Zero orphan / dangling / stale / drift lines mentioning any `login/*` packet.
- The global conflict count must **not increase** above the Task 0 Step 2 baseline.
- Every login cell in scope reads ✅ (or justified `n-a`) after regen.
- `fname-doc --check` and `operations --check` introduce no new failures.

A net decrease in conflicts is a bonus, not required.

---

## §F. Self-review (writing-plans checklist)

- **Spec coverage:** All 20 PRD/design cells are enumerated in §C and assigned to Tasks 1–8; PRD acceptance criteria map to Task 9 (every row ✅/n-a; per-distinct-fname; coupled artifacts; `matrix`/`fname-doc`/`operations` checks; module gates). The PRD's "duplicated CharacterSelect/AllCharacterListSelect rows" question is answered by the op-keyed §C table + the `#suffix` fan-out. The PRD's jms n-a question is Task 8 Step 5.
- **Placeholder scan:** no TBD/TODO; every command block is concrete with substitution rules defined in §B; markers/evidence commands are exact.
- **Type/name consistency:** report filenames are bare struct names everywhere (§A.1); marker `packet=` paths keep `login/<dir>/`; version keys (`gms_v83/84/87`, `jms_v185`), template infixes (`gms_83_1`…`jms_185_1`), and export stems (`gms_v83`…`gms_jms_185`) are used consistently per the §B table.
- **Design-deviation log:** §A documents every place this plan overrides design.md, with source evidence — the executor must trust §A/§C over design.md where they differ.
