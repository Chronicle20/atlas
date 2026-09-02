# Review — `540929015` fix(movement): stop echoing XOffset/YOffset to GMS v87 clients

- **Unit:** single commit `540929015`, `origin/main..HEAD` (7 files, +288/-61).
- **Brief:** `docs/tasks/fix-v87-movement-xoffset-encode/diagnosis.md`.
- **Verdict:** APPROVED_WITH_FINDINGS.

Scope reviewed: the diff itself, plus the contracts it depends on — every
production call site of `model.Movement.Encode`, the v87 tenant `types` table,
`libs/atlas-socket/request/reader.go` bounds behaviour, `deploy/k8s/base/versions.json`,
`docs/packets/audits/OPAQUE_LEDGER.md`, the gms_v87 audit records, and the
packet-audit marker tooling. No sibling-package survey.

---

## 1. Direction correctness — PASS

Every production writer of `model.Movement` is clientbound.

- Encode call sites (`grep model\.Movement`, non-test): clientbound
  `character/clientbound/movement.go:34`, `monster/clientbound/movement.go:70`,
  `pet/clientbound/movement.go:37`, `npc/clientbound/action.go:45`; serverbound
  `character/serverbound/move.go:88`, `monster/serverbound/movement.go:91`,
  `pet/serverbound/movement.go:42`, `npc/serverbound/action.go:49`.
- The four serverbound `Encode` methods are **unreachable from production**: the
  `movement` field is unexported and none of those four files declares a
  constructor (`grep "func New"` on all four returns nothing), so only the
  package's own tests can build one with elements. All four handlers use them
  Decode-only: `socket/handler/character_move.go:19`,
  `socket/handler/monster_movement.go:17`, `socket/handler/pet_movement.go:18`,
  `socket/handler/npc_action.go:27`.
- The relay path re-encodes into clientbound packets only:
  `movement/processor.go:88` (`charpkt.NewCharacterMovement`), `:138`
  (`npcpkt.NewNpcActionMove`), `:161` (`petpkt.NewPetMovement`), and the monster
  arm at `:209`.
- No other producer of the pair exists anywhere: `grep XOffset` outside
  `model/movement.go` and tests returns only a doc comment in
  `libs/atlas-packet/test/roundtrip.go:40`.

No serverbound production path is affected. PASS.

## 2. Other element types / FALL_DOWN ordering — PASS

- Only `NormalElement` touches the pair. `TeleportElement`
  (`movement.go:245`/`:346`), `StartFallDownElement` (`:253`/`:357`),
  `FlyingBlockElement` (`:264`/`:369`), `JumpElement` (`:274`/`:382`),
  `StatChangeElement` (`:284`/`:393`) reference neither offset in either
  direction, matching the client, where the pair lives only on the
  absolute-position arm.
- The tenant table confirms which attrs are absolute on v87:
  `services/atlas-configurations/seed-data/templates/template_gms_87_1.json`
  maps exactly indices 0 (`NORMAL`), 5 (`HANG_ON_BACK`), 15 (`FALL_DOWN`),
  17 (`WINGS`) to `Type: NORMAL`, and all 9 `types` arrays in that template are
  byte-identical — i.e. exactly the 0/5/15/17 set the change claims.
- FALL_DOWN ordering is unchanged and still correct: `fh` → `fhFallStart` →
  pair → tail, on both sides (`movement.go:229-241` decode, `movement.go:328-343`
  encode). Independently corroborated by the pre-existing struct-offset table in
  `docs/tasks/task-191-v92-v95-movement-types/movement-types-derivation.md:189`
  (`+12 fh, +14 fhFallStart, ... +20 xOffset, +22 yOffset`) and the v92 NORMAL
  arm on the following line, whose encode/decode both do `+14` under the
  `nAttr == 12` branch and then `+20`/`+22`.
- That same line also independently confirms the commit's *correction* of the
  old comment: `+24 usRandCnt, +26 usActualRandCnt`, i.e. `+0x18/+0x1A` are the
  rand counters, not the offsets. The removed "runtime option" caveat was indeed
  misattributed.

