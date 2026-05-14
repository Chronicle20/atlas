# Task-028 Post-Phase-B — Character-Domain Audit Closeout

## Final state

- Packets audited: 52 (35 clientbound + 17 serverbound) — character domain only (GMS v95 primary).
- Cross-version passes: GMS v83, GMS v87, JMS v185 — all character FNames covered.
- Character-domain verdicts (GMS v95): ✅ 31 / ❌ 20 / 🔍 1 / ⚠️ 0 / pending 0.
  - Clientbound: 35 audited — 16 ✅ / 19 ❌ (tool-limitation FPs documented; no unresolved wire bugs).
  - Serverbound: 17 audited — 15 ✅ / 1 ❌ (KeyMapChange — loop-count tool limitation) / 1 🔍 (Move — movement encoding review).
- Combined SUMMARY.md (task-027 login + task-028 character, GMS v95): 81 packets — 58 ✅ / 20 ❌ / 1 🔍.
- IDA-export coverage: GMS v95 / GMS v83 / GMS v87 / JMS v185 — character FNames populated.
- Total commits on branch: 43 (merge-base `c51166f6e`).

## Real wire bugs fixed

| Packet | File | IDA citation | Fix one-liner | Versions affected |
|---|---|---|---|---|
| `CharacterExpression` | `libs/atlas-packet/character/clientbound/expression.go` | `CUser::OnEmotion@0x8e0150` | Missing `duration` (int32) + `byItemOption` (bool) — truncated packet by 5 bytes | GMS v95 (v83/v87 correctly omit these fields; JMS emits duration but not byItemOption) |
| `ItemUpgrade` | `libs/atlas-packet/character/clientbound/item_upgrade.go` | `CUser::ShowItemUpgradeEffect@0x8e7b00` | Missing `enchantCategory` (int32) + `enchantResultFlag` (byte) — malformed observer stream for Vega/enchant scrolls | GMS v95 (v83/v87 use 4-byte layout; JMS emits enchantResultFlag but not enchantCategory) |
| `ExpressionRequest` | `libs/atlas-packet/character/serverbound/expression.go` | `CWvsContext::SendEmotionChange@0x9f9320` | Missing `duration` (int32) + `byItemOption` (bool) in serverbound encode/decode | GMS v95 (v83/v87/JMS use emotionId-only wire) |
| `CharacterDespawn` | `services/atlas-configurations/seed-data/templates/template_gms_95_1.json` | `CUserPool::OnPacket@0x94ddf0` dispatch case 0xB4 | Template had opcode `0xE7` (self-character range); other players remained visible on-screen after field exit | GMS v95 only |
| `CharacterList` | `libs/atlas-packet/model/character_list_entry.go` | `CLogin::OnSelectWorldResult@0x5dda00` | Duplicate `rankEnabled` byte from early-return pattern; hoisted `WriteBool(!m.gm)` before return to make byte unconditional | GMS v95 (flipped ❌ → ✅ after early-return suffix-taint fix) |
| `CharacterViewAllCharacters` | `libs/atlas-packet/character/clientbound/view_all.go` | `CLogin::OnViewAllCharResult@0x5de435` (case 0) | `viewAll=false` zero-value caused each entry to consume an extra family byte, misaligning rank fields | GMS v95 |

## Template opcode / enum fixes

| Template file | Old → New | IDA case-statement | Reason |
|---|---|---|---|
| `template_gms_95_1.json` | `CharacterDespawn` opcode `0xE7` → `0xB4` | `CUserPool::OnPacket@0x94ddf0` case 0xB4 = `OnUserLeaveField` | 0xE7 is the first opcode in the self-character (`OnUserLocalPacket`) range — wrong subsystem |
| `template_gms_95_1.json` | Added 10 missing character hot-path writers (CharacterSpawn/Attack*/Damage/BuffGive/BuffGiveForeign/Movement/SkillChange) | `CUserPool::OnUserEnterField@0x94db40` (0xB3), `CUserRemote::OnAttack@0x95a670` (0xD3–D6), `CUserRemote::OnHit@0x954c50` (0xDA), `CWvsContext::OnTemporaryStatSet@0xa02fc0` (0x1F), `CUserRemote::OnSetTemporaryStat` (0xE1), `CUserRemote::OnMove@0x948a80` (0xD2), `CWvsContext::OnChangeSkillRecordResult@0x9f5f30` (0x23) | Template was missing all 10 character-domain writer opcodes; clients could never receive spawn/attack/damage/buff packets |
| `template_gms_87_1.json` | Added 8 missing character hot-path writers (same as v95 minus Spawn + Movement) | Same IDA refs at v87 addresses | v87 template had same gap as v95 |
| `template_jms_185_1.json` | Added 8 missing character hot-path writers | Same IDA refs at JMS addresses | JMS template had same gap |

## Tooling improvements

- **Analyzer early-return suffix-taint** (Phase 0 / commit `b1af67f6d`): when a guarded block contains a `return` statement, the analyzer now marks the block's suffix as tainted so the flat call list doesn't double-count the continuation path. Flipped `CharacterList` from ❌ → ✅.
- **Registry support for `EncodeForeign`** (Phase 1 / commit `b4a594dea`): `CharacterTemporaryStat::EncodeForeign` registered in the TypeRegistry so sub-struct descent into `BuffGiveForeign` can resolve the foreign encoding path without a cycle.
- **FlattenWithRegistry cycle guard** (commit `32b585e8f`): added a visited-set check to `FlattenWithRegistry` to prevent infinite recursion on mutually-referencing types; unblocked character candidatesFromFName registration.
- **Registry fixtures for `AttackInfo`, `Pet`, `DamageTakenInfo`, `Movement` + element sub-types** (commits `b44578661`, `f8168998d`): test coverage asserting all complex sub-types are registered and resolvable without cycles.
- **Character candidatesFromFName** (commit `32b585e8f`): all 44+ character FNames mapped in `tools/packet-audit/cmd/run.go` so the pipeline can route IDA exports to the correct atlas struct.

