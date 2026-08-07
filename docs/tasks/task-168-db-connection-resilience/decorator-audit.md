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
| atlas-skills | skill/processor.go:248 | CooldownDecorator(characterId uint32) | registry read (`GetRegistry().Get(p.ctx, characterId, m.Id())`) | was yes | fixed (task 12) — now `model.ErrDecorator` + `degrade.Observe(p.l, "skills.skill.cooldown", characterId, err)` |

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
Every character/world/account fetch error is logged at `Errorf` level and either
aborts or (view-all) logs-and-continues with the failing world's list left empty
— none silently rebuild a character entry from partial data. The decorator's own
enrichment failure is now loud inside `InventoryDecorator` (Observe).

However, the three `*_selected*` handlers each dropped the error from
`channel.GetRandomInWorld(p.WorldId())` and let the zero-value channel model flow
into `SetChannelId(ch.ChannelId())` (→ 0) and `UpdateState(..., ChannelSelect{
IPAddress: "", Port: 0, ...})` — routing the player to a dead channel 0 / empty
IP with no log, no metric, no abort. This is exactly FR-5.4's shape (channel
service/DB down → silent dead-channel route), so it IS in scope. Result:
**3 silent sites found on the char-select path, all fixed in this task** (loud
abort mirroring the sibling `world.GetById`/`character.GetById` handling in the
same files: `l.WithError(err).Errorf(...)` + `// TODO issue error` + `return`).

| service | file:line | shape | silent today? | disposition |
|---|---|---|---|---|
| atlas-login | socket/handler/character_view_all.go:38-44 | `GetForWorld(InventoryDecorator())` per world; on err logs `Errorf` (:40) and continues with empty `cs` for that world | no | not-a-silent-degrade — error logged at `Errorf`; partial (empty) list is deliberate per-world isolation, loudly logged |
| atlas-login | socket/handler/character_list_world.go:48-52 | `GetForWorld(InventoryDecorator())`; on err logs `Errorf` (:50) and `return`s | no | not-a-silent-degrade — aborts + logs |
| atlas-login | socket/handler/character_view_all_selected.go:24-29 | `GetById(InventoryDecorator())`; on err logs `Errorf` (:26) and `return`s | no | not-a-silent-degrade — aborts + logs |
| atlas-login | socket/handler/character_view_all_selected_pic.go:27-32 | `GetById(InventoryDecorator())`; on err logs `Errorf` (:29) and `return`s | no | not-a-silent-degrade — aborts + logs |
| atlas-login | socket/handler/character_view_all_selected_pic_register.go:25-30 | `GetById(InventoryDecorator())`; on err logs `Errorf` (:27) and `return`s | no | not-a-silent-degrade — aborts + logs |
| atlas-login | socket/handler/character_view_all_selected.go:58 | channel fetch error dropped → routes to channel 0/empty IP | yes | fixed-in-task (loud abort: Errorf + return) |
| atlas-login | socket/handler/character_view_all_selected_pic.go:92 | channel fetch error dropped → routes to channel 0/empty IP | yes | fixed-in-task (loud abort: Errorf + return) |
| atlas-login | socket/handler/character_view_all_selected_pic_register.go:66 | channel fetch error dropped → routes to channel 0/empty IP | yes | fixed-in-task (loud abort: Errorf + return) |

The three `GetRandomInWorld` sites are a loud ABORT (not `degrade.Observe`):
routing to channel 0 / empty IP is broken, not a graceful partial, so they mirror
the sibling `world.GetById`/`character.GetById` error handling already present in
these same handlers — `l.WithError(err).Errorf(...)` + `// TODO issue error` +
`return`.

## Services modified by this audit (drives Task 14 bake list)

- services/atlas-skills (fix: CooldownDecorator — now `model.ErrDecorator` + `degrade.Observe`; go.mod pulls prometheus transitively via atlas-rest/degrade)
- services/atlas-login (fix: 3 char-select GetRandomInWorld silent error-drops — loud abort)

atlas-login's decorator (InventoryDecorator) was already made loud in Task 10 and
was not re-touched; the login change in this audit is the FR-5.4 fix to the three
`*_selected*` handlers' dropped `GetRandomInWorld` errors.
