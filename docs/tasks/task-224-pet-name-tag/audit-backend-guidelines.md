# Backend Guidelines Audit — task-224 (Pet Name Tag)

- **Scope:** Go changes on `task-224-pet-name-tag`, merge base `723519dc4` → head `2c9942bba`
- **Guidelines Source:** `.claude/skills/backend-dev-guidelines/resources/*`
- **Date:** 2026-08-13
- **Modules touched:** `libs/atlas-constants/pet` (new), `libs/atlas-packet`, `libs/atlas-saga`, `services/atlas-pets`, `services/atlas-saga-orchestrator`, `services/atlas-channel`
- **Build:** PASS (all 6 modules, `go build ./...`)
- **Tests:** PASS — `go test ./... -count=1` clean in all 6 modules (0 FAIL, 0 panic). **Caveat:** `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/saga/pet_name_tag_compensation_test.go` is gated behind `//go:build test` and is **not** exercised by the plain `go test ./...` invocation used here or by `tools/test-all-go.sh` — see Finding F3.
- **`go vet ./...`:** clean in all 6 modules.
- **`tools/goroutine-guard.sh`, `tools/redis-key-guard.sh`, `tools/template-opcode-order-guard.sh`, `tools/template-duplicate-binding-guard.sh`:** all exit 0.
- **Overall:** NEEDS-WORK (one Important finding, rest Minor/informational)

---

## Phase 0/2 — Domain discovery

None of the six touched packages has a `model.go` in the sense that gates the full DOM-* checklist as a "domain package" *for the diff's new surface* — the one genuine domain package touched is `services/atlas-pets/atlas.com/pets/pet` (has `model.go`, pre-existing, extended by this branch). Everything else is either:
- a **support/shared library** (`libs/atlas-constants/pet`, `libs/atlas-packet/*`, `libs/atlas-saga`), or
- an existing **saga/handler orchestration package** (`services/atlas-saga-orchestrator/.../saga`, `services/atlas-saga-orchestrator/.../pet`, `services/atlas-channel/.../socket/handler`, `.../kafka/consumer/*`) that predates this task and is extended by it, following its own established interior conventions (Processor Interface+Impl, mock sync, curried handler funcs).

The File Responsibilities Checklist was run against every new/changed file regardless of classification, per audit instructions.

---

