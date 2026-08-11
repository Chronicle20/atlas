# task-205 player trade — PR notes

What this branch ships, the one design decision that was reversed mid-flight,
and the residuals a reviewer should know were adjudicated rather than missed.

Authoritative background: `design.md` §5A (the escrow amendment, with its
"Amended during implementation" blocks) and the "Slice 7 as built" table at the
end of `plan.md`. Nothing here contradicts either; this is the reviewer's
summary of both.

## The headline: staged assets are ESCROWED, not reserved

The branch was built to §5.3's **reserve-at-staging** model — the staged item
stays in its owner's compartment under an atlas-inventory reservation, and the
swap happens at settlement. The first live test showed that model is
wire-incompatible with the reference client, and §5A replaced it.

Why it cannot work (design §5A.1, all read from the GMS v83 binary):
`CTradingRoomDlg::PutItem` @`0x7c359f` and `::PutMoney` @`0x7c37ca` both set
`m_bExclRequestSent = 1` immediately after `SendPacket`, and
`CWvsContext::CanSendExclRequest` @`0x485bf7` then refuses every later exclusive
request. Only three server packets clear that flag — the leading
`exclRequestSent` bool of `STAT_CHANGED` or `INVENTORY_OPERATION`, or a
`SET_FIELD`. Reserve-at-staging mutates neither inventory nor meso, so it emits
none of the three: the lock latched on the first stage and never cleared. The
reported symptom was "the mesos button stopped working after I put an item in".

What replaced it (design §5A.2–§5A.5):

- A staged item **genuinely leaves** its owner's compartment, into a new
  first-class custody destination in the accept/release family the fleet already
  uses for storage, cash shop and MTS: `accept_to_trade` / `release_from_trade`,
  backed by atlas-trades' own `trade_escrow_items` / `trade_escrow_mesos`.
- Staging is therefore a saga (`transfer_to_trade`), not a reservation call. The
  resulting real `INVENTORY_OPERATION` is what clears the client's lock — **no
  new clientbound packet was needed for the unlock**, which is the strongest
  argument for the amendment.
- `ITEM_STAGED` is emitted from the **staging saga's terminal status**, not from
  the custody consumer, so confirmation and refusal are one signal on one topic
  in one order. Between the swap and that status the item is *pending*: it holds
  its dialog slot but is announced to nobody.
- Escrowed **meso** is armed as a pending stake on the row *before* its
  `award_mesos` is submitted, and committed by a compare-and-set on the stake id
  when that saga completes. The naive ordering loses money: a teardown between
  the debit and the record destroys the only record of meso the player has
  already been charged.
- Settlement no longer releases from characters or issues negative
  `award_mesos`; the debit already happened. It is
  `release_from_trade` × n → `accept_to_character` × n → the two delivery
  `award_mesos`, tax destroyed by crediting the receiver less than was escrowed.

What this **removed** (design §5A.10): the `compartment` reserve/cancel command
producer, the reservation-refresh ticker, and `trade.reservationTtlSeconds` from
the tenant config. The planned stuck-escrow retry ticker was **not built** — the
orphan it would have swept announces itself through its own completing saga
(`returnOrphanedStage` / `refundOrphanedStake`). Net: the service ends this
change with one *fewer* background loop than it started with.

## Every refusal answers, or the dialog wedges

Design §5A.6. With the lock armed on send, a silent drop is no longer free.
atlas-channel's trade consumer now writes an empty `STAT_CHANGED` with
`exclRequestSent = true` — no stat payload, no trade packet — to the acting
character alone, on both the new `ITEM_REFUSED` and the existing `MESO_REFUSED`.
The slot still stays empty, so the player-visible behaviour is unchanged; only
the lock is released. atlas-trades still writes no packets (§2.2 preserved).

## Startup recovery is now implementable

Design §5A.9. `ReconcileAtBoot` runs two passes in a mandatory order:
settlements first (`Reconcile`), then stranded escrow (`ReconcileEscrow`). Rooms
are process-local, so every escrow row that survives a restart is orphaned by
definition and is returned to the owner the row names — one `trade_unwind` per
room, not per row.

