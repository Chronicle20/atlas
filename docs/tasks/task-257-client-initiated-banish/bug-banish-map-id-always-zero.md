# bug: banish never happens — `atlas-data` always reports `banish.map_id = 0`

**Reproduced:** tenant `03a434e5-a78f-4ae2-8ce7-258e53a017e9`, GMS 83.1, namespace
`atlas-pr-1451`, character `1` ("Atlas") in map `240000001`. 12× Shade (`5090000`)
summoned. Shade's only mob skill is `{id: 129, level: 5}` (banish). The mob visibly
casts; the character is never banished.

**Observed:**

1. `atlas-data` returns a banish record whose map id is zero and whose portal is the
   hard-coded default, while the message (a direct child node) resolves fine:

   ```
   GET /api/data/monsters/5090000   (tenant 03a434e5…, GMS 83.1)
   …"skills":[{"id":129,"level":5}],
     "banish":{"message":"As the Pentacle swayed in the shadows, you were brought to
     the Subway Ticketing Booth.","map_id":0,"portal_name":"sp"},…
   ```

   `map_id: 0` and `portal_name: "sp"` are exactly the defaults passed at
   `services/atlas-data/atlas.com/data/monster/reader.go:126-127`. The message
   proves the `info/ban` node exists — only its `banMap` subtree fails to resolve.

2. `atlas-monsters` logged **no** banish activity in 4h of live testing. Its only
   skill-related output is the picker, firing continuously:

   ```
   "Picker: monster [1000004] transition sentinel→casting(129) reason=sweep."
   ```

3. `atlas-channel` logged **no** `[MobBanishPlayer] read` line in the same window.
   Handler distribution over 4h (`--since=4h`, 4184 lines):
   `MonsterMovementHandle` 1596, `CharacterDamageHandle` 93,
   `CharacterExpressionHandle` 20, `CharacterMoveHandle` 10,
   `CharacterHealOverTimeHandle` 2. No unhandled/unregistered-opcode lines at all.

**Expected:** banish warps the character to the WZ banish map (Shade's message names
the Subway Ticketing Booth) at the WZ banish portal, then sends the WZ banish
message — prd.md §4.3 / design.md §7.3, implemented as
`ProcessorImpl.banishCharacter` (`services/atlas-monsters/.../monster/processor.go:1280`).

**Root cause (established, file:line):**
`services/atlas-data/atlas.com/data/monster/reader.go:126-127` calls

```go
mapId := uint32(b.GetIntegerWithDefault("banMap/0/field", 0))
portal := b.GetString("banMap/0/portal", "sp")
```

but `xml.Node.GetIntegerWithDefault` and `xml.Node.GetString`
(`services/atlas-data/atlas.com/data/xml/model.go:73` and `:82`) do **not** interpret
slash-separated paths. Each scans only the node's *direct* `<int>`/`<string>`
children for an exact `name` attribute match — `banMap` is an `<imgdir>` child, so
neither lookup can ever match and both always return the default. Consequently
`banish.map_id` is `0` for **every** monster in **every** tenant, and
`banish.portal_name` is always the literal `"sp"`.

With `map_id == 0`, both banish paths in `atlas-monsters` fail closed by design:

- `ProcessorImpl.Banish` (`.../monster/processor.go:1298`) returns
  `"template [%d] has no banish map; not banishing character…"`.
- `ProcessorImpl.executeBanish` (`.../monster/processor.go:1328`) logs
  `"Monster [%d] has no banish map configured."` and returns.

So the feature could not have worked for any mob on either path, independent of
anything task-257 changed. There is no test covering a populated `ban` node —
`services/atlas-data/atlas.com/data/monster/reader_test.go:1266` only asserts the
*absent*-`ban` case (`banish{"", 0, ""}`), which is why this survived.

`reader.go:111`'s `root.GetPoint("stand/0/origin", 0, 0)` (the `fixed_stance`
heuristic) is broken the same way — `GetPoint`
(`services/atlas-data/atlas.com/data/xml/model.go:147`) is also flat. These three
are the only slash-path call sites in `atlas-data`. Fixing `fixed_stance` is **out of
scope** for this bug; do not change its behavior.

## WZ structure (confirmed)

Verified against the serialized GMS 83.1 `Mob.wz` dump (1564 `*.img.xml` images).
Shade, `5090000.img.xml:30-38`, verbatim:

```xml
<imgdir name="ban">
  <string name="banMsg" value="As the Pentacle swayed in the shadows, you were brought to the Subway Ticketing Booth."></string>
  <imgdir name="banMap">
    <imgdir name="0">
      <int name="field" value="103000100"></int>
      <string name="portal" value="sp"></string>
    </imgdir>
  </imgdir>
</imgdir>
```

