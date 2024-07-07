package inventory

import (
	"atlas-character/tenant"
	"github.com/sirupsen/logrus"
	"os"
)

const (
	EnvCommandTopicEquipItem   = "COMMAND_TOPIC_EQUIP_ITEM"
	EnvCommandTopicUnequipItem = "COMMAND_TOPIC_UNEQUIP_ITEM"
)

type equipItemCommand struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	Source      int16        `json:"source"`
	Destination int16        `json:"destination"`
}

type unequipItemCommand struct {
	Tenant      tenant.Model `json:"tenant"`
	CharacterId uint32       `json:"characterId"`
	Source      int16        `json:"source"`
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
