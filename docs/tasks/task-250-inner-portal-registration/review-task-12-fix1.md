# Review — Task 12 fix round (commits 3340ece91, 3df557498)

Range reviewed: `25a72cdbc..HEAD` (`3340ece91` — data restoration;
`3df557498` — tool round-trip fix).
Finding being repaired: `docs/tasks/task-250-inner-portal-registration/review-task-12.md` §BLOCKING.
Briefs: `.superpowers/sdd/plan/task-12-fix1-brief.md`,
`.superpowers/sdd/plan/task-12-fix1-cont-brief.md`.
Report: `.superpowers/sdd/plan/reports/task-12-report.md` (appended sections
for this round).

## Scope confirmed

Reviewed the full diff of the two commits: the ten `docs/packets/ida-exports/*.json`
files, `docs/packets/audits/STATUS.md`/`status.json`, and
`tools/packet-audit/internal/idasrc/{export.go,extract.go,export_test.go}`.
Independently: (1) diffed each export file's `functions` map by key against
parent `bb7ec8dbd` with a Python script; (2) ran `matrix --check`,
`fname-doc --check`, `operations --check` fresh from the committed tree
(restoring any working-tree writes those commands made before re-checking);
(3) read `Selector`'s custom `MarshalJSON`/`UnmarshalJSON` and traced every Go
call site that builds a `Selector{}` literal (`infer.go`, `baseline_write.go`)
to confirm the marshal path they go through is unchanged; (4) wrote and ran a
throwaway (not committed) test proving the omitted-discriminator case also
round-trips correctly at runtime; (5) confirmed `go build ./... && go test
./...` in `tools/packet-audit` is green; (6) confirmed no stray
`tools/packet-audit/docs/` directory exists.

## Findings

### BLOCKING — `matrix --check` exits 1 at current HEAD; the committed `STATUS.md`/`status.json` are stale relative to the tool's own code, and this is caused by the two-commit split itself

Directed check 2 asked to run `matrix --check` and confirm it exits 0 with no
`decompile hash drift` lines. It does not:

```
$ go run ./tools/packet-audit matrix --check
note	n-a evidence consumed: CASHSHOP_CASH_ITEM_GACHAPON_RESULT × gms_v79 (docs/packets/feature-na-evidence.yaml)
note	n-a evidence consumed: USE_TELEPORT_ROCK × gms_v48 (docs/packets/feature-na-evidence.yaml)
matrix --check: docs/packets/audits/STATUS.md is stale — regenerate and commit
matrix --check: docs/packets/audits/status.json is stale
exit status 1
```

Root cause, confirmed by regenerating and diffing: `docs/packets/audits/STATUS.md`
and `status.json` each carry a `Tool:` / `toolSha` field
(`tools/packet-audit/internal/matrix/model.go:211`,
`tools/packet-audit/cmd/matrix.go:272`) computed from the **on-disk source of
`tools/packet-audit`'s own `.go` files** (`toolTreeSHA()`,
`tools/packet-audit/cmd/toolsha_test.go`). Commit A (`3340ece91`) committed
`STATUS.md`/`status.json` with `toolSha = a645319789...`, which matches the
tool tree as it stood *before* commit B changed `export.go`/`extract.go`.
Commit B (`3df557498`) then changed those two files but never regenerated or
re-committed `STATUS.md`/`status.json`. At the final HEAD (both commits
applied), the on-disk tool tree hashes to `83326298d5...`, and the committed
`STATUS.md`/`status.json` — still carrying the pre-commit-B hash — are
correctly flagged stale by `matrix --check`, which is why it now exits 1.

Regenerating `matrix` confirms this is the *only* difference — running
`go run ./tools/packet-audit matrix` and diffing against the committed files
shows exactly one changed line in each file (the `Tool:`/`toolSha` value);
every data field, including the `USE_INNER_PORTAL` row, is unchanged. So this
is not a data-correctness problem, and not the same class of bug as the
original finding — but it is a real, reproducible, currently-committed
failure of the acceptance criterion both briefs state explicitly:

