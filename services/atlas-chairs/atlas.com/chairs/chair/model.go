package chair

import "encoding/json"

type Model struct {
	id               uint32
	chairType        string
	lastHpRecoveryAt int64 // unix-milli of last honored HP recovery tick; zero = never
	lastMpRecoveryAt int64 // unix-milli of last honored MP recovery tick; zero = never
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Type() string {
	return m.chairType
}

func (m Model) LastHpRecoveryAt() int64 {
	return m.lastHpRecoveryAt
}

func (m Model) LastMpRecoveryAt() int64 {
	return m.lastMpRecoveryAt
}

func (m Model) WithHpRecoveryAt(at int64) Model {
	m.lastHpRecoveryAt = at
	return m
}

func (m Model) WithMpRecoveryAt(at int64) Model {
	m.lastMpRecoveryAt = at
	return m
}

func (m Model) MarshalJSON() ([]byte, error) {
	return json.Marshal(&struct {
		Id               uint32 `json:"id"`
		ChairType        string `json:"chairType"`
		LastHpRecoveryAt int64  `json:"lastHpRecoveryAt,omitempty"`
		LastMpRecoveryAt int64  `json:"lastMpRecoveryAt,omitempty"`
	}{
		Id:               m.id,
		ChairType:        m.chairType,
		LastHpRecoveryAt: m.lastHpRecoveryAt,
		LastMpRecoveryAt: m.lastMpRecoveryAt,
	})
}

func (m *Model) UnmarshalJSON(data []byte) error {
	t := &struct {
		Id               uint32 `json:"id"`
		ChairType        string `json:"chairType"`
		LastHpRecoveryAt int64  `json:"lastHpRecoveryAt"`
		LastMpRecoveryAt int64  `json:"lastMpRecoveryAt"`
	}{}
	if err := json.Unmarshal(data, t); err != nil {
		return err
	}
	m.id = t.Id
	m.chairType = t.ChairType
	m.lastHpRecoveryAt = t.LastHpRecoveryAt
	m.lastMpRecoveryAt = t.LastMpRecoveryAt
	return nil
}
