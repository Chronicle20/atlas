# PRD — Correct Reactor Type Semantics & Timer-Driven Progression

## Problem

Hit-driven reactors in Atlas are currently stuck at their terminal state instead of being destroyed and respawning. Observed symptom: in a map populated with 5 reactors of classification `0002001` ("Maple Island generic reactor" / box, action `mBoxItem0`), hitting each reactor advances state 0→1→2→3→4, then the reactor sits at state 4 forever. atlas-maps's 10-second spawn task prints `existing=[5], issuing=[0] CREATE commands` indefinitely, because the zombie state-4 reactors are still in the GET response. New reactors never spawn.

### Root causes

1. **atlas-data synthesises phantom events.** `services/atlas-data/atlas.com/data/reactor/reader.go:124-130` fabricates `{Type: 999, NextState: i+1}` for any state that exists in the .wz but has no `event` subtree. This synthetic marker leaks into `StateInfo` as if it were real reactor data.

2. **`persistsAtFinalState` matches the phantom.** `services/atlas-reactors/atlas.com/reactors/reactor/processor.go:259-270` scans every event in every state and returns `true` if any event is type `100` or `999`. For reactor `0002001` (all real events are type 0), the synthesised 999 in the terminal state falsely trips the check, and the reactor is kept alive as if it were an item reactor.

3. **Wrong matching scope.** Even for real item reactors, `persistsAtFinalState` is too broad — it checks "any event anywhere" rather than "the event that led to the end-state transition."

4. **`timeOut` casing bug.** wz files use `timeOut` 157 times and lowercase `timeout` only 3 times; reader.go reads lowercase only (`reader.go:89`), so Atlas loses nearly all animation timing metadata. This prevents timer-driven reactors from ever auto-progressing.

### Impact

- Reactor 2001 (and every other reactor whose terminal state has no events in the .wz, which is most breakable reactors) gets stuck at terminal state. No respawn.
- Type-101 timer-cyclic reactors (Balrog altars, buff reactors, Pink Bean PQ, Magatia PQ, Mu Lung PQ, PPQ, etc.) don't cycle — they depend on `timeOut` which is never read.
- Moon Bunny (`9101000`) and similar pure-timer reactors don't progress at all — they have no hit events, and the synthesis just papers over the missing timer.

## Goals

1. Reactor 2001 and equivalent hit-break reactors destroy correctly on terminal state and respawn via the existing cooldown + atlas-maps spawn mechanism.
2. Item reactors (type 100), GPQ skill reactors (types 5/6/7), and timer-cyclic reactors (type 101) still persist at their end-states — behavior the game depends on.
3. Timer-driven state progression works for type-101 reactors and for reactors with per-state `timeOut` values.
4. atlas-data stops fabricating event data it didn't read from the .wz.

## Non-Goals

- `activateByTouch` — deferred (see `docs/TODO.md`, Reactors Service section). All 9 affected reactors (GPQ 6109013, 6109014, 6109021-6109027) are still activatable via their type-5/6/7 skill events.
- Redesigning atlas-maps's periodic spawn task.
- Revisiting the Redis spot-claim mechanism (task-016 / earlier work in this session — still correct).
- Reactor script execution semantics (atlas-reactor-actions is unchanged).

## Reactor type taxonomy (survey of 419 `Reactor.wz/*.img.xml` files, 1,188 events)

| type | count | meaning                                                       | end-state behavior |
|------|-------|---------------------------------------------------------------|--------------------|
| 0    | 579   | Hit by any attack (default breakable)                         | destroy + cooldown |
| 100  | 205   | Item-drop trigger (has `lt`/`rb` area + item ID/quantity)     | **persist**        |
| 101  | 161   | Timer-driven auto-advance (paired with `timeOut`)             | **persist (cyclic)** |
| 2    | 106   | Hit from one direction (e.g. "hit from right")                | destroy + cooldown |
| 1    | 104   | Hit from the other direction (pair with type 2)               | destroy + cooldown |
| 5    | 15    | Skill-gated with `activeSkillID` (GPQ job-skill reactors)     | **persist**        |
| 6    | 9     | Same family as 5                                              | **persist**        |
| 7    | 9     | Same family as 5                                              | **persist**        |

No `type=999` exists in the wz. Every 999 in Atlas today is synthetic.

## Success Criteria

1. Hit a reactor 2001 five times: state progresses 0→1→2→3→4, destroy event is emitted, cooldown registered, Redis spot released.
2. Drop a moon-seed on a moonflower (9108xxx): transitions to state 1 and persists until instance teardown.
3. A reactor whose data has per-state `timeOut` auto-advances on its own after the configured milliseconds, with no hit required.
4. No `type 999` values appear anywhere in the system except as the literal value a .wz might encode (none in our data set).
5. `timeOut` values from the .wz are preserved into `timeoutInfo` (both casings honored).
6. Existing atlas-reactors tests pass; new tests cover the rule change and timer firing/cancellation.
7. The cross-service cycle (atlas-reactors → atlas-channel destroy broadcast → atlas-maps respawn) works end-to-end in a manual playtest: break all reactors in a test map, observe fresh reactors spawn via atlas-maps's 10s tick.
