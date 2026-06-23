# CWvsContext::OnMessage Dispatcher Family Migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate `CWvsContext::OnMessage` to the canonical discrete-per-mode audit shape, correct the v83 (and confirm v84) per-version mode table, drive every supported arm to verified across all five versions, resolve the jms ❌, and keep the family `dispatcher-lint`-clean with no baseline entry.

**Architecture:** The codec + config-driven body layer is already built and footgun-free (24 discrete `StatusMessage*` structs + 23 fixed-key body funcs). This task is the dispatcher-audit half: (a) fix the v83 mode-table correctness bug, (b) enroll the family with 24 `#`-entries so each arm is individually byte-verified, (c) decompose the GMS/jms exports and resolve the jms ❌. The exemplars are `CField::OnFieldEffect` and `CITC::OnNormalItemResult` (clean, non-baselined; **no bare-root case**).

**Tech Stack:** Go (`libs/atlas-packet`, `tools/packet-audit`, `services/atlas-channel`, `services/atlas-configurations`), ida-pro-mcp (decompile), packet-audit CLI (`dispatcher-lint`/`matrix`/`operations`/`fname-doc`), Docker buildx bake.

**Read `context.md` first** — it holds the grounded per-version mode table, the 24-arm map, the
D5 correction (`ResolveCode` returns 99, not no-op), the exemplar pointers, and the stop-and-ask gates.

---

## File Structure

Files created or modified, by responsibility:

- `docs/packets/dispatchers/character_status_message.yaml` — **modify**: correct the per-version
  mode table (v83 SP-absent + FAME..SKILL_EXPIRE shifted down one for v83). Source of truth.
- `tools/packet-audit/cmd/run.go` — **modify**: delete the bare `case "CWvsContext::OnMessage":`
  single-representative block; add 24 `case "CWvsContext::OnMessage#<Arm>":` entries.
- `docs/packets/ida-exports/{gms_v83,gms_v84,gms_v87,gms_v95,jms_v185}.json` — **modify**: splice
  the per-mode delegate structure (+ inner fan-out sub-functions) with real addresses.
- `libs/atlas-packet/character/clientbound/status_message_test.go` — **modify**: per-arm
  `// packet-audit:verify` markers (all 5 versions; v83 omits SP) with real IDA addresses.
- `libs/atlas-packet/character/clientbound/status_message.go` — **modify (citations only)**: add
  per-version decompile-citation comments where missing; fix a codec only if a fixture surfaces a
  read-order mismatch (none expected).
- `services/atlas-configurations/seed-data/templates/template_{gms_83_1,gms_84_1,gms_87_1,gms_95_1,jms_185_1}.json`
  — **modify (generated)**: `CharacterStatusMessage` `operations` maps regenerated from the yaml.
- `docs/packets/evidence/{version}/character.clientbound.StatusMessage*.yaml` — **create**: per-arm
  evidence records with real `decompile_sha256`.
- `docs/packets/registry/{version}.yaml` — **verify**: `SHOW_STATUS_INFO` row present per version
  (no change expected; jms confirm).
- `docs/packets/audits/STATUS.md` + `status.json` — **regenerate** via `packet-audit matrix`.
- `docs/tasks/task-104-message-dispatcher-family/runbook-live-config.md` — **create**: post-deploy
  live-config patch runbook.

`docs/packets/dispatcher-lint-baseline.yaml` — **must NOT gain an OnMessage entry** (it is not
there now and must stay absent).

---

## Task 0: Pre-flight — verify worktree, capture baseline gate state

**Files:** none (read-only).

- [ ] **Step 1: Confirm worktree and branch**

Run (cd into the task-104 worktree — `.worktrees/task-104-message-dispatcher-family` under the
repo root; resolve via `git worktree list` if needed):
```bash
git rev-parse --show-toplevel   # must end with /.worktrees/task-104-message-dispatcher-family
git branch --show-current        # must be task-104-message-dispatcher-family
```
Expected: paths/branch match. If not, STOP — you are in the wrong tree. All subsequent commands run
from this worktree root.

- [ ] **Step 2: Snapshot the four packet-audit gates (expect some to be RED now)**

Run from the worktree root:
```bash
go run ./tools/packet-audit dispatcher-lint; echo "lint=$?"
go run ./tools/packet-audit matrix --check; echo "matrix=$?"
go run ./tools/packet-audit fname-doc --check; echo "fname=$?"
go run ./tools/packet-audit operations --check; echo "ops=$?"
```
Expected now: `dispatcher-lint` should already be 0 (OnMessage is a single-rep, invisible to the
linter). `matrix`/`operations` may be 0 against the current (wrong) yaml — note exit codes so you
can detect regressions. Record the four numbers in the task notes.

- [ ] **Step 3: Snapshot the build/test gate**

Run:
```bash
( cd libs/atlas-packet && go build ./... && go test ./... ) ; echo "pkt=$?"
( cd tools/packet-audit && go build ./... ) ; echo "audit=$?"
```
Expected: PASS (exit 0). If RED, capture output — you must not introduce new failures.

- [ ] **Step 4: Confirm no consumer emits SP (the D5 safeguard)**

