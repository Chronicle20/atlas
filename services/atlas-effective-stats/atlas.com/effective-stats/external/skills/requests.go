package skills

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource           = "characters/%d/skills"
	CharacterSkillsAll = Resource
)

func getBaseRequest() string {
	return requests.RootUrl("SKILLS")
}

// characterSkillsUrl returns the list URL for a character's skills.
func characterSkillsUrl(characterId uint32) string {
	return fmt.Sprintf(getBaseRequest()+CharacterSkillsAll, characterId)
}

// identity is the no-op transformer for requests.DrainProvider, since
// RestModel is already the target type for this consumer.
func identity(m RestModel) (RestModel, error) {
	return m, nil
}

// RequestCharacterSkills fetches ALL skills for a character. The upstream
// atlas-skills list is now paginated (task-117); fetchPassiveBonuses (the
// sole caller) must see every skill to compute passive stat bonuses, so
// this drains every page rather than fetching just the first. The
// requests.Request[[]RestModel] return type is preserved so call sites are
// unchanged.
func RequestCharacterSkills(characterId uint32) requests.Request[[]RestModel] {
	return func(l logrus.FieldLogger, ctx context.Context) ([]RestModel, error) {
		return requests.DrainProvider[RestModel, RestModel](l, ctx)(characterSkillsUrl(characterId), 250, identity, model.Filters[RestModel]())()
	}
}
