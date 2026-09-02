# task-277 — Planning context

Companion to `plan.md`. Records what the plan assumes, what the survey resolved, and
where the plan deliberately departs from a default rule.

## The feature in one paragraph

`USE_SOLOMON_ITEM` and `USE_GACHA_EXP` are the two halves of one feature, not two
independent grants. A Writ of Solomon (classification 237) **credits** the client's
`GW_CharacterStat::nTempEXP` counter; the EXP-bar affordance **redeems** the whole
counter into real character EXP and zeroes it. Atlas persists that counter in the
`characters.gachapon_experience` column — a community misnomer kept deliberately, since
renaming it would churn atlas-login, atlas-cashshop and atlas-npc-shops for no gain.
This inverts PRD FR-7, which had the Solomon path calling `AwardExperience` directly;
the design corrected it from client evidence (design §0) and the plan implements the
corrected behaviour. **This is the one PRD amendment the user should be aware of.**

## Key files, by seam

| Seam | Owner | Entry point |
|---|---|---|
| Wire decode | `libs/atlas-packet` | `inventory/serverbound/solomon_item_use.go` (new), `character/serverbound/stored_experience_use.go` (new) |
| Opcode routing | `services/atlas-configurations/seed-data/templates/` | 8 template files |
| Socket dispatch | `atlas-channel` | `socket/handler/character_item_use.go`, `socket/handler/character_stored_experience_use.go` (new) |
| Ownership, reservation, eligibility | `atlas-consumables` | `consumable/solomon.go` (new), routing branch in `consumable/processor.go:300-355` |
| Counter write, EXP award | `atlas-character` | `character/processor.go` (two new `...AndEmit`/`(mb)` pairs) |
| WZ field exposure | `atlas-data` | `consumable/reader.go`, `consumable/rest.go` |

## Decisions the survey settled (do not re-derive)

- **`AwardExperience` nesting is safe.** `libs/atlas-database/transaction.go:9-14` —
  `ExecuteTransaction` short-circuits to `fn(db)` when `isTransaction(db)` (`:20-26`,
  tests whether `db.Statement.ConnPool` is a `gorm.TxCommitter`). So
  `p.WithTransaction(tx).AwardExperience(buf)(...)` joins the outer transaction rather
  than committing independently. Design Risk #1 and Alternative-C's "explicit
  implementation checkpoint" are both **closed**; no hoisting of the EXP arithmetic.
- **`atlas-consumables` cannot call `EnableActions`.** That helper is
  `atlas-channel`-package-local (`session/enable_actions.go`, needs a `session.Model`
  and a `writer.Producer`). The design's "cancel reservation + EnableActions + log"
  is delivered by `ConsumeError` (`consumable/processor.go:418-433`) emitting an
  `ERROR` event that `atlas-channel`'s unrecognized-type fallback arm
  (`kafka/consumer/consumable/consumer.go:141-148`) already turns into the empty
  `STAT_CHANGED` unstick. **No new `ErrorType` const** — adding a recognized type
  would fall out of that fallback and wedge the client.
- **Three things the design lists as work are already done.** `atlas-consumables`'
  `character.Model` already carries `gachaponExperience` with an accessor and REST
  decode (`character/model.go:41,209-211`; `character/rest.go:19,123,157`). The
  channel snapshot registry already maps `stat.TypeGachaponExperience` →
  `"gachapon_experience"` and applies it (`character/snapshot/registry.go:264,307`).
  `stat.TypeGachaponExperience` is already encoded as a 4-byte stat
  (`libs/atlas-packet/stat/clientbound/changed.go:86,137`) and `GACHAPON_EXPERIENCE`
  is already in every template's stat-index table. The plan consumes all three and
  changes none.
- **`SetGachaponExperience` is ambiguous in the design.** The *builder* method exists
  (`character/builder.go:321-324`) and the administrator's reflection case exists
  (`administrator.go:101-102`), but the `EntityUpdateFunction` setter does **not** —
  Task 8 adds it.
- **`RequestItemConsume` has two different signatures.** Channel-side
  (`atlas-channel/consumable/processor.go:43`) is
  `(f field.Model, characterId character.Id, itemId item.Id, source slot.Position, quantity int16, updateTime uint32)`;
  consumables-side (`atlas-consumables/consumable/processor.go:300`) is
  `(c channel.Model, characterId uint32, slot int16, itemId item2.Id, quantity int16, petId uint64)`
  — **slot and itemId are swapped, and the last arg is `petId`, not the tick**. The
  design's pseudocode uses the channel order throughout; an implementer following it
  literally into `atlas-consumables` would mis-order arguments. Task 10 calls this out
  inline.
