package templates

import (
	"atlas-configurations/templates/characters/preset"
	"context"
	"encoding/json"
	database "github.com/Chronicle20/atlas/libs/atlas-database"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	WithValidator(v *preset.Validator) Processor
	ByRegionAndVersionProvider(region string, majorVersion uint16, minorVersion uint16) model.Provider[RestModel]
	ByIdProvider(templateId uuid.UUID) model.Provider[RestModel]
	AllProvider() model.Provider[[]RestModel]
	GetAll() ([]RestModel, error)
	GetByRegionAndVersion(region string, majorVersion uint16, minorVersion uint16) (RestModel, error)
	GetById(templateId uuid.UUID) (RestModel, error)
	Create(input RestModel) (uuid.UUID, error)
	UpdateById(templateId uuid.UUID, input RestModel) error
	DeleteById(templateId uuid.UUID) error
}

type ProcessorImpl struct {
	l         logrus.FieldLogger
	ctx       context.Context
	db        *gorm.DB
	validator *preset.Validator
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	p := &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
	}
	return p
}

var _ Processor = (*ProcessorImpl)(nil)

func (p *ProcessorImpl) WithValidator(v *preset.Validator) Processor {
	p.validator = v
	return p
}

func (p *ProcessorImpl) ByRegionAndVersionProvider(region string, majorVersion uint16, minorVersion uint16) model.Provider[RestModel] {
	return model.Map(Make)(byRegionVersionEntityProvider(p.ctx)(region, majorVersion, minorVersion)(p.db))
}

func (p *ProcessorImpl) ByIdProvider(templateId uuid.UUID) model.Provider[RestModel] {
	return model.Map(Make)(byIdEntityProvider(p.ctx)(templateId)(p.db))
}

func (p *ProcessorImpl) AllProvider() model.Provider[[]RestModel] {
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

func (p *ProcessorImpl) GetAll() ([]RestModel, error) {
	return p.AllProvider()()
}

func (p *ProcessorImpl) GetByRegionAndVersion(region string, majorVersion uint16, minorVersion uint16) (RestModel, error) {
	return p.ByRegionAndVersionProvider(region, majorVersion, minorVersion)()
}

func (p *ProcessorImpl) GetById(templateId uuid.UUID) (RestModel, error) {
	return p.ByIdProvider(templateId)()
}

func (p *ProcessorImpl) Create(input RestModel) (uuid.UUID, error) {
	res, err := json.Marshal(input)
	if err != nil {
		return uuid.Nil, err
	}
	rm := &json.RawMessage{}
	err = rm.UnmarshalJSON(res)
	if err != nil {
		return uuid.Nil, err
	}

	// Generate UUID in Go for database portability
	templateId := uuid.New()
	err = database.ExecuteTransaction(p.db, func(db *gorm.DB) error {
		e := &Entity{
			Id:           templateId,
			Region:       input.Region,
			MajorVersion: input.MajorVersion,
			MinorVersion: input.MinorVersion,
			Data:         *rm,
		}
		err := db.Create(e).Error
		if err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return uuid.Nil, err
	}
	return templateId, nil
}

func (p *ProcessorImpl) UpdateById(templateId uuid.UUID, input RestModel) error {
	if p.validator != nil {
		assigned, errs := p.validator.Validate(p.ctx, input.Characters.Presets)
		input.Characters.Presets = assigned
		if len(errs) > 0 {
			return &validationFailureError{errors: errs}
		}
	}

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

func (p *ProcessorImpl) DeleteById(templateId uuid.UUID) error {
	return database.ExecuteTransaction(p.db, delete(p.ctx, templateId))
}
