# Version-Aware Skill/Job ID Semantics Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `libs/atlas-constants`' version-blind skill/job ID constants with per-version, two-axis (semantics + availability) resolution keyed on tenant `(region, major, minor)`, and migrate every version-sensitive consumer so a v0.48 Super GM Hide is handled as Hide, not a Brawler keydown attack.

**Architecture:** A checked-in, offline generator turns per-version source-of-record data (WZ id-set snapshots + hand-authored, IDB/WZ-cited divergences + meymink-cited availability manifests) into immutable generated Go: a stable `Identity` namespace, per-version `wireId↔Identity` maps, per-version availability sets, and a `constants.For(region,major,minor)` selector. **Version-sensitivity is confined to `Resolve`/`Wire`/`Available`; every semantic predicate operates on the version-independent `Identity`.** Consumers resolve the incoming wire id to an identity (or `Wire` an identity out) at each site; a scoped CI guard bans raw comparisons of the audited divergent ranges outside the resolver.

**Tech Stack:** Go 1.24+ (libs/atlas-constants, atlas-channel, atlas-character, atlas-data), YAML source-of-record + a Go generator, JSON:API (api2go) + gorilla/mux (atlas-data), Next.js 16 / React 19 / TanStack React Query / Zod (atlas-ui). Grounding via `mcp__ida-pro__func_query` (IDBs) + live atlas-data baseline (WZ) + meymink patch log (release history).

## Global Constraints

- **Provisioned versions** (from `deploy/k8s/base/versions.json`, the generation set): `gms` 12/48/61/72/79/83/84/87/92/95, `jms` 185 — all `minorVersion 1`. Canonical baseline = **GMS 83.1**.
- **`tenant.Region()` returns uppercase** (`"GMS"`, `"JMS"`); `versions.json` is lowercase — `constants.For` matches case-insensitively.
- **atlas major = GMS ×100** (v48 = GMS 0.48). `skill.Id`/`skill.Identity` are `uint32`; `job.Id`/`job.Identity` are `uint16`.
- **No invented values** (CLAUDE.md): every semantic entry cites an IDB/WZ source; every availability entry cites a meymink line. WZ presence/empty-String names are hints, never truth. Mark anything unverified as "unknown/unverified" and escalate — do not guess.
- **Identity naming rule**: identity constant name = existing `…Id` constant name **minus the trailing `Id`** (`SuperGmHideId`→`SuperGmHide`). Mechanical; the generator enforces it.
- **atlas-constants additions are additive-only** (`README.md:52-58`): do not rename/remove existing `…Id` constants (they remain, still typed `Id`); the new `Identity` layer is added alongside. Update the README package index for new subpackages/types.
- **Generator is offline & pure**: it reads only checked-in files (never the network); regen is byte-deterministic; a CI drift check fails on any diff.
- **Version-scope discipline**: only the audit's *divergent set* remaps (GM/SuperGM jobs `500/510/900/910`, Pirate `5xx`, skill `5101004`, the Big Bang v0.92 reorg set). Version-stable roots (Warrior `1xx`, Magician `2xx`, Bowman `3xx`, Thief `4xx`, Cygnus `1xxx`, Aran `20xx`, Evan `22xx`, beginner roots) do not shift across the provisioned GMS range and stay Id-keyed with an inline audit citation.
- **Grounding tooling**: meymink README is fetched with `curl` (WebFetch truncates it); IDA via `func_query`/session API (confirm the instance matches the version before reading); atlas-data live baseline via busybox `wget` + the four tenant headers.
- **Verification** (CLAUDE.md, all must be clean before "done"): `go test -race ./...`, `go vet ./...`, `go build ./...` per changed module; `tools/lint.sh --check`; `tools/redis-key-guard.sh`; `tools/goroutine-guard.sh`; the new `tools/skill-job-id-guard.sh`; the generator drift check; `docker buildx bake` for each service whose `go.mod` was touched; atlas-ui `npm run build` + `vitest`.

---

## File Structure

**libs/atlas-constants** (module `github.com/Chronicle20/atlas/libs/atlas-constants`)
- Create `gen/go.mod`, `gen/main.go`, `gen/main_test.go` — the offline generator (own module, `tools/`-style).
- Create `gen/identities.yaml` — union identity namespace (name/domain/canonicalToken/displayName).
- Create `gen/wzsnapshot/<r>_<maj>_<min>.json` (11 files) — per-version existing wire-id sets, hashed.
- Create `gen/semantics/<r>_<maj>_<min>.yaml` (11 files) — per-version `wireId→identityName`, divergences cited.
- Create `gen/availability/<r>_<maj>_<min>.yaml` (11 files) — released identity names + meymink citations.
- Create `skill/identity.go` (hand: `type Identity`, `Set`, methods, Identity-keyed predicates), `skill/identities_gen.go`, `skill/version_<r>_<maj>_<min>_gen.go`, `skill/baseline_gen.go`.
- Create `job/identity.go`, `job/identities_gen.go`, `job/version_<r>_<maj>_<min>_gen.go`, `job/baseline_gen.go`.
- Create `constants/for.go` (hand: `For`, `SkillJobSet`, fallback+log), `constants/registry_gen.go`.
- Modify `README.md:18-38` — add `constants` + `Identity` rows.