- `task-12-fix1-brief.md` Step 3: "`matrix --check` output must return to the
  parent's ~5-line shape... with **no** `decompile hash drift` lines."
  (Not applicable to this specific stale-metadata failure mode, but the same
  spirit — `matrix --check` must exit clean — is violated.)
- `task-12-fix1-cont-brief.md` Step 2: "Whatever you choose, `matrix --check`
  must still exit 0 afterwards. Re-run it and quote the actual exit code and
  line count."

The report (`.superpowers/sdd/plan/reports/task-12-report.md:347`) quotes
`matrix --check` exit 0 / 2 lines — that was true when it was run (before the
two commits were split apart, or before commit B was staged), but is not true
of the state actually landed in the repository. Commit A's own commit
message asserts "matrix --check now exits 0 (2 informational lines)" — also
no longer true once commit B is applied on top. Neither commit re-ran the
regenerate-and-commit step after the tool tree changed, so the two-commit
split itself introduced this regression.

Fix is mechanical (re-run `go run ./tools/packet-audit matrix` and commit the
result once both code changes are in place) but it is not yet done, and I did
not do it — reporting the state as found, restoring working-tree changes
after verifying.

### Everything else checked — no other blocking findings

1. **Directed check 1 — key-level diff, PASS for all ten files.** A Python
   script diffed each export's `functions` map by key against parent
   `bb7ec8dbd`. For every one of the ten files
   (gms_v48/v61/v72/v79/v83/v84/v87/v92/v95, gms_jms_185): `added =
   ['CUserLocal::TryRegisterTeleport']`, `removed = []`, `changed = 0`
   entries, no top-level (`binary`/`md5`/`generated_at`) difference. This is
   the exact key-level identity the fix-1 brief required and the original
   blocking finding is closed on the data side.

2. **`Selector`'s custom marshaller (directed check 3), PASS.**
   `tools/packet-audit/internal/idasrc/extract.go:31-83`. Confirmed:
   - `selectorJSON.Case` has no `omitempty` (matches pre-fix `Case int64
     json:"case"`; `Default`/`Guard` both keep their pre-fix `omitempty`),
     so no field beyond `Discriminator`'s presence-tracking behavior changed.
   - Round trip: an entry with the key **absent** stays absent
     (`discriminatorExplicit=false` on unmarshal when `sj.Discriminator ==
     nil`, `Marshal` only emits when `Discriminator != "" ||
     discriminatorExplicit`); an entry with an **explicit `""`** stays
     explicit (`discriminatorExplicit=true` set whenever the source key is
     present, even empty). I confirmed the omission-preserved case at
     runtime with a throwaway (not committed) test — see "Not evaluable /
     coverage gap" below for why this matters as a finding.
   - Every Go call site building a `Selector{}` literal —
     `tools/packet-audit/internal/idasrc/infer.go:27,83,318` (Discriminator
     set from a variable), `infer.go:300` (`Selector{Default: true}`),
     `infer.go:326` (`Selector{Guard: clause}`) — leaves
     `discriminatorExplicit` at its Go zero value (`false`), which reproduces
     the pre-fix `omitempty` behavior for programmatically-built values:
     the key is omitted whenever `Discriminator == ""`.
   - `WriteDispatch` (`tools/packet-audit/internal/idasrc/baseline_write.go:229`)
     calls `json.MarshalIndent(up.Dispatch, ...)` where `up.Dispatch` is
     `[]Selector` populated from those same Go-literal call sites, so it goes
     through the same `MarshalJSON` and produces the same omission behavior
     as before the fix.
   - Every field of `Selector` (`Discriminator`, `Case`, `Default`, `Guard`)
     is set on both the marshal and unmarshal side — no field silently
     dropped.

3. **`exportFn` field additions, PASS on collision-freedom.**
   `note`/`notes`/`region`/`_note` map to four distinct Go fields
   (`Note`/`Notes`/`Region`/`NoteUnderscore`) with four distinct JSON tags
   (`tools/packet-audit/internal/idasrc/export.go:39-49`); an entry carrying
   both `note` and `notes` keeps both fields distinct on round trip (`Notes`
   was pre-existing; `Note`/`Region`/`NoteUnderscore` are new — no tag
   overlaps any existing field).

