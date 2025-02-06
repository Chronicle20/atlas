package reactor

import (
	"atlas-data/xml"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"strconv"
	"strings"
)

func byIdProvider(ctx context.Context) func(mapId uint32) model.Provider[Model] {
	t := tenant.MustFromContext(ctx)
	return func(mapId uint32) model.Provider[Model] {
		return func() (Model, error) {
			m, err := GetModelRegistry().Get(t, mapId)
			if err == nil {
				return m, nil
			}
			nt, err := tenant.Create(uuid.Nil, t.Region(), t.MajorVersion(), t.MinorVersion())
			if err != nil {
				return Model{}, err
			}
			return GetModelRegistry().Get(nt, mapId)
		}
	}
}

func GetById(ctx context.Context) func(mapId uint32) (Model, error) {
	return func(mapId uint32) (Model, error) {
		return byIdProvider(ctx)(mapId)()
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

func RegisterReactor(l logrus.FieldLogger) func(ctx context.Context) func(path string) {
	return func(ctx context.Context) func(path string) {
		t := tenant.MustFromContext(ctx)
		return func(path string) {
			m, err := ReadFromFile(l)(ctx)(path)()
			if err == nil {
				l.Debugf("Processed reactor [%d].", m.Id())
				_ = GetModelRegistry().Add(t, m)
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
				m := NewModel(reactorId).AddState(0, []ReactorState{{theType: 999, reactorItem: nil, activeSkills: nil, nextState: 0}}, -1)
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
							ri = &ReactorItem{itemId: itemId, quantity: quantity}
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
						sdl = append(sdl, ReactorState{theType: t, reactorItem: ri, activeSkills: skillIds, nextState: ns})
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
