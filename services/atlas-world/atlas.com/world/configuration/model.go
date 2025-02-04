package configuration

import (
	"atlas-world/configuration/world"
	"errors"
	"github.com/google/uuid"
)

func (r *RestModel) FindWorld(tenantId uuid.UUID, index byte) (world.WorldRestModel, error) {
	var found = false
	var server world.RestModel
	for _, s := range r.Servers {
		if s.TenantId == tenantId {
			found = true
			server = s
			break
		}
	}
	if !found {
		return world.WorldRestModel{}, errors.New("tenant not found")
	}
	if len(server.Worlds) < 0 || int(index) >= len(server.Worlds) {
		return world.WorldRestModel{}, errors.New("index out of bounds")
	}
	return server.Worlds[index], nil
}
