# Version-Aware Skill/Job ID Semantics — Product Requirements Document

Version: v1
Status: Draft
Created: 2026-07-30

---

## 1. Overview

`libs/atlas-constants` is **version-blind**. Its `skill` package defines 556 `Id`
constants and its `job` package 82 `Id` constants, each a single fixed number on
the **v0.61+/v0.83 convention** (e.g. `job.PirateId = 500`, `job.BrawlerId = 510`,
`job.SuperGmId = 910`, `skill.BrawlerCorkscrewBlowId = 5101004`). Every Atlas
service that keys logic on these constants therefore assumes one version's
numbering.

But a skill/job ID is **not a stable identifier across client versions** — the
numeric space was reassigned as the game grew. Confirmed against live atlas-data
(GMS tenants, atlas-main):

- **v0.48**: job `500`/`510` = **GM/SuperGM** (skills resolve to "Haste (Normal)",
  "Heal + Dispel", "Haste (Super)"), and there is **no** job `900`/`910`. Skill
  `5101004` = **"Super GM Hide"**.
- **v0.62+**: the `5xx` job range and `5xxxxxx` skill range become **Pirate**;
  GM/SuperGM move to `900`/`910`; skill `5101004` = **Brawler Corkscrew Blow**, a
  keydown attack wired into `atlas-channel` skill decision logic
  (`skill/model.go`, IDA-verified for v61/72/79/83/87/95/jms — v48 deliberately
  excluded).

The consequence is **backend correctness bugs**, not just labels: on a v0.48
tenant a Super GM pressing Hide (`5101004`) is misidentified by the shared
constant as a Brawler keydown attack, and the `hide`/`healdispel`/keydown
handlers are keyed on the wrong IDs. It also caused the UI mislabel that surfaced
this: task-186's preset selector offers "Pirate" on v0.48 (where those IDs are
GM/SuperGM).

The numbering shifts again at later boundaries — the **Big Bang** overhaul
(GMS v0.92, "New Formulas") reorganized jobs and skills wholesale, and future
versions (GMS v1.11 / the v111 reference IDB) add more.

### The two-axis model

The root design error is treating ID→meaning as global. It is not; it has **two
independent, version-scoped axes** that must be modeled separately:

1. **Semantics** — *what does ID `N` mean in version `V`?* This is
   **IDB/WZ-grounded and derivable**: each client's own data is authoritative for
   its numbering.
2. **Availability** — *is a class/skill officially released/playable in version
   `V`?* This is **NOT in the WZ** — the WZ forward-declares unreleased content
   (e.g. v0.61's WZ carries empty-name Pirate *stubs*; v0.61 is the last
   pre-Pirate release). Availability is a **curated editorial fact** drawn from
   release history, with the WZ providing only hints (empty String name,
   presence).

### Chosen mechanism: generated per-version constant sets (Option 3)

Each supported version is modeled as its own complete, self-consistent world. A
**generator** produces, per version, a mapping from **stable semantic
identities** (the durable API — `GmHide`, `BrawlerCorkscrewBlow`, `Pirate`, `Gm`)
to that version's wire IDs, derived from that version's authoritative source
(IDB/WZ), and **gated by a curated availability manifest**. Services obtain a
version-scoped set via `constants.For(region, major, minor)` and read named
members that are correct for the tenant by construction. Adding a version = point
the generator at its source + author its availability manifest.

This was chosen over a sparse-override resolver and version-parameterized
accessors because the divergences are not sparse (Big Bang is a wholesale
reorg), the number of boundaries only grows, and generation keeps every version
grounded in ground truth with the least hand-maintained drift.

## 2. Goals

Primary goals:

- Model skill and job identity on the two axes (semantics + availability),
  version-scoped, replacing the single version-blind constant tables.
- Deliver **generated per-version constant sets** with a `constants.For(...)`
  selector keyed on the tenant's `(region, major, minor)`, exposing stable
  semantic names that resolve to version-correct wire IDs.
- Deliver a **curated, repo-owned availability manifest** per supported version,
  grounded in the release-history authority, distinct from WZ presence.
- Provide bidirectional resolution: `Resolve(version, wireId) → identity` (for
  incoming decision logic) and `Wire(version, identity) → wireId` (for emission).
