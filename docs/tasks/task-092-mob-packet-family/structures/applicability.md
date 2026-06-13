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
| RESET_MONSTER_ANIMATION (CMob::OnSuspendReset) | implement | implement | implement | implement | implement |
| MOB_AFFECTED (CMob::OnAffected) | implement | implement | implement | implement | implement |
| MONSTER_SPECIAL_EFFECT_BY_SKILL (CMob::OnSpecialEffectBySkill) | implement | implement | implement | implement | implement |
| MOB_CRC_KEY_CHANGED (CMobPool::OnMobCrcKeyChanged) | implement | implement | implement | implement | implement |
| CATCH_MONSTER (CMob::OnCatchEffect) | implement | implement | implement | implement | implement² |
| CATCH_MONSTER_WITH_ITEM (CMob::OnEffectByItem) | implement | implement | implement | implement | implement |
| MOB_SPEAKING (CMob::OnMobSpeaking) | implement | implement | implement | implement | VERSION-ABSENT |
| INC_MOB_CHARGE_COUNT (CMob::OnIncMobChargeCount) | implement | implement | implement | implement | VERSION-ABSENT |
| MOB_SKILL_DELAY (CMob::OnMobSkillDelay) | VERSION-ABSENT³ | implement | implement | implement | VERSION-ABSENT |
| SET_TAMING_MOB_INFO (CWvsContext::OnSetTamingMobInfo) | implement | implement | implement | implement | implement |
| BRIDLE_MOB_CATCH_FAIL (CWvsContext::OnBridleMobCatchFail) | implement | implement | implement | implement | implement |
| MONSTER_BOOK_SET_CARD (CWvsContext::OnMonsterBookSetCard) | implement | implement | implement | implement | implement |
| MONSTER_BOOK_SET_COVER (CWvsContext::OnMonsterBookSetCover) | implement | implement | implement | implement | implement |
| MONSTER_CARNIVAL_START (OnEnter) | implement | implement | implement | implement | implement |
| MONSTER_CARNIVAL_OBTAINED_CP (OnPersonalCP) | implement | implement | implement | implement | implement |
| MONSTER_CARNIVAL_PARTY_CP (OnTeamCP) | implement | implement | implement | implement | implement |
| MONSTER_CARNIVAL_SUMMON (OnRequestResult, a2≠0) | implement | implement | implement | implement | implement |
| MONSTER_CARNIVAL_MESSAGE (OnRequestResult, a2=0) | implement | implement | implement | implement | implement |
| MONSTER_CARNIVAL_DIED (OnProcessForDeath) | implement | implement | implement | implement | implement |
| MONSTER_CARNIVAL_LEAVE (OnShowMemberOutMsg) | implement | implement | implement | implement | implement |
| MONSTER_CARNIVAL_RESULT (OnShowGameResult) | implement | implement | implement | implement | implement |
| MOB_ESCORT_FULL_PATH (CMob::OnEscortFullPath) | VERSION-ABSENT | VERSION-ABSENT | VERSION-ABSENT⁴ | implement | VERSION-ABSENT |
| MOB_ESCORT_STOP (CMob::OnEscortStopEndPermmision) | VERSION-ABSENT | VERSION-ABSENT | VERSION-ABSENT | implement⁵ | VERSION-ABSENT |
| MOB_ESCORT_STOP_SAY (CMob::OnEscortStopSay) | VERSION-ABSENT | VERSION-ABSENT | VERSION-ABSENT | implement | VERSION-ABSENT |
| MOB_ESCORT_RETURN_BEFORE (CMob::OnEscortReturnBefore) | VERSION-ABSENT | VERSION-ABSENT | VERSION-ABSENT | implement | VERSION-ABSENT |
| MOB_NEXT_ATTACK (CMob::OnNextAttack) | VERSION-ABSENT | VERSION-ABSENT | VERSION-ABSENT | implement | VERSION-ABSENT |
| MOB_ATTACKED_BY_MOB (CMob::OnMobAttackedByMob) | VERSION-ABSENT | VERSION-ABSENT | VERSION-ABSENT | implement | VERSION-ABSENT |

## Serverbound

