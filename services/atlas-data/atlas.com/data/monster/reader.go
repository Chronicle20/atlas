package monster

import (
	"atlas-data/element"
	"atlas-data/xml"
	"context"
	"fmt"
	"github.com/Chronicle20/atlas-model/model"
	tenant "github.com/Chronicle20/atlas-tenant"
	"github.com/sirupsen/logrus"
	"math"
	"path/filepath"
	"strconv"
	"strings"
)

func parseMonsterId(filePath string) (uint32, error) {
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

func Read(l logrus.FieldLogger) func(ctx context.Context) func(np model.Provider[xml.Node]) model.Provider[RestModel] {
	return func(ctx context.Context) func(np model.Provider[xml.Node]) model.Provider[RestModel] {
		t := tenant.MustFromContext(ctx)
		return func(np model.Provider[xml.Node]) model.Provider[RestModel] {
			exml, err := np()
			if err != nil {
				return model.ErrorProvider[RestModel](err)
			}

			monsterId, err := parseMonsterId(exml.Name)
			if err != nil {
				return model.ErrorProvider[RestModel](err)
			}
			l.Debugf("Processing monster [%d].", monsterId)

			node, err := exml.ChildByName("info")
			if err != nil {
				return model.ErrorProvider[RestModel](err)
			}
			m := &RestModel{Id: monsterId}
			m.HP = uint32(node.GetIntegerWithDefault("maxHP", math.MaxInt32))
			m.Friendly = node.GetIntegerWithDefault("damagedByMob", 0) == 1
			m.WeaponAttack = uint32(node.GetIntegerWithDefault("PADamage", 0))
			m.WeaponDefense = uint32(node.GetIntegerWithDefault("PDDamage", 0))
			m.MagicAttack = uint32(node.GetIntegerWithDefault("MADamage", 0))
			m.MagicDefense = uint32(node.GetIntegerWithDefault("MDDamage", 0))
			m.MP = uint32(node.GetIntegerWithDefault("maxMP", 0))
			m.Experience = uint32(node.GetIntegerWithDefault("exp", 0))
			m.Level = uint32(node.GetIntegerWithDefault("level", 0))
			m.RemoveAfter = uint32(node.GetIntegerWithDefault("removeAfter", 0))
			m.Boss = node.GetIntegerWithDefault("boss", 0) > 0
			m.ExplosiveReward = node.GetIntegerWithDefault("explosiveReward", 0) > 0
			m.FFALoot = node.GetIntegerWithDefault("publicReward", 0) > 0
			m.Undead = node.GetIntegerWithDefault("undead", 0) > 0
			ms, err := GetMonsterStringRegistry().Get(t, monsterId)
			if err != nil {
				return model.ErrorProvider[RestModel](err)
			}

			m.Name = ms.Name()
			m.BuffToGive = uint32(node.GetIntegerWithDefault("buff", 0))
			m.CP = uint32(node.GetIntegerWithDefault("getCP", 0))
			m.RemoveOnMiss = node.GetIntegerWithDefault("removeOnMiss", 0) > 0
			m.CoolDamage = getCoolDamage(node)
			m.LoseItems = getLoseItems(node)
			m.SelfDestruction = getSelfDestruction(node)
			m.FirstAttack = getFirstAttack(node)
			m.DropPeriod = uint32(node.GetIntegerWithDefault("dropItemPeriod", 0) * 10000)
			hpBarBoss := getHPBarBoss(t, monsterId)
			if hpBarBoss {
				m.TagColor = byte(node.GetIntegerWithDefault("hpTagColor", 0))
				m.TagBackgroundColor = byte(node.GetIntegerWithDefault("hpTagBgcolor", 0))
			} else {
				m.TagColor = 0
				m.TagBackgroundColor = 0
			}
			m.AnimationTimes = getAnimationTimes(exml)
			m.Revives = getRevives(node)
			m.Resistances = getResistances(node)
			m.Skills = getSkills(node)
			m.Banish = getBanish(node)
			m.FixedStance = getFixedStance(exml, node)
			return model.FixedProvider(*m)
		}
	}
}

func getFixedStance(root xml.Node, node *xml.Node) uint32 {
	noFlip := node.GetIntegerWithDefault("noFlip", 0)
	if noFlip > 0 {
		x, _ := root.GetPoint("stand/0/origin", 0, 0)
		if x < 1 {
			return 5
		}
		return 4
	}
	return 0
}

func getBanish(node *xml.Node) banish {
	b, err := node.ChildByName("ban")
	if err != nil {
		return banish{}
	}
	message := b.GetString("banMsg", "")
	mapId := uint32(b.GetIntegerWithDefault("banMap/0/field", 0))
	portal := b.GetString("banMap/0/portal", "sp")
	return banish{
		Message:    message,
		MapId:      mapId,
		PortalName: portal,
	}
}

func getSkills(node *xml.Node) []skill {
	results := make([]skill, 0)
	s, err := node.ChildByName("skill")
	if err != nil {
		return results
	}
	for _, c := range s.ChildNodes {
		skillId := uint32(c.GetIntegerWithDefault("skill", 0))
		level := uint32(c.GetIntegerWithDefault("level", 0))
		results = append(results, skill{
			Id:    skillId,
			Level: level,
		})
	}
	return results
}

func getResistances(node *xml.Node) map[string]string {
	resistances := node.GetString("elemAttr", "")
	results := make(map[string]string)
	for i := 0; i < len(resistances); i += 2 {
		e, _ := element.FromChar(string(resistances[i]))
		ei, _ := strconv.Atoi(string(resistances[i+1]))
		ef, _ := element.EffectivenessByNumber(ei)
		results[e] = ef
	}
	return results
}

func getRevives(node *xml.Node) []uint32 {
	results := make([]uint32, 0)
	c, err := node.ChildByName("revive")
	if err != nil {
		return results
	}
	for _, c2 := range c.IntegerNodes {
		results = append(results, uint32(c.GetIntegerWithDefault(c2.Name, 0)))
	}
	return results
}

func getAnimationTimes(node xml.Node) map[string]uint32 {
	results := make(map[string]uint32)
	for _, c := range node.ChildNodes {
		if c.Name != "info" {
			delay := uint32(0)
			for _, c2 := range c.CanvasNodes {
				delay += uint32(c2.GetIntegerWithDefault("delay", 0))
			}
			results[c.Name] = delay
		}
	}
	return results
}

func getHPBarBoss(t tenant.Model, monsterId uint32) bool {
	g, err := GetMonsterGaugeRegistry().Get(t, monsterId)
	if err != nil {
		return false
	}
	return g.Exists()
}

func getFirstAttack(node *xml.Node) bool {
	c, err := node.ChildByName("firstAttack")
	if err != nil {
		return false
	}
	return math.Round(c.GetFloatWithDefault("firstAttack", 0)) > 0
}

func getSelfDestruction(node *xml.Node) selfDestruction {
	c, err := node.ChildByName("selfDestruction")
	if err != nil {
		return selfDestruction{}
	}
	action := byte(c.GetIntegerWithDefault("action", 0))
	removeAfter := c.GetIntegerWithDefault("removeAfter", -1)
	hp := c.GetIntegerWithDefault("hp", -1)
	return selfDestruction{
		Action:      action,
		RemoveAfter: removeAfter,
		HP:          hp,
	}
}

func getLoseItems(node *xml.Node) []loseItem {
	results := make([]loseItem, 0)
	c, err := node.ChildByName("loseItem")
	if err != nil {
		return results
	}
	if len(c.ChildNodes) == 0 {
		return results
	}
	for _, ci := range c.ChildNodes {
		results = append(results, getLoseItem(ci))
	}
	return results
}

func getLoseItem(node xml.Node) loseItem {
	id := uint32(node.GetIntegerWithDefault("id", 0))
	chance := byte(node.GetIntegerWithDefault("prop", 0))
	x := byte(node.GetIntegerWithDefault("x", 0))
	return loseItem{
		Id:     id,
		Chance: chance,
		X:      x,
	}
}

func getCoolDamage(node *xml.Node) coolDamage {
	c, err := node.ChildByName("coolDamage")
	if err != nil {
		return coolDamage{}
	}
	damage := uint32(c.GetIntegerWithDefault("coolDamage", 0))
	probability := uint32(c.GetIntegerWithDefault("coolDamageProb", 0))
	return coolDamage{Damage: damage, Probability: probability}
}
