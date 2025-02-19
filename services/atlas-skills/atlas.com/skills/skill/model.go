package skill

import "time"

type Model struct {
	id                uint32
	level             byte
	masterLevel       byte
	expiration        time.Time
	cooldownExpiresAt time.Time
}

func (m Model) Id() uint32 {
	return m.id
}

func (m Model) Level() byte {
	return m.level
}

func (m Model) MasterLevel() byte {
	return m.masterLevel
}

func (m Model) Expiration() time.Time {
	return m.expiration
}

func (m Model) CooldownExpiresAt() time.Time {
	return m.cooldownExpiresAt
}

func (m Model) SetCooldown(time time.Time) Model {
	return Model{
		id:                m.Id(),
		level:             m.Level(),
		masterLevel:       m.MasterLevel(),
		expiration:        m.Expiration(),
		cooldownExpiresAt: time,
	}
}
