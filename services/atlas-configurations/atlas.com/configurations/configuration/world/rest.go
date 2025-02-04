package world

import "atlas-configurations/configuration/channel"

type RestModel struct {
	Id       byte                `json:"id"`
	Channels []channel.RestModel `json:"channels"`
}
