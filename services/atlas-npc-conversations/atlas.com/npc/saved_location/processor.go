package saved_location

import (
	"context"
	"errors"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// ErrNotFound is returned when no saved location exists
var ErrNotFound = errors.New("saved location not found")

// Processor provides operations for querying saved location data
type Processor interface {
	// GetSavedLocation retrieves a saved location for a character by type
	// Returns ErrNotFound if no saved location exists
	GetSavedLocation(characterId uint32, locationType string) model.Provider[Model]
}

type processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewProcessor creates a new saved location processor
func NewProcessor(l logrus.FieldLogger, ctx context.Context) Processor {
	return &processor{
		l:   l,
		ctx: ctx,
	}
}

// GetSavedLocation retrieves a saved location for a character by type
func (p *processor) GetSavedLocation(characterId uint32, locationType string) model.Provider[Model] {
	return func() (Model, error) {
		provider := requests.Provider[RestModel, Model](p.l, p.ctx)(requestByCharacterAndType(characterId, locationType), Extract)
		result, err := provider()
		if err != nil {
			// Check if it's a 404/not found error
			if errors.Is(err, requests.ErrNotFound) {
				return Model{}, ErrNotFound
			}
			p.l.WithError(err).Errorf("Failed to get saved location '%s' for character %d", locationType, characterId)
			return Model{}, err
		}
		return result, nil
	}
}
