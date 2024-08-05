package information

import (
	"atlas-monsters/tenant"
	"github.com/Chronicle20/atlas-rest/requests"
	"github.com/opentracing/opentracing-go"
	"github.com/sirupsen/logrus"
)

func GetById(l logrus.FieldLogger, span opentracing.Span, tenant tenant.Model) func(monsterId uint32) (Model, error) {
	return func(monsterId uint32) (Model, error) {
		return requests.Provider[RestModel, Model](l)(requestById(l, span, tenant)(monsterId), Extract)()
	}
}
