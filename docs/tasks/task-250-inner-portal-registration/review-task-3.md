# Review: Task 3 — `InnerPortal` serverbound codec

**Range:** `b1ddb4db8..122fe88c1` (commit `122fe88c1`)
**Scope:** `libs/atlas-packet/portal/serverbound/inner_portal.go`,
`libs/atlas-packet/portal/serverbound/inner_portal_test.go`

## Brief vs amendment

The brief body describes six versions and "no gate required." Per the task
instructions, the `## SCOPE AMENDMENT` supersedes this: ten versions in
scope, `fieldKey` version-gated at `MajorAtLeast(61)` (GMS) / unconditional
for JMS. The implementer correctly followed the amendment, not the
superseded body — confirmed by reading both and diffing against the
delivered code.

## Findings

### PASS — Gate predicate uses `MajorAtLeast` idiom, not raw comparison

`inner_portal.go:79-81`:
```go
func encodesFieldKey(t tenant.Model) bool {
	return (t.IsRegion("GMS") && t.MajorAtLeast(61)) || t.Region() == "JMS"
}
```
This is character-for-character the same shape as the cited precedent at
`field/serverbound/general.go:47` (`(t.IsRegion("GMS") && t.MajorAtLeast(87)) || t.Region() == "JMS"`)
and structurally matches `cash/serverbound/check_name_change_possible.go:103-125`'s
`credentialIsString` (unexported gate fn, called from both `Encode` and
`Decode`, doc-comment citing the structure doc). No raw `>`/`>=` literal
comparison anywhere in the file (`grep -n '[<>]=\? *[0-9]' inner_portal.go`
finds nothing outside the `MajorAtLeast(61)` call).

Verified true/false behavior via the golden-byte test run (`go test
./portal/serverbound/... -run InnerPortal -v`): `gms_v61` through
`gms_v95` and `jms_v185` all produce the 13-byte body (fieldKey present);
`gms_v48` produces the 12-byte body (fieldKey absent). All ten subtests
pass. GMS≥61 → true, JMS v185 → true (via the unconditional region arm),
GMS v48 → false — matches the reviewer's three probe points exactly.

### PASS — gms_v48 fixture is 12 bytes / five fields; the other nine are 13 bytes / six fields

`inner_portal_test.go:15-17,29`:
```go
sixField := []byte{0x01, 0x02, 0x00, 0x73, 0x70, 0x64, 0x00, 0xC8, 0x00, 0x2C, 0x01, 0xCE, 0xFF}
fiveField := []byte{0x02, 0x00, 0x73, 0x70, 0x64, 0x00, 0xC8, 0x00, 0x2C, 0x01, 0xCE, 0xFF}
...
{"gms_v48", "GMS", 48, 1, fiveField, 0},
{"gms_v61", "GMS", 61, 1, sixField, 1}, // ... through jms_v185
```
Matches the amendment's tables exactly, including the `expectFieldKey: 0`
decode-side assertion for the v48 row (zero value, field not on wire) vs `1`
for the other nine. Confirmed the gate is load-bearing, not accidentally
satisfied either way: the implementer's report documents temporarily
removing the guard, which broke the build (`declared and not used: t`) — an
independent proof beyond "tests are green" that the v48 row would fail
without the gate.

### PASS — Encode/Decode are exact mirrors; struct is immutable with value-receiver accessors

`inner_portal.go:84-104`: `Encode` writes `fieldKey?, portalName, x, y,
targetX, targetY`; `Decode` reads the identical sequence in the identical
order, both gated by the same `encodesFieldKey(t)` call. All six accessors
(`FieldKey`, `PortalName`, `X`, `Y`, `TargetX`, `TargetY`) are value
receivers (`func (m InnerPortal) ...`); only `Decode` uses a pointer
receiver, matching `script.go`'s convention. No setters; all fields
unexported.

### PASS — Field order matches the structure docs

Cross-checked against `structures/gms_v95.md` (`fieldKey, portalName, x, y,
targetX, targetY`), `structures/gms_v61.md` (same order, gate boundary
confirmed present), and `structures/gms_v48.md` (`portalName, x, y,
targetX, targetY` — no `fieldKey`, boundary confirmed absent). The codec's
field declaration order and read/write order match all three exactly.

### PASS — Repo conventions / sibling comparison

Structurally a close copy of `script.go`'s shape (same package, same
`Operation()`/`String()`/`Encode`/`Decode` signatures, same
`response.NewWriter`/`request.Reader` usage), extended with the two target
fields and the version gate. `gofmt -l` and `go vet ./portal/...` both clean
(re-run in this review, no output). `packet-audit:fname` marker present
(`inner_portal.go:21`); no `packet-audit:verify` marker — correct per this
task's scope (Task 12 owns markers).

### PASS — Test honesty

`TestInnerPortalRoundTrip` loops the full `pt.Variants` slice (12 entries,
including `GMS v28` below the gate and `GMS v86` above it, neither of which
has a golden row) and asserts field-by-field round-trip equality for the
five ungated fields — this would fail if the gate mis-encoded/mis-decoded
`fieldKey` inconsistently within a single Encode/Decode pair, since a
mismatched byte count would desync the trailing fields (and `pt.RoundTrip`
independently asserts no trailing bytes per FR-2.4). Re-ran both test
functions live in this review; all 10 golden-byte subtests and all 12
round-trip subtests pass (`go test ./portal/serverbound/... -run InnerPortal
-v`).

## Not evaluable

- Whether `0x6a5462` / `0x7aa1e3` genuinely decompile the way the structure
  docs claim — that's an IDA-grounded claim from a prior task (Task 1/2's
  structure-doc derivation), out of this codec-only unit's review surface.
  Taken as given per the task instructions.
- Downstream consumption (service wiring, seed-template routes, registry
  rows, `packet-audit:verify` markers) — explicitly out of scope per the
  reviewer brief; later plan tasks own these.

## Verdict

No blocking findings. The implementation matches the scope-amendment
(not the superseded brief body) exactly: gate idiom, byte fixtures, field
order, accessor shape, and test honesty all check out against direct
evidence (structure docs, sibling codec, live test run).
