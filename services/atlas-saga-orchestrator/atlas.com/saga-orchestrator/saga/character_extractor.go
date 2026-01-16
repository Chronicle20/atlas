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
	case SpawnMonsterPayload:
		return p.CharacterId
	case CompleteQuestPayload:
		return p.CharacterId
	case StartQuestPayload:
		return p.CharacterId
	case ApplyConsumableEffectPayload:
		return p.CharacterId
	case SendMessagePayload:
		return p.CharacterId
	case DepositToStoragePayload:
		return p.CharacterId
	case UpdateStorageMesosPayload:
		return p.CharacterId
	case ShowStoragePayload:
		return p.CharacterId
	case TransferAssetPayload:
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
	default:
		return 0
	}
}
