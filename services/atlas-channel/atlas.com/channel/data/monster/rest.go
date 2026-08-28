package monster

import "strconv"

// RestModel is a projection of atlas-data's monster resource; only the
// fields the damage pipeline needs are declared, the rest of the
// attributes payload is ignored on unmarshal.
type RestModel struct {
	Id          uint32 `json:"-"`
	Boss        bool   `json:"boss"`
	FixedDamage uint32 `json:"fixed_damage"`
}

func (r RestModel) GetName() string {
	return "monsters"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func Extract(rm RestModel) (Model, error) {
	return Model{
		id:          rm.Id,
		boss:        rm.Boss,
		fixedDamage: rm.FixedDamage,
	}, nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:          m.id,
		Boss:        m.boss,
		FixedDamage: m.fixedDamage,
	}, nil
}
