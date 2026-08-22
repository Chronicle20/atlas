# Completeness critic — task-251-player-npcs

Branch: `task-251-player-npcs` · merge-base `5f299e4bb` · HEAD `9c35b3870`

## Verdict

**CLEAN — 0 findings.** A coverage-manifest now exists (uncommitted
working-tree file, `docs/tasks/task-251-player-npcs/coverage-manifest.yaml`)
declaring `ops: [IMITATED_NPC_DATA, REMOVE_NPC]` (both clientbound), all 10
version keys, with `IMITATED_NPC_DATA` noted `n-a` on `gms_v48`, and
`out_of_scope: []`. The declared scope, the actual codec delta, and the
actual matrix delta agree 1:1 across all three axes checked below.

## Step 1 — manifest resolved

- `claimedPackets`: `npc/clientbound` (both ops resolve to status.json rows
  with `direction: clientbound`, no packet-path split needed — single dir).
- `claimedOps`: `IMITATED_NPC_DATA × {gms_v61, gms_v72, gms_v79, gms_v83,
  gms_v84, gms_v87, gms_v92, gms_v95, jms_v185}` (9 keys — `gms_v48` is
  declared `n-a`, not a verified-coverage claim) and `REMOVE_NPC × {all 10
  keys}`.
- `outOfScope`: empty.

## Step 2 — CHANGED-BUT-UNCLAIMED

**Touched codecs** — independently re-derived, not taken from the
coordinator's manifest comment:

```
$ BASE=$(git merge-base origin/main HEAD)   # 5f299e4bb
$ git diff --name-only $BASE...HEAD -- 'libs/atlas-packet' | grep '\.go$' | grep -v '_test\.go$'
libs/atlas-packet/npc/clientbound/imitated_npc_data.go
libs/atlas-packet/npc/clientbound/remove.go
```

Confirmed exactly two non-test codec files, both under `npc/clientbound`,
both covered by `claimedPackets`. No struct file outside the declared dir
moved. (The full untriaged file list, including tests, is the same two pairs:
`imitated_npc_data.go`/`_test.go` and `remove.go`/`_test.go` — no third
file.)

**Touched version gates:**

```
$ git diff $BASE...HEAD -- 'libs/atlas-packet' | grep -E '^[+-].*(MajorVersion|MajorAtLeast|IsRegion|Region\(\))' | grep -v '^[+-][+-]'
+			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
+			ctx := test.CreateContext(v.Region, v.MajorVersion, v.MinorVersion)
```

Both hits are test-harness boilerplate (`test.CreateContext(v.Region, ...)`
inside the new `_test.go` files' per-version table loop), not a gate
condition (`MajorAtLeast`/`IsRegion`/`Region()` branch) in the codec itself.
Read of the two new source files (`imitated_npc_data.go`, `remove.go`, full
diff inspected) confirms neither contains a `MajorVersion`/`MajorAtLeast`/
`IsRegion` branch — `Remove`'s doc comment explicitly records the read shape
(`Decode4`, no further reads) was confirmed *identical* across all ten
versions, i.e. no per-version branching exists to gate. No gate change to
flag.

**Matrix delta:**

```
$ git diff $BASE...HEAD -- docs/packets/audits/status.json   # excluding toolSha/exportHashes
```

Exactly two rows changed, both `incomplete` ("no audit report") → the states
shown in Step 3 below: `IMITATED_NPC_DATA` (clientbound) and `REMOVE_NPC`
(clientbound). No other row's cells moved. Both rows are in `claimedPackets`.

No CHANGED-BUT-UNCLAIMED findings.

## Step 3 — CLAIMED-BUT-UNVERIFIED

Final (HEAD) `status.json` cells for the claimed pairs:

```
IMITATED_NPC_DATA (clientbound):
  gms_v48: n-a        (manifest declares this n-a, not a coverage claim — matches)
  gms_v61: verified
  gms_v72: verified
  gms_v79: verified
  gms_v83: verified
  gms_v84: verified
  gms_v87: verified
  gms_v92: verified
  gms_v95: verified
  jms_v185: verified

REMOVE_NPC (clientbound):
  gms_v48: verified
  gms_v61: verified
  gms_v72: verified
  gms_v79: verified
  gms_v83: verified
  gms_v84: verified
  gms_v87: verified
  gms_v92: verified
  gms_v95: verified
  jms_v185: verified
```

All 19 claimed pairs are `verified`; the one declared-`n-a` cell
(`IMITATED_NPC_DATA` × `gms_v48`) matches the manifest's note exactly, so it
is not a silent mismatch. No CLAIMED-BUT-UNVERIFIED findings.

## Tables

### CHANGED-BUT-UNCLAIMED

| kind | file-or-packet | evidence | recommendation |
|---|---|---|---|
| — | — | none found | — |

### CLAIMED-BUT-UNVERIFIED

| op | version | actual state | recommendation |
|---|---|---|---|
| — | — | none found | — |
