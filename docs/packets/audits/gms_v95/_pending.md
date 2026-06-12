# Accepted permanent exclusions — GMS v95 packet audit

> **Closeout of record: task-080 (packet-audit-closeout).** This file is a
> *registry*, not a work-list. It holds **zero actionable items**. Every entry
> below is a blessed permanent exclusion with IDA evidence and a one-line
> justification: a SUMMARY `❌`/`🔍` that is provably NOT a wire bug (an analyzer
> boundary, a representation/loop artifact, or a version-absent mode). Open
> deferrals that this file previously carried were either fixed during task-080
> (cited below) or moved into the registry after verification.
>
> Anything genuinely actionable was surfaced as a follow-up task by task-080
> Phase E — it is NOT parked here. See the master registry
> `docs/packets/ida-exports/_pending.md` for the cross-domain four-version
> registry; this file is the GMS-v95-domain subset retained for locality.
>
> **task-081 update (2026-06-11):** these exclusions were live-IDB-validated (not
> re-exported — that regresses the audit). The v95 partial-implementation residue is
> now machine-listed in `docs/packets/audits/gms_v95/_unimplemented.json` (123
> intentionally-unimplemented client sub-ops; 0 extra-mode), the per-mode shapes are
> backed by 72 persisted `dispatch` selectors in `gms_v95.json`, and the opaque
> types below are catalogued in `docs/packets/audits/OPAQUE_LEDGER.md`. The audit ❌
> for v95 fell 77 → 3. See master registry §12 + TOTAL.md §6.

---

## Resolved during task-080 (no longer pending)

| Former deferral | Resolution | task-080 sub-task |
|---|---|---|
| ROUTING-npc-continue-conversation-discriminator (handler read `== 2`, should be msgType 3/14/5/8/9) | **FIXED** — channel `npc_continue_conversation.go` discriminator corrected to 3/14 -> text, 5/8/9 -> selection | B2.1 (`4fa1e3d52`) |
| AFFECTEDAREA-create-shape / FieldAffectedAreaCreated (atlas served a bespoke v83 shape no audited client decodes) | **FIXED** — `affected_area_created.go` rewritten to the abs-RECT layout + `tStart` gated `GMS>=95`; channel passes skill id/level/type | B1.1 (`45711afce`, `fff7668ee`) |
| OP-FAMILY-messenger-serverbound (full mode enum unverified) | **VERIFIED, no fix** — client emits exactly {0,2,3,5,6}; modes 1/4 are clientbound-only | B3.1 (spike-messenger) |
| OP-FAMILY-messenger-decline (declineMode value space) | **VERIFIED, no fix** — boolean 0/non-zero selecting two StringPool templates; `byte` round-trip covers it | B3.2 (spike-messenger) |
| OP-FAMILY-npc-shop-operation (clientbound mode enum) | **VERIFIED, no fix** — every emitted mode is a handled, semantically-matching client arm; un-emitted modes are client no-op/default | B3.3 (spike-npc-shop) |
| OP-FAMILY-npc-shop-serverbound (op-byte values) | **VERIFIED, no fix** — BUY=0/SELL=1/RECHARGE=2/LEAVE=3 match the GMS client; LEAVE carries no body | B3.4 (spike-npc-shop) |
| SETFIELD-old-driver-id (`m_dwOldDriverID` version-introduction unknown) | **RESOLVED in task-068 Phase 3** — introduced between v87 and v95; gated `GMS>=95`; nHP width gated `GMS>=95` | task-068 (carried forward) |

---

## Accepted permanent exclusions

### Analyzer representation artifact — AvatarLook opaque byte-blob (opaque)

| Field | Value |
|---|---|
| Affected packets | `MessengerAdd` (`messenger/clientbound/add.go`), `MessengerUpdate` (`messenger/clientbound/update.go`) |
| IDA | `CUIMessenger::OnUpdate` / avatar-look `DecodeBuf` |
| Why excluded | Atlas encodes the AvatarLook as `WriteByteArray([]byte)`; the IDA side reads it as one opaque `DecodeBuf`. The flat analyzer cannot align a byte-array against the structured AvatarLook decode -> spurious `❌`. The AvatarLook encoder is the shared `model.Avatar` body, audited independently and byte-correct. |
| Justification | Opaque byte-blob vs structured decode; wire verified by the shared AvatarLook encoder. Analyzer-boundary type (A3 `Opaque` set). |

### Op-byte / dispatcher rows verified as wire-correct families

| Family | Evidence | Justification |
|---|---|---|
| OP-FAMILY-npc-shop-clientbound (`ShopOperationSimple/LevelRequirement/GenericError`) | `CShopDlg::OnPacket` per-mode arms | Per-mode wire shapes ✅ against v95 IDA; mode values are template config, verified equal to client (B3.3). |
| OP-FAMILY-npc-shop-serverbound (`Shop`/`ShopBuy`/`ShopSell`/`ShopRecharge`) | `CShopDlg::Send{Buy,Sell,Recharge}Request` | Per-op bodies ✅; op-byte values runtime-config, verified equal to client (B3.4). |

---

## Cross-check

`grep -nE 'DEFERRED|pending|TODO|🔍|FIXME|action'` against this file returns only
hits inside this registry's prose (the word "pending" in the title, "FIXED"
citations, and category labels) — **no open actionable item remains.**
