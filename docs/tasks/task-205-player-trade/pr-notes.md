# task-205 player trade — PR notes

Residuals, deploy steps and known gaps carried out of the 24-task implementation.
Everything here was found by review, adjudicated, and deliberately left in this
state — none of it is an oversight.

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

## Failure-path residuals (all require a failed or timed-out settlement)

- **The un-accept inverse is not instance-precise.** `RequestDestroyItem` destroys
  the *first slot matching the templateId* — systematically the recipient's
  pre-existing item, not the just-received one. For equips the count is conserved
  but the stats are not: the giver is restored from the snapshot while the
  recipient's own (possibly better-scrolled) instance is destroyed with nothing to
  restore it. Exact for stackables. Closing it requires atlas-inventory to return
  the created asset id on the compartment `ACCEPTED` event.
- **A recipient who consumes or moves the received item before a timeout-triggered
  walk** causes `RequestDestroyItem` to log "item not found" while the re-grant
  still fires — a dupe. Not closable from the orchestrator; the same shape exists
  in MTS today.
- **A meso-credit rollback can silently fail.** `RequestChangeMeso` returns `nil`
  after emitting `NOT_ENOUGH_MESO`, so if the recipient spends credited meso inside
  the compensation window the giver is still re-credited and the recipient keeps
  it. Narrow: one saga step, and the newest-first walk undoes meso first.
- **The display-failure abort is best-effort.** When a staged item cannot be
  rendered, the trade is cancelled rather than settled invisibly — but the cancel
  is a no-op once the room reaches `SETTLING`, and a failed produce is log-only.
  Both windows need the display failure *and* both confirms inside one Kafka round
  trip.
- **A restart with rooms in a non-settling state** has no counterpart to the
  settlement reconciler: nothing emits `CANCELLED` for live rooms at shutdown, so
  both clients keep a dead dialog and both sides' reservations strand for the
  remaining TTL. `Reconcile` covers `SETTLING` only, by design.

## Accepted design limitations

- **Nothing consumes atlas-inventory's reserve *failure*.** There is no
  `RESERVE_COMMAND_FAILED` in its status contract and the consumer discards the
  error, so closing this is a public contract change touching every reserve
  producer. Mitigated: settlement's pre-check re-reads the compartment, so an item
  that moved or vanished fails the trade rather than transferring wrongly.
- **Every refresh opens a brief no-hold window** between `CANCEL_RESERVATION` and
  `REQUEST_RESERVE` — atlas-inventory has no `RENEW` primitive. Worst case is a
  clean `LEAVE 8`.
- **Cross-family occupancy is best-effort**, checked in atlas-channel. No shared
  occupancy store exists across atlas-trades / atlas-mini-games / atlas-merchant.
- **Serverbound `PLAYER_INTERACTION` remains ❌** in the coverage matrix (PRD
  FR-8.9). That row aggregates ~60 candidate senders and grades worst-of-all; the
  trade codecs cannot move it. The four trade arms were deliberately keyed on their
  own base fnames so unfinished cells could not degrade the 8 currently-green cells.
  All ten of its cells are byte-identical to the merge base.
- **Reconciliation is not leadership-gated.** Safe at `replicas: 1`, and the
  record-delete arbiter is correct under concurrency regardless.

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

## Matrix movement

`✅ 3229 → 3265 (+36)`, `❌ 3582 → 3581`. No cell regressed. The four clientbound
trade arms landed as their own sub-struct rows: 39 verified cells plus one
evidence-backed `n-a` (jms_v185 has no meso-limit arm — its dispatcher at
`0x845d95` is a complete three-case switch with no default).
