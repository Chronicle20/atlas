# Task-069 Post-Phase-B — Misc-Domain Audit Closeout

## Final state

- **Packets audited (misc domain):** 28 new GMS v95 SUMMARY rows across account (2 new;
  AcceptTos pre-existing from task-027), channel (2), fame (4), merchant (7 employee-shop),
  quest (5), socket (5), stat (1), ui (3). `tool/` confirmed utility-only (0 packets).
- **Verdicts (GMS v95 misc):** 50 ✅ / 6 ❌ total in the v95 SUMMARY (28 login + 28 misc).
  Every misc ❌ is a documented static-analyzer artifact (mask/mode-driven, width-label, or
  locateAtlasFile collision) — see each report's `## Manual analysis` and TOTAL.md §4.
- **Cross-version coverage:** audited against GMS v83, v87, v95 + JMS v185
  (`docs/packets/audits/{gms_v83,gms_v87,gms_v95,jms_v185}/`; `docs/packets/ida-exports/{gms_v83,gms_v87,gms_v95,gms_jms_185}.json`).
- **TOTAL.md:** shipped at `docs/packets/audits/gms_v95/TOTAL.md` — cross-task ledger
  covering 027 + 028 + 065-069.

## Real wire bugs fixed

| Packet | File | IDA citation | Fix | Versions |
|---|---|---|---|---|
| stat Changed | `stat/clientbound/changed.go` | `CWvsContext::OnStatChanged`@0x9fd5d0, `GW_CharacterStat::DecodeChangeStat`@0x4fa000 | HP/MaxHP/MP/MaxMP int16→int32 (gated GMS≥95); add 2nd trailing flag byte (gated GMS≥95) | v95 (v83/v87/JMS confirmed int16 + 1 byte) |
| socket ChannelConnect | `socket/serverbound/channel_connect.go` | JMS `CClientSocket::OnConnect`@0x4b051f | gm/dummy1 field Encode1→Encode2 for `Region=="JMS"` (31 bytes vs GMS 30) | JMS v185 |

Both fixes verified by 4-variant round-trip + byte-level wire-shape tests; `atlas-login`
and `atlas-channel` build clean.

## Template opcode / enum fixes

None. No `template_*.json` opcode/enum changes were required for the misc domain (the audit
tool is FName + candidatesFromFName driven, not template-driven — see Tooling below).

## Tooling

- Added 28 `candidatesFromFName` cases (login-style, incl. synthetic `#Suffix` keys) in
  `tools/packet-audit/cmd/run.go` mapping misc IDA FNames → atlas structs.
- Added misc FName entries to `gms_v95.json`, `gms_v83.json`, and new `gms_v87.json` /
  `gms_jms_185.json`.
- **No analyzer changes** (`internal/atlaspacket/analyzer.go` untouched), per design §1.
- **Phase 1 skipped:** the `_body.go` files (`fame/response_body.go`, `ui/ui_open_body.go`,
  `merchant/operation_body.go`) are dispatcher helper functions, NOT registry-resolvable
  struct bodies — no TypeRegistry fixtures were applicable (see context.md correction).

## Plan corrections made during execution (documented in context.md)

1. **`--output` path:** the tool appends `<region>_v<major>` itself (run.go:42), so the
   correct value is `docs/packets/audits` (the parent), not `.../gms_v95`.
2. **Regression gate is semantic, not byte-identical:** SUMMARY row order is non-deterministic
   (map iteration) and the inherited login reports used a stale `../../` path prefix; Phase 0
   normalized them to the sibling convention. Gate = sorted packet→verdict set unchanged.
3. **Audit methodology:** the audit is FName × `candidatesFromFName` driven; the template
   writer/handler tables are dead code (`lookupFName` unused). The plan's template-first
   per-task steps do not affect the audit.

## Remaining work (deferred to `docs/packets/ida-exports/_pending.md`)

| Area | What | Why deferred |
|---|---|---|
| quest ActionStart/ActionComplete | missing `nItemPos` (uint32 delivery-item slot) between npcId and x,y | fix needs atlas-channel `quest_action.go` handler change too |
| quest ActionRestoreLostItem | models single item; IDA sends count-prefixed item list | struct redesign + handler change; rarely exercised |
| merchant modes 8/1/11 | missing/extra modes in `OnEntrustedShopCheckResult` switch | missing impl / possible hire-merchant (task-067) / unmapped |
| merchant serverbound | `HiredMerchantOperationHandle` bare const, no decoder | parsed in atlas-channel; out of libs scope |
| locateAtlasFile collisions | ChannelChange (buddy vs channel) | tool change out of scope; packet verified manually |

## Cross-version notes

- **v83:** all v95-era gates confirmed correct (HP/MP int16, 1 trailing byte, ui Lock 1 byte).
  `CLogin::SendSetGenderPacket` absent in v83. No fixes.
- **v87:** clean mirror of v83 (v87 < v90 < v95). `SendSetGenderPacket` present (unlike v83).
  No fixes.
- **JMS v185:** gates confirmed correct (JMS gets the GMS-narrow path correctly via
  `Region=="GMS"` guards). One real JMS divergence found + fixed: ChannelConnect gm field is
  2 bytes in JMS. `RegisterPin`/`SetGender` absent (JMS usesPin=false).

## Tool-domain confirmation

`libs/atlas-packet/tool/` is utility-only (`uint128.go`). Zero packet rows. Documented in
`_pending.md` and TOTAL.md §2.

## Coverage statement

`find libs/atlas-packet -maxdepth 1 -type d` cross-referenced against the 7-task matrix in
TOTAL.md §2 — every directory is owned by a task or is a documented non-wire exclusion
(model/, test/, tool/). No gaps.

## Integration (pre-PR)

This branch forked from main @ `3bab0d885`, BEFORE sibling tasks 028/065-068 merged. The
misc reports and SUMMARY here cover only login + misc. Landing requires integrating with
current main (`git merge main` or rebase): resolve `run.go` `candidatesFromFName` and the
version `*.json` files as UNIONS (main's sibling FNames + task-069's misc FNames), then
re-run all four audits to regenerate the SUMMARYs with the full packet set. Reports are
otherwise additive (sibling reports and misc reports are disjoint files; login reports use
the same format on both sides).
