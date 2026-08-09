# Backend Audit — task-204-dual-blade-job-tree (libs/atlas-constants scope)

- **Scope:** `libs/atlas-constants/gen/identities.yaml`, `gen/availability.go`, `job/parents.go`,
  `job/dual_blade_test.go`, the regenerated `*_gen.go` files, and
  `docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv`.
  (Per instructions, `services/atlas-ui/**` and `package-lock.json` in the same commit are
  out of scope for this Go-guidelines audit.)
- **Commit audited:** `fe4a63f92` (task-204) vs `origin/main` (`0a8bdbf70`)
- **Guidelines Source:** backend-dev-guidelines skill (this is a generator/shared-lib change,
  not a REST/Kafka domain service, so the DOM-*/SUB-*/FILE-* checklists that assume
  `processor.go`/`resource.go`/`model.go` shapes do not apply structurally; DOM-21
  "no atlas-constants duplication" is inherently satisfied since this change *is*
  atlas-constants. The applicable bar is generator/source-of-truth correctness.)
- **Date:** 2026-08-09
- **Build:** PASS — `go build ./...` in `libs/atlas-constants` clean.
- **Tests:** PASS — `go test ./... -count=1` in `libs/atlas-constants`: all packages ok
  (`job`, `skill`, `constants`, `field`, `item`, `map`, `merchant`, `monster`, `pet/skill`,
  `summon` — no failures).
- **Generator idempotency:** PASS — `cd gen && go run .` and `go run . -author-semantics`
  regenerate all `*_gen.go` and `gen/semantics/*.yaml` files with **zero** resulting diff
  (`git status --short` empty after both runs). This is the strongest available check for
  this scope: the committed generated files are provably a pure function of the committed
  sources (`identities.yaml`, `availability.go` `classOf`, `divergences.csv`).
- **Guards:** PASS — `tools/skill-job-id-guard.sh` clean (14 divergent consts checked, no
  raw comparisons against the new Dual Blade ids found); `tools/goroutine-guard.sh` clean
  (no bare `go` statements introduced).
- **Overall:** PASS

## Findings

### Source correctness (identities.yaml, availability.go, parents.go, divergences.csv)

