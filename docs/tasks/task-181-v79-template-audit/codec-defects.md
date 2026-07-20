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

## Remaining divergent writers (proven recipe: RE across IDBs → re-model codec →
correct exports if their read-order is wrong → re-pin evidence → selective
per-version report regen → route v79). SnowballState (DEFECT-1) and
AriantArenaUserScore (DEFECT-2) are the completed exemplars — the latter
confirming the count-loop export convention (no splice needed except for the
previously-unresolved v79 entry).
- Small: Tournament (v79 ≤2 bytes vs atlas 3), TournamentSetPrize (conditional
  int-pair), ContiMove (sub-dispatch — confirm emitted states).
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
