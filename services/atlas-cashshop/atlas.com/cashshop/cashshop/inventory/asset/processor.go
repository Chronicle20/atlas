package asset

import (
	"atlas-cashshop/cashshop/commodity"
	"atlas-cashshop/configuration"
	"atlas-cashshop/database"
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/kafka/message/item"
	"atlas-cashshop/kafka/producer"
	itemProducer "atlas-cashshop/kafka/producer/item"
	"context"

	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	ByIdProvider(id uint32) model.Provider[Model]
	GetById(id uint32) (Model, error)
	ByCompartmentIdProvider(compartmentId uuid.UUID) model.Provider[[]Model]
	GetByCompartmentId(compartmentId uuid.UUID) ([]Model, error)
	Create(mb *message.Buffer) func(compartmentId uuid.UUID, templateId uint32, commodityId uint32, quantity uint32, purchasedBy uint32) (Model, error)
	CreateAndEmit(compartmentId uuid.UUID, templateId uint32, commodityId uint32, quantity uint32, purchasedBy uint32) (Model, error)
	CreateWithCashId(mb *message.Buffer) func(compartmentId uuid.UUID, cashId int64, templateId uint32, commodityId uint32, quantity uint32, purchasedBy uint32) (Model, error)
	CreateWithCashIdAndEmit(compartmentId uuid.UUID, cashId int64, templateId uint32, commodityId uint32, quantity uint32, purchasedBy uint32) (Model, error)
	UpdateQuantity(id uint32, quantity uint32) error
	Delete(mb *message.Buffer) func(id uint32) error
	DeleteAndEmit(id uint32) error
	Release(mb *message.Buffer) func(id uint32) error
	ReleaseAndEmit(id uint32) error
	Expire(mb *message.Buffer) func(id uint32, replaceItemId uint32, replaceMessage string) error
	ExpireAndEmit(id uint32, replaceItemId uint32, replaceMessage string) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
	p   producer.Provider
	cp  commodity.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
		p:   producer.ProviderImpl(l)(ctx),
		cp:  commodity.NewProcessor(l, ctx),
	}
}

func (p *ProcessorImpl) ByIdProvider(id uint32) model.Provider[Model] {
	return model.Map(Make)(getByIdProvider(p.t.Id())(id)(p.db))
}

func (p *ProcessorImpl) GetById(id uint32) (Model, error) {
	return p.ByIdProvider(id)()
}

func (p *ProcessorImpl) ByCompartmentIdProvider(compartmentId uuid.UUID) model.Provider[[]Model] {
	return model.SliceMap(Make)(getByCompartmentIdProvider(p.t.Id())(compartmentId)(p.db))(model.ParallelMap())
}

func (p *ProcessorImpl) GetByCompartmentId(compartmentId uuid.UUID) ([]Model, error) {
	return p.ByCompartmentIdProvider(compartmentId)()
}

func (p *ProcessorImpl) Create(mb *message.Buffer) func(compartmentId uuid.UUID, templateId uint32, commodityId uint32, quantity uint32, purchasedBy uint32) (Model, error) {
	return func(compartmentId uuid.UUID, templateId uint32, commodityId uint32, quantity uint32, purchasedBy uint32) (Model, error) {
		var result Model
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			var period uint32 = 30
			if commodityId != 0 {
				c, err := p.cp.GetById(commodityId)
				if err == nil {
					period = c.Period()
				} else {
					p.l.WithError(err).Warnf("Failed to fetch commodity %d, using default period", commodityId)
				}
			}

			hourlyConfig := configuration.GetHourlyExpirations(p.l, p.ctx, p.t.Id())
			expiration := CalculateExpiration(period, templateId, hourlyConfig)

			entity, err := create(tx, p.t.Id(), compartmentId, templateId, commodityId, quantity, purchasedBy, expiration)()
			if err != nil {
				p.l.WithError(err).Errorf("Unable to create asset for compartment [%s] template [%d].", compartmentId, templateId)
				return err
			}

			m, err := Make(entity)
			if err != nil {
				return err
			}
			result = m

			return mb.Put(item.EnvStatusTopic, itemProducer.CreateStatusEventProvider(
				m.Id(),
				m.CashId(),
				m.TemplateId(),
				m.Quantity(),
				m.PurchasedBy(),
				m.Flag(),
			))
		})
		if txErr != nil {
			return Model{}, txErr
		}
		return result, nil
	}
}

