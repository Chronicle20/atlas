package character

import (
	"atlas-character/tenant"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	EnvEventTopicCharacterCreated = "EVENT_TOPIC_CHARACTER_CREATED"
)

type createdEvent struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	WorldId     byte         `json:"worldId"`
	Name        string       `json:"name"`
}

func lookupTopic(l logrus.FieldLogger) func(token string) string {
	return func(token string) string {
		t, ok := os.LookupEnv(token)
		if !ok {
			l.Warnf("%s environment variable not set. Defaulting to env variable.", token)
			return token

		}
		return t
	}
}