| Op | v83 | v84 | v87 | v95 | jms185 |
|---|---|---|---|---|---|
| TOUCH_MONSTER_ATTACK (CUserLocal::TryDoingBodyAttack) | implement⁸ | implement¹ | implement | implement | NAMED⁹ |
| MOB_BANISH_PLAYER (CUserLocal::SendBanMapByMobRequest) | IDB-UNRESOLVED⁶ | implement¹ | implement | implement | implement |
| MONSTER_BOOK_COVER (CUserLocal::SetMonsterBookCover) | implement⁷ | implement¹⁷ | implement⁷ | implement⁷ | implement⁷ |
| MOB_CRC_KEY_CHANGED_REPLY (CMobPool::OnMobCrcKeyChanged) | implement | implement | implement | implement | implement |
| MOB_DROP_PICKUP_REQUEST (CMob::SendDropPickUpRequest) | implement | implement¹ | implement | implement | implement |
| FIELD_DAMAGE_MOB (CMob::Update, shared) | implement | implement | implement | implement | implement |
| MOB_DAMAGE_MOB_FRIENDLY (CMob::Update, shared) | implement | implement | implement | implement | implement |
| MONSTER_BOMB (CMob::TryFirstSelfDestruction) | implement | implement¹ | implement | implement | implement |
| MOB_DAMAGE_MOB (CMob::SetDamagedByMob) | implement | implement | implement | implement | implement |
| MOB_SKILL_DELAY_END (CMob::Update, shared) | implement | implement | implement | implement | implement |
| MOB_TIME_BOMB_END (CMob::UpdateTimeBomb) | IDB-UNRESOLVED⁶ | implement¹ | IDB-UNRESOLVED⁶ | implement | implement |
| MOB_ESCORT_COLLISION (CMob::SendCollisionEscort) | VERSION-ABSENT | VERSION-ABSENT | VERSION-ABSENT¹⁰ | implement | implement |
| MONSTER_CARNIVAL (CUIMonsterCarnival::RequestSend) | implement | implement | implement | implement | implement |
| MOB_REQUEST_ESCORT_INFO (CMob::SendRequestEscortPath) | VERSION-ABSENT | VERSION-ABSENT | VERSION-ABSENT | implement | implement |
| MOB_ESCORT_STOP_END_REQUEST (CMob::SendEscortStopEndRequest) | VERSION-ABSENT | VERSION-ABSENT | VERSION-ABSENT | implement | implement |

## Footnotes

1. **v84 NAMED + HARVESTED 2026-06-13** (was the unnamed-placeholder footnote). The v84 IDB
   in-scope family was renamed by layout-match to its v83 twins and re-harvested: 22 resolved /
   0 unresolved, merged into `gms_v84.json` (440→464 keys). Cells WITHOUT a ¹ are now backed
   by a real v84 fname pin. Cells that KEEP ¹ are the residual serverbound senders still
   unnamed in the v84 IDB (no anchor symbol) — those stay v83-equivalent (v84 ≡ v83 wire,
   v83 codec path via the `MajorAtLeast(87)` gate). **v84 ≠ v83 on MOB_SKILL_DELAY (cb):**
   v84 HAS the handler (dispatcher case 261), v83 does not. See `gms_v84.md`.
2. **jms CATCH_MONSTER** dispatches via unnamed `sub_6EAE5F` → `CMob::ShowCatchEffect`
   @0x6E5F77 (in the export). Pin against ShowCatchEffect or the sub address; layout = 1×Decode1.
3. **v83 MOB_SKILL_DELAY = VERSION-ABSENT (corrected 2026-06-13).** The v83 `CMobPool::OnMobPacket`
   dispatcher (@0x67936D) switch ENDS at case 0xFF (OnMobAttackedByMob) — there is no
   skill-delay case. The v95 dispatcher introduces OnMobSkillDelay at case 303 (between
   OnIncMobChargeCount and the escort family). So v83 has no clientbound MOB_SKILL_DELAY
   handler at all: it is a later-version feature, NOT merely an unnamed symbol. v84/v87/v95
   DO have it (4×Decode4); v84 at dispatcher case 261. jms is version-absent (no GMS cluster).
4. **v87 MOB_ESCORT_FULL_PATH**: 0x111 is NOT dispatched (CMobPool::OnMobPacket cases end at
   0x110) and no `CMob::OnEscortFullPath` symbol exists → genuine VERSION-ABSENT in v87.
   Only v95 implements this op.
5. **v95 escort naming**: plan's MOB_ESCORT_RETURN_STOP / _STOP_SAY ARE the registry's
   MOB_ESCORT_STOP / MOB_ESCORT_STOP_SAY (OnEscortStopEndPermmision / OnEscortStopSay).
   MOB_ESCORT_STOP reads nothing (handler is `QAEXXZ`, no CInPacket).
