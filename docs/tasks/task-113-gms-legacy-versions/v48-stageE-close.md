# v48 Stage E — Close Reconciliation

**Final state:** gms_v48 verified **165** (52.9% of in-scope non-⬜), 🟡 2, ❌ 145, 🟥 0. `matrix --check` exit 0, 0 problem lines, 0 conflicts. All existing versions frozen throughout: v83 367 / v84 345 / v87 379 / v95 399 / jms 362 / v72 216 / v79 228 / v61 208.

Started the campaign at 0 verified (Stage D wire-up). Every genuinely-producible in-scope tier-1 cell (where the v61 **and** v83 anchors are verified) is now ✅ or a documented n-a disposition. This mirrors the v61/v72/v79 close state: the residual ❌ are not v48 regressions or unaddressed gaps — they are the two categories below.

## Category 1 — v48-absent arms, dispositioned n-a (render ❌ only because the matrix tool has no sub-struct n-a state)

Recorded in `docs/packets/audits/gms_v48/_unimplemented.json` with complete-enumeration IDA evidence. These cap their parent dispatcher op-cell exactly as the same arms cap it in the v61 anchor (so the op-cell is NOT a v48 regression):

- **Inventory:** ITEM_SORT / ITEM_SORT2 (gather/sort) — v48 has no 0x831xxx send region; complete `xrefs_to sub_4A2518` (500ms throttle, 72 refs) enumeration finds no compartment-guarded gather/sort body. Feature is post-v48.
- **Cash (CASHSHOP_OPERATION arms):** NameChange, TransferWorld (rides separate COutPacket(20)), EnableEquipSlot (folded into friendship-ring buy), IncCharacterSlot, BuySlotInc, IncTrunk, MoveCashItem L↔S, RebateLocker — none exist as COutPacket(160) mode arms in the 2009-era v48 cash shop (complete send-fn list: OnBuy/OnBuyCouple/OnBuyPackage/OnGift/OnBuyFriendship/OnBuyNormal/OnSetWish).
- **NPC conversation:** SayImage, AskQuiz, AskSpeedQuiz, AskBoxText, AskSlideMenu — v48 dispatcher `sub_5B0AE4` switches msgType over ONLY cases 0–9; these arms don't exist.
- **Interaction:** merchant Add/Remove-BlackList — v48-absent blacklist arms (= v61).
- **Character:** AutoDistributeAp — v48 `SendAbilityUpRequest` (sub_71CD00) has only the single-stat overload; no ZArray<StatPair> auto-distribute (= v61).
- **Party:** ChangeLeader (party-boss handover) — v48 op94 carries only invite(4)/kick(5); leadership handover post-dates v48 (= v61). Guild: BoardAuthKeyUpdate — guild BBS absent in v48.
- **Monster carnival, storage asset ops, note send-ops** — v48-absent / older-protocol shapes.

## Category 2 — pre-existing cross-version shared gaps (v83 and/or v61 anchor also ❌; OUT of task-113 scope)

task-113's gate is "v48 reaches ANCHOR parity," not "fix pre-existing gaps in v83+." 37 residual cells are ❌ in v48 **and** in the v83/v61 reference (consistent with v72's documented "~31 cells ❌ in both v72 and v79 anchor"). Families: cash 9, character 4, party 4, note 3, storage 3, field 2, interaction 2, inventory 2, npc 2, pet 2, buddy 1, guild 1, monster 1, summon 1. These would need a separate cross-version dispatcher-family effort that also uplifts v83+.

## Cross-version corrections made DURING the v48 campaign (net-positive, anchors held)

- **NPC_TALK sb = oid+x+y** confirmed (v48 sub_568A2A) — corroborates the Phase-5 fix of the v72/v79 oid-only false-pass.
- **Messenger legacyAdd gate** was on a misidentified function; real v61 Add reads 6 fields → gate narrowed to `<=28`, v61 evidence re-pinned (v61 held 208).
- **Interaction v48 avatar==v83 false-pass** corrected (now the gated single-int-pet AvatarLook).
- **SHOW_STATUS_INFO** "off-by-one" claim disproven by direct v61 decompile — template correctly left unchanged (a "fix" would have introduced a crash).
- **Guild ops table** SET_SKILL_RESPONSE 78→77 + phantom BOARD_AUTH_KEY_UPDATE dropped (IDA-verified: v48 OnGuildResult switch @0x7261EB handles modes 52–77 only; mode 78 unhandled). Commit c6184bb85e.

## Feature-sized shared-codec legacy branches added (all gated for the legacy range; all 8 anchors re-tested & held)

- **GW_CharacterStat / AvatarLook** legacy single-pet encoding (`< 61` / `>= 61` multi-pet).
- **CharacterTemporaryStat** legacy 8-byte int64 mask (local + foreign), bits 0–46 identical shift order (`< 61`).
- **Attack** ranged bulletX/Y omission, DamageInfo CRC, action-byte legacy gates.
- **CHANGE_MAP** chase byte, **GENERAL_CHAT** bOnlyBalloon, **FIELD_OBSTACLE** single-vs-list, **SPOUSE_CHAT**, **EffectWeather**, **IncreaseExperience**, **cash buyOmitsCurrency**, **ShopBuy discountPrice** — all legacy-gated, v61+ paths untouched.

## v28 boundary — FLAGGED FOR OWNER REVIEW

Several `< 61` legacy gates also catch GMS **v28** (`test.Variants[0]` in `libs/atlas-packet/test/context.go`) — a **test-only** variant (no `template_gms_28_1.json`, no IDB, round-trip-only). Controller decision (adopted, Close-C precedent): fold v28 into the v48 legacy wire (an older client shares v48's legacy shape far more plausibly than v83's modern shape; v28 had no verified wire either way), update v28 round-trip expectations, and comment each gate "unverified-by-inference (no v28 IDB)." Real-tenant impact is nil (v28 is not a shipping tenant). **Owner may override to v48-specific gating.**

## Follow-ups carried to Phase 5

1. NPC_TALK v72/v79 oid-only fixture false-pass (v48 confirmed oid+x+y) — fix the two anchors.
2. guild BBS_OPERATION / GUILD_OPERATION jms-fold matrix `lookupAnyVersion` tooling limitation (+98-row restructure) — separate tool task, shared with v61/v72/v79.
3. v28 boundary owner review (above).
4. NpcContinueConversation sb sub-row renders ❌ though the op is verified (matrix gap-fill artifact) — spot-check in review.
