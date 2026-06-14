package location

import (
	"context"
	"errors"

	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/sirupsen/logrus"
)

// ResolveMapId returns the character's current map from atlas-maps, or 0 when
// the location is missing (ErrNotFound, logged at Warn) or unreachable
// (infrastructure error, logged at Error). This mirrors the fallback policy the
// login service uses in its character_list writer.
func ResolveMapId(l logrus.FieldLogger, ctx context.Context, characterId uint32) _map.Id {
	f, err := GetField(l, ctx, characterId)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			l.Warnf("No atlas-maps location for character [%d]; using map 0.", characterId)
		} else {
			l.WithError(err).Errorf("Unable to resolve atlas-maps location for character [%d]; using map 0.", characterId)
		}
		return 0
	}
	return f.MapId()
}
