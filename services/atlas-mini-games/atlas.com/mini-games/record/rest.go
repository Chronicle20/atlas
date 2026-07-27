package record

import "fmt"

// RestModel is the JSON:API representation of a character's win/tie/loss
// record for one game type. Id is a synthetic "characterId-gameType" key —
// there is no single natural resource id since a character has one record
// per game type.
type RestModel struct {
	Id          string `json:"-"`
	CharacterId uint32 `json:"characterId"`
	GameType    string `json:"gameType"`
	Wins        uint32 `json:"wins"`
	Ties        uint32 `json:"ties"`
	Losses      uint32 `json:"losses"`
}

func (r RestModel) GetName() string {
	return "game-records"
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(strId string) error {
	r.Id = strId
	return nil
}

func Transform(m Model) (RestModel, error) {
	return RestModel{
		Id:          fmt.Sprintf("%d-%s", m.CharacterId(), m.GameType()),
		CharacterId: m.CharacterId(),
		GameType:    string(m.GameType()),
		Wins:        m.Wins(),
		Ties:        m.Ties(),
		Losses:      m.Losses(),
	}, nil
}
