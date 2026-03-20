package frederick

import (
	"atlas-merchant/kafka/message/asset"
	"context"

	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Processor interface {
	StoreItems(characterId uint32, items []StoredItem) error
	StoreMesos(characterId uint32, amount uint32) error
	GetItems(characterId uint32) ([]ItemModel, error)
	GetMesos(characterId uint32) ([]MesoModel, error)
	ClearItems(characterId uint32) error
	ClearMesos(characterId uint32) error
	CreateNotification(characterId uint32) error
	ClearNotifications(characterId uint32) error
}

type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
}

type StoredItem struct {
	ItemId       uint32
	ItemType     byte
	Quantity     uint16
	ItemSnapshot asset.AssetData
}

// StoreItems moves unsold listing items into Frederick storage for a character.
func (p *ProcessorImpl) StoreItems(characterId uint32, items []StoredItem) error {
	_, err := storeItems(p.t.Id(), characterId, items)(p.db.WithContext(p.ctx))()
	return err
}

// StoreMesos stores meso balance in Frederick storage for a character.
func (p *ProcessorImpl) StoreMesos(characterId uint32, amount uint32) error {
	_, err := storeMesos(p.t.Id(), characterId, amount)(p.db.WithContext(p.ctx))()
	return err
}

// GetItems retrieves all items stored at Frederick for a character.
func (p *ProcessorImpl) GetItems(characterId uint32) ([]ItemModel, error) {
	return model.SliceMap(MakeItem)(getItemsByCharacterId(characterId)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

// GetMesos retrieves all meso records stored at Frederick for a character.
func (p *ProcessorImpl) GetMesos(characterId uint32) ([]MesoModel, error) {
	return model.SliceMap(MakeMeso)(getMesosByCharacterId(characterId)(p.db.WithContext(p.ctx)))(model.ParallelMap())()
}

// ClearItems removes all items from Frederick storage for a character.
func (p *ProcessorImpl) ClearItems(characterId uint32) error {
	_, err := clearItems(characterId)(p.db.WithContext(p.ctx))()
	return err
}

// ClearMesos removes all meso records from Frederick storage for a character.
func (p *ProcessorImpl) ClearMesos(characterId uint32) error {
	_, err := clearMesos(characterId)(p.db.WithContext(p.ctx))()
	return err
}

// CreateNotification creates a Frederick notification record for a character.
func (p *ProcessorImpl) CreateNotification(characterId uint32) error {
	_, err := createNotification(p.t, characterId)(p.db.WithContext(p.ctx))()
	return err
}

// ClearNotifications removes all notification records for a character.
func (p *ProcessorImpl) ClearNotifications(characterId uint32) error {
	_, err := clearNotifications(characterId)(p.db.WithContext(p.ctx))()
	return err
}