- **Full coverage**: every skill and job identity in the current constant tables,
  and every version-sensitive consumer, is migrated — no half-migrated surface.
- Produce the grounding **multi-boundary audit** (v0.48↔v0.62, Big Bang
  v0.92–v0.95, forward-compat review of the v111 reference IDB) that the
  generator and manifests are built from.
- Reconcile task-186's preset selector so it gates on **availability**, not raw
  WZ presence (which wrongly offers forward-declared stubs like v0.61 Pirate).
- Make the v0.48 backend correctness bugs (hide/healdispel/keydown) go away as a
  consequence of migration.

Non-goals:

- Wire-format / packet / opcode changes (those are the packet-audit track).
- Supporting client versions the project does not provision (v111/GMS v1.11 is
  **reviewed for forward-compat**, not shipped in this pass).
- Non-GMS regions beyond noting region in the selector key (JMS v185 is included
  as a currently-provisioned tenant).
- Any interim/stopgap UI patch — this ships as the real fix (no throwaway work).

## 3. User Stories

- As an **atlas-channel skill handler**, I resolve an incoming skill ID to a
  stable identity for the caster's version, so a v0.48 Super GM Hide is handled
  as Hide and never as a Brawler keydown attack.
- As a **service emitting a skill/job** to a client, I look up the wire ID by
  stable identity for the target version, so the client receives the number *it*
  understands.
- As the **preset UI**, I offer exactly the classes/skills a tenant's version has
  **officially released** (not WZ stubs) and label them by that version's
  identity, so v0.48 shows GM/SuperGM and not Pirate.
- As a **developer adding a new supported version**, I point the generator at its
  IDB/WZ and author its availability manifest, and every consumer is correct
  without touching call sites.
- As a **reviewer**, I can read the curated availability manifest as versioned
  repo config and see exactly which classes/skills each version is asserted to
  have, with its release-history citation.

## 4. Functional Requirements

### 4.1 Semantic identity space
- FR-1.1 Define a stable, version-independent **identity** namespace for skills
  and jobs — the durable API (names, not numbers). It is the **union** across all
  supported versions; an identity need not exist in every version.
- FR-1.2 Identities preserve the meaningful constant names in use today
  (`GmHide`, `BrawlerCorkscrewBlow`, `Pirate`, `Gm`, …) so migration is a
  name-preserving substitution wherever possible.
- FR-1.3 Where Big Bang (or any boundary) renamed/merged/removed an entity, the
  audit records the identity relationship (same-identity rename, merge, or
  no-counterpart) explicitly; the generator must not silently guess.

### 4.2 Per-version semantics (generated)
- FR-2.1 For each supported version, generate an identity↔wireId map from that
  version's authoritative source (IDB, corroborated by WZ via atlas-data).
- FR-2.2 The generated map omits identities absent from that version's data.
- FR-2.3 Provide `Resolve(region, major, minor, wireId) → (identity, ok)` and
  `Wire(region, major, minor, identity) → (wireId, ok)`.
- FR-2.4 Generation is reproducible and checked in (generated Go, not
  runtime-fetched), with the source provenance recorded per version.

### 4.3 Availability (curated manifest)
- FR-3.1 A repo-owned, human-reviewed **availability manifest** per supported
  version lists the identities that are **officially released** in that version.
- FR-3.2 The manifest is authored from the release-history authority
  (meymink patch log) and each version cites its evidence; WZ presence/empty
  String names are hints only, never the source of truth.
- FR-3.3 Availability is a **subset** of semantic presence: an identity present
  in the WZ but not released (e.g. v0.61 Pirate stubs) is in the semantic map but
  NOT in the availability set.
- FR-3.4 Provide `Available(region, major, minor, identity) → bool`.

### 4.4 Version-scoped constant set + selector
- FR-4.1 `constants.For(region, major, minor)` returns a typed set composing 4.2
  and 4.3: named members resolving to version-correct wire IDs, plus availability.
- FR-4.2 The tenant's `(region, major, minor)` comes from
  `tenant.MustFromContext(ctx)`; the selector is the single entry point.
- FR-4.3 Unknown/unprovisioned versions fall back to the canonical baseline set
  and log, rather than panicking (define the canonical baseline explicitly).

