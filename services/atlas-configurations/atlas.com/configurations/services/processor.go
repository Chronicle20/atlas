package services

import (
	"atlas-configurations/services/service"
	"context"
	"encoding/json"
	"errors"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type ServiceType string

const (
	ServiceTypeLogin   = ServiceType("login-service")
	ServiceTypeChannel = ServiceType("channel-service")
	ServiceTypeDrops   = ServiceType("drops-service")
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
		db:  db,
	}
	return p
}

func (p *Processor) ByIdProvider(id uuid.UUID) model.Provider[interface{}] {
	return model.Map(Make)(byIdEntityProvider(p.ctx)(id)(p.db))
}

func (p *Processor) AllProvider() model.Provider[[]interface{}] {
	return model.SliceMap(Make)(allEntityProvider(p.ctx)(p.db))()
}

func Make(e Entity) (interface{}, error) {
	if e.Type == ServiceTypeLogin {
		var rm service.LoginRestModel
		err := json.Unmarshal(e.Data, &rm)
		if err != nil {
			return nil, err
		}
		rm.Id = e.Id.String()
		return rm, nil
	} else if e.Type == ServiceTypeChannel {
		var rm service.ChannelRestModel
		err := json.Unmarshal(e.Data, &rm)
		if err != nil {
			return nil, err
		}
		rm.Id = e.Id.String()
		return rm, nil
	} else if e.Type == ServiceTypeDrops {
		var rm service.GenericRestModel
		err := json.Unmarshal(e.Data, &rm)
		if err != nil {
			return nil, err
		}
		rm.Id = e.Id.String()
		return rm, nil
	}
	return nil, errors.New("invalid service type")
}

func (p *Processor) GetAll() ([]interface{}, error) {
	return p.AllProvider()()
}

func (p *Processor) GetById(id uuid.UUID) (interface{}, error) {
	return p.ByIdProvider(id)()
}
