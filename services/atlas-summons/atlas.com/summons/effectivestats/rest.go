package effectivestats

// RestModel mirrors the atlas-effective-stats service's character stats
// response (services/atlas-effective-stats/.../stat/rest.go). JSON tags match
// that resource exactly so the REST decode binds correctly.
type RestModel struct {
	Id           string `json:"-"`
	Strength     uint32 `json:"strength"`
	Dexterity    uint32 `json:"dexterity"`
	Luck         uint32 `json:"luck"`
	Intelligence uint32 `json:"intelligence"`
	WeaponAttack uint32 `json:"weaponAttack"`
	MagicAttack  uint32 `json:"magicAttack"`
}

func (r RestModel) GetName() string {
	return "effective-stats"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

// SetToOneReferenceID and SetToManyReferenceIDs satisfy api2go's unmarshal path
// when the upstream resource carries a relationships block.
func (r *RestModel) SetToOneReferenceID(_, _ string) error            { return nil }
func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }

func Extract(rm RestModel) (Model, error) {
	return Model{
		strength:     rm.Strength,
		dexterity:    rm.Dexterity,
		luck:         rm.Luck,
		intelligence: rm.Intelligence,
		weaponAttack: rm.WeaponAttack,
		magicAttack:  rm.MagicAttack,
	}, nil
}
