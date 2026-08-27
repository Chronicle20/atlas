# Backend Audit — task-272 (FINAL, pre-PR)

- **Service Path(s):** services/atlas-cashshop, atlas-channel, atlas-character,
  atlas-consumables, atlas-dragons, atlas-login, atlas-messages, atlas-npc-shops,
  atlas-pets, atlas-query-aggregator
- **Range audited:** `b284bcebf..HEAD` (`HEAD = 2138acf85`), whole branch
- **Guidelines Source:** backend-dev-guidelines skill (`resources/audit-checklist.md`)
- **Date:** 2026-08-27
- **Build:** PASS (verified as of `353d3bd10`; `git log 353d3bd10..HEAD` shows
  only a docs commit, `2138acf85`, with zero `.go` changes — the code tree at
  HEAD is byte-identical to what the flagless `tools/verify.sh` last exercised)
- **Tests:** PASS (same basis)
- **Overall:** PASS

## Build & Test Results

`git diff --stat 353d3bd10..HEAD -- '*.go'` returns empty — no Go source
changed since the last flagless `tools/verify.sh` PASS. Per the task brief,
build/test breakage over this range is not the open question; this audit
covers what the objective gate cannot see.

## Applicability

- **DOM structure (DOM-01..05, 11, 16):** Fired — every touched `character`
  package has `model.go`; `atlas-character/character` additionally has
  `provider.go`. Opened `file-responsibilities.md`.
- **FILE placement (FILE-01..06):** Fired — every changed Go package.
- **SUB sub-domain:** N/A — no changed package has `resource.go` without
  `model.go` (`git diff --name-only b284bcebf..HEAD -- '*resource.go'`
  returns empty for the whole range).
- **REST (DOM-06..09, 12..15, 17..19, 32):** Fired for the touched
  `rest.go`/`processor.go` files. No `resource.go` in the diff, so DOM-07/08/
  09/12/13/14/15/32 (all handler-file rules) are N/A on their own trigger;
  DOM-04/05/06/17/18/19 evaluated below.
- **Constants reuse (DOM-21):** N/A — diff adds only a `uint32` field
  (`spawnPoint`) to existing structs; no new type, const block, or
  numeric-literal classification.
- **Testing (DOM-10, 20, 24, 33):** Fired — diff touches many `_test.go`
  files and one mock (`query-aggregator/character/mock/processor.go`).
- **Cache (DOM-29):** N/A — no `cache.go` in scope; no processor gained
  cached state.
- **Messaging (DOM-30):** N/A — no new `AndEmit`/`producer.ProviderImpl`
  call site added; the one Kafka-consumer touch
  (`kafka/consumer/character/consumer.go`) only adds a `Build()` error
  branch ahead of the pre-existing `CreateAndEmit` call.
- **Multi-tenancy (DOM-31):** Fired (package has `rest.go`) — no new
  tenant/trace field added to any REST model; the diff only adds `spawnPoint`.
- **Migration hygiene (DOM-34/35):** N/A — no symbol moved to/from `libs/`.
- **Deploy & topics (DOM-22/23):** N/A — no `libs/` module or topic env var
  added.
- **Runtime safety (DOM-26):** Fired — no `go` statement in any non-test
  file changed in this range.
- **Channel wire values (DOM-25):** Fired (diff touches
  `services/atlas-channel`) — the `byte(c.SpawnPoint())` narrowing in
  `character_data.go`/`character_list.go` is a data-value truncation (a
  portal index), not a dispatcher mode / sub-op code / fail-reason code
  resolved via a client lookup switch; DOM-25's own trigger text ("dispatcher
  modes, sub-op codes, message/fail-reason codes... any byte the client feeds
  through a lookup switch") does not cover it. N/A on the rule's own trigger,
  not the family's.
- **Resilience (DOM-27, 28):** Fired — `Build()` on nine `character`
  packages became fallible; every enrichment/decorator fallback that reaches
  it was reviewed (see Checklist Results).
- **External clients (EXT-01..04):** N/A — no new `requests.*Request[T]`
  call site.
- **Scaffolding (SCAFFOLD-*):** N/A — no new service, channel writer/handler,
  or `routes.conf` change.
- **Security (SEC-*):** N/A — none of the ten services in scope handle
  auth/tokens/redirects in the touched files.
- **Foundational — patterns-provider.md:** N/A — no provider defined or
  composed; `character/provider.go`'s only change (`provider.go:88`) is
  binding `Build()`'s existing error, covered by DOM-01/11.
- **Foundational — patterns-functional.md:** Fired — `model.Decorator[Model]`
  composition is touched throughout; folded into the DOM-28 review below.

## Checklist Results

