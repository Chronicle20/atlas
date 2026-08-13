# task-220 Rollout — Meso Sack Cash Item

## Hard prerequisite: WZ re-ingest per tenant

`atlas-data` stores cash items as JSONB documents (`document.Storage[string, RestModel]`,
kind `CASH`). The new `meso` field is additive: **no existing document gains it on
deploy.** Until a tenant's WZ is re-ingested, `meso` is absent, the handler's
fail-closed guard trips, and every sack use is a logged rejection (loud in logs,
invisible to a smoke test that never uses a sack).

Re-ingest each tenant version, then verify:

    GET /api/data/cash/items/5200000
    GET /api/data/cash/items/5200001
    GET /api/data/cash/items/5200002

Each must return a non-zero `meso`. Record the observed values here per tenant —
do not assume they match GMS 83.1 (1,000,000 / 5,000,000 / 10,000,000); the
per-version amounts were unverified at design time.

| Tenant | 5200000 | 5200001 | 5200002 | Re-ingested |
|---|---|---|---|---|
| gms_v48 | | | | |
| gms_v61 | | | | |
| gms_v72 | | | | |
| gms_v79 | | | | |
| gms_v83 | | | | |
| gms_v84 | | | | |
| gms_v87 | | | | |
| gms_v92 | | | | |
| gms_v95 | | | | |
| jms_v185 | | | | |

Also record, where the item exists: `5202000` (v92/v95/JMS — pays its flat
`info/meso`; the client shows an amount-less "random" prompt on v92/v95, an
accepted cosmetic divergence) and `5200009`/`5200010` (v84/v87/v92/v95 — Maple
Point sacks; these must show no `meso` and must reject).

## Manual acceptance, once per tenant after re-ingest

- [ ] `5200000` on a fresh character: exactly 1 sack removed from CASH, the
      tenant's recorded amount credited, meso chat line renders, client responsive.
- [ ] `5200001` / `5200002`: same, at their recorded amounts.
- [ ] Near-ceiling character: mesos unchanged, sack still in inventory, pink text
      "You cannot hold any more mesos.", client responsive.
- [ ] `5200009` on a v87 tenant: nothing consumed, nothing awarded, warn logged
      naming the item id, client responsive.
- [ ] `5202000` on a v92 tenant: pays the flat `info/meso` amount.

## Known limitations (accepted, from the PRD's non-goals)

- Randomized payout (`mesomin`/`mesomax`/`mesostdev` on `5202000-2`) is not
  implemented; the flat `info/meso` value is paid. A gaussian roll would change
  only the handler's amount resolution — not the saga, not the wire.
- Maple Point sacks are rejected by the zero-amount guard rather than paid.
  Paying NX is a separate cash-shop concern.
- `gms_12` is out of scope: that template does not register
  `CharacterCashItemUseHandle` and no `gms_12` tenant exists.
- Whether JMS v185's `5202000` carries a base `info/meso` is unverified; the
  fail-closed guard covers it either way. The table above resolves it.
