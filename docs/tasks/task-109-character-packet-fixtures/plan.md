# Character Packet-Fixture Verification Campaign — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Drive all **47 `incomplete` character-family cells** (`docs/packets/audits/status.json`) to `verified` (✅) — or to an IDB-justified `n-a` only where a function is genuinely version-absent — landing each promotion as its coupled artifacts (per-version audit report + `packet-audit:verify` marker + evidence record where the grader needs it, plus any export splice, route-wiring, or codec fix), with `STATUS.md`/`status.json` regenerated.

**Architecture:** This is a packet *verification* campaign, not a feature. Every character wire codec lives in **`libs/atlas-packet/character/{clientbound,serverbound}/`** — the single Go module this campaign touches; `services/atlas-login` (lifecycle) and `services/atlas-channel` (in-field) only *consume* these codecs. Each cell is promoted by: confirming that version's client read/write order, ensuring the function (and any `#suffix` branch) is in the version's committed IDA export (surgical absent-only splice if missing), generating the per-version audit report from that export, stacking a `packet-audit:verify` marker on the byte-test (writing a brand-new cross-version byte-fixture for the three fully-unverified packets that have no test file), pinning evidence where the grader needs it, regenerating the matrix, and committing the cell's artifacts together. Live-IDA work is grouped **by IDB** (`select_instance` is shared global state — never interleave two versions).

**Tech Stack:** Go (`libs/atlas-packet` — the only module touched), the `tools/packet-audit` CLI (root report-gen pipeline, `export` harvest, `evidence pin`, `matrix`, `matrix --check`, `fname-doc --check`, `operations --check`), and the `mcp__ida-pro__*` MCP tools (decompile / rename / address lookup against the live IDBs).

> **Path convention:** `<worktree-root>` = `.worktrees/task-109-character-packet-fixtures/`. All paths are repo-relative from there. Run every command from `<worktree-root>`.

---

## §A. Planning-phase corrections to design.md (verified from source — READ FIRST)

`design.md` is correct on the goal, the three-phase shape (fully-unverified group first), the `candidatesFromFName` fan-out being already wired (§6), the Class-E name-and-splice doctrine (§5), and the module impact (§10). **But six concrete claims were falsified during planning by reading `run.go`, the committed exports, the committed reports, and the existing test files. Trust this plan's §A/§C over design.md where they differ** — each correction is evidenced so the executor can re-confirm.

1. **Report filenames and marker `packet=` paths are the DESCRIPTIVE `name` field from `run.go`, NOT `qualifiedWriterName`/`Character<Struct>`.** Design §2/§3 says the report file derives from `qualifiedWriterName = TitleCase(pkg)+Struct` (e.g. `CharacterEffectQuest`). **Wrong for every in-scope cell.** The in-scope character ops route in `tools/packet-audit/cmd/run.go` `candidatesFromFName` with **`pkg: ""`** (omitted) and a descriptive `name` (`run.go` returns e.g. `{name: "EffectQuest"}`, `{name: "BuffGive"}`, `{name: "CharacterSpawn"}`, `{name: "CharacterList"}`, `{name: "CharacterChairShow"}`). `qualifiedWriterName("", name)` (run.go:222) returns `name` verbatim. Verified on disk: the reports are `docs/packets/audits/gms_v87/{EffectQuest,BuffGive,CharacterList,CharacterChairShow,CharacterAppearanceUpdate}.json` — **never** `CharacterEffectQuest.json`. The marker `packet=` path matches exactly (verified in committed tests): `packet=character/clientbound/CharacterChairShow`, `packet=character/clientbound/EffectQuest`, `packet=character/serverbound/CheckName`. §C lists the exact `name`/filename per cell — use it, do not derive a `Character`-prefixed name.

2. **Many in-scope cells already carry their `packet-audit:verify` marker; the gap is the audit report or the fixture, not the marker.** Verified by grepping the test files:
   - `serverbound/key_map_change_test.go` already has markers for **all five** versions (v83/v84/v87/v95/jms). The four incomplete `CHANGE_KEYMAP` cells (v83/v87/v95/jms) and the v84 sub-struct are incomplete for report/verdict reasons, not a missing marker.
   - `serverbound/{create,delete,heal_over_time}_test.go`, `clientbound/{spawn,info}_test.go` carry v83/v84/v87/v95 markers — **missing only `jms_v185`**.
   - `serverbound/check_name_test.go`, `clientbound/chair_show_test.go` carry only v87/v95 — missing v83/v84 (chair also jms).
   - `clientbound/buff_give_test.go` carries v83/v84/v87/v95 for both `BuffGive` and `BuffGiveForeign` — missing only jms.
   - `clientbound/movement_test.go` carries v83/v87/v95 — missing v84/jms.
   - `serverbound/auto_distribute_ap_test.go` carries v83/v87/v95 — missing v84.
   §C records the per-cell marker state. **Always re-grep the test file before adding a marker** — never duplicate a line.

3. **status.json `note` strings are HINTS, not ground truth — per-cell regen-then-read is mandatory.** Counter-example proven in planning: `serverbound/KeyMapChange CHANGE_KEYMAP jms_v185` carries note `"no audit report"`, yet `docs/packets/audits/jms_v185/KeyMapChange.json` **exists** with `FlatInvalid: true` and `verdicts [0,0,2,2,2,2]` (verdict 2 = width mismatch; the evidence record is `category: TRUNCATION` documenting an export loop-count 89-vs-90 artifact). So that cell is a **Class-B verdict adjudication**, not a missing report. The canonical per-cell procedure (§D) always **regenerates the report and reads the actual `FlatInvalid`/`Verdict` values**, then branches — it never trusts the note.

4. **Three jms cells the design classified as Class-A (report-gen-only) are actually export-ABSENT → need a jms-IDB splice.** Verified key-presence in `docs/packets/ida-exports/gms_jms_185.json`:
   - `CLogin::OnViewAllCharResult#CharacterViewAll{Characters,Count,SearchFailed}` — **ABSENT on jms** (present v83/v84/v87/v95). So `clientbound/CharacterViewAllCharacters jms` (design §2 "no audit report"/Class A) is **Class E**: harvest+splice the `#suffix` keys from the jms IDB first.
   - `CWvsContext::SendStatChangeRequest` (HealOverTime) — **ABSENT on jms** (the export contains only the different fn `SendStatChangeRequestByItemOption`). So `serverbound/HealOverTime jms` (design §2 note "function present in jms export → Class A") is **Class E**: jms-IDB name+splice.
   - `CUserRemote::OnSetActivePortableChair` (ChairShow) — **ABSENT on jms** too (design §5 only flagged v83/v84). So `clientbound/CharacterChairShow jms` joins the v83/v84 Class-E chair cluster.

5. **Three "fully-unverified" packets already have partial committed reports — the deliverable is the NEW byte-fixture test FILE, not the report.** Verified: `list_test.go`, `appearance_update_test.go`, `effect_quest_test.go` are **absent** (the codecs `list.go`/`appearance_update.go`/`effect_quest.go` exist with no `*_test.go`). But `CharacterAppearanceUpdate.json` and `EffectQuest.json` reports **already exist for all five versions** (`🔍` verdicts), and `CharacterList.json` exists for v83/v84/v87/v95 (`❌`/`🔍`; jms has none). So Phase A's load-bearing work is **writing the three new cross-version byte-fixture files + markers + evidence**; report-gen re-runs to refresh, and the `🔍`/`❌` verdicts are adjudicated against the freshly-written fixture (a `🔍` "tier-1 without fixture" clears once the fixture exists and the report regenerates clean).