## `services/atlas-pets/atlas.com/pets/pet` (domain package, DOM-* checklist)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` exists, fluent setters, `Build()` validation | PASS | `pet/builder.go:60-65` — new `SetName` setter added; `Build()` (`pet/builder.go:107-133`) still enforces `name != ""` etc. Length bound (4–12) is deliberately enforced one layer up by `petconst.ValidateName`, not duplicated in `Build()`. |
| DOM-02 | `ToEntity()` on Model | PASS (pre-existing, unmodified) | `pet/entity.go` (not touched by diff; confirmed present, build succeeds) |
| DOM-03 | `Make(Entity)` in entity.go | PASS (pre-existing, unmodified) | `pet/entity.go` |
| DOM-04 | `Transform` in rest.go | PASS (pre-existing, unmodified) | `pet/rest.go`; new `handleUpdate` reuses it: `pet/resource.go:220` `model.Map(Transform(d.Context()))(...)` |
| DOM-05 | `TransformSlice` used for lists | PASS (pre-existing) | not touched by this diff |
| DOM-06 | Processor ctor accepts `logrus.FieldLogger` | PASS | `pet/processor.go:116` `func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor` |
| DOM-07 | Handlers pass `d.Logger()` | PASS | `pet/resource.go:193` `p := NewProcessor(d.Logger(), d.Context(), d.DB())` (new `handleUpdate`) |
| DOM-08 | POST/PATCH use `RegisterInputHandler` | PASS | `pet/resource.go:29` `r.HandleFunc("/{petId}", rest.RegisterInputHandler[RestModel](l)(db)(si)("update_pet", handleUpdate)).Methods(http.MethodPatch)` |
| DOM-09 | Transform errors handled | PASS | `pet/resource.go:220-224` — err checked, `WriteErrorResponse` called, no `_, _ :=` discard |
| DOM-10 | Test DB registers tenant callbacks | PASS | `pet/processor_test.go:80` `database.RegisterTenantCallbacks(l, db)`; new tests (`TestRenameAppliesAndEmits` etc.) reuse `testDatabase(t)` |
| DOM-11 | Providers use lazy evaluation | PASS (pre-existing provider.go untouched); processor uses `model.Map`/`model.CollapseProvider` throughout | `pet/processor.go:209-215` |
| DOM-12 | No `os.Getenv()` in handlers | PASS | `grep os.Getenv pet/resource.go` → no matches |
| DOM-13 | No cross-domain logic in handlers | PASS | `handleUpdate` only calls `p.GetById` / `p.RenameAndEmit`, both on the pet processor |
| DOM-14 | Handlers don't call providers directly | PASS | same as above — no `getById(...)` (lowercase provider) call in resource.go |
| DOM-15 | No direct entity creation/db writes in handlers | PASS | `grep -n "db.Create\|db.Save\|db.Delete" pet/resource.go` → no matches |
| DOM-16 | `administrator.go` has the write op | PASS | `pet/administrator.go:96-115` `updateName(db)(petId, name)` |
| DOM-17 | Domain error → HTTP status mapping | **FAIL (Important)** | See **Finding F1** below. |
| DOM-18 | JSON:API interface on RestModel | PASS (pre-existing, unmodified) | `pet/rest.go` |
| DOM-19 | Flat request structure | PASS (pre-existing) | `RestModel` reused as the PATCH input type, already flat |
| DOM-20 | Table-driven tests | **PARTIAL / Minor** | See **Finding F2** below. |
| DOM-21 | No duplication of atlas-constants types | PASS | `pet/resource.go:14` and `pet/processor.go:26` import `petconst "github.com/Chronicle20/atlas/libs/atlas-constants/pet"`; no local 4/12 literals reintroduced (`grep -n "MinNameLength\|MaxNameLength" services/atlas-pets` → only the imported package). |
| DOM-24 | Kafka producer stubbed in tests that emit | N/A (PASS) | `pet.Rename`/`RenameAndEmit` emit via `outbox.EmitProvider` (DB row write, not a synchronous Kafka producer call — `libs/atlas-outbox/provider.go:21-30`), so no producer stub is required. Confirmed by reading `EmitProvider`'s body: it calls `EnqueueBuffer` against the transaction, never `producer.ProviderImpl`. |
| DOM-27 | Transient DB errors → 503 not bare 500 | PASS | `handleUpdate`'s three error branches (`pet/resource.go:196, 206, 217, 224`) all call `server.WriteErrorResponse(d.Logger())(w)(err)`, which classifies via the process-wide classifier registered in `main.go:73`. No `w.WriteHeader(http.StatusInternalServerError)` literal in the new code. |

### Finding F1 — DOM-17: not-found and ownership-mismatch on `PATCH /pets/{petId}` both return 500, not 404/403 (Important)

`handleUpdate` (new in this branch) has two failure paths that the guideline's DOM-17 table names explicitly (`404 Not Found`, and implicitly an authorization failure) but that fall through `WriteErrorResponse`'s default (500, since no transient classifier match) instead of a specific status:

- **Not found:** `pet/resource.go:197` — `existing, err := p.GetById(petId)`. `GetById` → `getById(petId)` (`pet/provider.go:11-20`) uses `db...First(&result)`, which returns `gorm.ErrRecordNotFound` verbatim when the row is absent. `resource.go:198-201` passes this straight to `server.WriteErrorResponse`, which (with no `errors.Is(err, gorm.ErrRecordNotFound)` branch) yields 500.
- **Ownership mismatch:** `pet/processor.go:990-991` — `Rename` returns a plain `fmt.Errorf("pet [%d] is not owned by character [%d]", petId, actorId)` (not a sentinel). `resource.go:205-208` (the `RenameAndEmit` call) maps this to `WriteErrorResponse` too — again 500, no `errors.Is` branch exists to catch it.

