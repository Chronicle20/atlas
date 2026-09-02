# Existing Seed Audit — task-291

Thirteen reactor-action seeds existed before this task. Each was read against
its Cosmic source and given a recorded outcome. Sources are referenced as
`<Cosmic>/scripts/reactor/<id>.js`; seeds as
`deploy/seed/gms/83_1/reactor-actions/reactors/reactor-<id>.json` (the eleven
tenant copies are byte-identical).

## Outcomes

| Reactor | Source body | Seeded shape | Verdict |
|---|---|---|---|
| 2001 | `rm.dropItems(true, 2, 8, 15, 1)` | `drop_items` | **corrected**: `minMeso=2, maxMeso=8, mesoRange=15, item=1` → `mesoChance=2, mesoMin=8, mesoMax=15, minItems=1` |
| 9102002 | `rm.dropItems();` | `drop_items`, no params | correct |
| 9102003 | `rm.dropItems();` | `drop_items`, no params | correct |
| 9102004 | `rm.dropItems();` | `drop_items`, no params | correct |
| 9102005 | `rm.dropItems();` | `drop_items`, no params | correct |
| 9102006 | `rm.dropItems();` | `drop_items`, no params | correct |
| 9102007 | `rm.dropItems();` | `drop_items`, no params | correct |
| 9108000 | `eim` stage-advance script: increments PQ `stage` property, force-hits the `fullmoon` reactor, and on reaching stage 6 broadcasts a map message and spawns a monster | Two `actRules`: a `pq_custom_data stage = 5` guarded rule (`update_pq_state` increments `stage`, `hit_reactor` on `fullmoon`, `stage_clear_attempt`) ordered first, then an unguarded fallback rule (`update_pq_state` increments `stage`, `hit_reactor` on `fullmoon`) | correct — the declarative model of "advance stage, hit fullmoon, attempt stage-clear once stage reaches 6" matches the script's behavior; rule order (guarded rule first) preserves the source's stage-6 special case since `processor.go` takes the first matching rule |
| 9108001 | identical to 9108000 | identical to 9108000 | correct |
| 9108002 | identical to 9108000 | identical to 9108000 | correct |
| 9108003 | identical to 9108000 | identical to 9108000 | correct |
| 9108004 | identical to 9108000 | identical to 9108000 | correct |
| 9108005 | identical to 9108000 | identical to 9108000 | correct |

## Verification

- `grep -rn 'minMeso\|maxMeso\|mesoRange\|"item"' deploy/seed/*/*/reactor-actions/` — no output after the `reactor-2001.json` correction.
- `md5sum deploy/seed/*/*/reactor-actions/reactors/reactor-2001.json | awk '{print $1}' | sort -u | wc -l` — `1` (all eleven tenant copies byte-identical).
- All twelve PQ seeds (`9102002`–`9102007`, `9108000`–`9108005`) were confirmed byte-identical across all eleven tenant copies via `md5sum`; none required correction.

## Sampled Conversion Review (PRD §8 gate 2)

One converted reactor per shape, read against its body in `tier1-inventory.md`
(source: `docs/tasks/task-291-reactor-tier1-conversion/tier1-inventory.md`;
generated: `deploy/seed/gms/83_1/reactor-actions/reactors/reactor-<id>.json`).

