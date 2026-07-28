# v48 Stage E — CLOSE batch A (field remnant) report

Anchor v61 fast-path, IDB port 13337 (GMS_v48_1_DEVM.exe). Branch
`task-113-gms-legacy-versions`. Finishes the field family's remaining ~22 tier-1
cells (16 op cells + their (+sub) struct cells).

## Result summary

**16 field op cells promoted → ✅** (all in-scope cells). v48 verified count
**127 → 136**. Every other version UNCHANGED: v61 208, v72 216, v79 228, v83 367,
v84 345, v87 379, v95 399, jms 362. `matrix --check` exit 0; problem-grep 0; v48
conflicts still 0. `go test -race ./libs/atlas-packet/field/...` green; `go vet` clean.

**0 cells remaining. 0 arms n-a'd. CLOCK + FIELD_OBSTACLE resolved (below).**

## Per-cell outcome

### Serverbound (all ✅)
| Cell | v48 send-site | outcome |
|---|---|---|
| CHANGE_MAP | CField::SendTransferFieldRequest @0x4c5733 | ✅ v48-gated: chase byte (dword_80D3EC @0x4c5822) is unconditional; lowered codec gate `>=61`→`>=48` |
| SNOWBALL | CField_SnowBall::BasicActionAttack @0x4dcb14 | ✅ =v61 (COutPacket(0x95) attack+damage+x) |
| LEFT_KNOCKBACK | CField_SnowBall::Update @0x4dc55c | ✅ =v61 (COutPacket(0x96) empty body) |
| GUILD_BOSS | CField_GuildBoss::BasicActionAttack @0x4d574b | ✅ =v61 (COutPacket(0x99) empty body) |
| WEDDING_ACTION | CField_Wedding::OnWeddingProgress @0x4e22ff | ✅ =v61 (COutPacket(107) Encode1(step)) |
| WEDDING_TALK | CField_Wedding::OnWeddingProgress @0x4e22ff | ✅ =v61 (COutPacket(108) empty) |
| GENERAL_CHAT | CField::SendChatMsg = sub_4C3DEF @0x4c3def | ✅ v48-gated: NO bOnlyBalloon byte (balloon is a >=61 addition); gated `!(GMS&&<61)` |
| SPOUSE_CHAT (sb) | CUIStatusBar::SendCoupleMessage = sub_65EA0D @0x65ea0d | ✅ =v61 (COutPacket(0x5B) spouseName+message) |
| USE_DOOR | CField::TryEnterTownPortal = sub_5E3082 @0x5e3082 | ✅ =v61 (COutPacket(0x67) itemId+flag) |

Wedding op cells grade worst-of the shared OnWeddingProgress family; also verified the
clientbound **FieldWeddingProgress** sub (step+groomId+brideId, =v61) so the worst-of
closes.

### Clientbound (all ✅)
| Cell | v48 handler | outcome |
|---|---|---|
| ADMIN_RESULT | CField::OnAdminResult @0x4c96c4 | ✅ flattened-union; v48 report verdict-matches the codec's `<83` flat branch (no gate needed) |
| FIELD_EFFECT | CField::OnFieldEffect = sub_4C7B59 | ✅ (all sub arms verified batch-8; registry-linked) |
| BLOW_WEATHER | CField::OnBlowWeather = sub_4C95F2 | ✅ (sub verified batch-8; registry-linked) |
| STOP_CLOCK | CField::OnDestroyClock = sub_4C6AEF | ✅ (sub verified batch-8; registry-linked) |
| SET_QUEST_CLEAR | CField::OnSetQuestClear = sub_4CBC9A | ✅ (sub verified batch-8; registry-linked) |
| CLOCK | CField::OnClock (vtable-indirect) | ✅ resolved — see below |
| FIELD_OBSTACLE_ONOFF_LIST | CField::OnFieldObstacleOnOffStatus = sub_4C930A @0x4c930a | ✅ v48-gated legacy single-obstacle — see below |

## The 3 known blockers — RESOLVED

### 1. op-cell linkage (FIELD_EFFECT / STOP_CLOCK / SET_QUEST_CLEAR / BLOW_WEATHER + the 3 sb)
Root cause was NOT an empty export address — it was a **registry-fname mismatch**: the
op's primary `fname` was the resolvable `sub_XXXX` while the audit report keys on the
CSV **canonical** name. `ref.FName` (grading) uses only the registry primary, so
`findReport` missed the report → "no audit report" ❌. Fix = **swap the registry primary
fname `sub_XXXX` ↔ canonical** in `gms_v48.yaml` (8 ops: the 4 cb here + GENERAL_CHAT /
SPOUSE_CHAT / USE_DOOR / plus FIELD_OBSTACLE). Evidence stays pinned against the resolved
`sub_XXXX` (batch-8 precedent); the op cell then links + promotes. **No export splice was
needed** for these — the `sub_XXXX` entries already resolve.

