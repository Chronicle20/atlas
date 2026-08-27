# Review: task-246 cleanup commit (7b2196004)

Range reviewed: `6bd233780..7b2196004` (single commit `7b2196004`).
Brief: `.superpowers/sdd/plan/cleanup-brief.md`.
Implementer report: `.superpowers/sdd/plan/cleanup-report.md`.

## Scope

`git diff --stat 6bd233780..7b2196004`:

```
services/atlas-character-factory/atlas.com/character-factory/data/skill_requests_test.go  | 90 ++++++++++++++++++++++
services/atlas-character-factory/atlas.com/character-factory/factory/resource.go          | 38 +++++----
services/atlas-character-factory/atlas.com/character-factory/factory/resource_test.go     | 15 ++--
3 files changed, 118 insertions(+), 25 deletions(-)
```

Exactly the three files named in the brief. Nothing outside
`services/atlas-character-factory/atlas.com/character-factory` was touched; confirmed
`git diff 6bd233780..7b2196004 -- services/atlas-channel` is empty. Scope matches the brief.

## Findings

### 1. `factory/resource.go` — package-level mutable state genuinely removed

`grep -rn "mapleLifeProcessor" services/atlas-character-factory/.../character-factory/`
returns no hits post-commit — the package-level `var mapleLifeProcessor = NewProcessor` is gone,
not renamed. It is replaced by `newMapleLifeHandler(newProcessor func(logrus.FieldLogger) Processor) func(...) http.HandlerFunc`
(`factory/resource.go:87-102`), a pure function that closes over an injected factory — no
package-level mutable state remains.

`handleCreateMapleLife` (`factory/resource.go:80-82`) now reads:
```go
func handleCreateMapleLife(d *rest.HandlerDependency, c *rest.HandlerContext, in MapleLifeCreateRestModel) http.HandlerFunc {
	return newMapleLifeHandler(NewProcessor)(d, c, in)
}
```
This is a thin one-line delegation to the new helper, not a literal inline of
`NewProcessor(d.Logger())` the way `handleCreateFromPreset` (`factory/resource.go:116-129`) and
`handleCreateCharacter` (`factory/resource.go:159-176`) do. The brief's wording ("structurally
consistent with its siblings") is satisfied in the sense that matters — production call site is
`NewProcessor(d.Logger())` with identical body/error-handling/response logic to the siblings, and
the seam is a pure closure rather than mutable state — but the three handlers are not
byte-for-byte structurally identical; `handleCreateMapleLife` is the only one with an extra
indirection layer. This is the natural cost of making the Processor injectable without
package-level state and is not a defect; noting it as a non-blocking observation since the brief
used the word "structurally consistent" and a strict reading could expect full inlining.

**Verdict: PASS** (genuine removal, not a rename; non-blocking note on exact structural shape).

### 2. `TestMapleLifeRouteReturnsAcceptedWithTransactionId` still gates the handler

`postMapleLife` now builds the handler under test via
`newMapleLifeHandler(func(l logrus.FieldLogger) Processor { return fakeMapleLifeProcessor{...} })`
and drives it through `server.ParseInput[MapleLifeCreateRestModel](d, c, handler)` — the real
JSON:API decode path is preserved.

Mutation check performed directly (not merely re-reading the diff):
- Changed `factory/resource.go:103` (inside `newMapleLifeHandler`) from `w.WriteHeader(http.StatusAccepted)` to `w.WriteHeader(http.StatusTeapot)`.
- Ran `go test ./factory/... -run TestMapleLifeRouteReturnsAcceptedWithTransactionId -v`:
  ```
  resource_test.go:172: Expected status 202, got 418, body: {...}
  --- FAIL: TestMapleLifeRouteReturnsAcceptedWithTransactionId (0.00s)
  ```
- Reverted (`cp` from a pre-edit backup) and re-ran: PASS. `git diff -- factory/resource.go` empty afterward.

The test fails when the handler's behavior is actually wrong; the refactor did not neuter it.