Run:
```bash
grep -rn "IncreaseSkillPointBody\|NewStatusMessageIncreaseSkillPoint" services/ libs/ \
  | grep -v "status_message_body.go\|status_message.go\|status_message_test.go"
```
Expected: **no output** (zero call sites). If this prints a call site, STOP and re-evaluate Task 9
— a v83 emitter of SP is a live-crash path that must be gated.

---

## Task 1: IDA enumeration & grounding

**Files:** none yet (produces facts recorded into struct/test comments in later tasks; capture an
enumeration scratch note in the task notes / `context.md` addendum).

Goal: prove the per-version outer switch and the two inner fan-out handlers from IDA, confirm v84
per-arm order directly, and resolve jms `sub_B0931C`. **Cite every address.**

- [ ] **Step 1: Confirm each IDB version before reading**

For each (version, port): `v83=13342, v84=13337, v87=13341, v95=13340, jms=13339`:
```
select_instance(<port>)
```
Then verify it is the expected build (e.g. `func_query` for a version-distinctive symbol) before
trusting any address. Do NOT read an IDB without confirming its version (CLAUDE.md RE rule).

- [ ] **Step 2: Decompile the five `OnMessage` switches**

Using `decompile` at the switch addresses (context.md table): v83 `0xA209D4`, v84 `0xA6BDD9`,
v87 `0xAB8076`, v95 `0xA06C90`, jms `0xB078F3`. Record, per version: the case count, the
delegate sub-handler called per case, and its address. Confirm:
- v83 = 14 cases (0–0xD), **no** `OnIncSPMessage`.
- v84/v87/v95 = 15 cases (0–0xE), SP at case 4.
- jms = 16 cases (0–0xF), case 0xF = `sub_B0931C`.

- [ ] **Step 3: Decompile the two inner fan-out handlers per version**

`OnDropPickUpMessage` (mode 0) and `OnQuestRecordMessage` (mode 1). Record the inner discriminator
read order and the constant→arm mapping per version, and confirm they match the structs' baked
inner bytes (context.md arm table: drop `-2/-1/-3/0/1/2`, quest `0/1/2`). Expected stable across
versions; flag any drift.

- [ ] **Step 4: Confirm v84 per-arm semantic order directly (D8)**

v84 delegates are `sub_*` (unnamed). Decompile each v84 case's `sub_` and confirm the read order
matches the v95-named arm at the same case index — especially case 4 reads SP's
`short jobId + byte amount`. Do NOT fold from v83 or v95 by assumption. Record each v84 delegate
address.

- [ ] **Step 5: Resolve jms `sub_B0931C` (mode 0xF) — STOP-AND-ASK GATE (D7)**

Decompile `sub_B0931C` (`0xB0931C`). Determine whether its read order maps to an existing Atlas
arm:
- If it matches an existing `StatusMessage*` arm → record the mapping; it becomes that arm's jms
  `#`-entry.
- If it is a jms-only arm with **no** Atlas equivalent → **STOP. Report to the user** with the
  decompiled read order and ask how to disposition (it will be tracked ⬜/escalated, NOT invented).
  jms ✅ for the 15 shared arms does not depend on resolving 0xF.

- [ ] **Step 6: Record the jms per-delegate addresses**

Capture every jms delegate address (`OnDropPickUpMessage 0xB07A01` … `OnSkillExpireMessage
0xB088A4` confirmed this session; fill the 13 in between + any inner fan-out subs). These feed the
export splice (Task 4) and the verify markers (Task 5).

- [ ] **Step 7: Commit the enumeration note**

Append the grounded per-version delegate-address table to
`docs/tasks/task-104-message-dispatcher-family/context.md` (a new "## Enumeration results" section).
```bash
git add docs/tasks/task-104-message-dispatcher-family/context.md
git commit -m "task-104: IDA enumeration of OnMessage switch + inner fan-out (5 versions)"
```

---

## Task 2: Correct the per-version mode table in the dispatcher yaml

**Files:**
- Modify: `docs/packets/dispatchers/character_status_message.yaml`

- [ ] **Step 1: Replace the false "version-stable" banner and the mode table**

Rewrite the file to the IDA-grounded per-version table. Express v83's SP-absence by **omitting
`gms_v83`** from the `INCREASE_SKILL_POINT` key's `modes` map, and set v83 FAME..SKILL_EXPIRE one
lower than v84+.

