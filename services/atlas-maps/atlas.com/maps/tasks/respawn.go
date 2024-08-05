package tasks

import (
	"atlas-maps/map/character"
	"atlas-maps/map/monster"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"time"
)

const RespawnTask = "respawn_task"

type Respawn struct {
	l        logrus.FieldLogger
	interval int
}

func NewRespawn(l logrus.FieldLogger, interval int) *Respawn {
	return &Respawn{l, interval}
}

func (r *Respawn) Run() {
	r.l.Debugf("Executing spawn task.")

	span := opentracing.StartSpan(RespawnTask)
	mks := character.GetMapsWithCharacters(r.l, span)
	for _, mk := range mks {
		go monster.Spawn(r.l, span, mk.Tenant)(mk.WorldId, mk.ChannelId, mk.MapId)
	}
	span.Finish()
}

func (r *Respawn) SleepTime() time.Duration {
	return time.Millisecond * time.Duration(r.interval)
}
