# Backend Audit — task-227 world-transfer fixes, round 2 (re-audit of 88b53f027)

- **Service Path:** services/atlas-character, services/atlas-channel, services/atlas-configurations
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-17
- **Scope:** commit `88b53f027` (fix round 1 for the 5 blocking findings recorded in
  `docs/tasks/task-227-cash-name-change-world-transfer/audit-world-transfer-fixes.md`),
  plus a re-check of behavior preservation across the touched files.
- **Build:** PASS (3 modules touched by this commit)
- **Tests:** all passed, 0 failed
- **Overall:** PASS

## Build & Test Results

```
services/atlas-channel/atlas.com/channel:               go build ./... -> exit 0
services/atlas-channel/atlas.com/channel:                go test ./... -count=1 -> ok, all packages (pendingchange 0.158s, socket/handler 1.336s)
services/atlas-character/atlas.com/character:            go build ./... -> exit 0
services/atlas-character/atlas.com/character:             go test ./... -count=1 -> ok, all packages (pending_change 178.339s)
services/atlas-configurations/atlas.com/configurations:  go build ./... -> exit 0
services/atlas-configurations/atlas.com/configurations:  go test ./... -count=1 -> ok, all packages (templates 0.439s)
tools/lint.sh --check <3 modules>                         -> "0 issues." x3; only failure is the pre-existing,
                                                               unrelated ui:node-version gate (node v24 vs required v22)
```

## Part 1 — Verification of the 5 prior blocking findings

