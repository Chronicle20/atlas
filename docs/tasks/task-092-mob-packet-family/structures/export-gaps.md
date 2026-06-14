# Stage 0 — Export-Resolvability Audit (Task 0.2)

Checked 2026-06-13. For each of the 42 in-scope ops × each version where context.md §2
shows an opcode (not ABSENT), this audits whether the op's `fname` is a key in that
version's export `functions` map under `docs/packets/ida-exports/`.

Method: `jq -r '.functions | keys[]' <export>` → exact-string match against the fname;
on miss, a loose substring match on the method name (after `::`) to surface
demangled-vs-mangled discrepancies. Export filenames:
`gms_v83.json gms_v84.json gms_v87.json gms_v95.json gms_jms_185.json`.

Export key counts (curated decompile records, NOT full symbol dumps):
v83=306, v84=439, v87=319, v95=354, jms_v185=302.

---

## UNRESOLVED FNAMES — ESCALATE

**EVERY in-scope (op, version, fname) pair is UNRESOLVED. Zero of the 42 ops resolve
in any applicable-version export. `evidence pin` (which reads these exports and fails
if the fname is absent) will fail for all of them.**

Root cause (verified): the five `docs/packets/ida-exports/*.json` files are **curated
per-function decompile records** — each entry is one already-analyzed function with an
`address` + an ordered `calls[]` Decode/Encode list (see the `CLogin::OnCheckPasswordResult`
sample). They are NOT full IDA symbol dumps. They contain ~300–440 functions each, none
of which is an in-scope MOB/MONSTER handler/writer. Even `CMob::Update` (a heavily-used
class method) has zero exact key in any export. The MOB family was simply never exported.

Confirmed: no naming-format artifact. Keys are stored demangled (`Class::Method`,
e.g. `CAffectedAreaPool::OnAffectedAreaCreated`); the in-scope demangled fnames just are
not present. The only loose matches found are **unrelated** functions that happen to share
a method-name token (e.g. `CPet::SendDropPickUpRequest`, `CLogin::OnUpdatePinCodeResult`,
`CUIMessenger::OnPacket#Update`) — none is the target.

### Full unresolved list — (op, version, fname)

Every row below is an `N` and therefore a **pin blocker**. None of these is a
demangled-vs-mangled key mismatch; they are genuine absences (the function was not
exported). Resolution requires a **re-export** of each version IDB to add these
fnames (task-081 playbook), per context.md §3.4 / plan Task 0.2 Step 4.

#### Clientbound

