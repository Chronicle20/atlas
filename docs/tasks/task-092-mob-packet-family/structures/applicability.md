# task-092 — (op × version) applicability grid (Stage 1 authoritative)

Derived from context.md §2 + what actually resolved/dispatched in each IDB during the
Stage-1 harvest (2026-06-13). Per cell:

- **implement** — op present in this version's registry AND its handler/send-site dispatches
  in the IDB; Stage 2 implements a codec + pins evidence.
- **VERSION-ABSENT** — op genuinely not in this version (no registry row and/or no IDB
  dispatch). Stage 2 records `VERSION-ABSENT` evidence (n/a), no codec.
- **IDB-UNRESOLVED** — op IS present (real opcode/row) but its registry fname has no matching
  named symbol in this IDB (unnamed `sub_XXXX` send-site). NOT version-absent. Stage 2 must
  derive the real send-site before pinning; flagged as a blocker, never fabricated.

## Clientbound

| Op | v83 | v84 | v87 | v95 | jms185 |
|---|---|---|---|---|---|
| RESET_MONSTER_ANIMATION (CMob::OnSuspendReset) | implement | implement¹ | implement | implement | implement |
| MOB_AFFECTED (CMob::OnAffected) | implement | implement¹ | implement | implement | implement |
| MONSTER_SPECIAL_EFFECT_BY_SKILL (CMob::OnSpecialEffectBySkill) | implement | implement¹ | implement | implement | implement |
| MOB_CRC_KEY_CHANGED (CMobPool::OnMobCrcKeyChanged) | implement | implement | implement | implement | implement |
| CATCH_MONSTER (CMob::OnCatchEffect) | implement | implement¹ | implement | implement | implement² |
| CATCH_MONSTER_WITH_ITEM (CMob::OnEffectByItem) | implement | implement¹ | implement | implement | implement |
| MOB_SPEAKING (CMob::OnMobSpeaking) | implement | implement¹ | implement | implement | VERSION-ABSENT |
| INC_MOB_CHARGE_COUNT (CMob::OnIncMobChargeCount) | implement | implement¹ | implement | implement | VERSION-ABSENT |
| MOB_SKILL_DELAY (CMob::OnMobSkillDelay) | IDB-UNRESOLVED³ | implement¹ | implement | implement | VERSION-ABSENT |
| SET_TAMING_MOB_INFO (CWvsContext::OnSetTamingMobInfo) | implement | implement¹ | implement | implement | implement |
| BRIDLE_MOB_CATCH_FAIL (CWvsContext::OnBridleMobCatchFail) | implement | implement¹ | implement | implement | implement |
| MONSTER_BOOK_SET_CARD (CWvsContext::OnMonsterBookSetCard) | implement | implement¹ | implement | implement | implement |
| MONSTER_BOOK_SET_COVER (CWvsContext::OnMonsterBookSetCover) | implement | implement¹ | implement | implement | implement |
| MONSTER_CARNIVAL_START (OnEnter) | implement | implement¹ | implement | implement | implement |
| MONSTER_CARNIVAL_OBTAINED_CP (OnPersonalCP) | implement | implement¹ | implement | implement | implement |
| MONSTER_CARNIVAL_PARTY_CP (OnTeamCP) | implement | implement¹ | implement | implement | implement |
| MONSTER_CARNIVAL_SUMMON (OnRequestResult, a2≠0) | implement | implement¹ | implement | implement | implement |
| MONSTER_CARNIVAL_MESSAGE (OnRequestResult, a2=0) | implement | implement¹ | implement | implement | implement |
| MONSTER_CARNIVAL_DIED (OnProcessForDeath) | implement | implement¹ | implement | implement | implement |
| MONSTER_CARNIVAL_LEAVE (OnShowMemberOutMsg) | implement | implement¹ | implement | implement | implement |
| MONSTER_CARNIVAL_RESULT (OnShowGameResult) | implement | implement¹ | implement | implement | implement |
| MOB_ESCORT_FULL_PATH (CMob::OnEscortFullPath) | VERSION-ABSENT | VERSION-ABSENT | VERSION-ABSENT⁴ | implement | VERSION-ABSENT |
| MOB_ESCORT_STOP (CMob::OnEscortStopEndPermmision) | VERSION-ABSENT | VERSION-ABSENT | VERSION-ABSENT | implement⁵ | VERSION-ABSENT |
| MOB_ESCORT_STOP_SAY (CMob::OnEscortStopSay) | VERSION-ABSENT | VERSION-ABSENT | VERSION-ABSENT | implement | VERSION-ABSENT |
| MOB_ESCORT_RETURN_BEFORE (CMob::OnEscortReturnBefore) | VERSION-ABSENT | VERSION-ABSENT | VERSION-ABSENT | implement | VERSION-ABSENT |
| MOB_NEXT_ATTACK (CMob::OnNextAttack) | VERSION-ABSENT | VERSION-ABSENT | VERSION-ABSENT | implement | VERSION-ABSENT |
| MOB_ATTACKED_BY_MOB (CMob::OnMobAttackedByMob) | VERSION-ABSENT | VERSION-ABSENT | VERSION-ABSENT | implement | VERSION-ABSENT |

## Serverbound

