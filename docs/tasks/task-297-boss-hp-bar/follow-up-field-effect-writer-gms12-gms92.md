# Follow-up: `template_gms_12_1` and `template_gms_92_1` lack a working `FieldEffect` writer

Recorded from design OQ1 (resolved flag-and-defer). AC-15.

## Gap

`CField::OnFieldEffect` (writer key `FieldEffect`, see
`docs/packets/dispatchers/field_effect.yaml`) is the single dispatcher that carries all eight
`FIELD_EFFECT` sub-operations, including `BOSS_HP` (mode `5` on every version the yaml
documents). Two of the eleven seeded client templates do not route to that writer:

- `services/atlas-configurations/seed-data/templates/template_gms_12_1.json` has **no**
  `FieldEffect` writer entry and **no** `FieldEffectWeather` entry either — a `grep -n
  'FieldEffect'` against the file returns nothing.
- `services/atlas-configurations/seed-data/templates/template_gms_92_1.json` has a
  `FieldEffectWeather` writer only, at `template_gms_92_1.json:2540`
  (`"writer": "FieldEffectWeather",`). It has no `FieldEffect` writer entry.

Boss HP session announcements (task-297) target the `FieldEffect` writer's `BOSS_HP` key.
Neither template can serve that announcement.

## Evidence

Re-verified independently of the design doc, against this worktree's HEAD:

```
$ grep -l '"writer": "FieldEffect"' services/atlas-configurations/seed-data/templates/*.json
services/atlas-configurations/seed-data/templates/template_gms_48_1.json
services/atlas-configurations/seed-data/templates/template_gms_61_1.json
services/atlas-configurations/seed-data/templates/template_gms_72_1.json
services/atlas-configurations/seed-data/templates/template_gms_79_1.json
services/atlas-configurations/seed-data/templates/template_gms_83_1.json
services/atlas-configurations/seed-data/templates/template_gms_84_1.json
services/atlas-configurations/seed-data/templates/template_gms_87_1.json
services/atlas-configurations/seed-data/templates/template_gms_95_1.json
services/atlas-configurations/seed-data/templates/template_jms_185_1.json
```

Nine files. Cross-checked against the full template inventory (eleven files:
`gms_{12,48,61,72,79,83,84,87,92,95}_1` plus `jms_185_1`), the two files absent from that list
are `template_gms_12_1.json` and `template_gms_92_1.json`.

```
$ grep -n 'FieldEffect' services/atlas-configurations/seed-data/templates/template_gms_12_1.json \
    services/atlas-configurations/seed-data/templates/template_gms_92_1.json
services/atlas-configurations/seed-data/templates/template_gms_92_1.json:2540:        "writer": "FieldEffectWeather",
```

`template_gms_92_1.json` returns one hit (`FieldEffectWeather`, not `FieldEffect`).
`template_gms_12_1.json` returns **zero** hits — it carries neither writer.

```
$ cat docs/packets/dispatchers/field_effect.yaml
# FieldEffect — CField::OnFieldEffect per-version mode table.
# Version-STABLE: switch cases 0..7 byte-identical in v83 (0x5330f7) and v95
# (0x53b790), IDA-verified. All 8 keys match the seeded gms_83 table.
writer: FieldEffect
fname: CField::OnFieldEffect
op: FIELD_EFFECT
direction: clientbound
operations:
  - { key: SUMMON,           modes: { gms_v83: 0, gms_v84: 0, gms_v87: 0, gms_v95: 0, jms_v185: 0 } }
  - { key: TREMBLE,          modes: { gms_v83: 1, gms_v84: 1, gms_v87: 1, gms_v95: 1, jms_v185: 1 } }
  - { key: OBJECT,           modes: { gms_v83: 2, gms_v84: 2, gms_v87: 2, gms_v95: 2, jms_v185: 2 } }
  - { key: SCREEN,           modes: { gms_v83: 3, gms_v84: 3, gms_v87: 3, gms_v95: 3, jms_v185: 3 } }
  - { key: SOUND,            modes: { gms_v83: 4, gms_v84: 4, gms_v87: 4, gms_v95: 4, jms_v185: 4 } }
  - { key: BOSS_HP,          modes: { gms_v83: 5, gms_v84: 5, gms_v87: 5, gms_v95: 5, jms_v185: 5 } }
  - { key: BACKGROUND_MUSIC, modes: { gms_v83: 6, gms_v84: 6, gms_v87: 6, gms_v95: 6, jms_v185: 6 } }
  - { key: REWARD_RULLET,    modes: { gms_v83: 7, gms_v84: 7, gms_v87: 7, gms_v95: 7, jms_v185: 7 } }
```

