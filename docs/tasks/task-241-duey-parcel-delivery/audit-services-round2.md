# Backend Audit — task-241-duey-parcel-delivery (existing services, ROUND 2)

- **Service Paths:** `services/atlas-channel`, `services/atlas-character`, `services/atlas-configurations`, `services/atlas-npc-conversations`, `services/atlas-saga-orchestrator`; `libs/atlas-constants`, `libs/atlas-packet`, `libs/atlas-saga`
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-20
- **Build:** PASS
- **Tests:** all packages touched by the diff PASS
- **Overall:** PASS

This is a re-audit against the baseline at `docs/tasks/task-241-duey-parcel-delivery/audit-services.md`
(round 1), which raised 3 blocking findings. It re-verifies the four fix
commits (`491bd5172`, `a95b7c067`, `8f23610d8`, `c8579718f`) and answers three
specific rulings the controller asked for. Round-1 artifacts are left
untouched as the baseline; this file does not restate round-1's PASS
evidence for packages the fix commits did not touch.

## Build & Test Results

```
libs/atlas-saga:                                      go build ./... -> clean
                                                        go test ./... -count=1 -> ok

services/atlas-channel/atlas.com/channel:              go build ./... -> clean
                                                        go test ./parcel/... -count=1 -> ok

services/atlas-character/atlas.com/character:          go build ./... -> clean

services/atlas-configurations/atlas.com/configurations: go build ./... -> clean
                                                        go test ./... -count=1 -> ok (all packages)

services/atlas-npc-conversations/atlas.com/npc:        go build ./... -> clean
                                                        go test ./conversation/... ./saga/... -count=1 -> ok

services/atlas-saga-orchestrator/atlas.com/saga-orchestrator: go build ./... -> clean
                                                        go test ./... -count=1 -> ok (all packages)

tools/goroutine-guard.sh (repo-wide)                   -> exit 0
```

## Applicability

Unchanged from round 1's applicability table (`audit-services.md`), narrowed
to the four fix commits' touched families:

| Family | Fired? | Trigger observation |
|---|---|---|
| DOM structure (DOM-01) | Yes | `atlas-channel/parcel` gained `builder.go` in `a95b7c067` |
| FILE placement (FILE-01,02,05,06) | Yes | `a95b7c067` re-splits `atlas-channel/parcel` |
| Testing (DOM-20,33) | Yes | `event_acceptance_test.go`'s `allActions` list, new `show_parcel_roundtrip_test.go` |
| Constants reuse (DOM-21) | Yes | `8f23610d8` replaces `byte` with `inventory.Type` on two payload fields |
| Security (SEC-*) | Yes (narrow) | `c8579718f` gates `DueyActionHandle` with a validator — who may send `DUEY_ACTION` |
| Scaffolding/seed corpus | Yes | `c8579718f` template/corpus-count changes |

## Checklist Results

