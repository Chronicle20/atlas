package session

import (
	"context"
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	SessionsResource = "characters/%d/sessions"
	PlaytimeResource = "characters/%d/sessions/playtime"
)

func getBaseRequest(ctx context.Context) (string, error) {
	return requests.RootUrlFor(ctx, "CHARACTER")
}

// SessionsSinceUrl builds the sessions endpoint URL for a character, filtered
// to sessions since the given Unix timestamp. The endpoint paginates, so
// callers that need the whole since-filtered collection must drain every
// page (see requests.DrainProvider) rather than issuing a single GET.
func SessionsSinceUrl(ctx context.Context, characterId uint32, sinceUnix int64) (string, error) {
	root, err := getBaseRequest(ctx)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(root+SessionsResource+"?since=%d", characterId, sinceUnix), nil
}

// RequestPlaytimeSince fetches computed playtime since the given Unix timestamp
