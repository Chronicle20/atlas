package skill

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "characters/%d/skills"
	ById     = Resource + "/%d"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "SKILLS")
}

// characterSkillsUrl returns the list URL for a character's skills. It is a
// bare URL (not a requests.Request) because the list is now paginated
// server-side (task-117) and consumed via requests.DrainProvider, which
// appends its own page[number]/page[size] query params per request.
func characterSkillsUrl(ctx context.Context, characterId uint32) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+Resource, characterId), nil
}