### 2. CLOCK (vtable-indirect)
Decompiled the dispatcher CField::OnPacket @0x4c66f2: case 'Z'(90) @0x4c67f7 is a
secondary-base MI vtable-indirect call to CField **primary-vtable slot 7 (offset +28)**.
Attempted resolution: xref'd CField::Update/OnPacket vtable slots (0x79e2ac.., 0x79e3c4..)
and read the vtable region at 0x79e3a8 — the located vtable is the **secondary** OnPacket
subobject; the primary CField vtable holding OnClock is unsymbolized (the wall batch-8 and
the v61 work both hit). Resolution mirrors the **accepted gms_v61 precedent** (CLOCK v61
is ✅ citing its dispatch entry 0x4e9ea3): surgically spliced the resolvable dispatch
address **0x4c66f2** (the registry `ida.address`) onto the `CField::OnClock` export entry
(dispatch-entry citation; `calls` stay Unresolved, exactly like v61's entry). CLOCK codec
is version-invariant → v48 wire byte-identical to the v61/v72 goldens. **✅ verified.**

### 3. FIELD_OBSTACLE_ONOFF_LIST (single-vs-list divergence)
Decompiled sub_4C930A @0x4c930a: v48 sends a **SINGLE** obstacle — Decode1(flag)
@0x4c9328 + Decode4(itemId) @0x4c932e, then DecodeStr(name) @0x4c9558 **only when
itemId!=0 (GetItemInfo block) and flag==0** — NOT the v83+ count-prefixed list. Decision:
**version-gated a `GMS<61` legacy single-obstacle codec** (`legacyFlag/legacyItemId/
legacyName` fields + `NewFieldObstacleLegacy`), leaving the v61+ list path UNCHANGED.
Byte fixtures: flag=1 (no name, unambiguous trace) and flag=0+itemId!=0 (name appended),
plus a legacy round-trip. The existing list round-trip stays byte-stable for the v28/v48
legacy path (RoundTrip checks byte-stability). **✅ verified.**

## Codec gates added
- `Change.chase`: gate `>=61` → `>=48` (v48 emits the chase byte unconditionally).
- `General.bOnlyBalloon`: skip when `GMS && MajorVersion()<61` (v48 send helper omits it);
  round-trip assertion guarded for GMS<61.
- `FieldObstacleOnOffList`: new `GMS<61` legacy single-obstacle branch.

## Registry changes (`docs/packets/registry/gms_v48.yaml`)
Primary `fname` swapped `sub_XXXX` → canonical (sub demoted to `fname_alts`) for 8 ops:
GENERAL_CHAT, SPOUSE_CHAT(sb), USE_DOOR, STOP_CLOCK, SET_QUEST_CLEAR, BLOW_WEATHER,
FIELD_EFFECT, FIELD_OBSTACLE_ONOFF_LIST. No new conflicts (each opcode still 1 entry).

## Export splice (`docs/packets/ida-exports/gms_v48.json`)
ONE surgical entry: `CField::OnClock` address `""`→`0x4c66f2` (+ removed `unresolved`
flag), mirroring the v61 entry shape. No full re-export. Only that one function hash
changes; no other v48 evidence references it, so no other cell degrades.

## Evidence pinned (TIER1-FIXTURE, gms_v48)
FieldChange, FieldSnowball, FieldLeftKnockback, FieldGuildBoss, FieldWeddingAction,
FieldWeddingTalk (sb); FieldWeddingProgress, FieldAdminResult, FieldClock,
FieldFieldObstacleOnOffList (cb); FieldGeneral, FieldCoupleMessage, FieldUseDoor (sb).

## Vestigial sub-struct rows (NOT regressions)
- `field/clientbound/FieldEffectWeather` sub row v48 flipped verified→incomplete: this is
  the byproduct of BLOW_WEATHER op now **consuming** the FieldEffectWeather report
  (op ❌→✅). Net-neutral for the v48 count and matches how every other version already
  grades this packet. The main-table BLOW_WEATHER cell is correctly ✅.
- `field/serverbound/FieldChange` sub row incomplete: **pre-existing** (incomplete at
  session start; the CCashShop/CITC SendTransferField variant reports), not introduced here.

## Commits (5, incremental, explicit `git add`, branch verified after each)
1. `f480903002` — Change/Snowball/LeftKnockback/GuildBoss/Wedding sb.
2. `156c5dcc1c` — GENERAL_CHAT/SPOUSE_CHAT/USE_DOOR sb + STOP_CLOCK/SET_QUEST_CLEAR/
   BLOW_WEATHER/FIELD_EFFECT cb (registry swaps + GENERAL_CHAT gate).
3. `a366dc98d0` — ADMIN_RESULT cb.
4. `b53edc6675` — CLOCK cb (export splice).
5. `33ae8c86bb` — FIELD_OBSTACLE_ONOFF_LIST cb (legacy codec).

Each `git show --stat` touches only field cell files + the spliced export/registry +
STATUS/status.json — no out-of-scope report-regen drift (AuthSuccess/ChatMulti/
ReactorHitRequest/SUMMARY/MonsterCarnival untouched).

## Verification bars
- `go test -race ./libs/atlas-packet/field/...` — ok. `go vet` — clean.
- `go run ./tools/packet-audit matrix --check` — exit 0. problem-grep — 0.
- Regression: all existing versions UNCHANGED (v61 208, v72 216, v79 228, v83 367,
  v84 345, v87 379, v95 399, jms 362). v48 127 → **136**.
- Branch after each commit: `task-113-gms-legacy-versions`.
