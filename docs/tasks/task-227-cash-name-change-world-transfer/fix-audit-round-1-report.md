# Fix round 1 report — task-227

Fixes both red gates from the fix round: the flagless `tools/verify.sh` lint
failure, and the 5 blocking `backend-guidelines-reviewer` findings.

## What was implemented

### A. Lint fix — dead `storageWarningModeByte`

`services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_name_change_possible_test.go`

Confirmed the BUY-time tests (`cash_shop_operation_imprint_test.go`) already
have their own equivalent helper —
`worldMessageBuyRecorder.storageWarningModeByte` — covering
`TestBuyWorldTransferWarnsWhenStrandingStorage` and friends. The CHECK-handler
copy was genuinely dead (the storage-stranding warning moved to
`handleBuyWorldTransfer` per the underlying bug fix). Deleted it. Did **not**
re-introduce a CHECK-time POP_UP assertion; `storageWarningWasAnnounced()`
(asserting the absence of a WORLD_MESSAGE write on the CHECK path) is
untouched and still used by both `cash_shop_check_name_change_possible_test.go`
and `cash_shop_check_transfer_world_possible_test.go`.

### B. Backend-guidelines findings

1. **EXT-02** — `pendingchange/processor.go` `CheckTransferEligibilityIndependent`
   had zero coverage. Added `TestCheckTransferEligibilityIndependent` to
   `pendingchange/processor_test.go`, following the existing
   `TestRequestNameChange`/`TestCancelPendingChange` httptest table-driven
   pattern: 3 cases (200 eligible, 200 ineligible with reason, 500
   infrastructure error), asserting the real request path (URL,
   unmarshal-through-`EligibilityRestModel`, returned `(eligible, reason, err)`).

2. **EXT-01** — `pendingchange/rest.go` `EligibilityRestModel` gained
   `SetToOneReferenceID`/`SetToManyReferenceIDs` no-op stubs, matching the
   existing `worldRestModel`/`merchantShopRestModel`/`partyRestModel` pattern
   in `atlas-character/pending_change/rest.go`.

3. **DOM-04/DOM-05** — `pendingchange/rest.go` gained `Transform(RestModel) (RestModel, error)`
   (renamed from `processor.go`'s `identityRestModel`), `TransformEligibility`
   (renamed from `identityEligibilityRestModel`), and a new `TransformSlice`
   following the exact shape found in
   `services/atlas-ban/atlas.com/ban/report/rest.go` (the one other confirmed
   example of this convention in the monorepo). `processor.go`'s
   `GetByCharacterId`/`CheckTransferEligibilityIndependent` now call
   `Transform`/`TransformEligibility` directly.

4. **EXT-01** — `atlas-character/pending_change/rest.go`
   `familyMemberRestModel` gained the same no-op
   `SetToOneReferenceID`/`SetToManyReferenceIDs` stubs.

5. **DOM-20** — `atlas-configurations/.../templates/world_transfer_alias_test.go`:
   converted all three tests
   (`TestWorldTransferReasonAliasesResolveToAnchorCode`,
   `TestCannotTransferToNewWorldHasNoAlias`,
   `TestWorldTransferUnmappableTemplatesCarryNoAliases`) to
   `tests := []struct{...}` + `t.Run(tc.name, ...)`. The first two build their
   table dynamically from `c.Entries()` (file name -> per-case subtest); the
   third's table is the pre-existing fixed two-template list. Every original
   assertion — including the `world_full`/`world_unknown` ->
   `PLEASE_TRY_AGAIN` aliasing and "nothing may alias
   `CANNOT_TRANSFER_TO_NEW_WORLD`" — is unchanged, only the structure moved.

## Testing

```
cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./...
```
All packages `ok` (including `pendingchange` and `socket/handler`), no
failures.

```
cd services/atlas-character/atlas.com/character && go build ./... && go test ./...
```
All packages `ok`, no failures (`pending_change` included).

```
cd services/atlas-configurations/atlas.com/configurations && go build ./... && go test ./...
```
All packages `ok`, including `templates` — verified `-v` output shows every
new subtest (`.../template_gms_12_1.json`, etc.) running and passing
individually.

```
tools/lint.sh --check services/atlas-channel/atlas.com/channel \
  services/atlas-character/atlas.com/character \
  services/atlas-configurations/atlas.com/configurations
```
`0 issues.` for all three modules (the reported `ui:node-version` failure is
an unrelated pre-existing environment gap — Node v24 vs required v22 — and
does not touch any file in this diff).

## Files changed

- `services/atlas-channel/atlas.com/channel/pendingchange/processor.go`
- `services/atlas-channel/atlas.com/channel/pendingchange/processor_test.go`
- `services/atlas-channel/atlas.com/channel/pendingchange/rest.go`
- `services/atlas-channel/atlas.com/channel/socket/handler/cash_shop_check_name_change_possible_test.go`
- `services/atlas-character/atlas.com/character/pending_change/rest.go`
- `services/atlas-configurations/atlas.com/configurations/templates/world_transfer_alias_test.go`

## Self-review

- Confirmed `storageWarningWasAnnounced` (the CHECK-path no-POP_UP assertion)
  is untouched and still called from both CHECK-handler test files.
- Confirmed the PIC-attempt lockout / credential validation in the CHECK
  handler was not touched at all (no diff to `cash_shop_check_transfer_world_possible.go`).
- `TransformSlice` is currently unused by in-package callers (GetByCharacterId
  goes through `requests.SliceProvider` with per-element `Transform`, matching
  the existing architecture) — left the doc comment explicit about that so a
  future reader doesn't wonder why it isn't wired in; it exists to satisfy the
  DOM-05 convention and give external code a bulk entry point, exactly as
  `atlas-ban/report/rest.go`'s `TransformSlice` does relative to its own
  `Transform`.
- Verified `git status`/`git branch --show-current`/`git rev-parse --show-toplevel`
  after commit: correct worktree, correct branch
  (`task-227-cash-name-change-world-transfer`), only the six intended files
  staged.

## Issues or concerns

None. All five blocking backend-guidelines findings and the lint failure are
addressed; behavior is unchanged (only structural/coverage fixes), and the
constraints called out in the brief (no CHECK-time POP_UP, credential/PIC
lockout untouched, `world_full`/`world_unknown` alias `PLEASE_TRY_AGAIN`,
nothing aliases `CANNOT_TRANSFER_TO_NEW_WORLD`) all still hold per the test
runs above.
