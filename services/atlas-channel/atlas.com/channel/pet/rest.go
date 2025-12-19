package pet

import (
	"atlas-channel/pet/exclude"
	"github.com/Chronicle20/atlas-model/model"
	"strconv"
	"time"
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
