package services

import (
	database "github.com/Chronicle20/atlas-database"
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

var validServiceTypes = map[ServiceType]bool{
	ServiceTypeLogin:   true,
	ServiceTypeChannel: true,
	ServiceTypeDrops:   true,
}

func IsValidServiceType(t string) bool {
	return validServiceTypes[ServiceType(t)]
}

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
		rm.Type = string(e.Type)
		return rm, nil
	} else if e.Type == ServiceTypeChannel {
		var rm service.ChannelRestModel
		err := json.Unmarshal(e.Data, &rm)
		if err != nil {
			return nil, err
		}
		rm.Id = e.Id.String()
		rm.Type = string(e.Type)
		return rm, nil
	} else if e.Type == ServiceTypeDrops {
		var rm service.GenericRestModel
		err := json.Unmarshal(e.Data, &rm)
		if err != nil {
			return nil, err
		}
		rm.Id = e.Id.String()
		rm.Type = string(e.Type)
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

func (p *Processor) Create(input service.InputRestModel) (uuid.UUID, error) {
	serviceType := ServiceType(input.Type)
	if !IsValidServiceType(input.Type) {
		return uuid.Nil, errors.New("invalid service type")
	}

	// Marshal the input to JSON for storage
	res, err := json.Marshal(input)
	if err != nil {
		return uuid.Nil, err
	}
	rm := &json.RawMessage{}
	err = rm.UnmarshalJSON(res)
	if err != nil {
		return uuid.Nil, err
	}

	// Use ID from input if provided and valid, otherwise generate a new one
	var serviceId uuid.UUID
	if input.Id != "" {
		parsedId, parseErr := uuid.Parse(input.Id)
		if parseErr != nil {
			// If the provided ID is invalid, log and generate a new one
			p.l.WithError(parseErr).Warnf("Invalid UUID provided [%s], generating new one.", input.Id)
			serviceId = uuid.New()
		} else {
			// Use the provided UUID (including nil UUID if that's what was requested)
			serviceId = parsedId
		}
	} else {
		serviceId = uuid.New()
	}

	err = database.ExecuteTransaction(p.db, create(p.ctx, serviceId, serviceType, *rm))
	if err != nil {
		return uuid.Nil, err
	}
	return serviceId, nil
}

func (p *Processor) UpdateById(serviceId uuid.UUID, input service.InputRestModel) error {
	serviceType := ServiceType(input.Type)
	if !IsValidServiceType(input.Type) {
		return errors.New("invalid service type")
	}

	// Marshal the input to JSON for storage
	res, err := json.Marshal(input)
	if err != nil {
		return err
	}
	rm := &json.RawMessage{}
	err = rm.UnmarshalJSON(res)
	if err != nil {
		return err
	}

	return database.ExecuteTransaction(p.db, update(p.ctx, serviceId, serviceType, *rm))
}

func (p *Processor) DeleteById(serviceId uuid.UUID) error {
	return database.ExecuteTransaction(p.db, delete(p.ctx, serviceId))
}
