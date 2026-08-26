# Touch-Activated Reactors — Implementation Context

Companion to [plan.md](plan.md). Everything here was read out of the repository,
`Reactor.wz`, or the client IDBs while writing the plan; it exists so an
implementer does not re-derive it.

`<WZ_ROOT>` below stands for the checked-out Cosmic WZ tree on the developer's
machine (the directory containing `Reactor.wz/`); it is not a repository path.

---

## Services and modules touched

| Module root | Why |
|---|---|
| `libs/atlas-packet` | new `TouchingRequest` codec + fixtures |
| `tools/packet-audit` | `candidatesFromFName` case; matrix regeneration |
| `services/atlas-configurations` | seed-template routing (JSON only, no Go) |
| `services/atlas-data/atlas.com/data` | `activateByTouch` + `touchAreaInfo` on the reactor resource |
| `services/atlas-reactors/atlas.com/reactors` | the bulk: data plumbing, character client, latch, `Hit` split, `Touch` |
| `services/atlas-channel/atlas.com/channel` | handler + `Touch` producer |
| `services/atlas-reactor-actions/atlas.com/reactor` | `touchRules` + `TOUCH` consumer |

`atlas-ui` is untouched. No database migration anywhere.

---

## Facts an implementer would otherwise have to rediscover

### Packet lane

- **`TOUCHING_REACTOR` is tier-0** (`docs/packets/audits/status.json`:
  `"tier1": false`). Per `VERIFYING_A_PACKET.md` §7 that means **no evidence
  record** — pinning one on a tier-0 cell creates a standing freshness
  liability. What it does need (§9) is the codec, the `packet-audit:verify`
  marker, the generated audit report, and a routed opcode in that version's seed
  template.
- The sibling `DAMAGE_REACTOR` cells are the working reference: verified on
  v83/v84/v87/v95/jms with `docs/packets/audits/<v>/ReactorHitRequest.json`
  reports and markers in `hit_test.go`, and **no** evidence records under
  `docs/packets/evidence/`.
- **`qualifiedWriterName`** = TitleCase(pkg) + struct name, so struct
  `TouchingRequest` in `libs/atlas-packet/reactor/serverbound/` produces the
  packet id `reactor/serverbound/ReactorTouchingRequest` and the report filename
  `ReactorTouchingRequest.json`. The struct name carries no `Reactor` prefix.
- All eight target opcodes were confirmed **free** in each template's
  `socket.handlers` array at plan time (checked by parsing every template and
  comparing opcode sets). `0xC4`/`0xCE`/`0xDB` do collide with entries in the
  `socket.writers` array — that is a different array and not a conflict.
- Opcodes 196 (gms_v72) and 198 (gms_v79) have **no** serverbound registry entry
  today, so Task 3's inserts collide with nothing.
- **The v84 opcode is the sharpest risk in the packet lane.** `status.json`
  carries 206 for v84 by CSV import, identical to v83 — but `DAMAGE_REACTOR` is
  205 on v83 and **211** on v84, a +6 shift, and design §1.4 found no
  `68 CE 00 00 00` anywhere near v84's `CReactorPool` cluster. 212 (`0x0D4`) is
  the hypothesis to test first, not a value to write down. A wrong opcode here
  routes a live handler onto an unrelated packet.
- All ten IDBs were open at plan time. Session ids are in plan.md Task 1 but are
  **not stable across restarts** — always re-run `mcp__ida-pro__idb_list`.

### The `[TL, BR]` problem

The PRD's FR-14 bounds check cannot be implemented as specified, and this is the
single most important thing to carry forward:

- `atlas-data` populates `TL`/`BR` **only** inside the `if t == 100` branch at
  `services/atlas-data/atlas.com/data/reactor/reader.go:111`.
- **None of the ten `activateByTouch` templates contains a type-100 event.**
  Verified against `<WZ_ROOT>/Reactor.wz/`: `2406000`'s three states
  carry events of type 6 and 0 and nothing else; `6109013`'s two states carry
  types 6 and 7.
- So `TL`/`BR` are `(0,0)`/`(0,0)` for every reactor this feature targets, and
  an `[TL, BR]` check would reject 100% of legitimate touches.

