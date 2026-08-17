# Review: world-transfer eligibility-reasons fix round (62b0b733c..HEAD, 8 commits)

Reviewed against `bug-world-transfer-eligibility-reasons.md`, focusing on the
"Rulings (2026-08-17)" section and the five-step fix order. Scope: the diff of
`62b0b733c..HEAD` across `services/atlas-character`, `services/atlas-channel`,
`services/atlas-configurations`, `libs/atlas-packet`, and `deploy/`.

## 1. Cross-service seams — traced by hand

**(a) CHECK-time rejection produces the right packet arm end to end.**
Traced: atlas-channel `CashShopCheckTransferWorldPossibleHandleFunc`
(`services/atlas-channel/.../cash_shop_check_transfer_world_possible.go:139-157`)
→ `checkPossibleTransferEligibilityIndependentFunc` →
`pendingchange.Processor.CheckTransferEligibilityIndependent` (REST GET
`.../transfer-eligibility-independent`) → atlas-character
`handleGetTransferEligibilityIndependent` (`resource.go:107-134`) →
`ProcessorImpl.CheckTransferEligibilityIndependent`
(`processor_eligibility.go:109-116`) → `evaluateTransferEligibilityIndependent`
(gates 2, 6-11 only). A rejection returns through
`cashcb.CheckTransferWorldPossibleResultRejectedBody`, which routes
`in_family` to its own confirmed arm (StringPool 5017) and everything else to
`UNKNOWN_ERROR`. Verified end to end by
`TestTransferWorldPossibleRejectsOnIndependentGate` (in_family → byte 0x28)
and `TestTransferWorldPossibleRejectsOnIndependentGateWithoutDedicatedArm`
(banned → UNKNOWN_ERROR 0x2F) — both exercise the real handler, not just the
mapper. PASS.

**(b) `check_unavailable` flows atlas-character → `worldTransferRejectionReason`
→ resolved client code.** `runGates`
(`processor_eligibility.go:134-145`) turns any gate dependency error into the
reason string `"check_unavailable"`, uniformly across all 11 gates.
`worldTransferRejectionReason` forwards it verbatim (pinned by
`TestWorldTransferRejectionReasonPassesCheckUnavailableThrough`), and
`CashShopTransferWorldFailedBody` resolves it through the tenant's `errors`
table exactly like any other key (pinned end-to-end by
`TestCashShopTransferWorldFailedBodyResolvesCheckUnavailableAsConfigured`,
which asserts the *specific* byte 231, not merely "some byte"). At CHECK time,
`checkTransferWorldPossibleReasonArms` maps `check_unavailable` to
`UNKNOWN_ERROR` (`check_transfer_world_possible_result.go:320`, pinned by
`TestCheckTransferWorldPossibleReasonKeyRoutesCheckUnavailableToUnknownError`).
PASS.

**(c) The new destination-free entry point is actually reached from the CHECK
handler.** Confirmed by direct code read (not just grep) — see (a) above.
`checkPossibleTransferEligibilityIndependentFunc` is called inside
`CashShopCheckTransferWorldPossibleHandleFunc` at line 139, strictly after
credential validation and the PIC-attempt lockout, strictly before the
world-list lookup. PASS.

**(d) Destination-dependent gates are still evaluated at BUY time.**
`evaluateTransferEligibility` (BUY-time, `processor_eligibility.go:167-204`)
still runs all 11 gates including 1/3/4/5 (world_same, world_status,
character_slot, name_taken) in the original order; `handleBuyWorldTransfer`
still calls `pendingchange.RequestWorldTransfer` unconditionally
(`cash_shop_operation.go:319`). The gate logic in each `checkXxx` method was
moved, not rewritten — both `evaluateTransferEligibility` and
`evaluateTransferEligibilityIndependent` are thin orderings over the same
methods, so nothing was silently dropped by the split. PASS.

All four cross-service seam checks pass, each backed by a test that asserts
the NEW contract (not merely "did not crash").

## 2. Modal-collision fix (step 1) — no regression

`grep` of `cash_shop_check_transfer_world_possible.go` finds `POP_UP`/
`WorldMessageWriter`/`WorldMessagePopUpBody` nowhere outside a comment
(`grep -n "POP_UP\|WorldMessagePopUpBody\|WorldMessageWriter" ...` → one hit,
inside doc prose at line 176). `warnIfStrandingStorage` was moved wholesale
into `cash_shop_operation.go` and is called only from `handleBuyWorldTransfer`,
on the rejection-free path, after the pending-change record and purchase
request. Step 4's rewrite of the CHECK handler (commit `2b6fb05a8`) did not
reintroduce it: the new independent-gate-rejection branch and the
`eErr != nil` branch both answer via `announceTransferWorldPossible` only
(`cashcb.CheckTransferWorldPossibleResultWriter`), never
`chatpkt.WorldMessageWriter`. Regression-pinned by
`TestWorldTransferCheckNeverEmitsStorageWarning` (three sub-cases: would-be-
stranded, another-character-remains, lookup-errors — all assert no
`WORLD_MESSAGE` write) and by the new
`TestTransferWorldPossibleRejectionNeverEmitsStorageWarning`.

