# task-112 — Findings & per-version derivations

Backfilled/wired the dispatcher-family `operations` mode tables missing from the
v87/v95/jms seed templates (root cause: later version bring-ups copied opcode/handler
wiring but dropped the per-version operations tables). Mode bytes are **version-dependent**
and were re-derived from each version's IDA switch (never copied across versions). Each
value cited to a decompile case; the controller independently spot-checked the riskiest
piece in every version (UiOpen reorder) against named `CUI*` constructors.

## Class A — operations table added to an already-wired writer (opcode unchanged)
| family | dispatcher fname | v87 | v95 | jms |
|---|---|---|---|---|
| FameResponse | `CWvsContext::OnGivePopularityResult` | 0x26, ≡v83 | 0x25, ≡v83 | 0x24, ≡v83 |
| HiredMerchantOperation | `CWvsContext::OnEntrustedShopCheckResult` | 0x32, ≡v83 | 0x31, ≡v83 | 0x2F, ≡v83 |
| UiOpen | `CWvsContext::UI_Open` (REORDERED) | 0xE9, 21 keys | 0xFB, 20 keys | 0xE5, 21 keys |

**UiOpen** is the only non-trivial mapping — the UI-type enum is reordered per version.
Each case verified by its constructed `CUI*` singleton / `CUIWnd(this, <id>, …)` type-id:
- GUILD_BBS: **34** (v87/jms) vs **39** (v95). Controller-confirmed: v87 case 34 → `CUIGuildBBS`,
  v95 case 39 → `CUIGuildBBS::CUIGuildBBS`, jms case 34 → CUIWnd id 34.
- MONSTER_CARNIVAL 17, ENERGY_BAR 19, PARTY_SEARCH 21, ITEM_MAKER 22, RANKING 25, FAMILY 26,
  FAMILY_PEDIGREE 27, OPERATOR_BOARD 28/29, MEDAL_MEDAL_QUEST 30, WEB_EVENT 31, SKILLS_EX 32
  (consistent across versions). Low cases ITEM 0 … CHARACTER_INFORMATION 10 stable.
- **MONSTER_BOOK divergence**: present at case 9 in v87 and **jms** (controller-confirmed jms
  case 9 → `TSingleton<CUIMonsterBook>`), but **absent in v95** (no case 9, no `CUIMonsterBook`
  ctor in the v95 switch) → correctly **omitted** from the v95 table.

## Class B — new clientbound writer wired (opcode + table; no validator — these are writers)
| family | dispatcher | v87 op | v95 op | jms op | mode bytes |
|---|---|---|---|---|---|
| WorldMessage | `CWvsContext::OnBroadcastMsg` | 0x46 (SERVERMESSAGE 70) | 0x47 (71) | 0x3E (62) | ≡v83 (NOTICE 0 … UNKNOWN_8 18) |
| PetActivated | `CUser::OnPetPacket`→`OnPetActivated` | 0xB4 (SPAWN_PET 180) | 0xC6 (198) | 0xAD (173) | NORMAL 0 … UNKNOWN_2 4 (server→client enum) |
| NoteOperation | `CWvsContext::OnMemoResult` | (already present) | (already present) | 0x26 (MEMO_RESULT 38) | SHOW 3, SEND_SUCCESS 4, SEND_ERROR 5, REFRESH 7 |

Opcodes sourced from each version's packet-audit registry (`docs/packets/registry/<v>.yaml`)
and cross-checked against the IDA dispatch (`CWvsContext::OnPacket` case / `OnPetPacket` nType
branch). PetActivated cross-check: v83 SPAWN_PET=0xA8 matches the gms_83 PetActivated writer
opcode exactly. (PetActivated opcode collides numerically with a serverbound handler op in
some versions — harmless: `socket.writers` and `socket.handlers` are separate namespaces.)

