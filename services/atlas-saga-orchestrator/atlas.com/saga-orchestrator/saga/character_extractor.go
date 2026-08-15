package saga

// ExtractCharacterId extracts the character ID from a step's payload.
// Returns 0 if the payload type doesn't contain a character ID or is unknown.
func ExtractCharacterId(step Step[any]) uint32 {
	switch p := step.Payload().(type) {
	case AwardMesosPayload:
		return p.CharacterId
	case AwardItemActionPayload:
		return p.CharacterId
	case AwardExperiencePayload:
		return p.CharacterId
	case AwardLevelPayload:
		return p.CharacterId
	case AwardCurrencyPayload:
		return p.CharacterId
	case AwardFamePayload:
		return p.CharacterId
	case DestroyAssetPayload:
		return p.CharacterId
	case EquipAssetPayload:
		return p.CharacterId
	case UnequipAssetPayload:
		return p.CharacterId
	case ChangeJobPayload:
		return p.CharacterId
	case ChangeHairPayload:
		return p.CharacterId
	case ChangeFacePayload:
		return p.CharacterId
	case ChangeSkinPayload:
		return p.CharacterId
	case CreateSkillPayload:
		return p.CharacterId
	case UpdateSkillPayload:
		return p.CharacterId
	case ValidateCharacterStatePayload:
		return p.CharacterId
	case CreateAndEquipAssetPayload:
		return p.CharacterId
	case WarpToRandomPortalPayload:
		return p.CharacterId
	case WarpToPortalPayload:
		return p.CharacterId
	case WarpToSavedLocationPayload:
		return p.CharacterId
	case SpawnMonsterPayload:
		return p.CharacterId
	case CompleteQuestPayload:
		return p.CharacterId
	case StartQuestPayload:
		return p.CharacterId
	case ApplyConsumableEffectPayload:
		return uint32(p.CharacterId)
	case CancelConsumableEffectPayload:
		return uint32(p.CharacterId)
	case SendMessagePayload:
		return p.CharacterId
	case DepositToStoragePayload:
		return p.CharacterId
	case UpdateStorageMesosPayload:
		return p.CharacterId
	case ShowStoragePayload:
		return p.CharacterId
	case OpenNpcShopPayload:
		return p.CharacterId
	case StartItemConversationPayload:
		return p.CharacterId
	case StartNpcConversationPayload:
		return p.CharacterId
	case TransferToStoragePayload:
		return p.CharacterId
	case WithdrawFromStoragePayload:
		return p.CharacterId
	case AcceptToStoragePayload:
		return p.CharacterId
	case ReleaseFromCharacterPayload:
		return p.CharacterId
	case AcceptToCharacterPayload:
		return p.CharacterId
	case ReleaseFromStoragePayload:
		return p.CharacterId
	case TradeSettlementPayload:
		// A trade names TWO participants and this function can only surface one.
		// Sides[0] is picked for determinism ONLY — side order carries no role
		// meaning, so this is "a participant", never "the giver". Consumers that
		// must reach both participants (atlas-trades' LEAVE 8 notification) key
		// off the saga's transactionId, which is the trade ledger's idempotency
		// key, not off this field.
		return uint32(p.Sides[0].CharacterId)
	case TransferToTradePayload:
		return p.CharacterId
	case AcceptToTradePayload:
		// The escrow row's owner IS the staging character: an accept_to_trade
		// only ever follows that character's own release_from_character.
		//
		// ReleaseFromTradePayload has deliberately no case: it carries the
		// escrow row id alone (the row holds the owner), so there is nothing to
		// extract. It returns 0 = "unconstrained", which is correct — a release
		// is never routed through ForCharacter. Same posture as
		// ReleaseFromMtsHoldingPayload.
		return p.OwnerId
	case SelectGachaponRewardPayload:
		return p.CharacterId
	case EmitGachaponWinPayload:
		return p.CharacterId
	default:
		return 0
	}
}
