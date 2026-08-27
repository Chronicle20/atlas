// Character-slots domain model types.
//
// Replaces the old flat, always-4 `characterSlots` attribute that used to
// live on `AccountAttributes`. Slots are now scoped to an (account, world)
// pair — `GET accounts/{accountId}/worlds/{worldId}/character-slots` — so
// there is one of these per world an account has characters in, not one per
// account.

export interface CharacterSlots {
  id: string;
  attributes: CharacterSlotsAttributes;
}

export interface CharacterSlotsAttributes {
  worldId: number;
  slots: number;
}
