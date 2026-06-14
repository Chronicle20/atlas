# Design — Hex of the Beholder authentic buff accumulation

Status: Proposed (task-088 addendum)
Created: 2026-06-14
Spec: `addendum-hex-of-the-beholder-accumulation.md` (problem statement, options, decision)
Scope: `atlas-buffs` (capability) + `atlas-summons` (driver). **`atlas-channel`: no change.**

---

## 1. Goal

Make a Dark Knight's **Hex of the Beholder** apply **one randomly-chosen stat per
pulse**, each with its **own independent timer**, so buff icons accumulate one-at-a-time
(original-GMS behavior) instead of the whole set refreshing in lockstep (current
Cosmic-mirrored behavior).

**Non-goals:** changing the heal (`AURA_OF_BEHOLDER`, already single-stat); changing hex
stat values / interval / duration (correct per WZ); any packet/client change; changing
default behavior of any other buff.

## 2. Key architectural finding — the channel is already per-stat

`atlas-channel`'s buff consumer drives the client give/cancel **from the event's
`Changes` payload**, one stat at a time:

- `kafka/consumer/buff/consumer.go:handleStatusEventApplied` → `CharacterBuffGiveBody`
  → `cts.AddStat(...)` **per change** (`socket/writer/character_buff_give.go`).
- `handleStatusEventExpired` → `CharacterBuffCancelBody` → `cts.AddStat(...)` **per
  change** (`socket/writer/character_buff_cancel.go`).

So an `APPLIED` carrying one stat sets exactly that stat; an `EXPIRED` carrying one stat
cancels exactly that stat. The v83 client keys temporary stats by stat type, so multiple
stats sharing `sourceId = 1320009` already render and expire independently **on the
client**. **No channel work is required.** The entire gap is server-side state in
`atlas-buffs` plus what `atlas-summons` emits.

## 3. The blocker (recap) and the chosen approach