**Verdict: PASS.**

### 3. `data/skill_requests_test.go` — pins real JSON wire shape, not a Go round-trip

`TestGetSkillsByIds_DecodesEffectX` stands up an `httptest.Server` returning a literal JSON:API
document with `"effects": [{"x": 0}, {"x": 12}, {"x": 24}]`, points `DATA_SERVICE_URL` at it, and
drives the real `NewProcessor(...).GetSkillsByIds` → `requestSkillsByIds` →
`jsonapi.Unmarshal` → `SkillRestModel` → `SkillInfo` path. It does not construct a Go
`SkillRestModel`/`effect.RestModel` value and round-trip it — the JSON is literal text in the
test, and decoding goes through the real HTTP/JSON:API stack.

Independently verified the JSON tags against atlas-data's source (not trusting the test's own
comment):
- `services/atlas-data/atlas.com/data/skill/effect/rest.go:38`: `X int16 \`json:"x"\`` — matches.
- `services/atlas-data/atlas.com/data/skill/rest.go:15-16`: `MaxLevel uint8 \`json:"maxLevel"\`` and `Effects []effect.RestModel \`json:"effects"\`` — matches the fixture's `"maxLevel"` and `"effects"` keys.
- Character-factory's own `SkillRestModel`/`SkillEffectRestModel` (`data/skill_requests.go:12-29`) mirror these tags exactly (`json:"x"`, `json:"maxLevel"`, `json:"effects"`), and `SkillInfo.EffectX` (`data/processor.go:14-21`) is populated from them in `ProcessorImpl.GetSkillsByIds` (`data/processor.go:58-72`).

Mutation check performed directly:
- Changed `data/skill_requests.go:19` from `` X int16 `json:"x"` `` to `` X int16 `json:"xx"` ``.
- Ran `go test ./data/... -run TestGetSkillsByIds_DecodesEffectX -v`:
  ```
  skill_requests_test.go:87: EffectX[1] = 0, want 12
  skill_requests_test.go:87: EffectX[2] = 0, want 24
  --- FAIL: TestGetSkillsByIds_DecodesEffectX (0.01s)
  ```
- Reverted, re-ran: PASS. `git diff --stat` clean afterward.

This closes the gap named in Task 22's review: a wire-shape mismatch on `Effects[].X` →
`SkillInfo.EffectX` now fails a real test instead of being masked by `dmock.ProcessorMock`.

**Verdict: PASS.**

### 4. No production-behaviour change; no out-of-scope files touched

- `go build ./...`, `go vet ./...`, `go test ./...` from
  `services/atlas-character-factory/atlas.com/character-factory` all pass post-commit (no
  failures, no vet warnings).
- `handleCreateMapleLife`'s production code path (`NewProcessor(d.Logger())` →
  `processor.CreateMapleLife` → same status-code/marshal logic) is unchanged in behavior; only the
  seam used to substitute the processor in tests changed.
- `git diff 6bd233780..7b2196004 --name-only` lists exactly the three files above; atlas-channel's
  `newFactoryProcessorFunc` seam (explicitly ruled correct as-is in the brief) is untouched.

**Verdict: PASS.**

## Not evaluable

None — all four checklist items were directly verifiable within the commit's diff plus the one
external file (`services/atlas-data/atlas.com/data/skill/effect/rest.go` and
`services/atlas-data/atlas.com/data/skill/rest.go`) the new test's correctness genuinely depends
on.

## Tree state

Working tree confirmed clean after every mutation check (backups restored, `git diff --stat`
empty for all touched files). Pre-existing untracked docs/tools files unrelated to this review
(`docs/tasks/task-246-.../review-task-*.md`, `agent-ledger.tsv`, etc.) were present before this
review began and are outside this commit's scope.

## Verdict

APPROVED. All four checklist items pass with direct verification (mutation-tested, not just
read). One non-blocking structural note on item 1 (extra indirection layer vs. siblings' literal
inlining) — does not affect correctness or test integrity.
