package configuration

import (
	"atlas-character-factory/configuration/template"
	"errors"
	"github.com/google/uuid"
)

func (r *RestModel) FindTemplate(tenantId uuid.UUID, jobIndex uint32, subJobIndex uint32, gender byte) (template.RestModel, error) {
	for _, s := range r.Servers {
		if s.TenantId == tenantId {
			for _, t := range s.Templates {
				if t.JobIndex == jobIndex && t.SubJobIndex == subJobIndex && t.Gender == gender {
					return t, nil
				}
			}
			return template.RestModel{}, errors.New("template configuration not found")
		}
	}
	return template.RestModel{}, errors.New("tenant not found")
}
