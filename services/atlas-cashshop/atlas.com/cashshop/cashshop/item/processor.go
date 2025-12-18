package item

import (
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/kafka/message/item"
	"atlas-cashshop/kafka/producer"
	itemProducer "atlas-cashshop/kafka/producer/item"
	"context"
	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"gorm.io/gorm"
)

type Processor interface {
	ByIdProvider(itemId uint32) model.Provider[Model]
	GetById(itemId uint32) (Model, error)
	Create(mb *message.Buffer) func(templateId uint32) func(quantity uint32) func(purchasedBy uint32) (Model, error)
	CreateAndEmit(templateId uint32, quantity uint32, purchasedBy uint32) (Model, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
	p   producer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
		p:   producer.ProviderImpl(l)(ctx),
	}
	return p
}

func (p *ProcessorImpl) ByIdProvider(id uint32) model.Provider[Model] {
	return model.Map(Make)(byIdEntityProvider(p.t.Id(), id)(p.db))
}

func (p *ProcessorImpl) GetById(id uint32) (Model, error) {
	return p.ByIdProvider(id)()
}

func (p *ProcessorImpl) Create(mb *message.Buffer) func(templateId uint32) func(quantity uint32) func(purchasedBy uint32) (Model, error) {
	return func(templateId uint32) func(quantity uint32) func(purchasedBy uint32) (Model, error) {
		return func(quantity uint32) func(purchasedBy uint32) (Model, error) {
			return func(purchasedBy uint32) (Model, error) {
				entity, err := createEntityProvider(p.t.Id(), templateId, quantity, purchasedBy)(p.db)()
				if err != nil {
					return Model{}, err
				}

				m, err := Make(entity)
				if err != nil {
					return Model{}, err
				}

				err = mb.Put(item.EnvStatusTopic, itemProducer.CreateStatusEventProvider(
					m.Id(),
					m.CashId(),
					m.TemplateId(),
					m.Quantity(),
					m.PurchasedBy(),
					m.Flag(),
				))
				if err != nil {
					return Model{}, err
				}

				return m, nil
			}
		}
	}
}

func (p *ProcessorImpl) CreateAndEmit(templateId uint32, quantity uint32, purchasedBy uint32) (Model, error) {
	return message.EmitWithResult[Model, uint32](p.p)(model.Flip(model.Flip(p.Create)(templateId))(quantity))(purchasedBy)
}
