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
