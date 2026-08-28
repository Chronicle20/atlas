# Stale-comment sweep — gms_v84 Maple Life VERSION-ABSENT retraction

## What I did

Fixed the six comment locations flagged in `bug-maple-life-v84-registration.md`'s
"Still owed" list (reviewer finding 1). All are prose/doc-comment or test-doc
comment edits only — no struct field, encode/decode logic, version gate, or
test assertion touched.

### `libs/atlas-packet/maplelife/clientbound/error.go`
- Body-shape comment: `gms_v83, gms_v87, gms_v92, gms_v95 (... gms_v84 is
  VERSION-ABSENT ...)` → `gms_v83, gms_v84, gms_v87, gms_v92, gms_v95 (...
  gms_v84 registers this op too ... mis-flagged VERSION-ABSENT by an earlier,
  retracted pass)`.
- Receiver-address line: `gms_v84: VERSION-ABSENT.` →
  `gms_v84 0x7fda6f (derivation.md §2.0-CORRECTION).` — address taken verbatim
  from the registration doc's table (`CUICharacterSaleDlg__OnCreateNewCharacterResult_recv`
  at `0x7fda6f`).

### `libs/atlas-packet/maplelife/clientbound/result.go`
- Same two edits, mirrored: body-shape comment now lists gms_v84 in scope;
  receiver-address line `gms_v84: VERSION-ABSENT (§4.3).` →
  `gms_v84 0x7fd949 (derivation.md §2.0-CORRECTION).` (`CUICharacterSaleDlg__OnCheckDuplicatedIDResult_recv`).

### `libs/atlas-packet/maplelife/serverbound/check_name.go`
- Per-version opcode table: `gms_v84  VERSION-ABSENT — no CUICharacterSaleDlg
  code path exists at all` → `gms_v84  opcode 263 (0x107) — CUICharacterSaleDlg
  exists on this binary and was mis-flagged VERSION-ABSENT by an earlier,
  retracted pass`.

### `libs/atlas-packet/maplelife/clientbound/error_test.go`,
`result_test.go`, `serverbound/serverbound_test.go`
Each file's file-header block said `FOUR in-scope cells: gms_v83, gms_v87,
gms_v92, gms_v95. gms_v84 is VERSION-ABSENT ...`. Replaced with `FIVE
in-scope cells: gms_v83, gms_v84, gms_v87, gms_v92, gms_v95` plus the
retraction pointer to `derivation.md §2.0-CORRECTION`.

I deliberately did **not** add `gms_v84` to the per-file byte-fixture version
lists (`mleFixtureVersions` in error_test.go, `mlrFixtureVersions` in
result_test.go, the equivalent list in serverbound_test.go) — those are code,
not comments, and adding a new subtest is a behavior change the brief
excludes ("no version gate, or test assertion" changes). The three
"FOUR/four in-scope cells" header comments now say the corrected header
count (five) but note explicitly that the byte-fixture lists below still
enumerate the original four cells pending a dedicated gms_v84
evidence/fixture pass (packet-verifier work, out of this task's scope), and
that gms_v84's wire framing is nonetheless already exercised generically by
each package's `*RoundTrip` test, which iterates `pt.Variants` —
`libs/atlas-packet/test/context.go` already includes `{Name: "GMS v84", ...}`
in that shared list, so v84 round-trips through Encode/Decode on every test
run even without version-specific literal-byte fixtures. This is the accurate
state, not an invented rationale — I verified `pt.Variants` contains v84 by
reading `libs/atlas-packet/test/context.go`, and confirmed each
`*RoundTrip` test iterates `pt.Variants` (not the narrower fixture list) via
grep.

Downstream `four`/`FOUR` occurrences that still describe the (unchanged)
4-entry fixture-list literal itself (e.g. "The four in-scope cells this row
claims.", "the same bytes for all four") were left untouched — they remain
accurate descriptions of that specific array, which I did not modify.

## What I did not touch

- No struct fields, `Encode`/`Decode` logic, opcodes-in-code, or version
  gates — none exist to gate v84 out, confirmed by the earlier bug-fix pass
  and re-confirmed by reading all three source files before editing.
- No test assertions, fixture arrays, or subtest lists changed.
- `docs/packets/evidence/`, `docs/packets/audits/status.json`, `STATUS.md` —
  untouched, per the brief (packet-verifier scope).
- `template_jms_185_1.json` / anything jms_v185 — untouched.

## Verification

Module-local only, from `libs/atlas-packet`:

```
$ go build ./...
(no output — success)

$ go test ./...
...
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/maplelife/clientbound	0.031s
ok  	github.com/Chronicle20/atlas/libs/atlas-packet/maplelife/serverbound	0.025s
...
```

Full module test run passed (all packages `ok`, no failures) — see full
output captured during the session; the maplelife lines shown above are the
directly-relevant ones. `tools/verify.sh` was not run, per the brief.

## Files changed

- `libs/atlas-packet/maplelife/clientbound/error.go`
- `libs/atlas-packet/maplelife/clientbound/error_test.go`
- `libs/atlas-packet/maplelife/clientbound/result.go`
- `libs/atlas-packet/maplelife/clientbound/result_test.go`
- `libs/atlas-packet/maplelife/serverbound/check_name.go`
- `libs/atlas-packet/maplelife/serverbound/serverbound_test.go`

Commit: `2dfc1373f` — "docs(maplelife): fix stale VERSION-ABSENT comments for gms_v84"

## Self-review

- Diff is comment-lines only (verified via `git diff` — every hunk touches
  only `//`-prefixed lines).
- `go build ./...` and `go test ./...` both clean, no new failures.
- Checked variable names referenced in new comment text against the actual
  source (`mleFixtureVersions`, `mlrFixtureVersions`) rather than assuming a
  shared name — caught and fixed one mismatch (`result_test.go` initially
  named the wrong variable) before finalizing.
- No opcode values invented — 263/0x107, 359/0x167, 360/0x168, and the two
  receiver addresses (`0x7fda6f`, `0x7fd949`) all come directly from the
  registration doc's derivation table, not guessed.

## Issues or concerns

None. This is a low-risk, mechanical comment-accuracy fix with no behavior
change, confirmed by a clean module-local build/test run.
