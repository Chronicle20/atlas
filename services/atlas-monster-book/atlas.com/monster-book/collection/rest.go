package collection

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/monster"
)

type RestModel struct {
	Id               character.Id `json:"-"`
	BookLevel        uint16       `json:"bookLevel"`
	NormalCount      uint16       `json:"normalCount"`
	SpecialCount     uint16       `json:"specialCount"`
	TotalUniqueCards uint16       `json:"totalUniqueCards"`
	CoverCardId      item.Id      `json:"coverCardId"`
	CoverMonsterId   monster.Id   `json:"coverMonsterId"`
	ExpBonusPercent  uint16       `json:"expBonusPercent"`
}

func (r RestModel) GetName() string { return "monster-book" }
func (r RestModel) GetID() string   { return strconv.FormatUint(uint64(r.Id), 10) }
func (r *RestModel) SetID(id string) error {
	v, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return err
	}
	r.Id = character.Id(v)
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
		CoverMonsterId:   m.CoverMobId(),
		ExpBonusPercent:  m.ExpBonusPercent(),
	}, nil
}

type PatchInput struct {
	Id          character.Id `json:"-"`
	CoverCardId item.Id      `json:"coverCardId"`
}

func (p PatchInput) GetName() string { return "monster-book" }
func (p PatchInput) GetID() string   { return strconv.FormatUint(uint64(p.Id), 10) }
func (p *PatchInput) SetID(id string) error {
	v, err := strconv.ParseUint(id, 10, 32)
	if err != nil {
		return err
	}
	p.Id = character.Id(v)
	return nil
}
