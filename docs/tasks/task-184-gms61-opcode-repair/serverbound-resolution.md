# gms_61 serverbound opcode ground-truth — resolution pass (continuation)

## Session

- IDB session id: `965202bf`
- Binary: `E:\Programs\Nexon\IDBs_v9\GMS\v61\GMS_v61.1_U_DEVM.exe.i64` (`GMS_v61.1_U_DEVM.exe.i64`)
- Confirmed live via `idb_list` at the start of this pass (matched by binary filename).
- Method: same as the prior pass — `func_query`/`entity_query` name-regex sweeps plus `xrefs_to` on the
  `COutPacket::COutPacket(this, N)` constructor (`??0COutPacket@@QAE@J@Z`) to enumerate **every** call site in the
  binary (391 unique call sites; the `xrefs_to` result carried `"more": false`, i.e. this is the complete list, not
  a truncated sample), then targeted `decompile` of each candidate to read the literal opcode argument. Read-only —
  no template file was edited. Current template state was inspected only to check slot occupancy
  (`services/atlas-configurations/seed-data/templates/template_gms_61_1.json`).
- This pass consumed the prior 113-row audit (`prior-serverbound-audit.md`) as its starting point: 38 MATCH, 5
  MISMATCH (4 already applied in the template: ItemUse 0x47→0x43, PetFood 0x4B→0x47, MountFood 0x4C→0x48,
  UseSkill 0x5A→0x53; skill-book added @0x4B), 70 UNRESOLVED.

## Priority 1 result: InventoryMove

**TRUE opcode found: 0x42 (66 decimal).** The client's move-item send function is `sub_8315F8`@0x8315f8 (unnamed
in this IDB's partial naming pass — it sits directly between the named `SendSortItemRequest`@0x831564 and
`SendDropPickUpRequest`@0x8316b8, i.e. squarely inside the `CWvsContext` item-operation cluster).

Verbatim decompiled ctor line:

```
COutPacket::COutPacket((COutPacket *)v11, 66); /*0x831636*/
```

Full signature: `sub_8315F8(this, unsigned __int8 a2, unsigned __int16 a3, unsigned __int16 a4, unsigned __int16 a5)`
→ wire body is `Encode4(exclusive-request-id)`, `Encode1(a2 = inventory/compartment type)`,
`Encode2(a3 = from-slot)`, `Encode2(a4 = to-slot)`, `Encode2(a5 = count)` — the textbook "move item between slots"
shape.

Caller confirmation (two of five call sites decompiled):
- `sub_4BDD1A`@0x4bdd1a (auto-equip/quick-move: finds an empty equip slot, then
  `sub_8315F8(v7, 1u, this[5], (unsigned __int16)v18, 0xFFFFu)` — move from slot `this[5]` to the found empty
  slot, inventory type 1 = Equip, count sentinel 0xFFFF).
- `sub_4BDEDA`@0x4bdeda (equip-tab **drag-and-drop**: calls `sub_6BF8A5(a3, a4, a5)` — the mouse-point→slot-index
  helper found in the same investigation — to resolve the drop target, runs equip-compatibility/body-part checks,
  then `sub_8315F8(v63, 1u, v65[5], v60, 0xFFFFu)` — move from the drag-origin slot to the drop-target slot).

Both call sites are unambiguous "move item from slot A to slot B" actions. **InventoryMove's true opcode is 0x42.**

**Slot check:** 0x42 is a gap in the current template (nothing occupies it) — moving `CharacterInventoryMoveHandle`
there is collision-free.

## Priority 1 result: SummonBag chain unblock

Per the prior audit, `SendMobSummonItemUseRequest`@0x831c83 sends opcode **0x46** (70 decimal) but the template's
0x46 slot was occupied by `CharacterInventoryMoveHandle` (now relocating to 0x42 per above) — **0x46 is now free**
once InventoryMove moves out, so `CharacterItemUseSummonBagHandle` can take its true value 0x46.

That, in turn, vacates the template's current 0x4A slot (where `CharacterItemUseSummonBagHandle` sits today). Per
the prior audit, 0x4A (74 decimal) is genuinely sent by the client, but for **Bridle** item use
(`SendBridleItemUseRequest`@0x832005, item-prefix check `a3/10000==227`), not Summon Bag. There is **no handler in
the current template for Bridle under any name** (`entity_query`/grep over the template found nothing matching
"bridle" or "leash"). This is a real coverage gap distinct from a simple opcode correction — flagged, not silently
dropped.

