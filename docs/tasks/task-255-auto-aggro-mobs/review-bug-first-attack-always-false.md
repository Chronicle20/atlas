# Review: bug-first-attack-always-false (commit 38d4b98b4)

Task: task-255-auto-aggro-mobs
Range reviewed: `11663808b..38d4b98b4` (single commit)
Brief: `docs/tasks/task-255-auto-aggro-mobs/bug-first-attack-always-false.md`
Report: `docs/tasks/task-255-auto-aggro-mobs/report-bug-first-attack-always-false.md`
Reviewer: atlas-reviewer (sonnet)

## Scope

`git diff --stat 11663808b..38d4b98b4`:

```
services/atlas-data/atlas.com/data/monster/reader.go      | 10 +--------
services/atlas-data/atlas.com/data/monster/reader_test.go | 25 ++++++++++++++++++++--
2 files changed, 24 insertions(+), 11 deletions(-)
```

Matches the report exactly — no unexplained files, no scope creep. `scope_confirmed`.

## 1. Is the scalar read correct?

Read `services/atlas-data/atlas.com/data/xml/model.go:20-27,82-102`:

```go
func (n *Node) ChildByName(name string) (*Node, error) {
	for _, c := range n.ChildNodes { ... }   // only nested <imgdir> elements
	return nil, errors.New("child not found")
}

func (n *Node) GetIntegerWithDefault(name string, def int32) int32 {
	for _, c := range n.IntegerNodes { ... }  // scans <int name=.../> leaves directly under n
	for _, c := range n.StringNodes { ... }
	return def
}
```

`Node.ChildNodes` is populated only from nested `<imgdir>` tags; `Node.IntegerNodes` is populated from `<int>` leaves directly under the node. `firstAttack` in WZ is `<int name="firstAttack" value="1"/>` under `info` — a leaf, never an `<imgdir>`. So `ChildByName("firstAttack")` was guaranteed to always error (confirming the bug diagnosis), and `GetIntegerWithDefault("firstAttack", 0) > 0` (`reader.go:81`) is the correct read — it is textually identical in shape to the sibling reads in the same block (`reader.go:66-69,79`: `Boss`, `ExplosiveReward`, `Undead`, `RemoveOnMiss`). PASS.

## 2. Test assertion inversion

- `reader_test.go:42` fixture: `<int name="firstAttack" value="1"/>` inside the `info` block — confirmed present, unchanged by this commit (only the assertion at line 1258-1259 was touched).
- Reverted `reader.go` to the pre-fix `getFirstAttack` helper (via a scratch copy) and re-ran `go test ./monster/... -run TestReader -v`: `TestReader` fails with `reader_test.go:1259: FirstAttack mismatch: got false, expected true` — i.e. the corrected assertion genuinely fails against the old code and passes only with the fix. This is a corrected assertion, not a test bent to match new behavior. Restored `reader.go` afterward; `git status --short` on the file is clean. PASS.
- New test `TestReaderFirstAttackAbsentDefaultsFalse` (`reader_test.go:1499-1517`) uses a fixture with no `firstAttack` key in `info` and asserts `false` — covers the default path the bug file explicitly asked for. PASS.
- `math` import: still used at `reader.go:52` (`math.MaxInt32`), confirmed via grep; the report's claim is correct.

## 3. Blast-radius / consumer sweep

Repeated the report's grep independently: `grep -rniE "first_attack|firstattack" --include="*.go" services/atlas-monsters services/atlas-channel services/atlas-data`.

Findings match the report:
- The only behavioral read is `services/atlas-monsters/atlas.com/monsters/monster/processor.go:1926` (`SetAggro` gate 3, inside `if !info.FirstAttack() { ...drop... }`), read at `services/atlas-monsters/atlas.com/monsters/monster/processor.go:1880-1935`.
- `set_aggro_test.go` mocks `information.Model` via `NewModelBuilder().SetFirstAttack(...)` (a test-only builder, not through the atlas-data reader), so those tests are unaffected by this fix either way — correctly identified as out of the blast radius.
- atlas-channel's only match is a comment (`auto_aggro_gate.go:17`), not a behavioral read.
- No other branch on `FirstAttack`/`first_attack` exists in atlas-monsters, atlas-channel, or atlas-data itself beyond the reader/DTO plumbing already covered. PASS — the sweep holds up under an independent re-run.