Credential validation and the PIC-attempt lockout
(`cash_shop_check_transfer_world_possible.go:113-135`) are structurally
untouched by the diff — the new eligibility check is inserted strictly after
both existing `checkPossibleRecordPicAttemptFunc` calls, never interleaved.
`TestTransferWorldPossibleValidatesBeforeAnswering` and
`TestTransferWorldPossibleLockoutReusesUnknownError` (pre-existing, still
present) continue to pass. PASS, no regression.

## 3. Alias seeding (step 5)

Verified by extracting every template's `errors` table
(`template_gms_{48,61,72,79,83,84,87,92,95}_1.json`):

- `world_full` and `world_unknown` alias `PLEASE_TRY_AGAIN` everywhere the
  template has that code (79/83/84/87/92/95), never
  `CANNOT_TRANSFER_TO_NEW_WORLD` — matches the SUPERSEDED correction.
- No key aliases `CANNOT_TRANSFER_TO_NEW_WORLD` anywhere (pinned by
  `TestCannotTransferToNewWorldHasNoAlias`, which sweeps every template's
  `errors` table for a code collision with that anchor, not just the seeded
  keys).
- Every seeded numeric code is that template's own value — e.g.
  `world_same`/`CANNOT_TRANSFER_TO_SAME_WORLD` = 153/177/195/209/220/229/237/
  58/58 on gms_48/61/72/79/83/84/87/92/95 respectively, matching the doc's
  version table exactly; no cross-template copy-paste (pinned by
  `TestWorldTransferReasonAliasesResolveToAnchorCode`, which asserts
  `aliasCode == anchorCode` per-template).
- `template_gms_12_1.json` and `template_jms_185_1.json` carry **zero** diff
  in this range (`git diff --stat` confirms no change to either file) and
  `TestWorldTransferUnmappableTemplatesCarryNoAliases` pins that they have no
  `CANNOT_TRANSFER_TO_SAME_WORLD`-bearing `errors` table at all.
- `transferFailureReasonConfigured` (`cash_shop_operation.go`) turns an
  unconfigured reason into an explicit `l.Warnf` fallback rather than a silent
  99, on any template — not just the two named ones. `gms_48_1`/`gms_61_1`/
  `gms_72_1` correctly have no `PLEASE_TRY_AGAIN`-anchored aliases at all
  (that code doesn't exist in their tables either, per the doc's own version
  table), so `world_full`/`world_unknown`/`check_unavailable`/`unknown_error`
  fall through to the same logged-fallback path there too — a consistent,
  intentional consequence of "resolve per-template's own table," not a gap.

Ran `go test ./atlas.com/configurations/templates/ -run
"TestCannotTransferToNewWorldHasNoAlias|TestWorldTransfer"` — all 3 tests
PASS. All checks in this section PASS.

## 4. `inFamily` (step 2)

`services/atlas-character/atlas.com/character/pending_change/requests.go:113-132`:
non-404 errors now propagate as errors (not `true`), and the predicate is
`len(ms) > 1` rather than "call succeeded." Cross-checked against
atlas-families' actual contract
(`services/atlas-families/atlas.com/family/family/provider.go:52-100`,
`resource.go:166-186`): `GetFamilyTree` 404s (`ErrMemberNotFound`) only when
the character has no member row at all; a member with no senior/juniors
returns a tree containing only itself — exactly the case `len(ms) > 1`
correctly excludes. `TestInFamily` (`requests_test.go:107-171`) drives all
four cases (404, self-only 200, self+relative 200, 500) against a real
`httptest` JSON:API server and passed:
`ok atlas-character/pending_change 30.096s` (targeted run) and the full
`atlas-character` module `go test ./...` (backgrounded run) also completed
exit 0 with no FAIL lines. PASS.

## 5. Tests assert the NEW contract

Checked for pinned-old-behavior tests that only changed expected values
without asserting the new invariant:

- `check_transfer_world_possible_result_test.go`'s exhaustiveness test
  (`TestCheckTransferWorldPossibleResultReasonMapping`) was updated to add
  `"check_unavailable"` to `required` — this is exactly the failure mode the
  bug doc predicted ("that failure is the test doing its job"), confirmed
  correctly closed.
- `TestEligibilityGateErrorsReportCheckUnavailable` injects a dependency error
  into all 8 `gateDeps`-backed gates (3,4,6,7,8,9,10,11) and asserts
  `reason == "check_unavailable"`, not the old affirmative reason. The two
  local-DB gates (4's existing-character count, 5's name check) get their own
  dedicated error-injection tests via a closed DB handle. Gates 1 (world_same)
  and 2 (is_gm) have no dependency and correctly have no error-injection case.
  All 11 gates are accounted for.
