# Review — fix round 2: `modelBuilder.Build()` invariant (DOM-01 closure)

Commit: `353d3bd10` — "fix(atlas-character): validate modelBuilder.Build() invariant (DOM-01 closure)"
Brief: `.superpowers/sdd/plan/fix-round-2-brief.md`
Report: `.superpowers/sdd/plan/fix-round-2-report.md`
Convention authority: `docs/tasks/task-272-character-spawn-point-plumbing/builder-validation.md`

## Verdict: APPROVED

## Scope

Single commit, 21 files, `services/atlas-character/atlas.com/character` +
`docs/tasks/task-272-character-spawn-point-plumbing/builder-validation.md`.
Matches the brief's fence exactly — `git show --stat 353d3bd10` shows nothing
outside those two roots. No `libs/` change (confirms `model.Decorator` was
not widened — its owning file `libs/atlas-model/model/processor.go` is
untouched by this commit).

## 1. Is the derived invariant honest and correct?

**`builder.go:340-352`** (new `Build()`):
```go
func (c *modelBuilder) Build() (Model, error) {
	if c.accountId != 0 && c.name == "" {
		return Model{}, errors.New("name is required when accountId is set")
	}
	...
}
```

### Enumerated every construction site myself (not the report's word)

Production sites (`grep -rln "NewEmptyBuilder\|CloneModel" --include="*.go" .`
in the module, excluding `_test.go`, gives exactly 4 files):

- `character/provider.go:59-88` `modelFromEntity` — sets `SetAccountId(e.AccountId)` and `SetName(e.Name)` unconditionally from a DB row. PASS.
- `character/rest.go:128-157` `Extract` — sets `SetAccountId(m.AccountId)` and `SetName(m.Name)` unconditionally from the inbound REST payload. PASS.
- `kafka/consumer/character/consumer.go:371-391` `handleCreateCharacter` — sets `SetAccountId(c.Body.AccountId)` and `SetName(c.Body.Name)` unconditionally from the CREATE_CHARACTER command. PASS.
- `character/processor.go:282` `SkillModelDecorator` — `CloneModel(m).SetSkills(ms).Build()`; `m` is already a valid `Model` fetched via `modelFromEntity`, so `CloneModel` seeds `accountId`/`name` from it and only adds skills. PASS (by construction — clone never loses fields, only decorator sets change).

Test sites: 20 files touched, all use one of two shapes — either set
`accountId`+`name` together (the majority), or set neither (the two
documented exceptions: `hp_mp_gain_test.go:buildCharacter`, which sets only
`jobId`+`skills`, and `model_test.go`'s pre-existing `TestBuildPreservesHpMpUsed`/`TestCloneBuildRoundTripPreservesHpMpUsed`,
which set only `name`+`hpMpUsed` — `accountId` stays zero, satisfying the
implication vacuously). I read every `_test.go` diff hunk in the commit (all
21 files) and found no exception — every hunk is either (a) a mechanical
`m, err := ....Build()` / `if err != nil { t.Fatalf(...) }` adaptation, or
(b) the four new/edited `model_test.go` cases pinning the invariant directly.
No hidden fifth construction shape.

The invariant holds at every site, production and test, as claimed.

### Is it the strongest invariant that survives?

Checked the two obvious stronger candidates by hand against the enumerated
sites:

- **Unconditional `name != ""`** (Builder's shape minus the accountId leg):
  rejected by `hp_mp_gain_test.go:buildCharacter` (name empty) and by
  `TestBuildPreservesHpMpUsed`'s companion (accountId empty, but that one
  sets name — so this candidate alone doesn't fail there; it fails on
  `hp_mp_gain_test.go` and on `NewEmptyBuilder().Build()` used implicitly
  wherever only some fields are set without a name, e.g.
  `TestEligibilityGate5NameTaken`'s earlier partial states within the test
  file, and generally any `SetSkills`-only clone path). Fails.
- **Unconditional `id != 0`**: `character/rest.go:Extract`'s create path
  builds a Model with `id` still zero (id is DB-assigned on `Create`, not
  present on the inbound `RestModel` for a new character) — `SetId(m.Id)`
  runs but `m.Id` is the zero value for an unpersisted REST create payload.
  This candidate would reject the create path itself. Fails immediately at
  a production site — correctly not chosen.
- **`accountId != 0` alone** (drop the name leg): report and brief both
  already ruled this out — nothing in the sweep needs a bare accountId
  check without name coupled to it, and it would incorrectly *accept* a
  model with a real account and an empty name, which is exactly the defect
  the conditional form exists to catch. Weaker in a different, wrong
  direction (under-rejects), not stronger.

The conditional-implication shape is not a hedge; it is the tightest
invariant that both (a) rejects the one incoherent state the sweep can
identify (an account-owning model with no name) and (b) accepts every
observed legitimate partial. I could not find a stronger surviving
candidate, and the report's derivation reasoning matches what I found
independently.

## 2. Were any tests edited to conform to the invariant?

