package party_quest

import "time"

type TimerModel struct {
	characterId uint32
	duration    time.Duration
}

func (m TimerModel) CharacterId() uint32     { return m.characterId }
func (m TimerModel) Duration() time.Duration { return m.duration }
