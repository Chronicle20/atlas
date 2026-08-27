# Review: Task 4 — the cash-shop secondary-credential gate

Range: `3a0ac66df..eb8563e85` (1 commit)
Files: `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_credential.go` (new, 60 lines),
`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_credential_test.go` (new, 100 lines).

## Scope

Reviewed the diff in full (both new files, 160 lines total — well under the
slice-first threshold, read whole). Cross-checked against:
- `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use.go:1104` (MajorAtLeast precedent)
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_name_change_possible.go:131` (`remoteIpAddress` helper)
- `services/atlas-channel/atlas.com/channel/account/model.go` (`PIC()`, `BirthDate()` accessors)
- `services/atlas-channel/atlas.com/channel/account/processor.go` (`GetById`, `RecordPicAttempt` signatures)
- `libs/atlas-tenant/tenant.go:93` (`MajorAtLeast` signature)

No caller of `verifySecondaryCredential` exists yet — confirmed via
`grep -rn "verifySecondaryCredential("` returning only the definition. This
matches the brief's explicit note that consumers land in later tasks and is
not a defect.

## Findings

### 1. Interfaces produced — verbatim match (PASS)

- `ErrCredentialMismatch` (`cash_shop_credential.go:16`): `var ErrCredentialMismatch = errors.New("secondary credential mismatch")` — matches the brief's block character for character, including the doc comment.
- `credentialMatches` (`cash_shop_credential.go:22`): `func credentialMatches(usesPIC bool, storedPIC string, storedBirthDate uint32, spw string, birthday uint32) bool` — matches the brief signature exactly, and the body (lines 23–33) is transcribed verbatim from the brief's Step 3 code block, including the load-bearing comment about failing open on an unset credential.
- `verifySecondaryCredential` (`cash_shop_credential.go:38`): `func verifySecondaryCredential(l logrus.FieldLogger, ctx context.Context) func(s session.Model, spw string, birthday uint32) error` — matches the brief signature exactly.

### 2. `usesPIC` version decision — PASS

`cash_shop_credential.go:46`: `usesPIC := t.IsRegion("GMS") && t.MajorAtLeast(95)`. This is the exact shape at `character_cash_item_use.go:1104`/`:1090-ish` (`nameChangeCashSlotItemType`, `worldTransferCashSlotItemType` use the identical `t.IsRegion("GMS") && t.MajorAtLeast(95)` idiom). `MajorAtLeast(v uint16) bool` is a real method on `libs/atlas-tenant/tenant.go:93` — no raw `>`/`>=` comparison anywhere in the landed file. The report's claim that `shop_operation_gift.go` uses a raw `t.MajorVersion() >= 95` (and is therefore not the right reference) is consistent with what's in the diff not repeating that pattern. Region check (`IsRegion("GMS")`) is present, matching the precedent.

### 3. Eight subtests — verbatim match to the brief's table, and the load-bearing case is real (PASS)

`cash_shop_credential_test.go:17-88` — all eight subtests transcribed with identical `name`/`usesPIC`/`storedPIC`/`storedBirthDate`/`spw`/`birthday`/`expect` values to the brief's table, row for row.

The `pre-95 ignores pic entirely` case (`:81-87`): `usesPIC=false, storedPIC="5678", storedBirthDate=19940203, spw="wrong", birthday=19940203, expect=true`. Traced by hand against `credentialMatches`: with `usesPIC=false` the function takes the birthday branch and returns `storedBirthDate == birthday` → `19940203 == 19940203` → `true`, matching. If the implementation instead (incorrectly) consulted `spw`/`storedPIC` when `usesPIC` is false, it would compare `storedPIC("5678") == spw("wrong")` → `false`, which would fail the `expect: true` assertion. So this subtest is load-bearing exactly as billed — it would catch a regression where `spw` leaks into the pre-95 birthday path.

Ran the test directly: `go test ./socket/handler/... -run TestCredentialMatches -v` → `Go test: 9 passed in 1 packages` (1 parent + 8 subtests, all named per the table).

### 4. `RecordPicAttempt` called on mismatch only, correct IP helper (PASS)

`cash_shop_credential.go:48-53`: on `credentialMatches == true`, the function logs (if inert) and returns `nil` immediately — no call to `RecordPicAttempt` on this path.
`cash_shop_credential.go:55`: `RecordPicAttempt` is only reached in the mismatch branch: `account.NewProcessor(l, ctx).RecordPicAttempt(s.AccountId(), false, remoteIpAddress(s), "")`.

`remoteIpAddress(s session.Model) string` is a real, pre-existing helper at `cash_shop_check_name_change_possible.go:131-144`, in the same `handler` package (no import needed). `RecordPicAttempt(id uint32, success bool, ipAddress string, hwid string) (int, bool, error)` at `account/processor.go:85` — the call's argument order and types match (`s.AccountId()` → `id`, `false` → `success`, `remoteIpAddress(s)` → `ipAddress`, `""` → `hwid`). The 3-value return is destructured with `_, _, rErr` and the error is logged, not swallowed into a silent success or failure — reasonable for a best-effort audit call whose own failure shouldn't change the gate's already-decided `ErrCredentialMismatch` outcome.

### 5. Account-lookup error is returned, not swallowed (PASS — pre-ruled item confirmed absent as a defect)

`cash_shop_credential.go:40-43`: `a, err := account.NewProcessor(l, ctx).GetById(s.AccountId()); if err != nil { return err }` — returned directly before any credential comparison. Matches the brief's Step 4 requirement and the task's pre-ruled boundary (the unset-credential pass is correct; this is the different case that would have been a defect, and it isn't present).

### 6. No vacuous tests (PASS)

Every subtest's `expect` field is an independent literal (`true`/`false`) set from the brief's table, compared against `credentialMatches(...)`'s live return value at `:93-97`. None of the eight subtests compute `expect` from the same inputs it feeds the function under test — genuine assertions, not tautologies.

### 7. No stubs / repo conventions (PASS)

- No `// TODO`, no stub bodies, no hard-coded success response — `credentialMatches` and `verifySecondaryCredential` are both fully implemented with real branching logic.
- No `*_testhelpers.go` file created; the test is a pure table-driven test over `credentialMatches` with no session/account construction, so the Builder-pattern requirement doesn't apply here (correctly noted in the report — nothing to build).
- `account.Model` field names (`account/model.go`) confirm `PIC()`/`BirthDate()` accessors are real, not invented.

## Build/test verification (module-local, as instructed — not `tools/verify.sh`)

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```

`go build ./...` — clean, no output.
`go test ./...` — all packages report `ok` or `[no test files]`; `atlas-channel/socket/handler` is `ok`. Full tail of output captured; no `FAIL` anywhere. Also ran `go vet ./socket/handler/...` — clean, no output.

Targeted run: `go test ./socket/handler/... -run TestCredentialMatches -v` → `Go test: 9 passed in 1 packages`.

## Not evaluable

None — the unit is self-contained (pure function + one resolver function with no caller yet), and every claim in the report was checked directly against repo source rather than taken on faith.

## Verdict

All five items in the review brief check out: the `MajorAtLeast` idiom is used correctly with the region guard, the eight subtests are a verbatim, non-vacuous match to the brief's table including the load-bearing pre-95 case, `RecordPicAttempt` fires only on mismatch via the real `remoteIpAddress` helper, and the account-lookup error path is returned rather than swallowed. No stubs, no invented identifiers, builds and tests clean.