This is the same pattern the sibling `handleGetPet` and `handleCreate` already exhibit (pre-existing, not introduced by this branch), but `handleUpdate` is new code in this diff, and per the audit's own instruction, "N/A — the codebase does this" is not an exemption — DOM-17 states plainly that not-found errors must map to 404. Neither a 404 sentinel nor a 403/409 branch exists for this handler's two named failure modes.

**Severity:** Important (explicit REST guideline non-conformance in new code, not a widespread-therefore-exempt structural issue — DOM-17 names the exact mapping missed).

### Finding F2 — DOM-20: new tests are not table-driven (Minor)

None of the following new tests use the `tests := []struct{...}{}` + `t.Run` pattern the guideline specifies, even where the shape is naturally tabular (several loop over a slice of bad inputs with a bare `for` instead):

- `pet/processor_test.go:1777` `TestRenameAppliesAndEmits`
- `pet/processor_test.go:1802` `TestRenameIsIdempotent`
- `pet/processor_test.go:1858` `TestRenameRejectsInvalidName` — loops `for _, bad := range []string{"", "abc", "abcdefghijklm"}` without `t.Run` subtests
- `pet/processor_test.go:1876` `TestRenameRejectsNonOwner`
- `pet/resource_test.go` `TestPatchPetRejectsInvalidName`, `TestPatchPetRenamesPet`
- `libs/atlas-constants/pet/name_test.go` (all 6 tests) — `TestValidateNameAcceptsBounds`/`RejectsTooShort` loop over slices without `t.Run`

Every sibling test in the same files follows the same non-table-driven convention (this is the pre-existing style throughout `atlas-pets/pet` and `atlas-constants`), so this is a repo-wide pattern, not something invented by this branch — but per audit rules prevalence doesn't convert it to compliance. Rated Minor/non-blocking because DOM-20 is a style checklist item, not a File-Responsibilities/layering violation, and the tests are otherwise substantive (they assert the right behavior, including the emitted-event count via outbox introspection).

---

## `libs/atlas-constants/pet` (new support package)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| File responsibilities | Single-purpose file, no collapsed catch-all | PASS | `name.go` holds only `MinNameLength`/`MaxNameLength`/`NormalizeName`/`ValidateName` — no Processor/RestModel/requests mixed in |
| DOM-21 self-consistency | Constants are genuinely new (no shared equivalent already existed) | PASS | `grep -rn "PetName\|pet name" libs/atlas-constants/**/*.go` (pre-diff) found nothing pre-existing; this is the first canonical home for the bound |

Consumed correctly from all 3 call sites (`services/atlas-pets/atlas.com/pets/pet/resource.go:14`, `pet/processor.go:26`, `services/atlas-channel/atlas.com/channel/socket/handler/character_cash_item_use_pet_name_tag.go:15`) — confirmed via `grep -rln "atlas-constants/pet\"" services/atlas-pets services/atlas-channel services/atlas-saga-orchestrator` → exactly those three files, and no service redeclares `4`/`12` as local literals (`TestBoundsMatchClientDialog`, `libs/atlas-constants/pet/name_test.go:48-51`, pins the constants against the IDA citation instead of a magic number elsewhere).

---

## `libs/atlas-packet` (codec support library — cash/serverbound, pet/clientbound)

Not a REST domain package; File Responsibilities / DOM-04–09/17–19 don't apply (no Processor/RestModel/requests concepts in this library). Evaluated for correctness and DOM-25 (client wire values) instead.

