# v48 Stage E — Batch 7 (field family) report

Anchor v61 fast-path, IDB port 13337 (GMS_v48_1_DEVM.exe). Branch `task-113-gms-legacy-versions`.

## Scope
Field family tier-1 gms_v48 ❌/🟡 cells. Extracted authoritative set from STATUS.md/status.json:
**28 op-level + 12 sub-struct = 40 cells** (op/sub overlap where an op's representative struct is also a sub row). This is the largest non-character family; per the brief it was expected to exceed one session's budget.

## Completed this session — 9 cells promoted ❌/🟡 → ✅ (all clientbound, single-struct, fast-path == read-order-verified against v48 body)

| Op / packet | v48 fname @addr | v48 read order (IDA-verified) | outcome |
|---|---|---|---|
| MULTICHAT / FieldMultiChat | CField::OnGroupMessage @0x4c6dd6 | Decode1(mode)+DecodeStr(from)+DecodeStr(message) | ✅ =v61 |
| BLOCKED_MAP / FieldBlockedMap | CField::OnTransferFieldReqIgnored @0x4c6b01 | Decode1(reason) | ✅ |
| BLOCKED_SERVER / FieldBlockedServer | CField::OnTransferChannelReqIgnored @0x4c6be7 | Decode1(reason) | ✅ |
| FORCED_MAP_EQUIP / FieldForcedMapEquip | CField::OnFieldSpecificData @0x4c6ca1 | vtable-forward, empty wire | ✅ |
| SUMMON_ITEM_INAVAILABLE / FieldSummonItemUnavailable | CField::OnSummonItemInavailable @0x4c7b1f | Decode1(message) | ✅ |
| OX_QUIZ / FieldOxQuiz | CField::OnQuiz @0x4c9e20 | Decode1(enabled)+Decode1(category)+Decode2(number) | ✅ |
| GMEVENT_INSTRUCTIONS / FieldGmEventInstructions | CField::OnDesc @0x4ca491 | Decode1(index) | ✅ |
| SET_QUEST_TIME / FieldSetQuestTime | CField::OnSetQuestTime @0x4cbcad | Decode1(count)+N×[Decode4(questId)+DecodeBuffer8(start)+DecodeBuffer8(end)] | ✅ |
| SET_OBJECT_STATE / FieldSetObjectState | CField::OnSetObjectState @0x4cbe02 | DecodeStr(name)+Decode4(state) | ✅ |

Each: added a GMS/48-context byte-golden test (`Test…ByteOutputV48`) with a
`packet-audit:verify … version=gms_v48 ida=0x…` marker (stacked on the existing
version-invariant golden), pinned a TIER1-FIXTURE evidence record with `verifies:`,
regenerated the matrix. All 9 op cells confirmed `state: verified` in status.json.

Read orders were verified against the live v48 decompile (not blind-mirrored); each
matched its Go writer and the version-invariant golden bytes exactly.

## Codec gates added
None. All 9 completed cells are version-invariant (v48 read order identical to the
existing golden); no legacy version-gate was required. No senders needed naming
(all fnames already present + named in the export). No arms dispositioned n-a.

## Commits (2, incremental, explicit `git add`, branch verified after each)
1. `7c2b8d646f` — verify field/FieldMultiChat (tier-1) — pilot cell validating the pipeline.
2. `55175d917e` — verify field simple clientbound ops (blocked-map/server, forced-equip, summon-item, ox-quiz, gmevent, set-quest-time, set-object-state) (tier-1).

`git show --stat` on both: only field cell test files + `docs/packets/evidence/gms_v48/field.clientbound.*.yaml` + STATUS.md + status.json. No out-of-scope report-regen drift.

## Verification bars
- `go test ./libs/atlas-packet/field/clientbound/` — ok (full package green).
- `go vet ./libs/atlas-packet/field/clientbound/` — clean.
- `go run ./tools/packet-audit matrix --check` — exit 0.
- problem-grep (`orphan|dangling|stale|drift|unresolv|malformed` in STATUS.md) — 0.
- Regression: verified counts UNCHANGED for every existing version — v61 208, v72 216, v79 228, v83 367, v84 345, v87 379, v95 399, jms 362. v48 88 → **96** (+8 op cells; MULTICHAT was committed one commit earlier taking 87→88).
- Branch after each commit: `task-113-gms-legacy-versions`.

## EXACT remaining in-scope cells (31: 19 op + 12 sub) — for continuation

### Clientbound — heavier (vtable-indirect / dispatcher / version-divergent)
- **CLOCK** (field/clientbound/FieldClock) — CField::OnClock is vtable-indirect; **empty address in gms_v48.json export**. Codec is version-invariant (has v61 golden citing the dispatch entry). Approach: resolve OnClock body via the vtable slot or cite the v48 CLOCK dispatch case in CField::OnPacket, mirror the golden.
- **STOP_CLOCK** (field/clientbound/FieldStopClock) + **sub FieldStopClock** — sub_4C6AEF, empty export address (same vtable pattern).
- **SET_QUEST_CLEAR** (field/clientbound/FieldSetQuestClear) + **sub FieldSetQuestClear** — sub_4CBB78, empty export address.
- **WHISPER** (field/clientbound/FieldWhisperError) — dispatcher; op cell grades worst-of-8 sibling arms under CField::OnWhisper @0x4c71d5 (SendResult/Receive/FindResultCashShop/FindResultMap/FindResultChannel/FindResultError/Error/Weather). Verify every arm (brief: v48 OnWhisper is 8-mode).
- **SPOUSE_CHAT clientbound** (field/clientbound/FieldSpouseChat) — **DIVERGENT from v61**. IDA-verified: v48 CField::OnCoupleMessage @0x4c6faf branches on `Decode1(mode) - 3`, handling **mode 3 and mode 4** (NOT v61's mode 4/5). mode-4 arm = DecodeStr(sender)+Decode1(flag)+DecodeStr(chatText) (matches v61 mode-4); mode-3 arm = Decode1(flag)[+optional DecodeStr(text)] (partner path). Requires a version-gated codec for the legacy range (leave v61+ unchanged), not a blind mirror.
- **ADMIN_RESULT** (field/clientbound/FieldAdminResult) — version-gated multi-arm writer (mode discriminator); CField::OnAdminResult @0x4c96c4; existing markers only v79+; brief flags a `<83` branch → v48 takes the oldest arm. Per-arm verification needed.
- **ARIANT_RESULT** (field/clientbound/FieldAriantResult) — CField::OnWarnMessage @0x4ca7d4; not yet decompiled this session.
- **FIELD_EFFECT** (field/clientbound/FieldEffectBossHp) — dispatcher family; sub_4C7B59 @0x4c7b59, FieldEffect table 0–6 (no REWARD_RULLET). Sub-struct arms all ❌: **FieldEffectBossHp, FieldEffectString, FieldEffectSummon, FieldEffectTremble, FieldEffectWeather**. Verify every arm against the 0–6 table.
- **BLOW_WEATHER** (field/clientbound/FieldEffectWeather) — sub_4C95F2 @0x4c95f2 (shares FieldEffectWeather sub with FIELD_EFFECT).
- **FIELD_OBSTACLE_ONOFF_LIST** (field/clientbound/FieldFieldObstacleOnOffList) + **sub** — sub_4C930A @0x4c930a.

### Serverbound — each needs audit report + candidatesFromFName + routed-in-template (heavier)
- **CHANGE_MAP** (field/serverbound/FieldChange) + **sub FieldChange** — v61/v79 transfer-field revive gate + WarpToMap to carry-check for v48.
- **GENERAL_CHAT** (field/serverbound/FieldGeneral) + **sub FieldGeneral**.
- **SPOUSE_CHAT serverbound** (field/serverbound/FieldCoupleMessage) + **sub FieldCoupleMessage**.
- **USE_DOOR** (field/serverbound/FieldUseDoor) + **sub FieldUseDoor**.
- **WEDDING_ACTION** + **WEDDING_TALK** (both field/serverbound/FieldWeddingAction — one struct, two ops; verify once).
- **SNOWBALL** (field/serverbound/FieldSnowball).
- **LEFT_KNOCKBACK** (field/serverbound/FieldLeftKnockback).
- **GUILD_BOSS** (field/serverbound/FieldGuildBoss).

## Notes / concerns
- SPOUSE_CHAT clientbound mode divergence (3/4 vs 4/5) is a concrete, IDA-verified finding — a continuation should version-gate the SpouseChat codec for the legacy range rather than stack a v48 marker on the shared union golden.
- CLOCK / STOP_CLOCK / SET_QUEST_CLEAR have empty addresses in gms_v48.json (vtable-indirect harvest); `evidence pin` will still emit `address: ""` (matches the v61 FieldClock evidence precedent), but the `ida=` marker should cite a resolvable v48 address (dispatch case or vtable slot) — resolve before fixturing.
- No genuine blockers hit; the remaining cells are simply heavier (dispatcher per-arm, version-gated, serverbound report-gen) than one session's budget allowed after the 9 fast-path clientbound cells. Stopped cleanly per the brief's budget rule; no false passes.