## 4. Same-idiom check across reader.go — a real, unresolved defect found

Enumerated every `ChildByName(<literal>)` call in `reader.go`: `info`, `ban`, `skill`, `revive`, `selfDestruction`, `loseItem`, `coolDamage`, `attack{1,2,3}`, `attack{N}/info`.

Checked each name's actual WZ shape against `/mnt/d/Source/AtlasMS/wz/Mob.wz/*.img.xml`:
- `ban`, `skill`, `revive`, `selfDestruction`, `loseItem`, `attackN`, `attackN/info` — every occurrence across the corpus is `<imgdir name="...">`, never a scalar `<int>`/`<string>` leaf. `skill` in particular is a genuine imgdir of indexed sub-blocks (`info/skill/0/skill=126`, `.../1/skill=200`, ...), confirmed via `Mob.wz/2220000.img.xml`. These `ChildByName` calls are correct.
- **`coolDamage` is not.** Grepped every `Mob.wz/*.img.xml` for `imgdir name="coolDamage"` — zero matches across the whole corpus (`grep -c 'imgdir name="coolDamage"' ... | grep -v ':0$'` → empty). `coolDamage` and `coolDamageProb` are always scalar `<int>` leaves directly under `info` (e.g. `Mob.wz/9700001.img.xml`: `<int name="coolDamage" value="200"/><int name="coolDamageProb" value="9"/>`). So `getCoolDamage` (`reader.go:239-247`):
  ```go
  func getCoolDamage(node *xml.Node) coolDamage {
  	c, err := node.ChildByName("coolDamage")
  	if err != nil {
  		return coolDamage{}
  	}
  	...
  }
  ```
  has exactly the same bug class as the old `getFirstAttack` — `ChildByName("coolDamage")` always errors, so `m.CoolDamage` is silently `{0, 0}` for every monster in the game, today, on `main`, and still after this commit (the commit did not touch `getCoolDamage`).

This is **not a defect in the commit under review** — `getCoolDamage` is untouched by this diff, and fixing it is outside the stated fix scope (`firstAttack` only). It directly answers the bug file's "Not yet answered" question 2, which the implementer's report does not address at all (the report only discusses the consumer sweep and the fixed field, not the sibling-helper check the bug file explicitly asked for). Flagging as a non-blocking finding: a follow-up bug ticket for `getCoolDamage` (and a fresh, from-scratch check of every other `ChildByName` name against the WZ corpus, since this review only spot-verified the seven other names present in `reader.go`) is warranted.

## Additional checks

- `go build ./...` and `go test ./monster/...` in `services/atlas-data/atlas.com/data`: clean, `ok atlas-data/monster 0.022s` including both new/changed tests running individually (`-run TestReader -v`).
- Downstream mapping (`services/atlas-monsters/atlas.com/monsters/monster/information/rest.go:35,104`) is unchanged by this commit and was already correct per the bug file — confirmed present, not touched.

## Not evaluable

- Live re-test in a running namespace (the bug file's original repro environment) was not performed by the implementer and is outside this reviewer's tooling — the report and bug file both mark it explicitly as not yet done. Counted as not evaluable, not silently approved.
- Exhaustive sweep of every `ChildByName` scalar-vs-imgdir mismatch across the whole `atlas-data` reader package beyond `monster/reader.go` (e.g. other domain readers) was not performed — out of this commit's diff surface.

## Verdict rationale

The fix is correct, minimal, and precisely targeted at the diagnosed bug. The test fix is honest (independently reproduced the pre-fix failure). The consumer sweep in the report holds up under re-verification. The one real finding — `getCoolDamage` sharing the same defect class, still live on `main` — is a pre-existing, out-of-scope bug that the commit does not introduce or worsen, so it does not block this fix, but it should be filed as a follow-up bug (the bug file's own second open question is otherwise left unanswered by the implementer's report).
