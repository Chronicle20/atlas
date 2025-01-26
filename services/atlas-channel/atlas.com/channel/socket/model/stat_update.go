package model

type StatUpdate struct {
	stat  string
	value int64
}

func NewStatUpdate(stat string, value int64) StatUpdate {
	return StatUpdate{
		stat:  stat,
		value: value,
	}
}

func (u StatUpdate) Stat() string {
	return u.stat
}

func (u StatUpdate) Value() int64 {
	return u.value
}
