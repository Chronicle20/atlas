# v48 Stage E — Batch 8 (field CONTINUATION) report

Anchor v61 fast-path, IDB port 13337 (GMS_v48_1_DEVM.exe). Branch
`task-113-gms-legacy-versions`. Continues batch-7 (which did 9 field cells); this
batch tackled the remaining 31 field tier-1 cells.

## Result summary

**9 cells promoted ❌/🟡 → ✅** (7 sub-struct + 2 op), v48 verified count **96 → 105**.
All other versions UNCHANGED (v61 208, v72 216, v79 228, v83 367, v84 345, v87 379,
v95 399, jms 362). `matrix --check` exit 0; problem-grep 0; v48 conflicts still 0.

**22 cells remain** (5 sub + 17 op) — see the EXACT remaining list at the bottom.
Stopped cleanly per the brief's budget rule after resolving the flagged divergences;
no false passes.

## Per-cell outcomes

### Promoted this batch (✅)

| Cell | v48 fname @addr | outcome | evidence |
|---|---|---|---|
| FieldEffectSummon (sub) | sub_4C7B59 mode 0 @0x4c7b59 | ✅ =v61 (Decode1 effect + Decode4 x + Decode4 y) | TIER1-FIXTURE |
| FieldEffectTremble (sub) | sub_4C7B59 mode 1 | ✅ =v61 (Decode1 bHeavy + Decode4 delay) | TIER1-FIXTURE |
| FieldEffectString (sub) | sub_4C7B59 modes 2/3/4/6 | ✅ =v61 (DecodeStr name) | TIER1-FIXTURE |
| FieldEffectBossHp (sub) | sub_4C7B59 mode 5 | ✅ =v61 (3×Decode4 + 2×Decode1) | TIER1-FIXTURE |
| FieldStopClock (sub) | sub_4C6AEF @0x4c6aef | ✅ empty wire (destroys clock CWnd, no reads) | TIER1-FIXTURE |
| FieldSetQuestClear (sub) | sub_4CBC9A @0x4cbc9a | ✅ empty wire (registry corrected, see below) | TIER1-FIXTURE |
| ARIANT_RESULT (op + sub) | CField::OnWarnMessage @0x4ca7d4 | ✅ =v61 (single DecodeStr) | TIER1-FIXTURE |
| SPOUSE_CHAT clientbound (op + sub) | CField::OnCoupleMessage @0x4c6faf | ✅ =v61 flattened union (no gate needed) | TIER1-FIXTURE |
| FieldEffectWeather (sub) | sub_4C95F2 @0x4c95f2 | ✅ v48-gated (`< 61` no-bool) | TIER1-FIXTURE |

Each: v48 byte-golden `Test…ByteOutputV48` (or V48 assertions) with a
`packet-audit:verify … version=gms_v48 ida=0x…` marker stacked on the existing
goldens; pinned a TIER1-FIXTURE evidence record with `verifies:`; matrix regenerated;
all confirmed `state: verified` in status.json. Read orders were verified against the
live v48 decompile (not blind-mirrored).

## Flagged divergences from batch-7 — RESOLVED

