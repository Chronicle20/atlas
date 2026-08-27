# Task 13 review: packet coverage for `CharacterSpawn` / `CharacterInfo`

**Range:** `b6f255f71..ccfd35e81` (7 files, +138/-13)
**Files touched:** `docs/packets/audits/STATUS.md`, `docs/packets/audits/status.json`,
`docs/packets/evidence/gms_v92/character.clientbound.CharacterInfo.yaml` (new),
`docs/packets/evidence/gms_v92/character.clientbound.CharacterSpawn.yaml` (new),
`libs/atlas-packet/character/clientbound/info_test.go`,
`libs/atlas-packet/character/clientbound/spawn_test.go`,
`services/atlas-channel/atlas.com/channel/socket/writer/character_spawn_test.go`.

## Priority 1 — report's "addresses already match" claim

**Confirmed false for `gms_v48` `CharacterSpawn`, and this is a PRE-EXISTING defect, not a Task 13 regression.**

- Addendum supplied `gms_v48` CharacterSpawn = `ida=0x6b277b`, keyed to the export's
  named function `CUserPool::OnUserEnterField` (confirmed:
  `docs/packets/ida-exports/gms_v48.json` → `CUserPool::OnUserEnterField` @ `0x6b277b`).
- The marker actually checked in reads `ida=0x6bbc17`
  (`libs/atlas-packet/character/clientbound/spawn_test.go:100`), which the same
  export maps only to the **unnamed** function `sub_6BBC17`. There is also a
  second, unnamed entry at the SAME address as the named one, `sub_6B277B`
  (address `0x6b277b`), whose `calls` show it does one `Decode4` then
  `Delegate → sub_6BBC17` — i.e. `CUserPool::OnUserEnterField` is a thin
  wrapper and the real packet-body decode (name/guild/avatar/tail-flags) lives
  in the un-named `sub_6BBC17`.
- The evidence file `docs/packets/evidence/gms_v48/character.clientbound.CharacterSpawn.yaml`
  pins `function: sub_6BBC17`, not `CUserPool::OnUserEnterField` — consistent
  with the marker, and also not matching the addendum's named-function
  convention.
- `git log -S "0x6bbc17" -- libs/atlas-packet` confirms origin at `3d77511d0`
  ("GMS Legacy Versions (48.1/61.1/72.1/79.1)", PR #971) — this predates the
  branch this task is on. `git diff --name-only b6f255f71..ccfd35e81` confirms
  neither `spawn_test.go:100` nor the v48 evidence YAML is touched by this
  diff.
- The test's own doc comment (`spawn_test.go:96`) calls the function
  `CUserRemote::Init sub_6BBC17`. `CUserRemote::Init` does not appear as a
  name anywhere in `gms_v48.json`'s `functions` map — that name is an
  unverified inline claim, not grounded in the checked-in export.

**Verdict on the v48 cell's ✅:** the marker/evidence cite a real function
that does perform the CharacterSpawn body decode (its `calls` list —
`DecodeStr, DecodeStr, Decode2, Decode1, Decode2, Decode1, Delegate→AvatarLook::Decode, ...`
— matches the test's documented byte layout: name/guildname/mask, avatar,
position ints, tail flag bytes), so the byte fixture is not obviously
mis-targeted at the wrong packet. But it does not cite the export's *named*
top-level handler the way every sibling column and the addendum's convention
requires, and the test comment's function-name attribution (`CUserRemote::Init`)
is unsupported by the export. This is a real, pre-existing labeling defect —
**not introduced by Task 13** and **not Task 13's to fix**. It is a genuine
candidate for Task 15's `coverage-manifest.yaml` residual list (mislabeled
tier-1 evidence, function-name attribution not grounded in the export).

**Every other pre-existing marker the report vouched for genuinely matches**
(verified against each version's export `functions` map, not re-read from the
report):