```yaml
# CharacterStatusMessage — CWvsContext::OnMessage per-version mode table.
#
# SOURCE OF TRUTH for the CharacterStatusMessage writer.
#
# NOT version-stable: v84 inserts OnIncSPMessage at case 4, shifting INCREASE_FAME
# through SKILL_EXPIRE up by one from v84 onward. v83 has 14 cases (0-0xD) with NO
# SP arm (IDA CWvsContext::OnMessage @0xA209D4, 14 cases). v95 PDB names are
# authoritative: case 4=OnIncSPMessage, 5=OnIncPOPMessage(fame), 6=OnIncMoneyMessage(meso),
# 7=OnIncGPMessage, 8=OnGiveBuffMessage, ... 14=OnSkillExpireMessage.
# Per-version switch addrs: gms_v83 0xA209D4 (14 cases) · gms_v84 0xA6BDD9 (15) ·
# gms_v87 0xAB8076 (15) · gms_v95 0xA06C90 (15) · jms_v185 0xB078F3 (16; case 0xF
# = sub_B0931C, no Atlas arm — see task-104 D7).
#
# INCREASE_SKILL_POINT has no gms_v83 entry: v83 genuinely lacks the arm (absence,
# not a fabricated byte). v84 SP-at-4 IDA-confirmed per-arm (task-104 D8).

writer: CharacterStatusMessage
fname: CWvsContext::OnMessage
op: CHARACTER_STATUS_MESSAGE
direction: clientbound
operations:
  - { key: DROP_PICK_UP,         modes: { gms_v83: 0,  gms_v84: 0,  gms_v87: 0,  gms_v95: 0,  jms_v185: 0 } }
  - { key: QUEST_RECORD,         modes: { gms_v83: 1,  gms_v84: 1,  gms_v87: 1,  gms_v95: 1,  jms_v185: 1 } }
  - { key: CASH_ITEM_EXPIRE,     modes: { gms_v83: 2,  gms_v84: 2,  gms_v87: 2,  gms_v95: 2,  jms_v185: 2 } }
  - { key: INCREASE_EXPERIENCE,  modes: { gms_v83: 3,  gms_v84: 3,  gms_v87: 3,  gms_v95: 3,  jms_v185: 3 } }
  - { key: INCREASE_SKILL_POINT, modes: {              gms_v84: 4,  gms_v87: 4,  gms_v95: 4,  jms_v185: 4 } }
  - { key: INCREASE_FAME,        modes: { gms_v83: 4,  gms_v84: 5,  gms_v87: 5,  gms_v95: 5,  jms_v185: 5 } }
  - { key: INCREASE_MESO,        modes: { gms_v83: 5,  gms_v84: 6,  gms_v87: 6,  gms_v95: 6,  jms_v185: 6 } }
  - { key: INCREASE_GUILD_POINT, modes: { gms_v83: 6,  gms_v84: 7,  gms_v87: 7,  gms_v95: 7,  jms_v185: 7 } }
  - { key: GIVE_BUFF,            modes: { gms_v83: 7,  gms_v84: 8,  gms_v87: 8,  gms_v95: 8,  jms_v185: 8 } }
  - { key: GENERAL_ITEM_EXPIRE,  modes: { gms_v83: 8,  gms_v84: 9,  gms_v87: 9,  gms_v95: 9,  jms_v185: 9 } }
  - { key: SYSTEM_MESSAGE,       modes: { gms_v83: 9,  gms_v84: 10, gms_v87: 10, gms_v95: 10, jms_v185: 10 } }
  - { key: QUEST_RECORD_EX,      modes: { gms_v83: 10, gms_v84: 11, gms_v87: 11, gms_v95: 11, jms_v185: 11 } }
  - { key: ITEM_PROTECT_EXPIRE,  modes: { gms_v83: 11, gms_v84: 12, gms_v87: 12, gms_v95: 12, jms_v185: 12 } }
  - { key: ITEM_EXPIRE_REPLACE,  modes: { gms_v83: 12, gms_v84: 13, gms_v87: 13, gms_v95: 13, jms_v185: 13 } }
  - { key: SKILL_EXPIRE,         modes: { gms_v83: 13, gms_v84: 14, gms_v87: 14, gms_v95: 14, jms_v185: 14 } }
```

> If Task 1 proved any v84 value differs from SP-at-4, use the proven value instead and note it.

- [ ] **Step 2: Sanity-check the YAML parses**

Run:
```bash
go run ./tools/packet-audit operations --check 2>&1 | head -40
```
Expected: it will now report **drift** (the templates still carry the old v83 table) — that is
correct; Task 6 regenerates the templates. Confirm the failure is about template drift, not a YAML
parse error.

- [ ] **Step 3: Commit**

```bash
git add docs/packets/dispatchers/character_status_message.yaml
git commit -m "task-104: correct OnMessage per-version mode table (v83 SP-absent, fame+ shifted)"
```

---

## Task 3: Rewire run.go — 24 `#`-entries, delete the single representative

**Files:**
- Modify: `tools/packet-audit/cmd/run.go:392-398` (the `case "CWvsContext::OnMessage":` block)

- [ ] **Step 1: Delete the bare-root single-representative case**

Remove the entire block at `run.go:392-398`:
```go
	case "CWvsContext::OnMessage":
		// Struct family is StatusMessage*; writer = "CharacterStatusMessage".
		// CWvsContext::OnPacket case 38 (0x26) delegates here; dispatches on
		// a leading mode byte (0-14) to 15 sub-handlers.  The pipeline can only
		// model the outermost Decode1 (mode byte); sub-op enum drift is deferred
		// to _pending.md "## Sub-op enum drift — character domain".
		return []candidate{{name: "StatusMessageDropPickUpInventoryFull", dir: csvpkg.DirClientbound}}
```
There must be **no** bare `case "CWvsContext::OnMessage":` after this — mirror `CField::OnFieldEffect`
(`run.go` ~1823) and `CITC::OnNormalItemResult` (~1886), which have only `#`-entries and no root.

- [ ] **Step 2: Add the 24 `#`-entries**

Insert (where the deleted block was) one entry per arm. All structs live in
`character/clientbound`, so `dir: csvpkg.DirClientbound` and no `pkg` hint (they resolve in the
default `character` walk; if `locateAtlasFile` reports ambiguity, add `pkg: "character"`).

