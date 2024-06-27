package account

import (
	"atlas-account/tenant"
	"gorm.io/gorm"
	"time"
)

type EntityUpdateFunction func() ([]string, func(e *entity))

func create(db *gorm.DB) func(tenant tenant.Model, name string, password string) (Model, error) {
	return func(tenant tenant.Model, name string, password string) (Model, error) {
		a := &entity{
			TenantId: tenant.Id,
			Name:     name,
			Password: password,
		}

		err := db.Create(a).Error
		if err != nil {
			return Model{}, err
		}

		return modelFromEntity(*a)
	}
}

func update(db *gorm.DB) func(modifiers ...EntityUpdateFunction) IdOperator {
	return func(modifiers ...EntityUpdateFunction) IdOperator {
		return func(tenant tenant.Model, id uint32) error {
			e := &entity{}
			var columns []string
			for _, modifier := range modifiers {
				c, u := modifier()
				columns = append(columns, c...)
				u(e)
			}
			return db.Model(&entity{TenantId: tenant.Id, ID: id}).Select(columns).Updates(e).Error

		}
	}
}

func updateState(state State) EntityUpdateFunction {
	return func() ([]string, func(e *entity)) {
		var cs []string
		if state == LoggedIn {
			cs = []string{"State", "LastLogin"}
		} else {
			cs = []string{"State"}
		}

		uf := func(e *entity) {
			e.State = byte(state)
			if state == LoggedIn {
				e.LastLogin = time.Now().UnixNano()
			}
		}
		return cs, uf
	}
}

func modelFromEntity(a entity) (Model, error) {
	r := Builder(a.ID).
		SetName(a.Name).
		SetPassword(a.Password).
		SetState(State(a.State)).
		Build()
	return r, nil
}
