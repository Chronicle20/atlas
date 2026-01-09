package cosmetic

import (
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/sirupsen/logrus"
)

// RestAppearanceProvider implements AppearanceProvider by querying the atlas-query-aggregator service via REST
type RestAppearanceProvider struct {
	l   logrus.FieldLogger
	ctx context.Context
}

// NewRestAppearanceProvider creates a new REST-based appearance provider
func NewRestAppearanceProvider(l logrus.FieldLogger, ctx context.Context) AppearanceProvider {
	return &RestAppearanceProvider{
		l:   l,
		ctx: ctx,
	}
}

// GetCharacterAppearance retrieves character appearance data from the query aggregator service
func (p *RestAppearanceProvider) GetCharacterAppearance(ctx context.Context, characterId uint32) (CharacterAppearance, error) {
	p.l.Debugf("Querying appearance for character %d from query aggregator", characterId)

	// Make REST request to query aggregator
	provider := requests.Provider[RestCharacterModel, CharacterAppearance](p.l, ctx)(
		requestCharacterById(characterId),
		ExtractAppearance,
	)

	appearance, err := provider()
	if err != nil {
		p.l.WithError(err).Errorf("Failed to get character appearance for character %d", characterId)
		return CharacterAppearance{}, fmt.Errorf("failed to query character appearance: %w", err)
	}

	p.l.Debugf("Retrieved appearance for character %d: gender=%d, hair=%d (base=%d, color=%d), face=%d, skin=%d",
		characterId, appearance.Gender(), appearance.Hair(), appearance.HairBase(), appearance.HairColor(),
		appearance.Face(), appearance.SkinColor())

	return appearance, nil
}