### character (domain package, all ten services)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-01 | `builder.go` has `NewBuilder()`, fluent setters, validating `Build()` | PASS | `services/atlas-cashshop/atlas.com/cashshop/character/builder.go:123` (`id == 0`); `services/atlas-consumables/atlas.com/consumables/character/builder.go:127`; `services/atlas-dragons/atlas.com/dragons/character/builder.go:28`; `services/atlas-login/atlas.com/login/character/builder.go:96`; `services/atlas-messages/atlas.com/messages/character/builder.go:119`; `services/atlas-npc-shops/atlas.com/npc/character/builder.go:126`; `services/atlas-pets/atlas.com/pets/character/builder.go:121`; `services/atlas-query-aggregator/atlas.com/query-aggregator/character/builder.go:136`; `services/atlas-character/atlas.com/character/character/builder.go:65-68` (`Builder`, `accountId`/`name`) and `:352-355` (`modelBuilder`, the derived `accountId != 0 → name != ""` invariant, tested by `character/model_test.go:TestBuildErrorsWhenAccountIdSetWithoutName`). The round-2 open item against `modelBuilder`/`NewEmptyBuilder()` is closed per the controller ruling in `builder-validation.md` and independently confirmed here at `builder.go:340-359`. |
| DOM-01 call-site propagation | Every `.Build()` call site on a character builder binds and either propagates or handles the error | PASS | Production: `services/atlas-character/atlas.com/character/character/provider.go:59-88` (`modelFromEntity`, returns `(Model, error)` directly); `.../character/rest.go:127-158` (`Extract`, same); `.../kafka/consumer/character/consumer.go:373-399` (`handleCreateCharacter`, logs full saga-correlation fields and returns on error); `.../character/processor.go:277-289` (`SkillModelDecorator`, logs via `p.l.WithError(err)` and falls back to `m`, per the decorator ruling — error is unreachable by construction). Cross-service: `services/atlas-login/atlas.com/login/character/processor.go:104-155` (`decorateRankings`/`MergeRankings`, now `([]Model, error)`, propagated through `GetForWorld:89-98`). Tests: every new/edited `_test.go` binds `err` and calls `t.Fatalf`/`require.NoError`, e.g. `services/atlas-messages/atlas.com/messages/command/character/commands_test.go:20-30`. No `m, _ := ...Build()` or `_, _ :=` found on any character builder (`grep -n "Build()" | grep -E "_, _|, _ ="` empty across the range). |
| DOM-02/03 | `ToEntity`/`Make` in `entity.go` | N/A | No `character` package in scope has an `entity.go` (grep empty for all ten dirs). |
| DOM-04/05 | `Transform`/`TransformSlice` in `rest.go` | PASS | Every touched `rest.go` still defines `Transform(Model) (RestModel, error)` unchanged in shape, e.g. `services/atlas-pets/atlas.com/pets/character/rest.go:108-115` (only the field list grew by `SpawnPoint`). |
| DOM-06 | Processor constructor takes `logrus.FieldLogger` | PASS | All ten `NewProcessor(l logrus.FieldLogger, ...)` signatures unchanged, e.g. `services/atlas-query-aggregator/atlas.com/query-aggregator/character/processor.go:27`. |
| DOM-11 | Providers evaluate lazily | PASS (unaffected) / N/A (8 of 10, no `provider.go`) | `services/atlas-character/atlas.com/character/character/provider.go:59-88` still returns a `database.Query`-backed lazy provider; the diff only adds error binding on the `Build()` call inside `modelFromEntity`, not a new eager read. |
| DOM-17/18/19 | Status mapping / JSON:API interface / flat request models | PASS (unaffected) | No handler status-mapping code touched; `RestModel` shapes (`GetName`/`GetID`/`SetID`) unchanged except the added `SpawnPoint` field, e.g. `services/atlas-pets/atlas.com/pets/character/rest.go:20-40`. |
| DOM-21 | No redeclaration of a shared type | N/A | Diff adds only `spawnPoint uint32` fields and getters/setters — no new type or const block. |
| DOM-24 | Producer stub installed for tests reaching an emit path | Non-blocking, unchanged from round 2 | `services/atlas-cashshop/atlas.com/cashshop/coupon/{concurrency_test.go,processor_test.go}`, `services/atlas-dragons/atlas.com/dragons/dragon/processor_test.go`, `services/atlas-pets/atlas.com/pets/pet/processor_test.go` reach `AndEmit`/`producer.ProviderImpl`; this range's edits in those files are confined to `Build()` call-site error handling (see diffs at those paths), not producer wiring. Stub presence/absence is unchanged by this branch — see "Not evaluable." |
| DOM-27 | 503 not bare 500 | N/A | No handler in this range writes `http.StatusInternalServerError` (no `resource.go` touched). |
| DOM-28 | Fallible enrichment/decorator paths degrade loudly | PASS | `services/atlas-cashshop/atlas.com/cashshop/character/processor.go:50-58` (`InventoryDecorator`) — `p.l.WithError(err).Errorf(...)` before falling back, round-2 FAIL now cleared. `services/atlas-npc-shops/atlas.com/npc/character/processor.go:72-80` — same, cleared. `services/atlas-query-aggregator/atlas.com/query-aggregator/character/processor.go:48-66` (`InventoryDecorator` and `GuildDecorator`) — both log via `p.l.WithError(err).Errorf(...)`, cleared. `services/atlas-messages/atlas.com/messages/character/model.go:237-243` (`SetSkills`) — now carries the explanatory unreachability comment matching its siblings, cleared. `services/atlas-login/atlas.com/login/character/processor.go:183-195` — `model.ErrDecorator` + `degrade.Observe`, unchanged, still compliant. `services/atlas-character/atlas.com/character/character/processor.go:277-289` (`SkillModelDecorator`) — logs via `p.l.WithError(err).Errorf`, compliant. Per the controller's ruling #1, the local `Build()`-rebuild fallbacks in `services/atlas-consumables/atlas.com/consumables/character/model.go:275-310` and `services/atlas-pets/atlas.com/pets/character/model.go:254-262` (no logger in scope at the `Model`-method level, comment-only) are governed by the branch's log-or-comment convention, not the fetch-remote-data reading of DOM-28, and both carry the required unreachability comment — PASS under that ruling. |
| DOM-33 | Mock kept in sync with interface change | PASS | `services/atlas-query-aggregator/atlas.com/query-aggregator/character/mock/processor.go:19-21` — `GetById`'s stub updated to `character.NewBuilder().SetId(characterId).Build()` (two-value return) in the same diff as the interface's `Build()` signature change. No other `Processor`/`Provider`/`Administrator` interface in scope gained, lost, or re-signed a method in this range — `SetInventory`/`SetSkills`/`SetGuild` are plain `Model` methods, not interface methods, so DOM-33's own trigger does not reach them. |
| DOM-20 | Table-driven tests | PASS | Multi-case new tests are table-driven, e.g. `services/atlas-channel/atlas.com/channel/socket/writer/character_data_test.go:58-85` (`TestBuildCharacterData_SpawnPoint`, `tests := []struct{...}` + `t.Run`), `services/atlas-login/atlas.com/login/socket/writer/character_list_test.go:19-45` (same shape). The new single-assertion `builder_test.go` invariant tests across all nine services (e.g. `services/atlas-cashshop/atlas.com/cashshop/character/builder_test.go:8-13`) are not table-driven, but they match `testing-guide.md:21-27`'s own canonical worked example for exactly this test shape (`TestBuilderValidation` — a bare `_, err := NewBuilder()...Build(); require.Error(t, err)`), so DOM-20's generic table-driven preference does not override the guideline's own named exception for this pattern. |
| DOM-25 | Client wire codes resolved from tenant table | N/A (own trigger) | `services/atlas-channel/atlas.com/channel/socket/writer/character_data.go:47` and `services/atlas-login/atlas.com/login/socket/writer/character_list.go:53-54` cast `byte(c.SpawnPoint())` — a portal-index data value, not a dispatcher mode/sub-op/fail-reason code resolved via a client lookup switch. DOM-25's own trigger (client codes flowing through a lookup switch) does not fire on a plain numeric field narrowing. |

