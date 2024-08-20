package _map

import (
	"atlas-messages/character"
	"atlas-messages/command"
	"atlas-messages/tenant"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
	"regexp"
	"strconv"
	"strings"
)

func WarpMapCommandProducer(l logrus.FieldLogger, span opentracing.Span, t tenant.Model, worldId byte, channelId byte, c character.Model, m string) (command.Executor, bool) {
	if !c.Gm() {
		l.Debugf("Ignoring character [%d] command [%s], because they are not a gm.", c.Id(), m)
		return nil, false
	}

	if !strings.HasPrefix(m, "@warp map") {
		return nil, false
	}
	re := regexp.MustCompile("@warp map (\\d*)")
	match := re.FindStringSubmatch(m)
	if len(match) != 2 {
		return nil, false
	}

	mapId, err := strconv.ParseUint(match[1], 10, 32)
	if err != nil {
		return nil, false
	}

	exists := Exists(l, span, t)(uint32(mapId))
	if !exists {
		l.Debugf("Ignoring character [%d] command [%s], because they did not input a valid map.", c.Id(), m)
		return nil, false
	}

	return func(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) error {
		return WarpRandom(l, span, tenant)(worldId, channelId, c.Id(), uint32(mapId))
	}, true
}
