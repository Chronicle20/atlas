# Task 27 Review — `parcel_pending` -> CANNOT_TRANSFER_OUT mapping in nine GMS templates

**Commit reviewed:** `283a618da` (range `65d9c2704..283a618da`)
**Brief:** `.superpowers/sdd/plan/task-27-brief.md`
**Report:** `.superpowers/sdd/plan/task-27-report.md`

## Scope

`git diff --stat 65d9c2704..283a618da` shows exactly 9 files changed, 11
insertions(+), 2 deletions(-), all under
`services/atlas-configurations/seed-data/templates/`:

```
template_gms_48_1.json | 3 ++-
template_gms_61_1.json | 3 ++-
template_gms_72_1.json | 1 +
template_gms_79_1.json | 1 +
template_gms_83_1.json | 1 +
template_gms_84_1.json | 1 +
template_gms_87_1.json | 1 +
template_gms_92_1.json | 1 +
template_gms_95_1.json | 1 +
```

The 2-deletion files (`gms_48`, `gms_61`) are the ones where
`mts_listings_open` was previously the last key in the map, so the edit had to
add a trailing comma to that line as well as the new `parcel_pending` line —
correctly a diff of `- "mts_listings_open": N` / `+ "mts_listings_open": N,` /
`+ "parcel_pending": N`. No unrelated files touched. Matches scope described
in the brief. `scope_confirmed`: reviewed the full diff of 283a618da; no file
outside the nine templates was touched, and no other hunk exists inside the
nine files.

## 1. Per-file value correctness — PASS

Derived independently from each file's own diff hunk (not from the report or
controller table, then compared):

| file | `mts_listings_open` (existing) | `parcel_pending` (new) | match |
|---|---|---|---|
| template_gms_48_1.json:3339-3340 | 155 | 155 | yes |
| template_gms_61_1.json:4165-4166 | 179 | 179 | yes |
| template_gms_72_1.json:4425-4426 | 197 | 197 | yes |
| template_gms_79_1.json:4724-4725 | 211 | 211 | yes |
| template_gms_83_1.json:5072-5073 | 222 | 222 | yes |
| template_gms_84_1.json:5138-5139 | 231 | 231 | yes |
| template_gms_87_1.json:4844-4845 | 237 | 237 | yes |
| template_gms_92_1.json:3396-3397 | 60 | 60 | yes |
| template_gms_95_1.json:4951-4952 | 60 | 60 | yes |

Every one of the nine files maps `parcel_pending` to that same file's own
`mts_listings_open` value — no cross-file copy-paste error. These derived
values also match the controller's independently-derived cross-check table
(155/179/197/211/222/231/237/60/60) exactly, which is corroborating, not
substituting, evidence — the primary evidence is the per-file hunk showing
`parcel_pending` on the line immediately after `mts_listings_open` with the
identical numeral.

## 2. JSON validity and placement — PASS

- `python3 -c "json.load(open(f))"` succeeds for all nine touched files —
  each still parses as valid JSON.
- Placement: in every hunk, `"parcel_pending": N` is inserted as a sibling key
  immediately after `"mts_listings_open": N` inside the same object literal
  (same brace scope, same indentation level as `in_family`, `trade_open`,
  `merchant_open`), i.e. the world-transfer block-reason map, not merely
  appended somewhere else in the file. Confirmed for all nine files from the
  diff hunks (e.g. `template_gms_72_1.json:4423-4427`, siblings
  `CANNOT_TRANSFER_NO_EMPTY_SLOTS` / `no_character_slot` immediately follow in
  the same object, unaffected).
- `grep -c '"parcel_pending"'` on each of the nine files returns exactly `1`
  — no duplicate key, no accidental double-application by the script.

## 3. Non-target files — PASS

`git diff 65d9c2704..283a618da -- template_jms_185_1.json
template_gms_12_1.json` returns empty — both files are confirmed untouched,
consistent with the brief's statement that they carry no world-transfer
reason map.

## Not evaluable

None — the diff surface for this unit is small (9 one/two-line JSON
insertions) and fully within read/verify reach; no part of the review
required trusting an external contract this unit doesn't own.

## Verdict rationale

All in-scope checks pass with direct file:line evidence independently
re-derived from the committed tree, not from the report or the supplied
cross-check table. No blocking or non-blocking findings.
