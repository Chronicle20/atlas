# Version-Aware Skill/Job ID Semantics — Design

Task: task-187-version-aware-id-semantics
Status: Draft for review
Created: 2026-07-30
Consumes: `prd.md` (approved)

---

## 1. Summary

`libs/atlas-constants` treats a skill/job ID as a stable global identifier. It is
not: the numeric space was reassigned as the game grew (v0.48 `500`=GM vs
v0.62+ `500`=Pirate; skill `5101004`=Super GM Hide vs Brawler Corkscrew Blow;
the Big Bang v0.92 wholesale reorg). Because `atlas-channel` dispatches skill
handlers off a registry **keyed on a fixed wire ID at package `init()`**
(`Register(skill2.SuperGmHideId, Apply)` — no tenant in scope), a v0.48 Super GM
pressing Hide (`5101004`) is dispatched as if it were the v0.83 canonical
`5101004` (Brawler Corkscrew Blow keydown). That is a correctness bug, not a
label bug.

This design models identity on the two independent, version-scoped axes the PRD
requires — **semantics** (what does wire ID `N` mean in version `V`) and
**availability** (is the entity officially released in `V`) — and threads a
single tenant-keyed selector, `constants.For(region, major, minor)`, through
every version-sensitive consumer. Per-version semantic maps are produced by
**full per-version extraction** (chosen fork): each version's complete
identity↔wireId table is materialized from that version's own authoritative
source and checked in as generated Go. Availability is a curated, meymink-cited
manifest per version, served to the frontend by **atlas-data**. A **scoped CI
guard** bans raw comparisons of the audit-identified remapped ranges outside the
resolver.

### Decisions locked in review

- **Generation: full per-version extraction.** The generator materializes the
  entire identity↔wireId table for every provisioned version from that version's
  source-of-record (§4), not a baseline+divergence overlay.
- **Frontend availability: served by atlas-data.** The generator emits an
  availability+names JSON; atlas-data loads it and exposes a tenant-scoped
  resource; atlas-ui fetches it with the standard four headers (§7).
- **CI guard: scoped to the audited divergent set.** Only the remapped
  identities/ranges the audit names are guarded outside the resolver (§8).

---

## 2. The identity model

### 2.1 Identity is a distinct, named, generated type

Introduce a stable **identity** namespace per domain — `skill.Identity` and
`job.Identity` — as **distinct types from the wire `Id`**. Identities are the
durable API: names, not numbers. The identity set is the **union across all
provisioned versions** (FR-1.1); an identity need not exist in every version.

Name-preservation (FR-1.2) is a hard requirement: the meaningful constant names
in use today (`SuperGmHide`, `BrawlerCorkscrewBlow`, `Pirate`, `Gm`, …) become
the identity constant names verbatim, so migration is a name-preserving
substitution wherever a call site already uses a named constant. What changes is
the **type and how the wire number is obtained**: a wire number is no longer a
package-level constant but is resolved from a version set.

An identity's underlying token value is **opaque** — consumers must use the name,
never the number. For grounding and human-readability the generator assigns each
identity a token equal to that entity's wire ID in its **canonical version**
(v0.83 for entities present at baseline; the earliest version that defines it for
Big-Bang-introduced entities). This keeps every identity token traceable to a
real wire number, but the design forbids arithmetic on it outside the generated
resolver.

### 2.2 The two axes are modeled separately

- **Semantics** (§4): per-version generated map `wireId ↔ Identity`, derived from
  that version's authoritative source. Reproducible, checked in, provenance-stamped.
- **Availability** (§5): per-version curated set of released identities, authored
  from the meymink patch log. A **subset** of semantic presence — a WZ stub
  (v0.61 Pirate) is in the semantic map but absent from availability (FR-3.3).

The generator must never conflate the two: an identity in the availability
manifest that has no semantic wireId in that version, or a semantic entry the
audit did not bind, is a generation-time error, not a silent guess (FR-1.3).

---

## 3. Public API surface