## 3. Boundary 92 — claim VERIFIED, residual note

`deploy/k8s/base/versions.json:5-15` lists GMS 12, 48, 61, 72, 79, 83, 84, 87,
92, 95 and JMS 185. There is no GMS between 87 and 92, so `MajorAtLeast(92)`
and `MajorAtLeast(88)` are behaviourally identical for every shipped version.
The comment's claim is accurate.

Residual (non-blocking): `movement.go` now carries three different GMS
boundaries — inbound offsets 87 (`:88`), `StartVx/StartVy` 88 (`:129`, `:305`),
outbound offsets 92 (`:92`). A hypothetical GMS 88–91 tenant would get a
combination no evidence covers (writes `StartVx` but not the pair). Choosing 92
makes 88–91 behave like v87 outbound; choosing 88 would make them behave like
v92. Neither is evidence-backed, and the comment says so honestly. Only bites if
someone adds an 88–91 tenant.

## 4. Test changes — legitimate carve-out, with two holes

**The skipped assertion was already vacuous.** Verified from the reader, not
assumed: `libs/atlas-socket/request/reader.go:64` (`ReadInt16` requires
`len-pos > 1`, else returns 0 *without advancing*) and `:32` (`ReadByte`
likewise). Walking a v87 clientbound `CharacterMovement` blob: 4 header + 1
count + 1 attr + 13 element = 19 bytes; decode consumes 10 for x/y/vx/vy/fh
(pos 16), the offset pair then eats the real `bMoveAction` + the low byte of
`tElapse` (pos 18) and returns 0 for the second read, and the tail eats the last
byte and returns 0 — `Available() == 0`. So the pre-change `RoundTrip` on v87
passed while asserting nothing. Removing it loses no real coverage.

**The byte oracle claim holds.** `model/version_bounds_test.go`:
- `TestNormalElementMovementVersionBoundary` now asserts
  `bytes.Equal(encode(87), v83)` — full byte equality, not just length — and
  `len(encode(92)) == len(v83)+4`, `encode(95) == encode(92)`.
- `TestNormalElementOffsetsAreDirectional` pins *both* directions per version
  (83/84/87/92/95/JMS185): encode width 13 vs 17, and a hand-built inbound
  buffer that must be fully consumed with `BMoveAction==8 / TElapse==10`.
  This is genuinely fail-first: under the old symmetric `>=87` gate the v87 case
  encodes 17 bytes against `wantLen` 13 and fails.
- Pre-existing `model/movement_test.go:218 TestMovementV87NormalElementCarriesOffsets`
  is decode-only against a live captured frame, so it remains valid and still
  pins the inbound side against real bytes.
- `model/movement_test.go:120 TestMovementHeaderRoundTrip` still uses
  `test.RoundTrip` at v87, which is safe because its `Movement` has no elements.

Holes:
- **Character** keeps real v87 coverage (`character/clientbound/movement_test.go:84-92`
  pins the 4-byte prefix, blob equality and total length before the skip).
  **Monster and pet do not.** `monster/clientbound/movement_test.go:36`,
  `:48` and `pet/clientbound/movement_test.go:36` are the only tests those
  packets have at v87 (`test.Variants` includes GMS v87 at
  `libs/atlas-packet/test/context.go:21`; their other byte tests are v72/v79
  only). After the change, the GMS v87 subtest of each executes zero assertions.
- The helper's own guidance is wrong: `libs/atlas-packet/test/roundtrip.go:64-65`
  says "Use `RoundTrip` directly for serverbound packets — there Decode is the
  production path and the identity holds on every version." It does not: on v87
  the shared `NormalElement.Encode` omits the pair that `Decode` reads, in any
  direction. It only holds today because every serverbound movement round-trip
  passes `nil` options and no NORMAL element
  (`monster/serverbound/movement_test.go:58`, `npc/serverbound/action_test.go:53`).
  A future serverbound test that adds an element at v87 will trip the same
  silent over-read this commit documents.
