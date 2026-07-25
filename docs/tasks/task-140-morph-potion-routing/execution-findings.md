# Task-140 Execution Findings

## Tasks 1–5: complete and reviewed

All five code tasks landed on `task-140-morph-potion-routing` and passed per-task
spec+quality review:

| Task | Commit | What |
|---|---|---|
| 1 | `52b14136c` | `Morphs()` getter on the data-side consumable model |
| 2 | `1cb082260` | `selectMorph` (pure) + `rollMorph` (crypto/rand) seam in `consumable/morph.go` |
| 3 | `906d5cb82` | Behavior-preserving extraction of pure `computeEffectPlan` from `ApplyItemEffects` |
| 4 | `ed827c2dc` | morph-random branch with fixed-morph precedence (FR-7) |
| 5 | `bcb740c3c` | route classification 221 through `ConsumeStandard` (named constant) |

Verification (Task 6 Steps 1–3):
- `go test -race ./...`, `go vet ./...`, `go build ./...` — all clean from the module root.
- `tools/redis-key-guard.sh`, `tools/goroutine-guard.sh`, `tools/lint.sh --check` — all exit 0.
- Diff scope: only `services/atlas-consumables/**` + `docs/tasks/task-140-morph-potion-routing/**`. No `go.mod` change → no docker bake required.

## Task 6 Step 3b (legacy/live verification) surfaced a BLOCKER for the morphRandom feature

Read-only live query against `atlas-data` in namespace `atlas-main`
(`GET /api/data/consumables/{id}` with per-tenant REGION/MAJOR_VERSION headers).

### Full 221-family map (v83-era WZ, verified against Cosmic `Item.wz/Consume/0221.img.xml`)

- **28 fixed-morph potions** `02210000`–`02210043`: each authors `spec.morph=<id>`
  (ids 1..49), no `morphRandom`. These route through the fixed-morph branch and
  apply correctly. Confirmed live: `2210000` → `spec.morph=1`, empty `morphs`.
- **2 morphRandom potions** `02211000`, `02212000`: `spec.morph=0`, weighted
  `morphRandom` table. These are the ONLY items exercising Task 2's weighted seam.

### The blocker: atlas-data ingests the wrong WZ field name

Raw WZ (`0221.img.xml`, item `02211000/spec/morphRandom`):

```
0: morph=23 prop=15
1: morph=24 prop=10
2: morph=25 prop=10
3: morph=26 prop=10
4: morph=27 prop=15
5: morph=28 prop=10
6: morph=29 prop=15
7: morph=30 prop=15      (sum = 100)
```

The weight field is **`prop`** (a WZ `<string>`), but atlas-data reads **`"prob"`**:

```
services/atlas-data/atlas.com/data/consumable/reader.go:158
    prob := uint32(mo.GetIntegerWithDefault("prob", 0))   // <- WZ field is "prop"
```

`GetIntegerWithDefault` DOES coerce string nodes to int
(`services/atlas-data/atlas.com/data/xml/model.go:82`), so the fix is name-only.
Because `"prob"` never matches, every weight ingests as the `0` default. Live
result (v72/v79/v83 identical):

```
2211000 morphs = {"23":0,"24":0,"25":0,"26":0,"27":0,"28":0,"29":0,"30":0}
2212000 morphs = {"23":0,"24":0,"25":0,"26":0,"27":0,"28":0,"29":0,"30":0}
```

Zero total weight → `rollMorph` errors → `computeEffectPlan` warns and skips the
morph statup (design §6). **Net: 02211000 / 02212000 apply no morph in production**
on every reachable version — the exact "dead code" symptom this task set out to fix,
relocated one layer down.

### Per-version presence

| Version | 2210000 (fixed) | 2211000 (random) | 2212000 (random) |
|---|---|---|---|
| v48 | present (morph=1) | absent | absent |
| v61 | present (morph=1) | absent | absent |
| v72 | present (morph=1) | present, weights 0 | present, weights 0 |
| v79 | present (morph=1) | present, weights 0 | present, weights 0 |
| v83 | present (morph=1) | present, weights 0 | present, weights 0 |

### Scope conflict requiring a decision

The one-word fix (`reader.go:158` `"prob"` → `"prop"`, plus a regression test)
lives in **atlas-data**, but the plan's Global Constraints and PRD acceptance
criterion #6 mandate the diff touch **only** `services/atlas-consumables/`. The
design assumed the data was already correct ("the morph applier already exists").
That assumption is false for the two morphRandom items. Resolving this crosses an
explicit acceptance criterion, so it is escalated to the user (options: fix
atlas-data on this branch / fix in a separate branch / defer as a filed follow-up).

The fixed-morph majority (28 items) is unaffected and fully working regardless of
the decision.

## Post-merge live testing surfaced a second (pre-existing) bug: buff duration off by 1000x

Testing `2210000` on the PR env (`atlas-pr-1085`), the morph lasted ~10-15s instead
of its WZ `spec.time` = 3600000 ms (60 min). Live atlas-buffs log:

```
19:14:19.406  APPLY sourceId=-2210000 duration=3600 MORPH amount=1
19:14:28.992  Expired buff for character [1] from [-2210000]     # ~9.6s later
```

Root cause — a **unit mismatch that is a task-054 regression**:
- `consumable/processor.go` computed `plan.duration = val / 1000`, converting the
  WZ millisecond `time` spec to **seconds** (dates to `765be8b406`, 2026-01-04).
- atlas-buffs `buff/model.go:112` schedules expiry as
  `now + time.Duration(duration) * time.Millisecond` — i.e. it expects **ms**.
  This was `* time.Second` until `197324e40` ("feat(atlas-buffs): interpret
  Duration as ms (task-054)", 2026-05-03), which updated the skill path (atlas-data
  `skill/reader.go:169` multiplies WZ seconds by 1000) but MISSED the consumable
  `/1000` caller.
- Net since 2026-05-03: every timed consumable buff (attack/defense/speed potions,
  and now morph) expired ~1000x early, killed by the 10s atlas-buffs expiry sweep.

Fixed on this branch (user-approved): `plan.duration = val` (send the WZ ms value
as-is), tests updated to pin ms durations (`300000`/`600000`, not `300`/`600`).

## Cancellation semantics (verified, for reference)

A MORPH temporary stat is cancelled by: (1) **timed expiry** (now correct after the
duration fix) and (2) **death → respawn** (`respawn/processor.go:241` saga
`cancel_all_buffs` → atlas-buffs `CancelAll`). It is NOT cancelled by attacking
(`character_attack_*.go` never call `buff.Cancel`), by taking damage
(`character_damage.go` only changes HP), or by a plain map change (morph is
re-rendered). Right-click buff-cancel is not server-blocked (the CANCEL_ITEM
handler cancels by `sourceId = -itemId` with no MORPH filter), but the client
generally does not offer a cancel for morph icons — so nothing reaches the server.
These match the design's "morph-cancel-on-hit is a verified non-mechanic" note.
