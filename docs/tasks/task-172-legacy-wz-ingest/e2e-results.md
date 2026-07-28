# task-172 — E2E Ingest Results

End-to-end verification for legacy GMS v12 (monolithic `Data.wz`), GMS v48
(split archives), and JMS v185 (mixed per-image encryption) WZ ingest.

Task 9 splits into two verification layers. The **parser layer** (RC-1..RC-3 —
version+key detection, per-image key fallback, monolithic sub-archives) is
verified here against the real sample archives with zero cluster dependency.
The **service/DB layer** (MinIO fetch → `OpenArchive` → domain workers → tenant
documents; C-3.4 skip log; C-4 String rows; C-5 version warning) requires this
branch's `atlas-data` running in an isolated environment and is tracked below
as pending the `deploy-env` ingest.

## Sample sets

All three sets are present out-of-repo under `/tmp/wz` (verbatim client
archives, never committed):

| Set | Path | Layout |
|---|---|---|
| GMS v12 | `/tmp/wz/GMS/12/Data.wz` (262 MB) | Monolithic single archive |
| GMS v48 | `/tmp/wz/GMS/48/*.wz` (16 archives) | Split per-category |
| JMS v185 | `/tmp/wz/JMS/*.wz` (16 archives incl. `List.wz`) | Split, mixed per-image encryption |

## Layer 1 — Parser verification against the real archives (DONE)

A throwaway in-package harness opened each real archive through this branch's
`wz.Open` and forced the lazy parse of **every image** in the tree
(`Image.Properties()`), which is exactly where the C-2 per-image key fallback
fires. The harness was diagnostic only — deleted after the run, never
committed. Full run detail: `.superpowers/sdd/task-9-local-parse.md`.

| Set | Version detected | File key empty | Dirs | Images | Parsed OK | Parse errors | C-2 fallback hits | Expected magnitude |
|---|---|---|---|---|---|---|---|---|
| GMS v12 monolithic | 12 | true | 41 | 3,613 | 3,613 | **0** | 0 (unencrypted) | ~3,613 — exact match |
| GMS v48 split | 48 | true | 64 | 8,072 | 8,072 | **0** | 0 (unencrypted) | ~8,062 — within 0.1% |
| JMS v185 split (`List.wz` excluded) | 185 | true | 70 | 18,345 | 18,345 | **0** | 2,876 (15.7%) | ~19k — within 3.4% |

What this confirms against the design's residual-risk claims:

- **C-1 (two-phase detection)** — every archive detected its true game version
  (12 / 48 / 185) and correctly resolved the file-level key as empty/None. No
  archive fell into the old RC-1 trap of silently locking the GMS key on
  unencrypted data (that would have produced garbage names and parse errors;
  there were none).
- **C-2 (per-image key fallback)** — the JMS set genuinely mixes encryption:
  **2,876 of 18,345 images (15.7%)** failed under the file-level (empty) key
  and were re-parsed successfully under a different known key. This is
  measured directly from the winning image's `keyOverride`, not inferred from
  logs. `Mob.wz` is **100% fallback** (1,738/1,738), with `Character.wz` (489),
  `Map.wz` (415), and `Npc.wz` (160) also heavily exercising the path. Zero
  fallback failures — the design's stated target.
- **C-3 (monolithic layout, lib)** — the v12 `Data.wz` root exposes 41
  directories (category subdirs) and its images parse via the same reader; the
  monolithic archive is structurally handled by the parser.

Zero parse errors across all three generations (30,030 images total) — nothing
in the parser layer looks like a bug.

## Layer 2 — Service/DB ingest (PENDING isolated deploy)

The following can only be verified by running this branch's `atlas-data` ingest
against the three sets through MinIO + the tenant DB, because they live in the
service wiring, not the parser:

- **C-3 (service)** — `workers.OpenArchive` resolving a per-archive object vs. a
  monolithic `Data.wz` sub-view; `Base.wz` → root view for the character
  smap/zmap sidecars.
- **C-3.4 (skip-tolerance)** — for the v12 monolithic set, the dispatcher should
  **skip** categories absent from `Data.wz` (expect a `QUEST … absent from
  monolithic Data.wz — skipping worker` warn line) while a split-layout miss
  still fails loudly.
- **C-4 (legacy String adapter)** — `item_string_search_index` row count after
  ingesting the v12/v48 String data through the single-`Item.img` path
  (proves the legacy layout adapter harvests flat + nested-Eqp rows).
- **C-5 (version cross-check)** — warn-only line when the detected game version
  disagrees with the ingest params (never a failure).
- **Domain-reader schema drift** — the design flags that old `.img` document
  shapes may drift from current readers (design §Residual risk). This can only
  surface by writing documents to the DB, and is to be fixed iteratively on
  this branch when it appears.

### Why this is not run yet

