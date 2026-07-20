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
  (`sub_750E40`). Needs the real body reversed.

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

## Remaining divergent writers (proven recipe: RE across IDBs → re-model codec →
correct exports if their read-order is wrong → re-pin evidence → selective
per-version report regen → route v79). SnowballState (DEFECT-1),
AriantArenaUserScore (DEFECT-2), ContiMove (DEFECT-3), TournamentSetPrize
(DEFECT-4), and Tournament (DEFECT-5) are the completed exemplars — DEFECT-2
confirmed the count-loop export convention (no splice needed except for the
previously-unresolved v79 entry), DEFECT-3 confirmed a genuine state-gated
conditional-field false-pass spanning ALL previously-✅ versions, DEFECT-4
confirmed a flag-gated conditional-field false-pass where the non-v79
exports were already correct (only the codec and the v79 export needed
fixing), DEFECT-5 confirmed a flat-invariant-length false-pass (no gate
needed at all — the branch determines *meaning*, never byte count).
- Medium: TournamentMatchTable, MonsterCarnival
  Start/Summon/Message/Died/Leave (variable str/loop bodies).
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
