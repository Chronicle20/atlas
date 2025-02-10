package skill

import (
	"math"
	"time"
)

type Model struct {
	id          uint32
	level       byte
	masterLevel byte
	expiration  time.Time
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

func (m Model) IsFourthJob() bool {
	// TODO Cygnus, Aran, or Evan
	jobId := uint32(math.Floor(float64(m.id) / float64(10000)))
	return jobId == 112 || jobId == 122 || jobId == 132 || jobId == 212 || jobId == 222 || jobId == 232 || jobId == 412 || jobId == 422 || jobId == 512 || jobId == 522
}