A truthful service-layer E2E requires **this branch's** `atlas-data` image, not
the released `:latest` running in `atlas-main`. The sanctioned isolation
mechanism is the ephemeral PR environment triggered by the `deploy-env` label
on the branch's PR — a user-gated, resource-heavy deploy that, per the project
workflow, happens after code review opens the PR. Deploying a branch image into
the shared `atlas-main` namespace instead would ingest three legacy data sets
into a shared tenant and is not the correct path.

**To complete Layer 2** (after the PR is open): apply the `deploy-env` label,
then for each set — zip the `.wz` files, upload via the atlas-ui "process data"
flow (or the `.bruno` REST equivalents under `services/atlas-data/.bruno`) into
one tenant/scope per set, run processing, and record here: per-document-type
counts, `item_string_search_index` row count, the C-3.4 skip lines (expect
QUEST skipped for v12), the presence/absence of the C-5 warning, and any parse
warnings. Compare document magnitudes against the parser-layer image counts
above.

## Layer 2 — Service/DB ingest (live run in `atlas-pr-1013`)

Ran the GMS v12 monolithic set through this branch's `atlas-data`
(`atlas-data:pr-1013`) via the real upload → process flow (tenant
`3182f3ec…`, GMS 12.1). The ingest **Job completed successfully**.

Confirmed on live v12 data:

- **C-1 detection:** `Detected version 12 (hash=1651) with encryption=None,
  keyEmpty=true` — matches the parser-layer harness exactly.
- **C-3 monolithic service path:** `monolithic Data.wz detected … serving
  archives as sub-views`. Every category worker resolved its sub-view; maps
  (~482, with names), monsters, npcs, skills (e.g. `1001003` = "Iron Body",
  full effects), and all icon sets (mob 148, item 903, npc 245, skill 175)
  ingested.
- **C-3.4 skip-tolerance:** `QUEST: Quest.wz absent from monolithic Data.wz —
  skipping worker` — the one genuinely-absent category, skipped cleanly; the
  run did not fail.
- **C-5 version cross-check:** no warning (detected 12 == params 12), correct.
- No fatal parse errors; the run's only warnings were benign legacy-layout
  gaps (`MobSkill.img`/`Eqp.img` absent from v12's `String.wz`).

### Defect found and fixed on-branch: C-4 legacy item strings (v12 + v48)

Item search returned **only 6 pets (named "MISSINGNO")** and no real items.
Root cause: real GMS v12 **and** v48 `String.wz` ship the legacy single
`Item.img` (Con 378 / Ins 132 / Etc 372 / Eqp 14 sub-cats / Pet 6) **and** a
standalone `Pet.img`. The `Pet.img.xml` matched `resolveStringSources`'s
modern per-category list, so the resolver misclassified the set as *modern*
and skipped `Item.img` entirely — only the redundant standalone `Pet.img`
was processed. (Verified `String.wz` layouts: v12/v48 → `Item.img`+`Pet.img`,
no per-category; JMS v185 → `Consume/Cash/Etc/Ins/Pet/Eqp`, no `Item.img`.)

Fix: `resolveStringSources` now inverts the priority — **`Item.img` present ⇒
legacy layout, used as the sole (complete) item-string source**; modern
layouts (no `Item.img`) fall through unchanged. Regression test
`TestResolveStringSourcesLegacyItemImgWins`.

**Re-verified** (re-ingest on `atlas-data:pr-1013-05812c0`): the String worker
now logs `legacy String layout detected (single Item.img)`, and
`item_string_search_index` holds **~2,924 rows** (was ~6). `search=Potion`
returns Red/Orange/White/Blue/Dex Potion; item `2000000` = "Red Potion";
nested-Eqp item `1002000` = "Brown Flight Headgear" — confirming the single
`InitStringFlat` pass harvests both flat categories and the nested Eqp
subtree.

### Out of task-172 scope (triaged from the same session)

- **Map rendering fallback** — map *documents* ingested correctly (~482 maps
  with names); rendering is an `atlas-renders` concern downstream of ingest.
- **Skills/jobs list pages** — skill/job *data* ingested (individual skill GET
  works); the list endpoints' HTTP 400 is a pre-existing "list requires a
  filter" behavior, not an ingest gap.
- **Account creation 400** — `atlas-account`, unrelated to WZ ingest.

## Status

- **Parser layer (C-1, C-2, C-3-lib):** verified against all three real sample
  sets — 30,030 images, zero parse errors, C-2 fallback demonstrably exercised.
- **Service/DB layer, GMS v12:** ingest Job succeeded; C-1/C-3/C-3.4/C-5 all
  confirmed live. C-4 item-string defect found, fixed, and **re-verified** on
  the rebuilt image — item search returns real names (~2,924 string rows).
- **GMS v48 + JMS v185 live ingest:** still to run (v48 shares the v12 legacy
  String layout, so the C-4 fix covers it; JMS is modern).
