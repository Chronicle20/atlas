package commodities

import (
	"github.com/google/uuid"
)

// CommodityByItemRestModel is the JSON API representation used by the
// GET /commodities/items/{itemId} reverse-lookup endpoint.
type CommodityByItemRestModel struct {
	Id              uuid.UUID `json:"-"`
	NpcId           uint32    `json:"npcId"`
	TemplateId      uint32    `json:"templateId"`
	MesoPrice       uint32    `json:"mesoPrice"`
	DiscountRate    byte      `json:"discountRate"`
	TokenTemplateId uint32    `json:"tokenTemplateId"`
	TokenPrice      uint32    `json:"tokenPrice"`
	Period          uint32    `json:"period"`
	LevelLimit      uint32    `json:"levelLimit"`
}

func (r CommodityByItemRestModel) GetName() string { return "commodities" }
func (r CommodityByItemRestModel) GetID() string   { return r.Id.String() }

func (r *CommodityByItemRestModel) SetID(id string) error {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return err
	}
	r.Id = parsed
	return nil
}
