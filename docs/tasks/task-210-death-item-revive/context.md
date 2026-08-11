# task-210 — Implementation Context

Companion to [`plan.md`](plan.md). Everything below was verified against this
worktree during the planning session; file:line references are to the
`task-210-death-item-revive` branch at plan time.

---

## 1. Where the existing behaviour lives

| Concern | Location |
|---|---|
| Death → respawn entry point | `services/atlas-channel/atlas.com/channel/socket/handler/map_change.go:54` — `c.Hp() == 0` branch calls `respawn.NewProcessor(l, ctx).Respawn(s.Field().Channel(), s.CharacterId(), s.MapId())`. `p.Premium()` is decoded and discarded. |
| Respawn logic | `services/atlas-channel/atlas.com/channel/respawn/processor.go` — the only file in the package; **no tests exist today**. |
| Wheel detection | `processor.go:75-78` — `inv.Cash().FindFirstByItemId(uint32(item.WheelOfFortuneId))`, presence only, no quantity check. |
| Protective item | `processor.go:97` `findProtectiveItem` — Cash Safety Charm, then ETC Easter Basket, then ETC ProtectOnDeath. Returns `(*uint32, inventoryConst.Type)`; the type is discarded at the call site. |
| Exp loss | `processor.go:121` `calculateExpLoss` — beginner / `NoExpLossOnDeath` / protection / 1% town, 10% luck<50, 5% otherwise. **Explicitly untouched** (PRD non-goal). |
| Saga construction | `processor.go:166` `createRespawnSaga` — `consume_wheel_of_fortune`, `consume_protective_item`, `set_hp` (50), `deduct_experience`, `cancel_all_buffs`, `warp_to_spawn`. Both destroy steps already pass `Quantity: 1, RemoveAll: false`. |
| `Change.Premium()` | `libs/atlas-packet/field/serverbound/change.go:38`, decoded at `change.go:105`. |

## 2. Codebase patterns the new code must match

- **Codec shape:** `libs/atlas-packet/character/serverbound/expression.go` and
  `.../clientbound/expression.go` are the reference pair — private fields,
  value-receiver accessors, `Operation()` returning the handler/writer name
  constant, `Encode(l, ctx) func(options) []byte`, pointer-receiver
  `Decode(l, ctx) func(r, options)`.
- **Writer methods:** `w.WriteInt(uint32)`, `w.WriteInt32(int32)`,
  `w.WriteByte`, `w.WriteBool`, `w.WriteAsciiString`. Reader mirrors:
  `r.ReadUint32()`, `r.ReadInt32()`, `r.ReadByte()`, `r.ReadBool()`.
- **Fixture tests:** `libs/atlas-packet/character/serverbound/expression_test.go`
  — `pt.CreateContext(region, major, minor)`, `pt.Variants`, `pt.RoundTrip`,
  and `// packet-audit:verify packet=<path> version=<v> ida=<addr>` markers
  above the test that proves the cell. `pt.Variants`
  (`libs/atlas-packet/test/context.go:18`) contains v28, v83, v87, v95,
  jms185, v84, v86, v48, v61, v72, v79, v92 — every version this task needs.
- **Handler shape:** `socket/handler/character_expression.go` — decode, log
  `[%s] read [%s]`, delegate. Registration is two lines in `main.go`: the
  writer-name slice near line 673 and `handlerMap[...]` near line 877.
- **Broadcast:** `_map.NewProcessor(l, ctx).ForOtherSessionsInMap(field, refCharacterId, session.Announce(l)(ctx)(wp)(writerName)(encoder))`
  — see `kafka/consumer/expression/consumer.go:62`.
- **Own-session announce:** `session.NewProcessor(l, ctx).IfPresentByCharacterId(ch)(characterId, op)`
  (`session/processor.go:182`).
- **Processor taking a writer producer:** precedent is
  `movement.NewProcessor(l, ctx, wp)` (`movement/processor.go:46`).
