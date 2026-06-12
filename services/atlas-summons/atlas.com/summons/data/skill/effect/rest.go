package effect

// RestModel mirrors the relevant attributes of an atlas-data skill-effect
// resource. The JSON tags match the channel-side effect RestModel
// (services/atlas-channel/.../data/skill/effect/rest.go) so the same
// atlas-data response deserializes identically here.
type RestModel struct {
	WeaponAttack  int16             `json:"weaponAttack"`
	MagicAttack   int16             `json:"magicAttack"`
	Hp            uint16            `json:"hp"`
	Duration      int32             `json:"duration"`
	X             int16             `json:"x"`
	Y             int16             `json:"y"`
	Prop          float64           `json:"prop"`
	MonsterStatus map[string]uint32 `json:"monsterStatus"`
	Statups       []StatupRestModel `json:"statups"`
}

// StatupRestModel mirrors atlas-data's skill-effect `statups` element
// (services/atlas-data/.../data/skill/effect/statup/rest.go) so the same
// response deserializes identically here. Each entry is one buff stat delta.
type StatupRestModel struct {
	Type   string `json:"type"`
	Amount int32  `json:"amount"`
}

func Extract(rm RestModel) (Model, error) {
	statups := make([]StatChange, 0, len(rm.Statups))
	for _, su := range rm.Statups {
		statups = append(statups, StatChange{Type: su.Type, Amount: su.Amount})
	}
	return Model{
		weaponAttack:  rm.WeaponAttack,
		magicAttack:   rm.MagicAttack,
		hp:            rm.Hp,
		duration:      rm.Duration,
		x:             rm.X,
		y:             rm.Y,
		prop:          rm.Prop,
		monsterStatus: rm.MonsterStatus,
		statups:       statups,
	}, nil
}