| Check | Status | Evidence |
|-------|--------|----------|
| Ten-version clientbound coverage claim | PASS | `libs/atlas-packet/pet/clientbound/name_changed_test.go` has one `packet-audit:verify` byte-fixture test per version: v48 (`:17`), v61(`:36`), v72(`:55`), v79(`:76`), v83(`:95`), v84(`:117`), v87(`:137`), v92(`:158`), v95(`:179`), jms_v185(`:199`), plus `TestPetNameChangedDecodeRoundTripGMS` and `TestNameTagLayerAgreesWithActivated`. |
| DOM-25 — no client wire code as bare Go literal | PASS (with reasoning verified) | `NameTagLayer = byte(0)` (`pet/clientbound/name_changed.go:31`) is a per-pet decoration selector consumed identically by every version (`Activated.nameTag`, `activated.go:34-36`, and `NameChanged.nameTag`), not a dispatcher/sub-op/notice-reason code the client looks up via a table — it is outside DOM-25's scope (mode bytes, sub-op codes, message types, notice/fail-reason codes). It is also always `0` by design (Atlas has no per-pet name-tag inventory), so there is nothing to make tenant-configurable. |
| `CashSlotItemType(17)` classification constant | PASS (not a DOM-25 violation) | `character_cash_item_use.go:948-960` replicates the CLIENT's own `get_cashslot_item_type` classification logic to decide how to DECODE an inbound packet — this is inbound protocol-constant mirroring, not a value the server writes into an outbound client-interpreted lookup switch. The real bug fixed here (`itemId%10000 != 0` replacing the overflowing `10000*itemId/10000 != itemId`) is a correctness fix, not a config-resolution gap. |
| Serverbound sub-body codec well-formed, has both Encode/Decode | PASS | `libs/atlas-packet/cash/serverbound/item_use_pet_name_tag.go:47-67` |

---

## `libs/atlas-saga` (shared saga type/action/payload library)

| Check | Status | Evidence |
|-------|--------|----------|
| New `Type`/`Action`/`Payload` follow existing enum + switch-registration shape | PASS | `libs/atlas-saga/model.go:45` `PetNameTagUse Type = "pet_name_tag_use"`; `:95` `RenamePet Action = "rename_pet"`; `libs/atlas-saga/payloads.go:298-305` `RenamePetPayload`; `libs/atlas-saga/unmarshal.go:189-195` `case RenamePet:` registered in `Step[T].UnmarshalJSON` |
| No collapsed catch-all file | N/A | pre-existing `model.go`/`payloads.go`/`unmarshal.go` split is followed, new code added in the matching file for its responsibility |

---

## `services/atlas-saga-orchestrator` — `saga` package (extended, not new)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| Handler registration | PASS | `saga/handler.go:828-829` `case RenamePet: return h.handleRenamePet, true`; `saga/handler.go:1369-1381` `handleRenamePet` mirrors sibling `handleEvolvePet` shape |
| Event acceptance wiring | PASS | `saga/event_acceptance.go:56` `EventKindPetNameChanged`; `:151` `sharedsaga.RenamePet: {EventKindPetNameChanged}`; `:360` `EventKindPetNameChanged: OutcomeSuccess` |
| Reverse-walk / timeout classification completeness | PASS | `saga/timer.go:181` adds `PetNameTagUse` to `reverseWalkSagaTypes`; `:208` adds it to `allSagaTypes`; `saga/timer.go:253-257` `case PetNameTagUse: c.DispatchPetNameTagRollbacks(s)`. `TestEverySagaTypeIsClassified` (pre-existing exhaustiveness test, comment at `timer.go:188-193`) passed in the plain `go test ./...` run, confirming `PetNameTagUse` is not left unclassified. |
| Compensator wired into `CompensateFailedStep` | PASS | `saga/compensator.go:325-326` `if s.SagaType() == PetNameTagUse { return c.compensatePetNameTagUse(s, failedStep) }` |
| `EmitSagaFailed` carries a real characterId (not 0) | PASS | `saga/producer.go:192-199` adds `PetNameTagUse` alongside the pre-existing `MesoSackUse` special-case, using `petNameTagCharacterId(s)` (`compensator.go:1789-1806`) |
| Mock synchronization | PASS | `pet/mock/processor.go:16-17` (interface fields) and `:53-68` (method impls) both add `RenameAndEmitFunc`/`RenameFunc` — full nil-check pattern matches guideline's Mock Implementation Pattern |
| Processor shape (Interface+Impl, `NewProcessor(l,ctx)`, pure vs AndEmit) | PASS | `pet/processor.go:16-23` (interface), `:32-39` (`NewProcessor`), `:70-77` (`RenameAndEmit`/`Rename` pair) |
| Producer places fields correctly | PASS | `pet/producer.go:38-49` `RenameProvider` sets `ActorId: characterId` on the envelope, `Body: RenameCommandBody{Name: name}` |