| Op | versions unresolved | fname |
|---|---|---|
| RESET_MONSTER_ANIMATION | v83, v84, v87, v95, jms | CMob::OnSuspendReset |
| MOB_AFFECTED | v83, v84, v87, v95, jms | CMob::OnAffected |
| MONSTER_SPECIAL_EFFECT_BY_SKILL | v83, v84, v87, v95, jms | CMob::OnSpecialEffectBySkill |
| MOB_CRC_KEY_CHANGED | v83, v84, v87, v95, jms | CMobPool::OnMobCrcKeyChanged |
| CATCH_MONSTER | v83, v84, v87, v95, jms | CMob::OnCatchEffect |
| CATCH_MONSTER_WITH_ITEM | v83, v84, v87, v95, jms | CMob::OnEffectByItem |
| MOB_SPEAKING | v83, v84, v87, v95 | CMob::OnMobSpeaking |
| INC_MOB_CHARGE_COUNT | v83, v84, v87, v95 | CMob::OnIncMobChargeCount |
| MOB_SKILL_DELAY | v83, v84, v87, v95 | CMob::OnMobSkillDelay |
| SET_TAMING_MOB_INFO | v83, v84, v87, v95, jms | CWvsContext::OnSetTamingMobInfo |
| BRIDLE_MOB_CATCH_FAIL | v83, v84, v87, v95, jms | CWvsContext::OnBridleMobCatchFail |
| MONSTER_BOOK_SET_CARD | v83, v84, v87, v95, jms | CWvsContext::OnMonsterBookSetCard |
| MONSTER_BOOK_SET_COVER | v83, v84, v87, v95, jms | CWvsContext::OnMonsterBookSetCover |
| MONSTER_CARNIVAL_START | v83, v84, v87, v95, jms | CField_MonsterCarnival::OnEnter |
| MONSTER_CARNIVAL_OBTAINED_CP | v83, v84, v87, v95, jms | CField_MonsterCarnival::OnPersonalCP |
| MONSTER_CARNIVAL_PARTY_CP | v83, v84, v87, v95, jms | CField_MonsterCarnival::OnTeamCP |
| MONSTER_CARNIVAL_SUMMON | v83, v84, v87, v95, jms | CField_MonsterCarnival::OnRequestResult |
| MONSTER_CARNIVAL_MESSAGE | v83, v84, v87, v95, jms | CField_MonsterCarnival::OnRequestResult |
| MONSTER_CARNIVAL_DIED | v83, v84, v87, v95, jms | CField_MonsterCarnival::OnProcessForDeath |
| MONSTER_CARNIVAL_LEAVE | v83, v84, v87, v95, jms | CField_MonsterCarnival::OnShowMemberOutMsg |
| MONSTER_CARNIVAL_RESULT | v83, v84, v87, v95, jms | CField_MonsterCarnival::OnShowGameResult |
| MOB_ESCORT_FULL_PATH | v87, v95 | CMob::OnEscortFullPath |
| MOB_ESCORT_RETURN_STOP | v95 | CMob::OnEscortStopEndPermmision |
| MOB_ESCORT_RETURN_STOP_SAY | v95 | CMob::OnEscortStopSay |
| MOB_ESCORT_RETURN_BEFORE | v95 | CMob::OnEscortReturnBefore |
| MOB_NEXT_ATTACK | v95 | CMob::OnNextAttack |
| MOB_ATTACKED_BY_MOB | v95 | CMob::OnMobAttackedByMob |

#### Serverbound

| Op | versions unresolved | fname |
|---|---|---|
| TOUCH_MONSTER_ATTACK | v83, v84, v87, v95, jms | CUserLocal::TryDoingBodyAttack |
| MOB_BANISH_PLAYER | v83, v84, v87, v95, jms | CUserLocal::SendBanMapByMobRequest |
| MOB_CRC_KEY_CHANGED_REPLY | v83, v84, v87, v95, jms | CMobPool::OnMobCrcKeyChanged |
| MOB_DROP_PICKUP_REQUEST | v83, v84, v87, v95, jms | CMob::SendDropPickUpRequest |
| FIELD_DAMAGE_MOB | v83, v84, v87, v95, jms | CMob::Update |
| MOB_DAMAGE_MOB_FRIENDLY | v83, v84, v87, v95, jms | CMob::Update |
| MONSTER_BOMB | v83, v84, v87, v95, jms | CMob::TryFirstSelfDestruction |
| MOB_DAMAGE_MOB | v83, v84, v87, v95, jms | CMob::SetDamagedByMob |
| MOB_SKILL_DELAY_END | v83, v84, v87, v95, jms | CMob::Update |
| MOB_TIME_BOMB_END | v83, v84, v87, v95, jms | CMob::UpdateTimeBomb |
| MOB_ESCORT_COLLISION | v87, v95, jms | CMob::SendCollisionEscort |
| MONSTER_CARNIVAL | v83, v84, v87, v95, jms | CUIMonsterCarnival::RequestSend |
| MOB_REQUEST_ESCORT_INFO | v95, jms | CMob::SendRequestEscortPath |
| MOB_ESCORT_STOP_END_REQUEST | v95, jms | CMob::SendEscortStopEndRequest |
| MONSTER_BOOK_COVER | v83, v84, v87, v95, jms | (no fname yet — derived in Task 1.A.4; cannot check until then, but will need the same re-export to be pinnable) |

### What the executor must do (decision required from the user)

Per plan Task 0.2 Step 4 and R-CB.6: **do NOT auto-trigger a re-export, substitute a
fname, or fabricate a hash.** This is a hard escalation gate. The Stage-2 evidence-pin
step cannot succeed for any op until the exports carry the in-scope fnames. Options to
put to the user:

1. **Re-export each version IDB** to add the 42 MOB/MONSTER fnames (task-081 playbook),
   then regenerate the export hashes (STATUS.md header) — the canonical path. Stage 1
   (IDA harvest) already loads each IDB; the re-export can be folded into that pass.
