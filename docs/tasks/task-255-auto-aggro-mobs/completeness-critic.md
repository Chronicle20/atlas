# Completeness critic — task-255-auto-aggro-mobs

**Verdict: CLEAN** — 0 CHANGED-BUT-UNCLAIMED findings, 0 CLAIMED-BUT-UNVERIFIED findings.

- Branch: `task-255-auto-aggro-mobs` (confirmed via `git branch --show-current`)
- Diff base: `BASE=$(git merge-base origin/main HEAD)` = `d17404dbc23588202d2dae89173832f5cab96984`, HEAD = `b72b83c6260af78396c1ab659337ef4b6e66c30a`
- Manifest: `docs/tasks/task-255-auto-aggro-mobs/coverage-manifest.yaml` — declares `ops: [AUTO_AGGRO]` over all ten catalog versions, resolving to `monster/serverbound/MonsterAutoAggro`.

## Step 1 — resolved claim

`claimedPackets` = `{monster/serverbound, monster/serverbound/MonsterAutoAggro}`.
`claimedOps` = `AUTO_AGGRO × {gms_v48, gms_v61, gms_v72, gms_v79, gms_v83, gms_v84, gms_v87, gms_v92, gms_v95, jms_v185}` (10 cells).
`outOfScope` = `{monster-aggro-lease, SET_AGGRO, auto-aggro-proximity-gate}` (server-side atlas-monsters/atlas-channel domain logic and Kafka messaging, no codec).

## Step 2 — CHANGED-BUT-UNCLAIMED

**Touched codecs.** `git diff --name-only $BASE...HEAD -- 'libs/atlas-packet' | grep '\.go$' | grep -v '_test\.go$'` returns exactly one file:

```
libs/atlas-packet/monster/serverbound/auto_aggro.go
```

This is the declared `monster/serverbound` packet dir — CLAIMED. No other `libs/atlas-packet` source file was touched.

**Touched version gates.** `git diff $BASE...HEAD -- 'libs/atlas-packet' | grep -E '^[+-].*(MajorVersion|MajorAtLeast|IsRegion|Region\(\))'` returned only:

```
+			ctx := pt.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
```

— a line inside `auto_aggro_test.go`'s version-matrix test loop (table-driven test iterating all ten versions), not a gate on the codec itself. `auto_aggro.go` itself contains no `MajorVersion`/`MajorAtLeast`/`IsRegion`/`Region()` gate — the file's own doc comment confirms "IDENTICAL across all ten versions; no version gate," matching the manifest's `fields` note ("no version gates on the wire layout"). No unclaimed gate change.

**Matrix delta.** `git diff $BASE...HEAD -- docs/packets/audits/status.json` (ignoring `exportHashes`/`toolSha`) shows exactly one row added (`AUTO_AGGRO`, `monster/serverbound/MonsterAutoAggro`, all 10 cells `verified`) and the corresponding pre-existing stub row (same `op: AUTO_AGGRO`, no `packet` key, all cells `n-a`/`incomplete`) removed. No other row in `status.json` changed state. This is exactly the declared op — CLAIMED, no scope hole.

**Non-codec surface touched by the branch** (registry/ida-exports/seed-template/service files) is consistent with wiring the single declared opcode:
- `docs/packets/registry/*.yaml` — promotes the existing `CMob::ApplyControl` entries from `provenance: csv-import` to `ida-discovered` with a task-255 note; same op, same opcode, no new op introduced.
- `services/atlas-configurations/seed-data/templates/template_*_1.json` — adds one dispatcher-template row per version routing opcode `0xBD`(etc.)/`AutoAggro`/`CMob::ApplyControl` to the `channel` service — the seed-template routing the task description calls out, for the same single declared op.
- `services/atlas-channel/...` and `services/atlas-monsters/...` Go changes (aggro gate, SET_AGGRO Kafka consumer/producer, aggro lease/registry) — all covered by the manifest's `out_of_scope` list (`monster-aggro-lease`, `SET_AGGRO`, `auto-aggro-proximity-gate`); none of these touch `libs/atlas-packet`.

## Step 3 — CLAIMED-BUT-UNVERIFIED

All ten `claimedOps` cells were checked against the final (HEAD) `docs/packets/audits/status.json`:

| op | version | HEAD state |
|---|---|---|
| AUTO_AGGRO | gms_v48 | verified |
| AUTO_AGGRO | gms_v61 | verified |
| AUTO_AGGRO | gms_v72 | verified |
| AUTO_AGGRO | gms_v79 | verified |
| AUTO_AGGRO | gms_v83 | verified |
| AUTO_AGGRO | gms_v84 | verified |
| AUTO_AGGRO | gms_v87 | verified |
| AUTO_AGGRO | gms_v92 | verified |
| AUTO_AGGRO | gms_v95 | verified |
| AUTO_AGGRO | jms_v185 | verified |

No CLAIMED-BUT-UNVERIFIED findings.

## Non-blocking observation (not a scope-hole finding)

The manifest's `fields` prose describes the wire layout as "`Decode4(uniqueId) + Decode1(moveActionOrDistanceHint)`", but the shipped codec (`libs/atlas-packet/monster/serverbound/auto_aggro.go`) decodes two `uint32` fields (`mobId`, `distance`) via `ReadUint32()`/`ReadUint32()` — i.e. `Decode4 + Decode4`, not `Decode4 + Decode1`. This is a manifest-prose accuracy note, not a claimed-vs-changed scope mismatch (op/version/direction/packet path all agree), so it does not change the verdict above; flagging for the author to correct the manifest text if desired.