Full sweep of all 1564 images — **26** mobs have an `info/ban` node:

| Property | Result |
|---|---|
| have `banMsg` | 26 / 26 |
| have `banMap/0/field` | 26 / 26 |
| `banMap` index set | always exactly `{0}` — never multi-entry |
| portal child spelled `portal` | 19 |
| portal child spelled `potal` (WZ typo) | 3 — `9500194`, `9500303`, `9500304`, all value `out00` |
| no portal child at all | 4 — `9300139`, `9300140`, `9300151`, `9300152` (the `"sp"` default is correct for these) |

So `reader.go:126-127`'s path strings are right, the `<int>`/`<string>` element
types are right, and a single `banMap/0` entry is the only shape that occurs.

## Fix

- `services/atlas-data/atlas.com/data/xml/model.go` — add slash-path traversal so a
  `name` containing `/` walks `ChildNodes` for each leading segment and resolves the
  final segment against the leaf node's typed children. Preferred shape: one
  unexported `resolve(path string) (*Node, string)` helper used by `GetString`,
  `GetIntegerWithDefault`, `GetShort`, `GetBool`, `GetFloatWithDefault`, `GetDouble`
  and `GetPoint`. Names without `/` must keep today's exact behavior (byte-identical
  results for every existing call site); an unresolvable path returns the default.
  Note `ChildByName` (`:20`) returns `&c` over the range variable — leave its
  semantics alone, but do not copy that pattern.
- `services/atlas-data/atlas.com/data/xml/model_test.go` — tests that must fail
  before and pass after: nested `<imgdir name="banMap"><imgdir name="0"><int
  name="field" value="103000100"/><string name="portal" value="sp"/></imgdir></imgdir>`
  resolves via `GetIntegerWithDefault("banMap/0/field", 0)` and
  `GetString("banMap/0/portal", "")`; a missing intermediate segment returns the
  default; a flat name still resolves.
- `services/atlas-data/atlas.com/data/monster/reader_test.go` — add a monster
  fixture **with** a populated `info/ban` subtree and assert
  `banish{Message, MapId, PortalName}` are all three populated. Keep the existing
  absent-`ban` case at `:1266` unchanged.
- `services/atlas-channel/atlas.com/channel/socket/handler/mob_banish_player.go:22` —
  the `Banish` error is discarded with `_ =`. design.md's contract is "the caller
  logs once"; every rejection is currently silent, which is why the live run left no
  diagnostic trail. Log it at warn with the character id and template id.

## Not yet answered

- ~~Does the WZ actually name the node `banMap/0/field`?~~ **ANSWERED — see
  "WZ structure (confirmed)" below. The paths in `reader.go:126-127` are correct;
  only the flat getters are broken. No further WZ derivation is needed.**
- **Why the client never sent `MOB_BANISH_PLAYER` (opcode `0x38`) during the test.**
  Ruled out: the opcode is correct for GMS v83
  (`docs/packets/MapleStory Ops - ServerBound.csv:100`, v83 = 56 / `0x038`); it is
  present in the *live* tenant config (`GET /api/configurations/tenants` →
  `{"opCode":"0x38","validator":"LoggedInValidator","handler":"MobBanishPlayer",
  "fname":"CUserLocal::SendBanMapByMobRequest","services":["channel"]}`); and it is
  registered in `services/atlas-channel/atlas.com/channel/main.go:942`. So the
  routing is intact and the client simply did not send the packet. Determining the
  client-side trigger for `CUserLocal::SendBanMapByMobRequest` needs IDA, which was
  not available in the diagnosing session. **Not part of this fix** — do not
  speculate about it in code or comments.
