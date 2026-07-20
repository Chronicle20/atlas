# task-181 — mis-modeled clientbound codecs found via live-IDB (all versions)

Verifying the divergent writers against the **live IDBs** (not the export)
surfaced codecs that are wrong for **every** version, whose existing
`packet-audit:verify` markers are a **false pass** (the golden test asserts the
encoder's own output, not the client read order). These predate task-181 and are
on `main`.

## DEFECT-1: SnowballState — 1 snowball + unconditional tail (should be 2 + gated) — **FIXED**

**Resolution (task-181):** codec re-modelled (2 snowballs + first-gated damage
tail); `SnowballState.Encode/Decode` + channel wrapper + goldens corrected; the
false-derived read-order was spliced with the real 10-call order in the
gms_v79/v83/v84/v87/v95 exports; evidence re-pinned and the SnowballState report
regenerated per version. All five cells verify ✅ against the corrected body,
`matrix --check` clean. **Residual:** jms_v185 has no ida-export file (its reports
come from live mcp), so its SnowballState report still shows the old 8-field
layout — its cell is ✅ and its evidence hashes the correct client decompile, but
the report doc needs a live-jms mcp regen to reflect the 10-field body.

Original finding below.


`CField_SnowBall::OnSnowBallState`, re-read directly in six IDBs — **identical
structure in all of them**:

| version | session/IDB | addr |
|---|---|---|
| gms_v79 | GMS_v79_1_DEVM.exe | 0x5525bf |
| gms_v83 | MapleStory_dump.exe | 0x5750a3 |
| gms_v87 | GMSv87_4GB.exe | 0x5a3328 |
| gms_v95 | GMS_v95.0_U_DEVM.exe (PDB-backed) | 0x560ab0 |
| jms_v185 | MapleStory_dump_SCY.exe | 0x5c959d |
| gms_v84 | (byte-identical to v83, per project memory) | 0x584a1c |

Real wire (PDB names from v95):
```
Decode1  state           -> m_nState (bFirst = prev m_nState == -1)
Decode4  leftSnowmanHp   -> m_aSnowMan[0].m_nHP
Decode4  rightSnowmanHp  -> m_aSnowMan[1].m_nHP
2x { Decode2 x; Decode1 y } -> CSnowBall::SetPos(m_aSnowBall[0..1])   <-- TWO snowballs
if bFirst: Decode2 damageSnowBall; Decode2 damageSnowMan0; Decode2 damageSnowMan1
```

Atlas `SnowballState.Encode` writes `byte,int,int, short,byte, short,short,short`
— **one** snowball and the 3 damage shorts **unconditionally** (18 bytes). The
client reads 15 (non-initial) or 21 (initial) bytes. They never match.

**Correct model** (version-agnostic — no gate): `state byte, leftSnowmanHp uint32,
rightSnowmanHp uint32, snowball0{x uint16, y byte}, snowball1{x uint16, y byte},
first bool, damageSnowBall uint16, damageSnowMan0 uint16, damageSnowMan1 uint16`.
`first` is not on the wire (client gates on its own prior state == -1); the
server sets it for the initial snapshot, and Decode recovers it from the
trailing bytes' presence (`r.Available() >= 6`). The channel wrapper
`services/atlas-channel/.../writer/snowball_state.go` (its only caller — never
actually emitted) takes the widened signature. This fix was implemented and
green (`go test ./field/clientbound/` + atlas-channel build) but **backed out**
pending the blocker below.

## DEFECT-2: AriantArenaUserScore — single entry (should be a count-length list) — **FIXED**

**Resolution (task-181):** codec re-modelled as `entries []AriantArenaScoreEntry{Name,
Score}` with `count = len(entries)`, `Encode`/`Decode` looping over it; channel
wrapper (`AriantArenaUserScoreBody`, its only caller — never emitted) widened to
take `[]AriantArenaScoreEntry`. Re-verified the read order live in v79 (`@0x528799`),
v83 (`@0x53e5e1`), v95 PDB-backed (`@0x5492b0`), plus v87/jms addresses via
func_query — identical `Decode1(count)` + count-length loop of
`{DecodeStr(name), Decode4(score)}` in every version. As predicted, the
v83/v84/v87/v95/jms exports already held the correct count+one-iteration shape
(`[Decode1, DecodeStr, Decode4]`) — no splice needed there. Only the v79 export
was `unresolved` (function not found under that name at export time); spliced in
its address + calls, re-pinned evidence, and routed the writer in
`template_gms_79_1.json` (opcode `0x113`, previously unrouted) + registry entry.
All six cells now verify ✅ (`matrix --check` clean); goldens updated (2-entry +
empty-list cases) plus a `TestAriantArenaUserScoreByteOutputV79`.

Original finding below.

atlas models a single `count,name,score`; the client reads `Decode1(count)` then
a **count-length loop** of `DecodeStr,Decode4` into `ZArray<UserScore>`.
Re-confirmed in the live IDBs: v79 `OnUserScore @0x528799`, v95 (PDB-backed)
`@0x5492b0` — both loop. Same false-pass class as SnowballState (single-entry
model + single-entry export coincidentally match).

Fix shape: re-model as `entries []{name string, score uint32}` with `count = len`.
NOTE the export convention question — a variable count-loop can't be flat-expanded
like SnowballState's fixed 2x; the existing export `[Decode1, DecodeStr, Decode4]`
already represents the count + one-iteration shape, so the fix is likely
**codec-only** (Encode/Decode loop that flattens to that shape), no export splice.
Confirm against a precedent list writer's grading before landing.

- **TournamentMatchTable** — atlas `Encode` is an **empty stub**; v79
  `OnTournamentMatchTable @0x55871f` reads a real match-table struct
  (`sub_750E40`). Needs the real body reversed. **FIXED — see DEFECT-6.**

## DEFECT-3: ContiMove — unconditional single state byte (should be state + state-gated subState) — **FIXED**

**Resolution (task-181):** re-read `CField_ContiMove::OnContiMove` live across
five IDBs — gms_v79 `@0x5374c1`, gms_v83 `@0x54dca3`, gms_v87 `@0x577bbc`,
gms_v95 `@0x54d680` (PDB-backed, switch form), jms_v185 `@0x58e21b`
(gms_v84 `@0x55a4e2` byte-identical to v83) — all **identical structure**:
`Decode1(state)` dispatches on `(state-7)` to one of six arms. Descending into
each arm's body (not just the top-level dispatch) showed three of the six
(state 8/10/12 — `OnStartShipMoveField`/`OnMoveField`/`OnEndShipMoveField`,
named via `CShip::LeaveShipMove`/`AppearShip`/`DisappearShip`/`EnterShipMove`
in v83/v87/v95/jms) each `Decode1` a **second** `subState` byte; the other
three (state 7/9/11) are nullsubs that read nothing further. This is a genuine
**true false-pass**, not a route-only case: the prior atlas codec wrote/read
only the unconditional state byte, silently dropping subState for 8/10/12 —
and the v83/v84/v87/v95/jms ida-exports encoded the same wrong 1-call shape
(matching the false golden), so the pre-existing ✅ cells were false passes too.

Re-modelled `ContiMove{state byte, subState byte}` with a shared
`contiMoveHasSubState(state)` gate (state ∈ {8,10,12}) used by both `Encode`
(conditionally writes subState) and `Decode` (conditionally reads it) —
deterministic on the state value itself (not off-wire, unlike SnowballState's
`first`). Widened the channel wrapper `ContiMoveBody(state, subState)` (its
only caller — never actually emitted). Corrected the 2-call read order
(`Decode1` state + state-gated `Decode1` subState) in the
gms_v79/v83/v84/v87/v95/jms exports (v79 was `unresolved`; the other five held
the same wrong 1-call shape as the old codec), re-pinned all six evidence
records, regenerated the ContiMove report per version, and routed it in
`template_gms_79_1.json` (opcode `0x8C`, previously unrouted between `0x8B`
Clock and `0x8D` FieldTransportState) + registry entry. All six cells verify
✅ (`matrix --check` clean); goldens updated (nullsub state + both v83/v79
two-byte cases) plus `TestContiMoveByteOutputV79` /
`TestContiMoveByteOutputV79Nullsub`.

## DEFECT-4: TournamentSetPrize — unconditional int-pair (should be flag-gated) — **FIXED**

**Resolution (task-181):** re-read `CField_Tournament::OnTournamentSetPrize` live
across five IDBs — gms_v79 `@0x5587e3`, gms_v83 `@0x57b815`, gms_v87 `@0x5a9f62`,
gms_v95 `@0x5633a0` (PDB-backed), jms_v185 `@0x5cffa7` (gms_v84 `@0x58b326`
byte-identical to v83) — all **identical structure**: `Decode1(slot)`,
`Decode1(flag)`; only when `flag != 0` does the client `Decode4` two further
ints (both fed to `CItemInfo::GetItemName`, formatted into the client string
`"...PRIZE...1ST: %s...2ND: %s"` — SP_917 in v83/v79). When `flag == 0` no
further ints are read; `slot` instead selects one of two success/failure
StringPool messages. This is a genuine **true false-pass**, not a route-only
case, of the same class as ContiMove: the prior atlas codec wrote/read the two
item ids **unconditionally**, silently desyncing the client whenever
`flag == 0`. The v83/v84/v87/v95/jms ida-exports already held the CORRECT
guarded shape (`Decode4` rows carried `guard: "CInPacket::Decode1(v2)"`), so
only the codec was wrong there — the exports themselves were never spliced for
those five. Only the v79 export was `unresolved` (function not found under
that name at export time); spliced in its address + the same 4-call guarded
shape.

Re-modelled `TournamentSetPrize{slot byte, flag byte, itemId1 uint32, itemId2
uint32}` (renamed the trailing fields from `itemId`/`count` to `itemId1`/
`itemId2` — both are verified item ids, not an item+count pair) with a shared
`tournamentSetPrizeHasItems(flag)` gate (`flag != 0`) used by both `Encode`
(conditionally writes the two ints) and `Decode` (conditionally reads them) —
deterministic on the flag value itself, no off-wire recovery needed. Widened
the channel wrapper `TournamentSetPrizeBody(slot, flag, itemId1, itemId2)`
(its only caller — never actually emitted). Re-pinned the gms_v79 evidence
record, regenerated the TournamentSetPrize report (selective per-version
revert — the v79 report set is ~200+ files stale and regen churns all of
them), and routed it in `template_gms_79_1.json` (opcode `0x127`, previously
unrouted between `0x124` CharacterInteraction and `0x128` TournamentUew) +
registry entry. All six cells verify ✅ (`matrix --check` clean); goldens
updated (flag-set + flag-clear cases) plus `TestTournamentSetPrizeByteOutputV79`
/ `TestTournamentSetPrizeByteOutputV79NoItems`.

## DEFECT-5: Tournament — unconditional 3rd byte (true wire is flat 2 bytes) — **FIXED**

**Resolution (task-181):** re-read `CField_Tournament::OnTournament` live
across five IDBs — gms_v79 `@0x5585af`, gms_v83 `@0x57b61a`, gms_v87
`@0x5a9d67`, gms_v95 `@0x5631a0` (PDB-backed, condition negated but
functionally identical), jms_v185 `@0x5cfdac` (gms_v84 `@0x58b12b`
byte-identical to v83) — all **identical structure**. The leading
`if (Decode1() || (secType&1)==0)` reads the FIRST byte as part of the
branch condition itself (C `||` short-circuit: the second operand — a
purely client-local `TSecType` flag — is never a wire read, only evaluated
when the first `Decode1()` is falsy). Whichever arm the branch selects then
reads exactly **one further** `Decode1()` unconditionally (a rank/place
value formatted into a champion/finalist/round-N notice in one arm; a
start-state value formatted into a prize-not-set/insufficient-users notice
in the other). Both arms terminate immediately after that second byte — no
further `CInPacket` reads on either path. The wire is therefore a **flat,
unconditional two bytes**; there is no third byte and no gating is needed in
the codec (unlike ContiMove/TournamentSetPrize, byte count never varies).
This is a genuine **true false-pass**: the prior atlas codec wrote/read an
unconditional THIRD byte, permanently desyncing the client on every
`OnTournament` packet (the excess byte gets consumed as the start of the
next packet header) — and the v83/v84/v87/v95/jms ida-exports encoded the
same wrong 3-call shape (v83/v87/v95 even tagged the 2nd/3rd calls with
mutually-exclusive `guard` fields, but still listed both as separate rows
instead of collapsing the mutex to one position), so the pre-existing ✅
cells were false passes too.

Re-modelled `Tournament{mode byte, value byte}` — dropped the third field
entirely; `Encode`/`Decode` are unconditional two-byte read/writes, no gate
function needed. Widened the channel wrapper `TournamentBody(mode, value)`
(its only caller — never actually emitted). Corrected the 2-call
unconditional read order in all six exports (v79 was `unresolved`; the
other five held the wrong 3-call shape), re-pinned all six evidence
records, regenerated the Tournament report per version (selective revert —
regen churns hundreds of unrelated files), and routed it in
`template_gms_79_1.json` (opcode `0x125`, `CField_Tournament::OnPacket`
case 293, previously unrouted between `0x124` CharacterInteraction and
`0x127` TournamentSetPrize — confirmed against the live `OnPacket` switch
decompile) + registry entry. All six cells verify ✅ (`matrix --check`
clean); goldens replaced (2-byte golden) plus `TestTournamentByteOutputV79`.

## DEFECT-6: TournamentMatchTable — empty stub (true wire is a 769-byte fixed body) — **FIXED**

**Resolution (task-181):** `CField_Tournament::OnTournamentMatchTable` itself
only allocates + `CDialog::DoModal`'s a match-table dialog — ALL wire reads
live in that dialog's constructor, which the handler calls as an anonymous
helper in every version except v87/v95 (named `CMatchTableDlg::CMatchTableDlg`
there, PDB-backed in v95). Decompiled the handler AND its ctor helper live in
five IDBs — gms_v79 `OnTournamentMatchTable @0x55871f` → ctor helper
`sub_750E40 @0x750e40`, gms_v83 `@0x57b78a` → `sub_7DE42C @0x7de42c`, gms_v87
`@0x5a9ed7` → `CMatchTableDlg::CMatchTableDlg @0x83517f`, gms_v95 `@0x5630d0`
→ `CMatchTableDlg::CMatchTableDlg @0x780210` (PDB-backed field names), jms_v185
`@0x5cff1c` → `sub_864212 @0x864212` (gms_v84 `@0x58b29b` byte-identical to
v83) — all **identical structure**: `CInPacket::DecodeBuffer(this->m_aaMatch,
0x300)` (a single bulk 768-byte memcpy — the v95 PDB types `m_aaMatch` as
`unsigned int[32][6]`, but the wire itself carries one opaque buffer, not 192
individually-typed `Decode4` reads) followed by `this->m_nState =
CInPacket::Decode1()` (one trailing byte). No count prefix, no conditional
gating — both fields are fixed-size and always present, so the true wire body
is a flat, unconditional **769 bytes**. This is a genuine **true false-pass**:
the prior atlas codec's `Encode` was an **empty stub** (`return w.Bytes()`
with zero writes) while `Decode` was likewise a no-op — every previously-✅
export (v83/v84/v87/v95/jms all held `"calls": null`, matching the empty
stub) was a false pass; the client always expects the 769-byte body and would
desync on every `OnTournamentMatchTable` packet.

