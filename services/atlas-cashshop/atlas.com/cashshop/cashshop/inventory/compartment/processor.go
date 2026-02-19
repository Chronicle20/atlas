package compartment

import (
	"atlas-cashshop/cashshop/inventory/asset"
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/kafka/message/cashshop/compartment"
	"atlas-cashshop/kafka/producer"
	compartmentProducer "atlas-cashshop/kafka/producer/cashshop/inventory/compartment"
	"context"
	"errors"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

const DefaultCapacity = uint32(55)

type Processor interface {
	WithTransaction(tx *gorm.DB) Processor
	GetById(id uuid.UUID) (Model, error)
	ByIdProvider(id uuid.UUID) model.Provider[Model]
	GetByAccountIdAndType(accountId uint32, type_ CompartmentType) (Model, error)
	ByAccountIdAndTypeProvider(accountId uint32, type_ CompartmentType) model.Provider[Model]
	AllByAccountIdProvider(accountId uint32) model.Provider[[]Model]
	GetByAccountId(accountId uint32) ([]Model, error)
	Create(mb *message.Buffer) func(accountId uint32) func(type_ CompartmentType) func(capacity uint32) (Model, error)
	CreateAndEmit(accountId uint32, type_ CompartmentType, capacity uint32) (Model, error)
	UpdateCapacity(mb *message.Buffer) func(id uuid.UUID) func(capacity uint32) (Model, error)
	UpdateCapacityAndEmit(id uuid.UUID, capacity uint32) (Model, error)
	Delete(mb *message.Buffer) func(id uuid.UUID) error
	DeleteAndEmit(id uuid.UUID) error
	DeleteAllByAccountId(mb *message.Buffer) func(accountId uint32) error
	DeleteAllByAccountIdAndEmit(accountId uint32) error
	AcceptAndEmit(accountId uint32, characterId uint32, id uuid.UUID, type_ CompartmentType, cashId int64, templateId uint32, quantity uint32, commodityId uint32, purchasedBy uint32, flag uint16, transactionId uuid.UUID) error
	Accept(mb *message.Buffer) func(accountId uint32, characterId uint32, id uuid.UUID, type_ CompartmentType, cashId int64, templateId uint32, quantity uint32, commodityId uint32, purchasedBy uint32, flag uint16, transactionId uuid.UUID) error
	ReleaseAndEmit(accountId uint32, characterId uint32, id uuid.UUID, type_ CompartmentType, assetId uint32, transactionId uuid.UUID, cashId int64, templateId uint32) error
	Release(mb *message.Buffer) func(accountId uint32, characterId uint32, id uuid.UUID, type_ CompartmentType, assetId uint32, transactionId uuid.UUID, cashId int64, templateId uint32) error
}

type ProcessorImpl struct {
	l    logrus.FieldLogger
	ctx  context.Context
	db   *gorm.DB
	t    tenant.Model
	p    producer.Provider
	astP asset.Processor
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:    l,
		ctx:  ctx,
		db:   db,
		t:    tenant.MustFromContext(ctx),
		p:    producer.ProviderImpl(l)(ctx),
		astP: asset.NewProcessor(l, ctx, db),
	}
}

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{
		l:    p.l,
		ctx:  p.ctx,
		db:   tx,
		t:    p.t,
		p:    p.p,
		astP: p.astP,
	}
}

func (p *ProcessorImpl) DecorateAssets(m Model) Model {
	assets, err := p.astP.GetByCompartmentId(m.Id())
	if err != nil {
		return m
	}
	return Clone(m).SetAssets(assets).Build()
}

func (p *ProcessorImpl) GetById(id uuid.UUID) (Model, error) {
	return p.ByIdProvider(id)()
}

func (p *ProcessorImpl) ByIdProvider(id uuid.UUID) model.Provider[Model] {
	cp := model.Map[Entity, Model](Make)(getByIdProvider(id)(p.db.WithContext(p.ctx)))
	return model.Map(model.Decorate(model.Decorators(p.DecorateAssets)))(cp)
}

func (p *ProcessorImpl) ByAccountIdAndTypeProvider(accountId uint32, type_ CompartmentType) model.Provider[Model] {
	cp := model.Map[Entity, Model](Make)(getByAccountIdAndTypeProvider(accountId)(type_)(p.db.WithContext(p.ctx)))
	return model.Map(model.Decorate(model.Decorators(p.DecorateAssets)))(cp)
}

func (p *ProcessorImpl) GetByAccountIdAndType(accountId uint32, type_ CompartmentType) (Model, error) {
	return p.ByAccountIdAndTypeProvider(accountId, type_)()
}

func (p *ProcessorImpl) AllByAccountIdProvider(accountId uint32) model.Provider[[]Model] {
	cp := model.SliceMap[Entity, Model](Make)(getAllByAccountIdProvider(accountId)(p.db.WithContext(p.ctx)))(model.ParallelMap())
	return model.SliceMap(model.Decorate(model.Decorators(p.DecorateAssets)))(cp)(model.ParallelMap())
}