`atlas-buffs` stores active buffs as `map[int32]buff.Model` keyed by `sourceId`
(`character/registry.go:67`, `character/model.go`). Five hex stats under the single valid
`sourceId 1320009` overwrite each other → only the last survives, so the other four never
emit `EXPIRED` (they'd hang on the client forever) and never restore correctly on relogin.

**Chosen: Option A — opt-in `accumulate` mode** (per `addendum` §6). Default `Apply`
keeps today's exact replace-by-`sourceId` semantics; an opt-in flag switches storage to
per-`(sourceId, statType)` entries with independent timers. Only the Beholder hex sets it,
so every existing buff is provably unaffected (§7).

## 4. Contract change

Add one optional field to the `APPLY` command body, on **both** mirrored declarations:

- `atlas-buffs`: `kafka/message/character/kafka.go` `ApplyCommandBody`
- `atlas-summons`: `buff/producer.go` `ApplyCommandBody` (must stay byte-identical)

```go
type ApplyCommandBody struct {
    FromId     uint32       `json:"fromId"`
    SourceId   int32        `json:"sourceId"`
    Level      byte         `json:"level"`
    Duration   int32        `json:"duration"`
    Changes    []StatChange `json:"changes"`
    Accumulate bool         `json:"accumulate,omitempty"` // NEW
}
```

`omitempty` + zero-value `false` ⇒ every existing producer is byte-compatible and keeps
default semantics. **`APPLIED` / `EXPIRED` events are unchanged** — they already carry
per-stat `changes` + `sourceId` + `expiresAt`, which is all the channel needs.

## 5. atlas-buffs storage design

Change the character registry's buff map from `map[int32]` to `map[string]`, keyed by a
composite that distinguishes whole-source buffs from per-stat buffs:

```
key(normal)     = strconv(sourceId)                 e.g. "1320009"
key(accumulate) = strconv(sourceId) + ":" + statType e.g. "1320009:WEAPON_ATTACK"
```

This reuses **all** existing per-buff machinery unchanged — each accumulate stat is just a
normal single-change `buff.Model` with its own `expiresAt`. Only the map key type and a
key helper change.

**`Apply(..., accumulate bool)`** (`character/registry.go`, `character/processor.go`):
- `accumulate == false` (default): `m.buffs[key(sourceId)] = NewBuff(sourceId, …, changes)`
  — identical to today. Processor emits one `APPLIED` (whole buff), as today.
- `accumulate == true`: for **each** incoming change, store a single-change buff under
  `key(sourceId, change.Type)` with its own `expiresAt`; coexists with that source's other
  stats; re-applying the same stat overwrites just that key (timer refresh). Processor emits
  one `APPLIED` **per stat** (each with that stat's own `expiresAt`) so the channel sets
  each independently. (The hex sweep sends one stat per pulse, so this is normally a single
  `APPLIED`; per-stat emission is the correct general rule.)

**Unchanged by construction** (they iterate values and compare `b.SourceId()`, not the map
key):
- `Cancel(sourceId)` — already removes *all* entries whose `b.SourceId() == sourceId`, so it
  clears every per-stat hex entry in one call. ✓
- `GetExpired` — per-entry `b.Expired()`, returns each expired single-change buff → processor
  emits one `EXPIRED` per stat → channel cancels that stat. ✓ (This is what makes icons drop
  one-at-a-time.)
- `CancelAll`, `CancelByStatTypes`, `GetPoisonCharacters`, `HasImmunity` — iterate values.

**Touched call sites** (mechanical, key type `int32`→`string`):
- `character/model.go`: `buffs map[int32]` field, `Buffs()` getter, `MarshalJSON`/
  `UnmarshalJSON`. (JSON object keys are already strings, so Redis data is forward/backward
  compatible: existing `"1320009"` entries unmarshal unchanged.)
- `character/registry.go`: `make(map[int32]…)` → `map[string]`; `not`/`keep`/`m.buffs[…]`.
- Any caller of `Model.Buffs()` (REST handler / projection — `grep -rn '\.Buffs()'`).

## 6. atlas-summons sweep design

`summon/beholder_task.go:sweepBuff` currently emits one `APPLY` carrying **all** of
`m.BuffChanges()`. Change it to:

1. Pick **one** change at random from `m.BuffChanges()` (the snapshot pool — already the
   full level-appropriate set; no snapshot change needed).
2. Emit `APPLY` with `Changes = [that one]`, `Accumulate = true`, `Duration =
   m.BuffDuration()`, `SourceId = m.BuffSourceId()` (positive `1320009`, unchanged).
3. Keep the existing `SKILL` animation pulse emission as-is.

**Randomness must be injectable** for deterministic tests, mirroring the existing `emit`
field pattern on `BeholderTask`: add a `pick func(n int) int` field (default
`rand.Intn`; tests inject a stub). Vary per pulse — never a fixed seed. Random *with*
replacement is correct: re-rolling an active stat simply refreshes that stat's timer,
matching "a random buff each pulse."

`buff.ApplyProvider` / `applyProvider` (`atlas-summons/buff/producer.go`) gain the
`accumulate bool` parameter.

## 7. End-to-end flow

**Apply (per pulse):** `beholder_task.sweepBuff` → `APPLY{sourceId:1320009, changes:[pdd],
accumulate:true}` → atlas-buffs stores `"1320009:WEAPON_DEFENSE"` (own timer) → `APPLIED{
changes:[pdd], expiresAt}` → channel `AddStat(pdd)` → client shows the pdd icon. Next pulse
rolls `mdd` → stored under `"1320009:MAGIC_DEFENSE"` → second icon appears. Icons accumulate.

**Expiry (per stat):** atlas-buffs `Expiration` task → `GetExpired` finds `"1320009:…"` whose
`expiresAt` passed → `EXPIRED{changes:[that stat]}` → channel cancels just that stat → its
icon drops while the others remain.

## 8. Backward compatibility & risk

- **Existing buffs unaffected:** default `Apply` path unchanged; only callers passing
  `accumulate:true` (the hex sweep alone) get new behavior. A normal multi-stat recast still
  overwrites because it carries all its stats under the same `key(sourceId)` and never sets
  the flag (addendum §5.1).
- **Redis data:** map JSON keys were already strings; old data loads unchanged. No migration.
- **`atlas-buffs` is a shared service** — the map-key-type change is the one place with
  ripple; mitigated by keeping every value-iterating method and the `buff.Model`/event/REST
  shapes identical, and by full regression tests (§9).
- **Two-module change** ⇒ `docker buildx bake` for `atlas-buffs` **and** `atlas-summons`.

## 9. Test strategy

- **atlas-buffs (unit):** accumulate `Apply` of pdd then mdd under `1320009` ⇒ two entries,
  both retrievable, independent `expiresAt`; re-apply pdd ⇒ refreshes only pdd's timer;
  `GetExpired` after one expires ⇒ returns only that stat; `Cancel(1320009)` ⇒ removes all.
  **Regression:** default (`accumulate:false`) multi-stat `Apply` then recast ⇒ single entry,
  overwrites (does not accumulate); existing Apply/Cancel/CancelByStatTypes/expiry tests pass.
- **atlas-summons (unit):** with an injected `pick`, `sweepBuff` emits one `APPLY` with one
  change and `Accumulate:true`; over a sequence of injected picks the union covers the pool;
  re-roll refreshes. Update `beholder_task_test.go` (currently asserts the full set in one
  APPLY).
- **Contract:** a table test asserting `ApplyCommandBody` JSON round-trips with/without
  `accumulate` byte-identically across both service mirrors.

## 10. Verification (live)

On an ephemeral env (DrK with Hex ≥ L20): summon Beholder, observe buff icons appear
**one at a time** over successive pulses with independent countdowns; let one lapse and
confirm a single icon drops while others persist; relog and confirm per-stat restore.
Cross-check Loki: per-pulse `APPLIED{changes:[oneStat], accumulate}` and independent
`EXPIRED{changes:[oneStat]}` events. No client crash (sourceId stays `1320009`).

## 11. Decisions / open questions

- **Random with replacement** (re-roll allowed, refreshes timer) — matches original; default.
  Alternative (round-robin / without-replacement until full) is *not* authentic; rejected
  unless requested.
- **Pool source:** the level-appropriate `hex.Statups()` already snapshotted at spawn — no
  change. Lower levels naturally have a smaller pool (e.g. L1 = `pdd` only).
- **Per-stat duration:** use the hex effect's `time` (`m.BuffDuration()`) for each stat —
  same duration the WZ defines; only the *application* becomes incremental.