The replacement is the per-state canvas rectangle (design §5.1). The WZ shape is
`<imgdir name="<state>"><canvas name="0" width="W" height="H"><vector
name="origin" x="OX" y="OY"/></canvas>…</imgdir>` — a direct `CanvasNode` child
of the state directory, sibling to the `event` and `hit` subdirectories. In Go
that is `rid.CanvasNodes` filtered to `Name == "0"`, with `Width`/`Height` as
**strings** needing `strconv.Atoi`, and `GetPoint("origin", 0, 0)` returning
`(int32, int32)`.

`2406000` state 2 is a `1×1` canvas at origin `(0,0)` with no `event` node —
the data itself saying "this state is spent and untouchable". That is the
concrete justification for OQ-6's no-op-on-empty-state answer.

**The origin sign convention is inferred, not measured.** Task 6 Step 1 gates on
confirming it against `CReactorPool::LoadReactorLayer` (gms_v83 `@0x7348a0`). If
it is wrong the failure is loud and safe — every touch is rejected — but the
test table in Task 6 Step 2 must be corrected *before* the code is written, not
after.

### `atlas-reactors`

- **`reactor/data.Model` has no builder**, contrary to FR-11. It is constructed
  solely by `Extract(RestModel)` in `reactor/data/rest.go`. Plumbing goes
  through `RestModel` + `Extract`.
- Every `reactor/data` sub-model carries a `model_json.go` with hand-written
  `MarshalJSON`/`UnmarshalJSON`, because the whole `data.Model` is serialised
  into Redis by `atlas.TenantRegistry`. A new field that skips `model_json.go`
  survives a unit test and silently vanishes on the first registry round-trip.
  The new `area` package needs all three files.
- **The latch belongs in Redis, not in a process map.** `atlas-reactors` runs
  multi-pod; two pods would otherwise both accept the same touch.
  `TenantKeyedHash.SetNX` is the atomic primitive, already used by
  `TryClaimSpot`. `TenantKeyedSet` has no atomic add-if-absent, so the design's
  "set per reactor" is realised as a hash whose fields are character ids.
- **Latch on `(reactor, character)`, not `(reactor, character, state)`.**
  `6109013` cycles state 0 →(type 6)→ 1 →(type 7)→ 0. A state-keyed latch would
  let a motionless character re-trigger every time the cycle came back around.
  The character-keyed latch mirrors the client's `m_reactorOnLocalUser` map
  exactly: one activation per area entry.
- `Registry.Remove` is the single choke point for reactor teardown — `Destroy`,
  `DestroyInField` and the shutdown sweep all route through it — so one
  `ClearAllTouches` call there covers every path.
- Existing tests tolerate Kafka producer errors (`_ = NewProcessor(...).Hit(...)`)
  and assert on registry state. `producertest.InstallNoop()` runs in
  `TestMain`. `newTestData` at `processor_test.go:388` is the fixture helper;
  Task 12 extends its signature and must update its five existing call sites.
- `atlas-reactors` must not import `atlas-env` from the `reactor` package
  (env-domain-guard); nothing in this task needs to.

### The FR-16 inversion

This is the defect the task exists to prevent, and it deserves restating because
it is counter-intuitive: reusing `Hit`'s
`len(event.ActiveSkills()) == 0 || containsSkill(...)` predicate on the touch
path would, for a skill-gated event with a non-empty `activeSkills` list, leave
`nextState == -1`, fall through to `TriggerAndDestroy`, and **destroy** the
reactor on a walk-over instead of advancing it.

A caveat worth knowing: in the *Cosmic* `Reactor.wz` the ten touch templates'
events carry **no** `activeSkillID` node, so `ActiveSkills()` is empty and the
hit predicate would happen to match. The inversion is therefore latent against
this particular WZ set rather than immediately visible — which is exactly why
`TestTouch_SkillGatedStateAdvances` constructs a synthetic non-empty
`ActiveSkills` list rather than loading a real template. Other WZ sets on this
machine disagree on the template count (9 / 10 / 19), so the guard must not
depend on the mounted data.