```go
	// --- Character: CWvsContext::OnMessage (CHARACTER_STATUS_MESSAGE / SHOW_STATUS_INFO) ---
	// Mode-prefix dispatcher: Decode1(outer mode) → per-mode sub-handler. modes 0 (DropPickUp)
	// and 1 (QuestRecord) fan out on an inner discriminator (structural constant, not config-
	// resolved — task-104 D1). Outer mode is config-resolved from the CharacterStatusMessage
	// operations table (docs/packets/dispatchers/character_status_message.yaml). v83 has 14 arms
	// (no SP); v84+ has 15; jms has 16 (case 0xF sub_B0931C, no Atlas arm — D7). No bare-root
	// representative (FIELD_EFFECT/MTS model). Per-version verdicts via the test markers.
	case "CWvsContext::OnMessage#DropPickUpItemUnavailable":
		// mode 0, inner -2 (OnDropPickUpMessage "item unavailable", StringPool 2983).
		return []candidate{{name: "StatusMessageDropPickUpItemUnavailable", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#DropPickUpInventoryFull":
		// mode 0, inner -1 (default branch → "cannot pick up any more", StringPool 295).
		return []candidate{{name: "StatusMessageDropPickUpInventoryFull", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#DropPickUpGameFileDamaged":
		// mode 0, inner -3 (StringPool 5317 + chat 5311).
		return []candidate{{name: "StatusMessageDropPickUpGameFileDamaged", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#DropPickUpStackableItem":
		// mode 0, inner 0: Decode1(mode)+Decode1(0)+Decode4(itemId)+Decode4(amount).
		return []candidate{{name: "StatusMessageDropPickUpStackableItem", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#DropPickUpUnStackableItem":
		// mode 0, inner 2: Decode1(mode)+Decode1(2)+Decode4(itemId).
		return []candidate{{name: "StatusMessageDropPickUpUnStackableItem", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#DropLossStackableItem":
		// mode 0, inner 0, negative qty: Decode1(mode)+Decode1(0)+Decode4(itemId)+Decode4(-amount).
		return []candidate{{name: "StatusMessageDropLossStackableItem", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#DropLossUnStackableItem":
		// mode 0, inner 2: Decode1(mode)+Decode1(2)+Decode4(itemId).
		return []candidate{{name: "StatusMessageDropLossUnStackableItem", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#DropPickUpMeso":
		// mode 0, inner 1: Decode1(mode)+Decode1(1)+Decode1(partial)+Decode4(amount)+Decode2(bonus).
		return []candidate{{name: "StatusMessageDropPickUpMeso", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#ForfeitQuestRecord":
		// mode 1, inner 0: Decode1(mode)+Decode2(questId)+Decode1(0).
		return []candidate{{name: "StatusMessageForfeitQuestRecord", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#UpdateQuestRecord":
		// mode 1, inner 1: Decode1(mode)+Decode2(questId)+Decode1(1)+DecodeStr(info).
		return []candidate{{name: "StatusMessageUpdateQuestRecord", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#CompleteQuestRecord":
		// mode 1, inner 2: Decode1(mode)+Decode2(questId)+Decode1(2)+Decode8(FILETIME).
		return []candidate{{name: "StatusMessageCompleteQuestRecord", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#CashItemExpire":
		// mode 2: Decode1(mode)+Decode4(itemId).
		return []candidate{{name: "StatusMessageCashItemExpire", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#IncreaseExperience":
		// mode 3: OnIncEXPMessage. GMS>=95 trailing partyEXPRingEXP+cakePieEventBonus (gated in struct).
		return []candidate{{name: "StatusMessageIncreaseExperience", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#IncreaseSkillPoint":
		// mode 4 (v84+ ONLY; v83 absent → ⬜). Decode1(mode)+Decode2(jobId)+Decode1(amount).
		return []candidate{{name: "StatusMessageIncreaseSkillPoint", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#IncreaseFame":
		// OnIncPOPMessage. mode 4 (v83) / 5 (v84+): Decode1(mode)+Decode4(amount).
		return []candidate{{name: "StatusMessageIncreaseFame", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#IncreaseMeso":
		// OnIncMoneyMessage. mode 5 (v83) / 6 (v84+): Decode1(mode)+Decode4(amount).
		return []candidate{{name: "StatusMessageIncreaseMeso", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#IncreaseGuildPoint":
		// OnIncGPMessage. mode 6 (v83) / 7 (v84+): Decode1(mode)+Decode4(amount).
		return []candidate{{name: "StatusMessageIncreaseGuildPoint", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#GiveBuff":
		// OnGiveBuffMessage. mode 7 (v83) / 8 (v84+): Decode1(mode)+Decode4(itemId).
		return []candidate{{name: "StatusMessageGiveBuff", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#GeneralItemExpire":
		// mode 8 (v83) / 9 (v84+): Decode1(mode)+Decode1(count)+count*Decode4(itemId).
		return []candidate{{name: "StatusMessageGeneralItemExpire", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#SystemMessage":
		// mode 9 (v83) / 10 (v84+): Decode1(mode)+DecodeStr(message).
		return []candidate{{name: "StatusMessageSystemMessage", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#QuestRecordEx":
		// mode 10 (v83) / 11 (v84+): Decode1(mode)+Decode2(questId)+DecodeStr(info).
		return []candidate{{name: "StatusMessageQuestRecordEx", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#ItemProtectExpire":
		// mode 11 (v83) / 12 (v84+): Decode1(mode)+Decode1(count)+count*Decode4(itemId).
		return []candidate{{name: "StatusMessageItemProtectExpire", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#ItemExpireReplace":
		// mode 12 (v83) / 13 (v84+): Decode1(mode)+Decode1(count)+count*DecodeStr(message).
		return []candidate{{name: "StatusMessageItemExpireReplace", dir: csvpkg.DirClientbound}}
	case "CWvsContext::OnMessage#SkillExpire":
		// mode 13 (v83) / 14 (v84+): Decode1(mode)+Decode1(count)+count*Decode4(skillId).
		return []candidate{{name: "StatusMessageSkillExpire", dir: csvpkg.DirClientbound}}
```

