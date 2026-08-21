# Review — fix round for backend-guidelines audit (FILE-01 / DOM-20)

Commit under review: `1f90e3087fc731e97d72f2d0f313a53c3c557419`
Parent: `20f4ed8ce` (docs commit; the audit findings live in the tree at that SHA)
Brief: `.superpowers/sdd/plan/fix-backend-audit-brief.md`
Report: `.superpowers/sdd/plan/fix-backend-audit-report.md`

## Scope

Diff stat for `1f90e308`:

```
 character/position/processor_test.go               |  86 +++--
 monster/disease_callers_test.go                     | 254 +++++++--------
 monster/disease_targets_shell_test.go                | 360 +++++++++++----------
 monster/disease_targets.go => processor_disease_targets.go |   0
```

Matches the brief's file list exactly (FILE-01 rename + DOM-20 table-drive of
the three named test files). `disease_targets_test.go` was correctly left
untouched (not in the diff), matching the brief's explicit carve-out. No
production logic file other than the renamed one appears in the diff.
scope_confirmed: reviewed exactly this commit's 4 changed paths, plus the
renamed production file's content (byte-for-byte pre/post) and its
`routine.Go` fan-out.

## Finding 1 — FILE-01 rename (PASS)

- `git show 1f90e308 --stat -M` shows a pure `{disease_targets.go =>
  processor_disease_targets.go} | 0` — 0 insertions/deletions, git's own
  rename detection confirms no content change.
- Independently diffed `git show 1f90e308^:.../monster/disease_targets.go`
  against `git show 1f90e308:.../monster/processor_disease_targets.go` byte
  for byte: **identical**.
- `routine.Go(p.l, p.ctx, func(_ context.Context) {` present at line 73 of
  the post-rename file, unchanged — the fan-out this file carries was not
  touched, consistent with the report's claim that the file's body was never
  opened.

## Finding 2 — DOM-20 table-drive conversion (PASS, all 14 cases traced)

### `character/position/processor_test.go` (2 → 1, `TestProcessor_GetPosition`)

Diffed `20f4ed8:.../processor_test.go` against `1f90e308:.../processor_test.go`
line by line.

- Row "projects coordinates" reproduces `TestProcessor_GetPosition_ProjectsCoordinates`
  verbatim: same handler body (method/path/content-type/status/body
  assertions), same `wantX`/`wantY` (123, -456), same `characterId` (1001).
- Row "propagates not found" reproduces `TestProcessor_GetPosition_PropagatesNotFound`
  verbatim: same 404 handler, same `characterId` (9999), same
  `require.ErrorIs(t, err, requests.ErrNotFound)` assertion via `wantErr`.
- The one behavioral delta: the success case used `test.NewNullLogger()`
  pre-fix and now uses `logrus.New()` + `SetLevel(PanicLevel)` (matching the
  failure case, which already used that construction pre-fix). Both are
  discard-level loggers the test never asserts on — cosmetic, not a dropped
  assertion. `logrus/hooks/test` import removal is consistent with this
  (confirmed via `go build ./...` / `go vet` clean, no unused-import error).
- Non-vacuous: still exercises the real HTTP round trip and the real
  `requests.ErrNotFound` propagation path; nothing softened.

### `monster/disease_targets_shell_test.go` (8 → 1, `TestGetDiseaseTargets`)

Diffed `20f4ed8` against `1f90e308`. All 8 rows checked against their source
standalone tests:

| Row name | Pre-fix test | want / wantNoPositionCalls match? |
|---|---|---|
| boxless with multi-count returns controller only | `..._BoxlessWithMultiCountReturnsControllerOnly` | `want=[7]`, `wantNoPositionCalls=true` — matches |
| boxless with no controller returns nothing | `..._BoxlessWithNoControllerReturnsNothing` | `want=nil` (len 0), `wantNoPositionCalls=true` — matches |
| filters by bounding box | `..._FiltersByBoundingBox` | `want=[1 3]` out of `{1,2,3}` — matches (this is the 2-of-3 non-vacuous case) |
| preserves field listing order | `..._PreservesFieldListingOrder` | `want=[3 1]` — matches |
| position failure excludes only that character | `..._PositionFailureExcludesOnlyThatCharacter` | `want=[1 3]`, error map `{2: boom}` — matches |
| field listing failure returns nothing | `..._FieldListingFailureReturnsNothing` | `want=nil`, `wantNoPositionCalls=true` — matches |
| seduce caps across the shell | `..._SeduceCapsAcrossTheShell` | `want=[1 2]` out of `{1,2,3,4}`, count=2 — matches (2-of-4 non-vacuous case) |
| concurrent lookups preserve order | `..._ConcurrentLookupsPreserveOrder` | `want=` ascending 1..20, 20-goroutine fan-out with odd-id sleep — matches |