func (p *ProcessorImpl) GetByAccountId(accountId uint32) ([]Model, error) {
	return p.AllByAccountIdProvider(accountId)()
}

func (p *ProcessorImpl) Create(mb *message.Buffer) func(accountId uint32) func(type_ CompartmentType) func(capacity uint32) (Model, error) {
	return func(accountId uint32) func(type_ CompartmentType) func(capacity uint32) (Model, error) {
		return func(type_ CompartmentType) func(capacity uint32) (Model, error) {
			return func(capacity uint32) (Model, error) {
				p.l.Debugf("Creating compartment for account [%d] with type [%d] and capacity [%d].", accountId, type_, capacity)

				model, err := createEntity(p.db.WithContext(p.ctx), p.t, accountId, type_, capacity)
				if err != nil {
					p.l.WithError(err).Errorf("Could not create compartment for account [%d].", accountId)
					return Model{}, err
				}

				_ = mb.Put(compartment.EnvEventTopicStatus, compartmentProducer.CreateStatusEventProvider(model.Id(), byte(type_), capacity))

				return model, nil
			}
		}
	}
}

func (p *ProcessorImpl) CreateAndEmit(accountId uint32, type_ CompartmentType, capacity uint32) (Model, error) {
	mb := message.NewBuffer()
	m, err := p.Create(mb)(accountId)(type_)(capacity)
	if err != nil {
		return Model{}, err
	}

	for t, ms := range mb.GetAll() {
		if err = p.p(t)(model.FixedProvider(ms)); err != nil {
			return Model{}, err
		}
	}

	return m, nil
}

func (p *ProcessorImpl) UpdateCapacity(mb *message.Buffer) func(id uuid.UUID) func(capacity uint32) (Model, error) {
	return func(id uuid.UUID) func(capacity uint32) (Model, error) {
		return func(capacity uint32) (Model, error) {
			p.l.Debugf("Updating capacity of compartment [%s] to [%d].", id, capacity)

			model, err := updateCapacity(p.db.WithContext(p.ctx), id, capacity)
			if err != nil {
				p.l.WithError(err).Errorf("Could not update capacity of compartment [%s].", id)
				return Model{}, err
			}

			_ = mb.Put(compartment.EnvEventTopicStatus, compartmentProducer.UpdateStatusEventProvider(id, byte(model.Type()), capacity))

			return model, nil
		}
	}
}

func (p *ProcessorImpl) UpdateCapacityAndEmit(id uuid.UUID, capacity uint32) (Model, error) {
	mb := message.NewBuffer()
	m, err := p.UpdateCapacity(mb)(id)(capacity)
	if err != nil {
		return Model{}, err
	}

	for t, ms := range mb.GetAll() {
		if err = p.p(t)(model.FixedProvider(ms)); err != nil {
			return Model{}, err
		}
	}

	return m, nil
}

func (p *ProcessorImpl) Delete(mb *message.Buffer) func(id uuid.UUID) error {
	return func(id uuid.UUID) error {
		p.l.Debugf("Deleting compartment [%s].", id)

		m, err := p.ByIdProvider(id)()
		if err != nil {
			p.l.WithError(err).Errorf("Could not find compartment [%s] to delete.", id)
			return err
		}

		err = deleteEntity(p.db.WithContext(p.ctx), id)
		if err != nil {
			p.l.WithError(err).Errorf("Could not delete compartment [%s].", id)
			return err
		}

		_ = mb.Put(compartment.EnvEventTopicStatus, compartmentProducer.DeleteStatusEventProvider(id, byte(m.Type())))
		return nil
	}
}

func (p *ProcessorImpl) DeleteAndEmit(id uuid.UUID) error {
	mb := message.NewBuffer()
	err := p.Delete(mb)(id)
	if err != nil {
		return err
	}

	for t, ms := range mb.GetAll() {
		if err = p.p(t)(model.FixedProvider(ms)); err != nil {
			return err
		}
	}

	return nil
}

func (p *ProcessorImpl) DeleteAllByAccountId(mb *message.Buffer) func(accountId uint32) error {
	return func(accountId uint32) error {
		p.l.Debugf("Deleting all compartments for account [%d].", accountId)
		txErr := p.db.WithContext(p.ctx).Transaction(func(tx *gorm.DB) error {
			cscm, err := p.GetByAccountId(accountId)
			if err != nil {
				p.l.WithError(err).Errorf("Could not get compartments for account [%d].", accountId)
				return err
			}
			for _, ccm := range cscm {
				err = deleteEntity(p.db.WithContext(p.ctx), ccm.Id())
				if err != nil {
					p.l.WithError(err).Errorf("Could not delete compartment [%s].", ccm.Id())
					return err
				}

				_ = mb.Put(compartment.EnvEventTopicStatus, compartmentProducer.DeleteStatusEventProvider(ccm.Id(), byte(ccm.Type())))
			}
			return nil
		})
		if txErr != nil {
			p.l.WithError(txErr).Errorf("Could not delete all compartments for account [%d].", accountId)
			return txErr
		}
		return nil
	}
}

