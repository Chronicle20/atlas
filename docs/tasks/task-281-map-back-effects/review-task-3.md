# Review: Task 3 — SetBackEffect / ClearBackEffect clientbound codecs

Range reviewed: `e45a6ed34..a9cc197bb`
Diff: `.superpowers/sdd/plan/review-e45a6ed34..a9cc197bb.diff`
Module: `libs/atlas-packet`

## Scope

`git diff --stat e45a6ed34..a9cc197bb`:

```
 .../field/clientbound/clear_back_effect.go         | 40 ++++++++++++
 .../field/clientbound/clear_back_effect_test.go    | 35 +++++++++++
 .../field/clientbound/set_back_effect.go           | 71 ++++++++++++++++++++++
 .../field/clientbound/set_back_effect_test.go      | 37 +++++++++++
 4 files changed, 183 insertions(+)
```

Four new files, all inside `libs/atlas-packet/field/clientbound/`. No `docs/packets/registry/*.yaml`,
seed template, `docs/packets/evidence/`, `docs/packets/audits/`, or `feature-*.yaml` touched. Scope
confirmed — matches the brief exactly, nothing else in-tree changed.

## 1. Wire layout / Encode-Decode inverse (`set_back_effect.go`)

`set_back_effect.go:52-68`:

```go
func (m SetBackEffect) Encode(...) ... {
    w.WriteByte(m.effect)
    w.WriteInt(m.fieldId)
    w.WriteByte(m.pageId)
    w.WriteInt(m.duration)
    ...
}
func (m *SetBackEffect) Decode(...) ... {
    m.effect = r.ReadByte()
    m.fieldId = r.ReadUint32()
    m.pageId = r.ReadByte()
    m.duration = r.ReadUint32()
    ...
}
```

Order and widths match the constraint exactly: `nEffect` byte, `nFieldID` int32, `nPageID` byte,
`tDuration` int32 — 10 bytes total. Encode/Decode are structurally exact inverses (same field
order, same widths). PASS.

Golden test (`set_back_effect_test.go:19-26`) asserts `[]byte{0x00, 0x00, 0xE1, 0xF5, 0x05, 0x01,
0xE8, 0x03, 0x00, 0x00}` for `NewSetBackEffect(BackEffectShow, 100000000, 1, 1000)`. Verified
independently: `100000000` LE = `00 E1 F5 05`, `1000` LE = `E8 03 00 00` — matches. `go test -run
TestSetBackEffectGolden -v` passes. PASS.

## 2. `clear_back_effect.go` empty body

`clear_back_effect.go:20-38` — `Encode` returns `w.Bytes()` with no writes, `Decode` reads
nothing. `TestClearBackEffectGolden` asserts `len(actual) == 0`. PASS.

## 3. Immutability and structural parity with siblings

Both structs carry unexported fields only, value receivers for `Operation`/`String`/`Encode`
(and accessors on `SetBackEffect`), pointer receiver only for `Decode` — identical split to
`field_obstacle_on_off.go` and `field_obstacle_all_reset.go`. No exported mutable fields, no
setters, construction only via `NewSetBackEffect`/`NewClearBackEffect`. `clear_back_effect.go`
is a near-verbatim rename of `field_obstacle_all_reset.go` (same shape, same accessor-free body).
No gratuitous divergence in structure, naming, or test idiom found. PASS.

## 4. Marker audit (direct count)

```
set_back_effect_test.go:   9 × `packet-audit:verify`, packet=field/clientbound/FieldSetBackEffect
clear_back_effect_test.go: 9 × `packet-audit:verify`, packet=field/clientbound/FieldClearBackEffect
```

Versions on both files: `gms_v61 gms_v72 gms_v79 gms_v83 gms_v84 gms_v87 gms_v92 gms_v95
jms_v185` — exactly the ADDENDUM's final list. `gms_v48` does not appear on either file, and
`docs/tasks/task-281-map-back-effects/structures/gms_v48.md:1,113` confirms both cells are
VERSION-ABSENT there. `packet=` value on every marker line matches
`field/clientbound/FieldSetBackEffect` / `field/clientbound/FieldClearBackEffect` exactly, per
grep of both test files. PASS.

## 5. Address provenance (grepped against `structures/<version>.md`)

