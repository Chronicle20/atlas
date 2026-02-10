package quest

import (
	"atlas-data/xml"
	"testing"

	"github.com/sirupsen/logrus/hooks/test"
)

const testQuestInfoXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="QuestInfo.img">
  <imgdir name="2000">
    <string name="name" value="Mai's First Training"/>
    <string name="parent" value="Maple Island"/>
    <int name="area" value="10"/>
    <int name="order" value="1"/>
    <int name="autoStart" value="0"/>
    <int name="autoPreComplete" value="0"/>
    <int name="autoComplete" value="1"/>
    <int name="timeLimit" value="3600"/>
  </imgdir>
  <imgdir name="10000">
    <string name="name" value="Suspicious Offer?!"/>
    <int name="area" value="50"/>
    <int name="autoStart" value="1"/>
    <int name="autoPreComplete" value="1"/>
  </imgdir>
  <imgdir name="8248">
    <string name="name" value="Maple 7th Day Market opens tomorrow!"/>
    <int name="area" value="50"/>
    <int name="autoStart" value="1"/>
    <int name="autoPreComplete" value="1"/>
  </imgdir>
</imgdir>
`

const testCheckXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="Check.img">
  <imgdir name="2000">
    <imgdir name="0">
      <int name="npc" value="1002000"/>
      <int name="lvmin" value="8"/>
      <int name="lvmax" value="15"/>
      <imgdir name="job">
        <int name="0" value="0"/>
      </imgdir>
      <imgdir name="item">
        <imgdir name="0">
          <int name="id" value="4031013"/>
          <int name="count" value="1"/>
        </imgdir>
      </imgdir>
    </imgdir>
    <imgdir name="1">
      <int name="npc" value="1002000"/>
      <imgdir name="mob">
        <imgdir name="0">
          <int name="id" value="100100"/>
          <int name="count" value="5"/>
        </imgdir>
        <imgdir name="1">
          <int name="id" value="100101"/>
          <int name="count" value="3"/>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
  <imgdir name="10000">
    <imgdir name="0">
      <int name="npc" value="9000021"/>
      <int name="lvmin" value="10"/>
      <string name="start" value="2009060100"/>
      <string name="end" value="2009082600"/>
      <int name="normalAutoStart" value="1"/>
      <int name="infoNumber" value="10024"/>
    </imgdir>
    <imgdir name="1">
      <int name="npc" value="9000021"/>
      <imgdir name="quest">
        <imgdir name="0">
          <int name="id" value="9999"/>
          <int name="state" value="2"/>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
  <imgdir name="8248">
    <imgdir name="0">
      <int name="npc" value="9209001"/>
      <int name="normalAutoStart" value="1"/>
      <int name="dayByDay" value="1"/>
      <imgdir name="dayOfWeek">
        <string name="sat" value="1"/>
      </imgdir>
    </imgdir>
    <imgdir name="1">
      <int name="npc" value="9209001"/>
    </imgdir>
  </imgdir>
</imgdir>
`

const testActXML = `
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<imgdir name="Act.img">
  <imgdir name="2000">
    <imgdir name="0">
    </imgdir>
    <imgdir name="1">
      <int name="exp" value="500"/>
      <int name="money" value="1000"/>
      <int name="pop" value="5"/>
      <int name="nextQuest" value="2001"/>
      <imgdir name="item">
        <imgdir name="0">
          <int name="id" value="2000000"/>
          <int name="count" value="10"/>
          <int name="prop" value="-1"/>
        </imgdir>
        <imgdir name="1">
          <int name="id" value="1302000"/>
          <int name="count" value="1"/>
          <int name="job" value="2"/>
          <int name="gender" value="0"/>
          <int name="period" value="10080"/>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
  <imgdir name="10000">
    <imgdir name="0">
    </imgdir>
    <imgdir name="1">
      <int name="exp" value="10000"/>
      <imgdir name="skill">
        <imgdir name="0">
          <int name="id" value="1000001"/>
          <int name="skillLevel" value="1"/>
          <int name="masterLevel" value="10"/>
          <imgdir name="job">
            <int name="0" value="100"/>
            <int name="1" value="110"/>
          </imgdir>
        </imgdir>
      </imgdir>
    </imgdir>
  </imgdir>
  <imgdir name="8248">
    <imgdir name="0">
    </imgdir>
    <imgdir name="1">
    </imgdir>
  </imgdir>
</imgdir>
`