## Remaining work

| Area | What | Why deferred |
|---|---|---|
| `BuffCancel` / `BuffCancelForeign` / `BuffGive` / `BuffGiveForeign` | Sub-op dispatch on leading mode byte; atlas has per-mode structs but pipeline only sees the outermost Decode call | Phase 3: per-mode IDA sub-function trace for each temporary-stat struct |
| `EffectSimple` / `EffectQuest` / `EffectSkillUse` | `CUser::OnEffect` has 16+ sub-op modes (case 0–15+); flat analyzer cannot follow the mode-byte dispatch tree | Phase 3: per-mode sub-function trace |
| `StatusMessageDropPickUpInventoryFull` (and all other StatusMessage sub-types) | `CWvsContext::OnMessage` has 14–15 sub-op modes; pipeline report is representative (mode 0 only) | Phase 3: per-mode IDA sub-function trace for each atlas StatusMessage struct |
| `CharacterInfo` ❌ | Complex packet: bool-terminated pet list, optional taming mob, wishList loop, version-guarded monster book, MedalAchievementInfo sub-struct, chair list. Flat analyzer cannot track loop state or conditional sub-sequences. Manual cross-check confirms v95 encoding is correct. | Phase 3: loop-aware analyzer + sub-struct descent |
| `CharacterSitResult` ❌ | Branch-flattening FP: if/else exclusive branches union'd into sequential writes. Manual IDA check confirms `CUserLocal::OnSitResult` encoding matches. | Phase 3: exclusive if/else branch detection |
| `AddCharacterEntry` ❌ | Emits 18 trailing rank bytes that `CLogin::OnCreateNewCharacterResult` doesn't read (length-prefixed packets; client silently ignores trailing bytes). Functionally harmless. | Follow-up refactor: dedicated non-rank payload type |
| `CharacterViewAllCharacters` ❌ | `DecodeBuffer(0x10)` (bulk 16 bytes) in IDA vs 4 × `WriteInt` in atlas — same wire bytes; diff tool lacks DecodeBuf-expansion support | Phase 3: diff engine DecodeBuf expansion |
| `CharacterKeyMap` ❌ | Loop-count tool limitation: atlas emits 90 entries; v83/v87 IDA has 89. Extra entry is harmless (client treats as full keymap). | Phase 3: loop-count modeling |
| `Attack` ❌ | Complex: sub-op dispatch on attack type (melee/bow/magic/summon); per-type payload differs. Flat analyzer sees only the outermost Decode sequence. | Phase 3: attack-type sub-function trace |
| `CharacterSpawn` ❌ | `GW_CharacterStat` + `AvatarLook` sub-structs plus version-gated fields; flat analyzer cannot fully expand nested sub-struct chains | Phase 3: sub-struct descent |
| `CharacterSkillChange` ❌ | `SecondaryStat` sub-struct nested analysis beyond current analyzer depth | Phase 3: deep sub-struct descent |
| `CharacterAppearanceUpdate` ❌ | JMS uses list format for couple/friendship; GMS uses single-entry buffers — beyond flat analyzer scope | Phase 3: sub-struct descent |
| `CharacterMovement` ❌ | Movement encoding depends on movement type; flat analyzer cannot follow the type-dispatch branch | Phase 3: movement-type sub-function trace |
| `CharacterDamage` ❌ | Damage packet has attacker-type sub-dispatch; flat analysis cannot resolve | Phase 3: sub-function trace |
| `KeyMapChange` ❌ | Loop-count: atlas encodes all key entries; tool limitation same as CharacterKeyMap | Phase 3: loop-count modeling |
| `Move` 🔍 | Movement encoding review: `CVecCtrlUser::EndUpdateActive` JMS wire confirmed narrower than GMS v95; gates narrowed but movement sub-type encoding needs deeper audit | Phase 3: movement sub-type trace |
| `ExpressionRequest` (JMS semantic mismatch) | JMS opcode 0x2B carries charId in the slot atlas reads as emotionId; re-broadcast `CharacterExpression` carries the wrong value. Pre-existing on `main`, not introduced here. | Follow-up: dedicated JMS-aware decoder |
| `CLogin::SendCheckPasswordPacket` v87 `PartnerCode` | v87 appends `Encode4(PartnerCode)` after 3×Encode1 unknowns; atlas reads only `unknown2` for `>=95` — v87 trailing 4 bytes silently ignored. | Low-severity; deferred |
| `CharacterSelectRegisterPic` / `CharacterSelectWithPic` v87 opcode layouts | v87 PIC-register (0x1E) and PIC-select (0x1D) have different layouts than the v95 equivalents at 0x1C/0x1D. | Requires v87-specific handler variants; deferred |
| `GW_CharacterStat` HP/MHP/MP/MMP int16→int32 widening | Sub-struct field width gate (v83/v87: int16, v95: int32) inside complex CharacterList/ViewAll packets that the flat analyzer cannot reach | Phase 3: sub-struct descent |
| `HealOverTime` JMS trailing `timeGetTime()` field | JMS appends Encode4(timeGetTime()) after Encode1(nType); atlas reads 5 fields then stops. Functionally zero-impact. | Low-severity; deferred |
| `bCharSale` character creation path | `CLogin::SendNewCharPacket` opcode 0x17 branch for `m_bCharSale==true` (9× AL items, no SubJob/gender). Atlas decoder absent. Cash Shop character creation flow not wired. | Follow-up task |
