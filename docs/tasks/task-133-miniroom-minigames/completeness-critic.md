# Completeness critic — Phase-2 legacy minigame bring-up (task-133)

Scope: three commits on `task-133-miniroom-minigames`
(`e8772c0f4`, `60d855ddb`, `7f20f54e4`; range `e8772c0f4~1..7f20f54e4`).
Read-only audit. No codec/registry/template/yaml/evidence file was modified
by this pass.

**Verdict: 2 findings.** No CHANGED-BUT-UNCLAIMED scope hole. One
CLAIMED-BUT-UNVERIFIED structural gap (pre-existing, but this branch widened
it into 4 more version columns without adding matrix visibility). One
undocumented-but-correct disambiguation (a third open item beyond the two the
branch's own commit messages disclose). All four requested gates pass
(exit 0). Every IDA-derivation claim I spot-checked (7 functions across 4
IDBs) resolved to a real function whose structure matches the claim — no
fabrication found.

## 0. Coverage manifest

`docs/tasks/task-133-miniroom-minigames/coverage-manifest.yaml` does **not
exist** (confirmed via `find`). Per `docs/packets/PROCESS.md`'s "Coverage
manifest" section this is normally the first/blocking finding. The task's
scope is however unambiguously pinned by the three commit messages and by
`docs/tasks/task-133-miniroom-minigames/family-audit-minigame-legacy.md`
(committed inside `e8772c0f4`), so I used that as the declared-scope
surrogate for the CHANGED-BUT-UNCLAIMED check below rather than stopping.
**Recommendation:** add the manifest before merge — `ops: [PLAYER_INTERACTION,
UPDATE_CHAR_BOX]`, `versions: [gms_v48, gms_v61, gms_v72, gms_v79, gms_v84,
gms_v87, jms_v185]`.

## 1. CHANGED-BUT-UNCLAIMED — none found

Full file list for the three-commit range (`git diff --name-only
e8772c0f4~1..7f20f54e4`): `docs/packets/dispatchers/character_interaction.yaml`,
`docs/packets/dispatchers/character_interaction_handle.yaml`,
`docs/packets/registry/gms_v48.yaml`, the 6-version balloon audit
reports/evidence/exports, the 4 legacy `template_gms_{48,61,72,79}_1.json`
seed templates, `docs/packets/audits/{STATUS.md,status.json}`, 3
`*_test.go` marker/comment-only edits, and the family-audit doc itself. Every
file is inside the declared "legacy minigame + balloon" scope; nothing
touches an unrelated packet family.

- **No version-gate code changed.** `git diff e8772c0f4~1..7f20f54e4 --
  libs/atlas-packet | grep -E '^[+-].*(MajorVersion|MajorAtLeast|IsRegion|Region\(\))'`
  → empty. The only `.go` files touched are 3 `_test.go` comment/marker
  fixes (`operation_memory_game_retreat_answer_test.go`,
  `operation_memory_game_tie_answer_test.go`, `mini_room_balloon_test.go`) —
  no struct or gate logic changed.
- **Matrix delta matches the wire change, not hidden.** `git diff
  e8772c0f4~1..7f20f54e4 -- docs/packets/audits/status.json` shows: (a) the
  `UPDATE_CHAR_BOX`/`InteractionMiniRoomBalloon` row's `gms_v61/72/79/84/87`
  and `jms_v185` cells flip `incomplete`→`verified` (matches commits 1–2's
  claim); (b) a new synthesized `PLAYER_INTERACTION` op-row cell for
  `gms_v48` appears as `"state": "incomplete", "note": "tier-1 without
  fixture; verdict ❌", "opcode": 239` — this is the **expected, disclosed**
  side effect of registering the gms_v48 clientbound opcode (family-audit §5)
  — it is not silently green, `packet-audit matrix --check` still exits 0
  because ❌/incomplete states are non-blocking. Not a scope hole.

## 2. CLAIMED-BUT-UNVERIFIED

| op / packet | version(s) | actual state | recommendation |
|---|---|---|---|
| `PLAYER_INTERACTION` clientbound — all 11 `MEMORY_GAME_*` arms (`interaction/clientbound/InteractionMiniGame*`) | ALL 9 versions, including the 4 legacy columns this branch just wired | **No `packet-audit:verify` marker and no `status.json` row exists for any `InteractionMiniGame*` packet, any version.** Confirmed: `grep -c 'InteractionMiniGame' docs/packets/audits/status.json` → 0; `grep -rn packet-audit:verify libs/atlas-packet/interaction/clientbound/*_test.go` has zero `InteractionMiniGame*` hits. | This is a pre-existing gap (family-audit §0, "headline finding"), but `e8772c0f4` widened it: it added real config-resolved mode bytes to `character_interaction.yaml` for 4 more versions AND regenerated all 4 legacy templates' **writer** `operations` tables — i.e. it shipped new production wire behavior for gms_v48/61/72/79 with zero matrix-visible verification, despite the family-audit's own Recommendation #1 ("wire the parent versions first… before extending 4 more columns"). The mode values themselves check out (see §4) — the gap is that the branch can't currently promote any of these cells to ✅ because no fixture/marker infrastructure exists for the family at all. Follow-up: dispatch `packet-verifier`/`/verify-packet` for the 11 arms × 9 versions (or at minimum the 4 legacy + the 5 that Recommendation #1 says come first) before calling minigame clientbound coverage complete. |

## 3. Correctness of the derivation

### 3a. Confirmed-open items (as the branch's own commit messages disclose them)

- **case-58 EXIT_AFTER_GAME-candidate clientbound arm** (v48 Omok switch
  case 58 @ `0x573a10`, and its positional analogues v61 case59/v72 case59/
  v79 case64): explicitly flagged unresolved in `e8772c0f4`'s message
  ("the extra case-58 EXIT_AFTER_GAME notification arm (scope decision
  pending)") and again in `7f20f54e4` ("remains a scope decision"). Not
  modeled by any codec, not silently dropped — correctly disclosed both
  times.
- **v72 `RetreatAnswer` verify-marker semantic mispin**: confirmed via `git
  show 7f20f54e4 -- libs/atlas-packet/interaction/serverbound/operation_memory_game_retreat_answer_test.go`
  — the comment now states the pinned `ida=0x5febf2` marker is actually the
  v72 MemoryGame ASK_TIE handler (shared bool-answer body), not the true
  Omok `RETREAT_ANSWER` send site (`sub_64E953`); flagged in the commit
  message as a "cosmetic evidence follow-up (no byte/gate change)". The wire
  value itself (mode 49 = `0x31`) is unaffected — only the marker's `ida=`
  citation is loose. Correctly disclosed.

### 3b. Undocumented (but, on independent re-check, correct) — a third open item not surfaced in either commit message

`family-audit-minigame-legacy.md` §2.3 explicitly flags **RESULT vs SKIP as
ambiguous** for all four legacy versions/both dialog types (cases {55,56} on
v48 Omok, {55,56} on v48 MemoryGame, and their positional analogues on
v61/72/79) — "I could not disambiguate which literal number is which key
with certainty… **Recommendation:** a packet-verifier pass… should pull the
referenced UI string resource IDs… to settle RESULT vs SKIP definitively
before writing byte fixtures."

The shipped `character_interaction.yaml` (lines 81–82,
`MEMORY_GAME_RESULT`/`MEMORY_GAME_SKIP`) assigns the lower mode to RESULT and
the higher to SKIP for all four legacy versions, with **no comment citing
disambiguating evidence**, and neither `e8772c0f4` nor later commit messages
mention RESULT/SKIP at all (only case-58 and the v72 marker are called out as
open). A reviewer reading only the commit trail would not know this
previously-flagged ambiguity was ever addressed.

I independently re-decompiled the two v48 candidate addresses (port 13337) to
check whether the shipped assignment is actually right:

- `0x573e1d` (mode 55, Omok): decodes an outcome byte; if `==1`, pulls **two**
  string-pool resources (438, 1441) and starts a UI timer; else decodes a
  second "who" byte and picks between resource pairs (437/1442) or
  (439/1443) — win/lose/tie-shaped. Critically, **it also clears `this[674]`
  to 0 at the end** — the exact field that case 51 (READY) sets to 1 and case
  52 (UNREADY) clears (§2.1 of the family-audit) — i.e. it resets the
  ready-flag, consistent with a game-ending event.
- `0x5740df` (mode 56, Omok): decodes one byte, compares it to `this[48]`
  (local turn slot), sets a bool, and resets a **30000ms** timer — no
  strings, no ready-flag touch — consistent with a turn-skip/timeout
  notification (restarting the turn clock), not a game-ending event.

This supports RESULT=55/SKIP=56 (the shipped assignment) for v48, and by the
same positional-shift logic already validated in §4 below, for
v61/v72/v79 as well. **The value appears correct**, but the evidence trail
that justifies it does not exist anywhere in the repo — this violates the
project's grounding standard (a claim resolved without a citable derivation)
even though it isn't a wrong claim. Recommendation: add the two addresses +
the READY-flag/timer-value reasoning above as a yaml comment (mirroring how
the ASK_TIE/ASK_RETREAT swap correction was cited "3 ways" in `e8772c0f4`),
or route it through a proper `packet-verifier` pass per the family-audit's
own recommendation before it's relied on as settled.

### 3c. Spot-verified derivation claims — all confirmed real, no fabrication

The apparent contradiction between `family-audit-minigame-legacy.md` (an
earlier read-only pass — §1/§8 say v61/v72/v79 clientbound vtable walks were
"not attempted… unresolved") and `e8772c0f4`'s commit message (which claims
exactly those derivations) is **not** a fabrication: the do-mode
implementation performed the additional IDA work itself in the same commit;
the family-audit file was carried along as background context and was never
meant to be the final word. I independently re-decompiled every function
address the commit cites for this claim, on the live IDBs (list_instances
confirmed v48=13337, v61=13338, v72=13339, v79=13340, v84=13345):