Re-modelled `TournamentMatchTable{match [768]byte, state byte}` (raw
byte-array modeling of the `DecodeBuffer` blob, matching the existing
FILETIME/`DecodeBuffer(8)` convention used by `SetITC`/`MtsOperation` rather
than inventing per-field semantics for the opaque 32x6 grid) — `Encode`/
`Decode` are unconditional buffer-then-byte read/writes, no gate function
needed. Widened the channel wrapper `TournamentMatchTableBody(match, state)`
(its only caller — never actually emitted). Spliced the correct 2-call
`[DecodeBuffer, Decode1]` order into all six exports (v79 was `unresolved`;
the other five held `calls: null`), re-pinned all six evidence records,
regenerated the TournamentMatchTable report per version (selective revert —
regen churns hundreds of unrelated files), and routed it in
`template_gms_79_1.json` (opcode `0x126`, `CField_Tournament::OnPacket` case
294, between `0x125` Tournament and `0x127` TournamentSetPrize — confirmed
against the live `OnPacket` switch decompile) + registry entry. All six cells
verify ✅ (`matrix --check` clean); goldens replaced (non-zero 768-byte
fixture + trailing state byte) plus `TestTournamentMatchTableByteOutputV79`.

## DEFECT-7: MonsterCarnivalStart — route-only gap, codec already correct — **FIXED**