func TestReadQuestInfo(t *testing.T) {
	l, _ := test.NewNullLogger()

	quests := ReadQuestInfo(l)(xml.FromByteArrayProvider([]byte(testQuestInfoXML)))

	if len(quests) != 3 {
		t.Fatalf("expected 3 quests, got %d", len(quests))
	}

	// Test quest 2000
	q2000, exists := quests[2000]
	if !exists {
		t.Fatal("quest 2000 not found")
	}
	if q2000.Id != 2000 {
		t.Fatalf("expected quest id 2000, got %d", q2000.Id)
	}
	if q2000.Name != "Mai's First Training" {
		t.Fatalf("expected name 'Mai's First Training', got '%s'", q2000.Name)
	}
	if q2000.Parent != "Maple Island" {
		t.Fatalf("expected parent 'Maple Island', got '%s'", q2000.Parent)
	}
	if q2000.Area != 10 {
		t.Fatalf("expected area 10, got %d", q2000.Area)
	}
	if q2000.Order != 1 {
		t.Fatalf("expected order 1, got %d", q2000.Order)
	}
	if q2000.AutoStart {
		t.Fatal("expected autoStart false")
	}
	if q2000.AutoPreComplete {
		t.Fatal("expected autoPreComplete false")
	}
	if !q2000.AutoComplete {
		t.Fatal("expected autoComplete true")
	}
	if q2000.TimeLimit != 3600 {
		t.Fatalf("expected timeLimit 3600, got %d", q2000.TimeLimit)
	}

	// Test quest 10000
	q10000, exists := quests[10000]
	if !exists {
		t.Fatal("quest 10000 not found")
	}
	if !q10000.AutoStart {
		t.Fatal("expected autoStart true")
	}
	if !q10000.AutoPreComplete {
		t.Fatal("expected autoPreComplete true")
	}
}

func TestReadQuestCheck(t *testing.T) {
	l, _ := test.NewNullLogger()

	// Start with empty quests map
	quests := make(map[uint32]RestModel)
	quests[2000] = RestModel{Id: 2000, Name: "Test Quest"}
	quests[10000] = RestModel{Id: 10000, Name: "Test Quest 2"}
	quests[8248] = RestModel{Id: 8248, Name: "Maple 7th Day Market opens tomorrow!"}

	quests = ReadQuestCheck(l)(xml.FromByteArrayProvider([]byte(testCheckXML)))(quests)

	// Test quest 2000 start requirements
	q2000 := quests[2000]
	startReq := q2000.StartRequirements
	if startReq.NpcId != 1002000 {
		t.Fatalf("expected npc 1002000, got %d", startReq.NpcId)
	}
	if startReq.LevelMin != 8 {
		t.Fatalf("expected lvmin 8, got %d", startReq.LevelMin)
	}
	if startReq.LevelMax != 15 {
		t.Fatalf("expected lvmax 15, got %d", startReq.LevelMax)
	}
	if len(startReq.Jobs) != 1 || startReq.Jobs[0] != 0 {
		t.Fatalf("expected job [0], got %v", startReq.Jobs)
	}
	if len(startReq.Items) != 1 {
		t.Fatalf("expected 1 item requirement, got %d", len(startReq.Items))
	}
	if startReq.Items[0].Id != 4031013 || startReq.Items[0].Count != 1 {
		t.Fatalf("expected item {4031013, 1}, got %v", startReq.Items[0])
	}

	// Test quest 2000 end requirements
	endReq := q2000.EndRequirements
	if len(endReq.Mobs) != 2 {
		t.Fatalf("expected 2 mob requirements, got %d", len(endReq.Mobs))
	}
	if endReq.Mobs[0].Id != 100100 || endReq.Mobs[0].Count != 5 {
		t.Fatalf("expected mob {100100, 5}, got %v", endReq.Mobs[0])
	}
	if endReq.Mobs[1].Id != 100101 || endReq.Mobs[1].Count != 3 {
		t.Fatalf("expected mob {100101, 3}, got %v", endReq.Mobs[1])
	}

	// Test quest 10000
	q10000 := quests[10000]
	if q10000.StartRequirements.InfoNumber != 10024 {
		t.Fatalf("expected infoNumber 10024, got %d", q10000.StartRequirements.InfoNumber)
	}
	if !q10000.StartRequirements.NormalAutoStart {
		t.Fatal("expected normalAutoStart true")
	}
	if q10000.StartRequirements.Start != "2009060100" {
		t.Fatalf("expected start '2009060100', got '%s'", q10000.StartRequirements.Start)
	}
	if q10000.StartRequirements.End != "2009082600" {
		t.Fatalf("expected end '2009082600', got '%s'", q10000.StartRequirements.End)
	}

	// Test quest prerequisites
	if len(q10000.EndRequirements.Quests) != 1 {
		t.Fatalf("expected 1 quest requirement, got %d", len(q10000.EndRequirements.Quests))
	}
	if q10000.EndRequirements.Quests[0].Id != 9999 || q10000.EndRequirements.Quests[0].State != 2 {
		t.Fatalf("expected quest {9999, 2}, got %v", q10000.EndRequirements.Quests[0])
	}

	// Test quest 8248 - dayByDay and dayOfWeek
	q8248 := quests[8248]
	if !q8248.StartRequirements.DayByDay {
		t.Fatal("expected dayByDay true for quest 8248")
	}
	if !q8248.StartRequirements.NormalAutoStart {
		t.Fatal("expected normalAutoStart true for quest 8248")
	}
	if len(q8248.StartRequirements.DayOfWeek) != 1 {
		t.Fatalf("expected 1 dayOfWeek entry, got %d", len(q8248.StartRequirements.DayOfWeek))
	}
	if q8248.StartRequirements.DayOfWeek[0] != "sat" {
		t.Fatalf("expected dayOfWeek 'sat', got '%s'", q8248.StartRequirements.DayOfWeek[0])
	}
}

