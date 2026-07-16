# Backend Audit — task-131-random-reward-items

- **Service Path(s):** services/atlas-consumables, services/atlas-inventory, services/atlas-data, services/atlas-channel, libs/atlas-packet, services/atlas-configurations (seed-data templates)
- **Diff:** `b1c50b67d36c9c7174bdd2977de635b8b074051c..23e9d3c20`
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-16
- **Build:** PASS (all five modules: `go build ./...` clean)
- **Tests:** PASS (all five modules: `go test ./... -count=1` clean; no failures)
- **Overall:** NEEDS-WORK (build/tests are clean but multiple FAIL checks below)

## Build & Test Results

```
services/atlas-consumables/atlas.com/consumables : go build ./... -> clean; go test ./... -count=1 -> ok (all packages with tests pass)
services/atlas-inventory/atlas.com/inventory      : go build ./... -> clean; go test ./... -count=1 -> ok
services/atlas-data/atlas.com/data                : go build ./... -> clean; go test ./... -count=1 -> ok
services/atlas-channel/atlas.com/channel          : go build ./... -> clean; go test ./... -count=1 -> ok
libs/atlas-packet                                 : go build ./... -> clean; go test ./... -count=1 -> ok
```

No go.mod changes in any touched module — DOM-22 (Dockerfile lib-mention count) is not triggered.
No new Kafka topics introduced (EVENT_TOPIC_ASSET_STATUS / EVENT_TOPIC_COMPARTMENT_STATUS both pre-exist in `deploy/k8s/base/env-configmap.yaml:83,97`) — DOM-23 is satisfied for this diff.

## Critical Findings

### C1 — SCAFFOLD-07 / scope violation: `CharacterItemUseLotteryHandle` registered for v92, contradicting the task's own explicit scope decision, on an admittedly-unverified opcode

- **Evidence (scope declaration):** `docs/tasks/task-131-random-reward-items/context.md:17` — "**v92 is DROPPED from this task** (implemented versions: **v83, v84, v87, v95**)." `context.md:25-27` — "v92 has **no IDB** ... so the missing opcodes/modes cannot be verified — populating them would mean inventing values, which the project rules forbid."
- **Evidence (plan scope):** `docs/tasks/task-131-random-reward-items/plan.md:1634` — "## Task 13: atlas-configurations — seed-template handler entries (v83/v84/v87/v95)"; `plan.md:1636-1640` **Files:** list contains only `template_gms_83_1.json`, `template_gms_84_1.json`, `template_gms_87_1.json`, `template_gms_95_1.json` — `template_gms_92_1.json` is absent.
- **Evidence (rollout runbook explicitly forbids it):** `docs/tasks/task-131-random-reward-items/rollout.md:203-217` — "## Step 3: v92 / jms — explicitly out of scope, do not patch. **Do not** add the `CharacterItemUseLotteryHandle` handler entry ... for: v92 — ... the opcode (`0x07B`, registry/CSV lineage only) ... is unverified ... **Adding an unverified opcode to a live v92 tenant risks colliding with an existing handler/writer or routing to the wrong body.**"
- **Evidence (design doc marks the value unverified):** `docs/tasks/task-131-random-reward-items/design.md:48` — "| v92 | 0x07B | CSV (v92 has no IDB — template-lineage convention, flag in evidence). **Not in the coverage matrix version set** |"
- **Evidence (violation — the actual diff)::** `services/atlas-configurations/seed-data/templates/template_gms_92_1.json:158` adds:
  ```json
  {
    "opCode": "0x7B",
    "validator": "LoggedInValidator",
    "handler": "CharacterItemUseLotteryHandle"
  },
  ```
