# Context — task-094 Jobs & Skills Browser

Companion to `plan.md`. Captures the grounded facts an implementer needs that
aren't obvious from the design alone. Everything below was verified against the
working tree on 2026-06-13, not recalled.

## Scope

Frontend-only (`services/atlas-ui`). **No backend changes.** The `atlas-data`
job→skills and skill-definition endpoints, the service modules, and the
`useJobSkills` / `useSkillDefinition` hooks already exist. This task is a static
data file + three pure helpers + one aggregate hook + two pages + wiring.

Run everything from `services/atlas-ui`. `npm` is a broken Windows install under
WSL — **source nvm 22 first** (`source ~/.nvm/nvm.sh && nvm use 22`) before any
`npm` command. Gate on `npm run build` + `npm run test` + **no new lint errors**
(lint baseline is pre-existing-broken; do not gate on clean lint). `npm run
build` (tsc -b) **type-checks `*.test.ts` too**, so test call sites must compile.

## Key existing files (read before touching)

| File | Why it matters |
|------|----------------|
| `src/services/api/skills.service.ts` | `SkillDefinition`, `SkillEffect`, `SkillResource`, `getSkillById`. Task 1 extends all three. `maxLevel` is **not** currently mapped. |
| `src/services/api/jobs.service.ts` | `jobsService.getSkillsByJobId(jobId)` → `number[]`. Used as-is. |
| `src/lib/hooks/api/useJobSkills.ts` | `useJobSkills(tenant, jobId)` + `jobSkillsKeys`. Used as-is. |
| `src/lib/hooks/api/useSkillDefinition.ts` | `useSkillDefinition`, `skillDefinitionKeys`, `SkillDefinitionWithIcon`. Task 2 extracts a shared fetcher here; Task 3's new hook reuses `skillDefinitionKeys.detail` so the cache is shared. |
| `src/lib/utils/asset-url.ts` | `getAssetIconUrl(tenantId, region, major, minor, 'skill', skillId)`. `'skill'` already supported. |
| `src/lib/jobs.ts` | `getJobNameById(id): string | null` + `jobNameMap` (~90 ids). **Reuse for leaf names — do not duplicate.** |
| `src/lib/utils/job-tree.ts` | **Pre-existing** `JOB_TREE` (parent-linked) + `jobTreePath`. Confirms id→name coverage. We do NOT need its parent structure for a 3-level authored tree, but it proves the ids/names. |
| `src/lib/utils/skill-effect-format.ts` | **Pre-existing** statup label map (`LABELS`, PascalCase keys like `WeaponAttack`, `MaxHp`) + `formatStatup`. Task 5 exports this as `STATUP_LABELS` and reuses it — **do not author a second statup label map.** |
| `src/context/tenant-context.tsx` | `useTenant()` → `{ activeTenant }`. `activeTenant.attributes` has `region`, `majorVersion` (number), `minorVersion` (number). |
| `src/App.tsx` | All routes; lazy-import pattern. Task 10 adds `/jobs`, `/jobs/:jobId`. |
| `src/components/app-sidebar.tsx` | Static `items` array; Operations group. Task 10 adds the "Jobs" entry. |
| `src/lib/breadcrumbs/routes.ts` | `ROUTE_CONFIGS` with `labelResolver`. Task 10 adds `/jobs` + `/jobs/[id]`. |
| `src/pages/MonstersPage.tsx` | Reference for `useQueries` fan-out, icon `<img>`, sticky-header `Table`, page layout. |

## Verified data shapes (source of truth: atlas-data Go)

`services/atlas-data/atlas.com/data/skill/rest.go`:
`name`, `description`, `action (bool)`, `element (string)`,
`animationTime (uint32)`, **`maxLevel (uint8, json:"maxLevel")`**, `effects[]`.

`services/atlas-data/atlas.com/data/skill/effect/rest.go` — exact JSON keys the
per-level table enumerates (camelCase, verified):
`weaponAttack`, `magicAttack`, `weaponDefense`, `magicDefense`, `accuracy`,
`avoidability`, `speed`, `jump`, `hp`, `mp`, `hpR`, `mpR`, `MHPRRate`,
`MMPRRate`, `mhpr`, `mmpr`, `HPConsume`, `MPConsume`, `duration`, `overTime
(bool)`, `x`, `y`, `mobCount`, `moneyConsume`, `cooldown`, `morphId`, `prop
(float64)`, `itemConsume`, `itemConsumeAmount`, `damage`, `attackCount`,
`fixDamage`, `bulletCount`, `bulletConsume`, `statups[]`, plus structured
`lt`/`rb`/`monsterStatus`/`cardStats`/`cureAbnormalStatuses` (**out of scope**).

`statup.RestModel`: `type (string)`, `amount (int32)`. Matches the existing
`SkillEffectStatup`. Statup `type` strings are PascalCase enum values
(`WeaponAttack`, `Hp`, `MaxHp`, …) — the same keys `skill-effect-format.ts`
already labels.

## Resolved design decisions carried into the plan

- **Version filter**: static `minMajorVersion` per job node (design OQ-1).
  Concrete integers are authored with a cited basis and **confirmed by a live
  v83 probe** (Task 7) per CLAUDE.md "Verification Over Memory". The mechanism is
  correct regardless of the exact integers.
- **Skill detail surface**: in-page expandable panel, no `/skills/:id` route.
- **Skill-type heuristic**: `statups || overTime → Buff`, else `action → Active`,
  else `Passive`. Single pure helper, degrades safely.
- **Names**: leaf job names resolve via `getJobNameById`; the hierarchy authors
  only archetype/class group labels + `{ jobId, minMajorVersion }`. No name
  duplication.
- **Statup labels**: reuse `skill-effect-format.ts`'s map (exported as
  `STATUP_LABELS`). Scalar fields get a separate `FIELD_LABELS` map (camelCase
  effect keys).

## Dependencies / ordering

Tasks 1→2→3 are a chain (service shape → shared fetcher → aggregate hook).
Tasks 4, 5, 6 are independent pure helpers (any order). Task 7 (probe) confirms
Task 6's constants and should run before final sign-off. Tasks 8–9 (pages)
depend on 1–6. Task 10 (wiring) depends on 8–9. Task 11 is the final gate.