6. **Routing is template-present for the verified-sibling ops; confirm per serverbound cell via `matrix --check`, do not assume a gap.** Planning verified `template_jms_185_1.json` routes `0x9F → CharacterKeyMapChangeHandle` (so jms KeyMapChange IS routed). Per playbook §9 a serverbound cell needs its op **routed** in that version's seed template; the **arbiter is `matrix --check`'s `routedElsewhere && !routed` conflict line**. For each serverbound cell, after report+evidence, if (and only if) `matrix --check` emits that conflict for the packet, wire the route into the version's template mirroring a verified sibling version's entry (e.g. the v95 entry `{"opCode": "0x9F", "validator": "LoggedInValidator", "handler": "CharacterKeyMapChangeHandle"}`). Do not pre-emptively edit templates on an unverified assumption.

**Net effect on work shape:** all five IDBs are needed (v83, v84, v87, v95, jms), each with live decompiles for the Phase-A fixtures; v83/v84/jms additionally carry Class-E name+splice work; jms additionally carries two export splices (ViewAll, HealOverTime). A genuinely IDA-free Stage 1 exists (the jms Class-A holes whose function is present + the three GMS KeyMapChange report-gens), but it is smaller than the design's "bulk is deterministic report-gen" framing implies.

---

## §B. Reference facts (read once before any task)

### Per-version mapping

| Version key (marker `version=`) | Seed template | Committed export json | Audit report dir | IDA port (memory IDBs_v9 — **confirm by binary NAME via `list_instances`, never trust the number**) |
|---|---|---|---|---|
| `gms_v83` | `services/atlas-configurations/seed-data/templates/template_gms_83_1.json` | `docs/packets/ida-exports/gms_v83.json` | `docs/packets/audits/gms_v83/` | 13341 |
| `gms_v84` | `services/atlas-configurations/seed-data/templates/template_gms_84_1.json` | `docs/packets/ida-exports/gms_v84.json` | `docs/packets/audits/gms_v84/` | 13337 |
| `gms_v87` | `services/atlas-configurations/seed-data/templates/template_gms_87_1.json` | `docs/packets/ida-exports/gms_v87.json` | `docs/packets/audits/gms_v87/` | 13340 |
| `gms_v95` | `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` | `docs/packets/ida-exports/gms_v95.json` | `docs/packets/audits/gms_v95/` | 13339 |
| `jms_v185` | `services/atlas-configurations/seed-data/templates/template_jms_185_1.json` | `docs/packets/ida-exports/gms_jms_185.json` | `docs/packets/audits/jms_v185/` | 13338 (**use the clean `*_U_DEVM` build**, not the SMC retail dump — playbook §10) |

Substitution shorthands used below: `<V>` = template infix (`gms_83_1`/`gms_84_1`/`gms_87_1`/`gms_95_1`/`jms_185_1`), `<EXPORT>` = export stem (`gms_v83`/`gms_v84`/`gms_v87`/`gms_v95`/`gms_jms_185`), `<version-key>` = marker/audit-dir key (`gms_v83`/`gms_v84`/`gms_v87`/`gms_v95`/`jms_v185`).

### Promotion mechanism (what makes a cell ✅) — playbook §3/§7/§9

- **Clientbound tier-1** (every in-scope clientbound character row IS tier-1 — design §3): per-version **audit report** (verdict clean) + stacked **`packet-audit:verify` marker** + **pinned evidence** with a `verifies:` line.
- **Serverbound op rows** (`CheckName`, `CreateCharacter`, `DeleteCharacter`, `AutoDistributeAp ×2`, `HealOverTime`, `KeyMapChange CHANGE_KEYMAP`): report + marker + **pinned evidence** AND the op **routed** in that version's seed template (§A.6).
- **Serverbound sub-struct row** (`KeyMapChange` `op=None`, v84): report + marker; evidence/routing as the sibling op rows require — read `matrix --check`, mirror the v84 `CHANGE_KEYMAP` op-row treatment.

The **arbiter is `go run ./tools/packet-audit matrix --check`** — a cell is done when it reads `verified` in regenerated `status.json` and introduces no new `--check` problem mentioning a `character/*` packet, and the global conflict count does not increase (§E baseline).

### Report-gen command (deterministic, no live IDA)

```bash
go run ./tools/packet-audit \
  -csv-clientbound "docs/packets/MapleStory Ops - ClientBound.csv" \
  -csv-serverbound "docs/packets/MapleStory Ops - ServerBound.csv" \
  -template services/atlas-configurations/seed-data/templates/template_<V>.json \
  -ida-source docs/packets/ida-exports/<EXPORT>.json \
  -output /tmp/rpt-<version-key>
# then copy ONLY the needed report(s) into the committed audit dir:
cp /tmp/rpt-<version-key>/<version-key>/<Name>.json docs/packets/audits/<version-key>/<Name>.json
cp /tmp/rpt-<version-key>/<version-key>/<Name>.md   docs/packets/audits/<version-key>/<Name>.md
```

`<Name>` is the descriptive report name from §C (e.g. `EffectQuest`, `BuffGive`, `CharacterList`) — **never** `Character`-prefixed for the pkg-empty cells (§A.1). After copy, read the verdict:

```bash
python3 -c "import json;d=json.load(open('docs/packets/audits/<version-key>/<Name>.json'));print('FlatInvalid',d['FlatInvalid'],'verdicts',[r['Verdict'] for r in d['Rows']],'Addr',d.get('Address'))"
```

A cell is verdict-clean only when `FlatInvalid == False` **and** every `Rows[].Verdict == 0`.

### Export key-presence check (re-verify before assuming Stage 1 vs a splice)

```bash
python3 -c "import json,sys; d=json.load(open(sys.argv[1])); f=d.get('functions',d); print(sys.argv[2] in f)" \
  docs/packets/ida-exports/<EXPORT>.json "C...::Fn#Suffix"
```

### Export splice procedure (Class-E / splice cells, playbook §10) — surgical, absent-only

**Never overwrite a committed export** (re-running `export` drifts ~150 unrelated keys). To add ONE function (or `#suffix` branch):

```bash
# 1. Select & confirm the IDB FIRST (binary name, not port):
#    mcp__ida-pro__list_instances → select_instance(<port>) → confirm name.
# 2. Harvest to a TEMP file targeting that IDB's port:
go run ./tools/packet-audit export \
  --version <version-key> \
  --prior-export "" \
  --pending /tmp/roster-<version-key>.md \
  --descent-depth 12 \
  --ida-url http://127.0.0.1:<PORT>/mcp --ida-port <PORT> \
  --output /tmp/harvest-<version-key>.json
# 3. Surgically splice ONLY the needed key(s) into the committed export with a small
#    Python merge (absent-only for helpers; overwrite a stub only for the one sender).
#    Strip any {op: Delegate, ref: COutPacket} ctor artifact from a spliced send entry.
# 4. git diff docs/packets/ida-exports/<EXPORT>.json — confirm ONLY the intended keys changed.
```

