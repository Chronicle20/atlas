package wallet

import (
	"atlas-cashshop/kafka/message"
	"atlas-cashshop/kafka/message/wallet"
	"atlas-cashshop/kafka/producer"
	wallet2 "atlas-cashshop/kafka/producer/wallet"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	WithTransaction(tx *gorm.DB) Processor
	ByAccountIdProvider(accountId uint32) model.Provider[Model]
	GetByAccountId(accountId uint32) (Model, error)
	Create(mb *message.Buffer) func(accountId uint32) func(credit uint32) func(points uint32) func(prepaid uint32) (Model, error)
	CreateAndEmit(accountId uint32, credit uint32, points uint32, prepaid uint32) (Model, error)
	Update(mb *message.Buffer) func(accountId uint32) func(credit uint32) func(points uint32) func(prepaid uint32) (Model, error)
	UpdateAndEmit(accountId uint32, credit uint32, points uint32, prepaid uint32) (Model, error)
	AdjustCurrency(accountId uint32, currencyType uint32, amount int32) (Model, error)
	Delete(mb *message.Buffer) func(accountId uint32) error
	DeleteAndEmit(accountId uint32) error
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

func (p *ProcessorImpl) WithTransaction(tx *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   p.l,
		ctx: p.ctx,
		db:  tx,
		t:   p.t,
		p:   p.p,
	}
}

func (p *ProcessorImpl) ByAccountIdProvider(accountId uint32) model.Provider[Model] {
	return model.Map(Make)(byAccountIdEntityProvider(p.t.Id(), accountId)(p.db))
}

func (p *ProcessorImpl) GetByAccountId(accountId uint32) (Model, error) {
	return p.ByAccountIdProvider(accountId)()
}

func (p *ProcessorImpl) Create(mb *message.Buffer) func(accountId uint32) func(credit uint32) func(points uint32) func(prepaid uint32) (Model, error) {
	return func(accountId uint32) func(credit uint32) func(points uint32) func(prepaid uint32) (Model, error) {
		return func(credit uint32) func(points uint32) func(prepaid uint32) (Model, error) {
			return func(points uint32) func(prepaid uint32) (Model, error) {
				return func(prepaid uint32) (Model, error) {
					p.l.Debugf("Initializing wallet information for account [%d]. Credit [%d], Points [%d], and Prepaid [%d].", accountId, credit, points, prepaid)
					c, err := createEntity(p.db, p.t, accountId, credit, points, prepaid)
					if err != nil {
						p.l.WithError(err).Errorf("Could not create wallet information for account [%d].", accountId)
						return Model{}, err
					}

					_ = mb.Put(wallet.EnvEventTopicStatus, wallet2.CreateStatusEventProvider(accountId, credit, points, prepaid))
					return c, err
				}
			}
		}
	}
}

func (p *ProcessorImpl) CreateAndEmit(accountId uint32, credit uint32, points uint32, prepaid uint32) (Model, error) {
	return message.EmitWithResult[Model, uint32](p.p)(model.Flip(model.Flip(model.Flip(p.Create)(accountId))(credit))(points))(prepaid)
}

func (p *ProcessorImpl) Update(mb *message.Buffer) func(accountId uint32) func(credit uint32) func(points uint32) func(prepaid uint32) (Model, error) {
	return func(accountId uint32) func(credit uint32) func(points uint32) func(prepaid uint32) (Model, error) {
		return func(credit uint32) func(points uint32) func(prepaid uint32) (Model, error) {
			return func(points uint32) func(prepaid uint32) (Model, error) {
				return func(prepaid uint32) (Model, error) {
					p.l.Debugf("Updating wallet information for account [%d]. Credit [%d], Points [%d], and Prepaid [%d].", accountId, credit, points, prepaid)
					c, err := updateEntity(p.db, p.t, accountId, credit, points, prepaid)
					if err != nil {
						p.l.WithError(err).Errorf("Could not update wallet information for account [%d].", accountId)
						return Model{}, err
					}

					_ = mb.Put(wallet.EnvEventTopicStatus, wallet2.UpdateStatusEventProvider(accountId, credit, points, prepaid))
					return c, err
				}
			}
		}
	}
}

func (p *ProcessorImpl) UpdateAndEmit(accountId uint32, credit uint32, points uint32, prepaid uint32) (Model, error) {
	return message.EmitWithResult[Model, uint32](p.p)(model.Flip(model.Flip(model.Flip(p.Update)(accountId))(credit))(points))(prepaid)
}

func (p *ProcessorImpl) AdjustCurrency(accountId uint32, currencyType uint32, amount int32) (Model, error) {
	// Get current wallet
	m, err := p.GetByAccountId(accountId)
	if err != nil {
		p.l.WithError(err).Errorf("Could not get wallet for account [%d] to adjust currency.", accountId)
		return Model{}, err
	}

	// Calculate new values based on currency type
	newCredit := m.Credit()
	newPoints := m.Points()
	newPrepaid := m.Prepaid()

	switch currencyType {
	case 1: // Credit
		if amount < 0 && uint32(-amount) > newCredit {
			p.l.Errorf("Insufficient credit balance for account [%d]. Current: %d, Requested deduction: %d", accountId, newCredit, -amount)
			return Model{}, fmt.Errorf("insufficient credit balance")
		}
		newCredit = uint32(int32(newCredit) + amount)
	case 2: // Points
		if amount < 0 && uint32(-amount) > newPoints {
			p.l.Errorf("Insufficient points balance for account [%d]. Current: %d, Requested deduction: %d", accountId, newPoints, -amount)
			return Model{}, fmt.Errorf("insufficient points balance")
		}
		newPoints = uint32(int32(newPoints) + amount)
	case 3: // Prepaid
		if amount < 0 && uint32(-amount) > newPrepaid {
			p.l.Errorf("Insufficient prepaid balance for account [%d]. Current: %d, Requested deduction: %d", accountId, newPrepaid, -amount)
			return Model{}, fmt.Errorf("insufficient prepaid balance")
		}
		newPrepaid = uint32(int32(newPrepaid) + amount)
	default:
		p.l.Errorf("Invalid currency type [%d] for account [%d].", currencyType, accountId)
		return Model{}, fmt.Errorf("invalid currency type: %d", currencyType)
	}

	p.l.Debugf("Adjusting currency for account [%d]. Type: %d, Amount: %d. New values - Credit: %d, Points: %d, Prepaid: %d",
		accountId, currencyType, amount, newCredit, newPoints, newPrepaid)

	return p.UpdateAndEmit(accountId, newCredit, newPoints, newPrepaid)
}

func (p *ProcessorImpl) Delete(mb *message.Buffer) func(accountId uint32) error {
	return func(accountId uint32) error {
		p.l.Debugf("Account [%d] was deleted. Cleaning up wallet information...", accountId)
		err := deleteEntity(p.ctx)(p.db, p.t.Id(), accountId)
		if err != nil {
			return err
		}

		_ = mb.Put(wallet.EnvEventTopicStatus, wallet2.DeleteStatusEventProvider(accountId))
		return nil
	}
}

func (p *ProcessorImpl) DeleteAndEmit(accountId uint32) error {
	return message.Emit(p.p)(model.Flip(p.Delete)(accountId))
}
