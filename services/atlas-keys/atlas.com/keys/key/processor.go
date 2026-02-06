package key

import (
	"context"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

var defaultKey = []int32{18, 65, 2, 23, 3, 4, 5, 6, 16, 17, 19, 25, 26, 27, 31, 34, 35, 37, 38, 40, 43, 44, 45, 46, 50, 56, 59, 60, 61, 62, 63, 64, 57, 48, 29, 7, 24, 33, 41, 39}
var defaultType = []int8{4, 6, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 4, 5, 5, 4, 4, 5, 6, 6, 6, 6, 6, 6, 5, 4, 5, 4, 4, 4, 4, 4}
var defaultAction = []int32{0, 106, 10, 1, 12, 13, 18, 24, 8, 5, 4, 19, 14, 15, 2, 17, 11, 3, 20, 16, 9, 50, 51, 6, 7, 53, 100, 101, 102, 103, 104, 105, 54, 22, 52, 21, 25, 26, 23, 27}

var entityModelMapper = model.Map(Make)
var entitySliceMapper = model.SliceMap(Make)

// Processor defines the interface for key processing operations
type Processor interface {
	// ByCharacterIdProvider returns a provider for keys by character ID
	ByCharacterIdProvider(characterId uint32) model.Provider[[]Model]

	// GetByCharacterId gets keys by character ID
	GetByCharacterId(characterId uint32) ([]Model, error)

	// Reset resets keys for a character
	Reset(transactionId uuid.UUID, characterId uint32) error

	// CreateDefault creates default keys for a character
	CreateDefault(transactionId uuid.UUID, characterId uint32) error

	// Delete deletes keys for a character
	Delete(transactionId uuid.UUID, characterId uint32) error

	// ChangeKey changes a key binding
	ChangeKey(transactionId uuid.UUID, characterId uint32, key int32, theType int8, action int32) error
}

// ProcessorImpl implements the Processor interface
type ProcessorImpl struct {
	l   logrus.FieldLogger
	ctx context.Context
	db  *gorm.DB
	t   tenant.Model
}

// NewProcessor creates a new Processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context, db *gorm.DB) Processor {
	return &ProcessorImpl{
		l:   l,
		ctx: ctx,
		db:  db,
		t:   tenant.MustFromContext(ctx),
	}
}

// ByCharacterIdProvider returns a provider for keys by character ID
func (p *ProcessorImpl) ByCharacterIdProvider(characterId uint32) model.Provider[[]Model] {
	return entitySliceMapper(byCharacterIdEntityProvider(p.t.Id(), characterId)(p.db))()
}

// GetByCharacterId gets keys by character ID
func (p *ProcessorImpl) GetByCharacterId(characterId uint32) ([]Model, error) {
	p.l.Debugf("Retrieving key map for character [%d].", characterId)
	return p.ByCharacterIdProvider(characterId)()
}

// Reset resets keys for a character
func (p *ProcessorImpl) Reset(_ uuid.UUID, characterId uint32) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		err := deleteByCharacter(tx, p.t, characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to delete for character %d.", characterId)
			return err
		}
		for i := 0; i < len(defaultKey); i++ {
			_, err := create(tx, p.t, characterId, defaultKey[i], defaultType[i], defaultAction[i])
			if err != nil {
				p.l.WithError(err).Errorf("Unable to create key binding for character %d. key = %d type = %d action = %d.", characterId, defaultKey[i], defaultType[i], defaultAction[i])
				return err
			}
		}
		return nil
	})
}

// CreateDefault creates default keys for a character
func (p *ProcessorImpl) CreateDefault(_ uuid.UUID, characterId uint32) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		for i := 0; i < len(defaultKey); i++ {
			_, err := create(tx, p.t, characterId, defaultKey[i], defaultType[i], defaultAction[i])
			if err != nil {
				p.l.WithError(err).Errorf("Unable to create key binding for character %d. key = %d type = %d action = %d.", characterId, defaultKey[i], defaultType[i], defaultAction[i])
				return err
			}
		}
		return nil
	})
}

// Delete deletes keys for a character
func (p *ProcessorImpl) Delete(_ uuid.UUID, characterId uint32) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		err := deleteByCharacter(tx, p.t, characterId)
		if err != nil {
			p.l.WithError(err).Errorf("Unable to delete for character %d.", characterId)
			return err
		}
		return nil
	})
}

// ChangeKey changes a key binding
func (p *ProcessorImpl) ChangeKey(_ uuid.UUID, characterId uint32, key int32, theType int8, action int32) error {
	return p.db.Transaction(func(tx *gorm.DB) error {
		_, err := byCharacterKeyEntityProvider(p.t.Id(), characterId, key)(tx)()
		if err != nil {
			_, err = create(tx, p.t, characterId, key, theType, action)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to create key binding for character %d. key = %d type = %d action = %d.", characterId, key, theType, action)
				return err
			}
		} else {
			err = update(tx, p.t, characterId, key, theType, action)
			if err != nil {
				p.l.WithError(err).Errorf("Unable to update key binding for character %d. key = %d type = %d action = %d.", characterId, key, theType, action)
				return err
			}
		}
		return nil
	})
}
