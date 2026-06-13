# Stage 0 — Registry Gap Inventory (Task 0.3)

Phase-1 work list. Read 2026-06-13 from `docs/packets/registry/{gms_v83,gms_v84,gms_v87,gms_v95,jms_v185}.yaml`.
Records present/absent + recorded `direction`/`fname`/`provenance` per op per version, then the
fname mislabels & missing rows that Phase 1 (Stage 1, IDA harvest) must fix in the yamls.

Registry row counts: v83=577, v84=575, v87=614, v95=696, jms_v185=608.

---

## A. Per-op presence + recorded fname (44 op-names spanning the 42 ops)

`P=present`, `—=absent`. `dir`: cb=clientbound, sb=serverbound. fname shown where it
diverges per version (mislabels in **bold**); `""`=empty fname.

### Clientbound

| Op | dir | v83 | v84 | v87 | v95 | jms |
|---|---|---|---|---|---|---|
| RESET_MONSTER_ANIMATION | cb | P 244 | P 250 | P 260 | P 292 | P 261 | fname CMob::OnSuspendReset (all) |
| MOB_AFFECTED | cb | P 245 | P 245 | P 261 | P 293 | P 262 | CMob::OnAffected (all) |
| MONSTER_SPECIAL_EFFECT_BY_SKILL | cb | P 247 | P 247 | P 263 | P 295 | P 264 | CMob::OnSpecialEffectBySkill (all) |
| MOB_CRC_KEY_CHANGED | cb | P 249 | P 249 | P 265 | P 297 | P 266 | CMobPool::OnMobCrcKeyChanged (all) |
| CATCH_MONSTER | cb | P 251 | P 251 | P 267 | P 299 | P 268 | jms fname_alts:[sub_6EAE5F], provenance manual |
| CATCH_MONSTER_WITH_ITEM | cb | P 252 | P 252 | P 268 | P 300 | P 269 | CMob::OnEffectByItem (all) |
| MOB_SPEAKING ⚠ | cb | P 254 **OnIncMobChargeCount** | P 254 OnMobSpeaking | P 270 **OnIncMobChargeCount** | P 301 OnMobSpeaking | — (VERSION-ABSENT) |
| INC_MOB_CHARGE_COUNT ⚠ | cb | P 255 **OnMobAttackedByMob** | P 255 OnIncMobChargeCount | P 271 **OnMobSkillDelay** | P 302 OnIncMobChargeCount | — (VERSION-ABSENT) |
| MOB_SKILL_DELAY ⚠ | cb | P 256 OnMobSkillDelay | P 256 OnMobSkillDelay | P 272 **OnMobAttackedByMob** | P 303 OnMobSkillDelay | — (VERSION-ABSENT) |
| SET_TAMING_MOB_INFO | cb | P 48 | P 48 | P 48 | P 47 | P 45 | CWvsContext::OnSetTamingMobInfo (all) |
| BRIDLE_MOB_CATCH_FAIL | cb | P 79 | P 79 | P 81 | P 82 | P 73 | CWvsContext::OnBridleMobCatchFail (all) |
| MONSTER_BOOK_SET_CARD | cb | P 83 | P 83 | P 85 | P 86 | P 87 | CWvsContext::OnMonsterBookSetCard (all) |
| MONSTER_BOOK_SET_COVER | cb | P 84 | P 84 | P 86 | P 87 | P 88 | CWvsContext::OnMonsterBookSetCover (all) |
| MONSTER_CARNIVAL_START | cb | P 289 | P 289 | P 306 | P 346 | P 313 | CField_MonsterCarnival::OnEnter (all) |
| MONSTER_CARNIVAL_OBTAINED_CP | cb | P 290 | P 290 | P 307 | P 347 | P 314 | OnPersonalCP (all) |
| MONSTER_CARNIVAL_PARTY_CP | cb | P 291 | P 291 | P 308 | P 348 | P 315 | OnTeamCP (all) |
| MONSTER_CARNIVAL_SUMMON | cb | P 292 | P 292 | P 309 | P 349 | P 316 | OnRequestResult (shared w/ MESSAGE) |
| MONSTER_CARNIVAL_MESSAGE | cb | P 293 | P 293 | P 310 | P 350 | P 317 | OnRequestResult (shared w/ SUMMON) |
| MONSTER_CARNIVAL_DIED | cb | P 294 | P 294 | P 311 | P 351 | P 318 | OnProcessForDeath (all) |
| MONSTER_CARNIVAL_LEAVE | cb | P 295 | P 295 | P 312 | P 352 | P 319 | OnShowMemberOutMsg (all) |
| MONSTER_CARNIVAL_RESULT | cb | P 296 | P 296 | P 313 | P 353 | P 320 | OnShowGameResult (all) |
| MOB_ESCORT_FULL_PATH | cb | — | — | P 273 ⚠note | P 304 | — | v87 note: 0x111 NOT dispatched in v87 IDB, no symbol — likely CSV off-by-one tail; confirm/remove in Task 1.C |
| MOB_ESCORT_RETURN_BEFORE | cb | — | — | — | P 307 | — | CMob::OnEscortReturnBefore (v95 only) |
| MOB_NEXT_ATTACK | cb | — | — | — | P 308 | — | CMob::OnNextAttack (v95 only) |
| MOB_ATTACKED_BY_MOB | cb | — | — | — | P 309 | — | CMob::OnMobAttackedByMob (v95 only) |
| MOB_ESCORT_STOP | cb | — | — | — | P 305 | — | = plan's MOB_ESCORT_RETURN_STOP; fname CMob::OnEscortStopEndPermmision |
| MOB_ESCORT_STOP_SAY | cb | — | — | — | P 306 | — | = plan's MOB_ESCORT_RETURN_STOP_SAY; fname CMob::OnEscortStopSay |

