# Review: Task 15 — fix round 1

**Range:** `ef896ed71..ac3c8bde5` (commit `ac3c8bde5`)
**Scope:** `docs/tasks/task-269-ring-pair-behavior/plan.md`,
`docs/tasks/task-269-ring-pair-behavior/coverage-manifest.yaml` — documentation-only, per
`git diff --name-only ef896ed71..ac3c8bde5`.

## Verdict: ADDRESSED

Both blocking findings from `reviews/task-15.md` are genuinely closed on independent evidence,
not merely edited. One minor citation-line off-by-one found; non-blocking.

## Blocking 1 — six plan.md A5 slips (Rulings 7, 23, 25, 29, 32, 35)

All six located and corrected. Verified independently, not by re-reading the brief's claims:

- **Ruling 7** (fixture table, `plan.md:187-195`). Recomputed every row's byte count from the
  literal hex groups in that row (not trusted from the diff):
  - GMS couple only: `01`+8+8+4+`00`+`00` = 1+8+8+4+1+1 = **23** ✓ (was 24, now 23)
  - GMS friendship only: `00`+`01`+8+8+4+`00` = 1+1+8+8+4+1 = **23** ✓ (was 24, now 23)
  - GMS marriage only: 1+1+1+4+4+4 = **15** ✓ (unchanged, was already right)
  - GMS all three = couple-arm(21) + friendship-arm(21) + marriage-arm(13) = **55** ✓ (was 45,
    now 55) — computed directly from `ring.go`'s `EncodeField`, not by summing the single-arm
    rows (which double-count the other arms' empty flag bytes)
  - GMS v95 all three: byte-identical to GMS v83 all-three → **55** ✓
  - JMS couple only: `01`+`01000000`(4)+8+8+4+`00`+`00` = 1+4+8+8+4+1+1 = **27** ✓ (was 28, now
    27)
  All six corrected figures are arithmetically true against their own row contents.

- **Ruling 23** (`plan.md:733`). Corrected line:
  `func requestByCharacterId(ctx context.Context, characterId uint32) (string, error)` — a bare
  URL, consumed via `requests.DrainProvider`. Confirmed this is the real repo convention, not an
  invented signature: `services/atlas-channel/atlas.com/channel/quest/requests.go:14-24`
  (`characterQuestsUrl`) is the byte-for-byte precedent — same `(string, error)` return, same
  "paginated, consumed via `requests.DrainProvider`" doc-comment rationale.

- **Ruling 25** (`plan.md:854-857`, FR-15 selection rule). Corrected prose: "numerically highest
  (least negative — ring1 before ring2 before ring3 before ring4) equipped slot position wins."
  Checked against the slot constants cited two paragraphs above (`ring1=-12, ring2=-13,
  ring3=-15, ring4=-16`) and the worked example directly below (7777 in ring1/-12, 1111 in
  ring2/-13 → `Couple.OwnSN == 7777`): -12 is numerically highest among the four, ring1 wins,
  7777 wins — prose and example now agree, and the arithmetic is right (not merely
  self-consistent after being edited to match a wrong prose, since -12 genuinely is the least
  negative of the four listed constants).

- **Ruling 29** (`plan.md:427`). Corrected to "total length grows by exactly 20." Derived
  independently from `libs/atlas-packet/model/ring.go:72-84` (`encodePair`): a populated GMS arm
  writes flag(1) + OwnSN int64(8) + PartnerSN int64(8) + ItemId int(4) = 21 bytes. The 3-byte
  empty span (couple flag + friendship flag + marriage flag, all zero) is replaced by the
  21-byte populated couple block plus the two still-zero friendship/marriage flags (21+2=23);
  23−3 = **20**. Matches the code, not just the brief's assertion.

- **Ruling 32** (`plan.md:1007-1010`, FR-4). Corrected: cache clears "only on session `Destroy`
  (logout, disconnect, timeout, and channel change all funnel through it)... an intra-channel
  map transfer must NOT drop the entry." Traced to real source —
  `services/atlas-channel/atlas.com/channel/session/processor.go:409-429` (`Destroy`) calls
  `clearRingsOnDestroy` (`processor.go:498-503`), with a doc comment that is nearly word-for-word
  what the plan now says: "logout, disconnect, timeout, and channel change all funnel through
  Destroy... an intra-channel map transfer must NOT drop the entry." (This landed on the branch
  via `b6f255f71 fix(atlas-channel): invalidate the ring cache on session destroy (FR-4)`,
  predating this fix round — the plan text now accurately describes already-shipped behavior.)

