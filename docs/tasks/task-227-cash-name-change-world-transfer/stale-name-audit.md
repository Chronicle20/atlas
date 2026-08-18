# Stale character-name audit, and the coupon-consumption decision

Follow-up to `bug-purchase-path-sets-assetid.md`, from live testing of a
successful name change on `atlas-pr-1370`.

## 1. Party window — FIXED (commit `2147a433f`)

Two independent gaps, both real, both now closed:

- **No `NAME_CHANGED` consumer in atlas-parties.** atlas-buddies, atlas-guilds
  and atlas-mts all consume it; atlas-parties never did. Added: it updates the
  registry name and emits a `NAME_CHANGED` party-member status event, which
  atlas-channel now consumes and turns into a party-update re-announce. That
  re-announce is what redraws the window — `toPartyMembers` reads `m.Name()`
  off the atlas-parties model, so the registry copy is the only name the client
  ever sees.
- **Login did not re-sync the name.** `character.Login`'s relog refresher pulls
  GM, level and job from atlas-character with the comment *"can have changed
  while offline … with no event this service saw"*. Name is the textbook case of
  that — a pending `NAME_CHANGE` applies at `LOGOUT` — and was omitted. This is
  why the stale name survived relogs, not merely the session.

## 2. Audit: everywhere else a character name is held as a copy

| Holder | Storage | Refreshes on rename? | Verdict |
|---|---|---|---|
| atlas-buddies | DB (`buddy` entity) | yes — consumes `NAME_CHANGED` | OK |
| atlas-guilds | DB (roster names) | yes — consumes `NAME_CHANGED` | OK |
| atlas-mts | DB (listing seller name) | yes — consumes `NAME_CHANGED` | OK |
| **atlas-parties** | registry (Redis) | **now yes** | **fixed here** |
| **atlas-rankings** | DB (`ranking.Name`) | **indirectly** — the recompute task re-reads every character from atlas-character each cycle (base tick 60s, tenant-configurable interval) | **latency, not staleness** — see below |
| **atlas-merchant** | DB (`blacklist.Name`, unique index) | **no** | **open** — a blacklisted player who renames escapes the blacklist, and the row keeps a name that no longer resolves |
| atlas-messengers | in-memory, set on entry | n/a — repopulated per messenger session | OK |
| atlas-trades | DB (`ledger`, `settlement`) | no, deliberately | OK — historical records should read as-of-the-time |

### atlas-rankings

Not a stored-forever-stale bug. `Recompute` rebuilds every row from
`p.characters()`, name included, on each cycle. What the UI showed was the
window between the rename and the next recompute. Worth confirming the tenant's
configured interval before deciding anything — if it is long, the fix is the
interval or a `NAME_CHANGED`-triggered recompute, not a new name-sync path.
**Unverified: the actual configured interval for this tenant.**

### atlas-merchant

The only genuine never-refreshes copy found. Storing the name is also what makes
it wrong: the blacklist should key on `characterId`, which is what the seam
elsewhere in the repo does. Not fixed here — it is a data-model change on a
service this task does not otherwise touch, and it needs its own decision about
migrating existing rows.

## 3. Consuming the coupons on apply — DECISION NEEDED

Requested: when the name change applies, consume any/all Character Name Change
items in the player's inventory.

Grounding (from `derivation.md`, IDA-derived, not assumed): `5400000` is the
name-change item and `5401000` is world transfer, compared as **exact ids** in
`CCashShop::ProcessBuy` on every GMS version v48–v95. The client's *use*-side
dispatcher instead buckets by prefix (`nItemID / 1000 == 5400`), and
`character_cash_item_use.go:1431` already mirrors that prefix test. So "any/all"
should mean the `5400xxx` band, matching the client's own classification.

**The obstacle:** the saga's `destroy_asset` action cannot express it. Its
handler (`compartment/processor.go` `RequestDestroyItem`) resolves *the first*
asset matching the template id and destroys that one slot — so even
`removeAll: true` clears a single slot. The reported case has two coupons in two
slots (assets 27 and 28, slots 1 and 2), so one `destroy_asset` step provably
misses one.

Options:

- **A — atlas-character enumerates.** It would have to read the player's cash
  inventory to find every matching asset. Costs a new atlas-character →
  atlas-inventory dependency, which does not exist today (no client, no env
  var). Against CLAUDE.md's service-boundary guidance.
- **B — new `destroy_all_assets` saga action (recommended).** Put the
  enumeration where the inventory lookup already lives: the orchestrator's
  compartment processor already calls `RequestCompartment`. A new action loops
  every matching asset instead of taking the first. atlas-character emits one
  step; no new service seam; existing `destroy_asset` semantics untouched, so no
  other caller changes. Also fixes the "removeAll only clears one slot"
  limitation for any future caller.
- **C — single `destroy_asset` for `5400000` with `removeAll`.** Cheapest,
  handles the one-slot case only. Does not satisfy "any/all".

Also undecided, and worth an explicit answer: should this fire only on
`APPLIED`, or also when a request is `REJECTED` for losing the name? The
request as stated says apply.
