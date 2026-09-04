package monster

import "testing"

func TestExtractCarriesHide(t *testing.T) {
	tests := []struct {
		name string
		hide bool
	}{
		{name: "hidden", hide: true},
		{name: "visible", hide: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rm := RestModel{
				Id:       7,
				Template: 9400545,
				MobTime:  0,
				Team:     1,
				CY:       -10,
				F:        1,
				FH:       3,
				RX0:      -50,
				RX1:      50,
				X:        100,
				Y:        200,
				Hide:     tt.hide,
			}

			sp, err := Extract(rm)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if sp.Hide != tt.hide {
				t.Fatalf("expected Hide %v, got %v", tt.hide, sp.Hide)
			}

			if sp.Id != 7 {
				t.Errorf("expected Id 7, got %d", sp.Id)
			}
			if sp.Template != 9400545 {
				t.Errorf("expected Template 9400545, got %d", sp.Template)
			}
			if sp.MobTime != 0 {
				t.Errorf("expected MobTime 0, got %d", sp.MobTime)
			}
			if sp.Team != 1 {
				t.Errorf("expected Team 1, got %d", sp.Team)
			}
			if sp.Cy != -10 {
				t.Errorf("expected Cy -10, got %d", sp.Cy)
			}
			if sp.F != 1 {
				t.Errorf("expected F 1, got %d", sp.F)
			}
			if sp.Fh != 3 {
				t.Errorf("expected Fh 3, got %d", sp.Fh)
			}
			if sp.Rx0 != -50 {
				t.Errorf("expected Rx0 -50, got %d", sp.Rx0)
			}
			if sp.Rx1 != 50 {
				t.Errorf("expected Rx1 50, got %d", sp.Rx1)
			}
			if sp.X != 100 {
				t.Errorf("expected X 100, got %d", sp.X)
			}
			if sp.Y != 200 {
				t.Errorf("expected Y 200, got %d", sp.Y)
			}
		})
	}
}

func TestClassify(t *testing.T) {
	t.Run("partitions and preserves order", func(t *testing.T) {
		points := []SpawnPoint{
			{Id: 1, MobTime: 30, Hide: false},
			{Id: 2, MobTime: 0, Hide: false},
			{Id: 3, MobTime: -1, Hide: false},
			{Id: 4, MobTime: -2, Hide: false},
			{Id: 5, MobTime: 0, Hide: true},
			{Id: 6, MobTime: -1, Hide: true},
		}

		c := Classify(points)

		assertIds(t, "Recurring", c.Recurring, []uint32{1, 2})
		assertIds(t, "OneTime", c.OneTime, []uint32{3, 4})
		assertIds(t, "Hidden", c.Hidden, []uint32{5, 6})
	})

	t.Run("empty input", func(t *testing.T) {
		c := Classify(nil)

		if len(c.Recurring) != 0 {
			t.Errorf("expected 0 Recurring, got %d", len(c.Recurring))
		}
		if len(c.OneTime) != 0 {
			t.Errorf("expected 0 OneTime, got %d", len(c.OneTime))
		}
		if len(c.Hidden) != 0 {
			t.Errorf("expected 0 Hidden, got %d", len(c.Hidden))
		}
	})
}

func assertIds(t *testing.T, bucket string, got []SpawnPoint, want []uint32) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("%s: expected %d entries, got %d", bucket, len(want), len(got))
	}
	for i, w := range want {
		if got[i].Id != w {
			t.Errorf("%s[%d]: expected id %d, got %d", bucket, i, w, got[i].Id)
		}
	}
}