- **Effect bodies:** `libs/atlas-packet/character/effect_body.go:135,141` —
  `CharacterProtectOnDieItemUseEffectBody` /
  `...ForeignBody`, both resolving the mode through
  `atlas_packet.WithResolvedCode("operations", "PROTECT_ON_DIE_ITEM_USE")`.
  The codec is `clientbound/effect.go:374` (`EffectProtectOnDie`) — mode,
  safetyCharm, usesRemaining, days, and `itemId` only when `!safetyCharm`.

## 3. Testing constraints (why Unit C is refactored the way it is)

`services/atlas-channel/.../respawn/` has no tests and no mocks. The service's
dependency style is `NewProcessor(l, ctx)` constructing concrete REST clients
internally; there are **no** mock packages for `inventory`, `data/map`, or
`saga` (only `character/mock` and `map/mock` exist). The service's own testing
convention is `httptest` servers plus `t.Setenv("<SVC>_SERVICE_URL", …)` — see
`mount/processor_test.go`.

Standing up four fake services to test a branch decision would be far more
fragile than the decision it tests, and `saga.Create` publishes to Kafka, which
`httptest` cannot intercept. Hence **plan Task 6 extracts the decision into a
pure `planRespawn`** taking already-fetched models and returning a
`respawnPlan`. That gives full coverage of the premium gate, charge gate, and
protection selection with zero I/O, and leaves `Respawn` as a thin fetch →
plan → saga → announce shell. Test fixtures use the production Builders (per
CLAUDE.md: no `*_testhelpers.go`).

