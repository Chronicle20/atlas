package monster

import (
	"atlas-data/element"
	"atlas-data/tenant"
	"atlas-data/wz"
	"atlas-data/xml"
	"errors"
	"fmt"
	"math"
	"strconv"
)

func Read(tenant tenant.Model, monsterId uint32) (*Model, error) {
	i, err := findMonster(tenant, monsterId)
	if err != nil {
		return nil, err
	}

	exml, err := xml.Read(i.Path())
	if err != nil {
		return nil, err
	}
	return getMonsterFromXML(tenant, monsterId, exml)
}

func findMonster(tenant tenant.Model, monsterId uint32) (*wz.FileEntry, error) {
	path := fmt.Sprintf("%07d", monsterId)
	if val, ok := wz.GetFileCache().GetFile(tenant, "Mob.wz", path+".img.xml"); ok == nil {
		return val, nil
	}
	return nil, errors.New(fmt.Sprintf("monster %d not found", monsterId))
}

func findUIWindow(tenant tenant.Model) (*wz.FileEntry, error) {
	if val, ok := wz.GetFileCache().GetFile(tenant, "UI.wz", "UIWindow.img.xml"); ok == nil {
		return val, nil
	}
	return nil, errors.New("file not found")
}

func findMonsterNames(tenant tenant.Model) (*wz.FileEntry, error) {
	if val, ok := wz.GetFileCache().GetFile(tenant, "String.wz", "Mob.img.xml"); ok == nil {
		return val, nil
	}
	return nil, errors.New("file not found")
}

func getMonsterFromXML(tenant tenant.Model, monsterId uint32, exml *xml.Node) (*Model, error) {
	node, err := exml.ChildByName("info")
	if err != nil {
		return nil, err
	}
	m := &Model{id: monsterId}
	m.hp = uint32(node.GetIntegerWithDefault("maxHP", math.MaxInt32))
	m.friendly = node.GetIntegerWithDefault("damagedByMob", 0) == 1
	m.weaponAttack = uint32(node.GetIntegerWithDefault("PADamage", 0))
	m.weaponDefense = uint32(node.GetIntegerWithDefault("PDDamage", 0))
	m.magicAttack = uint32(node.GetIntegerWithDefault("MADamage", 0))
	m.magicDefense = uint32(node.GetIntegerWithDefault("MDDamage", 0))
	m.mp = uint32(node.GetIntegerWithDefault("maxMP", 0))
	m.experience = uint32(node.GetIntegerWithDefault("exp", 0))
	m.level = uint32(node.GetIntegerWithDefault("level", 0))
	m.removeAfter = uint32(node.GetIntegerWithDefault("removeAfter", 0))
	m.boss = node.GetIntegerWithDefault("boss", 0) > 0
	m.explosiveReward = node.GetIntegerWithDefault("explosiveReward", 0) > 0
	m.ffaLoot = node.GetIntegerWithDefault("publicReward", 0) > 0
	m.undead = node.GetIntegerWithDefault("undead", 0) > 0
	m.name = getMonsterName(tenant, monsterId)
	m.buffToGive = uint32(node.GetIntegerWithDefault("buff", 0))
	m.cp = uint32(node.GetIntegerWithDefault("getCP", 0))
	m.removeOnMiss = node.GetIntegerWithDefault("removeOnMiss", 0) > 0
	m.coolDamage = getCoolDamage(node)
	m.loseItems = getLoseItems(node)
	m.selfDestruction = getSelfDestruction(node)
	m.firstAttack = getFirstAttack(node)
	m.dropPeriod = uint32(node.GetIntegerWithDefault("dropItemPeriod", 0) * 10000)
	hpBarBoss := getHPBarBoss(tenant, monsterId)
	if hpBarBoss {
		m.tagColor = byte(node.GetIntegerWithDefault("hpTagColor", 0))
		m.tagBackgroundColor = byte(node.GetIntegerWithDefault("hpTagBgcolor", 0))
	} else {
		m.tagColor = 0
		m.tagBackgroundColor = 0
	}
	m.animationTimes = getAnimationTimes(exml)
	m.revives = getRevives(node)
	m.resistances = getResistances(node)
	m.skills = getSkills(node)
	m.banish = getBanish(node)
	m.fixedStance = getFixedStance(exml, node)
	return m, nil
}