**docs/tasks/task-187-version-aware-id-semantics/audit/**
- Create `README.md` (narrative), `v048-v062.md`, `bigbang-v092-v095.md`, `v111-forward-compat.md`, `cygnus-anchor.md`, `divergences.csv`, `availability.csv` — the grounding deliverable (Tasks 1, 15).

**services/atlas-channel/atlas.com/channel/**
- Modify `skill/handler/registry.go`, `skill/handler/common.go`, the 7 registration files, `hide/hide.go`, `healdispel/healdispel.go`, `socket/handler/character_skill_prepare.go`, `socket/handler/character_buff_cancel.go`, `socket/writer/character_attack_common.go`, `socket/handler/character_attack_common.go`, `socket/handler/character_attack_combo.go`, `socket/handler/character_skill_use.go`, `socket/writer/npc_shop.go`, `kafka/consumer/buff/consumer.go`, `character/buff/hidden.go`, `skill/handler/mob_select.go`, `skill/handler/mount.go`, `skill/handler/resurrection/recipients.go`.

**services/atlas-character/atlas.com/character/**
- Modify `character/processor.go`, `character/point_reset.go`, `skill/model.go` (leave `character/model.go:84/100` with citation).

**services/atlas-data/atlas.com/data/**
- Create `jobavailability/` (`resource.go`, `rest.go`, `processor.go`) and `skillavailability/` (same); modify `main.go:173-200`.

**services/atlas-ui/src/**
- Create `services/api/availability.service.ts`, `lib/hooks/api/useJobAvailability.ts`, `lib/hooks/api/useSkillAvailability.ts` + tests; modify `lib/hooks/usePresetJobOptions.ts`, `components/features/characters/presets/JobSkillsAddButton.tsx`.

**tools/**
- Create `tools/skill-job-id-guard.sh`; modify CI workflow + CLAUDE.md verification list.

---

## Task 1: The multi-boundary audit (grounding deliverable)

Produced **first** — it is the concrete input every generator source-of-record file consumes. This task authors grounded data, not code; its "test" is a structural validator over the machine-readable output.

**Files:**
- Create: `docs/tasks/task-187-version-aware-id-semantics/audit/README.md`
- Create: `docs/tasks/task-187-version-aware-id-semantics/audit/v048-v062.md`
- Create: `docs/tasks/task-187-version-aware-id-semantics/audit/bigbang-v092-v095.md`
- Create: `docs/tasks/task-187-version-aware-id-semantics/audit/cygnus-anchor.md`
- Create: `docs/tasks/task-187-version-aware-id-semantics/audit/divergences.csv`
- Create: `docs/tasks/task-187-version-aware-id-semantics/audit/availability.csv`

**Interfaces:**
- Produces: `divergences.csv` with header `region,major,minor,domain,wireId,identityName,evidence` (one row per non-stable `wireId→identity` binding, `evidence` = IDB `func_query` name/offset or WZ path). `availability.csv` with header `region,major,minor,domain,identityName,released,meymink` (`released`=`true|false`, `meymink`=patch-log line ref). These two CSVs are the audit's machine-readable contract consumed by Tasks 4 and 5.

- [ ] **Step 1: Pin the meymink release anchors.** `curl -s https://raw.githubusercontent.com/meymink/Maplestory-Patch-Logs/master/README.md -o /tmp/meymink.md` then `grep -niE 'pirate|aran|evan|dual blade|big bang|new formula|cygnus|resistance|mechanic' /tmp/meymink.md`. Record the exact line for each anchor (v0.48 Explorers+GM; v0.61 last pre-Pirate; v0.62 Pirate; v0.80 Aran; v0.84 Evan; v0.88 Dual Blade; v0.92 Big Bang; v0.95 Mechanic/Resistance) and **pin the Cygnus original GMS release version** (OQ-3) into `cygnus-anchor.md`. If a value cannot be confirmed from the log, mark it `UNVERIFIED` and escalate — do not guess.

- [ ] **Step 2: Snapshot each version's WZ id-set from the live baseline.** For each of the 11 provisioned tuples, query the live atlas-data baseline (busybox `wget` of `GET /api/data/skills` drained + `GET /api/data/jobs`, with `TENANT_ID/REGION/MAJOR_VERSION/MINOR_VERSION` headers — see memory `reference_query_atlas_data_skill_per_version`). This produces the raw existing-id sets consumed by Task 3. Record provenance (tenant id, timestamp) in `README.md`.

- [ ] **Step 3: Resolve the divergent semantics from the IDBs.** For the v0.48↔v0.62 boundary confirm, per version, via `mcp__ida-pro__func_query` (select the matching instance first): job `500/510` = GM/SuperGM (v48) vs Pirate/Brawler (v62+); GM/SuperGM at `900/910` (v62+) vs absent (v48); skill `5101004` = SuperGmHide (v48) vs BrawlerCorkscrewBlow (v62+); the hide/healdispel/keydown skill ids per version. Write each finding with its IDB evidence into `v048-v062.md` and as rows in `divergences.csv`.

- [ ] **Step 4: Enumerate the Big Bang v0.92–v0.95 reorg.** For each affected identity record the cross-boundary relationship explicitly as **same-identity rename**, **merge**, or **no-counterpart** (FR-1.3, OQ-4) with IDB/WZ evidence in `bigbang-v092-v095.md`; add the resulting `wireId→identity` rows to `divergences.csv`. A no-counterpart identity is recorded as absent in that version — never guessed.

- [ ] **Step 5: Author availability.** From the Step-1 anchors, write `availability.csv`: for each version × identity in the divergent/class set, `released=true|false` with the meymink line. v0.61 Pirate identities → `released=false` (WZ stub); v0.48 Pirate/Aran/Evan → absent/`false`; GM/SuperGM → `true` per version.

- [ ] **Step 6: Write the structural validator test.** Create `docs/tasks/task-187-version-aware-id-semantics/audit/validate_test.go` (own tiny `package audit` module or a check folded into the generator in Task 2 — put it in the generator module). Test: every `divergences.csv` row has non-empty `evidence`; every `availability.csv` row has non-empty `meymink`; `region`∈{gms,jms}; `(major,minor)` ∈ the provisioned set; `domain`∈{skill,job}. (This becomes `gen/audit_validate_test.go` once the generator module exists — if Task 1 runs first, keep it as a standalone `go run` script `audit/validate.go` and port it in Task 2.)

- [ ] **Step 7: Run the validator.** Run: `go run ./docs/tasks/task-187-version-aware-id-semantics/audit/validate.go` (or the ported test). Expected: `OK: N divergence rows, M availability rows, all cited`.

- [ ] **Step 8: Commit**

```bash
git add docs/tasks/task-187-version-aware-id-semantics/audit
git commit -m "docs(task-187): multi-boundary skill/job semantics + availability audit"
```

---

## Task 2: Identity namespace + generator scaffold

**Files:**
- Create: `libs/atlas-constants/gen/go.mod`
- Create: `libs/atlas-constants/gen/main.go`
- Create: `libs/atlas-constants/gen/identities.go` (identities.yaml loader + emitter)
- Create: `libs/atlas-constants/gen/identities.yaml`
- Create: `libs/atlas-constants/gen/main_test.go`
- Create: `libs/atlas-constants/skill/identities_gen.go` (generated)
- Create: `libs/atlas-constants/job/identities_gen.go` (generated)
- Create: `libs/atlas-constants/skill/identity.go` (hand: `type Identity`)
- Create: `libs/atlas-constants/job/identity.go` (hand: `type Identity`)

**Interfaces:**
- Consumes: Task 1's `divergences.csv`/`availability.csv` (for later tasks; not yet used here). The current `skill/constants.go` + `job/constants.go` const tables (to bootstrap `identities.yaml`).
- Produces: `skill.Identity`/`job.Identity` distinct types; generated identity constants (name = const minus `Id`); `skill.IdentityName(Identity) string` / `job.IdentityName(Identity) string`. The generator binary `go run ./gen` and its `-check` drift mode. `gen.LoadIdentities()` and `gen.EmitIdentities()`.

- [ ] **Step 1: Bootstrap `identities.yaml` mechanically.** Write a one-off extraction (in `gen/main.go` behind a `-bootstrap-identities` flag) that parses `skill/constants.go` and `job/constants.go` const blocks and emits one YAML entry per `…Id` constant: `{name: <const minus Id>, domain: skill|job, canonicalToken: <value>, displayName: <humanized name>}`. Run it, hand-review, then add any Big-Bang-introduced identity names from Task 1's audit that have no v83 constant. Commit `identities.yaml` as the checked-in source.

- [ ] **Step 2: Write the failing generator test for identity emission.**

```go
// gen/main_test.go
func TestEmitIdentities_Golden(t *testing.T) {
    ids, err := LoadIdentities("identities.yaml")
    if err != nil { t.Fatal(err) }
    skillGo, jobGo := EmitIdentities(ids)
    // SuperGmHide is a skill identity with canonical token 9101004
    if !strings.Contains(skillGo, "SuperGmHide Identity = 9101004") {
        t.Fatalf("missing SuperGmHide constant:\n%s", skillGo)
    }
    // Pirate is a job identity with canonical token 500
    if !strings.Contains(jobGo, "Pirate Identity = 500") {
        t.Fatalf("missing Pirate constant:\n%s", jobGo)
    }
    // token uniqueness per domain is validated
    if err := ValidateIdentityTokens(ids); err != nil {
        t.Fatalf("token collision: %v", err)
    }
}
```

- [ ] **Step 3: Run it to confirm it fails.** Run: `cd libs/atlas-constants/gen && go test ./... -run TestEmitIdentities_Golden`. Expected: FAIL (`LoadIdentities`/`EmitIdentities` undefined).

- [ ] **Step 4: Implement the loader/emitter + token validation.** In `gen/identities.go` implement `LoadIdentities`, `EmitIdentities` (emits `skill/identities_gen.go` + `job/identities_gen.go` with a `// Code generated by gen; DO NOT EDIT.` header, ascending by token, each `Name Identity = <token>` plus a `var identityNames = map[Identity]string{…}`), and `ValidateIdentityTokens` (errors on duplicate token within a domain). Hand-write `skill/identity.go` and `job/identity.go` with `type Identity uint32`/`uint16` and `func IdentityName(id Identity) string { return identityNames[id] }`.

- [ ] **Step 5: Run the test to confirm it passes, then generate.** Run: `cd libs/atlas-constants/gen && go test ./...` (PASS), then `go run . ` to emit the `_gen.go` files, then `cd .. && go build ./...` (PASS).

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-constants/gen libs/atlas-constants/skill/identity.go libs/atlas-constants/skill/identities_gen.go libs/atlas-constants/job/identity.go libs/atlas-constants/job/identities_gen.go
git commit -m "feat(atlas-constants): identity namespace + generator scaffold (task-187)"
```

---

## Task 3: Per-version WZ id-set snapshots

**Files:**
- Create: `libs/atlas-constants/gen/wzsnapshot/<r>_<maj>_<min>.json` (11 files)
- Create: `libs/atlas-constants/gen/wzsnapshot/snapshots.go` (loader + hash pin)
- Create: `libs/atlas-constants/gen/wzsnapshot/snapshots_test.go`

**Interfaces:**
- Consumes: Task 1 Step-2 raw baseline drains.
- Produces: `gen.LoadSnapshot(region string, major, minor uint16) (skillIds []uint32, jobIds []uint16, hash string, err error)`. Each JSON file: `{"region","major","minor","hash","skills":[…],"jobs":[…]}` where `hash` = sha256 of the sorted id lists.

- [ ] **Step 1: Materialize the 11 snapshot files** from Task 1's drains — sorted, de-duplicated wire-id arrays, with the sha256 `hash` field filled. (These are the pinned, offline inputs; the generator never re-fetches.)

- [ ] **Step 2: Write the failing loader test.**

```go
func TestLoadSnapshot_HashPinned(t *testing.T) {
    skills, jobs, hash, err := LoadSnapshot("gms", 48, 1)
    if err != nil { t.Fatal(err) }
    if len(skills) == 0 || len(jobs) == 0 { t.Fatal("empty snapshot") }
    if hash != HashIds(skills, jobs) { t.Fatalf("snapshot hash drift: file=%s computed=%s", hash, HashIds(skills, jobs)) }
    // v0.48 has no 900/910 GM jobs at the *900* range only if audit says so; assert Pirate-range 5101004 skill exists
    if !contains32(skills, 5101004) { t.Fatal("expected wire 5101004 present in v48 snapshot") }
}
```

- [ ] **Step 3: Run it — expect FAIL** (`LoadSnapshot` undefined). Run: `cd libs/atlas-constants/gen && go test ./wzsnapshot/ -run TestLoadSnapshot_HashPinned`.

- [ ] **Step 4: Implement `LoadSnapshot`/`HashIds`/`contains32`** in `wzsnapshot/snapshots.go` (read+unmarshal the JSON, recompute + compare hash, sort helpers).

- [ ] **Step 5: Run — expect PASS.** Run: `cd libs/atlas-constants/gen && go test ./wzsnapshot/`.

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-constants/gen/wzsnapshot
git commit -m "feat(atlas-constants): pinned per-version WZ id-set snapshots (task-187)"
```

---

## Task 4: Per-version semantics — source-of-record + generated maps + `Set.Resolve/Wire`

**Files:**
- Create: `libs/atlas-constants/gen/semantics/<r>_<maj>_<min>.yaml` (11 files)
- Create: `libs/atlas-constants/gen/semantics.go` (join + validate + emit)
- Create: `libs/atlas-constants/skill/version_<r>_<maj>_<min>_gen.go` (11, generated)
- Create: `libs/atlas-constants/job/version_<r>_<maj>_<min>_gen.go` (11, generated)
- Modify: `libs/atlas-constants/skill/identity.go` (add `Set` + `Resolve`/`Wire`)
- Modify: `libs/atlas-constants/job/identity.go` (add `Set` + `Resolve`/`Wire`)
- Create: `libs/atlas-constants/gen/semantics_test.go`

**Interfaces:**
- Consumes: `gen.LoadIdentities`, `gen.LoadSnapshot`, Task 1 `divergences.csv`.
- Produces: `skill.Set{}` / `job.Set{}` immutable structs with `Resolve(wireId) (Identity, bool)` + `Wire(Identity) (Id, bool)`; generated `skill.versionSet_gms_83_1` (unexported vars) referenced later by the registry. `gen.EmitSemantics(version) (skillGo, jobGo string, err error)`.

- [ ] **Step 1: Author the 11 semantics YAMLs.** Each file: a `provenance` header + `stable: auto` marker (meaning "join every snapshot id whose value equals a known canonical token to that identity") + an explicit `divergent:` list copied from `divergences.csv` for that version (`wireId → identityName`, each with its `evidence`). The generator will error if a divergent wireId is absent from the snapshot or binds to an unknown identity.

- [ ] **Step 2: Write the failing join/emit test.**

```go
func TestEmitSemantics_v48_HideNotCorkscrew(t *testing.T) {
    m, err := BuildSemantics("gms", 48, 1) // map[wireId]identityName, both domains
    if err != nil { t.Fatal(err) }
    if m.Skill[5101004] != "SuperGmHide" {
        t.Fatalf("v48 wire 5101004 should be SuperGmHide, got %q", m.Skill[5101004])
    }
    if m.Job[500] != "Gm" { t.Fatalf("v48 job 500 should be Gm, got %q", m.Job[500]) }
}

func TestEmitSemantics_v62_CorkscrewAndPirate(t *testing.T) {
    m, err := BuildSemantics("gms", 72, 1) // 72 is the earliest provisioned post-Pirate col
    if err != nil { t.Fatal(err) }
    if m.Skill[5101004] != "BrawlerCorkscrewBlow" { t.Fatal("v72 5101004 should be BrawlerCorkscrewBlow") }
    if m.Job[500] != "Pirate" { t.Fatal("v72 job 500 should be Pirate") }
}
```

- [ ] **Step 3: Run — expect FAIL** (`BuildSemantics` undefined). Run: `cd libs/atlas-constants/gen && go test ./... -run TestEmitSemantics`.

- [ ] **Step 4: Implement `BuildSemantics` + `EmitSemantics`.** `BuildSemantics`: start from the snapshot ids, auto-bind each id whose value == a canonical identity token (skip ids with no matching canonical token unless a divergent entry names them), then overlay the `divergent` list (overriding auto-binds). Validate: no wireId binds to an unknown identity; no duplicate wireId; every divergent entry has evidence + is in the snapshot. `EmitSemantics`: emit `version_<r>_<maj>_<min>_gen.go` per domain with two package-level maps (`wireId→Identity`, `Identity→wireId`) and a constructor `func newSet_<key>() Set`.

- [ ] **Step 5: Add `Set.Resolve`/`Set.Wire`.** In `skill/identity.go` (and `job/identity.go`):

```go
type Set struct {
    byWire     map[Id]Identity
    byIdentity map[Identity]Id
    available  map[Identity]struct{} // filled in Task 5
    names      map[Identity]string   // filled in Task 5
}
func (s Set) Resolve(wireId Id) (Identity, bool) { id, ok := s.byWire[wireId]; return id, ok }
func (s Set) Wire(id Identity) (Id, bool)        { w, ok := s.byIdentity[id]; return w, ok }
```

- [ ] **Step 6: Run tests + generate + build.** Run: `cd libs/atlas-constants/gen && go test ./...` (PASS), `go run .` (emits the 22 version files), `cd .. && go build ./... && go test ./...` (PASS).

- [ ] **Step 7: Commit**

```bash
git add libs/atlas-constants/gen/semantics* libs/atlas-constants/skill/version_*_gen.go libs/atlas-constants/job/version_*_gen.go libs/atlas-constants/skill/identity.go libs/atlas-constants/job/identity.go
git commit -m "feat(atlas-constants): per-version semantics maps + Resolve/Wire (task-187)"
```

---

## Task 5: Availability manifests + generated availability sets + `Available`

**Files:**
- Create: `libs/atlas-constants/gen/availability/<r>_<maj>_<min>.yaml` (11 files)
- Create: `libs/atlas-constants/gen/availability.go`
- Modify: the generated `skill/version_*_gen.go` / `job/version_*_gen.go` (add availability + names into `newSet_<key>`)
- Modify: `libs/atlas-constants/skill/identity.go` / `job/identity.go` (add `Available`, `AvailableIdentities`, `Name`)
- Create: `libs/atlas-constants/gen/availability_test.go`

**Interfaces:**
- Consumes: Task 1 `availability.csv`, `gen.BuildSemantics` (to enforce availability ⊆ semantics).
- Produces: `Set.Available(Identity) bool`, `Set.AvailableIdentities() []Identity` (sorted by wire id), `Set.Name(Identity) string`.

- [ ] **Step 1: Author the 11 availability YAMLs** from `availability.csv`: each lists released identity names + meymink citation. v0.61 Pirate identities omitted (stub, not released); v0.48 omits Pirate/Aran/Evan entirely.

- [ ] **Step 2: Write the failing test — availability ⊂ semantics and the v61-stub rule.**

```go
func TestAvailability_v61PirateStubPresentNotAvailable(t *testing.T) {
    sem, _ := BuildSemantics("gms", 61, 1)
    avail, err := BuildAvailability("gms", 61, 1)
    if err != nil { t.Fatal(err) }
    // Pirate exists in v61 WZ semantics (stub) ...
    if _, ok := sem.JobByName["Pirate"]; !ok { t.Fatal("v61 semantics should contain Pirate stub") }
    // ... but is NOT released (availability)
    if avail.Job["Pirate"] { t.Fatal("v61 Pirate must be present-but-unavailable") }
}
func TestAvailability_SubsetOfSemantics(t *testing.T) {
    if err := ValidateAvailabilitySubset("gms", 84, 1); err != nil { t.Fatal(err) }
}
```

- [ ] **Step 3: Run — expect FAIL** (`BuildAvailability`/`ValidateAvailabilitySubset` undefined). Run: `cd libs/atlas-constants/gen && go test ./... -run TestAvailability`.

- [ ] **Step 4: Implement `BuildAvailability` + `ValidateAvailabilitySubset`**, and extend `EmitSemantics` so `newSet_<key>` also populates `available` and `names` (name from `identities.yaml` displayName). Add the accessors to `identity.go`:

```go
func (s Set) Available(id Identity) bool { _, ok := s.available[id]; return ok }
func (s Set) Name(id Identity) string    { return s.names[id] }
func (s Set) AvailableIdentities() []Identity {
    out := make([]Identity, 0, len(s.available))
    for id := range s.available { out = append(out, id) }
    sort.Slice(out, func(i, j int) bool { return s.byIdentity[out[i]] < s.byIdentity[out[j]] })
    return out
}
```

- [ ] **Step 5: Run tests + regen + build.** Run: `cd libs/atlas-constants/gen && go test ./...` (PASS), `go run .`, `cd .. && go build ./... && go test ./...` (PASS).

- [ ] **Step 6: Commit**

```bash
git add libs/atlas-constants/gen/availability* libs/atlas-constants/skill libs/atlas-constants/job
git commit -m "feat(atlas-constants): per-version availability + Available/Name (task-187)"
```

---

## Task 6: `constants.For` selector + registry + canonical baseline/fallback

**Files:**
- Create: `libs/atlas-constants/constants/for.go`
- Create: `libs/atlas-constants/constants/registry_gen.go` (generated)
- Modify: `libs/atlas-constants/gen/main.go` (emit `registry_gen.go` + `baseline_gen.go`)
- Create: `libs/atlas-constants/skill/baseline_gen.go`, `libs/atlas-constants/job/baseline_gen.go` (generated)
- Create: `libs/atlas-constants/constants/for_test.go`

**Interfaces:**
- Consumes: the generated `skill.newSet_<key>` / `job.newSet_<key>` constructors, GMS 83.1 as baseline.
- Produces: `constants.For(region string, major, minor uint16) SkillJobSet` and `type SkillJobSet struct { Skill skill.Set; Job job.Set }`.

- [ ] **Step 1: Write the failing selector test.**

```go
func TestFor_ResolvesPerVersion(t *testing.T) {
    v48 := constants.For("GMS", 48, 1)
    id, ok := v48.Skill.Resolve(skill.Id(5101004))
    if !ok || id != skill.SuperGmHide { t.Fatal("v48 5101004 must resolve to SuperGmHide") }

    v72 := constants.For("gms", 72, 1) // case-insensitive region
    id2, ok2 := v72.Skill.Resolve(skill.Id(5101004))
    if !ok2 || id2 != skill.BrawlerCorkscrewBlow { t.Fatal("v72 5101004 must resolve to BrawlerCorkscrewBlow") }
}
func TestFor_UnknownFallsBackToBaseline(t *testing.T) {
    got := constants.For("GMS", 200, 7) // unprovisioned
    want := constants.For("GMS", 83, 1) // canonical baseline
    id, _ := got.Skill.Resolve(skill.Id(5101004))
    wid, _ := want.Skill.Resolve(skill.Id(5101004))
    if id != wid { t.Fatal("unknown version must fall back to GMS 83.1 baseline") }
}
```

- [ ] **Step 2: Run — expect FAIL** (`constants.For` undefined). Run: `cd libs/atlas-constants && go test ./constants/`.

- [ ] **Step 3: Implement `For` + emit the registry.** Extend the generator to emit `constants/registry_gen.go`:

```go
// Code generated by gen; DO NOT EDIT.
package constants
type versionKey struct { region string; major, minor uint16 }
var registry = map[versionKey]SkillJobSet{
    {"GMS", 48, 1}: {Skill: skill.NewSetGMS481(), Job: job.NewSetGMS481()},
    // … all 11 …
}
var baseline = SkillJobSet{Skill: skill.NewSetGMS831(), Job: job.NewSetGMS831()}
```

(Generated exported constructors `skill.NewSetGMS481()` wrap the unexported `newSet_gms_48_1`.) Hand-write `for.go`:

```go
package constants

import ( "strings"; "sync"; "github.com/sirupsen/logrus" )

var loggedMisses sync.Map // versionKey -> struct{}

func For(region string, major, minor uint16) SkillJobSet {
    k := versionKey{strings.ToUpper(region), major, minor}
    if s, ok := registry[k]; ok { return s }
    if _, dup := loggedMisses.LoadOrStore(k, struct{}{}); !dup {
        logrus.StandardLogger().WithFields(logrus.Fields{
            "region": k.region, "major": major, "minor": minor,
        }).Warn("constants.For: unprovisioned version; using GMS 83.1 baseline")
    }
    return baseline
}
```

- [ ] **Step 4: Run tests + regen + build.** Run: `cd libs/atlas-constants/gen && go run . && cd .. && go build ./... && go test ./...` (PASS).

- [ ] **Step 5: Commit**

```bash
git add libs/atlas-constants/constants libs/atlas-constants/skill/baseline_gen.go libs/atlas-constants/job/baseline_gen.go libs/atlas-constants/gen
git commit -m "feat(atlas-constants): constants.For selector + baseline fallback (task-187)"
```

---

## Task 7: Identity-keyed predicates + golden anchors + generator drift CI

**Files:**
- Modify: `libs/atlas-constants/skill/identity.go` (port `IsKeyDownSkill`, shoot lists, mount, point-reset to `Identity`)
- Modify: `libs/atlas-constants/job/identity.go` (port `IsAIdentity`, `IsBeginnerIdentity`, `IsCygnusIdentity`, `IsFourthJobIdentity`, `GetTypeIdentity`, `AdvancementIdentity`, `GetSkillBookIdentity`, `FromSkillIdentity`)
- Create: `libs/atlas-constants/skill/identity_test.go`, `libs/atlas-constants/job/identity_test.go`
- Create: `libs/atlas-constants/constants/golden_test.go`
- Create: `libs/atlas-constants/gen/drift_test.go`
- Modify: the CI workflow under `.github/workflows/` (add generator drift job)

**Interfaces:**
- Consumes: the identity constants + `Set` from Tasks 2/4/5.
- Produces: the version-independent predicate functions listed in `context.md §2`, so consumers can run them after `Resolve`.

- [ ] **Step 1: Write failing predicate tests.**

```go
// skill/identity_test.go
func TestIsKeyDownSkill_Identity(t *testing.T) {
    if !IsKeyDownSkill(BrawlerCorkscrewBlow) { t.Fatal("Corkscrew is a keydown identity") }
    if IsKeyDownSkill(SuperGmHide) { t.Fatal("Hide is not keydown") }
}
// job/identity_test.go
func TestIsAIdentity_GmFamily(t *testing.T) {
    if !IsAIdentity(SuperGm, Gm) { t.Fatal("SuperGm is-a Gm family") }
}
```

- [ ] **Step 2: Run — expect FAIL.** Run: `cd libs/atlas-constants && go test ./skill/ ./job/ -run 'Identity'`.

- [ ] **Step 3: Port the predicates to `Identity`.** Copy the bodies from `skill/model.go`/`job/model.go`, replacing the `Id`-typed lists/args with the identity constants (arithmetic on the canonical token is valid because identity tokens preserve canonical numbering). Example for skill:

```go
func IsKeyDownSkill(id Identity) bool {
    return IsIdentity(id, FirePoisonArchMagicianExplosion, /* …the same 16, as identities… */ BrawlerCorkscrewBlow)
}
func IsIdentity(id Identity, refs ...Identity) bool {
    for _, r := range refs { if id == r { return true } }
    return false
}
```

Job math ports directly, e.g.:

```go
func IsCygnusIdentity(id Identity) bool { return Type(uint16(id)/1000) == TypeCygnus }
func GetSkillBookIdentity(id Identity) int {
    if id >= EvanStage2 && id <= EvanStage10 { return int(uint16(id) - 2209) }
    return 0
}
```

- [ ] **Step 4: Write the golden-anchor acceptance test** (`constants/golden_test.go`) pinning every prd.md §10 anchor:

```go
func TestGoldenAnchors(t *testing.T) {
    cases := []struct{ region string; maj, min uint16; wire uint32; wantSkill skill.Identity }{
        {"GMS", 48, 1, 5101004, skill.SuperGmHide},
        {"GMS", 72, 1, 5101004, skill.BrawlerCorkscrewBlow},
    }
    for _, c := range cases {
        got, ok := constants.For(c.region, c.maj, c.min).Skill.Resolve(skill.Id(c.wire))
        if !ok || got != c.wantSkill { t.Fatalf("%v: got %v", c, got) }
    }
    // v48 job 500 == Gm ; v72 job 500 == Pirate
    if id, _ := constants.For("GMS",48,1).Job.Resolve(job.Id(500)); id != job.Gm { t.Fatal("v48 500 != Gm") }
    if id, _ := constants.For("GMS",72,1).Job.Resolve(job.Id(500)); id != job.Pirate { t.Fatal("v72 500 != Pirate") }
    // v61 Pirate: present in semantics, absent from availability
    v61 := constants.For("GMS",61,1)
    if _, ok := v61.Job.Wire(job.Pirate); !ok { t.Fatal("v61 Pirate must have a wire id (semantic presence)") }
    if v61.Job.Available(job.Pirate) { t.Fatal("v61 Pirate must be unavailable") }
    // one Big Bang remap (fill from audit)
}
```

- [ ] **Step 5: Write the generator drift test** (`gen/drift_test.go`): re-run emission into a temp dir and diff byte-for-byte against the checked-in `_gen.go` files; fail on any difference.

- [ ] **Step 6: Run all — expect PASS.** Run: `cd libs/atlas-constants && go test -race ./... && cd gen && go test ./...`.

- [ ] **Step 7: Wire the drift check into CI.** Add a job to the Go CI workflow that runs `cd libs/atlas-constants/gen && go run . && git diff --exit-code libs/atlas-constants` (fails if regen changed anything). Add `constants` + `Identity` rows to `libs/atlas-constants/README.md:18-38`.

- [ ] **Step 8: Commit**

```bash
git add libs/atlas-constants tools .github
git commit -m "feat(atlas-constants): identity predicates + golden anchors + drift CI (task-187)"
```

---

## Task 8: atlas-channel — identity-keyed registry + dispatch resolution

The structural fix. **v0.48 correctness is proven here.**

**Files:**
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/registry.go`
- Modify: `services/atlas-channel/atlas.com/channel/skill/handler/common.go:170-174`
- Modify: `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go:695`
- Create: `services/atlas-channel/atlas.com/channel/skill/handler/registry_test.go`

