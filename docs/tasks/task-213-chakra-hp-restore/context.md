# task-213 — Implementation Context

Companion to [`plan.md`](plan.md). Read this first if you are picking the task up cold.
Authoritative inputs: [`prd.md`](prd.md), [`design.md`](design.md).

---

## 1. The one-paragraph model

Chakra reaches the server as **two packets, ~1500 ms apart**. Keypress sends
`CUserLocal::DoActiveSkill_Prepare` (`CharacterSkillPrepareHandle`); animation end sends
an ordinary `USE_SKILL` (`CharacterUseSkillHandle`). Everything follows from that shape:
the activation gate and the recovery window belong on the **first** packet, the heal
belongs on the **second**, and MP/cooldown are charged exactly once by the generic
`UseSkill` block that only runs on the second — so an interrupted cast costs nothing and
there is no refund to write. The client never computes the heal itself; it renders
whatever HP the server reports.

---

## 2. Files that matter

### New

| Path | What |
|---|---|
| `services/atlas-channel/atlas.com/channel/character/chakra/formula.go` | `CanActivate`, `Base`, `Recovery`, `Applied`, `EffectiveMaxHpOrBase` — pure, stdlib-only |
| `services/atlas-channel/atlas.com/channel/character/chakra/registry.go` | tenant-keyed in-process recovery-window singleton, lazy TTL + `routine.Go` sweeper |
| `services/atlas-channel/atlas.com/channel/skill/handler/chakra/chakra.go` | the `Handler` on `skill2.ChiefBanditChakra` |

### Touched

| Path | Why |
|---|---|
| `socket/handler/character_damage_mitigation.go` | `chakraPct` term, applied **first** |
| `socket/handler/character_damage.go` | read the window, feed `chakraPct`, interrupt after the hit lands |
| `socket/handler/character_skill_prepare.go` | activation gate + open the window |
| `socket/handler/character_skill_use.go` | pre-cost reject when no window is open |
| `socket/handler/character_move.go`, `socket/handler/map_change.go`, `socket/init.go` | interruption + sweeper startup |
| `skill/handler/registrations/registrations.go` | blank import |
| `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` | bind the prepare handler at `0x68` |
| `services/atlas-data/atlas.com/data/skill/common_test.go` | pin the v95 `common` expansion |

### Read-only references you will want open

- `services/atlas-channel/atlas.com/channel/character/statreset/registry.go` — the exact
  registry shape to copy (singleton, `tenant.Model` key, `ClearCharacter` on destroy).
- `services/atlas-channel/atlas.com/channel/skill/handler/heal/heal.go` + `formula.go` —
  the per-skill handler package layout, and `effectiveMaxHpOrBase`'s defensive narrowing.
- `services/atlas-channel/atlas.com/channel/skill/handler/common.go:101-211` — `UseSkill`:
  cost/cooldown (`:137-165`) run **before** handler dispatch (`:200-206`). This is why the
  gate cannot live inside the handler.
- `services/atlas-channel/atlas.com/channel/socket/handler/character_skill_use.go:116-139`
  — the Hero Enrage precedent: reject before `handler.UseSkill`, call `enableActions`.
- `services/atlas-channel/atlas.com/channel/socket/handler/character_damage.go:190-320` —
  `processDamageTaken`; the `damageMitigationDeps` seam is how this file is tested.

---

## 3. Decisions already made — do not re-litigate

