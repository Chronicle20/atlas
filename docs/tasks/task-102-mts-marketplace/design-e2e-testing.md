# task-102 addendum — E2E test endpoints for MTS verification

Status: proposed (awaiting approval)
Scope: `services/atlas-mts` + a playbook doc. No channel/UI/schema changes.

## 1. Problem

Every MTS bug on this branch (~20 `fix(mts)` / `fix(saga-orchestrator)` commits)
was found the same way: deploy → click around in the real client → read logs →
fix → redeploy. Two properties make finishing the branch especially expensive:

- **Time**: auction `endsAt` is `now + DurationHours`, config-floored at
  `auctionMinHours` (default 24). Expiry, winning, and escrow release only run
  in the periodic sweep. Vetting "auction expires with no bids" honestly takes
  a day per attempt.
- **Actors**: "someone buys my listing", "someone outbids me", and "I win an
  auction against a competitor" all need a second character driven by a second
  client session.

## 2. Goals / non-goals

Goals — make each of these verifiable in minutes, against a deployed env, with
the real v83 client as the observer:

1. Browse/search/pagination at realistic listing volume.
2. Fixed-price purchase in both directions (I buy; someone buys from me).
3. Auction lifecycle: bid, outbid (escrow release), win at expiry (settle to
   winner + seller credit), expire with no bids (return to seller).
4. Wallet movement + transaction history correctness for all of the above.

Non-goals:

- CI-automated headless E2E (can be layered on these endpoints later).
- Load/perf testing.
- Simulating the client's packet layer (the real client stays the packet-level
  verifier; these endpoints simulate *other people*, not *me*).

## 3. Approaches considered

- **A. Env-gated test resource inside `atlas-mts` (chosen).** New REST routes
  registered only when a test flag is set; they seed data directly and emit the
  same Kafka commands the channel emits for simulated actors. Smallest surface,
  full saga fidelity where it matters, zero prod exposure by default.
- **B. Separate test-harness sidecar service.** Keeps test code out of the
  service binary, but needs its own deploy/route/auth plumbing and duplicates
  the command-envelope types. Too heavy for a per-task verification tool.
- **C. Config-shrink only** (allow minutes-long auctions on a test tenant plus
  manual alt characters). No new code paths, but still needs a second human
  client for two-actor flows, and every scenario remains a manual multi-client
  choreography. Doesn't solve the actual pain.

## 4. Design (approach A)

### 4.1 Gating and access — two independent layers

1. **Env flag**: `main.go` registers the test route initializer only when
   `MTS_TEST_ROUTES_ENABLED=true`. The flag is set in **no** overlay; enable it
   ad hoc with `kubectl set env deployment/atlas-mts MTS_TEST_ROUTES_ENABLED=true`
   (and unset to remove). When unset, the routes do not exist (404), and the
   compiled-in handlers are unreachable.
2. **No ingress route**: the endpoints are deliberately **not** added to
   `deploy/shared/routes.conf`. Even with the flag on, they are unreachable
   from outside the cluster. Access is via
   `kubectl port-forward svc/atlas-mts 8080:8080` + curl.

Endpoints live under a dedicated `/test` prefix and require the standard
tenant headers, like every other REST route (all effects are tenant-scoped).

### 4.2 Endpoints

All bodies are JSON:API input envelopes (`{data:{type,attributes}}`), matching
the service's `RegisterInputHandler` infrastructure.

**1. `POST /test/listings/seed`** — populate sample listings (direct DB insert).

Attributes: `worldId`, `entries[]` — each with `saleType` (`fixed|auction`),
`count`, `templateId`, `quantity`, `listValue`, optional `buyNowPrice`,
`category`/`subCategory`, optional `sellerId`/`sellerName`/`sellerAccountId`
(default: synthetic seller), and for auctions `durationSeconds` (arbitrary —
seconds, not hours; direct insert deliberately bypasses the config min/max so
an auction can expire in 30s) plus optional `startingBid`. A per-call cap
(200 rows) prevents accidental floods.

Each seeded listing is created `active` through the existing `listing`
administrator **with a real ITC serial allocated via the `serial` package** —
the client addresses listings by `nITCSN`, so seeded rows render and are
purchasable/biddable exactly like organic ones. Response echoes the created
listings (`id`, `serial`, `endsAt`).

**2. `POST /test/listings/{listingId}/expire`** — time-travel one auction.

Conditionally sets `ends_at = now − 1s` where the row is an `active` auction;
409 otherwise. Nothing else is touched — expiry/settlement still happens only
via the sweep, i.e. the production path.

