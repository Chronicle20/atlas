package item

// Death protection items - respawn and experience loss prevention
const (
	// WheelOfFortuneId is a cash item that allows respawning in the same map after death
	WheelOfFortuneId = Id(5510000)

	// SafetyCharmId is a cash item that prevents experience loss on death
	SafetyCharmId = Id(5130000)

	// EasterBasketId is a use item that prevents experience loss on death
	EasterBasketId = Id(4031283)

	// ProtectOnDeathId is a use item that prevents experience loss on death
	ProtectOnDeathId = Id(4140903)
)

// IsDeathProtectionItem returns true if the item prevents experience loss on death
func IsDeathProtectionItem(id Id) bool {
	return Is(id, SafetyCharmId, EasterBasketId, ProtectOnDeathId)
}

// IsWheelOfFortune returns true if the item allows respawn in same map
func IsWheelOfFortune(id Id) bool {
	return id == WheelOfFortuneId
}

// IsSafetyCharm returns true if the item is the cash shop safety charm
func IsSafetyCharm(id Id) bool {
	return id == SafetyCharmId
}
