# Task-068 Post-Phase-B — World-Domain Audit Closeout

## Final state

- World-domain packets audited per version (Portal / Field / Npc FNames):
  - **GMS v95**: 50 rows — 43 ✅ / 7 ❌
  - **GMS v83**: 48 rows — 39 ✅ / 9 ❌
  - **GMS v87**: 49 rows — 43 ✅ / 6 ❌
  - **JMS v185**: 49 rows — 30 ✅ / 19 ❌
- Cross-version passes complete: GMS v95 (Phase 0/1), GMS v83, GMS v87, JMS v185 (Phases 2–3).
- IDA-export coverage: GMS v95 / GMS v83 / GMS v87 / JMS v185 — world-domain FNames populated.
- All remaining ❌ are either tool-limitation false positives (manually verified correct, documented in `docs/packets/ida-exports/_pending.md`) or genuine structural divergences deferred to sibling tasks (`AffectedAreaCreated`, `EffectWeather`). No unresolved in-scope wire bugs.
- Total commits on branch: 32 (merge-base from `main`).

## Real wire bugs fixed

| Packet | File | Fix one-liner | Versions affected | Commit |
|---|---|---|---|---|
| `field/serverbound/change` | `libs/atlas-packet/field/serverbound/change.go` | x/y branch was inverted — gate the x/y read on `portalName` being non-empty per the v95 client | GMS v95 | `ef6a96392` |
| `field/clientbound/warp_to_map` | `libs/atlas-packet/field/clientbound/warp_to_map.go` | `nHP` width: GMS ≥ v95 → 4 bytes, else 2 bytes (v83/v87/JMS = 2) | GMS v95 (4B), v83/v87/JMS (2B) | `9067fe2f1` + `85fd59a76` (initial `37f072c11`) |
| `field/clientbound/set_field` + `warp_to_map` | `libs/atlas-packet/field/clientbound/set_field.go`, `warp_to_map.go` | `m_dwOldDriverID` emitted only for GMS ≥ v95 | GMS v95 only (absent v83/v87/JMS) | `9067fe2f1` (confirmed v87 `dd74fe25c`) |
| `npc/clientbound/guide_talk` | `libs/atlas-packet/npc/clientbound/guide_talk.go` | Leading bool was inverted (Message=false / Idx=true) to match v95 client branch | GMS v95 | `0d1a20d9b` |
| `npc/clientbound/conversation` SayImage | `libs/atlas-packet/npc/clientbound/conversation.go` | Image count is a `byte`, not an `int` | all versions | `9a4dad51d` |
| `npc/clientbound/conversation` AskMemberShopAvatar | `libs/atlas-packet/npc/clientbound/conversation.go` | Candidate count is a `byte`, not an `int` | GMS (absent in JMS185) | `f5be63196` |
| `npc/serverbound/shop_buy` + AskSlideMenu | `libs/atlas-packet/npc/serverbound/shop_buy.go` | ShopBuy `discountPrice` GMS-only; AskSlideMenu `slideDlgType` present for GMS > v83 \|\| JMS | GMS / JMS gates | `334556527` |

Plus the audit-tool determinism fix:

- `tools/packet-audit` — candidate → FName selection made deterministic (`cmd/run.go`). Commit `2aed0733e`.

## Tooling

- **TypeRegistry fixtures** for `ShopCommodity` plus the 15 conversation structs, so the analyzer can resolve sub-struct descent for the npc conversation cluster.
- **Candidate-selection determinism fix** in `cmd/run.go` (commit `2aed0733e`) — the candidate → FName mapping is now deterministic so re-runs produce stable reports.
- **NO `analyzer.go` changes** — design §3 honored; all world-domain coverage achieved via fixtures + deterministic selection, not analyzer surgery.

## Remaining work / deferrals

(Per `docs/packets/ida-exports/_pending.md` world-domain sections.)

| Area | What | Status |
|---|---|---|
| `FieldAffectedAreaCreated` | The atlas struct matches NEITHER v83/v87/v95/JMS — needs a structural rewrite (new model fields + v95 `tStart` gate) to the v95 `CAffectedAreaPool::OnAffectedAreaCreated` layout. Same 8-field divergence confirmed across v87/JMS185 passes. | **Deferred — sibling task** |
| `FieldEffectWeather` | JMS185 shape divergence vs the GMS `EffectWeather` struct (leading `!active` byte + trailing conditional); needs refactor. | **Deferred — sibling task** |
| Analyzer tool-limitations | `FieldClock`, `NpcAction`, `NpcShopList`, `NpcContinueConversationSelection`, `AskPet`/`AskPetAll`, and the `NpcConversation` envelope all show ❌ in SUMMARY but are manually verified correct (loop/conditional/exclusive-branch flatten limitations). Documented per-field in their `.md` reports + `_pending.md`. | **Manually verified ✅ (tool FP)** |
| Op-family enums | Dialog-type / op-family enums are config-driven (template layer), not encoder-resolved. | **By design** |
| atlas-channel routing | `ContinueConversation` `msgType == 2` likely should be `== 3`/`14` — this is in atlas-channel routing, out of packet-lib scope. | **Deferred — sibling task** |

## Cross-version notes

- **`m_dwOldDriverID` + 4-byte `nHP`** were introduced between GMS v87 and v95. JMS185 stays 2-byte `nHP` and emits NO `oldDriverID`.
- **Dialog-type switch enum differs by version**: v83 has no `SayImage` case (shifted enum); v87/v95/JMS185 have `SayImage = 1`. `AskMemberShopAvatar` is absent in JMS185. These are handled at the config/template layer — the encoders are discriminator-agnostic.
- **ShopBuy `discountPrice`** is GMS-only.
- **AskSlideMenu** leading int (`slideDlgType`) is present for GMS > v83 and for JMS.