### 4.5 Consumer migration (full coverage)
- FR-5.1 Every version-sensitive consumer of `skill`/`job` ID constants is
  migrated to resolve via the version-scoped set — no raw-constant comparisons in
  version-sensitive decision paths. Known break points: `atlas-channel`
  `skill/handler/hide`, `skill/handler/healdispel`, keydown-prepare handler,
  `socket/writer/character_attack_common`; plus `atlas-character` job-keyed logic.
- FR-5.2 A guard (lint/CI, in the spirit of the existing `tools/*-guard.sh`) bans
  raw comparison of the version-remapped ranges outside the resolver, so a new
  unguarded site can't reintroduce the bug.
- FR-5.3 Migration is exhaustive across all 638 identities and all services;
  partial migration is not an acceptable end state (per the "full coverage"
  decision).

### 4.6 Preset UI reconciliation (task-186 follow-through)
- FR-6.1 The preset class + job-skill selectors gate on **availability** for the
  tenant's version, not `/api/data/jobs` WZ presence, so v0.61 does not offer
  Pirate stubs and v0.48 does not offer Pirate at all.
- FR-6.2 Class/skill **names** shown come from the version's identity mapping, so
  v0.48 job `500` renders "GM", not "Pirate".
- FR-6.3 If availability is not yet exposed to the frontend by an API, define how
  the UI obtains it (see §5).

### 4.7 The audit (grounding deliverable)
- FR-7.1 Produce a multi-boundary **divergence + availability audit** covering:
  v0.48↔v0.62 (`5xx` GM↔Pirate, GM/SuperGM `500/510`→`900/910`), Big Bang
  v0.92–v0.95 reorg, and a forward-compat review of the v111 reference IDB.
- FR-7.2 The audit output is the concrete data the generator + manifests consume:
  per-version identity↔wireId and per-version availability, with citations.
- FR-7.3 Pin the release anchors, including the one still open: Cygnus Knights'
  original GMS release (~v0.6x).

## 5. API Surface

New/affected internal APIs (final shapes decided in design):

- `constants.For(region string, major, minor int) SkillJobSet` — version-scoped
  set; `SkillJobSet` exposes typed skill/job members + `Resolve`/`Wire`/
  `Available`.
- `skill.Resolve/Wire/Available(...)`, `job.Resolve/Wire/Available(...)` — the
  underlying primitives.

Frontend-facing (only if 4.6 needs it): if the preset UI must gate on
availability, either atlas-data's `/api/data/jobs` gains an
availability/release signal, or a small availability endpoint/config is exposed.
JSON:API conventions apply; tenant scoped via the standard four headers. This is
an open question (§9).

## 6. Data Model

- **Semantic identity tables** — generated Go per version (checked in), keyed by
  identity, valued by wire ID, with source provenance.
- **Availability manifest** — repo-owned, human-reviewed config per supported
  version (format/location per the "no preference" decision; e.g. one file per
  version), listing released identities with release-history citations.
- **Canonical baseline** — the explicitly-designated default set for
  unknown/unprovisioned versions (candidate: current v0.83 convention).
- No new DB tables required for the core; if 4.6 needs a served availability
  signal, any atlas-data storage follows existing `document` patterns and
  `tenant_id`/canonical-fallback scoping.

### Grounded release-history anchors (from the meymink patch log; atlas major = GMS ×100)

| GMS ver | atlas major | Release-relevant content |
|---|---|---|
| 0.48 (Dec 2007) | 48 | Explorers + GM/SuperGM only (`500`/`510` = GM/SuperGM; no `900`/`910`) |
| 0.61 (Oct 2008) | 61 | "Pre Pirate Quests" — **last pre-Pirate**; WZ has Pirate stubs |
| 0.62 (Nov 2008) | — | **Pirate Class** released |
| 0.72 (Jun 2009) | 72 | Temple of Time; Pirate live; no Cygnus/Aran/Evan |
| 0.79 (Nov 2009) | 79 | "Pre Aran Quests"; Episode 1 |
| 0.80 (Dec 2009) | — | **Aran Class** released |
| 0.83 (Feb 2010) | 83 | Episode One expansion (current constant convention baseline) |
| 0.84 (Mar 2010) | 84 | **Evan Class** released |
| 0.87 (Jun 2010) | 87 | "Pre Dual Blade Quests" |
| 0.88 (Jul 2010) | — | **Dual Blade Class** released |
| 0.92 (Nov 2010) | 92 | **Big Bang / "New Formulas"** — job/skill reorg |
| 0.95 (Jan 2011) | 95 | **Mechanic / Resistance** content |
| ~0.6x | — | Cygnus Knights original release — **pin exact version in audit** |
| 1.11 | 111 | Reference IDB — forward-compat review only |