| Op | v83 | v84 | v87 | v95 | jms185 |
|---|---|---|---|---|---|
| TOUCH_MONSTER_ATTACK (CUserLocal::TryDoingBodyAttack) | IDB-UNRESOLVED⁶ | implement¹ | implement | implement | IDB-UNRESOLVED⁶ |
| MOB_BANISH_PLAYER (CUserLocal::SendBanMapByMobRequest) | IDB-UNRESOLVED⁶ | implement¹ | implement | implement | implement |
| MONSTER_BOOK_COVER (CUserLocal::SetMonsterBookCover) | implement⁷ | implement¹⁷ | implement⁷ | implement⁷ | implement⁷ |
| MOB_CRC_KEY_CHANGED_REPLY (CMobPool::OnMobCrcKeyChanged) | implement | implement | implement | implement | implement |
| MOB_DROP_PICKUP_REQUEST (CMob::SendDropPickUpRequest) | implement | implement¹ | implement | implement | implement |
| FIELD_DAMAGE_MOB (CMob::Update, shared) | implement | implement¹ | implement | implement | implement |
| MOB_DAMAGE_MOB_FRIENDLY (CMob::Update, shared) | implement | implement¹ | implement | implement | implement |
| MONSTER_BOMB (CMob::TryFirstSelfDestruction) | implement | implement¹ | implement | implement | implement |
| MOB_DAMAGE_MOB (CMob::SetDamagedByMob) | implement | implement¹ | implement | implement | implement |
| MOB_SKILL_DELAY_END (CMob::Update, shared) | implement | implement¹ | implement | implement | implement |
| MOB_TIME_BOMB_END (CMob::UpdateTimeBomb) | IDB-UNRESOLVED⁶ | implement¹ | IDB-UNRESOLVED⁶ | implement | implement |
| MOB_ESCORT_COLLISION (CMob::SendCollisionEscort) | VERSION-ABSENT | VERSION-ABSENT | IDB-UNRESOLVED⁶ | implement | implement |
| MONSTER_CARNIVAL (CUIMonsterCarnival::RequestSend) | implement | implement¹ | implement | implement | implement |
| MOB_REQUEST_ESCORT_INFO (CMob::SendRequestEscortPath) | VERSION-ABSENT | VERSION-ABSENT | VERSION-ABSENT | implement | implement |
| MOB_ESCORT_STOP_END_REQUEST (CMob::SendEscortStopEndRequest) | VERSION-ABSENT | VERSION-ABSENT | VERSION-ABSENT | implement | implement |

## Footnotes

1. **v84 = "implement" means v83-equivalent.** The v84 IDB symbolizes this family as
   unnamed `sub_XXXX`; only `CMobPool::OnMobCrcKeyChanged` resolved by fname (1/25). v84 is
   byte-identical to v83 (task-083) and takes the v83 codec path (`MajorAtLeast(87)` gate),
   so Stage 2 reuses the v83 layouts. The v84 export was NOT enriched for these ops (the
   fnames are unnamed in the IDB); v84 evidence should reference v83-equivalence rather than
   a v84 fname pin. See `gms_v84.md`.
2. **jms CATCH_MONSTER** dispatches via unnamed `sub_6EAE5F` → `CMob::ShowCatchEffect`
   @0x6E5F77 (in the export). Pin against ShowCatchEffect or the sub address; layout = 1×Decode1.
3. **v83 MOB_SKILL_DELAY**: registry fname `CMob::OnMobSkillDelay` (csv-import) has NO named
   symbol in the v83 IDB — the dispatcher's named cases stop at 0xFF (OnMobAttackedByMob).
   The op row exists (opcode 256) but is unpinnable until the real handler is derived. NOT
   version-absent. v87/v95 DO have OnMobSkillDelay named (4×Decode4).
4. **v87 MOB_ESCORT_FULL_PATH**: 0x111 is NOT dispatched (CMobPool::OnMobPacket cases end at
   0x110) and no `CMob::OnEscortFullPath` symbol exists → genuine VERSION-ABSENT in v87.
   Only v95 implements this op.
5. **v95 escort naming**: plan's MOB_ESCORT_RETURN_STOP / _STOP_SAY ARE the registry's
   MOB_ESCORT_STOP / MOB_ESCORT_STOP_SAY (OnEscortStopEndPermmision / OnEscortStopSay).
   MOB_ESCORT_STOP reads nothing (handler is `QAEXXZ`, no CInPacket).
6. **IDB-UNRESOLVED**: real op, but the registry's conceptual csv-import fname has no named
   symbol in this IDB (unnamed send-site). v83: TryDoingBodyAttack / SendBanMapByMobRequest /
   UpdateTimeBomb. v87: UpdateTimeBomb / SendCollisionEscort. jms: TryDoingBodyAttack. These
   are present in OTHER versions' IDBs (e.g. TryDoingBodyAttack named in v87/v95), so the
   symbol exists in the binary family — the specific IDB just didn't name it. Stage 2 derives
   the send-site or pins from a sibling version's layout (byte-equivalent).
7. **MONSTER_BOOK_COVER fname newly set this stage** to `CUserLocal::SetMonsterBookCover`
   (ida-discovered) in all 5 registries. Resolved + in-export for v83/v87/v95/jms. v84's
   send-site is unnamed (footnote 1) — fname left empty there; inherit v83.

## Summary counts

- Fully implementable + resolved (codec + pin in Stage 2): all carnival (8 ops), the
  monster-book/taming/bridle cluster, the mob clientbound effect cluster, and the escort
  family on v95/jms where named.
- VERSION-ABSENT cells: the GMS clientbound escort/next-attack family on v83/v84/v87/jms;
  MOB_SPEAKING/INC/SKILL_DELAY on jms; MOB_ESCORT_FULL_PATH on v87; the serverbound escort
  sends on v83/v84/v87 (and partly v87).
- IDB-UNRESOLVED (Stage-2 must derive a send-site, do not fabricate): v83
  {TOUCH_MONSTER_ATTACK, MOB_BANISH_PLAYER, MOB_TIME_BOMB_END, MOB_SKILL_DELAY},
  v87 {MOB_TIME_BOMB_END, MOB_ESCORT_COLLISION}, jms {TOUCH_MONSTER_ATTACK}.
