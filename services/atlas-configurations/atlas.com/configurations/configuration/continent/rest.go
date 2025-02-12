package continent

import "atlas-configurations/configuration/drop"

type RestModel struct {
	Id    int32            `json:"id"`
	Items []drop.RestModel `json:"items"`
}
