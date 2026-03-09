package model

import (
	packetmodel "github.com/Chronicle20/atlas-packet/model"
)

// Type aliases re-exporting from atlas-packet/model
type MonsterAppearType = packetmodel.MonsterAppearType
type MonsterTemporaryStatType = packetmodel.MonsterTemporaryStatType
type MonsterTemporaryStatValue = packetmodel.MonsterTemporaryStatValue
type MonsterBurnedInfo = packetmodel.MonsterBurnedInfo
type MonsterTemporaryStat = packetmodel.MonsterTemporaryStat
type Monster = packetmodel.MonsterModel

// Constants
const (
	MonsterAppearTypeNormal             = packetmodel.MonsterAppearTypeNormal
	MonsterAppearTypeRegen              = packetmodel.MonsterAppearTypeRegen
	MonsterAppearTypeRevived            = packetmodel.MonsterAppearTypeRevived
	MonsterAppearTypeSuspended          = packetmodel.MonsterAppearTypeSuspended
	MonsterAppearTypeDelay              = packetmodel.MonsterAppearTypeDelay
	MonsterAppearTypeBalrog             = packetmodel.MonsterAppearTypeBalrog
	MonsterAppearTypeSmoke              = packetmodel.MonsterAppearTypeSmoke
	MonsterAppearTypeWerewolf           = packetmodel.MonsterAppearTypeWerewolf
	MonsterAppearTypeKingSlimeMinion    = packetmodel.MonsterAppearTypeKingSlimeMinion
	MonsterAppearTypeSummoningRock      = packetmodel.MonsterAppearTypeSummoningRock
	MonsterAppearTypeEyeOfHorus         = packetmodel.MonsterAppearTypeEyeOfHorus
	MonsterAppearTypeBlueStars          = packetmodel.MonsterAppearTypeBlueStars
	MonsterAppearTypeSmoke2             = packetmodel.MonsterAppearTypeSmoke2
	MonsterAppearTypeTheBoss            = packetmodel.MonsterAppearTypeTheBoss
	MonsterAppearTypeGrimPhantomBlack   = packetmodel.MonsterAppearTypeGrimPhantomBlack
	MonsterAppearTypeGrimPhantomBlue    = packetmodel.MonsterAppearTypeGrimPhantomBlue
	MonsterAppearTypeThorn              = packetmodel.MonsterAppearTypeThorn
	MonsterAppearTypeUnknown            = packetmodel.MonsterAppearTypeUnknown
	MonsterAppearTypeFrankenstein       = packetmodel.MonsterAppearTypeFrankenstein
	MonsterAppearTypeFrankensteinEnraged = packetmodel.MonsterAppearTypeFrankensteinEnraged
	MonsterAppearTypeOrbit              = packetmodel.MonsterAppearTypeOrbit
	MonsterAppearTypeHiver              = packetmodel.MonsterAppearTypeHiver
	MonsterAppearTypeSmoke3             = packetmodel.MonsterAppearTypeSmoke3
	MonsterAppearTypeSmoke4             = packetmodel.MonsterAppearTypeSmoke4
	MonsterAppearTypePrimeMinister      = packetmodel.MonsterAppearTypePrimeMinister
	MonsterAppearTypePrimeMinister2     = packetmodel.MonsterAppearTypePrimeMinister2
	MonsterAppearTypeOlivia             = packetmodel.MonsterAppearTypeOlivia
	MonsterAppearTypeWingedEvilStump    = packetmodel.MonsterAppearTypeWingedEvilStump
	MonsterAppearTypeWingedEvilStump2   = packetmodel.MonsterAppearTypeWingedEvilStump2
	MonsterAppearTypeApsu               = packetmodel.MonsterAppearTypeApsu
	MonsterAppearTypeBlackFluid         = packetmodel.MonsterAppearTypeBlackFluid
	MonsterAppearTypeHiver2             = packetmodel.MonsterAppearTypeHiver2
	MonsterAppearTypeDragonRider        = packetmodel.MonsterAppearTypeDragonRider
)

// Function aliases
var NewMonsterTemporaryStatType = packetmodel.NewMonsterTemporaryStatType
var MonsterTemporaryStatTypeByName = packetmodel.MonsterTemporaryStatTypeByName
var NewMonsterTemporaryStat = packetmodel.NewMonsterTemporaryStat
var NewMonster = packetmodel.NewMonster
