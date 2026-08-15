package character

import (
	"fmt"

	"github.com/Chronicle20/atlas/libs/atlas-constants/world"
	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource          = "characters"
	ById              = Resource + "/%d"
	ByIdWithInventory = Resource + "/%d?include=inventory"
	ByName            = Resource + "?name=%s&include=inventory"
	ByAccountInWorld  = Resource + "?accountId=%d&worldId=%d"
)

func getBaseRequest() string {
	return requests.RootUrl("CHARACTERS")
}

func requestById(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ById, id))
}

func requestByIdWithInventory(id uint32) requests.Request[RestModel] {
	return requests.GetRequest[RestModel](fmt.Sprintf(getBaseRequest()+ByIdWithInventory, id))
}

func requestByName(name string) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](fmt.Sprintf(getBaseRequest()+ByName, name))
}

// accountInWorldUrl returns the list URL for an account's characters in a
// world (atlas-character resource.go:33-34,99,
// "get_characters_for_account_in_world"). It is a bare URL (not a
// requests.Request), because that endpoint is served by
// character.Processor.GetForAccountInWorldProvider — a PAGED provider — so
// the response is a paged JSON:API document, not the bare array
// requestByName's []RestModel return type assumes. Callers drain it with
// requests.DrainProvider, mirroring note/requests.go's characterNotesUrl.
func accountInWorldUrl(accountId uint32, worldId world.Id) string {
	return fmt.Sprintf(getBaseRequest()+ByAccountInWorld, accountId, worldId)
}
