# Stage 0 — Matrix Baseline (Task 0.1)

Captured 2026-06-13 from the task-092 worktree (`task-092-mob-packet-family`).

## `matrix --check` baseline

```
$ go run ./tools/packet-audit matrix --check; echo "exit=$?"
exit=0
```

**Exit code: 0 — baseline is CLEAN.** No pre-existing conflicts/drift/orphan/stale
blockers. Any `--check` failure that appears later in this task is attributable to
task-092 work, not a pre-existing condition.

Tool revision (STATUS.md header): `5d241239e5f7ed5d7a124ffafbedff672e6a920f`.
Export hashes (STATUS.md header):
- gms_v83: `359495963d99164fb331ed7c1c1a9a7920a85126122821fdb8bfebfad3f3df5b`
- gms_v84: `e94fde3eb9bec89d4abd84cce666b7a64e4780c9ac9235d14e25383a9793a3ef`
- gms_v87: `04665aba8142ba592fa97e5e312b513ed3f252c061f33c6a067d10233a126b49`
- gms_v95: `b00cae68c1f5896d2712c46c68c34b5194168cef370bdc6c6126db541d9cc5d3`
- jms_v185: `bfc651ad5f107ae48635d475f944e3e8a6f985d0a8c33d67f3ac2cb9f99fcb5b`

## Current STATUS.md cell state for the 42 in-scope ops

Symbols: ✅ verified · 🟡 partial · ❌ incomplete · ⬜ n-a · 🟥 conflict.
Per-version cell from `docs/packets/audits/STATUS.md` (verified 2026-06-13).

### Clientbound

