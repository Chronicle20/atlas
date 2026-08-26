# Review: bug-banish-map-id-always-zero fix

**Commit range:** `12dba5cef..1c42f52e0` (single commit `1c42f52e0`, "fix(atlas-data): resolve
slash-separated paths in xml.Node getters")
**Brief:** `docs/tasks/task-257-client-initiated-banish/bug-banish-map-id-always-zero.md`

## Scope confirmed

Diff touches exactly the four files the brief's Fix section calls for:

- `services/atlas-data/atlas.com/data/xml/model.go` (+80/-22)
- `services/atlas-data/atlas.com/data/xml/model_test.go` (+41)
- `services/atlas-data/atlas.com/data/monster/reader_test.go` (+37)
- `services/atlas-channel/atlas.com/channel/socket/handler/mob_banish_player.go` (+3/-1)

No unrelated files touched. `reader.go` itself is unchanged (the brief's fix targets the
shared primitive, not the call site), and `ChildByName` (`model.go:21`) is untouched as
instructed. Scope matches the brief.

## Findings

### 1. `resolve` — flat-name behavior is byte-identical (PASS)

`model.go:37-57`:

```go
func (n *Node) resolve(path string) (*Node, string) {
	if !strings.Contains(path, "/") {
		return n, path
	}
	...
}
```

Every one of `GetShort`, `GetBool`, `GetString`, `GetIntegerWithDefault`,
`GetFloatWithDefault`, `GetDouble`, `GetPoint` now opens with `node, leaf :=
n.resolve(name)` and, for the no-slash case, `node == n` and `leaf == name` exactly, so
the subsequent `for _, c := range node.XNodes { if c.Name == leaf {...} }` loops are
identical to the pre-fix loops over `n.XNodes`/`name`. Verified by reading every getter
in the post-fix file (`model.go:59-220`) side by side with `git diff`; each preserves its
original parse-error and not-found paths (`return def` unchanged in every branch).

`go build`/`go vet` on `atlas-data/xml` and `atlas-data/monster` are clean, and the full
`atlas-data` test suite (`go test ./...` from `services/atlas-data/atlas.com/data`) passes,
including every pre-existing reader test across item/map/npc/quest/skill/reactor/monster —
none of those call slash-containing names, so this is a real regression check, not just a
green build on the two new tests.

### 2. WZ sweep for names containing `/` (PASS — no collision found)

Swept the serialized GMS 83.1 dump (`<wz-dump-root>/GMS/83.1`) for any `name="..."`
attribute containing a literal `/`, restricted to the archives `atlas-data` readers
actually parse:

```
grep -rlE 'name="[^"]*/[^"]*"' Item.wz Map.wz Mob.wz Npc.wz Quest.wz Reactor.wz Skill.wz \
  TamingMob.wz Etc.wz Character.wz Effect.wz Sound.wz String.wz UI.wz Morph.wz Base.wz
```
→ zero hits in Item/Map/Mob/Npc/Quest/Reactor/Skill/TamingMob/Etc/Character/Effect/Sound/
UI/Morph/Base.wz. One hit in `String.wz/Consume.img.xml`, but it is a `<null name="[...]">`
key (item-description text used as a dictionary key, e.g. `name="[Restores all HP/MP,
...]"`) — `<null>` has no corresponding field in `xml.Node` (only `int`/`string`/`vector`/
`double`/`canvas`/`imgdir` are unmarshaled), so it can never enter `IntegerNodes`/
`StringNodes`/etc. and can never be looked up by any getter regardless of this change.
Confirmed no call site in `data/workers/stringw.go` (the only consumer of
`String.wz/Consume.img.xml`) passes that literal string to any getter. There is no WZ node
name in the readers' scope that would be silently misinterpreted as a path by this fix.

Also confirmed (grep across `services/atlas-data/atlas.com/data`, excluding `_test.go`,
for any `Get*("...slash...")` call): the only three slash-path call sites in the whole
service are the two `banMap/...` calls and the one `stand/0/origin` call in
`monster/reader.go:111,126,127`, matching the brief's claim exactly.

### 3. `fixed_stance` — brief flags a possible regression; disproven against real WZ data (finding, non-blocking)

The brief (and this review's own instructions) called out that `getFixedStance`
(`reader.go:105-112`) would now "actually resolve" `stand/0/origin` and could return `4`
instead of the always-`5` it returned before. I checked this against the full Mob.wz
dump rather than reasoning abstractly, because `resolve` only walks `ChildNodes`
(`xml:"imgdir"`), and `stand`'s indexed children in every WZ mob image found are typed
as something else:

```python
# swept every noFlip>0 monster (146 files) in Mob.wz for the shape of stand's child "0"
total noFlip>0: 146
canvas: 139   nostand: 6   other: 1
```

- 139/146 have `<canvas name="0">...<vector name="origin" .../>...</canvas>` under
  `stand` — a `<canvas>` element is parsed into `Node.CanvasNodes`, never
  `Node.ChildNodes`, so `resolve`'s second-segment walk (looking for child `"0"` in
  `stand.ChildNodes`) always fails → `resolve` returns `(nil, "")` → `GetPoint` returns
  the default `(0,0)` → `x == 0` → `getFixedStance` still returns `5`. Verified directly:
  `Mob.wz/4220001.img.xml:35-41` (a `noFlip=1` monster) has `<imgdir name="stand">
  <canvas name="0" ...><vector name="origin" x="66" y="76">`.
- 6/146 have no `stand` node at all → `resolve` fails at the first segment.
- 1/146 (`9300018.img.xml`) has `<uol name="0" value="../move/0">` under `stand` — a
  `<uol>` (WZ symlink) element, also not unmarshaled into `ChildNodes`.

So across every monster in the dump that has `noFlip > 0`, `stand/0` is never an
`<imgdir>`, meaning the two-segment path can never resolve, meaning `GetPoint` always
falls through to its default `(0, 0)` both before and after this commit. **The stated
regression does not materialize against current data** — `fixed_stance` returns exactly
the same value (`5`, for every `noFlip>0` monster) before and after this fix. This is not
a defect in the change; it is worth recording because it means the "out of scope, do not
change fixed_stance behavior" instruction was honored by accident of WZ shape rather than
by the code explicitly guarding against it — if some future WZ revision ever nests `stand`
under an actual `<imgdir name="0">` with a `<vector name="origin">` directly (not via
`<canvas>`), `fixed_stance` would silently start returning `4` for that monster, with no
test pinning the current `5` behavior end-to-end via `getFixedStance` (only the getter's
own behavior is tested, not `reader.go`'s consumption of it for `noFlip>0` monsters).
Non-blocking: no such WZ shape exists in the confirmed dump, and the brief explicitly
scoped `fixed_stance` fixing out.

### 4. New tests are honest (PASS)

Verified directly rather than by inspection: checked out the parent commit
(`12dba5cef`) into a scratch worktree, applied only the new test diffs
(`model_test.go` + `reader_test.go`) on top of the pre-fix `model.go`/`reader.go`, and ran
them:

```
=== RUN   TestSlashPathResolution
    model_test.go:298: GetIntegerWithDefault("banMap/0/field") = 0, want 103000100
    model_test.go:303: GetString("banMap/0/portal") = "", want "sp"
--- FAIL: TestSlashPathResolution (0.00s)
=== RUN   TestReaderBanishPopulated
    reader_test.go:1576: Banish={... MapId:0 PortalName:sp}, want {... MapId:103000100 PortalName:sp}
--- FAIL: TestReaderBanishPopulated (0.00s)
```

Both fail pre-fix and pass post-fix (`go test -run 'TestSlashPathResolution|
TestReaderBanishPopulated' -v` on `1c42f52e0` → both PASS). These are genuine
regression tests, not tests that pass either way.

`reader_test.go`'s new `TestReaderBanishPopulated` fixture uses the real WZ shape:
`info/ban/banMsg` (string), `info/ban/banMap/0/field` (int), `info/ban/banMap/0/portal`
(string) — byte-for-byte the structure the brief captured from `Mob.wz/5090000.img.xml:
30-38` for Shade, down to the exact `banMsg` text. The pre-existing absent-`ban` case at
`reader_test.go:1266` (`TestReaderBanishAbsentDefaults` et al.) is untouched by the diff,
per the brief's instruction to keep it unchanged — confirmed by `git diff --stat` showing
only additive `+37` lines with no deletions in that file.

### 5. `mob_banish_player.go` — error now logged (PASS)

`mob_banish_player.go:20-22`, post-fix:

```go
if err := monster.NewProcessor(l, ctx).Banish(s.Field(), s.CharacterId(), p.MobTemplateId()); err != nil {
	l.WithError(err).Warnf("Character [%d] not banished by template [%d].", s.CharacterId(), p.MobTemplateId())
}
```

Matches the brief's requirement exactly: warn level, `err` attached via `WithError`,
message includes character id and template id. `go build`/`go vet` clean on
`atlas-channel`; `go test ./socket/... ./monster/...` passes. There is no dedicated unit
test for this handler before or after this commit (handler tests in this package
generally exercise the decode/dispatch path, not the processor-error branch) — this is a
pre-existing gap, not one introduced by this change, so it is not a blocking finding, but
it is not evaluable as "does a test pin the new warn-log contract."

## Not evaluable

- Whether the new `l.WithError(err).Warnf(...)` line is actually asserted by any test —
  no test exercises the error branch of `MobBanishPlayerHandleFunc` (before or after this
  commit). Cannot confirm the log format is pinned against regression; can only confirm it
  compiles and matches the brief's prose requirement by inspection.
- Live re-test outcome (does a real client-triggered banish now warp the character) is
  outside this review's surface — the brief's own "Resolution" section is still marked
  TBD and the "why the client never sent `MOB_BANISH_PLAYER`" question is explicitly
  out of scope for this fix.

## Summary

The fix is narrowly scoped, byte-identical for every existing flat-name call site
(verified by full local test suite pass plus manual per-getter diff read), closes the
exact defect described in the brief (verified via before/after test run against a scratch
checkout of the parent commit), and does not introduce the `fixed_stance` regression the
review brief worried about (verified against a full sweep of the real WZ dump, not just
the two call sites the brief named). The one gap is a test-coverage gap that predates this
commit (no unit test for the handler's error-logging branch), not a defect in the diff
itself.