`/tmp/roster-<version-key>.md` lists the base fname(s), one per line (e.g. `- CLogin::OnViewAllCharResult`); the exporter's branch descent emits the `#suffix` synthetic entries. For a **present-but-unnamed** sender (Class E), first **name it** in the IDB via the byte signature `6A <op> 8D 8D ?? ?? ?? ?? E8` structure-matched to the named v87/v95 twin (playbook §10), then harvest.

### Marker idiom

Stack one `// packet-audit:verify` line per version above the test function. Format (the last path segment is the descriptive `name`, §A.1):

```
// packet-audit:verify packet=character/<dir>/<Name> version=<version-key> ida=0x<addr>
```

`ida=` is the **function's own address** from the report `Address` / decompile, NOT the dispatch opcode. Reference idioms: `clientbound/buff_give_test.go` (8 stacked markers across two structs), `serverbound/create_test.go` (4 stacked).

### Evidence pin idiom

```bash
go run ./tools/packet-audit evidence pin \
  --packet character/<dir>/<Name> --version <version-key> \
  --ida "<FName-incl-#suffix>" --category TIER1-FIXTURE
# then OPEN docs/packets/evidence/<version-key>/character.<dir>.<Name>.yaml and add:
#   verifies:
#     - <test file path>#<TestName>
```

Reference records: `docs/packets/evidence/gms_v83/character.clientbound.BuffGive.yaml`, `docs/packets/evidence/jms_v185/character.serverbound.KeyMapChange.yaml`.

---

## §C. The 47 cells — verified planning-time state and recipe

Keyed by (packet, op, version). "Report" / "fn in export" / "marker" are the **verified planning-time** states. "Stage/IDB" routes the work. Read §D for the procedure each recipe abbreviates. The verdict symbol in a note is the audit report's verdict, NOT the cell state — **re-adjudicate per cell against a fresh report (§A.3)**.

### Stage 1 — report-gen-only (no live IDA): function present, no decompile needed

| # | packet / op | ver | report (current) | fn in export | marker | recipe |
|---|---|---|---|---|---|---|
| 1 | `clientbound/AddCharacterEntry` ADD_NEW_CHAR_ENTRY | jms | none | `OnCreateNewCharacterResult` ✓ | add jms | regen `AddCharacterEntry` → +jms marker on `add_entry_test.go` → pin evidence |
| 2 | `clientbound/BuffGive` GIVE_BUFF | jms | none | `OnTemporaryStatSet` ✓ | add jms | regen `BuffGive` → +jms marker on `buff_give_test.go` → pin evidence |
| 3 | `clientbound/BuffGiveForeign` GIVE_FOREIGN_BUFF | jms | none | `OnSetTemporaryStat` ✓ | add jms | regen `BuffGiveForeign` → +jms marker on `buff_give_test.go` → pin evidence |
| 4 | `clientbound/CharacterInfo` CHAR_INFO | jms | none | `OnCharacterInfo` ✓ | add jms | regen `CharacterInfo` → +jms marker on `info_test.go` → pin evidence |
| 5 | `clientbound/CharacterSpawn` SPAWN_PLAYER | jms | none | `OnUserEnterField` ✓ | add jms | regen `CharacterSpawn` → +jms marker on `spawn_test.go` → pin evidence |
| 6 | `serverbound/CheckName` CHECK_CHAR_NAME | jms | exists? (regen) | `SendCheckDuplicateIDPacket` ✓ | add jms | regen `CheckName` → +jms marker on `check_name_test.go` → pin evidence → confirm routed (§A.6) |
| 7 | `serverbound/CreateCharacter` CREATE_CHAR | jms | none | `SendNewCharPacket` ✓ | add jms | regen `CreateCharacter` → +jms marker on `create_test.go` → pin evidence → confirm routed |
| 8 | `serverbound/DeleteCharacter` DELETE_CHAR | jms | none | `SendDeleteCharPacket` ✓ | add jms | regen `DeleteCharacter` → +jms marker on `delete_test.go` → pin evidence → confirm routed |
| 9 | `serverbound/KeyMapChange` CHANGE_KEYMAP | v83 | exists (regen) | `SaveFuncKeyMap` ✓ | present | regen `KeyMapChange` → verdict clean → keep/confirm v83 marker → pin/confirm evidence → confirm routed |
| 10 | `serverbound/KeyMapChange` CHANGE_KEYMAP | v87 | exists (regen) | `SaveFuncKeyMap` ✓ | present | regen `KeyMapChange` → adjudicate → confirm marker/evidence/routing |
| 11 | `serverbound/KeyMapChange` CHANGE_KEYMAP | v95 | exists (regen) | `SaveFuncKeyMap` ✓ | present | regen `KeyMapChange` → adjudicate → confirm marker/evidence/routing |
| 12 | `serverbound/KeyMapChange` (sub-struct, op=None) | v84 | exists (regen) | `SaveFuncKeyMap` ✓ | present | regen `KeyMapChange` → adjudicate the sub-struct row → confirm marker/evidence/routing |

