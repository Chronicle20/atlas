# Opaque register-boundary type ledger (FR-4)

> **Purpose.** The packet-audit analyzer and the live `validate` layer both stop at
> the **register boundary**: an IDA type that reads as a single `DecodeBuf`/`EncodeBuf`
> token, or a struct with no statically-decomposable layout, cannot be diffed field-by
> -field against Atlas's encoder. task-080 flagged these as the **OPAQUE** bucket in
> `docs/packets/ida-exports/_pending.md` §4. This ledger (task-081, FR-4) records, per
> opaque type, **whether it is decomposable** and — where it is not — **the byte-level
> test that verifies Atlas's encoder in lieu of analyzer decomposition**. No opaque
> type remains in an unexamined "analyzer-skipped" state.
>
> **Discipline:** for an opaque type, the byte test beside the struct (asserting the
> exact wire slice against the IDA-traced read-order) is the oracle — not the analyzer
> verdict. If the analyzer/validate and the byte test disagree, the byte test wins.

## How to read the disposition column

- **DECOMPOSED** — the analyzer descends the sub-struct (task-080 A3); fields are
  verified inline. Listed here only if it was *formerly* opaque.
- **VERIFIED-EXCEPTION** — genuinely undecomposable (mask/mode-driven or a packed
  register blob). Atlas's encoder is confirmed against the client by the cited
  byte-level test; the analyzer/validate row stays an accepted exclusion.

## Ledger

| Opaque type | Appears in (packets) | IDA boundary | Decomposable? | Disposition · verifying test |
|---|---|---|---|---|
| **mob temporary-stat blob** | `MonsterSpawn`, `MonsterStatSet`, `MonsterStatReset`, `MonsterControl` | `CMob::SetTemporaryStat` / `CMob::Init` / `ProcessStatSet` — one `bytes` token; Atlas expands ~25 mask-gated fields | No — mask-driven variable layout | **VERIFIED-EXCEPTION** · `monster/clientbound/{spawn,stat,control}_test.go`, `model/character_temporary_stat_test.go`. `MonsterControl`'s hardcoded `byte(5)` aggro is a semantic note, not a wire-shape diff (width/position match). |
| **mob move-path** | `MonsterMovement`, `MonsterMovementRequest` | `CMob::GenerateMovePath` sub-struct | No — element-loop over move fragments | **VERIFIED-EXCEPTION** · `model/movement_test.go`, `monster/clientbound/movement_test.go`, `monster/serverbound/movement_test.go` (byte-for-byte, task-065 `e32a3d809`). |
| **`model.Movement` (shared path)** | pet + monster movement, `*MovementRequest` | shared movement encoder reads as one `DecodeBuf` | No — element-loop | **VERIFIED-EXCEPTION** · `model/movement_test.go`. |
| **`CPet` body** | `PetActivated`, `PetMovement`, `PetCommandResponse`, `PetCommand`, `PetChatRequest`, `PetDropPickUp`, `PetMovementRequest` | `CPet::Init` expands as `DecodeBuf` placeholder; prefix fields ✅ | Partial — prefix decomposed, body opaque | **VERIFIED-EXCEPTION** · `model/pet_test.go`, `pet/clientbound/*_test.go`, `pet/serverbound/*_test.go` (incl. the v95 `updateTime` gate, `pet/serverbound/chat_test.go::TestChatUpdateTimeGate`). |
| **AvatarLook byte-blob** | `MessengerAdd`, `MessengerUpdate` | `CUIMessenger::OnUpdate` reads AvatarLook as one opaque `DecodeBuf`; Atlas emits `WriteByteArray` | No — structured look encoded as a byte array | **VERIFIED-EXCEPTION** · `messenger/clientbound/{add,update}_test.go`; the shared `model.Avatar` encoder is audited independently and byte-correct. |
| **`model.Asset` / `GW_ItemSlotBase`** | `InventoryAdd`, `StorageUpdateAssets`, `InventoryChangeBatch` | per-tab item-slot loop reads as an opaque sub-struct | No — per-tab loop + type-tagged item bodies | **VERIFIED-EXCEPTION** · `model/asset_test.go`, `model/asset_v84_test.go`, `inventory/clientbound/change_batch_test.go`, `storage/clientbound/show_test.go`. Runtime callers pass exactly one tab → wire ✅. |
| **`GUILDMEMBER` packed array** | `GuildInfo`, `GuildMemberJoined` | `GUILDMEMBER::Decode` reads a packed `DecodeBuffer(0x25 = 37)` per member; Atlas loops per element | No — packed fixed-stride array | **VERIFIED-EXCEPTION** · `guild/clientbound/info_test.go` (37-byte member body verified, task-066 `29a248285`). |
| **`interaction.Visitor` / `Room` / per-item asset** | `InteractionInteractionEnter`, `InteractionInteractionEnterResultSuccess`, `InteractionInteractionUpdateMerchant` | mini-room sub-structs flattened vs a single buffer; headers ✅ | Partial — headers decomposed, body opaque | **VERIFIED-EXCEPTION** · `interaction/clientbound/interaction_test.go` (v95). |
| **`GW_CharacterStat` (stat mask)** | `Changed` (stat), CharacterData stat block | mask-driven; Atlas emits only set fields in config-mask order | No — mask-driven variable layout | **VERIFIED-EXCEPTION** · `model/character_temporary_stat_test.go`, `model/character_statistics_test.go`. The 2 real v95 wire bugs here (HP/MP int32, 2nd trailing byte) were fixed in task-069; the residual `Changed` ❌ is the mask static-artifact. |

## Coverage statement

Every type in task-080's A3 opaque set maps to a row above. None is decomposable by
the analyzer (all are mask-driven, packed-array, or element-loop register boundaries),
so each carries a **VERIFIED-EXCEPTION**: a byte-level test beside the struct asserts
Atlas's wire against the IDA-traced read-order. The matching SUMMARY `❌`/`🔍` rows are
accepted exclusions in `_pending.md` §4 (OPAQUE) — they are "analyzer/validate cannot
model this," not "Atlas is wrong." task-081's live `validate` layer corroborates this:
the opaque packets surface as `divergent` length-close representation diffs with **0
confirmed real wire bugs**.

## Adding a row

When a new opaque type appears in a future pass: (1) confirm via IDA it is genuinely
undecomposable (mask/packed/loop, not just unharvested); (2) ensure a byte-level test
beside the struct asserts the exact wire for every version it targets; (3) add a row
here and classify the SUMMARY residue under `_pending.md` §4. Never leave an opaque
type as a bare analyzer skip.