- **Why it fails:** Every one of the task's own planning artifacts (context.md, plan.md's Task 13 Files: list, rollout.md's Step 3) explicitly excludes v92 from this change, for the specific reason that the opcode is unverified (no IDA instance exists for v92) and could misroute an unrelated real client packet. `docs/packets/audits/STATUS.md`'s coverage matrix confirms v92 has no tracked column at all (9 columns: v48/v61/v72/v79/v83/v84/v87/v95/JMS185) — so this opcode was never run through the packet-audit verification pipeline. The committed template change does exactly what three separate task documents instruct not to do, using a value each of those documents calls "CSV lineage only" / "unverified." Per DOM-25's uniformity ruling and project memory ("Unresolved packet-audit fnames escalate — stop-and-ask; never substitute or fake"), this is a stop-and-ask case that was not stopped on.
- **Severity:** Critical (production-safety risk to any v92 tenant seeded from this template; direct contradiction of the task's own documented scope).

### C2 — Rollout runbook (a committed deliverable) contains a factually false claim about the jms template

- **Evidence:** `docs/tasks/task-131-random-reward-items/rollout.md:218-222` — "**jms** — out of scope per design §2.6. No jms IDA instance was available to verify the client-side routing/body assumption for jms, **so no handler entry was added to `template_jms_185_1.json`** and none should be PATCHed onto live jms tenants either."
- **Evidence (contradicting code):** `services/atlas-configurations/seed-data/templates/template_jms_185_1.json:315` adds exactly that entry:
  ```json
  {
    "opCode": "0x6B",
    "validator": "LoggedInValidator",
    "handler": "CharacterItemUseLotteryHandle"
  },
  ```
  and `docs/packets/audits/STATUS.md:649` shows `LOTTERY_ITEM_USE_REQUEST` promoted to `✅` for `JMS185` at opcode `0x6B`, backed by a real IDA-verified byte fixture: `libs/atlas-packet/inventory/serverbound/lottery_item_use_test.go:24` (`packet-audit:verify ... version=jms_v185 ida=0xaf6900`) and `docs/packets/audits/jms_v185/InventoryLotteryItemUse.md:3,7` (`IDA: 0xaf6900`, `Verdict: ✅`).
- **Note:** `design.md:38,53` documents that jms was later brought into scope with real verification ("IDA-verified live during the main-merge scope expansion (task-131)" / "jms | 0x06B | IDA-verified (fn 0xaf6900); registry/CSV agree. **In scope** (§2.6)"), so the underlying jms engineering is sound and evidenced. The defect is that `context.md` (lines 27, 32: "This is the same situation that excluded jms" / "jms was already out of scope") and `rollout.md` (Step 3) were never updated to match — a committed runbook that operators would follow literally describes behavior the code does not have. Per CLAUDE.md's "Grounding & Honesty" rule, a doc that doesn't match the code it describes is a real defect, not a nitpick — an operator following rollout.md's Step 3 would incorrectly believe jms received no template change.
- **Severity:** Important (documentation/grounding integrity, not a runtime risk — the jms change itself is verified).

## File Responsibilities Findings

### F1 — FILE-02/FILE-03: collapsed REST+requests file in `atlas-consumables/inventory` (task-102 wallet.go pattern recurrence)

- **File:** `services/atlas-consumables/atlas.com/consumables/inventory/accommodation.go`
- **Evidence:** lines 17-53 define `accommodationInputRestModel` / `accommodationOutputRestModel` with the full JSON:API interface (`GetName`/`GetID`/`SetID`/`SetToOneReferenceID`/`SetToManyReferenceIDs`) — this is `rest.go`'s exclusive responsibility per file-responsibilities.md ("Define `RestModel` struct implementing JSON:API interface ... Define request models"). Lines 55-61 define `requestCheckAccommodation(...)` calling `requests.PostRequest[accommodationOutputRestModel]` — this is `requests.go`'s exclusive responsibility ("REST client functions for calling other microservices").
- **Why it's not a style nit:** the SAME package already has a correctly-split `rest.go` (RestModel + Transform/Extract, `services/atlas-consumables/atlas.com/consumables/inventory/rest.go:11-114`) and `requests.go` (`getBaseRequest`/`requestById`, `services/atlas-consumables/atlas.com/consumables/inventory/requests.go:14-20`) for its pre-existing symbol. The new `accommodation.go` bypasses that established split and re-creates both responsibilities in one new file — the exact anti-pattern flagged by task-102's `wallet.go` precedent that this audit's own mindset section calls out by name.
- **Fix:** move `accommodationInputRestModel`/`accommodationOutputRestModel` (+ their JSON:API methods) into `rest.go`, and `requestCheckAccommodation` into `requests.go`.
- **Severity:** Important.

### F2 — FILE-01: Processor methods in a bare-topic-named file in `atlas-inventory/compartment`

- **File:** `services/atlas-inventory/atlas.com/inventory/compartment/accommodation.go`
- **Evidence:** `func (p *ProcessorImpl) CanAccommodate(...)` at line 49 and `func (p *ProcessorImpl) accommodatesOne(...)` at line 73 are `ProcessorImpl` methods (the interface method itself was added to `Processor` in `processor.go` per the diff to that file: `+ CanAccommodate(characterId uint32, reqs []AccommodationRequest) ([]AccommodationResult, error)`), but the method bodies live in `accommodation.go` — a bare topic-name file, exactly the pattern FILE-01 names as disallowed ("a bare topic name like `custody.go`/`register.go`").
- **Fix:** rename to `processor_accommodation.go` (the sanctioned large-Processor split-file convention already used elsewhere, e.g. `processor_custody.go`), or move the methods into `processor.go` directly.
- **Severity:** Important.

## External HTTP Client Checklist Findings

Both of the following are NEW packages in this diff that call another atlas service via `requests.GetRequest[T]`/`PostRequest[T]`, so the EXT-* checklist applies.

### E1 — EXT-01 / EXT-02: `atlas-consumables/data/itemstring` (new package, calls atlas-data)

- **File:** `services/atlas-consumables/atlas.com/consumables/data/itemstring/rest.go:1-15`
- **EXT-01 FAIL:** `RestModel` implements only `GetName()`/`GetID()`/`SetID()` (lines 8-15). It does NOT implement `SetToOneReferenceID`/`SetToManyReferenceIDs`, even as no-ops. Per EXT-01, api2go errors on any response carrying a `relationships` block without these methods — this is the exact class of bug task-037 hit twice.
- **EXT-02 FAIL:** the package has zero test files (`go test` output: `?   atlas-consumables/data/itemstring   [no test files]`). There is no httptest-backed (or any) test verifying `GetName(itemId)` actually round-trips a representative atlas-data JSON:API response.
- **Severity:** Important.

### E2 — EXT-02: `atlas-consumables/inventory.CanAccommodate` (new cross-service call, calls atlas-inventory)

- **File:** `services/atlas-consumables/atlas.com/consumables/inventory/` (package has zero `*_test.go` files)
- **Evidence:** `requestCheckAccommodation` (accommodation.go:55-61) is a brand-new `requests.PostRequest[accommodationOutputRestModel]` call wired into the reward pre-roll gate (`consumable/processor.go:957` — `ok, err := p.ip.CanAccommodate(characterId, accItems)`), a load-bearing check that decides whether a reward-box use is rejected. There is no test — httptest-backed or otherwise — exercising the actual JSON:API unmarshal of a representative `accommodationOutputRestModel` response.
- **Severity:** Important.

(Note: `accommodationOutputRestModel`/`accommodationInputRestModel` on the consumables side, and `AccommodationInputRestModel`/`AccommodationOutputRestModel` on the atlas-inventory side, DO both correctly implement `SetToOneReferenceID`/`SetToManyReferenceIDs` as no-ops — EXT-01 passes for the accommodation REST models themselves; only `data/itemstring` fails EXT-01.)

## Resilience Pattern Finding (DOM-28)

### R1 — Silent degradation without the `degrade.Observe` metric in `emitRewardPresentation`

- **File:** `services/atlas-consumables/atlas.com/consumables/consumable/processor.go`
- **Evidence:**
  - Lines 1120-1125: `p.cp.GetById()(characterId)` (character-name enrichment for the `/name` substitution) — on error, `p.l.WithError(err).Warnf(...)` and falls through with `name = ""`. No `degrade.Observe(...)` call.
  - Lines 1126-1130: `itemstring.NewProcessor(p.l, p.ctx).GetName(won.ItemId())` (item-name enrichment for `/item` substitution) — on error, `p.l.WithError(err).Warnf(...)` and `return` (skips the announce). No `degrade.Observe(...)` call.
- **Guideline:** `patterns-resilience.md` — "A decorator or enrichment step that fails its fetch MUST NOT silently return the un-enriched model... it logs Warn with the entity id and increments `atlas_enrichment_degraded_total{component}`... These patterns are mandatory for new code." Both sites here are new code (task-131) that fetch remote data and branch on error with only half the contract (Warn log present, metric increment absent).
- **Severity:** Important (this pattern is currently adopted in only 2 of ~14 services per repo-wide grep — atlas-skills, atlas-login — but the guideline explicitly states it is mandatory for new code, and prevalence elsewhere does not exempt a deviation per this audit's own grading rule).

## Domain / Sub-Domain Checklist Highlights (PASS items worth recording)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06/07 | Processor accepts `FieldLogger`, handlers pass `d.Logger()` | PASS | `services/atlas-inventory/atlas.com/inventory/compartment/accommodation_rest.go:73` — `NewProcessor(d.Logger(), d.Context(), db)` |
| DOM-08 | POST uses `RegisterInputHandler[T]` | PASS | `services/atlas-inventory/atlas.com/inventory/compartment/resource.go:26` — `rest.RegisterInputHandler[AccommodationInputRestModel](l)(si)("check_accommodation", ...)` |
| DOM-17/DOM-27 | Error mapping uses `WriteErrorResponse`, not bare 500 | PASS | `accommodation_rest.go:76` — `server.WriteErrorResponse(d.Logger())(w)(err)` |
| DOM-25 | Client-interpreted wire value config-resolved | PASS | `libs/atlas-packet/character/effect_body.go:233-242` — `CharacterLotteryUseEffectBody`/`...ForeignBody` both use `atlas_packet.WithResolvedCode("operations", string(CharacterEffectLotteryUse), ...)`, never a literal mode byte |
| DOM-25 (v61 table cleanup) | Removing a lineage-copied, unverified operations entry | PASS | `template_gms_61_1.json` — `"LOTTERY_USE": 14` removed from both `operations` blocks; documented and IDA-justified at `design.md:142-149`/`rollout.md:142-149` (v61's real case 14 is a different, `{path}`-only arm; sending the 3-field body there would misparse) |
| Reward once-handler design | Handler registration ordering / cleanup on both success and failure paths | PASS | `consumable/processor.go:1036-1064` (`ConsumeReward`) registers both once-handlers *before* emitting `CREATE_ASSET`, and every early-return path (`RegisterHandler` failure for either topic, `RequestCreateItem` failure) calls `consumer.GetManager().RemoveHandler` on whichever handle(s) were already registered — no leaked handler on the setup-failure paths. `grantRewardOnConfirmed` (line 1070) and `grantRewardOnFailed` (line 1095) each deregister their sibling on the terminal (mutually-exclusive) success/failure signal. |
| CreateAssetAndEmit CREATION_FAILED re-emit fix | Rejection re-emitted outside the rolled-back tx via direct producer, mirroring the existing drop-pickup reject pattern | PASS | `services/atlas-inventory/atlas.com/inventory/compartment/processor.go:994-1027`; covered by new test `TestCreateAssetAndEmitInventoryFull` (`compartment/processor_test.go:875-931`), which correctly reuses the pre-existing `installCapturingProducer` helper and restores via `producertest.InstallNoop()` (not the unsafe bare `ResetInstance`) — DOM-24 satisfied. |
| DOM-24 | Kafka producer stubbed via shared `producertest`, no `t.Cleanup(ResetInstance)` | PASS | `compartment/processor_test.go:77-82` (`TestMain` → `producertest.InstallNoop()`); `installCapturingProducer`'s restore func (line 72-74) calls `producertest.InstallNoop()`, never a bare `ResetInstance` left dangling via `t.Cleanup` |
| Reader per-entry reward fields | `Effect`/`worldMsg`/`period` parsed with correct WZ casing and defaults | PASS | `services/atlas-data/atlas.com/data/consumable/reader.go:167-180`; table-driven test `TestReaderRewardFields` (`reader_test.go:1013-1073`) verifies both a fully-populated entry and a defaults-only entry |
| DOM-21 | No atlas-constants duplication | PASS | New types (`RewardModel`, `SpecType` values, reward-table roll logic) are service-specific reward semantics with no `libs/atlas-constants` equivalent; `item2.Id`/`item2.GetClassification`/`inventory2.Type`/`world.Id` etc. are reused throughout, not redeclared |

## Summary

### Blocking (must fix)
- **C1** — Remove the `CharacterItemUseLotteryHandle` handler entry from `template_gms_92_1.json:158` (opCode `0x7B`) — contradicts context.md/plan.md/rollout.md's explicit v92 exclusion and uses an admittedly-unverified opcode.
- **C2** — Update `rollout.md` Step 3 (and `context.md`'s jms references) to match reality: jms WAS verified and the template WAS changed; the runbook currently tells an operator the opposite of what the code does.
- **F1** — Split `atlas-consumables/inventory/accommodation.go` into its `rest.go` (REST models) and `requests.go` (request function) responsibilities.
- **F2** — Rename/move `atlas-inventory/compartment/accommodation.go`'s `ProcessorImpl` methods into `processor.go` or `processor_accommodation.go`.
- **E1** — Add `SetToOneReferenceID`/`SetToManyReferenceIDs` to `data/itemstring.RestModel`; add an httptest-backed test for `itemstring.GetName`.
- **E2** — Add an httptest-backed test for `inventory.CanAccommodate`'s JSON:API round trip.

### Non-Blocking (should fix)
- **R1** — Add `degrade.Observe(...)` metric increments alongside the existing Warn logs in `emitRewardPresentation`'s two enrichment fetches.
- No dedicated integration-style test exists for `RequestItemReward`/`ConsumeReward`'s once-handler orchestration (registration ordering, cleanup-on-failure) — consistent with sibling reservation flows (`RequestScroll`/`ConsumeScroll`) which are similarly untested at that level, so not blocking, but worth a follow-up given the added complexity of the two-topic once-handler design.