- No test covers the v87 outbound FALL_DOWN shape (`fhFallStart` present, pair
  absent); the directional test uses `ElemType: 0` with `nil` options.

## 5. Audit / coverage records — checks pass, but passing is not sufficient

- The evidence records are unaffected *by construction*:
  `docs/packets/audits/gms_v87/CharacterMovement.md` / `.json` model the move
  path as a single opaque `bytes` row ("CMovePath::OnMovePacket — opaque
  movement path block"), so a field-level change inside the blob cannot
  invalidate a row. Same for MonsterMovement / PetMovement. Nothing there needed
  editing.
- But the justification for that opacity did change.
  `docs/packets/audits/OPAQUE_LEDGER.md:36` grants the `mob move-path`
  VERIFIED-EXCEPTION on the strength of "`monster/clientbound/movement_test.go`,
  `monster/serverbound/movement_test.go` (byte-for-byte)". For GMS v87 the
  clientbound half of that citation is now a no-op. The ledger row was not
  updated.
- The `packet-audit:verify ... version=gms_v87` markers at
  `monster/clientbound/movement_test.go:32` and
  `pet/clientbound/movement_test.go:32` now sit on tests that skip v87.
  `matrix --check` cannot detect this: `tools/packet-audit/internal/marker/marker.go:28`
  scans `_test.go` files for the comment prefix and only validates syntax and
  duplicate `(packet,version)` keys. So "matrix/fname-doc/operations/dispatcher-lint
  all pass" is **not** sufficient evidence that v87 is still covered.
- The "Encode/Decode gates must be textually identical" convention
  (`docs/tasks/task-191-v92-v95-movement-types/context.md:189`,
  `plan.md:49`) is deliberately broken here, correctly and with a test that
  pins the asymmetry. It is a doc-only convention — no guard script references
  it (`grep "textually identical"` finds no tooling), so nothing failed to be
  updated mechanically. The exception is recorded only in `movement.go`'s
  comment and in `diagnosis.md`, not in any packet-process doc.
- No stale references to the old identifier: `grep gmsMovementElementOffsets`
  finds only `diagnosis.md:16`.

## 6. Adjacent path the fix does not cover (non-blocking, but real)

Summon and dragon movement never go through `model.Movement`: the serverbound
handler stores the whole body as opaque bytes
(`libs/atlas-packet/summon/serverbound/move.go:76`) and the channel rebroadcasts
it verbatim (`socket/writer/summon.go:24`, `socket/writer/dragon.go:18`,
consumed at `kafka/consumer/summon/consumer.go:113` and
`kafka/consumer/dragon/consumer.go:133`). On GMS v87 the client's own
`CMovePath::Encode` writes the pair, so those rebroadcast blobs still carry 4
bytes per absolute fragment that the receiving v87 client does not read — the
same desync class this commit fixes for characters, NPCs, mobs and pets. Not
introduced by this diff and not the reported symptom (NPC walk paths), so not
blocking this unit, but v87 summon/dragon movement should be expected to remain
wrong until a follow-up rewrites those blobs instead of echoing them.

## 7. Verification run

`go test ./model/... ./character/... ./monster/... ./pet/... ./npc/...` in
`libs/atlas-packet` — all `ok`. Green build is not itself an approval; see §4-5.

## Not evaluable

1. **The disassembly claims themselves.** `CMovePath::Decode` @0x6c6e86 not
   reading `+0x14/+0x16` on GMS v87, and the `mov ax,[edi+14h]`/`[edi+16h]` at
   6c720a/6c7218, are the entire basis for the change. This environment has no
   IDB; `docs/packets/ida-exports/gms_v87.json` contains call-graph entries only
   (`address` / `direction` / `calls`), no decompiled bodies. The v92/v95
   structure is corroborated by `movement-types-derivation.md:189`, and the v87
   *inbound* side by the live-frame test, but the v87 *read-side absence* is
   accepted on the author's report alone.
2. **Live confirmation on tenant GMS 87.1.** No field capture post-change is
   available here; the reported symptom (NPC teleporting) is not verified fixed.
