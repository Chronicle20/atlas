# Packet completeness audit — task-241 (Duey/Parcel delivery)

**Verdict: 3 findings** (0 CLAIMED-BUT-UNVERIFIED against status.json op-row
cells as literally graded; 1 high-severity CHANGED-BUT-UNCLAIMED (gate) whose
`out_of_scope` characterization does not match its actual blast radius; 2
coverage-honesty findings where the matrix's "verified" state does not match
what the manifest itself, or the underlying evidence, actually supports).

Base: `d9ec287b8` (confirmed ancestor of `HEAD` via `git merge-base
--is-ancestor`). Branch: `task-241-duey-parcel-delivery` (confirmed via `git
branch --show-current`).

Manifest present at `docs/tasks/task-241-duey-parcel-delivery/coverage-manifest.yaml`.
`ops: [PARCEL, DUEY_ACTION]`, `versions:` the 8 non-n/a keys, `out_of_scope:
[CASHSHOP_OPERATION, model/asset]`.

## CHANGED-BUT-UNCLAIMED

| kind | file/packet | evidence | recommendation |
|---|---|---|---|
| gate | `libs/atlas-packet/model/asset.go` (shared `encodeEquipableInfo`/`decodeEquipableInfo`) | `git diff d9ec287b8...HEAD -- libs/atlas-packet/model/asset.go` (commit `a8adafb12`) adds `if t.IsRegion("GMS") && t.MajorAtLeast(92) { w.WriteByte(0); w.WriteByte(0); w.WriteShort(0)×5 }` (mirrored in decode) — a new 12-byte gate on the shared equip-asset codec. The manifest lists `model/asset` under `out_of_scope` with the note "shared item-slot blob encoded inside Parcel (incidental)". That characterization undersells the change: this is not incidental struct churn, it is a new version-gated wire behavior on a codec used well beyond Parcel — `grep -rl "model\.Asset\b" libs/atlas-packet --include='*.go' \| grep -v _test.go` returns 19 files, including `character/data.go`, `inventory/clientbound/change.go`, `inventory/change_entry.go`, `storage/clientbound/show.go`, `cash/clientbound/shop_operation_body.go`, `field/clientbound/mts_operation.go`, `merchant/clientbound/shop_scanner_result.go`. Several of the ops those files back are already `verified` at `gms_v92`/`gms_v95` in `status.json` (e.g. `CHARLIST`, `CHAR_INFO`, `INVENTORY_OPERATION`, `ADD_NEW_CHAR_ENTRY` — confirmed via `python3` scan of `docs/packets/audits/status.json` filtered to `character/clientbound`, `inventory/clientbound`, etc., cell `gms_v95: verified`). Commit `a8adafb12`'s own message scopes its fixture correction to `libs/atlas-packet/parcel/clientbound/v92_test.go` only ("Re-derive the v92 item-bearing fixtures in ... v92_test.go"); it does not claim to have checked whether any other already-verified op's `gms_v92`/`gms_v95` fixture encodes an Equip-type asset and would now silently gain/lose these 12 bytes. | Either (a) sweep the 19 `model.Asset`-consuming files' `gms_v92`/`gms_v95` fixtures for an Equip-type asset payload and confirm none assert the pre-fix byte layout (re-run `go test ./...` in each affected package, not just `parcel`), or (b) rewrite the manifest's `out_of_scope` note to honestly describe this as "shared codec gate, audited for blast radius against every model.Asset caller" once that sweep is done — the current note ("incidental") is not accurate for a new gate that changes wire output. |