| Check | Status | Evidence |
|---|---|---|
| New job identities (430-434) canonicalTokens are unique/contiguous, no collision with existing constants | PASS | `libs/atlas-constants/gen/identities.yaml` new block (job entries `BladeRecruit=430` … `BladeMaster=434`); `go build` succeeded (a duplicate `Identity` map key or duplicate canonicalToken would fail generation/build) |
| New skill identities (43xxxxx) floor-divide by 10000 into the 430-434 range, matching the `classOf` range arm | PASS | `libs/atlas-constants/gen/availability.go:64-70` (new `case t >= 430 && t <= 434: return "DualBlade"`); every skill `canonicalToken` in the new `identities.yaml` block (4300000…4341008) floor-divides into 430-434 |
| `classOf` DualBlade arm placed so it cannot be shadowed by a future broader 4xx arm | PASS (documented risk, correctly called out) | `gen/availability.go:64-70` comment explicitly flags the hazard; `job/dual_blade_test.go:98-116` (`TestDualBladeSitsInsideTheThiefRange`) is a regression test that fails if the range boundary is ever violated |
| `availability.csv` DualBlade rows already existed and are consistent with the new identities (released=false gms 12-87, true gms 92/95/jms185) | PASS | `docs/tasks/task-187-version-aware-id-semantics/audit/availability.csv:72-81,109` — pre-existing rows (not touched by this diff), cross-checked against `job/dual_blade_test.go:29-33` expectations |
| `computeAvailable`/`AvailabilityMap` correctly applies a job-domain CSV row to gate the skill domain too (no per-domain row needed) | PASS | `gen/availability.go:222-231` (`AvailabilityMap{Job: m, Skill: m}` — same underlying matrix for both domains, pre-existing design, unaffected by this change but verified it covers the new skill identities via `releaseEligible` → `classOf("skill", token/10000)`) |
| gms 87 stub handling (WZ presence without release) is modeled the same way as the existing `CygnusStage4` precedent | PASS | `gen/availability.go:64-70` comment + `divergences.csv:20` (`gms,87,1,job,430,DualBlade (unreleased WZ stub)`, pre-existing row); `job/dual_blade_test.go:43-63` (`TestResolveWire_DualBladeStubBoundButUnreleasedAtGms87`) exercises the Resolve/Wire-bound-but-Available-false split |
| `parents.go` roots the 5-step Dual Blade chain at `Rogue` (not `Beginner`), linear, no cycle | PASS | `libs/atlas-constants/job/parents.go:74-79`; `Rogue` confirmed to exist as `Identity = 400` at `job/identities_gen.go:34`; `go build` succeeded (a cycle or reference to an undefined identity would fail compilation); regression pinned by `job/dual_blade_test.go:65-96` (`TestParentWire_DualBladeIsRootedAtRogue`, asserts `ParentWire` not just `ParentIdentity`, across gms 92.1/95.1/jms 185.1) |
| gms92→gms95 Endure→Shadow-Resistance rename for job 431 modeled consistently with the pre-existing Assassin(410)/Bandit(420) analogues | PASS | `identities.yaml` new block: `BladeAcolyteEndure=4310000` / `BladeAcolyteShadowResistance=4310004`; `divergences.csv` new rows (diff lines 9-10) mirror the shape and evidence-citation style of the pre-existing `4100002→4100006` and `4200001→4200006` rows immediately above them in the same file |
| `divergences.csv` new rows are excluded (not silently applied as wire↔identity overrides) and traceable | PASS | Regenerating `gen/semantics/gms_92_1.yaml` and `gms_95_1.yaml` via `-author-semantics` reproduces the committed `excluded:` block additions exactly (excluded-row counts 16→17 and 11→12, diff lines confirmed byte-identical after regen with zero residual `git status` diff) |
| New test file uses table-driven style consistent with sibling `availability_test.go` | PASS | `libs/atlas-constants/job/dual_blade_test.go:14-16` (`var dualBlade = []Identity{...}`) mirrors the pre-existing `cygnusStage4` slice pattern in `job/availability_test.go:7-10`; loops with `t.Errorf`/`t.Fatalf` per entry, no ad hoc single-case assertions |
| No `os.Getenv`, no hand-edited generated files, no manual edits to `*_gen.go` | PASS | All `*_gen.go` diffs are byte-identical to a fresh `go run .` output (verified above); no `os.Getenv` in the diff |

### Caveats (non-blocking, disclosed for the record)

- The WZ display-name and quest-gate evidence cited in the `identities.yaml` and
  `parents.go` comments (e.g. `4300000='Katara Mastery'` via `GET /api/data/skills/{id}`,
  quest 2351 `demandSummary`) could not be independently re-verified against a live
  provisioned tenant from this audit (no live-cluster access in this session's scope).
  The comments cite specific tenant UUIDs, endpoints, and dates, which is the correct
  evidentiary format per the repo's grounding rules; this is flagged as *unverified by
  this audit pass*, not as a defect — nothing in the static sources or generated output
  contradicts the cited evidence, and the divergences.csv rows the comments reference
  (`gms,87,1,job,430` and `gms,92,1,job,430`) do pre-date this commit and match the
  claimed WZ names.
- `libs/atlas-constants/job/model.go:114` and
  `services/atlas-character-factory/atlas.com/character-factory/job/model.go:13` still
  carry a `// jobId = job.BladeRecruit TODO` comment. This predates the diff (neither file
  is touched by commit `fe4a63f92`) and is out of the stated audit scope, but is noted here
  since it is the one loose end this task could plausibly have closed. Recorded as
  non-blocking / out-of-scope, not a finding against this commit.

## Summary

### Blocking (must fix)
- None.

### Non-Blocking (should fix)
- Pre-existing `// TODO` referencing `job.BladeRecruit` in
  `libs/atlas-constants/job/model.go:114` and
  `services/atlas-character-factory/atlas.com/character-factory/job/model.go:13` is now
  resolvable (the identity exists) but was not touched by this commit — flag for a
  follow-up, not this task's scope.

## Worktree Cleanliness

`git status --short` in `.worktrees/task-204-dual-blade-job-tree` is clean of stray edits
from this audit (verified before and after generator re-runs; the only untracked file,
`docs/tasks/task-204-dual-blade-job-tree/frontend-audit.md`, pre-existed this audit session
and was not created or modified by it).
