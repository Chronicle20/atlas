package collection

import "strconv"

type RestModel struct {
	Id               uint32 `json:"-"`
	BookLevel        uint16 `json:"bookLevel"`
	NormalCount      uint16 `json:"normalCount"`
	SpecialCount     uint16 `json:"specialCount"`
	TotalUniqueCards uint16 `json:"totalUniqueCards"`
	CoverCardId      uint32 `json:"coverCardId"`
	ExpBonusPercent  uint16 `json:"expBonusPercent"`
}

func (r RestModel) GetName() string { return "monster-book" }
func (r RestModel) GetID() string   { return strconv.FormatUint(uint64(r.Id), 10) }
func (r *RestModel) SetID(id string) error {
	v, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return err
	}
	r.Id = uint32(v)
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:               m.CharacterId(),
		BookLevel:        m.BookLevel(),
		NormalCount:      m.NormalCount(),
		SpecialCount:     m.SpecialCount(),
		TotalUniqueCards: m.TotalUniqueCards(),
		CoverCardId:      m.CoverCardId(),
		ExpBonusPercent:  m.ExpBonusPercent(),
	}, nil
}

type PatchInput struct {
	Id          uint32 `json:"-"`
	CoverCardId uint32 `json:"coverCardId"`
}

func (p PatchInput) GetName() string { return "monster-book" }
func (p PatchInput) GetID() string   { return strconv.FormatUint(uint64(p.Id), 10) }
func (p *PatchInput) SetID(id string) error {
	v, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return err
	}
	p.Id = uint32(v)
	return nil
}
