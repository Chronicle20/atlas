# task-187 — Context & Key Decisions

Companion to `plan.md`. Captures the concrete files, the API contract, and the
non-obvious decisions an implementer must not re-derive. Consumes `prd.md` +
`design.md` (both approved). All paths are worktree-relative.

---

## 1. The one architectural insight that scopes the whole task

**Version-sensitivity lives in exactly one place: the wire-ID ↔ identity
mapping (and the availability set). Every *semantic* property is a property of
the version-independent `Identity`.**

- A wire number means different things per version → `Resolve(wireId) → Identity`
  and `Wire(Identity) → wireId` are the only version-scoped operations, plus
  `Available(Identity) → bool`.
- Once you hold an `Identity`, "is `BrawlerCorkscrewBlow` a keydown skill?",
  "is `SuperGm` a GM job?", "what advancement tier is `Hero`?" are **version-
  independent** facts. So the membership predicates (`IsKeyDownSkill`,
  `IsShootSkill*`, mount, point-reset-excluded, `job.IsA/Is/IsBeginner/IsCygnus/
  Advancement/GetType/getSkillBook/FromSkillId`) get re-expressed **over
  `Identity`** and stay single global tables — the generator does **not** emit
  per-version flags.

This is why the migration is mechanical: at every version-sensitive site,
`Resolve` the incoming wire id to an identity (or `Wire` an identity out), then
run the existing predicate against the identity form.

**Corollary — stable-range sites stay put.** Only the audit's *divergent set*
(GM/SuperGM jobs `500/510/900/910`, Pirate `5xx`, skill `5101004`, the Big Bang
`v0.92` reorg set) actually remaps across versions. Version-stable roots
(Warrior `1xx`, Magician `2xx`, Bowman `3xx`, Thief `4xx`, Cygnus `1xxx`, Aran
`20xx`, Evan `22xx`, beginner roots) do **not** shift across the provisioned GMS
range, so sites keyed solely on them (e.g. `character/model.go:84` `IsCygnus`,
`:100` `IsBeginner`) stay Id-keyed with an inline audit citation. The scoped CI
guard (Task 12) guards **only** the divergent set, so it will not flag them.

## 2. API contract (must be identical everywhere it appears)

New top-level package `libs/atlas-constants/constants`:

```go
type SkillJobSet struct {
    Skill skill.Set
    Job   job.Set
}

// For returns the version-scoped set. region is tenant.Region() (case-insensitive;
// "GMS"/"gms" both match). Unknown (region,major,minor) → canonical baseline
// (GMS 83.1) and a once-per-key warning log.
func For(region string, major, minor uint16) SkillJobSet
```

`libs/atlas-constants/skill`:

```go
type Identity uint32 // distinct type from Id; token == canonical (v83) wire id, opaque

// identity constant name == existing const name minus the "Id" suffix
const SuperGmHide Identity = 9101004
const BrawlerCorkscrewBlow Identity = 5101004
// ... one per current …Id constant, generated

type Set struct { /* immutable; holds the two maps + availability set */ }
func (s Set) Resolve(wireId Id) (Identity, bool)
func (s Set) Wire(id Identity) (Id, bool)
func (s Set) Available(id Identity) bool
func (s Set) AvailableIdentities() []Identity // sorted; for atlas-data enumeration
func (s Set) Name(id Identity) string         // display name from identities.yaml

// version-independent predicates, ported to Identity:
func IsKeyDownSkill(id Identity) bool
func IsShootSkillNotUsingShootingWeapon(id Identity) bool
func IsShootSkillNotConsumingBullet(id Identity) bool
func IsTamedMountSkill(id Identity) bool
func SkillOnlyMountVehicleId(id Identity, level int) (int32, bool)
func IsPointResetExcluded(id Identity) bool
func IsIdentity(id Identity, refs ...Identity) bool // Identity form of Is(...)
```

`libs/atlas-constants/job` (mirror; `job.Id` is `uint16`, `job.Identity uint16`):