6. **IDB-UNRESOLVED (post-2026-06-13 residual)**: real op, send-site not a discrete named
   function in this IDB. Remaining after this naming pass:
   - **v83 MOB_BANISH_PLAYER** (SendBanMapByMobRequest): the v95 standalone is a 0x77-byte
     one-Encode4 wrapper called only from Update@CUserLocal; v83 Update@CUserLocal builds its
     COutPacket inline — the send is **inlined**, not a separate sub. Left unnamed (no guess).
   - **v83 / v87 MOB_TIME_BOMB_END** (UpdateTimeBomb): the v95 standalone 0x155-byte private
     CMob method is **inlined into Update@CMob** in v83 and v87 (the standalone helper is a
     later-build refactor). Verified by checking every GetBodyRect-calling unnamed CMob sub.
   Stage 2 pins from a sibling version's byte-equivalent layout (v95) rather than fabricating
   a per-version fname.
7. **MONSTER_BOOK_COVER fname newly set this stage** to `CUserLocal::SetMonsterBookCover`
   (ida-discovered) in all 5 registries. Resolved + in-export for v83/v87/v95/jms. v84's
   send-site is unnamed (footnote 1) — fname left empty there; inherit v83.
8. **v83 TOUCH_MONSTER_ATTACK RESOLVED 2026-06-13**: was unnamed `sub_9581A9`; renamed to
   `CUserLocal::TryDoingBodyAttack` (layout-matched to v95 by callee fingerprint) + harvested
   (1 resolved / 0 unresolved). COutPacket opcode 0x30. See `gms_v83.md`.
9. **jms TOUCH_MONSTER_ATTACK = NAMED-but-not-auto-harvestable**: `sub_A2AB71` renamed to
   `CUserLocal::TryDoingBodyAttack` (byte-confirmed: called twice from Update@CUserLocal after
   FindBodyAttackMob; body-attack callee + stack-frame fingerprint). The symbol now resolves
   by address, BUT jms Hex-Rays FAILS to decompile this 0x1a3a-byte function, so the export
   harvester can't capture its layout (no JSON entry merged). Stage 2 inherits the v83/v95
   TOUCH_MONSTER_ATTACK layout. See `jms_v185.md`.
10. **v87 MOB_ESCORT_COLLISION = VERSION-ABSENT (corrected 2026-06-13)**: the escort family is
    absent in v87 — the dispatcher ends at OnMobAttackedByMob/0x110 with no escort cases, and
    there is no `CVecCtrlMob::CollisionDetectEscortDest` (the v95 caller of SendCollisionEscort)
    nor any `Escort` symbol. Escort post-dates v87. Was mislabelled IDB-UNRESOLVED.

## Summary counts

- Fully implementable + resolved (codec + pin in Stage 2): all carnival (8 ops), the
  monster-book/taming/bridle cluster, the mob clientbound effect cluster, the escort
  family on v95/jms, and (newly this stage) the full v84 in-scope handler family + v84
  Update/SetDamagedByMob/RequestSend, and v83 TOUCH_MONSTER_ATTACK.
- VERSION-ABSENT cells: the GMS clientbound escort/next-attack family on v83/v84/v87/jms;
  MOB_SPEAKING/INC_MOB_CHARGE_COUNT/MOB_SKILL_DELAY on jms; **MOB_SKILL_DELAY (cb) on v83**
  (corrected — no dispatcher case); MOB_ESCORT_FULL_PATH and **MOB_ESCORT_COLLISION on v87**
  (corrected); the serverbound escort sends on v83/v84/v87.
- IDB-UNRESOLVED residual (Stage-2 pins from the v95 byte-equivalent, do not fabricate):
  v83 {MOB_BANISH_PLAYER (inlined), MOB_TIME_BOMB_END (inlined)},
  v87 {MOB_TIME_BOMB_END (inlined)},
  v84 {TryFirstSelfDestruction, SendDropPickUpRequest, UpdateTimeBomb, TryDoingBodyAttack,
  SendBanMapByMobRequest, SetMonsterBookCover — all unnamed in v84 IDB}.
- NAMED-not-harvestable: jms TOUCH_MONSTER_ATTACK (Hex-Rays decompile fails; symbol exists).
