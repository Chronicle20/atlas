# Completeness Critic — task-176-gm-hide-controller-relinquish

**Verdict: CLEAN — 0 findings.**

Branch range: `be0c9433823c5115b90842f210793f455f64a6a8..HEAD` (merge-base with
`origin/main`).

## Step 1 — manifest resolution

`docs/tasks/task-176-gm-hide-controller-relinquish/coverage-manifest.yaml`:

```yaml
ops:
  - SPAWN_NPC_REQUEST_CONTROLLER
versions:
  - gms_v83
  - gms_v84
  - gms_v87
  - gms_v95
  - jms_v185
fields:
  - "npc/clientbound/RemoveController: new flag-0 (remove) arm of CNpcPool::OnNpcChangeController; ..."
out_of_scope: []
```

`op: SPAWN_NPC_REQUEST_CONTROLLER` resolves in `docs/packets/audits/status.json`
to a single row:

```json
{
  "kind": "op",
  "op": "SPAWN_NPC_REQUEST_CONTROLLER",
  "packet": "npc/clientbound/NpcSpawnRequestController",
  "direction": "clientbound",
  ...
}
```

→ `claimedPackets = { npc/clientbound }` (and the specific packet path
`npc/clientbound/NpcSpawnRequestController`). `claimedOps = { SPAWN_NPC_REQUEST_CONTROLLER
× {gms_v83, gms_v84, gms_v87, gms_v95, jms_v185} }`. `outOfScope = {}`.

## Step 2 — CHANGED-BUT-UNCLAIMED

**Touched codecs.**

```
$ git diff --name-only $BASE...HEAD -- 'libs/atlas-packet' | grep '\.go$' | grep -v '_test\.go$'
libs/atlas-packet/npc/clientbound/remove_controller.go
```

Dir `npc/clientbound` matches `claimedPackets` — CLAIMED. No unclaimed codec
touch. (No other `libs/atlas-packet` files changed in this branch — full
diffstat: `2 files changed, 95 insertions(+)`, both under
`npc/clientbound/remove_controller*.go`.)

**Touched version gates.**

```
$ git diff $BASE...HEAD -- 'libs/atlas-packet' | grep -E '^[+-].*(MajorVersion|MajorAtLeast|IsRegion|Region\(\))' | grep -v '^[+-][+-]'
(no output)
```

No gate lines added or removed anywhere under `libs/atlas-packet` in this
branch. Consistent with the manifest's own claim ("same opcode as the
verified grant arm; no wire change to any existing codec") and with
`RemoveController.Decode` in `remove_controller.go` reading unconditionally
(`ReadByte()` then `ReadUint32()`, no `MajorAtLeast`/version branch at all —
the new arm is version-flat).

**Matrix delta.**

```
$ git diff $BASE...HEAD -- docs/packets/audits/status.json
(no output)
```

`status.json` is byte-identical to the base — no cell transitioned state.
Consistent with the new code reusing the existing verified op/opcode
(`SPAWN_NPC_REQUEST_CONTROLLER`) rather than introducing a new matrix row.
Confirmed independently via `go run ./tools/packet-audit matrix --check`,
exit 0 (no drift) in this worktree.

## Step 3 — CLAIMED-BUT-UNVERIFIED

HEAD `status.json` row for `SPAWN_NPC_REQUEST_CONTROLLER`, cells for the five
claimed versions:

| version | state (HEAD) |
|---|---|
| gms_v83 | verified |
| gms_v84 | verified |
| gms_v87 | verified |
| gms_v95 | verified |
| jms_v185 | verified |

All five `claimedOps` pairs are `verified`. No claim outstanding.

(Note: the op's row also carries `gms_v48` = `n-a` and `gms_v61`/`gms_v72`/
`gms_v79` = `partial` — none of these versions are in the manifest's
`versions` list, so they are correctly excluded from the claim, not silently
passed.)

## Summary

- CHANGED-BUT-UNCLAIMED: none.
- CLAIMED-BUT-UNVERIFIED: none.
- The single touched file (`libs/atlas-packet/npc/clientbound/remove_controller.go`,
  new type `RemoveController`, reusing `Operation() = NpcSpawnRequestControllerWriter`)
  falls entirely inside the declared `npc/clientbound` scope, introduces no
  version gate, and causes no matrix-cell transition — matching the manifest's
  own characterization of the change as a same-opcode, non-wire-breaking new
  decode arm on an already-verified op across all five declared versions.
