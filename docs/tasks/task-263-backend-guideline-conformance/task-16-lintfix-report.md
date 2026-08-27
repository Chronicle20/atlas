# Task 16 fix round — clear `lint & format guard` FAIL

## Summary

Cleared both failure classes reported by `tools/verify.sh --quick --base c3ad6b3c4` for the
`lint & format guard` check. No correctness change — `go build`/`go vet`/`go test` already passed
on all five affected modules before this round and continue to pass after.

## Class 1 — gofumpt formatting

Ran `tools/lint.sh --fmt --go <module>` per module (the guard's own fix-mode entry point, not bare
`gofmt`):

- `services/atlas-channel/atlas.com/channel`
- `services/atlas-character/atlas.com/character`
- `services/atlas-messages/atlas.com/messages`
- `services/atlas-monsters/atlas.com/monsters`

All rewrites were import-block reordering (gofumpt groups a package's own-module imports before
third-party/stdlib) or struct-literal field alignment whitespace. Reviewed every diff by hand — no
assertion, expected value, or test logic changed in any `_test.go` file.

One extra file not named in the brief was reformatted by the same `--fmt` pass:
`services/atlas-channel/atlas.com/channel/guild/thread/rest_test.go` (import reorder only, same
class as the 8 enumerated sites, part of Task 16's own new test files). Included since it is the
same mechanical fix and leaving it would have left the module non-clean.

## Class 2 — staticcheck S1016 struct conversions

For each site, read both struct declarations first to confirm field-for-field identity (same
names, order, types) before converting:

- `services/atlas-channel/atlas.com/channel/monster/information/rest.go:32` —
  `AttackInfo{Pos, ConMP, AttackAfter}` vs `AttackInfoRestModel{Pos, ConMP, AttackAfter}` — identical.
  `AttackInfoRestModel{Pos: a.Pos, ConMP: a.ConMP, AttackAfter: a.AttackAfter}` → `AttackInfoRestModel(a)`.
- `services/atlas-monsters/atlas.com/monsters/monster/information/rest.go:92` —
  `Skill{Id, Level uint32}` vs local `skill{Id, Level uint32}` — identical.
  `skill{Id: s.Id, Level: s.Level}` → `skill(s)`.
- `services/atlas-monsters/atlas.com/monsters/monster/information/rest.go:96` —
  `AttackInfo{Pos, ConMP, AttackAfter}` vs `AttackInfoRestModel{Pos, ConMP, AttackAfter}` — identical.
  `AttackInfoRestModel{Pos: a.Pos, ConMP: a.ConMP, AttackAfter: a.AttackAfter}` → `AttackInfoRestModel(a)`.
- `services/atlas-summons/atlas.com/summons/data/skill/effect/rest.go:31` —
  `StatChange{Type string, Amount int32}` vs `StatupRestModel{Type string, Amount int32}` — identical.
  `StatupRestModel{Type: su.Type, Amount: su.Amount}` → `StatupRestModel(su)`.

All four conversions compiled cleanly with no struct reshaping — every premise held.

## Verification

Per module: `go build ./... && go vet ./... && go test ./...` — exit 0, no failing/skipped tests,
output pristine (only `ok`/`no test files` lines) for all five modules.

Then `tools/lint.sh --check --fmt --go <module>`:

```
$ tools/lint.sh --check --fmt --go services/atlas-channel/atlas.com/channel
lint.sh: OK

$ tools/lint.sh --check --fmt --go services/atlas-character/atlas.com/character
lint.sh: OK

$ tools/lint.sh --check --fmt --go services/atlas-messages/atlas.com/messages
lint.sh: OK

$ tools/lint.sh --check --fmt --go services/atlas-monsters/atlas.com/monsters
lint.sh: OK

$ tools/lint.sh --check --fmt --go services/atlas-summons/atlas.com/summons
lint.sh: OK
```

## Files changed

- `services/atlas-channel/atlas.com/channel/data/skill/effect/rest_test.go` — gofumpt (import order)
- `services/atlas-channel/atlas.com/channel/data/skill/rest_test.go` — gofumpt (import order)
- `services/atlas-channel/atlas.com/channel/guild/rest_test.go` — gofumpt (import order)
- `services/atlas-channel/atlas.com/channel/guild/thread/rest_test.go` — gofumpt (import order,
  not in brief but same class, part of Task 16's own new tests)
- `services/atlas-channel/atlas.com/channel/monster/information/rest.go` — S1016 conversion
- `services/atlas-character/atlas.com/character/data/skill/effect/rest_test.go` — gofumpt
- `services/atlas-character/atlas.com/character/data/skill/rest_test.go` — gofumpt
- `services/atlas-messages/atlas.com/messages/data/skill/effect/rest_test.go` — gofumpt
- `services/atlas-messages/atlas.com/messages/data/skill/rest_test.go` — gofumpt
- `services/atlas-monsters/atlas.com/monsters/monster/information/rest.go` — S1016 conversions (x2)
- `services/atlas-monsters/atlas.com/monsters/monster/information/rest_test.go` — gofumpt (struct
  literal field alignment)
- `services/atlas-summons/atlas.com/summons/data/skill/effect/rest.go` — S1016 conversion

## Self-review

- Confirmed every `_test.go` diff was whitespace/import-order only via `git diff`; no assertion
  values changed.
- Confirmed each S1016 conversion's two struct declarations by reading both, before editing —
  all four were genuinely field-identical (name, order, type); none required reshaping.
- No struct field added, removed, or reordered anywhere.
- `go build`/`go vet`/`go test` clean on all five modules; `tools/lint.sh --check --fmt --go`
  reports OK on all five.
- Did not run `tools/verify.sh` (controller re-runs the gate).
- Did not touch anything under `docs/` except this report.

## Commit

`6a575d0` — `fix(task-263): clear gofumpt and staticcheck S1016 findings from Task 16`
Branch confirmed: `task-263-backend-guideline-conformance`.

No concerns.