### `atlas-reactor-actions`

- Unknown command types are **warned and ignored**
  (`script/consumer.go:70` `default: l.Warnf(...)`), so Tasks 11–13 can land
  before Task 15 without an error flood — only a warn flood. Sequencing is a
  courtesy, not a correctness constraint.
- Rules are stored as JSONB in `Entity.Data`, not as columns. `touchRules` is an
  additive JSON key: **no migration.** Tagging it `omitempty` keeps every
  existing stored script's serialised bytes unchanged.
- The `hitRules` fallback in `ProcessTouch` is a deliberate design call
  (design §7): none of the ten templates has a script yet and authoring them is
  a PRD non-goal, so without the fallback this task ships a mechanism that
  advances state and then hits `no_match`. Distinguishability survives via the
  `TOUCH` command type, the `"touch"` event label, and the fact that a script
  declaring `touchRules` never reaches the hit path.

### Cross-service

- The character-position read has a precedent to copy verbatim:
  `services/atlas-maps/atlas.com/maps/character/` (added for the mist tick).
  `atlas-character`'s REST `x`/`y` is live, not persisted-stale — `character/rest.go:82`
  projects from `GetTemporalRegistry()`, which the movement consumer updates on
  every movement command.
- `atlas-channel`'s `session.Model` carries **no position** — only
  `characterId`, `field`, and connection state. Checking bounds at the channel
  edge would need the same REST call *and* would put anti-cheat on the edge
  rather than at the authority. Hence the check lives in `atlas-reactors`.
- Both services declare their own copy of the Kafka envelope (as they already do
  for `HitCommandBody`). Task 11 and Task 13 must keep the json tags identical.
- `template_gms_92_1.json` routes **no** `ReactorHitHandle` at all. That is a
  pre-existing gap. This task adds `TouchReactorHandle` there (its opcode is a
  matrix column) and deliberately does not fix the hit gap — doing so would be a
  behaviour change to a version this task is not otherwise touching.

---

## Deliberately oversized tasks

`tools/plan-lint.sh` F4 flags tasks touching more than ~6 files or more than one
service. Three tasks trip it on purpose:

- **Task 3** (registry + support files, ~7 files) — the same one-line correction
  repeated across parallel per-version files. There is no shared logic to get
  wrong and splitting it would multiply dispatch overhead for zero review
  benefit.
- **Task 4** (seed templates, 8 files) — one identical JSON object inserted at
  the sorted position in eight files. Same reasoning.
- **Task 5** (export splice + reports + matrix, 8 versions) — the artifacts must
  land together or `matrix --check` fails; §8 of `VERIFYING_A_PACKET.md`
  requires the test, reports, `STATUS.md` and `status.json` in one commit.
  Splitting it would mean committing a knowingly-red gate.

**Task 8** also trips F4's multi-service check, but only because its `Files`
block cites `services/atlas-maps/atlas.com/maps/character/` as the read-only
package to copy. Nothing in `atlas-maps` is edited.

Task 12 is the largest genuinely-logic-bearing task (one file plus one new test
file), and it is the one whose review matters most. It was kept whole because
the rejection ladder and the state selection are one decision, not two.

---

## Verification commands

Per-module, for implementers:

```sh
cd libs/atlas-packet                            && go build ./... && go test ./reactor/...
cd tools/packet-audit                           && go build ./...
cd services/atlas-data/atlas.com/data           && go build ./... && go test ./reactor/... ./point/...
cd services/atlas-reactors/atlas.com/reactors   && go build ./... && go test ./...
cd services/atlas-channel/atlas.com/channel     && go build ./... && go test ./...
cd services/atlas-reactor-actions/atlas.com/reactor && go build ./... && go test ./...
```

Repo-wide gates (controller / `atlas-verifier`, never an implementer):

```sh
tools/template-opcode-order-guard.sh
tools/template-duplicate-binding-guard.sh
tools/template-symbol-check.sh services/atlas-configurations/seed-data/templates/template_gms_83_1.json
go run ./tools/packet-audit matrix --check
tools/verify.sh
```