**Interfaces:**
- Consumes: `constants.For`, `skill.Identity`, `skill.Set.Resolve`.
- Produces: `Register(id skill.Identity, h Handler)`, `Lookup(id skill.Identity) (Handler, bool)` (retyped); the dispatch helper `resolveIdentity(ctx, wire) (skill.Identity, bool)`.

- [ ] **Step 1: Write the failing v0.48 correctness test.** In `registry_test.go`, register a sentinel handler under `skill.SuperGmHide`, build a v0.48 tenant context, resolve wire `5101004` via `constants.For(t...).Skill.Resolve`, and assert `Lookup` returns the Hide handler — and that resolving `5101004` under a v72 context does **not** return it (it resolves to `BrawlerCorkscrewBlow`, unregistered here).

```go
func TestDispatch_v48HideNotCorkscrew(t *testing.T) {
    called := false
    Register(skill.SuperGmHide, func(l) func(context.Context) func(...) { /* set called=true */ })
    // v48 ctx
    id48, _ := constants.For("GMS",48,1).Skill.Resolve(skill.Id(5101004))
    if _, ok := Lookup(id48); !ok { t.Fatal("v48 5101004 must dispatch Hide") }
    // v72 ctx
    id72, _ := constants.For("GMS",72,1).Skill.Resolve(skill.Id(5101004))
    if _, ok := Lookup(id72); ok { t.Fatal("v72 5101004 must NOT dispatch Hide (it is Corkscrew)") }
}
```