## Blocked / skipped (not produced — justified)
- **GuildBBS (jms)** — BLOCKED, genuine external blocker: the GUILD_BBS clientbound packet
  **does not exist in jms v185**. Evidence: no `CUIGuildBBS::OnGuildBBSPacket` symbol; no BBS
  case in `CWvsContext::OnPacket`; no GUILD_BBS op in `jms_v185.yaml`; clientbound opcode slots
  contiguous with no BBS gap. jms routes guild boards through the external web board. Wiring a
  guessed opcode would crash the client, so left unwired. (v87/v95 already had GuildBBS.)
- **MtsOperation (jms)** — SKIPPED per scope: the MTS feature is blocked/planned
  (`project_mts_feature_planned`), so the jms writer is correctly absent. v87/v95 already had it.

## Verification
- All 3 templates valid JSON; `atlas-configurations` and `atlas-channel` build green.
- No Go/TS changed — seed-template data only. No automated byte-fixture exists for these
  writers, so correctness rests on IDA citation + the controller spot-checks above.
- **Runtime note**: seed templates apply at tenant creation; existing v87/v95/jms tenants need
  a config refresh to pick up the new tables/writers (`bug_new_opcodes_not_in_live_tenant_config`).

## Plan-Adherence Review (independent audit, 2026-06-24)

**Verdict: FAITHFULLY IMPLEMENTED.** Every in-scope family was wired/backfilled in
every in-scope version with the gms_83 key set (modulo justified per-version byte shifts
and the one justified MONSTER_BOOK omission); the diff touches only the 3 templates + docs;
no other writer was disturbed; both builds are green; all 3 templates parse as JSON. No
Critical or Important issues. Three Minor notes below.

### 1. Coverage vs plan — PASS
Verified against `template_gms_83_1.json` key sets. Post-change tables (sorted by byte):

| family | v87 | v95 | jms |
|---|---|---|---|
| FameResponse | op 0x26, 7 keys ≡v83 | op 0x25, 7 keys ≡v83 | op 0x24, 7 keys ≡v83 |
| HiredMerchantOperation | op 0x32, 11 keys ≡v83 | op 0x31, 11 keys ≡v83 | op 0x2F, 11 keys ≡v83 |
| UiOpen | op 0xE9, 21 keys | op 0xFB, 20 keys (MONSTER_BOOK omitted) | op 0xE5, 21 keys |
| WorldMessage (new) | op 0x46, 19 keys ≡v83 | op 0x47, 19 keys ≡v83 | op 0x3E, 19 keys ≡v83 |
| PetActivated (new) | op 0xB4, 5 keys ≡v83 | op 0xC6, 5 keys ≡v83 | op 0xAD, 5 keys ≡v83 |
| NoteOperation | (pre-existing 0x29) | (pre-existing 0x28) | op 0x26 (NEW), 4 keys ≡v83 |
| GuildBBS | (pre-existing 0x3B) | (pre-existing 0x3B) | BLOCKED — not wired |

- UiOpen key divergences match findings.md exactly: GUILD_BBS=34 (v87/jms) vs 39 (v95);
  MONSTER_BOOK present at 9 in v87/jms, absent in v95 (20-key table). MONSTER_CARNIVAL 17 /
  ENERGY_BAR 19 / PARTY_SEARCH 21 / ITEM_MAKER 22 / RANKING 25 / FAMILY 26 / FAMILY_PEDIGREE 27
  / OPERATOR_BOARD 28 / OPERATOR_BOARD_STATE 29 / MEDAL_MEDAL_QUEST 30 / WEB_EVENT 31 /
  SKILLS_EX 32 consistent across all three (a uniform −1 shift from gms_83's 18/20/22/23/26…
  for the high block, plus the GUILD_BBS relocation). Internally consistent.

