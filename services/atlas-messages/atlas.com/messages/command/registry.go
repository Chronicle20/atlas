package command

import (
	"atlas-messages/character"
	"atlas-messages/tenant"
	"github.com/opentracing/opentracing-go"
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

func (r *registry) Get(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model, worldId byte, channelId byte, character character.Model, m string) (Executor, bool) {
	for _, c := range r.commandRegistry {
		e, found := c(l, span, tenant, worldId, channelId, character, m)
		if found {
			return e, found
		}
	}
	return nil, false
}