- [ ] **Step 2: Run — expect FAIL** (`Register` still takes `skill2.Id`). Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/ -run TestDispatch_v48`.

- [ ] **Step 3: Retype the registry.** In `registry.go`: `var registry = map[skill.Identity]Handler{}`, `func Register(id skill.Identity, h Handler)`, `func Lookup(id skill.Identity) (Handler, bool)`, `func Unregister(id skill.Identity)` (import `skill` = atlas-constants; keep the existing `skill2` alias convention of the file).

- [ ] **Step 4: Resolve at dispatch.** In `common.go` (tenant already at L275), replace `common.go:170-174` `if h, ok := Lookup(skill2.Id(info.SkillId())); ok {` with:

```go
set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())
if id, ok := set.Skill.Resolve(skill2.Id(info.SkillId())); ok {
    if h, hok := Lookup(id); hok { h(l)(ctx)(wp, f, characterId, info, e) }
}
```

In `socket/handler/character_attack_common.go:695`, replace `handler.Lookup(skill3.Id(ai.SkillId()))` with a resolve-then-Lookup using `constants.For(...)` from the `t` already at L712 (hoist `t` above the gate).

- [ ] **Step 5: Run — expect PASS + build.** Run: `cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/ -run TestDispatch_v48 && go build ./...`. (Registrations in Task 9 still reference the old signature — expect build breaks in the registration packages only; Task 9 fixes them. If subagent-driven, land Tasks 8+9 together before asserting a clean service build.)

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler/registry.go services/atlas-channel/atlas.com/channel/skill/handler/common.go services/atlas-channel/atlas.com/channel/skill/handler/registry_test.go services/atlas-channel/atlas.com/channel/socket/handler/character_attack_common.go
git commit -m "feat(atlas-channel): identity-keyed skill dispatch (task-187)"
```

