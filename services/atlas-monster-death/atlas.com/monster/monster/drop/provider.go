package drop

import (
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func (p *ProcessorImpl) byMonsterIdProvider(monsterId uint32) model.Provider[[]Model] {
	return requests.DrainProvider[RestModel, Model](p.l, p.ctx)(monsterDropsUrl(monsterId), 250, Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByMonsterId(monsterId uint32) ([]Model, error) {
	return p.byMonsterIdProvider(monsterId)()
}
