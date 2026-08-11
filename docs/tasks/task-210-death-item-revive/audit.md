## Backend guidelines

- **Scope:** changed Go packages on `task-210-death-item-revive`, range `eca47150f..HEAD`:
  - `libs/atlas-packet/character/serverbound/use_death_item.go` (+ test)
  - `libs/atlas-packet/character/clientbound/show_upgrade_tomb_effect.go` (+ test)
  - `services/atlas-channel/atlas.com/channel/socket/handler/use_death_item.go` (+ test)
  - `services/atlas-channel/atlas.com/channel/respawn/` — `plan.go` (new), `plan_test.go` (new), `processor.go` (reworked)
  - `services/atlas-channel/atlas.com/channel/socket/handler/map_change.go` (call site)
  - `services/atlas-channel/atlas.com/channel/main.go` (registration)
  - `tools/packet-audit/cmd/run.go` (two `candidatesFromFName` cases)
  - `services/atlas-configurations/atlas.com/configurations/socket/corpus_test.go` (count literal)
- **Build/Test:** pre-verified clean by the controller — `go test -race`/`go vet`/`go build` clean in every changed module; all 8 repo-root guards (including `goroutine-guard.sh` and `redis-key-guard.sh`) exit 0; `docker buildx bake` clean for `atlas-channel` and `atlas-configurations`. Not re-run in this pass.
- **Pre-adjudicated design decisions (see `context.md` §6), evaluated on their merits below rather than flagged as oversights:**
  1. `respawn.NewProcessor` takes `wp writer.Producer`; `announceProtectOnDie` (two `session.Announce` calls, no branching) is not covered by fake HTTP/Kafka tests — decision logic was extracted into pure `planRespawn` specifically so it is testable without mocks.
  2. `planRespawn` takes a three-field `mapFacts` struct rather than `map_.Model`, because `data/map.Model` has private fields and no exported constructor.
  3. `expirationDays`'s "days" semantic is documented as unverified against the live client message (design OQ-3), with a code comment explaining how to correct it if wrong.

### Domain / Support Checklist Results

`respawn` classifies as a **support package** (no `model.go`, no `resource.go` — internal business-logic package invoked from a socket handler). It still runs the File Responsibilities checklist. `libs/atlas-packet/character/{serverbound,clientbound}` are wire-codec packages (established sibling pattern across the whole `libs/atlas-packet` tree, outside the REST-domain file-responsibilities table); graded against DOM-21/DOM-25/DOM-26 and general correctness rather than FILE-01..06, which target JSON:API domain packages.

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| FILE-01 | `Processor` interface/impl confined to `processor.go` | PASS | `services/atlas-channel/atlas.com/channel/respawn/processor.go:26` (`type Processor interface`), `:34` (`ProcessorImpl`), `:45` (`NewProcessor`), all `(p *ProcessorImpl)` methods at `:60`, `:94`, `:176` — none in `plan.go`. |
| FILE-06 | No package-named catch-all file bundling ≥2 responsibilities | PASS | `respawn/plan.go` contains only pure helper functions and value types (`mapFacts`, `respawnPlan`, `findWheelOfFortune`, `usesRemaining`, `expirationDays`, `planRespawn`, `respawnSagaSteps`) — no Processor/RestModel/administrator logic. `respawn/processor.go` holds the `Processor`/`ProcessorImpl` plus two module-private helpers (`findProtectiveItem`, `calculateExpLoss`) that are Processor-adjacent pure functions, not a second responsibility class. |
| DOM-06 | Processor accepts `logrus.FieldLogger` | PASS | `respawn/processor.go:45` `func NewProcessor(l logrus.FieldLogger, ctx context.Context, wp writer.Producer) Processor`. |
| DOM-12 | No `os.Getenv()` in handler/processor code | PASS | `grep os.Getenv` over `socket/handler/use_death_item.go` and `respawn/*.go` returns zero matches. |
| DOM-13/14 | No cross-domain orchestration or provider calls from the handler | PASS | `socket/handler/use_death_item.go:43,50` call `character.NewProcessor(...).GetById()` and `channelInventory.NewProcessor(...).GetByCharacterId(...)` — processor calls, not provider calls; `map_change.go:56` calls `respawn.NewProcessor(l, ctx, wp).Respawn(...)`, a processor call. |
| DOM-15 | No direct entity/DB writes in handler | PASS | No `db.Create`/`db.Save`/`db.Delete` in `socket/handler/use_death_item.go`; the relay handler is read-only (session broadcast only). |
| DOM-21 | No redeclaration of `libs/atlas-constants` types | PASS | `respawn/plan.go:11-13` imports `field`, `item`, `_map` from `libs/atlas-constants`; `WheelOfFortuneId`, `SafetyCharmId`, `EasterBasketId`, `ProtectOnDeathId`, `IsWheelOfFortune`, `IsSafetyCharm` are all pre-existing in `libs/atlas-constants/item/death_protection.go` (unmodified by this diff — confirmed via `git diff eca47150f..HEAD -- libs/atlas-constants/item/death_protection.go` producing no output) and consumed, not redeclared, in `use_death_item.go:23-30` (`item.IsWheelOfFortune`) and `respawn/plan.go:50,99,123,126,129`. `field.Model`/`_map.Id` used directly, no local aliases. |
| DOM-21 (minor) | Test literals vs. named constants | MINOR/NON-BLOCKING | `services/atlas-channel/atlas.com/channel/socket/handler/use_death_item_test.go:10,22` use raw literals `5510000`/`5130000` instead of `item.WheelOfFortuneId`/`item.SafetyCharmId`. Not a redeclaration (nothing new is declared), so it does not fail DOM-21 as written, but it is a readability/drift risk in test code that the sibling `plan_test.go` (e.g. `plan_test.go:81` `uint32(item.WheelOfFortuneId)`) avoids. |
| DOM-25 | No hard-coded client-interpreted wire values | PASS | The two new codecs (`use_death_item.go`, `show_upgrade_tomb_effect.go`) carry only plain data fields (`itemId`, `x`, `y`, `characterId`) with no client-side mode/effect byte to resolve. The mode byte for the follow-on protect-on-die announcement is resolved via `atlas_packet.WithResolvedCode("operations", string(CharacterEffectProtectOnDieItemUse), ...)` in `libs/atlas-packet/character/effect_body.go:136,142` — pre-existing code, unmodified by this diff (`git diff eca47150f..HEAD -- libs/atlas-packet/character/effect_body.go` is empty), invoked from `respawn/processor.go:106-107,113-114`. New opcodes (`CharacterUseDeathItemHandle` handler, `CharacterShowUpgradeTombEffect` writer) are seeded in all 8 templates, e.g. `services/atlas-configurations/seed-data/templates/template_gms_72_1.json:405,3315-3316` and `template_gms_92_1.json:444,2212-2213`, and the v92 template additionally adds the two `CharacterEffect`/`CharacterEffectForeign` writer rows needed for the announce path (`template_gms_92_1.json:2228,2298`). |
| DOM-26 | Goroutines spawned via `routine.Go` | PASS | `grep -nE '^\s*go (func\|[A-Za-z_])'` over all seven changed non-test Go files returns zero matches — no bare `go` statements introduced. |
| DOM-24 | Kafka producer stubbed in tests that emit | N/A (no emit paths in changed tests) | `services/atlas-channel/atlas.com/channel/respawn/plan_test.go` and `socket/handler/use_death_item_test.go` exercise only pure functions (`planRespawn`, `usesRemaining`, `expirationDays`, `respawnSagaSteps`, `canShowTombEffect`) — `grep -n "AndEmit\|producertest\|producer.Provider\|message.Emit"` over both files returns zero matches, and neither calls `Respawn(...)` or `announceProtectOnDie` (the only paths that touch `wp writer.Producer`). Consistent with pre-adjudicated decision 1. |
| DOM-20 | Table-driven tests | PASS | `respawn/plan_test.go:136-160` (`TestUsesRemaining`), `:178-198` (`TestExpirationDays`); `socket/handler/use_death_item_test.go:11-23` (`TestCanShowTombEffect`); codec tests use a `variants`/`tests` table plus `pt.Variants` round-trip loop, e.g. `libs/atlas-packet/character/serverbound/use_death_item_test.go:28-56`. |