---

## Task 9: atlas-channel — registrations + hide/healdispel/keydown/buff-cancel

**Files:**
- Modify: `skill/handler/hide/hide.go` (:36, :65, :131, :134), `healdispel/healdispel.go` (:28, :102), `heal/heal.go:42`, `mprecovery/mprecovery.go:19`, `mysticdoor/mysticdoor.go:26`, `timeleap/timeleap.go:33,101-102`, `resurrection/resurrection.go:25-27`, `resurrection/recipients.go:31`
- Modify: `socket/handler/character_skill_prepare.go:27-34,51`, `socket/handler/character_buff_cancel.go:28,38,45`, `character/buff/hidden.go:15`, `kafka/consumer/buff/consumer.go:187,236`
- Modify existing tests in those packages.

**Interfaces:**
- Consumes: `Register(skill.Identity, …)`, `constants.For`, the Identity constants + `skill.IsKeyDownSkill(Identity)`.

- [ ] **Step 1: Update the 7 registration call sites** to pass identities: `Register(skill.SuperGmHide, Apply)`, `Register(skill.SuperGmHealDispel, Apply)`, `Register(skill.ClericHeal, Apply)`, `Register(skill.BrawlerMPRecovery, Apply)`, `Register(skill.PriestMysticDoor, Apply)`, `Register(skill.BuccaneerTimeLeap, Apply)`, and the 3 resurrection identities. (Registration is version-independent — the handler is keyed on the stable identity.)

