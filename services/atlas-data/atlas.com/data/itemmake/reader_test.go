package itemmake

import (
	"atlas-data/xml"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"

	"github.com/Chronicle20/atlas/libs/atlas-model/model"
)

const testXML = `<imgdir name="ItemMake.img">
  <imgdir name="0">
    <imgdir name="04260000">
      <int name="reqLevel" value="0"/>
      <int name="reqSkillLevel" value="0"/>
      <int name="itemNum" value="1"/>
      <int name="tuc" value="0"/>
      <int name="meso" value="0"/>
      <imgdir name="recipe">
        <imgdir name="0">
          <int name="item" value="4000000"/>
          <int name="count" value="1"/>
        </imgdir>
      </imgdir>
      <imgdir name="randomReward">
        <imgdir name="0">
          <int name="item" value="4260000"/>
          <int name="itemNum" value="1"/>
          <int name="prob" value="70"/>
        </imgdir>
        <imgdir name="1">
          <int name="item" value="4260001"/>
          <int name="itemNum" value="1"/>
          <int name="prob" value="25"/>
        </imgdir>
        <imgdir name="2">
          <int name="item" value="4260002"/>
          <int name="itemNum" value="1"/>
          <int name="prob" value="5"/>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
  <imgdir name="1">
    <imgdir name="01082002">
      <int name="reqLevel" value="30"/>
      <int name="reqSkillLevel" value="2"/>
      <int name="itemNum" value="1"/>
      <int name="tuc" value="7"/>
      <int name="meso" value="1200"/>
      <int name="catalyst" value="4130000"/>
      <int name="reqItem" value="4000021"/>
      <int name="reqEquip" value="1002419"/>
      <imgdir name="recipe">
        <imgdir name="0">
          <int name="item" value="4011001"/>
          <int name="count" value="5"/>
        </imgdir>
        <imgdir name="1">
          <int name="item" value="4011002"/>
          <int name="count" value="3"/>
        </imgdir>
        <imgdir name="2">
          <int name="item" value="4021007"/>
          <int name="count" value="1"/>
        </imgdir>
      </imgdir>
      <imgdir name="reqQuest">
        <int name="21614" value="3"/>
      </imgdir>
    </imgdir>
  </imgdir>
  <imgdir name="2">
    <imgdir name="02020000">
      <int name="reqLevel" value="10"/>
      <int name="itemNum" value="3"/>
      <int name="meso" value="500"/>
      <imgdir name="recipe">
        <imgdir name="0">
          <int name="item" value="4000001"/>
          <int name="count" value="2"/>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
  <imgdir name="4">
    <imgdir name="04030000">
      <int name="reqLevel" value="15"/>
      <int name="itemNum" value="1"/>
      <int name="meso" value="800"/>
    </imgdir>
  </imgdir>
  <imgdir name="8">
    <imgdir name="08000000">
      <int name="reqLevel" value="20"/>
      <int name="itemNum" value="1"/>
      <int name="meso" value="900"/>
    </imgdir>
  </imgdir>
  <imgdir name="16">
    <imgdir name="16000000">
      <int name="reqLevel" value="25"/>
      <int name="itemNum" value="1"/>
      <int name="meso" value="1000"/>
    </imgdir>
  </imgdir>
  <imgdir name="1">
    <imgdir name="NOT_A_NUMBER">
      <int name="reqLevel" value="1"/>
    </imgdir>
  </imgdir>
</imgdir>`

func identity(m RestModel) RestModel {
	return m
}

func readFixture(t *testing.T) map[uint32]RestModel {
	t.Helper()

	l, _ := test.NewNullLogger()
	rms := Read(l)(xml.FromByteArrayProvider([]byte(testXML)))
	rmm, err := model.CollectToMap[RestModel, uint32, RestModel](rms, func(m RestModel) uint32 { return m.Id }, identity)()
	if err != nil {
		t.Fatal(err)
	}
	return rmm
}

func TestReadCoversEveryTopLevelGroup(t *testing.T) {
	m := readFixture(t)

	expected := map[uint32]uint32{
		4260000:  0,
		1082002:  1,
		2020000:  2,
		4030000:  4,
		8000000:  8,
		16000000: 16,
	}

	for id, group := range expected {
		rm, ok := m[id]
		if !ok {
			t.Fatalf("m[%d] does not exist.", id)
		}
		if rm.Group != group {
			t.Fatalf("m[%d].Group = %d, want %d", id, rm.Group, group)
		}
	}
}

func TestReadScalars(t *testing.T) {
	m := readFixture(t)

	rm, ok := m[1082002]
	if !ok {
		t.Fatalf("m[1082002] does not exist.")
	}
	if rm.ReqLevel != 30 {
		t.Fatalf("rm.ReqLevel = %d, want 30", rm.ReqLevel)
	}
	if rm.ReqSkillLevel != 2 {
		t.Fatalf("rm.ReqSkillLevel = %d, want 2", rm.ReqSkillLevel)
	}
	if rm.ItemNum != 1 {
		t.Fatalf("rm.ItemNum = %d, want 1", rm.ItemNum)
	}
	if rm.Tuc != 7 {
		t.Fatalf("rm.Tuc = %d, want 7", rm.Tuc)
	}
	if rm.Meso != 1200 {
		t.Fatalf("rm.Meso = %d, want 1200", rm.Meso)
	}
	if rm.Catalyst != 4130000 {
		t.Fatalf("rm.Catalyst = %d, want 4130000", rm.Catalyst)
	}
	if rm.ReqItem != 4000021 {
		t.Fatalf("rm.ReqItem = %d, want 4000021", rm.ReqItem)
	}
	if rm.ReqEquip != 1002419 {
		t.Fatalf("rm.ReqEquip = %d, want 1002419", rm.ReqEquip)
	}
}

