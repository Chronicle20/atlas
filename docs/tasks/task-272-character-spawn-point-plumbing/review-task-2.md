# Review: Task 2 — atlas-login spawnPoint plumbing

Commit range: `c4b1fd3be..a03a83ea4` (single commit `a03a83ea4`)
Module root: `services/atlas-login/atlas.com/login`

## Scope

`git diff --stat c4b1fd3be..a03a83ea4` shows exactly the five files listed in
the brief:

```
character/model.go            |  4 +-
character/rest.go             |  1 +
character/rest_test.go        | 15 +++++--
socket/writer/character_list.go      |  2 +-
socket/writer/character_list_test.go | 48 +++
```

`git diff --stat c4b1fd3be..a03a83ea4 -- services/atlas-character libs/atlas-packet`
returns empty — neither forbidden path appears in the diff. Matches the
brief's file list and constraints.

## Requirement-by-requirement

1. **`Model.SpawnPoint()` returns `uint32` and returns `m.spawnPoint`.**
   `character/model.go:222-224`:
   ```go
   func (m Model) SpawnPoint() uint32 {
       return m.spawnPoint
   }
   ```
   PASS. Confirmed `spawnPoint uint32` field exists at `model.go:44` and is
   copied in the builder's `Build()` at `model.go:311`.

2. **Narrowing to `byte` only at the wire call site, via explicit cast.**
   `socket/writer/character_list.go:56`: `uint32(mapId), byte(c.SpawnPoint()),`
   — the only `byte(...)` cast of `SpawnPoint()` in the diff. PASS.

3. **`services/atlas-character/**` and `libs/atlas-packet/**` absent from
   diff.** Confirmed via targeted `git diff --stat` above (empty output).
   PASS.