Two things in that path are load-bearing and easy to get wrong:

- **The exclusion set is captured BEFORE the settlement pass.** `Reconcile`
  deletes each record as it resolves it, so reading the unresolved set
  afterwards returns nothing and the sweep would treat rows belonging to a
  just-resolved settlement — whose release saga is still in flight, moments
  behind in the same process — as stranded. That is the double-delivery.
- **A refunded meso row is ZEROED, not deleted.** The unwind saga refunds meso
  through a bare `award_mesos` that deletes nothing, so a surviving non-zero row
  would be refunded again by the next sweep; and a stake still in flight
  resolves against that row by its `pending_stake_id`, so deleting it would
  strand a debit the player has already been charged.

Both are pinned by tests in `trade/settlement_reconcile_test.go`, which drives
the boot path against a real database and httptest-backed atlas-maps and
atlas-saga-orchestrator roots — the only way to reach the ordering, since only a
real terminal answer makes `Reconcile` delete a record mid-run.

## Deploy steps (do these or the deploy breaks)

- **`atlas-trades-main` must be created by hand on the Postgres host** before the
  main deploy. Pods crash-loop on SQLSTATE 3D000 until it exists. See
  [`docs/adding-a-new-service.md`](../../adding-a-new-service.md) §6.1.
- **The main-overlay image is pinned at `newTag: latest`** until the first main
  publish pins a `main-<sha>`. The images entry exists, so the bump workflow will
  pin it; this matches the `atlas-mini-games` precedent (`99650b7ed`).
- **atlas-data must be re-ingested** before `tradeBlock` reads true — the flag is
  surfaced by the readers this branch changed, but existing ingested rows predate it.
- `TRADES_SERVICE_URL` is unregistered, so `requests.RootUrl("TRADES")` falls back
  to `BASE_SERVICE_URL`. Consistent with `MINI_GAMES` today, but it should be an
  explicit decision rather than an inherited default.

## Behavioural gaps a player or GM can hit

- **Config changes need a restart.** The per-tenant trade-config cache is never
  invalidated and atlas-trades subscribes to no `TRADE_CONFIG_UPDATED`/`DELETED`
  event, so a changed tax table or a flipped `taxEnabled` applies only after an
  atlas-trades pod restart.
- **`taxTiers` cannot be cleared via the API** — an empty array reads as "not
  mentioned". Returning a tenant to the shipped defaults needs an explicit
  six-field PATCH, or DELETE + re-seed.
- **The trade-config seed endpoint is manual-only** (`POST …/trade-configs/seed`).
  Mitigated: an unseeded tenant runs on the full shipped defaults, verified end to
  end — there is no zero-value leak.
- **No atlas-ui editor** for `trade-configs` (explicit PRD non-goal).
- **Enter-refusals are silent on gms_v92 and jms_v185.** Those templates bind 0 and
  2 `enterError` keys respectively, and the code now skips the write rather than
  emitting the `99` sentinel (which the resolver documents as likely to crash the
  client). The refusal still happens; the player just gets no message. Populating
  those tables needs a `CMiniRoomBaseDlg` enter-result pass plus a StringPool read —
  deliberately not filled by copying v83, since that inherited-copy habit is what
  produced the gms_v48 opcode bug this branch fixed.
- **Staging is now an async round trip.** A stage that is slow delays the client's
  unlock rather than corrupting state — `m_bExclRequestSent` *is* the client's own
  in-flight indicator, and that is its native behaviour for every other exclusive
  request. The PRD's 200 ms staging NFR is restated as "the unlock must reach the
  client within 1 s p99" (design §5A.11). A stage that never resolves is bounded by
  the **orchestrator's** saga timeout, whose `SAGA_FAILED` becomes `ITEM_REFUSED`;
  deliberately no second, service-local deadline.

## Failure-path residuals

