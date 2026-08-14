# Commodity existence sweep — NPC 9090000 shop (task-221, task 13)

Date: 2026-08-13

## Method

Queried the live `atlas-data` service in cluster namespace `atlas-main` for each of
the 26 candidate commodity template ids (PRD FR-5.2) against every GMS tenant it
currently serves. `atlas-data` parses per-tenant WZ trees, so a `200` with a real
body means the id exists in that tenant's ingested item set; a genuinely absent id
returns `404` (confirmed with a sanity probe on a made-up id, see below).

Tenant ids resolved first via:

```
kubectl exec -n atlas-main atlas-tenants-6d8c7f68f8-gtn4v -c atlas-tenants -- \
  wget -O - -q http://localhost:8080/api/tenants
```

Result: 9 GMS tenants (`48, 61, 72, 79, 83, 84, 87, 92, 95`) + 1 JMS tenant
(`185`). **No GMS v12 tenant exists in this environment** — matches the standing
project note that baseline data is loaded for all supported versions except v12.
v12 is therefore `not-ingested` for all 26 rows, per the brief's rule.

For each of the 9 ingested GMS tenants, each id was queried with:

```
kubectl exec -n atlas-main atlas-data-66b8cc4ff7-5jgvz -- \
  wget -O - -q -S "http://localhost:8080/api/data/<path>" \
  --header "TENANT_ID: <tenant-uuid>" --header "REGION: GMS" \
  --header "MAJOR_VERSION: <ver>" --header "MINOR_VERSION: 1"
```

where `<path>` is `consumables/{id}` for all `20xxxxx` ids and `cash/items/{id}`
for `5041000`.

**Sanity check that 404 vs 200 is meaningful** (not a canonical/version-blind
fallback that would make every id "present" regardless of the queried tenant):

- A made-up id, `consumables/2099999` on GMS v83, returned `HTTP/1.1 404 Not Found`.
- `GET /api/data/consumables?page[size]=1` total counts differ sharply per
  tenant: v83 → `"total":2286`, v61 → `"total":1549`, v48 → `"total":820`.
  This confirms each tenant serves a distinct, version-specific consumable set
  (not one shared canonical list), so a `200` on a given tenant genuinely means
  that version's WZ set contains the id.

## Results

`present` / `absent` / `not-ingested` per templateId × version. All 9 ingested
GMS tenants returned `HTTP/1.1 200` with a real (non-stub-error) body for all 26
ids — every commodity from the PRD's FR-5.2 list exists in every ingested GMS
version's baseline, including the pre-v83 legacy versions (48, 61, 72, 79). The
pre-v83-absence candidates flagged in the plan brief (`5041000`,
`2022189`/`90`/`91`/`95`, `2002023`-`025`) turned out to be present on every
ingested version too — verified live, not assumed.

| templateId | v12 | v48 | v61 | v72 | v79 | v83 | v84 | v87 | v92 | v95 |
|---|---|---|---|---|---|---|---|---|---|---|
| 2010003 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2061000 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2060000 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2030000 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2022195 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2020015 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2020014 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2020013 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2020012 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2022190 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2001002 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2001001 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2001000 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2022191 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2022189 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2010004 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2010001 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2010002 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2010000 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2002025 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2002024 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2002023 | not-ingested | present | present | present | present | present | present | present | present | present |
| 5041000 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2022000 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2022003 | not-ingested | present | present | present | present | present | present | present | present | present |
| 2022192 | not-ingested | present | present | present | present | present | present | present | present | present |

## Outcome applied to seed files

- No commodity was `absent` on any ingested version, so no row was dropped from
  any of the 9 ingested-version shop files (`48_1, 61_1, 72_1, 79_1, 83_1, 84_1,
  87_1, 92_1, 95_1`) — each carries the full 26-commodity list, in the order
  given in the plan brief's table (its left column top-to-bottom, then its
  right column top-to-bottom).
- `12_1` is seeded with the same full 26-commodity list per the brief's
  `not-ingested` rule (an extra row for a possibly-nonexistent item is inert in
  `atlas-npc-shops`; a missing row is a visible content gap). **v12 is
  unverified** — its WZ set is not loaded in this environment and could not be
  checked.

## Unverified

- **GMS v12**: no tenant is provisioned in this environment (`GET
  /api/tenants` lists 48/61/72/79/83/84/87/92/95/185 only, no v12). None of the
  26 commodity existence claims could be checked against v12's actual item
  set. Seeded per the brief's `not-ingested` rule rather than guessed.
- **mesoPrice values**: taken directly from the plan brief (PRD FR-5.2), not
  from each item's WZ "price" field (which differs, e.g. `2010003`'s WZ price
  is `50` per `atlas-data`, while the shop's `mesoPrice` is `100` per FR-5.2).
  This is expected — a shop's sell price is a shop-authored value, not the
  item's intrinsic WZ price — but is noted here since the two numbers legally
  diverge and that divergence was directly observed.