`field_effect.yaml` documents mode columns for `gms_v83`, `gms_v84`, `gms_v87`, `gms_v95`,
and `jms_v185` only. There is no `gms_v92` (or `gms_v12`, `gms_v48`, `gms_v61`, `gms_v72`,
`gms_v79`) column — consistent with the writer not existing on the templates that lack it,
and with the mode table not yet having IDA-verified those versions' switch layouts.

### Divergence from the design's OQ1 statement

Design OQ1 states both `template_gms_12_1.json` and `template_gms_92_1.json` "carry only
`FieldEffectWeather`." That is true for `template_gms_92_1.json` (confirmed above) but **not**
for `template_gms_12_1.json`, which carries neither `FieldEffect` nor `FieldEffectWeather` —
it has no field-effect-family writer of any kind. The net effect for this task is unchanged
(neither template can serve a `BOSS_HP` `FieldEffect` announcement), but the design's premise
that gms_12 falls back to `FieldEffectWeather` does not hold; it has nothing to fall back to.

## Blast radius

The gap is not `BOSS_HP`-specific. `FieldEffect` is the single writer for the entire
`FIELD_EFFECT` operation family (`field_effect.yaml`, eight keys): `SUMMON`, `TREMBLE`,
`OBJECT`, `SCREEN`, `SOUND`, `BOSS_HP`, `BACKGROUND_MUSIC`, `REWARD_RULLET`. On
`template_gms_12_1` and `template_gms_92_1`, none of these eight sub-effects can be announced
today — not just the boss HP gauge, but background music cues, screen/sound effects, field
tremble, and reward roulette prompts. This follow-up is scoped to the family as a whole, not
narrowed to `BOSS_HP`.

## Why this is out of scope for task-297

Adding a `FieldEffect` writer (or, for `gms_92`, promoting the existing
`FieldEffectWeather`-only routing) to these two templates is a per-client-version protocol
bring-up: it requires deriving the packet's IDB fname and switch-case mode table for `gms_v12`
and `gms_v92` from the client binaries for those versions, then running `packet-audit`
verification against that derivation before the writer can be trusted. `field_effect.yaml`'s
existing comment ("Version-STABLE: switch cases 0..7 byte-identical in v83 ... and v95,
IDA-verified") shows that stability has only been confirmed for the versions already listed —
it cannot be assumed for `gms_v12` or `gms_v92` without doing the same IDA work. That
derivation is a distinct, self-contained unit of work and does not belong inside a boss-HP
feature task.

## How the feature degrades on these two tenants today (NFR-3)

Per NFR-3, the boss HP announcement path fails closed, not open. `session.Announce`
(`services/atlas-channel/atlas.com/channel/session/processor.go:265-270`) resolves the named
writer via `writerProducer(writerName)`:

```go
w, err := writerProducer(writerName)
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
    return err
}
```

On a tenant whose template has no `FieldEffect` writer registered, `writerProducer("FieldEffect")`
returns an error. That error is recorded on the span and returned to the caller, where it is
logged (per the boss-HP announcement call site's existing error-logging convention) and
nothing else in the session is affected — no panic, no disconnect, no interruption to any
other packet on the connection. On a `gms_12` or `gms_92` tenant, a boss encounter proceeds
normally except that the top-of-screen boss HP gauge (and the other seven `FIELD_EFFECT`
sub-effects) never appears; the mob's existing over-head HP bar is untouched, since it does
not go through this writer.
