# Findings — the CMovePath tail across the sibling serverbound movement ops (jms v185)

Read-only audit. `CMovePath::Flush` @`0x70ba2c` calls
`CMovePath::Encode(this, oPacket, 0)` @`0x70bbb6`; the `pbPassive` argument is
never read inside `Encode`, so the tail
(count @`0x70b8ec`, nibble loop @`0x70b8f3`, rect @`0x70b942`/`0x70b950`/
`0x70b95e`/`0x70b96c`) is written by EVERY sender that reaches Flush.

## 1. npc/serverbound/action.go — MISDECODING IN PRODUCTION

The only sibling with live wire proof. `[PKT IN] handler=NPCActionHandle
op=0x00d0 len=40`, same tenant/session as the Part 2 evidence:

    d0 00 02000000 ff ff 4103 7d00 01 00 4103 7d00 00000000 0900
    00000000 05 3804 00 4103 7d00 4103 7d00

Atlas (`npc/serverbound/action.go:56-65`) decodes objectId=2, unk=255,
unk2=255, then movement startX=833, startY=125, numElems=1, and finishes the
NORMAL element at reader offset 31 of 40 — leaving **9 bytes**:

    00 | 41 03 7d 00 | 41 03 7d 00
    ^count=0  ^left=833,top=125   ^right=833,bottom=125

`1 + 0 + 8`, and the rect is a single-point bounding box consistent with the
one zero-velocity element just decoded. Sender `CNpc::GenerateMovePath`
@`0x7199ce` header matches Atlas exactly (`Encode4(dwNpcId)` @`0x719a7e`,
`Encode1(nAction)` @`0x719a89`, `Encode1(nChatIdx)` @`0x719a94`, then Flush
@`0x719ab0`) — the header is fine, only the tail is missing.

This one leaks silently: nothing warns on leftover bytes on this path, so it
is invisible except by byte accounting.

## 2. monster/serverbound/movement.go — tail missing (decompile-confirmed)

`CMob::GenerateMovePath` @`0x6e8892`. Its full header — dwMobID, nMobCtrlSN,
flag, nAction, ti, multiTargetForBall, randTimeForAreaAttack, moveFlags,
hackedCode, flyCtxTargetX/Y, hackedCodeCRC, Flush @`0x6e9423`, then bChasing,
hasTarget, bChasing2, bChasingHack, tChaseDuration — matches Atlas field for
field, so the existing `|| JMS` header gates (`movement.go:74,80,83,87,93`) are
independently confirmed correct. Scope is tail-only.

No live capture obtained; decompile-confirmed, NOT live-confirmed.

## 3. pet/serverbound/movement.go — tail missing (decompile-confirmed)

`CVecCtrlPet::EndUpdateActive` @`0xaa25ab`: `EncodeBuffer(petId, 8)`
@`0xaa25fc` then Flush @`0xaa2609`. Header matches Atlas. Tail-only.

No live capture obtained; decompile-confirmed, NOT live-confirmed.

## 4-5. summon + dragon — NOT AFFECTED

`summon/serverbound/move.go:72-85` and `dragon/serverbound/move.go:60-68` both
read `rawMovement = r.ReadBytes(r.Available())` — an opaque slurp that already
swallows the tail and rebroadcasts it faithfully. Nothing parses past the
identity field, so there is nothing to misalign. Senders
`CVecCtrlSummoned::EndUpdateActive` @`0xaa5fc6` and
`CVecCtrlDragon::EndUpdateActive` @`0xa91786` match Atlas's field order. No fix.

## Where the fix must go

`model.Movement` is imported by BOTH directions for character, monster, npc and
pet (`{character,monster,npc,pet}/clientbound/*`). The tail is serverbound-only,
so each fix goes in the serverbound wrapper after `m.movement.Decode(...)`
returns — the pattern already used by `moveKeyPadTail` in
`character/serverbound/move.go`. Never inside `model.Movement`.

## Same false-verified class as the character cell

`monster/.../movement_test.go:20` and `pet/.../movement_test.go:13` both carry
`packet-audit:verify ... version=jms_v185` markers backed only by
`test.RoundTrip`. A round-trip is symmetric and cannot see a field the client
sends and the decoder never reads — the same reason the character Move and
melee attack cells held ✅ while broken.

## Environment note

The channel pod named in the earlier evidence is gone; the running pod at the
time of this audit was `atlas-channel-576848586c-6sfm2`. Log sweeps for
MonsterMovementHandle / PetMovementHandle timed out, which is why those two are
decompile-only. The NPC frame above came from a log snapshot already captured
earlier in this session.