**Resolution (task-181):** `CField_MonsterCarnival::OnEnter` decompiled live in
five IDBs — gms_v79 `@0x548324`, gms_v83 `@0x565397`, gms_v87 `@0x59011d`,
gms_v95 `@0x55a6c0` (PDB-backed field names), jms_v185 `@0x5b014c` (gms_v84
byte-identical to v83) — **identical structure in all of them**:
`Decode1(team)`, then 6x `Decode2` (`SetPersonalCP(personalCp,
personalTotal)` + `SetTeamCP(team, myTeamCp, myTeamTotal)` +
`SetTeamCP(!team, enemyTeamCp, enemyTeamTotal)`), then a loop over the
client-local `m_aSummonedMob` array reading one `Decode1` (spelled level) per
element. The loop bound is **not wire-read**: each iteration checks
`m_aSummonedMob.a[-1]` (the array's own stored element count, the standard
ZArray header-count convention in this client, index `-1` before the data
pointer) against a running counter — confirmed identical in all five
decompiles, including the v95 PDB names (`m_aSummonedMob`, `SetPersonalCP`,
`SetTeamCP`) that anchor the field order.

Unlike DEFECT-1..6, this was **not a false pass** — the pre-existing atlas
`MonsterCarnivalStart` codec (`team byte, personalCp/personalTotal/myTeamCp/
myTeamTotal/enemyTeamCp/enemyTeamTotal uint16, spelled []byte` with the slice
length left to the caller, matching the off-wire loop bound) already modelled
this shape correctly, and v83/v84/v87/v95/jms were already ✅ with correct
exports (`[Decode1, Decode2 x6, Decode1]`). The only gap was **v79**: its
export held `unresolved` (function not found at export time) and opcode
`0x10B` (`CField_MonsterCarnival::OnPacket` case 267, confirmed against the
live dispatcher switch decompile) was never routed in `template_gms_79_1.json`.
Spliced the real read order into the v79 export, re-pinned its evidence,
regenerated the MonsterCarnivalStart report (selective revert), routed
`0x10B` in the v79 template between `0x10A` GuildBossPulleyStateChange and
`0x10C` MonsterCarnivalObtainedCP, added the registry entry, and added
`TestMonsterCarnivalStartByteOutputV79`. All six cells verify ✅ (`matrix
--check` clean).