- [ ] **Step 3: Build the tool**

Run:
```bash
( cd tools/packet-audit && go build ./... ) ; echo "build=$?"
```
Expected: exit 0.

- [ ] **Step 4: Commit**

```bash
git add tools/packet-audit/cmd/run.go
git commit -m "task-104: enroll OnMessage as a 24-arm dispatcher family in run.go (no bare root)"
```

---

## Task 4: Export decomposition — GMS delegate structure + jms 16-delegate splice

**Files:**
- Modify: `docs/packets/ida-exports/{gms_v83,gms_v84,gms_v87,gms_v95,jms_v185}.json`

This is a **surgical JSON splice** (never overwrite the file). Use the per-mode delegate shape from
the guild/field exports: a base dispatcher entry (`address`, `direction`, `note`) plus one
`<Fname>#<Arm>` entry per arm with `address` + `calls[]` (each `{op, comment}`), and per-version
`dispatch[]` guards where an arm is version-gated (e.g. SP absent on v83).

- [ ] **Step 1: Splice the GMS v83 export**

Add the base `CWvsContext::OnMessage` entry (addr `0xA209D4`, note: 14 cases, no SP) and the 23
v83 `#`-arm entries (all arms EXCEPT `#IncreaseSkillPoint`), each at its delegate address from
Task 1 with `calls[]` matching the struct's `Encode`. Do not add an `#IncreaseSkillPoint` entry to
the v83 export (it is absent). No `address: "0x0"` placeholders.

- [ ] **Step 2: Splice GMS v84/v87/v95 exports**

For each: base entry at its switch address; all 24 `#`-arm entries (including
`#IncreaseSkillPoint`) at their delegate addresses. v95 delegate names are authoritative; v84
delegates are `sub_*` (use the addresses confirmed in Task 1 Step 4). GMS≥95 `#IncreaseExperience`
`calls[]` includes the trailing `partyEXPRingEXP`/`cakePieEventBonus` Decode4s.

- [ ] **Step 3: Splice the jms export (FR-6 core)**

Add the base entry (`0xB078F3`, 16 cases) and all 15 shared `#`-arm entries at their jms delegate
addresses (Task 1 Step 6). For mode 0xF (`sub_B0931C`): handle per the Task 1 Step 5 resolution —
if it mapped to an existing arm, add that `#`-entry; if escalated, add NO invented entry and record
the ⬜/escalated disposition in the task notes (do not fabricate).

- [ ] **Step 4: Validate exports parse and fname-doc is consistent**

Run:
```bash
for v in gms_v83 gms_v84 gms_v87 gms_v95 jms_v185; do
  python3 -c "import json;json.load(open('docs/packets/ida-exports/$v.json'));print('$v ok')"
done
go run ./tools/packet-audit fname-doc --check; echo "fname=$?"
```
Expected: all `ok`; `fname-doc --check` exit 0. If `fname-doc` flags an undocumented fname, resolve
it (do not suppress).

- [ ] **Step 5: Commit**

```bash
git add docs/packets/ida-exports/
git commit -m "task-104: decompose OnMessage exports (GMS per-mode delegates + jms 16-delegate splice)"
```

---

## Task 5: Per-arm fixtures + `// packet-audit:verify` markers

**Files:**
- Modify: `libs/atlas-packet/character/clientbound/status_message_test.go`

The 24 round-trip tests already exist. The work is attaching a `// packet-audit:verify` marker per
arm per **applicable** version with the **real delegate IDA address** from Task 1. Marker form
(see the existing block at `status_message_test.go:10-13`):
```
// packet-audit:verify packet=character/clientbound/<StructName> version=<version> ida=0x<addr>
```

- [ ] **Step 1: Replace the legacy 4-marker block**

The current markers (lines 10-13) all point at `StatusMessageDropPickUpInventoryFull` with the
switch address. Replace them with the per-arm delegate-addressed markers for the InventoryFull arm
(v83/v84/v87/v95/jms) and proceed to add the rest.

- [ ] **Step 2: Add markers for the 23 always-present arms (5 versions each)**

For every arm EXCEPT `StatusMessageIncreaseSkillPoint`, add five marker lines (gms_v83, gms_v84,
gms_v87, gms_v95, jms_v185) above its test function, each citing that version's delegate address.
Example for the meso arm:
```go
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseMeso version=gms_v83 ida=0x<v83 OnIncMoneyMessage addr>
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseMeso version=gms_v84 ida=0x<v84 addr>
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseMeso version=gms_v87 ida=0x<v87 addr>
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseMeso version=gms_v95 ida=0x<v95 addr>
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseMeso version=jms_v185 ida=0x<jms addr>
func TestStatusMessageIncreaseMeso(t *testing.T) { ... }
```

