package item

import (
	"net/url"

	"github.com/Chronicle20/atlas/libs/atlas-rest/requests"
)

const (
	Resource = "data/item-strings"
)

func getBaseRequest() string {
	return requests.RootUrl("DATA")
}

// requestByName queries atlas-data's item-string search index by name. The
// endpoint returns a JSON:API list of item-string resources whose id is the item
// template id and whose `name` attribute is the item name. The query is URL-escaped.
func requestByName(query string) requests.Request[[]RestModel] {
	return requests.GetRequest[[]RestModel](getBaseRequest() + Resource + "?search=" + url.QueryEscape(query))
}
