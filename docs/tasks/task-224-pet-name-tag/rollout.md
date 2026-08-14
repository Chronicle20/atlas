# task-224 Rollout — Pet Name Tag

## Prerequisite: none

Unlike task-220 (Meso Sack), this feature reads **no WZ value**. `Item.wz/Cash/0517.img.xml`
carries only `z`, `slotMax`, `cash` and icon canvases — no payout node, nothing to ingest.
The rename payload is player-supplied and the consumption is template-keyed, so there is no
re-ingest step and no version-parity survey to run.

## Required: reconcile live tenant socket configs

All **ten** seed templates gained a `PetNameChanged` writer at each version's
`PET_NAMECHANGE` opcode — `gms_48`, `gms_61`, `gms_72`, `gms_79`, `gms_83`, `gms_84`,
`gms_87`, `gms_92`, `gms_95`, and `jms_185`. This is a correction from the plan-time
assumption that `gms_48` was `n-a` for this opcode: IDA verification during
implementation found `PET_NAMECHANGE` present in the v48 client
(`CUser::OnPetPacket @0x69221b` case `'q'` / opcode `0x071` → `CPet::OnNameChanged
@0x58da70`, decoding a string plus a byte selecting the pet template's nameTag — the
same `str + byte` shape as every other GMS version). All ten packet-coverage matrix
cells for `PET_NAMECHANGE` are `✅` (STATUS.md row 205); there is no `n-a` cell for
this op.

**A live tenant whose socket config predates this change will silently drop the new
writer** — the packet is simply never emitted, with no error anywhere
(`bug_new_opcodes_not_in_live_tenant_config`).

For every live tenant, reconcile its socket configuration to the updated template
for its version before announcing the feature. Verify per tenant that the
`writers` array contains a `PetNameChanged` entry, at the correct opcode, with a
non-empty `fname`.

## Post-deploy verification

1. Use a Pet Name Tag with a pet summoned — the new name renders immediately for
   the player and for another character standing in the map.
2. Have a third character enter the map afterwards — the spawn body carries the
   new name.
3. Relog and change channel — the name persists.
4. Despawn and respawn the pet — the name tag decoration does not flicker
   (`NameTagLayer` is shared by both codecs).
5. Attempt a 3-character and a 13-character name — both are rejected with pink
   text, the client unlocks, and the tag remains in the cash slot.
6. Repeat step 1 on a `gms_48` tenant specifically — this version was previously
   assumed excluded and needs its own confirmation that the reconciled template
   actually emits `PetNameChanged` at `0x071`.