func (p *ProcessorImpl) CreateAndEmit(compartmentId uuid.UUID, templateId uint32, commodityId uint32, quantity uint32, purchasedBy uint32) (Model, error) {
	var result Model
	err := message.Emit(p.p)(func(buf *message.Buffer) error {
		var e error
		result, e = p.Create(buf)(compartmentId, templateId, commodityId, quantity, purchasedBy)
		return e
	})
	return result, err
}

func (p *ProcessorImpl) CreateWithCashId(mb *message.Buffer) func(compartmentId uuid.UUID, cashId int64, templateId uint32, commodityId uint32, quantity uint32, purchasedBy uint32) (Model, error) {
	return func(compartmentId uuid.UUID, cashId int64, templateId uint32, commodityId uint32, quantity uint32, purchasedBy uint32) (Model, error) {
		var result Model
		txErr := database.ExecuteTransaction(p.db, func(tx *gorm.DB) error {
			var period uint32 = 30
			if commodityId != 0 {
				c, err := p.cp.GetById(commodityId)
				if err == nil {
					period = c.Period()
				} else {
					p.l.WithError(err).Warnf("Failed to fetch commodity %d, using default period", commodityId)
				}
			}

			hourlyConfig := configuration.GetHourlyExpirations(p.l, p.ctx, p.t.Id())
			expiration := CalculateExpiration(period, templateId, hourlyConfig)

			entity, err := findOrCreateByCashId(tx, p.t.Id(), cashId, compartmentId, templateId, commodityId, quantity, purchasedBy, expiration)()
			if err != nil {
				return err
			}

			m, err := Make(entity)
			if err != nil {
				return err
			}
			result = m

			return mb.Put(item.EnvStatusTopic, itemProducer.CreateStatusEventProvider(
				m.Id(),
				m.CashId(),
				m.TemplateId(),
				m.Quantity(),
				m.PurchasedBy(),
				m.Flag(),
			))
		})
		if txErr != nil {
			return Model{}, txErr
		}
		return result, nil
	}
}

func (p *ProcessorImpl) CreateWithCashIdAndEmit(compartmentId uuid.UUID, cashId int64, templateId uint32, commodityId uint32, quantity uint32, purchasedBy uint32) (Model, error) {
	var result Model
	err := message.Emit(p.p)(func(buf *message.Buffer) error {
		var e error
		result, e = p.CreateWithCashId(buf)(compartmentId, cashId, templateId, commodityId, quantity, purchasedBy)
		return e
	})
	return result, err
}

func (p *ProcessorImpl) UpdateQuantity(id uint32, quantity uint32) error {
	return updateQuantity(p.db, p.t.Id(), id, quantity)
}

func (p *ProcessorImpl) Delete(_ *message.Buffer) func(id uint32) error {
	return func(id uint32) error {
		return deleteById(p.db, p.t.Id(), id)
	}
}

func (p *ProcessorImpl) DeleteAndEmit(id uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Delete(buf)(id)
	})
}

func (p *ProcessorImpl) Release(_ *message.Buffer) func(id uint32) error {
	return func(id uint32) error {
		p.l.Debugf("Releasing asset [%d].", id)
		return deleteById(p.db, p.t.Id(), id)
	}
}

func (p *ProcessorImpl) ReleaseAndEmit(id uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Release(buf)(id)
	})
}

func (p *ProcessorImpl) Expire(mb *message.Buffer) func(id uint32, replaceItemId uint32, replaceMessage string) error {
	return func(id uint32, replaceItemId uint32, replaceMessage string) error {
		p.l.Debugf("Expiring cash shop asset [%d].", id)

		a, err := p.GetById(id)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to find asset [%d] for expiration.", id)
			return err
		}

		err = deleteById(p.db, p.t.Id(), id)
		if err != nil {
			return err
		}

		err = mb.Put(item.EnvStatusTopic, itemProducer.ExpireStatusEventProvider(replaceItemId, replaceMessage))
		if err != nil {
			return err
		}

		if replaceItemId > 0 {
			p.l.Debugf("Creating replacement asset [%d] for expired cash shop asset [%d].", replaceItemId, id)
			_, err = p.Create(mb)(a.CompartmentId(), replaceItemId, 0, 1, 0)
			if err != nil {
				p.l.WithError(err).Warnf("Failed to create replacement asset [%d] for expired cash shop asset.", replaceItemId)
				return nil
			}
		}

		p.l.Debugf("Expired cash shop asset [%d].", id)
		return nil
	}
}

func (p *ProcessorImpl) ExpireAndEmit(id uint32, replaceItemId uint32, replaceMessage string) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Expire(buf)(id, replaceItemId, replaceMessage)
	})
}
