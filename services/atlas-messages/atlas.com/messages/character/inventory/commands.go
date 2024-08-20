package inventory

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

func AwardItemCommandProducer(l logrus.FieldLogger, span opentracing.Span, t tenant.Model, c character.Model, m string) (command.Executor, bool) {
	if !c.Gm() {
		l.Debugf("Ignoring character [%d] command [%s], because they are not a gm.", c.Id(), m)
		return nil, false
	}

	if !strings.HasPrefix(m, "@award item") {
		return nil, false
	}

	var itemId uint32
	var quantity int16

	re := regexp.MustCompile("@award item (\\d*) (\\d*)")
	match := re.FindStringSubmatch(m)
	if len(match) == 3 {
		tItemId, err := strconv.ParseUint(match[1], 10, 32)
		if err != nil {
			return nil, false
		}
		itemId = uint32(tItemId)

		tQuantity, err := strconv.ParseInt(match[2], 10, 16)
		if err != nil {
			return nil, false
		}
		quantity = int16(tQuantity)
	}
	re = regexp.MustCompile("@award item (\\d*)")
	match = re.FindStringSubmatch(m)
	if len(match) == 2 {
		tItemId, err := strconv.ParseUint(match[1], 10, 32)
		if err != nil {
			return nil, false
		}
		itemId = uint32(tItemId)
		quantity = 1
	} else {
		return nil, false
	}

	exists := Exists(l, span, t)(itemId)
	if !exists {
		l.Debugf("Ignoring character [%d] command [%s], because they did not input a valid item.", c.Id(), m)
		return nil, false
	}

	return func(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) error {
		return GainItem(l, span, tenant)(c.Id(), itemId, quantity)
	}, true
}

func GainItem(l logrus.FieldLogger, span opentracing.Span, model tenant.Model) func(characterId uint32, itemId uint32, quantity int16) error {
	return func(characterId uint32, itemId uint32, quantity int16) error {
		return nil
	}
}
