package cash

import "atlas-cashshop/cash/wishlist"

type Model struct {
	characterId uint32
	credit      uint32
	points      uint32
	prepaid     uint32
	wishlist    []wishlist.Model
}
