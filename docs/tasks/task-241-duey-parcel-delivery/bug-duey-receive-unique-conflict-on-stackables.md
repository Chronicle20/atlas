# Bug — DUEY_ACTION RECEIVE rejects any item the recipient already holds, one-of-a-kind or not

- **Task**: task-241-duey-parcel-delivery
- **Branch**: task-241-duey-parcel-delivery (`bfa713a5c` at diagnosis)
- **Environment**: `atlas-pr-1434`, tenant `a049bb75-1ccc-4cb8-ac6a-bd604dfbbe5b`, GMS 83.1
- **Reported**: character `Chronicle` (id 2) gets "unable to retrieve mesos & items due to the
  item being only one of a kind" retrieving ordinary potions.
- **Predecessor**: this is the gate immediately after
  `bug-duey-receive-item-type-never-persisted.md` (fixed in `d41d70d39`). That fix is working —
  the compartment now loads; the request reaches the next check and fails there.

## Reproduced

Yes — live, from `atlas-channel-68b486b978-rdg7x` (the post-fix image):

```
[DueyActionHandle] read [mode [4]]
Character [2] attempted to receive parcel [dce1997d-...] but already holds item [2000004].
Character [2] attempted to receive parcel [3989e2e4-...] but already holds item [2000005].
```

That log line is `duey_action_receive.go:168`, whose arm announces
`ParcelRecvUniqueConflictBody()` → `RECV_UNIQUE_CONFLICT` (0x16) → SP_3911, the message the
player saw.

Item 2000004 (Orange Potion) per atlas-data in that same environment:

```json
{"only": false, "quest": false, "slotMax": 100, "tradeBlock": false}
```

`only` is **false** — it is not a one-of-a-kind item, and a 100-per-stack consumable at that.
The receive should have succeeded.

## Observed

`receiveParcel` rejects whenever the recipient already holds **any** asset with the same
template id (`duey_action_receive.go:167-171`):

```go
if _, found := cp.FindFirstByItemId(*itemId); found {
    reject(parcelcb.ParcelRecvUniqueConflictBody())
```

So every stackable — potions, arrows, ETC drops — is unclaimable the moment the recipient
holds one of the same kind, which is the normal case.

## Expected

Per this task's own design doc (`design.md:584`, §7.2 Receive):

> | recipient already holds a **one-of-a-kind** copy | `RECV_UNIQUE_CONFLICT` (0x16) |

The check must be conditioned on the item actually being one-of-a-kind — WZ `info/only` — not
on mere template co-occurrence. For `only == false` the parcel is claimed and the item merges
into / lands beside the existing stack.

## Root cause

The implementation dropped the "one-of-a-kind" qualifier from the design's rule and reduced it
to a template-equality test. Nothing consults the item's `only` flag; atlas-channel has no
reader for it today.

`only` is available upstream but unevenly exposed: `services/atlas-data/.../consumable/rest.go:54`
carries `Only bool \`json:"only"\`` (read at `consumable/reader.go:56`), and **no other item
category does** — equipment, setup, etc and cash each read their `info` node for `tradeBlock`
and friends but never `only`. Adding it is one line per reader plus one field per REST model;
these readers parse the WZ XML live (no re-ingest needed), so the flag becomes queryable
immediately.

## Fix

Three layers. Ruling made before dispatch, do not re-litigate: **extend the existing
`atlas-channel/data/tradeability` package rather than adding a sibling.** It already fetches
exactly these five per-compartment atlas-data resources and dispatches on `inventory.Type`
(`data/tradeability/processor.go:35-50`); a second package would double the REST traffic and
the maintenance for the same five endpoints. Widen its package doc from "the two WZ questions
the karma gates ask" to the WZ item properties atlas-channel gates on, and keep the package
name — renaming would churn every karma call site for no behavioural gain.

