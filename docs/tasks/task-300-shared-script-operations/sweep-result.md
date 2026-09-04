# FR-20 sweep re-run (Task 14)

Command (run from the worktree root):

```
python3 docs/tasks/task-300-shared-script-operations/sweep-seed-scripts.py
```

Date run: 2026-09-04 13:33:34 UTC

## Verbatim output

```
scanned 22780 seed documents

== parameter shapes (op, param, kind) -> count
  ('drop_message', 'message', 'text') -> 77  e.g. 'The gate is not opened yet.'
  ('drop_message', 'messageType', 'text') -> 22  e.g. 'PINK_TEXT'
  ('send_message', 'message', 'text') -> 129  e.g. 'You have acquired a Dragon Egg.'
  ('send_message', 'messageType', 'text') -> 129  e.g. 'PINK_TEXT'
  ('spawn_monster', 'count', 'integer') -> 33
  ('spawn_monster', 'mapId', 'integer') -> 11
  ('spawn_monster', 'monsterId', 'integer') -> 110
  ('spawn_monster', 'x', 'integer') -> 110
  ('spawn_monster', 'y', 'integer') -> 110
  ('warp', 'mapId', 'integer') -> 594
  ('warp', 'portalId', 'integer') -> 209
  ('warp', 'portalName', 'text') -> 374  e.g. 'east00'
  ('warp_to_map', 'mapId', 'context-ref') -> 209  e.g. '{context.destination}'
  ('warp_to_map', 'mapId', 'integer') -> 3168
  ('warp_to_map', 'portalId', 'context-ref') -> 44  e.g. '{context.returnPortalId}'
  ('warp_to_map', 'portalId', 'integer') -> 3038
  ('warp_to_map', 'portalName', 'text') -> 284  e.g. 'st00'

== warp / warp_to_map missing mapId
  none

== message-type keys (op, messageType, type) -> count
  ('drop_message', 'PINK_TEXT', None) -> 22
  ('drop_message', None, None) -> 55
  ('send_message', 'BLUE_TEXT', None) -> 22
  ('send_message', 'PINK_TEXT', None) -> 52
  ('send_message', 'POP_UP', None) -> 55

== spawn_monster with an explicit mapId (OQ-3 instance rule)
  deploy/seed/gms/12_1/npc-conversations/npc/npc-1063017.json: {'mapId': '910510202', 'monsterId': '9300346', 'x': '95', 'y': '200'}
  deploy/seed/gms/48_1/npc-conversations/npc/npc-1063017.json: {'mapId': '910510202', 'monsterId': '9300346', 'x': '95', 'y': '200'}
  deploy/seed/gms/61_1/npc-conversations/npc/npc-1063017.json: {'mapId': '910510202', 'monsterId': '9300346', 'x': '95', 'y': '200'}
  deploy/seed/gms/72_1/npc-conversations/npc/npc-1063017.json: {'mapId': '910510202', 'monsterId': '9300346', 'x': '95', 'y': '200'}
  deploy/seed/gms/79_1/npc-conversations/npc/npc-1063017.json: {'mapId': '910510202', 'monsterId': '9300346', 'x': '95', 'y': '200'}
  deploy/seed/gms/83_1/npc-conversations/npc/npc-1063017.json: {'mapId': '910510202', 'monsterId': '9300346', 'x': '95', 'y': '200'}
  deploy/seed/gms/84_1/npc-conversations/npc/npc-1063017.json: {'mapId': '910510202', 'monsterId': '9300346', 'x': '95', 'y': '200'}
  deploy/seed/gms/87_1/npc-conversations/npc/npc-1063017.json: {'mapId': '910510202', 'monsterId': '9300346', 'x': '95', 'y': '200'}
  deploy/seed/gms/92_1/npc-conversations/npc/npc-1063017.json: {'mapId': '910510202', 'monsterId': '9300346', 'x': '95', 'y': '200'}
  deploy/seed/gms/95_1/npc-conversations/npc/npc-1063017.json: {'mapId': '910510202', 'monsterId': '9300346', 'x': '95', 'y': '200'}
  deploy/seed/jms/185_1/npc-conversations/npc/npc-1063017.json: {'mapId': '910510202', 'monsterId': '9300346', 'x': '95', 'y': '200'}
```

## Comparison against design §7

| design §7 row | re-run result | match |
|---|---|---|
| every `spawn_monster` `monsterId`/`x`/`y`/`count`/`mapId` value is a plain integer | all `spawn_monster` param shapes reported are `'integer'` (`monsterId` x110, `x` x110, `y` x110, `count` x33, `mapId` x11) | yes |
| no seeded `spawn_monster` carries `team` | `team` does not appear in the parameter-shapes table | yes |
| no `warp`/`warp_to_map` operation is missing `mapId` | `== warp / warp_to_map missing mapId` reports `none` | yes |
| no `drop_message` uses the `type` key or a numeric `5`/`6` | message-type table shows only `messageType` keys (`PINK_TEXT` or absent); no `type` key or numeric value appears | yes |
| no `send_message` is missing `messageType` | `send_message` has 129 `message` occurrences and 129 `messageType` occurrences — every `send_message` carries `messageType` | yes |
| `deploy/seed/*/npc-conversations/npc/npc-1063017.json` is the one cross-map spawn | the `spawn_monster` with explicit `mapId` section lists only `npc-1063017.json` (11 tenant copies, all `mapId: 910510202`) | yes |

All six rows agree with design §7's recorded findings. No mismatch to fix under FR-20.