- [ ] **Step 3: Add markers for `StatusMessageIncreaseSkillPoint` — v84/v87/v95/jms ONLY**

SP is absent on v83: add **four** markers (no `gms_v83` line). v83 stays ⬜ for this arm.
```go
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseSkillPoint version=gms_v84 ida=0x<v84 addr>
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseSkillPoint version=gms_v87 ida=0x<v87 addr>
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseSkillPoint version=gms_v95 ida=0x<v95 addr>
// packet-audit:verify packet=character/clientbound/StatusMessageIncreaseSkillPoint version=jms_v185 ida=0x<jms addr>
```

- [ ] **Step 4: jms 0xF disposition**

If Task 1 Step 5 mapped `sub_B0931C` to an existing arm, ensure that arm carries its jms marker.
If escalated, add no jms marker for any invented arm; the jms 0xF cell is ⬜/escalated.

- [ ] **Step 5: Run the round-trip tests**

Run:
```bash
( cd libs/atlas-packet && go test ./character/... ) ; echo "test=$?"
```
Expected: PASS. RoundTrip asserts zero leftover bytes after decode for every variant. If a test
fails, the codec read order is wrong for that version — fix `status_message.go` to match the
decompiled order (this is the only place a real codec change is expected, and it would mean a prior
codec bug, not a fixture bug).

- [ ] **Step 6: Add per-version decompile citations to the structs (FR-1.1)**

In `status_message.go`, where a struct comment lacks per-version function+address citations, add
them (e.g. `// CWvsContext::OnIncMoneyMessage — v83 0x… / v95 0x…`). Citation-only; no logic change.

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-packet/character/clientbound/status_message_test.go libs/atlas-packet/character/clientbound/status_message.go
git commit -m "task-104: per-arm verify markers + struct citations for OnMessage (v83 omits SP)"
```

---

## Task 6: Reconcile the seed templates from the corrected yaml

**Files:**
- Modify (generated): `services/atlas-configurations/seed-data/templates/template_{gms_83_1,gms_84_1,gms_87_1,gms_95_1,jms_185_1}.json`

- [ ] **Step 1: Generate the operations maps from the yaml**

Run from the worktree root:
```bash
go run ./tools/packet-audit operations
```
This rewrites each template's `CharacterStatusMessage` `operations` block from
`character_status_message.yaml`. The gms_83 block should now read (no `INCREASE_SKILL_POINT`,
FAME=4 … SKILL_EXPIRE=13):
```json
"operations": {
  "DROP_PICK_UP": 0,
  "QUEST_RECORD": 1,
  "CASH_ITEM_EXPIRE": 2,
  "INCREASE_EXPERIENCE": 3,
  "INCREASE_FAME": 4,
  "INCREASE_MESO": 5,
  "INCREASE_GUILD_POINT": 6,
  "GIVE_BUFF": 7,
  "GENERAL_ITEM_EXPIRE": 8,
  "SYSTEM_MESSAGE": 9,
  "QUEST_RECORD_EX": 10,
  "ITEM_PROTECT_EXPIRE": 11,
  "ITEM_EXPIRE_REPLACE": 12,
  "SKILL_EXPIRE": 13
}
```
v84/v87/v95/jms keep the SP-at-4 / FAME-at-5 layout.

- [ ] **Step 2: Verify the diff matches expectations**

Run:
```bash
git diff --stat services/atlas-configurations/seed-data/templates/
git diff services/atlas-configurations/seed-data/templates/template_gms_83_1.json | sed -n '1,40p'
```
Expected: gms_83 loses `INCREASE_SKILL_POINT` and FAME..SKILL_EXPIRE drop by one. The other four
templates should be unchanged (they already had the correct layout) — if any of them changes,
investigate (it may indicate a pre-existing drift the yaml now corrects; confirm against IDA).

- [ ] **Step 3: Gate**

Run:
```bash
go run ./tools/packet-audit operations --check ; echo "ops=$?"
```
Expected: exit 0 (templates now match the yaml).

- [ ] **Step 4: Commit**

```bash
git add services/atlas-configurations/seed-data/templates/
git commit -m "task-104: regenerate CharacterStatusMessage operations maps from corrected yaml"
```

---

## Task 7: dispatcher-lint — confirm clean with no baseline entry

**Files:** none (verification); `docs/packets/dispatcher-lint-baseline.yaml` must remain unchanged.

- [ ] **Step 1: Run the linter**

Run:
```bash
go run ./tools/packet-audit dispatcher-lint ; echo "lint=$?"
```
Expected: exit 0, and OnMessage is now scanned (it has >1 `#`-entry). Because the body layer is
already INV-2/3/5 clean and run.go has no dangling candidate (INV-4), it should pass **without** a
baseline entry.

- [ ] **Step 2: Confirm no baseline entry was added**

Run:
```bash
grep -n "OnMessage" docs/packets/dispatcher-lint-baseline.yaml ; echo "exit=$?"
```
Expected: no match (exit 1). If OnMessage appears in the baseline, REMOVE it and fix the real
violation instead — adding a baseline entry for new work is banned (DISPATCHER_FAMILY.md).

