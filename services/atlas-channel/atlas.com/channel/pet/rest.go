package pet

import (
	"atlas-channel/pet/exclude"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

type RestModel struct {
	Id         uint32              `json:"-"`
	CashId     uint64              `json:"cashId"`
	TemplateId uint32              `json:"templateId"`
	Name       string              `json:"name"`
	Level      byte                `json:"level"`
	Closeness  uint16              `json:"closeness"`
	Fullness   byte                `json:"fullness"`
	Expiration time.Time           `json:"expiration"`
	OwnerId    uint32              `json:"ownerId"`
	Lead       bool                `json:"lead"`
	Slot       int8                `json:"slot"`
	X          int16               `json:"x"`
	Y          int16               `json:"y"`
	Stance     byte                `json:"stance"`
	FH         int16               `json:"fh"`
	Excludes   []exclude.RestModel `json:"excludes"`
	Flag       uint16              `json:"flag"`
	PurchaseBy uint32              `json:"purchaseBy"`
}

func (r RestModel) GetName() string {
	return "pets"
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

// Transform converts the domain Model into the wire RestModel.
//
// Lead is deliberately left at its zero value: Model has no lead field
// (Model.Lead() derives slot == 0) and Extract never reads RestModel.Lead,
// so there is nothing on Model for Transform to read it back from.
func Transform(m Model) (RestModel, error) {
	es := make([]exclude.RestModel, 0, len(m.excludes))
	for _, e := range m.excludes {
		erm, err := exclude.Transform(e)
		if err != nil {
			return RestModel{}, err
		}
		es = append(es, erm)
	}

	return RestModel{
		Id:         m.id,
		CashId:     m.cashId,
		TemplateId: m.templateId,
		Name:       m.name,
		Level:      m.level,
		Closeness:  m.closeness,
		Fullness:   m.fullness,
		Expiration: m.expiration,
		OwnerId:    m.ownerId,
		Slot:       m.slot,
		X:          m.x,
		Y:          m.y,
		Stance:     m.stance,
		FH:         m.fh,
		Excludes:   es,
		Flag:       m.flag,
		PurchaseBy: m.purchaseBy,
	}, nil
}

func Extract(rm RestModel) (Model, error) {
	es, err := model.SliceMap(exclude.Extract)(model.FixedProvider(rm.Excludes))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}

	return Model{
		id:         rm.Id,
		cashId:     rm.CashId,
		templateId: rm.TemplateId,
		name:       rm.Name,
		level:      rm.Level,
		closeness:  rm.Closeness,
		fullness:   rm.Fullness,
		expiration: rm.Expiration,
		ownerId:    rm.OwnerId,
		slot:       rm.Slot,
		x:          rm.X,
		y:          rm.Y,
		stance:     rm.Stance,
		fh:         rm.FH,
		excludes:   es,
		flag:       rm.Flag,
		purchaseBy: rm.PurchaseBy,
	}, nil
}