func (p *ProcessorImpl) DeleteAllByAccountIdAndEmit(accountId uint32) error {
	mb := message.NewBuffer()
	err := p.DeleteAllByAccountId(mb)(accountId)
	if err != nil {
		return err
	}

	for t, ms := range mb.GetAll() {
		if err = p.p(t)(model.FixedProvider(ms)); err != nil {
			return err
		}
	}

	return nil
}

func (p *ProcessorImpl) AcceptAndEmit(accountId uint32, characterId uint32, id uuid.UUID, type_ CompartmentType, cashId int64, templateId uint32, quantity uint32, commodityId uint32, purchasedBy uint32, flag uint16, transactionId uuid.UUID) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Accept(buf)(accountId, characterId, id, type_, cashId, templateId, quantity, commodityId, purchasedBy, flag, transactionId)
	})
}

func (p *ProcessorImpl) Accept(mb *message.Buffer) func(accountId uint32, characterId uint32, id uuid.UUID, type_ CompartmentType, cashId int64, templateId uint32, quantity uint32, commodityId uint32, purchasedBy uint32, flag uint16, transactionId uuid.UUID) error {
	return func(accountId uint32, characterId uint32, id uuid.UUID, type_ CompartmentType, cashId int64, templateId uint32, quantity uint32, commodityId uint32, purchasedBy uint32, flag uint16, transactionId uuid.UUID) error {
		p.l.Debugf("Handling accepting asset for account [%d], compartment [%s], type [%d], template [%d], cashId [%d].", accountId, id, type_, templateId, cashId)

		ccm, err := p.GetById(id)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to get compartment for ID [%s].", id)
			_ = mb.Put(compartment.EnvEventTopicStatus, compartmentProducer.ErrorStatusEventProvider(id, byte(type_), "UNKNOWN_ERROR", transactionId))
			return err
		}

		// Use defaults if not provided
		if quantity == 0 {
			quantity = 1
		}
		if purchasedBy == 0 {
			purchasedBy = accountId
		}

		// Create the flattened asset directly with preserved cashId
		createdAsset, err := p.astP.CreateWithCashId(mb)(id, cashId, templateId, commodityId, quantity, purchasedBy)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to create asset for compartment [%s] with cashId [%d] template ID [%d].", id, cashId, templateId)
			_ = mb.Put(compartment.EnvEventTopicStatus, compartmentProducer.ErrorStatusEventProvider(id, byte(type_), "ASSET_CREATION_FAILED", transactionId))
			return err
		}

		_ = mb.Put(compartment.EnvEventTopicStatus, compartmentProducer.AcceptedStatusEventProvider(accountId, characterId, id, byte(ccm.Type()), transactionId, createdAsset.Id()))
		return nil
	}
}

func (p *ProcessorImpl) ReleaseAndEmit(accountId uint32, characterId uint32, id uuid.UUID, type_ CompartmentType, assetId uint32, transactionId uuid.UUID, cashId int64, templateId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		return p.Release(buf)(accountId, characterId, id, type_, assetId, transactionId, cashId, templateId)
	})
}

func (p *ProcessorImpl) Release(mb *message.Buffer) func(accountId uint32, characterId uint32, id uuid.UUID, type_ CompartmentType, assetId uint32, transactionId uuid.UUID, cashId int64, templateId uint32) error {
	return func(accountId uint32, characterId uint32, id uuid.UUID, type_ CompartmentType, assetId uint32, transactionId uuid.UUID, cashId int64, templateId uint32) error {
		p.l.Debugf("Handling releasing asset for account [%d], compartment [%s], type [%d].", accountId, id.String(), type_)

		ccm, err := p.GetById(id)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to get compartment for ID [%s].", id)
			_ = mb.Put(compartment.EnvEventTopicStatus, compartmentProducer.ErrorStatusEventProvider(id, byte(type_), "UNKNOWN_ERROR", transactionId))
			return err
		}

		_, found := ccm.FindById(assetId)
		if !found {
			p.l.Errorf("Asset with ID [%d] not found in compartment [%s].", assetId, ccm.Id())
			_ = mb.Put(compartment.EnvEventTopicStatus, compartmentProducer.ErrorStatusEventProvider(id, byte(type_), "ITEM_NOT_FOUND", transactionId))
			return errors.New("asset not found")
		}

		err = p.astP.Release(mb)(assetId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to release asset [%d] for account [%d].", assetId, accountId)
			return err
		}

		_ = mb.Put(compartment.EnvEventTopicStatus, compartmentProducer.ReleasedStatusEventProvider(accountId, characterId, id, byte(type_), transactionId, assetId, cashId, templateId))
		return nil
	}
}
