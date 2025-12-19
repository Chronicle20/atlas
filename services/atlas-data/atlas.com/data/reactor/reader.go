package reactor

import (
	"atlas-data/point"
	"atlas-data/xml"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
	"path/filepath"
	"strconv"
	"strings"
)

func parseReactorId(filePath string) (uint32, error) {
	baseName := filepath.Base(filePath)
	if !strings.HasSuffix(baseName, ".img") {
		return 0, fmt.Errorf("file does not match expected format: %s", filePath)
	}
	idStr := strings.TrimSuffix(baseName, ".img")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, err
	}
	return uint32(id), nil

}

func Read(l logrus.FieldLogger) func(path string, id uint32, np xml.IdProvider) model.Provider[RestModel] {
	return func(path string, id uint32, np xml.IdProvider) model.Provider[RestModel] {
		exml, err := np(path, id)()
		if err != nil {
			return model.ErrorProvider[RestModel](err)
		}

		reactorId, err := parseReactorId(exml.Name)
		if err != nil {
			return model.ErrorProvider[RestModel](err)
		}
		l.Debugf("Processing reactor [%d].", reactorId)

		info, err := exml.ChildByName("info")
		if err != nil {
			return model.ErrorProvider[RestModel](err)
		}
		if info == nil {
			m := RestModel{
				Id: reactorId,
				StateInfo: map[int8][]ReactorStateRestModel{
					0: {{Type: 999, ReactorItem: nil, ActiveSkills: nil, NextState: 0}},
				},
				TimeoutInfo: map[int8]int32{
					0: -1,
				},
			}
			return model.FixedProvider(m)
		}

		link := info.GetString("link", "")
		if link != "" {
			var linkId int
			linkId, err = strconv.Atoi(link)
			if err != nil {
				return model.ErrorProvider[RestModel](err)
			}
			ln, err := Read(l)(path, uint32(linkId), np)()
			if err != nil {
				return model.ErrorProvider[RestModel](err)
			}
			ln.Id = reactorId
			return model.FixedProvider(ln)
		}

		loadArea := info.GetIntegerWithDefault("activateByTouch", 0) != 0

		m := RestModel{Id: reactorId, StateInfo: map[int8][]ReactorStateRestModel{}, TimeoutInfo: map[int8]int32{}}
		rid, err := exml.ChildByName("0")
		i := int8(0)
		for rid != nil {
			areaSet := false
			sdl := make([]ReactorStateRestModel, 0)
			ed, _ := rid.ChildByName("event")
			if ed != nil {
				timeout := ed.GetIntegerWithDefault("timeout", -1)

				for _, md := range ed.ChildNodes {
					t := md.GetIntegerWithDefault("type", 0)
					var ri *ReactorItemRestModel
					if t == 100 {
						itemId := uint32(md.GetIntegerWithDefault("0", 0))
						quantity := uint16(md.GetIntegerWithDefault("1", 0))
						ri = &ReactorItemRestModel{ItemId: itemId, Quantity: quantity}
						if !areaSet || loadArea {
							x, y := md.GetPoint("tl", 0, 0)
							m.TL = point.RestModel{
								X: int16(x),
								Y: int16(y),
							}
							x, y = md.GetPoint("rb", 0, 0)
							m.BR = point.RestModel{
								X: int16(x),
								Y: int16(y),
							}
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
					sdl = append(sdl, ReactorStateRestModel{Type: t, ReactorItem: ri, ActiveSkills: skillIds, NextState: ns})
				}
				m.StateInfo[i] = sdl
				m.TimeoutInfo[i] = timeout
			} else {
				m.StateInfo[i] = []ReactorStateRestModel{{
					Type:      999,
					NextState: i + 1,
				}}
				m.TimeoutInfo[i] = -1
			}
			i++
			rid, _ = exml.ChildByName(strconv.Itoa(int(i)))
		}
		return model.FixedProvider(m)
	}
}