- **Classification 237 is unnamed today** — the gap between `ClassificationBullet = 233`
  and `ClassificationConsumableMonsterCard = 238` in `libs/atlas-constants/item/constants.go`.
- **Packet-name convention:** the registry `packet:` value and the marker `packet=`
  value must match byte-for-byte. `inventory/serverbound` names carry an `Inventory`
  prefix (`inventory/serverbound/InventorySummonBagItemUse` for Go type
  `SummonBagItemUse`); `character/serverbound` names do not
  (`character/serverbound/ItemCancel`). Hence
  `inventory/serverbound/InventorySolomonItemUse` and
  `character/serverbound/StoredExperienceUse`.
- **All eight in-scope versions are already in `pt.Variants`** (`libs/atlas-packet/test/context.go:18-41`),
  so the round-trip loop covers every column with no test-harness change.

## Task ordering and dependencies

```
1 (packet Solomon) ─┐
2 (packet redeem)  ─┼─> 3 (registry) ─> 4 (evidence + matrix)
                    │
5 (data fields) ────┴─> 6 (consumables data) ─┐
7 (const 237 + credit seam) ──────────────────┼─> 10 (ConsumeSolomon) ─┐
8 (character CREDIT) ─────────────────────────┘                        │
9 (character REDEEM) ──────────────────────────────────> 11 (channel) ─┼─> 12 (templates) ─> 13 (docs) ─> 14 (gate)
```

Tasks 1/2, 5, and 7/8 are independent of one another and can run in parallel. Task 11
depends on Tasks 1, 2 and 9. Task 12 depends on Tasks 1, 2 and 11.

## Deliberately oversized tasks (F4)

- **Task 3 (10 files).** All ten receive the same one- or two-line declarative YAML
  edit; eight are the `packet:` key alone. There is no build step and no logic. Splitting
  it would produce ten commits whose only relationship is "the same line, elsewhere".
- **Task 12 (8 files).** One identical two-entry JSON insertion per template, differing
  only in `opCode`. The acceptance criterion is a single grep across all eleven
  templates, which a split would fragment.
- **Task 11 (7 files, 1 service).** Four of the seven — the kafka message const, the
  producer, the processor interface entry, the processor method — are the four halves
  of one Kafka producer seam and do not compile apart.

Neither codemod candidate is worth a rewriter: Tasks 3 and 12 are data-file edits, not
AST transformations, so `docs/codemod-vs-agents.md`'s break-even does not apply.

## Risks carried forward from design §7

1. **Un-reingested tenants** serve neither `spec/exp` nor `info/maxLevel`. The chosen
   failure is "reject and return the Writ", never "destroy for zero EXP" — Task 10's
   test table pins both the absent and the negative case. Task 13 records the
   operational re-ingest follow-up in `docs/TODO.md`.
2. **`maxLevel` presence on `2370000`–`2370012` is unconfirmed per item.** The client
   reads `info/maxLevel`, but the repo's WZ enumeration only recorded the `spec` side.
   The plan treats zero as "no upper bound", so an absent field degrades to "no gate"
   rather than "reject everything". Worth confirming against tenant WZ during
   implementation, but nothing blocks on it.
3. **Two `STAT_CHANGED` events per redeem** (one from `AwardExperience` carrying
   `EXPERIENCE`, one added by the redeem path carrying `GACHAPON_EXPERIENCE`). This is
   deliberate — it is what lets `AwardExperience` be reused verbatim, which is what
   makes FR-14 (shared level-cap and overflow behaviour, including the `AWARD_LEVEL`
   command) fall out for free.

## What this plan does NOT do

- No DB migration. `characters.gachapon_experience` already exists
  (`character/entity.go:40`).
- No new service, no monster-kill EXP diversion, no gachapon reward-pool wiring. Design
  §0 resolved FR-15's accrual source to the Writ itself, so FR-18's escalation never
  fired.
- No broader `spec` parse in `atlas-data` beyond `exp` and `maxLevel` — the PRD's
  non-goals keep that a separate task.
- No wire change to any already-verified version, and no change to `gms_v12`,
  `gms_v48` or `gms_v61` beyond their positive-absence evidence entries.
