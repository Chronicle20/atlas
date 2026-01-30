package history

import (
	"strconv"
	"time"

	"github.com/Chronicle20/atlas-constants/channel"
	"github.com/Chronicle20/atlas-constants/world"
)

type RestModel struct {
	Id          string     `json:"-"`
	CharacterId uint32     `json:"characterId"`
	WorldId     world.Id   `json:"worldId"`
	ChannelId   channel.Id `json:"channelId"`
	LoginTime   time.Time  `json:"loginTime"`
	LogoutTime  *time.Time `json:"logoutTime,omitempty"`
}

func (r RestModel) GetID() string {
	return r.Id
}

func (r *RestModel) SetID(id string) error {
	r.Id = id
	return nil
}

func (r RestModel) GetName() string {
	return "sessions"
}

func TransformToRest(m Model) RestModel {
	return RestModel{
		Id:          strconv.FormatUint(m.Id(), 10),
		CharacterId: m.CharacterId(),
		WorldId:     m.WorldId(),
		ChannelId:   m.ChannelId(),
		LoginTime:   m.LoginTime(),
		LogoutTime:  m.LogoutTime(),
	}
}

func TransformSliceToRest(models []Model) []RestModel {
	result := make([]RestModel, len(models))
	for i, m := range models {
		result[i] = TransformToRest(m)
	}
	return result
}

// PlaytimeResponse is the response for playtime computation endpoint
type PlaytimeResponse struct {
	Id            string `json:"-"`
	CharacterId   uint32 `json:"characterId"`
	TotalSeconds  int64  `json:"totalSeconds"`
	FormattedTime string `json:"formattedTime"`
}

func (r PlaytimeResponse) GetID() string {
	return r.Id
}

func (r *PlaytimeResponse) SetID(id string) error {
	r.Id = id
	return nil
}

func (r PlaytimeResponse) GetName() string {
	return "playtime"
}

func FormatDuration(d time.Duration) string {
	h := int(d.Hours())
	m := int(d.Minutes()) % 60
	s := int(d.Seconds()) % 60
	return time.Date(0, 0, 0, h, m, s, 0, time.UTC).Format("15:04:05")
}
