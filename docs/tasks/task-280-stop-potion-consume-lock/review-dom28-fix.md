# Review: fix-dom28-brief (DOM-28 fix round)

**Range reviewed:** `6e44e93b9..7f308a97b` (1 commit, `7f308a97b`)
**Brief:** `.superpowers/sdd/plan/fix-dom28-brief.md`

## Scope confirmed

`git diff --stat 6e44e93b9..7f308a97b` shows exactly two files touched:

- `services/atlas-consumables/atlas.com/consumables/consumable/processor.go` (+1)
- `services/atlas-consumables/atlas.com/consumables/consumable/processor_potion_lock_test.go` (+14/-1)

This matches the brief's "Files" section exactly. No files under `character/buff/` or
`vega.go` appear in the diff.

## Findings

### 1. `degrade.Observe` call — signature and label convention

`libs/atlas-rest/degrade/degrade.go:23`:

```go
func Observe(l logrus.FieldLogger, component string, entityId uint32, err error)
```

New call at `processor.go:203`:

```go
degrade.Observe(l, "consumable.potion-lock.buffs", characterId, err)
```

- `l` is the `logrus.FieldLogger` parameter of `resolvePotionLocked` (function takes `l`
  as a parameter, not `p.l`, per its doc comment at processor.go:191-195 — matches
  the existing pattern where the function is deliberately parameterized for
  testability). Correct arg type.
- `characterId` is `uint32` per `resolvePotionLocked(l logrus.FieldLogger, bp buff.Processor, characterId uint32) bool` (processor.go:200) — matches `entityId uint32`.
- `err` is the error returned by `bp.GetByCharacterId(characterId)` in the same branch — correct.
- Label `"consumable.potion-lock.buffs"` follows the same `"consumable.<domain>.<field>"`
  shape as the two existing call sites, `"consumable.reward.name"` (processor.go:1382)
  and `"consumable.reward.item-string"` (processor.go:1387). PASS.

Verified by direct read of `libs/atlas-rest/degrade/degrade.go`, not assumed from the
two call sites alone (per brief instruction).

### 2. Scope — resolveZombified and character/buff/ untouched

- `resolveZombified` (processor.go:182-188) is unchanged in this diff: still a bare
  `Warnf` with no `degrade.Observe` call. Confirmed via `git diff` showing no hunk
  touching lines 182-188, and via direct read of the current file.
- `git diff 6e44e93b9..7f308a97b -- 'services/*character/buff/*'` returns empty —
  no files under `character/buff/` are touched.
- No `vega.go` in the diff stat.

PASS — scope matches the brief exactly.

### 3. Test change — real assertion, no weakening

`processor_potion_lock_test.go`, `TestRequestItemConsume_BuffReadErrorFailsOpen`:

- The pre-existing `warnEntries` filter/assertion (`len == 1`, contains `"555"`) is
  preserved unchanged — not weakened.
- New `degradeEntries` filter matches log lines containing `"Enrichment degraded"`,
  which is unique to `degrade.Observe`'s `Warnf` format string
  (`"Enrichment degraded for component [%s], entity [%d]; returning un-enriched model."`,
  degrade.go:26). Absent the fix, this slice would be empty and
  `assert.Len(t, degradeEntries, 1)` would fail — this is a real regression guard, not
  a tautology.
- Second assertion (`assert.Contains(t, degradeEntries[0], "consumable.potion-lock.buffs")`)
  pins the specific label, catching a copy-paste of the wrong component string.

Ran the two directly relevant tests locally (not `tools/verify.sh`, per instructions):

```
go test ./consumable/... -run 'TestRequestItemConsume_BuffReadErrorFailsOpen|TestResolvePotionLocked' -v
```

Both pass; log output confirms both the Warn line and the `degrade.Observe` Warn line
fire with the expected component/entity values (`entity [555]`,
`component [consumable.potion-lock.buffs]`). `go build ./consumable/...` also clean.

PASS.

## Not evaluable

- Prometheus counter increment itself (`atlas_enrichment_degraded_total`) is not
  asserted by the test (only its side-effect log line is) — the brief explicitly
  permitted this ("extend... ONLY IF... cheaply observable... If not, do NOT invent
  a test harness"). The log-line proxy is a reasonable, low-cost choice consistent
  with the brief's guidance; counting this as satisfying brief intent, not a gap.
- Did not run `tools/verify.sh` or `tools/lint.sh` (explicitly out of scope for this
  review per instructions — owned by a concurrent agent).

## Verdict

All three specific check items pass with cited evidence. Scope is exactly the two
files named in the brief. No blocking findings.