All evidence below is drawn from `git diff 57b109008..88b53f027` (the fix
commit's actual diff against its immediate parent), not from the fix report's
prose.

| Prior finding | Status | Evidence |
|---|---|---|
| EXT-02: `pendingchange.CheckTransferEligibilityIndependent` had zero test coverage | CLOSED | `services/atlas-channel/atlas.com/channel/pendingchange/processor_test.go:250-349` — new `TestCheckTransferEligibilityIndependent`, table-driven (`tests := []struct{...}`, `t.Run`), spins a real `httptest.NewServer`, wires it via the package's pre-existing `withCharactersServiceURL(t, server.URL)` helper (`processor_test.go:16`, unchanged by this commit) and `newTestProcessor()` (`processor_test.go:21`, unchanged), and calls `p.CheckTransferEligibilityIndependent(1)` — the real `Processor` method, not a stub. Three cases: 200 eligible (asserts `eligible=true`), 200 ineligible with `reason: "world_full"` (asserts the unmarshalled reason string), and a 500 with an unparseable body (`rawBody: "not json at all {{{"`, asserts `err != nil`). The test also asserts the request path equals `/characters/1/transfer-eligibility-independent` (`processor_test.go:293-295`), which is the real HTTP call, not a bypassed unmarshal. This is genuine `httptest`-backed coverage of the client method, driven through the real `Transform`/`TransformEligibility` path (see next row) rather than a stub that asserts nothing. |
| EXT-01: `pendingchange.EligibilityRestModel` missing `SetToOneReferenceID`/`SetToManyReferenceIDs` | CLOSED | `services/atlas-channel/atlas.com/channel/pendingchange/rest.go:117-123` — both methods added as no-op stubs (`func (r *EligibilityRestModel) SetToOneReferenceID(_, _ string) error { return nil }` and the ManyReferenceIDs equivalent), satisfying the api2go `MarshalReferences`-adjacent interface the guideline requires. |
| DOM-04/DOM-05: `pendingchange/rest.go` had no `func Transform(`/`func TransformSlice(` | CLOSED | `rest.go:125-129` — `func Transform(r RestModel) (RestModel, error)` now lives in `rest.go`; `rest.go:133-144` — `func TransformSlice(rs []RestModel) ([]RestModel, error)` added, iterating and applying `Transform` per element; `rest.go:146-149` — `TransformEligibility` (the `EligibilityRestModel` counterpart). `processor.go`'s old `identityRestModel`/`identityEligibilityRestModel` (previously in `processor.go`, wrong file per DOM-02/FILE-02) are deleted — `grep -rn identityRestModel\|identityEligibilityRestModel services/atlas-channel/.../pendingchange/` returns nothing — and `processor.go:57,68` now call `Transform`/`TransformEligibility` directly. `TransformSlice` is unused by in-package callers (`GetByCharacterId` still applies `Transform` per-element via `requests.SliceProvider`, per the pattern in the same file) but is an exported package-level func, so it does not trip Go's unused-symbol compile/lint checks; `tools/lint.sh` confirms 0 issues. |
| EXT-01: `atlas-character/pending_change/rest.go` `familyMemberRestModel` missing the same two methods | CLOSED | `services/atlas-character/atlas.com/character/pending_change/rest.go:248-249` — both no-op stubs added, in the identical style already used by three sibling models in the same file (`worldRestModel:181-182`, `merchantShopRestModel:277-278`, `partyRestModel:327-328`) — confirming this is a genuine pre-existing convention, not an invented pattern. |
| DOM-20: `world_transfer_alias_test.go` — three multi-case tests not table-driven | CLOSED | `services/atlas-configurations/atlas.com/configurations/templates/world_transfer_alias_test.go` — `TestWorldTransferReasonAliasesResolveToAnchorCode` (diff hunk at old L64) now builds `tests := []struct{ name string; entry CatalogEntry }` from `c.Entries()` and wraps the per-template body in `t.Run(tc.name, func(t *testing.T){...})`; `TestCannotTransferToNewWorldHasNoAlias` (old L108) does the same; `TestWorldTransferUnmappableTemplatesCarryNoAliases`-equivalent block (old L134, the fixed two-template list) becomes `tests := []struct{ name string }{{name: "template_gms_12_1.json"}, {name: "template_jms_185_1.json"}}` + `t.Run`. Every original assertion body (the alias-map walk, the `t.Errorf` messages, the `CANNOT_TRANSFER_TO_NEW_WORLD` no-alias check) is reproduced unchanged inside the new `t.Run` closures — confirmed by diffing the closure bodies against the pre-fix bodies line for line; only control flow moved. |

All five findings are closed with fresh, direct evidence from the actual diff — not the fix report's narrative.

## Part 2 — New violations introduced by the fix round

Checked every file this commit touches against the DOM-*/FILE-*/EXT-*
checklist:

| ID | Check | Status | Evidence |
|---|---|---|---|
| FILE-02 | RestModel/Transform/JSON:API methods live in `rest.go` | PASS | All new methods (`SetToOneReferenceID`, `SetToManyReferenceIDs`, `Transform`, `TransformSlice`, `TransformEligibility`) landed in `pendingchange/rest.go`, not `processor.go` — this is itself the fix for the prior FILE-adjacent DOM-04/05 gap. |
| FILE-01 | Processor logic stays in `processor.go` | PASS | `pendingchange/processor.go` diff only removes the misplaced identity-transform funcs and updates two call sites (`processor.go:57,68` per the diff) — no new business logic added there. |
| DOM-33 | Mocks updated for interface change | PASS (by absence) | `Processor` interface's method set is unchanged (`CheckTransferEligibilityIndependent`'s signature is untouched — only its internal transform-func argument changed); `grep` for `pendingchange.Processor` mocks under `services/atlas-channel/` returns nothing. |
| DOM-31 | Tenant/trace never a REST field | PASS | No `tenant.` reference added in any of the 4 production-code files touched (`processor.go`, `rest.go` in both `pendingchange` and `pending_change`). |
| EXT-04 | URL via `requests.RootUrl` | PASS (unchanged) | New test only adds an `httptest.NewServer`; production URL construction (`requestTransferEligibilityIndependent`, unchanged by this commit) still composes via `requests.RootUrl`, per round-1 audit evidence at `requests.go:87-93` (not touched in this commit). |
| DOM-20 | Table-driven tests | PASS | The new `TestCheckTransferEligibilityIndependent` (channel) and all three rewritten `templates` tests use `tests := []struct{...}` + `t.Run`. |
| FILE-06 | No catch-all file | PASS | No new file added; all changes are to existing single-purpose files. |
| Gofmt/lint | No new lint issues | PASS | `tools/lint.sh --check` on all three modules returns `0 issues.`; the sole reported failure (`ui:node-version`) is a pre-existing environment gate unrelated to any file in this diff. |

