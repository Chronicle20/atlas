# CharacterAttackMeleeRequest (← `CUserLocal::TryDoingNormalAttack`)

- **IDA:** 0xa122be
- **Atlas file:** `libs/atlas-packet/character/serverbound/attack_request.go`
- **Variant:** JMS/v185
- **Branch depth:** 0
- **Verdict:** ❌

## Wire-level diff

| # | Atlas writes | v? reads | Verdict | Note |
|---|---|---|---|---|
| 0 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 1 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 2 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 3 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 4 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 5 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 6 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 7 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 8 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 9 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 10 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 11 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 12 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 13 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 14 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 15 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 16 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 17 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 18 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 19 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 20 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 21 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 22 | int32 | byte `` | ❌ | atlas: extra — client never reads this field |
| 23 | byte | byte `` | ❌ | atlas: extra — client never reads this field |
| 24 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |
| 25 | int16 | byte `` | ❌ | atlas: extra — client never reads this field |

---

## task-150 note — Meso Explosion variant (hand-added; keep on regeneration)

CLOSE_RANGE_ATTACK carries a Meso Explosion (4211006) variant written by a
dedicated sender, `sub_A3AAB1` @ `0xa3aab1` in the jms IDB. The sender's
packet-encode tail is SCY code-flow-virtualized (`JUMPOUT(0xD29D2D)`), so the
jms serverbound variant read order is **not statically verifiable** in the
available dump. Atlas implements the jms variant from the deltas verified
byte-identical across the eight GMS versions (gms_v48–v95) plus the jms
**clientbound** meso branch (`CUserRemote::OnAttack` @ `0xa53999` region),
which was IDA-verified to match. No verify marker or evidence record was added
for the unreadable tail (task-150 design §2.3). gms_v92 (no IDB) follows the
GMS >= 87 family branch and gms_12 the very-legacy GMS < 48 branch; both are
template-only and unverified (design §2.4). Fixture:
`libs/atlas-packet/character/serverbound/attack_request_test.go#TestAttackMeleeRequestMesoExplosion`.


---

## fix-jms185-attack-decode note — live-capture verification (hand-added; keep on regeneration)

The wire-diff table above is not usable evidence for this cell: the jms v185
sender's packet-encode tail is code-flow-virtualized (last reachable call is
`DR_check` @ `0xa16b74`; nothing decodable between there and the function end at
`0xa18dda`), so the generated diff reads every client field as an untyped
`byte`. This is the same limitation the task-150 note records for the Meso
Explosion variant.

This cell previously held ✅ on `TestAttackMeleeRequest`, a `pt.RoundTrip`. A
round-trip is symmetric and passes for any self-consistent layout — including a
wrong one, which is what was shipped: the live JMS 185.1 client sends the GMS
v84-style damage-randomizer blocks, a single skill-data CRC, a melee trailing
word, and the per-mob anti-hack CRC, none of which the region-gated decoder
consumed. The decode ran 28 bytes short in the head plus 4 bytes per attacked
mob, `skillId` decoded out of dr0/dr1 as `0x27FFE51C`, and atlas-channel's
unowned-skill check destroyed the session on every attack.

The cell is now carried by `TestAttackMeleeRequestBytesJMS185`
(`libs/atlas-packet/character/serverbound/attack_request_test.go`), which pins
two `[PKT IN ]` captures taken from the live client on the atlas-main k3s
environment (JMS 185.1, op `0x0023`, 2026-09-02): a 53-byte swing hitting
nothing and a 79-byte swing connecting with one mob. Both decode with zero
bytes remaining onto self-evidently correct values and re-encode byte-for-byte.
Per `VERIFYING_A_PACKET.md` ("Producible prerequisite vs genuine blocker") an
undecompilable binary is a genuine blocker for decompile-derived bytes; live
wire is the higher-ranked evidence available. Full derivation:
`docs/tasks/fix-jms185-attack-decode/diagnosis.md`.

**Still unverified on jms v185:** `CharacterAttackMagicRequest` and
`CharacterAttackRangedRequest`. They share the head fixed here, but the magic
secondary dr-block, the magic trailing word, and the ranged bullet/exJablin
fields have no jms capture and remain on GMS-only gates. Their cells rest on
round-trips and carry the same false-positive risk this note describes.
