# Review — Task 16: Derive the Maple Life content values

Commit range: `0ef41c109..677b62c51` (one commit, `677b62c51`, docs-only —
new file `docs/tasks/task-246-maple-life-character-creation/maple-life-content.md`,
441 lines). No Go files changed; `tools/verify.sh` correctly not run.

Scope confirmed: the diff is exactly the one new file the brief specified.
This review evaluated evidence integrity per the task's own governing
constraint (Amendment Global Constraint "never invent a content value" +
CLAUDE.md's evidence rule), walking §1–§3's tables against their cited
sources rather than spot-checking, and checking §4 for completeness in both
directions.

## Method

Every citable claim in §1 and §3 that resolves to a local, readable artifact
(WZ XML node, repo Go source, repo seed-data JSON, `libs/atlas-constants`)
was independently re-read and diffed against the document's stated value,
using the controller-supplied local WZ root (`Cosmic/wz/`, v83-era). IDA-only
claims (Step 1's StringPool cipher decode, Step 2's `OnCreate` budget
exhaustion) cannot be independently re-run in this review (no IDA session
available) and are evaluated for internal consistency and plausibility
instead — flagged under "Not evaluable" rather than silently passed.

## §1 — WZ paths and per-gender option lists

**PASS, and decisively confirmed.**

- `Etc.wz/MakeCharInfo.img.xml` read directly from the local WZ root.
  `Info/CharMale` and `Info/CharFemale` exist as claimed, and every one of
  the eight per-gender option-list tables (slots 0-7, faces/hairstyles/
  haircolours/skincolours/tops/bottoms/shoes/weapons) in §1 is byte-for-byte
  identical to the WZ file's `<int>` values.
- The "`PremiumChar{Male,Female}` happen to hold byte-identical data" claim
  (§1, "For what it's worth...") is also independently confirmed:
  `Etc.wz/MakeCharInfo.img.xml`'s `PremiumCharMale`/`PremiumCharFemale`
  nodes carry the identical int values for slots 0-7 as
  `Info/CharMale`/`Info/CharFemale`. The claim is accurate, not an
  unverified aside.
- The §1 cross-check against `template_gms_95_1.json`'s `jobIndex:0,
  subJobIndex:0` entries was independently re-read
  (`services/atlas-configurations/seed-data/templates/template_gms_95_1.json`)
  and matches exactly (including the `hairColors` reordering the document
  itself calls out as "same set, different order").
- Step 1's IDA cipher-decode narrative (StringPool ids 1525/1526 →
  `Etc/MakeCharInfo.img/Info/CharMale`/`CharFemale`, overturning the
  `PremiumChar{Male,Female}` lead) cannot be independently re-run without an
  IDA session — see "Not evaluable" below. However the *result* the
  narrative claims (the resolved paths, and that they and `PremiumChar*`
  hold identical data in this snapshot) is independently confirmed against
  the WZ file, which is strong corroborating evidence the decode is real
  rather than a plausible-sounding guess landing on a name the controller
  already told the implementer was a WZ root node (the brief's addendum
  lists `CharMale`/`CharFemale`/`PremiumCharMale`/`PremiumCharFemale` as
  the confirmed root-level names — so the implementer could not have
  "guessed" `Info/CharMale` from that list alone, since `Info` is a level
  above those four, which raises the derivation's credibility further).

## §2 — ordinal → class table

**PASS**, correctly scoped as out-of-review per the task instructions:
ordinals 0/1 correctly cite §11 A2 as already-closed (verified against
`design.md:806-808`, A6, which does say "Derived: 0 = Warrior, 1 =
Magician"); ordinals 2/3/4 are UNCONFIRMED with the guide's point-5 order
shipped, exactly as the brief's Step 2 fallback authorizes. Not re-litigated
per the review's explicit out-of-scope list.

## §3 — per-class content

**Job ids** — PASS. All five values (100/200/300/400/500) and the
`RogueId`-for-Thief naming footnote independently confirmed against
`libs/atlas-constants/job/constants.go:96,106,116,123,130` (the document
cites `:127` for `RogueId`; actual line is `:123` — see finding below).

**AP/SP arithmetic** — PASS, fully reproduced.
`computeOnLevelAddedAP`/`computeOnLevelAddedSP`
(`services/atlas-character/atlas.com/character/character/processor.go:1685-1709`)
read directly: flat `5` AP/level for non-Cygnus jobs, flat `3` SP/level for
non-Beginner jobs. 29 level-ups (1→30) × 5 = **145 AP**, × 3 = **87 SP**,
matching the document exactly.

**Stat floors and per-class weapon ids** — PASS, decisively confirmed. All
five `deploy/seed/gms/95_1/npc-conversations/npc/npc-{1022000,1032001,
1012100,1052001,1090000}.json` files were read directly:
`rebalance_ap`→`targets` floors (35/20/25/25/20 for
str/int/dex/dex/dex respectively) and `award_item` weapon ids (`1302077`,
`1372043`, `1452051`, `1332063`, plus Pirate's two-candidate
`1492000`/`1482000`) match the document's table exactly, including the
`change_job` `jobId` params confirming the NPC→job mapping is not
alphabetical (as the document itself notes).

**`stats` table** — PASS. `baseStat = 4` confirmed at
`character/processor.go:2168` (`const baseStat uint16 = 4` inside
`ResetStats`). The five rows' arithmetic (`145 − (floor − 4)`) is correct:
Warrior 145−31=114, Magician 145−16=129, Bowman 145−21=124, Thief
145−21=124, Pirate 145−16=129 — all recomputed and matching.

**`spSkillId`** — PASS. `WarriorImprovedMaxHpIncreaseId = Id(1000001)` and
`MagicianImprovedMaxMpIncreaseId = Id(2000001)` confirmed at
`libs/atlas-constants/skill/constants.go`, adjacent to the cited region.

**`hp`/`mp` interval** — PASS, and correctly modeled as an interval rather
than invented as a point value. `resolveHPMPGainParams`
(`character/processor.go:1720-1802`) read directly: Warrior
`hpLower/Upper=24/28, mpLower/Upper=4/6`; Magician `10/14, 22/24`;
Bowman-**and**-Thief share one branch (`job.Bowman` and `job.Rogue` are in
the same `IsAIdentity` list) at `20/24, 14/16` — this is why the document's
Bowman and Thief rows are identical, and that is correct per the source, not
a copy-paste error; Pirate `22/28, 18/23`. All five rows' `50 + 29×[lower,
upper]` / `5 + 29×[lower,upper]` arithmetic was recomputed and matches the
document exactly (e.g. Warrior 746..862 / 121..179; Pirate 688..862 /
527..672).

**`mapId`** — PASS with one accuracy gap (finding below). Bowman/Magician's
"single match" claims were independently re-swept across every `Map.wz/Map/*`
subdirectory and confirmed genuinely single-candidate
(`101000003.img.xml`, `100000201.img.xml`). Thief's and Pirate's two-candidate
claims are also confirmed exactly
(`105100301.img.xml`/`103000003.img.xml`; `120000101.img.xml`/`912010200.img.xml`).
**Warrior's "single match" claim does not hold up to a full sweep** — see
finding below.

**`equipment` (top/bottom/shoes)** — PASS, honestly flagged as a synthesis
in both §3's own prose and §4 item 5 (no per-class WZ resource exists; the
generic per-gender index-0 option is used). This is the one §3 entry the
implementer itself flags as weaker than a direct citation, and the review
agrees that flag is warranted and sufficient — it is not overclaimed.

**`inventory`/`meso` package** — PASS. `2000002` (White Potion, price 160),
`2000006` (Mana Elixir, price 310), and `3010000` (The Relaxer) all
independently confirmed to exist at the cited WZ paths
(`Item.wz/Consume/0200.img.xml`, `Item.wz/Install/0301.img.xml`) with the
cited prices, and their names independently confirmed in
`String.wz/Consume.img.xml` / `String.wz/Ins.img.xml` ("White Potion",
"Mana Elixir", "The Relaxer" respectively).

## §4 — honesty in both directions

**Mostly PASS.** Every UNCONFIRMED/UNRESOLVED marker present in §1-§3
(class ordinal order, hp/mp interval, Thief/Pirate mapId, Pirate weapon,
top/bottom/shoes synthesis, gender non-selectability) has a corresponding
§4 entry — no silent gap found in that direction. §4 item 5 (top/bottom/shoes
synthesis) is the self-flagged case the review brief pointed at, and it is
accurately described.

However, one §3 cell is presented with more confidence than a full sweep of
its own cited method supports — see Finding 1 below, which §4 does not
capture.

## Findings

### Finding 1 (non-blocking) — Warrior's `mapId` "single match / fair confidence" does not survive a full sweep

`docs/tasks/task-246-maple-life-character-creation/maple-life-content.md:317`
records Warrior's map candidate as `102000003 (single match)`, confidence
"fair — one candidate," using the stated methodology "grep for the NPC's
numeric id inside each map's `life` node."

Re-running that exact methodology across every `Map.wz/Map/*` subdirectory
(not just `Map1`) finds NPC id `1022000` in **four** maps, not one:
`Map1/102000003.img.xml`, and `Map9/910200000.img.xml`,
`Map9/910200001.img.xml`, `Map9/910200002.img.xml`. The three `Map9` hits
are a "Hidden Street / Hidden Relic"-named event map series
(`String.wz/Map.img.xml`, node `910200000`) — almost certainly the same kind
of non-overworld reused-NPC-id noise the document itself correctly flags as
suspect for Pirate's `912010200` candidate ("very likely a JMS-only map…the
`912` map-id prefix pattern is not a GMS Pirate-area convention"). The
document applies that filtering logic to Pirate but not to Warrior, where
the identical pattern (a `9xx`-prefixed event map sharing the NPC id)
exists and was apparently not found — i.e. the sweep for Warrior was
narrower than the sweep for Pirate/Thief, even though both are described
with the same "grep" methodology.

This does not change the practical value recorded (`102000003` is still
almost certainly the right pick — the three Map9 hits are obviously
non-candidates once seen), and the document's own blanket caveat ("Treat
all five as UNCONFIRMED for live-testing purposes") already covers the
residual risk. But the specific claim "single match" is factually wrong for
this row, and the asymmetric depth of sweep between Warrior and Pirate is
exactly the "spot-check presented as a full sweep" pattern CLAUDE.md warns
against. Recommend: either broaden the mapId sweep to the same depth for
all five rows and correct the "single match" wording (should read "one
non-event-map candidate; also appears in three Map9 event maps
(910200000-2), treated as non-candidates for the same reason as Pirate's
912010200"), or leave as-is if the controller judges the downstream value
unaffected — but the current wording overstates certainty this pass did not
actually establish for that one row.

### Finding 2 (non-blocking) — off-by-4 line citation for `RogueId`

`maple-life-content.md`'s job-id table cites `libs/atlas-constants/job/constants.go:127`
for `RogueId = Id(400)`. The actual declaration is at `constants.go:123`
(confirmed by direct read: `RogueId                    = Id(400)` at line
123). A four-line citation drift — harmless for a human reader locating the
constant, but a citation that doesn't resolve to its stated line is exactly
the kind of small integrity gap this review is asked to catch. Does not
affect the value itself (`400` is correct).

## Not evaluable

- **Step 1's IDA cipher-decode derivation** (StringPool `GetString`/`Decode<char>`/`rotatel`
  chain, the 16-byte `ms_aKey`, the seed-as-bit-shift brute-force). No IDA
  session was available to this review to re-run the decompile or replay
  the cipher independently. The *result* is strongly corroborated (see §1
  above — the resolved node exists, holds the exact claimed values, and is
  independently distinguishable from the `PremiumChar*` lead the brief
  flagged as unconfirmed), but the mechanism itself is taken on the
  implementer's account.
- **Step 2's `OnCreate` budget-exhaustion claim** (125,000-character
  decompile, 51 `StringPool::GetBSTR` hits). Not independently re-run for
  the same reason. The document's handling of this gap (UNCONFIRMED,
  guide's ordering shipped, explicit rationale) is evaluated for honesty
  only, not for whether a deeper pass could have closed it — that
  evaluation is out of this review's reach without IDA access.

## Section ordering

**PASS.** §1 (WZ paths + per-gender option lists) → §2 (ordinal→class table)
→ §3 (five class entries) → §4 (gap list), matching the brief's Step 4
ordering requirement for Task 20's `<content §N>` markers.

## Verdict rationale

No content value in §1-§3 was found to be invented, unsourced, or
materially misrepresented. The two findings are a wording/confidence
overstatement on one map-id row (practical value unaffected) and a
four-line citation drift (value unaffected). Both are corrections, not
defects that would cause Task 20/21 to copy a wrong value. Approved with
findings.