## DEFECT-8: MonsterCarnivalSummon + MonsterCarnivalMessage — route-only gap, both codecs already correct — **FIXED**

**Resolution (task-181):** `CField_MonsterCarnival::OnRequestResult` is a
single dispatched function selected by a leading `int a2` argument passed by
its caller — `CField_MonsterCarnival::OnPacket` (`sub_54827B` in the v79 IDB)
switches on the packet opcode and calls `OnRequestResult(1, packet)` for
case 270 (`0x10E`, SUMMON) and `OnRequestResult(0, packet)` for case 271
(`0x10F`, MESSAGE) — confirmed against the live dispatcher switch decompile
in v79 (`@0x54827b`). Decompiled `OnRequestResult` itself live in five IDBs —
gms_v79 `@0x54850a`, gms_v83 `@0x56557d`, gms_v87 `@0x590303`, gms_v95
`@0x55a890` (PDB-backed, parameter literally named `bResult`), jms_v185
`@0x5b0332` (gms_v84 `@0x572284` byte-identical to v83) — **identical
two-branch structure in all of them**:

- `a2 != 0` (SUMMON, opcode `0x10E`): `Decode1` tab, `Decode1` idx, `DecodeStr`
  name, then `RequestResult(tab, idx, name)` — no further packet reads.
