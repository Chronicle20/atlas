# Bug — client closes with "error 38": truncated CharacterDamage packet for mob attack index >= 1

Environment: `atlas-pr-1456` (PR for task-254), tenant `27401cd5-79a5-430b-8529-77ed6953d86e`,
GMS **83.1**, world 0, channel 0. Accounts `Atlas` (id 1, character 1) and `Chronicle`
(id 2, character 2), partied, both in map **240020200**, fighting monster template
**8140101**.

Status: **root cause established.** Not a task-254 regression — the defect is in
`libs/atlas-packet` and predates the branch.

## Reproduced

Not on demand. Observed twice in one ~15-minute live session, both captured in
`atlas-channel` logs (2026-08-26) and Tempo traces. The trigger condition is rare in
casual play, which is why it took a boss with a multi-attack table to surface.

## Observed

Two abrupt client closes. In each, the client that died is **not** the character that
took damage — it is another character in the same map:

| # | Damage event | Damaged char | `nAttackIdx` | Damage | Client that closed | Close |
|---|--------------|--------------|--------------|--------|--------------------|-------|
| 1 | 20:43:26.899Z | 1 (`Atlas`) | **1** | 1335 | 2 (`Chronicle`), session `485d5832` | 20:43:27.509Z |
| 2 | 20:44:55.440Z | 2 (`Chronicle`) | **1** | 1279 | 1 (`Atlas`), session `08c3c8f7` | 20:44:56.063Z |

Each damage event produced a `CharacterDamage` (`session.Announce`, Tempo traces
`6f4cd888efb7d47f832e2565727c850f` and `ebb7e211356dc6c5d449d7d1de00e9f0`) broadcast to
the *other* sessions in the map ~2 ms later; the recipient closed ~0.6 s afterwards.
No server-side error, warn, or panic; no span recorded an error. The client closed the
socket.

### Sweep (not a spot-check)

Across the whole 15-minute log there were **51** `CharacterDamageHandle` reads:

| `nAttackIdx` | Count | Encoded correctly? | Closes |
|---|---|---|---|
| -3 (`DamageTypeObstacle`) | 1 | yes (block correctly omitted) | 0 |
| -1 (`DamageTypePhysical`) | 29 | yes | 0 |
| 0 (`DamageTypeMagic`) | 3 | yes | 0 |
| **>= 1 (mob attack index)** | **2** | **no — truncated** | **2** |

Both, and only both, `nAttackIdx >= 1` events are followed by the other client's close.
There are no other unexplained closes in the log. The mob's own log line confirms the
source: `Monster [1000005] using basic attack pos [1]` and `Monster [1000038] using
basic attack pos [1]` immediately precede the two events; every other mob attack in the
window was `pos [0]`.

## Expected

`CharacterDamage` (`DAMAGE_PLAYER`, op `0xC0` on v83, `CUserRemote::OnHit`) is encoded
with the field set the client reads, for every `nAttackIdx > -2`.

## Root cause

`libs/atlas-packet/character/clientbound/damage.go:52`

```go
if m.attackIdx == model.DamageTypePhysical || m.attackIdx == model.DamageTypeMagic {
```

`DamageTypePhysical = -1` and `DamageTypeMagic = 0`
(`libs/atlas-packet/model/damage_taken_info.go:19-20`), so the block is written **only
for exactly -1 or 0**. The client's condition is `nAttackIdx > -2` — i.e. every value
from -1 upward, including all mob attack indices `>= 1`
(`docs/packets/ida-exports/gms_v83.json`, `CUserRemote::OnHit` @ `0x9832e3`:
`ptHit=monsterTemplateId (only if nAttackIdx > -2)`, `bLeft (only if nAttackIdx > -2)`,
`v20: power guard flag (only if nAttackIdx > -2)`, `v4: stance action (always, after
power guard block, if nAttackIdx > -2)`).

The serverbound sibling already gets this right:
`libs/atlas-packet/model/damage_taken_info.go:122` and `:174` both use
`if m.nAttackIdx >= DamageTypePhysical`, which is exactly `> -2`.

For `nAttackIdx = 1` the server therefore emits a 13-byte body
(`cid(4) + attackIdx(1) + damage(4) + damageRepeat(4)`) where the client reads 20
(`cid(4) + attackIdx(1) + damage(4) + mobTemplateId(4) + bLeft(1) + pgFlag(1) +
stance(1) + damageRepeat(4)`). The client consumes the repeated-damage int as
`ptHit=monsterTemplateId` and then reads **past the end of the packet** — the
`Decode1` for `bLeft`, which is the second `Decode1` in `CUserRemote::OnHit`, and the
first out-of-bounds read. That is the fault the user pinned in the client
(`CUserRemote::OnHit`, second `Decode1`). The client aborts the socket, which surfaces
as the "error 38" dialog.