func TestReadQuestAct(t *testing.T) {
	l, _ := test.NewNullLogger()

	// Start with empty quests map
	quests := make(map[uint32]RestModel)
	quests[2000] = RestModel{Id: 2000, Name: "Test Quest"}
	quests[10000] = RestModel{Id: 10000, Name: "Test Quest 2"}
	quests[8248] = RestModel{Id: 8248, Name: "Maple 7th Day Market opens tomorrow!"}

	quests = ReadQuestAct(l)(xml.FromByteArrayProvider([]byte(testActXML)))(quests)

	// Test quest 2000 end actions
	q2000 := quests[2000]
	endAct := q2000.EndActions
	if endAct.Exp != 500 {
		t.Fatalf("expected exp 500, got %d", endAct.Exp)
	}
	if endAct.Money != 1000 {
		t.Fatalf("expected money 1000, got %d", endAct.Money)
	}
	if endAct.Fame != 5 {
		t.Fatalf("expected fame 5, got %d", endAct.Fame)
	}
	if endAct.NextQuest != 2001 {
		t.Fatalf("expected nextQuest 2001, got %d", endAct.NextQuest)
	}

	// Test item rewards
	if len(endAct.Items) != 2 {
		t.Fatalf("expected 2 item rewards, got %d", len(endAct.Items))
	}
	item0 := endAct.Items[0]
	if item0.Id != 2000000 || item0.Count != 10 || item0.Prop != -1 {
		t.Fatalf("expected item {2000000, 10, -1}, got {%d, %d, %d}", item0.Id, item0.Count, item0.Prop)
	}
	item1 := endAct.Items[1]
	if item1.Id != 1302000 || item1.Count != 1 {
		t.Fatalf("expected item id 1302000 count 1, got id %d count %d", item1.Id, item1.Count)
	}
	if item1.Job != 2 {
		t.Fatalf("expected job 2, got %d", item1.Job)
	}
	if item1.Gender != 0 {
		t.Fatalf("expected gender 0, got %d", item1.Gender)
	}
	if item1.Period != 10080 {
		t.Fatalf("expected period 10080, got %d", item1.Period)
	}

	// Test quest 10000 skill rewards
	q10000 := quests[10000]
	if q10000.EndActions.Exp != 10000 {
		t.Fatalf("expected exp 10000, got %d", q10000.EndActions.Exp)
	}
	if len(q10000.EndActions.Skills) != 1 {
		t.Fatalf("expected 1 skill reward, got %d", len(q10000.EndActions.Skills))
	}
	skill := q10000.EndActions.Skills[0]
	if skill.Id != 1000001 {
		t.Fatalf("expected skill id 1000001, got %d", skill.Id)
	}
	if skill.Level != 1 {
		t.Fatalf("expected skill level 1, got %d", skill.Level)
	}
	if skill.MasterLevel != 10 {
		t.Fatalf("expected master level 10, got %d", skill.MasterLevel)
	}
	if len(skill.Jobs) != 2 || skill.Jobs[0] != 100 || skill.Jobs[1] != 110 {
		t.Fatalf("expected jobs [100, 110], got %v", skill.Jobs)
	}
}

func TestReadQuestIntegration(t *testing.T) {
	l, _ := test.NewNullLogger()

	// Simulate full processing pipeline
	quests := ReadQuestInfo(l)(xml.FromByteArrayProvider([]byte(testQuestInfoXML)))
	quests = ReadQuestCheck(l)(xml.FromByteArrayProvider([]byte(testCheckXML)))(quests)
	quests = ReadQuestAct(l)(xml.FromByteArrayProvider([]byte(testActXML)))(quests)

	if len(quests) != 3 {
		t.Fatalf("expected 3 quests, got %d", len(quests))
	}

	// Verify quest 2000 has all data merged
	q2000 := quests[2000]
	if q2000.Name != "Mai's First Training" {
		t.Fatal("quest info not preserved after merge")
	}
	if q2000.StartRequirements.NpcId != 1002000 {
		t.Fatal("check data not merged")
	}
	if q2000.EndActions.Exp != 500 {
		t.Fatal("act data not merged")
	}
}
