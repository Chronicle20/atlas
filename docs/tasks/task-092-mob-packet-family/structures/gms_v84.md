# gms_v84 — MOB/MONSTER byte layouts (Stage 1 harvest)

IDB: `GMS_v84.1_U_DEVM.exe` (v84_1), port 13341. Harvested 2026-06-13.

## TL;DR — v84 is byte-identical to v83; its IDB does NOT name this family

task-083 established v84 ≡ v83 byte-wise (and v84 takes the v83 codec path via the
`MajorAtLeast(87)` gate — v84 < 87). **The v84 IDB symbolizes the MOB/MONSTER family as
unnamed `sub_XXXX` functions** (or custom non-mangled labels like
`CMob__Update_ctrl_send_0xC4_0xC5friendlyDmg_0xC8`), so the demangled-fname export
harvest resolves **only 1 of 25** in-scope roster fnames:

```
export: 1 resolved, 0 descended-helper, 24 unresolved
```

The single resolved fname is `CMobPool::OnMobCrcKeyChanged` (the only MSVC-mangled symbol
in the family). It was merged into `docs/packets/ida-exports/gms_v84.json` (439 → 440 keys,
absent+resolved-only — the 24 unresolved stubs were filtered out, NOT merged).

**Consequence:** v84 cannot be fname-pinned for the other 24 ops from this IDB. Stage 2
should treat v84 as v83-equivalent (same wire layout, v83 codec path) rather than expecting
distinct v84 evidence. Deriving named handlers for v84 would require a full sub_XXXX renaming
campaign in the IDB, which is out of Stage-1 scope and unnecessary given the equivalence.

## Registry state (NO edits needed this version)

- MOB_SPEAKING / INC_MOB_CHARGE_COUNT / MOB_SKILL_DELAY fnames already correct
  (`CMob::OnMobSpeaking` / `CMob::OnIncMobChargeCount` / `CMob::OnMobSkillDelay`),
  fixed by an earlier discover pass. Verified, not changed.
- MONSTER_BOOK_COVER (sb): fname stays `""` — **the send-site is unnamed in the v84 IDB**
  (no `SetMonsterBookCover`/`MonsterBookCover`/`BookCover` symbol). v83-equivalent
  (CUserLocal::SetMonsterBookCover); the v84 address is an unnamed sub. Left empty rather
  than fabricate; Stage 2 inherits the v83 layout.

## Dispatcher evidence — CMobPool::OnMobPacket @ 0x68FEF7

The v84 mob-cluster dispatcher (`switch(a2)` on the opcode). Most targets are unnamed subs;
a few carry custom labels. This case→address map is the v84 IDB evidence for the cluster:

| case | target address | label (if any) |
|---|---|---|
| 245 | 0x6820EA | CMob::OnMove |
| 246 | 0x68253D | CMob::OnCtrlAck |
| 248 | 0x682603 | CMob::OnStatSet |
| 249 | 0x682726 | CMob::OnStatReset |
| 250 | 0x682802 | CMob__OnSuspendReset_recv_0xFA |
| 251 | 0x682977 | sub_682977 |
| 252 | 0x6829C4 | CMob::OnDamaged |
| 253 | 0x683BE9 | sub_683BE9 |
| 256 | 0x68393B | CMob::OnHPIndicator |
| 257 | 0x6839BB | sub_6839BB |
| 258 | 0x683C9F | sub_683C9F |
| 259 | 0x687743 | sub_687743 |
| 260 | 0x687655 | sub_687655 |
| 261 | 0x688524 | sub_688524 |
| 262 | 0x68749A | sub_68749A |

Note: the raw dispatcher case labels do NOT line up 1:1 with the Atlas registry opcodes
(the v84 registry opcodes were reconciled to a different scheme in task-085 — e.g. registry
MOB_AFFECTED=245 vs dispatcher case 245→OnMove). Because v84 ≡ v83, Stage 2 should transcribe
the **v83 layouts** (`gms_v83.md`) for v84 rather than re-deriving from these unnamed subs.

## Byte layouts (only the resolved fname)

### CMobPool::OnMobCrcKeyChanged
- **address:** 0x690354
- **calls (1):** `Decode4`

(All other in-scope ops: unresolved in this IDB — see TL;DR. Layouts = v83-identical.)

## Stage-2 blockers (v84 IDB naming)

| scope | status |
|---|---|
| 24 in-scope MOB/MONSTER ops (all except MOB_CRC_KEY_CHANGED) | UNRESOLVED in v84 IDB (unnamed sub_XXXX). Treat as v83-equivalent; do NOT pin against a fabricated v84 fname. |
| MONSTER_BOOK_COVER (sb) fname | empty; v84 send-site unnamed. Inherit v83. |
| TOUCH_MONSTER_ATTACK / MOB_BANISH_PLAYER / MOB_TIME_BOMB_END | same csv-import conceptual fnames as v83; also unnamed here. |
