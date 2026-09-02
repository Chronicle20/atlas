# Brief — does the serverbound CMovePath tail exist on each GMS client?

## Background

`libs/atlas-packet/character/serverbound/move.go` now models an 18-byte tail on
serverbound MOVE_PLAYER for JMS v185 only, behind `moveKeyPadTail`. Derived
from the jms IDB, `CMovePath::Encode` @`0x70b6c4` (reached from
`CMovePath::Flush` @`0x70ba2c`, which delegates all encoding to it):

    @0x70b8ec  Encode1(len(m_aKeyPadState))     entry COUNT, not a byte count
    @0x70b8f3  loop i += 2, Encode1(nType):
               nType = state[i] & 0xF, and for every i but the last
               nType |= state[i+1] << 4         two 4-bit entries per byte
    @0x70b942  Encode2(m_rcMove.left)
    @0x70b950  Encode2(m_rcMove.top)
    @0x70b95e  Encode2(m_rcMove.right)
    @0x70b96c  Encode2(m_rcMove.bottom)

Total: `1 + ceil(count/2) + 8`.

Critically it is **serverbound-only**. The client's read side,
`CMovePath::Decode` @`0x70b3ce`, consumes the keypad block and the rect only
under `if (bPassive)`, and the clientbound entry `CUserRemote::OnMove`
@`0xa443ee` calls `CMovePath::OnMovePacket(..., iPacket, 0)` — `bPassive = 0`.
So the tail must never be added to the shared `model.Movement`, which the
clientbound encoder also uses.

The gate is JMS-only because jms is the one client that was read. The tail is
unconditional in that encoder, so GMS probably carries it too — but the
existing GMS `Move` fixtures pin **Atlas's own output**, not captured client
wire, so they would not have caught its absence. This must be settled per
version by reading the binary, never by analogy.

## The question, per GMS version

1. `CMovePath::Encode` — find it (known: v83 `0x68a563`, v87 `0x6c70fe`,
   v92 `0x65a260`, v95 `0x666e20`; the rest must be located). After its
   per-element loop, does it write the keypad count + packed-nibble loop + four
   `Encode2` rect fields? Quote the address of each write.
2. If the shape differs (e.g. no nibble packing, one entry per byte, a
   different field count, or no rect), record exactly what it does instead.
3. `CMovePath::Decode` — is the keypad/rect block gated on a `bPassive`-style
   parameter, as on jms?
4. The clientbound entry (`CUserRemote::OnMove` or the version's equivalent) —
   what value does it pass for that parameter? This is what proves the tail is
   serverbound-only on that version too.

## Rules

- Read-only. Change no file.
- Quote the decompiled line and its address for every claim. An address with no
  quoted line is not evidence.
- Distrust IDB symbol names (CLAUDE.md); anchor on the `COutPacket::Encode*`
  call sequence.
- If a function is unnamed, locate it structurally and say how.
- If something is genuinely unreadable (virtualized/SMC), say so and name the
  address where the trail stops. Do not guess the layout.

## Report

- `PRESENT` / `ABSENT` / `DIFFERENT (describe)` / `UNREADABLE (address)`.
- The per-field addresses.
- The `bPassive` answer for items 3 and 4.

## Consumers

The gate `moveKeyPadTail` in `character/serverbound/move.go` is widened to the
versions that come back PRESENT. Every widened version needs a byte fixture
before its cell can be re-claimed — round-trips cannot catch this class of bug,
which is how the jms cell held a false ✅ in the first place.
