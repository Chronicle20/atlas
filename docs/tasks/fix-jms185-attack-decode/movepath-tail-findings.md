# Findings — the serverbound CMovePath tail is present on ALL TEN client versions

Nine read-only IDA investigations, one per GMS IDB, run against the brief in
`movepath-tail-brief.md`. Every one reports **PRESENT**, with the same field
order and the same `1 + ceil(count/2) + 8` size formula as jms v185:

    Encode1(len(m_aKeyPadState))        entry COUNT, not a byte count
    loop i += 2: Encode1(nType)         nType = state[i]&0xF, and unless i is
                                        the last index, |= state[i+1]<<4
    Encode2(m_rcMove.left)
    Encode2(m_rcMove.top)
    Encode2(m_rcMove.right)
    Encode2(m_rcMove.bottom)

## Per-version addresses

| version | CMovePath::Encode | count | nibble loop | rect (l/t/r/b) |
|---|---|---|---|---|
| gms_v48 | `0x56201a` (unnamed, `sub_56201A`) | `0x5621bd` | `0x5621c2`–`0x5621f9` | `0x56220b` `0x562219` `0x562227` `0x562235` |
| gms_v61 | `0x5e298d` | `0x5e2b72` | `0x5e2b8d`–`0x5e2bae` | `0x5e2bc0` `0x5e2bce` `0x5e2bdc` `0x5e2bef` |
| gms_v72 | `0x634ddb` | `0x634fc0` | `0x634fdb`–`0x634ffc` | `0x63500e` `0x63501c` `0x63502a` `0x635038` |
| gms_v79 | `0x6575fa` | `0x6577df` | `0x6577e4`–`0x65781b` | `0x65782d` `0x65783b` `0x657849` `0x65785c` |
| gms_v83 | `0x68a563` | `0x68a748` | `0x68a763`–`0x68a784` | `0x68a796` `0x68a7a4` `0x68a7b2` `0x68a7c0` |
| gms_v84 | `0x6a121a` (unnamed, `sub_6A121A`) | `0x6a141e` | `0x6a1423`–`0x6a145a` | `0x6a146c` `0x6a147a` `0x6a1488` `0x6a1496` |
| gms_v87 | `0x6c70fe` | `0x6c734c` | `0x6c7383`–`0x6c738b` | `0x6c73a2` `0x6c73b0` `0x6c73be` `0x6c73cc` |
| gms_v92 | `0x65a260` | `0x65a61b` | `0x65a620`–`0x65a68c` | `0x65a6da` `0x65a71c` `0x65a75e` `0x65a7a1` |
| gms_v95 | `0x666e20` | `0x6671db` | `0x6671e0`–`0x66724c` | `0x66729a` `0x6672dc` `0x66731e` `0x667361` |
| jms_v185 | `0x70b6c4` | `0x70b8ec` | `0x70b8f3`–`0x70b92b` | `0x70b942` `0x70b950` `0x70b95e` `0x70b96c` |

v48 and v84 have no symbol for the encoder; both were located structurally as
the function `CMovePath::Flush` delegates to (v84: `CMovePath__Flush`
@`0x6a1567`, call @`0x6a16ea`), anchored on the
Encode2/Encode2/Encode1-header-then-per-element-switch shape.

On several versions Hex-Rays inlines `Encode1`/`Encode2` into direct buffer
stores (v92, v95); the agents recorded the store address and noted the
inlining rather than claiming a call that isn't there.

Field identity of the rect was confirmed, not assumed: on v48/v61/v72/v79/v83/
v84 the `+48/+52/+56/+60` slots are visibly initialized as the running
min/max accumulator inside the element loop, so they are left/top/right/bottom.

## The decode side differs three ways — all reaching the same conclusion

The tail is serverbound-only on every version, but by three distinct
mechanisms. Recorded because "same as jms" would be false for seven of them.

| versions | decode-side mechanism |
|---|---|
| gms_v92, gms_v95, jms_v185 | live `bPassive` parameter; the keypad/rect block is inside `if (bPassive)`, and the clientbound `CUserRemote::OnMove` passes literal `0` (v92 `0x925447`, v95 `0x948a97`, jms via `OnMovePacket(..., 0)`) |
| gms_v48, gms_v72, gms_v83, gms_v84, gms_v87 | no such parameter exists in the compiled interface — `retn 4`, one stack arg, call sites push one value — and `Decode` contains no tail-reading instructions at all |
| gms_v79 | the parameter exists in the signature but is never tested: clobbered as scratch at `0x657400` and reused as a "previous X" cache; the optimizer stopped materializing a value at the call site |

Three IDBs (v79, v84, v87, and v83's `OnMovePacket`) carry a mangled name
advertising `(CInPacket&, int)` that the compiled code does not honour. Every
agent checked the disassembly rather than the demangled signature. On v83 the
*decompiled* `OnMovePacket` even appears to forward a third argument; that is
"positive sp value" decompiler noise, disproved by the stack cleanup.

## Consequence

`moveKeyPadTail` in `libs/atlas-packet/character/serverbound/move.go` is
currently `t.IsRegion("JMS")`. It should be unconditional: every supported
client writes this tail on serverbound MOVE_PLAYER.

**It must stay out of `model.Movement`.** That model is shared with the
clientbound encoder, and no version's client reads the tail on the clientbound
path. Emitting it there would break movement broadcasts on all ten versions.

## What this does NOT establish

These are encoder-shape derivations. Only jms has captured wire
(`TestCharacterMoveBytesJMS185`). The GMS `Move` fixtures pin **Atlas's own
output**, so they cannot confirm a client contract in either direction — which
is exactly why this defect survived nine verified GMS cells. Any GMS fixture
written from this document is decompile-derived and must say so; do not
describe it as pinned to observed traffic.

Widening the gate changes GMS decode behaviour in production. The risk
asymmetry: failing to read a tail that exists is harmless (nothing follows it),
whereas reading one that does not exist would over-read. Nine independent
decompiles say it exists on all nine.