No new violations found in the fix-round diff.

## Part 3 — Behavior-preservation checks

1. **CHECK handler emits no POP_UP world message.** `git diff 57b109008..88b53f027` touches only `cash_shop_check_name_change_possible_test.go` in `socket/handler`, and only by deleting the dead `storageWarningModeByte` helper (18 lines removed, 0 added). The production handler file `cash_shop_check_transfer_world_possible.go` is **not in this commit's file list at all** (confirmed via `git diff 57b109008..88b53f027 --stat` — only 6 files changed, none of them handler `.go` production files). `storageWarningWasAnnounced()` — the assertion that no WORLD_MESSAGE write occurred on the CHECK path — is untouched and still invoked at `cash_shop_check_name_change_possible_test.go:432` and four call sites in `cash_shop_check_transfer_world_possible_test.go` (143, 160, 174, 365). Behavior unchanged.

2. **Credential validation and PIC-attempt lockout in the CHECK handler are untouched.** `cash_shop_check_transfer_world_possible.go` (the file carrying `transferWorldCredentialMatches` and the PIC-lockout logic per the round-1 audit) does not appear in this commit's changed-file list at all — confirmed by the `--stat` output above (only `processor.go`, `processor_test.go`, `rest.go` ×2 packages, one test file, one seed test file). No behavior change possible in a file this commit does not touch.

3. **`world_full`/`world_unknown` alias `PLEASE_TRY_AGAIN`; nothing aliases `CANNOT_TRANSFER_TO_NEW_WORLD`.** Confirmed structurally in Part 1's DOM-20 row: the diff moves each test's body into a `t.Run` closure verbatim — the `worldTransferAliasGroups` walk, the `CANNOT_TRANSFER_TO_NEW_WORLD` no-alias check, and every `t.Errorf` message are byte-identical to the pre-fix bodies, only wrapped in `t.Run(tc.name, func(t *testing.T) {...})`. `go test ./...` for `atlas-configurations/templates` passes (`ok atlas-configurations/templates 0.439s`), and the fix report additionally verified `-v` output shows every per-template subtest running individually — this audit did not re-run `-v` itself but the closure-body diff is sufficient to confirm no assertion was weakened or dropped.

4. **DOM-04/05 rename in `pendingchange` did not change what the transform does.** `Transform`/`TransformEligibility` are byte-identical in behavior to the deleted `identityRestModel`/`identityEligibilityRestModel` — both are `return r, nil` (confirmed at `rest.go:127-128` and `rest.go:147-148`, and by diff showing only the deletion of the old funcs from `processor.go` plus their re-addition under new names in `rest.go`, with no logic change). The two call sites in `processor.go` (`GetByCharacterId`, `CheckTransferEligibilityIndependent`) pass the renamed funcs as the same positional argument to the same `requests.SliceProvider`/`requests.Provider` calls — confirmed via the diff hunk showing only the identifier swapped (`identityRestModel` -> `Transform`, `identityEligibilityRestModel` -> `TransformEligibility`), no change to the surrounding call shape.

All four behavior-preservation checks hold.

## Not evaluable from the diff

None — this re-audit's scope (the fix commit `88b53f027` plus the specific
behavior-preservation questions the brief named) was fully answered from the
commit's own diff, the unchanged helper definitions it calls into (read
directly), and the module-local build/test/lint runs.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- None.