> If `dispatcher-lint` reports a violation: INV-4 (a `#`-entry name doesn't resolve to a struct →
> fix the struct name in run.go), or INV-1 (a struct mapped by >1 `#`-entry → you duplicated an
> entry). The body funcs are already INV-2/3 clean; do not "fix" them.

---

## Task 8: Evidence records

**Files:**
- Create: `docs/packets/evidence/{gms_v83,gms_v84,gms_v87,gms_v95,jms_v185}/character.clientbound.StatusMessage*.yaml`

- [ ] **Step 1: Decide whether evidence files are required for these cells**

Read `docs/packets/audits/VERIFYING_A_PACKET.md` and check the guild precedent: confirm whether a
TIER1-FIXTURE cell promotes on the `// packet-audit:verify` marker alone or whether it also needs an
evidence YAML. Record the finding. If markers alone promote (as `status.json` likely shows for the
existing InventoryFull cells), this task is a no-op — note that and skip to Task 9.

- [ ] **Step 2: Write one evidence record per arm per applicable version (only if required)**

Use the schema from `docs/packets/evidence/<v>/guild.clientbound.*.yaml`:
```yaml
packet: character/clientbound/<StructName>
direction: clientbound
version: <version>
category: TIER1-FIXTURE
ida:
    function: CWvsContext::OnMessage#<Arm>
    address: "0x<delegate addr>"
    decompile_sha256: <sha256 of the decompiled delegate text>
```
Compute `decompile_sha256` from the exact decompiled function text captured in Task 1. v83 gets no
SP evidence file. jms 0xF: none unless resolved.

- [ ] **Step 3: Commit (only if evidence files were written)**

```bash
git add docs/packets/evidence/
git commit -m "task-104: per-arm evidence records for OnMessage arms"
```

---

## Task 9: Regenerate the coverage matrix

**Files:**
- Modify (generated): `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`

- [ ] **Step 1: Regenerate**

Run from the worktree root:
```bash
go run ./tools/packet-audit matrix
```

- [ ] **Step 2: Inspect the SHOW_STATUS_INFO row**

Run:
```bash
grep -n "SHOW_STATUS_INFO\|CWvsContext::OnMessage" docs/packets/audits/STATUS.md
```
Expected: the op row now aggregates the 24 arms worst-of. jms cell ❌→✅ for the shared arms (if all
15 verified). v83 cell ✅ (its 14 arms verified; SP ⬜ does not pull the row down — version-absent
is not a failure). Confirm no cell is a fabricated ✅ — every ✅ must trace to a marker+address.

- [ ] **Step 3: Gate**

Run:
```bash
go run ./tools/packet-audit matrix --check ; echo "matrix=$?"
```
Expected: exit 0 (the freshly generated files match; `toolSha` stamp matches HEAD).

- [ ] **Step 4: Commit**

```bash
git add docs/packets/audits/STATUS.md docs/packets/audits/status.json
git commit -m "task-104: regenerate coverage matrix (OnMessage 24-arm worst-of; jms shared arms ✅)"
```

---

## Task 10: Call-site & validator verification (no logic change)

**Files:** read-only verification across `services/atlas-channel`.

- [ ] **Step 1: Confirm every consumer routes through the body funcs (no literal-mode construction)**

Run:
```bash
grep -rn "NewStatusMessage" services/atlas-channel/ ; echo "direct-construct exit=$?"
grep -rln "CharacterStatusMessage.*Body" services/atlas-channel/
```
Expected: the first command returns **no** matches (consumers must not construct structs with a
literal mode — they call the `…Body` funcs). The second lists the ~8 consumer files (drop,
compartment, quest, asset, character, system_message, conversation_reward_notice). If a consumer
constructs a struct directly, re-route it through the matching body func.

- [ ] **Step 2: Confirm the writer seed entry keeps a non-empty opcode (FR-8.2 trap)**

Run:
```bash
grep -n "CharacterStatusMessage" services/atlas-configurations/seed-data/templates/template_gms_83_1.json
```
Expected: the `opCode: "0x27"` + `writer: CharacterStatusMessage` block is intact (Task 6 only
touched `operations`). A writer with an empty/missing opcode is silently dropped.

- [ ] **Step 3: Re-confirm no SP emission on v83 (D5)**

Re-run Task 0 Step 4's grep. Expected: still zero SP call sites. Document in the task notes:
"v83 SP is ⬜ because (a) the key is absent from the v83 operations table and (b) no consumer emits
SP; `ResolveCode` would return 99 if ever called on v83, which is acceptable as v83 has no SP arm."

- [ ] **Step 4: Build atlas-channel against the updated libs**

Run:
```bash
( cd services/atlas-channel && go build ./... ) ; echo "channel-build=$?"
```
Expected: exit 0 (no API changed, so this should be clean).

---

## Task 11: Full build/vet/test + bake gates

**Files:** none.

- [ ] **Step 1: Module-level build/vet/test**

Run from the worktree root, per changed module:
```bash
for m in libs/atlas-packet tools/packet-audit services/atlas-channel services/atlas-configurations; do
  echo "=== $m ===";
  ( cd "$m" && go build ./... && go vet ./... && go test -race ./... ) ; echo "$m exit=$?";
done
```
Expected: every module exit 0.

- [ ] **Step 2: Redis key guard**

Run:
```bash
GOWORK=off tools/redis-key-guard.sh ; echo "redis=$?"
```
Expected: exit 0 (no Redis usage introduced).