- `a2 == 0` (MESSAGE, opcode `0x10F`): a single `Decode1` message-selector
  byte, then a `switch` over cached `StringPool::GetString` IDs
  (`SP_4082..SP_4086` / raw IDs `0x101B..0x101F` depending on the version's
  StringPool table) to format a chat-log line — the displayed text comes from
  the client's local string table, never from the packet, and no further
  bytes are read after the selector.

Unlike DEFECT-1..6, this was **not a false pass** — the pre-existing atlas
`MonsterCarnivalSummon` (`tab byte, idx byte, name string`) and
`MonsterCarnivalMessage` (`message byte`) codecs already modelled both
branch shapes correctly, and v83/v84/v87/v95/jms were already ✅ for both
ops with correct 4-call/1-call exports. The only gap was **v79**: both ops'
shared export entry (`CField_MonsterCarnival::OnRequestResult`) held
`unresolved` (function not found at export time), and neither opcode `0x10E`
nor `0x10F` was routed in `template_gms_79_1.json`. Spliced the real
guard-gated read order (`Decode1` guarded `!a2` for MESSAGE, then the
unconditional `Decode1, Decode1, DecodeStr` for SUMMON — mirroring the
existing v83/v84/v87/jms export shape) into the v79 export, re-pinned both
evidence records, regenerated both reports (selective revert — the root
regen command still churns ~217 unrelated files; reverted everything except
the two `MonsterCarnival{Summon,Message}` report pairs and `git clean`ed the
recreated strays), and routed `0x10E`/`0x10F` in the v79 template between
`0x10D` MonsterCarnivalPartyCP and `0x112` MonsterCarnivalResult, plus two
registry entries. Added `TestMonsterCarnivalSummonByteOutputV79` and
`TestMonsterCarnivalMessageByteOutputV79`. Both v79 cells verify ✅ (`matrix
--check` clean); all five other versions remain ✅ for both ops.

