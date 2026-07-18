# task-159 Context — Maple Warrior All-Stats Bonus

Companion to `plan.md`. Key files, decisions, and dependencies for executors.

## Scope

One service: **atlas-effective-stats** (module `atlas-effective-stats` at
`services/atlas-effective-stats/atlas.com/effective-stats/`). No wire changes
(REST/Kafka payloads unchanged; bonus JSON gains one additive field).

## Key Files

| File | Role | Change |
|---|---|---|
| `stat/model.go` | `Bonus` value type, constructors, JSON, buff/statup mapping | +`basePercent` field, `BasePercent()`, `NewBasePercentBonus`, `WithSource`; +`BonusesForBuffChange`; delete `MapBuffStatType` (keep `MapStatupType`) |
| `character/model.go:269` | `ComputeEffectiveStats` | +base-percent accumulator: `effective = floor((base + flat + Σ floor(base×pct/100)) × (1+mult))` |
| `character/processor.go:200-229` | `AddBuffBonuses` / `AddPassiveBonuses` re-sourcing | `stat.NewFullBonus(...)` → `b.WithSource(source)` (dimension-preserving) |
| `kafka/consumer/buff/consumer.go:50-67` | live-apply path | migrate to `BonusesForBuffChange("", type, amount)` |
| `character/initializer.go:174-202` | login-time path (`fetchBuffBonuses`) | migrate to `BonusesForBuffChange("buff:<id>", type, amount)` |
| `docs/domain.md:85` | service doc | rewrite `MapBuffStatType` paragraph for the new API |
| `stat/model_test.go`, `character/model_test.go`, `character/processor_test.go` | tests | see plan Tasks 1-5 |

## Decisions Already Made (do not relitigate)

1. **Bonus-constructing API** (`BonusesForBuffChange(source, buffType, amount) []Bonus`),
   not a mapping-entries API — the amount→bonus conversion lives in one place so the
   live-apply and initializer paths cannot drift (design §2.1, FR-11 by construction).
2. **Additive `basePercent int32` field** on `Bonus`, not a kind-discriminator rebuild.
   Raw integer percent from the wire (10 = +10%), never pre-divided to float —
   integer division reproduces the client's truncation exactly (design §2.2).
3. **Truncation per bonus inside the accumulation loop** (`base * pct / 100` Go int
   division per bonus), never on a summed rate (design §2.3).
4. **Basis = `baseValues` only** (from `m.baseStats`). Equipment bonuses stay in
   `flatBonuses`; equipment snapshot loop (`snap.bonuses`, Amount-only) untouched.
5. **`WithSource` at the model**, so future `Bonus` dimensions survive processor
   re-sourcing automatically (design §2.4). `StoreEquipmentBonuses`' re-sourcing loop
   is deliberately NOT migrated.
6. **`MapBuffStatType` deleted, not aliased** (FR-5). Deletion happens in Task 4
   (after consumers migrate) so every intermediate task compiles.
7. **JSON backward compatibility**: absent `basePercent` decodes to 0; no data
   migration — active-MW characters self-heal on next buff event or login (PRD §6).
8. Out of scope: HYPER_BODY semantics, `MapStatupType`, atlas-data/atlas-buffs
   emitters, client display (PRD §2 non-goals).

## Ground Truth

Client semantics IDA-verified at both range ends (PRD §4.1): v83
`BasicStat::SetFrom` @0x77ec9f, v95 @0x732ba0 — `nSTR += rate * base.nSTR / 100`
per primary stat, raw base only, HP/MP untouched. Producer chain verified: atlas-data
`skill/reader.go:315-318` emits `MAPLE_WARRIOR` with amount = X% for all 14 variants.

## Dependencies / Test Infrastructure

- Task order: 1 → 2 → 3 → 4 → 5 → 6 (each compiles and passes green independently;
  2 and 3 both depend only on 1 but the plan orders them serially).
- Processor tests use `setupProcessorTest(t)` (miniredis + `InitRegistry`,
  `character/processor_test.go:20-40`). Model tests use `createTestTenant()` +
  immutable `With*` builders. No `*_testhelpers.go` (project rule).
- `stat.Bonus` is comparable (all fields comparable) — tests compare with `!=`.
- `stat.NewBase(strength, dexterity, luck, intelligence, maxHp, maxMp)` — note
  LUK before INT; lifecycle test expectations depend on this order.
- Registry upsert keys bonuses on `(source, statType)` (`Model.WithBonus`,
  `character/model.go:147`): the 4 MW bonuses coexist (distinct types), re-apply
  is idempotent, `RemoveBuffBonuses` → `WithoutBonusesBySource` drops all 4.

## Verification Gate (Task 6)

`go test -race ./...`, `go vet ./...`, `go build ./...` in the module;
`docker buildx bake atlas-effective-stats` from the worktree root (mandatory);
`tools/redis-key-guard.sh` from the worktree root; then
`superpowers:requesting-code-review` before any PR.
