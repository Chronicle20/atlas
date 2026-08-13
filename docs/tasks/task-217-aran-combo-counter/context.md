# Aran Combo Counter — Implementation Context

Companion to `plan.md`. Everything here was read from the repo or from
`design.md`'s IDB/WZ findings — nothing is recalled from general MapleStory
knowledge.

---

## 1. The one-paragraph version

An Aran holding a polearm and owning Combo Ability builds a combo count as
they hit monsters. **The client drives the increment**: `CMob::OnHit` calls
`CUserLocal::RequestIncCombo`, which sends a **body-less** packet. The server
re-derives every gate, advances a count it owns, and echoes it back with
`SHOW_COMBO` (one 4-byte little-endian value). The count decays when the
player stops hitting things — and the client runs the *same* idle timer and
clears its own HUD, so server decay must agree with it rather than drive it.

---

## 2. The three findings that reshape the PRD

`design.md` closed all nine of the PRD's open questions. Three of the answers
change the implementation materially; the plan follows the design, not the
PRD sketch.

### 2.1 The count does NOT live on the `ARAN_COMBO` buff stat

The PRD's FR-3 inherited task-142's shape (count as a stat value on a buff,
advanced by `UPDATE_STAT_VALUE` `INCREMENT`). That fits Warrior combo orbs. It
does not fit here:

- `DrawCombo` reads the count **exclusively** from `SHOW_COMBO`'s body.
- The `ARAN_COMBO` secondary stat is decoded by
  `SecondaryStat::DecodeForLocal` as a **signed short** and its only client
  consumers are `Reset`, `DecodeForLocal`, `CheckByTime`, and
  `IsCalcDamageStat` — it is a damage-calculation input.
- Storing the count there would conflate two values, truncate above 32767,
  and force a Kafka round trip (channel → atlas-buffs → `STAT_UPDATED` →
  channel) per melee hit before `SHOW_COMBO` could be written.

**Consequence:** the count lives in a process-local `ComboMirror` in
atlas-channel, and **atlas-buffs needs no change at all** (a deviation from
PRD §8). atlas-buffs still owns the buff icon and the stat.

### 2.2 The server cannot clear the client's counter

`DrawCombo` opens with `if (m_nCombo <= 0) return;` **without releasing its
digit layers**. `SHOW_COMBO 0` therefore leaves stale digits on screen. There
is no clientbound packet that clears the HUD.

`CUserLocal::Update` calls `ClearCombo` after the idle window with no
`SHOW_COMBO`, and `ClearCombo` sends an ordinary `CANCEL_BUFF` for the Combo
Ability id.

**Consequence:** the decay tick sends nothing, the idle window is *configured
per version to match the client*, and the incoming `CANCEL_BUFF` is a
first-class reset input. PRD FR-4.4 ("tell the client") is deliberately
omitted as impossible.

### 2.3 Every opcode in the PRD's table was already correct

All twelve (2 ops × 6 versions) were IDA-verified during design; **zero
registry corrections are needed**, and the serverbound body is empty on every
version. FR-6.1's "hard prerequisite" is already discharged. Execution's
packet work is evidence-pinning and matrix promotion (plan Task 13), not
derivation.

---

## 3. Key files, with what they are for

### Reference implementations to copy from

