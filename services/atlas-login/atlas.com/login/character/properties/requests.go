package properties

import (
	"atlas-login/rest/requests"
	"fmt"
	"os"
)

const (
	CharactersResource          = "characters/"
	CharactersByName            = CharactersResource + "?name=%s"
	CharactersForAccountByWorld = CharactersResource + "?accountId=%d&worldId=%d"
	CharactersById              = CharactersResource + "%d"
)

func getBaseRequest() string {
	return os.Getenv("ACCOUNT_SERVICE_URL")
}

func requestPropertiesByName(name string) requests.Request[RestModel] {
	return requests.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+CharactersByName, name))
}

func requestPropertiesByAccountAndWorld(accountId uint32, worldId byte) requests.Request[RestModel] {
	return requests.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+CharactersForAccountByWorld, accountId, worldId))
}

func requestPropertiesById(characterId uint32) requests.Request[RestModel] {
	return requests.MakeGetRequest[RestModel](fmt.Sprintf(getBaseRequest()+CharactersById, characterId))
}
