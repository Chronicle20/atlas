package tenants

import (
	"atlas-configurations/outbox"
	"atlas-configurations/tenants/characters/preset"
	"context"
	"encoding/json"
	"os"
	"time"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outboxlib "github.com/Chronicle20/atlas/libs/atlas-outbox"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// EnvTenantStatusTopic names the env var carrying the Kafka topic that
// tenant config CRUD events are enqueued onto. Unset = enqueue skipped
// (matches the EnvServiceStatusTopic convention in services/processor).
const EnvTenantStatusTopic = "EVENT_TOPIC_CONFIGURATION_TENANT_STATUS"

func tenantOutboxKey(id uuid.UUID) []byte {
	return []byte("tenant:" + id.String())
}

func enqueueTenantStatus(tx *gorm.DB, id uuid.UUID, config any) error {
	topic := os.Getenv(EnvTenantStatusTopic)
	if topic == "" {
		return nil
	}
	var value []byte
	if config != nil {
		v, err := outbox.NewTenantEnvelope(id, config, time.Now())
		if err != nil {
			return err
		}
		value = v
	}
	return outboxlib.Enqueue(tx, outboxlib.Message{
		Topic: topic,
		Key:   tenantOutboxKey(id),
		Value: value,
	})
}

type Processor struct {
	l         logrus.FieldLogger
	ctx       context.Context
	db        *gorm.DB
	validator *preset.Validator
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) *Processor {
	p := &Processor{
		l:   l,
		ctx: ctx,
		db:  db,
	}
	return p
}

func (p *Processor) WithValidator(v *preset.Validator) *Processor {
	p.validator = v
	return p
}

func (p *Processor) ByIdProvider(id uuid.UUID) model.Provider[RestModel] {
	return model.Map(Make)(byIdEntityProvider(p.ctx)(id)(p.db))
}

func (p *Processor) ByRegionAndVersionProvider(region string, majorVersion uint16, minorVersion uint16) model.Provider[RestModel] {
	return model.Map(Make)(byRegionVersionEntityProvider(p.ctx)(region, majorVersion, minorVersion)(p.db))
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

func (p *Processor) GetById(id uuid.UUID) (RestModel, error) {
	return p.ByIdProvider(id)()
}

func (p *Processor) GetByRegionAndVersion(region string, majorVersion uint16, minorVersion uint16) (RestModel, error) {
	return p.ByRegionAndVersionProvider(region, majorVersion, minorVersion)()
}

func (p *Processor) UpdateById(tenantId uuid.UUID, input RestModel) error {
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

	return database.ExecuteTransaction(p.db, func(db *gorm.DB) error {
		if err := update(p.ctx, tenantId, input.Region, input.MajorVersion, input.MinorVersion, *rm)(db); err != nil {
			return err
		}
		return enqueueTenantStatus(db, tenantId, input)
	})
}

func (p *Processor) DeleteById(tenantId uuid.UUID) error {
	return database.ExecuteTransaction(p.db, func(db *gorm.DB) error {
		if err := delete(p.ctx, tenantId)(db); err != nil {
			return err
		}
		return enqueueTenantStatus(db, tenantId, nil)
	})
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

	// Use ID from input if provided, otherwise generate a new one
	var tenantId uuid.UUID
	if input.Id != "" {
		tenantId, err = uuid.Parse(input.Id)
		if err != nil {
			return uuid.Nil, err
		}
	} else {
		tenantId = uuid.New()
	}

	err = database.ExecuteTransaction(p.db, func(db *gorm.DB) error {
		e := &Entity{
			Id:           tenantId,
			Region:       input.Region,
			MajorVersion: input.MajorVersion,
			MinorVersion: input.MinorVersion,
			Data:         *rm,
		}
		if err := db.Create(e).Error; err != nil {
			return err
		}
		return enqueueTenantStatus(db, tenantId, input)
	})
	if err != nil {
		return uuid.Nil, err
	}
	return tenantId, nil
}