| Decision | Where it came from |
|---|---|
| Heal = `2.9 × effective LUK × y / 100`, deterministic | design §3.4. The client provably does **not** compute it on any of ten IDBs. Community-sourced, labelled UNVERIFIED in code. Owner-approved over blocking. |
| No RNG in the heal | PRD FR-7.6, design §6.4. `Recovery` takes no RNG parameter deliberately. |
| Chakra is **not** a keydown skill | design §3.5, reproducing task-161. `libs/atlas-constants` is untouched. FR-10.3, not FR-10.2. |
| No CTS / packet-model entry | design §3.6. The client's flag is `m_nPrepareSkillID`, never encoded. |
| The damage factor is applied **first**, before Achilles/Combo Barrier/Magic Guard/etc. | design §3.3 — the client writes back to the same stack slot every later term reads. |
| No version gate anywhere | design §4.2, §5.4. `x` is 200→112 on v12/v48 (amplify) and 99→70 on v61+ (reduce); the WZ data carries the direction. |
| TTL is 5000 ms | design §4.3. A safety bound, not a timing model — the client closes the window by sending `USE_SKILL`. |
| Window opens on prepare, not on `USE_SKILL` | design §6.1. Starting at `USE_SKILL` opens the window exactly when the client closes it. |
| Interruption on movement is a crafted-client defence | design §3.7. `CUserLocal::IsImmovable` is true for the whole window, so an authentic client never triggers it. |

---

## 4. Four plan-phase corrections to `design.md`

These were forced by reading the code and the packet registry. Each is restated at the
task that implements it.

1. **Registry lives in `character/chakra`, not `skill/handler/chakra`.** `skill/handler/*`
   packages import `atlas-channel/socket/handler` (`heal/heal.go:16`), so `socket/handler`
   cannot import back into `skill/handler/chakra` — and the damage, move, map-change,
   prepare and use paths all live in `socket/handler`. Design §5.1's layout is an import
   cycle. `character/statreset` is the precedent for the corrected placement.

2. **No HP re-check at `USE_SKILL`.** Design §5.2 lists one; it contradicts PRD FR-1.3 and
   design §3.2's own finding that the client has no post-gate re-check. The
   window-presence check closes the same crafted-client hole more tightly. Gate on window
   presence only.

3. **`template_gms_12_1.json` is not edited.** Design §6.3 says v12 and v92 both bind the
   `CharacterSkillPrepareForeign` writer but not the handler. Verified: that is true of
   **v92 only**. v12's template is a 24-handler stub with no `CharacterUseSkillHandle` and
   no such writer, and GMS 12 has no `docs/packets/registry/` entry and no IDA export — so
   there is no authority for a `SKILL_EFFECT` opcode. v12 is recorded out of reach per
   FR-9.4. v92's opcode is `0x68` (`docs/packets/registry/gms_v92.yaml:2704`, `SKILL_EFFECT`
   serverbound `104`).

4. **The handler does not broadcast the cast effect.** `character_skill_use.go` already
   announces self + foreign unconditionally after `handler.UseSkill` returns. FR-8.1 is
   satisfied there; re-announcing (as `heal.go` does) would send it twice.

---

## 5. Signatures you need to match

```go
// character/chakra
func CanActivate(hp uint16, maxHp uint16) bool
func Base(luck uint32) int32
func Recovery(base int32, y int16) int32
func Applied(heal int32, hp uint16, maxHp uint16) int16
func EffectiveMaxHpOrBase(effective uint32, base uint16) uint16

type Entry struct { SkillLevel byte; X int16; Y int16; StartedAt time.Time }
const TTL = 5000 * time.Millisecond

func GetRegistry() *Registry
func (r *Registry) Start(t tenant.Model, characterId uint32, level byte, x, y int16, now time.Time)
func (r *Registry) Get(t tenant.Model, characterId uint32, now time.Time) (Entry, bool)
func (r *Registry) Clear(t tenant.Model, characterId uint32) bool
func (r *Registry) Sweep(now time.Time) int
func (r *Registry) StartSweeper(l logrus.FieldLogger, ctx context.Context)
```

Existing, consumed as-is:

```go
character.Model.Hp() uint16 / MaxHp() uint16 / Luck() uint16       // character/model.go:128-136
character.Processor.ChangeHP(f field.Model, characterId uint32, amount int16) error  // character/processor.go:44
effective_stats.RestModel{ Luck uint32; MaxHp uint32 }             // effective_stats/rest.go
effective_stats.Processor.GetByCharacterId(worldId, channelId, characterId) (RestModel, error)
dataskill.Processor.GetEffect(skillId uint32, level byte) (effect.Model, error)
effect.Model.X() int16 / .Y() int16                                // data/skill/effect/model.go:177,190
skill.IsIdentity(id, skill.ChiefBanditChakra) bool
constants.For(t.Region(), t.MajorVersion(), t.MinorVersion()).Skill.Resolve(skill.Id(...)) (Identity, bool)
channelhandler.Register(id skill2.Identity, h Handler)             // skill/handler/registry.go:36
routine.Go(l, ctx, func(ctx context.Context) { ... })              // socket/init.go:44
```

