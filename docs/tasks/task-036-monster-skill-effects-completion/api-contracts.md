# API Contracts — task-036

Companion to `prd.md`. Locks Kafka payload shapes and consumer responsibilities so the plan phase has no ambiguity to resolve.

---

## 1. `EVENT_TOPIC_MONSTER_STATUS` — extended `StatusEffectAppliedBody`

**Producer:** atlas-monsters (`kafka/producer/...`)
**Consumers:** atlas-channel (existing `handleStatusEffectApplied`; new `StatusMirror.OnApplied`)

### Body — full shape (extension highlighted)

```jsonc
{
  "effectId": "uuid",
  "sourceType": "MONSTER_SKILL" | "PLAYER_SKILL",
  "sourceCharacterId": 0,
  "sourceSkillId": 0,
  "sourceSkillLevel": 0,
  "statuses": { "WEAPON_REFLECT": 30 },
  "duration": 60000,
  "tickInterval": 0,

  // NEW — populated only when status category is reflect.
  "reflectKind": "PHYSICAL" | "MAGICAL" | "",
  "reflectPercent": 30,
  "reflectRange": 200,
  "reflectMaxDamage": 32767
}
```

### Marshalling rules

- `reflectKind` is a string, defaults to `""`. Do **not** use `omitempty` (cjson safety).
- Numeric reflect fields default to `0` for non-reflect statuses. Always serialize.
- The `Statuses` map MUST always be marshalled — even if empty — as `{}` not `null`. (Existing code; re-verify under cjson audit.)

### Consumer contract — atlas-channel