### Serverbound

| Op | dir | v83 | v84 | v87 | v95 | jms |
|---|---|---|---|---|---|---|
| TOUCH_MONSTER_ATTACK | sb | P 47 | P 47 | P 49 | P 50 | P 38 | CUserLocal::TryDoingBodyAttack (all) |
| MOB_BANISH_PLAYER | sb | P 56 | P 56 | P 59 | P 61 | P 48 | CUserLocal::SendBanMapByMobRequest (all) |
| MONSTER_BOOK_COVER ⚠ | sb | P 57 `""` | P 57 `""` | P 60 `""` | P 62 `""` | P 49 `""` | **fname EMPTY in ALL 5** — gap §3.2 |
| MOB_CRC_KEY_CHANGED_REPLY | sb | P 164 | P 164 | P 174 | P 190 | P 158 | CMobPool::OnMobCrcKeyChanged (all) |
| MOB_DROP_PICKUP_REQUEST | sb | P 190 | P 190 | P 202 | P 229 | P 196 | CMob::SendDropPickUpRequest (all) |
| FIELD_DAMAGE_MOB | sb | P 191 | P 191 | P 203 | P 230 | P 197 | CMob::Update (shared 3-way) |
| MOB_DAMAGE_MOB_FRIENDLY | sb | P 192 | P 197 | P 204 | P 231 | P 198 | CMob::Update (shared 3-way) |
| MONSTER_BOMB | sb | P 193 | P 193 | P 205 | P 232 | P 199 | CMob::TryFirstSelfDestruction (all) |
| MOB_DAMAGE_MOB | sb | P 194 | P 199 | P 206 | P 233 | P 200 | CMob::SetDamagedByMob (all) |
| MOB_SKILL_DELAY_END | sb | P 195 | P 195 | P 207 | P 234 | P 201 | CMob::Update (shared 3-way) |
| MOB_TIME_BOMB_END | sb | P 196 | P 196 | P 208 | P 235 | P 202 | CMob::UpdateTimeBomb (all) |
| MOB_ESCORT_COLLISION | sb | — | — | P 209 | P 236 | P 203 | CMob::SendCollisionEscort (v87/v95/jms) |
| MONSTER_CARNIVAL | sb | P 218 | P 218 | P 231 | P 262 | P 229 | CUIMonsterCarnival::RequestSend (all) |
| MOB_REQUEST_ESCORT_INFO | sb | — | — | — | P 237 | P 204 | CMob::SendRequestEscortPath (v95/jms) |
| MOB_ESCORT_STOP_END_REQUEST | sb | — | — | — | P 238 | P 205 | CMob::SendEscortStopEndRequest (v95/jms) |

---

