package monster

type DamageEntryModel struct {
	characterId uint32
	damage      uint32
}

func NewDamageEntryModel(characterId uint32, damage uint32) DamageEntryModel {
	return DamageEntryModel{
		characterId: characterId,
		damage:      damage,
	}
}

func (d DamageEntryModel) CharacterId() uint32 {
	return d.characterId
}

func (d DamageEntryModel) Damage() uint32 {
	return d.damage
}
