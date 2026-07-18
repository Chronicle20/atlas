package tenant

import (
	"atlas-tenants/kafka/message"
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	database "github.com/Chronicle20/atlas/libs/atlas-database"
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	outbox "github.com/Chronicle20/atlas/libs/atlas-outbox"
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

	// ByIdProvider returns a provider for a tenant by ID
	ByIdProvider(id uuid.UUID) model.Provider[Model]

	// AllProvider returns a paged provider for all tenants
	AllProvider(page model.Page) model.Provider[model.Paged[Model]]
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
}

// NewProcessor creates a new processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// Create creates a new tenant
func (p *ProcessorImpl) Create(mb *message.Buffer) func(name string, region string, majorVersion uint16, minorVersion uint16) (Model, error) {
	return func(name string, region string, majorVersion uint16, minorVersion uint16) (Model, error) {
		m, err := NewModelBuilder().
			SetName(name).
			SetRegion(region).
			SetMajorVersion(majorVersion).
			SetMinorVersion(minorVersion).
			Build()
		if err != nil {
			return Model{}, err
		}

		e := FromModel(m)

		err = CreateTenant(p.db, e)
		if err != nil {
			return Model{}, err
		}

		// CreateRoute and add the Kafka message to the buffer
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
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, string](outbox.EmitProvider(p.l, p.ctx, tx))(func(mb *message.Buffer) func(string) (Model, error) {
			return func(name string) (Model, error) {
				return NewProcessor(p.l, p.ctx, tx).Create(mb)(name, region, majorVersion, minorVersion)
			}
		})(name)
		return err
	})
	return result, txErr
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

		e = NewEntityBuilder().
			SetId(e.ID).
			SetName(name).
			SetRegion(region).
			SetMajorVersion(majorVersion).
			SetMinorVersion(minorVersion).
			Build()

		err = UpdateTenant(p.db, e)
		if err != nil {
			return Model{}, err
		}

		m, err := Make(e)
		if err != nil {
			return Model{}, err
		}

		// CreateRoute and add the Kafka message to the buffer
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
	var result Model
	txErr := database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		var err error
		result, err = message.EmitWithResult[Model, uuid.UUID](outbox.EmitProvider(p.l, p.ctx, tx))(func(mb *message.Buffer) func(uuid.UUID) (Model, error) {
			return func(id uuid.UUID) (Model, error) {
				return NewProcessor(p.l, p.ctx, tx).Update(mb)(id, name, region, majorVersion, minorVersion)
			}
		})(id)
		return err
	})
	return result, txErr
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

		// CreateRoute and add the Kafka message to the buffer
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
	return database.ExecuteTransaction(p.db.WithContext(p.ctx), func(tx *gorm.DB) error {
		return message.Emit(outbox.EmitProvider(p.l, p.ctx, tx))(func(mb *message.Buffer) error {
			return NewProcessor(p.l, p.ctx, tx).Delete(mb)(id)
		})
	})
}

// GetById gets a tenant by ID
func (p *ProcessorImpl) GetById(id uuid.UUID) (Model, error) {
	return model.Map(Make)(GetByIdProvider(id)(p.db))()
}

// ByIdProvider returns a provider for a tenant by ID
func (p *ProcessorImpl) ByIdProvider(id uuid.UUID) model.Provider[Model] {
	return model.Map(Make)(GetByIdProvider(id)(p.db))
}

// AllProvider returns a paged provider for all tenants
func (p *ProcessorImpl) AllProvider(page model.Page) model.Provider[model.Paged[Model]] {
	return model.MapPaged(Make)(getAll(page)(p.db))(model.ParallelMap())
}