```go
type Identity uint16
const SuperGm Identity = 910
const Pirate  Identity = 500
const Gm      Identity = 900
// ...
type Set struct { /* ... */ }
func (s Set) Resolve(wireId Id) (Identity, bool)
func (s Set) Wire(id Identity) (Id, bool)
func (s Set) Available(id Identity) bool
func (s Set) AvailableIdentities() []Identity
func (s Set) Name(id Identity) string

// ported to Identity (canonical-token arithmetic; identity tokens preserve
// canonical numbering, so the existing math ports directly):
func IsAIdentity(id Identity, refs ...Identity) bool
func IsBeginnerIdentity(id Identity) bool
func IsCygnusIdentity(id Identity) bool
func IsFourthJobIdentity(id Identity) bool
func GetTypeIdentity(id Identity) Type
func AdvancementIdentity(id Identity) int
func GetSkillBookIdentity(id Identity) int
func FromSkillIdentity(sid skill.Identity) Identity
```

Identity constant naming rule: **existing const name minus trailing `Id`**
(`SuperGmHideId`→`SuperGmHide`, `PirateId`→`Pirate`). Purely mechanical; the
generator enforces it so no name is invented.

## 3. Generator inputs/outputs (all checked in; generator is offline & pure)

Location: `libs/atlas-constants/gen/` (its own module under `tools/`-style
convention, or a `gen` subpackage run via `go run ./gen`; see plan Task 2).

Inputs (checked in):
- `gen/identities.yaml` — the union namespace: `{name, domain, canonicalToken,
  displayName}` per identity. Bootstrapped mechanically from current
  `constants.go` (name minus `Id`, token = current value), plus Big-Bang
  identities added by the audit.
- `gen/wzsnapshot/<region>_<major>_<minor>.json` — the set of skill/job wire
  numbers that **exist** in that version, drained once from the live atlas-data
  baseline (`GET /api/data/skills`, `/api/data/jobs`), pinned by content hash.