- Related observation, also **not** part of this fix: the mob-move packets carry
  `skillData [117966209]` (`0x07080501` → `SkillId()` = 1, `SkillLevel()` = 5) on
  `nActionAndDir [42]`/`[43]`, i.e. `atlas-channel` decodes the client's skill echo
  as skill **1**, not the 129 it served into `MoveMonsterAck`. That belongs to the
  mob-skill/auto-aggro work (PR #1460), not task-257. Recorded here so it is not
  rediscovered.

## Follow-up: `potal` typo — IN SCOPE (operator ruling, supersedes context.md)

context.md recorded the `potal` WZ typo as "out of scope by decision" for task-257.
The operator has since ruled the opposite: **handling the inconsistency is in
scope.** This section is the authority; context.md's note is superseded.

Evidence (GMS 83.1 `Mob.wz`, all 1564 images swept — see the table above): of the
26 mobs with an `info/ban` node, 3 spell the portal child `potal` and carry a real
value — `9500194` → `out00`, `9500303` → `out00`, `9500304` → `out00`. Today
those three fall through to the `"sp"` default and land the player on the wrong
portal. The 4 mobs with no portal child at all (`9300139`, `9300140`, `9300151`,
`9300152`) are correct on the `"sp"` default and must stay that way.

### Fix

- `services/atlas-data/atlas.com/data/monster/reader.go:127` — resolve the portal
  as `banMap/0/portal`, falling back to `banMap/0/potal`, and only then to `"sp"`.
  Keep `"sp"` as the final default. Do NOT make the fallback generic in
  `xml.Node` — this is a WZ data typo local to the mob `ban` node, and burying it
  in the shared getter would apply it to unrelated lookups. Comment the fallback
  with the three affected template ids so it is not later "cleaned up".
- `services/atlas-data/atlas.com/data/monster/reader_test.go` — three cases:
  `portal` present wins; only `potal` present resolves to its value; neither
  present yields `"sp"`. The `portal`-wins case must assert that a node carrying
  BOTH spellings prefers `portal`.

### Required sweep before implementing

The table above covers **GMS 83.1 only** — that is the only version serialized
locally (`tmp/*/GMS/83.1/`). Sweep the `info/ban` subtree of `Mob.wz` for the
other scopes in MinIO (`atlas-wz/shared/regions/GMS/versions/{48,61,72,79,84,87,
92,95}.1` and `JMS`) and record the per-version counts here in the same table
shape. If any scope shows a third spelling, a multi-entry `banMap`, or a `field`
stored as `<string>` rather than `<int>`, STOP and report rather than widening the
fallback on your own judgment.

**Sweep result (done):** downloaded each scope's raw `Mob.wz` from MinIO
(`atlas-wz/shared/regions/GMS/versions/{48,61,72,79,83,84,87,92,95}.1/Mob.wz`
and `atlas-wz/shared/regions/JMS/versions/185.1/Mob.wz` — 185.1 is the only
JMS version present) via `kubectl -n minio port-forward svc/minio 9000:9000`
and `mc cp`, then walked each with a scratch tool built on `libs/atlas-wz`
(same parse path as `services/atlas-data/atlas.com/data/data/wztoxml`, not
committed) that inspects every image's `info/ban` subtree directly
in-memory. The GMS 83.1 run was validated first against the table above
(26/26/0/19/3/4, byte-identical) before trusting the tool for the other
scopes.

| Scope | `ban` nodes | `banMap/0/field` | multi-index | `portal` | `potal` | neither | `field` as string | 3rd spelling |
|---|---|---|---|---|---|---|---|---|
| GMS 48.1 | 0 | 0 | 0 | 0 | 0 | 0 | 0 | 0 |
| GMS 61.1 | 22 | 22 | 0 | 15 | 3 | 4 | 0 | 0 |
| GMS 72.1 | 25 | 25 | 0 | 18 | 3 | 4 | 0 | 0 |
| GMS 79.1 | 26 | 26 | 0 | 19 | 3 | 4 | 0 | 0 |
| GMS 83.1 | 26 | 26 | 0 | 19 | 3 | 4 | 0 | 0 |
| GMS 84.1 | 26 | 26 | 0 | 19 | 3 | 4 | 0 | 0 |
| GMS 87.1 | 26 | 26 | 0 | 19 | 3 | 4 | 0 | 0 |
| GMS 92.1 | 30 | 30 | 0 | 23 | 3 | 4 | 0 | 0 |
| GMS 95.1 | 21 | 21 | 0 | 14 | 3 | 4 | 0 | 0 |
| JMS 185.1 | 26 | 26 | 0 | 19 | 3 | 4 | 0 | 0 |

GMS 48.1 shows zero `ban` nodes because none of the 26 templates that ever
carry one (including the three `potal` mobs) exist yet at that version — the
file has only 687 images total vs. 1564 at 83.1; confirmed by name-searching
the parsed image list for `5090000`/`9500194`/`9300139` and finding none.
Every scope that does have the affected mobs shows the same three `potal`
ids (`9500194`, `9500303`, `9500304`, all value `out00`) and the same four
no-portal-child ids (`9300139`, `9300140`, `9300151`, `9300152`); no scope
introduced a multi-entry `banMap`, a `field` stored as `<string>`, or a
third portal-child spelling. The `portal`-first, `potal`-fallback, `"sp"`-
default fix is safe as scoped.

## Resolution

_(to be filled in: fixing commit, gate verdict, live re-test outcome)_
