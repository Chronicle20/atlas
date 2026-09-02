# Brief — stop rebroadcasting the raw v87 move-path for summon and dragon

## Why

Commit `540929015` on this branch fixed the GMS v87 `CMovePath` asymmetry for
every movement packet that goes through `model.Movement`: the v87 client WRITES
the per-element `XOffset`/`YOffset` pair on absolute-position (NORMAL) fragments
(`CMovePath::Encode` @0x6c70fe, `mov ax,[edi+14h]` / `[edi+16h]` at 6c720a /
6c7218) but NEVER READS it back (`CMovePath::Decode` @0x6c6e86 — 154
instructions, ends 0x6c709a, absolute arm goes from `fh` and the attr-15
`fhFallStart` straight to the `bMoveAction`/`tElapse` tail). So Atlas must read
the pair off a v87 client and must not write it back. `model.Movement` now has
two gates for this: `movementElementOffsetsInbound` (`!GMS || MajorAtLeast(87)`)
and `movementElementOffsetsOutbound` (`!GMS || MajorAtLeast(92)`).

Summon and dragon movement never go through that codec. They capture the
client's move-path as an opaque `[]byte` and rebroadcast it VERBATIM to the
other sessions in the map. On a v87 tenant that blob still carries the pair, so
the observing clients desync exactly as NPCs did — the receiving
`CMovePath::Decode` reads `xOffset`'s low byte as `bMoveAction` and `yOffset` as
`tElapse`, then reads the real `bMoveAction`/`tElapse` as the next fragment's
attr and body. Summons and dragons teleport for everyone except their owner
(the owner renders locally and is not sent the packet).

This is the failure the source note that started this investigation describes:
"if your source simply copies packet contents for NPC animation packets, that
won't work. You have to parse the packet and then serialize it again the way the
client expects it."

## Required approach

Re-serialize at ENCODE time, inside the two clientbound codecs. Do NOT change
the Kafka message contract and do NOT change the `[]byte` that crosses it.

In `SummonMove.Encode` and `DragonMove.Encode`, decode `rawMovement` into a
`model.Movement` using the `options` already passed to `Encode`, then write
`movement.Encode(...)` output in its place. That inherits the inbound/outbound
gate split for free, needs no new version logic, and keeps the change inside
`libs/atlas-packet`.

Two things to get right:

- The decode side must use the tenant's own `types` table from `options`. That
  is the same table `model.Movement.Decode` already consumes; it is how a
  fragment attr resolves to NORMAL. Without it every fragment falls back to a
  bare `model.Element` and the re-encode silently truncates.
- Decoding a blob that was captured from THE SAME tenant is the only supported
  case here — the blob is produced by that tenant's client and consumed by that
  tenant's clients. Do not add cross-version translation.

If, while implementing, the re-encode turns out to need information the opaque
blob does not carry, STOP and report rather than inventing a field. Note the
trailing keypad-state + `m_rcMove` block: the client's `CMovePath::Encode`
always writes it, but v87's `CMovePath::Decode` has no `bPassive` parameter and
never reads it, and `model.Movement.Decode` does not consume it either. Confirm
what the existing `rawMovement` blob actually contains before deciding whether
the re-encoded form should reproduce it. On v83/v92/v95/JMS `CMovePath::Decode`
DOES read that trailer when `bPassive` is set — establish from the IDB whether
`bPassive` is 1 at `CSummonedPool::OnMove` / `CDragon::OnMove` before changing
behaviour on those versions. A wrong answer here regresses versions that work
today.

## Files

Codecs (the change belongs here):
- `libs/atlas-packet/summon/clientbound/move.go` — `SummonMove`, holds
  `rawMovement []byte`, `Encode` writes cid, oid, blob.
- `libs/atlas-packet/dragon/clientbound/move.go` — `DragonMove`, holds
  `rawMovement []byte`, `Encode` writes ownerCharacterId, blob. Carries
  `packet-audit:fname CDragon::OnMove`.

Serverbound capture (read for context; probably unchanged — inbound decode of
the pair is already correct because the blob is not reinterpreted):
- `libs/atlas-packet/summon/serverbound/move.go` — `Move.Decode` stores
  `rawMovement` and extracts startX/startY from its first 4 bytes.
- `libs/atlas-packet/dragon/serverbound/move.go` — same shape.

Existing tests to update:
- `libs/atlas-packet/summon/clientbound/move_test.go`,
  `libs/atlas-packet/summon/clientbound/v61_test.go`
- `libs/atlas-packet/dragon/clientbound/move_test.go`
- `libs/atlas-packet/summon/serverbound/move_test.go`,
  `libs/atlas-packet/dragon/serverbound/move_test.go`

Channel plumbing (read only — should NOT need changing if the approach above
holds; if you find it must change, that is a finding to report):
- `services/atlas-channel/atlas.com/channel/socket/writer/summon.go:24`,
  `writer/dragon.go:18`
- `services/atlas-channel/atlas.com/channel/socket/handler/summon_move.go:29`,
  `handler/dragon_move.go:29`
- `services/atlas-channel/atlas.com/channel/{summon,dragon}/processor.go` —
  `Move(...)` takes `rawMovement []byte`
- `services/atlas-channel/atlas.com/channel/kafka/message/{summon,dragon}/kafka.go`
  — the blob crosses Kafka. Leave the contract alone.

## Reference

- `libs/atlas-packet/model/movement.go` — the two gates and the full per-version
  evidence table live in the comment above them. Read it first; it has the
  addresses for every version.
- `docs/tasks/fix-v87-movement-xoffset-encode/diagnosis.md` — the root cause.
- `docs/tasks/fix-v87-movement-xoffset-encode/review.md` — the review that
  raised this; see the first non-blocking finding.

## Verification

- `cd libs/atlas-packet && go test ./...` must pass.
- A test must prove the v87 emitted summon and dragon blobs no longer carry the
  pair, and that v83/v92/v95/JMS are byte-unchanged. `libs/atlas-packet/model/
  version_bounds_test.go` is the pattern to copy — assert against the
  pre-existing layout, not against the new code's own output.
- `go run ./tools/packet-audit matrix --check`, `fname-doc --check`,
  `operations --check`, `dispatcher-lint` must all exit 0 from the repo root.
- Do NOT run `tools/verify.sh` yourself; the controller dispatches that
  separately.

## Scope

Module-local. Do not touch `model/movement.go`'s gates, the NPC/monster/pet/
character movement packets, or anything already committed on this branch.
Work in the existing worktree
`.worktrees/fix-v87-movement-xoffset-encode` on branch
`fix-v87-movement-xoffset-encode`. Commit your work; leave the tree clean.
