package account

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Builder struct {
	tenantId  uuid.UUID
	id        uint32
	name      string
	password  string
	pin       string
	pic       string
	state     State
	gender    byte
	banned    bool
	tos       bool
	updatedAt time.Time
}

func NewBuilder(tenantId uuid.UUID, name string) *Builder {
	return &Builder{
		tenantId: tenantId,
		name:     name,
		state:    StateNotLoggedIn,
		gender:   0,
		banned:   false,
		tos:      false,
	}
}

func (b *Builder) SetId(id uint32) *Builder {
	b.id = id
	return b
}

func (b *Builder) SetPassword(password string) *Builder {
	b.password = password
	return b
}

func (b *Builder) SetPin(pin string) *Builder {
	b.pin = pin
	return b
}

func (b *Builder) SetPic(pic string) *Builder {
	b.pic = pic
	return b
}

func (b *Builder) SetState(state State) *Builder {
	b.state = state
	return b
}

func (b *Builder) SetGender(gender byte) *Builder {
	b.gender = gender
	return b
}

func (b *Builder) SetBanned(banned bool) *Builder {
	b.banned = banned
	return b
}

func (b *Builder) SetTOS(tos bool) *Builder {
	b.tos = tos
	return b
}

func (b *Builder) SetUpdatedAt(updatedAt time.Time) *Builder {
	b.updatedAt = updatedAt
	return b
}

func (b *Builder) Build() (Model, error) {
	if b.name == "" {
		return Model{}, errors.New("name is required")
	}

	return Model{
		tenantId:  b.tenantId,
		id:        b.id,
		name:      b.name,
		password:  b.password,
		pin:       b.pin,
		pic:       b.pic,
		state:     b.state,
		gender:    b.gender,
		banned:    b.banned,
		tos:       b.tos,
		updatedAt: b.updatedAt,
	}, nil
}