func getMonsterName(tenant tenant.Model, monsterId uint32) string {
	uiwz, err := findMonsterNames(tenant)
	if err != nil {
		return "MISSINGNO"
	}
	exml, err := xml.Read(uiwz.Path())
	if err != nil {
		return "MISSINGNO"
	}
	m, err := exml.ChildByName(fmt.Sprintf("%d", monsterId))
	if err != nil {
		return "MISSINGNO"
	}
	return m.GetString("name", "MISSINGNO")
}

func getFixedStance(root *xml.Node, node *xml.Node) uint32 {
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

func getBanish(node *xml.Node) *Banish {
	b, err := node.ChildByName("ban")
	if err != nil {
		return nil
	}
	message := b.GetString("banMsg", "")
	mapId := uint32(b.GetIntegerWithDefault("banMap/0/field", 0))
	portal := b.GetString("banMap/0/portal", "sp")
	return &Banish{
		message:    message,
		mapId:      mapId,
		portalName: portal,
	}
}

func getSkills(node *xml.Node) []Skill {
	results := make([]Skill, 0)
	s, err := node.ChildByName("skill")
	if err != nil {
		return results
	}
	for _, c := range s.ChildNodes {
		skillId := uint32(c.GetIntegerWithDefault("skill", 0))
		level := uint32(c.GetIntegerWithDefault("level", 0))
		results = append(results, Skill{
			id:    skillId,
			level: level,
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

func getAnimationTimes(node *xml.Node) map[string]uint32 {
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

func getHPBarBoss(tenant tenant.Model, monsterId uint32) bool {
	uiwz, err := findUIWindow(tenant)
	if err != nil {
		return false
	}
	exml, err := xml.Read(uiwz.Path())
	if err != nil {
		return false
	}
	d, err := exml.ChildByName("MobGage/Mob")
	if err != nil {
		return false
	}
	if len(d.CanvasNodes) == 0 {
		return false
	}
	for _, c := range d.CanvasNodes {
		if c.Name == strconv.Itoa(int(monsterId)) {
			return true
		}
	}
	return false
}

func getFirstAttack(node *xml.Node) bool {
	c, err := node.ChildByName("firstAttack")
	if err != nil {
		return false
	}
	return math.Round(c.GetFloatWithDefault("firstAttack", 0)) > 0
}

func getSelfDestruction(node *xml.Node) *SelfDestruction {
	c, err := node.ChildByName("selfDestruction")
	if err != nil {
		return nil
	}
	action := byte(c.GetIntegerWithDefault("action", 0))
	removeAfter := c.GetIntegerWithDefault("removeAfter", -1)
	hp := c.GetIntegerWithDefault("hp", -1)
	return &SelfDestruction{
		action:      action,
		removeAfter: removeAfter,
		hp:          hp,
	}
}

func getLoseItems(node *xml.Node) []LoseItem {
	results := make([]LoseItem, 0)
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

func getLoseItem(node xml.Node) LoseItem {
	id := uint32(node.GetIntegerWithDefault("id", 0))
	chance := byte(node.GetIntegerWithDefault("prop", 0))
	x := byte(node.GetIntegerWithDefault("x", 0))
	return LoseItem{
		itemId: id,
		chance: chance,
		x:      x,
	}
}

func getCoolDamage(node *xml.Node) *CoolDamage {
	c, err := node.ChildByName("coolDamage")
	if err != nil {
		return nil
	}
	damage := uint32(c.GetIntegerWithDefault("coolDamage", 0))
	probability := uint32(c.GetIntegerWithDefault("coolDamageProb", 0))
	return &CoolDamage{damage: damage, probability: probability}
}
