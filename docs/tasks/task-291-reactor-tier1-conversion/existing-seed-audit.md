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