| Op | Version | Marker addr | Export named function | Match |
|---|---|---|---|---|
| CharacterSpawn | gms_v61 | `0x7bd862` | `CUserPool::OnUserEnterField` @ `0x7bd862` | ✅ |
| CharacterSpawn | gms_v72 | `0x87bc74` | `CUserPool::OnUserEnterField` @ `0x87bc74` | ✅ |
| CharacterSpawn | gms_v79 | `0x8c8978` | `CUserPool::OnUserEnterField` @ `0x8c8978` | ✅ |
| CharacterInfo | gms_v48 | `0x71caed` | `CWvsContext::OnCharacterInfo` @ `0x71caed` | ✅ |
| CharacterInfo | gms_v61 | `0x8455ed` | `CWvsContext::OnCharacterInfo` @ `0x8455ed` | ✅ |
| CharacterInfo | gms_v72 | `0x91b961` | `CWvsContext::OnCharacterInfo` @ `0x91b961` | ✅ |
| CharacterInfo | gms_v79 | `0x96d8d5` | `CWvsContext::OnCharacterInfo` @ `0x96d8d5` | ✅ (addendum's own stated verification method) |

So the report's blanket claim was wrong only for the one cell (v48 spawn); the
implementer's *decision* to skip a duplicate v48 marker was correct regardless
(the scanner hard-errors on duplicate `(packet,version)` pairs — confirmed by
`matrix --check` still exiting 0 with no duplicate markers added).

## Priority 2 — fixture honesty (fault injection)

Perturbed `model/ring.go:74` (`w.WriteByte(0)` → `w.WriteByte(9)` inside
`encodePair`'s nil branch — untouched by this diff) and re-ran the affected
tests:

- `TestCharacterSpawnV92Golden` — FAIL (byte mismatch at the couple-arm flag
  byte, exactly where expected).
- `TestCharacterSpawnRingBlocks` (all subtests, including the new
  `empty_is_unchanged/GMS_v92` and `couple_populated_v92`) — FAIL.
- `TestCharacterInfoV92Golden` — PASS unaffected (CharacterInfo doesn't carry
  a `RingSet` field directly in this fixture path; expected, this op doesn't
  encode via `RingSet.EncodeField`).
- `TestCharacterSpawnV48Golden` (pre-existing) — FAIL, confirming that fixture
  is also live, not dead.
- `TestCharacterSpawnBodyCarriesRings/no_ring` in
  `services/atlas-channel/.../character_spawn_test.go` (the A2 fix) — FAIL,
  confirming the captured-literal assertion is a real net, not a tautology.

Restored `model/ring.go` from a pre-perturbation backup; re-ran the full
affected test set to confirm green; `git status --porcelain` is clean (only
pre-existing untracked `docs/tasks/task-269-ring-pair-behavior/agent-ledger.tsv`
and `reviews/` dir, neither part of this diff).

**Conclusion: the fixtures are real coverage, not decorative.**

## Priority 3 — A2 tautology retirement

Confirmed via `git diff b6f255f71..ccfd35e81 -- services/atlas-channel/.../character_spawn_test.go`:
the `no_ring` subtest now does
```go
want, _ := hex.DecodeString("c800...0000")
if !bytes.Equal(got, want) { ... }
```
replacing the prior `got2 := ...Encode(...); bytes.Equal(got, got2)` pattern.
`got` is a single live encode; `want` is a captured hex literal. No `EncodeField`
(or any encode call) appears on both sides — the Task 5/6 trap was not copied
forward. Fault-injection above confirms this subtest goes red under a
perturbation to the empty-ring path, which the old encode-vs-encode form could
not have caught.

The addendum's stated path (`libs/atlas-packet/character/clientbound/character_spawn_test.go`)
does not exist; the implementer correctly located and fixed the real file in
the `atlas-channel` module instead, and said so in the report. Confirmed
correct.

## Priority 4 — scope and no-regression

- **Codec files untouched:** `git diff --name-only` confirms `spawn.go`,
  `info.go`, and `model/ring.go` are not in the diff. No wire behavior changed.
- **`matrix --check` exits 0** — quoted output:
  ```
  note	n-a evidence consumed: CASHSHOP_CASH_ITEM_GACHAPON_RESULT × gms_v79 (docs/packets/feature-na-evidence.yaml)
  note	n-a evidence consumed: USE_TELEPORT_ROCK × gms_v48 (docs/packets/feature-na-evidence.yaml)
  ```
  (both notes pre-exist and are unrelated to this task's packets; exit code 0).
- **No regressed cell:** `git diff b6f255f71..ccfd35e81 -- docs/packets/audits/STATUS.md`
  shows exactly two data-row cells flip, both ❌→✅: `CHAR_INFO` gms_v92
  column and `SPAWN_PLAYER` gms_v92 column. No other cell in either row, and
  no other row, changed. The v92 summary-row stat line also updates
  consistently (62→64 verified, 676→674 not-yet, 7.0%→7.2%).
- **Freshness:** re-running `go run ./tools/packet-audit matrix` in place
  leaves `git status --porcelain` on `docs/packets/audits/` empty — the
  committed STATUS.md/status.json are current, not stale.
- **New evidence YAMLs carry real `verifies:` targets:**
  - `docs/packets/evidence/gms_v92/character.clientbound.CharacterSpawn.yaml`
    → `verifies: [spawn_test.go#TestCharacterSpawnV92Golden, spawn_test.go#TestCharacterSpawnRingBlocks]`
    — both functions exist (`spawn_test.go:143`, `spawn_test.go:168`).
  - `docs/packets/evidence/gms_v92/character.clientbound.CharacterInfo.yaml`
    → `verifies: [info_test.go#TestCharacterInfoV92Golden]` — exists
    (`info_test.go:218`).
  - Both YAMLs' `ida.function`/`ida.address` correctly resolve to the export's
    *named* functions (`CUserPool::OnUserEnterField` @ `0x92a4e0`,
    `CWvsContext::OnCharacterInfo` @ `0x9daa40`) — unlike the pre-existing v48
    spawn defect above, these two new records are done right.
- **Ledger/reviews not committed:** `git log b6f255f71..ccfd35e81 --name-only`
  contains nothing under `docs/tasks/task-269-ring-pair-behavior/`.

## Findings

### Non-blocking (pre-existing, not Task 13's fault)

1. `libs/atlas-packet/character/clientbound/spawn_test.go:100` and
   `docs/packets/evidence/gms_v48/character.clientbound.CharacterSpawn.yaml`
   cite the unnamed `sub_6BBC17` (a callee of the real named handler
   `CUserPool::OnUserEnterField`) rather than the named function itself, and
   the test's doc comment attributes it to an unverified name
   (`CUserRemote::Init`) not present in the export. Predates this branch
   (PR #971, commit `3d77511d0`). Not introduced or touched by Task 13.
   Recommend routing to Task 15's `coverage-manifest.yaml` residual list, per
   the reviewer brief's own guidance — this is exactly that kind of item.

No other non-blocking notes; no not-evaluable items — every check in the
brief and addendum was verifiable from the diff, the checked-in exports, and
live test runs.

## Verdict rationale

Task 13's actual diff is correct: it adds exactly the missing v92 columns for
both ops, uses named-function addresses pinned correctly in new evidence, ties
`verifies:` to real test functions, fixes the A2 tautology in the right file
(recovering from a stale addendum path) without reintroducing the Task 5/6
trap, and leaves the ✅ matrix for every other cell untouched. `matrix --check`
is green and the committed STATUS.md/status.json are fresh. The one hard fact
in dispute — the report's claim that the pre-existing v48 spawn marker's
address matches the addendum exactly — is false, but the underlying defect
predates this task's range and this task's own decision (skip a duplicate v48
marker) was correct regardless of the flawed justification. That is a report
accuracy issue, not a code defect in this unit, so it is recorded here as a
non-blocking finding rather than a blocking one.