- `gen/semantics/<region>_<major>_<minor>.yaml` — per version `wireId →
  identityName`. Stable entries auto-joined from the snapshot; **divergent
  entries hand-authored with an inline IDB/WZ citation** (the audit's output).
- `gen/availability/<region>_<major>_<minor>.yaml` — released identity names +
  meymink citation per version.

Outputs (generated Go, committed):
- `skill/version_<r>_<maj>_<min>_gen.go`, `job/version_<r>_<maj>_<min>_gen.go` —
  immutable `wireId↔Identity` + availability set per version.
- `skill/identities_gen.go`, `job/identities_gen.go` — the `Identity` constants +
  `Name` table.
- `constants/registry_gen.go` — `map[versionKey]SkillJobSet`.
- `skill/baseline_gen.go`/`job/baseline_gen.go` — the canonical (GMS 83.1) set.

CI drift check re-runs the generator and fails on any diff (spirit of
`tools/*-guard.sh`).

## 4. Provisioned versions (from `deploy/k8s/base/versions.json`)

`gms` 12, 48, 61, 72, 79, 83, 84, 87, 92, 95; `jms` 185 — all `minorVersion 1`.
Canonical baseline = **GMS 83.1**. Note `gms 12` (GMS 0.12, early beta) is
provisioned though prd.md §6 lists only 48+; it is covered by its own snapshot +
a minimal availability manifest so it is not a silent gap. `tenant.Region()`
returns **uppercase** (`"GMS"`); `versions.json` uses lowercase — `For`
normalizes case.

## 5. Grounding sources (no invented values — CLAUDE.md)

- **Semantics divergences**: per-version IDBs via `mcp__ida-pro__func_query`
  (see project memory `reference_ida_mcp_new_api`, `reference_ida_harvest_subagents`);
  corroborated by WZ via the atlas-data live baseline.
- **WZ id-sets**: live atlas-data baseline per version — busybox `wget` of
  `GET /api/data/skills` and `/api/data/jobs` with the four tenant headers (see
  memory `reference_query_atlas_data_skill_per_version`).
- **Availability**: meymink patch log —
  `https://raw.githubusercontent.com/meymink/Maplestory-Patch-Logs` README.md
  (~4700 lines; **WebFetch truncates it — `curl` the raw file then `grep`**).
  Anchors (atlas major = GMS×100): v0.48 Explorers+GM only; **v0.61 last
  pre-Pirate** (WZ Pirate stubs present, not released); **v0.62 Pirate**;
  **v0.80 Aran**; **v0.84 Evan**; **v0.88 Dual Blade**; **v0.92 Big Bang**;
  **v0.95 Mechanic/Resistance**. Cygnus original release pinned by the audit
  (OQ-3).

## 6. Load-bearing files by subsystem (from the exploration pass)

**libs/atlas-constants**
- `skill/constants.go` — `type Id uint32` (L3); 556 `…Id = Id(n)` consts
  (L2907-3464); `var Skills map[Id]Skill` (L2362); `SuperGmHideId` L3253,
  `BrawlerCorkscrewBlowId` L3198.
- `skill/model.go` — `IsKeyDownSkill` (L58-76, 16 ids incl. Corkscrew), `Is`
  (L78-85), shoot lists (L37-56).
- `skill/point_reset.go` — `IsPointResetExcluded` (range+modulo, L7-31).
- `skill/mount.go` — `IsTamedMountSkill`, `SkillOnlyMountVehicleId` (L13-28).
- `job/constants.go` — `type Id uint16` (L3); consts L94-177 (`PirateId=500`
  L130, `BrawlerId=510` L131, `GmId=900` L138, `SuperGmId=910` L139); `var Jobs`
  L9-92; `type Type uint16` L179-185.
- `job/model.go` — `IsA` (L22-30), `Is` branch math `jobId/10` (L32-36),
  `FromSkillId` `floor(sid/10000)` (L38-46), `GetType` `Type(jobId/1000)`
  (L60-66), `GetSkillBook` Evan `jobId-2209` (L68-73), `MpEaterSkillId` map
  (L78-104).
- `job/advancement.go` — `Advancement` range/modulo tiering (L8-23).
- README package index `README.md:18-38` — must be updated (additive-only rule
  `README.md:52-58`).
- go.mod module: `github.com/Chronicle20/atlas/libs/atlas-constants`.
- No existing generator anywhere; model on `tools/packet-audit`
  (`cmd/`/`internal/`/`main.go` + golden `main_test.go`) or `tools/seed-splitters`.

**atlas-channel** (root `services/atlas-channel/atlas.com/channel/`;
`skill2`=atlas-constants skill in most files, but `skill3`=atlas-constants and
`skill2`=local in `socket/*/character_attack_*`):
- `skill/handler/registry.go` — `var registry map[skill2.Id]Handler` (L26),
  `Register` (L30), `Lookup` (L35). **Retarget key to `skill.Identity`.**
- `skill/handler/common.go` — `UseSkill` (L84), dispatch `Lookup(skill2.Id(
  info.SkillId()))` (L170-174); `t := tenant.MustFromContext(ctx)` already at
  L275; crash/dispel `Is` sets L363-388; ShadowStars cmp L94.
- Registrations (blank-imported via `skill/handler/registrations/registrations.go`):
  `hide/hide.go:36`, `healdispel/healdispel.go:28`, `heal/heal.go:42`,
  `mprecovery/mprecovery.go:19`, `mysticdoor/mysticdoor.go:26`,
  `timeleap/timeleap.go:33`, `resurrection/resurrection.go:25-27` (3).
- `hide/hide.go` — gate `job.IsA(c.JobId(), job.SuperGmId)` L65; buff id
  `int32(skill2.SuperGmHideId)` L131/134. `healdispel/healdispel.go` — gate L102.
- `socket/handler/character_skill_prepare.go` — `shouldBroadcastKeydown`
  L27-34, `IsKeyDownSkill` L29-30, call L51.
- `socket/writer/character_attack_common.go` — mastery cascade
  `computeMasteryForWeapon` L69-193 (28 skill + 38 job refs; densest file),
  `getMasteryFromSkill` L195-230; existing version branch `t.MajorVersion()>=95`
  L181 (the selector-shape precedent).
- `socket/handler/character_attack_common.go` — `handler.Lookup(skill3.Id(
  ai.SkillId()))` gate L695 (`t` at L712); MesoExplosion `Is` L673.
- `socket/handler/character_attack_combo.go` — combo id families L42-66,173.
- `socket/handler/character_skill_use.go` — L95/128-130,137 (entry to UseSkill).
- `socket/handler/character_buff_cancel.go` — L28/38/45.
- `socket/writer/npc_shop.go` — mastery slot-max L20-24.
- `kafka/consumer/buff/consumer.go` L187/236, `character/buff/hidden.go` L15 —
  `SuperGmHideId` source-id match.
- `skill/handler/mob_select.go` L77, `skill/handler/mount.go` L52,
  `skill/handler/resurrection/recipients.go` L31.
- Tenant version already in scope at every site via `ctx`.

**atlas-character** (root `services/atlas-character/atlas.com/character/`;
module `atlas-character`):
- `character/processor.go` — `getMaxHpGrowth` L1072-1131, `getMaxMpGrowth`
  L1133-1204, `resolveHPMPGainParams` L1589-1674 (GM/SuperGM line **L1639**),
  level-up driver L1482-1504, `ProcessJobChange` L1693-1771, `RequestDistributeSp`
  L1045 (`FromSkillId`), `computeOnLevelAddedAP` L1556, `computeOnLevelAddedSP`
  L1570, `getSkillBook` **L1773-1775** (`jobId-2209` arithmetic).
  `ProcessorImpl` carries `t tenant.Model` (L145) — version in scope at methods.
- `character/point_reset.go` — AP-reset policy tables L20-126 (`job.Is` roots,
  incl. magician `job.Id(200)` L59); called from `TransferAP` L2095 (`p.t` in
  scope).
- `character/model.go` — `MaxClassLevel` `IsCygnus` L84, `IsBeginner` L100
  (value-receivers, no tenant — **stay Id-keyed**, version-stable ranges).
- `skill/model.go` — `job.FromSkillId(skill.Id(...))` L34.

**atlas-data** (root `services/atlas-data/atlas.com/data/`; module `atlas-data`):
- Sibling resource pattern to copy: `skill/resource.go` (L19-118),
  `job/resource.go` (L20-111), `rest.go` (`RestModel`+`GetName`),
  `processor.go` (docType), route reg in `main.go` L173-200 (jobs=184,skills=183).
- Availability resources are **net-new and backed by `constants.For` directly**
  (not the WZ document store, not a second embedded JSON) — atlas-data already
  imports `libs/atlas-constants`, so this keeps one source of truth. Tenant
  `(region,major,minor)` off `d.Context()` via `tenant.MustFromContext`.

**atlas-ui** (`services/atlas-ui/`):
- `src/lib/hooks/usePresetJobOptions.ts` (L27-31 gate on `useJobs` WZ presence) —
  **the task-186 hook to reconcile**.
- `src/lib/jobs/job-advancement-tree.ts` — `JOB_LIST` L119, `jobName` L130.
- `src/lib/hooks/api/useJobs.ts` (query-hook pattern), `useJobSkills.ts`.
- `src/services/api/jobs.service.ts` (service pattern; `BASE_PATH` L16),
  `src/lib/api/client.ts` (`getListDocument`/`getOne` L353-415; four headers
  injected centrally L102-106), `src/lib/headers.tsx` L3-40.
- Consumers: `components/features/characters/presets/JobCombobox.tsx` (L29-115),
  `JobSkillsAddButton.tsx` (L33-128).
- Test patterns: `hooks/__tests__/usePresetJobOptions.test.tsx`,
  `services/api/__tests__/jobs.service.test.ts`.
- Build type-checks tests; npm needs nvm22; lint pre-broken (memory
  `reference_atlas_ui_npm_nvm_and_lint_baseline`).

## 7. Refinements of the design (recorded so they aren't re-litigated)

1. **Version-independent predicates** (§1 above) — design §8.2 hinted at it
   ("version-stable helpers stay"); this plan makes it the load-bearing rule:
   the generator emits **only** wire↔identity + availability, never per-version
   flags. Reduces generator surface and makes migration mechanical.
2. **atlas-data availability backed by `constants.For` directly** rather than a
   generator-emitted JSON that atlas-data re-embeds (design §7.1). Same external
   contract (atlas-data serves it), one fewer serialization format, zero drift.
3. **Identity token = canonical (v83) wire id**, opaque, arithmetic forbidden
   outside generated code; generator validates token uniqueness per domain and
   errors on collision (none expected for baseline + Big Bang additions).
4. **Stable-range sites stay Id-keyed** with audit citations; only the divergent
   set migrates and is guarded (design §8.3 scope).

## 8. Verification gates (CLAUDE.md) that must pass at the end

`go test -race ./...`, `go vet ./...`, `go build ./...` clean in every changed
module (atlas-constants, atlas-channel, atlas-character, atlas-data); the new
`tools/skill-job-id-guard.sh` + generator drift check clean; `tools/lint.sh
--check`, `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh` clean;
`docker buildx bake` for every service whose `go.mod` was touched (atlas-channel,
atlas-character, atlas-data — atlas-constants is a lib, so bake its consumers);
atlas-ui `npm run build` (type-checks tests) + `vitest` clean.