## B. Confirmed known issues (context.md §3) + verification result

### B.1 — fname mislabels: MOB_SPEAKING / INC_MOB_CHARGE_COUNT / MOB_SKILL_DELAY  ✅ CONFIRMED, worse than stated

context.md §3.1 said v83/v84/v87 are mislabeled. Actual registry state (verified):

- **v83**: MOB_SPEAKING→`CMob::OnIncMobChargeCount` (WRONG, provenance manual w/ realign
  note), INC_MOB_CHARGE_COUNT→`CMob::OnMobAttackedByMob` (WRONG, manual note),
  MOB_SKILL_DELAY→`CMob::OnMobSkillDelay` (looks correct but sits one slot below the two
  already-shifted rows — re-verify the whole 0xFE/0xFF/0x100 cluster against the v83 IDB
  dispatcher in Task 1.A.3; the existing "realigned to IDB" notes already point the first
  two rows at Inc/Attacked, implying the intended labels are MOB_SPEAKING=OnMobSpeaking,
  INC=OnIncMobChargeCount, SKILL_DELAY=OnMobSkillDelay).
- **v84**: all three CORRECT (OnMobSpeaking / OnIncMobChargeCount / OnMobSkillDelay) —
  fixed by an earlier discover-ops pass. **No Phase-1 fix needed for v84.**
- **v87**: three-way rotation, ALL WRONG: MOB_SPEAKING→`OnIncMobChargeCount`,
  INC_MOB_CHARGE_COUNT→`OnMobSkillDelay`, MOB_SKILL_DELAY→`OnMobAttackedByMob`
  (all provenance manual w/ "realigned to IDB" notes that are themselves the mislabel).
  Plan Task 1.C.3 calls out only the MOB_SKILL_DELAY fix; **also fix MOB_SPEAKING and
  INC_MOB_CHARGE_COUNT in v87** — the whole cluster is rotated.
- **v95**: all three CORRECT (csv-import). **No Phase-1 fix needed for v95.**
- **jms**: all three VERSION-ABSENT (not in registry). Correct per context.md §2.

**Phase-1 action:** in Task 1.A (v83) and Task 1.C (v87), decompile `CMobPool::OnMobPacket`
case labels and set each row's fname to the actually-dispatched handler; `provenance: manual`
with the IDA case→target citation in `note`. Target intended labels:
`MOB_SPEAKING=CMob::OnMobSpeaking`, `INC_MOB_CHARGE_COUNT=CMob::OnIncMobChargeCount`,
`MOB_SKILL_DELAY=CMob::OnMobSkillDelay`.

### B.2 — MONSTER_BOOK_COVER missing fname  ✅ CONFIRMED (all 5 versions)

`fname: ""` in v83, v84, v87, v95, jms (all provenance csv-import). The serverbound
send-site has never been derived. **Phase-1 action (Task 1.A.4 + each version):** derive
the client send-site for the book-cover request (cover card id submit) from each IDB; set
`fname` + `ida.address`, `provenance: ida-discovered`. Until then this op cannot be
evidence-pinned (already flagged in export-gaps.md).

### B.3 — MOB_ESCORT_RETURN_STOP / MOB_ESCORT_RETURN_STOP_SAY absent everywhere  ✅ CONFIRMED

Neither name exists in ANY registry. v95 instead carries clientbound `MOB_ESCORT_STOP`
(opcode 305, `CMob::OnEscortStopEndPermmision`) and `MOB_ESCORT_STOP_SAY` (opcode 306,
`CMob::OnEscortStopSay`). These are almost certainly the same ops the plan calls
`MOB_ESCORT_RETURN_STOP` (2.F10) / `MOB_ESCORT_RETURN_STOP_SAY` (2.F11). **Phase-1 action
(Task 1.D.3):** decide naming — either (a) treat the existing `MOB_ESCORT_STOP` /
`MOB_ESCORT_STOP_SAY` rows as the canonical ops and align the plan/codec struct names to
them, or (b) rename/add rows to `MOB_ESCORT_RETURN_STOP*`. The Atlas-side opcodes/writers
must match whatever the registry+templates use. These are v95-only.

---

## C. Newly-found gaps (beyond context.md §3)