| File | Change |
|---|---|
| `services/atlas-data/atlas.com/data/equipment/rest.go` + `reader.go` | Add `Only bool \`json:"only"\``; set `Only: info.GetBool("only", false)` alongside `TradeBlock` (~`reader.go:114`). |
| `services/atlas-data/atlas.com/data/setup/rest.go` + `reader.go` | Same; alongside `m.TradeBlock` (~`reader.go:47`). |
| `services/atlas-data/atlas.com/data/etc/rest.go` + `reader.go` | Same; alongside `m.TradeBlock` (~`reader.go:47`). |
| `services/atlas-data/atlas.com/data/cash/rest.go` + `reader.go` | Same; alongside `m.TradeBlock` (~`reader.go:87`). |
| `services/atlas-channel/atlas.com/channel/data/tradeability/rest.go` | Add `Only bool \`json:"only"\`` to all five `*RestModel`s; add `only` to `Model` with an `Only() bool` accessor; extend `NewModel` and every `extract`. |
| `services/atlas-channel/atlas.com/channel/socket/handler/duey_action_receive.go` | Add a `getItemOnly func(it inventory.Type, templateId item.Id) (bool, error)` to `dueyReceiveDeps`, wired to `tradeability.NewProcessor(l, ctx).Get(...).Only()`. In `receiveParcel`, gate the existing `FindFirstByItemId` rejection behind that flag being true. |

Lookup-failure posture — match `tradeability`'s existing contract, which its own processor doc
states explicitly ("An error means the LOOKUP FAILED; every caller must treat that as a
refusal, never as a permissive default"): on error, reject with
`ParcelIncorrectRequestBody()` and log at error level. Do **not** fall through to "assume not
unique" and do **not** fall back to the old template-equality behaviour.

Tests (Builder pattern, no `*_testhelpers.go`):

- `duey_action_receive_test.go` — a stackable the recipient already holds with `only == false`
  builds the receive saga and announces no rejection (this is the reported regression); the
  same setup with `only == true` still announces `RECV_UNIQUE_CONFLICT`; a `getItemOnly` error
  announces `INCORRECT_REQUEST` and starts no saga.
- `data/tradeability/processor_test.go` — `Only` extracts correctly for at least the
  consumable and one non-consumable resource.
- atlas-data: extend whichever reader tests cover `TradeBlock` to also assert `Only`.

## Not yet answered

- **The free-slot check has the same shape of defect and is NOT fixed here.**
  `duey_action_receive.go:162` rejects with `RECV_NO_FREE_SLOTS` when
  `len(cp.Assets()) >= cp.Capacity()`, which ignores that a stackable can merge into an
  existing partial stack (2000004's `slotMax` is 100). A recipient with a full inventory but a
  half-full potion stack will be told they have no room when they do. Left out deliberately:
  it needs `slotMax` and stack arithmetic, it is a different rejection arm, and the player has
  not hit it. Flagged for the user's call, not silently folded in.
- Whether any OTHER consumer of `RECV_UNIQUE_CONFLICT`-style logic in this task's surface made
  the same design→code reduction was not swept.

## Resolution

Fixed by **`5dabf1e31`** — "fix(duey): gate receive unique-conflict rejection on WZ info/only".
`only` was added to the equipment/setup/etc/cash readers and REST models in atlas-data (live
XML read, no re-ingest), carried through atlas-channel's existing `data/tradeability` reader,
and the `FindFirstByItemId` rejection is now gated behind it. A lookup failure refuses rather
than defaulting permissive.

- **Gate**: `tools/verify.sh --quick --base bfa713a5c` → PASS (exit 0, 91 modules). `--quick`
  skips the docker bake, so this is not the pre-PR gate.
- **Review**: `atlas-reviewer`, verdict `APPROVED`, 0 blocking / 0 non-blocking — see
  `reviews/review-duey-preflight-gates.md`. It confirmed `only` is read from the WZ info node
  in each of the five categories and reaches atlas-channel with the right JSON tag per resource.
- **Live re-test**: NOT yet confirmed — needs the branch to redeploy to `atlas-pr-1434`, then
  retrieving a stackable the recipient already holds.
