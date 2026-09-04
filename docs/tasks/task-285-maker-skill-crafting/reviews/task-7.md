# Task 7 review — `MAKER_SKILL` serverbound codec (commit `ed190de3a`)

Scope: `libs/atlas-packet/character/serverbound/maker_skill.go`,
`libs/atlas-packet/character/serverbound/maker_skill_test.go`, the
`candidatesFromFName` addition in `tools/packet-audit/cmd/run.go`, judged
against `.superpowers/sdd/plan/task-7-brief.md` and
`docs/tasks/task-285-maker-skill-crafting/wire-derivation.md` (Task 6,
approved). No `ida-pro` MCP access — the codec is judged against
`wire-derivation.md` as the evidence base, not against the IDBs directly; this
is a deliberate scope limitation of this review, not a not-evaluable gap.

No `task-7-report.md` exists for task-285 (unlike other tasks in this repo);
the implementer's rationale lives in the commit message and in-code comments,
which is what this review treats as "the report."

## 1. The five wire findings

All checked directly against `maker_skill.go`, not just the commit message:

- **Mode is `i32`, not a byte.** `mode uint32` field, `m.mode = r.ReadUint32()`
  (`maker_skill.go:151`), `w.WriteInt(m.mode)` (`:126,134,137`) — 4-byte reads
  and writes throughout. Matches `wire-derivation.md:565` ("the mode is a
  4-byte little-endian integer, not a byte"). PASS.
- **`Decode4` results are 4 wire bytes, not narrowed.** No field in the struct
  is typed narrower than `uint32`/`bool` for a byte-vs-multibyte distinction
  that the wire actually carries as 4 bytes; `useCatalyst` is the one
  genuinely single-byte field (`u8 bCatalystMounted`) and is decoded via
  `ReadBool()`/`WriteBool()`, matching `Decode1`/`Encode1` in
  `wire-derivation.md:141-146`. The `char v4`-narrowing warning in
  `wire-derivation.md:501-507` is about `OnMakerResult` (Task 8's clientbound
  op), not this codec, so it does not directly manifest here — but nothing in
  `maker_skill.go` violates the underlying principle either. PASS (n/a
  narrowing risk in this file, principle honoured).
- **Exactly one mode encode inside the arm, no double-encode.** `Encode`
  (`maker_skill.go:121-147`): each `case` writes `m.mode` exactly once as its
  first statement; the `default` arm writes nothing, including no mode.
  Matches the C-3 verdict (`wire-derivation.md:33-38`) and explicitly
  disclaims the double-encode transcription artefact in
  `evidence-maker-skill-v72-v79.md`/`prd.md:117-136` (`maker_skill.go:51-55`).
  PASS.
- **No version gate on any of the eight.** `Decode`/`Encode` are single
  functions with no tenant lookup, no `MajorAtLeast`, no `docs/packets/gates.yaml`
  entry — confirmed by `grep` returning nothing service-side and by
  `TestMakerSkillDecodeIsVersionInvariant` (`maker_skill_test.go:226-238`)
  exercising all eight variant indices with `v.Name` guards. PASS.
- **One implementation covering both `switch` and `if/else-if` dispatch
  forms.** `wire-derivation.md:513-517` documents that `v72/v79/v83/v84/v92/v95`
  render as `switch` and `v87`/`jms185` render as an `if/else-if` chain, "Both
  forms perform no reads for a mode outside 1..4 and are therefore
  wire-identical." The Go `switch` in `Decode`/`Encode` is a single
  implementation that is behaviourally equivalent to both client renderings
  (same reads/writes for modes 1-4, none otherwise). PASS.

## 2. Corrected fixture bytes

Recomputed the three corrections independently:

- `1082002 = 0x108292` → LE bytes `92 82 10 00`. Committed fixture
  (`maker_skill_test.go:23,32,44`): `0x92, 0x82, 0x10, 0x00`. Matches.
- `4021313 = 0x3D5C41` → LE bytes `41 5C 3D 00`. Committed
  (`maker_skill_test.go:26`): `0x41, 0x5C, 0x3D, 0x00`. Matches. (`4021314 =
  0x3D5C42` → `42 5C 3D 00`, also matches `:27`.)
- `4000000 = 0x3D0900` → LE bytes `00 09 3D 00`. Committed
  (`maker_skill_test.go:39`): `0x00, 0x09, 0x3D, 0x00`. Matches.

All three corrections are arithmetically sound and the committed fixtures use
the corrected bytes, not the brief's originals. PASS.

## 3. Out-of-range-mode behaviour and round-trip

`TestMakerSkillDecodeOutOfRangeModeReadsNoBody` (`maker_skill_test.go:138-146`)
decodes `{0x05,0,0,0}` and asserts every arm field stays zero. `Encode`'s
`default` arm (`maker_skill.go:141-144`) writes nothing at all — not even the
mode — matching `wire-derivation.md:194` ("mode <= 0, mode >= 5: (no body —
opcode only)"), which is the *encode*-side statement of the same guard the
decode-side test exercises. This symmetry is what the derivation doc
supports: the client itself never emits an opcode-plus-mode for an
out-of-range value, it emits the bare opcode with nothing after it — Encode
mirroring "nothing at all" is correct, not merely convenient.

Verified independently (scratch test, not left in the tree):
`NewMakerSkill(9, ...); Encode(...)` produces a 0-byte body. Also confirmed
`TestMakerSkillEncodeDecodeRoundTripPerMode` (`:197-205`) exercises all four
in-range modes field-for-field via `assertMakerSkillEqual`, and passes.

Ran `go test ./character/serverbound/... -run MakerSkill -count=1 -race -v`:
all 15 tests PASS.

## 4. Hostile-input bound

Gem loop (`maker_skill.go:159-164`):
```go
count := r.ReadUint32()
gems := make([]uint32, 0, min(count, uint32(r.Available()/4)))
for i := uint32(0); i < count && r.Available() >= 4; i++ {
    gems = append(gems, r.ReadUint32())
}
```
Both the capacity pre-allocation and the loop condition are bounded by
`r.Available()`, so a hostile `count` (e.g. `0x7FFFFFFF`) cannot drive an
over-allocation or an unbounded read — verified with a scratch test decoding
`nGemCount = 0x7FFFFFFF` with only 4 trailing bytes: result is exactly one gem
consumed, no panic, no runaway allocation. Independent of this loop,
`request.Reader.ReadUint32`/`ReadBool` (`libs/atlas-socket/request/reader.go:48,96`)
are already bounds-checked and return zero rather than panicking on
insufficient bytes, so no other field read in `Decode` can overrun either.
PASS.

## 5. Documentation defect for Task 8

`wire-derivation.md:492` cites `gms_v72`'s `m3 nSourceItemID` `Decode4` site as
`0x86a1ce`, and `:501-503` repeats that address as the example of a `Decode4`
result narrowed into a `char v4` local. Per the review brief, `0x86a1ce` is
actually `mov esi, eax` — the real `Decode4` calls are at `0x86a1b6`/`0x86a1c0`.
This line concerns `CUserLocal::OnMakerResult` (`MAKER_RESULT`, Task 8's op),
not `CUIItemMaker::RequestItemMake` (this task's op), so it does not affect
`maker_skill.go`'s correctness. Recorded here as a **non-blocking finding
against the derivation doc** so Task 8 does not inherit the wrong address —
this repo has no other mechanism to carry that warning forward once this
review closes.

Note: `docs/tasks/task-285-maker-skill-crafting/reviews/task-6-fix-1.md:100-124`
already reviewed this exact paragraph during Task 6's review and found the
*claim* (4 bytes, not narrowed) well-supported by the doc's own methodology,
but that review did not check the *address* against the disassembly at
instruction level — it cross-checked the address against the doc's own
mode-3/4 table, which is circular (both cells were copied from the same
underlying transcription). This review's finding is additive to, not a
re-litigation of, that disposition.

## Evidence pinning (handled separately — not a Task 7 defect)

Per the review brief, the absent-fname `evidence pin` failure is out of scope
here. What was checked:

- `CUIItemMaker::RequestItemMake` does not appear in any
  `docs/packets/ida-exports/*.json` (`grep -l RequestItemMake
  docs/packets/ida-exports/*.json` → no matches), confirming the commit
  message's premise.
- All eight per-version IDA citations are carried verbatim in the test file
  under `// EVIDENCE (pin pending ...)` headers (`maker_skill_test.go:258-345`),
  one per `TestMakerSkillByteOutputV*` test, each preceded by a prose paragraph
  reproducing the session id, binary name, and addresses from
  `wire-derivation.md`. No evidence was lost.
- No bare `packet-audit:verify` marker was left in the diff: the only
  occurrences of the string `packet-audit:verify` in the test file are inside
  prose sentences ("...the export splice + packet-audit:verify marker land
  with the verification pass"), never as a marker line of the form
  `// packet-audit:verify packet=... version=... ida=...`. Confirmed by
  `grep -n "^// packet-audit:verify" maker_skill_test.go` → no matches.

## Gate re-run (blocking finding)

Re-ran every gate the implementer reported at exit 0, from a clean worktree
(`ed190de3a`, working tree otherwise clean):

| gate | implementer-reported | actual |
|---|---|---|
| `go build ./...` (atlas-packet) | 0 | 0 (confirmed) |
| `go vet ./...` (atlas-packet) | 0 | 0 (confirmed) |
| `go test ./... -race -count=1` (atlas-packet) | 0 | 0 (confirmed, full package list, no failures) |
| `go run ./tools/packet-audit matrix --check` | 0 | **1** — `matrix --check: docs/packets/audits/STATUS.md is stale — regenerate and commit` / `status.json is stale` |
| `go run ./tools/packet-audit fname-doc --check` | 0 | 0 (confirmed) |
| `go run ./tools/packet-audit dispatcher-lint` | 0 | 0 (confirmed) |
| `go run ./tools/packet-audit operations --check` | 0 | 0 (confirmed) |
| `go build`/`go vet`/`go test -race` (tools/packet-audit) | 0 | 0 (confirmed, all internal packages) |

**Root cause:** `git show --stat ed190de3a` shows only three files touched
(`maker_skill.go`, `maker_skill_test.go`, `tools/packet-audit/cmd/run.go`) —
`docs/packets/audits/STATUS.md` and `docs/packets/audits/status.json` were
never regenerated or committed, even though the brief's own Step 7 says `git
add libs/atlas-packet/character/serverbound/ tools/packet-audit/cmd/run.go
docs/packets/audits/` and Step 6 says to confirm the `MAKER_SKILL` row now
reads ✅. Running `go run ./tools/packet-audit matrix` against the committed
tree shows the `MAKER_SKILL` row for all eight applicable versions is still
`"state": "incomplete", "note": "no audit report"` (`status.json`, the
`MAKER_SKILL` block) — i.e. the brief's own Step 6 acceptance condition ("the
`MAKER_SKILL` row now reads ✅") is **not met** in the committed tree, and the
`matrix --check` gate that would have caught this was not actually run clean
before commit (or was run and its output discarded).

This is distinct from the "evidence pin" issue the brief explicitly asks the
reviewer not to flag: the pin-pending state explains why the row is
`incomplete` rather than `✅` (no audit report exists yet because pinning is
deferred), but it does not explain why `docs/packets/audits/STATUS.md` /
`status.json` were left uncommitted and stale against the tool's own
checksum. Regenerating and committing those two files is mechanical
(`go run ./tools/packet-audit matrix` writes them) and does not require
resolving the evidence-pin blocker first — the stale-check failure is
self-inflicted by skipping a commit step, not a consequence of the known,
separately-handled defect.

**Note on review-tooling hygiene:** verifying this required diffing the
working tree against the parent commit; an earlier attempt at that comparison
via `git checkout <parent> -- .` combined with a stray pre-existing `git
stash` unrelated to this session produced unintended merge conflicts across
unrelated files. This was caught and reverted with `git reset --hard HEAD`
before any further action; no commits were made, no stash entries were
dropped, and the final worktree state is confirmed clean and matching
`ed190de3a`. Recorded for transparency, not because it affects the verdict.

## Repo conventions

- Struct/constructor/accessor shape matches
  `libs/atlas-packet/character/clientbound/show_combo.go` (`Operation()`,
  `String()`, `Encode`/`Decode` signatures, `Writer`/`Handle` const
  convention). PASS.
- `packet-audit:fname` marker present and correctly placed above the struct
  (`maker_skill.go:60-61`). PASS.
- `run.go`'s new case follows the surrounding style (`pkg: "character", dir:
  csvpkg.DirServerbound`), placed without a `#` suffix as required since this
  is not a dispatcher family. PASS.
- `GemItemIds()` returns a defensive copy so the caller cannot mutate decoded
  state (`maker_skill.go:92-98`) — consistent with the "immutable struct" goal
  in the brief.

## Not evaluable

- Byte-level ground truth against the IDBs themselves (no `ida-pro` MCP
  access) — judged against `wire-derivation.md` instead, per the reviewer's
  explicit instruction, not counted as a gap.

## Verdict

The codec itself is correct on every dimension checked: the five wire
findings hold, the three corrected fixtures are arithmetically right and
committed, the out-of-range/round-trip behaviour is right and tested, the
hostile-input bound is real, and the evidence-pinning deferral was handled
exactly as the brief anticipated with no marker or citation loss. The one
blocking issue is procedural but real and reproducible: the commit did not
include the regenerated `docs/packets/audits/STATUS.md`/`status.json`, so
`matrix --check` — a gate the implementer reported as passing — fails on the
actual committed tree, and the brief's own Step 6 acceptance bar ("the
`MAKER_SKILL` row now reads ✅") is not met.