- **Ruling 35** (`plan.md:1113-1123`, note on Task 13/14 `### Files`). Verified both claims by
  direct file inspection:
  - `libs/atlas-packet/character/clientbound/v61_test.go` and `v72_test.go` both carry
    `packet-audit:verify` markers for `CharacterSpawn`, `CharacterInfo`, and
    `CharacterAppearanceUpdate` (confirmed via `grep -n "packet-audit:verify"` on both files).
  - `character_spawn_test.go` genuinely lives at
    `services/atlas-channel/atlas.com/channel/socket/writer/character_spawn_test.go` — a
    different module from `libs/atlas-packet` (confirmed via `find`).

No slip survives elsewhere in plan.md — swept for the pre-fix literal strings ("grows by exactly
18", "| 24 |", "| 28 |", "| 45 |", "most negative first", "lowest equipped slot position",
"Request[[]RestModel]" as a return type, "Map/channel transfer") and the only hit is the
corrected Ruling-23 line itself, which legitimately contains `requests.Request[[]RestModel]` as
prose contrasting with the real return type.

## Blocking 2 — Ruling 1's WITHDRAWN record

Present at `coverage-manifest.yaml:12-35` (header comment block), reads **"WITHDRAWN as
never-applicable (not 'resolved', not 'discharged'...)"** verbatim — correct wording, not
"resolved"/"discharged". Both citations checked directly:

- `tools/packet-audit/internal/evidence/hash.go:14-38` — exact match, this is the full body of
  `FunctionHash` (opening `func` line through closing brace), which hashes
  `file.Functions[fname]` from the on-disk IDA export JSON, never touching Go encoder output.
  Citation is precise.
- `tools/packet-audit/internal/matrix/evidence_input.go:29-30` — off by one line: line 29 is a
  closing brace (`}`), and the `evidence.FunctionHash(exp, r.IDA.Function)` call the comment
  describes is actually on line 30 (with `if err != nil {` on line 31). The substance of the
  claim (this call compares export-vs-export, never encoder output) is correct; the line range
  is imprecise by one line. **Non-blocking** — a reader following the citation lands one line
  early, in the immediate vicinity of the real call, not somewhere unrelated.
- `grep -c "packet-audit" tools/verify.sh` → confirmed **0** by running it myself.
  `.github/workflows/packet-matrix.yml` is confirmed to be the only place this check runs.

## Non-blocking — manifest schema-deviation disclosure

`coverage-manifest.yaml:7-14` adds the header-comment note disclosing the `versions:` per-op
mapping as an intentional deviation from PROCESS.md's flat-list example, with the
CharacterSpawn/CharacterInfo (10 versions) vs. CharacterAppearanceUpdate/CharacterData (9
versions, v92 excluded / v48 n-a) rationale. The per-op `versions:` mapping form itself was kept
unchanged (confirmed via diff — only comment lines were added, `versions:` structure below is
untouched). Matches the brief's non-blocking item exactly.

## Scope check (P4)

- `git diff --name-only ef896ed71..ac3c8bde5` → exactly the two documentation files. No Go file
  touched.
- `coverage-manifest.yaml` parses cleanly as YAML (`python3 -c "import yaml; yaml.safe_load(...)"`
  succeeded).
- Per-op `versions:` mapping form preserved; only comment lines added.
- `agent-ledger.tsv` and `reviews/` remain untracked (`git status --porcelain`) — not committed
  by this fix commit.

## Not evaluable

None. Full P1–P4 checklist evaluated against real repo evidence within the two-file diff plus
the files it cites (`ring.go`, `quest/requests.go`, `session/processor.go`,
`hash.go`/`evidence_input.go`, `verify.sh`, `v61_test.go`/`v72_test.go`,
`character_spawn_test.go`).

## Summary

| Item | Status |
|---|---|
| Ruling 7 (fixture arithmetic) | Corrected and verified arithmetically true |
| Ruling 23 (bare URL) | Corrected and verified against real repo precedent |
| Ruling 25 (FR-15 prose/example) | Corrected; prose now agrees with example and with the slot constants |
| Ruling 29 (20-byte delta) | Corrected; derived independently from `ring.go`, matches |
| Ruling 32 (FR-4 clears on Destroy only) | Corrected; matches shipped `session/processor.go` code near-verbatim |
| Ruling 35 (Task 13/14 Files gaps) | Corrected; both file claims verified to exist as described |
| Ruling 1 WITHDRAWN record | Present, correct wording, citations verified (one off-by-one line, non-blocking) |
| Schema-deviation disclosure | Added; mapping form kept as instructed |
| Scope | Two files only, no Go touched, YAML valid |
