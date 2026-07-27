package game

import (
	"strconv"

	"github.com/Chronicle20/atlas/libs/atlas-constants/channel"
	"github.com/Chronicle20/atlas/libs/atlas-constants/item"
	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
)

// PrizeRestModel is the JSON representation of a ladder Rung resolved for a
// session's current position.
type PrizeRestModel struct {
	ItemId   item.Id `json:"itemId"`
	Quantity uint32  `json:"quantity"`
	Meso     uint32  `json:"meso"`
}

// RestModel is the JSON:API representation of an RPS session, keyed by
// characterId. Prize is only populated when a prize is resolved at the
// session's current rung (GET responses); POST responses never carry one, a
// freshly Start-ed session is always rung 0 / no prize.
type RestModel struct {
	Id          uint32          `json:"-"`
	CharacterId uint32          `json:"characterId"`
	WorldId     world.Id        `json:"worldId"`
	ChannelId   channel.Id      `json:"channelId"`
	NpcId       uint32          `json:"npcId"`
	Rung        int             `json:"rung"`
	Status      string          `json:"status"`
	Prize       *PrizeRestModel `json:"prize,omitempty"`
}

func (r RestModel) GetName() string {
	return "rps-games"
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

// Transform converts a domain Model into its RestModel representation with
// no prize attached.
func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:          m.CharacterId(),
		CharacterId: m.CharacterId(),
		WorldId:     m.WorldId(),
		ChannelId:   m.ChannelId(),
		NpcId:       m.NpcId(),
		Rung:        m.Rung(),
		Status:      string(m.Status()),
	}, nil
}

// TransformWithPrize converts a domain Model into its RestModel
// representation, attaching the given resolved prize (if prizeOk) - used by
// GET /rps/games/{characterId}, where the prize varies with the session's
// current rung.
func TransformWithPrize(m Model, prize Rung, prizeOk bool) (RestModel, error) {
	rm, err := Transform(m)
	if err != nil {
		return RestModel{}, err
	}
	if prizeOk {
		rm.Prize = &PrizeRestModel{
			ItemId:   prize.ItemId,
			Quantity: prize.Quantity,
			Meso:     prize.Meso,
		}
	}
	return rm, nil
}
