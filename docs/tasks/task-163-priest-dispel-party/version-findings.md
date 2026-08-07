# task-163 — Per-version audit of the Dispel (2311001) serverbound skill-use decode

**Status:** read-only evidence audit. No production code was changed by this task.

**Why:** `libs/atlas-packet/model/skill_usage_info.go` gates three optional field
groups on hard-coded, version-blind raw wire-id lists (`isAntiRepeatBuffSkill`,
`isPartyBuff`, `isMobAffectingBuff`), each carrying a `// TODO this is not all
inclusive` comment. `2311001` is a member of all three. Those lists were derived
from the GMS v83 client only. If the membership disagrees with a given client,
every field after the mismatch reads at the wrong offset, the bitmap decodes to
garbage/0, `SelectPartyMembersInMap` returns nil, and Dispel silently degrades to
caster-only with no error. That failure shipped twice already (task-111 Bishop
Resurrection 2321006; task-155/PR#1136 Buccaneer Time Leap 5121010).

---

## Step 1 — The current decoder's assumed layout for 2311001

From `libs/atlas-packet/model/skill_usage_info.go` (`SkillUsageInfo.Decode`),
with `2311001` a member of `isAntiRepeatBuffSkill`, `isPartyBuff` **and**
`isMobAffectingBuff`, and the `PriestDispelId` special-case inside the
`isPartyBuff` arm:

| # | Field | Width | Gate in the decoder |
|---|---|---|---|
| 1 | `updateTime` | 4 | unconditional |
| 2 | `skillId` | 4 | unconditional |
| 3 | `skillLevel` | 1 | unconditional |
| 4 | `castX` | 2 | `isAntiRepeatBuffSkill(2311001)` → true |
| 5 | `castY` | 2 | `isAntiRepeatBuffSkill(2311001)` → true |
| 6 | `affectedPartyMemberBitmap` | 1 | `isPartyBuff(2311001)` → true |
| 7 | `delay` | 2 | `skillId == PriestDispelId` (inside the party arm) |
| 8 | `nMobCount` | 1 | `isMobAffectingBuff(2311001)` → true |
| 9 | `affectedMobIds[N]` | 4 × N | loop over `nMobCount` |
| 10 | `delay` | 2 | tail of the mob arm — **overwrites** field 7 |

Note the **double `delay` read** (fields 7 and 10). Both writes land in the same
`m.delay`; the second wins. This is the hypothesis every version below is tested
against.

`spiritJavelinItemId` is gated on `skill.NightLordShadowStarsId` (4121006) and is
absent for 2311001 — correct on every client read below.

---

## Step 2 — Per-version audit

### How each version was read

No checked-in export exists for this packet on **any** version. The op is
`SPECIAL_MOVE` (`docs/packets/audits/status.json`, row 260, `direction:
serverbound`), whose `fnames` list includes `CUserLocal::SendSkillUseRequest`.
Its matrix cells are `n-a` (gms_v48) or `incomplete` / "no audit report"
(all others), and no `SPECIAL_MOVE`/`SendSkillUseRequest` evidence file exists
under any `docs/packets/audits/<version>/` directory. **The export avenue is
therefore unavailable for every version** and each row below was derived from the
version's IDB via ida-pro-mcp.

The client-side contract, as read on v83 and confirmed structurally on every
readable version, is:

* `castX`/`castY` are gated on `is_antirepeat_buff_skill(skillId)` — a hard-coded
  predicate function inside the client.
* The bitmap byte is gated on the **runtime** argument `dwAffectedMemberBitmap != 0`,
  not on a skill-id list. That argument comes from `CUserLocal::FindParty`, which
  returns `0x80` (self bit) when the caster is not in a party and a 6-slot mask
  otherwise — i.e. non-zero for 2311001 in both cases.
* The `delay` after the bitmap is gated on `skillId == 2311001` (literally, in
  the client).
* The mob-count byte + id array are gated on the **runtime** argument
  `nMobCount >= 0`.
* Both runtime arguments are supplied by `CUserLocal::DoActiveSkill_StatChange`,
  which sets `Party = 0` unless `dwTargetFlag & 2` and `HitMobInRect = -1` unless
  `dwTargetFlag & 4`. `dwTargetFlag` is a literal pushed by the per-skill
  dispatch arm in `CUserLocal::DoActiveSkill`. **For 2311001 the pushed value is
  `6` (= 0x2 | 0x4) on every version where the arm was located** — so both the
  bitmap and the mob list are present.

### Findings table

| Version | castX/castY written? | bitmap written? | mob list written? | Matches current decoder? | Evidence |
|---|---|---|---|---|---|
| gms_48 | **NO** | yes (`if (a4)`) | unverified — dispatch arm not located | **NO — diverges** (but unreachable, see below) | IDB `GMS_v48_1_DEVM.exe.i64`, `CUserLocal::SendSkillUseRequest` @0x6AFA91 (mangled name pre-existing in IDB). Full decompile shows **no `is_antirepeat_buff_skill` block at all**; encode order is `Encode4(updateTime) Encode4(skillId) Encode1(slv) [4121006→Encode4] [a4→Encode1 [2311001→Encode2]] [a5>=0→Encode1+Encode4×N] Encode2`. Opcode `COutPacket(…, 70)` = 0x46. The 2311001 site is at 0x6AFC30 (`cmp dword ptr [ebx], 234359h`). Whole-binary scan for the 0x234359 immediate returned exactly 2 sites (0x6AC0DB, 0x6AFC32) — the `DoActiveSkill` `push`-flag arm is not among them. |
| gms_61 | **NO** | yes (`if (a4)`) | yes — `dwTargetFlag = 6` | **NO — diverges** | IDB `GMS_v61.1_U_DEVM.exe.i64`, `CUserLocal::SendSkillUseRequest` @0x7BA213 (mangled name pre-existing). Decompile shows **no `is_antirepeat_buff_skill` block**; same order as gms_48. Opcode `COutPacket(…, 83)` = 0x53. 2311001 site @0x7BA3BB. Dispatch arm: `mov eax, 234359h; cmp ebx, eax` @0x7B6625 → `jz loc_7B66E1` → `push 6` @0x7B66E1. |
| gms_72 | yes | yes | yes — `dwTargetFlag = 6` | **yes** | IDB `GMS_v72.1_U_DEVM.exe.i64`, `CUserLocal::SendSkillUseRequest` @0x8774D9 (was `sub_8774D9`; **named by this audit**). Has the antirepeat block (`Encode2(x); Encode2(y)`). `is_antirepeat_buff_skill` @0x877789 (was `sub_877789`; **named by this audit**) — decompile contains `a1 == 2311001 → return 1`. Opcode `COutPacket(…, 90)` = 0x5A. Dispatch arm @0x8728F8 (`cmp ebx, eax` where eax = 0x234359, derived from `add eax, 0FFFFFFFCh` on base 0x23435D and confirmed by the sibling `sub eax, 23435Ah` at 0x872936) → `jz loc_87292D` → `push 6`. |
| gms_79 | yes | yes | yes — `dwTargetFlag = 6` | **yes** | IDB `GMS_v79_1_DEVM.exe.i64`, `CUserLocal::SendSkillUseRequest` @0x8C4007 (mangled name pre-existing). Has the antirepeat block. `is_antirepeat_buff_skill` @0x8C42BD (was `sub_8C42BD`; **named by this audit**) — contains `a1 == 2311001 → return 1`. Opcode `COutPacket(…, 89)` = 0x59. 2311001 site @0x8C4233. Dispatch: `mov eax, 234359h; cmp ebx, eax` @0x8BF19D → `jz short loc_8BF225` → `push 6` @0x8BF225. |
| gms_83 | yes | yes | yes — `dwTargetFlag = 6` | **yes** (reference) | IDB `MapleStory_dump.exe.i64`, `CUserLocal::SendSkillUseRequest` @0x96D399. Encode order: `Encode4(update_time) Encode4(skillId) Encode1(nSLV) [is_antirepeat_buff_skill→Encode2(x) Encode2(y)] [4121006→Encode4] [dwAffectedMemberBitmap→Encode1 [2311001→Encode2(a2[0])]] [nMobCount>=0→Encode1 + Encode4×N] Encode2(a2[0])`. Opcode `COutPacket(…, 0x5B)`. `is_antirepeat_buff_skill` @0x96D6CA contains `a1 == 2311001 → return 1`. `CUserLocal::FindParty` @0x96DB3F returns 0x80 when solo, else a 6-slot mask. `CUserLocal::DoActiveSkill_StatChange` @0x969E21 gates `Party` on `dwTargetFlag & 2` and `HitMobInRect` on `dwTargetFlag & 4`. Dispatch arm: `sub eax, 234359h; jz short loc_967EB7` @0x967E88 → `push 6` @0x967EB7. |
| gms_84 | yes | yes | yes — `dwTargetFlag = 6` | **yes** | IDB `GMS_v84.1_U_DEVM.i64`, `CUserLocal::SendSkillUseRequest` @0x9AD149 (was `sub_9AD149`; **named by this audit**). Has the antirepeat block. `is_antirepeat_buff_skill` @0x9AD4E4 (was `sub_9AD4E4`; **named by this audit**) — contains `a1 == 2311001 → return 1`. Opcode `COutPacket(…, 91)` = 0x5B. 2311001 site @0x9AD45D. Dispatch chain in `sub_9A6142` (DoActiveSkill): `sub eax, 231C4Dh` @0x9A7008 then `sub eax, 270Ch` @0x9A7013 (2301005 + 9996 = 2311001) → `jz short loc_9A7042` → `push 6` @0x9A7042. |
| gms_87 | yes | yes | yes — `dwTargetFlag = 6` | **yes** | IDB `GMSv87_4GB.exe.i64`, `CUserLocal::SendSkillUseRequest` @0x9F1D61 (mangled name pre-existing). Has the antirepeat block. `is_antirepeat_buff_skill` @0x9F20FC (was `sub_9F20FC`; **named by this audit**) — contains `a1 == 2311001 → return 1`. Opcode `COutPacket(…, 0x5E)`. 2311001 site @0x9F2075. Dispatch chain in `CUserLocal::DoActiveSkill` @0x9EA7B9: `sub eax, 231C4Bh` @0x9EB7A2, `dec`, `dec`, `sub eax, 270Ch` @0x9EB7BB (2301003 + 2 + 9996 = 2311001) → `push 6` @0x9EB7C6. |
| gms_92 | yes | yes | yes — `dwTargetFlag = 6` | **yes** | IDB `GMS_v92_1_DEVM.exe.i64`, `CUserLocal::SendSkillUseRequest` @0x91D310 (was `sub_91D310`; **named by this audit**). Has the antirepeat block. `is_antirepeat_buff_skill` @0x919150 (was `sub_919150`; **named by this audit**) — contains `a1 == 2311001 → return 1`. Opcode `COutPacket(…, 0x66)`. 2311001 site @0x91D6EC. Dispatch: `cmp esi, 234359h` @0x92330F → `jz short loc_923345` → `push 6` @0x92334D. **No matrix column and no export exists for gms_92** — this row is IDB-only by design (see brief Step 2). |
| gms_95 | yes | yes | yes — `dwTargetFlag = 6` | **yes on the wire** — but see the effect-data finding below | IDB `GMS_v95.0_U_DEVM.exe.i64`, `CUserLocal::SendSkillUseRequest` @0x93E930 (mangled name pre-existing; PDB-backed, fully typed). Has the antirepeat block. `is_antirepeat_buff_skill` @0x939DC0 (pre-named) — `sub eax, 234359h; jz` @0x939E87. Opcode `COutPacket(…, 103)` = 0x67. 2311001 site @0x93ED3E. Dispatch: `cmp esi, 234359h` @0x9462D8 → `jz loc_94639B` → `push 6 ; dwTargetFlag` @0x94639B → `call CUserLocal::DoActiveSkill_StatChange` (IDA labels the operand `dwTargetFlag` from the PDB signature). |
| jms_185 | yes | yes | yes — `dwTargetFlag = 6` | **yes** | IDB `MapleStory_dump_SCY.exe.i64`, `CUserLocal::SendSkillUseRequest` @0xA3DE65 (mangled name pre-existing). Has the antirepeat block (`Encode2(x); Encode2(y)`). `is_antirepeat_buff_skill` @0xA3E223 (pre-named) — decompile contains `nSkillID == 2311001 \|\| nSkillID == 2311003 \|\| nSkillID == 2321000 → return 1`. Opcode `COutPacket(…, 0x56)` = 86, agreeing with both `status.json` (86) and `template_jms_185_1.json`. 2311001 delay site inside the bitmap arm confirmed. Dispatch arm in `CUserLocal::DoActiveSkill` @0xA35C3F: `mov eax, 234359h; cmp ebx, eax` @0xA36E14 → `jz short loc_A36E9B` → `push 6` @0xA36E9B → falls into the shared `loc_A376D4` call site → `CUserLocal::DoActiveSkill_StatChange` @0xA3934E (was `sub_A3934E`; **named by this audit**), which gates `dwAffectedMemberBitmap` on `dwTargetFlag & 2` and the mob list on `dwTargetFlag & 4` before calling `SendSkillUseRequest` — structurally identical to v83 @0x969E21. |

### Divergences found

**DIV-1 — gms_48 and gms_61 write no `castX`/`castY`.**
Neither client has an `is_antirepeat_buff_skill` gate in
`CUserLocal::SendSkillUseRequest` at all; the function jumps straight from
`Encode1(nSLV)` to the `4121006` / bitmap / mob-count blocks. The current decoder
unconditionally reads 4 bytes of `castX`/`castY` for 2311001 because
`isAntiRepeatBuffSkill` contains `PriestDispelId`. On these two versions that is a
**4-byte over-read**: the decoder would take the bitmap byte from what is
actually the low byte of the client's `delay`, and every subsequent field would
be misaligned.

Actual gms_48 / gms_61 layout for 2311001:

```
updateTime(4) skillId(4) slv(1) bitmap(1) delay(2) mobCount(1) mobIds(4×N) delay(2)
```

Reachability of DIV-1:

* **gms_61 is reachable and therefore genuinely broken.**
  `services/atlas-configurations/seed-data/templates/template_gms_61_1.json`
  routes `CharacterUseSkillHandle` at `opCode` `0x53` with `LoggedInValidator`,
  which matches the client's `COutPacket(…, 83)` exactly. A v61 Dispel cast
  reaches the handler and decodes wrong.
* **gms_48 is unreachable, so DIV-1 is latent there.**
  `template_gms_48_1.json` contains **zero** `CharacterUseSkillHandle` entries
  (82 `"handler"` entries total), and `status.json` records `SPECIAL_MOVE`
  `gms_v48` as `state: "n-a", opcode: -1`. No skill-use packet is routed on v48,
  so no per-skill handler fires. If v48 is ever wired up, DIV-1 applies.

**DIV-2 — gms_95 has no effect data for 2311001 (Step 4 finding, see below).**

### Observations (not divergences; recorded for whoever owns the matrix)

* `status.json` records `SPECIAL_MOVE` `gms_v61` opcode as `90` (0x5A). Both the
  v61 client (`COutPacket(…, 83)`) and `template_gms_61_1.json` (`0x53`) say
  **83**. The matrix value looks copied from the v72 column. Every other
  version's `status.json` opcode agrees with both the client and the template
  (v72 0x5A, v79 0x59, v83 0x5B, v84 0x5B, v87 0x5E, v95 0x67, jms 0x56).
* `status.json` records `gms_v48` `SPECIAL_MOVE` as `n-a` / opcode `-1`, but the
  v48 client does construct the packet with opcode `70` (0x46). "n-a" here
  reflects that Atlas does not route it, not that the client lacks it.

---

## Step 3 — gms_12 is unreachable

```
$ grep -c CharacterUseSkillHandle services/atlas-configurations/seed-data/templates/template_gms_12_1.json
0
$ grep -c '"handler"' services/atlas-configurations/seed-data/templates/template_gms_12_1.json
24
```

Both values match the expected `0` and `24`. No skill-use packet is routed on
gms_12, so no per-skill handler fires. **Dispel is unreachable on gms_12 and is
out of scope for this task.** (gms_12 also has no tenant in the live main
environment — the tenant list returned 10 tenants, none at major version 12.)

---

## Step 4 — Per-version `atlas-data` effect confirmation

Avenue used: live `atlas-main` namespace via the kubernetes MCP tools.
Tenant list from `GET /api/tenants` on `atlas-tenants-d64677746-4cpvf`; skill
data from `GET /api/data/skills/2311001` against the in-cluster `atlas-data`
service with that tenant's `TENANT_ID` / `REGION` / `MAJOR_VERSION` /
`MINOR_VERSION` headers.

| Version | Effect resolved? | maxLevel | prop curve (L1 → L20) |
|---|---|---|---|
| gms_48 | yes — 20 effect entries | 20 | 0.34 → 1.0 |
| gms_61 | yes — 20 | 20 | 0.34 → 1.0 |
| gms_72 | yes — 20 | 20 | 0.34 → 1.0 |
| gms_79 | yes — 20 | 20 | 0.34 → 1.0 |
| gms_83 | yes — 20 | 20 | 0.34 → 1.0 |
| gms_84 | yes — 20 | 20 | 0.34 → 1.0 |
| gms_87 | yes — 20 | 20 | 0.34 → 1.0 |
| gms_92 | yes — 20 | 20 | 0.34 → 1.0 |
| **gms_95** | **NO — `"effects": []`** | **0** | **none** |
| jms_185 | yes — 20 | 20 | 0.34 → 1.0 |
| gms_12 | n/a — no tenant exists | — | — |

Full observed prop curve on every version that has one (identical across all
nine):

```
0.34 0.38 0.42 0.46 0.50 0.54 0.58 0.62 0.66 0.70
0.73 0.76 0.79 0.82 0.85 0.88 0.91 0.94 0.97 1.00
```

This is the 34 %-at-L1 → 100 %-at-L20 curve, re-verified from live data rather
than cited from memory. `MPConsume` is 15 and `mobCount` is 6 on every level of
every version that resolves.

**DIV-2 — gms_95 resolves no effect for 2311001.** The live response is:

```json
{"data":{"type":"skills","id":"2311001","attributes":{"name":"Dispel",
"description":"Nullifies all enemy magic effects within the targeted area while
removing all abnormal conditions suffered by nearby party members.",
"action":true,"element":"NEUTRAL","animationTime":0,"maxLevel":0,"effects":[]}}}
```

Name and description are present, so the skill row exists — only the per-level
effect array is empty and `maxLevel` is 0. Per the brief, a version with no
effect aborts the cast in `CharacterUseSkillHandleFunc` before `UseSkill`, which
is a silent, version-specific no-op the Dispel handler cannot compensate for. The
v95 wire decode is correct; the cast simply never gets that far. This looks like
a WZ-ingest gap for the v95 tenant rather than a codec problem, but it was not
investigated further — this task is read-only.

---

## Step 5 — What could not be read, and why

| Item | Status | Reason |
|---|---|---|
| jms_185 client (all three columns) | **RESOLVED — now verified** | Originally unverified: the IDA session `b6864e54` (`MapleStory_dump_SCY.exe.i64`) timed out on every query across four attempts during the Step-2 pass, and again on two attempts during the task-4 retry. **On a seventh attempt the session responded** (`server_health` returned `auto_analysis_ready:true, hexrays_ready:true`) and the full read was completed — see the jms_185 row in the Step-2 findings table. jms_185 **matches the current decoder** on all three columns. No export exists for `SPECIAL_MOVE` on jms_v185, so this row is IDB-derived like every other. The decoder's JMS branch is now evidence-backed rather than retained as a no-wire-change default. |
| gms_48 `dwTargetFlag` for 2311001 (mob-list presence) | **unverified** | The `DoActiveSkill` dispatch arm was not located: a whole-binary scan for the 0x234359 immediate returned only 2 sites, neither of which is the flag-push arm, so the arm must reach 2311001 through a jump table or a relative-subtract chain that was not traced. The castX/castY divergence (DIV-1) is independently conclusive from the `SendSkillUseRequest` decompile, and gms_48 is unreachable anyway, so this gap does not change the row's verdict. |
| Checked-in export evidence for **any** version | **unavailable** | `SPECIAL_MOVE` has no evidence file in any `docs/packets/audits/<version>/` directory; all matrix cells are `incomplete` ("no audit report") or `n-a`. Every verified row above is IDB-derived, with the IDB, function name and address cited. |

**Scoreboard:** **10 of 10 audited versions verified from a client; 0 unverified.**
gms_12 recorded as unreachable rather than audited (Step 3). One sub-item remains
open — the gms_48 `dwTargetFlag` arm (row above) — which does not change that
version's verdict, since DIV-1 is independently conclusive from its
`SendSkillUseRequest` decompile and gms_48 routes no skill-use handler at all.

---

## Recommended follow-up (recorded, not implemented)

1. **Fix DIV-1.** `isAntiRepeatBuffSkill` must not return true for 2311001 on
   gms_48 and gms_61 — those clients have no such predicate at all, so the gate
   needs a version boundary at major ≥ 72 (verified true at v72 and above, false
   at v61 and v48). This is a live decode bug on the gms_61 tenant today.
   Whether the boundary is expressed via the version-aware resolver
   (`constants.For(region,major,minor)`) or a `MajorAtLeast` gate on the codec is
   a design decision for the owning task.
2. ~~**Resolve jms_185.**~~ **DONE** — completed on a later retry once the IDA
   session became responsive. jms_185 matches the current decoder; the gate's JMS
   branch is evidence-backed. See the Step-2 row and the Step-5 entry.
2a. **JMS-only decoder gap (new, out of scope for task-163).** The jms_185
   `SendSkillUseRequest` writes two conditional fields the decoder does not model:
   `if (skillId == 33101005) Encode1(1)` and `if (skillId == 33101004) Encode1(0)`,
   both sitting between the spirit-javelin field and the party bitmap. Neither is
   2311001, so Dispel is unaffected — but a cast of either skill on a JMS tenant
   would misalign every field after it, the same failure class as DIV-1. Recorded
   for whoever owns those skills.
3. **Investigate DIV-2.** The gms_95 tenant has no per-level effect data for
   2311001 in the live main baseline; likely a WZ-ingest gap. Dispel is a no-op
   on v95 regardless of codec correctness.
4. **Correct the matrix opcode for `SPECIAL_MOVE` / gms_v61** (90 → 83), which
   disagrees with both the client and the seed template.