> Stage-1 escalation rule: if any cell's regenerated report is NOT verdict-clean, do **not** force it — escalate that cell into its version's Stage-2 IDB task (§D.4 adjudication). KeyMapChange v87/v95/v84 and CheckName jms are the most likely escalations (KeyMapChange's TRUNCATION family, §A.3).

### Stage 2 — live-IDA, grouped by IDB

**Phase-A new-fixture cells (Class C):** `CharacterList` (`list_test.go` — NEW), `CharacterAppearanceUpdate` (`appearance_update_test.go` — NEW), `EffectQuest` two ops (`effect_quest_test.go` — NEW). One cross-version table-driven file per packet; the file is **created in the v83 IDB task** and **appended** (one version row + marker + evidence) in the v84/v87/v95/jms IDB tasks. Reports for AppearanceUpdate/EffectQuest exist (`🔍`) all five versions; CharacterList exists v83/v84/v87/v95 (`❌`/`🔍`), none on jms.

| # | packet / op | ver | report | fn in export | new fixture? | recipe (per §D) |
|---|---|---|---|---|---|---|
| 13–17 | `clientbound/CharacterList` CHARLIST | v83,v84,v87,v95,jms | v83❌ v84🔍 v87❌ v95❌ jms:none | `OnSelectWorldResult` ✓ all | **yes** `list_test.go` | decompile char-list read order per version (full nested avatar/look block); write fixture row; +marker; regen; adjudicate ❌; pin evidence |
| 18–22 | `clientbound/CharacterAppearanceUpdate` UPDATE_CHAR_LOOK | v83,v84,v87,v95,jms | 🔍 all | `OnAvatarModified` ✓ all | **yes** `appearance_update_test.go` | decompile avatar-look read order per version; write fixture row; +marker; regen (🔍 clears with fixture); pin evidence |
| 23–27 | `clientbound/EffectQuest` SHOW_FOREIGN_EFFECT | v83,v84,v87,v95,jms | 🔍 all | `OnEffect` ✓ all | **yes** `effect_quest_test.go` | decompile `OnEffect` mode body per version; write fixture row; +marker (`EffectQuest`); regen; pin evidence |
| 28–32 | `clientbound/EffectQuest` SHOW_ITEM_GAIN_INCHAT | v83,v84,v87,v95,jms | 🔍 all | `OnEffect` ✓ all | **same file, distinct op** | distinct fixture case + **its own** marker line + **its own** evidence record per version (op-keyed, §A/design §1.1) |

**Class-E name-and-splice cells:** function export-ABSENT but verified on v87/v95 → present-but-unnamed in the older/jms IDB → name + splice → then report-gen. `n-a` ONLY if the IDB genuinely lacks it (verify by trying — playbook "producible vs blocker").

| # | packet / op | ver | fn in export | marker | recipe |
|---|---|---|---|---|---|
| 33 | `clientbound/CharacterExpression` FACIAL_EXPRESSION | v83 | `OnEmotion` ✗ (✓v87/v95) | add v83 | name+splice `OnEmotion` (v83 IDB) → regen `CharacterExpression` → +v83 marker on `expression_test.go` → pin evidence |
| 34 | `clientbound/CharacterExpression` FACIAL_EXPRESSION | v84 | `OnEmotion` ✗ | add v84 | name+splice `OnEmotion` (v84 IDB) → regen → +v84 marker → pin evidence |
| 35 | `clientbound/CharacterChairShow` SHOW_CHAIR | v83 | `OnSetActivePortableChair` ✗ | add v83 | name+splice (v83 IDB) → regen `CharacterChairShow` → +v83 marker on `chair_show_test.go` → pin evidence |
| 36 | `clientbound/CharacterChairShow` SHOW_CHAIR | v84 | `OnSetActivePortableChair` ✗ | add v84 | name+splice (v84 IDB) → regen → +v84 marker → pin evidence |
| 37 | `clientbound/CharacterChairShow` SHOW_CHAIR | jms | `OnSetActivePortableChair` ✗ (§A.4) | add jms | name+splice (jms IDB) → regen → +jms marker → pin evidence |
| 38 | `serverbound/CheckName` CHECK_CHAR_NAME | v83 | `SendCheckDuplicateIDPacket` ✗ | add v83 | name+splice (v83 IDB) → regen `CheckName` → +v83 marker on `check_name_test.go` → pin evidence → confirm routed |
| 39 | `serverbound/CheckName` CHECK_CHAR_NAME | v84 | `SendCheckDuplicateIDPacket` ✗ | add v84 | name+splice (v84 IDB) → regen → +v84 marker → pin evidence → confirm routed |

**Splice + adjudication cells (jms export gaps & GMS ❌ verdicts):**

| # | packet / op | ver | report | fn in export | marker | recipe |
|---|---|---|---|---|---|---|
| 40 | `clientbound/CharacterViewAllCharacters` VIEW_ALL_CHAR | jms | none | `OnViewAllCharResult#CharacterViewAll*` ✗ (§A.4) | add jms | harvest+splice the 3 `#suffix` keys (jms IDB) → regen `CharacterViewAllCharacters` → +jms marker on `view_all_test.go` → pin evidence |
| 41 | `clientbound/CharacterViewAllCharacters` VIEW_ALL_CHAR | v84 | 🚫 | `#suffix` ✓ v84 | add v84 | decompile to adjudicate 🚫 (v84 IDB) → regen → +v84 marker → pin evidence |
| 42 | `clientbound/CharacterMovement` MOVE_PLAYER | v84 | ❌ | `OnMove` ✓ | add v84 | decompile `OnMove` (v84 IDB), adjudicate ❌ vs v83 (body ≡ v83 below ~0x3D) → regen → +v84 marker on `movement_test.go` → pin evidence |
| 43 | `clientbound/CharacterMovement` MOVE_PLAYER | jms | ❌ | `OnMove` ✓ | add jms | decompile `OnMove` (jms IDB), adjudicate ❌ (jms is the real-delta suspect, §D.4) → regen → +jms marker → pin evidence |
| 44 | `serverbound/AutoDistributeAp` DISTRIBUTE_AP | v84 | ❌ | `SendAbilityUpRequest#DistributeAp` ✓ | add v84 | decompile (v84 IDB) adjudicate → regen `AutoDistributeAp` → +v84 marker on `auto_distribute_ap_test.go` → pin evidence → confirm routed |
| 45 | `serverbound/AutoDistributeAp` AUTO_DISTRIBUTE_AP | v84 | ❌ | `SendAbilityUpRequest#AutoDistributeAp` ✓ | add v84 | decompile (v84 IDB) adjudicate → regen → +v84 marker (distinct op) → pin evidence → confirm routed |
| 46 | `serverbound/HealOverTime` HEAL_OVER_TIME | jms | none | `SendStatChangeRequest` ✗ (§A.4) | add jms | name+splice (jms IDB) → regen `HealOverTime` → +jms marker on `heal_over_time_test.go` → pin evidence → confirm routed |
| 47 | `serverbound/KeyMapChange` CHANGE_KEYMAP | jms | exists, FlatInvalid (verdict 2 ×4, TRUNCATION) | `SaveFuncKeyMap` ✓ | present | adjudicate the truncation verdicts (jms IDB): confirm the documented loop-89-vs-90 export artifact (`evidence …KeyMapChange.yaml` `category: TRUNCATION`) → regen → keep marker → refresh evidence → confirm routed |

> **Verdict-clean rule (every cell):** ✅ requires the regenerated report `FlatInvalid: false` and every `Rows[].Verdict == 0`. A surviving non-zero verdict is either (a) **stale** (regen against the up-to-date export clears it) or (b) a **real wire delta** → STOP, fix-first (§D.4). Never re-pin a ❌ as ✅ without a decompile that explains it. The KeyMapChange TRUNCATION family (#47) is the one documented VERIFIED-EXCEPTION — re-confirm the export artifact before accepting it, don't generalize it.

---

## §D. Canonical per-cell procedure (referenced by the tasks)

Every cell follows this skeleton. §C gives the concrete substitutions; do not improvise beyond them.

**D.1 — Ensure the function is in the export.** If §C says the fn/`#suffix` is ✗, run the §B splice procedure against the cell's IDB FIRST (Class E: name the present-but-unnamed sender, then harvest). Confirm with the §B key-presence check. If ✓, skip.

**D.2 — Generate the report.** Run the §B report-gen for the version; copy `<Name>.{json,md}` into `docs/packets/audits/<version-key>/`. Read `FlatInvalid` + every `Rows[].Verdict`.

**D.3 — If verdict clean (FlatInvalid false, all 0):** add/confirm the stacked marker (§B; re-grep first — §A.2), write/extend the byte-fixture if §C says a new fixture/row is needed (§D.7 for Phase-A files), pin evidence (§B) for every in-scope row (all are tier-1 / serverbound), then go to D.5.

**D.4 — If verdict NOT clean:** `select_instance` the cell's IDB (confirm by binary name), decompile the cell's fname (descend into helper reads), write the full ordered read/write list, compare against the Atlas codec in `libs/atlas-packet/character/<dir>/<file>.go`.
  - **Stale/cosmetic** (codec matches the decompile; the ❌ was a pre-splice/pre-fixture artifact) → the splice/regen already fixed it; re-read the report → D.3.
  - **Real wire delta** (inserted field / changed width / different guard) → **STOP, surface it** in the PR description and `docs/tasks/task-109-character-packet-fixtures/audit.md`. Land the **codec fix first** as its own commit in `libs/atlas-packet/character/<dir>/` (add the version branch; update the cross-version byte-test to expect the divergence; `go test -race ./...` green), then resume D.2 against the corrected codec. No delta is *expected* for v84 (body ≡ v83 below ~0x3D — memory `bug_majorversion_gt83_is_off_by_one_v87`); **jms is the prime real-delta suspect** (genuinely different structure).

**D.5 — Regenerate the matrix and verify promotion.**
```bash
go run ./tools/packet-audit matrix
go run ./tools/packet-audit matrix --check
```
Confirm the cell reads `verified` in `docs/packets/audits/status.json` and `--check` reports no new `character/*` problem and no increase in the global conflict count (§E baseline). For a serverbound cell, if `--check` emits `routedElsewhere && !routed` for the packet, wire the route into the version's template (§A.6) and re-run.

**D.6 — Commit the cell's coupled artifacts together** (report `.json`+`.md`, test/marker change, evidence yaml, any export splice, any template route-wire, any codec fix, plus regenerated `STATUS.md`+`status.json`):
```bash
git add docs/packets/audits/<version-key>/<Name>.json docs/packets/audits/<version-key>/<Name>.md \
        libs/atlas-packet/character/<dir>/<test>.go \
        docs/packets/evidence/<version-key>/character.<dir>.<Name>.yaml \
        docs/packets/ida-exports/<EXPORT>.json \
        docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "verify(character): <packet>/<op> <version-key> ✅"
git rev-parse --show-toplevel   # must end with /.worktrees/task-109-character-packet-fixtures
git branch --show-current       # must be task-109-character-packet-fixtures
```
(Drop `git add` lines for artifacts a cell doesn't produce.) For the Phase-A packets, one commit per (packet × version) cell — the shared test file accretes one version row per commit.

**D.7 — Phase-A new-fixture file authoring.** For `CharacterList`/`CharacterAppearanceUpdate`/`EffectQuest`:
- Use the repo idiom: `package clientbound`, import `pt "github.com/Chronicle20/atlas/libs/atlas-packet/test"`, loop `for _, v := range pt.Variants` OR a golden-byte table accessing `pt.Variants` by index (reference `libs/atlas-packet/party/clientbound/invite_test.go` for the golden-byte table; `buff_give_test.go` for the round-trip+golden-tail mix). `pt.Variants` already includes v83/v84/v87/v95/jms (`libs/atlas-packet/test/context.go:18-32`).
- The fixture MUST exercise the **full body** including the nested avatar/look block (CharacterList, AppearanceUpdate) or the effect mode body (EffectQuest) end-to-end — a length-only or mode-only assertion is a false pass (memory `feedback_dispatcher_mode_byte_is_false_pass`). Hand-compute expected bytes from the decompiled read order; cite the decompile line per field in a comment (playbook §5).
- Create the file in the v83 IDB task with the v83 row + v83 marker; each later IDB task **appends** its version's row + stacks its marker. If a version's body diverges (jms), the table carries a per-version `want`.

---

## Task 0: Baseline snapshot (no code change)

**Files:** none (read-only baseline capture).

- [ ] **Step 1: Confirm worktree + green build**
```bash
pwd   # .../.worktrees/task-109-character-packet-fixtures
git branch --show-current   # task-109-character-packet-fixtures
( cd libs/atlas-packet && go build ./... && go test -race ./character/... && go vet ./... )
```
Expected: build/test/vet clean (the character codecs already pass).

- [ ] **Step 2: Capture the matrix-check baseline (the §E reference)**
```bash
go run ./tools/packet-audit matrix --check 2>&1 | tee /tmp/matrix-check-baseline.txt
grep -c "conflict" /tmp/matrix-check-baseline.txt          # record this number
grep -i "character/" /tmp/matrix-check-baseline.txt        # record any pre-existing character lines
```
Expected: a non-zero exit from the pre-existing registry-seed conflict backlog (playbook §8). **Record the conflict count and any `character/` lines** — the acceptance bar is "no increase, no new character line" (§E).

- [ ] **Step 3: Confirm the 47-cell work list matches §C**
```bash
python3 - <<'EOF'
import json
d=json.load(open("docs/packets/audits/status.json"))
rows = d["rows"] if isinstance(d,dict) and "rows" in d else d
n=0
for r in rows:
    if not r.get("packet","").startswith("character/"): continue
    for v,c in r.get("cells",{}).items():
        if c.get("state") not in ("verified","n-a"):
            n+=1; print(r["packet"], r.get("op"), v, c.get("note"))
print("TOTAL:", n)
EOF
```
Expected: `TOTAL: 47`, matching §C. If it differs, the matrix drifted since planning — re-reconcile §C before proceeding. **No commit.**

---

## Stage 1 — report-gen-only cells (no live IDA)

Cells #1–#12 (§C Stage-1 table). Function present in the committed export; the work is report-gen + marker + evidence (+ routing confirm for serverbound). Each is its own commit via §D. **If a regenerated report is not verdict-clean, escalate that cell to its Stage-2 IDB task — do not force it.**

### Task 1: jms clientbound Class-A holes — `AddCharacterEntry`, `BuffGive`, `BuffGiveForeign`, `CharacterInfo`, `CharacterSpawn` (cells #1–#5)

**Files:** `docs/packets/audits/jms_v185/{AddCharacterEntry,BuffGive,BuffGiveForeign,CharacterInfo,CharacterSpawn}.{json,md}`; markers on `clientbound/{add_entry,buff_give,buff_give,info,spawn}_test.go`; evidence `docs/packets/evidence/jms_v185/character.clientbound.{AddCharacterEntry,BuffGive,BuffGiveForeign,CharacterInfo,CharacterSpawn}.yaml`.

- [ ] **Step 1: Report-gen all five** — §B report-gen with `<V>=jms_185_1`, `<EXPORT>=gms_jms_185`, `<version-key>=jms_v185`; copy the five `<Name>.{json,md}`. (`AddCharacterEntry`/`BuffGiveForeign` are the descriptive `name`s — §A.1.)
- [ ] **Step 2: Read each verdict** (§B). Expected clean. Any non-clean → escalate that struct to **Task 9 (jms IDB)** §D.4; continue the rest.
- [ ] **Step 3: Add the `jms_v185` marker** to each test (re-grep first — buff_give carries v83/v84/v87/v95 for BOTH `BuffGive` and `BuffGiveForeign`; add the jms line for each). `ida=` = each report `Address`.
- [ ] **Step 4: Pin evidence** for each (clientbound tier-1): `evidence pin --packet character/clientbound/<Name> --version jms_v185 --ida "<FName>" --category TIER1-FIXTURE`; add `verifies:` lines pointing at each test func.
- [ ] **Step 5: `go test -race ./character/...`** green; **regen matrix + check (§D.5)** — confirm the five jms cells → `verified`.
- [ ] **Step 6: Commit** each cell separately (§D.6) — five commits (or group the two buff cells, shared file).

### Task 2: jms serverbound Class-A holes — `CheckName`, `CreateCharacter`, `DeleteCharacter` (cells #6, #7, #8)

**Files:** `docs/packets/audits/jms_v185/{CheckName,CreateCharacter,DeleteCharacter}.{json,md}`; markers on `serverbound/{check_name,create,delete}_test.go`; evidence `docs/packets/evidence/jms_v185/character.serverbound.{CheckName,CreateCharacter,DeleteCharacter}.yaml`; possibly `template_jms_185_1.json` (only if §D.5 routing conflict).

- [ ] **Step 1: Report-gen all three** (`<V>=jms_185_1`); copy; read verdicts. Expected clean.
- [ ] **Step 2: Add the `jms_v185` marker** to each test (`create_test.go`/`delete_test.go` carry v83/v84/v87/v95; `check_name_test.go` carries only v87/v95 — add jms).
- [ ] **Step 3: Pin evidence** for each (serverbound): `--ida "CLogin::SendCheckDuplicateIDPacket"` / `"CLogin::SendNewCharPacket"` / `"CLogin::SendDeleteCharPacket"`; add `verifies:`.
- [ ] **Step 4: `go test -race ./character/...`** green; **regen matrix + check (§D.5).** If `--check` emits `routedElsewhere && !routed` for any of these jms packets, wire the route into `template_jms_185_1.json` mirroring a verified sibling (§A.6) and re-run. Confirm all three → `verified`.
- [ ] **Step 5: Commit** each cell separately (§D.6).

### Task 3: GMS `KeyMapChange` report-gen — v83, v87, v95, + v84 sub-struct (cells #9, #10, #11, #12)

**Files:** `docs/packets/audits/{gms_v83,gms_v87,gms_v95,gms_v84}/KeyMapChange.{json,md}`; markers already present on `serverbound/key_map_change_test.go` (all 5 versions — confirm); evidence `docs/packets/evidence/{gms_v83,gms_v87,gms_v95,gms_v84}/character.serverbound.KeyMapChange.yaml`.

- [ ] **Step 1: Report-gen each** (`<V>` = `gms_83_1`/`gms_87_1`/`gms_95_1`/`gms_84_1`); copy `KeyMapChange.{json,md}`; read verdicts.
- [ ] **Step 2: Branch per version.** Verdict clean → confirm the existing marker `ida=` matches the report `Address` (fix if drifted); pin/confirm evidence; confirm routed (`CharacterKeyMapChangeHandle` is present in these templates — §A.6). **Verdict NOT clean** (the TRUNCATION family may surface verdict 2 like jms, §A.3) → escalate that version to its Stage-2 IDB task §D.4; if it is the same documented loop-89-vs-90 truncation artifact, treat as the VERIFIED-EXCEPTION (refresh evidence `category: TRUNCATION`), else fix-first.
- [ ] **Step 3: `go test -race ./character/...`** green; **regen matrix + check (§D.5)** — confirm each promoted cell → `verified`.
- [ ] **Step 4: Commit** each cell separately (§D.6).

---

## Stage 2 — live-IDA cells, grouped by IDB

**Discipline (playbook §10):** `select_instance` is shared global state. Do ALL of one IDB's cells, then move on. Before any decompile/harvest: `mcp__ida-pro__list_instances`, pick the instance whose loaded binary NAME matches the target version, `select_instance(<port>)`. Never hardcode the port — confirm by name. jms: the clean `*_U_DEVM` build only.

**Ordering:** Task 4 (v83) FIRST — it creates the three Phase-A test files (`list_test.go`, `appearance_update_test.go`, `effect_quest_test.go`) with the v83 rows; Tasks 5–8 append their version rows (§D.7). Then v84, v87, v95, jms in any order (each independent after v83).

### Task 4: v83 IDB — Phase-A v83 rows + Class-E v83 cluster (cells #13, #18, #23, #28, #33, #35, #38)

**IDB:** v83 (confirm by name). **Files:** NEW `libs/atlas-packet/character/clientbound/{list,appearance_update,effect_quest}_test.go`; `docs/packets/ida-exports/gms_v83.json` (Class-E splices); markers on `clientbound/{expression,chair_show}_test.go` + `serverbound/check_name_test.go`; reports + evidence under `gms_v83/`.

- [ ] **Step 1: Select the v83 instance** — `list_instances` → `select_instance(<v83 port>)`; confirm binary name.
- [ ] **Step 2: `CharacterList` v83 (Phase A, cell #13)** — decompile `CLogin::OnSelectWorldResult` char-list read order (full nested avatar/look block); CREATE `list_test.go` with the v83 golden-byte row (§D.7), cite decompile lines; +v83 marker `packet=character/clientbound/CharacterList`; regen `CharacterList` (`<V>=gms_83_1`); adjudicate the `❌` (§D.4 — v83 is a reference version, the codec is likely right and the ❌ was a no-fixture artifact); pin evidence (`--ida "CLogin::OnSelectWorldResult"`); commit (§D.6).
- [ ] **Step 3: `CharacterAppearanceUpdate` v83 (Phase A, cell #18)** — decompile `CUserRemote::OnAvatarModified`; CREATE `appearance_update_test.go` with the v83 row; +v83 marker; regen (🔍 clears with fixture); pin evidence; commit.
- [ ] **Step 4: `EffectQuest` v83 — both ops (Phase A, cells #23, #28)** — decompile `CUser::OnEffect` and identify the `SHOW_FOREIGN_EFFECT` and `SHOW_ITEM_GAIN_INCHAT` mode bodies; CREATE `effect_quest_test.go` with a v83 case per op; +**two** v83 markers (`packet=character/clientbound/EffectQuest` once per op address if they differ, else stack per playbook); regen `EffectQuest`; pin **two** evidence records (op-keyed); commit each op cell separately.
- [ ] **Step 5: `CharacterExpression` v83 (Class E, cell #33)** — name `CUser::OnEmotion` in the v83 IDB (byte-signature + v87/v95 twin match, §B) → harvest+splice into `gms_v83.json` → regen `CharacterExpression` → +v83 marker on `expression_test.go` → pin evidence → commit.
- [ ] **Step 6: `CharacterChairShow` v83 (Class E, cell #35)** — name `CUserRemote::OnSetActivePortableChair` in the v83 IDB → splice → regen `CharacterChairShow` → +v83 marker on `chair_show_test.go` → pin evidence → commit.
- [ ] **Step 7: `CheckName` v83 (Class E, cell #38)** — name `CLogin::SendCheckDuplicateIDPacket` in the v83 IDB → splice → regen `CheckName` → +v83 marker on `check_name_test.go` → pin evidence → confirm routed (`CharacterCheckNameHandle` present in `template_gms_83_1.json`) → commit.
- [ ] **Step 8: After all v83 cells** — `go test -race ./character/...` green; `git diff docs/packets/ida-exports/gms_v83.json` shows ONLY the three spliced fns (OnEmotion, OnSetActivePortableChair, SendCheckDuplicateIDPacket). Each cell already committed per §D.6.

### Task 5: v84 IDB — Phase-A v84 rows + Class-E v84 + Class-B ❌ adjudication (cells #14, #19, #24, #29, #34, #36, #39, #41, #42, #44, #45)

**IDB:** v84 (confirm by name). **Files:** append v84 rows to the three Phase-A test files; `docs/packets/ida-exports/gms_v84.json` (Class-E splices); markers on `clientbound/{expression,chair_show,movement,view_all}_test.go` + `serverbound/{auto_distribute_ap,check_name}_test.go`; reports + evidence under `gms_v84/`.

- [ ] **Step 1: Select the v84 instance** — confirm name. (v84 body ≡ v83 below ~0x3D — expect stale ❌s that clear, no real deltas; still read each client.)
- [ ] **Step 2: Phase-A v84 rows (cells #14, #19, #24, #29)** — for `CharacterList`, `CharacterAppearanceUpdate`, `EffectQuest` (both ops): decompile each v84 read order, APPEND the v84 row to the existing test file (§D.7), stack the v84 marker, regen, adjudicate (v84≡v83 ⇒ clean), pin evidence; commit each cell.
- [ ] **Step 3: `CharacterExpression` v84 + `CharacterChairShow` v84 + `CheckName` v84 (Class E, cells #34, #36, #39)** — name+splice `OnEmotion`, `OnSetActivePortableChair`, and `CLogin::SendCheckDuplicateIDPacket` in the v84 IDB → splice each into `gms_v84.json` → regen (`CharacterExpression`/`CharacterChairShow`/`CheckName`) → +v84 markers on `expression_test.go`/`chair_show_test.go`/`check_name_test.go` → pin evidence → for `CheckName` confirm routed (`CharacterCheckNameHandle` present in `template_gms_84_1.json`) → commit each.
- [ ] **Step 4: `CharacterViewAllCharacters` v84 (cell #41, verdict 🚫)** — decompile `CLogin::OnViewAllCharResult` (the `#suffix` keys are present in the v84 export); adjudicate the 🚫 (§D.4); regen `CharacterViewAllCharacters`; +v84 marker on `view_all_test.go`; pin evidence; commit.
- [ ] **Step 5: `CharacterMovement` v84 (cell #42, verdict ❌)** — decompile `CUserRemote::OnMove`; adjudicate ❌ vs v83 (expect stale); regen `CharacterMovement`; +v84 marker on `movement_test.go`; pin evidence; commit.
- [ ] **Step 6: `AutoDistributeAp` v84 — both ops (cells #44, #45, verdict ❌)** — decompile `CWvsContext::SendAbilityUpRequest#{DistributeAp,AutoDistributeAp}`; adjudicate each ❌; regen `AutoDistributeAp` (the report `name`); +v84 markers (one per op) on `auto_distribute_ap_test.go`; pin **two** evidence records (op-keyed `#DistributeAp` / `#AutoDistributeAp`); confirm routed (`CharacterAutoDistributeApHandle` present in `template_gms_84_1.json`); commit each op cell.
- [ ] **Step 7: After all v84 cells** — `go test -race ./character/...` green; `git diff gms_v84.json` shows only the intended splices. Pull in any Task 1–3 v84 escalations here.

### Task 6: v87 IDB — Phase-A v87 rows only (cells #15, #20, #25, #30)

**IDB:** v87 (confirm by name). **Files:** append v87 rows to the three Phase-A test files; reports + evidence under `gms_v87/`. (Everything else on v87 is already ✅ — these four are the only in-scope v87 cells.)

- [ ] **Step 1: Select the v87 instance** — confirm name.
- [ ] **Step 2: Append v87 rows** for `CharacterList` (#15), `CharacterAppearanceUpdate` (#20), `EffectQuest` both ops (#25, #30): decompile each v87 read order, APPEND the v87 row (§D.7), stack v87 marker, regen (`<V>=gms_87_1`), adjudicate any ❌, pin evidence; commit each cell.
- [ ] **Step 3:** `go test -race ./character/...` green; matrix regen+check confirms the four v87 cells → `verified`.

### Task 7: v95 IDB — Phase-A v95 rows only (cells #16, #21, #26, #31)

**IDB:** v95 (confirm by name). **Files:** append v95 rows to the three Phase-A test files; reports + evidence under `gms_v95/`.

- [ ] **Step 1: Select the v95 instance** — confirm name.
- [ ] **Step 2: Append v95 rows** for `CharacterList` (#16), `CharacterAppearanceUpdate` (#21), `EffectQuest` both ops (#26, #31): decompile, APPEND v95 row (§D.7), stack v95 marker, regen (`<V>=gms_95_1`), adjudicate, pin evidence; commit each cell.
- [ ] **Step 3:** `go test -race ./character/...` green; matrix regen+check confirms the four v95 cells → `verified`.

### Task 8: jms IDB — Phase-A jms rows + Class-E jms chair + jms splices + jms ❌ adjudication (cells #17, #22, #27, #32, #37, #40, #43, #46, #47)

**IDB:** jms `*_U_DEVM` (confirm by name; NOT the SMC retail dump). **Files:** append jms rows to the three Phase-A files; `docs/packets/ida-exports/gms_jms_185.json` (splices: chair, ViewAll ×3 suffix, HealOverTime); markers on `clientbound/{chair_show,view_all,movement}_test.go` + `serverbound/heal_over_time_test.go`; reports + evidence under `jms_v185/`. **jms is the prime real-wire-delta suspect (§D.4) — read each client carefully, do not assume GMS shape.**

- [ ] **Step 1: Select the jms `*_U_DEVM` instance** — `list_instances` → confirm the `_U_DEVM` binary name → `select_instance(<jms port>)`.
- [ ] **Step 2: Phase-A jms rows (cells #17, #22, #27, #32)** — `CharacterList` jms (base fn present; no committed jms report → decompile + report-gen), `CharacterAppearanceUpdate` jms (report exists 🔍), `EffectQuest` jms both ops (report exists 🔍): decompile each jms read order, APPEND the jms row to each Phase-A file (§D.7 — a divergent jms body carries its own `want`), stack jms marker, regen (`<V>=jms_185_1`), adjudicate (real delta → fix-first §D.4), pin evidence; commit each cell.
- [ ] **Step 3: `CharacterChairShow` jms (Class E, cell #37)** — name `CUserRemote::OnSetActivePortableChair` in the jms IDB (ABSENT, §A.4) → splice into `gms_jms_185.json` → regen `CharacterChairShow` → +jms marker on `chair_show_test.go` → pin evidence → commit. (If genuinely absent in the IDB after trying → IDB-justified `n-a` with the justification in `audit.md`; but v87/v95 ✅ make present-but-unnamed the expected outcome.)
- [ ] **Step 4: `CharacterViewAllCharacters` jms (splice, cell #40)** — harvest+splice the three `CLogin::OnViewAllCharResult#CharacterViewAll{Characters,Count,SearchFailed}` keys from the jms IDB (roster base `- CLogin::OnViewAllCharResult`; descent emits the suffixes) → regen `CharacterViewAllCharacters` → +jms marker on `view_all_test.go` → pin evidence → commit.
- [ ] **Step 5: `CharacterMovement` jms (cell #43, ❌)** — decompile `CUserRemote::OnMove` in the jms IDB; adjudicate ❌ — **most plausible real delta** (§D.4): if jms move structure differs, fix-first `libs/atlas-packet/character/clientbound/movement.go` (version branch + cross-version test update, own commit), then regen; +jms marker; pin evidence; commit.
- [ ] **Step 6: `HealOverTime` jms (Class E, cell #46)** — name `CWvsContext::SendStatChangeRequest` in the jms IDB (ABSENT; only `...ByItemOption` in the export — confirm the right sender by the `COutPacket` opcode, not the symbol, playbook §10) → splice → regen `HealOverTime` → +jms marker on `heal_over_time_test.go` → pin evidence → **confirm routed**: if `--check` shows `routedElsewhere && !routed`, wire `CharacterHealOverTimeHandle` into `template_jms_185_1.json` mirroring the v95 entry (§A.6) → commit.
- [ ] **Step 7: `KeyMapChange` jms (cell #47, FlatInvalid TRUNCATION)** — decompile `CFuncKeyMappedMan::SaveFuncKeyMap`; confirm the verdict-2 rows are the documented loop-89-vs-90 export truncation artifact (`evidence …jms_v185/character.serverbound.KeyMapChange.yaml` `category: TRUNCATION`) — if so it is the VERIFIED-EXCEPTION: refresh the report + evidence and promote; if it is instead a real width delta, fix-first (§D.4). Keep the existing jms marker; confirm routed (present); commit.
- [ ] **Step 8: After all jms cells** — `go test -race ./character/...` green; `git diff gms_jms_185.json` shows ONLY the intended splices (chair, 3× ViewAll suffix, HealOverTime). Pull in any Task 1–3 jms escalations here.

---

## Task 9: Final verification gate (acceptance)

**Files:** none new (final regen + gate).

- [ ] **Step 1: Full matrix regen** — `go run ./tools/packet-audit matrix`.
- [ ] **Step 2: Confirm zero incomplete character cells**
```bash
python3 - <<'EOF'
import json
d=json.load(open("docs/packets/audits/status.json"))
rows = d["rows"] if isinstance(d,dict) and "rows" in d else d
bad=[(r["packet"],r.get("op"),v) for r in rows if r["packet"].startswith("character/")
     for v,c in r.get("cells",{}).items() if c.get("state") not in ("verified","n-a")]
print("remaining incomplete character cells:", bad)
assert not bad, bad
print("OK — all character cells verified or n-a")
EOF
```
Expected: `OK`.
- [ ] **Step 2b: Sanity-check any `n-a`** — if any chair/expression/heal cell ended `n-a`, confirm the status note / `audit.md` carries the IDB-confirmed justification (not a bare downgrade). The v87/v95 ✅ siblings make `n-a` the unexpected outcome — flag it.
- [ ] **Step 3: `matrix --check` no-new-problems gate (§E)**
```bash
go run ./tools/packet-audit matrix --check 2>&1 | tee /tmp/matrix-check-final.txt
grep -i "character/" /tmp/matrix-check-final.txt    # expect: no orphan/dangling/stale/drift lines for character
grep -c "conflict" /tmp/matrix-check-final.txt       # expect: <= baseline from Task 0 Step 2
```
- [ ] **Step 4: `fname-doc` and `operations` checks introduce no new failures**
```bash
go run ./tools/packet-audit fname-doc --check 2>&1 | tail -5
go run ./tools/packet-audit operations --check 2>&1 | tail -5
```
- [ ] **Step 5: Go module gates (`libs/atlas-packet` — the only module touched)**
```bash
( cd libs/atlas-packet && go test -race ./... && go vet ./... && go build ./... )
tools/redis-key-guard.sh   # from repo root; character codecs touch no Redis → clean
```
Expected: all clean. **No `go.mod` was touched → no `docker buildx bake` required** (CLAUDE.md gate is conditional on a `go.mod` change; design §10).
- [ ] **Step 6: If any real wire delta was found (§D.4)** — confirm `docs/tasks/task-109-character-packet-fixtures/audit.md` records it and the PR description will surface it (jms Movement / jms List are the suspects). Otherwise note "no wire deltas; verification-only."
- [ ] **Step 7: Final commit** of `STATUS.md`/`status.json` if not already coupled into a cell commit, then run `superpowers:requesting-code-review` BEFORE any PR (CLAUDE.md "Code Review Before PR").

---

## §E. `matrix --check` exit-code bar (design §8 / playbook §8)

`matrix --check` exits 1 from a pre-existing 🟥 registry-seed conflict backlog unrelated to character. The acceptance bar is **"no new problems," not a clean exit 0**:
- Zero orphan / dangling / stale / drift lines mentioning any `character/*` packet.
- The global conflict count must **not increase** above the Task 0 Step 2 baseline.
- Every character cell in scope reads ✅ (or IDB-justified `n-a`) after regen.
- `fname-doc --check` and `operations --check` introduce no new failures.

A net decrease (clearing character-specific lines) is a bonus, not required.

---

## §F. Self-review (writing-plans checklist)

- **Spec coverage:** All 47 design/PRD cells are enumerated in §C (Stage-1 #1–#12; Stage-2 #13–#47) and assigned to Tasks 1–8; PRD acceptance criteria map to Task 9 (every row ✅/n-a; each duplicated-path row per distinct fname — EffectQuest ×2 ops #23–#32, AutoDistributeAp ×2 ops #44/#45, KeyMapChange op-row #9–#11 + sub-struct #12; coupled artifacts per cell §D.6; `matrix`/`fname-doc`/`operations` checks; module gates). PRD Open Questions: CharacterList latent? → emitted, Phase-A fixture gap (§A.5 / design §1.1). Duplicated rows? → op-keyed §C. Service owner? → consumer split; codecs in `libs/atlas-packet/character/` (§A intro). Genuine n-a? → only Class-E if an IDB confirms absence (Task 4/5/8), else name+splice.
- **Placeholder scan:** no TBD/TODO; every command block is concrete with substitution rules defined in §B; markers/evidence/splice commands are exact. The only deliberately deferred specifics are per-version byte values for the Phase-A fixtures (#13–#32) and adjudication outcomes (#41–#47) — these are *derived at execution from the live decompile per CLAUDE.md "Verification Over Memory"; inventing them in the plan would violate the no-invention rule*. §D.7 fixes the authoring procedure so this is a method, not a placeholder.
- **Type/name consistency:** report/marker names are the descriptive `name` everywhere (§A.1 — `EffectQuest`/`BuffGive`/`CharacterList`/`CharacterChairShow`/`CharacterAppearanceUpdate`, never `Character`-prefixed for the pkg-empty cells); marker `packet=` keeps `character/<dir>/`; version keys, template infixes, and export stems used consistently per the §B table; new test files named `list_test.go`/`appearance_update_test.go`/`effect_quest_test.go` (matching the existing codec filenames).
- **Design-deviation log:** §A documents every override of design.md with source evidence (filename convention, pre-existing markers, stale notes, three jms export gaps the design missed, partial pre-existing Phase-A reports, routing-via-matrix-check). The executor trusts §A/§C over design.md where they differ.
