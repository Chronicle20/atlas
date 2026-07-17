package dataskill

import "strconv"

// RestModel is the trimmed atlas-data skill projection: per-level effect x
// (the berserk threshold percentage — the WZ `berserk` field is a dead type
// marker in Atlas and MUST NOT be read; design §2).
type RestModel struct {
	Id      uint32        `json:"-"`
	Effects []EffectModel `json:"effects"`
}

type EffectModel struct {
	X int16 `json:"x"`
}

func (r RestModel) GetName() string {
	return "skills"
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(idStr string) error {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}
