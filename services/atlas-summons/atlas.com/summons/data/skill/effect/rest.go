package effect

// RestModel mirrors the relevant attributes of an atlas-data skill-effect
// resource. The JSON tags match the channel-side effect RestModel
// (services/atlas-channel/.../data/skill/effect/rest.go) so the same
// atlas-data response deserializes identically here.
type RestModel struct {
	WeaponAttack  int16             `json:"weaponAttack"`
	MagicAttack   int16             `json:"magicAttack"`
	Duration      int32             `json:"duration"`
	X             int16             `json:"x"`
	Y             int16             `json:"y"`
	Prop          float64           `json:"prop"`
	MonsterStatus map[string]uint32 `json:"monsterStatus"`
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		weaponAttack:  rm.WeaponAttack,
		magicAttack:   rm.MagicAttack,
		duration:      rm.Duration,
		x:             rm.X,
		y:             rm.Y,
		prop:          rm.Prop,
		monsterStatus: rm.MonsterStatus,
	}, nil
}
