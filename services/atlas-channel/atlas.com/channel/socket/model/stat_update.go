package model

import "github.com/Chronicle20/atlas-constants/stat"

type StatUpdate struct {
	statType stat.Type
	value    int64
}

func NewStatUpdate(statType stat.Type, value int64) StatUpdate {
	return StatUpdate{
		statType: statType,
		value:    value,
	}
}

func (u StatUpdate) Stat() stat.Type {
	return u.statType
}

func (u StatUpdate) Value() int64 {
	return u.value
}
