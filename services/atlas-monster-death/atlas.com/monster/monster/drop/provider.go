package drop

import (
	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func (p *ProcessorImpl) byMonsterIdProvider(monsterId uint32) model.Provider[[]Model] {
	return requests.SliceProvider[RestModel, Model](p.l, p.ctx)(requestForMonster(monsterId), Extract, model.Filters[Model]())
}

func (p *ProcessorImpl) GetByMonsterId(monsterId uint32) ([]Model, error) {
	return p.byMonsterIdProvider(monsterId)()
}
