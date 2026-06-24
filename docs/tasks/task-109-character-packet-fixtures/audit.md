# task-109 audit — KeyMapChange serverbound consistency (all 5 versions)

## Scope
The entire KeyMapChange serverbound packet (opcode 0x87 v83 / 0x8B v84 / 0x8F v87 /
0x9F v95 / 0x8A jms) across gms_v83, gms_v84, gms_v87, gms_v95, jms_v185 — the 10
matrix cells (5 `CHANGE_KEYMAP` op rows + 5 `None` sub-struct rows).

## Problem (re-confirmed, not trusted blindly)
KeyMapChange is ONE physical serverbound packet. The registry `CHANGE_KEYMAP` entry
carries a primary fname + 2 alts, all sub-handlers of the same dispatcher
(`SaveFuncKeyMap`, `ChangePetConsumeItemID`, `ChangePetConsumeMPItemID`).

`tools/packet-audit/cmd/run.go` `candidatesFromFName` maps **only** `SaveFuncKeyMap`
→ `{name: "KeyMapChange"}`; `ChangePetConsumeItemID`/`ChangePetConsumeMPItemID` →
`nil` (no report, "covered by KeyMapChange mode!=0"). So there is exactly ONE audit
report per version, named `KeyMapChange.json`, keyed by IDAName base `SaveFuncKeyMap`.

The op row resolves its report via `FNameToWriter[version][primaryFName]`
(`grade.go:findReport`). When the registry **primary** is `SaveFuncKeyMap`, the op
row finds the report and consumes the writer (`build.go usedWriters`) → no orphan
`None` sub-struct row is emitted. When the primary is `ChangePetConsumeItemID`, the
op row's `findReport` misses (that fname produces no report) → op row `incomplete`,
and the unconsumed `KeyMapChange` report becomes the verified `None` sub-struct row.

Pre-change state (re-confirmed from regenerated status.json):
| version | CHANGE_KEYMAP | None | primary |
|---|---|---|---|
| gms_v83 | verified | incomplete | SaveFuncKeyMap (prior task 4b) |
| gms_v84 | verified | incomplete | SaveFuncKeyMap |
| gms_v87 | incomplete | verified | ChangePetConsumeItemID |
| gms_v95 | incomplete | verified | ChangePetConsumeItemID |
| jms_v185 | incomplete | verified | ChangePetConsumeItemID |

(The v83/v84 `None` cells are union-row gap-fills — `build.go:178` `StateIncomplete`
"no audit report" — because v87/v95/jms emit the orphan sub-struct row.)

## Model chosen: option (b) — uniform primary = SaveFuncKeyMap in all 5 versions
WHY: The report is keyed on `SaveFuncKeyMap` and is the ONLY report `run.go`
produces for this dispatcher. Making `SaveFuncKeyMap` the registry primary in all 5
versions:
- resolves the `KeyMapChange` report on the `CHANGE_KEYMAP` op row in every version →
  all 5 op cells verified (tier-1 promote: marker.Found && hasEvidence &&
  evidence.Fresh — all present; verdict advisory per grade.go:198);
- consumes the report writer via the op row in every version → the orphan `None`
  sub-struct row is no longer emitted at all (it disappears, rather than leaving 5
  gap-filled incomplete cells).

This is the smallest, most uniform registry change (3 fname-primary swaps, the two
ChangePet* demoted to alts; v83/v84 already had this shape) and creates ZERO new
incomplete cells. Option (a) — verify both rows per version — is not cleanly
supported: there is no second report (run.go returns nil for the ChangePet* arms),
so a `None` sub-struct row would need a fabricated second report. Option (b) is the
tooling-native shape.

The two ChangePet* arms remain documented as `fname_alts` (the same dispatcher's
mode!=0 sub-handlers, covered by the codec's `mode != 0` branch).

## TRUNCATION exception — confirmed per version (IDA, task-109)
The `KeyMapChange.json` report is `FlatInvalid: true, verdicts [0,0,2,2,2,2]`
(v84: `[0,0,0,2,2,2]`) in every version — verdict 2 = width mismatch, the documented
benign loop-vs-flattened static-diff artifact (the export models one flattened entry;
the codec emits a variable-length loop). v83/v84 were confirmed by the prior task.
v87/v95/jms re-confirmed live this task:

- **v87** `CFuncKeyMappedMan::SaveFuncKeyMap @0x5bd3f4` (GMSv87_4GB.exe, port 13340):
  `COutPacket(0x8F=143)` + `Encode4(0)`=mode + `Encode4(count=*(v9-4))` + per-entry
  loop `Encode4(keyIdx)` + `sub_503876 @0x503876 = EncodeBuffer(Src, 5)`. Per-entry =
  4 + 5 = **9 bytes**.
- **v95** `CFuncKeyMappedMan::SaveFuncKeyMap @0x568a60` (GMS_v95.0_U_DEVM.exe, port
  13339): `COutPacket(159)` + `Encode4(0)`=mode + count + per-entry `Encode4(keyIdx)`
  + `FUNCKEY_MAPPED::Encode @0x4f6d80 = EncodeBuffer(this, 5)`. Per-entry = **9 bytes**.
- **jms** `CFuncKeyMappedMan::SaveFuncKeyMap @0x5e7b48` (MapleStory_dump_SCY.exe, port
  13338 — see caveat): `COutPacket(0x8A=138)` + `Encode4(0)`=mode + `Encode4(count)`
  + per-entry `Encode4(keyIdx)` + `FUNCKEY_MAPPED::Encode @0x510939 =
  EncodeBuffer(this, 5)`. Per-entry = **9 bytes**. The internal key-table scan loop is
  `< 94` (vs 89 GMS) — that bounds the scan over the key table, NOT the wire count
  (wire count is the dynamic changed-entry count `*(v9-4)`); it is exactly the
  documented benign loop-count artifact.

All three match the Atlas codec
`libs/atlas-packet/character/serverbound/key_map_change.go`: `mode int32` +
`count int32` + per-entry `[KeyId int32 + TheType int8 + Action int32]` (9 bytes).
**No per-version wire delta. Codec unchanged. Verification-only.**

### jms IDB caveat (surfaced, not blocking)
The plan prefers the jms `*_U_DEVM` build, but the only reachable jms instance is the
SMC retail dump (`MapleStory_dump_SCY.exe`, port 13338). `SaveFuncKeyMap @0x5e7b48`
decompiles cleanly there (not in an SMC-obfuscated region) and the wire structure is
unambiguous and identical to GMS, so the confirmation holds. The existing jms
evidence record (`category: TRUNCATION`) already documented the same artifact and was
not altered.

## Final state (from status.json)
All 10 KMC cells resolve to: the 5 `CHANGE_KEYMAP` op cells = `verified`; the `None`
sub-struct row no longer exists (consumed). No `n-a` used. Incomplete character cell
count: 40 → 35 (cleared exactly the 5 KMC incompletes; introduced none).
`matrix --check` EXIT 0, 0 conflicts, 0 character problem lines (unchanged from
baseline). `fname-doc --check` OK; `operations --check` OK (1 pre-existing unrelated
jms NoteOperation note). `go test -race`/`vet`/`build` green in libs/atlas-packet.
