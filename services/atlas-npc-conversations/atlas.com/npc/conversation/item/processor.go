package item

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	tenant "github.com/Chronicle20/atlas/libs/atlas-tenant"
)

type Processor interface {
	// Create creates a new item conversation
	Create(model Model) (Model, error)

	// Update updates an existing item conversation
	Update(id uuid.UUID, model Model) (Model, error)

	// Delete deletes an item conversation
	Delete(id uuid.UUID) error

	// ByIdProvider returns a provider for retrieving an item conversation by ID
	ByIdProvider(id uuid.UUID) model.Provider[Model]

	// ByItemIdProvider returns a provider for retrieving an item conversation by item ID
	ByItemIdProvider(itemId uint32) model.Provider[Model]

	// AllProvider returns a provider for retrieving one page of item conversations
	AllProvider(page model.Page) model.Provider[model.Paged[Model]]

	// DeleteAllForTenant deletes all item conversations for the current tenant
	DeleteAllForTenant() (int64, error)

	// Count returns the number of item conversations for the current tenant and the max updated_at timestamp.
	// Returns (0, nil, nil) when the tenant has no rows.
	Count() (int64, *time.Time, error)
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	t   tenant.Model
	db  *gorm.DB
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	t := tenant.MustFromContext(ctx)

	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		t:   t,
		db:  db,
	}
}

var _ Processor = (*ProcessorImpl)(nil)

// ByIdProvider returns a provider for retrieving an item conversation by ID
func (p *ProcessorImpl) ByIdProvider(id uuid.UUID) model.Provider[Model] {
	return model.Map[Entity, Model](Make)(getByIdProvider(id)(p.db.WithContext(p.ctx)))
}

// ByItemIdProvider returns a provider for retrieving an item conversation by item ID
func (p *ProcessorImpl) ByItemIdProvider(itemId uint32) model.Provider[Model] {
	return model.Map[Entity, Model](Make)(getByItemIdProvider(itemId)(p.db.WithContext(p.ctx)))
}

// AllProvider returns a provider for retrieving one page of item conversations
func (p *ProcessorImpl) AllProvider(page model.Page) model.Provider[model.Paged[Model]] {
	ep := getAllPagedProvider(page)(p.db.WithContext(p.ctx))
	return model.MapPaged(Make)(ep)(model.ParallelMap())
}

// Create creates a new item conversation
func (p *ProcessorImpl) Create(m Model) (Model, error) {
	p.l.Debugf("Creating item conversation for item [%d]", m.ItemId())

	result, err := createItemConversation(p.db.WithContext(p.ctx))(p.t.Id())(m)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to create item conversation for item [%d]", m.ItemId())
		return Model{}, err
	}
	return result, nil
}

// Update updates an existing item conversation
func (p *ProcessorImpl) Update(id uuid.UUID, m Model) (Model, error) {
	p.l.Debugf("Updating item conversation [%s]", id)

	result, err := updateItemConversation(p.db.WithContext(p.ctx))(id)(m)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to update item conversation [%s]", id)
		return Model{}, err
	}
	return result, nil
}

// Delete deletes an item conversation
func (p *ProcessorImpl) Delete(id uuid.UUID) error {
	p.l.Debugf("Deleting item conversation [%s]", id)

	err := deleteItemConversation(p.db.WithContext(p.ctx))(id)
	if err != nil {
		p.l.WithError(err).Errorf("Failed to delete item conversation [%s]", id)
		return err
	}
	return nil
}

// DeleteAllForTenant deletes all item conversations for the current tenant
func (p *ProcessorImpl) DeleteAllForTenant() (int64, error) {
	p.l.Debugf("Deleting all item conversations for tenant [%s]", p.t.Id())

	count, err := deleteAllItemConversations(p.db.WithContext(p.ctx))
	if err != nil {
		p.l.WithError(err).Errorf("Failed to delete all item conversations for tenant [%s]", p.t.Id())
		return 0, err
	}
	return count, nil
}

// Count returns the number of item conversations for the current tenant and the max updated_at timestamp.
// The tenant filter is applied automatically via the registered tenant callbacks on the GORM context.
func (p *ProcessorImpl) Count() (int64, *time.Time, error) {
	var count int64
	if err := p.db.WithContext(p.ctx).Model(&Entity{}).Count(&count).Error; err != nil {
		return 0, nil, err
	}
	if count == 0 {
		return 0, nil, nil
	}
	row := p.db.WithContext(p.ctx).Model(&Entity{}).Select("MAX(updated_at)").Row()
	var raw sql.NullString
	if err := row.Scan(&raw); err != nil {
		return 0, nil, err
	}
	if !raw.Valid || raw.String == "" {
		return count, nil, nil
	}
	t, err := parseDBTime(raw.String)
	if err != nil || t.IsZero() {
		return count, nil, nil
	}
	return count, &t, nil
}

func parseDBTime(s string) (time.Time, error) {
	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05.999999999-07:00",
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
	}
	for _, f := range formats {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, nil
}