No. Diffed every `_test.go` file in the commit (`administrator_test.go`,
`hp_mp_gain_test.go`, `kafka_integration_test.go`,
`login_logout_channel_override_test.go`, `meso_outbox_test.go`,
`model_test.go`, `name_validity_resource_test.go`,
`patch_integration_test.go`, `processor_test.go`, `producer_test.go`,
`rest_test.go`, `pending_change_applier_test.go`,
`processor_eligibility_test.go`, `refund_idempotency_test.go`,
`task_test.go`). Every hunk outside `model_test.go` is one of:
- `x := ...Build()` → `x, err := ...Build()` + `if err != nil { t.Fatalf(...) }` (signature adaptation, no fixture change), or
- a helper gaining a `*testing.T` parameter (`hp_mp_gain_test.go:buildCharacter`, `pending_change/processor_eligibility_test.go:buildCharacter`) purely to plumb the new error through `t.Fatalf`, with every call site updated to pass `t` — no assertion or fixture value changed.

No fixture values (`SetName`, `SetAccountId`, field values) changed anywhere
in the diff. `model_test.go`'s new tests are additive, not edits to existing
assertions. This matches both the brief's requirement and the report's claim.

## 3. Is the invariant actually enforced and tested?

Ran the new tests directly:
```
go test -run 'TestBuild' -v ./character/...
--- PASS: TestBuilder_Build_MissingAccountId
--- PASS: TestBuilder_Build_MissingName
--- PASS: TestBuilder_Build_WithIdentity
--- PASS: TestBuildPreservesHpMpUsed
--- PASS: TestBuildErrorsWhenAccountIdSetWithoutName
--- PASS: TestBuildSucceedsWithAccountIdAndName
--- PASS: TestBuildSucceedsWithNeitherAccountIdNorName
PASS
```
`TestBuildErrorsWhenAccountIdSetWithoutName` (`model_test.go`) asserts
`err == nil` is a failure — i.e., it requires `Build()` to return a non-nil
error for `SetAccountId(1000)` with no name. Deleting the `if c.accountId !=
0 && c.name == "" { ... }` guard in `builder.go` would make this the only
code path (falls straight to the unconditional `return Model{...}, nil`),
so the test would then observe `err == nil` and fail via `t.Fatalf`. The
test is a genuine pin, not decoration — confirmed by reading the guard and
the assertion together (not by mutating the tree, which is out of scope for
this review).

`go build ./...` and `go vet ./...` in the module: clean, no output.
`go test ./character/...`: `ok` (cached from the implementer's own run;
`-run 'TestBuild'` re-run above forces execution and confirms freshness).

## 4. Error-handling convention at every updated call site

- `character/provider.go:60` `modelFromEntity` — already `(Model, error)`; now returns the builder's real error instead of a hardcoded `nil` (`builder.go` diff: `return r, nil` → tail-return `Build()` directly). Correct propagation, no swallow.
- `character/rest.go:157` `Extract` — same shape; `Build(), nil` → tail-return `Build()`. Correct propagation.
- `character/processor.go:282-289` `SkillModelDecorator` — `model.Decorator[Model]` has no error return; binds `err`, logs via `p.l.WithError(err).Errorf(...)` (in-scope `logrus.FieldLogger` — `ProcessorImpl.l`, confirmed field at `processor.go:162`), falls back to `m`, with a code comment explaining why the error is unreachable at this call site. Matches the branch convention and the brief's explicit ruling against `model.ErrDecorator`/`degrade.Observe` here.
- `kafka/consumer/character/consumer.go:391-399` `handleCreateCharacter` — `message.Handler` has no error return; binds `err`, logs via the in-scope `l logrus.FieldLogger` handler parameter with full saga-correlation fields (`transaction_id`, `account_id`, `world_id`, `name`), then `return`s without calling `CreateAndEmit`. No `_`, no swallow.
- No `degrade.Observe` or `model.ErrDecorator` introduced anywhere in the diff (`grep` confirms neither string appears in the commit's changed files).
- Every test call site binds `err` and calls `t.Fatalf` on it — checked each of the ~20 files above; none discard with `_`.

## Not evaluable

None. The full review surface (the commit's diff plus the four production
call sites whose contracts the invariant depends on) was read and verified
directly; nothing required assuming an unread file's behavior.

## Non-blocking notes

- `SkillModelDecorator`'s "unreachable" comment is provably correct given
  `CloneModel` only copies existing fields and adds skills (verified by
  reading `CloneModel` in `builder.go`, not just trusting the comment) — no
  action needed, noted only because the brief called out unreachability
  comments as a convention to watch.
- The invariant's documentation in `builder-validation.md` includes a
  "strongest invariant that survives" derivation section that matches what
  I independently re-derived (including candidates it does not mention
  explicitly, like `id != 0`, which I checked separately above) — the
  written record is accurate, not just plausible-sounding.

---

verdict: APPROVED
artifact: docs/tasks/task-272-character-spawn-point-plumbing/review-fix-round-2.md
scope_confirmed: commit 353d3bd10, all 21 changed files (services/atlas-character/atlas.com/character + builder-validation.md), matches the fix-round-2 brief's fence exactly
blocking: 0
non_blocking: 0
not_evaluable: 0
