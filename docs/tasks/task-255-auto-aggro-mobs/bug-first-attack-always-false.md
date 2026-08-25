# bug: `first_attack` is always `false` — every AUTO_AGGRO claim is denied

Task: task-255-auto-aggro-mobs
PR: atlas-pr-1460
Reported: 2026-08-25
Status: diagnosed, root cause established

## Reproduced

- Tenant `9442c436-25d8-4c15-9a83-ff36aea53ced`, region `GMS`, version `83.1`.
- Namespace `atlas-pr-1460`, pods `atlas-channel-857599d865-vrt54`,
  `atlas-monsters-765ff9c6f8-55hpm`.
- Character `1`, map `200040001`, mobs `4230105` (Nependeath) and `4130102`
  (Dark Nependeath).

## Observed

The client-side half of the loop works. Over the session the channel decoded
561 `AUTO_AGGRO` packets and admitted them across 15 distinct mob object ids:

```
[AutoAggro] read [mobId [1000025], distance [25]]
Requesting auto-aggro of monster [1000025] for character [1].
```

Exactly **one** `AGGRO_CHANGED` event was emitted in the whole session, and it
was not from auto-aggro — it followed a player hit:

```
17:38:51.064  Applying damage to monster [1000028]. Character [1]. Lines [1].
17:38:51.275  Message received {... "uniqueId":1000028, "type":"AGGRO_CHANGED",
                "body":{"controllerCharacterId":1,"controllerHasAggro":true}}
```

That is the pre-existing damage-driven aggro path. **Auto-aggro has granted
zero claims.** All 561 `SET_AGGRO` commands were dropped inside atlas-monsters.

The only "attack" the player experienced is body-contact damage
(`bodyAttack=1` on both templates) — `CharacterDamageHandle ... nAttackIdx [-1],
damage [0], monsterTemplate [4230105]` — which happens without aggro. Mob
movement stays `nActionAndDir [-1]` throughout: the client's hostile AI never
engages.

atlas-monsters runs at `LOG_LEVEL=info` in this namespace, so the per-gate
`Debugf` drops are invisible. Setting it to `debug` with `kubectl set env` was
reverted by the GitOps reconciler within seconds, so the gate was identified by
querying the data service directly instead.

## Root cause

`atlas-data` returns `first_attack: false` for **both** templates, so
`SetAggro` gate 3 (`!info.FirstAttack()` →
`services/atlas-monsters/atlas.com/monsters/monster/processor.go:1926`) denies
every claim.

Live REST, `GET /api/data/monsters/{id}` against
`atlas-ingress.atlas-pr-1460`:

```
4230105 Nependeath       ... "first_attack":false ...
4130102 Dark Nependeath  ... "first_attack":false ...
```

The WZ data says otherwise. `/mnt/d/Source/AtlasMS/wz/Mob.wz/4230105.img.xml`
and `4130102.img.xml`, inside `<imgdir name="info">`:

```xml
<int name="firstAttack" value="1"/>
```

The reader is wrong. `services/atlas-data/atlas.com/data/monster/reader.go:198`:

```go
func getFirstAttack(node *xml.Node) bool {
	c, err := node.ChildByName("firstAttack")
	if err != nil {
		return false
	}
	return math.Round(c.GetFloatWithDefault("firstAttack", 0)) > 0
}
```

`firstAttack` is a scalar `<int>` leaf **attribute** of the `info` node, not a
child `imgdir`. `ChildByName("firstAttack")` therefore always errors and the
function always returns `false` — for every monster in the game, not just these
two. Every sibling boolean in the same block reads the attribute directly:

```go
m.Boss            = node.GetIntegerWithDefault("boss", 0) > 0
m.ExplosiveReward = node.GetIntegerWithDefault("explosiveReward", 0) > 0
m.Undead          = node.GetIntegerWithDefault("undead", 0) > 0
m.RemoveOnMiss    = node.GetIntegerWithDefault("removeOnMiss", 0) > 0
```

The bug predates task-255 — the PRD's premise that `firstAttack` "is already
parsed correctly" is false — but it makes the task's feature a total no-op, so
it is in scope here.

The unit test enshrines the defect. `reader_test.go` line 42 puts
`<int name="firstAttack" value="1"/>` in the fixture's `info` block, and line
1258 asserts:

```go
if rm.FirstAttack != false {
	t.Errorf("FirstAttack mismatch: got %t, expected false", rm.FirstAttack)
}
```

So the test passes only because the reader is broken.

Downstream mapping is correct and needs no change:
`services/atlas-monsters/atlas.com/monsters/monster/information/rest.go:35,104`
carries `first_attack` into `Model.firstAttack` faithfully.