- [ ] **Step 2: Write the failing keydown-resolution test** for `shouldBroadcastKeydown` — a v0.48 Super GM Hide wire (`5101004`) must **not** be treated as keydown, while a v72 Corkscrew wire (`5101004`) must. Add to `character_skill_prepare` test.

- [ ] **Step 3: Run — expect FAIL.** Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/handler/ -run Keydown`.

- [ ] **Step 4: Migrate keydown + hide/healdispel/buff sites.** `character_skill_prepare.go`: `set := constants.For(t...); id, ok := set.Skill.Resolve(skill.Id(skillId)); return ok && skill.IsKeyDownSkill(id)`. `character_buff_cancel.go:45` same. `hide.go`/`healdispel.go` gates stay (`job.IsA(c.JobId(), job.SuperGmId)`) **only if** the audit confirms the caller's `JobId()` is already version-correct for the tenant; otherwise resolve the job id through `constants.For(...).Job` first — follow the audit. Buff-source-id matches (`hidden.go:15`, `consumer.go`): resolve the source wire to `skill.SuperGmHide` identity and compare, using the version from the buff's tenant context.

- [ ] **Step 5: Run — expect PASS + build the whole service.** Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test ./skill/handler/... ./socket/handler/...`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-channel/atlas.com/channel/skill/handler services/atlas-channel/atlas.com/channel/socket/handler/character_skill_prepare.go services/atlas-channel/atlas.com/channel/socket/handler/character_buff_cancel.go services/atlas-channel/atlas.com/channel/character/buff/hidden.go services/atlas-channel/atlas.com/channel/kafka/consumer/buff/consumer.go
git commit -m "feat(atlas-channel): identity-based hide/healdispel/keydown (task-187)"
```

---

## Task 10: atlas-channel — mastery cascade + remaining sites

**Files:**
- Modify: `socket/writer/character_attack_common.go` (mastery cascade L69-230), `socket/handler/character_attack_combo.go` (L42-66,173), `socket/handler/character_skill_use.go` (L95,128-130), `socket/writer/npc_shop.go` (L20-24), `skill/handler/common.go` (crash/dispel Is sets L363-388, ShadowStars L94, mount L144), `skill/handler/mob_select.go:77`, `skill/handler/mount.go:52`, `socket/handler/character_attack_common.go:673` (MesoExplosion), `socket/writer/character_attack_ranged.go`, `character_attack_melee.go`, `socket/handler/effects.go`.

**Interfaces:**
- Consumes: `constants.For`, identity constants, `job.IsAIdentity`, `skill.IsIdentity`, `skill.SkillOnlyMountVehicleId(Identity,int)`.

- [ ] **Step 1: Write a failing mastery test** proving `computeMasteryForWeapon` picks the right mastery skill for a version-stable job on v83 AND still works on v95 (the existing `>=95` branch), routed through resolution — pin one representative (e.g. Page sword mastery). Since these jobs are version-stable roots, the test asserts the *refactor preserves behavior* (no regression), not a remap.

- [ ] **Step 2: Run — expect FAIL** (helper signature not yet identity-aware). Run: `cd services/atlas-channel/atlas.com/channel && go test ./socket/writer/ -run Mastery`.

- [ ] **Step 3: Migrate the divergent-set sites; leave version-stable roots.** The mastery cascade is keyed on Warrior/Magician/etc. **version-stable** roots — per the Global Constraints, those stay as-is with a one-line audit citation comment. Migrate only the sites that touch the divergent set: any `skill2.Is(...)`/comparison involving GM/SuperGM/Pirate/`5101004`/Big Bang ids, and the generic `Is` membership sets in `common.go:363-388` (resolve the incoming wire to an identity, then `skill.IsIdentity(id, skill.CrusaderArmorCrash, …)`). `mount.go:52`: resolve then `skill.SkillOnlyMountVehicleId(id, level)`. `character_skill_use.go:95/128-130`: resolve then compare to `skill.HeroEnrage`, `skill.DarkKnightBeholder`, etc. `character_attack_common.go:673` MesoExplosion: resolve then `skill.IsIdentity(id, skill.ChiefBanditMesoExplosion)`.

- [ ] **Step 4: Run — expect PASS + full service build.** Run: `cd services/atlas-channel/atlas.com/channel && go build ./... && go test -race ./...`.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-channel
git commit -m "feat(atlas-channel): migrate remaining version-sensitive skill/job sites (task-187)"
```

---

## Task 11: atlas-character — job-keyed logic migration

**Files:**
- Modify: `character/processor.go` (GM/SuperGM + Pirate-root branches: `resolveHPMPGainParams:1639`, `getMaxHpGrowth` pirate L1108-1118, `getMaxMpGrowth` pirate L1174-1179, `ProcessJobChange` pirate roots, `RequestDistributeSp:1045` `FromSkillId`, `getSkillBook:1773`)
- Modify: `character/point_reset.go` (magician `job.Id(200)` L59 stays stable; migrate only GM/Pirate-touching rows if the audit flags them)
- Modify: `skill/model.go:34` (`FromSkillId`)
- Leave with citation: `character/model.go:84` (`IsCygnus`), `:100` (`IsBeginner`) — version-stable.

