package itemstring

import (
	"context"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

type Processor struct {
	l   logrus.FieldLogger
	ctx context.Context
}

func NewProcessor(l logrus.FieldLogger, ctx context.Context) *Processor {
	return &Processor{l: l, ctx: ctx}
}

// identity is used in place of model.Identity because model.Identity has the
// signature func(M) M, while requests.Provider requires a model.Transformer
// (func(M) (N, error)). RestModel needs no transformation, so this simply
// passes it through.
func identity(r RestModel) (RestModel, error) {
	return r, nil
}

func (p *Processor) GetName(itemId uint32) (string, error) {
	rm, err := requests.Provider[RestModel, RestModel](p.l, p.ctx)(requestById(itemId), identity)()
	if err != nil {
		return "", err
	}
	return rm.Name, nil
}