No other CHANGED-BUT-UNCLAIMED codec or gate hits: the only other touched
`.go` files (`git diff --name-only d9ec287b8...HEAD -- 'libs/atlas-packet' \|
grep '\.go$' \| grep -v _test) are `parcel/clientbound/parcel.go`,
`parcel/clientbound/parcel_body.go`, `parcel/parcel.go`,
`parcel/serverbound/action.go`, `parcel/serverbound/action_parcel_id.go`,
`parcel/serverbound/action_send.go` — all under the claimed `PARCEL`/
`DUEY_ACTION` packet dirs. The matrix delta (`git diff d9ec287b8...HEAD --
docs/packets/audits/status.json`, filtered to `"op"`/`"packet"`/`"kind"`
lines) touches only `PARCEL`, `DUEY_ACTION`, and their four
`parcel/serverbound/ParcelAction*` sub-struct rows — no other op row's state
changed as a side effect of the shared-codec edit, which is consistent with
(but does not by itself prove) finding 1 having no live blast radius; it only
shows the matrix wasn't *regenerated* against the other ops, not that they
were checked.

## Coverage-honesty findings (matrix state vs. actual mapped coverage)

These are not classic CLAIMED-BUT-UNVERIFIED (the manifest's literal `ops ×
versions` cells all read `verified` in the final `status.json`), but the
`verified` state itself does not mean what it should for two of the sixteen
claimed cells, per the batch-8 "worst-of-candidates only sees registered
candidates" defect:

1. **`PARCEL` × `jms_v185` reads `verified` while only 7 of 21 arms are
   mapped.** `docs/packets/dispatchers/parcel.yaml` gives a `jms_v185` mode
   only for `OPEN`, `SUCCESSFULLY_SENT`, `PARCEL_REMOVED`, `PARCEL_ARRIVED`,
   `ALARM_NAMED`, `OPEN_QUICK`, `ALARM_GENERIC` (7 keys out of 21); the other
   14 keys have no `jms_v185:` entry at all. `libs/atlas-packet/parcel/clientbound/v185_test.go`
   correspondingly only exercises those same 7 arms (`grep -n '^func Test'`).
   The manifest's own `partial_coverage.jms_v185` block admits exactly this
   ("Only the 7 keys with a direct body-match are populated for jms_v185; the
   rest are left unset"). Yet the `status.json` op row for `PARCEL`
   (`packet: parcel/clientbound/ParcelAlarmGeneric`) reports
   `"jms_v185": {"state": "verified", "opcode": 352}` — a plain `verified`,
   not `partial`/`🟡`. The op-row grading is keyed off a single registered
   candidate (`ParcelAlarmGeneric`); zero `docs/packets/evidence/*/parcel.clientbound.*.yaml`
   files exist for *any* PARCEL arm across *any* version (`find
   docs/packets/evidence -iname "*parcel*" | grep -v serverbound` returns
   nothing) — the 14 unmapped jms_v185 arms, and in fact all 21 clientbound
   arms structurally, are invisible to the candidate pool rather than graded
   as failures, exactly the class this manifest itself flags as a "known
   ceiling." **Recommendation:** either downgrade the manifest's `versions`
   claim for `PARCEL`×`jms_v185` to reflect partial coverage explicitly (it
   already does, in prose, via `partial_coverage` — but the matrix cell
   should say so too), or open a follow-up to correlate the StringPool ids
   and close the 14-arm gap so `verified` is actually earned.

2. **`DUEY_ACTION`'s `CLOSE` arm (`ParcelActionClose`) shows `incomplete /
   "no audit report"` on 6 of the 8 claimed versions in its own sub-struct
   row, despite audit reports existing on disk for all of them.** The
   `"kind": "sub-struct", "packet": "parcel/serverbound/ParcelActionClose"`
   row is wholly new in this branch's diff (`git diff d9ec287b8...HEAD --
   docs/packets/audits/status.json` shows it as an all-`+` block) and reads
   `verified` only for `gms_v72`/`gms_v79`; `gms_v83`, `gms_v84`, `gms_v87`,
   `gms_v92`, `gms_v95`, `jms_v185` all read `"state": "incomplete", "note":
   "no audit report"`. But `docs/packets/audits/gms_v83/ParcelActionClose.json`
   exists and is structurally identical to the `gms_v72` counterpart that
   *does* read `verified` (`"Verdict": 0` — `VerdictMatch`/✅ per
   `tools/packet-audit/internal/diff/diff.go` — in both). The manifest's
   `duey_action_arms` list claims `CLOSE` undifferentiated across all 8
   versions, and the op-level `DUEY_ACTION` row (keyed to the
   `ParcelActionReceive` candidate) does read `verified` for all 8 — so the
   op-level "verified" is not actually corroborated by the `CLOSE` arm's own
   sub-struct grading on 6 of those 8 versions. This looks like either a
   matrix-generation bug (an existing, apparently-passing audit report not
   being picked up for the sub-struct row) or a genuine gap masked by the
   op-row's single-candidate grading; either way it is evidence the task
   should not rely on, unexamined. **Recommendation:** before PR, re-run
   `packet-audit matrix` and diff the regenerated `ParcelActionClose`
   sub-struct cells against the ones currently committed; if they still read
   `incomplete` despite the on-disk audit reports, escalate as a
   `packet-audit` tool defect rather than accepting `DUEY_ACTION` as fully
   verified.

## CLAIMED-BUT-UNVERIFIED

None at the literal manifest `ops × versions × status.json op-row` level —
all 16 cells (`PARCEL`, `DUEY_ACTION` × `gms_v72, v79, v83, v84, v87, v92,
v95, jms_v185`) read `"state": "verified"` in the final `status.json` (spot
check via `python3` parse of both op rows). `gms_v48`/`gms_v61` correctly read
`n-a` and match the manifest's `not_applicable` declaration. See the
Coverage-honesty section above for the two cases where `verified` does not
mean what a literal reading implies.