func TestReadAbsentScalarsDefaultToZero(t *testing.T) {
	m := readFixture(t)

	rm, ok := m[2020000]
	if !ok {
		t.Fatalf("m[2020000] does not exist.")
	}
	if rm.Tuc != 0 {
		t.Fatalf("rm.Tuc = %d, want 0", rm.Tuc)
	}
	if rm.Catalyst != 0 {
		t.Fatalf("rm.Catalyst = %d, want 0", rm.Catalyst)
	}
	if rm.ReqItem != 0 {
		t.Fatalf("rm.ReqItem = %d, want 0", rm.ReqItem)
	}
	if rm.ReqEquip != 0 {
		t.Fatalf("rm.ReqEquip = %d, want 0", rm.ReqEquip)
	}
	if rm.ReqSkillLevel != 0 {
		t.Fatalf("rm.ReqSkillLevel = %d, want 0", rm.ReqSkillLevel)
	}
}

func TestReadRecipeOrder(t *testing.T) {
	m := readFixture(t)

	rm, ok := m[1082002]
	if !ok {
		t.Fatalf("m[1082002] does not exist.")
	}

	expected := []MaterialRestModel{
		{ItemId: 4011001, Count: 5},
		{ItemId: 4011002, Count: 3},
		{ItemId: 4021007, Count: 1},
	}

	if len(rm.Recipe) != len(expected) {
		t.Fatalf("len(rm.Recipe) = %d, want %d", len(rm.Recipe), len(expected))
	}
	for i, e := range expected {
		if rm.Recipe[i] != e {
			t.Fatalf("rm.Recipe[%d] = %+v, want %+v", i, rm.Recipe[i], e)
		}
	}
}

func TestReadRandomRewardOrder(t *testing.T) {
	m := readFixture(t)

	rm, ok := m[4260000]
	if !ok {
		t.Fatalf("m[4260000] does not exist.")
	}

	expected := []RewardRestModel{
		{ItemId: 4260000, ItemNum: 1, Prob: 70},
		{ItemId: 4260001, ItemNum: 1, Prob: 25},
		{ItemId: 4260002, ItemNum: 1, Prob: 5},
	}

	if len(rm.RandomReward) != len(expected) {
		t.Fatalf("len(rm.RandomReward) = %d, want %d", len(rm.RandomReward), len(expected))
	}
	for i, e := range expected {
		if rm.RandomReward[i] != e {
			t.Fatalf("rm.RandomReward[%d] = %+v, want %+v", i, rm.RandomReward[i], e)
		}
	}
}

func TestReadRandomRewardAbsentIsEmpty(t *testing.T) {
	m := readFixture(t)

	rm, ok := m[1082002]
	if !ok {
		t.Fatalf("m[1082002] does not exist.")
	}
	if len(rm.RandomReward) != 0 {
		t.Fatalf("len(rm.RandomReward) = %d, want 0", len(rm.RandomReward))
	}
}

func TestReadReqQuest(t *testing.T) {
	m := readFixture(t)

	rm, ok := m[1082002]
	if !ok {
		t.Fatalf("m[1082002] does not exist.")
	}
	if len(rm.ReqQuest) != 1 {
		t.Fatalf("len(rm.ReqQuest) = %d, want 1", len(rm.ReqQuest))
	}
	if rm.ReqQuest[0].QuestId != 21614 {
		t.Fatalf("rm.ReqQuest[0].QuestId = %d, want 21614", rm.ReqQuest[0].QuestId)
	}
	if rm.ReqQuest[0].State != 3 {
		t.Fatalf("rm.ReqQuest[0].State = %d, want 3", rm.ReqQuest[0].State)
	}

	other, ok := m[4260000]
	if !ok {
		t.Fatalf("m[4260000] does not exist.")
	}
	if len(other.ReqQuest) != 0 {
		t.Fatalf("len(other.ReqQuest) = %d, want 0", len(other.ReqQuest))
	}
}

func TestReadRecipeAbsentIsEmpty(t *testing.T) {
	m := readFixture(t)

	rm, ok := m[4030000]
	if !ok {
		t.Fatalf("m[4030000] does not exist.")
	}
	if len(rm.Recipe) != 0 {
		t.Fatalf("len(rm.Recipe) = %d, want 0", len(rm.Recipe))
	}
}

func TestReadSkipsMalformedEntryWithoutAborting(t *testing.T) {
	m := readFixture(t)

	for _, id := range []uint32{4260000, 1082002, 2020000, 4030000, 8000000, 16000000} {
		if _, ok := m[id]; !ok {
			t.Fatalf("m[%d] does not exist.", id)
		}
	}
	if _, ok := m[0]; ok {
		t.Fatalf("m[0] exists, want no entry with a zero id.")
	}
}