**Interfaces:**
- Consumes: `constants.For` (via `p.t`), `job.Resolve`, `job.IsAIdentity`, `job.FromSkillIdentity`, `job.GetSkillBookIdentity`.

- [ ] **Step 1: Write the failing v0.48 GM HP test.** A v0.48 GM (job `500`) leveling must hit the **GM/SuperGM 30000/30000** branch (`resolveHPMPGainParams:1639`), NOT the Pirate branch (which also matches `500` on v48 pre-migration). Build a v48 `ProcessorImpl`, call `resolveHPMPGainParams` for a job-500 character, assert `hpLower==30000`.

```go
func TestResolveHPMPGain_v48Gm(t *testing.T) {
    p := newV48ProcessorForJob(t, job.Id(500)) // Builder pattern; tenant GMS 48.1
    lo, hi, _, _ := p.resolveHPMPGainParams(charWithJob(500))
    if lo != 30000 || hi != 30000 { t.Fatalf("v48 job 500 is GM → 30000, got %d/%d", lo, hi) }
}
```

- [ ] **Step 2: Run — expect FAIL** (currently `500` matches the Pirate branch first). Run: `cd services/atlas-character/atlas.com/character && go test ./character/ -run TestResolveHPMPGain_v48Gm`.

- [ ] **Step 3: Migrate the GM/Pirate branches to identity.** At each divergent-set branch, resolve `p.set().Job.Resolve(c.JobId())` once (add a small `func (p *ProcessorImpl) set() constants.SkillJobSet { return constants.For(p.t.Region(), p.t.MajorVersion(), p.t.MinorVersion()) }` helper), then branch on `job.IsAIdentity(id, job.Gm, job.SuperGm)` / `job.IsAIdentity(id, job.Pirate, …)`. Order the GM/SuperGM check **before** the Pirate check so v48 `500`→`Gm` wins. Thread the set into the free funcs `computeOnLevelAddedAP/SP`, `getSkillBook` (add a `job.Identity` param — callers have `p.t`). `getSkillBook`: resolve then `job.GetSkillBookIdentity(id)`. `FromSkillId` sites: `job.FromSkillIdentity(skillIdentity)` after resolving the skill wire.

- [ ] **Step 4: Run — expect PASS.** Run: `cd services/atlas-character/atlas.com/character && go test -race ./character/ ./skill/`.

- [ ] **Step 5: Add the citation comments** to `character/model.go:84,100` and any version-stable branch left in place: `// version-stable per task-187 audit (audit/README.md): Cygnus 1xxx roots unchanged across provisioned GMS versions`.

- [ ] **Step 6: Commit**

```bash
git add services/atlas-character
git commit -m "feat(atlas-character): version-aware GM/Pirate job resolution (task-187)"
```

---

## Task 12: The scoped CI guard

**Files:**
- Create: `tools/skill-job-id-guard.sh`
- Create: `tools/skill-job-id-guard_test` fixtures (a `good`/`bad` sample dir) — optional but recommended
- Modify: the Go CI workflow (`.github/workflows/…`)
- Modify: `CLAUDE.md` verification list (add item)

**Interfaces:**
- Produces: a guard that greps for raw comparisons of the audited divergent ranges outside the resolver package.

- [ ] **Step 1: Write the guard.** `tools/skill-job-id-guard.sh` (POSIX `sh`, `set -eu`): grep the Go tree for literal comparisons against the divergent ids — `500`,`510`,`900`,`910`,`5101004`, and the Big Bang reorg ids from `audit/divergences.csv` — in `==`/`case`/`Is(`/`IsA(` context, **excluding** `libs/atlas-constants/**`, `_gen.go`, and `*_test.go`. Read the divergent id list from `audit/divergences.csv` so the guard grows with the audit (addresses OQ-6). Emit `FILE:LINE` for each hit and exit non-zero if any.

- [ ] **Step 2: Run it — expect it flags nothing now** (all divergent sites migrated). Run: `sh tools/skill-job-id-guard.sh`. Expected: `skill-job-id-guard: clean`.

- [ ] **Step 3: Prove it catches a regression.** Temporarily add `if jobId == job.Id(500) {` to a non-resolver file, run the guard, confirm it fails with that `FILE:LINE`, then revert.

- [ ] **Step 4: Wire into CI + CLAUDE.md.** Add the guard to the Go CI workflow alongside `redis-key-guard.sh`/`goroutine-guard.sh`, and append an item to the CLAUDE.md "Build & Verification" list.

- [ ] **Step 5: Commit**

```bash
git add tools/skill-job-id-guard.sh .github CLAUDE.md
git commit -m "feat(tools): scoped skill/job remapped-id guard (task-187)"
```

---

## Task 13: atlas-data — availability resources

**Files:**
- Create: `services/atlas-data/atlas.com/data/jobavailability/resource.go`, `rest.go`, `processor.go`
- Create: `services/atlas-data/atlas.com/data/skillavailability/resource.go`, `rest.go`, `processor.go`
- Modify: `services/atlas-data/atlas.com/data/main.go:173-200`
- Create: `services/atlas-data/atlas.com/data/jobavailability/resource_test.go`

**Interfaces:**
- Consumes: `constants.For` (atlas-data already imports `libs/atlas-constants`), tenant from `d.Context()`.
- Produces: `GET /api/data/job-availability` and `/api/data/skill-availability` returning released `{wireId, name}` for the tenant version. `GetName()` = `"job-availability"` / `"skill-availability"`.

- [ ] **Step 1: Write the failing resource test.** Build a request with GMS 48.1 headers to `handleGetJobAvailability`, assert the response includes `{id:500, name:"GM"}` and **excludes** any Pirate entry; a GMS 72.1 request includes `{id:500, name:"Pirate"}`.

