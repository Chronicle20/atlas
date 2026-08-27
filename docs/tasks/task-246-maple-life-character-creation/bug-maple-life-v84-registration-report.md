# Report: bug-maple-life-v84-registration

Scope was the first five bullets of the brief's `## Fix` section only, per
the controller's ruling. Evidence records under
`docs/packets/evidence/gms_v84/` and `docs/packets/audits/status.json` /
`STATUS.md` were explicitly excluded (packet-verifier work) and not touched.

## What I implemented

1. **`services/atlas-configurations/seed-data/templates/template_gms_84_1.json`**
   - `socket.handlers`: added `MapleLifeCheckNameHandle` at `0x107`
     (`LoggedInValidator`, `fname: CUICharacterSaleDlg::SendCheckDuplicateIDPacket`),
     inserted between `ItcOperationHandle` (`0x104`) and `ItemUpgradeUpdateHandle`
     (`0x10B`) — the same relative position the entry occupies in
     `template_gms_83_1.json` (between the ITC operation handler and the item
     upgrade handler).
   - `socket.writers`: added `MapleLifeResult` at `0x167` and `MapleLifeError`
     at `0x168`, with the exact `fname`s and `options.operations` tables from
     the brief, inserted between the `MtsOperation` writer and `ViciousHammer`
     (`0x16C`) — the same relative position as gms_83_1's `0x15D`/`0x15E`
     between `MtsOperation` (`0x15C`) and `ViciousHammer` (`0x162`).
   - `mapleLife` block: copied verbatim from `template_gms_83_1.json`.

