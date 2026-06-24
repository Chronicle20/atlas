# Resurrection Skill — Design

Task: task-111-resurrection-skill
Status: Approved design
Created: 2026-06-24
Inputs: `prd.md` (approved)

---

## 1. Summary

Resurrection (`2321006` Bishop, `9001005` GM, `9101005` SuperGM) revives dead
characters in place at full HP. It is implemented as an active-skill handler in
atlas-channel, registered exactly like Heal and Mystic Door, and dispatched from
the generic `UseSkill` path (which has already validated ownership/level, loaded
the WZ effect, and consumed MP + applied cooldown by the time the handler runs).

Mechanically each revive is a two-command sequence per dead recipient:

1. **Set HP to full** — emit an absolute `SET_HP` command (new atlas-channel
   producer) with a `0xFFFF` sentinel; atlas-character clamps it to the
   recipient's effective MaxHP, yielding a true full-HP restore.
2. **Warp to death position** — call the existing
   `portal.WarpToPosition(f, recipientId, sameMapId, deathX, deathY)` (the
   task-093 chase-warp primitive). Because the recipient is in the client death
   stance, the resulting `SET_FIELD` fires the client's native `OnRevive`,
   closing the death prompt and standing the avatar up at its death coordinates.

The holy-light skill-use effect is broadcast to the caster and other players in
the map, reusing the Heal handler's `AnnounceSkillUse` / `AnnounceForeignSkillUse`
helpers. No new REST endpoints, Kafka topics, tables, or migrations.

This is a faithful v83 implementation: full-HP revive, range from WZ, MP/cooldown
from WZ, **no invincibility window**, no experience interaction.

## 2. Key findings from code reconnaissance

These ground the design and resolve several PRD open questions at design time
(verified by reading source, cited inline):

- **The warp reads HP fresh, not from a stale channel cache.** `warpCharacter`
  loads the character via `character.NewProcessor(...).GetById()` immediately
  before building `WarpToPositionBody(... c.Hp() ...)`
  (`kafka/consumer/character/consumer.go:242,253`). So FR-10's "HP set before the
  warp packet is built" is satisfied as long as the `SET_HP` command persists in
  atlas-character before the warp's much longer async chain
  (channel → portals → maps `MAP_CHANGED` → channel → REST `GetById`) reaches
  that fetch. The ordering is heavily skewed safe (1 hop + DB write vs. 3+ hops +
  REST round-trip), not a tight race. We still emit `SET_HP` first.

- **atlas-character's `SetHP` clamps to effective MaxHP internally**
  (`character/processor.go:1178-1185`): it fetches effective stats and clamps the
  requested amount down to effective max. Therefore the handler can send a
  `0xFFFF` sentinel and receive a correct full-HP fill **without fetching
  effective stats per recipient** — a meaningful simplification over Heal, which
  must compute exact deltas.

- **atlas-channel has no absolute HP producer** — only relative `ChangeHP`
  (`int16`, `character/producer.go:57`). `int16` (max 32767) cannot express a
  "fill to max" sentinel and would force a per-recipient effective-stats fetch,
  so we add an absolute `SET_HP` producer (decision below).

- **`GmResurrectionId` (9001005) does NOT exist** in
  `libs/atlas-constants/skill/constants.go` — only `BishopResurrectionId`
  (`2321006`) and `SuperGmResurrectionId` (`9101005`) are defined. PRD FR-1's
  claim that all three constants exist is incorrect; this design adds the missing
  constant + `Skill` var + registry-map entry.

- **The shared party selector hard-codes the living-only filter**
  (`skill/handler/recipients.go:174` — `if mc.Hp() == 0 { continue }`).
  Resurrection needs the inverse (dead-only), so the selector internals are
  generalized with an alive/dead predicate (decision below).

## 3. Architecture decisions

### D1 — Full-HP restore via a new absolute `SET_HP` producer *(user-approved)*