- `TestInFamily`'s "200 with self-only tree" and "500" cases are new
  assertions the pre-fix code would have failed (old code returned
  `true`/`nil` for both).
- `TestWorldTransferCheckNeverEmitsStorageWarning` and
  `TestBuyWorldTransferWarnsWhenStrandingStorage` /
  `TestBuyWorldTransferNoWarningWhenAnotherCharacterRemains` /
  `TestBuyWorldTransferStorageWarningLookupFailsOpen` together pin both halves
  of the timing move (CHECK never emits; BUY emits under the same conditions
  the old CHECK-time logic used, including the fail-open contract).
- `TestTransferWorldPossibleRejectsOnIndependentGate*` and
  `TestCheckTransferWorldPossibleReasonKeyRoutesInFamilyToItsOwnArm` assert
  the specific wire byte for `in_family` (0x28 / IN_FAMILY), not merely
  "rejected."

No neutered or old-behavior-pinning test found in the reviewed diff.

## Build/test verification actually run in this review

- `services/atlas-configurations`: `go test ./atlas.com/configurations/templates/`
  (world-transfer alias tests) — PASS.
- `services/atlas-channel/atlas.com/channel`: `go build ./...` and
  `go test ./...` — all packages PASS, including `socket/handler` and
  `pendingchange`.
- `libs/atlas-packet`: `go build ./...` and `go test ./...` — PASS.
- `services/atlas-character/atlas.com/character`: `go build ./...`,
  `go vet ./...` (clean), targeted `go test ./pending_change/...` for the
  new/changed tests (all PASS), and the full-module `go test ./...`
  (backgrounded, completed with exit code 0, no FAIL lines).
- `tools/service-registration-guard.sh` — reports "clean" (atlas-families is
  no longer in the `ALLOW_NO_DEPLOYMENT` exception list and now has a real
  base manifest, kustomization entry, and ingress route).
- `deploy/shared/routes.conf` vs `deploy/k8s/base/routes.conf.template.generated`
  diverge only in the expected FQDN-templating way (`gen-routes.sh`'s job),
  confirming the generated file was actually regenerated, not hand-edited out
  of sync.

I did not run the flagless `tools/verify.sh` (out of scope for this review —
review is a different gate per CLAUDE.md, and the per-module builds/tests
above cover everything the diff touches).

## Non-blocking findings

1. **Four of the five per-step fix reports referenced in the task brief are
   untracked, not committed.** `git status --short` shows
   `fix-storage-warning-timing-report.md`, `fix-families-gate-report.md`,
   `fix-reason-code-seeding-report.md`, and `fix-check-unavailable-report.md`
   as `??` (untracked); only `fix-check-time-rejection-report.md` (step 4) is
   actually part of the commit range. `git log --all -- 
   .../fix-check-unavailable-report.md` returns nothing — the file has never
   been committed anywhere. This doesn't affect code correctness (the
   reviewed diff itself is complete and correct), but it means a future
   reader following the task's own "per-step reports are in the same folder"
   pointer will find four of the five reports missing from history, and they
   are not part of what a PR built from this branch would actually ship.
   Recommend committing them (or explicitly deciding they're working notes
   and deleting them) before the branch is called done.
2. There are also several other untracked docs (`agentic-cost-audit.md`,
   `audit-frontend.md`, `audit.md`) in the same task folder from unrelated
   review/audit tooling — not part of this review's scope, noted only because
   they showed up in the same `git status` and should not be swept in
   silently by a later `git add -A`.

## Not evaluable

- The three client-text rules the doc flags as real-but-ungated (StringPool
  4003 "currently married," 4010 "transferred within 30 days," 4015
  "transfer already requested") are explicitly out of scope per the doc
  itself ("FR/design coverage question, not a bug in this fix") — not
  evaluated, correctly not touched by this diff.
- I did not independently re-derive the StringPool text-recovery methodology
  (XOR keystream, seed rotation) in "Rendered StringPool text (verified
  2026-08-17)" — that predates this commit range (it's `design.md`/bug-doc
  content, not code) and re-verifying a binary-derivation claim is outside
  what a code diff review can check. Treated as given per the task
  instruction not to re-litigate the rulings.
- Live-client rendering of the new arms/aliases was not re-verified against a
  running client in this review (no live environment available) — the
  reviewed evidence is the wire-byte assertions in tests, which is the
  correct proxy given the task's own verification notes.

## Verdict

All five requirement areas (cross-service seams, modal-collision-fix
non-regression, alias-table correction, `inFamily` honesty, and test-contract
honesty) check out against the bug doc's rulings, with tests that assert the
new behavior rather than merely not crashing. The only findings are
process/documentation gaps (uncommitted step reports), not code defects.