| Shape | Reactor | Source body | Emitted rules | Verdict |
|---|---|---|---|---|
| `drop_items` only | `1012000` | `rm.dropItems(true, 2, 20, 40);` | `actRules`: one `drop_items` op, `meso=true, mesoChance=2, mesoMin=20, mesoMax=40` | matches — `dropItems(meso, chance, min, max)` positional args map 1:1 onto the four params |
| `spray_items` only | `1052001` | `rm.sprayItems(true, 1, 500, 1000, 15);` | `actRules`: one `spray_items` op, `meso=true, mesoChance=1, mesoMin=500, mesoMax=1000, minItems=15` | matches — `sprayItems(meso, chance, min, max, minItems)` args map 1:1 |
| `spawn_monster` only | `8001000` | `rm.spawnMonster(9400112, 1, 420, 160);` | `actRules`: one `spawn_monster` op, `monsterId=9400112, count=1, x=420, y=160` | matches |
| `weaken_area_boss` only | `2619003` | `rm.weakenAreaBoss(6090004, "Rurumo has been poisoned. It may finally be defeatable!");` | `actRules`: one `weaken_area_boss` op, `monsterId=6090004, message="Rurumo has been poisoned. It may finally be defeatable!"` | matches |
| empty `act()` | `9018000` | `function act() {\n}` | `actRules: []`, `hitRules: []` | matches — no operations emitted for an empty body, per FR-19 |
| `reactor_state` + `weaken_area_boss` | `2119000` | guarded `hit()`: `if (rm.getReactor().getState() !== 0) { return } rm.weakenAreaBoss(6090000, "As the tombstone lit up and vanished, Lich lost all his magic abilities.")`; `act()` is empty | `hitRules`: one rule with condition `reactor_state = 0`, operation `weaken_area_boss` (`monsterId=6090000, message="As the tombstone lit up and vanished, Lich lost all his magic abilities."`); `actRules: []` | matches — the `state !== 0` early-return guard is correctly inverted to the `reactor_state = 0` gate on the rule |
| `update_pq_state` only | `2008006` | `rm.getEventInstance().setProperty("statusStg3", "0");` | `actRules`: one `update_pq_state` op, `updates=statusStg3=0` | matches |
| `drop_items` + `update_pq_state` | `2002003` | `rm.dropItems(); var eim = rm.getEventInstance(); eim.setProperty("statusStg7", "1");` | `actRules`: one rule (`drop_items_update_pq_state`) with two ordered operations — `drop_items` (no params), then `update_pq_state` (`updates=statusStg7=1`) | matches — operation order preserves source statement order |
| `update_pq_state` + `spawn_monster` | `2511000` | `openedBoxes` increment via `eim.getIntProperty`/`setIntProperty`, then `rm.spawnMonster(9300109, 3); rm.spawnMonster(9300110, 5);` | `actRules`: one rule (`update_pq_state_spawn_monster`) with three ordered operations — `update_pq_state` (`increments=openedBoxes`), `spawn_monster` (`monsterId=9300109, count=3`), `spawn_monster` (`monsterId=9300110, count=5`) | matches — the read-increment-write pattern is recognized as `increments`, both spawns preserved in source order with counts |
| `update_pq_state` + `spray_items` | `2512001` | `openedChests` increment, then `rm.sprayItems(true, 1, 50, 100, 15);` | `actRules`: one rule (`update_pq_state_spray_items`) with two ordered operations — `update_pq_state` (`increments=openedChests`), `spray_items` (`meso=true, mesoChance=1, mesoMin=50, mesoMax=100, minItems=15`) | matches |
| loop unroll (mandated) | `2201001` | `for (var i = 0; i < 3; i++) { rm.spawnMonster(9300007); }` | `actRules`: one `spawn_monster` op, `monsterId=9300007, count=3` (no `x`/`y`, unspecified in source) | matches — the three identical zero-arg calls are folded into one op with `count=3` |
| loop unroll (mandated) | `2511001` | `for (var i = 0; i < 6; i++) { rm.spawnMonster(9300124); rm.spawnMonster(9300125); }` | `actRules`: one rule with two ordered `spawn_monster` ops — `monsterId=9300124, count=6` then `monsterId=9300125, count=6` | matches — the two calls per iteration are unrolled per-monster-id into two ops, each folded to `count=6`, in source order |
| meso-shift regression (mandated) | `2001` | `rm.dropItems(true, 2, 8, 15, 1);` | `actRules`: one `drop_items` op, `meso=true, mesoChance=2, mesoMin=8, mesoMax=15, minItems=1` | matches the corrected mapping already recorded in the Outcomes table above — **discrepancy from the brief**: `2001` is *not* present in `tier1-inventory.md` (`grep -n '^### \`2001\`$' tier1-inventory.md` returns no match). It is one of the thirteen pre-existing seeds audited above under Task 6/7 and corrected there; it is not part of the 159-reactor Tier-1 corpus that `tier1-inventory.md` documents. Its source body used here is the one already recorded in this file's Outcomes table, not `tier1-inventory.md`. |

All thirteen sampled reactors' generated `actRules`/`hitRules` were read directly
from `deploy/seed/gms/83_1/reactor-actions/reactors/reactor-<id>.json` and
compared operation-by-operation, parameter-by-parameter, against the quoted
source body. No mismatches were found in the sample.
