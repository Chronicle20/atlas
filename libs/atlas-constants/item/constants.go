package item

import "math"

type Id uint32

type Classification uint32

const (
	ClassificationThrowingStar = Classification(207)
	ClassificationBullet       = Classification(233)
)

func GetClassification(itemId Id) Classification {
	return Classification(math.Floor(float64(itemId) / float64(10000)))
}

func IsThrowingStar(itemId Id) bool {
	return GetClassification(itemId) == ClassificationThrowingStar
}

func IsBullet(itemId Id) bool {
	return GetClassification(itemId) == ClassificationBullet
}

func Is(itemId Id, references ...Id) bool {
	for _, r := range references {
		if itemId == r {
			return true
		}
	}
	return false
}

const (
	UseRedBeanPorridge = Id(2022001)
	UseAirBubble       = Id(2022040)
	UseSoftWhiteBun    = Id(2022186)
)