- **A return can legitimately fail.** If the owner's inventory filled while the
  trade was open, `accept_to_character` fails and the saga compensates, leaving the
  row in escrow. Nothing is lost — the row is durable, still names its owner, is
  visible in the REST surface, and is retried by the next boot sweep — but the
  window lasts until the player makes room or the pod restarts. A ticker was
  considered and rejected (design §5A.8): a background loop re-attempting a saga on
  a cadence with no natural period, for a case the player can clear themselves.
- **The un-accept inverse is not instance-precise.** The compensation for a failed
  `accept_to_character` destroys the *first slot matching the templateId*
  (`saga/compensator.go:661`) — systematically the recipient's pre-existing item,
  not the just-received one. For equips the count is conserved but the stats are
  not. Exact for stackables. Closing it requires atlas-inventory to return the
  created asset id on the compartment `ACCEPTED` event. Same shape as MTS today.
- **A meso refund dispatched during a reverse walk is best-effort.** The walk
  logs and continues when the inverse `award_mesos` fails to dispatch
  (`saga/compensator.go:1260`) rather than aborting the chain, which is correct —
  aborting would strand the remaining inverses — but it means one leg can fail
  while the rest of the compensation lands.
- **The display-failure abort is best-effort.** When a staged item cannot be
  rendered, atlas-channel cancels the trade rather than settling state neither
  client can see; the cancel is a no-op once the room reaches `SETTLING`, and a
  failed cancel is log-only ("the room may remain open until the participant
  changes map, channel or logs out").
- **A restart with rooms in a non-settling state emits no `CANCELLED`.** The
  escrow *is* returned by the boot sweep, so nothing is lost — but nothing tells
  the two clients their room is gone, so both keep a dead dialog until they change
  map, channel or log out. `Reconcile` covers `SETTLING` only, by design.

## Accepted design limitations

- **Cross-family occupancy is best-effort**, checked in atlas-channel. No shared
  occupancy store exists across atlas-trades / atlas-mini-games / atlas-merchant.
- **Serverbound `PLAYER_INTERACTION` remains ❌** in the coverage matrix (PRD
  FR-8.9). That row aggregates ~60 candidate senders and grades worst-of-all; the
  trade codecs cannot move it. The four trade arms were deliberately keyed on their
  own base fnames so unfinished cells could not degrade the 8 currently-green cells.
  All ten of its cells are byte-identical to the merge base.
- **Reconciliation is not leadership-gated.** Safe at `replicas: 1`, and the
  record-delete arbiter is correct under concurrency regardless.
- **FR-3.5 (a staged item cannot be un-staged) is unaffected** and, if anything,
  better justified under escrow: the item is genuinely gone.

## Pre-existing issues found but not fixed here

- The same `99`-sentinel class exists on the non-trade mini-room paths
  (store-permit rejection, merchant and mini-game consumers). Out of scope for a
  trade branch; not made worse by it.
- **`//go:build test` files are never compiled by CI** — `tools/test-all-go.sh` and
  the GitHub action both run `go test ./...` with no `-tags=test`. Seven tagged
  files exist, all in atlas-saga-orchestrator; this branch added none of them and
  every task-205 test is untagged and does run.
- The routes drift check (`deploy/shared/test/routes_nginxt.sh`) is operator-run and
  invoked by nothing in CI.
- `gms_v48` is missing `EXIT: 10` from its interaction handler table (non-trade).

## One defect worth a reviewer's attention

`escrowStore` reads must be rebound onto `emit`'s transaction
(`ProcessorImpl.withTx`). A reader left on the root handle takes a *second*
pooled connection to answer a question asked from inside a transaction — a
deterministic deadlock at pool size 1 and a latent one in production whenever the
pool is exhausted — and it reads outside the transaction, so a command could miss
its own earlier write. Found and fixed while wiring up the escrow store; it is
invisible in normal operation.

## Matrix movement

`✅ 3229 → 3265 (+36)`, `❌ 3582 → 3581`. No cell regressed. The four clientbound
trade arms landed as their own sub-struct rows: 39 verified cells plus one
evidence-backed `n-a` (jms_v185 has no meso-limit arm — its dispatcher at
`0x845d95` is a complete three-case switch with no default).
