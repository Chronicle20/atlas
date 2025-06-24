package tenant

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// Processor defines the interface for tenant operations
type Processor interface {
	// Create creates a new tenant
	Create(name string, region string, majorVersion uint16, minorVersion uint16) (Model, error)

	// Update updates an existing tenant
	Update(id uuid.UUID, name string, region string, majorVersion uint16, minorVersion uint16) (Model, error)

	// Delete deletes a tenant
	Delete(id uuid.UUID) error

	// GetById gets a tenant by ID
	GetById(id uuid.UUID) (Model, error)

	// GetAll gets all tenants
	GetAll() ([]Model, error)
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

// Create creates a new tenant
func (p *ProcessorImpl) Create(name string, region string, majorVersion uint16, minorVersion uint16) (Model, error) {
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

	// In a real implementation, we would emit a Kafka message here
	p.l.WithFields(logrus.Fields{
		"tenantId": m.Id().String(),
		"event":    "CREATED",
		"name":     m.Name(),
		"region":   m.Region(),
	}).Info("Tenant created")

	return m, nil
}

// Update updates an existing tenant
func (p *ProcessorImpl) Update(id uuid.UUID, name string, region string, majorVersion uint16, minorVersion uint16) (Model, error) {
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

	m := Make(e)

	// In a real implementation, we would emit a Kafka message here
	p.l.WithFields(logrus.Fields{
		"tenantId": m.Id().String(),
		"event":    "UPDATED",
		"name":     m.Name(),
		"region":   m.Region(),
	}).Info("Tenant updated")

	return m, nil
}

// Delete deletes a tenant
func (p *ProcessorImpl) Delete(id uuid.UUID) error {
	// First get the tenant to ensure it exists and to log its details
	provider := GetByIdProvider(id)(p.db)
	e, err := provider()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("tenant not found")
		}
		return err
	}

	m := Make(e)

	err = DeleteTenant(p.db, id)
	if err != nil {
		return err
	}

	// In a real implementation, we would emit a Kafka message here
	p.l.WithFields(logrus.Fields{
		"tenantId": m.Id().String(),
		"event":    "DELETED",
		"name":     m.Name(),
		"region":   m.Region(),
	}).Info("Tenant deleted")

	return nil
}

// GetById gets a tenant by ID
func (p *ProcessorImpl) GetById(id uuid.UUID) (Model, error) {
	provider := GetByIdProvider(id)(p.db)
	e, err := provider()
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Model{}, errors.New("tenant not found")
		}
		return Model{}, err
	}

	return Make(e), nil
}

// GetAll gets all tenants
func (p *ProcessorImpl) GetAll() ([]Model, error) {
	provider := GetAllProvider()(p.db)
	entities, err := provider()
	if err != nil {
		return nil, err
	}

	models := make([]Model, len(entities))
	for i, e := range entities {
		models[i] = Make(e)
	}

	return models, nil
}
