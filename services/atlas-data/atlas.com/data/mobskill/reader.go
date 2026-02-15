package mobskill

import (
	"atlas-data/xml"
	"strconv"

	"github.com/Chronicle20/atlas-model/model"
	"github.com/sirupsen/logrus"
)

func Read(l logrus.FieldLogger) func(np model.Provider[xml.Node]) model.Provider[[]RestModel] {
	return func(np model.Provider[xml.Node]) model.Provider[[]RestModel] {
		exml, err := np()
		if err != nil {
			return model.ErrorProvider[[]RestModel](err)
		}

		res := make([]RestModel, 0)
		for _, skillNode := range exml.ChildNodes {
			skillId, err := strconv.Atoi(skillNode.Name)
			if err != nil {
				continue
			}

			levelNode, err := skillNode.ChildByName("level")
			if err != nil {
				continue
			}

			for _, lvlNode := range levelNode.ChildNodes {
				level, err := strconv.Atoi(lvlNode.Name)
				if err != nil {
					continue
				}

				m := readLevel(l, uint16(skillId), uint16(level), lvlNode)
				res = append(res, m)
			}
		}

		l.Debugf("Processed [%d] mob skills.", len(res))
		return model.FixedProvider(res)
	}
}

func readLevel(l logrus.FieldLogger, skillId uint16, level uint16, node xml.Node) RestModel {
	m := RestModel{
		SkillId: skillId,
		Level:   level,
	}
	m.MpCon = uint32(node.GetIntegerWithDefault("mpCon", 0))
	m.Duration = uint32(node.GetIntegerWithDefault("time", 0))
	m.Hp = uint32(node.GetIntegerWithDefault("hp", 100))
	m.X = node.GetIntegerWithDefault("x", 0)
	m.Y = node.GetIntegerWithDefault("y", 0)
	m.Prop = uint32(node.GetIntegerWithDefault("prop", 100))
	m.Count = uint32(node.GetIntegerWithDefault("count", 1))
	m.Limit = uint32(node.GetIntegerWithDefault("limit", 0))
	m.SummonEffect = uint32(node.GetIntegerWithDefault("summonEffect", 0))

	// Handle "interval" and the known typo "inteval"
	interval := node.GetIntegerWithDefault("interval", 0)
	if interval == 0 {
		interval = node.GetIntegerWithDefault("inteval", 0)
	}
	m.Interval = uint32(interval)

	// Bounding box (lt = left-top, rb = right-bottom)
	m.LtX, m.LtY = node.GetPoint("lt", 0, 0)
	m.RbX, m.RbY = node.GetPoint("rb", 0, 0)

	// Summon monster IDs from numbered children
	m.Summons = getSummons(node)

	l.Debugf("Processing mob skill [%d] level [%d].", skillId, level)
	return m
}

func getSummons(node xml.Node) []uint32 {
	results := make([]uint32, 0)
	for _, c := range node.IntegerNodes {
		// Summon entries are numbered children (0, 1, 2, ...)
		_, err := strconv.Atoi(c.Name)
		if err != nil {
			continue
		}
		val, err := strconv.ParseUint(c.Value, 10, 32)
		if err != nil {
			continue
		}
		results = append(results, uint32(val))
	}
	return results
}
