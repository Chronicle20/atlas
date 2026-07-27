# CharacterAttackMeleeRequest (← `CUserLocal::TryDoingMeleeAttack`)

- **IDA:** 
- **Atlas file:** `libs/atlas-packet/character/serverbound/attack_request.go`
- **Variant:** GMS/v48
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | unresolved `function not found in IDB` | 🚫 | IDA read-order unresolved: function not found in IDB |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 12 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 13 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 14 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 19 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 20 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 21 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 22 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 23 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 24 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 25 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 26 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

---

## task-150 note — Meso Explosion variant (hand-added; keep on regeneration)

CLOSE_RANGE_ATTACK carries a Meso Explosion (4211006) variant written by a
dedicated sender (this version: `0x6ae4d7`), dispatched from `DoActiveSkill`
case 4211006. IDA-verified deltas vs. the standard melee attack (design §2.1):
per-mob `Encode1(damageLineCount)` replaces the int16 delay; trailing
`{dropId int32, hitMask byte}` list after characterX/Y; trailing int16 delay.
All base-layout fields (per-mob CRC, action width, head CRCs) follow this
version's existing standard-melee gates (design §2.1a) — the variant adds no
new gate. This version has no per-mob CRC (< v61); the meso mode's damage
loop keeps the shared `>= 61` gate, so the read/write falls out for free. No
new packet-audit:verify marker is pinned: the meso sender is an
fname_alt absent from the IDA export, so a second marker would orphan under
`matrix --check` (Task 3 rationale). Fixture:
`libs/atlas-packet/character/serverbound/attack_request_test.go#TestAttackMeleeRequestMesoExplosion`.