### 2. No regressions / no scope creep — PASS
`git diff --stat main..HEAD` = only the 3 templates (+ plan.md/findings.md). The only writer
names appearing in the diff are the 6 in-scope families. Removed lines are exactly the Class A
trio (FameResponse/HiredMerchant/UiOpen ×3) where the closing `}` became `,` to append the
options block — opcodes preserved (0x26/0x32/0xE9 v87; 0x25/0x31/0xFB v95; 0x24/0x2F/0xE5 jms),
no opcode changed. Pre-existing families NoteOperation/GuildBBS/MtsOperation on v87/v95 were
NOT disturbed (confirmed against `git show main:` — they already existed there).

### 3. Class B opcode soundness — PASS
WorldMessage / PetActivated / NoteOperation got new `socket.writers` entries with a clientbound
opcode and NO validator field. Each new opcode is unique within its template's `socket.writers`
(normalized to int: WorldMessage 0x46/0x47/0x3E, PetActivated 0xB4/0xC6/0xAD, NoteOperation jms
0x26 — none collide with another writer). The only duplicate writer-opcodes in any template
(login-phase 0x00 Auth*, server-list 0x0A/0x02, portal 0x45/0x3D SpawnPortal/RemoveTownDoor)
are pre-existing on main and in separate socket phases — not introduced here, do not involve the
new writers. Writer-name strings match real Go writer constants exactly:
`note/clientbound/operation.go:12 NoteOperationWriter="NoteOperation"`,
`chat/clientbound/world_message.go:12 WorldMessageWriter="WorldMessage"`,
`pet/clientbound/activated.go:12 PetActivatedWriter="PetActivated"`,
`guild/clientbound/bbs.go:12 GuildBBSWriter="GuildBBS"`.

### 4. Justified exclusions — COHERENT
- **GuildBBS (jms)**: confirmed absent from jms `socket.writers`; the blocked-justification
  (no jms BBS clientbound packet) is coherent. Note the distinct UiOpen `GUILD_BBS:34` key IS
  present in jms — correct and non-contradictory: that is the UI-window-open sub-op (opening the
  board), a different packet from the GuildBBS *result* writer. Not a silent drop.
- **MtsOperation (jms)**: confirmed absent from jms `socket.writers`; present on v87 (0x171) /
  v95 (0x19C). Matches the planned MTS-blocked skip. Not a silent drop.

### 5. Builds — PASS
`( cd services/atlas-configurations/atlas.com/configurations && go build ./... )` → OK.
`( cd services/atlas-channel/atlas.com/channel && go build ./... )` → OK.
All 3 templates pass `python3 -m json.tool`.

### 6. Honesty check — MINOR NOTES
- **Minor (documentation, not a defect):** findings.md §"Class B" describes NoteOperation jms as
  carrying only the `operations` block, but the implementation correctly ALSO ported the gms_83
  `errors` block (RECEIVER_ONLINE:0/RECEIVER_UNKNOWN:1/RECEIVER_INBOX_FULL:2). This is a faithful
  full-options copy of the gms_83 NoteOperation entry (which has both blocks) — more complete than
  the findings text implies, so a doc undercount, not an error.
- **Minor (unverifiable here):** the mode bytes and Class B opcodes are IDA-derived with no
  automated byte-fixture (the plan acknowledges this — IDA citation + build are the only gates).
  This audit independently verified key-set coverage, opcode uniqueness, writer-name validity, JSON,
  builds, and findings↔template internal consistency, but did NOT re-run the IDA decompiles. The
  UiOpen GUILD_BBS 34/39 split, the v95 MONSTER_BOOK omission, and the WorldMessage/PetActivated
  opcodes rest on the implementers' cited (controller-spot-checked) decompile cases and were not
  independently re-derived from the IDBs in this review. No internal inconsistency was found
  (every asserted byte matches the committed template; no key is mapped to a byte the findings
  call absent).
- **Minor (consistency nit):** findings.md §"Class A" table lists FameResponse jms as `0x24`
  and HiredMerchant jms as `0x2F` — both match the template. No discrepancy.

**Recommendation: READY — no required fixes.** Optional: amend findings.md §Class B to note the
NoteOperation jms `errors` block was also ported.
