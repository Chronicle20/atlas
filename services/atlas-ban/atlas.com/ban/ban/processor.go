package ban

import (
	"atlas-ban/kafka/message"
	ban2 "atlas-ban/kafka/message/ban"
	"atlas-ban/kafka/producer"
	"context"
	"strconv"

	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	Create(banType BanType, value string, reason string, reasonCode byte, permanent bool, expiresAt int64, issuedBy string) (Model, error)
	CreateAndEmit(banType BanType, value string, reason string, reasonCode byte, permanent bool, expiresAt int64, issuedBy string) (Model, error)
	Delete(banId uint32) error
	DeleteAndEmit(banId uint32) error
	GetById(banId uint32) (Model, error)
	GetByTenant() ([]Model, error)
	GetByType(banType BanType) ([]Model, error)
	CheckBan(ip string, hwid string, accountId uint32) (*Model, error)
	ByIdProvider(banId uint32) model.Provider[Model]
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
	p   producer.Provider
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
		p:   producer.ProviderImpl(l)(ctx),
	}
}

func (p *ProcessorImpl) Create(banType BanType, value string, reason string, reasonCode byte, permanent bool, expiresAt int64, issuedBy string) (Model, error) {
	p.l.Debugf("Creating ban type [%d] value [%s] reason [%s].", banType, value, reason)
	m, err := create(p.db)(p.t, banType, value, reason, reasonCode, permanent, expiresAt, issuedBy)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to create ban for value [%s].", value)
		return Model{}, err
	}
	p.l.Infof("Created ban [%d] type [%d] value [%s].", m.Id(), banType, value)
	return m, nil
}

func (p *ProcessorImpl) CreateAndEmit(banType BanType, value string, reason string, reasonCode byte, permanent bool, expiresAt int64, issuedBy string) (Model, error) {
	var result Model
	err := message.Emit(p.p)(func(buf *message.Buffer) error {
		m, err := p.Create(banType, value, reason, reasonCode, permanent, expiresAt, issuedBy)
		if err != nil {
			return err
		}
		result = m
		return buf.Put(ban2.EnvEventTopicStatus, createdEventProvider(m.Id()))
	})
	return result, err
}

func (p *ProcessorImpl) Delete(banId uint32) error {
	p.l.Debugf("Deleting ban [%d].", banId)
	err := deleteById(p.db)(p.t, banId)
	if err != nil {
		p.l.WithError(err).Errorf("Unable to delete ban [%d].", banId)
		return err
	}
	p.l.Infof("Deleted ban [%d].", banId)
	return nil
}

func (p *ProcessorImpl) DeleteAndEmit(banId uint32) error {
	return message.Emit(p.p)(func(buf *message.Buffer) error {
		err := p.Delete(banId)
		if err != nil {
			return err
		}
		return buf.Put(ban2.EnvEventTopicStatus, deletedEventProvider(banId))
	})
}

func (p *ProcessorImpl) GetById(banId uint32) (Model, error) {
	return p.ByIdProvider(banId)()
}

func (p *ProcessorImpl) ByIdProvider(banId uint32) model.Provider[Model] {
	return model.Map(Make)(entityById(p.t, banId)(p.db))
}

func (p *ProcessorImpl) GetByTenant() ([]Model, error) {
	return model.SliceMap(Make)(entitiesByTenant(p.t)(p.db))(model.ParallelMap())()
}

func (p *ProcessorImpl) GetByType(banType BanType) ([]Model, error) {
	return model.SliceMap(Make)(entitiesByType(p.t, banType)(p.db))(model.ParallelMap())()
}

func (p *ProcessorImpl) CheckBan(ip string, hwid string, accountId uint32) (*Model, error) {
	// Check exact IP bans
	if ip != "" {
		bans, err := model.SliceMap(Make)(activeExactBans(p.t, BanTypeIP, ip)(p.db))(model.ParallelMap())()
		if err != nil {
			return nil, err
		}
		if len(bans) > 0 {
			return &bans[0], nil
		}

		// Check CIDR range bans
		allIPBans, err := model.SliceMap(Make)(activeIPBans(p.t)(p.db))(model.ParallelMap())()
		if err != nil {
			return nil, err
		}
		for _, b := range allIPBans {
			if isCIDR(b.Value()) && ipMatchesCIDR(ip, b.Value()) {
				return &b, nil
			}
		}
	}

	// Check HWID bans
	if hwid != "" {
		bans, err := model.SliceMap(Make)(activeExactBans(p.t, BanTypeHWID, hwid)(p.db))(model.ParallelMap())()
		if err != nil {
			return nil, err
		}
		if len(bans) > 0 {
			return &bans[0], nil
		}
	}

	// Check account bans
	if accountId > 0 {
		accountValue := strconv.FormatUint(uint64(accountId), 10)
		bans, err := model.SliceMap(Make)(activeExactBans(p.t, BanTypeAccount, accountValue)(p.db))(model.ParallelMap())()
		if err != nil {
			return nil, err
		}
		if len(bans) > 0 {
			return &bans[0], nil
		}
	}

	return nil, nil
}