### Sub-checks specific to the handler relay (`use_death_item.go`)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| — | Owner excluded from broadcast (documented client-echo hazard) | PASS | `socket/handler/use_death_item.go:64-69` uses `_map.NewProcessor(...).ForOtherSessionsInMap(...)`, with a comment explaining the client already plays the effect locally. |
| — | Authorization gate is a pure, unit-tested function | PASS | `canShowTombEffect` (`use_death_item.go:23-31`) is pure and covered by `TestCanShowTombEffect` (`use_death_item_test.go:9-31`), including the "claims another item" and "alive" negative cases. |
| — | No state mutation / no saga creation from the relay handler | PASS | Handler body (`use_death_item.go:37-74`) only reads character/inventory and broadcasts; the wheel is spent later by `respawn.Respawn` via `MAP_CHANGE`, matching the doc comment at `use_death_item.go:33-36` and the `respawnSagaSteps` ordering test (`plan_test.go:266-286`, `TestOneDeathConsumesOneCharge`). |

### Design-decision evaluation

1. **`wp writer.Producer` threaded into `respawn.NewProcessor`, announce path untested.** Accepted on its stated rationale: `announceProtectOnDie` (`respawn/processor.go:94-118`) is two straight-line `session.Announce` calls with no conditional logic beyond the `a == nil` early return (`:96-97`), and the actual decision logic (which item, how many charges, which map) was extracted into `planRespawn`/`findProtectiveItem`/`usesRemaining`/`expirationDays`, all of which are covered by `plan_test.go`. No FAIL — the risk surface left untested is a thin, un-branching wiring layer, not business logic.
2. **`mapFacts` struct instead of `map_.Model`.** Verified: `services/atlas-channel/atlas.com/channel/data/map` model was not modified by this diff, and `mapFactsOf(m map_.Model) mapFacts` (`respawn/plan.go:26-32`) is a narrow, one-directional adapter — it does not leak into `processor.go` in a way that duplicates `map_.Model` responsibilities. Accepted.
3. **`expirationDays` "days" semantic marked unverified (OQ-3).** The code comment at `respawn/plan.go:75-82` explicitly states the reading is "defensible... and not a verified one," names the exact IDA site (`CUser::OnEffect mode-6 arm @0x937e81`), and gives the concrete fix if the live message renders swapped. This satisfies the project's grounding/honesty rule (documented unverified assumption with a stated correction path) rather than being an undocumented gap — no FAIL.

### Summary

**Overall backend-guidelines status for the audited scope: PASS.**

#### Blocking (must fix)
- None.

#### Non-Blocking (should fix)
- DOM-21 (style/minor): `services/atlas-channel/atlas.com/channel/socket/handler/use_death_item_test.go:10,22` — replace raw literals `5510000`/`5130000` with `item.WheelOfFortuneId`/`item.SafetyCharmId` for consistency with `respawn/plan_test.go`'s usage of named constants.