- [ ] **Step 3: Docker bake atlas-channel**

Run from the worktree root:
```bash
docker buildx bake atlas-channel ; echo "bake=$?"
```
Expected: exit 0. (libs/atlas-packet is the only Go module whose source changed in a way the
channel image consumes; bake confirms the shared Dockerfile COPYs it. No new lib was added, so no
Dockerfile/go.work edit is expected.)

- [ ] **Step 4: Re-run all four packet-audit gates together (final)**

Run:
```bash
go run ./tools/packet-audit dispatcher-lint && \
go run ./tools/packet-audit matrix --check && \
go run ./tools/packet-audit fname-doc --check && \
go run ./tools/packet-audit operations --check ; echo "all-gates=$?"
```
Expected: `all-gates=0`.

---

## Task 12: Live-config runbook (authored, NOT executed)

**Files:**
- Create: `docs/tasks/task-104-message-dispatcher-family/runbook-live-config.md`

Seed templates apply only at tenant creation; existing tenants do NOT get the corrected mode table
(see `bug_new_opcodes_not_in_live_tenant_config`). The v83 correction must be PATCHed into live
tenants post-deploy.

- [ ] **Step 1: Write the runbook**

Include: (a) how to enumerate live tenants + their versions (k8s/Grafana MCP); (b) for each
**v83** tenant, the PATCH to the `CharacterStatusMessage` writer `operations` map to the corrected
v83 table (drop `INCREASE_SKILL_POINT`, FAME=4 … SKILL_EXPIRE=13); (c) confirm v84/v87/v95/jms
tenants already match (no PATCH needed unless they drifted); (d) restart the channel pods (the
projection does not hot-reload writer operations); (e) post-restart verification: channel logs show
no `unhandled message op` for MESSAGE and a v83 fame gain renders as fame (not meso) and a
skill-expire renders. Mark the runbook **execution-gated on merge/deploy + operator authorization**.

- [ ] **Step 2: Commit**

```bash
git add docs/tasks/task-104-message-dispatcher-family/runbook-live-config.md
git commit -m "task-104: live-config runbook for v83 OnMessage mode-table correction (post-deploy)"
```

---

## Task 13: Code review before PR

**Files:** `docs/tasks/task-104-message-dispatcher-family/audit.md` (written by reviewers).

- [ ] **Step 1: Run the modular code review**

Invoke `superpowers:requesting-code-review`. It will dispatch `plan-adherence-reviewer` (verify all
13 tasks landed) and `backend-guidelines-reviewer` (Go DOM-* — Go files changed). No TS changed, so
no frontend reviewer. Reviewers write to `audit.md`.

- [ ] **Step 2: Address findings**

Use `superpowers:receiving-code-review` to triage. Re-run the affected gate(s) after any fix.

- [ ] **Step 3: Final gate sweep before opening the PR**

Re-run Task 11 Step 4 (all four packet-audit gates) + `go test -race` in the changed modules.
Expected: all exit 0. Only then open the PR.

---

## Self-Review (completed by plan author)

**Spec coverage** (PRD §4 FR-* and §10 acceptance):
- FR-1 grounding/honesty → Task 1 (decompile every value, cite addresses), Task 5 Step 6, D7
  stop-and-ask in Task 1 Step 5. ✓
- FR-2 IDA enumeration → Task 1. ✓
- FR-3 discrete-per-mode structs → already built; verified in Task 3 (one `#`-entry each) + Task 5
  (full-body round-trip). ✓
- FR-4 config-driven modes → already built (body funcs); operations table corrected Task 2/6. ✓
- FR-5 run.go rewire (remove single rep, no phantom root) → Task 3. ✓
- FR-6 export completeness (jms 16-delegate + GMS delegate structure, real addresses) → Task 4. ✓
- FR-7 per-version verification → Task 5 (markers) + Task 9 (matrix). ✓
- FR-8 call-site migration / validator retained → Task 10. ✓
- FR-9 seed templates + matrix → Task 6 + Task 9. ✓
- FR-10 four gates + build/bake → Task 7, Task 11. ✓
- Acceptance §10 message.yaml naming → resolved per design D9 (use existing
  `character_status_message.yaml`; Task 2). ✓
- Live-config runbook → Task 12. ✓

**Placeholder scan:** No "TBD"/"add error handling"/"similar to Task N". The only intentional
`0x<addr>` placeholders are the per-version delegate addresses that MUST be filled from live IDA
during Task 1 — they cannot be invented at plan time (CLAUDE.md grounding rule); each is explicitly
flagged as "from Task 1". jms 0xF is a real stop-and-ask, not a deferral.

**Type/name consistency:** Struct names, body-func names, and operations keys match
`status_message.go` / `status_message_body.go` exactly (cross-checked against the read of both
files). `#`-entry suffixes derive from struct-name suffixes; DropLoss splits into two
(`#DropLossStackableItem`, `#DropLossUnStackableItem`) per design D2 (24 total).

**Known deviation from the design, justified:** Design D5 says the v83 SP path "resolves to a
no-op." `resolve.go:27` proves `ResolveCode` returns 99 and still encodes. The plan corrects this:
the real safeguard is the **absent call site + absent key** (Task 0 Step 4 / Task 10 Step 3), and
v83 SP is ⬜ by genuine arm-absence. Captured in `context.md`.
