package pet

import (
	"atlas-pets/pet/exclude"
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
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

func Transform(ctx context.Context) func(m Model) (RestModel, error) {
	t := tenant.MustFromContext(ctx)
	return func(m Model) (RestModel, error) {
		tm := GetTemporalRegistry().GetById(ctx, t, m.Id())
		if tm == nil {
			return RestModel{}, errors.New("temporal data not found")
		}
		es, err := model.SliceMap(exclude.Transform)(model.FixedProvider(m.Excludes()))(model.ParallelMap())()
		if err != nil {
			return RestModel{}, err
		}

		return RestModel{
			Id:         m.Id(),
			CashId:     m.CashId(),
			TemplateId: m.TemplateId(),
			Name:       m.Name(),
			Level:      m.Level(),
			Closeness:  m.Closeness(),
			Fullness:   m.Fullness(),
			Expiration: m.Expiration(),
			OwnerId:    m.OwnerId(),
			Slot:       m.Slot(),
			X:          tm.X(),
			Y:          tm.Y(),
			Stance:     tm.Stance(),
			FH:         tm.FH(),
			Excludes:   es,
			Flag:       m.Flag(),
			PurchaseBy: m.PurchaseBy(),
		}, nil
	}
}

func Extract(rm RestModel) (Model, error) {
	es, err := model.SliceMap(exclude.Extract)(model.FixedProvider(rm.Excludes))(model.ParallelMap())()
	if err != nil {
		return Model{}, err
	}

	return NewModelBuilder(rm.Id, rm.CashId, rm.TemplateId, rm.Name, rm.OwnerId).
		SetLevel(rm.Level).
		SetCloseness(rm.Closeness).
		SetFullness(rm.Fullness).
		SetExpiration(rm.Expiration).
		SetSlot(rm.Slot).
		SetExcludes(es).
		SetFlag(rm.Flag).
		SetPurchaseBy(rm.PurchaseBy).
		Build()
}
