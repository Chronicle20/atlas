package rps

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// RestModel is the JSON:API representation of an RPS session, mirroring
// atlas-rps's own game.RestModel (services/atlas-rps/atlas.com/rps/game/rest.go).
// Only CharacterId/WorldId/ChannelId/NpcId are relevant on the POST
// /rps/games request body atlas-rps decodes; Rung/Status/Prize are only ever
// populated on atlas-rps's responses.
type RestModel struct {
	Id          uint32     `json:"-"`
	CharacterId uint32     `json:"characterId"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	NpcId       uint32     `json:"npcId"`
}

func (r RestModel) GetID() string {
	return strconv.Itoa(int(r.Id))
}

func (r *RestModel) SetID(strId string) error {
	// POST /rps/games bodies carry no "id" (the session is keyed by the
	// characterId attribute, assigned server-side) - treat a missing id as
	// unset rather than a parse error.
	if strId == "" {
		r.Id = 0
		return nil
	}
	id, err := strconv.Atoi(strId)
	if err != nil {
		return err
	}
	r.Id = uint32(id)
	return nil
}

func (r RestModel) GetName() string {
	return "rps-games"
}