Provisioned tenants today (atlas-main): GMS v48/61/72/79/83/84/87/92/95, JMS v185.

## 7. Service Impact

- **libs/atlas-constants** — new identity namespace, generated per-version
  semantics, availability manifests, selector + resolver primitives, the
  generator itself. The 638 existing constants become the canonical-baseline
  identity set.
- **atlas-channel** — migrate all version-sensitive skill/job decision sites
  (hide, healdispel, keydown-prepare, attack writer, and any others the audit
  finds) to the resolver; add the CI guard.
- **atlas-character** — migrate job-keyed logic (HP/MP gain, etc.).
- **atlas-ui** — preset selectors gate on availability + name by version identity
  (supersedes task-186's WZ-presence gating in `usePresetJobOptions`).
- **atlas-data** — grounding source for the audit; possibly a served availability
  signal if §5 open question resolves that way.
- **CI / tooling** — generator invocation + drift check; the raw-comparison guard.

## 8. Non-Functional Requirements

- **Correctness first**: the v0.48 hide/healdispel/keydown misfires must be
  provably fixed; add tests per migrated site (table-driven, per version).
- **Grounding/honesty**: every semantic entry traces to an IDB/WZ source and
  every availability entry to a release-history citation; no invented values
  (per project grounding rules).
- **Multi-tenancy**: all resolution keyed on tenant `(region, major, minor)` from
  context; no global mutable state; sets are immutable/generated.
- **Performance**: version-set lookup is O(1)/map; generated at build time, not
  per request.
- **Testability**: the pgx-free logic (resolver, manifests, generated maps) is
  unit-testable; add golden tests pinning key remaps (v0.48 `5101004`=Hide,
  `500`=GM; v0.62 `500`=Pirate).
- **Observability**: unknown-version fallback logs; a metric/log when a resolve
  misses.

## 9. Open Questions

- **OQ-1** Exact generated-set API shape and the canonical-baseline designation.
- **OQ-2** How the frontend obtains availability (served by atlas-data vs
  repo-shared config vs a small endpoint) — FR-6.3.
- **OQ-3** Cygnus Knights' exact original GMS release version (audit pins it).
- **OQ-4** Big Bang identity relationships (renames/merges/removals) — the audit
  must enumerate; some skills have no cross-boundary counterpart.
- **OQ-5** Generator source of record: which per-version IDBs are authoritative,
  and how WZ (via atlas-data) corroborates.
- **OQ-6** Whether the raw-comparison guard can be made precise enough to avoid
  false positives across 638 constants.

## 10. Acceptance Criteria

- [ ] Two-axis model implemented: per-version **semantics** (generated) and
      per-version **availability** (curated manifest) exist for every provisioned
      version.
- [ ] `constants.For(region, major, minor)` returns a version-correct set;
      `Resolve`/`Wire`/`Available` behave per spec, keyed on tenant context.
- [ ] Golden tests pin the anchors: v0.48 `5101004`→`GmHide` & `500`→GM;
      v0.62 `500`→Pirate & `5101004`→BrawlerCorkscrewBlow; a Big Bang remap; a
      v0.61 Pirate stub is **present in semantics but absent from availability**.
- [ ] All 638 identities and all version-sensitive consumers migrated (full
      coverage); the CI guard bans raw comparisons of remapped ranges.
- [ ] v0.48 backend correctness verified: Super GM Hide is handled as Hide, not a
      Brawler keydown attack (test + reasoning).
- [ ] Preset selectors gate on availability and name by version identity: v0.48
      shows GM/SuperGM (not Pirate); v0.61 does not offer Pirate stubs.
- [ ] The multi-boundary audit is committed as the generator/manifest source,
      with release-history citations and the Cygnus anchor pinned.
- [ ] v111/GMS v1.11 reviewed for forward-compat; findings recorded; not shipped.
- [ ] `go test -race ./...` / `go vet` / lint clean in every changed module;
      atlas-ui vitest + build clean; guards clean.