- **v61** `sub_5F72F8` (Omok) and `sub_5AFE65` (MemoryGame): case sets
  `{44,45,48,49,52,53,55,56,57,58,59}` and `{44,45,52,53,55,56,57,62}` —
  exactly v48's case sets (§2.1/§2.2 of the family-audit) shifted **+1**,
  confirming the commit's "v48=v61-1" claim.
- **v72** `sub_64DFD3` (Omok) and `sub_5FE397` (MemoryGame): byte-for-byte
  identical switch structure (same case values, same relative default
  arithmetic) to v61's two functions, confirming "v72=v61".
- **v79** `sub_671DA8` (Omok) and `sub_61CE73` (MemoryGame): case sets
  `{49,50,53,54,57,58,60,61,62,63,64}` and `{49,50,57,58,60,61,62,67}` —
  exactly v83's values (50,51,54,55,58,59,61,62,63,64,68 / matching
  MemoryGame subset) shifted **-1**, confirming "v79=v83-1".
- **v84 balloon** `0x96ffb6`: `Decode1`(roomType, early-return if 0) →
  `Decode4`(roomId) → `DecodeStr`(title) → 5×`Decode1` — matches the claimed
  "roomType→roomId→title→5×byte" read order exactly.

All 7 spot-checked addresses are real functions whose case/read structure
matches what the yaml comments and commit messages claim. Combined with §3a
(the two disclosed opens) and §3b (the one undisclosed-but-correct open),
I found no case where a cited address didn't exist or didn't match its
claimed shape.