### Finding F3 — new compensation-revert test is invisible to the mandated `go test ./...` (Minor/informational, pre-existing pattern)

`saga/pet_name_tag_compensation_test.go` (new, covers `TestPetNameTagCompensationRevertsName` and `TestPetNameTagCompensationSkipsUncompletedRename` — the FR-7.4 name-revert-on-consume-failure safety behavior) carries `//go:build test` at line 1. Verified directly:

```
go test ./saga/... -run TestPetNameTagCompensationRevertsName -v -count=1
  → testing: warning: no tests to run

go test -tags test ./saga/... -run TestPetNameTagCompensationRevertsName -v -count=1
  → --- PASS: TestPetNameTagCompensationRevertsName (0.00s)
```

`tools/test-all-go.sh` (the repo's canonical "test every module" script, `tools/test-all-go.sh:6`) runs plain `go test ./...` with no `-tags`. This means the FR-7.4 revert-on-failure test — arguably the single most safety-critical test in this feature (does a failed cash-item consume leave the pet's name reverted?) — never executes under the project's own mandated verification command (CLAUDE.md item 1) or under this audit's Phase 1 run.

This build-tag convention is **not introduced by this branch**: `meso_sack_compensation_test.go`, `late_event_integration_test.go`, and `step_event_matching_integration_test.go` already carry the same tag, and `producer_testseam.go`/`processor_testseam.go` (also pre-existing, also `//go:build test`) supply the test-only seams these files depend on. The new file follows the existing (pre-established) convention exactly. Confirmed it passes when run correctly (`-tags test`), so this is not a correctness defect — it is a verification-visibility gap the branch inherited rather than created. Flagged as Minor/informational per audit instructions (prevalence doesn't exempt, but a pattern this deeply load-bearing across the existing test suite is a service-wide test-harness issue, not a task-224-specific regression) — worth a follow-up ticket to either (a) drop the `test` build tag from this file family, or (b) add `-tags test` to `tools/test-all-go.sh`.

---

## `services/atlas-channel` — `socket/handler`, `kafka/consumer/pet`, `kafka/consumer/saga`

| Check | Status | Evidence |
|-------|--------|----------|
| Cash-slot dispatch bug fix is correct | PASS | `character_cash_item_use.go:955-961` — old `10000*itemId/10000 != itemId` overflowed `uint32` for `itemId=5170000` (`10000*5170000 = 51,700,000,000` wraps to `160,392,448`); new `itemId%10000 != 0` matches the cited IDA arm `get_cashslot_item_type @0x48645b, case 517`. `TestGetCashSlotItemTypePetNameTag` (`character_cash_item_use_pet_name_tag_test.go:19-32`) exercises exactly the overflow item id (5170000) and a neighbor (5170001) that must NOT match. |
| SEC: ownership not trusted from client input | PASS | `character_cash_item_use_pet_name_tag.go:104-118` — the target pet is resolved server-side from `petsForOwnerFunc(l, ctx, s.CharacterId())` (session-derived character id, not any client-supplied field) filtered to `Slot()==0`; `:117-121` re-checks `target.OwnerId() != s.CharacterId()` as defense-in-depth. No field of the incoming packet supplies an owner or pet id at all (the wire body is `name` only — confirmed by the codec at `libs/atlas-packet/cash/serverbound/item_use_pet_name_tag.go:47-52`, which decodes only `m.name`). |
| Producer stub for the emit path | PASS (via pre-existing package helper) | `character_cash_item_use_pet_name_tag_test.go:96-97,120-121` calls `installCapturingProducer()` (pre-existing helper, `cash_item_gachapon_test.go:47-58`, unmodified by this diff) before invoking `handlePetNameTagUse`, which transitively calls `saga.NewProcessor(...).Create(...)` → `producer.ProviderImpl` (`services/atlas-channel/atlas.com/channel/saga/processor.go:34-36`, a REAL Kafka producer call, not outbox). Restore is `producertest.InstallNoop()` (`cash_item_gachapon_test.go:58`), not `t.Cleanup(producer.ResetInstance)` — DOM-24(e) anti-pattern avoided. |
| DOM-24(d) — service-local writer vs shared `producertest` | Observation, not a new violation | `installCapturingProducer` swaps the process-wide producer singleton with a **capturing** writer (not a no-op), which the shared `producertest.InstallNoop()` cannot provide (it only discards). `testing-guide.md`'s Pattern B explicitly names "one that captures messages for assertion" as an acceptable variant. This helper is pre-existing (not added by this branch) and is reused, not duplicated, by the new test file. |
| Parallel map-iteration closures capture only immutable values | PASS | `kafka/consumer/pet/consumer.go:456-486` `handleNameChanged` — the `ForSessionsInMap` callback closes over `e.OwnerId` (uint32 copy), `e.Body.Slot` (int8 copy), `e.Body.Name` (string copy), `l`, `ctx`, `wp` — no shared mutable state, consistent with the documented `bug_channel_foreachinmap_parallel_shared_state` concern cited in the code's own comment. |
| Saga-failure rendering (`petNameTagFailureMessage`) | PASS (deliberately generic, confirmed) | `kafka/consumer/saga/consumer.go:410-416` `petNameTagFailureMessage` ignores `errorCode` and returns one message; `consumer.go:380-392` gates it on `saga.SagaTypePetNameTagUse`, announces pink text then an empty `StatChanged` (enable-actions) unlock — matches the task brief's "known and deliberate" note; `TestPetNameTagFailureMessage` (`consumer_test.go`) asserts all three sampled codes (including `""`) produce the same string, proving the code, not just the comment, ignores `errorCode`. |
| `produceWriters()` / seed template wiring (SCAFFOLD-07-equivalent) | PASS | `main.go:730` registers `petcb.PetNameChangedWriter`; all 10 seed templates (`services/atlas-configurations/seed-data/templates/template_{gms_48,61,72,79,83,84,87,92,95}_1.json`, `template_jms_185_1.json`) gained a `"writer": "PetNameChanged"` entry with non-empty `fname` (`template_gms_83_1.json:3288-3295` shown; all 10 confirmed via `git diff --stat`, each +8 lines). `tools/template-opcode-order-guard.sh` and `tools/template-duplicate-binding-guard.sh` both exit 0 across all 22 template writer/handler arrays. |
| Contract mirror drift (pet, saga kafka.go) | PASS | `services/atlas-channel/atlas.com/channel/kafka/message/pet/kafka.go` and `services/atlas-saga-orchestrator/atlas.com/saga-orchestrator/kafka/message/pet/kafka.go` diffs are field-for-field identical to the owner (`services/atlas-pets/.../kafka/message/pet/kafka.go`) for every new symbol (`CommandPetRename`, `RenameCommandBody`, `StatusEventTypeNameChanged`, `NameChangedStatusEventBody`) — confirmed by side-by-side diff. Both mirrors carry `contract_mirror_test.go` fixtures (`diff` between the two files' bodies is empty) asserting the owner's exact wire JSON unmarshals correctly on the mirror side — the documented substitute for the (non-existent) mirror-guard script this contract lacks. |

---

## SUB-* (sub-domain / action-event packages)

No new package matches "has `resource.go` but no `model.go`" in this diff's file list. N/A.

## EXT-* (external HTTP client checklist)

No new `requests.GetRequest[T]`/`requests.PostRequest[T]`/`requests.RootUrl(...)` call sites in the diff (`grep` across the full branch diff returns zero matches). N/A.

## SCAFFOLD-* (new service / new channel packet family)

No new `services/atlas-<name>/` directory added; the writer/handler additions are to existing `atlas-channel` and `libs/atlas-packet` packages, and SCAFFOLD-07's template-writer requirement was independently verified above (PASS) even though the full scaffolding trigger (`git diff --name-status` for a new `main.go`) doesn't fire. N/A for SCAFFOLD-01–06/08.

## SEC-* (auth-related)

Not an auth service; the one access-control-relevant surface (PATCH ownership) is covered under the atlas-channel and atlas-pets sections above (PASS, with a real httptest-backed end-to-end proof at `services/atlas-pets/atlas.com/pets/pet/resource_test.go` `TestPatchPetRenamesPet`, which POSTs a payload with `OwnerId: 999` and asserts the stored owner `1` is unchanged and the rename still succeeds because the handler used the stored row).

---

## Summary

### Blocking (must fix)
- **F1 (DOM-17, Important):** `services/atlas-pets/atlas.com/pets/pet/resource.go` `handleUpdate` — a not-found pet (`gorm.ErrRecordNotFound` from `p.GetById`, `resource.go:197`) and a non-owner rename attempt from the operator PATCH endpoint (`pet/processor.go:990-991`, surfaced at `resource.go:206`) both return HTTP 500 instead of a specific 404/403. Add `errors.Is(err, gorm.ErrRecordNotFound)` → 404 and a sentinel ownership error → 403/409 branch before falling through to `WriteErrorResponse`.

### Non-Blocking (should fix)
- **F2 (DOM-20, Minor):** New tests across `atlas-pets/pet` and `atlas-constants/pet` are not table-driven; several loop over multiple inputs with a bare `for` instead of `t.Run` subtests. Consistent with pre-existing sibling tests in the same files, but still a deviation from the documented pattern.
- **F3 (informational, service-wide):** `saga/pet_name_tag_compensation_test.go`'s `//go:build test` tag (inherited from the existing `meso_sack_compensation_test.go` convention) makes the new FR-7.4 name-revert safety test invisible to the plain `go test ./...` this repo's own tooling (`tools/test-all-go.sh`) and CLAUDE.md's verification step run. Confirmed the test passes correctly under `-tags test`. Recommend a follow-up to either untag this test family or teach `tools/test-all-go.sh` about `-tags test`.
- **Observation:** `installCapturingProducer` (pre-existing atlas-channel test helper, reused not duplicated by this branch) is a service-local capturing writer rather than the shared `producertest` package — permitted by `testing-guide.md`'s own text for message-capture assertions, not treated as a violation, but noted since DOM-24(d) reads narrowly.

### Passed (selected highlights)
- DOM-21 (atlas-constants reuse) — clean across all three consumers, no reinvented 4/12 literal.
- SEC ownership check — proven end-to-end via httptest, not just by code inspection.
- Kafka contract mirrors (atlas-pets → atlas-channel/atlas-saga-orchestrator) — byte-identical new types, both carry round-trip fixture tests substituting for the missing mirror-guard script.
- Saga wiring completeness (`allSagaTypes`, `reverseWalkSagaTypes`, event acceptance/outcome tables, compensator dispatch, mock sync) — all four registration points hit, `TestEverySagaTypeIsClassified` passes.
- Ten-version clientbound packet coverage with byte fixtures, plus the `NameTagLayer` cross-codec consistency test.
- Parallel-iteration closure safety in the map-broadcast handler.
