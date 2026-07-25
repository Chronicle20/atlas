# Backend Audit (DOM/SEC) — task-126 AP/SP Reset Items

- **Scope:** Whole-branch review of `38d4d0ba2..3e30c212c` (task-126-ap-sp-reset-items)
- **Guidelines Source:** backend-dev-guidelines skill
- **Date:** 2026-07-02
- **Build / Tests / Vet:** PASS (controller ran all gates green; not re-run here)
- **Overall:** PASS (no blocking DOM/SEC failures; three Minors, all non-blocking)

## Character of the change

This is a cross-service **command / saga / Kafka** feature, not a REST domain feature.
No new REST domain package (`model.go` + `resource.go` + `rest.go`) was introduced, so the
REST-specific DOM rows (DOM-04/05 Transform, DOM-08 RegisterInputHandler, DOM-17/18/19
JSON:API, etc.) are **N/A**. The new/changed units are:

- `libs/atlas-constants/job/advancement.go`, `skill/point_reset.go` — pure shared helpers.
- `libs/atlas-packet/cash/serverbound/item_use_point_reset.go` — packet codec + test.
- `libs/atlas-saga/{model,payloads,unmarshal}.go` — new actions/type + payloads.
- `atlas-character`, `atlas-skills` — new processor methods on existing domain packages.
- `atlas-channel/pointreset` — support package (validation + player messages), no `model.go`.
- `atlas-saga-orchestrator` — handlers, compensator, consumers.

## DOM checklist results (applicable rows)

| ID | Check | Status | Evidence |
|----|-------|--------|----------|
| DOM-06 | Processor accepts `FieldLogger` | PASS | skills `skill/processor.go:87`, macro `macro/processor.go:38`, character processor unchanged — all `logrus.FieldLogger` |
| DOM-11/immutability | Immutable model + Builder for mutations | PASS | `TransferAP` computes running values then one `dynamicUpdate(tx)(mods...)` — `character/processor.go:2064-2115`; `Build()` now copies `hpMpUsed` — `character/model.go:428` (commit be6747bc8 fixes a real pre-existing drop-to-zero bug) |
| Buffer/Emit | Side-effects via `message.Buffer` + `message.Emit` | PASS | `TransferAPAndEmit` `character/processor.go:1925-1929`; `TransferSpAndEmit` `skills/skill/processor.go:314-318`; all producers via `producer.SingleMessageProvider` |
| Tenant context | `tenant.MustFromContext(ctx)`, GORM tenant callbacks | PASS | `skills/skill/processor.go:99`; tests use shared `databasetest.NewInMemoryTenantDB` (`transfer_ap_test.go:35`) which registers tenant callbacks |
| Transaction correctness | gorm-native tx where atomicity spans tables | PASS | `TransferSp` uses `p.db.Transaction(...)` (NOT the no-op `database.ExecuteTransaction`) so skill rows + macro rows commit/rollback together — `skills/skill/processor.go:355`, documented at 320-326. Nested `sp.Update(mb)`/`mp.Update(mb)` run against `tx` because `ExecuteTransaction`'s no-op runs the callback against the passed db. Correct and required. |
| DOM-21 | atlas-constants reuse (no duplicate shared types/helpers) | PASS | New shared predicates `job.Advancement` (`job/advancement.go:8`) and `skill.IsPointResetExcluded` (`skill/point_reset.go:8`) live in `libs/atlas-constants` and are reused by BOTH `atlas-channel/pointreset/model.go:150-151,147` and `atlas-skills/skill/processor.go:346,349`. No service redeclares a shared type. |
| Error handling | No silent swallow of meaningful errors | PASS | Rejections intentionally return `nil` + buffer a typed ERROR status event (documented `character/processor.go:1931-1946`); consumer handlers log processor errors (`character consumer.go:414`, `skills consumer.go:115`) |
| DOM-24 | Kafka producer stubbed in emitting tests | PASS | Tests exercise the **buffer-form inner** methods `TransferAP(mb)` / `TransferSp(mb)` with `message.NewBuffer()` (`transfer_ap_test.go:75`, `transfer_sp_test.go:185`) — no `AndEmit`, no `producer.ProviderImpl` path, so no unstubbed 42s hang |
| Table-driven tests | PASS | `transfer_sp_test.go` and `transfer_ap_test.go` are per-case `t.Run`/table style with a Builder-based fixture (`newTransferApFixture`) — no `*_testhelpers.go` |

## SEC review

Not an auth/token service, so SEC-01..04 are N/A. Security-relevant design note (PASS):
authoritative validation is **server-side and not trusted from the channel** — `atlas-skills.TransferSp`
re-validates job tree / exclusion / tier / level (`skills/skill/processor.go:340-377`) and
`atlas-character.TransferAP` re-validates floors/caps/pool-minimums (`character/processor.go:1958-2061`).
`JobId`/`ItemTier`/`TargetMaxLevel` ride along from atlas-channel (trusted server-to-server caller);
the channel's `pointreset` pre-validation is a cheap UX gate only, explicitly not the source of truth
(`pointreset/model.go:103-105`).

## Triage of the three known Minors

1. **Raw `job.Id(100)` literals in `character/point_reset.go`** — CONFIRMED, non-blocking.
   Not a DOM-21 violation: the code uses the shared `job.Id` type and shared `job.Is` helper and
   defines **no** duplicate constant; it merely spells branch-root values as literals instead of the
   existing named constants (all verified present: `job.WarriorId=100`, `FighterId=110`, `PageId=120`,
   `SpearmanId=130`, `MagicianId=200`, `FirePoisonWizardId=210`, `IceLightningWizardId=220`,
   `ClericId=230`, `BowmanId=300`, `HunterId=310`, `CrossbowmanId=320`, `RogueId=400`, `AssassinId=410`,
   `BanditId=420`, `PirateId=500`, `BrawlerId=510`, `GunslingerId=520` — `libs/atlas-constants/job/constants.go`).
   Values verified correct. Same literal style at `transfer_ap_test.go:66`. Readability nit only.

2. **Log-level inconsistency on saga-failing rejection** — CONFIRMED, non-blocking.
   `handleCharacterApTransferErrorEvent` (`saga-orchestrator/kafka/consumer/character/consumer.go:200`)
   and `handleSkillErrorEvent` (`.../consumer/skill/consumer.go:113`) log at Debug, while sibling
   `handleCharacterMesoErrorEvent` (`consumer/character/consumer.go:175`) logs at Error. Defensible —
   a fully-compensated user-facing validation reject is arguably not an Error-level system fault — but
   inconsistent. Cosmetic.

3. **Per-service mirroring of kafka consts / payload aliases** — CONFIRMED as the established Atlas DDD
   boundary pattern, NOT a DOM violation. Channel `saga/model.go` re-exports via `= sharedsaga.X` aliases;
   orchestrator `saga/model.go` does the same. Not flagged.

## New observations (Minor / informational)

- **Swallowed error on saga dispatch** — `character_cash_item_use_point_reset.go:91,127` use
  `_ = saga.NewProcessor(l, ctx).Create(...)`. If saga creation fails, the client UI is never
  re-enabled and the player gets no feedback. However this exactly mirrors the established sibling
  dispatch `character_cash_item_use.go:99`, so it is consistent, not a regression. Non-blocking.

## Verdict

**PASS.** No Critical, Important, or blocking DOM/SEC findings. Immutable-model + Builder,
message.Buffer/Emit, tenant-context, the gorm-native `TransferSp` transaction, and DOM-21
shared-constant reuse are all correctly implemented. Only three cosmetic Minors, none of which
block the branch.