4. **Fixtures use non-zero `spawnPoint`.** `rest_test.go` fixture uses
   `SetSpawnPoint(25)`; writer test table uses `7` and `256` (truncation
   case, product `0` is the *expected output*, not the fixture's set value).
   PASS.

5. **No `Extract∘Transform` idempotence-only proof; assertions compare
   against literal input.** `rest_test.go` asserts `got.SpawnPoint() != 25`
   and `got.spawnPoint != m.spawnPoint` (both against the literal `25`
   used to build `m`). Writer test asserts `entry.Statistics().SpawnPoint()
   != tt.want` against literal table values. PASS.

6. **No producer for `spawnPoint`; `UpdateSpawnPoint` not called; no dedup
   of the eight model copies; Rank/RankMove/JobRank/JobRankMove stubs
   untouched.** Grep of the diff for `UpdateSpawnPoint` and the four
   Rank-family identifiers finds no hits inside the changed hunks; the
   `character_list.go` line touched is only the `SpawnPoint` argument.
   PASS.

7. **Test setup for `Model` uses the `Builder`; `RestModel` struct literal
   is not required here since the test flows through `Transform`/`Extract`
   round trip, and the writer test uses `character.NewBuilder()`.**
   Confirmed both test files use `NewBuilder()`; no ad hoc struct literals
   or test-only constructors introduced. PASS.

8. **Doc comment on `TestTransformRoundTrip` corrected to remove the
   stale "never calls SetSpawnPoint" claim.** `rest_test.go:10-16` — the
   `SetSpawnPoint` fragment is removed from the "never calls" list and a
   new sentence "spawnPoint DOES survive as of task-272 and is asserted
   below" is added. Verified against the actual `Extract` builder chain
   (`rest.go:129-157`): `SetPets`, `SetEquipment`, `SetInventory`,
   `SetRank`, `SetRankMove`, `SetJobRank`, `SetJobRankMove` are indeed
   absent from the chain, so the corrected comment is accurate — not just
   edited to match the brief's suggested text but actually true of the
   code. PASS.

## Correctness of the change itself

- The `byte()` cast at `character_list.go:56` truncates any value above 255
  (wire format is one byte); the new writer test pins this explicitly with
  a `set: 256, want: 0` case and documents it as a pre-existing wire-format
  property, not a new bug. This is honest framing — the task is plumbing
  `spawnPoint` through, not fixing the wire format's byte width.
- `RestModel.SpawnPoint` field (`rest.go:38`, `json:"spawnPoint"`) already
  existed pre-task (part of the `services/atlas-character` REST payload
  contract, out of scope here) and is now consumed by `Extract`. No new
  REST/JSON field was introduced in this diff.
- `Transform` (`rest.go:124`) already emitted `SpawnPoint: m.spawnPoint` —
  confirmed unmodified in this diff — so the round trip only needed the
  `Extract` side wired, which is exactly what changed.

## Test honesty (would the new/amended tests fail without the fix)

- `TestTransformRoundTrip`: with the stub `SpawnPoint() byte { return 0 }`
  and no `SetSpawnPoint` call in `Extract`, `got.SpawnPoint()` would be `0`
  (or fail to compile against the `!= 25` comparison depending on prior
  return type — either way the assertion is not satisfiable pre-fix).
  Reproduced locally that the test currently passes against the live
  (fixed) code:
  ```
  --- PASS: TestTransformRoundTrip (0.00s)
  ```
- `TestToCharacterListEntry_SpawnPoint`: pre-fix, `c.SpawnPoint()` was
  hard-stubbed to return `0` regardless of what the builder set, so both
  subtests ("in range" want 7, "truncates above 255" want 0-but-for-the-
  wrong-reason) would fail/be meaningless pre-fix; post-fix the "in range"
  case is a real, tight pin (7→7) and would fail immediately if the field
  wired incorrectly. Reproduced locally:
  ```
  --- PASS: TestToCharacterListEntry_SpawnPoint (0.47s)
      --- PASS: TestToCharacterListEntry_SpawnPoint/in_range (0.27s)
      --- PASS: TestToCharacterListEntry_SpawnPoint/truncates_above_255 (0.20s)
  ```
- Did not revert the fix and re-run to observe RED, per the "do not
  re-run the module test suite" instruction and to avoid transient edits
  to tracked files during review; the implementer's report documents the
  same choice and reasoning. The literal-value assertion structure (not
  idempotence-based) makes the tests structurally load-bearing regardless.

## Cross-service seams

None applicable — this task is entirely internal to `atlas-login`; it
consumes an already-existing REST field (`RestModel.SpawnPoint`) and an
already-existing packet-model accessor
(`CharacterStatistics.SpawnPoint() byte`, `libs/atlas-packet`) without
modifying either producer's contract. Both referenced accessors were
verified to exist exactly as the brief described:
- `libs/atlas-packet/model/character_statistics.go:86` —
  `func (m CharacterStatistics) SpawnPoint() byte { return m.spawnPoint }`
- `libs/atlas-packet/model/character_list_entry.go:37` —
  `func (m CharacterListEntry) Statistics() CharacterStatistics`

## Build and test verification (module-local, not the full monorepo suite)

Ran from `services/atlas-login/atlas.com/login`:
```
go build ./...
go test ./character/... ./socket/writer/... -run 'TransformRoundTrip|SpawnPoint' -v
```
Build succeeded with no output (no errors). Both targeted tests pass, output
matches the implementer's report verbatim.

## Not evaluable

- Did not re-run the full `atlas-login` module test suite (`go test ./...`)
  per the task instruction to rely on the implementer's report for that
  evidence; relied on the report's pasted "all packages pass" output for
  suite-wide confidence beyond the two directly-relevant packages.
- "Observable wire output must be unchanged (the persisted column is
  always 0 today)" is a runtime/data claim about `atlas-character`'s
  current DB state, not something verifiable from this diff or module
  alone; taken as background context, not a claim this diff needs to
  prove.

## Verdict

No blocking findings. Diff matches the brief file-for-file and
requirement-for-requirement; forbidden-path constraints hold; test
fixtures use non-zero values and assert against literal inputs, not
idempotence; doc-comment correction verified accurate against the actual
`Extract` builder chain; targeted tests reproduced passing locally.