## Security Review

N/A — SEC-* trigger did not fire; none of the ten services in scope handle
authentication, authorization, tokens, redirects, or secrets in the touched
files.

## Not evaluable from the diff

- DOM-24 (producer-stub wiring) for `services/atlas-cashshop/atlas.com/cashshop/coupon/{concurrency_test.go,processor_test.go}`, `services/atlas-dragons/atlas.com/dragons/dragon/processor_test.go`, `services/atlas-pets/atlas.com/pets/pet/processor_test.go`: these packages reach an emit path, and this range's edits touch only `Build()` call-site error handling within them. Whether a `producertest.InstallNoop()` or per-test `WithProducer(...)` stub is actually wired lives in each package's `TestMain`/`NewProcessor` construction outside the changed hunks — unchanged carry-forward from round 2, not newly introduced or newly resolved by this range.
- Whether `libs/atlas-model`'s `model.ErrDecorator`/`degrade.Observe` contract at `services/atlas-login/atlas.com/login/character/processor.go:183-195` matches the library's current source was not re-read in this round; the existing passing test (`TestInventoryDecoratorDegradesLoudly`) is corroborating but not the file:line citation the evidence bar requires for the library side.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- DOM-24: producer-stub wiring in `atlas-cashshop/coupon`, `atlas-dragons/dragon`, `atlas-pets/pet` test packages was not re-verified this round (unchanged from round 2, not touched by this branch's edits).