2. Descope ops whose fname genuinely does not exist in a given IDB (none proven yet —
   all 42 are "not exported," not "not present in binary").
3. Correct a fname only if Stage-1 IDA decompile shows the registry fname is wrong AND a
   different (correct) symbol IS already in the export — none observed here.

Until the user decides, Stage 2 (codec + evidence pin + matrix promotion) is BLOCKED.
Stage 1 (IDA harvest of byte layouts + registry fname fixes) can still proceed because
it reads the live IDB, not the exports.

---

## Per-op detail (exact-match result + loose-match note)

All rows resolve **N**. "loose" column lists up to 3 unrelated functions sharing the
method-name token (evidence the method name itself is a common token, not the target).

| Op | Ver | fname | resolves | note |
|---|---|---|---|---|
| RESET_MONSTER_ANIMATION | 83/84/87/95/jms | CMob::OnSuspendReset | N | no key contains `OnSuspendReset` |
| MOB_AFFECTED | 83/84/87/95/jms | CMob::OnAffected | N | loose only: CAffectedAreaPool::OnAffectedArea{Created,Removed} (unrelated) |
| MONSTER_SPECIAL_EFFECT_BY_SKILL | 83/84/87/95/jms | CMob::OnSpecialEffectBySkill | N | no `OnSpecialEffectBySkill` key |
| MOB_CRC_KEY_CHANGED | 83/84/87/95/jms | CMobPool::OnMobCrcKeyChanged | N | no `OnMobCrcKeyChanged` key |
| CATCH_MONSTER | 83/84/87/95/jms | CMob::OnCatchEffect | N | no `OnCatchEffect` key |
| CATCH_MONSTER_WITH_ITEM | 83/84/87/95/jms | CMob::OnEffectByItem | N | no `OnEffectByItem` key |
| MOB_SPEAKING | 83/84/87/95 | CMob::OnMobSpeaking | N | no `OnMobSpeaking` key |
| INC_MOB_CHARGE_COUNT | 83/84/87/95 | CMob::OnIncMobChargeCount | N | no `OnIncMobChargeCount` key |
| MOB_SKILL_DELAY | 83/84/87/95 | CMob::OnMobSkillDelay | N | no `OnMobSkillDelay` key |
| SET_TAMING_MOB_INFO | 83/84/87/95/jms | CWvsContext::OnSetTamingMobInfo | N | no `OnSetTamingMobInfo` key (R-MARK op — pin still blocked) |
| BRIDLE_MOB_CATCH_FAIL | 83/84/87/95/jms | CWvsContext::OnBridleMobCatchFail | N | no `OnBridleMobCatchFail` key |
| MONSTER_BOOK_SET_CARD | 83/84/87/95/jms | CWvsContext::OnMonsterBookSetCard | N | no `OnMonsterBookSetCard` key |
| MONSTER_BOOK_SET_COVER | 83/84/87/95/jms | CWvsContext::OnMonsterBookSetCover | N | no `OnMonsterBookSetCover` key |
| MONSTER_CARNIVAL_START | 83/84/87/95/jms | CField_MonsterCarnival::OnEnter | N | no `MonsterCarnival` key in any export |
| MONSTER_CARNIVAL_OBTAINED_CP | 83/84/87/95/jms | CField_MonsterCarnival::OnPersonalCP | N | no `MonsterCarnival` key |
| MONSTER_CARNIVAL_PARTY_CP | 83/84/87/95/jms | CField_MonsterCarnival::OnTeamCP | N | no `MonsterCarnival` key |
| MONSTER_CARNIVAL_SUMMON | 83/84/87/95/jms | CField_MonsterCarnival::OnRequestResult | N | no `MonsterCarnival` key |
| MONSTER_CARNIVAL_MESSAGE | 83/84/87/95/jms | CField_MonsterCarnival::OnRequestResult | N | no `MonsterCarnival` key |
| MONSTER_CARNIVAL_DIED | 83/84/87/95/jms | CField_MonsterCarnival::OnProcessForDeath | N | no `MonsterCarnival` key |
| MONSTER_CARNIVAL_LEAVE | 83/84/87/95/jms | CField_MonsterCarnival::OnShowMemberOutMsg | N | no `MonsterCarnival` key |
| MONSTER_CARNIVAL_RESULT | 83/84/87/95/jms | CField_MonsterCarnival::OnShowGameResult | N | no `MonsterCarnival` key |
| MOB_ESCORT_FULL_PATH | 87/95 | CMob::OnEscortFullPath | N | no `Escort` key |
| MOB_ESCORT_RETURN_STOP | 95 | CMob::OnEscortStopEndPermmision | N | no `Escort` key |
| MOB_ESCORT_RETURN_STOP_SAY | 95 | CMob::OnEscortStopSay | N | no `Escort` key |
| MOB_ESCORT_RETURN_BEFORE | 95 | CMob::OnEscortReturnBefore | N | no `Escort` key |
| MOB_NEXT_ATTACK | 95 | CMob::OnNextAttack | N | no `OnNextAttack` key |
| MOB_ATTACKED_BY_MOB | 95 | CMob::OnMobAttackedByMob | N | no `OnMobAttackedByMob` key |
| TOUCH_MONSTER_ATTACK | 83/84/87/95/jms | CUserLocal::TryDoingBodyAttack | N | no `TryDoingBodyAttack` key |
| MOB_BANISH_PLAYER | 83/84/87/95/jms | CUserLocal::SendBanMapByMobRequest | N | no `SendBanMapByMobRequest` key |
| MOB_CRC_KEY_CHANGED_REPLY | 83/84/87/95/jms | CMobPool::OnMobCrcKeyChanged | N | no `OnMobCrcKeyChanged` key |
| MOB_DROP_PICKUP_REQUEST | 83/84/87/95/jms | CMob::SendDropPickUpRequest | N | loose only: CPet/CWvsContext::SendDropPickUpRequest (wrong class) |
| FIELD_DAMAGE_MOB | 83/84/87/95/jms | CMob::Update | N | loose only: many unrelated `*::*Update*` (no `CMob::Update`) |
| MOB_DAMAGE_MOB_FRIENDLY | 83/84/87/95/jms | CMob::Update | N | same as FIELD_DAMAGE_MOB (shared fname) |
| MONSTER_BOMB | 83/84/87/95/jms | CMob::TryFirstSelfDestruction | N | no `TryFirstSelfDestruction` key |
| MOB_DAMAGE_MOB | 83/84/87/95/jms | CMob::SetDamagedByMob | N | no `SetDamagedByMob` key |
| MOB_SKILL_DELAY_END | 83/84/87/95/jms | CMob::Update | N | same as FIELD_DAMAGE_MOB (shared fname) |
| MOB_TIME_BOMB_END | 83/84/87/95/jms | CMob::UpdateTimeBomb | N | no `UpdateTimeBomb` key |
| MOB_ESCORT_COLLISION | 87/95/jms | CMob::SendCollisionEscort | N | no `Escort` key |
| MONSTER_CARNIVAL | 83/84/87/95/jms | CUIMonsterCarnival::RequestSend | N | no `CUIMonsterCarnival` key |
| MOB_REQUEST_ESCORT_INFO | 95/jms | CMob::SendRequestEscortPath | N | no `Escort` key |
| MOB_ESCORT_STOP_END_REQUEST | 95/jms | CMob::SendEscortStopEndRequest | N | no `Escort` key |
| MONSTER_BOOK_COVER | 83/84/87/95/jms | (TBD Task 1.A.4) | N | fname not yet derived; will also need re-export to pin |

### Shared-fname note (carried to Stage 1)

Three serverbound ops point at `CMob::Update` (FIELD_DAMAGE_MOB, MOB_DAMAGE_MOB_FRIENDLY,
MOB_SKILL_DELAY_END) and two clientbound carnival ops point at
`CField_MonsterCarnival::OnRequestResult` (MONSTER_CARNIVAL_SUMMON, MONSTER_CARNIVAL_MESSAGE).
Even after re-export, a single export entry per shared fname will back multiple ops' pins —
fine for `evidence pin` (it keys on fname), but Stage 1 must still derive distinct byte
layouts per op (the SUMMON-vs-MESSAGE caveat in plan Cluster E).
