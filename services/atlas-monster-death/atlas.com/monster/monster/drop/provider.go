package drop

import (
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func (p *ProcessorImpl) byMonsterIdProvider(monsterId uint32) model.Provider[[]Model] {
	url, err := monsterDropsUrl(p.ctx, monsterId)
	if err != nil {
		return model.ErrorProvider[[]Model](err)
	}
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(url, 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByMonsterId(monsterId uint32) ([]Model, error) {
	return p.byMonsterIdProvider(monsterId)()
}
