# Task 15 review — coverage manifest, service docs, wrap-up

Range: `9df12f9d0..ef896ed71` (single commit `ef896ed71`).
Brief: `.superpowers/sdd/plan/task-15-brief.md` (`## Controller addendum (Session 4)`, A1-A7).
Report: `.superpowers/sdd/plan/task-15-report.md`.

## Scope confirmed

Diff matches the report: `docs/tasks/task-269-ring-pair-behavior/coverage-manifest.yaml`
(new), `services/atlas-channel/docs/kafka.md` (+1 line), `libs/atlas-packet/character/clientbound/appearance_update_test.go`
(comment-only, +4/-3). `git diff --stat 9df12f9d0..ef896ed71` shows exactly these
three files, `+182/-3`. No codec file, no `status.json`/`STATUS.md` change
(`git diff --stat ... -- docs/packets/audits/` empty). `git status --short` at
review time shows only the two untracked controller artifacts
(`agent-ledger.tsv`, `reviews/`), confirming they were not committed. No scope
mismatch.

## P1 — `versions:` per-op mapping vs. PROCESS.md's flat-list schema

**PROCESS.md's documented schema** (`docs/packets/PROCESS.md:141-156`) is a
flat scalar list under `versions:` applied uniformly across all `ops:`
entries. The manifest instead uses a list of single-key mappings, one per op,
each with its own version list (`coverage-manifest.yaml:41-71`).

The stated reason is legitimate and independently verifiable: `CharacterSpawn`/
`CharacterInfo` really do cover all 10 versions while `CharacterAppearanceUpdate`/
`CharacterData` cover 9 (`gms_v48` n-a, `gms_v92` excluded) — a single flat
list would falsely claim v92/v48 coverage for the latter two ops. Confirmed
against `docs/packets/audits/STATUS.md` is out of this task's scope to
re-derive, but the report's citations (`SPAWN_PLAYER`/`CHAR_INFO` all-10,
`UPDATE_CHAR_LOOK`/`SET_FIELD` nine-with-`gms_v48` n-a) are internally
consistent with Task 14's report, which this branch's Task 14 review already
verified.

**Who actually reads this file:** there is no mechanical parser under
`tools/packet-audit/` for `coverage-manifest.yaml` — `grep -rn
"coverage-manifest" tools/packet-audit/` returns nothing. The sole consumer is
the `packet-completeness-critic` agent (`.claude/agents/packet-completeness-critic.md`),
an LLM that `Read`s the file and is instructed to "resolve every [ops] entry...
Build claimedOps — the op × version pairs from ops × versions." Because this is
LLM comprehension over a heavily-commented YAML file (not a strict library
parser), the per-op mapping will almost certainly be read correctly, not
silently parsed to nothing — the comments make the per-op intent explicit at
each entry, and grouping the version list under the SAME op-name key it
describes is, if anything, easier to disambiguate than a flat list would be
paired with the 4-entry `ops:` list.

**Finding (non-blocking):** the deviation is real and undisclosed in the
tracked artifact itself. The manifest's own header comment
(`coverage-manifest.yaml:1-8`) does not flag that `versions:` diverges from
PROCESS.md's schema — that disclosure exists only in the implementer's
self-review section of `task-15-report.md`, which is untracked (`git ls-files
.superpowers/sdd/plan/task-15-report.md` → empty). A future reader of the
committed manifest, or a stricter future tool built against PROCESS.md's
literal schema, has no signal in the repo that this is an intentional,
reasoned deviation rather than an authoring mistake. Recommend a one-line
comment in the manifest header (or a PROCESS.md schema addendum for the
per-op-coverage case) — not blocking, since the actual consumer will read it
correctly as written.

## P2 — controller addendum A1-A7 swept item by item

- **A1 (Ruling 4 / Ruling 36 in `out_of_scope:`)** — PASS. Ruling 4's four
  addresses (`coverage-manifest.yaml:105-109`) verified to exist at the cited
  offsets: `grep -n "0x98367e\|0xa090f4\|0x954110\|0xa57221"` against
  `docs/packets/ida-exports/gms_v83.json:7767`, `gms_v87.json:2992`,
  `gms_v95.json:8764`, `gms_jms_185.json:2140` — all four addresses exist in
  those exports. Ruling 36's citation (`coverage-manifest.yaml:114-126`)
  verified against `spawn_test.go:89` ("CUserRemote::Init sub_6BBC17") and
  `docs/packets/evidence/gms_v48/character.clientbound.CharacterSpawn.yaml:6`
  (`function: sub_6BBC17`) — matches. No export was hand-edited (diff stat
  confirms `docs/packets/ida-exports/` untouched).