**3. `POST /test/sweep`** — run one `task.Sweep` immediately.

Identical call the 60s ticker makes. Returns the swept count. Combined with
(2), an auction outcome is observable seconds after the trigger instead of a
day later.

**4. `POST /test/purchase`** — simulate another character buying a listing.

Attributes: `listingId`, `buyerId`, `buyerAccountId`, `buyNow` (bool —
distinguishes auction buyout from plain fixed-price buy). The handler loads
the listing, then emits the **exact `BuyCommandBody` Kafka command** on
`COMMAND_TOPIC_MTS` (same envelope the channel emits from the client's ITC BUY
arm, keyed by the listing's serial). Returns 202 + the emitted command echo.
From the command onward, 100% of the production path runs: consumer →
processor → saga → orchestrator → wallet debit → custody move → status events
→ channel push to the online seller.

**5. `POST /test/bid`** — simulate a competing bidder.

Attributes: `listingId`, `bidderId`, `bidderAccountId`, `amount` → emits the
exact `PlaceBidCommandBody` command. Enables outbid verification (my escrow
released when the fake bidder outbids me) and win-at-expiry against a
competitor, in either direction.

**NX for the fake buyer/bidder**: no new endpoint — atlas-cashshop already
exposes `PATCH /api/accounts/{accountId}/wallet` (routed in ingress). The
playbook documents topping up the alt's Prepaid before simulated purchases.

### 4.3 Fidelity ledger (what is real vs synthetic)

| Surface | Fidelity |
|---|---|
| purchase / bid simulation | Production path end-to-end from the Kafka command onward; byte-identical envelope to the channel's. |
| sweep trigger | Production `task.Sweep` — same function the ticker calls. |
| expire trigger | The **only** faked datum is `ends_at`; all downstream behavior is production code. |
| seeded listings | Row + serial are real; custody provenance is synthetic (no item was debited from anyone). Buying one runs the full settle; take-home grants a real item from the snapshot. Use a real `sellerAccountId` when the seller-credit side matters. |

### 4.4 Actor prerequisites

Simulated buyers/bidders must be real characters on the tenant (an alt), with
Prepaid NX topped up via the wallet PATCH. Synthetic ids are accepted for
*sellers* of seeded listings only (their role is passive until settle-credit).

### 4.5 Error handling

Standard JSON:API errors. Validation: seed cap, positive prices/quantities,
`expire` requires active auction, `purchase`/`bid` require an existing listing
in a purchasable/biddable state (clear 409 with the actual state otherwise —
the async command would otherwise fail invisibly in the consumer).

### 4.6 Testing the test surface

- Route gating: flag off → routes absent; flag on → registered (httptest).
- Seed: N rows created with distinct serials; browse returns them; auction
  `endsAt` honors `durationSeconds`.
- Expire + sweep: seeded 30s auction → expire → sweep → holding row with
  `origin=expired` (no-bids arm) — reusing the `periodic_test` fixtures.
- Purchase/bid: assert the emitted command JSON equals what the channel-side
  producer emits for the same inputs (pins the fidelity claim).

### 4.7 Playbook

`docs/tasks/task-102-mts-marketplace/e2e-test-playbook.md` — curl recipes per
scenario with expected observations (client view, wallet balances, transaction
history, logs):

1. Seed 60 mixed listings → browse/search/paginate in client.
2. I buy a seeded fixed-price listing (real client, real saga).
3. Fake buyer purchases **my** listing → I see it sold; my points credited.
4. I bid on a seeded auction; fake bidder outbids me → my escrow returns.
5. I bid, expire, sweep → I win: item in my Transfer Inventory, seller credited.
6. Fake bidder wins my auction → settle to them, my points credited.
7. Auction expires with no bids → item back in my Transfer Inventory.
8. Take-home after each settle path; transaction history reflects all rows.

## 5. Rollout / cleanup position

The endpoints ship in the task-102 PR (they are part of making the feature
verifiable, and future MTS regressions need the same loop). Prod safety =
default-off env flag + no ingress route. If we later want them gone from prod
binaries entirely, a build tag can follow — out of scope here.

## 6. Finish-line sequence for the branch

1. Land the test endpoints + playbook (this design).
2. Execute the playbook against the PR env with the real client — the actual
   E2E vetting pass; fix what it surfaces.
3. Run the plan's verification phase (race / vet / build / bake / redis-guard,
   packet matrix cells still promoted).
4. `superpowers:requesting-code-review`, address findings, open the PR.
