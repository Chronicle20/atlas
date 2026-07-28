# CharacterAttackMeleeRequest (← `CUserLocal::TryDoingMeleeAttack`)

- **IDA:** 0x7a45f1
- **Atlas file:** `libs/atlas-packet/character/serverbound/attack_request.go`
- **Variant:** GMS/v61
- **Branch depth:** 0
- **Verdict:** 🔍
- **Flat-diff-invalid:** the wire shape depends on a runtime discriminator a flat positional diff cannot model — the Atlas writer branches on a non-version condition (a data-dependent field or an untraced version-derived local), and/or the client reads fields conditionally (e.g. `mode <= 1`). The verdict is capped to 🔍; the row-level mismatches below are a modeling limitation, not a verified wire bug — confirm per-branch via byte-level tests.

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ✅ |  |
| 1 | byte | byte `` | ✅ |  |
| 2 | int32 | int32 `` | ✅ |  |
| 3 | int32 | int32 `` | ✅ |  |
| 4 | int32 | byte `` | ❌ | width mismatch |
| 5 | int32 | byte `` | ❌ | width mismatch |
| 6 | byte | byte `` | ✅ |  |
| 7 | byte | byte `` | ✅ |  |
| 8 | int32 | int32 `` | ✅ |  |
| 9 | int32 | int32 `` | ✅ |  |
| 10 | int16 | byte `` | ❌ | width mismatch |
| 11 | int16 | byte `` | ❌ | width mismatch |
| 12 | byte | byte `` | ✅ |  |
| 13 | int32 | byte `` | ❌ | width mismatch |
| 14 | byte | int16 `` | 🔍 | sub-struct: di — see _substruct/ |
| 15 | int16 | int16 `` | ✅ |  |
| 16 | int16 | int16 `` | ✅ |  |
| 17 | int16 | int16 `` | ✅ |  |
| 18 | int16 | int16 `` | ✅ |  |
| 19 | int32 | int32 `` | ✅ |  |
| 20 | int32 | int32 `` | ✅ |  |
| 21 | byte | int16 `` | ❌ | width mismatch |
| 22 | int16 | int16 `` | ✅ |  |
| 23 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

---

## task-150 note — Meso Explosion variant (hand-added; keep on regeneration)

CLOSE_RANGE_ATTACK carries a Meso Explosion (4211006) variant written by a
dedicated sender (this version: `0x7b8a39`), dispatched from `DoActiveSkill`
case 4211006. IDA-verified deltas vs. the standard melee attack (design §2.1):
per-mob `Encode1(damageLineCount)` replaces the int16 delay; trailing
`{dropId int32, hitMask byte}` list after characterX/Y; trailing int16 delay.
All base-layout fields (per-mob CRC, action width, head CRCs) follow this
version's existing standard-melee gates (design §2.1a) — the variant adds no
new gate. No new packet-audit:verify marker is pinned: the meso sender is an
fname_alt absent from the IDA export, so a second marker would orphan under
`matrix --check` (Task 3 rationale). Fixture:
`libs/atlas-packet/character/serverbound/attack_request_test.go#TestAttackMeleeRequestMesoExplosion`.

