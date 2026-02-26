// exports.go provides cross-package access to listing providers and administrators.
// The shop package coordinates listing operations within transactions (e.g., adding,
// removing, and purchasing listings). These exported functions allow shop to call
// listing-internal functions while maintaining the package's encapsulation boundary.
package listing

import (
	database "github.com/Chronicle20/atlas-database"
	"github.com/google/uuid"
)

// GetByShopId is the exported provider for use by other packages.
func GetByShopId(shopId uuid.UUID) database.EntityProvider[[]Entity] {
	return getByShopId(shopId)
}

// CountByShopId is the exported provider for use by other packages.
func CountByShopId(shopId uuid.UUID) database.EntityProvider[int64] {
	return countByShopId(shopId)
}

// DeleteByShopId is the exported provider for use by other packages.
func DeleteByShopId(shopId uuid.UUID) database.EntityProvider[bool] {
	return deleteByShopId(shopId)
}

// CreateListing is the exported provider for use by other packages.
func CreateListing(entity *Entity) database.EntityProvider[Entity] {
	return createListing(entity)
}

// DeleteListing is the exported provider for use by other packages.
func DeleteListing(id uuid.UUID) database.EntityProvider[bool] {
	return deleteListing(id)
}

// UpdateBundles is the exported provider for use by other packages.
func UpdateBundles(id uuid.UUID, bundlesRemaining uint16, quantity uint16, expectedVersion uint32) database.EntityProvider[int64] {
	return updateBundles(id, bundlesRemaining, quantity, expectedVersion)
}

// GetByShopIdAndDisplayOrder is the exported provider for use by other packages.
func GetByShopIdAndDisplayOrder(shopId uuid.UUID, displayOrder uint16) database.EntityProvider[Entity] {
	return getByShopIdAndDisplayOrder(shopId, displayOrder)
}

// DecrementDisplayOrderAfter is the exported provider for use by other packages.
func DecrementDisplayOrderAfter(shopId uuid.UUID, afterOrder uint16) database.EntityProvider[int64] {
	return decrementDisplayOrderAfter(shopId, afterOrder)
}

// UpdateListingFields is the exported provider for use by other packages.
func UpdateListingFields(id uuid.UUID, pricePerBundle uint32, bundleSize uint16, bundleCount uint16) database.EntityProvider[int64] {
	return updateListingFields(id, pricePerBundle, bundleSize, bundleCount)
}
