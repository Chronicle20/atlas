package character

import (
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

func (p *ProcessorImpl) GetById(characterId uint32) (Model, error) {
	return requests.Provider[RestModel, Model](p.l, p.ctx)(requestById(characterId), Extract)()
}