Add a `SetHP` command + producer to atlas-channel's character package, mirroring
`ChangeHP`. The handler emits two direct commands per recipient (`SetHP`, then
`WarpToPosition`), matching the Mystic Door direct-command style rather than a
saga. atlas-character already consumes `SET_HP` (`CommandSetHP = "SET_HP"`,
`SetHPAndEmit`); we only add the producing side.

- New on the channel side:
  - `CommandSetHP = "SET_HP"` and
    `SetHPCommandBody{ ChannelId channel.Id; Amount uint16 }` in
    `kafka/message/character/kafka.go` (field names/JSON tags identical to
    atlas-character's `SetHPBody` so the command deserializes unchanged).
  - `SetHPCommandProvider(f, characterId, amount uint16)` in
    `character/producer.go`, shaped exactly like `ChangeHPCommandProvider`
    (zero-uuid `TransactionId`, same as `ChangeHP`, which already works this way).
  - `SetHP(f field.Model, characterId uint32, amount uint16) error` on
    `character.Processor` + `ProcessorImpl` + the mock.
- The handler calls `cp.SetHP(f, recipientId, math.MaxUint16)` → atlas-character
  clamps to effective max → full HP.

Rejected alternatives: relative `ChangeHP` (can't express full-fill in `int16`,
needs per-recipient stats fetch); respawn-style saga (needs a new
`WarpToPosition` saga action and adds orchestration weight the fresh-`GetById`
read makes unnecessary).

### D2 — Generalize the shared selector; add two dead-target selectors

`skill/handler/recipients.go` already enumerates party members with channel/map/
session/bitmap/range filters. Generalize the internal `selectPartyMembers` to take
an HP predicate (`wantDead bool`, or a small `hpFilter` enum) instead of the
hard-coded `Hp() == 0` skip, then expose:

- `SelectDeadInRangePartyMembers(l, ctx, f, casterId, casterX, casterY, e, bitmap)`
  — Bishop variant. Same as `SelectInRangePartyMembers` but keeps only `Hp()==0`
  members. Reuses the LT/RB rectangle, bitmap, same-channel/map, live-session
  filters unchanged.
- `SelectDeadInRangeMapPlayers(l, ctx, f, casterId, casterX, casterY, e)` — GM /
  SuperGM variant, party-agnostic. Enumerates **all** sessions in the caster's
  field via `_map.NewProcessor(...).ForSessionsInMap(f, ...)`
  (`map/processor.go:45`), loads each character via `GetById`, and keeps those
  with `Hp()==0`, excluding the caster, inside the caster-relative LT/RB
  rectangle. Concurrency: `ForSessionsInMap` runs its callback concurrently, so
  the accumulator is mutex-guarded exactly as `inMapCharacterIdsFunc`
  (`recipients.go:57-70`) already does.

Existing callers of `SelectInRangePartyMembers` / `SelectPartyMembersInMap`
(Heal, buffs) keep their living-only behavior — the generalization preserves the
current default. Test seams (`loadCasterPartyFunc`, `inMapCharacterIdsFunc`,
`loadPartyMemberFunc`) are reused; a `loadMapPlayerFunc` seam is added for the
GM-variant per-session character load so the new selector is unit-testable
without the live processor stack.

The bitmap MSB-first slot mapping (`recipients.go:157`) and the missing-rectangle
fallback (`recipients.go:99-104`) are preserved as-is.

### D3 — In-place warp to each recipient's own death coordinates

Each selected recipient carries its `X`/`Y` (death position) captured at
selection time — the same way `PartyRecipient` does (`recipients.go:185-191`) and
the GM selector does from `GetById`. The handler warps the recipient to
`(deathX, deathY)` on the **same** map (`f.MapId()`), reusing
`portal.WarpToPosition` unchanged. No `libs/atlas-packet` changes: the existing
`WarpToPositionBody` chase encoding is already version-branched and IDA-cited for
GMS v83/v84/v87/v95 and JMS v185 (`field/clientbound/warp_to_map.go`), and the
revive is driven by the death-stance gate, not the packet's `revive` byte
(which stays `0` — `warp_to_map.go:109`). The OQ-2 `revive`-byte variant is a
**conditional, live-gated fallback** (§7), not built up front.

## 4. New package: `skill/handler/resurrection/`

Mirrors `skill/handler/mysticdoor` and `skill/handler/heal` layout.

```
skill/handler/resurrection/
  resurrection.go        // init() registers all three IDs; Apply handler
  recipients.go          // thin variant-to-selector mapping (which selector per skillId)
  resurrection_test.go   // handler behavior (no-op, ordering, broadcast) via seams
  recipients_test.go     // selector dispatch per variant
```

### 4.1 `init()` + registration

```go
func init() {
    channelhandler.Register(skill2.BishopResurrectionId, Apply)
    channelhandler.Register(skill2.GmResurrectionId, Apply)        // new constant (§5)
    channelhandler.Register(skill2.SuperGmResurrectionId, Apply)
}
```

Add the blank import to `skill/handler/registrations/registrations.go`:

```go
_ "atlas-channel/skill/handler/resurrection" // Resurrection — task-111
```

### 4.2 `Apply` — handler signature & lifecycle

Same signature as Heal/Mystic Door:
`func(l) func(ctx) func(wp writer.Producer, f field.Model, characterId uint32, info packetmodel.SkillUsageInfo, e effect.Model) error`.

Lifecycle:

1. Load caster (`GetById`) for `X`, `Y`, `Level` (caster position is the rectangle
   origin; caster level feeds the effect packet). On error: log, return `nil`
   (the client was already re-enabled by `UseSkill`).
2. `warnIfMissingRectangle`-style guard (reuse the Heal pattern or its shared
   equivalent) for skills lacking LT/RB — log once; with no rectangle the dead
   selectors return empty → clean no-op.
3. Select recipients by variant (FR-5/FR-6):
   - `BishopResurrectionId` → `SelectDeadInRangePartyMembers(... info.AffectedPartyMemberBitmap())`.
   - `GmResurrectionId` / `SuperGmResurrectionId` → `SelectDeadInRangeMapPlayers(...)`.
   The variant→selector mapping lives in `recipients.go`.
4. If no recipients: log the empty set, broadcast the self/foreign skill-use
   effect, return `nil` (FR-9 clean no-op; MP/cooldown already consumed upstream).
5. For each recipient (order mirrors Heal's simple iteration):
   - `cp.SetHP(f, r.Id, math.MaxUint16)` — full HP (atlas-character clamps).
   - `portal.NewProcessor(l, ctx).WarpToPosition(f, r.Id, f.MapId(), r.X, r.Y)`.
   - On per-recipient error: log and continue (FR per §8 — a single failure must
     not abort the whole cast).
6. Broadcast effects (FR-12): `AnnounceSkillUse` to the caster's own session and
   `AnnounceForeignSkillUse` to other sessions in the map, identical to Heal
   (`heal.go:156-164`).
7. Debug-log cast, resolved recipient set (count + ids), and each revive
   (id + death coords) per FR (§8 observability).

No XP path (FR-13), no `character_damage.go` changes (invincibility out of scope).

### 4.3 Recipient model

The handler's per-recipient working type needs only `Id`, `X`, `Y` (death
coords). It can consume the shared `handler.PartyRecipient` directly for the
Bishop path and a parallel lightweight struct for the GM path, or both selectors
can return `[]handler.PartyRecipient` for uniformity (the GM selector populates
`Id`/`X`/`Y`; `Hp`/`MaxHp` are unused downstream). Returning the shared
`PartyRecipient` from both keeps the handler's per-recipient loop type-uniform;
this is the chosen shape.

## 5. `libs/atlas-constants` change

Add the missing GM constant alongside the existing GM skills
(`skill/constants.go`, near `GmHaste`/`GmHide`, lines ~1438-1480):

```go
GmResurrectionId = Id(9001005)

var GmResurrection = Skill{ id: GmResurrectionId, /* fields mirroring SuperGmResurrection */ }
```

and the `GmResurrectionId: GmResurrection` entry in the id→Skill registry map
(mirroring the `SuperGmResurrectionId` entry at line ~2699). Field values for the
`Skill` var follow the existing `SuperGmResurrection` definition (read it before
editing). This touches `libs/atlas-constants`, so its bake target is in scope for
verification (§9).

## 6. Data flow

```
client cast ─► UseSkill dispatch (ownership/level OK, WZ effect loaded,
                MP consumed + cooldown applied)
            └─► registry.Lookup(skillId) ─► resurrection.Apply
                    │
                    ├─ load caster (GetById): X, Y, Level
                    ├─ select dead recipients (party | map), each with death X/Y
                    └─ per recipient:
                         ├─ SET_HP(0xFFFF) ──► COMMAND_TOPIC_CHARACTER
                         │      └─ atlas-character SetHPAndEmit ─ clamps to eff. max ─ persists ─ STAT_CHANGED
                         └─ WarpToPosition(mapId, deathX, deathY) ──► EnvPortalCommandTopic
                                └─ atlas-portals ─► atlas-maps CHANGE_MAP ─► MAP_CHANGED (UseTargetPosition,TargetX,TargetY)
                                       └─ atlas-channel warpCharacter:
                                              GetById (reads NOW-FULL Hp) ─► WarpToPositionBody ─► SET_FIELD
                                                     └─ client death stance ⇒ OnRevive ⇒ stand up at (deathX,deathY)
                    └─ AnnounceSkillUse (caster) + AnnounceForeignSkillUse (others in map)
```

Reused unchanged: `WarpToPositionBody`/`SetFieldWriter`, the portal→maps→channel
warp chain, the `MAP_CHANGED` event carrying `UseTargetPosition/TargetX/TargetY`.

## 7. Conditional fallback (OQ-2 — built only if live testing requires it)

If live v83 testing shows the death-stance gate alone does not reliably fire
`OnRevive`, add a chase **+ revive** variant in
`libs/atlas-packet/field/clientbound/warp_to_map.go` that writes the `revive`
byte as `1` instead of the hard-coded `0` (`warp_to_map.go:109`), plumb a
`UseRevive` flag through the portal/maps warp command + `MAP_CHANGED` body and the
channel `warpCharacter` branch (`consumer.go:252`). This is **not** implemented up
front — the static analysis (`CStage::OnSetField` @0x776020) indicates the
death-stance gate suffices, and the PRD scopes this as a live verification gate.
The plan phase should carry it as a clearly-bounded contingency task, not a
default deliverable.

## 8. Error handling, concurrency, observability

- **Per-recipient isolation:** a `SetHP` or `WarpToPosition` failure for one
  recipient is logged and skipped; the cast continues for the others and returns
  `nil`.
- **Caster-load failure / missing rectangle / empty recipient set:** log, still
  broadcast the self/foreign effect, return `nil`. Never return an error that
  would surface as a failed cast after MP/cooldown were already spent.
- **Concurrency:** the GM map-wide selector mutex-guards its accumulator
  (`ForSessionsInMap` callback is concurrent). Per recipient the `SetHP`-then-
  `WarpToPosition` emission order is preserved; cross-recipient order is
  unconstrained (each is independent).
- **Multi-tenancy:** every processor call threads `ctx` (tenant from context);
  selection and warps are scoped to the caster's field/channel. No cross-tenant
  access.
- **Observability:** debug-log the cast (caster id, skill id, level), the resolved
  recipient set (count + ids), and each revive (id + death coords), consistent
  with Heal/Mystic Door.

## 9. Testing strategy

Unit tests only (no new integration surface); follow the Heal/Mystic Door test
style with package-level function seams (no `*_testhelpers.go`, Builder pattern
for any model setup):

- **Selectors** (`skill/handler/recipients_test.go`, extend existing):
  - `SelectDeadInRangePartyMembers` keeps only `Hp()==0` members; living members,
    out-of-range, wrong-channel/map, not-in-session, and unset-bitmap members are
    excluded. Verifies the living-only callers are unaffected by the
    generalization (existing tests stay green).
  - `SelectDeadInRangeMapPlayers` returns all dead in-range players regardless of
    party; excludes the caster and living/out-of-range players; mutex-safe under
    multiple sessions (seam returns >1 session).
- **Handler** (`resurrection_test.go`): via seams, assert
  - variant→selector dispatch (Bishop→party, GM/SuperGM→map),
  - per recipient `SetHP(0xFFFF)` is emitted **before** `WarpToPosition` with the
    recipient's death coords and `f.MapId()`,
  - empty recipient set ⇒ no SetHP/warp, effect still broadcast, `nil` return,
  - one recipient's failure doesn't abort the others.
- **Constants:** a trivial assertion that `GmResurrectionId == 9001005` and is in
  the registry map (mirrors existing constant coverage if any).

## 10. Faithfulness / values (verify against WZ, not memory)

- Range: `e.LT()`/`e.RB()` from the WZ effect — `lt(-400,-350)`/`rb(400,250)` in
  v83 per the PRD; **read from the effect at runtime, never hard-coded**.
- MP (`mpCon`) and cooldown (`cooltime`): applied by the generic `UseSkill` path
  from the WZ effect; the handler does not re-implement them.
- Full-HP restore: `0xFFFF` sentinel clamped to effective max by atlas-character.
- **No invincibility** (v83 `232.img.xml` has no such property);
  `character_damage.go` untouched.

## 11. Out of scope (carried from PRD §2)

Post-revive invincibility; experience/death-penalty interaction; the
`character_damage.go` mitigation TODOs; reviving pets/summons; self-revive.

## 12. Live verification gates (carried from PRD §9)

Empirical, settled on the running environment during implementation — verification
gates, not planning blockers:

- **OQ-1** dead-player chase warp fires `OnRevive` (confirm live on v83).
- **OQ-2** `revive` byte vs. death-stance gate — build §7 fallback only if OQ-1
  is unreliable.
- **OQ-3** same-map warp despawn/respawn flicker for observers.
- **OQ-4** tracked `X`/`Y` reflects the actual death position closely enough for
  "in place"; if not, source death coords from atlas-maps location state.
- **OQ-5** per-version revive parity (v87/v95/JMS) and per-version existence of
  the GM/SuperGM IDs + their WZ range.

## 13. File-level change inventory

New:
- `services/atlas-channel/.../skill/handler/resurrection/resurrection.go`
- `services/atlas-channel/.../skill/handler/resurrection/recipients.go`
- `services/atlas-channel/.../skill/handler/resurrection/resurrection_test.go`
- `services/atlas-channel/.../skill/handler/resurrection/recipients_test.go`

Modified (atlas-channel):
- `skill/handler/registrations/registrations.go` — blank import.
- `skill/handler/recipients.go` — generalize selector with HP predicate; add
  `SelectDeadInRangePartyMembers`, `SelectDeadInRangeMapPlayers`, `loadMapPlayerFunc`.
- `character/processor.go` — `SetHP` method on `Processor`/`ProcessorImpl`.
- `character/producer.go` — `SetHPCommandProvider`.
- `character/mock/processor.go` — `SetHP` mock.
- `kafka/message/character/kafka.go` — `CommandSetHP` + `SetHPCommandBody`.

Modified (libs):
- `libs/atlas-constants/skill/constants.go` — `GmResurrectionId` + `GmResurrection`
  + registry-map entry.

Conditional (only if OQ-2 fails live): `libs/atlas-packet/field/clientbound/warp_to_map.go`
+ portal/maps warp command + `MAP_CHANGED` body + channel `warpCharacter` branch.

## 14. Verification (per CLAUDE.md)

Modules touched: `atlas-channel` and `libs/atlas-constants` (always); `atlas-packet`
+ `atlas-portals`/`atlas-maps` only if the §7 fallback is built.

- `go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed
  module.
- `docker buildx bake atlas-channel` (and any other service whose `go.mod` is
  touched — `libs/atlas-constants` is consumed by many services, so a broad bake /
  `docker buildx bake all-go-services` is the safe check).
- `tools/redis-key-guard.sh` clean from the repo root.
- Live: exercise on v83 (OQ-1); record per-version parity (OQ-5).