Note: `status.json`/`STATUS.md` display a pre-existing, unrelated cosmetic
bug for the SUMMON row's "packet" column (it shows
`monster/carnival/clientbound/MonsterCarnivalMessage` instead of
`MonsterCarnivalSummon`) — `tools/packet-audit/internal/matrix/build.go`'s
`rowPacketAndTier()` picks the alphabetically-first writer sharing an FName
without matching it back to the specific op row, and both ops share
`CField_MonsterCarnival::OnRequestResult`. This does not affect per-op cell
grading (`worstCandidateCell()` grades each writer independently — both ops
show correct, independent ✅ verdicts) or `matrix --check`/`gatecheck`
(no `gates.yaml` entry keys on either packet name today). Left unfixed as
out-of-scope for this route-only task; flagging for a future cleanup of
`build.go`'s FName→packet resolution when multiple ops share one dispatcher.

## DEFECT-9: MonsterCarnivalDied + MonsterCarnivalLeave — route-only gap, both codecs already correct — **FIXED**

**Resolution (task-181):** `CField_MonsterCarnival::OnProcessForDeath`
(MonsterCarnivalDied) and `CField_MonsterCarnival::OnShowMemberOutMsg`
(MonsterCarnivalLeave) are two distinct opcode-dispatched handlers —
`CField_MonsterCarnival::OnPacket` (`sub_54827B` in the v79 IDB) switches on
the packet opcode and calls `sub_548774(a3)` for case 272 (`0x110`, DIED) and
`sub_5488EF(a3)` for case 273 (`0x111`, LEAVE) — confirmed against the live
dispatcher switch decompile in v79 (`@0x54827b`). Decompiled both handlers
live in five IDBs:

- `OnProcessForDeath` — gms_v79 `@0x548774`, gms_v83 `@0x5657e7`, gms_v87
  `@0x590568`, gms_v95 `@0x55ab90` (PDB-backed), jms_v185 `@0x5b0597`
  (gms_v84 `@0x5724ee` byte-identical to v83) — **identical structure in all
  five**: `Decode1` team (team color selector: `!=0` ⇒ MAPLE_BLUE, `0` ⇒
  MAPLE_RED), `DecodeStr` name (defeated character name), `Decode1` lostCp
  (CP lost by the team; `<=0` ⇒ "no CP lost" message variant) — all three
  reads happen unconditionally before any branching; everything after is
  StringPool lookups + `CHATLOG_ADD`, never wire data.
- `OnShowMemberOutMsg` — gms_v79 `@0x5488ef`, gms_v83 `@0x565962`, gms_v87
  `@0x5906e3`, gms_v95 `@0x55ad80` (PDB-backed), jms_v185 `@0x5b070f`
  (gms_v84 `@0x572669` byte-identical to v83) — **identical structure in all
  five**: `Decode1` leader (`==6` ⇒ "leader quit, X appointed" message
  variant — the second `Decode1` call itself is unconditional, evaluated as
  part of an `if` condition, so no read is ever skipped), `Decode1` team
  (same color selector), `DecodeStr` name (quitting character name).

