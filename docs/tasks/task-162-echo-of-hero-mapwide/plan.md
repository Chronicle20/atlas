# Echo of Hero — Map-Wide Buff Application — Implementation Plan

Task: task-162-echo-of-hero-mapwide
Source: `prd.md` v2, `design.md` v2
Updated: 2026-08-07 — rebased onto main; restructured for the registry-handler
design (D1 revised) and the 11-version scope (FR-5).

---

## Global Constraints

- **`common.go` and `recipients.go` MUST NOT be modified.** The design's D1/D2
  revision makes this task purely additive. A non-empty diff in either file
  means the implementation drifted from the design — stop and re-read design §3.5.
- **No wire-id literals in the diff.** No `1005`, `10001005`, `20001005`,
  `20011005`, `9101004`, or `5101004` may appear in new code. Registration is on
  `skill.Identity` constants; the hidden check is `buff.IsGmHidden`.
  `tools/skill-job-id-guard.sh` enforces this.
- **No `MajorVersion()` comparison, version table, or gates.yaml entry.** Version
  correctness is structural (design §7.3).
- **No new goroutines** (`tools/goroutine-guard.sh`), no raw keyed go-redis
  (`tools/redis-key-guard.sh`), no `*_testhelpers.go` files.
- **DOM-21:** define no new constants — all four identities exist in
  `libs/atlas-constants/skill/identities_gen.go`.
- Run `tools/lint.sh` (fix mode) before each commit.

## File Structure

| File | Change |
|---|---|
| `services/atlas-channel/atlas.com/channel/skill/handler/echoofhero/echoofhero.go` | **new** — `init()` registration ×4, `echoDeps`, `applyEchoOfHero` core, `Apply` production wiring |
| `services/atlas-channel/atlas.com/channel/skill/handler/echoofhero/echoofhero_test.go` | **new** — core-loop + registration + version-resolution tests |
| `services/atlas-channel/atlas.com/channel/skill/handler/registrations/registrations.go` | **+1 line** — blank import |

Reference template throughout: `skill/handler/healdispel/` (same map-wide shape,
same `isGmHidden` seam, same deps-struct pattern).

---

### Task 1: Echo of Hero core fan-out loop (TDD)

Builds the pure, offline-testable core: given a recipient list and seams, apply
the buff to everyone except the caster, dead characters, and hidden GMs.

- [ ] **Step 1: Write the failing tests**

  Create `echoofhero/echoofhero_test.go`. Follow `healdispel_test.go`'s shape:
  a `capture` struct recording applies, a `newDeps(...)` helper returning
  `echoDeps`, and `channelhandler.NewPartyRecipientBuilder()` for recipients.

  Fixtures:
  ```go
  const casterId, aliveA, aliveB, deadC, hiddenD = uint32(100), 101, 102, 103, 104

  func mkRecipient(id uint32, hp uint16) channelhandler.PartyRecipient {
      return channelhandler.NewPartyRecipientBuilder().
          SetId(id).SetHp(hp).SetMaxHp(1000).SetMp(100).SetMaxMp(100).Build()
  }
  ```

  Tests:

  | Test | Asserts |
  |---|---|
  | `TestAppliesToAllLivingNonCaster` | recipients `{caster, aliveA, aliveB}` → `applyBuff` called for `aliveA`, `aliveB` only |
  | `TestCasterSkippedNotDoubleBuffed` | caster present in selector output → **zero** `applyBuff` calls for `casterId` (FR-1.1) |
  | `TestDeadRecipientSkipped` | `deadC` has `Hp()==0` → no apply for it (FR-2.2) |
  | `TestHiddenGmSkipped` | `isGmHidden(hiddenD)` returns true → no apply for it (FR-2.3) |
  | `TestHiddenCheckErrorSkipsOnlyThatRecipient` | `isGmHidden(aliveA)` returns error → `aliveA` skipped, `aliveB` still applied (FR-2.5) |
  | `TestApplyErrorDoesNotAbortRemaining` | `applyBuff(aliveA)` errors → `aliveB` still applied (D4) |
  | `TestZeroDurationAppliesToNobody` | `e.Duration()==0` → zero applies (FR-1.2) |
  | `TestNoStatUpsAppliesToNobody` | `len(e.StatUps())==0` → zero applies (FR-1.2) |
  | `TestEmptyMapIsNoOp` | selector returns empty → zero applies, no error |

  Build the `effect.Model` the same way the sibling handler tests do — check
  `healdispel_test.go` / `mprecovery_test.go` for the in-package construction
  idiom and mirror it rather than inventing a builder.

