package crystalband

import (
	"strconv"
)

type RestModel struct {
	// Id is the band's minimum level; it is the JSON:API resource id rather
	// than an attribute, since (tenant_id, min_level) is the business key.
	Id            string `json:"-"`
	MaxLevel      uint32 `json:"maxLevel"`
	CrystalItemId uint32 `json:"crystalItemId"`
	Count         uint32 `json:"count"`
}

func (r RestModel) GetName() string {
	return "crystalBands"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:            strconv.FormatUint(uint64(m.MinLevel()), 10),
		MaxLevel:      m.MaxLevel(),
		CrystalItemId: uint32(m.CrystalItemId()),
		Count:         m.Count(),
	}, nil
}
