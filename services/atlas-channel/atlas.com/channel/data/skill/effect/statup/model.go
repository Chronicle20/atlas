package statup

type Model struct {
	buffType string
	amount   int32
}

func (s Model) Mask() string {
	return s.buffType
}

func (s Model) Amount() int32 {
	return s.amount
}

// NewModel builds a statup carrying the given buff-type mask and amount.
// Used to synthesize the MONSTER_RIDING statup for tamed mounts, where the
// vehicle id (the equipped taming-mob item id) is read from slot -18 rather
// than from the skill's WZ effect data.
func NewModel(buffType string, amount int32) Model {
	return Model{
		buffType: buffType,
		amount:   amount,
	}
}
