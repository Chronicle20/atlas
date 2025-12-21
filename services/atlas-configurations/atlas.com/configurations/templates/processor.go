package templates

import (
	"atlas-configurations/database"
	"context"
	"encoding/json"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
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

func (p *Processor) ByRegionAndVersionProvider(region string, majorVersion uint16, minorVersion uint16) model.Provider[RestModel] {
	return model.Map(Make)(byRegionVersionEntityProvider(p.ctx)(region, majorVersion, minorVersion)(p.db))
}

func (p *Processor) ByIdProvider(templateId uuid.UUID) model.Provider[RestModel] {
	return model.Map(Make)(byIdEntityProvider(p.ctx)(templateId)(p.db))
}

func (p *Processor) AllProvider() model.Provider[[]RestModel] {
	return func() ([]RestModel, error) {
		return model.SliceMap(Make)(allEntityProvider(p.ctx)(p.db))()()
	}
}

func Make(e Entity) (RestModel, error) {
	var rm RestModel
	err := json.Unmarshal(e.Data, &rm)
	if err != nil {
		return RestModel{}, err
	}
	rm.Id = e.Id.String()
	return rm, nil
}

func (p *Processor) GetAll() ([]RestModel, error) {
	return p.AllProvider()()
}

func (p *Processor) GetByRegionAndVersion(region string, majorVersion uint16, minorVersion uint16) (RestModel, error) {
	return p.ByRegionAndVersionProvider(region, majorVersion, minorVersion)()
}

func (p *Processor) GetById(templateId uuid.UUID) (RestModel, error) {
	return p.ByIdProvider(templateId)()
}

func (p *Processor) Create(input RestModel) (uuid.UUID, error) {
	res, err := json.Marshal(input)
	if err != nil {
		return uuid.Nil, err
	}
	rm := &json.RawMessage{}
	err = rm.UnmarshalJSON(res)
	if err != nil {
		return uuid.Nil, err
	}

	var templateId uuid.UUID
	err = database.ExecuteTransaction(p.db, func(db *gorm.DB) error {
		e := &Entity{
			Region:       input.Region,
			MajorVersion: input.MajorVersion,
			MinorVersion: input.MinorVersion,
			Data:         *rm,
		}
		err := db.Save(e).Error
		if err != nil {
			return err
		}
		templateId = e.Id
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	return templateId, nil
}

func (p *Processor) UpdateById(templateId uuid.UUID, input RestModel) error {
	res, err := json.Marshal(input)
	if err != nil {
		return err
	}
	rm := &json.RawMessage{}
	err = rm.UnmarshalJSON(res)
	if err != nil {
		return err
	}

	return database.ExecuteTransaction(p.db, update(p.ctx, templateId, input.Region, input.MajorVersion, input.MinorVersion, *rm))
}

func (p *Processor) DeleteById(templateId uuid.UUID) error {
	return database.ExecuteTransaction(p.db, delete(p.ctx, templateId))
}