---

## 6. WZ values, per version (from `design.md` §4.2)

`Skill.wz/421.img/skill/4211001` carries per level exactly `hs`, `mpCon`, `time`, `x`, `y`.
`y` = recovery rate %, `x` = damage-taken %. `time` is vestigial (always 1) and must not be
read as the recovery window. On v83 the `String.wz` `h1..h30` tooltip strings are **stale**
(they still carry the v48 numbers) — never use tooltips as a data source.

| Versions | maxLevel | `x` (damage taken %) | `y` (recovery %) | Source |
|---|---|---|---|---|
| GMS 12, 48 | 30 | 200 → 112 (**amplifies**) | 9 → 200 | explicit `level` nodes |
| GMS 61, 72, 79, 83, 84, 87, 92; JMS 185 | 30 | 99 → 70 (**reduces**) | 68 → 300 | explicit `level` nodes, byte-identical across all eight |
| GMS 95 | 10 | 96 → 60 (reduces) | 120 → 300 | `common` formula node |

The PRD FR-6.2 reference table is the **v12/v48 table only** — correct for two columns,
wrong for the other nine. The WZ wins; nothing is hardcoded.

---

## 7. Traps

- **Import cycle.** See §4.1. If you put the registry under `skill/handler/`, the build
  breaks the moment `socket/handler` imports it.
- **`x = 0` reads as "no window", never as "zero damage".** A v95 `common` expansion
  regression would otherwise make the caster invincible during the window. Task 10 pins it.
- **The `<= 1 → 1` floor is the client's, and it is `<=`, not `<`.** It applies to the
  multiplied value, not the input.
- **Don't add a `MajorAtLeast` gate.** The v61 table inversion is data, not code. A future
  reader "correcting" the reductions back to amplification fails Task 3's tests loudly —
  that is deliberate.
- **`SetDamaged` mentions 4211001 next to the keydown machinery** (v83 `0x95924F`, v95
  `0x935C67`) in the form `if (id == 4211001 || is_keydown_skill(id)) { if (is_keydown_skill(id)) {…} }`.
  The inner test filters Chakra straight back out; the compare is inert. It is **not**
  evidence of keydown membership (design §3.5).
- **A template handler entry with a missing or empty `validator` is silently dropped** at
  load. Use `LoggedInValidator`, matching the nine templates that already bind it.
- **`tools/lint.sh --check` false-fails without nvm loaded** (its atlas-ui half). Load nvm
  before believing a failure — and before believing a pass.
- **`ForEachInMap` runs in parallel** in atlas-channel. The registry must be race-clean;
  Task 2's concurrency test is the guard.
- **`skillLevelOf` already exists** in `socket/handler` (`character_skill_use.go`). Reuse it.

---

## 8. Verification (CLAUDE.md "Build & Verification")

Changed modules: `services/atlas-channel/atlas.com/channel`, `services/atlas-data/atlas.com/data`.
No `go.mod` touched → `docker buildx bake` is **not** required (confirm with
`git diff main --stat -- '**/go.mod'` before relying on that).

```
go test -race ./...   # both modules
go vet ./...          # both modules
go build ./...        # both modules
tools/redis-key-guard.sh          # not engaged — no Redis — must still be clean
tools/goroutine-guard.sh          # the sweeper must use routine.Go
tools/skill-job-id-guard.sh
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-movement-types-guard.sh
tools/lint.sh --check             # tools/lint.sh (no flags) to fix
```

Then `superpowers:requesting-code-review` before the PR — reviewers pinned to Sonnet/Haiku,
never an expensive model. Confirm the worktree is still
`.worktrees/task-213-chakra-hp-restore` on branch `task-213-chakra-hp-restore` and clean
after the subagent run.
