package tenant

import (
	"atlas-tenants/kafka/message"
	"atlas-tenants/kafka/producer"
	"context"
	"errors"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Processor defines the interface for tenant operations
type Processor interface {
	// Create creates a new tenant
	Create(mb *message.Buffer) func(name string, region string, majorVersion uint16, minorVersion uint16) (Model, error)

	// CreateAndEmit creates a new tenant and emits a Kafka message
	CreateAndEmit(name string, region string, majorVersion uint16, minorVersion uint16) (Model, error)

	// Update updates an existing tenant
	Update(mb *message.Buffer) func(id uuid.UUID, name string, region string, majorVersion uint16, minorVersion uint16) (Model, error)

	// UpdateAndEmit updates an existing tenant and emits a Kafka message
	UpdateAndEmit(id uuid.UUID, name string, region string, majorVersion uint16, minorVersion uint16) (Model, error)

	// Delete deletes a tenant
	Delete(mb *message.Buffer) func(id uuid.UUID) error

	// DeleteAndEmit deletes a tenant and emits a Kafka message
	DeleteAndEmit(id uuid.UUID) error

	// GetById gets a tenant by ID
	GetById(id uuid.UUID) (Model, error)

	// GetAll gets all tenants
	GetAll() ([]Model, error)

	// ByIdProvider returns a provider for a tenant by ID
	ByIdProvider(id uuid.UUID) model.Provider[Model]

	// AllProvider returns a provider for all tenants
	AllProvider() model.Provider[[]Model]
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	p   producer.Provider
}

// NewProcessor creates a new processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		p:   producer.ProviderImpl(l)(ctx),
	}
}

// Create creates a new tenant
func (p *ProcessorImpl) Create(mb *message.Buffer) func(name string, region string, majorVersion uint16, minorVersion uint16) (Model, error) {
	return func(name string, region string, majorVersion uint16, minorVersion uint16) (Model, error) {
		m := NewBuilder().
			SetName(name).
			SetRegion(region).
			SetMajorVersion(majorVersion).
			SetMinorVersion(minorVersion).
			Build()

		e := Entity{
			ID:           m.Id(),
			Name:         m.Name(),
			Region:       m.Region(),
			MajorVersion: m.MajorVersion(),
			MinorVersion: m.MinorVersion(),
		}

		err := CreateTenant(p.db, e)
		if err != nil {
			return Model{}, err
		}

		// Create and add the Kafka message to the buffer
		err = mb.Put(EventTopicTenantStatus, CreateStatusEventProvider(
			m.Id(),
			EventTypeCreated,
			m.Name(),
			m.Region(),
			m.MajorVersion(),
			m.MinorVersion(),
		))
		if err != nil {
			return Model{}, err
		}

		p.l.WithFields(logrus.Fields{
			"tenantId": m.Id().String(),
			"event":    EventTypeCreated,
			"name":     m.Name(),
			"region":   m.Region(),
		}).Info("Tenant created")

		return m, nil
	}
}

// CreateAndEmit creates a new tenant and emits a Kafka message
func (p *ProcessorImpl) CreateAndEmit(name string, region string, majorVersion uint16, minorVersion uint16) (Model, error) {
	return message.EmitWithResult[Model, string](p.p)(func(mb *message.Buffer) func(string) (Model, error) {
		return func(name string) (Model, error) {
			return p.Create(mb)(name, region, majorVersion, minorVersion)
		}
	})(name)
}

// Update updates an existing tenant
func (p *ProcessorImpl) Update(mb *message.Buffer) func(id uuid.UUID, name string, region string, majorVersion uint16, minorVersion uint16) (Model, error) {
	return func(id uuid.UUID, name string, region string, majorVersion uint16, minorVersion uint16) (Model, error) {
		// First get the tenant to ensure it exists
		provider := GetByIdProvider(id)(p.db)
		e, err := provider()
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return Model{}, errors.New("tenant not found")
			}
			return Model{}, err
		}

		e.Name = name
		e.Region = region
		e.MajorVersion = majorVersion
		e.MinorVersion = minorVersion

		err = UpdateTenant(p.db, e)
		if err != nil {
			return Model{}, err
		}

		m, err := Make(e)
		if err != nil {
			return Model{}, err
		}

		// Create and add the Kafka message to the buffer
		err = mb.Put(EventTopicTenantStatus, CreateStatusEventProvider(
			m.Id(),
			EventTypeUpdated,
			m.Name(),
			m.Region(),
			m.MajorVersion(),
			m.MinorVersion(),
		))
		if err != nil {
			return Model{}, err
		}

		p.l.WithFields(logrus.Fields{
			"tenantId": m.Id().String(),
			"event":    EventTypeUpdated,
			"name":     m.Name(),
			"region":   m.Region(),
		}).Info("Tenant updated")

		return m, nil
	}
}

// UpdateAndEmit updates an existing tenant and emits a Kafka message
func (p *ProcessorImpl) UpdateAndEmit(id uuid.UUID, name string, region string, majorVersion uint16, minorVersion uint16) (Model, error) {
	return message.EmitWithResult[Model, uuid.UUID](p.p)(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
		return func(id uuid.UUID) (Model, error) {
			return p.Update(mb)(id, name, region, majorVersion, minorVersion)
		}
	})(id)
}

// Delete deletes a tenant
func (p *ProcessorImpl) Delete(mb *message.Buffer) func(id uuid.UUID) error {
	return func(id uuid.UUID) error {
		// First get the tenant to ensure it exists and to log its details
		provider := GetByIdProvider(id)(p.db)
		e, err := provider()
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("tenant not found")
			}
			return err
		}

		m, err := Make(e)
		if err != nil {
			return err
		}

		err = DeleteTenant(p.db, id)
		if err != nil {
			return err
		}

		// Create and add the Kafka message to the buffer
		err = mb.Put(EventTopicTenantStatus, CreateStatusEventProvider(
			m.Id(),
			EventTypeDeleted,
			m.Name(),
			m.Region(),
			m.MajorVersion(),
			m.MinorVersion(),
		))
		if err != nil {
			return err
		}

		p.l.WithFields(logrus.Fields{
			"tenantId": m.Id().String(),
			"event":    EventTypeDeleted,
			"name":     m.Name(),
			"region":   m.Region(),
		}).Info("Tenant deleted")

		return nil
	}
}

// DeleteAndEmit deletes a tenant and emits a Kafka message
func (p *ProcessorImpl) DeleteAndEmit(id uuid.UUID) error {
	return message.Emit(p.p)(func(mb *message.Buffer) error {
		return p.Delete(mb)(id)
	})
}

// GetById gets a tenant by ID
func (p *ProcessorImpl) GetById(id uuid.UUID) (Model, error) {
	return model.Map(Make)(GetByIdProvider(id)(p.db))()
}

// GetAll gets all tenants
func (p *ProcessorImpl) GetAll() ([]Model, error) {
	return model.SliceMap(Make)(GetAllProvider()(p.db))(model.ParallelMap())()
}

// ByIdProvider returns a provider for a tenant by ID
func (p *ProcessorImpl) ByIdProvider(id uuid.UUID) model.Provider[Model] {
	return model.Map(Make)(GetByIdProvider(id)(p.db))
}

// AllProvider returns a provider for all tenants
func (p *ProcessorImpl) AllProvider() model.Provider[[]Model] {
	return model.SliceMap(Make)(GetAllProvider()(p.db))(model.ParallelMap())
}