One wrinkle forced a small design choice: `map_.Model` (`data/map/model.go:8`)
has all-private fields, **no builder, and no exported constructor**, so a test
in package `respawn` cannot construct one. `planRespawn` therefore takes a
three-field `mapFacts` value (`ReturnMapId`, `Town`, `NoExpLossOnDeath`) built
by `mapFactsOf(map_.Model)` in the I/O shell, rather than the model itself.
Saga construction is split the same way: a pure
`respawnSagaSteps(f, characterId, rp, now) []saga.Step` that the tests assert
against, wrapped by `createRespawnSaga`, which is the only part that publishes
to Kafka. That is what lets the plan pin the step *ordering* (consume before
warp — design C4's failure semantics) and the "exactly one wheel DestroyAsset
per death" acceptance criterion without a broker.

## 4. Facts verified during planning (beyond design.md)

1. **All sixteen template slots are free.** Every `(template, opCode)` pair in
   the plan's opcode table has no existing handler/writer entry. Checked by
   parsing all eight JSON templates.
2. **v95 and jms185 bind `CharacterEffect` twice.** v95 at `0xE0` and `0xE9`;
   jms185 at `0xCC` and `0xD5` — both entries carry identical 26-arm
   `operations` tables, and neither is named `CharacterEffectForeign`, unlike
   every other v61+ template. `RegisterTenantWriterOptions`
   (`socket/writer/options_registry.go:24`) keys by writer *name*, so one
   silently wins. `STATUS.md:265` names `SHOW_FOREIGN_EFFECT` at v95 `0x0E0` /
   jms185 `0x0CC` and `STATUS.md:279` the local arm at `0x0E9` / `0x0D5`,
   which identifies the lower opcode as the foreign one. **This blocks Task 8's
   foreign broadcast on two of eight versions**, so plan Task 9 fixes it. It
   was not in the design doc.
3. **v92's template is thin generally,** not just missing the effect writers:
   122 writers / 66 handlers versus 213–221 / 126–138 on its neighbours. The
   plan fixes only the two entries this task needs (Task 10); the wider gap is
   pre-existing and out of scope.
4. **v92's effect opcodes are already in the registry** — `SHOW_FOREIGN_EFFECT`
   `0x0E2`, `SHOW_ITEM_GAIN_INCHAT` `0x0EB` (`STATUS.md:265,279`). Only the
   `operations` mode table needs IDA derivation, and it genuinely does: v87 has
   `PROTECT_ON_DIE_ITEM_USE` = 6, v95 = 8, so v92 cannot be copied from either.
5. **`asset.Model.Quantity()`** (`asset/model.go:163`) returns the real
   quantity for stackables *and* for non-pet cash items (`HasQuantity()` at
   `:159`), so the Wheel of Destiny and the Safety Charm both carry usable
   charge counts. Confirms design OQ-2.
6. **Registry rows** for `USE_DEATHITEM` and `SHOW_UPGRADE_TOMB_EFFECT` exist
   with `fname`s but with `provenance: csv-import` and **no `packet:` link** —
   the field that ties a row to its codec (`gms_v83.yaml:2074`, `:1003`). Task 3
   adds it on all eight versions plus the two brand-new v72/v79 rows.

## 5. Sequencing and parallelism

```
Task 1 (serverbound codec) ─┐
Task 2 (clientbound codec) ─┼─→ Task 3 (registry) ─┐
                            └─→ Task 4 (templates) ─┼─→ Task 11 (verify 16 cells)
                            └─→ Task 5 (handler)   ─┘

Task 6 (premium) → Task 7 (charges) → Task 8 (protect effect)
                                          ↑
Task 9 (v95/jms foreign writer) ──────────┤ prerequisites for the
Task 10 (v92 effect writers) ─────────────┘ effect to resolve at runtime

everything → Task 12 (sweep + review)
```

Tasks 1 and 2 are independent of each other. Tasks 9 and 10 are independent of
6–8 as *edits*, but 8's behaviour is incomplete on v92/v95/jms185 without them,
so land them before claiming FR-5.1 done. Tasks 6→7→8 must run in order — each
builds on the previous task's signatures.

## 6. Decisions locked in (do not relitigate during execution)

- **`USE_DEATHITEM` consumes nothing.** It fires from `CUIRevive::OnCreate`
  before the player chooses. The PRD's FR-2.3 single-consume guard is
  deliberately not built; the acceptance criterion is met by construction
  because only the `MAP_CHANGE` path emits a `DestroyAsset` step.
- **No version gate on either codec.** All eight versions are byte-identical.
- **Owner is excluded from the tomb broadcast** — the client already plays its
  own copy locally (design OQ-4).
- **Client coordinates are relayed as received.** The server does not track
  `m_ptRevive`; substituting a server position would desync owner and
  bystanders. Bounded trust of client input for a cosmetic field, gated by the
  dead-and-owns-a-wheel check.
- **Field-limit gating of the wheel (OQ-5) is not implemented.** The client's
  `is_fieldtype_upgradetomb_usable` keys off a `CField` subclass Atlas has no
  model for; honouring `premium` makes the client's own gate effective.
- **`EffectProtectOnDie.days` is sourced from `asset.Expiration()`** and is the
  one field in this task whose semantic is unproven (design OQ-3). The code
  comment says exactly how to correct it if the live message renders the two
  bytes transposed.

## 7. Known traps

- **Import cycle risk:** `respawn` gains imports of `atlas-channel/session` and
  `atlas-channel/map` in Task 8. Verify with `go build ./...` immediately;
  do not work around a cycle silently if one appears.
- **`tools/lint.sh --check` false-fails without nvm** on PATH. Source nvm and
  re-run before reporting it clean.
- **Cross-worktree golangci-lint lock contention** — if the lint guard hangs,
  another worktree's run holds the lock.
- **Do not hand-edit `STATUS.md` / `status.json`.** Regenerate via the
  packet-audit tooling; a hand edit is a false verify.
- **`packet-audit` needs `-ida-database`** and the correct IDB resolved by
  binary *name* from `idb_list`; `select_instance` is dead.
- **Model cost:** pin `packet-verifier` and the review agents to Sonnet.
- **Subagent cwd:** every dispatched agent must operate inside
  `.worktrees/task-210-death-item-revive`, never the main repo.
