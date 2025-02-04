package world

import (
	"github.com/google/uuid"
)

type RestModel struct {
	TenantId uuid.UUID        `json:"tenantId"`
	Worlds   []WorldRestModel `json:"worlds"`
}

type WorldRestModel struct {
	Name              string `json:"name"`
	Flag              string `json:"flag"`
	ServerMessage     string `json:"serverMessage"`
	EventMessage      string `json:"eventMessage"`
	WhyAmIRecommended string `json:"whyAmIRecommended"`
}