Every `ida=` address in both test files was grepped against the corresponding `structures/*.md`
file. All 18 addresses resolve verbatim to a line in the matching version's structures record
(SET: v61 `0x5a8316` @ `gms_v61.md:12,55,91`; v72 `0x5f5b4f` @ `gms_v72.md:28`; v79 `0x614572` @
`gms_v79.md:25,60`; v83 `0x6445c5` @ `gms_v83.md:23,56`; v84 `0x659e3c` @ `gms_v84.md:19`; v87
`0x67dcdb` @ `gms_v87.md:23,57`; v92 `0x606d80` @ `gms_v92.md:29,67`; v95 `0x612850` @
`gms_v95.md:19,30,52`; jms_v185 `0x6ba27f` @ `jms_v185.md:24,56` — CLEAR: v61 `0x5a871b` @
`gms_v61.md:12,56,111,115`; v72 `0x5f5f54` @ `gms_v72.md:29,57`; v79 `0x614977` @
`gms_v79.md:26,78`; v83 `0x6449ca` @ `gms_v83.md:24,75,79`; v84 `0x65a241` @ `gms_v84.md:21,57`;
v87 `0x67e0e0` @ `gms_v87.md:24,70,74`; v92 `0x612ef0` @ `gms_v92.md:30,91`; v95 `0x61f230` @
`gms_v95.md:21,65,69`; jms_v185 `0x6ba684` @ `jms_v185.md:25,72,76`).

None of the addresses were checked against `docs/packets/registry/*.yaml` — that file was not
consulted by the implementer per the report, and it is out of this task's scope anyway. PASS.

## 6. Test quality

Golden tests assert real byte-level fixtures (`bytes.Equal` against a hand-derived slice;
`len(actual) == 0` for the empty body) — not tautologies. Round-trip tests exercise
`test.Variants` (12 variants including `GMS_v28`, `GMS_v86`, `GMS_v48` beyond the 9 audited
versions) via `test.RoundTrip`, which encodes then decodes and compares. Ran independently:

```
go test ./field/clientbound/... -run BackEffect -v
```

All golden and round-trip subtests PASS (21 subtests total across both files, including the 12
`test.Variants` entries for each of Set/Clear). PASS.

## 7. Scope

Confirmed above — only the four new files under `libs/atlas-packet/field/clientbound/`. PASS.

## 8. Build / test — independently confirmed

From `libs/atlas-packet`:

```
go build ./...          → clean, no output
go test ./...            → all packages ok / no test files, no FAIL
go test ./field/clientbound/... -run BackEffect -v → PASS (21 subtests)
gofmt -l field/clientbound/set_back_effect.go field/clientbound/set_back_effect_test.go \
       field/clientbound/clear_back_effect.go field/clientbound/clear_back_effect_test.go
                          → no output (clean)
```

Matches the implementer's report. PASS.

## Constraint checks

- No `MajorAtLeast` gate anywhere in either file — confirmed by reading both files in full.
  PASS.
- `duration` documented only as "fade" length, never "lifetime"/"expiry" — confirmed, no such
  wording anywhere in `set_back_effect.go`. PASS.
- Writer constants exact: `SetBackEffectWriter = "SetBackEffect"` (`set_back_effect.go:13`),
  `ClearBackEffectWriter = "ClearBackEffect"` (`clear_back_effect.go:13`). PASS.
- Packet ids used in markers exactly `field/clientbound/FieldSetBackEffect` /
  `field/clientbound/FieldClearBackEffect` — these are not Go identifiers (consistent with the
  sibling `FieldObstacleOnOff` pattern, which is likewise never referenced as a literal string
  inside `libs/atlas-packet` Go source, only in docs/tooling). No defect. PASS.

## Non-blocking notes

- `set_back_effect.go:14-21` — the `BackEffectShow`/`BackEffectHide` doc comment ends "...which is
  why atlas-maps rejects it at the command consumer," a sentence that only becomes true once
  Task 6 lands. Per the controller's prior ruling this is accepted as verbatim brief text and is
  not blocking; noted for completeness only.

## Not evaluable

None. Everything in the review checklist was directly verifiable within the diff plus the
`structures/*.md` records it cites.

## Verdict

APPROVED. All ten checklist items pass with direct evidence; no blocking findings.
