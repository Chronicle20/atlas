package skill

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters/%d/skills"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "SKILLS")
}

// characterSkillsUrl returns the list URL for a character's skills. The
// atlas-skills list endpoint is paginated (task-117); it is consumed via
// requests.DrainProvider, which appends its own page[number]/page[size]
// query params per request.
func characterSkillsUrl(ctx context.Context, characterId uint32) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+Resource, characterId), nil
}