4. **New round-trip test, PASS with one coverage gap (directed check 4).**
   `TestSpliceExportPreservesCuratedProvenanceKeys`
   (`tools/packet-audit/internal/idasrc/export_test.go:448-528`) asserts, on
   an untouched entry, that `note`/`notes`/`region`/`_note` survive by typed
   field comparison AND that an explicit `"discriminator": ""` survives by a
   raw-byte `strings.Contains` check on the marshaled output — this is
   presence checking, not merely value equality, as directed check 4 asked.
   The report's RED evidence (`task-12-report.md:271-287`) is genuine: a
   quoted `go build` failure (`kept.Note undefined`, etc.) against the
   pre-fix struct, restored and re-run GREEN.
   **Gap:** the test only exercises the case where `discriminator` is present
   (`"": ""`). It does not include a case where the source entry's
   `dispatch` selector **omits** the `discriminator` key and assert the key
   stays omitted after the splice round trip — the other of the "both
   discriminator states" directed check 4 calls for. I confirmed at runtime
   (via a throwaway, uncommitted test, deleted after use) that the omission
   case is in fact handled correctly by the shipped code — this is a test
   coverage gap, not a functional defect, but it means the ~65 existing
   selectors that legitimately omit `discriminator` (the exact case Step 2
   of the fix1-cont brief asked the implementer to validate) are not pinned
   by an automated regression test, only by a one-time manual `grep` check
   quoted in the report.

5. **Scope hygiene (directed check 6), PASS.** `git diff --stat
   25a72cdbc..HEAD` touches exactly: the ten export JSONs,
   `docs/packets/audits/STATUS.md`/`status.json`,
   `tools/packet-audit/internal/idasrc/export.go`,
   `tools/packet-audit/internal/idasrc/extract.go`, and the new
   `export_test.go`. Nothing under `services/`, no codec, no evidence YAML,
   no registry, no seed template. No `tools/packet-audit/docs/` directory
   exists in the tree or in either commit's diff. `git add -A`/`.` was not
   used — the pending unrelated `docs/tasks/task-250-inner-portal-registration/*`
   review artifacts remain untracked, and commit boundaries match the
   brief's Step 4/Step 6 split (data vs. tool fix).

6. **All ten `USE_INNER_PORTAL` cells still ✅ (directed check 7), PASS.**
   `docs/packets/audits/STATUS.md:615` — all ten columns read `✅`. Confirmed
   this is unaffected by the stale-toolSha issue (only the `Tool:` header
   line differs on regeneration; the data row is identical).

7. **`go build ./... && go test ./...` in `tools/packet-audit`, PASS.** All
   packages build; all tests pass (`internal/idasrc` included, exercising
   the new test).

8. **`fname-doc --check` / `operations --check`, PASS.** Both exit 0
   (`fname-doc check OK (269 structs...)`, `operations check OK (0
   absent-writer note(s))`), matching the report.

## Not evaluable

- Whether the omitted-discriminator round trip is exercised by any test
  *elsewhere* in the repository (outside `tools/packet-audit`) was not
  swept — out of this unit's touched-file scope.

## Verdict rationale

The original blocking finding — silent data loss of curated IDA provenance
across the ten export files — is genuinely closed: the key-level diff is
clean for all ten files, and the `Selector` custom marshaller correctly
reproduces both discriminator presence states on manual verification, with
no collision between the new `exportFn` fields. But the fix round introduces
a new, currently-reproducible failure of its own stated acceptance
criterion: `matrix --check` exits 1 at HEAD because the two-commit split left
`STATUS.md`/`status.json` stale against the tool-tree hash after commit B
changed the tool's source. This is a small, mechanical, one-command fix (a
final `matrix` regenerate + commit) — but it is not done, both briefs
required `matrix --check` to exit 0, and the report's exit-0 quote does not
reflect the state actually on the branch. That is blocking. The test-coverage
gap on the omitted-discriminator case is non-blocking (behavior verified
correct, not defective) but should be closed before this data class is
touched again.