Everything below lives in `libs/atlas-constants`. Nothing is runtime-fetched.

```go
// package constants (new top-level selector package)

// For returns the version-scoped set for a tenant. region is tenant.Region();
// major/minor are tenant.MajorVersion()/MinorVersion(). Unknown/unprovisioned
// (region,major,minor) returns the canonical baseline set and logs once (§6).
func For(region string, major, minor uint16) SkillJobSet

type SkillJobSet struct {
    Skill SkillSet
    Job   JobSet
}

// SkillSet / JobSet expose the two primitives plus availability. Typed named
// members are provided as methods so an absent identity is explicit, e.g.
//   wireId, ok := set.Skill.Wire(skill.SuperGmHide)
type SkillSet interface {
    Resolve(wireId skill.Id) (skill.Identity, bool)   // incoming decode
    Wire(id skill.Identity) (skill.Id, bool)          // outgoing emit
    Available(id skill.Identity) bool                 // released in this version
}
type JobSet interface {
    Resolve(wireId job.Id) (job.Identity, bool)
    Wire(id job.Identity) (job.Id, bool)
    Available(id job.Identity) bool
}
```

Underlying primitives (FR-2.3 / §5.4 API) live in the domain packages:
`skill.Identity`, `skill.Resolve/Wire/Available` bound to a version key, and the
same for `job`. `constants.For` composes them so callers have one entry point
(FR-4.2). `tenant.MustFromContext(ctx)` supplies the key at the call site; the
selector itself is context-free and pure (multi-tenancy NFR — no global mutable
state, sets are immutable/generated).

Design note — named-member ergonomics: rather than a giant interface with 638
methods, named access is `set.Skill.Wire(skill.SuperGmHide)` /
`set.Skill.Resolve(wireId)`. The identity constants (`skill.SuperGmHide`, …) are
the "named members"; `Wire`/`Resolve` are the accessors. This keeps the set a
small, testable surface while preserving the durable names.

---

## 4. Per-version semantics — full extraction (generated)

### 4.1 Source-of-record files (checked in, provenance-stamped)

For each provisioned version the repo holds a **source-of-record** data file
under `libs/atlas-constants/gen/semantics/<region>_<major>_<minor>.yaml`. Each
file is the complete `wireId → identityName` table for that version, with a
provenance header recording where each entry came from:

- **Stable entries (~95%):** wireId == canonical identity token. Extracted
  mechanically from that version's WZ id set (the set of skill/job numbers that
  exist), snapshotted once from the atlas-data live baseline
  (`GET /api/data/skills`, `/api/data/jobs` with the tenant's four headers) and
  pinned by snapshot hash. The WZ tells us *which numbers exist*; the mechanical
  join binds each existing number to the identity whose canonical token is that
  number.
- **Divergent entries (the audit set):** hand-authored per version and
  **IDB-corroborated**, because WZ String names are ambiguous or empty at the
  boundaries (v0.48 `5101004`→`SuperGmHide`, `500`→`Gm`, `510`→`SuperGm`; v0.92
  Big Bang remaps). Each divergent entry cites its IDB function/offset or WZ
  evidence inline. These are exactly the entries the multi-boundary audit (§9)
  produces.

Reproducibility (FR-2.4): the source-of-record files are the pinned inputs; the
generator is a **pure function** over them and never touches the network. Regen
after editing a source file is deterministic and diffable.

### 4.2 The generator

`libs/atlas-constants/gen/` holds a Go generator (`go run ./gen`, wired into
`go generate` and a CI drift check). It:

1. Loads the identity namespace (the union — a single `identities.yaml` listing
   every identity name + its canonical token + domain).
2. For each version's source-of-record file, joins wireIds to identities,
   validates (every wireId binds to a known identity; no duplicate wireId; every
   divergent entry carries a citation), and emits
   `libs/atlas-constants/skill/version_<region>_<major>_<minor>_gen.go` and the
   job equivalent — immutable maps `wireId→Identity` and `Identity→wireId`.
3. Loads each version's availability manifest (§5) and emits the availability
   set + the availability/names JSON artifact for atlas-data (§7).
4. Emits the canonical-baseline set (§6) from the v0.83 source-of-record.

Generation is checked in (generated Go, not runtime); a CI job re-runs the
generator and fails on any diff (drift check), same spirit as the existing
`tools/*-guard.sh`.

### 4.3 Why full extraction (recorded, per the chosen fork)

Full extraction materializes each version's complete table rather than inheriting
unchanged entries from a baseline. Cost: every version's WZ id set must be
snapshotted and every divergence hand-bound. Benefit: no version silently depends
on baseline correctness; each version's table stands on its own provenance, and a
future boundary that shifts a currently-"stable" id cannot be missed by an
overlay that assumed it stable. The mechanical stable-id join keeps the hand-work
bounded to the audit's divergent set even though the *output* is a full table.

---

## 5. Availability — curated manifest (served)

### 5.1 Manifest

One human-reviewed file per version under
`libs/atlas-constants/gen/availability/<region>_<major>_<minor>.yaml`, listing
the identity names **officially released** in that version, each with a
release-history citation (meymink patch log line/version). WZ presence and
empty-String names are hints recorded as comments, never the source of truth
(FR-3.2). Availability is a subset of §4 semantic presence (FR-3.3); the
generator errors if a manifest lists an identity with no semantic wireId in that
version.

### 5.2 Anchors (from prd.md §6, meymink; atlas major = GMS ×100)

The manifest authoring uses the grounded anchors: v0.48 Explorers+GM/SuperGM
only; **v0.61 last pre-Pirate** (Pirate stubs present, not released); **v0.62
Pirate**; **v0.80 Aran**; **v0.84 Evan**; **v0.88 Dual Blade**; **v0.92 Big
Bang**; **v0.95 Mechanic/Resistance**. The Cygnus original-release version is
pinned by the audit (§9, OQ-3). JMS v185 gets its own manifest.

### 5.3 `Available`

`set.Skill.Available(id)` / `set.Job.Available(id)` return whether the identity
is in that version's manifest. This is the gate the preset UI reads (§7) and is
distinct from `Wire(...) ok` (semantic presence).

---

## 6. Canonical baseline & fallback

The **canonical baseline** is the **v0.83 GMS** set — the convention the current
638 constants already encode — designated explicitly (FR-4.3, Data Model).
`constants.For` returns it for any `(region, major, minor)` with no generated
set, and logs once per unknown key (observability NFR: fallback logs; a
resolve-miss logs/metrics). Choosing v0.83 means the existing name-preserving
constants remain the default behavior for any not-yet-modeled version, so
migration cannot regress a currently-correct path.

---

## 7. atlas-data availability resource + atlas-ui reconciliation

### 7.1 atlas-data serves availability (chosen fork)

The generator emits a JSON artifact (released identities + version-correct names,
keyed by `(region,major,minor)`) that atlas-data embeds/loads. atlas-data exposes
it as a tenant-scoped JSON:API resource (a sibling to `/api/data/jobs`, e.g.
`GET /api/data/job-availability` and `/api/data/skill-availability`, `GetName()`
= the resource type). The server resolves the tenant's `(region,major,minor)`
from context (four headers) and returns only that version's released set + names.
Canonical-fallback/`tenant_id` scoping follows existing atlas-data `document`
patterns; no new DB table is required for the core (the artifact is served from
the embedded generated data).

### 7.2 atlas-ui

`usePresetJobOptions` (and the job-skill selector) currently gate on
`/api/data/jobs` **WZ presence**, which wrongly offers forward-declared stubs
(v0.61 Pirate) and mislabels (v0.48 `500`="Pirate"). Reconcile it to:

- Gate class/job-skill options on **availability** from §7.1 (FR-6.1) — v0.61
  offers no Pirate; v0.48 offers no Pirate at all.
- Render **names** from the version identity mapping (FR-6.2) — v0.48 `500`
  renders "GM", not "Pirate".

This supersedes task-186's WZ-presence gating. JSON:API conventions + the four
tenant headers apply (frontend-dev-guidelines). This is the real fix, not an
interim UI patch (non-goal: no stopgap).

---

## 8. Consumer migration (full coverage) + CI guard

### 8.1 atlas-channel — identity-based dispatch

The structural fix. Today `skill/handler/registry.go` keys `registry` on
`skill2.Id` and `Register` is called from `init()` with a fixed wire ID. Change:

- The registry keys on `skill.Identity`. `Register(skill.SuperGmHide, Apply)` —
  name preserved, type is now `Identity`.
- Dispatch in `skill/handler/common.go` `UseSkill` (`ctx` in scope): resolve the
  incoming wire id to an identity via the caster's version set, then `Lookup`:
  `set := constants.For(...from ctx...); id, ok := set.Skill.Resolve(skill.Id(info.SkillId())); if ok { Lookup(id) }`.
- `character_attack_common.go:695`'s `handler.Lookup(skill3.Id(ai.SkillId()))`
  registered-check resolves the same way.

Consequence: on a v0.48 tenant, wire `5101004` resolves to `SuperGmHide`,
dispatches the Hide handler, and never reaches the Brawler keydown path. The
v0.48 correctness bug is fixed as a property of the dispatch, not a special case.

### 8.2 Other version-sensitive sites

Migrate every version-sensitive comparison to resolve via the set (FR-5.1):

- `skill/handler/hide/hide.go`, `.../healdispel/healdispel.go` — the `job.IsA(c.JobId(), job.SuperGmId)` gate and `int32(skill2.SuperGmHideId)` buff ids resolve through the set for the caster's version.
- `socket/handler/character_skill_prepare.go` `shouldBroadcastKeydown` / `skill.IsKeyDownSkill` — keydown membership is version-scoped.
- `socket/writer/character_attack_common.go` `computeMasteryForWeapon` / `getMasteryFromSkill` — job/skill-keyed mastery.
- `atlas-character` job-keyed logic (HP/MP gain, advancement) — resolve job identities per version.
- Any additional site the audit (§9) surfaces during the full sweep. Migration is
  exhaustive across all 638 identities and all services (FR-5.3); partial is not
  an acceptable end state.

The version-blind helpers that are genuinely version-stable (e.g. arithmetic
`job.Is` branch math) stay as-is only where the audit confirms no boundary shifts
them; anything the audit flags moves behind the resolver.

### 8.3 The scoped guard (chosen fork)

`tools/skill-job-id-guard.sh` (spirit of the existing guards; wired into CI and
the CLAUDE.md verification list) flags raw literal/constant comparisons **only
against the audit-identified remapped identities/ranges** (GM/SuperGM jobs
`500`/`510`/`900`/`910`, skill `5101004` and the Big Bang reorg set) outside the
resolver package. Precise, low false-positive, and it grows only as the audit
finds new divergences (addresses OQ-6). It does not attempt to guard all 638
constants.

---

## 9. The multi-boundary audit (grounding deliverable)

Produced **first**; it is the concrete input the generator source-of-record files
and manifests consume (FR-7.2). Committed under
`docs/tasks/task-187-version-aware-id-semantics/audit/`. Covers:

- **v0.48 ↔ v0.62:** `5xx` GM↔Pirate, GM/SuperGM `500/510`→`900/910`, skill
  `5101004` SuperGmHide↔BrawlerCorkscrewBlow, the hide/healdispel/keydown skill
  ids. Per-version identity↔wireId + availability, IDB/WZ + meymink cited.
- **Big Bang v0.92–v0.95:** the job/skill reorg. Each affected identity's
  cross-boundary relationship recorded explicitly as same-identity rename, merge,
  or no-counterpart (FR-1.3, OQ-4) — the generator must not guess.
- **Forward-compat review of the v111 reference IDB** — findings recorded, **not
  shipped** (non-goal).
- **Cygnus original GMS release version pinned** (OQ-3, FR-7.3).

Grounding/honesty NFR: every semantic entry traces to an IDB/WZ source, every
availability entry to a release-history citation; no invented values. IDA lookups
use `func_query`/session API per project RE discipline; the meymink raw file is
curl+grep'd (WebFetch truncates it).

---

## 10. Testing

- **Golden anchor tests** (acceptance): v0.48 `5101004`→`SuperGmHide` & `500`→`Gm`;
  v0.62 `500`→`Pirate` & `5101004`→`BrawlerCorkscrewBlow`; one Big Bang remap; a
  v0.61 Pirate stub **present in semantics, absent from availability**.
- **Per-migrated-site table tests** (per version): the v0.48 Hide dispatch is
  handled as Hide, not a Brawler keydown (correctness acceptance); mastery/keydown
  sites resolve correctly per version.
- **Generator determinism test:** regen from pinned source-of-record files
  produces byte-identical Go (drift check).
- **atlas-ui vitest:** preset selectors gate on availability + name by identity
  (v0.48 GM/SuperGM not Pirate; v0.61 no Pirate stubs).
- Builder pattern for test setup; no `*_testhelpers.go`. pgx-free logic
  (resolver, manifests, generated maps) is unit-tested offline.

---

## 11. Component boundaries

| Unit | Purpose | Depends on |
|---|---|---|
| `constants.For` / `SkillJobSet` | Single tenant-keyed entry point composing semantics + availability | skill/job version sets |
| `skill`/`job` `Identity` + `Resolve`/`Wire`/`Available` | Per-domain primitives over generated per-version maps | generated version_*_gen.go |
| generated `version_*_gen.go` | Immutable per-version identity↔wireId maps | source-of-record yaml |
| availability manifests | Curated released-set per version | meymink audit |
| `libs/atlas-constants/gen` | Pure generator: yaml → Go + JSON artifact | source-of-record + manifests |
| atlas-channel identity registry | Version-correct handler dispatch | constants.For |
| atlas-data availability resource | Serve availability+names to UI | generated JSON artifact |
| atlas-ui preset selectors | Gate on availability, name by identity | atlas-data resource |
| `tools/skill-job-id-guard.sh` | Ban raw comparison of remapped ranges | audit divergent set |
| audit (`docs/.../audit/`) | Grounding source for generator+manifests | IDB/WZ/meymink |

Each unit has a single purpose and a well-defined interface; the resolver is the
only place wire↔identity arithmetic lives, and the guard enforces that.

---

## 12. Risks & open items

- **Extraction volume (accepted, chosen fork):** full per-version tables mean a
  WZ id-set snapshot per version. Bounded by pinning snapshots as checked-in
  source-of-record; the hand-work stays on the audit's divergent set.
- **OQ-4 Big Bang relationships:** some skills have no cross-boundary counterpart;
  the audit records this explicitly and the generator treats no-counterpart as
  "absent in that version," never a guess.
- **OQ-5 generator source of record:** per-version IDBs are authoritative for
  divergences; WZ (via atlas-data snapshot) corroborates the stable id set. The
  source-of-record file records which per entry.
- **Guard precision (OQ-6):** scoped-to-divergent-set keeps false positives near
  zero; the tradeoff is the guard only catches the ranges the audit knows — a new
  boundary must extend both the audit and the guard together.
- **JMS v185:** included as a provisioned tenant; its semantics/availability come
  from its own source-of-record + manifest (region in the selector key).

---

## 13. Acceptance mapping

Every prd.md §10 acceptance criterion maps to a section above: two-axis model
(§2/§4/§5), `constants.For` behavior (§3/§6), golden anchors (§10), full
migration + scoped guard (§8), v0.48 correctness (§8.1/§10), preset selectors
(§7), the committed audit (§9), v111 forward-compat review (§9), and the
clean-build/lint/guard gate (CLAUDE.md verification list).