### SPOUSE_CHAT clientbound (batch-7 flagged 3/4 vs 4/5 divergence)
IDA-verified: v48 `CField::OnCoupleMessage @0x4c6faf` dispatches on `Decode1(mode) - 3`
(shifted DOWN one vs v61's `- 4`): mode-4 arm = DecodeStr(sender)@0x4c6fe8 +
Decode1(flag)@0x4c6ff4 + DecodeStr(chatText)@0x4c6fff; mode-3 arm =
Decode1(partnerFlag)@0x4c70d4 + DecodeStr(partnerText)@0x4c711e. **No codec gate was
needed**: the Atlas `SpouseChat` codec is a version-invariant *flattened union* (it
writes sender/flag/chatText/partnerFlag/partnerText positionally, never branching on
the mode value). The mode-4 (sender) arm sits at the lower address in v48, so the
flattened address-order is byte-identical to v61, and mode value 4 is valid in v48
(it selects the sender arm). Added a v48 marker + golden; no divergence remains.

### BLOW_WEATHER / FieldEffectWeather (batch-7 flagged, then this batch found the real divergence)
v48 `CField::OnBlowWeather = sub_4C95F2 @0x4c95f2` reads Decode4(itemId)@0x4c9604 then,
for a weather-type item, DecodeStr(message)@0x4c9669 — **NO leading bool**. Confirmed
against the v61 twin `sub_4ED39C` (op 106), which reads the same itemId-first shape.
The `!active` leading bool in `encodeGMS` is a **v83+ addition** (v83/84/87/95 verified
markers, no v61/72/79/48). Added `encodeGMSLegacy`/`decodeGMSLegacy` gated
`MajorVersion() < 61` (itemId + optional message), leaving v61+ UNCHANGED per the
brief. decodeGMSLegacy keys the message on `Available() > 0` for round-trip symmetry.

### CLOCK / STOP_CLOCK / SET_QUEST_CLEAR vtable-indirect send-sites (batch-7 flagged empty addr)
- **STOP_CLOCK** RESOLVED: `sub_4C6AEF @0x4c6aef` (dispatch case 'b'=98 in
  `CField::OnPacket @0x4c66f2`) — destroys `this[92]` (the clock CWnd), reads nothing.
  Verified against empty codec.
- **SET_QUEST_CLEAR** RESOLVED + **registry mislabel corrected**: the registry mapped
  SET_QUEST_CLEAR to op 93 / `sub_4CBB78`, but that handler reads an 8-byte
  DecodeBuffer FILETIME (a v48-inserted per-quest-timer packet). The REAL SetQuestClear
  is op 94 / `sub_4CBC9A` — reads nothing, returns `sub_4CC659(global+184)`,
  **structurally identical to v61 `CField::OnSetQuestClear @0x4ef90b`** (empty, QuestMan
  helper). v48 inserted an extra opcode between CLOCK and SetQuestClear vs v61's adjacent
  layout (v48 SetQuestTime is op 95, not 94). Swapped the two registry op labels
  (op 93 → placeholder `IDA_0X05D`; op 94 `IDA_0X05E` → `SET_QUEST_CLEAR`), each with a
  body-verification note.
- **CLOCK** DEFERRED (genuine partial blocker — see remaining). `CField::OnClock` is a
  **virtual** method (dispatch case 'Z'=90 is a vtable-indirect call to CField vtable
  slot 7, offset +0x1C). The CField vftable is **not symbolized** in this IDB
  (`??_7CField@@6B@` search + `*CField*6B*` globals filter both returned 0 hits), and the
  data xrefs to `CField::OnPacket` (0x79e3c4/43c/584/604) are subclass-vtable slots whose
  base I could not pin without reading and identifying each full vtable — exceeds this
  session's budget. The Clock codec is version-invariant with a v61 golden, so once the
  OnClock body address is resolved the sub is a fast-path; only the citable address is
  blocking.

## FIELD_EFFECT dispatcher arms (table 0–6, sub_4C7B59 @0x4c7b59)
Verified against the v48 switch on `Decode1(mode)@0x4c7b71`:
- mode 0 = **Summon**: Decode1(effect)@0x4c7f1f + Decode4(x)@0x4c7f29 + Decode4(y)@0x4c7f31 ✅
- mode 1 = **Tremble**: Decode1(bHeavy)@0x4c7eef + Decode4(delay)@0x4c7ef2 ✅
- modes 2/3/4/6 = **String** (screen/object/sound/BGM): DecodeStr(name)@0x4c7eb0 ✅
- mode 5 = **BossHp**: Decode4(monsterId)@0x4c7c55 + Decode4(curHp)@0x4c7c5e +
  Decode4(maxHp)@0x4c7c68 + Decode1(tagColor)@0x4c7c74 + Decode1(tagBg)@0x4c7c76 ✅
- **NO mode 7** (REWARD_RULLET absent) → `FieldEffectRewardRullet` is version-absent in
  v48 (n-a, mirroring v61; not a batch-8 target row).

## Senders named / arms n-a'd
- Named/body-verified handlers: sub_4C6AEF (OnDestroyClock), sub_4CBC9A (OnSetQuestClear),
  sub_4C95F2 (OnBlowWeather), sub_4C7B59 (OnFieldEffect), sub_4C930A
  (OnFieldObstacleOnOffStatus — see obstacle finding below). CField::OnWarnMessage and
  CField::OnCoupleMessage were already named in the IDB.
- n-a: FieldEffectRewardRullet (v48-absent mode 7) — inherited from v61 disposition,
  not a scope row.

## Obstacle finding (informs the remaining FIELD_OBSTACLE cell)
v48 `sub_4C930A @0x4c930a` (op 85) reads Decode1(flag) + Decode4(reactor/itemId) +
conditional DecodeStr(name) — a **SINGLE** obstacle, byte-shape identical to v61
`CField::OnFieldObstacleOnOffStatus @0x4ed30b`. The Atlas `FieldObstacleOnOffList`
codec encodes a **count-prefixed list** (`Decode4 count; N×[DecodeStr+Decode4]`), which
matches neither v48 nor v61. This is a genuine single-vs-list divergence requiring a
codec design decision (a `< 61` single-obstacle legacy branch, or dispositioning the
List struct as not-this-version); left as remaining rather than guess a codec shape.

## Op-level cells for the unnamed-sub handlers (STOP_CLOCK / SET_QUEST_CLEAR / FIELD_EFFECT / BLOW_WEATHER)
The **sub-struct** cells verified (they grade on marker+evidence). The **op-level**
cells stay `incomplete: "no audit report"` because their canonical CSV fname
(`CField::OnDestroyClock` / `OnSetQuestClear` / `OnFieldEffect` / `OnBlowWeather`)
resolves to an **empty address** in `docs/packets/ida-exports/gms_v48.json` — v48's
handlers are unnamed `sub_XXXX` in the IDB, so the export's CSV-seeded canonical stubs
were never populated. This is producible (IDB-rename → targeted `packet-audit export`
re-harvest of just those fnames → splice the address+calls into the committed export →
regenerate the specific reports), but was NOT done this session to avoid late-session
export/report-gen drift (the export is non-idempotent and the report-regen surface is
broad). FIELD_EFFECT additionally caps at 🧩 (StateFamily) as a dispatcher regardless.
ARIANT_RESULT and SPOUSE_CHAT-cb op cells DID promote because their canonical fnames are
already named + resolved in the v48 export.

## Gates added
- `EffectWeather.encodeGMSLegacy` / `decodeGMSLegacy` gated `GMS MajorVersion() < 61`
  (itemId-first, no leading bool). v61+ unchanged.
- No other codec gates. (SpouseChat needed none — flattened union.)

## Registry change
- `docs/packets/registry/gms_v48.yaml`: swapped op labels of the op-93 (`sub_4CBB78`,
  now `IDA_0X05D`) and op-94 (`sub_4CBC9A`, now `SET_QUEST_CLEAR`) entries, each with a
  body-verification note. No new conflicts (each opcode still has exactly one entry).

## Commits (4, incremental, explicit `git add`, branch verified after each)
1. `d110491fc7` — verify field/FieldEffect dispatcher subs (summon/tremble/string/bosshp).
2. `5e141e3cb7` — verify field/StopClock + SetQuestClear (+ registry mislabel correction).
3. `2b5d48ffb2` — verify field/AriantResult + SpouseChat clientbound.
4. `f1d69fea93` — verify field/EffectWeather (BLOW_WEATHER) with `< 61` no-bool gate.

`git show --stat` on each: only field cell test files (+ effect_weather.go codec + the
gms_v48.yaml registry) + `docs/packets/evidence/gms_v48/field.clientbound.*.yaml` +
STATUS.md + status.json. No out-of-scope report-regen drift (AuthSuccess/ChatMulti/
ReactorHitRequest/SUMMARY/MonsterCarnival untouched).

## Verification bars
- `go test ./libs/atlas-packet/field/...` — ok (clientbound + serverbound green).
- `go vet ./libs/atlas-packet/field/clientbound/` — clean.
- `go run ./tools/packet-audit matrix --check` — exit 0.
- problem-grep (`orphan|dangling|stale|drift|unresolv|malformed` in STATUS.md) — 0.
- Regression: verified counts UNCHANGED for every existing version (v61 208, v72 216,
  v79 228, v83 367, v84 345, v87 379, v95 399, jms 362). v48 96 → **105** (+9).
- Branch after each commit: `task-113-gms-legacy-versions`.

## EXACT remaining in-scope cells (22: 5 sub + 17 op)

### Clientbound sub (1)
- **FieldFieldObstacleOnOffList** — single-vs-list divergence (see Obstacle finding);
  needs a codec decision before fixturing.

### Clientbound op (8)
- **CLOCK** — OnClock is virtual (CField vtable slot 7); CField vftable not symbolized
  → citable body address unresolved (see CLOCK note). Codec is version-invariant fast-path
  once the address is resolved.
- **STOP_CLOCK**, **SET_QUEST_CLEAR**, **BLOW_WEATHER**, **FIELD_EFFECT** — sub cells
  DONE; op cells blocked on the export canonical-fname-address populate (see the op-level
  note). FIELD_EFFECT also caps at 🧩 as a dispatcher.
- **WHISPER** — `CField::OnWhisper @0x4c71d5` is an 8-mode dispatcher (SendResult /
  Receive / FindResult{CashShop,Map,Channel,Error} / Error / Weather). Not attempted;
  needs per-arm decompose + fixtures for all sibling structs.
- **ADMIN_RESULT** — `CField::OnAdminResult @0x4c96c4`. The flattened-union address-order
  is `mode + s,s,s,b,b,b,b,b,b,i,b`, which DIFFERS from the current codec's `< 83`
  (v79) branch (`b,b,b,i,b,b,s,s,s,b,b,b`); needs a `< 61` flat-schema branch. Not
  completed.
- **FIELD_OBSTACLE_ONOFF_LIST** — same single-vs-list divergence as its sub.

### Serverbound (13 = 9 op + 4 sub) — none attempted this batch
- **CHANGE_MAP** (op + sub FieldChange), **GENERAL_CHAT** (op + sub FieldGeneral),
  **SPOUSE_CHAT serverbound** (op + sub FieldCoupleMessage), **USE_DOOR** (op + sub
  FieldUseDoor), **WEDDING_ACTION**, **WEDDING_TALK**, **SNOWBALL**, **LEFT_KNOCKBACK**,
  **GUILD_BOSS**. Each needs send-site body-verification + candidatesFromFName +
  report-gen + routed-in-template. (Per batch-7/Stage-B: GENERAL_CHAT/CHANGE_MAP=30/
  USE_DOOR=103 opcodes already registered — these are the lighter byte-fixture cells to
  start with next session.)

## Notes / concerns
- The op-level cells for the four unnamed-sub handlers (StopClock/SetQuestClear/
  FieldEffect/BlowWeather) are the main "sub-done-but-op-pending" gap. Resolving them is a
  coherent producible workflow (IDB rename → targeted re-harvest → export address splice →
  targeted report regen) but was deferred to protect the committed export/reports from
  drift late in a long session — flagged here rather than silently skipped.
- CLOCK is the only *genuine* address blocker (virtual method, unsymbolized vftable);
  everything else remaining is either a codec decision (obstacle single-vs-list),
  a `< 61` flat-schema branch (AdminResult), an 8-arm decompose (Whisper), or the
  serverbound report-gen pipeline.