| File | Why it matters |
|---|---|
| `services/atlas-channel/atlas.com/channel/character/buff/beacon.go` | The exact shape `ComboMirror` copies: tenant-keyed map, `sync.RWMutex`, `sync.Once` singleton, and the documented "process-local, accepted degradation on restart" contract. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo.go` | The deps-seam pattern (`comboOrbDeps` / `comboOrbProductionDeps` / `comboOrbTryUpdate`) the Aran handler mirrors, and the "all failures logged and swallowed" contract at line 158-166. |
| `services/atlas-channel/atlas.com/channel/socket/handler/character_attack_combo_test.go` | The real test helpers: `comboTestSkill` (line 113, via `skill.Extract`), `comboTestCharacter` (122, via `character.NewModelBuilder()`), `comboTestEffect` (136, via `effect.Extract`). |
| `libs/atlas-packet/character/serverbound/chalkboard_close.go` + `_test.go` | The empty-body serverbound codec and its `pt.Variants` / `pt.RoundTrip` / `pt.Encode` test shape with `packet-audit:verify` markers. |
| `services/atlas-buffs/atlas.com/buffs/tasks/poison.go` + `character/processor.go:281` | The tick-task shape and the `tenant.WithContext` fan-out the decay tick reuses (`ProcessPoisonTicks`). |
| `services/atlas-channel/atlas.com/channel/session/processor.go:406-449` | `Destroy` and `clearBattleshipOnDestroy` — the extracted-hook pattern the combo session-teardown clear copies verbatim. |
| `services/atlas-data/atlas.com/data/skill/reader_test.go:3264` | The XML-driven reader test (`Read` → `CollectToMap` → `findStatup`) the `ARAN_COMBO` statup test copies. |

### Files being changed

See plan.md's **File Structure** table. The short version: two new codecs in
`libs/atlas-packet`, one new identity in `libs/atlas-constants`, a new
`character/combo` package plus one handler and three one-line hooks in
atlas-channel, one branch in atlas-data's skill reader, and two JSON entries
in each of six seed templates.

---

## 4. Facts verified during planning (do not re-derive)

| Fact | Evidence |
|---|---|
| `20000017` **is** in the WZ snapshots for gms 79/83/84/87/92/95 and jms 185 | `grep -rn "20000017" libs/atlas-constants/gen/wzsnapshot/` — hits in seven files. This settles PRD open question 8 in favour of adding the constant. |
| Identity tables are **generated** | `libs/atlas-constants/gen/identities.yaml` + `cd libs/atlas-constants/gen && go run .`; `go run . -check` is the drift gate. Never hand-edit `*_gen.go`. |
| `reader.go:470`'s Combo Ability branch is at **function level**, not inside the `e.OverTime()` block (which starts at line 234) | Indentation: the `} else if` is one tab. A bare `level/x` XML node reaches it, so the statup test needs no `effect` child. |
| `item.WeaponTypePolearm` is `cat - 30 == 14`, i.e. `cat == 44` | `libs/atlas-constants/item/constants.go:135-162`. The client's `GetWeaponType(...) == 44` and Combo Ability's WZ `weapon = 44` are the same number on the same basis. |
| `equippedWeapon` is the canonical equipped-weapon reader | `socket/handler/character_attack_projectile.go:191-197` — `c.Equipment().Get("weapon")`, then `s.Equipable`. |
| The attack pipeline already fetches with **both** decorators | `character_attack_common.go:745` — `cp.GetById(cp.InventoryDecorator, cp.SkillModelDecorator)`. The eligibility refresh rides this fetch; `comboOrbTryUpdate` is called at line 981. |
| Handler options reach the handler as `readerOptions map[string]interface{}` | Every `…HandleFunc` signature. This is how `idleResetMs` gets from the template into the code, and why the mirror stores the window per entry — the decay tick has no options in hand. |
| atlas-channel already registers tick tasks | `main.go:326` — `tasks.Register(l, rt.Context())(channel3.NewHeartbeat(...))`. `tasks.Register` (`tasks/task.go`) spawns via `routine.Go`, satisfying the goroutine guard. |
| Handler names are packet-package constants, bound by string in templates | `main.go:913` — `handlerMap[charsb.CharacterBuffCancelHandle] = handler.CharacterBuffCancelHandleFunc`; the template's `"handler"` value must equal the const's **value**. |
| Writer names must also be listed in main.go's writer-name slice | ~line 793, e.g. `charcb.CharacterHintWriter`. |
| Matrix rows exist and are all `❌` today | `docs/packets/audits/STATUS.md:352` (`SHOW_COMBO`), `:726` (`ARAN_COMBO_COUNTER`). Registry provenance is `csv-import` (`docs/packets/registry/gms_v83.yaml:2882`) — design verified the values are nonetheless correct. |

---

## 5. Decisions locked in the plan

| Decision | Why |
|---|---|
| Count in a process-local mirror, not Redis, not a buff stat | §2.1 above. Combo lives 3–5 s, dies with the session, and a session is pinned to one channel process. Also keeps `tools/redis-key-guard.sh` out of scope. |
| Cap = **99999** | The client's 5-slot digit-layer array. Nothing in WZ governs a cap; the tier cues at 30/100/200 are client-hardcoded and need no server involvement. |
| Idle window as the handler option `idleResetMs`, stored per mirror entry | v95 is 5000 ms, everything else 3000 ms. Config-resolved per the project's "client wire values are config-resolved" rule (DOM-25) instead of a compiled major-version branch — and the `MajorVersion() > 83` off-by-one is a known project bug. |
| Decay tick lives in **atlas-channel**, 1 Hz, walks the mirror only | Follows the count. An empty mirror is an empty walk (FR-4.5). |
| No job-range gate — the skill check *is* the job gate | Matches the client exactly (`CMob::OnHit` applies only `GetSkillLevel > 0`, weapon type 44, damage > 0). A range check would reject legitimate states. |
| The cancel predicate takes no job id | `character_buff_cancel.go` runs for every buff cancel and does not fetch the character; the two Combo Ability ids belong to disjoint job branches, so the id alone identifies the branch. |
| `reader.go:470`: `100` → `int32(e.X())`, extended to cover `20000017` | The `100` has no provenance in WZ or the client; every sibling Aran statup passes `e.X()`. **Residual unknown, stated plainly:** the exact arithmetic the client's `CalcDamage` performs with the value was not traced. Nothing server-side reads `ARAN_COMBO`, so the change is inert server-side and strictly better-grounded than `100`. |
| Combo *consumption* stays out of scope | PRD non-goal. Not observable as drift: `DoActiveSkill` calls `ClearCombo` when Combo Smash / Fenrir / Tempest fire, and that cancel resets the server count for free. |

---

## 6. Traps this project has hit before, live in this task

- **A seed-template handler with a missing/empty validator is silently
  dropped.** Both new entries carry a validator / `fname`.
- **New opcodes absent from a *live* tenant's config are silently dropped.**
  Live tenants must be reconciled to the updated templates after merge, or the
  symptom is "no counter at all, and no server-side log".
- **Template entries must sit at their sorted `opCode` position** —
  `tools/template-opcode-order-guard.sh` enforces it; never append beside a
  semantically-related entry.
- **A registry `fname` edit stales the packet matrix.** No registry edits are
  needed here, which sidesteps it.
- **`tools/lint.sh --check` false-fails without nvm on PATH**, and contends on
  a golangci-lint lock across worktrees. Source nvm before concluding a
  failure is real.
- **`go build` against `go.work` will not catch a missing `COPY libs/...`** in
  the shared Dockerfile — only `docker buildx bake` will, and only for
  services whose `go.mod` actually moved.

---

## 7. Verification gates for this task

Per CLAUDE.md §Build & Verification, scoped to what this change touches:

**Required:** `go test -race ./...`, `go vet ./...`, `go build ./...` in
`libs/atlas-packet`, `libs/atlas-constants`, `services/atlas-channel/...`,
`services/atlas-data/...`; `tools/redis-key-guard.sh`;
`tools/goroutine-guard.sh`; `tools/buff-duration-guard.sh`;
`tools/skill-job-id-guard.sh`; `tools/template-opcode-order-guard.sh`;
`tools/template-duplicate-binding-guard.sh`;
`tools/template-movement-types-guard.sh`; `tools/lint.sh --check`;
`docker buildx bake atlas-<svc>` for each service whose `go.mod` moved.

**Not required:** `tools/service-registration-guard.sh` — no service was added
and none of `services.json`, `deploy/k8s`, `docker-bake.hcl`, `go.work`, or
`tools/db-bootstrap.sh` changed. Confirm with a `git diff --name-only` check
before skipping it.

**Also required before the PR:** `superpowers:requesting-code-review`
(dispatches `plan-adherence-reviewer` + `backend-guidelines-reviewer`; no
atlas-ui changes, so no frontend reviewer). Pin review subagents to
Sonnet/Haiku per the project's model-cost preference.

---

## 8. Task dependency order

```
Task 1 (serverbound codec) ─┐
Task 2 (clientbound codec) ─┤
Task 3 (LegendComboAbilityId) ─┬─> Task 4 (atlas-data statup)
                               │
Task 5 (ComboMirror) ──> Task 6 (gates) ──> Task 7 (handler + wiring)
                                              ├─> Task 8 (attack hook)
                                              ├─> Task 9 (reset paths)
                                              └─> Task 10 (decay tick)
Tasks 1,2 ──> Task 11 (six templates)
Task 12 (v92 IDB rename) — independent, no repo change
Tasks 1,2,11 ──> Task 13 (twelve matrix cells)
everything ──> Task 14 (docs + full sweep + code review)
```

Tasks 1, 2, 3, 5, and 12 are independent of each other and can be done in any
order. Task 7 needs 1, 2, 5, and 6. Task 4 needs 3.