- [ ] **Step 2: Run — expect FAIL** (package doesn't exist). Run: `cd services/atlas-data/atlas.com/data && go test ./jobavailability/`.

- [ ] **Step 3: Implement, mirroring `job/`.** `rest.go`: `RestModel{ Id uint32; Name string }` (or `uint16` for jobs) + `GetName()`. `processor.go`: resolve `t := tenant.MustFromContext(ctx)`, `set := constants.For(t.Region(), t.MajorVersion(), t.MinorVersion())`, iterate `set.Job.AvailableIdentities()`, for each `wire,_ := set.Job.Wire(id)` build `{Id: uint32(wire), Name: set.Job.Name(id)}`. `resource.go`: `InitResource(db)` registering `GET /data/job-availability` (no id sub-route). No DB/docType needed — the data is `constants.For`. `main.go`: `.AddRouteInitializer(jobavailability.InitResource(db)(GetServer()))` adjacent to jobs (L184) and the skill sibling adjacent to skills (L183).

- [ ] **Step 4: Run — expect PASS + build.** Run: `cd services/atlas-data/atlas.com/data && go test ./jobavailability/ ./skillavailability/ && go build ./...`.

- [ ] **Step 5: Commit**

```bash
git add services/atlas-data
git commit -m "feat(atlas-data): tenant-version job/skill availability resources (task-187)"
```

---

## Task 14: atlas-ui — availability-gated preset selectors

**Files:**
- Create: `services/atlas-ui/src/services/api/availability.service.ts`
- Create: `services/atlas-ui/src/lib/hooks/api/useJobAvailability.ts`
- Create: `services/atlas-ui/src/lib/hooks/api/useSkillAvailability.ts`
- Modify: `services/atlas-ui/src/lib/hooks/usePresetJobOptions.ts`
- Modify: `services/atlas-ui/src/components/features/characters/presets/JobSkillsAddButton.tsx` (name source)
- Create/Modify: `src/lib/hooks/__tests__/usePresetJobOptions.test.tsx`, `src/services/api/__tests__/availability.service.test.ts`

**Interfaces:**
- Consumes: `/api/data/job-availability` (list of `{id, name}`), the JSON:API `api.getListDocument`, the four tenant headers (central), React Query.
- Produces: `useJobAvailability(tenant)` → `{ jobs: Array<{id:number,name:string}> }`; `usePresetJobOptions()` now returns `{id,name}[]` gated on availability + named by version.

- [ ] **Step 1: Write the failing hook test.** Mirror `usePresetJobOptions.test.tsx`: mock `useJobAvailability` to return `[{id:500,name:"GM"}]` for a v48 tenant → `usePresetJobOptions()` returns `[{id:500,name:"GM"}]` (not "Pirate"); mock v61 availability without Pirate → Pirate absent; pending → returns `[]`-or-full per the "unknown ≠ empty" rule (match existing behavior: full graph fallback becomes "all availability rows once loaded; while pending, keep the current graceful fallback").

- [ ] **Step 2: Run — expect FAIL.** Run (nvm22): `cd services/atlas-ui && npx vitest run src/lib/hooks/__tests__/usePresetJobOptions.test.tsx`.

- [ ] **Step 3: Implement service + hooks.** `availability.service.ts` `getJobAvailability()` → `api.getListDocument<JobAvailabilityResource>('/api/data/job-availability?page[size]=250')` mapping `.attributes.{id,name}`. `useJobAvailability.ts`: `useQuery({ queryKey: jobAvailabilityKeys.list(tenant?.id), queryFn: () => availabilityService.getJobAvailability(), enabled: !!tenant?.id, staleTime: 30*60_000, gcTime: 24*60*60_000 })` (mirror `useJobs.ts`).

- [ ] **Step 4: Reconcile `usePresetJobOptions`.** Replace the `useJobs` gate (L27-31) with `useJobAvailability`: options = the availability rows directly (`{id, name}` — names now come from the version, superseding `JOB_LIST`/`jobName`). Keep the "pending = graceful fallback, not empty" behavior. Update `JobSkillsAddButton.tsx`/`JobCombobox.tsx` to render `option.name` from the hook (they already consume `usePresetJobOptions()`).

- [ ] **Step 5: Run — expect PASS.** Run: `cd services/atlas-ui && npx vitest run src/lib/hooks src/services/api`.

- [ ] **Step 6: Full UI build (type-checks tests too).** Run: `cd services/atlas-ui && npm run build`. Expected: success.

- [ ] **Step 7: Commit**

```bash
git add services/atlas-ui/src
git commit -m "feat(atlas-ui): availability-gated, version-named preset selectors (task-187)"
```

---

## Task 15: v111 forward-compat review (recorded, not shipped)

**Files:**
- Create: `docs/tasks/task-187-version-aware-id-semantics/audit/v111-forward-compat.md`

**Interfaces:**
- Produces: a findings doc only — no code, no generated version for v111 (non-goal).

- [ ] **Step 1: Review the v111 reference IDB** via `func_query` (select the v111 instance). Record, per the audit template, which identities the Big Bang→v1.11 window adds/renames/removes relative to v95, and whether the identity namespace as generated would need new names to model v111. Explicitly mark it **NOT SHIPPED** and list what a future v111 bring-up would need (a snapshot, a semantics YAML, an availability manifest).

- [ ] **Step 2: Commit**

```bash
git add docs/tasks/task-187-version-aware-id-semantics/audit/v111-forward-compat.md
git commit -m "docs(task-187): v111 forward-compat review (not shipped)"
```

---

## Task 16: Full verification sweep + finalize

**Files:**
- Modify: `libs/atlas-constants/README.md` (confirm index rows), any lint fixups.

- [ ] **Step 1: Per-module Go gates.** For each of `libs/atlas-constants`, `services/atlas-channel/atlas.com/channel`, `services/atlas-character/atlas.com/character`, `services/atlas-data/atlas.com/data`: run `go test -race ./...`, `go vet ./...`, `go build ./...`. All clean.

- [ ] **Step 2: Repo-root guards.** From the worktree root: `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `sh tools/skill-job-id-guard.sh`, the generator drift check (`cd libs/atlas-constants/gen && go run . && git diff --exit-code libs/atlas-constants`), and `tools/lint.sh --check`. All clean (run `tools/lint.sh` fix mode first if needed).

- [ ] **Step 3: Docker bake the touched services.** `go.mod` was touched in atlas-channel/atlas-character/atlas-data (new import) — from the worktree root: `docker buildx bake atlas-channel atlas-character atlas-data`. (atlas-constants is a lib; baking its consumers covers the shared Dockerfile `COPY` correctness.) All succeed.

- [ ] **Step 4: atlas-ui gates.** `cd services/atlas-ui && npm run build && npx vitest run`. Clean.

- [ ] **Step 5: Code review before PR.** Invoke `superpowers:requesting-code-review` (dispatches plan-adherence + backend-guidelines + frontend-guidelines reviewers, pinned to Sonnet/Haiku per project policy). Address findings.

- [ ] **Step 6: Final commit**

```bash
git add -A
git commit -m "chore(task-187): verification sweep + docs finalize"
```

---

## Self-Review

**Spec coverage** (prd.md §10 / design §13):
- Two-axis model (semantics generated + availability curated) → Tasks 1,4,5.
- `constants.For` version-correct + `Resolve`/`Wire`/`Available` keyed on tenant → Tasks 4,5,6.
- Golden anchors (v48 5101004→GmHide & 500→Gm; v72 500→Pirate & 5101004→Corkscrew; a Big Bang remap; v61 Pirate stub present-in-semantics-absent-in-availability) → Task 7 Step 4.
- All identities + version-sensitive consumers migrated; scoped guard → Tasks 8–12.
- v0.48 correctness (Hide not Corkscrew) → Task 8 Step 1, Task 11 Step 1.
- Preset selectors gate on availability + name by version → Tasks 13,14.
- Committed multi-boundary audit w/ citations + Cygnus anchor → Tasks 1,15.
- v111 forward-compat reviewed, not shipped → Task 15.
- Clean go test/vet/lint/guards + atlas-ui build/vitest → Task 16.

**Placeholder scan:** every code step shows real code or an exact command; bulk-migration tasks (9,10,11) give the transform pattern plus the exact file:line site list from the exploration pass rather than "similar to above". The generator/emit function names are defined in Task 2/4/5 and reused consistently.

**Type consistency:** `skill.Identity` (uint32) / `job.Identity` (uint16); `Set.Resolve/Wire/Available/AvailableIdentities/Name`; `constants.For(region string, major, minor uint16) SkillJobSet{Skill skill.Set; Job job.Set}`; `Register(skill.Identity, Handler)`/`Lookup(skill.Identity)`; predicates `skill.IsKeyDownSkill(Identity)` / `job.IsAIdentity(Identity,...)`. These names are used identically across Tasks 2–14.

**Known open threads to resolve during execution (not blockers):** the exact Big Bang divergence rows (OQ-4) and the Cygnus anchor (OQ-3) are produced by Task 1 and must exist before Tasks 4/5 generate; the guard's divergent-id list (OQ-6) is sourced from `audit/divergences.csv` so it stays in sync.
