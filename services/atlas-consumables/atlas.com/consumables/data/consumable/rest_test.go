package consumable

import "testing"

func TestExtractRewardFields(t *testing.T) {
	rm := RewardRestModel{ItemId: 1132010, Count: 1, Prob: 100, Effect: "Effect/BasicEff/Event1/Good", WorldMsg: "/name got /item", Period: 7200}
	got, err := ExtractReward(rm)
	if err != nil {
		t.Fatal(err)
	}
	if got.ItemId() != 1132010 || got.Count() != 1 || got.Prob() != 100 {
		t.Fatalf("base = {%d,%d,%d}", got.ItemId(), got.Count(), got.Prob())
	}
	if got.Effect() != "Effect/BasicEff/Event1/Good" {
		t.Errorf("Effect() = %q", got.Effect())
	}
	if got.WorldMsg() != "/name got /item" {
		t.Errorf("WorldMsg() = %q", got.WorldMsg())
	}
	if got.Period() != 7200 {
		t.Errorf("Period() = %d", got.Period())
	}
}

func TestExtractPropagatesRewardsToModel(t *testing.T) {
	rm := RestModel{Id: 2022309, Rewards: []RewardRestModel{{ItemId: 1, Count: 1, Prob: 10, Period: -1}}}
	m, err := Extract(rm)
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Rewards()) != 1 {
		t.Fatalf("len(m.Rewards()) = %d, want 1", len(m.Rewards()))
	}
	if m.Rewards()[0].Prob() != 10 || m.Rewards()[0].Period() != -1 {
		t.Errorf("reward = prob %d period %d", m.Rewards()[0].Prob(), m.Rewards()[0].Period())
	}
}
