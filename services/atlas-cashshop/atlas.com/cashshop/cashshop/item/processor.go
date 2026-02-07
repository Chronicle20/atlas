package item

import (
	"atlas-cashshop/cashshop/commodity"
	"atlas-cashshop/configuration"
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
	Create(mb *message.Buffer) func(templateId uint32) func(commodityId uint32) func(quantity uint32) func(purchasedBy uint32) (Model, error)
	CreateAndEmit(templateId uint32, commodityId uint32, quantity uint32, purchasedBy uint32) (Model, error)
	CreateWithCashId(mb *message.Buffer) func(cashId int64) func(templateId uint32) func(commodityId uint32) func(quantity uint32) func(purchasedBy uint32) (Model, error)
	CreateWithCashIdAndEmit(cashId int64, templateId uint32, commodityId uint32, quantity uint32, purchasedBy uint32) (Model, error)
	UpdateQuantity(itemId uint32, quantity uint32) error
	Delete(mb *message.Buffer) func(itemId uint32) error
	DeleteAndEmit(itemId uint32) error
	Expire(mb *message.Buffer) func(itemId uint32) func(replaceItemId uint32) func(replaceMessage string) error
	ExpireAndEmit(itemId uint32, replaceItemId uint32, replaceMessage string) error
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
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
		p:   producer.ProviderImpl(l)(ctx),
		cp:  commodity.NewProcessor(l, ctx),
	}
	return p
}

func (p *ProcessorImpl) ByIdProvider(id uint32) model.Provider[Model] {
	return model.Map(Make)(byIdEntityProvider(p.t.Id(), id)(p.db))
}

func (p *ProcessorImpl) GetById(id uint32) (Model, error) {
	return p.ByIdProvider(id)()
}

func (p *ProcessorImpl) Create(mb *message.Buffer) func(templateId uint32) func(commodityId uint32) func(quantity uint32) func(purchasedBy uint32) (Model, error) {
	return func(templateId uint32) func(commodityId uint32) func(quantity uint32) func(purchasedBy uint32) (Model, error) {
		return func(commodityId uint32) func(quantity uint32) func(purchasedBy uint32) (Model, error) {
			return func(quantity uint32) func(purchasedBy uint32) (Model, error) {
				return func(purchasedBy uint32) (Model, error) {
					// Fetch commodity to get period
					var period uint32 = 30 // default to 30 days if commodity not found
					if commodityId != 0 {
						c, err := p.cp.GetById(commodityId)
						if err == nil {
							period = c.Period()
						} else {
							p.l.WithError(err).Warnf("Failed to fetch commodity %d, using default period", commodityId)
						}
					}

					// Get hourly expirations config
					hourlyConfig := configuration.GetHourlyExpirations(p.l, p.ctx, p.t.Id())

					entity, err := create(p.t.Id(), templateId, commodityId, quantity, purchasedBy, period, hourlyConfig)(p.db)()
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
}

func (p *ProcessorImpl) CreateAndEmit(templateId uint32, commodityId uint32, quantity uint32, purchasedBy uint32) (Model, error) {
	return message.EmitWithResult[Model, uint32](p.p)(model.Flip(model.Flip(model.Flip(p.Create)(templateId))(commodityId))(quantity))(purchasedBy)
}

func (p *ProcessorImpl) CreateWithCashId(mb *message.Buffer) func(cashId int64) func(templateId uint32) func(commodityId uint32) func(quantity uint32) func(purchasedBy uint32) (Model, error) {
	return func(cashId int64) func(templateId uint32) func(commodityId uint32) func(quantity uint32) func(purchasedBy uint32) (Model, error) {
		return func(templateId uint32) func(commodityId uint32) func(quantity uint32) func(purchasedBy uint32) (Model, error) {
			return func(commodityId uint32) func(quantity uint32) func(purchasedBy uint32) (Model, error) {
				return func(quantity uint32) func(purchasedBy uint32) (Model, error) {
					return func(purchasedBy uint32) (Model, error) {
						// Fetch commodity to get period
						var period uint32 = 30 // default to 30 days if commodity not found
						if commodityId != 0 {
							c, err := p.cp.GetById(commodityId)
							if err == nil {
								period = c.Period()
							} else {
								p.l.WithError(err).Warnf("Failed to fetch commodity %d, using default period", commodityId)
							}
						}

						// Get hourly expirations config
						hourlyConfig := configuration.GetHourlyExpirations(p.l, p.ctx, p.t.Id())

						entity, err := findOrCreateByCashId(p.t.Id(), cashId, templateId, commodityId, quantity, purchasedBy, period, hourlyConfig)(p.db)()
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
	}
}

func (p *ProcessorImpl) CreateWithCashIdAndEmit(cashId int64, templateId uint32, commodityId uint32, quantity uint32, purchasedBy uint32) (Model, error) {
	return message.EmitWithResult[Model, uint32](p.p)(model.Flip(model.Flip(model.Flip(model.Flip(p.CreateWithCashId)(cashId))(templateId))(commodityId))(quantity))(purchasedBy)
}

func (p *ProcessorImpl) UpdateQuantity(itemId uint32, quantity uint32) error {
	return updateQuantity(p.db, p.t.Id(), itemId, quantity)
}

func (p *ProcessorImpl) Delete(_ *message.Buffer) func(itemId uint32) error {
	return func(itemId uint32) error {
		return deleteById(p.db, p.t.Id(), itemId)
	}
}

func (p *ProcessorImpl) DeleteAndEmit(itemId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Delete(buf)(itemId)
	})
}

func (p *ProcessorImpl) Expire(mb *message.Buffer) func(itemId uint32) func(replaceItemId uint32) func(replaceMessage string) error {
	return func(itemId uint32) func(replaceItemId uint32) func(replaceMessage string) error {
		return func(replaceItemId uint32) func(replaceMessage string) error {
			return func(replaceMessage string) error {
				p.l.Debugf("Expiring cash shop item [%d].", itemId)

				// Delete the item
				err := deleteById(p.db, p.t.Id(), itemId)
				if err != nil {
					return err
				}

				// Emit EXPIRED status event
				err = mb.Put(item.EnvStatusTopic, itemProducer.ExpireStatusEventProvider(replaceItemId, replaceMessage))
				if err != nil {
					return err
				}

				// If there's a replacement item, create it
				if replaceItemId > 0 {
					p.l.Debugf("Creating replacement item [%d] for expired cash shop item [%d].", replaceItemId, itemId)
					_, err = p.Create(mb)(replaceItemId)(0)(1)(0)
					if err != nil {
						p.l.WithError(err).Warnf("Failed to create replacement item [%d] for expired cash shop item.", replaceItemId)
						return nil // Don't fail the expiration if we can't create the replacement
					}
				}

				p.l.Debugf("Expired cash shop item [%d].", itemId)
				return nil
			}
		}
	}
}

func (p *ProcessorImpl) ExpireAndEmit(itemId uint32, replaceItemId uint32, replaceMessage string) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Expire(buf)(itemId)(replaceItemId)(replaceMessage)
	})
}