## Expected

`first_attack` is `true` for any template whose WZ `info/firstAttack` is
non-zero. With that, `SetAggro` gate 3 passes, the controller path stamps the
lease and emits `AGGRO_CHANGED`, and
`handleStatusEventAggroChanged`
(`services/atlas-channel/atlas.com/channel/kafka/consumer/monster/consumer.go:386`)
re-issues `MonsterControl` with `aggro = 1`, at which point the client's hostile
AI chases and attacks.

## Fix

- `services/atlas-data/atlas.com/data/monster/reader.go:198-204` — replace
  `getFirstAttack` with a direct attribute read matching the sibling booleans:
  `node.GetIntegerWithDefault("firstAttack", 0) > 0`. Prefer inlining at the
  call site (`reader.go:81`) so the shape matches `Boss`/`Undead`/`RemoveOnMiss`;
  delete the helper. Check whether the `math` import is still needed —
  `reader.go:52` uses `math.MaxInt32`, so it is; do not drop it.
- `services/atlas-data/atlas.com/data/monster/reader_test.go:1258-1260` — the
  fixture at line 42 carries `firstAttack value="1"`, so the assertion must
  become `!= true` / `expected true`. Sweep the rest of the file for any other
  `FirstAttack` assertion whose fixture disagrees.
- Add a reader test case for a fixture whose `info` has **no** `firstAttack`
  key, asserting `false`, so the default path stays covered.

No change is needed in atlas-monsters, atlas-channel, or libs/atlas-packet.

## Not yet answered

- Whether other Atlas services already read `first_attack` and silently depend
  on it being universally `false`. A grep at diagnosis time found only the
  atlas-data reader/DTO and the atlas-monsters information model, but this was
  not swept exhaustively across all 14 services.
- ~~Whether any other field in `reader.go` uses the same wrong
  `ChildByName(<scalar>)` idiom.~~ **Answered: yes.** Code review found
  `getCoolDamage` (`reader.go:239-247`) has the identical defect — it calls
  `node.ChildByName("coolDamage")`, but a corpus grep over
  `Mob.wz/*.img.xml` shows `coolDamage` and `coolDamageProb` are always scalar
  `<int>` leaves and `<imgdir name="coolDamage">` never occurs. `m.CoolDamage`
  is therefore `{0,0}` for every monster, before and after this fix.
  Pre-existing, unrelated to auto-aggro, NOT fixed here — needs its own bug file.
  `getSelfDestruction` and `getLoseItems` target genuine child `imgdir`s and
  are correct.
- Live re-test after the fix: not yet performed.

## Resolution

Fixed by commit `38d4b98b4` — `fix(atlas-data): read firstAttack as a scalar
attribute, not a child imgdir`.

- Code review (`review-bug-first-attack-always-false.md`): **APPROVED_WITH_FINDINGS**,
  0 blocking, 1 non-blocking. The reviewer reverted `reader.go` to the pre-fix
  helper in a scratch copy and confirmed the corrected assertion genuinely fails
  pre-fix (`reader_test.go:1259: FirstAttack mismatch: got false, expected true`)
  and passes post-fix. The non-blocking finding is the `getCoolDamage` defect,
  now tracked in `bug-cool-damage-always-zero.md` and fixed in `06159cb0b`.
- Module verification: `go build ./... && go test ./...` in
  `services/atlas-data/atlas.com/data` passes; re-run uncached by the controller.
- **Repo-wide gate: NOT green.** `tools/verify.sh --quick --base 11663808b`
  exited 1 on `lint.sh: LINT FAIL — libs/atlas-outbox`,
  `panic: file requires newer Go version go1.27 (application built with go1.26)`.
  This is pre-existing toolchain drift, not caused by this change: `atlas-outbox`
  is untouched by the commit, and the identical failure reproduces on `main` with
  none of these commits present (`tools/lint.sh --check --go libs/atlas-outbox`
  → exit 1, same panic). `go.work` declares `go 1.26.0` and no module declares
  1.27; the installed toolchain is `go1.27.0` and the pinned golangci-lint binary
  is built with Go 1.26. Needs golangci-lint re-pinned (or Go pinned back) before
  any change on this machine can gate green.
- **Live re-test: NOT yet performed.** Requires the PR image to rebuild with
  these commits. Confirmation signal: an `AGGRO_CHANGED` event in the
  atlas-channel log for a mob in map 200040001 that the character has NOT
  damaged. Body-contact damage alone does not distinguish the fix, because
  `bodyAttack=1` on both templates produces damage without aggro.
