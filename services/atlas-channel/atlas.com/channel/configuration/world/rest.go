package world

import "atlas-channel/configuration/channel"

type RestModel struct {
	Id       byte                `json:"id"`
	Channels []channel.RestModel `json:"channels"`
}