## 4. Gate re-run (each command run individually, not in a loop)

```
cd tools/packet-audit && GOWORK=off go build -o /tmp/pc ./     → built clean
/tmp/pc dispatcher-lint     → "dispatcher-lint: clean"                    exit 0
/tmp/pc operations --check  → "operations check OK (0 absent-writer note(s))" exit 0
/tmp/pc fname-doc --check   → "fname-doc check OK (238 structs without an audit report carry no fname)" exit 0
/tmp/pc matrix --check      → (no output)                                 exit 0
```

All four green. Additionally (not requested but fast/free): `GOWORK=off go
test ./interaction/...` from `libs/atlas-packet` — `ok` for `interaction`,
`interaction/clientbound`, `interaction/serverbound`.

## Summary for the reviewer

- No scope hole: every file this branch touched is inside its declared
  legacy-minigame-bring-up scope; no version-gate code and no unrelated
  packet family moved.
- Every mode-byte/read-order claim I spot-checked against live IDA is real
  and matches — the legacy clientbound derivation work is genuine, not
  copy-pasted or invented, despite superficially contradicting the
  earlier-committed family-audit doc (that doc predates this session's
  do-mode work).
- The one **structural** gap is pre-existing and disclosed by the branch's
  own prior audit doc, but this branch made it bigger without addressing it:
  the entire clientbound MEMORY_GAME family — now including 4 more legacy
  versions — has zero matrix-visible verification. Landing config-resolved
  wire behavior into 4 production seed templates with no fixture trail is
  the exact shape of risk this critic exists to flag.
- One ambiguity the branch's own audit flagged as open (RESULT/SKIP) was
  resolved in the shipped artifact without a citation — I re-derived it
  independently and it checks out, but the repo doesn't show that work, so a
  future reader has no way to tell it from a coin-flip.