Version scope: the `Decode` shape and the `> -2` guard are identical in every
checked-in export — v48 `0x6bea8d`, v61 `0x7cb9ff`, v72 `0x88c5ad`, v79 `0x8d9489`,
v83 `0x9832e3`, v84 `0x9c3681`, v87 `0xa08d57`, v92 `0x931350`, v95 `0x954c50`,
jms185 `0xa56e2e`. **All versions are affected.**

Why the tests never caught it: `Decode` at `damage.go:75` carries the *same* wrong
predicate, so encode/decode round-trips are self-consistent and green. The verified
matrix cell (`docs/packets/audits/STATUS.md:259`, v83 `0x0C0` ✅) was pinned with
`nAttackIdx` of -1/0 only.

## Fix

- `libs/atlas-packet/character/clientbound/damage.go:52` — replace the equality pair
  with `if m.attackIdx >= model.DamageTypePhysical {`, matching
  `damage_taken_info.go:122` and the client's `> -2`.
- `libs/atlas-packet/character/clientbound/damage.go:75` — same change on `Decode`, so
  the round trip stays symmetric.
- `libs/atlas-packet/character/clientbound/damage_test.go` — add coverage for
  `attackIdx = 1` (and a boundary case at `DamageTypeCounter = -2`, which must still
  omit the block). Existing cases only exercise -1/0.
- `libs/atlas-packet/character/clientbound/v61_test.go`, `v72_test.go`, `v79_test.go` —
  same boundary cases per version fixture where those files assert the layout.
- Re-pin the `DAMAGE_PLAYER` evidence/fixture if the fixture bytes change
  (`docs/packets/audits/STATUS.md:259` row; regenerate the matrix rather than editing
  the row by hand).

Use the project's Builder pattern for test setup; do not add test-only constructors.

## Not yet answered

1. **Possible second defect, same file, not fixed by the above.** `damage.go:56` gates
   the extra stance byte on `t.Region() == "GMS" && t.MajorVersion() >= 95`, but the
   v87 export (`0xa08d57`) also lists a `Decode1 | stance flags` after the stance
   action, the same as v95. If that annotation is right, v87 is one byte short. The v87
   cell is ✅ in the matrix, so either the annotation post-dates the verification or the
   fixture does not reach that field. Confirm against the v87 IDB before changing the
   gate — do not widen it on the strength of the export comment alone.
2. Whether any other clientbound codec re-derives this predicate as an equality pair
   rather than `>= DamageTypePhysical`. Worth a grep for
   `DamageTypeMagic` / `DamageTypePhysical` comparisons while in the file.

## Ruled out (recorded so the next session does not re-walk them)

- **`0xDF` is a red herring.** `Read a unhandled message with op 0xDF` appears
  throughout the log on sessions that keep running (20:32:17, 20:33:07, 20:39:10,
  20:46:53, …). It is `PARTY_SEARCH_UPDATE` / `CWvsContext::SendCancelPartyWanted`,
  unregistered in `template_gms_83_1.json` (handlers jump 0xDA → 0xE4) and harmlessly
  dropped at `libs/atlas-socket/server.go:332`. It is routine client chatter, not a
  crash signal. (An earlier revision of this file treated it as the shutdown packet.)
- **`PartyMemberHP` is not the culprit.** It was the last non-movement write before both
  closes, but it also fires repeatedly with no close, its opcode (`0xC9`) and layout
  match the CSV and the verified matrix cell, and the crashing and non-crashing traces
  are span-for-span identical.
- **Session `246d9087` ending at 20:37:03.839Z is not this bug** — normal
  cash-shop-return handoff into session `485d5832` on the same character.
- **Send-IV corruption from concurrent writes.** Every write goes through
  `Model.announceEncrypted` (`services/atlas-channel/.../session/model.go:123`) holding
  a shared `*sync.Mutex` across encrypt and `con.Write`; `CloneSession` preserves the
  mutex pointer and the aliased IV backing array.
- **Encoder buffer reuse across recipients.** `Announce` invokes `encoder(l, ctx)` once
  per recipient, so each announce allocates a fresh `response.Writer`.

## Incidental findings (not this bug, not in scope)

- `session/model.go:73` `CloneSession` does not copy `gm`, so the GM flag is dropped by
  every cloning setter (`setMapId`, `setCharacterId`, …).
- `session/model.go:127-128` allocates `tmp` and copies into it, then discards it via
  `tmp = append([]byte{0,0,0,0}, b...)`. Dead code.

## Resolution

Pending. Fill in the fix commit and the live re-test result (a mob using attack index
>= 1 while a second character stands in the map) once landed.