1. `handleStatusEffectApplied` continues to drive the existing `MonsterStatSet` broadcast.
2. Same handler also calls `StatusMirror.OnApplied(uniqueId, body)` to populate the in-memory mirror.
3. For `Statuses` keys matching `VENOM_1|VENOM_2|VENOM_3`: the consumer MUST translate to a single `VENOM` entry on the wire packet (same skill id / level / amount of the slot's snapshot value); apply order is irrelevant — first-write-wins is acceptable.

---

## 2. `EVENT_TOPIC_MONSTER_STATUS` — `StatusEffectExpiredBody` and `StatusEffectCancelledBody`

No body changes. New consumer responsibilities only:

1. `StatusMirror.OnExpired(uniqueId, statuses)` and `OnCancelled(uniqueId, statuses)` — remove entries from the mirror.
2. For `VENOM_N` keys: the wire `MonsterStatReset` for `VENOM` MUST be emitted **only when the last venom slot is removed**. The mirror tracks slot counts to make this decision; consumer queries the mirror before producing the reset packet.

---

## 3. `EVENT_TOPIC_MONSTER_STATUS` — `StatusEventDamageReflectedBody` (already exists)

**Producer:** atlas-channel (NEW — produced from the attack handler)
**Consumer:** atlas-channel (existing `handleDamageReflected` — unchanged)

```jsonc
{
  "type": "DAMAGE_REFLECTED",
  "uniqueId": 12345,         // monster
  "body": {
    "characterId": 67890,    // attacker who takes the reflect damage
    "reflectDamage": 250
  }
}
```

### Producer rules (attack handler)

1. Compute reflect per damage entry (FR-4.3.1).
2. Set the entry's `damage` to `0` **before** any HP/aggro write so the monster ignores the hit.
3. Produce the event with the reflected amount.
4. Do NOT also produce a `DAMAGED` event for the same entry — the `MonsterDamage` echo is not desired for reflected hits.

---

## 4. `EVENT_COMMAND_TOPIC_MIST` — atlas-maps inbound

**Producer:** atlas-monsters (`executeMist`)
**Consumer:** atlas-maps (`mist` domain command consumer)

### Topic env

`COMMAND_TOPIC_MIST` resolved via the existing `topic.EnvProvider` pattern.

### Header

Standard `tenant.Model` header parser; consumer fails fast on missing tenant.

### Body — `MIST_CREATE`

```jsonc
{
  "tenant": "...",
  "type": "MIST_CREATE",
  "body": {
    "worldId": 0,
    "channelId": 0,
    "mapId": 100000000,
    "instance": "uuid-or-nil",
    "ownerType": "MONSTER",
    "ownerId": 12345,
    "origin":  { "x": 100, "y": 200 },
    "ltX": -50, "ltY": -30, "rbX": 50, "rbY": 30,
    "disease": "POISON",
    "diseaseValue": 80,
    "diseaseDuration": 30000,
    "duration": 10000,
    "tickIntervalMs": 1000
  }
}
```

### Body — `MIST_CANCEL`

```jsonc
{
  "tenant": "...",
  "type": "MIST_CANCEL",
  "body": { "mistId": "uuid" }
}
```

### Consumer rules

1. `MIST_CREATE` → `mist.NewProcessor(l, ctx).Create(body)` → registry insert + `MIST_CREATED` event emit.
2. `MIST_CANCEL` → registry remove + `MIST_DESTROYED` event emit (reason `CANCELLED`).
3. Ticker (sibling task) walks active mists every 1 s, expires by `ExpiresAt`, and emits `MIST_DESTROYED` with reason `EXPIRED`.

---

## 5. `EVENT_TOPIC_MIST` — atlas-maps outbound

**Producer:** atlas-maps (`mist` domain producer)
**Consumer:** atlas-channel (new `mist` consumer that broadcasts `AffectedAreaCreated/Removed` packets)

### Topic env

`EVENT_TOPIC_MIST`.

### Body — `MIST_CREATED`

```jsonc
{
  "tenant": "...",
  "worldId": 0,
  "channelId": 0,
  "mapId": 100000000,
  "instance": "uuid-or-nil",
  "mistId": "uuid",
  "type": "MIST_CREATED",
  "body": {
    "ownerType": "MONSTER",
    "ownerId": 12345,
    "origin": { "x": 100, "y": 200 },
    "ltX": -50, "ltY": -30, "rbX": 50, "rbY": 30,
    "duration": 10000
  }
}
```

### Body — `MIST_DESTROYED`

```jsonc
{
  "tenant": "...", "worldId": 0, "channelId": 0,
  "mapId": 100000000, "instance": "uuid-or-nil",
  "mistId": "uuid",
  "type": "MIST_DESTROYED",
  "body": { "reason": "EXPIRED" | "CANCELLED" }
}
```

### Consumer rules — atlas-channel

1. `MIST_CREATED` → `ForSessionsInMap(field, AffectedAreaCreatedWriter(mistId, origin, bounds, duration))`.
2. `MIST_DESTROYED` → `ForSessionsInMap(field, AffectedAreaRemovedWriter(mistId))`.

Disease metadata is intentionally absent from outbound events — it is internal to the mist tick task in atlas-maps.

---

## 6. `MistTickTask` — internal contract (atlas-maps)

Not a Kafka contract; documented here to lock the apply path.

For each active mist, every `tickInterval`:

1. List character IDs in `mist.Field()` via the existing `_map.CharacterIdsInFieldProvider` (or atlas-maps' equivalent).
2. For each character, query position; if position is within absolute bounds:
   1. Produce an apply-disease command on `EnvCommandTopicCharacterBuff` — same shape as `applyDiseaseCommandProvider` in atlas-monsters. Disease value, duration, source skill id are taken from the mist record.
3. Update `mist.LastTick`.

If the character has Holy Shield active, atlas-buffs already rejects the apply (`character/processor.go:43-44`). No extra check is needed in atlas-maps.

---

## 7. `EVENT_COMMAND_TOPIC_CHARACTER_BUFF` — apply-disease re-use

No change. Both atlas-monsters' existing `executeDebuff` and the new mist tick task produce the same disease-apply command. The consumer in atlas-buffs is unchanged.

---

## 8. `PoisonTick` task — atlas-buffs

Not a new topic. The task produces existing character-damage commands.

### Per-tick algorithm

```
for each tenant t:
  ctx := tenant.WithContext(taskCtx, t)
  entries := character.GetRegistry().GetPoisonCharacters(ctx)
  now := time.Now()
  for each e in entries:
    last, ok := character.GetRegistry().GetLastPoisonTick(ctx, e.CharacterId)
    if ok && now.Sub(last) < tickInterval: continue
    produce CHARACTER_DAMAGE command for (e.CharacterId, amount=e.Amount, source=POISON)
    character.GetRegistry().UpdatePoisonTick(ctx, e.CharacterId, now)
```

### Topic / shape resolution (open)

Plan phase MUST identify the existing character-damage command topic. If none exists, the plan adds `EVENT_COMMAND_TOPIC_CHARACTER_DAMAGE` with body `{characterId, amount, source}` — see `prd.md` §9 question 3.

---

## 9. cjson empty-array audit checklist

| Body | Slice fields to verify |
|---|---|
| `StatusEventDamagedBody` | `DamageEntries` |
| any new mist body | none currently — but verify if helpers like `[]character.Id{}` get added |
| `StatusEffectAppliedBody` | none, but the embedded `Statuses` map must remain `{}` not `null` |

For each slice field, the marshalling test asserts:

```go
b := Body{Slice: nil}
out, _ := json.Marshal(b)
require.Contains(t, string(out), `"slice":[]`)
```
