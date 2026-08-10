// Package location is the atlas-maps character-location REST client. It answers
// the one question the invite path asks of it: which field is the invite target
// standing in, so an invite can be refused when the two characters are not in
// the same field (FR-2.2).
//
// atlas-maps owns character location — atlas-character's GET response does not
// carry a mapId (services/atlas-character/atlas.com/character/character/rest.go:43),
// which is why this is a separate reader rather than a field on data/character.
package location

import (
	"strconv"

	"github.com/google/uuid"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/character"
	_map "github.com/Chronicle20/atlas/libs/atlas-constants/map"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RestModel mirrors the JSON:API shape returned by atlas-maps's
// GET /characters/{id}/location endpoint. The no-op relationship stubs are
// required by the api2go contract (see libs/atlas-rest/CLAUDE.md).
type RestModel struct {
	Id        character.Id `json:"-"`
	WorldId   world.Id     `json:"worldId"`
	ChannelId channel.Id   `json:"channelId"`
	MapId     _map.Id      `json:"mapId"`
	Instance  uuid.UUID    `json:"instance"`
}

func (r RestModel) GetName() string { return "character-locations" }

func (r RestModel) GetID() string { return strconv.FormatUint(uint64(r.Id), 10) }

func (r *RestModel) SetID(s string) error {
	v, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return err
	}
	r.Id = character.Id(v)
	return nil
}

func (r *RestModel) SetToOneReferenceID(_ string, _ string) error { return nil }

func (r *RestModel) SetToManyReferenceIDs(_ string, _ []string) error { return nil }
