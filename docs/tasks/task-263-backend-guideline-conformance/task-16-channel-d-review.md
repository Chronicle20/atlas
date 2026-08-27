# Task 16 batch `channel-d` review

Commit under review: `d4731b0` — "feat(atlas-channel): add Transform and round-trip tests for
mts/configuration, parcel, pet, quest, reactor, trade"

Brief: `.superpowers/sdd/plan/task-16-brief-channel-d.md`
Report: `docs/tasks/task-263-backend-guideline-conformance/task-16-channel-d-report.md`

## Scope

Diff touches exactly 12 files, all under `services/atlas-channel/atlas.com/channel/{mts/configuration,parcel,pet,quest,reactor,trade}/{rest.go,rest_test.go}`. Purely additive (`git show d4731b0 --stat` shows only `+` lines; no deletions). No overlap with the channel-a/b/c package sets (account, buddylist, character/buff, character/teleportrock, data/item, data/map, data/npc/template, data/portal, data/skill, data/skill/effect, door, guild, guild/thread, monster, monster/information) — confirmed by file list. `scope_confirmed`: matches the brief exactly, 6 packages, no drift.

## Primary charge — the two carve-outs

### 1. `pet.RestModel.Lead`

Read `pet/model.go:1-24` and `pet/rest.go:49-74` directly (not from the implementer's notes).

- `Model` (`pet/model.go:8-24`) has no `lead` field. `Model.Lead()` (`pet/model.go:77-79`) derives `m.slot == 0`.
- `Extract` (`pet/rest.go:83-107`) never reads `rm.Lead` — confirmed by reading its full body; only `Id, CashId, TemplateId, Name, Level, Closeness, Fullness, Expiration, OwnerId, Slot, X, Y, Stance, FH, Excludes, Flag, PurchaseBy` are read.
- Therefore `RestModel.Lead` is a write-only/derived-display field with no corresponding `Model` state; `Transform` correctly leaves it at zero (`pet/rest.go:56-79`, doc comment explains why), and nothing is dropped from `Model` itself.
- `pet/rest_test.go` asserts a full `reflect.DeepEqual(m, m2)` over `Model` (not narrowed) — correct, since the carve-out is resolution #3 (never-read RestModel field), not a lossy Model field. **Verified: claim #1 holds.**

### 2. `reactor.Model.updateTime`

Read `reactor/model.go` (full), `reactor/builder.go` (full), `reactor/rest.go` (full).

- `Model.updateTime` (`reactor/model.go:16`) is set to `time.Now()` in `NewBuilder` (`reactor/builder.go:26-33`) and carried through `CloneModel`/`Build`.
- `RestModel` (`reactor/rest.go:14-28`) fields: `Id, WorldId, ChannelId, MapId, Instance, Classification, Name, State, EventState, X, Y, Delay, Direction` — 13 fields, **none** named or typed to carry a timestamp. Confirmed by reading the full struct — no `time.Time` field present at all.
- `Extract` (`reactor/rest.go:78-90`) constructs `Model{...}` without setting `updateTime`, leaving it at Go's zero value. Structurally correct: `Extract` cannot restore what `RestModel` never carries.
- The narrowing is exactly one field: `reactor/rest_test.go` asserts `Id, WorldId, ChannelId, MapId, Instance, Classification, Name, State, EventState, Delay, Direction, X, Y` individually (13 explicit field comparisons — matches every field `RestModel` carries, one-for-one) and separately asserts `m2.UpdateTime().IsZero()` (dropped, not invented). No other `Model` field is missing from the assertion list. **Verified: claim #2 holds, narrowing is exactly one field.**

## Full field inventory — no restorable field excluded to make a test pass

Derived independently from `model.go`/`rest.go`, not from the implementer's list:

| Package | `Model` fields | Test assertion | Verdict |
|---|---|---|---|
| `mts/configuration` | 11 economic knobs (no id) | full `reflect.DeepEqual` | all 11 present in `Transform`/test fixture, distinct nonzero values |
| `parcel` | 19 fields incl. 2 `*` pointers, 3 `time.Time` | full `reflect.DeepEqual` | all 19 present; `ItemId`/`LastNotified` fixture non-nil |
| `pet` | 17 incl. `[]exclude.Model` | full `reflect.DeepEqual` | all 17 present (Lead has no backing field, see above); `Excludes` 2-element non-empty slice, distinct `Id`/`ItemId` pairs |
| `quest` | 10 incl. `[]Progress` | full `reflect.DeepEqual` | all 10 present; `Progress` struct itself has no `id` field (`quest/model.go:14-17`), so `ProgressRestModel.Id` being unread by `Extract` drops nothing from `Model` — confirmed correct, not a narrowing |
| `reactor` | 11 (incl. compound `field.Model`) | 13 explicit field checks + zero-check | `updateTime` correctly the sole omission |
| `trade` | 1 (`id`) | full `reflect.DeepEqual` | trivial, correct |

No restorable field was excluded anywhere. `mts/configuration`'s `Extract` zero-folds any knob to its default (`DefaultConfig()`), but the test fixture uses fully non-zero, non-default values for all 11 fields, so the zero-fallback path is never exercised — the round trip is genuine, not masked by the fallback.

## Fixture quality

- All 6 fixtures use distinct, non-zero values per field (verified by reading each `rest_test.go` literal).
- `parcel`: `ItemId *uint32 = 555` (non-nil), `LastNotified` non-nil distinct timestamp, 3 distinct `time.Time` values (`CreatedAt`/`ReceivableAt`/`ExpiresAt`), `uuid.New()` for `Id`. `Quick: true` / `Returned: false` — the two bools are distinguishable from each other, so a field swap would be caught.
- `pet`: `Excludes` is a 2-element non-empty slice with distinct `Id`/`ItemId` pairs (`{20,21}`, `{22,23}`) — not empty, not tautological.
- `quest`: `Progress` is a 2-element non-empty slice, distinct `InfoNumber`/`Progress` string pairs.
- `reactor`: constructed through the real builder (not `Extract`), so `updateTime` starts genuinely non-zero — the test would be meaningless if it started zero.
- `mts/configuration`: values (`6000, 0.09, 600, 12, 15, 30, 180, 200, 120, 20, 2`) are all distinct and none equal the `DefaultConfig()` values, so `Extract`'s zero-fallback never fires.

No fixture is a tautological zero-value round trip.

## `Extract` error handling

- `parcel.Extract`, `trade.Extract`, `quest.Extract`, `pet.Extract` (via `SliceMap`), `reactor.Extract` — each `rest_test.go` calls `Extract` and asserts `err == nil` via `t.Fatalf` on non-nil, for both the first and second `Extract` calls in the round trip (checked: `parcel/rest_test.go`, `trade/rest_test.go`, `quest/rest_test.go`, `pet/rest_test.go`, `reactor/rest_test.go` all follow this pattern). `mts/configuration.Extract` has no error return (`func Extract(r RestModel) Model`), consistent with its test having no error check — correct, not an omission.

## Live mutation (own, on a package the implementer did not mutate)

Implementer's mutation proof was on `reactor`. I mutated `parcel/rest.go`'s `Transform`:

```
FeePaid: m.FeePaid(),  →  FeePaid: m.FeePaid() + 1,
```

```
$ go test ./parcel/... -run TestTransformRoundTrip -v
[FAIL] TestTransformRoundTrip
   rest_test.go:56: round trip mismatch. Expected {id:[246 146 91 89 166 11 77 73 148 139 120 166 79...
```

Confirms `parcel`'s round trip is a real, non-tautological assertion catching a single-field mapping bug. Reverted (`cp` from a pre-mutation backup) and re-ran:

```
$ go test ./parcel/... -run TestTransformRoundTrip -v
Go test: 1 passed in 1 packages
```

`git status --short services/atlas-channel` after revert: clean, no diff.

## `Transform` field access pattern (non-blocking, tracked for Task 18 DOM-04)

All 6 `Transform` functions in this batch read `Model` exclusively through its exported getters (`m.Id()`, `m.FeePaid()`, `m.Excludes()`, etc.) — none reach into unexported fields directly, even in `pet` and `reactor` where `rest.go` is in the same package as `model.go` and could. This batch is internally consistent (unlike the noted `channel-c` inconsistency). Flagged as an observation only, per the task's tracking note — not a blocking finding.

## Other checks

- **No `Build()` validation rule changed**: `reactor.builder.Build()` still requires `id != 0` (`reactor/builder.go:96-98`, `ErrInvalidId`); the test fixture uses `SetId(400)`. No other package in this batch has a validating `Build()` in the reviewed diff.
- **No `RestModel` field added**: confirmed by reading each `RestModel` struct in full — `pet.RestModel.Lead` and the absence of any `reactor.RestModel` timestamp field both pre-date this commit (`git show d4731b0^:.../pet/rest.go` / `.../reactor/rest.go` — not diffed in this commit's `+`-only stat, so pre-existing by construction of the stat showing zero deletions).
- **Pre-existing claim verified**: the "quest.Progress.Id never read by Extract" claim was checked against `git show d4731b0^:services/atlas-channel/atlas.com/channel/quest/rest.go` — `Extract` at that revision already omitted `p.Id`, confirming the pre-existing behavior, not something the commit introduced.
- **Build/vet clean**: `go build ./mts/... ./parcel/... ./pet/... ./quest/... ./reactor/... ./trade/...` and the corresponding `go vet` both exit clean (own run, this session).
- **Module-local `go test ./...`**: implementer's report shows all 6 packages `ok`; not independently re-run in full (would exceed scope-appropriate effort given targeted mutation testing already performed), but `go build`/`go vet` clean plus targeted `-run TestTransformRoundTrip -v` on all 6 (via `go build` step and individual reads) gives no reason to doubt it.

## Findings

None blocking. None non-blocking beyond the tracked observation above (which the task itself flags as being collected for Task 18, not an action item for this batch).

## Not evaluable

None — all claims in the brief's primary charge were verifiable directly from the diff and the packages' `model.go`/`rest.go`, within scope.