| Op | STATUS FName | v83 | v84 | v87 | v95 | jms185 |
|---|---|---|---|---|---|---|
| RESET_MONSTER_ANIMATION | CMob::OnSuspendReset | ❌ 0xF4 | ❌ 0xFA | ❌ 0x104 | ❌ 0x124 | ❌ 0x105 |
| MOB_AFFECTED | CMob::OnAffected | ❌ 0xF5 | ❌ 0xF5 | ❌ 0x105 | ❌ 0x125 | ❌ 0x106 |
| MONSTER_SPECIAL_EFFECT_BY_SKILL | CMob::OnSpecialEffectBySkill | ❌ 0xF7 | ❌ 0xF7 | ❌ 0x107 | ❌ 0x127 | ❌ 0x108 |
| MOB_CRC_KEY_CHANGED | CMobPool::OnMobCrcKeyChanged | ❌ 0xF9 | ❌ 0xF9 | ❌ 0x109 | ❌ 0x129 | ❌ 0x10A |
| CATCH_MONSTER | CMob::OnCatchEffect; sub_6EAE5F | ❌ 0xFB | ❌ 0xFB | ❌ 0x10B | ❌ 0x12B | ❌ 0x10C |
| CATCH_MONSTER_WITH_ITEM | CMob::OnEffectByItem | ❌ 0xFC | ❌ 0xFC | ❌ 0x10C | ❌ 0x12C | ❌ 0x10D |
| MOB_SPEAKING ⚠ | CMob::OnIncMobChargeCount; CMob::OnMobSpeaking | ❌ 0xFE | ❌ 0xFE | ❌ 0x10E | ❌ 0x12D | ⬜ (ABSENT) |
| INC_MOB_CHARGE_COUNT ⚠ | CMob::OnIncMobChargeCount; CMob::OnMobAttackedByMob; CMob::OnMobSkillDelay | ❌ 0xFF | ❌ 0xFF | ❌ 0x10F | ❌ 0x12E | ⬜ (ABSENT) |
| MOB_SKILL_DELAY ⚠ | CMob::OnMobAttackedByMob; CMob::OnMobSkillDelay | ❌ 0x100 | ❌ 0x100 | ❌ 0x110 | ❌ 0x12F | ⬜ (ABSENT) |
| SET_TAMING_MOB_INFO | CWvsContext::OnSetTamingMobInfo | ❌ 0x30 | ❌ 0x30 | ❌ 0x30 | ❌ 0x2F | ❌ 0x2D |
| BRIDLE_MOB_CATCH_FAIL | CWvsContext::OnBridleMobCatchFail | ❌ 0x4F | ❌ 0x4F | ❌ 0x51 | ❌ 0x52 | ❌ 0x49 |
| MONSTER_BOOK_SET_CARD | CWvsContext::OnMonsterBookSetCard | ❌ 0x53 | ❌ 0x53 | ❌ 0x55 | ❌ 0x56 | ❌ 0x57 |
| MONSTER_BOOK_SET_COVER | CWvsContext::OnMonsterBookSetCover | ❌ 0x54 | ❌ 0x54 | ❌ 0x56 | ❌ 0x57 | ❌ 0x58 |
| MONSTER_CARNIVAL_START | CField_MonsterCarnival::OnEnter | ❌ 0x121 | ❌ 0x121 | ❌ 0x132 | ❌ 0x15A | ❌ 0x139 |
| MONSTER_CARNIVAL_OBTAINED_CP | CField_MonsterCarnival::OnPersonalCP | ❌ 0x122 | ❌ 0x122 | ❌ 0x133 | ❌ 0x15B | ❌ 0x13A |
| MONSTER_CARNIVAL_PARTY_CP | CField_MonsterCarnival::OnTeamCP | ❌ 0x123 | ❌ 0x123 | ❌ 0x134 | ❌ 0x15C | ❌ 0x13B |
| MONSTER_CARNIVAL_SUMMON | CField_MonsterCarnival::OnRequestResult | ❌ 0x124 | ❌ 0x124 | ❌ 0x135 | ❌ 0x15D | ❌ 0x13C |
| MONSTER_CARNIVAL_MESSAGE | CField_MonsterCarnival::OnRequestResult | ❌ 0x125 | ❌ 0x125 | ❌ 0x136 | ❌ 0x15E | ❌ 0x13D |
| MONSTER_CARNIVAL_DIED | CField_MonsterCarnival::OnProcessForDeath | ❌ 0x126 | ❌ 0x126 | ❌ 0x137 | ❌ 0x15F | ❌ 0x13E |
| MONSTER_CARNIVAL_LEAVE | CField_MonsterCarnival::OnShowMemberOutMsg | ❌ 0x127 | ❌ 0x127 | ❌ 0x138 | ❌ 0x160 | ❌ 0x13F |
| MONSTER_CARNIVAL_RESULT | CField_MonsterCarnival::OnShowGameResult | ❌ 0x128 | ❌ 0x128 | ❌ 0x139 | ❌ 0x161 | ❌ 0x140 |
| MOB_ESCORT_FULL_PATH | CMob::OnEscortFullPath | ⬜ | ⬜ | ❌ 0x111 | ❌ 0x130 | ⬜ |
| MOB_ESCORT_STOP (= plan MOB_ESCORT_RETURN_STOP) | CMob::OnEscortStopEndPermmision | ⬜ | ⬜ | ⬜ | ❌ 0x131 | ⬜ |
| MOB_ESCORT_STOP_SAY (= plan MOB_ESCORT_RETURN_STOP_SAY) | CMob::OnEscortStopSay | ⬜ | ⬜ | ⬜ | ❌ 0x132 | ⬜ |
| MOB_ESCORT_RETURN_BEFORE | CMob::OnEscortReturnBefore | ⬜ | ⬜ | ⬜ | ❌ 0x133 | ⬜ |
| MOB_NEXT_ATTACK | CMob::OnNextAttack | ⬜ | ⬜ | ⬜ | ❌ 0x134 | ⬜ |
| MOB_ATTACKED_BY_MOB | CMob::OnMobAttackedByMob | ⬜ | ⬜ | ⬜ | ❌ 0x135 | ⬜ |

### Serverbound

