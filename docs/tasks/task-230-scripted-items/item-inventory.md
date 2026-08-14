# task-230 — the `0243` scripted-item inventory

Discharges PRD FR-6.2. Every value below is read from `Item.wz/Consume/0243.img.xml` in a local
v83-era extracted WZ tree; display names from `String.wz/Consume.img.xml` in the same tree. Nothing
here is asserted from memory or from general MapleStory knowledge.

**Count: 23 items.** The PRD's seeding research note claimed "24 items … e.g. 2430000-2430005";
the count was wrong by one, though those six ids do exist. Ids are not contiguous — the range
jumps from `2430016` to `2430026`.

## Where the fields actually live

All three fields are authored under the item's **`spec`** node, not `info`:

```
<imgdir name="02430010">
  <imgdir name="info">
    <int name="tradeBlock" value="1"/>
    <int name="slotMax" value="100"/>
  <imgdir name="spec">
    <string name="script" value="openTreasure"/>
    <int name="npc" value="2040030"/>
    <int name="runOnPickup" value="1"/>
```

**Zero of the 23 items carry `info/npc`.** `services/atlas-data/atlas.com/data/consumable/reader.go:75`
reads `npc` from the `info` node, so `Npc` resolves to `0` for every item in this table today.
`runOnPickup` (`reader.go:76`) has the identical defect. See design §3.1 — this is the same defect
class as the already-fixed `consumeOnPickup` (`reader.go:151-153`).

For contrast, the `0239` family — the other half of this task — genuinely authors `npc` under
`info` (verified on `02390001`, npc `9090002`), so the existing read is correct there and the fix
must be spec-first-with-info-fallback, not a move.

## The items

| Item id | Name | `spec/script` | `spec/npc` | `spec/runOnPickup` |
|---|---|---|---|---|
| `2430000` | Torn Cygnus' Book Volume 1 | `firstSignus` | `9000046` | — |
| `2430001` | Torn Cygnus' Book Volume 2 | `seSignus` | `9000046` | — |
| `2430002` | Torn Cygnus' Book Volume 3 | `lastSignus` | `9000046` | — |
| `2430003` | Cygnus Quiz | `signus` | `9000046` | — |
| `2430004` | Richie Gold's Random Key Pot | `KeySettingByItem` | `2084000` | — |
| `2430005` | Memorial Map | `checkMiroByItem` | `2084000` | — |
| `2430006` | Mysterious Piece of Paper | `q7222_Start` | `1063014` | — |
| `2430007` | Empty Compass | `compassMake` | `2084002` | — |
| `2430008` | Golden Compass | `compassUse` | `2084002` | — |
| `2430009` | Pure Perfume | `perfumeUse` | `2040030` | — |
| `2430010` | Mysterious Artifact | `openTreasure` | `2040030` | **yes** |
| `2430011` | Agent Summon | `summonEventNpc` | `9010000` | — |
| `2430012` | Agent Removal | `vanishEventNpc` | `9010000` | — |
| `2430013` | Peng Peng Popsicle | `item_2430013` | `9010000` | — |
| `2430014` | Killer Mushroom Spore | `killarmush` | `1300010` | — |
| `2430015` | Thorn Remover | `removethorns` | `1300011` | — |
| `2430016` | Crystal Chest | `icebox` | `9010010` | — |
| `2430026` | Mystery Box  | `yellowDayBox1` | `9010010` | — |
| `2430027` | Mystery Box  | `yellowDayBox2` | `9010010` | — |
| `2430028` | Mystery Box  | `yellowDayBox3` | `9010010` | — |
| `2430029` | Mystery Box  | `yellowDayBox4` | `9010010` | — |
| `2430030` | Golden Compass | `compassUse_cash` | `2084002` | — |
| `2430031` | Instant Camera | `snapShot` | `9000070` | — |
| `2430032` | Black Bag | `blackBag` | `1013203` | — |

Distinct NPC avatars in the family: `9000046`, `2084000`, `1063014`, `2084002`, `2040030`,
`9010000`, `1300010`, `1300011`, `9010010`, `9000070`, `1013203`.

## Notes

- **`2430010` is the only `runOnPickup` item.** That is a *pickup* trigger, not a use trigger, and
  is out of scope for this task. Design §8 deliberately does not seed it, so the distinction stays
  observable now that §3.1 makes the flag visible for the first time.
- **Script bodies are largely unavailable.** Only `killarmush` (`2430014`) and `removethorns`
  (`2430015`) exist in the local Cosmic tree. Both are map-gated, quest-touching, and consume
  themselves via `im.removeAll(...)` — confirming PRD §4.5's claim that Cosmic scripts do their own
  consumption. **No claim is made about the other 21 scripts' original behaviour.**
- **`2430030` duplicates `2430008`'s name and avatar** ("Golden Compass", npc `2084002`) with script
  `compassUse_cash` — a cash-shop variant of the same item. Not a transcription error.
- **`2430026`–`2430029`** share the display name "Mystery Box" (trailing space present in the WZ
  string) across four distinct scripts `yellowDayBox1`–`4`.

## Out of family: item `3994225`

`gms_v95` alone additionally whitelists `nItemID == 3994225` in
`CWvsContext::SendScriptRunItemRequest` (`0x9de7a0`). It is **not** a `0243` item:

| Field | Value |
|---|---|
| Name | Evolving Ring Upgrade Potion |
| WZ location | `Item.wz/Install/0399.img.xml` (Setup/Install, `3xxxxxx`) |
| `spec/script` | `consume_3994255` |
| `spec/npc` | `9000021` |

Excluded from this task by design decision D-3. `services/atlas-data/atlas.com/data/setup/reader.go`
parses **no** `spec` node at all and exposes neither `script` nor `npc`, so supporting it means
extending that reader plus a second inventory type on the destroy step. Design §3.3 specifies the
required rejection behaviour so the gap is loud rather than silent.