- **A2 (seven ring-record fields, unverified semantics)** — PASS, recorded
  verbatim at `coverage-manifest.yaml:128-138`.
- **A3 (Ruling 17 — FR-8's v28 boundary)** — PASS, recorded in both `fields:`
  (`coverage-manifest.yaml:75`) and `residual:` (`coverage-manifest.yaml:140-146`),
  correctly stating `info.go`'s gate excludes v28 and that `gms_v48` was
  substituted.
- **A4 (Ruling 1 — WITHDRAWN as never-applicable)** — **FAIL, undischarged.**
  Grepped all three touched files plus the untracked report for "Ruling 1",
  "withdrawn", "discharged", "decompile", "evidence hash", "never-applicable":
  zero matches anywhere in the commit's tracked output. The only place this
  branch's session record still calls it "discharge" is
  `.superpowers/sdd/plan/task-14-report.md:141` ("## Ruling 1 discharge (A2)"),
  which is untracked (`git ls-files` empty) and was never corrected. The
  controller addendum's explicit instruction — "Write it as withdrawn /
  never-applicable, NOT as 'discharged'... so the next reader is [not]
  misled" — was not carried into any durable, git-tracked artifact by this
  wrap-up task. There is no factual defect here (Task 14's review already
  confirmed the premise is false, from real source:
  `tools/packet-audit/internal/evidence/hash.go:14-38`,
  `internal/matrix/evidence_input.go:29-30`, `grep -c "packet-audit"
  tools/verify.sh` → 0), only a bookkeeping instruction this task was
  specifically dispatched to discharge and did not.
- **A5 (plan.md authoring slips — Rulings 7, 23, 25, 29, 32, 35)** — **FAIL,
  undischarged.** `plan.md` was not touched by this commit (`git diff
  9df12f9d0..ef896ed71 -- .superpowers/sdd/plan/` empty for `plan.md`;
  `docs/tasks/task-269-ring-pair-behavior/plan.md` is also untouched by any
  commit in range). Verified the slips are still present verbatim:
  - Ruling 29 — `docs/tasks/task-269-ring-pair-behavior/plan.md:427` still
    reads "total length grows by exactly 18," contradicted by the manifest's
    own accurate arithmetic elsewhere in this branch (21-byte populated arm
    vs 1-byte empty = 20, per the ruling).
  - Ruling 23 — `plan.md:733` still states the literal
    `requests.Request[[]RestModel]` signature the ruling says is wrong.
  - Ruling 25 — `plan.md:854` still reads "most negative first" as the FR-15
    selection rule, unchanged.
  - Rulings 7 and 32 not independently re-checked line-by-line (out of this
    review's budget) but the same sweep (`grep -n` for their subject strings)
    turned up no correction markers anywhere in `plan.md` or the manifest.
  None of the six rulings is mentioned anywhere in `coverage-manifest.yaml`,
  `kafka.md`, or `task-15-report.md`. A5 was not corrected in `plan.md` and
  was not recorded as a deferred residual either — it was silently dropped.
- **A6 (stale comment fix)** — PASS. `git diff 9df12f9d0..ef896ed71 --
  libs/atlas-packet/` touches only lines 278-280 of
  `appearance_update_test.go`, comment text only. The new wording ("builds
  from the ringPopulatedJMSHex / ringEmptyHex captured literals above, not by
  calling the production codec on the want side") now matches the file's own
  doc comment at lines 55-62, which already described the hex-literal
  approach. No literal, assertion, or encode path changed — confirmed by
  reading the full hunk; it is a 2-line comment reword only.
- **A7 (Task 3 / Task 9 / Task 12 minors)** — PASS, all three recorded
  verbatim in `coverage-manifest.yaml:159-172`, and the "not residual"
  carve-out for Tasks 5/6/11's retired tautologies is correctly stated at
  `coverage-manifest.yaml:174-177`.

Net: 5 of 7 controller-addendum items (A1, A2, A3, A6, A7) are correctly
discharged with verified evidence. Two (A4, A5) are silently missing from
every tracked artifact this task produced, despite both being named
explicitly in the addendum this task was dispatched to close out.

## P3 — Ruling 1 wording accuracy

Moot for what was actually written, since A4 (above) was never written into
any tracked file by this commit. Independently re-verified the underlying
claim for completeness, since a future task will need to record it correctly:
`tools/packet-audit/internal/evidence/hash.go:14-38` computes `decompile_sha256`
over the IDA export JSON entry, and `grep -c "packet-audit" tools/verify.sh`
→ `0` (confirmed identically to Task 14's review). No wording in this
commit's diff describes the evidence hash at all, so there is nothing here
that misleads a reader — the defect is omission, captured under A4 above, not
inaccurate wording.

## P4 — scope leak / cell degradation

- No codec `.go` file (non-test) in the diff — confirmed by
  `git diff --stat`.
- `packet-audit:verify` marker sweep: `git diff 9df12f9d0..ef896ed71 | grep -c
  "packet-audit:verify"` → 1, and that single hit is prose inside a new
  manifest comment referencing an existing marker location
  (`spawn_test.go:100`), not an added/removed marker line. No marker was
  added or removed.
- `docs/packets/audits/status.json` and `STATUS.md`: `git diff --stat
  9df12f9d0..ef896ed71 -- docs/packets/audits/status.json docs/packets/audits/STATUS.md`
  → empty (unchanged).
- Untracked controller artifacts (`agent-ledger.tsv`, `reviews/`) are present
  in the working tree but `git status --short` shows them as `??` — not
  staged, not committed. Confirmed via the same status check used for scope
  confirmation above.
- `services/atlas-channel/docs/kafka.md`'s new bullet is traceable to real
  code: `kafka/consumer/asset/consumer.go:428` (`GetRingSet(c.Id(),
  c.Equipment())`, in `updateAppearance`, broadcast via
  `charcb.CharacterAppearanceUpdateWriter` at line 429) for the MOVED/equip
  path, and `kafka/consumer/cashshop/consumer.go:519,522`
  (`ring.NewProcessor(l, ctx).Invalidate(...)` for both the purchasing
  character and the partner in `handleStatusEventRingPurchased`, no broadcast
  call in that handler) for the residual limitation. Both citations check out
  against the current source.

## Verdict rationale

The manifest and service-doc content that WAS written is accurate, well-cited,
and independently verifiable — no fabricated evidence, no scope leak, YAML
parses (`python3 -c "import yaml; yaml.safe_load(...)"` → OK), A1/A2/A3/A6/A7
are genuinely discharged. But this task's brief is explicit that its whole
job is the "complete residual inventory" (A1-A7), and two of the seven named
items — A4 (Ruling 1's wording correction) and A5 (six `plan.md` authoring
slips) — were dropped without a trace: not corrected, not recorded as
deferred, not mentioned anywhere in the commit or the report's "what I did
not do" section (which only lists Step 3/5 of the original brief body, not
A4/A5 from the addendum). For a task whose sole purpose is bookkeeping
completeness, an unacknowledged silent drop of two of seven addendum items is
a blocking defect in the unit's own terms.

---

verdict: CHANGES_REQUIRED
artifact: docs/tasks/task-269-ring-pair-behavior/reviews/task-15.md
scope_confirmed: reviewed the full diff 9df12f9d0..ef896ed71 (coverage-manifest.yaml new file, kafka.md +1 line, appearance_update_test.go comment-only); traced kafka.md's claims into kafka/consumer/asset/consumer.go and kafka/consumer/cashshop/consumer.go; verified coverage-manifest.yaml's cited IDA addresses against docs/packets/ida-exports/*.json and the gms_v48 evidence yaml; checked docs/packets/audits/status.json and STATUS.md unchanged; swept the controller addendum A1-A7 against plan.md, the manifest, kafka.md, and the report
blocking: 2
  - .superpowers/sdd/plan/task-15-brief.md (Controller addendum A5) / docs/tasks/task-269-ring-pair-behavior/plan.md:427,733,854 — six plan.md authoring slips (Rulings 7, 23, 25, 29, 32, 35) that the addendum names as "to correct" were neither corrected in plan.md nor recorded anywhere in this commit's output; still present verbatim (e.g. plan.md:427 "grows by exactly 18" contradicting Ruling 29's verified 20-byte delta).
  - .superpowers/sdd/plan/task-15-brief.md (Controller addendum A4) — Ruling 1's required "WITHDRAWN as never-applicable" (not "discharged") wording was not written into coverage-manifest.yaml, kafka.md, or the report; the only extant record still calls it "Ruling 1 discharge (A2)" (task-14-report.md:141, untracked/uncorrected).
non_blocking: 1
  - docs/tasks/task-269-ring-pair-behavior/coverage-manifest.yaml:1-8,41-71 — the per-op `versions:` mapping is a reasoned, correctly-read-by-its-actual-consumer deviation from PROCESS.md's flat-list schema, but the manifest's own header comment does not disclose the deviation (only the untracked task-15-report.md self-review does); add a one-line disclosure comment.
not_evaluable: 0
