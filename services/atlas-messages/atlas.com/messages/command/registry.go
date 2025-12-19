package command

import (
	"atlas-messages/character"
	"context"
	"github.com/sirupsen/logrus"
	"sync"
)

type registry struct {
	commandRegistry []Producer
}

var once sync.Once
var r *registry

func Registry() *registry {
	once.Do(func() {
		r = &registry{}
		r.commandRegistry = make([]Producer, 0)
	})
	return r
}

func (r *registry) Add(svs ...Producer) {
	for _, sv := range svs {
		r.commandRegistry = append(r.commandRegistry, sv)
	}
}

func (r *registry) Get(l logrus.FieldLogger, ctx context.Context, worldId byte, channelId byte, character character.Model, m string) (Executor, bool) {
	for _, c := range r.commandRegistry {
		e, found := c(l)(ctx)(worldId, channelId, character, m)
		if found {
			return e, found
		}
	}
	return nil, false
}
