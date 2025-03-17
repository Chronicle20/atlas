package reactor

import (
	"atlas-data/document"
	"atlas-data/xml"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"path/filepath"
	"strconv"
	"strings"
)

var DocType = "REACTOR"

func byIdProvider(ctx context.Context) func(db *gorm.DB) func(id uint32) model.Provider[Model] {
	return func(db *gorm.DB) func(id uint32) model.Provider[Model] {
		t := tenant.MustFromContext(ctx)
		return func(id uint32) model.Provider[Model] {
			return func() (Model, error) {
				m, err := GetModelRegistry().Get(t, id)
				if err == nil {
					return m, nil
				}
				m, err = document.Get[Model](ctx)(db)(DocType, id)
				if err == nil {
					_ = GetModelRegistry().Add(t, m)
					return m, nil
				}
				nt, err := tenant.Create(uuid.Nil, t.Region(), t.MajorVersion(), t.MinorVersion())
				m, err = GetModelRegistry().Get(nt, id)
				if err == nil {
					return m, nil
				}
				nctx := tenant.WithContext(ctx, nt)
				m, err = document.Get[Model](nctx)(db)(DocType, id)
				if err == nil {
					_ = GetModelRegistry().Add(nt, m)
					return m, nil
				}
				return Model{}, err
			}
		}
	}
}

func GetById(ctx context.Context) func(db *gorm.DB) func(id uint32) (Model, error) {
	return func(db *gorm.DB) func(id uint32) (Model, error) {
		return func(id uint32) (Model, error) {
			return byIdProvider(ctx)(db)(id)()
		}
	}
}

func parseReactorId(filePath string) (uint32, error) {
	baseName := filepath.Base(filePath)
	if !strings.HasSuffix(baseName, ".img.xml") {
		return 0, fmt.Errorf("file does not match expected format: %s", filePath)
	}
	idStr := strings.TrimSuffix(baseName, ".img.xml")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, err
	}
	return uint32(id), nil

}

func RegisterReactor(db *gorm.DB) func(l logrus.FieldLogger) func(ctx context.Context) func(path string) {
	return func(l logrus.FieldLogger) func(ctx context.Context) func(path string) {
		return func(ctx context.Context) func(path string) {
			return func(path string) {
				m, err := ReadFromFile(l)(ctx)(path)()
				if err != nil {
					return
				}
				err = document.Create(ctx)(db)(DocType, m.GetId(), &m)
				if err != nil {
					return
				}

				l.Debugf("Processed reactor [%d].", m.GetId())
			}
		}
	}
}

func ReadFromFile(l logrus.FieldLogger) func(ctx context.Context) func(path string) model.Provider[Model] {
	return func(ctx context.Context) func(path string) model.Provider[Model] {
		return func(path string) model.Provider[Model] {
			reactorId, err := parseReactorId(path)
			if err != nil {
				return model.ErrorProvider[Model](err)
			}
			l.Debugf("Processing reactor [%d].", reactorId)

			exml, err := xml.Read(path)
			if err != nil {
				return model.ErrorProvider[Model](err)
			}

			info, err := exml.ChildByName("info")
			if err != nil {
				return model.ErrorProvider[Model](err)
			}
			if info == nil {
				m := NewModel(reactorId).AddState(0, []ReactorState{{Type: 999, ReactorItem: nil, ActiveSkills: nil, NextState: 0}}, -1)
				return model.FixedProvider(m)
			}

			link := info.GetString("link", "")
			if link != "" {
				_, err := strconv.Atoi(link)
				if err != nil {
					return model.ErrorProvider[Model](err)
				}
				return ReadFromFile(l)(ctx)(strings.ReplaceAll(path, strconv.Itoa(int(reactorId)), link))
			}

			loadArea := info.GetIntegerWithDefault("activateByTouch", 0) != 0

			m := NewModel(reactorId)
			rid, err := exml.ChildByName("0")
			i := int8(0)
			for rid != nil {
				areaSet := false
				sdl := make([]ReactorState, 0)
				ed, _ := rid.ChildByName("event")
				if ed != nil {
					timeout := ed.GetIntegerWithDefault("timeOut", -1)

					for _, md := range ed.ChildNodes {
						t := md.GetIntegerWithDefault("type", 0)
						var ri *ReactorItem
						if t == 100 {
							itemId := uint32(md.GetIntegerWithDefault("0", 0))
							quantity := uint16(md.GetIntegerWithDefault("1", 0))
							ri = &ReactorItem{ItemId: itemId, Quantity: quantity}
							if !areaSet || loadArea {
								m = m.SetTL(md.GetPoint("tl", 0, 0))
								m = m.SetRB(md.GetPoint("rb", 0, 0))
								areaSet = true
							}
						}
						skillIds := make([]uint32, 0)
						activeSkillId, _ := md.ChildByName("activeSkillID")
						if activeSkillId != nil {
							for _, s := range activeSkillId.ChildNodes {
								skillIds = append(skillIds, uint32(md.GetIntegerWithDefault(s.Name, 0)))
							}
						}
						ns := int8(md.GetIntegerWithDefault("state", 0))
						sdl = append(sdl, ReactorState{Type: t, ReactorItem: ri, ActiveSkills: skillIds, NextState: ns})
					}

					m = m.AddState(i, sdl, timeout)
				}
				i++
				rid, _ = exml.ChildByName(strconv.Itoa(int(i)))
			}
			return model.FixedProvider(m)
		}
	}
}