- The shared `diseaseTargetProcessor` helper (with its `sync.Mutex`-guarded
  `*positionCalls = append(...)` at lines 41-44 of both pre/post files) is
  byte-identical pre/post — confirmed via direct read of both files.
- The "concurrent lookups preserve order" row's own local `sync.Mutex` guard
  around its `positionCalls` append (inside the row's `build` closure,
  post-fix lines 192-195) is present and matches the pre-fix
  `TestGetDiseaseTargets_ConcurrentLookupsPreserveOrder` body (pre-fix lines
  204-207) — the mutex was not dropped in the conversion.
- Non-vacuous confirmed directly: "filters by bounding box" asserts `[1 3]`
  out of 3 in-box candidates (character 2 sits outside the box at x=400) and
  "seduce caps across the shell" asserts `[1 2]` out of 4 in-box candidates —
  both would fail against a whole-field selector that returned all
  candidates.

### `monster/disease_callers_test.go` (4 → 1, `TestExecuteDiseaseCaller`)

Diffed `20f4ed8` against `1f90e308`. All 4 rows checked:

| Row name | Pre-fix test | wantEventCount / wantTopic match? |
|---|---|---|
| dispel targets only in-box characters | `TestExecuteDispel_TargetsOnlyInBoxCharacters` | `2` / `EnvCommandTopicCharacterBuff` — matches (2-of-3 non-vacuous case) |
| dispel has no cap for non-seduce | `TestExecuteDispel_NoCapForNonSeduce` | `4` / `EnvCommandTopicCharacterBuff` — matches (FR-3.1 no-cap assertion preserved) |
| banish targets only in-box characters | `TestExecuteBanish_TargetsOnlyInBoxCharacters` | `2` / `EnvCommandTopicPortal`, banish-info hook installed — matches |
| banish with no banish map emits nothing | `TestExecuteBanish_NoBanishMapEmitsNothing` | `0` / `wantTopic=""` — matches; harness returns before the topic loop when `wantTopic==""` (post-fix line 121-123), and pre-fix the topic loop ran over an empty `*events` slice (0 iterations) — same effective behavior, not a dropped assertion |

- The `testInformationLookup` hook install/defer-restore semantics are
  preserved per-row via the `infoModel` func field (only installed when
  non-nil, restored via `defer` inside the `t.Run` closure) — matches the
  original per-test `defer func() { testInformationLookup = prevHook }()`
  pattern.
- "dispel targets only in-box characters" and "banish targets only in-box
  characters" both assert 2-of-3 candidates targeted (character 2 at x=400
  is outside the box) — non-vacuous against a whole-field selector.

## Build/test verification (independently re-run, not just trusted from report)

```
$ cd services/atlas-monsters/atlas.com/monsters
$ go build ./...
(clean)
$ go vet ./monster/... ./character/position/...
(clean)
```

Did not re-run the full `-race -count=5` sweep (module-local build+vet is
sufficient to confirm the diff compiles and is not vacuous by inspection;
the report already includes a full green `-race -count=5` transcript for the
touched packages, which is consistent with the row-by-row trace above).

## Verdict

All 14 original test cases survive as table rows with the same effective
inputs and the same assertions (see per-file tables above); none dropped,
merged into a shared closure that erases per-case behavior, or softened. The
one cosmetic delta (logger construction unification in
`processor_test.go`) does not touch an assertion. The FILE-01 rename is
byte-identical. The `-race` mutex guard and the `routine.Go` fan-out are both
intact and untouched.

No blocking findings.
