# Decorator Audit — task-168 (FR-5.3 / FR-5.4)

Method: `grep -rn 'model\.Decorator\[' services/ --include='*.go'` filtered to
implementation/declaration sites (`_test.go`, `/mock/`, variadic
`decorators ...model.Decorator`, `[]model.Decorator`, and `model.Decorate(` call
sites excluded). Each returned body was read and classified. A second sweep
(`grep 'func '` for any `Decorator[` return type not spelled `model.Decorator`)
confirmed no alternately-imported decorator implementations exist — the only
other `Decorator[` hits under `func` are the `InitConsumers` variadic
`model2.Decorator[consumer.Config]` parameters (Kafka consumer config, not model
enrichment decorators).

Decorator universe fleet-wide: **2 implementations** (both listed below). The
grep also surfaced their interface-method declarations at
`login/character/processor.go:24` and `skills/skill/processor.go:52`; those are
signatures, not bodies, and share the disposition of their implementation.

## model.Decorator implementations

| service | file:line | decorator | fetch kind | silent today? | disposition |
|---|---|---|---|---|---|
| atlas-login | character/processor.go:109 | InventoryDecorator | REST (inventory via `inventory.ProcessorImpl.GetByCharacterId`) | was yes | fixed-in-task (task 10) — now `model.ErrDecorator` + `degrade.Observe(p.l, "login.character.inventory", m.Id(), err)` |
| atlas-skills | skill/processor.go:247 | CooldownDecorator(characterId uint32) | registry read (`GetRegistry().Get(p.ctx, characterId, m.Id())`) | was yes | fixed (task 12) — now `model.ErrDecorator` + `degrade.Observe(p.l, "skills.skill.cooldown", characterId, err)` |

Notes on CooldownDecorator fix:
- Entity id passed to `Observe` is `characterId` (the owning character in
  closure scope), not `m.Id()` (the skill id) — a per-character cooldown's
  meaningful subject is the character. `Observe`'s signature is unchanged.
- The original silently discarded `CloneModel(m).SetCooldownExpiresAt(ct).Build()`'s
  error via `_`. The fix now propagates that build error too (a build failure is
  also a degradation), so both the registry-fetch error and the model-build
  error route through `degrade.Observe`.
- Component string is exactly `"skills.skill.cooldown"` (`<service>.<package>.<what-was-fetched>`).

## Non-decorator silent-degrade shapes on the character-select path (FR-5.4)

Traced atlas-login char-select flow: `character_view_all.go`,
`character_list_world.go`, `character_view_all_selected.go`,
`character_view_all_selected_pic.go`,
`character_view_all_selected_pic_register.go`. Each calls
`GetForWorld(InventoryDecorator())(...)` or `GetById(InventoryDecorator())(...)`.
Every handler-level fetch error is logged at `Errorf` level and either aborts or
(view-all) logs-and-continues with the failing world's list left empty — none
silently rebuild a character entry from partial data. The decorator's own
enrichment failure is now loud inside `InventoryDecorator` (Observe). Result:
**zero silent-degrade findings** on the char-select path.

| service | file:line | shape | silent today? | disposition |
|---|---|---|---|---|
| atlas-login | socket/handler/character_view_all.go:38-44 | `GetForWorld(InventoryDecorator())` per world; on err logs `Errorf` (:40) and continues with empty `cs` for that world | no | not-a-silent-degrade — error logged at `Errorf`; partial (empty) list is deliberate per-world isolation, loudly logged |
| atlas-login | socket/handler/character_list_world.go:48-52 | `GetForWorld(InventoryDecorator())`; on err logs `Errorf` (:50) and `return`s | no | not-a-silent-degrade — aborts + logs |
| atlas-login | socket/handler/character_view_all_selected.go:24-29 | `GetById(InventoryDecorator())`; on err logs `Errorf` (:26) and `return`s | no | not-a-silent-degrade — aborts + logs |
| atlas-login | socket/handler/character_view_all_selected_pic.go:27-32 | `GetById(InventoryDecorator())`; on err logs `Errorf` (:29) and `return`s | no | not-a-silent-degrade — aborts + logs |
| atlas-login | socket/handler/character_view_all_selected_pic_register.go:25-30 | `GetById(InventoryDecorator())`; on err logs `Errorf` (:27) and `return`s | no | not-a-silent-degrade — aborts + logs |

Out-of-scope observation (not a decorator/enrichment silent-degrade, recorded for
completeness): `channel.GetRandomInWorld` at
`character_view_all_selected*.go` (e.g. selected.go:58, pic.go:92,
pic_register.go:66) discards its `err` and immediately dereferences `ch`. This is
a required-fetch-unchecked bug, not a "build character entry from partial data"
silent degradation, so it is outside FR-5.4's decorator-audit scope. Flagged here
so it is not lost.

## Services modified by this audit (drives Task 14 bake list)

- services/atlas-skills (fix: CooldownDecorator — now `model.ErrDecorator` + `degrade.Observe`; go.mod pulls prometheus transitively via atlas-rest/degrade)

atlas-login was NOT modified by this audit: its only decorator
(InventoryDecorator) was already made loud in Task 10, and its char-select path
has no remaining silent-degrade shape.