### saga-orchestrator/saga — `Step[T].UnmarshalJSON` completeness (round-1 blocking item 1)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Cross-service seam / functional correctness | `Step[T].UnmarshalJSON` covers every declared `Action` | PASS | `saga/model.go:1584-1589` adds `case ShowParcel:`, unmarshaling into `ShowParcelPayload`. `grep -n "^\s*case " saga/model.go` shows all 122 `Action` constants declared in `libs/atlas-saga/model.go` (confirmed by diffing the declared-constant set against the switch's case list) now have a matching case, including the pre-existing `DestroyAllAssets` (`model.go:1122`) and `RenamePet` (`model.go:1518`) which were already covered before this branch. |
| Round-trip test exists | New regression test asserts the fix | PASS | `saga/show_parcel_roundtrip_test.go:20-38` — `TestStep_ShowParcel_JSONRoundTrip` marshals a `Step[ShowParcelPayload]` and asserts `decoded.UnmarshalJSON` round-trips `Action()`/`Payload()` intact. `go test ./saga/...` passes. |
| **`allActions` coverage-list genuine completeness (controller's item 1)** | Every `Action` constant declared in `libs/atlas-saga/model.go` appears in `event_acceptance_test.go`'s `allActions` | **FAIL (pre-existing, not introduced by this branch)** | Diffing `grep -oP '^\s*\K[A-Za-z]+(?=\s+Action = )' libs/atlas-saga/model.go` (122 declared constants) against `event_acceptance_test.go:15-60`'s `allActions` slice (120 entries) shows **`DestroyAllAssets` and `RenamePet` are still absent** from `allActions` — `event_acceptance_test.go:14-61`. Both actions predate this branch (`git log --oneline -- libs/atlas-saga/model.go` shows `DestroyAllAssets` from task-102, `RenamePet` from task-224) and both already have `acceptanceTable` entries (`event_acceptance.go:151,184`) and handler dispatch (`handler.go:839-840,895-896`) — so no runtime bug results today. But the claim that the coverage list "is now genuinely complete against the action set" is FALSE: the fix only appended the five parcel-custody actions it needed and did not audit the rest of the list, so `TestAcceptanceTable_EveryActionRepresented` (`event_acceptance_test.go:66-72`) still cannot catch a future `acceptanceTable` gap for either of these two actions, and the same coverage-list mechanism that let `ShowParcel` slip through undetected for this long still has two known holes. |

### atlas-channel/parcel — package split (round-1 blocking items 2/3)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` exists (package has `model.go`) | PASS | `services/atlas-channel/atlas.com/channel/parcel/builder.go:1-165` — standard `Builder`/`validate()`/`Build() (Model, error)` shape. |
| FILE-01 | Processor in `processor.go` | PASS | `parcel/processor.go:16-77` — only `Processor`, `NewProcessor`, and its methods remain. |
| FILE-02 | RestModel/Extract/JSON:API methods in `rest.go` | PASS | `parcel/rest.go:16-108` — `RestModel` struct, `GetName`/`GetID`/`SetID`/`SetToOneReferenceID`/`SetToManyReferenceIDs`, and `Extract(` all moved here. |
| FILE-05 | Domain `Model` / `Builder` placement | PASS | `parcel/model.go:17-59` — `Model` struct and its accessor methods; `ToPacket`/`WireId` alongside it (`model.go:63-97`); `Builder` in `builder.go`. |
| FILE-06 | No single file carrying ≥2 responsibilities | PASS | Processor (`processor.go`), RestModel+Extract (`rest.go`), domain Model+Builder (`model.go`+`builder.go`), requests (`requests.go`) are now cleanly separated — the collapsed-file shape round 1 flagged (task-102 `wallet.go` pattern) is gone. |
| EXT-01 | Ref-ID stubs on target RestModels | PASS (unchanged) | `parcel/rest.go:74,76` (`RestModel`); `parcel/requests.go:74,76` (`discardRestModel`); `:105,107` (`notifyRestModel`) — carried forward intact through the split. |
| EXT-02 | httptest-backed integration test | FAIL (unchanged from round 1, not in this fix's scope) | `grep -rln httptest services/atlas-channel/atlas.com/channel/parcel/` → no matches. Non-blocking per round 1; not addressed by any of the four fix commits, so it remains open. |

### atlas-saga-orchestrator/parcel — EXT-02

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| EXT-02 | httptest-backed integration test | FAIL (unchanged from round 1, not in this fix's scope) | `grep -rln httptest services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/` → no matches. Carried forward, non-blocking. |

### atlas-configurations — `DueyActionHandle` validator (controller's item 2)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Security-relevant handler gating | `DueyActionHandle` carries a validator, and the chosen validator is the narrowest available | PASS | All 8 templates carrying the feature (`template_gms_72_1.json` … `template_jms_185_1.json`) bind `DueyActionHandle` to `"validator": "LoggedInValidator"` (verified per-file via `grep -B3 '"handler": "DueyActionHandle"'` on each). `grep -rhoP '"validator"\s*:\s*"\K[^"]+' seed-data/templates/` confirms the corpus's validator vocabulary is exactly `{LoggedInValidator, NoOpValidator}` — the implementer's "only two validators exist" claim holds. `LoggedInValidator` is not a no-op label: `services/atlas-channel/atlas.com/channel/socket/handler/handle.go:33-44` implements it as a real gate — `account.NewProcessor(l, ctx).IsLoggedIn(s.AccountId())`, and on failure it calls `session.NewProcessor(l, ctx).Destroy(s)`, terminating the session — registered at `main.go:1033`. The five cited neighbouring handlers (`NPCShopHandle`, `OwlActionHandle`, `CharacterInventoryMoveHandle` confirmed directly in `template_gms_92_1.json`; `NPCContinueConversationHandle`, `HiredMerchantOperationHandle` confirmed in `template_gms_84_1.json`, the templates that carry them) all use `LoggedInValidator` too — the reasoning is not fabricated. |
| Validation gate enforces non-empty, not membership | `socket.Validate` only requires the field be non-empty | PASS (informational, not a finding) | `services/atlas-configurations/atlas.com/configurations/socket/validate.go:122-127` — `validateCollection` checks `strings.TrimSpace(b.Validator) == ""`, with no check against a known-validator set. This means a typo'd validator name would pass this gate and later fail at runtime lookup in `atlas-channel`'s `validatorMap` rather than at seed time — a pre-existing gap in the validation rule itself, not introduced by `c8579718f`, and outside this branch's fix scope (no `ID` in the checklist covers validator-name membership). |
| Corpus count | `corpus_test.go`'s literal matches the actual template corpus size | PASS | `services/atlas-configurations/atlas.com/configurations/socket/corpus_test.go:63` asserts `total != 3333`; `go test ./socket/...` passes against the actual seeded corpus. |

### `libs/atlas-saga` payload type change (controller's item 3)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-21 | No redeclaration of shared domain constant; wire format preserved across the type change | PASS | `libs/atlas-saga/payloads.go` — `TransferToParcelPayload.SourceInventoryType` and `WithdrawFromParcelPayload.InventoryType` are `inventory.Type` (`libs/atlas-constants/inventory/constants.go:9` — `type Type int8`). `grep -rn "func.*Type.*MarshalJSON\|func.*Type.*UnmarshalJSON" libs/atlas-constants/inventory/` → no matches — no custom JSON codec exists, so `encoding/json` encodes `inventory.Type` as a plain signed-integer literal, identical in output to `byte` for the actual value domain. The value domain is closed and small: `TypeValueEquip..TypeValueCash = 1..5` (`inventory/constants.go:12-16`), well within both `byte`'s (0-255) and `int8`'s (-128..127) positive range, so the JSON encoding is byte-for-byte identical for every value this field can hold. |
| Cross-service boundary conversions are explicit and minimal | Every call site the compiler flagged converts correctly, no silent truncation | PASS | `git show 8f23610d8` — three targeted conversions: `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/processor.go` (`byte(payload.SourceInventoryType)` into the still-`byte`-typed `ReleaseFromCharacterPayload.InventoryType` and the `compartment.RequestCompartment` call), `duey_action_receive.go:202` (`InventoryType: inventory.Type(p.ItemType())`), `duey_action_send.go:176-190,205` (`sourceInventoryType` retyped from `byte` to `inventory.Type`, dropping a now-redundant `byte(it)` cast). All three are minimal, direct casts with no field reordering or lossy narrowing. |
| Module builds/tests | `libs/atlas-saga`, `atlas-channel`, `atlas-saga-orchestrator` all green after the type change | PASS | `go build ./...` clean and `go test ./... -count=1` passing in all three modules (see Build & Test Results above). |

## Security Review

`c8579718f` is the only change in this round's scope that is security-relevant
(it gates who may send `DUEY_ACTION`). Verified above: `DueyActionHandle` now
requires `LoggedInValidator`, a real runtime gate
(`services/atlas-channel/atlas.com/channel/socket/handler/handle.go:33-44`)
rather than a config-only label, closing the pre-login gap `NoOpValidator`
would have left open. No other SEC-* trigger fired in this round's diff.

## Not evaluable from the diff

- Whether `event_acceptance_test.go`'s `allActions` gap for `DestroyAllAssets`/`RenamePet` (see finding above) also exists as a live `acceptanceTable`/handler-dispatch gap elsewhere in the file beyond the two entries checked — the full 1800+ line `handler.go`/`event_acceptance.go` pair was not swept end-to-end for every one of the 122 actions; only the two flagged gaps in the coverage *list* were confirmed to already have real entries.
- Whether any other seed-corpus handler besides the five cited neighbours (`NPCShopHandle`, `OwlActionHandle`, `CharacterInventoryMoveHandle`, `NPCContinueConversationHandle`, `HiredMerchantOperationHandle`) uses `NoOpValidator` for a comparably NPC-gated action, which would weaken the "LoggedInValidator is the narrowest available and the established convention" claim — the full 8-template x N-handler cross product was not swept, only the specific handlers the fix commit's message named.

## Summary

### Blocking (must fix)

None. All three round-1 blocking findings are resolved:
- `ShowParcel` unmarshal gap — fixed at `saga/model.go:1584-1589`, with a dedicated round-trip test.
- `atlas-channel/parcel` collapsed-file violation (FILE-02/05/06) — resolved by the rest.go/model.go/builder.go split.
- `atlas-channel/parcel` missing `builder.go` (DOM-01) — resolved.

### Non-Blocking (should fix)

- **Coverage-list gap (new finding this round)** — `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/event_acceptance_test.go:14-61`'s `allActions` is still missing `DestroyAllAssets` and `RenamePet`. Both already have correct `acceptanceTable`/handler coverage today, so this is not a live bug, but the completeness test cannot detect a *future* regression in either action's registration. The claim that this list "is now genuinely complete against the action set" does not hold — it was only extended by exactly the entries this branch's own work needed.
- **EXT-02** (carried forward, unchanged) — `services/atlas-channel/atlas.com/channel/parcel/` and `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/parcel/` still have no `httptest`-backed integration test.
- **Validator-name membership gap** (informational, pre-existing, not introduced by `c8579718f`) — `socket.Validate` (`services/atlas-configurations/atlas.com/configurations/socket/validate.go:122-127`) only checks a validator field is non-empty, not that it names a validator `atlas-channel` actually registers; a seed-time typo would pass this gate and only fail at runtime dispatch.