2. **`mapleLife` block verification (ruling #3).** Before copying I compared
   gms_83_1's `mapleLife` block against gms_87_1's, gms_92_1's and gms_95_1's
   — all four templates that already carry the block. All four are
   byte-identical (`json.dumps(..., sort_keys=True)` equality check passed
   for 83 vs 87; spot-printed class/job/map data for 92/95 matched too — same
   5 jobs (Warrior 100/Magician 200/Bowman 300/Thief 400/Pirate 500), same
   maps, same equipment, same stats). gms_84 sits directly between 83 and 87
   in this identical run, so a straight copy is the right call, not an
   invention — nothing diverges. I also checked `characters.templates`
   jobIndex counts (84/87 both have 4 Beginner-variant templates, 92/95 have
   5) as a sanity check; that axis is unrelated to `mapleLife.classes.jobId`
   (Beginner-creation variants vs. actual playable jobs) and does not bear on
   the mapleLife block's correctness.

3. **`docs/packets/registry/gms_v84.yaml`** — added three rows, each
   `provenance: ida-discovered` with the `ida.address` decimal conversions of
   the brief's hex addresses:
   - `MAPLELIFE_CHECK_NAME`, serverbound, opcode 263, `ida.address: 8378474`
     (`0x7fd86a`), inserted between `ITC_OPERATION` (260) and
     `ITEM_UPGRADE_UPDATE` (267) — same relative position as the JSON
     handlers list.
   - `MAPLELIFE_RESULT`, clientbound, opcode 359, `ida.address: 8378697`
     (`0x7fd949`).
   - `MAPLELIFE_ERROR`, clientbound, opcode 360, `ida.address: 8378991`
     (`0x7fda6f`), both inserted between `MTS_OPERATION` (348) and
     `VICIOUS_HAMMER` (364) — the same contextual cluster the bug's
     "positive derivation" section ties them to (same `CField` forwarder
     family, `sub_544395`/`this[134]`).
   - Confirmed no opcode collisions: dumped the full registry with `pyyaml`
     and checked `(direction, opcode)` uniqueness after the edit — the only
     duplicate found is a pre-existing `serverbound 236` collision unrelated
     to this change (untouched).

4. **`docs/tasks/task-246-maple-life-character-creation/derivation.md`** —
   appended a `### §2.0-CORRECTION` subsection at the end of the file (after
   `§6.4`), per the file's "do not renumber" header rule. It records the
   retraction, walks through why each of §2.0's three findings was wrong
   (symbol-coverage artifact / void RTTI test / swapped `CField` symbols),
   gives the structural-fingerprint table and the dispatcher decompile, and
   states the final opcodes. §2.0 itself is untouched.

5. **`libs/atlas-packet/maplelife/**`** — checked for `MajorAtLeast`/version
   gates: `grep -rn "MajorAtLeast\|Major\b\|version\." libs/atlas-packet/maplelife/`
   found no gate logic (only doc comments saying "for every in-scope
   version" / "in-scope version"). Also checked the channel-service handlers
   that wire these codecs
   (`services/atlas-channel/.../socket/handler/maple_life_check_name.go`,
   `maple_life_create.go`) for `MajorAtLeast`/`Major(` — none found. The
   codecs are version-agnostic; **no change was needed**, matching the
   brief's fallback instruction.

## Incidental fix: corpus_test.go

`go test ./...` in `atlas-configurations` failed after the template/registry
edit: `socket/corpus_test.go`'s `TestValidate_AcceptsEverySeedTemplate`
asserts an exact total binding count (`3384`) with a long narrative comment
enumerating every task's additions. Adding 3 new bindings (1 handler + 2
writers) to gms_84 pushed the real count to 3387. I updated the expected
count to `3387` and appended a clause to the narrative describing this bug
fix's 3 gms_84 bindings, consistent with the file's existing style (each past
task's contribution recorded inline). This is the only test file this change
required touching; I did not touch anything else in that test file.

## Testing

Module-local, from `services/atlas-configurations/atlas.com/configurations`:

```
go build ./...
go test ./...
```

Both pass. Full test output tail (after the corpus_test.go fix):

```
ok  	atlas-configurations/socket	0.015s
ok  	atlas-configurations/templates	(cached)
...
ok  	atlas-configurations/tenants	(cached)
ok  	atlas-configurations/tenants/characters	(cached)
ok  	atlas-configurations/tenants/characters/preset	(cached)
ok  	atlas-configurations/tenants/maplelife	(cached)
ok  	atlas-configurations/tenants/socket	(cached)
```

No `FAIL` lines remain (before the fix, `--- FAIL: TestValidate_AcceptsEverySeedTemplate`
was the only failure).

JSON/YAML validity was checked directly:

```python
import json
d = json.load(open('services/atlas-configurations/seed-data/templates/template_gms_84_1.json'))
# 'mapleLife' in d.keys(); MapleLifeCheckNameHandle in socket.handlers at 0x107;
# MapleLifeResult/MapleLifeError in socket.writers at 0x167/0x168 — all confirmed.

import yaml
d = yaml.safe_load(open('docs/packets/registry/gms_v84.yaml'))
# MAPLELIFE_RESULT clientbound 359, MAPLELIFE_ERROR clientbound 360,
# MAPLELIFE_CHECK_NAME serverbound 263 — all confirmed, no new collisions.
```

## Files changed

- `services/atlas-configurations/seed-data/templates/template_gms_84_1.json`
- `docs/packets/registry/gms_v84.yaml`
- `docs/tasks/task-246-maple-life-character-creation/derivation.md`
- `services/atlas-configurations/atlas.com/configurations/socket/corpus_test.go`
  (incidental — expected-count assertion, not in the brief's file list, but
  required to keep the module's tests green after the seed-data edit)

Commit: `7f1dc0a43` — "fix(atlas-configurations): register Maple Life ops for gms_v84.1"

## Self-review findings

- Verified the JSON insertion was a pure addition (`diff` against the
  pre-edit file showed only added lines, nothing altered/removed).
- Verified placement order in both the handlers/writers lists and the
  registry rows follows the existing adjacency convention (matching where
  gms_83_1 places the analogous entries relative to its neighbours), rather
  than a blind append.
- Did not touch `docs/packets/evidence/gms_v84/` or
  `docs/packets/audits/status.json`/`STATUS.md`, per the ruling.
- Did not renumber or rewrite any existing derivation.md section; the
  correction is a pure append.
- No opcode collisions introduced (checked programmatically).

## Concerns

- None regarding the in-scope work. The brief's own "Not yet answered"
  section (jms_v185 unresolved, no live re-test) is unchanged by this task
  and remains open — that's expected, out of scope here.
- `services/atlas-configurations/atlas.com/configurations/socket/corpus_test.go`
  was not in the brief's file list but had to be updated to keep
  `go test ./...` green after the seed-data edit (Contract 2: "if your own
  module-local go build/go test fails, that IS yours — fix it before
  reporting"). Flagging in case the controller wants a reviewer to double
  check the count arithmetic (3384 + 3 = 3387, and the actual added rows are
  exactly 1 handler + 2 writers = 3).

## Resolution section in the bug file

I did not fill in the `## Resolution` section of
`bug-maple-life-v84-registration.md` — that section explicitly asks to
"record the fixing commit, the gate verdict, and whether live testing
confirmed it, before closing," which depends on the repo-wide gate verdict
(`tools/verify.sh`) that is the task-verifier's job, not mine. Leaving it for
the controller/verifier to close out with the gate result.