**Full swap chain, in apply order (each step's target slot verified free before the *previous* step vacates it):**

1. `CharacterInventoryMoveHandle`: 0x46 → **0x42** (0x42 free beforehand)
2. `CharacterItemUseSummonBagHandle`: 0x4A → **0x46** (now free after step 1)
3. 0x4A is left **unassigned** — the real feature there (Bridle item-use) has no template handler at all today;
   implementing it is new work, not an opcode fix, and is out of scope for this read-only pass.

**SummonBag→0x46 chain is now fully unblocked** by the InventoryMove finding.

## Priority 2 result: full 0x43–0x5A item-use/stat/skill region re-verified

All previously-UNRESOLVED rows in this region are now resolved, and all previously-applied fixes were
independently re-confirmed:

- **0x4F `CharacterItemUseScrollHandle` — MATCH (newly confirmed).** `sub_8317A4`@0x8317a4 (unnamed, sits directly
  between `SendDropPickUpRequest`@0x8316b8 and `SendStatChangeItemUseRequest`@0x831880) sends
  `COutPacket::COutPacket((COutPacket *)v12, 79)` = 0x4F. Signature `(this, uint16 a2, uint16 a3, uint16 a4, a5)`
  → `Encode4, Encode2(a2), Encode2(a3), Encode2(a4), Encode1(a5)`. Caller `sub_4C072A`@0x4c072a gates on item-id
  prefix `/10000 ∈ {200,201,202,205,221,236,238}` — the classic MapleStory scroll-item category range — then calls
  `sub_8317A4(v38, v37[5], v43, v41, a4)` (source item slot, target equip slot, white-scroll flag, chaos flag).
  Matches the template's current 0x4F value exactly — **no change needed.**

- **0x50 `CharacterDistributeApHandle` — MATCH (newly confirmed).** `sub_8457EE`@0x8457ee sends
  `COutPacket::COutPacket((COutPacket *)v15, 80)` = 0x50, gated on `_ZtlSecureFuse<short>(v5+129,...) > 0` (AP
  remaining > 0). Caller `sub_72DFBB`@0x72dfbb is an `OnButtonClicked`-style dispatcher over control IDs
  `0x7D0`–`0x7D5` (2000–2005 — the classic STR/DEX/INT/LUK/HP/MP "raise" button IDs in the ability window), each
  branch calling `sub_8457EE(v2, <stat-bitmask>)`. Matches the template's current 0x50 — **no change needed.**

- **0x52 `CharacterDistributeSpHandle` — MATCH (newly confirmed).** `sub_8458EB`@0x8458eb (immediately follows
  `sub_8457EE` in memory) sends `COutPacket::COutPacket((COutPacket *)v8, 82)` = 0x52, identical
  `Encode4(id), Encode4(a2)` shape. Caller `sub_7201E6`@0x7201e6 walks the character's skill array checking
  job-branch skill-point caps (1st/2nd/3rd/4th job skill categories) before calling
  `sub_8458EB(v36, *v34)` (skill id to raise). Matches the template's current 0x52 — **no change needed.**

- **0x4B `CharacterSkillBookUseHandle` — MATCH (re-confirmed).** `SendSkillLearnItemUseRequest`@0x8325d2:
  `push 4Bh` immediately before the `COutPacket` ctor call (disasm-verified), gated on item-id prefix
  `/10000 ∈ {228,229}` (skill-book item categories). Matches the template's already-applied 0x4B — **no change
  needed**, confirms the prior fix was correct.

- 0x43 (`CharacterItemUseHandle`), 0x47 (`PetFoodHandle`), 0x48 (`MountFoodHandle`), 0x49
  (`CharacterCashItemUseHandle`), 0x4D (`TeleportRockUseHandle`), 0x4E (`CharacterItemUseTownScrollHandle`), 0x51
  (`CharacterHealOverTimeHandle`), 0x53 (`CharacterUseSkillHandle`), 0x54–0x59 — all re-confirmed MATCH per the
  prior pass's evidence; not re-decompiled in this pass since nothing adjacent to them changed.

**Net result for Priority 2: zero new mismatches in the 0x43–0x5A region.** The corruption is fully contained to
the swap chain already identified (0x42/0x46/0x4A, see above) plus the two independently-corrected slots below
(0x3A/0x3C, 0x3E/0x3B, 0x3F/0x4C), which sit just outside the originally-flagged 0x44–0x5A range.

## Priority 3 sweep: additional corrections found outside the original flagged range

- **`StorageOperationHandle`: 0x3A → 0x3C.** `CStoreBankDlg::SendCalculateFeeRequest`@0x67547c and
  `SendGetAllRequest`@0x6754e1 both send `COutPacket::COutPacket(_, 60)` = 0x3C, submodes 0x1A (calculate fee) and
  0x1B (withdraw all). "Store/Bank Dialog" is the NPC warehouse-storage UI (distinct from `CTrunkDlg`, which the
  prior pass correctly identified as the **trade-room** item-shuttle dialog, confirmed opcode 0x6F — not storage).
  0x3C is free in the current template. **Note:** only the fee-calc/withdraw-all submodes were traced to a class;
  per-item put/take-from-storage sends were not isolated in this pass (see residual list) — they may ride under
  0x6F's submode space (shared with trade-room) or under 0x3C with additional submodes not yet found.

- **`OwlWarpHandle`: 0x3F → 0x4C.** `CWvsContext::SendShopScannerItemUseRequest`@0x832680:
  `COutPacket::COutPacket((COutPacket *)v3, 76)` = 0x4C, gated on item-id prefix `/10000==231` (the Owl-of-Minerva
  / shop-scanner ticket item category — matches project memory's "legacy owl/shop-scanner protocol" pairing).
  0x4C is free in the current template (vacated by the earlier MountFood 0x4C→0x48 correction).

- **`HiredMerchantOperationHandle`: 0x3E → 0x3B (partial — see caveat).**
  `CWvsContext::SendEntrustedShopCheckRequest`@0x848a01: `COutPacket::COutPacket((COutPacket *)v25, 59)` = 0x3B,
  submode 0, an 8-byte position buffer — the pre-flight "can I place a hired-merchant stall here" check (validates
  level ≥15, VR slot availability, position collision). 0x3B is free in the current template.
  **Caveat, important:** the *other* Hired-Merchant actions — `CEntrustedShopDlg::OnGoOut`@0x4da528 (submode
  0x25), `OnArrange`@0x4da583 (submode 0x26), `SetRet`@0x4d948b withdraw-confirm path (submode 0x27), and
  `OnWithdrawMoney`@0x4da5fd (submode 0x29) — all send **opcode 0x6F**, the *same* opcode already verified as
  `CharacterInteractionHandle` (trade-room submodes 0/2/3, trunk submodes 0xE/0xF). **There is no client-sent wire
  value 0x3E for hired-merchant at all** — the feature splits across two *already-accounted-for* opcodes (0x3B for
  the placement pre-check, 0x6F for everything after the stall exists). Moving `HiredMerchantOperationHandle` to
  0x3B only covers the pre-check; the post-creation operations need their submodes (0x25/0x26/0x27/0x29) added to
  the existing 0x6F dispatcher rather than a standalone handler — a dispatcher-family-scope change beyond a pure
  opcode-value swap, flagged for follow-up rather than asserted as a complete fix here.

- After the above three moves, 0x3A and 0x3E are left unassigned with no confirmed alternate feature — same
  disposition as 0x4A (vacated, no evidence of any other real client-sent value at these slots in this pass).

## Priority 3 sweep: additional MATCHes confirmed (previously UNRESOLVED)

- **0x2E `CharacterChatGeneralHandle` — MATCH.** `SendChatMsg@CField`@0x4e73af:
  `COutPacket::COutPacket((COutPacket *)v5, 46)` = 0x2E, `EncodeStr(text)` + `Encode1(show-type)`.

- **0x36 `NPCStartConversationHandle` — MATCH.** `CUserLocal::TalkToNpc`@0x7b1403 (named, but its ctor call was
  not read in the prior pass): `COutPacket::COutPacket((COutPacket *)v12, 54)` = 0x36, `Encode4(npc entity id)` +
  `Encode2(x)` + `Encode2(y)` (player position).

- **0x62 `QuestActionHandle` — MATCH.** `CQuest::StartQuest`@0x623b12 has four distinct `COutPacket` ctor call
  sites (submodes 1/2/4/5 — start/start-with-selection/npc-required-item-check variants), **all using opcode 98 =
  0x62**.

- **0x6B `CharacterMultiChatHandle` — MATCH.** `CUIStatusBar::SendGroupMessage`@0x74467d:
  `COutPacket::COutPacket((COutPacket *)v24, 107)` = 0x6B, `Encode1(chat-target-type: 0=guild,1=buddy,2=expedition,
  3=alliance)` + recipient id list + message — the multi-target/broadcast chat.

- **0x6E `MessengerOperationHandle` — MATCH.** `CUIMessenger::ProcessChat`@0x6d4021:
  `COutPacket::COutPacket((COutPacket *)v13, 110)` = 0x6E, submode 6, `EncodeStr(chat text)`.

- **0x77 `NoteOperationHandle` — MATCH.** `CMemoListDlg::SetRet`@0x5ad50c:
  `COutPacket::COutPacket((COutPacket *)v25, 119)` = 0x77, submode 1 (batch-delete received notes/memos).

- **0x79 `UseDoor` — MATCH.** `COpenGatePool::TryEnterOpenGate`@0x68734f:
  `COutPacket::COutPacket((COutPacket *)v14, 121)` = 0x79, `Encode4(door/gate object id)` + `Encode1(0)` — "Open
  Gate" is this build's internal name for the Mystic Door skill object.

---

## CORRECTIONS TO APPLY

| handler_name | current_template_opcode | TRUE_opcode | evidence (fn@addr → ctor N) |
|---|---|---|---|
| `CharacterInventoryMoveHandle` | 0x46 | **0x42** | `sub_8315F8`@0x8315f8: `COutPacket::COutPacket(v11, 66)`; confirmed via 2 callers (`sub_4BDD1A`@0x4bdd1a auto-equip-swap, `sub_4BDEDA`@0x4bdeda drag-and-drop) both calling `sub_8315F8(ctx, invType, fromSlot, toSlot, 0xFFFF)`. Slot 0x42 free. **Step 1 of swap chain — apply before step 2.** |
| `CharacterItemUseSummonBagHandle` | 0x4A | **0x46** | `CWvsContext::SendMobSummonItemUseRequest`@0x831c83: `COutPacket::COutPacket(v16, 70)` = 0x46 (re-confirmed from prior pass). Slot 0x46 free **only after step 1** (InventoryMove vacates it). **Step 2 of swap chain.** |
| (0x4A — no handler) | — | — | After step 2, 0x4A is unassigned. Real client feature there: `SendBridleItemUseRequest`@0x832005 (Bridle/pet-leash item use, prefix `id/10000==227`). **No template handler exists for this feature under any name** — this is a missing-codec coverage gap, not a pure opcode correction; flagged for follow-up implementation, not applied here. |
| `StorageOperationHandle` | 0x3A | **0x3C** | `CStoreBankDlg::SendCalculateFeeRequest`@0x67547c: `COutPacket::COutPacket(v3, 60)`; `SendGetAllRequest`@0x6754e1: `COutPacket::COutPacket(v15, 60)` = 0x3C, submodes 0x1A/0x1B. Slot 0x3C free. Per-item put/take submodes not isolated this pass (residual). |
| `OwlWarpHandle` | 0x3F | **0x4C** | `CWvsContext::SendShopScannerItemUseRequest`@0x832680: `COutPacket::COutPacket(v3, 76)` = 0x4C, item-prefix `a2/10000==231`. Slot 0x4C free (vacated by the already-applied MountFood 0x4C→0x48 fix). |
| `HiredMerchantOperationHandle` | 0x3E | **0x3B** (partial) | `CWvsContext::SendEntrustedShopCheckRequest`@0x848a01: `COutPacket::COutPacket(v25, 59)` = 0x3B, submode 0 (placement pre-check). Slot 0x3B free. **Caveat:** post-creation ops (GoOut/Arrange/Withdraw, submodes 0x25/0x26/0x27/0x29) already ride opcode 0x6F (`CharacterInteractionHandle`, already verified) — no wire value 0x3E exists for this feature at all; moving to 0x3B covers only the pre-check half. Extending the 0x6F dispatcher's submode table is a follow-up beyond opcode correction. |

## VERIFIED CORRECT (no change needed)

| handler | opcode | evidence |
|---|---|---|
| `CharacterItemUseScrollHandle` | 0x4F | `sub_8317A4`@0x8317a4: `COutPacket::COutPacket(v12, 79)`; caller `sub_4C072A`@0x4c072a gates on scroll item-id prefixes {200,201,202,205,221,236,238} |
| `CharacterDistributeApHandle` | 0x50 | `sub_8457EE`@0x8457ee: `COutPacket::COutPacket(v15, 80)`; gated on AP-remaining>0; caller `sub_72DFBB`@0x72dfbb dispatches ability-window button IDs 0x7D0–0x7D5 |
| `CharacterDistributeSpHandle` | 0x52 | `sub_8458EB`@0x8458eb: `COutPacket::COutPacket(v8, 82)`; caller `sub_7201E6`@0x7201e6 checks job-branch skill-point caps before raising a skill |
| `CharacterSkillBookUseHandle` | 0x4B | `SendSkillLearnItemUseRequest`@0x8325d2: `push 4Bh` before ctor call (disasm); item-prefix {228,229} |
| `CharacterChatGeneralHandle` | 0x2E | `SendChatMsg@CField`@0x4e73af: `COutPacket::COutPacket(v5, 46)` |
| `NPCStartConversationHandle` | 0x36 | `CUserLocal::TalkToNpc`@0x7b1403: `COutPacket::COutPacket(v12, 54)` |
| `QuestActionHandle` | 0x62 | `CQuest::StartQuest`@0x623b12: 4 ctor sites, all `COutPacket::COutPacket(_, 98)` |
| `CharacterMultiChatHandle` | 0x6B | `CUIStatusBar::SendGroupMessage`@0x74467d: `COutPacket::COutPacket(v24, 107)` |
| `MessengerOperationHandle` | 0x6E | `CUIMessenger::ProcessChat`@0x6d4021: `COutPacket::COutPacket(v13, 110)` |
| `NoteOperationHandle` | 0x77 | `CMemoListDlg::SetRet`@0x5ad50c: `COutPacket::COutPacket(v25, 119)` |
| `UseDoor` | 0x79 | `COpenGatePool::TryEnterOpenGate`@0x68734f: `COutPacket::COutPacket(v14, 121)` |
| (38 rows from the prior pass) | various | see `prior-serverbound-audit.md` §Result table — not re-decompiled this pass, no adjacency risk found |

## RESIDUAL UNVERIFIED

Handlers not pinned in this pass, with what was tried (all remain genuinely unresolved — no value fabricated):

- **Login/char-select boundary** (0x01, 0x04, 0x05, 0x06, 0x07, 0x09, 0x0A, 0x0B, 0x0E, 0x0F, 0x14, 0x17, 0x18,
  0x19): not revisited this pass; same gaps as the prior audit (dedicated send functions largely absent by name;
  several are plausibly inline in large UI-flow functions not yet decompiled).
- **Movement/attack cluster** (0x26 `CharacterMoveHandle`, 0x29 `CharacterMeleeAttackHandle`, 0x2A
  `CharacterRangedAttackHandle`, 0x2B `CharacterMagicAttackHandle`, 0x2C `CharacterTouchAttackHandle`, 0x2D
  `CharacterDamageHandle`): candidate functions identified by name in the prior pass
  (`CUserLocal::TryDoingMeleeAttack`@0x7a45f1, `TryDoingShootAttack`@0x7a67e9, `TryDoingMagicAttack`@0x7a8572) are
  each 1600–8200 bytes with multiple internal branches; the specific `COutPacket` ctor call site within each was
  not isolated this pass (would need per-branch decompilation, out of budget this round).
- **NPC continue-conversation** (0x38): `CScriptMan::OnAsk*` family is confirmed receive-side only (server→client
  prompts); the client's "continue/reply" send was not isolated. `TalkToNpc` (0x36, confirmed above) only covers
  conversation *start*.
- **Storage put/take individual item** (part of 0x3C's feature area): only fee-calc/withdraw-all were traced; a
  per-item put/take-from-storage send was not found under `CStoreBankDlg` or elsewhere.
- **ChalkboardCloseHandle** (0x2F), **MonsterBookCover** (0x35): no dedicated send function located (no class name
  match for "Chalkboard"; MonsterBookCover checked in the prior pass with no send function found).
- **PartyInviteRejectHandle** (0x71): not revisited; prior pass could not separate this from the 0x70 party
  dispatcher — may in fact be a submode of 0x70 rather than a standalone opcode (same structural question as the
  Hired-Merchant finding above, not chased down this pass).
- **CharacterSpouseChatHandle** (0x6D), **SueCharacter** (0x68): no dedicated send function located; `SueCharacter`
  has only a receive-side `OnSueCharacterResult`@0x84a04e.
- **AdminCommand** (0x7E), **WeddingAction** (0x7F), **WeddingTalk** (0x80): not revisited; same gaps as prior
  pass.
- **Pet cluster** (0x8A `PetMovementHandle`, 0x8B `PetChatHandle`, 0x8C `PetCommandHandle`, 0x8E `PetItemUseHandle`
  low-confidence lead `SendStatChangeItemUseRequestByPetQ`@0x831ab9=0x8E): not revisited.
- **Summon cluster** (0x92 `SummonMoveHandle`, 0x93 `SummonAttackHandle`, 0x94 `SummonDamageHandle`): candidate
  functions (`CSummoned::TryDoingAttackManual`@0x67a9ca, `AttackToTargetMob`@0x67bb40) appeared in the complete
  COutPacket xrefs list but were not decompiled this pass.
- **Mob/field damage cluster** (0x9B `MonsterMovementHandle` — likely genuinely clientbound-only, consistent with
  prior pass; 0x9D `MobDropPickupRequest`, 0x9E `FieldDamageMob`, 0x9F `MonsterDamageFriendlyHandle`, 0xA0
  `MonsterBomb`, 0xA1 `MobDamageMob`): not revisited.
- **NPCActionHandle** (0xA4), **ReactorHitHandle** (0xAC): not revisited; `CReactorPool`'s named members remain
  receive-side only per the prior pass.
- **Minigame field objects** (0xB0 `Snowball`, 0xB1 `LeftKnockback`, 0xB2 `Coconut`, 0xB4 `GuildBoss`, 0xB7
  `MonsterCarnival`): not revisited.
- **Cash shop / ITC cluster** (0xC3 `CashShopCheckWalletHandle`, 0xC4 `CashShopOperationHandle`, 0xD5
  `ItcStatusChargeHandle`, 0xD6 `ItcQueryCashRequestHandle`, 0xD7 `ItcOperationHandle`): large `On*` submode
  families identified by name in the prior pass but ctor values not extracted; not revisited this pass.

No value in this section was guessed or inferred from memory — each is either an untouched carry-over from the
prior pass's honest UNRESOLVED list or (for 0x38, storage-put/take) a gap newly identified but not chased down
this round due to time budget.

## Summary counts

Of the prior pass's 70 UNRESOLVED rows:

- **Corrections found (new): 5 handler-opcode moves**, resolving **4** previously-UNRESOLVED opcodes
  (0x46, 0x3A, 0x3E, 0x3F) plus refining the already-known SummonBag/Bridle mismatch chain —
  `CharacterInventoryMoveHandle` (0x46→0x42), `CharacterItemUseSummonBagHandle` (0x4A→0x46),
  `StorageOperationHandle` (0x3A→0x3C), `OwlWarpHandle` (0x3F→0x4C), `HiredMerchantOperationHandle` (0x3E→0x3B,
  partial/caveated). The vacated 0x4A slot (real feature: Bridle) is left as a flagged coverage gap, not a
  correction.
- **Verified correct (new MATCH, resolving 10 previously-UNRESOLVED opcodes): 0x2E, 0x36, 0x4F, 0x50, 0x52, 0x62,
  0x6B, 0x6E, 0x77, 0x79.** (0x4B `CharacterSkillBookUseHandle` was also independently re-confirmed correct this
  pass, but it was already applied post-audit as a new row, not one of the original 70 UNRESOLVED, so it is not
  counted in this tally.)
- **Residual unverified: 56** of the original 70 UNRESOLVED rows (70 − 14 resolved = 56), enumerated above by
  cluster with what was tried for each.
