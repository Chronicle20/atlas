## Summary

Repairs the **gms_61 serverbound/clientbound opcode corruption** deferred from task-125. Every gms_61 packet was verified **exhaustively against the client itself** — no cross-version opcode comparison.

> task-125 (#1091) is **merged** — this branch is rebased onto `main` and targets `main` directly. main already carries task-125's 4 gms_61 item-use fixes; this adds the remaining repair on top, in strictly-ascending opcode order (per main's new `template-opcode-order-guard.sh`).

## Method — verify the packets, don't copy opcodes

An initial pass bounded corruption by diffing gms_61's template opcodes against gms_72's — a cross-version heuristic. That was **replaced by exhaustive per-packet verification** against the gms_61 client (`GMS_v61.1_U_DEVM.exe.i64`, session `965202bf`):

- **Clientbound (153 writers):** checked against the client's receive-dispatch — **232 `case N:` arms** enumerated across `CClientSocket::ProcessPacket`, `CLogin`/`CWvsContext`/`CField::OnPacket`, and 20 leaf sub-dispatchers.
- **Serverbound (114 handlers):** checked against **all 391 `COutPacket(N)` send sites** (`xrefs_to` the ctor, exhaustive).

The heuristic **missed two real corruptions** (`PartyInviteReject`, `WeddingAction`) that the send-site sweep caught — the reason for doing this by verification, not comparison. Per-handler `fn@addr` + verbatim `COutPacket`/`case` evidence is in `docs/tasks/task-184-gms61-opcode-repair/{clientbound,serverbound}-exhaustive.md`.

## Corrections (7)

**Serverbound handlers:**

| handler | was | now | evidence |
|---|---|---|---|
| `CharacterInventoryMoveHandle` | 0x46 | **0x42** | `sub_8315F8` `COutPacket(_, 66)` |
| `CharacterItemUseSummonBagHandle` | 0x4A | **0x46** | `SendMobSummonItemUseRequest`@0x831c83 → 70 (0x4A was the Bridle item; freed by InventoryMove→0x42) |
| `StorageOperationHandle` | 0x3A | **0x3C** | `CStoreBankDlg::SendGetAllRequest`@0x6754e1 → 0x3C |
| `OwlWarpHandle` | 0x3F | **0x4C** | `SendShopScannerItemUseRequest`@0x832680 → 0x4C |
| `PartyInviteRejectHandle` | 0x71 | **0xCB** | accept+decline branches `COutPacket(_, 203)`; nothing on 0x71 |
| `WeddingAction` | 0x7F | **0x7D** | propose + accept/decline `COutPacket(_, 125)`; nothing on 0x7F |

**Clientbound writer:**

| writer | was | now | evidence |
|---|---|---|---|
| `RPSGame` | 0xF2 | **0xFC** | `CField::OnPacket` `case 252: CRPSGameDlg::OnPacket` — resolves the 0xF2 collision with `StorageOperation` (correctly 0xF2 via `case 242: CTrunkDlg::OnPacket`) |

Post-repair: **zero** serverbound handler-opcode collisions; the only writer collisions are the three intended variant families (`0x00` Auth, `0x0A` ServerList, `0x40` portal spawn/remove — matching canonical gms_83). Combined with task-125's 4 item-use fixes, **11 gms_61 opcodes corrected total.**

## Verified state

- **Clientbound:** 152/153 MATCH. The 1 flagged item (`FieldEffectWeather@0x6A` = `OnPlayJukeBox` in the dispatch) is **NOT gms_61 corruption** — `FieldEffectWeather = FieldEffect + 2` holds in every version (v48 0x54/0x56 … v61 0x68/0x6A … v95 0x9A/0x9E); the Atlas-writer-name vs client-function-name discrepancy is identical across all versions (a cross-version naming/codec question, out of scope for a gms_61 opcode repair). Left unchanged.
- **Serverbound:** 100 MATCH, **2 MISMATCH — both fixed** (PartyInviteReject, WeddingAction). The `ServerListRequestHandle@0x0B` item the exhaustive pass flagged is **NOT corruption** — the duplicate binding on `0x04`+`0x0B` (with `0x0C` unclaimed) is identical in the canonical gms_83 and gms_72 templates; gms_83 login works, so it's Atlas's intentional design (the client's `0x0B` PIN-flow send → send world list). Also 2 dead bindings (HiredMerchant/AdminCommand), ~8 inert NO-SEND, 1 unresolved (SpouseChat).

## Diagnosed but not changed (structural — not a template opcode value)

These aren't "copied opcodes" fixable by reading the IDB; each needs an atlas-channel/atlas-login handler-architecture decision (removal/repoint), and none misroutes a live packet:

- **DEAD bindings** — `HiredMerchantOperationHandle` (0x3E), `AdminCommand` (0x7E): the client sends nothing on these; hired-merchant ops ride `0x6F` (interaction dispatcher, submodes 0x25/0x26/0x29), GM `/`-commands ride `0x2E` (chat).
- **~8 inert NO-SEND handlers** (client sends nothing on the bound opcode): `MobDropPickup`0x9D (unified under 0xA9), `MonsterBookCover`0x35 (local), `WeddingTalk`0x80 (covered by 0x7D), `CharacterLoggedIn`0x14/`StartError`0x19 (receive-only), `ChalkboardClose`0x2F (no class), `Coconut`0xB2/`MonsterCarnival`0xB7 (stub classes). Harmless; not corruption.
- **`CharacterSpouseChat@0x6D`** — genuinely unresolved (no distinct spouse-chat send located).

## Verification

`template_gms_61_1.json` parses; no handler-opcode collisions; writer collisions limited to the 3 intended variant families. atlas-configurations builds; seeder/templates tests pass (fresh run). Pure seed-data JSON (7 opcode values) — no Go/Dockerfile change.

## Rollout (post-merge)

Existing gms_61 tenants need a socket-config PATCH for these 7 opcodes (+ task-125's entries) and a channel restart; templates apply only at tenant creation.

🤖 Generated with [Claude Code](https://claude.com/claude-code)

https://claude.ai/code/session_014gRFWEaFwj42enf8wb5vY6
