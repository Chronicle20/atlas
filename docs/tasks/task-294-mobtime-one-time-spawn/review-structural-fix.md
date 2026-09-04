# Review: structural backend-audit fix (task-294)

Range: `4457c3547..e001bc7c4` (2 commits)
- `70035541a` refactor(maps): split map/monster registry into administrator and provider
- `e001bc7c4` refactor(maps): add builders for spawn point models, EXT-01 rest setters

Brief: `.superpowers/sdd/plan/fix-audit-structural-brief.md`
Report: `.superpowers/sdd/plan/fix-audit-structural-report.md`

## Scope

Reviewed both commits in full: the registry split (Part A) and the two new
builder.go files + EXT-01 rest.go setters (Parts B/C). Verified verbatim-move
claim by mechanical extraction/diff of every moved function and Lua script var
against the pre-image, not by inspection alone. Ran module-local
`go build`, `go vet`, `gofmt -l .`, `go test ./... -count=1` from
`services/atlas-maps/atlas.com/maps`.

## Findings

### PASS — Part A is a pure relocation, byte-for-byte

`git diff 4457c3547..70035541a` touches only
`map/monster/{registry,administrator,provider}.go`. Extracted every function
body (with its full doc comment) and every `var ...Script = goredis.NewScript(...)`
block from the pre-image `registry.go` and diffed each against its landing
site in `administrator.go` / `provider.go`, byte-for-byte via a small Python
script (not `git diff`, which would just show relocation noise):

- 8 write methods in `administrator.go`: `InitializeForMap`,
  `ReserveEligibleSpawnPoints`, `ResetCooldown`, `ClaimOneTimeSpawnPoints`,
  `RearmOneTime`, `Reset`, `FlushTenant`, `SetSpawnPointsForMap` — all
  MATCH, same receiver `*SpawnPointRegistry`, same body byte-for-byte.
- 3 read methods in `provider.go`: `Count`, `CountOneTime`,
  `GetSpawnPointsForMap` — all MATCH.
- 5 Lua script vars, all present and byte-identical in `administrator.go`:
  `initializeScript`, `reserveEligibleScript`, `resetCooldownScript`,
  `claimOneTimeScript`, `rearmOneTimeScript`.
- `rearmOneTimeScript` specifically (`administrator.go:1-3`, in the extracted
  block) still reads `return redis.call('HDEL', KEYS[1], ARGV[1])`, and
  `RearmOneTime` (`administrator.go`, tail of file) still calls
  `rearmOneTimeScript.Run(...)` and interprets the result as `deleted > 0` —
  the 16fc28cde atomicity fix survived the move intact, correctly paired.
- `registry.go` retains exactly the plumbing set named in the brief:
  `storedSpawnPoint`, `SpawnPointRegistry`, const/var blocks, `fieldSuffix`,
  `newRegistry`, `InitRegistry`, `GetRegistry`, `recurringKey`/`oneTimeKey`/
  `metaKey`, `toStored`/`fromStored`. No write or read method remains there.
- Import blocks of all three files are pruned to exactly what each file uses
  (verified `registry.go` dropped `context`, `encoding/json`, `strconv`,
  `uuid`, `logrus`; `provider.go` imports only `character`, `context`,
  `encoding/json`; `administrator.go` carries the full original set).

### PASS — Part B builders match `mist/builder.go` shape and the hard constraint

- `map/monster/builder.go` (new): `NewBuilder()` → fluent `SetSpawnPoint`,
  `SetNextSpawnAt` → `Build() CooldownSpawnPoint`. Doc comment explicitly
  states the fields stay exported and why (literal construction elsewhere,
  `toStored`/`fromStored` JSON round-trip dependency) — matches the brief's
  hard constraint verbatim.
- `data/map/monster/builder.go` (new): `NewBuilder()` → one setter per each
  of the 12 `SpawnPoint` fields (`SetId`, `SetTemplate`, `SetMobTime`,
  `SetTeam`, `SetCy`, `SetF`, `SetFh`, `SetRx0`, `SetRx1`, `SetX`, `SetY`,
  `SetHide`) → `Build() SpawnPoint`. Field list and types match
  `data/map/monster/model.go` exactly, including the `Hide` field added by
  a prior task-294 commit.
- Both `Build()` methods return a value type with no validation, mirroring
  `mist/builder.go`'s documented no-validation posture — consistent, not a
  deviation flagged as a defect.
- Confirmed **known-accepted**: neither struct's fields were unexported.
  Per instructions this is not reported as a finding — it was an explicit
  brief constraint, and the report records the same rationale
  (`fix-audit-structural-report.md`, "Constraint recorded per brief").

### PASS — Part C EXT-01 setters verbatim

`git diff 70035541a..e001bc7c4 -- .../data/map/monster/rest.go` shows exactly
the two-method addition, textually identical to
`data/map/info/rest.go:32-38`:

```go
func (r *RestModel) SetToOneReferenceID(_ string, _ string) error { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
```

### PASS — no test file touched

`git diff --stat 4457c3547..e001bc7c4 -- '*_test.go'` is empty across both
commits.

### PASS — module-local verification, from `services/atlas-maps/atlas.com/maps`

```
go build ./...          → clean, no output
go vet ./...             → clean, no output
gofmt -l .                → clean, no output
go test ./... -count=1   → all packages ok, including map/monster (5.77s)
                            and data/map/monster (0.024s) — the two touched
                            packages — with zero test modifications, i.e.
                            every pre-existing test asserts the restructure
                            didn't change behavior.
```

`tools/verify.sh` was intentionally not run, per instructions.

### Not evaluable

- None. The unit is entirely module-local Go code with no cross-service
  seam (no Kafka message, no REST contract change beyond the two no-op
  EXT-01 setters, which are JSON:API plumbing already proven by the
  `data/map/info` precedent).

## Verdict rationale

Both commits do exactly what the brief specifies: Part A is a mechanically
verified verbatim move (every function and Lua script byte-identical,
correct file landing per the brief's explicit checklist), Part B adds
builders that respect the documented hard constraint, Part C copies the
EXT-01 pattern exactly. No test files touched. Build/vet/fmt/test all clean.
No blocking findings.
