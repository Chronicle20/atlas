package _map

import (
	"atlas-data/map/monster"
	"atlas-data/map/npc"
	"atlas-data/map/portal"
	"atlas-data/map/reactor"
	npc2 "atlas-data/npc"
	"atlas-data/point"
	"atlas-data/xml"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
	"github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"math"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
)

func parseMapId(filePath string) (uint32, error) {
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

func ReadFromFile(l logrus.FieldLogger) func(ctx context.Context) func(path string) model.Provider[Model] {
	return func(ctx context.Context) func(path string) model.Provider[Model] {
		t := tenant.MustFromContext(ctx)
		return func(path string) model.Provider[Model] {
			mapId, err := parseMapId(path)
			if err != nil {
				return model.ErrorProvider[Model](err)
			}
			l.Debugf("Processing map [%d].", mapId)

			exml, err := xml.Read(path)
			if err != nil {
				return model.ErrorProvider[Model](err)
			}

			i, err := exml.ChildByName("info")
			if err != nil {
				return model.ErrorProvider[Model](err)
			}

			link := i.GetString("link", "")
			if link != "" {
				// TODO
				//linkedMapId, err := strconv.Atoi(link)
				//if err != nil {
				//	return nil, err
				//}
				//return Read(tenant, uint32(linkedMapId))
			}

			m := &Model{Id: mapId}
			m.ReturnMapId = uint32(i.GetIntegerWithDefault("returnMap", 0))
			m.MonsterRate = i.GetFloatWithDefault("mobRate", 0)

			firstUserEnter := i.GetString("onFirstUserEnter", strconv.Itoa(int(mapId)))
			if firstUserEnter == "" {
				firstUserEnter = strconv.Itoa(int(mapId))
			}
			m.OnFirstUserEnter = firstUserEnter

			onUserEnter := i.GetString("onUserEnter", strconv.Itoa(int(mapId)))
			if onUserEnter == "" {
				onUserEnter = strconv.Itoa(int(mapId))
			}
			m.OnUserEnter = onUserEnter

			m.FieldLimit = uint32(i.GetIntegerWithDefault("fieldLimit", 0))
			m.MobInterval = uint32(i.GetIntegerWithDefault("createMobInterval", 5000))
			m.Portals = getPortals(exml)
			m.TimeMob = getTimeMob(i)
			m.MapArea = getMapArea(exml, i)
			m.FootholdTree = getFootholdTree(exml)
			m.Areas = getAreas(exml)
			m.Seats = getSeats(exml)
			m.Name = getPlaceName(t, mapId)
			m.StreetName = getStreetName(t, mapId)
			m.Clock = getClock(exml)
			m.EverLast = i.GetIntegerWithDefault("everlast", 0) > 0
			m.Town = i.GetIntegerWithDefault("town", 0) > 0
			m.DecHp = uint32(i.GetIntegerWithDefault("decHP", 0))
			m.ProtectItem = uint32(i.GetIntegerWithDefault("protectItem", 0))
			m.ForcedReturnMapId = uint32(i.GetIntegerWithDefault("forcedReturn", 999999999))
			m.Boat = isBoat(exml)
			m.TimeLimit = i.GetIntegerWithDefault("timeLimit", -1)
			m.FieldType = uint32(i.GetIntegerWithDefault("fieldType", 0))
			m.MobCapacity = uint32(i.GetIntegerWithDefault("fixedMobCapacity", 500))
			m.Recovery = i.GetFloatWithDefault("recovery", 1)
			m.BackgroundTypes = getBackgroundTypes(exml)
			m.Reactors = getReactors(exml)
			monsters, npcs := getLife(t, exml)
			m.Monsters = monsters
			m.Npcs = npcs
			//TODO player NPCS and CPQ support

			lp := &point.Model{X: m.MapArea.X, Y: m.MapArea.Y}
			rp := &point.Model{X: m.MapArea.X + m.MapArea.Width, Y: m.MapArea.Y}
			fallback := &point.Model{X: m.MapArea.X + int16(math.Floor(float64(m.MapArea.Width/2))), Y: m.MapArea.Y}

			lp = bSearchDropPos(m.FootholdTree, lp, fallback)
			rp = bSearchDropPos(m.FootholdTree, rp, fallback)
			m.XLimit = XLimit{
				Min: uint32(lp.X + 14),
				Max: uint32(rp.X - 14),
			}
			return model.FixedProvider(*m)
		}
	}
}

const (
	PortalTypeTeleport uint8 = 1
	PortalTypeMap      uint8 = 2
	PortalTypeDoor     uint8 = 6
)

var portalId uint32

func getPortals(exml *xml.Node) []portal.Model {
	portals := make([]portal.Model, 0)
	p, err := exml.ChildByName("portal")
	if err != nil {
		return portals
	}
	for _, c := range p.ChildNodes {
		var pid uint32
		pt := uint8(c.GetIntegerWithDefault("pt", 0))
		if pt == PortalTypeDoor {
			pid = atomic.AddUint32(&portalId, 1)
		} else {
			pstr, err := strconv.Atoi(c.Name)
			if err != nil {
				continue
			}
			pid = uint32(pstr)
		}

		portal := portal.Model{
			Id:          strconv.Itoa(int(pid)),
			Name:        c.GetString("pn", ""),
			Target:      c.GetString("tn", ""),
			Type:        pt,
			X:           int16(c.GetIntegerWithDefault("x", 0)),
			Y:           int16(c.GetFloatWithDefault("y", 0)),
			TargetMapId: uint32(c.GetIntegerWithDefault("tm", 0)),
			ScriptName:  c.GetString("script", ""),
		}
		portals = append(portals, portal)
	}
	return portals
}

func getTimeMob(i *xml.Node) *TimeMob {
	tm, err := i.ChildByName("timeMob")
	if err != nil {
		return nil
	}
	id := uint32(tm.GetIntegerWithDefault("id", 0))
	message := tm.GetString("message", "")
	return &TimeMob{
		Id:      id,
		Message: message,
	}
}

func getMapArea(exml *xml.Node, i *xml.Node) Rectangle {
	bounds := make([]int16, 4)
	bounds[0] = int16(i.GetIntegerWithDefault("VRTop", 0))
	bounds[1] = int16(i.GetIntegerWithDefault("VRBottom", 0))

	if bounds[0] == bounds[1] {
		mm, err := exml.ChildByName("miniMap")
		if err == nil {
			bounds[0] = int16(mm.GetIntegerWithDefault("centerX", 0) * -1)
			bounds[1] = int16(mm.GetIntegerWithDefault("centerY", 0) * -1)
			bounds[2] = int16(mm.GetIntegerWithDefault("height", 0))
			bounds[3] = int16(mm.GetIntegerWithDefault("width", 0))
			return Rectangle{
				X:      bounds[0],
				Y:      bounds[1],
				Width:  bounds[3],
				Height: bounds[2],
			}
		} else {
			dist := 1 << 18
			return Rectangle{
				X:      int16(-dist / 2),
				Y:      int16(-dist / 2),
				Width:  int16(dist),
				Height: int16(dist),
			}
		}
	} else {
		bounds[2] = int16(i.GetIntegerWithDefault("VRLeft", 0))
		bounds[3] = int16(i.GetIntegerWithDefault("VRRight", 0))
		return Rectangle{
			X:      bounds[2],
			Y:      bounds[0],
			Width:  bounds[3] - bounds[2],
			Height: bounds[1] - bounds[0],
		}
	}
}

func getFootholdTree(exml *xml.Node) *FootholdTree {
	footholds := make([]Foothold, 0)
	var lx int16
	var ly int16
	var ux int16
	var uy int16

	fr, err := exml.ChildByName("foothold")
	if err == nil {
		for _, fc := range fr.ChildNodes {
			for _, fh := range fc.ChildNodes {
				for _, f := range fh.ChildNodes {
					x1 := int16(f.GetIntegerWithDefault("x1", 0))
					y1 := int16(f.GetIntegerWithDefault("y1", 0))
					x2 := int16(f.GetIntegerWithDefault("x2", 0))
					y2 := int16(f.GetIntegerWithDefault("y2", 0))
					id, err := strconv.Atoi(f.Name)
					if err != nil {
						continue
					}
					foothold := Foothold{
						Id:     uint32(id),
						First:  &point.Model{X: x1, Y: y1},
						Second: &point.Model{X: x2, Y: y2},
					}
					if x1 < lx {
						lx = x1
					}
					if x2 > ux {
						ux = x2
					}
					if y1 < ly {
						ly = y1
					}
					if y2 > uy {
						uy = y2
					}
					footholds = append(footholds, foothold)
				}
			}
		}
	}
	return NewFootholdTree(lx, ly, ux, uy).Insert(footholds)
}

func getAreas(exml *xml.Node) []Rectangle {
	results := make([]Rectangle, 0)
	a, err := exml.ChildByName("area")
	if err != nil {
		return results
	}
	for _, area := range a.ChildNodes {
		x1 := int16(area.GetIntegerWithDefault("x1", 0))
		y1 := int16(area.GetIntegerWithDefault("y1", 0))
		x2 := int16(area.GetFloatWithDefault("x2", 0))
		y2 := int16(area.GetIntegerWithDefault("y2", 0))
		result := Rectangle{
			X:      x1,
			Y:      y1,
			Width:  x2 - x1,
			Height: y2 - y1,
		}
		results = append(results, result)
	}
	return results
}

func getSeats(exml *xml.Node) uint32 {
	s, err := exml.ChildByName("seat")
	if err != nil {
		return 0
	}
	return uint32(len(s.PointNodes))
}

func getPlaceName(tenant tenant.Model, mapId uint32) string {
	md, err := GetMapStringRegistry().Get(tenant, mapId)
	if err != nil {
		return ""
	}
	return md.MapName()
}

func getStreetName(tenant tenant.Model, mapId uint32) string {
	md, err := GetMapStringRegistry().Get(tenant, mapId)
	if err != nil {
		return ""
	}
	return md.StreetName()
}

func getClock(exml *xml.Node) bool {
	_, err := exml.ChildByName("clock")
	return err != nil
}

func isBoat(exml *xml.Node) bool {
	_, err := exml.ChildByName("shipObj")
	return err != nil
}

func getBackgroundTypes(exml *xml.Node) []BackgroundType {
	results := make([]BackgroundType, 0)
	bts, err := exml.ChildByName("back")
	if err != nil {
		return results
	}
	for _, bt := range bts.ChildNodes {
		layerNum, err := strconv.Atoi(bt.Name)
		if err != nil {
			continue
		}
		backgroundType := bt.GetIntegerWithDefault("type", 0)
		results = append(results, BackgroundType{LayerNumber: uint32(layerNum), BackgroundType: uint32(backgroundType)})
	}

	return results
}

func getReactors(exml *xml.Node) []reactor.Model {
	results := make([]reactor.Model, 0)
	rd, err := exml.ChildByName("reactor")
	if err != nil {
		return results
	}
	uid := uint32(0)
	for _, r := range rd.ChildNodes {
		id := r.GetString("id", "")
		x := int16(r.GetIntegerWithDefault("x", 0))
		y := int16(r.GetIntegerWithDefault("y", 0))
		reactorTime := uint32(r.GetIntegerWithDefault("reactorTime", 0))
		name := r.GetString("name", "")
		fd := byte(r.GetIntegerWithDefault("f", 0))

		c, err := strconv.Atoi(id)
		if err != nil {
			continue
		}

		results = append(results, reactor.Model{
			Id:             uid,
			Classification: uint32(c),
			Name:           name,
			X:              x,
			Y:              y,
			Delay:          reactorTime * 1000,
			Direction:      fd,
		})
		uid += 1
	}
	return results
}

func getLife(t tenant.Model, exml *xml.Node) ([]monster.Model, []npc.Model) {
	monsters := make([]monster.Model, 0)
	npcs := make([]npc.Model, 0)
	ld, err := exml.ChildByName("life")
	if err != nil {
		return monsters, npcs
	}

	for i, life := range ld.ChildNodes {
		idstr := life.GetString("id", "")
		id, err := strconv.Atoi(idstr)
		if err != nil {
			continue
		}
		lifeType := life.GetString("type", "")
		team := life.GetIntegerWithDefault("team", -1)
		cy := int16(life.GetIntegerWithDefault("cy", 0))
		f := uint32(life.GetIntegerWithDefault("f", 0))
		fh := uint16(life.GetIntegerWithDefault("fh", 0))
		rx0 := int16(life.GetIntegerWithDefault("rx0", 0))
		rx1 := int16(life.GetIntegerWithDefault("rx1", 0))
		x := int16(life.GetIntegerWithDefault("x", 0))
		y := int16(life.GetIntegerWithDefault("y", 0))
		hide := life.GetIntegerWithDefault("hide", 0)
		mobTime := uint32(life.GetIntegerWithDefault("mobTime", 0))

		if lifeType == "m" {
			monster := monster.Model{
				Id:       uint32(i + 1),
				Template: uint32(id),
				MobTime:  mobTime,
				Team:     team,
				CY:       cy,
				F:        f,
				FH:       fh,
				RX0:      rx0,
				RX1:      rx1,
				X:        x,
				Y:        y,
				Hide:     hide == 1,
			}
			monsters = append(monsters, monster)
		} else if lifeType == "n" {
			nn, err := npc2.GetNpcStringRegistry().Get(t, uint32(id))
			if err != nil {
				continue
			}

			npc := npc.Model{
				Id:       strconv.Itoa(i + 1),
				Template: uint32(id),
				Name:     nn.Name(),
				CY:       cy,
				F:        f,
				FH:       fh,
				RX0:      rx0,
				RX1:      rx1,
				X:        x,
				Y:        y,
				Hide:     hide == 1,
			}
			npcs = append(npcs, npc)
		}

	}

	return monsters, npcs
}
