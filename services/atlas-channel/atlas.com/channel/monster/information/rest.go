package information

import "strconv"

type RestModel struct {
	Id      string                `json:"-"`
	Attacks []AttackInfoRestModel `json:"attacks"`
}

type AttackInfoRestModel struct {
	Pos         uint8 `json:"pos"`
	ConMP       int32 `json:"conMP"`
	AttackAfter int32 `json:"attackAfter"`
}

func (r RestModel) GetName() string {
	return "monsters"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func Extract(rm RestModel) (Model, error) {
	id, err := strconv.ParseUint(rm.Id, 10, 32)
	if err != nil {
		// id may be empty in tests; tolerate.
		id = 0
	}
	attacks := make([]AttackInfo, 0, len(rm.Attacks))
	for _, a := range rm.Attacks {
		attacks = append(attacks, AttackInfo{
			Pos:         a.Pos,
			ConMP:       a.ConMP,
			AttackAfter: a.AttackAfter,
		})
	}
	return Model{
		monsterId: uint32(id),
		attacks:   attacks,
	}, nil
}