Like DEFECT-7/8, this was **not a false pass** — the pre-existing atlas
`MonsterCarnivalDied` (`team byte, name string, lostCp byte`) and
`MonsterCarnivalLeave` (`leader byte, team byte, name string`) codecs already
modelled both shapes correctly, and v83/v84/v87/v95/jms were already ✅ for
both ops with correct 3-call exports. The only gap was **v79**: both export
entries held `unresolved` (function not found at export time), and neither
opcode `0x110` nor `0x111` was routed in `template_gms_79_1.json`. Spliced
the real read order into the v79 export for both ops, re-pinned both evidence
records, regenerated both reports (selective revert — the root regen command
still churns ~217 unrelated files; reverted everything except the two
`MonsterCarnival{Died,Leave}` report pairs and `git clean`ed the recreated
strays), and routed `0x110`/`0x111` in the v79 template between `0x10F`
MonsterCarnivalMessage and `0x112` MonsterCarnivalResult, plus two registry
entries. Added `TestMonsterCarnivalDiedByteOutputV79` and
`TestMonsterCarnivalLeaveByteOutputV79`. Both v79 cells verify ✅ (`matrix
--check` clean); all five other versions remain ✅ for both ops.

## DEFECT-10: MtsOperation (MTS_OPERATION) — route-only gap, dispatcher family already correct — **FIXED**

**Resolution (task-181):** `CITC::OnNormalItemResult` is the mode-prefix
dispatcher already built discrete-per-mode (task-096) in
`libs/atlas-packet/field/clientbound/mts_operation.go` (35 structs) +
`libs/atlas-packet/field/mts_operation_body.go`, verified ✅ for
gms_v83/v84/v87/v95 with `docs/packets/dispatchers/mts_operation.yaml`
documenting the mode table as version-stable. **v79 and jms_v185 were the only
gaps** (⬜, unrouted). Per `docs/packets/DISPATCHER_FAMILY.md` this claim of
version-stability had to be verified, not assumed — the dispatcher itself
(`CITC::OnNormalItemResult @0x57f4a7`) plus **all 35 per-arm sub-handlers**
were re-decompiled live in the v79 IDB (`88dfa464`) and diffed field-by-field
against the existing v83-cited body doc-comments:

- Dispatcher case-label set: identical — `0x15,0x16,0x17,0x18,0x1D-0x38,0x3C,
  0x3D,0x3E` (35 cases), same target sub-handler names, same `Decode1(mode)`
  discriminator.
- Every arm's read order matched byte-for-byte with zero divergence: the 7
  list/item-blob arms (`GetItcListDone`/`GetSearchItcListDone`/
  `GetUserPurchaseItemDone`/`GetUserSaleItemDone`/`LoadWishSaleListDone` +
  the two conditional-tail arms `RegisterSaleEntryFailed` (reason==0x48 gates
  a trailing `Decode2`) and `SuccessBidInfo` (itemId>0 gates a trailing
  `Decode4`+`DecodeBuffer(8)`)), the 2 "TwoInts" arms
  (`MoveItcPurchaseItemLtoSDone`, `NotifyCancelWishResult`), the "Reason"
  arms (single `Decode1` reason byte), and the 24 notice-only "Empty" arms
  (mode byte only, no further `CInPacket::Decode*`) — all reproduced exactly
  in v79. No per-arm fix was needed.