| Op | STATUS FName | v83 | v84 | v87 | v95 | jms185 |
|---|---|---|---|---|---|---|
| TOUCH_MONSTER_ATTACK | CUserLocal::TryDoingBodyAttack | ❌ 0x2F | ❌ 0x2F | ❌ 0x31 | ❌ 0x32 | ❌ 0x26 |
| MOB_BANISH_PLAYER | CUserLocal::SendBanMapByMobRequest | ❌ 0x38 | ❌ 0x38 | ❌ 0x3B | ❌ 0x3D | ❌ 0x30 |
| MONSTER_BOOK_COVER | (no fname in STATUS) | ❌ 0x39 | ❌ 0x39 | ❌ 0x3C | ❌ 0x3E | ❌ 0x31 |
| MOB_CRC_KEY_CHANGED_REPLY | CMobPool::OnMobCrcKeyChanged | ❌ 0xA4 | ❌ 0xA4 | ❌ 0xAE | ❌ 0xBE | ❌ 0x9E |
| MOB_DROP_PICKUP_REQUEST | CMob::SendDropPickUpRequest | ❌ 0xBE | ❌ 0xBE | ❌ 0xCA | ❌ 0xE5 | ❌ 0xC4 |
| FIELD_DAMAGE_MOB | CMob::Update | ❌ 0xBF | ❌ 0xBF | ❌ 0xCB | ❌ 0xE6 | ❌ 0xC5 |
| MOB_DAMAGE_MOB_FRIENDLY | CMob::Update | ❌ 0xC0 | ❌ 0xC5 | ❌ 0xCC | ❌ 0xE7 | ❌ 0xC6 |
| MONSTER_BOMB | CMob::TryFirstSelfDestruction | ❌ 0xC1 | ❌ 0xC1 | ❌ 0xCD | ❌ 0xE8 | ❌ 0xC7 |
| MOB_DAMAGE_MOB | CMob::SetDamagedByMob | ❌ 0xC2 | ❌ 0xC7 | ❌ 0xCE | ❌ 0xE9 | ❌ 0xC8 |
| MOB_SKILL_DELAY_END | CMob::Update | ❌ 0xC3 | ❌ 0xC3 | ❌ 0xCF | ❌ 0xEA | ❌ 0xC9 |
| MOB_TIME_BOMB_END | CMob::UpdateTimeBomb | ❌ 0xC4 | ❌ 0xC4 | ❌ 0xD0 | ❌ 0xEB | ❌ 0xCA |
| MOB_ESCORT_COLLISION | CMob::SendCollisionEscort | ⬜ | ⬜ | ❌ 0xD1 | ❌ 0xEC | ❌ 0xCB |
| MONSTER_CARNIVAL | CUIMonsterCarnival::RequestSend | ❌ 0xDA | ❌ 0xDA | ❌ 0xE7 | ❌ 0x106 | ❌ 0xE5 |
| MOB_REQUEST_ESCORT_INFO | CMob::SendRequestEscortPath | ⬜ | ⬜ | ⬜ | ❌ 0xED | ❌ 0xCC |
| MOB_ESCORT_STOP_END_REQUEST | CMob::SendEscortStopEndRequest | ⬜ | ⬜ | ⬜ | ❌ 0xEE | ❌ 0xCD |

## Observations carried into Phase 1

- Every in-scope cell is currently `❌ incomplete` or `⬜ n-a` (no `✅`/`🟡`/`🟥`).
  Matches context.md §5.
- **Plan-vs-STATUS name mismatch:** the plan's Cluster-F ops `MOB_ESCORT_RETURN_STOP`
  / `MOB_ESCORT_RETURN_STOP_SAY` (2.F10/2.F11) do NOT appear in STATUS.md under those
  names. STATUS.md instead has v95-only `MOB_ESCORT_STOP` (0x131,
  `CMob::OnEscortStopEndPermmision`) and `MOB_ESCORT_STOP_SAY` (0x132,
  `CMob::OnEscortStopSay`). This is consistent with context.md §3.3 (those plan names
  are absent from the registry) and is the registry gap to resolve in Task 1.D.3.
- ⚠ rows = the cross-mislabeled fname cluster (MOB_SPEAKING / INC_MOB_CHARGE_COUNT /
  MOB_SKILL_DELAY); STATUS.md shows multiple semicolon-joined fnames for these,
  reflecting the registry mislabel (context.md §3.1).
- MONSTER_BOOK_COVER has an EMPTY fname column in STATUS.md (context.md §3.2 gap).