1. **MOB_ESCORT_FULL_PATH in v87 is suspect.** The v87 row (opcode 273/0x111) carries a
   `note`: the v87 IDB `CMobPool::OnMobPacket` dispatch range ENDS at 0x110 and there is
   **no `CMob::OnEscortFullPath` symbol** in the v87 IDB — it's likely the tail of the
   0x10D–0x110 CSV off-by-one, not a real v87 op. Plan 2.F5 lists MOB_ESCORT_FULL_PATH for
   v87+v95. **Phase-1 action (Task 1.C.2):** confirm whether 0x111 dispatches in v87; if
   not, mark v87 VERSION-ABSENT for this op (implement v95 only) rather than fabricating a
   v87 codec. v95's row (304/0x130) has no such caveat — treat as real pending IDB confirm.

2. **v84 serverbound damage-opcode drift is intentional, not a gap.** MOB_DAMAGE_MOB_FRIENDLY
   (v84 opcode 197, v83 192) and MOB_DAMAGE_MOB (v84 199, v83 194) were re-derived from the
   v84 IDB COutPacket literals (provenance manual). These differ from the v83 sequence and
   from STATUS.md's `0xC5`/`0xC7` — already reconciled in task-085. **No action**, but the
   Stage-2 v84 template entries must use 197/199, NOT a +0 copy of v83. Recorded so the
   executor doesn't "fix" them back.

3. **CATCH_MONSTER jms uses an unnamed sub.** jms row carries `fname_alts: [sub_6EAE5F]`
   and provenance manual: case 0x10C dispatches to unnamed `sub_6EAE5F` which calls
   `CMob::ShowCatchEffect` (= OnCatchEffect, unnamed in this IDB). **Phase-1 note:** for jms
   evidence-pin the fname `CMob::OnCatchEffect` may not exist in the IDB/export under that
   name; `sub_6EAE5F` is the real address. Carry into export-gaps re-export scope.

4. **Three serverbound ops share `CMob::Update`** (FIELD_DAMAGE_MOB, MOB_DAMAGE_MOB_FRIENDLY,
   MOB_SKILL_DELAY_END) and two carnival cb ops share `CField_MonsterCarnival::OnRequestResult`
   (SUMMON, MESSAGE). Not a registry error (the dispatcher demuxes on a mode/sub-opcode
   inside one function), but Stage 1 must derive DISTINCT byte layouts per op from inside
   the shared function (plan Cluster A & E caveats).

---

## D. Phase-1 (Stage 1) registry-edit work list (the deliverable burndown)

| # | Version | Op(s) | Edit | Provenance |
|---|---|---|---|---|
| 1 | v83 (Task 1.A.3) | MOB_SPEAKING, INC_MOB_CHARGE_COUNT, (re-verify MOB_SKILL_DELAY) | set fname to IDB-dispatched handler (target: OnMobSpeaking / OnIncMobChargeCount / OnMobSkillDelay) | manual + IDA citation |
| 2 | v87 (Task 1.C.3) | MOB_SPEAKING, INC_MOB_CHARGE_COUNT, MOB_SKILL_DELAY | un-rotate all three fnames to IDB-dispatched handlers | manual + IDA citation |
| 3 | ALL 5 (Task 1.A.4 + per-version) | MONSTER_BOOK_COVER (sb) | derive send-site, set fname + ida.address | ida-discovered |
| 4 | v95 (Task 1.D.3) | MOB_ESCORT_STOP / MOB_ESCORT_STOP_SAY ↔ plan MOB_ESCORT_RETURN_STOP* | reconcile naming; confirm rows are the intended ops (no new rows unless IDB shows extras) | csv-import existing / ida-discovered if renamed |
| 5 | v87 (Task 1.C.2) | MOB_ESCORT_FULL_PATH | confirm 0x111 dispatches; if absent, mark VERSION-ABSENT (drop v87 from scope, keep v95) | manual + IDA citation |
| 6 | jms (Task 1.E) | CATCH_MONSTER | already has fname_alts sub_6EAE5F; ensure evidence uses the real (sub) address at pin time | manual |
| — | v84 / v95 | the 3 ⚠ mislabel ops | NO EDIT — already correct | — |

**No fname row should ever be deleted** (registry README rule); mislabels are corrected
in place with `provenance: manual` + an IDA case→target citation in `note`.