- CITC::OnPacket (`0x57f39b`) case 324 (`0x144`) → `CITC::OnNormalItemResult`,
  confirmed via live decompile (322=`0x142`=MtsChargeParamResult,
  323=`0x143`=MtsOperation2, 324=`0x144`=MtsOperation — sequential, matches
  the seed template's existing 0x142/0x143 routing).

Spliced all 35 `CITC::OnNormalItemResult#<Arm>` export entries in
`docs/packets/ida-exports/gms_v79.json` (previously `unresolved`) with the
v79 addresses + the (now-confirmed-identical) call lists, routed `0x144` in
`template_gms_79_1.json` (copying the version-stable `operations` /
`noticeFailReasons` / `processStatusCodes` config tables verbatim — identical
across v83/v84/v87/v95), added the `MTS_OPERATION` registry entry in
`docs/packets/registry/gms_v79.yaml`, appended a `gms_v79` marker line to each
of the 35 `packet-audit:verify` blocks in `mts_operation_test.go` (same golden
bytes — the mode values are numerically identical across versions, no new
test functions needed), and pinned all 35 evidence records
(`evidence pin --category TIER1-FIXTURE`). Selective report regen kept only
the 35 `FieldMtsResult*` report pairs under `docs/packets/audits/gms_v79/`
(reverted the rest + `git clean`ed the recreated strays). All 35 v79 cells
verify ✅; the `MTS_OPERATION` op-row is now ✅ for
v79/v83/v84/v87/v95 (`matrix --check` clean). The previously-documented
"BLOCKER: can't cleanly regenerate the v79 matrix reports" section below is
now **stale** — the cited `CCashShop::TrySendQueryCashRequest` export defect
is no longer present in `gms_v79.json` (already fixed by an earlier pass on
this task) and the selective regen ran clean with no conflict reintroduction.

jms_v185 has **no CITC op at all** (registry-absent per the family yaml — the
JMS client build never shipped the MTS feature), so jms_v185 stays ⬜
(version-absent, not a gap) and was correctly out of scope for this bring-up.

## Remaining divergent writers (proven recipe: RE across IDBs → re-model codec →
correct exports if their read-order is wrong → re-pin evidence → selective
per-version report regen → route v79). SnowballState (DEFECT-1),
AriantArenaUserScore (DEFECT-2), ContiMove (DEFECT-3), TournamentSetPrize
(DEFECT-4), Tournament (DEFECT-5), TournamentMatchTable (DEFECT-6),
MonsterCarnivalStart (DEFECT-7), MonsterCarnivalSummon + MonsterCarnivalMessage
(DEFECT-8), and MonsterCarnivalDied + MonsterCarnivalLeave (DEFECT-9) are the
completed exemplars — DEFECT-2 confirmed the count-loop export convention (no
splice needed except for the previously-unresolved v79 entry), DEFECT-3
confirmed a genuine state-gated conditional-field false-pass spanning ALL
previously-✅ versions, DEFECT-4 confirmed a flag-gated conditional-field
false-pass where the non-v79 exports were already correct (only the codec and
the v79 export needed fixing), DEFECT-5 confirmed a flat-invariant-length
false-pass (no gate needed at all — the branch determines *meaning*, never
byte count), DEFECT-6 confirmed an empty-stub false-pass whose true body lives
behind a helper/ctor indirection the handler itself never shows, DEFECT-7
confirmed the inverse case: a codec that was already right, where the only
defect was an unrouted v79 opcode + an unresolved v79 export, DEFECT-8
confirmed the same inverse case for a 2-way arg-dispatched (not
opcode-dispatched) function sharing one fname across two ops, and DEFECT-9
confirmed the same inverse case again for two independently opcode-dispatched
handlers (no shared fname, no arg-dispatch) in the same family.
- Large: MtsOperation — `CITC::OnNormalItemResult` 35-arm dispatcher family
  (`DISPATCHER_FAMILY.md`).

## BLOCKER: can't cleanly regenerate the v79 matrix reports

Promoting a corrected codec needs its audit report regenerated
(`docs/packets/audits/gms_v79/Field*.json`). The root regen command
(`go run ./tools/packet-audit -csv-clientbound … -csv-serverbound … -template …
-ida-source docs/packets/ida-exports/gms_v79.json`) **exits 1** on a pre-existing
serverbound export defect:

```
resolve CCashShop::TrySendQueryCashRequest: call 0 (…): unknown primitive "COutPacket"
```

and, in the same run, rewrites 200+ unrelated v79 reports (non-deterministic vs
the committed set) and reintroduces a `MOB_SKILL_DELAY_END` conflict + the
`FieldAriantScore` stray. So a corrected SnowballState codec cannot be promoted
to a matching, `--check`-clean report without first fixing that pipeline error
(or learning the narrower per-packet regen recipe the batch-1 agent used).

**Recommended sequence:** (1) fix the `CCashShop::TrySendQueryCashRequest`
serverbound export/resolution so `packet-audit` regen is clean and
deterministic; (2) re-apply the SnowballState codec fix (spec above) + regen its
6 cells + add gms_v79; (3) do AriantArenaUserScore / TournamentMatchTable the
same way. Flag all three as false-pass regressions found on `main` in the PR.
