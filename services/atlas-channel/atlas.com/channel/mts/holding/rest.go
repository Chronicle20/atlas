package holding

import "github.com/Chronicle20/atlas/libs/atlas-constants/world"

// RestModel mirrors atlas-mts's holding.RestModel (the JSON:API "holdings"
// resource). It carries the item snapshot plus the persistent ItcSn serial. Only
// the fields the channel renders into an ITCITEM are consumed here. Holdings carry
// no relationships block, so no api2go relationship stubs are required.
type RestModel struct {
	Id         string `json:"-"`
	WorldId    byte   `json:"worldId"`
	ItcSn      uint32 `json:"itcSn"`
	OwnerId    uint32 `json:"ownerId"`
	Origin     string `json:"origin"`
	TemplateId uint32 `json:"templateId"`
	Quantity   uint32 `json:"quantity"`
}

func (r RestModel) GetName() string { return "holdings" }
func (r RestModel) GetID() string   { return r.Id }

func (r *RestModel) SetID(idStr string) error {
	r.Id = idStr
	return nil
}

func Extract(r RestModel) (Model, error) {
	return Model{
		id:         r.Id,
		worldId:    world.Id(r.WorldId),
		itcSn:      r.ItcSn,
		ownerId:    r.OwnerId,
		origin:     r.Origin,
		templateId: r.TemplateId,
		quantity:   r.Quantity,
	}, nil
}