- [ ] **Step 2: Run the tests to verify they fail**

  ```sh
  cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/echoofhero/...
  ```
  Expect compile failure (no `echoofhero.go` yet). That is the correct red state.

- [ ] **Step 3: Implement the core**

  Create `echoofhero/echoofhero.go` with the package doc, `echoDeps`, and
  `applyEchoOfHero` exactly as sketched in design §3.1:

  1. Gate: `if e.Duration() <= 0 || len(e.StatUps()) == 0 { return nil }`
  2. `rs := d.selectInMap(f)`; sort ascending by `Id()`
  3. Per recipient — skip caster, skip `Hp()==0`, skip/count on `isGmHidden`
     error, skip on hidden, else `d.applyBuff(r.Id())` counting apply failures
  4. Emit the `echo_of_hero_apply_summary` debug log with every counter

  Counters: `applied`, `skippedCaster`, `skippedDead`, `skippedHidden`,
  `fetchFailures`, `applyFailures`. Return `nil` — a cast is never failed by a
  per-recipient problem (FR-2.5).

- [ ] **Step 4: Run the tests to verify they pass**

  ```sh
  cd services/atlas-channel/atlas.com/channel && go test -race ./skill/handler/echoofhero/...
  ```

- [ ] **Step 5: Commit**

  ```sh
  git add services/atlas-channel/atlas.com/channel/skill/handler/echoofhero/
  git commit -m "feat(atlas-channel): Echo of Hero map-wide fan-out core [task-162]"
  ```

---

### Task 2: Registration, production wiring, and version correctness (TDD)

Wires the core to the registry and to real processors, and pins the version
behavior that FR-5 depends on.

- [ ] **Step 1: Write the failing tests**

  Append to `echoofhero_test.go`:

  | Test | Asserts |
  |---|---|
  | `TestRegistration` | `channelhandler.Lookup(id)` returns a non-nil handler for each of `skill2.BeginnerEchoOfHero`, `NoblesseEchoOfHero`, `LegendEchoOfHero`, `EvanEchoOfHero` (precedent: `resurrection_test.go:104`, `timeleap_test.go:186`) |
  | `TestVersionResolution_UnboundOnV12AndV48` | `constants.For("GMS", 12, 1).Skill.Resolve(skill2.Id(1005))` and the v48 equivalent both return `ok == false` — proving the handler is unreachable, and the task inert, on the two versions that ship no X005 (design §7.2) |
  | `TestVersionResolution_BeginnerOnlyOnV61` | v61 resolves wire `1005` → `BeginnerEchoOfHero` (`ok`), while wire `10001005` (Noblesse) returns `ok == false` |
  | `TestVersionResolution_EvanUnboundBeforeV84` | v79 resolves `20011005` → `ok == false`; v84 resolves it → `EvanEchoOfHero` |

  These four are the mechanical proof of design §7.3 — that registering all four
  identities is correct on all 11 versions without version code. They mirror
  `registry_test.go`'s `TestDispatch_v48HideNotCorkscrew`, the established
  version-correctness precedent in this package.

  > Note: the resolve assertions read the generated version sets directly, so
  > they will fail loudly if a future constants regeneration changes X005
  > availability — which is the intent. If one fails, update design §7.2's table
  > from the generated source; do not weaken the test.

- [ ] **Step 2: Run the tests to verify they fail**

  ```sh
  cd services/atlas-channel/atlas.com/channel && go test ./skill/handler/echoofhero/...
  ```
  `TestRegistration` fails (no `init()` yet); the resolution tests should
  **pass immediately** — they assert existing constants behavior. If any
  resolution test fails on first run, the availability table in design §7.2 is
  wrong: re-derive it from `libs/atlas-constants/skill/version_*_gen.go` and
  correct the docs before continuing.

