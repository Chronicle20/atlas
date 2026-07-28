# CharacterAttackMeleeRequest (← `CUserLocal::TryDoingMeleeAttack`)

- **IDA:** 0x8c22fd
- **Atlas file:** `libs/atlas-packet/character/serverbound/attack_request.go`
- **Variant:** GMS/v79
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
| 4 | int32 | int32 `` | ✅ |  |
| 5 | int32 | byte `` | ❌ | width mismatch |
| 6 | byte | int16 `` | ❌ | width mismatch |
| 7 | int16 | byte `` | ❌ | width mismatch |
| 8 | byte | byte `` | ✅ |  |
| 9 | byte | int32 `` | ❌ | width mismatch |
| 10 | int32 | int32 `` | ✅ |  |
| 11 | int16 | byte `` | ❌ | width mismatch |
| 12 | int16 | byte `` | ❌ | width mismatch |
| 13 | byte | byte `` | ✅ |  |
| 14 | int32 | byte `` | ❌ | width mismatch |
| 15 | byte | int16 `` | 🔍 | sub-struct: di — see _substruct/ |
| 16 | int16 | int16 `` | ✅ |  |
| 17 | int16 | int16 `` | ✅ |  |
| 18 | int16 | int16 `` | ✅ |  |
| 19 | int16 | byte `` | ❌ | width mismatch |
| 20 | int32 | int32 `` | ✅ |  |
| 21 | int32 | int32 `` | ✅ |  |
| 22 | byte | int16 `` | ❌ | width mismatch |
| 23 | int16 | int16 `` | ✅ |  |
| 24 | int16 | byte `` | ❌ | width mismatch |
| 25 | byte | int32 `` | ❌ | atlas: short — missing trailing field |
| 26 | byte | byte `` | ❌ | atlas: short — missing trailing field |
| 27 | byte | int16 `` | ❌ | atlas: short — missing trailing field |

---

## task-150 note — Meso Explosion variant (hand-added; keep on regeneration)

CLOSE_RANGE_ATTACK carries a Meso Explosion (4211006) variant written by a
dedicated sender (this version: `0x8c22fd`), dispatched from `DoActiveSkill`
case 4211006. IDA-verified deltas vs. the standard melee attack (design §2.1):
per-mob `Encode1(damageLineCount)` replaces the int16 delay; trailing
`{dropId int32, hitMask byte}` list after characterX/Y; trailing int16 delay.
All base-layout fields (per-mob CRC, action width, head CRCs) follow this
version's existing standard-melee gates (design §2.1a) — the variant adds no
new gate: this version's short (int16) attack-action width and its two head
skill-data CRCs (both present ≥ v79) are read on the meso sender exactly as
on the standard sender. No new packet-audit:verify marker is pinned: the
meso sender is an fname_alt absent from the IDA export, so a second marker
would orphan under `matrix --check` (Task 3 rationale). Fixture:
`libs/atlas-packet/character/serverbound/attack_request_test.go#TestAttackMeleeRequestMesoExplosion`.