- [ ] **Step 3: Implement registration and production wiring**

  Add to `echoofhero.go`:

  ```go
  func init() {
      channelhandler.Register(skill2.BeginnerEchoOfHero, Apply)
      channelhandler.Register(skill2.NoblesseEchoOfHero, Apply)
      channelhandler.Register(skill2.LegendEchoOfHero, Apply)
      channelhandler.Register(skill2.EvanEchoOfHero, Apply)
  }
  ```

  And `Apply`, matching `channelhandler.Handler`'s signature and mirroring
  `healdispel.Apply`'s deps construction (`healdispel.go:160-206`):

  ```go
  bp := buff.NewProcessor(l, ctx)
  d := echoDeps{
      selectInMap: func(f field.Model) []channelhandler.PartyRecipient {
          return channelhandler.SelectAllCharactersInMap(l, ctx, f)
      },
      isGmHidden: func(id uint32) (bool, error) {
          bs, err := bp.GetByCharacterId(id)
          if err != nil {
              return false, err
          }
          return buff.IsGmHidden(ctx, bs), nil
      },
      applyBuff: bp.Apply(f, characterId, int32(info.SkillId()), info.SkillLevel(), e.Duration(), e.StatUps()),
  }
  return applyEchoOfHero(l, f, characterId, info, e, d)
  ```

  `buff.Processor.Apply` returns `model.Operator[uint32]`
  (`libs/atlas-model/model/processor.go:50` — `func(M) error`), so `applyBuff`
  should be typed as that rather than a bare func literal. It is the same
  operator `common.go:176` builds for the caster. Use `e.StatUps()` (not a
  rewritten set): the Shadow Stars rewrite in `UseSkill` applies to that skill
  only.

- [ ] **Step 4: Add the blank import**

  In `skill/handler/registrations/registrations.go`, insert in alphabetical
  position among the existing imports:

  ```go
  _ "atlas-channel/skill/handler/echoofhero"   // Echo of Hero map-wide — task-162
  ```

- [ ] **Step 5: Run the tests to verify they pass**

  ```sh
  cd services/atlas-channel/atlas.com/channel && go test -race ./skill/handler/...
  ```
  The whole `skill/handler` tree, to catch registry-collision or regression.

- [ ] **Step 6: Commit**

  ```sh
  git add services/atlas-channel/atlas.com/channel/skill/handler/
  git commit -m "feat(atlas-channel): register Echo of Hero handler for all versions [task-162]"
  ```

---

### Task 3: Full verification gate

- [ ] **Step 1: Full atlas-channel suite with race detector**

  ```sh
  cd services/atlas-channel/atlas.com/channel && go test -race ./...
  ```

- [ ] **Step 2: Vet and build**

  ```sh
  cd services/atlas-channel/atlas.com/channel && go vet ./... && go build ./...
  ```

- [ ] **Step 3: Repo guards** (all from the worktree root, never with a global
      `GOWORK=off` prefix)

  ```sh
  tools/redis-key-guard.sh
  tools/goroutine-guard.sh
  tools/skill-job-id-guard.sh
  tools/lint.sh --check
  ```
  `skill-job-id-guard.sh` is the one that matters most here — it is the
  mechanical proof that no version-divergent wire literal entered the diff.
  `lint.sh --check` needs nvm 22 on PATH or it false-fails.

- [ ] **Step 4: Confirm the diff is scoped**

  ```sh
  git diff --name-only main...HEAD -- libs/ services/atlas-data services/atlas-buffs \
      services/atlas-configurations
  # must be empty (AC: libs/atlas-packet diff empty, no template changes)

  git diff --name-only main...HEAD -- \
      services/atlas-channel/atlas.com/channel/skill/handler/common.go \
      services/atlas-channel/atlas.com/channel/skill/handler/recipients.go
  # must be empty (design §3.5)
  ```

- [ ] **Step 5: Docker bake — only if `go.mod` changed**

  Not expected (no new module deps). If `services/atlas-channel/atlas.com/channel/go.mod`
  moved:
  ```sh
  docker buildx bake atlas-channel
  ```

- [ ] **Step 6: Acceptance-criteria sweep**

  Walk `prd.md` §10 line by line and record evidence (file:line or test name)
  for each. Anything not demonstrable is not done.

- [ ] **Step 7: Code review before PR** *(mandatory — CLAUDE.md)*

  Run `superpowers:requesting-code-review`; it dispatches
  `plan-adherence-reviewer` + `backend-guidelines-reviewer` (Go only — no
  frontend changes). Findings land in
  `docs/tasks/task-162-echo-of-hero-mapwide/audit.md`. Pin reviewer subagents to
  the cheaper model per project preference, and ensure they run **inside this
  worktree**.

---

## Manual verification (post-merge, optional)

Not required for the PR gate, but the highest-value smoke test given the
version scope: on a `gms_v83` tenant, park two characters and a dead character
in one map, cast Echo of Hero, and confirm the buff icon appears on both live
characters and not the dead one. A `gms_v61` tenant additionally confirms the
Beginner-only binding dispatches correctly (`OQ-3`).
